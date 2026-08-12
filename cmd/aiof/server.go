package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"golang.org/x/time/rate"

	"aiof/internal/api"
	"aiof/internal/gen/orchestrator/v1/orchestratorv1connect"
	"aiof/internal/ingestion"
	"aiof/internal/llm"
	"aiof/internal/orchestrator"
	"aiof/internal/store"
	"aiof/internal/telemetry"

	"github.com/spf13/cobra"
)

var serverCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the AIOF Orchestrator daemon",
	Run:   runServer,
}

func runServer(cmd *cobra.Command, args []string) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	slog.SetDefault(logger)

	shutdownCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	ctx := shutdownCtx

	// 1. Load settings
	geminiKey := os.Getenv("GEMINI_API_KEY")
	dbURL := os.Getenv("DATABASE_URL")
	if geminiKey == "" {
		logger.Warn("GEMINI_API_KEY is not set. Please set it in your environment to test generative routing.")
	}
	if dbURL == "" {
		logger.Warn("DATABASE_URL is not set. Bypassing persistence mode.")
	}

	// 2. Initialize Persistence Layer
	var storeDB *store.Store
	var err error
	if dbURL != "" {
		storeDB, err = store.NewStore(ctx, dbURL)
		if err != nil {
			logger.Error("FATAL: Failed to initialize Postgres store. Enterprise persistence mandate requires WAL.", slog.Any("error", err))
			os.Exit(1) // Crash on startup instead of allowing mocked ephemeral state
		} else {
			defer storeDB.Close()
		}
	}

	// 3. Initialize Cloud Core Integration
	var geminiClient *llm.GeminiClient
	if geminiKey != "" {
		geminiClient, err = llm.NewGeminiClient(ctx, geminiKey)
		if err != nil {
			logger.Error("Failed to initialize Gemini Client", slog.Any("error", err))
		} else {
			defer geminiClient.Close()
			logger.Info("Gemini API Client initialized with Context Caching support")
		}
	}

	// 4. Initialize Orchestration & Connect RPC (Phase 1.1 & 1.3)
	rpcServer := orchestrator.NewRPCServer(logger, storeDB)
	
	// Initialize eBPF Boundary Tracing
	ebpfHook := telemetry.NewEBPFHook(logger)
	if err := ebpfHook.Attach(ctx, os.Getpid()); err != nil {
		logger.Warn("Failed to attach eBPF tracing hook (may require root)", slog.Any("error", err))
	} else {
		defer ebpfHook.Detach()
		go ebpfHook.Correlate(ctx)
	}
	// 3.5 Initialize Local Bare-Metal Cluster (Phase 6)
	var sgClient *llm.SGLangClient
	socketPath := "/tmp/sglang.sock"
	if os.Getenv("ENABLE_SGLANG") == "true" || socketExists(socketPath) {
		if socketExists(socketPath) {
			sgConfig := llm.SGLangConfig{
				SocketPath:          socketPath,
				EnablePrefillDecode: true,
				MooncakeTransfer:    true,
				LMCacheGlobalPool:   "redis://localhost:6379/1",
				TurboQuantBits:      3, // 3-Bit Cache Compression
			}
			sgClient = llm.NewSGLangClient(logger, sgConfig)
			logger.Info("SGLang local inference cluster enabled", slog.String("socket", socketPath))
		} else {
			logger.Warn("ENABLE_SGLANG=true set, but socket not found. Falling back to Gemini Cloud API.", slog.String("socket", socketPath))
		}
	} else {
		logger.Info("Local SGLang engine not running. Using Gemini Cloud API as primary engine.")
	}

	orch := orchestrator.NewOrchestrator(logger, storeDB, geminiClient, sgClient)
	go orch.StartBackgroundListener(ctx)

	// 5. Initialize Memory & Background Compaction (Phase 3.1)
	var memory *ingestion.DualIndexMemory
	if storeDB != nil {
		memory = ingestion.NewDualIndexMemory(logger, storeDB.Pool())
		compactionAgent := ingestion.NewCompactionAgent(logger, memory, geminiClient)
		go compactionAgent.StartBackgroundCompaction(ctx, storeDB)
	}

	mux := http.NewServeMux()
	path, handler := orchestratorv1connect.NewOrchestratorServiceHandler(rpcServer)
	mux.Handle(path, handler)

	// Initialize API Handlers
	apiServer := api.NewServer(logger, storeDB, memory)
	apiServer.RegisterHandlers(mux)

	var apiLimiter = rate.NewLimiter(rate.Limit(100), 200)

	// Global CORS Middleware for ConnectRPC and all endpoints
	withCORS := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !apiLimiter.Allow() {
				http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
				return
			}
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, Authorization, Connect-Protocol-Version, Connect-Timeout-Ms, Grpc-Timeout, X-Grpc-Web, X-User-Agent")
			w.Header().Set("Access-Control-Expose-Headers", "Connect-Protocol-Version, Connect-Timeout-Ms, Grpc-Status, Grpc-Message")
			
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
			next.ServeHTTP(w, r)
		})
	}

	// Use h2c for HTTP/2 without TLS to allow HTTP/3 routing/proxies in front.
	srv := &http.Server{
		Addr:    "0.0.0.0:8080",
		Handler: h2c.NewHandler(withCORS(mux), &http2.Server{}),
	}

	// Graceful Shutdown handled by shutdownCtx setup at top of function
	go func() {
		logger.Info("Starting Orchestrator Core", slog.String("addr", srv.Addr), slog.String("rpc_path", path))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Server failed", slog.Any("error", err))
			os.Exit(1)
		}
	}()

	<-shutdownCtx.Done()
	logger.Info("Shutdown signal received, draining connections...")
	drainCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(drainCtx); err != nil {
		logger.Error("Forced shutdown", slog.Any("error", err))
	}
	logger.Info("Server stopped gracefully")
}

func socketExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeSocket != 0
}
