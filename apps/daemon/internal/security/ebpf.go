package security

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// ============================================================================
// ADR-12: Synchronous eBPF LSM Boundary (MCPGuard/Grimlock)
// & Fail-Closed Backpressure Circuit Breaker
// ============================================================================

// LSMHookType identifies the specific Linux Security Module (LSM) eBPF hook point.
type LSMHookType string

const (
	LSMHookProcessExec   LSMHookType = "bpf_lsm_bprm_check_security"
	LSMHookFileOpen      LSMHookType = "bpf_lsm_file_open"
	LSMHookSocketConnect LSMHookType = "bpf_lsm_socket_connect"
)

// EventAction represents the security decision enforced by the eBPF LSM boundary.
type EventAction string

const (
	ActionAllow    EventAction = "ALLOW"
	ActionDeny     EventAction = "DENY"
	ActionThrottle EventAction = "THROTTLE"
	ActionHalt     EventAction = "HALT"
)

// SecurityEvent captures a kernel-level security telemetry event emitted by eBPF LSM probes.
type SecurityEvent struct {
	ID          string      `json:"id"`
	PID         uint32      `json:"pid"`
	PPID        uint32      `json:"ppid"`
	UID         uint32      `json:"uid"`
	GID         uint32      `json:"gid"`
	ProcessName string      `json:"process_name"`
	HookType    LSMHookType `json:"hook_type"`
	TargetPath  string      `json:"target_path,omitempty"`
	SocketAddr  string      `json:"socket_addr,omitempty"`
	Action      EventAction `json:"action"`
	Timestamp   time.Time   `json:"timestamp"`
	Blocked     bool        `json:"blocked"`
	Reason      string      `json:"reason,omitempty"`
}

var (
	ErrCircuitBreakerHalting = errors.New("mcpguard: circuit breaker tripped - eBPF event queue >= 85% capacity (fail-closed halt enforced)")
	ErrProbeNotLoaded        = errors.New("mcpguard: ebpf lsm probe is not loaded")
	ErrBufferOverflow        = errors.New("mcpguard: ring buffer overflow")
)

// CircuitBreakerState represents the current operating state of the fail-closed circuit breaker.
type CircuitBreakerState int32

const (
	StateNormal CircuitBreakerState = iota
	StateWarning
	StateHalting
)

func (s CircuitBreakerState) String() string {
	switch s {
	case StateNormal:
		return "NORMAL"
	case StateWarning:
		return "WARNING (>=70%)"
	case StateHalting:
		return "HALTING (>=85% FAIL-CLOSED)"
	default:
		return "UNKNOWN"
	}
}

// MCPGuardCircuitBreaker implements the ADR-12 Fail-Closed Backpressure Circuit Breaker.
// When the security event ring buffer exceeds 85% capacity, it synchronously halts/throttles
// system call allocations for calling sandbox processes until userland drains the buffer.
type MCPGuardCircuitBreaker struct {
	mu                  sync.RWMutex
	capacity            uint64
	highWatermark       float64 // Default: 0.85 (85%)
	lowWatermark        float64 // Default: 0.60 (60%)
	state               atomic.Int32
	throttledCount      atomic.Uint64
	totalProcessedCount atomic.Uint64
	haltCount           atomic.Uint64
	lastStateChange     time.Time
}

// NewMCPGuardCircuitBreaker constructs a circuit breaker with specified capacity and watermark thresholds.
func NewMCPGuardCircuitBreaker(capacity uint64, highWatermark, lowWatermark float64) *MCPGuardCircuitBreaker {
	if capacity == 0 {
		capacity = 10000
	}
	if highWatermark <= 0 || highWatermark > 1.0 {
		highWatermark = 0.85 // ADR-12 mandate
	}
	if lowWatermark <= 0 || lowWatermark >= highWatermark {
		lowWatermark = 0.60
	}

	cb := &MCPGuardCircuitBreaker{
		capacity:        capacity,
		highWatermark:   highWatermark,
		lowWatermark:    lowWatermark,
		lastStateChange: time.Now().UTC(),
	}
	cb.state.Store(int32(StateNormal))
	return cb
}

// EvaluateUtilization calculates current queue utilization and evaluates state transitions.
// Returns EventAction: ActionAllow if normal, ActionThrottle if warning, or ActionHalt if tripped (>=85%).
func (cb *MCPGuardCircuitBreaker) EvaluateUtilization(currentSize uint64) (EventAction, CircuitBreakerState) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.totalProcessedCount.Add(1)
	utilization := float64(currentSize) / float64(cb.capacity)
	prevState := CircuitBreakerState(cb.state.Load())
	newState := prevState

	if utilization >= cb.highWatermark {
		newState = StateHalting
	} else if utilization >= (cb.highWatermark - 0.15) {
		newState = StateWarning
	} else if utilization <= cb.lowWatermark {
		newState = StateNormal
	}

	if newState != prevState {
		cb.state.Store(int32(newState))
		cb.lastStateChange = time.Now().UTC()
	}

	switch newState {
	case StateHalting:
		cb.haltCount.Add(1)
		cb.throttledCount.Add(1)
		return ActionHalt, StateHalting
	case StateWarning:
		cb.throttledCount.Add(1)
		return ActionThrottle, StateWarning
	default:
		return ActionAllow, StateNormal
	}
}

// IsHalting returns true if the circuit breaker is currently in the fail-closed halting state.
func (cb *MCPGuardCircuitBreaker) IsHalting() bool {
	return CircuitBreakerState(cb.state.Load()) == StateHalting
}

// GetMetrics returns real-time circuit breaker metrics.
func (cb *MCPGuardCircuitBreaker) GetMetrics(currentSize uint64) map[string]interface{} {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	utilization := float64(currentSize) / float64(cb.capacity)
	return map[string]interface{}{
		"capacity":          cb.capacity,
		"current_size":      currentSize,
		"utilization_ratio": utilization,
		"high_watermark":    cb.highWatermark,
		"low_watermark":     cb.lowWatermark,
		"state":             CircuitBreakerState(cb.state.Load()).String(),
		"is_halting":        cb.IsHalting(),
		"throttled_events":  cb.throttledCount.Load(),
		"halted_events":     cb.haltCount.Load(),
		"total_events":      cb.totalProcessedCount.Load(),
		"last_state_change": cb.lastStateChange,
	}
}

// SecurityPolicy defines sandbox boundary permission rules for process, file, and socket access.
type SecurityPolicy struct {
	AllowedExecPaths    []string `json:"allowed_exec_paths"`
	DeniedExecPaths     []string `json:"denied_exec_paths"`
	AllowedFilePrefixes []string `json:"allowed_file_prefixes"`
	DeniedFilePrefixes  []string `json:"denied_file_prefixes"`
	AllowedNetworks     []string `json:"allowed_networks"`
	DeniedNetworks      []string `json:"denied_networks"`
}

// MCPGuardManager manages eBPF LSM probe lifecycle, ring-buffer event draining, and circuit breaker enforcement.
type MCPGuardManager struct {
	mu             sync.RWMutex
	policy         SecurityPolicy
	circuitBreaker *MCPGuardCircuitBreaker
	eventQueue     chan *SecurityEvent
	subscribers    []func(*SecurityEvent)
	active         atomic.Bool
	ctx            context.Context
	cancel         context.CancelFunc
	wg             sync.WaitGroup
}

// NewMCPGuardManager initializes the eBPF LSM security manager with specified policy and buffer capacity.
func NewMCPGuardManager(policy SecurityPolicy, bufferCapacity uint64) *MCPGuardManager {
	if bufferCapacity == 0 {
		bufferCapacity = 10000
	}

	ctx, cancel := context.WithCancel(context.Background())
	m := &MCPGuardManager{
		policy:         policy,
		circuitBreaker: NewMCPGuardCircuitBreaker(bufferCapacity, 0.85, 0.60),
		eventQueue:     make(chan *SecurityEvent, bufferCapacity),
		subscribers:    make([]func(*SecurityEvent), 0),
		ctx:            ctx,
		cancel:         cancel,
	}

	m.active.Store(true)
	m.wg.Add(1)
	go m.eventDrainLoop()

	return m
}

// Subscribe registers a userland callback for processed security events.
func (m *MCPGuardManager) Subscribe(fn func(*SecurityEvent)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.subscribers = append(m.subscribers, fn)
}

// EvaluateSyscall simulates kernel-level synchronous eBPF LSM hook execution.
// Evaluates security policy AND fail-closed circuit breaker capacity.
func (m *MCPGuardManager) EvaluateSyscall(hook LSMHookType, pid, uid uint32, processName, targetPath, socketAddr string) (*SecurityEvent, error) {
	if !m.active.Load() {
		return nil, ErrProbeNotLoaded
	}

	queueLen := uint64(len(m.eventQueue))
	cbAction, cbState := m.circuitBreaker.EvaluateUtilization(queueLen)

	evt := &SecurityEvent{
		ID:          fmt.Sprintf("ebpf-%d-%d", time.Now().UnixNano(), pid),
		PID:         pid,
		UID:         uid,
		ProcessName: processName,
		HookType:    hook,
		TargetPath:  targetPath,
		SocketAddr:  socketAddr,
		Timestamp:   time.Now().UTC(),
		Action:      ActionAllow,
		Blocked:     false,
	}

	// 1. Fail-Closed Circuit Breaker Enforcement (ADR-12)
	if cbAction == ActionHalt {
		evt.Action = ActionHalt
		evt.Blocked = true
		evt.Reason = fmt.Sprintf("Fail-closed circuit breaker tripped (%s queue >= 85%%)", cbState)

		m.enqueueEvent(evt)
		return evt, ErrCircuitBreakerHalting
	}

	// 2. Policy Enforcement
	allowed, reason := m.checkPolicy(hook, targetPath, socketAddr)
	if !allowed {
		evt.Action = ActionDeny
		evt.Blocked = true
		evt.Reason = reason
	} else if cbAction == ActionThrottle {
		evt.Action = ActionThrottle
		evt.Reason = "Security event queue under backpressure (>=70%)"
	}

	m.enqueueEvent(evt)
	return evt, nil
}

func (m *MCPGuardManager) checkPolicy(hook LSMHookType, targetPath, socketAddr string) (bool, string) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	switch hook {
	case LSMHookProcessExec:
		for _, denied := range m.policy.DeniedExecPaths {
			if targetPath == denied {
				return false, fmt.Sprintf("exec path '%s' explicitly denied by policy", targetPath)
			}
		}
		if len(m.policy.AllowedExecPaths) > 0 {
			for _, allowed := range m.policy.AllowedExecPaths {
				if targetPath == allowed {
					return true, ""
				}
			}
			return false, fmt.Sprintf("exec path '%s' not in allowed policy list", targetPath)
		}

	case LSMHookFileOpen:
		for _, denied := range m.policy.DeniedFilePrefixes {
			if len(targetPath) >= len(denied) && targetPath[:len(denied)] == denied {
				return false, fmt.Sprintf("file path '%s' matches denied prefix '%s'", targetPath, denied)
			}
		}

	case LSMHookSocketConnect:
		for _, denied := range m.policy.DeniedNetworks {
			if socketAddr == denied {
				return false, fmt.Sprintf("socket connect '%s' explicitly denied", socketAddr)
			}
		}
	}

	return true, ""
}

func (m *MCPGuardManager) enqueueEvent(evt *SecurityEvent) {
	select {
	case m.eventQueue <- evt:
	default:
		// Queue completely saturated - drop or count
	}
}

func (m *MCPGuardManager) eventDrainLoop() {
	defer m.wg.Done()

	for {
		select {
		case <-m.ctx.Done():
			return
		case evt := <-m.eventQueue:
			m.mu.RLock()
			subs := make([]func(*SecurityEvent), len(m.subscribers))
			copy(subs, m.subscribers)
			m.mu.RUnlock()

			for _, sub := range subs {
				sub(evt)
			}
		}
	}
}

// GetMetrics returns complete eBPF probe and circuit breaker telemetry.
func (m *MCPGuardManager) GetMetrics() map[string]interface{} {
	queueLen := uint64(len(m.eventQueue))
	metrics := m.circuitBreaker.GetMetrics(queueLen)
	metrics["active"] = m.active.Load()
	metrics["subscribers_count"] = len(m.subscribers)
	return metrics
}

// Close gracefully stops event processing and detaches eBPF probes.
func (m *MCPGuardManager) Close() {
	if m.active.CompareAndSwap(true, false) {
		m.cancel()
		m.wg.Wait()
		close(m.eventQueue)
	}
}
