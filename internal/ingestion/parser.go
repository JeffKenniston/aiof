package ingestion

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// Embedder generates vector embeddings for text content.
type Embedder interface {
	EmbedText(ctx context.Context, text string) ([]float32, error)
}

// ASTParser traverses the repository and extracts structural components using go/ast.
type ASTParser struct {
	logger   *slog.Logger
	memory   *DualIndexMemory
	embedder Embedder
}

func NewASTParser(logger *slog.Logger, memory *DualIndexMemory) *ASTParser {
	return &ASTParser{
		logger: logger,
		memory: memory,
	}
}

// ParseRepository walks the given path and ingests source files into the Dual-Index Memory.
func (p *ASTParser) ParseRepository(ctx context.Context, rootPath string) error {
	p.logger.Info("Starting AST parsing for repository", slog.String("root", rootPath))

	fset := token.NewFileSet()

	return filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			// Basic ignore list instead of full .gitignore parsing to save complexity
			// for this enterprise implementation.
			name := info.Name()
			if name == ".git" || name == "node_modules" || name == "vendor" || name == ".idea" || name == "dist" {
				return filepath.SkipDir
			}
			return nil
		}

		if strings.HasSuffix(info.Name(), ".go") {
			p.logger.Debug("Ingesting file via AST", slog.String("file", path))
			
			// Actual Go AST Parsing
			node, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
			if err != nil {
				p.logger.Error("Failed to parse file AST", slog.String("file", path), slog.Any("error", err))
				return nil // Skip invalid files
			}
			
			var dependencies []string
			for _, imp := range node.Imports {
				dependencies = append(dependencies, imp.Path.Value)
			}
			
			// Convert raw AST into semantic document (simplified)
			contentBuilder := strings.Builder{}
			ast.Inspect(node, func(n ast.Node) bool {
				switch x := n.(type) {
				case *ast.FuncDecl:
					contentBuilder.WriteString("Function: " + x.Name.Name + "\n")
				case *ast.TypeSpec:
					contentBuilder.WriteString("Type: " + x.Name.Name + "\n")
				}
				return true
			})

			content := contentBuilder.String()
			
			var realEmbedding []float32
			if p.embedder != nil {
				emb, err := p.embedder.EmbedText(ctx, content)
				if err != nil {
					p.logger.Warn("embedder failed, using zero vectors", slog.Any("error", err))
					realEmbedding = make([]float32, 768)
				} else {
					realEmbedding = emb
				}
			} else {
				p.logger.Warn("no embedder configured, using zero vectors")
				realEmbedding = make([]float32, 768)
			}
			
			if p.memory != nil {
				if err := p.memory.UpsertEntity(ctx, path, content, map[string]string{"type": "go_source"}, realEmbedding, dependencies, "SEMANTIC"); err != nil {
					p.logger.Error("Failed to upsert parsed AST", slog.String("file", path), slog.Any("error", err))
				}
			}
		}
		return nil
	})
}
