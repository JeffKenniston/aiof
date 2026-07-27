package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

func (s *Server) handleGraph(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")
	
	if s.storeDB == nil {
		w.Write([]byte(`{"nodes": [], "edges": []}`))
		return
	}
	
	if s.memory == nil {
		s.logger.Warn("Database not available for Knowledge Graph")
		w.Write([]byte(`{"nodes": [], "edges": []}`))
		return
	}
	
	graph, err := s.memory.FetchKnowledgeGraph(r.Context())
	if err != nil {
		s.logger.Warn("Failed to fetch knowledge graph, returning empty", slog.Any("error", err))
		w.Write([]byte(`{"nodes": [], "edges": []}`))
		return
	}
	
	json.NewEncoder(w).Encode(graph)
}
