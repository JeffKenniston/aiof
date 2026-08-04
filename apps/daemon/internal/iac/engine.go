package iac

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

// ============================================================================
// ADR-29: Agentic Infrastructure-as-Code (IaC) & Cloud Ops Engine
// ============================================================================

// IaCType defines supported Infrastructure-as-Code targets per ADR-29.
type IaCType string

const (
	IaCTypeTerraform     IaCType = "terraform"
	IaCTypeOpenTofu      IaCType = "opentofu"
	IaCTypeKubernetes    IaCType = "kubernetes"
	IaCTypeHelm          IaCType = "helm"
	IaCTypeDockerCompose IaCType = "docker-compose"
)

// DeploymentState defines the formal state machine states for cloud deployments.
type DeploymentState string

const (
	StatePending         DeploymentState = "PENDING"
	StateDryRunVerified  DeploymentState = "DRY_RUN_VERIFIED"
	StateCanaryDeploying DeploymentState = "CANARY_DEPLOYING"
	StateCanaryPromoted  DeploymentState = "CANARY_PROMOTED"
	StateCommitted       DeploymentState = "COMMITTED"
	StateFailed          DeploymentState = "FAILED"
	StateRolledBack      DeploymentState = "ROLLED_BACK"
)

// Severity indicates OPA security compliance violation levels.
type Severity string

const (
	SeverityCritical Severity = "CRITICAL"
	SeverityHigh     Severity = "HIGH"
	SeverityMedium   Severity = "MEDIUM"
	SeverityLow      Severity = "LOW"
)

// Common IaC Errors
var (
	ErrManifestInvalid        = errors.New("iac: manifest syntax or structure invalid")
	ErrOPAPolicyViolation     = errors.New("iac: critical OPA security policy violation detected")
	ErrDryRunFailed           = errors.New("iac: deployment plan dry-run validation failed")
	ErrCanaryHealthFailed     = errors.New("iac: canary telemetry metrics exceeded failure threshold")
	ErrRollbackFailed         = errors.New("iac: automated rollback state transition failed")
	ErrInvalidStateTransition = errors.New("iac: illegal state machine transition")
)

// ----------------------------------------------------------------------------
// IaC Manifest & Resource Specifications
// ----------------------------------------------------------------------------

// IaCResource represents a single cloud or infrastructure resource declaration.
type IaCResource struct {
	ID       string            `json:"id"`
	Name     string            `json:"name"`
	Kind     string            `json:"kind"` // e.g. "aws_s3_bucket", "Deployment", "Service"
	Provider string            `json:"provider"`
	Payload  string            `json:"payload"` // Raw HCL, YAML, or JSON
	Config   map[string]any    `json:"config"`
	Metadata map[string]string `json:"metadata,omitempty"`
	Hash     string            `json:"hash"`
}

// ComputeHash calculates a deterministic SHA-256 hash over the resource payload.
func (r *IaCResource) ComputeHash() string {
	h := sha256.New()
	h.Write([]byte(r.Kind + ":" + r.Name + ":" + r.Payload))
	r.Hash = hex.EncodeToString(h.Sum(nil))
	return r.Hash
}

// IaCManifest represents a compiled, multi-resource IaC deployment artifact.
type IaCManifest struct {
	ID           string        `json:"id"`
	Name         string        `json:"name"`
	Type         IaCType       `json:"type"`
	Version      string        `json:"version"`
	Resources    []IaCResource `json:"resources"`
	Environment  string        `json:"environment"`
	CreatedAt    time.Time     `json:"created_at"`
	ManifestHash string        `json:"manifest_hash"`
}

// ComputeHash computes a cryptographic digest across all contained resources.
func (m *IaCManifest) ComputeHash() string {
	h := sha256.New()
	h.Write([]byte(string(m.Type) + ":" + m.Name + ":" + m.Version + ":" + m.Environment))
	for _, res := range m.Resources {
		h.Write([]byte(res.ComputeHash()))
	}
	m.ManifestHash = hex.EncodeToString(h.Sum(nil))
	return m.ManifestHash
}

// ----------------------------------------------------------------------------
// Open Policy Agent (OPA) Security Compliance Engine
// ----------------------------------------------------------------------------

// OPAViolation describes a specific policy non-compliance finding.
type OPAViolation struct {
	PolicyID     string   `json:"policy_id"`
	ResourceName string   `json:"resource_name"`
	Severity     Severity `json:"severity"`
	Message      string   `json:"message"`
	Remediation  string   `json:"remediation"`
}

// ComplianceReport summarizes OPA security evaluation results.
type ComplianceReport struct {
	Passed         bool           `json:"passed"`
	RulesEvaluated int            `json:"rules_evaluated"`
	CriticalCount  int            `json:"critical_count"`
	HighCount      int            `json:"high_count"`
	MediumCount    int            `json:"medium_count"`
	Violations     []OPAViolation `json:"violations"`
	EvaluatedAt    time.Time      `json:"evaluated_at"`
}

// OPAPolicyEvaluator executes security rules over synthesized IaC manifests.
type OPAPolicyEvaluator struct {
	mu          sync.RWMutex
	customRules []func(res *IaCResource) *OPAViolation
}

func NewOPAPolicyEvaluator() *OPAPolicyEvaluator {
	eval := &OPAPolicyEvaluator{}
	eval.registerBuiltInRules()
	return eval
}

func (e *OPAPolicyEvaluator) registerBuiltInRules() {
	// Rule 1: Prevent unencrypted storage volumes
	e.customRules = append(e.customRules, func(res *IaCResource) *OPAViolation {
		payloadLower := strings.ToLower(res.Payload)
		if (strings.Contains(res.Kind, "s3_bucket") || strings.Contains(res.Kind, "ebs_volume") || strings.Contains(res.Kind, "PersistentVolume")) &&
			(strings.Contains(payloadLower, "encrypted = false") || strings.Contains(payloadLower, "encrypted: false")) {
			return &OPAViolation{
				PolicyID:     "OPA-SEC-001",
				ResourceName: res.Name,
				Severity:     SeverityCritical,
				Message:      "Storage resource explicitly disables encryption at rest",
				Remediation:  "Enable SSE-KMS or AES256 server-side encryption",
			}
		}
		return nil
	})

	// Rule 2: Enforce non-root execution in containers
	e.customRules = append(e.customRules, func(res *IaCResource) *OPAViolation {
		payloadLower := strings.ToLower(res.Payload)
		if (res.Kind == "Deployment" || res.Kind == "Pod" || res.Kind == "StatefulSet") &&
			(strings.Contains(payloadLower, "runasuser: 0") || strings.Contains(payloadLower, "privileged: true")) {
			return &OPAViolation{
				PolicyID:     "OPA-SEC-002",
				ResourceName: res.Name,
				Severity:     SeverityCritical,
				Message:      "Kubernetes workload configures privileged root execution",
				Remediation:  "Set runAsNonRoot: true and drop privileged capabilities",
			}
		}
		return nil
	})

	// Rule 3: Detect missing CPU/Memory resource limits
	e.customRules = append(e.customRules, func(res *IaCResource) *OPAViolation {
		payloadLower := strings.ToLower(res.Payload)
		if (res.Kind == "Deployment" || res.Kind == "DaemonSet") && !strings.Contains(payloadLower, "limits:") {
			return &OPAViolation{
				PolicyID:     "OPA-SEC-003",
				ResourceName: res.Name,
				Severity:     SeverityHigh,
				Message:      "Container workload missing explicit CPU/Memory resource limits",
				Remediation:  "Define resources.limits for CPU and memory in container specification",
			}
		}
		return nil
	})

	// Rule 4: Plaintext secret detection in environment variables
	e.customRules = append(e.customRules, func(res *IaCResource) *OPAViolation {
		payloadLower := strings.ToLower(res.Payload)
		if strings.Contains(payloadLower, "password=") || strings.Contains(payloadLower, "secret_key=") || strings.Contains(payloadLower, "api_key=") {
			return &OPAViolation{
				PolicyID:     "OPA-SEC-004",
				ResourceName: res.Name,
				Severity:     SeverityHigh,
				Message:      "Hardcoded plaintext secret detected in environment configuration",
				Remediation:  "Inject secrets using SecretKeyRef or HashiCorp Vault integration",
			}
		}
		return nil
	})
}

// EvaluateManifest runs all OPA policy rules against the manifest resources.
func (e *OPAPolicyEvaluator) EvaluateManifest(manifest *IaCManifest) *ComplianceReport {
	e.mu.RLock()
	defer e.mu.RUnlock()

	report := &ComplianceReport{
		Passed:      true,
		EvaluatedAt: time.Now().UTC(),
	}

	for _, res := range manifest.Resources {
		for _, rule := range e.customRules {
			report.RulesEvaluated++
			if violation := rule(&res); violation != nil {
				report.Violations = append(report.Violations, *violation)
				switch violation.Severity {
				case SeverityCritical:
					report.CriticalCount++
					report.Passed = false
				case SeverityHigh:
					report.HighCount++
					report.Passed = false
				case SeverityMedium:
					report.MediumCount++
				}
			}
		}
	}

	return report
}

// ----------------------------------------------------------------------------
// Canary Telemetry Observer (eBPF Traces & OpenTelemetry Spans)
// ----------------------------------------------------------------------------

// eBPFKernelTrace represents a kernel-level syscall metric collected during canary deployment.
type eBPFKernelTrace struct {
	Syscall    string `json:"syscall"`
	LatencyNs  int64  `json:"latency_ns"`
	DropCount  uint64 `json:"drop_count"`
	ErrorCount uint64 `json:"error_count"`
}

// OTELSpan represents an OpenTelemetry distributed trace span.
type OTELSpan struct {
	TraceID      string `json:"trace_id"`
	SpanID       string `json:"span_id"`
	ServiceName  string `json:"service_name"`
	DurationMs   int64  `json:"duration_ms"`
	StatusCode   string `json:"status_code"` // "STATUS_OK", "STATUS_ERROR"
	ErrorMessage string `json:"error_message,omitempty"`
}

// CanaryMetrics summarizes aggregated health telemetry during canary observation.
type CanaryMetrics struct {
	ErrorRatePercentage float64 `json:"error_rate_percentage"`
	LatencyP99Ms        int64   `json:"latency_p99_ms"`
	eBPFDropRate        float64 `json:"ebpf_drop_rate"`
	OTELSpanErrors      int     `json:"otel_span_errors"`
	TotalSpans          int     `json:"total_spans"`
}

// CanaryHealthThresholds defines telemetry tolerance limits for canary promotion.
type CanaryHealthThresholds struct {
	MaxErrorRatePct float64 `json:"max_error_rate_pct"` // Default 1.0%
	MaxP99LatencyMs int64   `json:"max_p99_latency_ms"` // Default 500ms
	MaxeBPFDropRate float64 `json:"max_ebpf_drop_rate"` // Default 0.01%
}

func DefaultThresholds() CanaryHealthThresholds {
	return CanaryHealthThresholds{
		MaxErrorRatePct: 1.0,
		MaxP99LatencyMs:  500,
		MaxeBPFDropRate: 0.01,
	}
}

// CanaryObserver collects and evaluates telemetry during canary rollouts.
type CanaryObserver struct {
	thresholds CanaryHealthThresholds
}

func NewCanaryObserver(thresholds CanaryHealthThresholds) *CanaryObserver {
	return &CanaryObserver{thresholds: thresholds}
}

// EvaluateTelemetry analyzes kernel traces and OTEL spans to assess canary health.
func (co *CanaryObserver) EvaluateTelemetry(traces []eBPFKernelTrace, spans []OTELSpan) (*CanaryMetrics, bool) {
	metrics := &CanaryMetrics{
		TotalSpans: len(spans),
	}

	var totalLatency int64
	for _, span := range spans {
		totalLatency += span.DurationMs
		if span.DurationMs > metrics.LatencyP99Ms {
			metrics.LatencyP99Ms = span.DurationMs
		}
		if span.StatusCode == "STATUS_ERROR" {
			metrics.OTELSpanErrors++
		}
	}

	if len(spans) > 0 {
		metrics.ErrorRatePercentage = (float64(metrics.OTELSpanErrors) / float64(len(spans))) * 100.0
	}

	var totalSyscalls uint64
	var totalDrops uint64
	for _, trace := range traces {
		totalSyscalls += 1
		totalDrops += trace.DropCount
	}
	if totalSyscalls > 0 {
		metrics.eBPFDropRate = (float64(totalDrops) / float64(totalSyscalls)) * 100.0
	}

	healthy := metrics.ErrorRatePercentage <= co.thresholds.MaxErrorRatePct &&
		metrics.LatencyP99Ms <= co.thresholds.MaxP99LatencyMs &&
		metrics.eBPFDropRate <= co.thresholds.MaxeBPFDropRate

	return metrics, healthy
}

// ----------------------------------------------------------------------------
// Deployment Choreographer & Rollback State Machine
// ----------------------------------------------------------------------------

// DeploymentRecord represents an active or historical deployment execution plan.
type DeploymentRecord struct {
	ID             string            `json:"id"`
	Manifest       IaCManifest       `json:"manifest"`
	BackupManifest *IaCManifest      `json:"backup_manifest,omitempty"`
	CurrentState   DeploymentState   `json:"current_state"`
	StateHistory   []StateTransition `json:"state_history"`
	Compliance     ComplianceReport  `json:"compliance"`
	CanaryResult   *CanaryMetrics    `json:"canary_result,omitempty"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
}

// StateTransition tracks formal state machine transitions.
type StateTransition struct {
	FromState DeploymentState `json:"from_state"`
	ToState   DeploymentState `json:"to_state"`
	Timestamp time.Time       `json:"timestamp"`
	Reason    string          `json:"reason"`
}

// DeploymentChoreographer manages the dry-run, canary, promotion, and rollback state machine.
type DeploymentChoreographer struct {
	mu            sync.Mutex
	evaluator     *OPAPolicyEvaluator
	observer      *CanaryObserver
	records       map[string]*DeploymentRecord
}

func NewDeploymentChoreographer(evaluator *OPAPolicyEvaluator, observer *CanaryObserver) *DeploymentChoreographer {
	if observer == nil {
		observer = NewCanaryObserver(DefaultThresholds())
	}
	if evaluator == nil {
		evaluator = NewOPAPolicyEvaluator()
	}
	return &DeploymentChoreographer{
		evaluator: evaluator,
		observer:  observer,
		records:   make(map[string]*DeploymentRecord),
	}
}

// CreateDeploymentRecord initializes a new deployment state machine.
func (dc *DeploymentChoreographer) CreateDeploymentRecord(manifest IaCManifest, backup *IaCManifest) (*DeploymentRecord, error) {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	if len(manifest.Resources) == 0 {
		return nil, ErrManifestInvalid
	}

	manifest.ComputeHash()
	if backup != nil {
		backup.ComputeHash()
	}

	rec := &DeploymentRecord{
		ID:             fmt.Sprintf("dep-%x", manifest.ManifestHash[:8]),
		Manifest:       manifest,
		BackupManifest: backup,
		CurrentState:   StatePending,
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}

	rec.StateHistory = append(rec.StateHistory, StateTransition{
		FromState: "",
		ToState:   StatePending,
		Timestamp: time.Now().UTC(),
		Reason:    "Deployment record created",
	})

	dc.records[rec.ID] = rec
	return rec, nil
}

// TransitionState applies a validated state machine transition.
func (dc *DeploymentChoreographer) transition(rec *DeploymentRecord, toState DeploymentState, reason string) error {
	fromState := rec.CurrentState

	// Validate legal state machine transitions
	valid := false
	switch fromState {
	case StatePending:
		valid = (toState == StateDryRunVerified || toState == StateFailed)
	case StateDryRunVerified:
		valid = (toState == StateCanaryDeploying || toState == StateCommitted || toState == StateFailed)
	case StateCanaryDeploying:
		valid = (toState == StateCanaryPromoted || toState == StateRolledBack || toState == StateFailed)
	case StateCanaryPromoted:
		valid = (toState == StateCommitted || toState == StateFailed)
	case StateCommitted, StateFailed, StateRolledBack:
		valid = false // Terminal states
	}

	if !valid {
		return fmt.Errorf("%w: cannot transition from %s to %s", ErrInvalidStateTransition, fromState, toState)
	}

	rec.CurrentState = toState
	rec.UpdatedAt = time.Now().UTC()
	rec.StateHistory = append(rec.StateHistory, StateTransition{
		FromState: fromState,
		ToState:   toState,
		Timestamp: time.Now().UTC(),
		Reason:    reason,
	})

	return nil
}

// ExecuteDryRun performs dry-run plan verification and OPA policy evaluation.
func (dc *DeploymentChoreographer) ExecuteDryRun(ctx context.Context, deploymentID string) error {
	dc.mu.Lock()
	rec, exists := dc.records[deploymentID]
	if !exists {
		dc.mu.Unlock()
		return fmt.Errorf("iac: deployment record %s not found", deploymentID)
	}
	dc.mu.Unlock()

	// 1. Evaluate OPA Security Compliance
	report := dc.evaluator.EvaluateManifest(&rec.Manifest)
	rec.Compliance = *report

	if !report.Passed {
		_ = dc.transition(rec, StateFailed, "OPA security policy check failed")
		return fmt.Errorf("%w: %d critical/high policy violations", ErrOPAPolicyViolation, report.CriticalCount+report.HighCount)
	}

	// 2. Perform Dry-Run Plan Validation
	for _, res := range rec.Manifest.Resources {
		if res.Name == "" || res.Kind == "" {
			_ = dc.transition(rec, StateFailed, "Resource manifest missing name or kind")
			return fmt.Errorf("%w: resource missing required fields", ErrDryRunFailed)
		}
	}

	// Transition to StateDryRunVerified
	return dc.transition(rec, StateDryRunVerified, "Dry-run plan validation and OPA security checks passed")
}

// ExecuteCanaryDeploy deploys canary workloads and observes telemetry metrics.
func (dc *DeploymentChoreographer) ExecuteCanaryDeploy(ctx context.Context, deploymentID string, traces []eBPFKernelTrace, spans []OTELSpan) error {
	dc.mu.Lock()
	rec, exists := dc.records[deploymentID]
	if !exists {
		dc.mu.Unlock()
		return fmt.Errorf("iac: deployment record %s not found", deploymentID)
	}
	dc.mu.Unlock()

	if err := dc.transition(rec, StateCanaryDeploying, "Initiating canary deployment observation window"); err != nil {
		return err
	}

	// Observe telemetry
	metrics, healthy := dc.observer.EvaluateTelemetry(traces, spans)
	rec.CanaryResult = metrics

	if !healthy {
		// Canary failed -> Trigger Automated Rollback State Machine (ADR-29)
		reason := fmt.Sprintf("Canary health check failed (ErrorRate: %.2f%%, P99Latency: %dms)", metrics.ErrorRatePercentage, metrics.LatencyP99Ms)
		rbErr := dc.TriggerRollback(ctx, deploymentID, reason)
		if rbErr != nil && !errors.Is(rbErr, ErrRollbackFailed) {
			return rbErr
		}
		return fmt.Errorf("%w: %s", ErrCanaryHealthFailed, reason)
	}

	// Promote canary
	if err := dc.transition(rec, StateCanaryPromoted, "Canary telemetry verified within health thresholds"); err != nil {
		return err
	}

	// Final commit
	return dc.transition(rec, StateCommitted, "Deployment promoted and committed successfully")
}

// TriggerRollback executes automated rollback to previous stable backup state.
func (dc *DeploymentChoreographer) TriggerRollback(ctx context.Context, deploymentID string, reason string) error {
	dc.mu.Lock()
	rec, exists := dc.records[deploymentID]
	if !exists {
		dc.mu.Unlock()
		return fmt.Errorf("iac: deployment record %s not found", deploymentID)
	}
	dc.mu.Unlock()

	if rec.BackupManifest == nil {
		// No backup available — still mark as rolled back but log the degraded state
		_ = dc.transition(rec, StateRolledBack, "Rollback executed without backup manifest (degraded): "+reason)
		return fmt.Errorf("%w: no backup manifest provided for rollback; state is rolled back", ErrRollbackFailed)
	}

	// Execute rollback state machine transition
	err := dc.transition(rec, StateRolledBack, "Automated rollback executed: "+reason)
	if err != nil {
		// Force state to rolled back if state transition rule blocks direct rollback
		rec.CurrentState = StateRolledBack
		rec.UpdatedAt = time.Now().UTC()
		rec.StateHistory = append(rec.StateHistory, StateTransition{
			FromState: rec.CurrentState,
			ToState:   StateRolledBack,
			Timestamp: time.Now().UTC(),
			Reason:    "Automated rollback forced: " + reason,
		})
	}

	return nil
}

func (dc *DeploymentChoreographer) GetRecord(id string) (*DeploymentRecord, bool) {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	rec, ok := dc.records[id]
	return rec, ok
}

// ----------------------------------------------------------------------------
// IaC Synthesis Engine
// ----------------------------------------------------------------------------

// IaCEngine provides synthesis and validation entrypoints for IaC targets.
type IaCEngine struct {
	evaluator     *OPAPolicyEvaluator
	choreographer *DeploymentChoreographer
}

func NewIaCEngine() *IaCEngine {
	eval := NewOPAPolicyEvaluator()
	obs := NewCanaryObserver(DefaultThresholds())
	return &IaCEngine{
		evaluator:     eval,
		choreographer: NewDeploymentChoreographer(eval, obs),
	}
}

// SynthesizeManifest compiles raw inputs into a validated IaCManifest.
func (e *IaCEngine) SynthesizeManifest(name string, targetType IaCType, resources []IaCResource) (*IaCManifest, error) {
	if name == "" || len(resources) == 0 {
		return nil, ErrManifestInvalid
	}

	manifest := &IaCManifest{
		ID:          fmt.Sprintf("manifest-%x", time.Now().UnixNano()),
		Name:        name,
		Type:        targetType,
		Version:     "v1.0.0",
		Resources:   resources,
		Environment: "production",
		CreatedAt:   time.Now().UTC(),
	}

	manifest.ComputeHash()
	return manifest, nil
}

func (e *IaCEngine) GetChoreographer() *DeploymentChoreographer {
	return e.choreographer
}

func (e *IaCEngine) GetEvaluator() *OPAPolicyEvaluator {
	return e.evaluator
}
