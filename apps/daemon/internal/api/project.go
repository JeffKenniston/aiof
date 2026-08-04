package api

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"

	"aiof/internal/project"
)

func (s *Server) handleProjects(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == http.MethodOptions {
		return
	}

	w.Header().Set("Content-Type", "application/json")
	activeProj, err := s.projectEngine.GetActiveProject()
	activePath := s.activeProjectDir
	if err == nil && activeProj != nil {
		activePath = activeProj.Metadata.RootPath
	}

	recents := s.projectEngine.GetRecentProjects()
	recentMaps := make([]map[string]string, 0, len(recents))
	for _, rp := range recents {
		recentMaps = append(recentMaps, map[string]string{
			"name": rp.Name,
			"path": rp.RootPath,
		})
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"active":         activePath,
		"active_project": activeProj,
		"projects":       recentMaps,
		"modalities":     project.AllModalities,
	})
}

func (s *Server) handleProjectsLoad(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == http.MethodOptions {
		return
	}

	if r.Method == http.MethodPost {
		var req struct {
			Path string `json:"path"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err == nil {
			req.Path = s.expandPath(req.Path)
			proj, err := s.projectEngine.LoadProject(r.Context(), req.Path)
			if err == nil {
				s.projectMu.Lock()
				s.activeProjectDir = proj.Metadata.RootPath
				s.projectMu.Unlock()

				s.addRecentProject(proj.Metadata.Name, proj.Metadata.RootPath)

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(proj)
			} else {
				http.Error(w, "Invalid directory path or project load failed: "+err.Error(), http.StatusBadRequest)
			}
		} else {
			http.Error(w, "Invalid request payload", http.StatusBadRequest)
		}
	} else {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleProjectsCreate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == http.MethodOptions {
		return
	}

	if r.Method == http.MethodPost {
		var req struct {
			Name       string           `json:"name"`
			ParentPath string           `json:"parentPath"`
			Modality   project.Modality `json:"modality,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err == nil {
			req.ParentPath = s.expandPath(req.ParentPath)
			modality := req.Modality
			if modality == "" {
				modality = project.ModalitySoftwareEngineering
			}

			proj, err := s.projectEngine.CreateProject(r.Context(), req.Name, req.ParentPath, modality)
			if err != nil {
				http.Error(w, "Failed to create project: "+err.Error(), http.StatusInternalServerError)
				return
			}

			s.addRecentProject(req.Name, proj.Metadata.RootPath)

			s.projectMu.Lock()
			s.activeProjectDir = proj.Metadata.RootPath
			s.projectMu.Unlock()

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(proj)
		} else {
			http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		}
	} else {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleSettingsSync(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.projectMu.Lock()
	dir := s.activeProjectDir
	s.projectMu.Unlock()

	aiofDir := project.GetProjectConfigDir(dir)
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
	if r.Method == http.MethodOptions {
		return
	}

	s.projectMu.Lock()
	dir := s.activeProjectDir
	s.projectMu.Unlock()

	targetPath := filepath.Join(project.GetProjectConfigDir(dir), "project.json")
	data, err := os.ReadFile(targetPath)
	if err != nil {
		http.Error(w, "Settings not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(data)
}
