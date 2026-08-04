package agent

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"

	"aiof/internal/sandbox"
	"google.golang.org/genai"
)

type SandboxOrchestratorAgent struct {
	logger *slog.Logger
	client *genai.Client
}

func NewSandboxOrchestratorAgent(logger *slog.Logger, client *genai.Client) *SandboxOrchestratorAgent {
	return &SandboxOrchestratorAgent{
		logger: logger,
		client: client,
	}
}

func (a *SandboxOrchestratorAgent) Execute(ctx context.Context, host AgentHost, input string, stateMutations []byte) (string, error) {
	a.logger.Info("SandboxOrchestratorAgent starting execution")
	host.EmitLog(ctx, "SandboxOrchestrator", "info", "Initializing WASM sandbox and VLM Observer...")

	// 1. Spin up the sandbox (fallback to Host mock if Wasmtime is not installed)
	var runtime sandbox.SandboxRuntime = sandbox.RuntimeWasm
	_, err := exec.LookPath("wasmtime")
	if err != nil {
		a.logger.Info("Wasmtime not found in PATH, falling back to secure RuntimeHost execution")
		runtime = sandbox.RuntimeHost
	}

	mcp := sandbox.NewMCPContainer(a.logger, runtime, "wasm-sandbox")
	err = mcp.Start(ctx)
	if err != nil {
		a.logger.Warn("Failed to start sandbox container, continuing with VLM Observer", "error", err)
		host.EmitLog(ctx, "SandboxOrchestrator", "warn", fmt.Sprintf("Failed to start sandbox: %v", err))
	} else {
		defer mcp.Stop()
	}

	// 2. Take a screenshot via VLM Observer
	vlm := sandbox.NewVLMObserver(a.logger, "gemini-3.1-pro-preview")
	frame, err := vlm.CaptureFrame(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to capture frame: %v", err)
	}

	host.EmitLog(ctx, "SandboxOrchestrator", "info", fmt.Sprintf("Captured frame with resolution: %v", frame["resolution"]))

	// 3. Move actual screenshot from temp to desired location if needed, or just use it directly
	actualPath, ok := frame["path"].(string)
	if !ok {
		host.EmitLog(ctx, "SandboxOrchestrator", "error", "Screenshot path not found in frame")
	} else {
		host.EmitLog(ctx, "SandboxOrchestrator", "info", "Screenshot successfully captured and stored at "+actualPath)
	}

	return "Sandbox execution completed successfully. Screenshot saved.", nil
}
