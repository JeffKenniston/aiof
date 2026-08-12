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

	"aiof/internal/ingestion"
	"aiof/internal/store"
)

type Server struct {
	logger           *slog.Logger
	storeDB          *store.Store
	memory           *ingestion.DualIndexMemory
	
	activeProjectDir string
	projectMu        sync.Mutex
	recentProjectsMu sync.Mutex
}

func NewServer(logger *slog.Logger, storeDB *store.Store, memory *ingestion.DualIndexMemory) *Server {
	return &Server{
		logger:           logger,
		storeDB:          storeDB,
		memory:           memory,
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

	// Health Endpoint
	mux.HandleFunc("/api/health", s.handleHealth)
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
	home, _ := os.UserHomeDir()
	file := filepath.Join(home, ".aiof", "recent_projects.json")
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
	home, _ := os.UserHomeDir()
	os.MkdirAll(filepath.Join(home, ".aiof"), 0755)
	file := filepath.Join(home, ".aiof", "recent_projects.json")
	
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
