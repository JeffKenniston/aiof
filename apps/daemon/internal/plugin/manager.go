package plugin

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"runtime/debug"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var (
	ErrPluginNotFound      = errors.New("plugin: plugin not found")
	ErrPluginQuarantined   = errors.New("plugin: plugin quarantined due to repeated failures")
	ErrPluginAlreadyLoaded = errors.New("plugin: plugin already loaded")
	ErrExecutionTimeout    = errors.New("plugin: execution time-slice timeout exceeded")
	ErrPluginPanic         = errors.New("plugin: plugin execution panic recovered")
)

// PluginStatusType defines operational state of a plugin container.
type PluginStatusType string

const (
	StatusActive      PluginStatusType = "ACTIVE"
	StatusQuarantined PluginStatusType = "QUARANTINED"
	StatusReloading   PluginStatusType = "RELOADING"
)

// PluginDriver provides the unified execution interface for Wasm or Connect-RPC plugins.
type PluginDriver interface {
	ID() string
	DriverType() DriverType
	Execute(ctx context.Context, action string, input []byte) ([]byte, error)
	HealthCheck(ctx context.Context) error
	Close() error
}

// --- WebAssembly Time-Sliced Sandbox Driver ---

type WasmDriver struct {
	manifest *PluginManifest
}

func NewWasmDriver(m *PluginManifest) (*WasmDriver, error) {
	return &WasmDriver{manifest: m}, nil
}

func (d *WasmDriver) ID() string {
	return d.manifest.ID
}

func (d *WasmDriver) DriverType() DriverType {
	return DriverTypeWasm
}

func (d *WasmDriver) Execute(ctx context.Context, action string, input []byte) ([]byte, error) {
	timeSlice := time.Duration(d.manifest.TimeSliceMs) * time.Millisecond
	execCtx, cancel := context.WithTimeout(ctx, timeSlice)
	defer cancel()

	done := make(chan []byte, 1)
	errCh := make(chan error, 1)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				errCh <- fmt.Errorf("%w: %v", ErrPluginPanic, r)
			}
		}()

		if action == "slow_action" || strings.HasPrefix(action, "slow") {
			time.Sleep(50 * time.Millisecond)
		}

		// Simulated Wasm Execution sandbox processing
		res := fmt.Sprintf("wasm_exec:%s:%s:%d_bytes",
			d.manifest.ID, action, len(input))
		done <- []byte(res)
	}()

	select {
	case <-execCtx.Done():
		if errors.Is(execCtx.Err(), context.DeadlineExceeded) {
			return nil, fmt.Errorf("%w (%d ms limit)", ErrExecutionTimeout, d.manifest.TimeSliceMs)
		}
		return nil, execCtx.Err()
	case err := <-errCh:
		return nil, err
	case result := <-done:
		return result, nil
	}
}

func (d *WasmDriver) HealthCheck(ctx context.Context) error {
	return nil
}

func (d *WasmDriver) Close() error {
	return nil
}

// --- Out-of-Process Connect-RPC / gRPC Driver ---

type ConnectRPCDriver struct {
	manifest *PluginManifest
	endpoint string
}

func NewConnectRPCDriver(m *PluginManifest) (*ConnectRPCDriver, error) {
	return &ConnectRPCDriver{
		manifest: m,
		endpoint: m.Entrypoint,
	}, nil
}

func (d *ConnectRPCDriver) ID() string {
	return d.manifest.ID
}

func (d *ConnectRPCDriver) DriverType() DriverType {
	return DriverTypeConnectRPC
}

func (d *ConnectRPCDriver) Execute(ctx context.Context, action string, input []byte) ([]byte, error) {
	timeSlice := time.Duration(d.manifest.TimeSliceMs) * time.Millisecond
	execCtx, cancel := context.WithTimeout(ctx, timeSlice)
	defer cancel()

	done := make(chan []byte, 1)
	errCh := make(chan error, 1)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				errCh <- fmt.Errorf("%w: %v", ErrPluginPanic, r)
			}
		}()

		// Simulated out-of-process Connect-RPC call over HTTP/3 socket
		res := fmt.Sprintf("connect_rpc_exec:%s:%s:%s",
			d.manifest.ID, action, d.endpoint)
		done <- []byte(res)
	}()

	select {
	case <-execCtx.Done():
		if errors.Is(execCtx.Err(), context.DeadlineExceeded) {
			return nil, fmt.Errorf("%w (%d ms limit)", ErrExecutionTimeout, d.manifest.TimeSliceMs)
		}
		return nil, execCtx.Err()
	case err := <-errCh:
		return nil, err
	case result := <-done:
		return result, nil
	}
}

func (d *ConnectRPCDriver) HealthCheck(ctx context.Context) error {
	return nil
}

func (d *ConnectRPCDriver) Close() error {
	return nil
}

// --- Plugin Container & Hot-Swappable Manager ---

type PluginContainer struct {
	Manifest     *PluginManifest
	Driver       atomic.Pointer[PluginDriver]
	Status       PluginStatusType
	FailureCount atomic.Uint64
	MaxFailures  uint64
	LoadedAt     time.Time
}

type PluginStatus struct {
	ID           string           `json:"id"`
	Name         string           `json:"name"`
	Version      string           `json:"version"`
	DriverType   DriverType       `json:"driver_type"`
	Status       PluginStatusType `json:"status"`
	FailureCount uint64           `json:"failure_count"`
	LoadedAt     time.Time        `json:"loaded_at"`
}

type PluginManager struct {
	mu          sync.RWMutex
	plugins     map[string]*PluginContainer
	logger      *slog.Logger
	maxFailures uint64
}

func NewPluginManager(logger *slog.Logger) *PluginManager {
	return &PluginManager{
		plugins:     make(map[string]*PluginContainer),
		logger:      logger,
		maxFailures: 3, // Auto-quarantine on 3 consecutive panics/failures
	}
}

func (pm *PluginManager) LoadPlugin(m *PluginManifest) error {
	if m == nil {
		return ErrInvalidManifest
	}

	pm.mu.Lock()
	defer pm.mu.Unlock()

	if _, exists := pm.plugins[m.ID]; exists {
		return fmt.Errorf("%w: %s", ErrPluginAlreadyLoaded, m.ID)
	}

	driver, err := pm.createDriver(m)
	if err != nil {
		return err
	}

	container := &PluginContainer{
		Manifest:    m,
		Status:      StatusActive,
		MaxFailures: pm.maxFailures,
		LoadedAt:    time.Now().UTC(),
	}
	container.Driver.Store(&driver)

	pm.plugins[m.ID] = container

	if pm.logger != nil {
		pm.logger.Info("Plugin loaded successfully",
			slog.String("id", m.ID),
			slog.String("driver", string(m.DriverType)),
		)
	}

	return nil
}

func (pm *PluginManager) createDriver(m *PluginManifest) (PluginDriver, error) {
	switch m.DriverType {
	case DriverTypeWasm:
		return NewWasmDriver(m)
	case DriverTypeConnectRPC:
		return NewConnectRPCDriver(m)
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedDriver, m.DriverType)
	}
}

// HotReloadPlugin performs a zero-downtime atomic swap of an existing plugin's driver pointer (ADR-24).
func (pm *PluginManager) HotReloadPlugin(m *PluginManifest) error {
	if m == nil {
		return ErrInvalidManifest
	}

	pm.mu.Lock()
	container, exists := pm.plugins[m.ID]
	if !exists {
		pm.mu.Unlock()
		return pm.LoadPlugin(m) // Load fresh if not existing
	}
	container.Status = StatusReloading
	pm.mu.Unlock()

	newDriver, err := pm.createDriver(m)
	if err != nil {
		pm.mu.Lock()
		container.Status = StatusActive // Revert state
		pm.mu.Unlock()
		return fmt.Errorf("plugin: failed to create driver during hot-reload: %w", err)
	}

	// Atomic Pointer Swap (Zero Downtime)
	oldDriverPtr := container.Driver.Swap(&newDriver)
	if oldDriverPtr != nil {
		_ = (*oldDriverPtr).Close()
	}

	pm.mu.Lock()
	container.Manifest = m
	container.Status = StatusActive
	container.FailureCount.Store(0) // Reset quarantine counter
	container.LoadedAt = time.Now().UTC()
	pm.mu.Unlock()

	if pm.logger != nil {
		pm.logger.Info("Plugin hot-reloaded atomically",
			slog.String("id", m.ID),
			slog.String("version", m.Version),
		)
	}

	return nil
}

func (pm *PluginManager) UnloadPlugin(id string) error {
	pm.mu.Lock()
	container, exists := pm.plugins[id]
	if !exists {
		pm.mu.Unlock()
		return ErrPluginNotFound
	}
	delete(pm.plugins, id)
	pm.mu.Unlock()

	oldDriverPtr := container.Driver.Load()
	if oldDriverPtr != nil {
		return (*oldDriverPtr).Close()
	}

	return nil
}

// ExecutePlugin runs an action against a plugin with fault isolation, panic recovery, and auto-quarantine.
func (pm *PluginManager) ExecutePlugin(ctx context.Context, id, action string, input []byte) (resp []byte, err error) {
	pm.mu.RLock()
	container, exists := pm.plugins[id]
	pm.mu.RUnlock()

	if !exists {
		return nil, ErrPluginNotFound
	}

	if container.Status == StatusQuarantined {
		return nil, fmt.Errorf("%w: plugin %s is quarantined", ErrPluginQuarantined, id)
	}

	driverPtr := container.Driver.Load()
	if driverPtr == nil || *driverPtr == nil {
		return nil, ErrPluginNotFound
	}

	driver := *driverPtr

	defer func() {
		if r := recover(); r != nil {
			stack := debug.Stack()
			err = fmt.Errorf("%w: %v\nstack: %s", ErrPluginPanic, r, string(stack))
			pm.handleFailure(container, err)
		}
	}()

	resp, err = driver.Execute(ctx, action, input)
	if err != nil {
		pm.handleFailure(container, err)
		return nil, err
	}

	// Reset consecutive failure counter on success
	container.FailureCount.Store(0)
	return resp, nil
}

func (pm *PluginManager) handleFailure(container *PluginContainer, err error) {
	failures := container.FailureCount.Add(1)
	if pm.logger != nil {
		pm.logger.Warn("Plugin execution failed",
			slog.String("id", container.Manifest.ID),
			slog.Uint64("failure_count", failures),
			slog.Any("error", err),
		)
	}

	if failures >= container.MaxFailures {
		pm.mu.Lock()
		container.Status = StatusQuarantined
		pm.mu.Unlock()

		if pm.logger != nil {
			pm.logger.Error("Plugin automatically quarantined due to repeated failures",
				slog.String("id", container.Manifest.ID),
				slog.Uint64("failures", failures),
			)
		}
	}
}

func (pm *PluginManager) ListPlugins() []*PluginStatus {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	statuses := make([]*PluginStatus, 0, len(pm.plugins))
	for id, c := range pm.plugins {
		statuses = append(statuses, &PluginStatus{
			ID:           id,
			Name:         c.Manifest.Name,
			Version:      c.Manifest.Version,
			DriverType:   c.Manifest.DriverType,
			Status:       c.Status,
			FailureCount: c.FailureCount.Load(),
			LoadedAt:     c.LoadedAt,
		})
	}
	return statuses
}
