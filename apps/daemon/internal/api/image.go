package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"
	"aiof/internal/config"
)

type GenerateImageRequest struct {
	Prompt string `json:"prompt"`
}

type GenerateImageResponse struct {
	URL string `json:"url"`
}

func (s *Server) handleGenerateImage(w http.ResponseWriter, r *http.Request) {
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

	var req GenerateImageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	s.logger.Info("Generating image for prompt", "prompt", req.Prompt)

	assetsDir := filepath.Join(config.GetHomeDir(), "assets")
	os.MkdirAll(assetsDir, 0755)
	
	fileName := fmt.Sprintf("generated_%d.png", time.Now().UnixNano())
	filePath := filepath.Join(assetsDir, fileName)

	// In a real implementation this hits google.golang.org/api/imggen. 
	// For now, write a 1x1 transparent PNG.
	transparentPNG := []byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 
		0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52, 
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01, 
		0x08, 0x06, 0x00, 0x00, 0x00, 0x1f, 0x15, 0xc4, 
		0x89, 0x00, 0x00, 0x00, 0x0a, 0x49, 0x44, 0x41, 
		0x54, 0x78, 0x9c, 0x63, 0x00, 0x01, 0x00, 0x00, 
		0x05, 0x00, 0x01, 0x0d, 0x0a, 0x2d, 0xb4, 0x00, 
		0x00, 0x00, 0x00, 0x49, 0x45, 0x4e, 0x44, 0xae, 
		0x42, 0x60, 0x82,
	}

	if err := os.WriteFile(filePath, transparentPNG, 0644); err != nil {
		http.Error(w, "Failed to save generated image", http.StatusInternalServerError)
		return
	}

	resp := GenerateImageResponse{
		URL: fmt.Sprintf("/api/read?path=%s", fileName), // We would map this better in production
	}

	json.NewEncoder(w).Encode(resp)
}
