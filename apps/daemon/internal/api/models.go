package api

import (
	"encoding/json"
	"net/http"
	"strings"
)

func (s *Server) handleModels(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var models []string
	
	if s.geminiClient != nil && s.geminiClient.Client() != nil {
		iter, err := s.geminiClient.Client().Models.List(r.Context(), nil)
		if err == nil {
			for {
				m, err := iter.Next(r.Context())
				if err != nil {
					break
				}
				if strings.Contains(m.Name, "gemini") {
					models = append(models, strings.TrimPrefix(m.Name, "models/"))
				}
			}
		}
	}
	
	if len(models) == 0 {
		models = []string{
			"gemini-3.1-pro-preview", 
			"gemini-3.6-flash", 
			"gemini-3.5-flash-lite", 
			"claude-3-5-sonnet", 
			"claude-3-opus", 
			"claude-3-haiku", 
			"gpt-4o", 
			"deepseek-coder",
		}
	}
	
	json.NewEncoder(w).Encode(models)
}
