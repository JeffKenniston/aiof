package api

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"aiof/internal/config"
	"aiof/internal/ingestion"
	"aiof/internal/llm"
	"aiof/internal/project"
	"aiof/internal/store"
)

type Server struct {
	logger           *slog.Logger
	storeDB          *store.Store
	memory           *ingestion.DualIndexMemory
	geminiClient     *llm.GeminiClient
	projectEngine    *project.Engine
	
	activeProjectDir string
	projectMu        sync.Mutex
	recentProjectsMu sync.Mutex
}

func NewServer(logger *slog.Logger, storeDB *store.Store, memory *ingestion.DualIndexMemory, geminiClient *llm.GeminiClient) *Server {
	pEngine := project.NewEngine(logger, nil, nil)
	return &Server{
		logger:           logger,
		storeDB:          storeDB,
		memory:           memory,
		geminiClient:     geminiClient,
		projectEngine:    pEngine,
		activeProjectDir: ".",
	}
}

func (s *Server) RegisterHandlers(mux *http.ServeMux) {
	// File System Endpoints
	mux.HandleFunc("/api/files", s.handleFiles)
	mux.HandleFunc("/api/save", s.handleSave)
	mux.HandleFunc("/api/read", s.handleRead)
	mux.HandleFunc("/api/rename", s.handleRename)
	mux.HandleFunc("/api/delete", s.handleDelete)
	mux.HandleFunc("/api/create_node", s.handleCreateNode)

	// Project & Settings Endpoints
	mux.HandleFunc("/api/projects", s.handleProjects)
	mux.HandleFunc("/api/projects/load", s.handleProjectsLoad)
	mux.HandleFunc("/api/projects/create", s.handleProjectsCreate)
	mux.HandleFunc("/api/settings/sync", s.handleSettingsSync)
	mux.HandleFunc("/api/settings/load", s.handleSettingsLoad)

	// Graph Endpoint
	mux.HandleFunc("/api/graph", s.handleGraph)

	// Terminal Endpoint
	mux.HandleFunc("/api/terminal", s.handleTerminal)

	// Health & Models Endpoint
	mux.HandleFunc("/api/health", s.handleHealth)
	mux.HandleFunc("/api/models", s.handleModels)

	// Chat Endpoint
	mux.HandleFunc("/api/chat/title", s.handleChatTitle)

	// AI Endpoints
	mux.HandleFunc("/api/generate_image", s.handleGenerateImage)
}

// ProjectMeta is used for recent projects tracking
type ProjectMeta struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

func (s *Server) sanitizePath(base, userPath string) (string, error) {
	full := filepath.Join(base, filepath.Clean(userPath))
	rel, err := filepath.Rel(base, full)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("path traversal blocked: %s", userPath)
	}
	return full, nil
}

func (s *Server) expandPath(pathStr string) string {
	if strings.HasPrefix(pathStr, "~") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, pathStr[1:])
		}
	}
	return pathStr
}

func (s *Server) getRecentProjects() []ProjectMeta {
	s.recentProjectsMu.Lock()
	defer s.recentProjectsMu.Unlock()
	return s.getRecentProjectsLocked()
}

func (s *Server) getRecentProjectsLocked() []ProjectMeta {
	file := filepath.Join(config.GetConfigDir(), "recent_projects.json")
	data, err := os.ReadFile(file)
	var projects []ProjectMeta
	if err == nil {
		json.Unmarshal(data, &projects)
	}
	return projects
}

func (s *Server) addRecentProject(name, path string) {
	s.recentProjectsMu.Lock()
	defer s.recentProjectsMu.Unlock()
	
	file := filepath.Join(config.GetConfigDir(), "recent_projects.json")
	
	projects := s.getRecentProjectsLocked()
	// Remove if already exists
	var updated []ProjectMeta
	for _, p := range projects {
		if p.Path != path {
			updated = append(updated, p)
		}
	}
	// Prepend
	updated = append([]ProjectMeta{{Name: name, Path: path}}, updated...)
	data, _ := json.MarshalIndent(updated, "", "  ")
	os.WriteFile(file, data, 0644)
}

func (s *Server) handleChatTitle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Prompt string `json:"prompt"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	if s.geminiClient == nil {
		http.Error(w, "Gemini client not initialized", http.StatusInternalServerError)
		return
	}
	
	ctx := r.Context()
	resp, _, err := s.geminiClient.RoutePrompt(ctx, req.Prompt, "auto", "You are a titling assistant. Generate a concise 3-5 word title summarizing the user's prompt. Return ONLY the title string, no quotes, no extra text.")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	title := strings.Trim(resp, "\"'\n\r\t ")
	
	json.NewEncoder(w).Encode(map[string]string{"title": title})
}
