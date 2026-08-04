package sandbox

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"aiof/internal/telemetry"
)

// MCPRequest represents a Model Context Protocol JSON-RPC request.
type MCPRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int         `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

// MCPResponse represents a JSON-RPC response.
type MCPResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *MCPError       `json:"error,omitempty"`
}

type MCPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// MCPContainer manages Phase 4.1 & 4.2 Sandbox Containerization.
// It connects to an isolated Firecracker/Wasm environment via standard stdio MCP transport.
type SandboxRuntime string

const (
	RuntimeFirecracker SandboxRuntime = "firecracker"
	RuntimeWasm        SandboxRuntime = "wasm"
	RuntimeHost        SandboxRuntime = "host"
	RuntimeNodeMCP     SandboxRuntime = "node_mcp"
)

type MCPContainer struct {
	logger     *slog.Logger
	sandbox    string // Identifier for the isolated environment (e.g. rootfs or wasm module)
	runtime    SandboxRuntime
	cmd        *exec.Cmd
	stdin      io.WriteCloser
	stdout     io.ReadCloser
	mu         sync.Mutex
	writeMu    sync.Mutex
	reqID      int64
	pending    map[int64]chan *MCPResponse
	pendingMu  sync.Mutex
	cancelRead context.CancelFunc
	
	startTime  time.Time
	lastCPU    float64
	lastCheck  time.Time
}

func (m *MCPContainer) GetPID() int {
	if m.cmd != nil && m.cmd.Process != nil {
		return m.cmd.Process.Pid
	}
	return 0
}

func (m *MCPContainer) GetStats() (cpuPercent float64, memMb int, uptimeSec int) {
	pid := m.GetPID()
	if pid == 0 {
		return 0, 0, 0
	}
	
	uptimeSec = int(time.Since(m.startTime).Seconds())
	
	// Read Memory
	if data, err := os.ReadFile(fmt.Sprintf("/proc/%d/statm", pid)); err == nil {
		var size, resident int
		fmt.Sscanf(string(data), "%d %d", &size, &resident)
		memMb = (resident * 4) / 1024 // Pages to MB (assuming 4KB pages)
	}

	// Read CPU
	if data, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid)); err == nil {
		fields := strings.Fields(string(data))
		if len(fields) > 14 {
			utime, _ := strconv.ParseFloat(fields[13], 64)
			stime, _ := strconv.ParseFloat(fields[14], 64)
			totalTicks := utime + stime
			
			now := time.Now()
			if !m.lastCheck.IsZero() {
				timeDelta := now.Sub(m.lastCheck).Seconds()
				tickDelta := totalTicks - m.lastCPU
				
				// CPU % = (ticks / USER_HZ) / timeDelta * 100
				// Typically USER_HZ is 100 on Linux
				cpuPercent = (tickDelta / 100.0) / timeDelta * 100.0
			}
			
			m.lastCPU = totalTicks
			m.lastCheck = now
		}
	}
	
	return cpuPercent, memMb, uptimeSec
}

// NewMCPContainer initializes a new MCP process manager.
func NewMCPContainer(logger *slog.Logger, runtime SandboxRuntime, sandbox string) *MCPContainer {
	if sandbox == "" {
		sandbox = "mcp-agent-eval-env-v1" // Default
	}
	return &MCPContainer{
		logger:  logger,
		sandbox: sandbox,
		runtime: runtime,
	}
}

// Start launches the sandbox container process and binds stdio for the MCP JSON-RPC protocol.
func (m *MCPContainer) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.logger.Info("Starting MCP Sandbox process", slog.String("runtime", string(m.runtime)), slog.String("target", m.sandbox))
	
	m.startTime = time.Now()
	m.pending = make(map[int64]chan *MCPResponse)

	if m.runtime == RuntimeWasm {
		// 4.2 WebAssembly (Wasm) Runtimes
		m.cmd = exec.CommandContext(ctx, "wasmtime", "run", m.sandbox)
	} else if m.runtime == RuntimeFirecracker {
		// 4.1 Firecracker MicroVM Provisioning
		m.cmd = exec.CommandContext(ctx, "firectl", 
			"--kernel=/var/lib/firecracker/vmlinux", 
			"--root-drive=" + m.sandbox, 
			"--firecracker-binary=/usr/local/bin/firecracker")
	} else if m.runtime == RuntimeNodeMCP {
		m.cmd = exec.CommandContext(ctx, "node", "dist/index.js")
		if m.sandbox != "" {
			m.cmd.Dir = m.sandbox
		}
	} else if m.runtime == RuntimeHost {
		pyExec := "python3"
		if os.Getenv("OS") == "Windows_NT" || strings.Contains(strings.ToLower(os.Getenv("OS")), "windows") {
			pyExec = "python"
		}
		m.cmd = exec.CommandContext(ctx, pyExec, "-c", `
import sys, json, subprocess, shlex
for line in sys.stdin:
    if not line.strip(): continue
    try:
        req = json.loads(line)
        cmd = req['params']['code']
        args = shlex.split(cmd)
        try:
            # Secure: shell=False prevents shell injection, timeout prevents hanging
            out = subprocess.check_output(args, stderr=subprocess.STDOUT, timeout=60, shell=False)
            res = {"jsonrpc": "2.0", "id": req["id"], "result": out.decode('utf-8')}
        except subprocess.CalledProcessError as e:
            res = {"jsonrpc": "2.0", "id": req["id"], "error": {"code": -32603, "message": e.output.decode('utf-8')}}
        except subprocess.TimeoutExpired as e:
            res = {"jsonrpc": "2.0", "id": req["id"], "error": {"code": -32603, "message": "Execution timed out"}}
    except Exception as e:
        res = {"jsonrpc": "2.0", "id": req.get("id", 1) if isinstance(req, dict) else 1, "error": {"code": -32700, "message": str(e)}}
    sys.stdout.write(json.dumps(res) + "\n")
    sys.stdout.flush()
`)
		if m.sandbox != "" {
			os.MkdirAll(m.sandbox, 0755)
			m.cmd.Dir = m.sandbox // use sandbox field as project directory for host runtime
		}
	} else {
		return fmt.Errorf("unsupported sandbox runtime")
	}
	
	stdin, err := m.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to attach stdin: %w", err)
	}
	m.stdin = stdin

	stdout, err := m.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to attach stdout: %w", err)
	}
	m.stdout = stdout

	if err := m.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start mcp container: %w", err)
	}

	// Phase 4.1: Sandbox eBPF Boundary Tracing (OTEL integration)
	if m.cmd.Process != nil {
		ebpfHook := telemetry.NewEBPFHook(m.logger)
		if err := ebpfHook.Attach(ctx, m.cmd.Process.Pid); err != nil {
			m.logger.Warn("Failed to attach eBPF tracing hook to sandbox", slog.Any("error", err), slog.Int("pid", m.cmd.Process.Pid))
		} else {
			m.logger.Info("eBPF tracing hook attached successfully to sandbox", slog.Int("pid", m.cmd.Process.Pid))
			// Start OpenTelemetry forwarding
			go ebpfHook.Correlate(ctx)
		}
	}

	readCtx, cancel := context.WithCancel(context.Background())
	m.cancelRead = cancel
	go m.readLoop(readCtx)

	return nil
}

func (m *MCPContainer) readLoop(ctx context.Context) {
	scanner := bufio.NewScanner(m.stdout)
	for scanner.Scan() {
		line := scanner.Bytes()
		var res MCPResponse
		if err := json.Unmarshal(line, &res); err != nil {
			m.logger.Error("failed to unmarshal mcp response", slog.Any("error", err))
			continue
		}

		m.pendingMu.Lock()
		ch, ok := m.pending[int64(res.ID)]
		if ok {
			delete(m.pending, int64(res.ID))
		}
		m.pendingMu.Unlock()

		if ok {
			ch <- &res
		}
	}

	// On scanner error or EOF, close all pending channels and return
	m.pendingMu.Lock()
	for id, ch := range m.pending {
		close(ch)
		delete(m.pending, id)
	}
	m.pendingMu.Unlock()
}

// Stop gracefully terminates the sandbox container.
func (m *MCPContainer) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.cancelRead != nil {
		m.cancelRead()
	}

	if m.stdin != nil {
		m.stdin.Close()
	}
	
	var killErr error
	if m.cmd != nil && m.cmd.Process != nil {
		killErr = m.cmd.Process.Kill()
	}
	
	m.pendingMu.Lock()
	for id, ch := range m.pending {
		close(ch)
		delete(m.pending, id)
	}
	m.pendingMu.Unlock()

	return killErr
}

// ExecuteCode routes the generated LLM outputs securely into the MCP transport layer over stdio.
func (m *MCPContainer) ExecuteCode(ctx context.Context, codePayload []byte) (string, error) {
	if len(codePayload) == 0 {
		return "", fmt.Errorf("empty code payload prevents MCP execution")
	}

	m.mu.Lock()
	m.reqID++
	id := m.reqID
	m.mu.Unlock()

	req := MCPRequest{
		JSONRPC: "2.0",
		ID:      int(id),
		Method:  "execute_code",
		Params: map[string]interface{}{
			"code": string(codePayload),
		},
	}

	reqData, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to encode mcp request: %w", err)
	}

	m.logger.Info("Routing codebase outputs to deterministic MCP sandbox", slog.String("sandbox_id", m.sandbox), slog.Int("rpc_id", int(id)))

	m.pendingMu.Lock()
	ch := make(chan *MCPResponse, 1)
	m.pending[id] = ch
	m.pendingMu.Unlock()

	m.writeMu.Lock()
	_, err = m.stdin.Write(append(reqData, '\n'))
	m.writeMu.Unlock()

	if err != nil {
		m.pendingMu.Lock()
		delete(m.pending, id)
		m.pendingMu.Unlock()
		return "", fmt.Errorf("failed to write to mcp stdio: %w", err)
	}

	select {
	case res, ok := <-ch:
		if !ok {
			return "", fmt.Errorf("mcp connection closed")
		}
		if res.Error != nil {
			return "", fmt.Errorf("mcp execution error (code: %d): %s", res.Error.Code, res.Error.Message)
		}
		return string(res.Result), nil
	case <-ctx.Done():
		m.pendingMu.Lock()
		delete(m.pending, id)
		m.pendingMu.Unlock()
		return "", fmt.Errorf("mcp read timed out: %w", ctx.Err())
	}
}

// CallTool routes an arbitrary MCP standard tool call.
func (m *MCPContainer) CallTool(ctx context.Context, name string, args map[string]interface{}) (json.RawMessage, error) {
	m.mu.Lock()
	m.reqID++
	id := m.reqID
	m.mu.Unlock()

	req := MCPRequest{
		JSONRPC: "2.0",
		ID:      int(id),
		Method:  "tools/call",
		Params: map[string]interface{}{
			"name": name,
			"arguments": args,
		},
	}

	reqData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to encode mcp request: %w", err)
	}

	m.pendingMu.Lock()
	ch := make(chan *MCPResponse, 1)
	m.pending[id] = ch
	m.pendingMu.Unlock()

	m.writeMu.Lock()
	_, err = m.stdin.Write(append(reqData, '\n'))
	m.writeMu.Unlock()

	if err != nil {
		m.pendingMu.Lock()
		delete(m.pending, id)
		m.pendingMu.Unlock()
		return nil, fmt.Errorf("failed to write to mcp stdio: %w", err)
	}

	select {
	case res, ok := <-ch:
		if !ok {
			return nil, fmt.Errorf("mcp connection closed")
		}
		if res.Error != nil {
			return nil, fmt.Errorf("mcp execution error (code: %d): %s", res.Error.Code, res.Error.Message)
		}
		return res.Result, nil
	case <-ctx.Done():
		m.pendingMu.Lock()
		delete(m.pending, id)
		m.pendingMu.Unlock()
		return nil, fmt.Errorf("mcp read timed out: %w", ctx.Err())
	}
}
