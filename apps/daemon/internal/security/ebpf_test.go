package security

import (
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestCircuitBreakerWatermarks(t *testing.T) {
	cb := NewMCPGuardCircuitBreaker(100, 0.85, 0.60)

	// Normal state (<70%)
	action, state := cb.EvaluateUtilization(50)
	if action != ActionAllow || state != StateNormal {
		t.Errorf("Expected ActionAllow & StateNormal at 50%% utilization, got %s & %s", action, state)
	}

	// Warning state (>=70% and <85%)
	action, state = cb.EvaluateUtilization(75)
	if action != ActionThrottle || state != StateWarning {
		t.Errorf("Expected ActionThrottle & StateWarning at 75%% utilization, got %s & %s", action, state)
	}

	// Halting state (>=85% - ADR-12 Fail-Closed)
	action, state = cb.EvaluateUtilization(88)
	if action != ActionHalt || state != StateHalting {
		t.Errorf("Expected ActionHalt & StateHalting at 88%% utilization, got %s & %s", action, state)
	}

	if !cb.IsHalting() {
		t.Errorf("Expected cb.IsHalting() to be true")
	}

	// Recovery back to normal (<60%)
	action, state = cb.EvaluateUtilization(55)
	if action != ActionAllow || state != StateNormal {
		t.Errorf("Expected ActionAllow & StateNormal after recovery to 55%%, got %s & %s", action, state)
	}
}

func TestMCPGuardManagerPolicyEnforcement(t *testing.T) {
	policy := SecurityPolicy{
		AllowedExecPaths:   []string{"/usr/bin/python3", "/usr/bin/go"},
		DeniedExecPaths:    []string{"/bin/nc", "/usr/bin/nmap"},
		DeniedFilePrefixes: []string{"/etc/shadow", "/etc/sudoers"},
		DeniedNetworks:     []string{"10.0.0.1:22", "192.168.1.1:80"},
	}

	mgr := NewMCPGuardManager(policy, 1000)
	defer mgr.Close()

	var eventCount atomic.Uint64
	mgr.Subscribe(func(evt *SecurityEvent) {
		eventCount.Add(1)
	})

	// 1. Allowed Exec Path
	evt, err := mgr.EvaluateSyscall(LSMHookProcessExec, 101, 1000, "python3", "/usr/bin/python3", "")
	if err != nil || evt.Blocked {
		t.Errorf("Expected allowed process execution, got blocked: %v, reason: %s", err, evt.Reason)
	}

	// 2. Denied Exec Path
	evt, err = mgr.EvaluateSyscall(LSMHookProcessExec, 102, 1000, "nc", "/bin/nc", "")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if !evt.Blocked || evt.Action != ActionDeny {
		t.Errorf("Expected denied execution for /bin/nc, got action %s", evt.Action)
	}

	// 3. Denied File Path
	evt, err = mgr.EvaluateSyscall(LSMHookFileOpen, 103, 1000, "cat", "/etc/shadow", "")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if !evt.Blocked || evt.Action != ActionDeny {
		t.Errorf("Expected denied file open for /etc/shadow, got action %s", evt.Action)
	}

	// 4. Denied Network Socket
	evt, err = mgr.EvaluateSyscall(LSMHookSocketConnect, 104, 1000, "ssh", "", "10.0.0.1:22")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if !evt.Blocked || evt.Action != ActionDeny {
		t.Errorf("Expected denied socket connect for 10.0.0.1:22, got action %s", evt.Action)
	}

	time.Sleep(100 * time.Millisecond)
	if eventCount.Load() != 4 {
		t.Errorf("Expected 4 subscriber events received, got %d", eventCount.Load())
	}
}

func TestFailClosedCircuitBreakerTripping(t *testing.T) {
	policy := SecurityPolicy{}
	// Small buffer capacity = 10 events
	mgr := NewMCPGuardManager(policy, 10)
	defer mgr.Close()

	// Fill queue to 9 items (90% >= 85% high watermark)
	for i := 0; i < 9; i++ {
		mgr.eventQueue <- &SecurityEvent{
			ID:          "dummy",
			PID:         uint32(i + 1),
			HookType:    LSMHookProcessExec,
			ProcessName: "test",
		}
	}

	// Next syscall evaluation should trigger ActionHalt and return ErrCircuitBreakerHalting
	evt, err := mgr.EvaluateSyscall(LSMHookProcessExec, 999, 1000, "worker", "/usr/bin/app", "")
	if !errors.Is(err, ErrCircuitBreakerHalting) {
		t.Errorf("Expected ErrCircuitBreakerHalting when queue >= 85%%, got %v", err)
	}
	if evt == nil || evt.Action != ActionHalt || !evt.Blocked {
		t.Errorf("Expected evt.Action == ActionHalt and Blocked == true")
	}

	metrics := mgr.GetMetrics()
	if metrics["is_halting"] != true {
		t.Errorf("Expected metrics is_halting == true")
	}
}
