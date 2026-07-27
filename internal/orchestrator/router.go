// HELLO WORLD
package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"aiof/internal/agent"
	"aiof/internal/llm"
	"aiof/internal/store"
	"aiof/internal/sandbox"
	"google.golang.org/genai"
)

const (
	colorRunningCloud = "#f1fa8c"
	colorRunningLocal = "#ffb86c"
	colorRunning      = "#58a6ff"
	colorError        = "#ff5555"
	colorSuccess      = "#50fa7b"
)

type chatMessage struct {
	ID          string `json:"id"`
	Role        string `json:"role"`
	Content     string `json:"content"`
	AgentName   string `json:"agentName"`
	Timestamp   int64  `json:"timestamp"`
	WorkspaceID string `json:"workspaceId,omitempty"`
	ThreadID    string `json:"threadId,omitempty"`
	ModelUsed   string `json:"modelUsed,omitempty"`
	Reasoning   []string `json:"reasoning,omitempty"`
}

type routeLogEntry struct {
	ID        string `json:"id"`
	Timestamp int64  `json:"timestamp"`
	Agent     string `json:"agent"`
	Severity  string `json:"severity"`
	Message   string `json:"message"`
}


// Orchestrator acts as the primary event bus routing layer.
type Orchestrator struct {
	logger       *slog.Logger
	store        *store.Store
	geminiClient   *llm.GeminiClient
	openvinoClient *llm.OpenVINOClient
	routeLLM       *llm.RouteLLMClassifier
	
	tasks   map[string]context.CancelFunc
	tasksMu sync.Mutex
	
	telemetry *TelemetryEmitter
}

// NewOrchestrator creates a new instance of the orchestration middleware.
func NewOrchestrator(logger *slog.Logger, s *store.Store, g *llm.GeminiClient, ov *llm.OpenVINOClient, routeLLM *llm.RouteLLMClassifier) *Orchestrator {
	o := &Orchestrator{
		logger:         logger,
		store:          s,
		geminiClient:   g,
		openvinoClient: ov,
		routeLLM:       routeLLM,
		tasks:          make(map[string]context.CancelFunc),
	}
	o.telemetry = NewTelemetryEmitter(o)
	return o
}

// StartBackgroundListener continuously listens to the Store EventBus for user intents
// pushed from the frontend (e.g. a task_queue insertion) and dynamically routes them.
func (o *Orchestrator) StartBackgroundListener(ctx context.Context) {
	if o.store == nil {
		o.logger.Warn("Store bypassed; Orchestrator background listener disabled.")
		return
	}

	o.logger.Info("Orchestrator background listener started")
	go o.telemetry.Start(ctx)

	// Spawn the VLM-based MCP Server
	vlmContainer := sandbox.NewMCPContainer(o.logger, sandbox.RuntimeNodeMCP, "mcp/vlm-browser")
	if err := vlmContainer.Start(ctx); err != nil {
		o.logger.Warn("Failed to start VLM MCP server", slog.Any("error", err))
		vlmContainer = nil
	} else {
		defer vlmContainer.Stop()
		o.logger.Info("VLM Browser MCP Server successfully started")
	}

	sem := make(chan struct{}, 100)
	for _, doc := range o.store.StreamMutations(ctx) {
		// Process agent_intents to drive the VLM Browser
		if doc.DocumentType == "agent_intents" && vlmContainer != nil {
			type AgentIntent struct {
				ID         string                 `json:"id"`
				IntentType string                 `json:"intentType"`
				Payload    map[string]interface{} `json:"payload"`
				Status     string                 `json:"status"`
			}
			var intent AgentIntent
			if err := json.Unmarshal(doc.Payload, &intent); err == nil && intent.Status == "pending" {
				go func(i AgentIntent) {
					var res json.RawMessage
					var err error
					if i.IntentType == "MANUAL_NAVIGATE" {
						res, err = vlmContainer.CallTool(ctx, "navigate", map[string]interface{}{"url": i.Payload["url"]})
					} else if i.IntentType == "MANUAL_CLICK" {
						res, err = vlmContainer.CallTool(ctx, "click", map[string]interface{}{
							"x": i.Payload["x"],
							"y": i.Payload["y"],
						})
					}
					
					if err == nil && res != nil {
						// Parse MCP Result
						type mcpContent struct {
							Type     string `json:"type"`
							Data     string `json:"data"`
							MimeType string `json:"mimeType"`
						}
						type mcpResult struct {
							Content []mcpContent `json:"content"`
						}
						var resultData mcpResult
						if json.Unmarshal(res, &resultData) == nil {
							for _, c := range resultData.Content {
								if c.Type == "image" && c.Data != "" {
									frameID := fmt.Sprintf("frame-%d", time.Now().UnixNano())
									frameDoc := fmt.Sprintf(`{"id": "%s", "timestamp": %d, "base64Data": "%s"}`, frameID, time.Now().UnixMilli(), c.Data)
									o.emitState(ctx, "vlm_frames", frameID, frameDoc)
								}
							}
						}
					}
					
					// Mark consumed
					i.Status = "consumed"
					payload, _ := json.Marshal(i)
					o.emitState(ctx, "agent_intents", i.ID, string(payload))
				}(intent)
			}
		}

		// If the frontend pushes a new task into the task queue, we intercept and act on it.
		if doc.DocumentType == "task_queue" {
			type TaskPayload struct {
				ID        string          `json:"id"`
				Title     string          `json:"title"`
				Column    string          `json:"column"`
				ProjectID string          `json:"project_id"`
				ViewMode  string          `json:"view_mode"`
				ExtendedThinking bool     `json:"extended_thinking,omitempty"`
				Agent     string          `json:"agent"`
				WorkspaceID string        `json:"workspaceId,omitempty"`
				ThreadID    string        `json:"threadId,omitempty"`
				Extra     json.RawMessage `json:"extra,omitempty"`
			}
			var task TaskPayload
			if err := json.Unmarshal(doc.Payload, &task); err != nil {
				continue
			}
			
			if task.Column == "cancelled" {
				o.tasksMu.Lock()
				if cancel, exists := o.tasks[task.ID]; exists {
					cancel()
					delete(o.tasks, task.ID)
					o.logger.Info("Cancelled active task via Stop button", "taskID", task.ID)
				}
				o.tasksMu.Unlock()
				continue
			}
			
			// Only process new active tasks
			if task.Column == "active" {
				title := task.Title
				taskID := doc.ID
				if taskID == "" {
					taskID = fmt.Sprintf("task-%d", time.Now().UnixNano())
				}
				task.ID = taskID
				projectDir := task.ProjectID
				viewMode := task.ViewMode
				if projectDir == "" {
					projectDir = "."
				}
				
				taskCtx, cancel := context.WithCancel(ctx)
				o.tasksMu.Lock()
				o.tasks[taskID] = cancel
				o.tasksMu.Unlock()
				
				// Change task to "completed" asynchronously
				sem <- struct{}{}
				go func(t TaskPayload, tid string, pDir string, vm string, wid string, thid string, tCtx context.Context) {
					defer func() {
						<-sem
						o.tasksMu.Lock()
						delete(o.tasks, tid)
						o.tasksMu.Unlock()
					}()
					changes := o.processTask(tCtx, title, pDir, vm, wid, thid)
					
					// Update task in db to completed so it doesn't stay stuck
					t.ID = tid
					t.Column = "completed"
					t.Agent = "Executor"
					t.Extra = json.RawMessage(fmt.Sprintf(`{"proposedChanges": %q}`, changes))
					payload, err := json.Marshal(t)
					if err != nil {
						o.logger.Error("failed to marshal task payload", "error", err)
						return
					}
					o.emitState(ctx, "task_queue", tid, string(payload))
				}(task, taskID, projectDir, viewMode, task.WorkspaceID, task.ThreadID, taskCtx)
			}
		}
	}
	
	o.logger.Info("Orchestrator background listener stopped")
}

func (o *Orchestrator) processTask(ctx context.Context, prompt string, projectDir string, viewMode string, workspaceID string, threadID string) string {
	o.logger.Info("Orchestrator processing new task intent", slog.String("prompt", prompt), slog.String("projectDir", projectDir), slog.String("viewMode", viewMode))
	startTime := time.Now()
	var taskError bool

	// 1. Emit Agent Graph State (Router Running)
	o.emitState(ctx, "agent_graph_state", "router", fmt.Sprintf(`{"id": "router", "label": "Router", "model": "Pro", "color": "%s", "status": "running"}`, colorRunning))
	o.emitLog(ctx, "Router", "info", fmt.Sprintf("Evaluating intent: %s", prompt))

	if o.geminiClient == nil {
		o.emitLog(ctx, "Router", "error", "GEMINI_API_KEY not configured. Cannot perform generative routing.")
		o.emitState(ctx, "agent_graph_state", "router", fmt.Sprintf(`{"id": "router", "label": "Router", "model": "Pro", "color": "%s", "status": "error"}`, colorError))
		o.emitChatError(ctx, "GEMINI_API_KEY not configured. Cannot perform generative routing.", workspaceID, threadID)
		taskError = true
		o.telemetry.RecordRequest(float64(time.Since(startTime).Milliseconds()), taskError)
		return ""
	}

	var routeData agent.CognitiveSequence
	var finalModelUsed string
	
	if o.routeLLM != nil {
		modelClass, tier, err := o.routeLLM.PredictComplexity(ctx, prompt)
		if err == nil && (modelClass == llm.ClassFlash || modelClass == llm.ClassFlashLite) {
			o.emitLog(ctx, "Router", "info", fmt.Sprintf("Matrix Factorization determined task is %s (Threshold < 0.5) - %s. Bypassing heavy generative routing.", modelClass, tier))
			
			if modelClass == llm.ClassFlashLite {
				// Fast execution for conversational / trivial intents
				res, err := o.geminiClient.Client().Models.GenerateContent(ctx, llm.ModelFlashLite, genai.Text(prompt), nil)
				var reply string
				if err == nil && res != nil && len(res.Candidates) > 0 && res.Candidates[0].Content != nil && len(res.Candidates[0].Content.Parts) > 0 {
					reply = res.Candidates[0].Content.Parts[0].Text
				} else {
					reply = "Received your message."
				}
				
				chatMsg := chatMessage{
					ID:          fmt.Sprintf("reply-%d", time.Now().UnixNano()),
					Role:        "agent",
					Content:     reply,
					AgentName:   "Archimedes",
					Timestamp:   time.Now().UnixMilli(),
					WorkspaceID: workspaceID,
					ThreadID:    threadID,
					ModelUsed:   string(llm.ModelFlashLite),
					Reasoning:   []string{fmt.Sprintf("Trivial Prompt detected (Tier: %s).", tier), "Executing fast-path API generation..."},
				}
				chatPayload, _ := json.Marshal(chatMsg)
				o.emitState(ctx, "chat_messages", chatMsg.ID, string(chatPayload))
				o.emitState(ctx, "agent_graph_state", "router", fmt.Sprintf(`{"id": "router", "label": "Router", "model": "FlashLite", "color": "%s", "status": "success"}`, colorSuccess))
				o.telemetry.RecordRequest(float64(time.Since(startTime).Milliseconds()), false)
				return ""
			}
			
			routeData = agent.CognitiveSequence{
				Reasoning:   []string{"Task is trivial, bypassing generative routing."},
				TargetNodes: []string{"code_developer_agent"},
				ChatReply:   "Task is trivial, executing directly.",
			}
			finalModelUsed = string(modelClass)
			goto SkipGenerativeRouting
		}
	}

	{
		// 2. Perform Generative Routing via LLM Gateway (Asymmetric Routing)
		resp, modelUsed, err := o.geminiClient.RoutePrompt(ctx, prompt, viewMode, "")
		if err != nil {
			o.emitLog(ctx, "Router", "error", fmt.Sprintf("Routing failed (%s): %v", modelUsed, err))
			o.emitState(ctx, "agent_graph_state", "router", fmt.Sprintf(`{"id": "router", "label": "Router", "model": "Pro", "color": "%s", "status": "error"}`, colorError))
			o.emitChatError(ctx, fmt.Sprintf("Routing failed: %v", err), workspaceID, threadID)
			taskError = true
			o.telemetry.RecordRequest(float64(time.Since(startTime).Milliseconds()), taskError)
			return ""
		}

		if err := json.Unmarshal([]byte(resp), &routeData); err != nil {
			slog.Error("failed to unmarshal route response", "error", err, "response", resp)
			o.emitChatError(ctx, "Failed to parse routing sequence from LLM.", workspaceID, threadID)
			taskError = true
			o.telemetry.RecordRequest(float64(time.Since(startTime).Milliseconds()), taskError)
			return ""
		}
		
		finalModelUsed = string(modelUsed)
		o.emitLog(ctx, "Router", "info", fmt.Sprintf("Generative Routing Complete (%s). Response length: %d bytes.", modelUsed, len(resp)))
	}

SkipGenerativeRouting:

	chatReply := "Task routed successfully."
	if routeData.ChatReply != "" {
		chatReply = routeData.ChatReply
	}

	chatMsg := chatMessage{
		ID:          fmt.Sprintf("reply-%d", time.Now().UnixNano()),
		Role:        "agent",
		Content:     chatReply,
		AgentName:   "Archimedes",
		Timestamp:   time.Now().UnixMilli(),
		WorkspaceID: workspaceID,
		ThreadID:    threadID,
		ModelUsed:   finalModelUsed,
		Reasoning:   routeData.Reasoning,
	}
	chatPayload, err := json.Marshal(chatMsg)
	if err != nil {
		slog.Error("failed to marshal chat message", "error", err)
		taskError = true
		o.telemetry.RecordRequest(float64(time.Since(startTime).Milliseconds()), taskError)
		return ""
	}
	o.emitState(ctx, "chat_messages", chatMsg.ID, string(chatPayload))

	o.emitState(ctx, "agent_graph_state", "router", fmt.Sprintf(`{"id": "router", "label": "Router", "model": "Pro", "color": "%s", "status": "success"}`, colorSuccess))

	// Prepare decoupled AgentHost
	host := &orchestratorHost{o: o, projectDir: projectDir}
	
	// Create the SetValuedRouter
	r := agent.NewSetValuedRouter(o.logger)

	// Register the Fabricator Agent (Arbiter Pattern)
	r.RegisterAgent("fabricator_agent", agent.NewFabricatorAgent(o.logger, o.geminiClient.Client()))
	
	// Register the CodeDeveloperAgent
	codeDev := agent.NewCodeDeveloperAgent(o.logger, o.geminiClient.Client())
	r.RegisterAgent("code_developer_agent", codeDev)
	
	// Register the QATesterAgent
	qaTester := agent.NewQATesterAgent(o.logger, o.geminiClient.Client())
	r.RegisterAgent("qa_tester_agent", qaTester)

	// Register the ResearchAgent
	researcher := agent.NewResearchAgent(o.logger, o.geminiClient.Client())
	r.RegisterAgent("research_agent", researcher)
	
	// Register the ArchitectAgent
	architect := agent.NewArchitectAgent(o.logger, o.geminiClient.Client())
	r.RegisterAgent("architecture_agent", architect)

	// Register the Extended Enterprise Agents
	r.RegisterAgent("devops_agent", agent.NewDevOpsAgent(o.logger, o.geminiClient.Client()))
	r.RegisterAgent("secops_agent", agent.NewSecOpsAgent(o.logger, o.geminiClient.Client()))
	r.RegisterAgent("database_agent", agent.NewDatabaseAgent(o.logger, o.geminiClient.Client()))
	r.RegisterAgent("ui_designer_agent", agent.NewUIDesignerAgent(o.logger, o.geminiClient.Client()))
	r.RegisterAgent("doc_agent", agent.NewDocAgent(o.logger, o.geminiClient.Client()))

	// Register the Bleeding Edge Agents
	r.RegisterAgent("sre_agent", agent.NewSREAgent(o.logger, o.geminiClient.Client()))
	r.RegisterAgent("chaos_agent", agent.NewChaosAgent(o.logger, o.geminiClient.Client()))
	r.RegisterAgent("mlops_agent", agent.NewMLOpsAgent(o.logger, o.geminiClient.Client()))
	r.RegisterAgent("product_owner_agent", agent.NewProductOwnerAgent(o.logger, o.geminiClient.Client()))
	r.RegisterAgent("localization_agent", agent.NewLocalizationAgent(o.logger, o.geminiClient.Client()))

	// Register the Hyper-Specialized Agents (New)
	r.RegisterAgent("performance_agent", agent.NewPerformanceAgent(o.logger, o.geminiClient.Client()))
	r.RegisterAgent("mobile_agent", agent.NewMobileAgent(o.logger, o.geminiClient.Client()))
	r.RegisterAgent("data_agent", agent.NewDataAgent(o.logger, o.geminiClient.Client()))
	r.RegisterAgent("a11y_agent", agent.NewA11yAgent(o.logger, o.geminiClient.Client()))
	r.RegisterAgent("redteam_agent", agent.NewRedTeamAgent(o.logger, o.geminiClient.Client()))
	r.RegisterAgent("scrum_master_agent", agent.NewScrumMasterAgent(o.logger, o.geminiClient.Client()))
	r.RegisterAgent("legal_agent", agent.NewLegalAgent(o.logger, o.geminiClient.Client()))

	// Register the Futuristic & Niche Agents (Extreme Scaling)
	r.RegisterAgent("devex_agent", agent.NewDevExAgent(o.logger, o.geminiClient.Client()))
	r.RegisterAgent("finops_agent", agent.NewFinOpsAgent(o.logger, o.geminiClient.Client()))
	r.RegisterAgent("hardware_agent", agent.NewHardwareAgent(o.logger, o.geminiClient.Client()))
	r.RegisterAgent("web3_agent", agent.NewWeb3Agent(o.logger, o.geminiClient.Client()))
	r.RegisterAgent("graphics_agent", agent.NewGraphicsAgent(o.logger, o.geminiClient.Client()))
	r.RegisterAgent("support_agent", agent.NewSupportAgent(o.logger, o.geminiClient.Client()))
	r.RegisterAgent("quantum_agent", agent.NewQuantumAgent(o.logger, o.geminiClient.Client()))

	// Register Sandbox & VLM Agents (Phase 4)
	sandboxAgent := agent.NewSandboxOrchestratorAgent(o.logger, o.geminiClient.Client())
	r.RegisterAgent("sandbox_orchestrator", sandboxAgent)
	
	studentAgent := agent.NewStudentAgent(o.logger, o.geminiClient.Client())
	r.RegisterAgent("student_agent", studentAgent)
	r.RegisterAgent("browser_automation_agent", studentAgent)
	
	// Dispatch the agent graph concurrently
	globalProposedChanges, err := r.Dispatch(ctx, host, prompt, routeData)
	if err != nil {
		o.emitLog(ctx, "Router", "error", fmt.Sprintf("Graph execution failed: %v", err))
		taskError = true
		o.telemetry.RecordRequest(float64(time.Since(startTime).Milliseconds()), taskError)
		return ""
	}

	// 3. Phase 2.4/2.5 Consensus Protocols (Validator Agent)
	// Run the Validator on the globalProposedChanges
	validator := agent.NewValidatorAgent(o.logger, o.geminiClient.Client())
	decision, err := validator.Execute(ctx, host, globalProposedChanges, nil)
	if err != nil {
		o.emitLog(ctx, "Router", "warn", fmt.Sprintf("Validator rejected changes: %v", err))
		o.emitChatError(ctx, fmt.Sprintf("Validation Failed: %v\nDecision: %s", err, decision), workspaceID, threadID)
		// In a real multi-agent flow, we might route back to CodeDeveloper for revision.
		taskError = true
		o.telemetry.RecordRequest(float64(time.Since(startTime).Milliseconds()), taskError)
		return ""
	}

	o.emitLog(ctx, "Router", "info", fmt.Sprintf("Validator approved changes: %s", decision))

	o.telemetry.RecordRequest(float64(time.Since(startTime).Milliseconds()), false)
	return globalProposedChanges
}

type orchestratorHost struct {
	o          *Orchestrator
	projectDir string
}

func (h *orchestratorHost) EmitLog(ctx context.Context, agentName, sev, msg string) {
	h.o.emitLog(ctx, agentName, sev, msg)
}

func (h *orchestratorHost) EmitState(ctx context.Context, docType, id, payload string) {
	h.o.emitState(ctx, docType, id, payload)
}

func (h *orchestratorHost) GetProjectDir() string {
	return h.projectDir
}

func (h *orchestratorHost) GetLatestVLMFrame(ctx context.Context) string {
	if h.o.store == nil {
		return ""
	}
	doc, err := h.o.store.GetLatestDocument(ctx, "vlm_frames")
	if err == nil && doc != nil {
		type framePayload struct {
			Base64Data string `json:"base64Data"`
		}
		var f framePayload
		if err := json.Unmarshal(doc.Payload, &f); err == nil {
			return f.Base64Data
		}
	}
	return ""
}

func (o *Orchestrator) emitState(ctx context.Context, docType string, id string, payload string) {
	if o.store == nil {
		return
	}
	if err := o.store.WriteMutation(ctx, store.Document{
		ID:           id,
		DocumentType: docType,
		Payload:      []byte(payload),
		UpdatedAt:    time.Now().UnixMilli(),
		IsDeleted:    false,
	}); err != nil {
		o.logger.Error("emitState: WriteMutation failed", slog.String("docType", docType), slog.String("id", id), slog.Any("error", err))
	}
}

func (o *Orchestrator) emitLog(ctx context.Context, agentName, severity, message string) {
	o.logger.Info("Agent Log", slog.String("agent", agentName), slog.String("severity", severity), slog.String("message", message))
	logID := fmt.Sprintf("log-%d", time.Now().UnixNano())
	logEntry := routeLogEntry{
		ID:        logID,
		Timestamp: time.Now().UnixMilli(),
		Agent:     agentName,
		Severity:  severity,
		Message:   message,
	}
	payload, err := json.Marshal(logEntry)
	if err != nil {
		slog.Error("failed to marshal log entry", "error", err)
		return
	}
	o.emitState(ctx, "agent_log", logID, string(payload))
}

func (o *Orchestrator) emitChatError(ctx context.Context, message string, workspaceID string, threadID string) {
	chatMsg := chatMessage{
		ID:          fmt.Sprintf("reply-%d", time.Now().UnixNano()),
		Role:        "agent",
		Content:     fmt.Sprintf("System Error: %s", message),
		AgentName:   "Archimedes",
		Timestamp:   time.Now().UnixMilli(),
		WorkspaceID: workspaceID,
		ThreadID:    threadID,
	}
	chatPayload, err := json.Marshal(chatMsg)
	if err == nil {
		o.emitState(ctx, "chat_messages", chatMsg.ID, string(chatPayload))
	}
}
