package plugin

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestParsePluginManifestYAML(t *testing.T) {
	manifestYAML := []byte(`
id: custom-code-parser
name: Custom Code Parser Plugin
version: 1.2.0
description: Hot-swappable Wasm code parser plugin
driver_type: wasm
entrypoint: ./bin/parser.wasm
time_slice_ms: 500
max_memory_bytes: 33554432
capabilities:
  filesystem:
    allowed_read_paths: ["/workspace", "/tmp"]
    allowed_write_paths: ["/tmp"]
  network:
    allowed_hosts: ["api.github.com"]
    allowed_ports: [443]
  ipc:
    allowed_channels: ["code.parsed"]
`)

	m, err := ParsePluginManifest(manifestYAML)
	if err != nil {
		t.Fatalf("failed to parse valid YAML manifest: %v", err)
	}

	if m.ID != "custom-code-parser" {
		t.Errorf("expected ID 'custom-code-parser', got '%s'", m.ID)
	}
	if m.DriverType != DriverTypeWasm {
		t.Errorf("expected driver type 'wasm', got '%s'", m.DriverType)
	}
	if m.TimeSliceMs != 500 {
		t.Errorf("expected time_slice_ms 500, got %d", m.TimeSliceMs)
	}

	// Test capability validation
	if err := m.ValidateFilesystemAccess("/workspace/main.go", false); err != nil {
		t.Errorf("expected allowed read access to /workspace, got: %v", err)
	}
	if err := m.ValidateFilesystemAccess("/etc/passwd", false); err == nil {
		t.Errorf("expected denied read access to /etc/passwd, got nil")
	}
	if err := m.ValidateNetworkAccess("api.github.com", 443); err != nil {
		t.Errorf("expected allowed network access to api.github.com:443, got: %v", err)
	}
	if err := m.ValidateNetworkAccess("malicious.com", 80); err == nil {
		t.Errorf("expected denied network access to malicious.com, got nil")
	}
}

func TestPluginManagerLifecycleAndHotReload(t *testing.T) {
	pm := NewPluginManager(nil)

	m1 := &PluginManifest{
		ID:          "router-plugin",
		Name:        "Router Plugin",
		Version:     "1.0.0",
		DriverType:  DriverTypeWasm,
		Entrypoint:  "./plugin.wasm",
		TimeSliceMs: 500,
	}

	// 1. Load Plugin
	if err := pm.LoadPlugin(m1); err != nil {
		t.Fatalf("failed to load plugin: %v", err)
	}

	// Verify status
	statuses := pm.ListPlugins()
	if len(statuses) != 1 || statuses[0].ID != "router-plugin" {
		t.Fatalf("expected 1 active router-plugin, got: %v", statuses)
	}

	// 2. Execute Action
	ctx := context.Background()
	res, err := pm.ExecutePlugin(ctx, "router-plugin", "route", []byte("prompt_bytes"))
	if err != nil {
		t.Fatalf("failed to execute plugin: %v", err)
	}
	if len(res) == 0 {
		t.Errorf("expected non-empty execution response")
	}

	// 3. Hot-Reload Plugin (Zero-Downtime Atomic Swap)
	m2 := &PluginManifest{
		ID:          "router-plugin",
		Name:        "Router Plugin Upgraded",
		Version:     "2.0.0",
		DriverType:  DriverTypeConnectRPC,
		Entrypoint:  "localhost:50051",
		TimeSliceMs: 1000,
	}

	if err := pm.HotReloadPlugin(m2); err != nil {
		t.Fatalf("failed to hot-reload plugin: %v", err)
	}

	// Verify atomic upgrade
	statusesAfter := pm.ListPlugins()
	if len(statusesAfter) != 1 || statusesAfter[0].Version != "2.0.0" || statusesAfter[0].DriverType != DriverTypeConnectRPC {
		t.Errorf("expected upgraded plugin v2.0.0 connect_rpc, got: %v", statusesAfter)
	}

	// Execute upgraded plugin
	res2, err := pm.ExecutePlugin(ctx, "router-plugin", "route", []byte("prompt_bytes"))
	if err != nil {
		t.Fatalf("failed to execute upgraded plugin: %v", err)
	}
	if len(res2) == 0 {
		t.Errorf("expected non-empty execution response from upgraded plugin")
	}

	// 4. Unload Plugin
	if err := pm.UnloadPlugin("router-plugin"); err != nil {
		t.Fatalf("failed to unload plugin: %v", err)
	}

	if _, err := pm.ExecutePlugin(ctx, "router-plugin", "route", nil); !errors.Is(err, ErrPluginNotFound) {
		t.Errorf("expected ErrPluginNotFound after unload, got: %v", err)
	}
}

func TestPluginAutoQuarantineOnFailures(t *testing.T) {
	pm := NewPluginManager(nil)

	m := &PluginManifest{
		ID:          "flaky-plugin",
		Name:        "Flaky Plugin",
		Version:     "1.0.0",
		DriverType:  DriverTypeWasm,
		Entrypoint:  "./flaky.wasm",
		TimeSliceMs: 1, // Ultra short time slice to force timeout
	}

	if err := pm.LoadPlugin(m); err != nil {
		t.Fatalf("failed to load plugin: %v", err)
	}

	ctx := context.Background()

	// Trigger repeated timeouts to breach MaxFailures threshold
	for i := 0; i < 3; i++ {
		time.Sleep(2 * time.Millisecond) // Ensure timeout
		_, _ = pm.ExecutePlugin(ctx, "flaky-plugin", "slow_action", nil)
	}

	// Verification: Plugin should now be quarantined
	_, err := pm.ExecutePlugin(ctx, "flaky-plugin", "slow_action", nil)
	if !errors.Is(err, ErrPluginQuarantined) {
		t.Errorf("expected ErrPluginQuarantined after 3 failures, got: %v", err)
	}
}
