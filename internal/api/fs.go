package api

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
)

type FileNode struct {
	Name     string      `json:"name"`
	Type     string      `json:"type"`
	Size     int64       `json:"size,omitempty"`
	Children []*FileNode `json:"children,omitempty"`
}

func buildTree(root string, depth int) ([]*FileNode, error) {
	if depth > 4 { return nil, nil }
	entries, err := os.ReadDir(root)
	if err != nil { return nil, err }
	var nodes []*FileNode
	for _, e := range entries {
		if e.Name() == ".git" || e.Name() == "node_modules" || e.Name() == "dist" {
			continue
		}
		node := &FileNode{Name: e.Name(), Type: "file"}
		if e.IsDir() {
			node.Type = "dir"
			children, _ := buildTree(filepath.Join(root, e.Name()), depth+1)
			node.Children = children
		} else {
			if info, err := e.Info(); err == nil {
				node.Size = info.Size()
			}
		}
		nodes = append(nodes, node)
	}
	return nodes, nil
}

func (s *Server) handleFiles(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if r.Method == "OPTIONS" {
		return
	}
	
	s.projectMu.Lock()
	dir := s.activeProjectDir
	s.projectMu.Unlock()

	tree, err := buildTree(dir, 0)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tree)
}

func (s *Server) handleSave(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" {
		return
	}
	
	var req struct {
		Filepath string `json:"filepath"`
		Content  string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	s.projectMu.Lock()
	fullPath, err := s.sanitizePath(s.activeProjectDir, req.Filepath)
	s.projectMu.Unlock()
	
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	if err := os.WriteFile(fullPath, []byte(req.Content), 0644); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleRead(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if r.Method == "OPTIONS" {
		return
	}
	
	filePath := r.URL.Query().Get("filepath")
	if filePath == "" {
		http.Error(w, "missing filepath", http.StatusBadRequest)
		return
	}
	
	s.projectMu.Lock()
	fullPath, err := s.sanitizePath(s.activeProjectDir, filePath)
	s.projectMu.Unlock()
	
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	data, err := os.ReadFile(fullPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "text/plain")
	w.Write(data)
}

func (s *Server) handleRename(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" { return }
	var req struct { OldPath string `json:"oldPath"`; NewPath string `json:"newPath"` }
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest); return
	}
	s.projectMu.Lock()
	oldFull, err1 := s.sanitizePath(s.activeProjectDir, req.OldPath)
	newFull, err2 := s.sanitizePath(s.activeProjectDir, req.NewPath)
	s.projectMu.Unlock()
	if err1 != nil || err2 != nil {
		http.Error(w, "invalid path", http.StatusBadRequest); return
	}
	if err := os.Rename(oldFull, newFull); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError); return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" { return }
	var req struct { Path string `json:"path"` }
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest); return
	}
	s.projectMu.Lock()
	fullPath, err := s.sanitizePath(s.activeProjectDir, req.Path)
	s.projectMu.Unlock()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest); return
	}
	if err := os.RemoveAll(fullPath); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError); return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleCreateNode(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" { return }
	var req struct { Path string `json:"path"`; IsDir bool `json:"isDir"` }
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest); return
	}
	s.projectMu.Lock()
	fullPath, err := s.sanitizePath(s.activeProjectDir, req.Path)
	s.projectMu.Unlock()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest); return
	}
	
	if req.IsDir {
		if err := os.MkdirAll(fullPath, 0755); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError); return
		}
	} else {
		if err := os.WriteFile(fullPath, []byte(""), 0644); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError); return
		}
	}
	w.WriteHeader(http.StatusOK)
}
