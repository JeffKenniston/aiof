package agent

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// ============================================================================
// ADR-20: Autonomic Graph Plasticity & Self-Healing Swarm Re-Choreography
// ============================================================================

const (
	// DefaultCellFusionThreshold defines the inter-node message rate (msg/min)
	// that triggers Cell Fusion between two interacting sub-agents.
	DefaultCellFusionThreshold = 50.0

	// DefaultStallTimeout defines the maximum elapsed duration without activity
	// before a sub-agent node is declared stalled, triggering Cell Fission.
	DefaultStallTimeout = 10 * time.Second

	// DefaultErrorRateThreshold defines the ratio of failed messages (0.0 - 1.0)
	// that marks a node as degraded and eligible for self-healing replacement.
	DefaultErrorRateThreshold = 0.30
)

var (
	ErrNodeNotFound             = errors.New("plasticity: agent node not found in topology graph")
	ErrInvalidEdge              = errors.New("plasticity: invalid execution edge between non-existent nodes")
	ErrFusionFailed             = errors.New("plasticity: cell fusion evaluation failed")
	ErrFissionFailed            = errors.New("plasticity: cell fission self-healing replacement failed")
	ErrEngineClosed             = errors.New("plasticity: graph plasticity engine is closed")
	ErrTenantIsolationViolation = errors.New("plasticity: cross-tenant cell fusion denied by OPA policy")
)

// MetricSample represents a single telemetry observation between sub-agent nodes.
type MetricSample struct {
	SourceAgentID string        `json:"source_agent_id"`
	TargetAgentID string        `json:"target_agent_id"`
	Latency       time.Duration `json:"latency"`
	IsError       bool          `json:"is_error"`
	TokenCount    int           `json:"token_count"`
	Timestamp     time.Time     `json:"timestamp"`
}

// InterNodeTelemetry holds aggregated traffic statistics for an ordered node pair (A -> B).
type InterNodeTelemetry struct {
	SourceAgentID  string        `json:"source_agent_id"`
	TargetAgentID  string        `json:"target_agent_id"`
	TotalMessages  uint64        `json:"total_messages"`
	TotalErrors    uint64        `json:"total_errors"`
	TotalTokens    uint64        `json:"total_tokens"`
	MessagesPerMin float64       `json:"messages_per_min"`
	AvgLatency     time.Duration `json:"avg_latency"`
	P95Latency     time.Duration `json:"p95_latency"`
	ErrorRate      float64       `json:"error_rate"`
	LastActive     time.Time     `json:"last_active"`
}

// NodeState tracks active execution status and capabilities of a sub-agent node in the DAG.
type NodeState struct {
	AgentID         string            `json:"agent_id"`
	Role            string            `json:"role"`
	SystemPrompt    string            `json:"system_prompt"`
	AllowedTools    []string          `json:"allowed_tools"`
	TenantID        string            `json:"tenant_id,omitempty"`      // OPA Multi-Tenant Isolation
	SecurityLabel   string            `json:"security_label,omitempty"` // OPA Security Label
	Metadata        map[string]string `json:"metadata,omitempty"`
	IsFused         bool              `json:"is_fused"`
	FusedFromAgents []string          `json:"fused_from_agents,omitempty"`
	Status          string            `json:"status"` // "ACTIVE", "STALLED", "FAILED", "FUSED", "REPLACED"
	LastActive      time.Time         `json:"last_active"`
	TotalInbound    uint64            `json:"total_inbound"`
	TotalOutbound   uint64            `json:"total_outbound"`
	TotalErrors     uint64            `json:"total_errors"`
}

// ExecutionEdge represents a directed dependencies or communication channel in the DAG.
type ExecutionEdge struct {
	FromAgentID string  `json:"from_agent_id"`
	ToAgentID   string  `json:"to_agent_id"`
	Weight      float64 `json:"weight"`
	Active      bool    `json:"active"`
}

// TelemetryObserver collects and processes sliding-window metrics across DAG execution nodes.
type TelemetryObserver struct {
	mu           sync.RWMutex
	samples      []MetricSample
	windowSize   time.Duration
	nodeActivity map[string]time.Time
}

func NewTelemetryObserver(windowSize time.Duration) *TelemetryObserver {
	if windowSize <= 0 {
		windowSize = 1 * time.Minute
	}
	return &TelemetryObserver{
		samples:      make([]MetricSample, 0, 1024),
		windowSize:   windowSize,
		nodeActivity: make(map[string]time.Time),
	}
}

func (o *TelemetryObserver) RecordSample(sample MetricSample) {
	o.mu.Lock()
	defer o.mu.Unlock()

	if sample.Timestamp.IsZero() {
		sample.Timestamp = time.Now().UTC()
	}

	o.samples = append(o.samples, sample)
	o.nodeActivity[sample.SourceAgentID] = sample.Timestamp
	o.nodeActivity[sample.TargetAgentID] = sample.Timestamp
	o.pruneOldSamples(sample.Timestamp)
}

func (o *TelemetryObserver) pruneOldSamples(now time.Time) {
	cutoff := now.Add(-o.windowSize)
	validIdx := 0
	for i, s := range o.samples {
		if s.Timestamp.After(cutoff) {
			validIdx = i
			break
		}
	}
	if validIdx > 0 {
		o.samples = o.samples[validIdx:]
	}
}

func (o *TelemetryObserver) GetInterNodeTelemetry(sourceID, targetID string) InterNodeTelemetry {
	o.mu.RLock()
	defer o.mu.RUnlock()

	now := time.Now().UTC()
	cutoff := now.Add(-o.windowSize)

	var count, errors, tokens uint64
	var latencies []time.Duration
	var lastActive time.Time

	for _, s := range o.samples {
		if s.Timestamp.Before(cutoff) {
			continue
		}
		if (s.SourceAgentID == sourceID && s.TargetAgentID == targetID) ||
			(s.SourceAgentID == targetID && s.TargetAgentID == sourceID) {
			count++
			tokens += uint64(s.TokenCount)
			if s.IsError {
				errors++
			}
			latencies = append(latencies, s.Latency)
			if s.Timestamp.After(lastActive) {
				lastActive = s.Timestamp
			}
		}
	}

	telemetry := InterNodeTelemetry{
		SourceAgentID: sourceID,
		TargetAgentID: targetID,
		TotalMessages: count,
		TotalErrors:   errors,
		TotalTokens:   tokens,
		LastActive:    lastActive,
	}

	minutes := o.windowSize.Minutes()
	if minutes > 0 {
		telemetry.MessagesPerMin = float64(count) / minutes
	}

	if count > 0 {
		telemetry.ErrorRate = float64(errors) / float64(count)

		sort.Slice(latencies, func(i, j int) bool {
			return latencies[i] < latencies[j]
		})

		var totalDuration time.Duration
		for _, l := range latencies {
			totalDuration += l
		}
		telemetry.AvgLatency = totalDuration / time.Duration(count)

		p95Idx := int(float64(count) * 0.95)
		if p95Idx >= len(latencies) {
			p95Idx = len(latencies) - 1
		}
		telemetry.P95Latency = latencies[p95Idx]
	}

	return telemetry
}

// PlasticityEngine orchestrates real-time topology re-choreography (Cell Fusion & Cell Fission).
type PlasticityEngine struct {
	mu                 sync.RWMutex
	nodes              map[string]*NodeState
	edges              map[string]map[string]*ExecutionEdge // From -> To -> Edge
	observer           *TelemetryObserver
	fusionThreshold    float64       // Msg/min threshold for Cell Fusion
	stallTimeout       time.Duration // Duration before declaring node stalled
	errorRateThreshold float64       // Error rate threshold for Cell Fission
	closed             atomic.Bool
	ctx                context.Context
	cancel             context.CancelFunc

	// Callbacks for swarm node replacement
	nodeSwarmFactory func(ctx context.Context, role, systemPrompt string, tools []string) (*NodeState, error)
	opaPolicyChecker func(ctx context.Context, nodeA, nodeB *NodeState) error // OPA Policy Gating
}

type PlasticityOptions struct {
	FusionThreshold    float64
	StallTimeout       time.Duration
	ErrorRateThreshold float64
	WindowSize         time.Duration
	NodeSwarmFactory   func(ctx context.Context, role, systemPrompt string, tools []string) (*NodeState, error)
	OPAPolicyChecker   func(ctx context.Context, nodeA, nodeB *NodeState) error
}

func NewPlasticityEngine(opts PlasticityOptions) *PlasticityEngine {
	if opts.FusionThreshold <= 0 {
		opts.FusionThreshold = DefaultCellFusionThreshold
	}
	if opts.StallTimeout <= 0 {
		opts.StallTimeout = DefaultStallTimeout
	}
	if opts.ErrorRateThreshold <= 0 {
		opts.ErrorRateThreshold = DefaultErrorRateThreshold
	}
	if opts.WindowSize <= 0 {
		opts.WindowSize = 1 * time.Minute
	}

	ctx, cancel := context.WithCancel(context.Background())
	engine := &PlasticityEngine{
		nodes:              make(map[string]*NodeState),
		edges:              make(map[string]map[string]*ExecutionEdge),
		observer:           NewTelemetryObserver(opts.WindowSize),
		fusionThreshold:    opts.FusionThreshold,
		stallTimeout:       opts.StallTimeout,
		errorRateThreshold: opts.ErrorRateThreshold,
		ctx:                ctx,
		cancel:             cancel,
		nodeSwarmFactory:   opts.NodeSwarmFactory,
		opaPolicyChecker:   opts.OPAPolicyChecker,
	}

	return engine
}

// RegisterNode adds a new sub-agent node into the active DAG topology.
func (e *PlasticityEngine) RegisterNode(node *NodeState) error {
	if e.closed.Load() {
		return ErrEngineClosed
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	if node.AgentID == "" {
		return errors.New("plasticity: agent_id cannot be empty")
	}

	if node.Status == "" {
		node.Status = "ACTIVE"
	}
	if node.LastActive.IsZero() {
		node.LastActive = time.Now().UTC()
	}

	e.nodes[node.AgentID] = node
	if _, exists := e.edges[node.AgentID]; !exists {
		e.edges[node.AgentID] = make(map[string]*ExecutionEdge)
	}

	return nil
}

// AddEdge connects two nodes in the execution DAG topology.
func (e *PlasticityEngine) AddEdge(fromID, toID string, weight float64) error {
	if e.closed.Load() {
		return ErrEngineClosed
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	if _, exists := e.nodes[fromID]; !exists {
		return fmt.Errorf("%w: source node %s missing", ErrInvalidEdge, fromID)
	}
	if _, exists := e.nodes[toID]; !exists {
		return fmt.Errorf("%w: target node %s missing", ErrInvalidEdge, toID)
	}

	edge := &ExecutionEdge{
		FromAgentID: fromID,
		ToAgentID:   toID,
		Weight:      weight,
		Active:      true,
	}

	if _, exists := e.edges[fromID]; !exists {
		e.edges[fromID] = make(map[string]*ExecutionEdge)
	}
	e.edges[fromID][toID] = edge

	return nil
}

// RecordMessage Telemetry updates message counters and registers samples with the observer.
func (e *PlasticityEngine) RecordMessage(fromID, toID string, latency time.Duration, isError bool, tokens int) {
	if e.closed.Load() {
		return
	}

	now := time.Now().UTC()
	e.observer.RecordSample(MetricSample{
		SourceAgentID: fromID,
		TargetAgentID: toID,
		Latency:       latency,
		IsError:       isError,
		TokenCount:    tokens,
		Timestamp:     now,
	})

	e.mu.Lock()
	defer e.mu.Unlock()

	if src, exists := e.nodes[fromID]; exists {
		src.TotalOutbound++
		src.LastActive = now
		if isError {
			src.TotalErrors++
		}
	}

	if dst, exists := e.nodes[toID]; exists {
		dst.TotalInbound++
		dst.LastActive = now
	}
}

// EvaluatePlasticity scans the graph topology for Cell Fusion and Cell Fission triggers.
func (e *PlasticityEngine) EvaluatePlasticity(ctx context.Context) ([]string, error) {
	if e.closed.Load() {
		return nil, ErrEngineClosed
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	var actions []string
	now := time.Now().UTC()

	// 1. Evaluate Cell Fusion Candidate Pairs (>50 msgs/min)
	fusedPairs := make(map[string]bool)
	nodeIDs := make([]string, 0, len(e.nodes))
	for id, n := range e.nodes {
		if n.Status == "ACTIVE" {
			nodeIDs = append(nodeIDs, id)
		}
	}

	for i := 0; i < len(nodeIDs); i++ {
		for j := i + 1; j < len(nodeIDs); j++ {
			aID := nodeIDs[i]
			bID := nodeIDs[j]

			if fusedPairs[aID] || fusedPairs[bID] {
				continue
			}

			telemetry := e.observer.GetInterNodeTelemetry(aID, bID)
			if telemetry.MessagesPerMin >= e.fusionThreshold {
				fusedNode, err := e.executeCellFusionLocked(aID, bID)
				if err == nil {
					fusedPairs[aID] = true
					fusedPairs[bID] = true
					msg := fmt.Sprintf("CELL_FUSION: Fused [%s, %s] -> %s (rate: %.1f msg/min)", aID, bID, fusedNode.AgentID, telemetry.MessagesPerMin)
					actions = append(actions, msg)
				}
			}
		}
	}

	// 2. Evaluate Cell Fission & Self-Healing (Stalls & Errors)
	for id, node := range e.nodes {
		if node.Status != "ACTIVE" && node.Status != "STALLED" {
			continue
		}

		isStalled := now.Sub(node.LastActive) > e.stallTimeout
		var errorRate float64
		if total := node.TotalInbound + node.TotalOutbound; total > 0 {
			errorRate = float64(node.TotalErrors) / float64(total)
		}
		isDegraded := errorRate >= e.errorRateThreshold

		if isStalled || isDegraded {
			reason := "STALL"
			if isDegraded {
				reason = fmt.Sprintf("HIGH_ERROR_RATE(%.2f)", errorRate)
			}

			replacementNode, err := e.executeCellFissionLocked(ctx, id, reason)
			if err == nil {
				msg := fmt.Sprintf("CELL_FISSION: Replaced failed node [%s] (%s) -> %s", id, reason, replacementNode.AgentID)
				actions = append(actions, msg)
			}
		}
	}

	return actions, nil
}

// executeCellFusionLocked performs prompt fusion and topology re-wiring. Must be called with lock held.
func (e *PlasticityEngine) executeCellFusionLocked(agentIDA, agentIDB string) (*NodeState, error) {
	nodeA, existsA := e.nodes[agentIDA]
	nodeB, existsB := e.nodes[agentIDB]
	if !existsA || !existsB {
		return nil, ErrNodeNotFound
	}

	// OPA Multi-Tenant & Security Label Isolation Gating
	if nodeA.TenantID != "" && nodeB.TenantID != "" && nodeA.TenantID != nodeB.TenantID {
		return nil, ErrTenantIsolationViolation
	}
	if nodeA.SecurityLabel != "" && nodeB.SecurityLabel != "" && nodeA.SecurityLabel != nodeB.SecurityLabel {
		return nil, ErrTenantIsolationViolation
	}
	if e.opaPolicyChecker != nil {
		if err := e.opaPolicyChecker(e.ctx, nodeA, nodeB); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrTenantIsolationViolation, err)
		}
	}

	fusedID := fmt.Sprintf("fused-%s-%s", agentIDA, agentIDB)

	// Combine & Harmonize System Prompts
	fusedPrompt := fmt.Sprintf(`
# Fused System Prompt [Cell Fusion: %s + %s]
You are a fused, multi-domain autonomous sub-agent combining the personas and roles of:
- Primary Role A: %s
- Primary Role B: %s

## Integrated Instructions
%s

---

%s
`, agentIDA, agentIDB, nodeA.Role, nodeB.Role, strings.TrimSpace(nodeA.SystemPrompt), strings.TrimSpace(nodeB.SystemPrompt))

	// Merge Allowed Tools
	toolMap := make(map[string]bool)
	for _, t := range nodeA.AllowedTools {
		toolMap[t] = true
	}
	for _, t := range nodeB.AllowedTools {
		toolMap[t] = true
	}
	fusedTools := make([]string, 0, len(toolMap))
	for t := range toolMap {
		fusedTools = append(fusedTools, t)
	}
	sort.Strings(fusedTools)

	tenantID := nodeA.TenantID
	if tenantID == "" {
		tenantID = nodeB.TenantID
	}
	secLabel := nodeA.SecurityLabel
	if secLabel == "" {
		secLabel = nodeB.SecurityLabel
	}

	fusedNode := &NodeState{
		AgentID:         fusedID,
		Role:            fmt.Sprintf("Fused(%s, %s)", nodeA.Role, nodeB.Role),
		SystemPrompt:    fusedPrompt,
		AllowedTools:    fusedTools,
		TenantID:        tenantID,
		SecurityLabel:   secLabel,
		IsFused:         true,
		FusedFromAgents: []string{agentIDA, agentIDB},
		Status:          "ACTIVE",
		LastActive:      time.Now().UTC(),
		Metadata: map[string]string{
			"fused_at": time.Now().UTC().Format(time.RFC3339),
		},
	}

	// Update node statuses
	nodeA.Status = "FUSED"
	nodeB.Status = "FUSED"
	e.nodes[fusedID] = fusedNode
	e.edges[fusedID] = make(map[string]*ExecutionEdge)

	// Re-wire DAG execution edges
	for fromID, targetMap := range e.edges {
		for toID, edge := range targetMap {
			if !edge.Active {
				continue
			}

			// Inbound edges to A or B -> re-wire to fusedNode
			if (toID == agentIDA || toID == agentIDB) && fromID != agentIDA && fromID != agentIDB {
				edge.Active = false
				e.edges[fromID][fusedID] = &ExecutionEdge{
					FromAgentID: fromID,
					ToAgentID:   fusedID,
					Weight:      edge.Weight,
					Active:      true,
				}
			}

			// Outbound edges from A or B -> re-wire from fusedNode
			if (fromID == agentIDA || fromID == agentIDB) && toID != agentIDA && toID != agentIDB {
				edge.Active = false
				e.edges[fusedID][toID] = &ExecutionEdge{
					FromAgentID: fusedID,
					ToAgentID:   toID,
					Weight:      edge.Weight,
					Active:      true,
				}
			}
		}
	}

	return fusedNode, nil
}

// executeCellFissionLocked swarms a replacement node and rewires DAG paths. Must be called with lock held.
func (e *PlasticityEngine) executeCellFissionLocked(ctx context.Context, failedAgentID, reason string) (*NodeState, error) {
	failedNode, exists := e.nodes[failedAgentID]
	if !exists {
		return nil, ErrNodeNotFound
	}

	failedNode.Status = "FAILED"
	replacementID := fmt.Sprintf("replacement-%s-%d", failedAgentID, time.Now().UnixNano()%100000)

	var replacementNode *NodeState
	var err error

	if e.nodeSwarmFactory != nil {
		replacementNode, err = e.nodeSwarmFactory(ctx, failedNode.Role, failedNode.SystemPrompt, failedNode.AllowedTools)
	}

	if err != nil || replacementNode == nil {
		// Fallback: Default replacement instantiation
		replacementNode = &NodeState{
			AgentID:      replacementID,
			Role:         failedNode.Role,
			SystemPrompt: failedNode.SystemPrompt,
			AllowedTools: failedNode.AllowedTools,
			Status:       "ACTIVE",
			LastActive:   time.Now().UTC(),
			Metadata: map[string]string{
				"replaced_agent_id":  failedAgentID,
				"replacement_reason": reason,
			},
		}
	} else if replacementNode.AgentID == "" {
		replacementNode.AgentID = replacementID
	}

	e.nodes[replacementNode.AgentID] = replacementNode
	e.edges[replacementNode.AgentID] = make(map[string]*ExecutionEdge)

	// Re-wire DAG Edges from Failed Node to Replacement Node
	for fromID, targetMap := range e.edges {
		for toID, edge := range targetMap {
			if !edge.Active {
				continue
			}

			// Inbound edges to failed node -> redirect to replacement node
			if toID == failedAgentID {
				edge.Active = false
				e.edges[fromID][replacementNode.AgentID] = &ExecutionEdge{
					FromAgentID: fromID,
					ToAgentID:   replacementNode.AgentID,
					Weight:      edge.Weight,
					Active:      true,
				}
			}

			// Outbound edges from failed node -> redirect from replacement node
			if fromID == failedAgentID {
				edge.Active = false
				e.edges[replacementNode.AgentID][toID] = &ExecutionEdge{
					FromAgentID: replacementNode.AgentID,
					ToAgentID:   toID,
					Weight:      edge.Weight,
					Active:      true,
				}
			}
		}
	}

	return replacementNode, nil
}

func (e *PlasticityEngine) GetNode(agentID string) (*NodeState, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	node, exists := e.nodes[agentID]
	if !exists {
		return nil, ErrNodeNotFound
	}
	return node, nil
}

func (e *PlasticityEngine) GetActiveNodes() []*NodeState {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var active []*NodeState
	for _, n := range e.nodes {
		if n.Status == "ACTIVE" {
			active = append(active, n)
		}
	}
	return active
}

func (e *PlasticityEngine) GetActiveEdges() []*ExecutionEdge {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var active []*ExecutionEdge
	for _, targetMap := range e.edges {
		for _, edge := range targetMap {
			if edge.Active {
				active = append(active, edge)
			}
		}
	}
	return active
}

func (e *PlasticityEngine) Close() {
	if e.closed.CompareAndSwap(false, true) {
		e.cancel()
	}
}
