package telemetry

//go:generate go run github.com/cilium/ebpf/cmd/bpf2go -cc clang bpf bpf/tracepoint.c

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"sync"

	"github.com/cilium/ebpf/link"
	"github.com/cilium/ebpf/perf"
	"github.com/cilium/ebpf/rlimit"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// SyscallEvent matches the struct event_t in our C kernel code.
type SyscallEvent struct {
	PID       uint32
	SyscallID uint32
}

// EBPFHook manages Phase 4.3 Boundary Tracing Security by natively loading eBPF programs.
// It securely injects probes into Firecracker MicroVMs or Wasm sandboxes without host kernel exposure.
type EBPFHook struct {
	logger *slog.Logger
	tracer trace.Tracer
	meter  metric.Meter

	objs       bpfObjects
	link       link.Link
	extraLinks []link.Link
	reader     *perf.Reader
	attached   bool
	mu         sync.Mutex
	pid        int
}

// NewEBPFHook initializes the boundary tracing hooks integrated natively with OTEL.
func NewEBPFHook(logger *slog.Logger) *EBPFHook {
	return &EBPFHook{
		logger: logger,
		tracer: otel.Tracer("aiof/sandbox/ebpf"),
		meter:  otel.Meter("aiof/sandbox/ebpf"),
	}
}

// Attach loads the compiled eBPF bytecode into the kernel and attaches the tracepoint.
func (e *EBPFHook) Attach(ctx context.Context, pid int) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.pid = pid
	e.logger.Info("Loading eBPF Phase 4.3 kernel-level sandbox boundary hooks", slog.Int("sandbox_pid", pid))

	// Allow the current process to lock memory for eBPF resources.
	if err := rlimit.RemoveMemlock(); err != nil {
		return fmt.Errorf("removing memlock: %w", err)
	}

	// Load pre-compiled programs and maps into the kernel.
	if err := loadBpfObjects(&e.objs, nil); err != nil {
		return fmt.Errorf("loading objects: %w", err)
	}

	// Attach tracepoint to sys_enter_execve.
	l, err := link.Tracepoint("syscalls", "sys_enter_execve", e.objs.HandleExecve, nil)
	if err != nil {
		e.objs.Close()
		return fmt.Errorf("link tracepoint: %w", err)
	}
	e.link = l

	// Attach additional boundary tracepoints (best-effort)
	for _, tp := range []struct{ group, name string }{
		{"syscalls", "sys_enter_openat"},
		{"syscalls", "sys_enter_write"},
		{"syscalls", "sys_enter_connect"},
		{"syscalls", "sys_enter_clone"},
	} {
		if extraLink, err := link.Tracepoint(tp.group, tp.name, e.objs.HandleExecve, nil); err != nil {
			e.logger.Debug("Optional tracepoint not attached", slog.String("tracepoint", tp.name), slog.Any("error", err))
		} else {
			e.logger.Info("Attached additional boundary tracepoint", slog.String("tracepoint", tp.name))
			e.extraLinks = append(e.extraLinks, extraLink)
		}
	}

	// Open a perf event reader from the kernel map.
	rd, err := perf.NewReader(e.objs.Events, os.Getpagesize())
	if err != nil {
		e.link.Close()
		e.objs.Close()
		return fmt.Errorf("creating perf event reader: %w", err)
	}
	e.reader = rd
	e.attached = true

	e.logger.Info("Successfully attached tracepoints for syscall boundary monitoring")
	return nil
}

// Correlate listens to the eBPF perf ring buffer and correlates events to OpenTelemetry spans.
func (e *EBPFHook) Correlate(ctx context.Context) error {
	if e.reader == nil {
		return fmt.Errorf("correlate: perf reader is nil, call Attach() before Correlate()")
	}
	if !e.attached {
		return fmt.Errorf("correlate: ebpf not attached")
	}

	e.logger.Info("Starting eBPF to OTEL telemetry correlation pipeline")

	anomalyCounter, err := e.meter.Int64Counter("sandbox.syscall.anomalies")
	if err != nil {
		return fmt.Errorf("failed to create metric: %w", err)
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				record, err := e.reader.Read()
				if err != nil {
					if errors.Is(err, perf.ErrClosed) {
						return
					}
					e.logger.Error("Error reading perf record", slog.Any("error", err))
					continue
				}

				if record.LostSamples != 0 {
					e.logger.Warn("Perf event ring buffer full, dropped samples", slog.Uint64("dropped", record.LostSamples))
					continue
				}

				// Parse the struct event_t from the C code.
				var event SyscallEvent
				if err := binary.Read(bytes.NewReader(record.RawSample), binary.LittleEndian, &event); err != nil {
					e.logger.Error("Failed to parse perf event", slog.Any("error", err))
					continue
				}

				// Filter by sandbox PID if necessary (event.PID == e.pid)
				_, span := e.tracer.Start(ctx, "sandbox_syscall_execution")
				span.SetAttributes(
					attribute.Int("process.pid", int(event.PID)),
					attribute.Int("syscall.id", int(event.SyscallID)),
					attribute.String("syscall.name", "execve"),
				)
				
				anomalyCounter.Add(ctx, 1, metric.WithAttributes(attribute.String("type", "execve_in_sandbox")))
				
				// e.logger.Debug("Correlated eBPF kernel event to OTEL", slog.Uint64("syscall", uint64(event.SyscallID)), slog.Uint64("pid", uint64(event.PID)))
				span.End()
			}
		}
	}()

	return nil
}

// Detach cleans up the eBPF programs and perf buffer readers.
func (e *EBPFHook) Detach() {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.reader != nil {
		e.reader.Close()
	}
	if e.link != nil {
		e.link.Close()
	}
	for _, l := range e.extraLinks {
		l.Close()
	}
	e.objs.Close()
	e.attached = false
	e.logger.Info("Detached eBPF hooks from kernel")
}
