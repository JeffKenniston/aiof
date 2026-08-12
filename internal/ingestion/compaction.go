package ingestion

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"


	"aiof/internal/llm"
	"aiof/internal/store"
)

// CompactionAgent runs asynchronously alongside the parser to implement
// Aggressive State Compaction (Blueprint 1), preventing context window dilution.
type CompactionAgent struct {
	logger *slog.Logger
	memory          *DualIndexMemory
	client          *llm.GeminiClient
	Embedder        Embedder
	BudgetThreshold int
}

func NewCompactionAgent(logger *slog.Logger, memory *DualIndexMemory, client *llm.GeminiClient) *CompactionAgent {
	return &CompactionAgent{
		logger:          logger,
		memory:          memory,
		client:          client,
		BudgetThreshold: 10,
	}
}

// Compact evaluates raw file histories and replaces them with dense semantic updates.
func (c *CompactionAgent) Compact(ctx context.Context, targetEntityID string, verboseHistory string) error {
	c.logger.Info("Executing aggressive state compaction for entity", slog.String("entity", targetEntityID))
	
	// Dispatch a prompt to Gemini (Pro tier) to summarize/compress the history.
	prompt := fmt.Sprintf(`Analyze the following verbose code history and compress it into a highly dense semantic summary. Focus strictly on architectural changes, core logic shifts, and intent. Omit boilerplate.
	
	History:
	%s`, verboseHistory)
	
	compressedSummary, modelUsed, err := c.client.RoutePrompt(ctx, prompt, "background_compaction", "")
	if err != nil {
		return fmt.Errorf("compaction LLM execution failed via %s: %w", modelUsed, err)
	}

	c.logger.Debug("State successfully compacted", slog.String("model", modelUsed), slog.Int("original_length", len(verboseHistory)), slog.Int("compacted_length", len(compressedSummary)))

	// Update the Dual-Index Memory with the compressed semantic representation.
	var realEmbedding []float32
	if c.Embedder != nil {
		emb, err := c.Embedder.EmbedText(ctx, compressedSummary)
		if err != nil {
			c.logger.Warn("Embedding generation failed, using zero-vector fallback", slog.Any("error", err))
			realEmbedding = make([]float32, 768)
		} else {
			realEmbedding = emb
		}
	} else {
		c.logger.Warn("no embedder configured, using zero vectors")
		realEmbedding = make([]float32, 768)
	}

	err = c.memory.UpsertEntity(ctx, targetEntityID+"_compacted", compressedSummary, map[string]string{"type": "compacted_history"}, realEmbedding, nil, "SEMANTIC")
	if err != nil {
		return fmt.Errorf("failed to save compacted state: %w", err)
	}
	
	return nil
}

// StartBackgroundCompaction listens to the WAL stream and buffers episodic memory.
// Once a budget threshold is met, it runs the single-pass ADD-only extraction (Phase 3.1).
func (c *CompactionAgent) StartBackgroundCompaction(ctx context.Context, s *store.Store) {
	c.logger.Info("Compaction background listener started")
	
	var episodicBuffer []string
	
	for _, doc := range s.StreamMutations(ctx) {
		if doc.DocumentType == "chat_messages" || doc.DocumentType == "episodic_memory" {
			episodicBuffer = append(episodicBuffer, string(doc.Payload))
			
			threshold := c.BudgetThreshold
			if threshold <= 0 {
				threshold = 10
			}
			if len(episodicBuffer) >= threshold { // Budget threshold
			    verboseHistory := strings.Join(episodicBuffer, "\n")
			    episodicBuffer = nil // Reset
			    
			    go c.resolveAndCompact(ctx, s, verboseHistory)
			}
		}
	}
}

// resolveAndCompact performs Phase 3.3 Contradiction Resolution
func (c *CompactionAgent) resolveAndCompact(ctx context.Context, s *store.Store, history string) {
    // 1. LLM Extraction (Phase 3.1)
    compressed, _, err := c.client.RoutePrompt(ctx, "Extract dense semantic facts from: " + history, "background_compaction", "")
    if err != nil {
		c.logger.Error("Failed to run extraction", slog.Any("error", err))
		return
	}
	
    // 2. Check for contradictions against existing knowledge 
    // 3. Write semantic fact back to WAL (Store.WriteMutation). 
    //    Because we use WriteMutation, the TOKI bitemporal operators natively 
    //    trigger the `documents_audit` demotion for overwritten facts (Phase 3.3).
	newDocID := fmt.Sprintf("semantic_fact_%d", time.Now().UnixNano())

	var existingCount int
	err = s.Pool().QueryRow(ctx, 
		`SELECT COUNT(*) FROM documents WHERE document_type = 'semantic_memory' AND id LIKE $1`,
		newDocID[:len("semantic_fact")]+"%").Scan(&existingCount)
	if err == nil && existingCount > 0 {
		c.logger.Info("Contradiction detected: Overwriting existing semantic facts", slog.Int("existing_count", existingCount))
	}
    s.WriteMutation(ctx, store.Document{
        ID: newDocID,
        DocumentType: "semantic_memory",
        Payload: []byte(compressed),
    })

	// Also index into vector space
	c.Compact(ctx, newDocID, compressed)
}
