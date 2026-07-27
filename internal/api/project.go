package api

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"aiof/internal/config"
)

func getProjectConfigDir(projectDir string) string {
	hash := sha256.Sum256([]byte(projectDir))
	return filepath.Join(config.GetConfigDir(), "projects", fmt.Sprintf("%x", hash))
}

func (s *Server) handleProjects(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" { return }
	
	s.projectMu.Lock()
	current := s.activeProjectDir
	s.projectMu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"active": current,
		"projects": s.getRecentProjects(),
	})
}

func (s *Server) handleProjectsLoad(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" { return }
	
	if r.Method == "POST" {
		var req struct { Path string `json:"path"` }
		if err := json.NewDecoder(r.Body).Decode(&req); err == nil {
			req.Path = s.expandPath(req.Path)
			// verify directory exists
			if stat, err := os.Stat(req.Path); err == nil && stat.IsDir() {
				name := filepath.Base(req.Path)
				s.addRecentProject(name, req.Path)
				
				s.projectMu.Lock()
				s.activeProjectDir = req.Path
				s.projectMu.Unlock()
				
				w.WriteHeader(http.StatusOK)
			} else {
				http.Error(w, "Invalid directory path", http.StatusBadRequest)
			}
		}
	}
}

func (s *Server) handleProjectsCreate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" { return }
	
	if r.Method == "POST" {
		var req struct {
			Name       string `json:"name"`
			ParentPath string `json:"parentPath"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err == nil {
			req.ParentPath = s.expandPath(req.ParentPath)
			fullPath := filepath.Join(req.ParentPath, req.Name)
			
			// Create the project directory
			if err := os.MkdirAll(fullPath, 0755); err != nil {
				http.Error(w, "Failed to create directory: "+err.Error(), http.StatusInternalServerError)
				return
			}
			
			// Initialize the global .aiof config dir for this project
			aiofDir := getProjectConfigDir(fullPath)
			os.MkdirAll(aiofDir, 0755)
			
			s.addRecentProject(req.Name, fullPath)
			
			s.projectMu.Lock()
			s.activeProjectDir = fullPath
			s.projectMu.Unlock()
			
			w.WriteHeader(http.StatusOK)
		}
	}
}

func (s *Server) handleSettingsSync(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" { return }
	if r.Method != "POST" { http.Error(w, "Method not allowed", http.StatusMethodNotAllowed); return }

	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest); return
	}

	s.projectMu.Lock()
	dir := s.activeProjectDir
	s.projectMu.Unlock()

	aiofDir := getProjectConfigDir(dir)
	if err := os.MkdirAll(aiofDir, 0755); err != nil {
		http.Error(w, "Failed to create .aiof directory: "+err.Error(), http.StatusInternalServerError)
		return
	}

	targetPath := filepath.Join(aiofDir, "project.json")
	data, err := json.MarshalIndent(req, "", "  ")
	if err != nil {
		http.Error(w, "Failed to encode settings: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if err := os.WriteFile(targetPath, data, 0644); err != nil {
		http.Error(w, "Failed to write project.json: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleSettingsLoad(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if r.Method == "OPTIONS" { return }
	
	s.projectMu.Lock()
	dir := s.activeProjectDir
	s.projectMu.Unlock()

	targetPath := filepath.Join(getProjectConfigDir(dir), "project.json")
	data, err := os.ReadFile(targetPath)
	if err != nil {
		http.Error(w, "Settings not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}
