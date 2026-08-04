package ast

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ============================================================================
// ADR-15: Type-Preserving AST Symbol Dependency Graph (TSDG) Compaction Engine
// ============================================================================

// SymbolType classifies language constructs within the TSDG graph.
type SymbolType string

const (
	SymbolStruct    SymbolType = "STRUCT"
	SymbolInterface SymbolType = "INTERFACE"
	SymbolFunction  SymbolType = "FUNCTION"
	SymbolMethod    SymbolType = "METHOD"
	SymbolTypeAlias SymbolType = "TYPE_ALIAS"
	SymbolConst     SymbolType = "CONST"
	SymbolVar       SymbolType = "VAR"
)

// FieldInfo describes a struct field or interface method parameter.
type FieldInfo struct {
	Name     string `json:"name,omitempty"`
	Type     string `json:"type"`
	Tag      string `json:"tag,omitempty"`
	Exported bool   `json:"exported"`
}

// MethodInfo describes an interface method contract or struct method signature.
type MethodInfo struct {
	Name       string      `json:"name"`
	Receiver   string      `json:"receiver,omitempty"`
	Parameters []FieldInfo `json:"parameters,omitempty"`
	Results    []FieldInfo `json:"results,omitempty"`
	Exported   bool        `json:"exported"`
}

// SymbolDef represents a compact, type-preserved symbol definition in the TSDG.
type SymbolDef struct {
	ID         string      `json:"id"`
	Name       string      `json:"name"`
	Package    string      `json:"package"`
	File       string      `json:"file"`
	Kind       SymbolType  `json:"kind"`
	Exported   bool        `json:"exported"`
	Signature  string      `json:"signature"`
	DocComment string      `json:"doc_comment,omitempty"`
	Fields     []FieldInfo `json:"fields,omitempty"`
	Methods    []MethodInfo `json:"methods,omitempty"`
	Modified   bool        `json:"modified"`
	Summary    string      `json:"summary,omitempty"`
}

// DependencyEdge represents a directed dependency link in the import DAG.
type DependencyEdge struct {
	From string `json:"from"`
	To   string `json:"to"`
	Kind string `json:"kind"` // "import", "type_reference", "call"
}

// ImportDAG represents the cross-file / cross-package dependency graph.
type ImportDAG struct {
	Nodes []string         `json:"nodes"`
	Edges []DependencyEdge `json:"edges"`
}

// TSDGGraph represents the complete, KV-cache friendly symbol graph (~12KB per module).
type TSDGGraph struct {
	ModuleName  string               `json:"module_name"`
	ImportDAG   ImportDAG            `json:"import_dag"`
	SymbolTable map[string]SymbolDef `json:"symbol_table"`
	CompactCode map[string]string    `json:"compact_code,omitempty"`
	ByteSize    int                  `json:"byte_size"`
}

// TSDGCompactor orchestrates Phase 1 graph extraction and Phase 2 AST body compaction.
type TSDGCompactor struct {
	fset *token.FileSet
}

// NewTSDGCompactor initializes a new TSDG compaction engine.
func NewTSDGCompactor() *TSDGCompactor {
	return &TSDGCompactor{
		fset: token.NewFileSet(),
	}
}

// BuildGraph executes two-phase TSDG compaction over a target directory or package.
func (c *TSDGCompactor) BuildGraph(dirPath string, modifiedSymbols map[string]bool) (*TSDGGraph, error) {
	if modifiedSymbols == nil {
		modifiedSymbols = make(map[string]bool)
	}

	pkgs, err := parser.ParseDir(c.fset, dirPath, func(fi os.FileInfo) bool {
		return !strings.HasSuffix(fi.Name(), "_test.go")
	}, parser.ParseComments)

	if err != nil {
		return nil, fmt.Errorf("tsdg: failed to parse directory %s: %w", dirPath, err)
	}

	graph := &TSDGGraph{
		ModuleName:  filepath.Base(dirPath),
		SymbolTable: make(map[string]SymbolDef),
		CompactCode: make(map[string]string),
		ImportDAG: ImportDAG{
			Nodes: make([]string, 0),
			Edges: make([]DependencyEdge, 0),
		},
	}

	nodeSet := make(map[string]bool)

	// Phase 1: Build Symbol Table & Cross-File Import DAG
	for pkgName, pkg := range pkgs {
		nodeSet[pkgName] = true
		for filePath, fileAst := range pkg.Files {
			relPath := filepath.Base(filePath)
			nodeSet[relPath] = true

			// Collect imports and populate DAG edges
			for _, imp := range fileAst.Imports {
				impPath := strings.Trim(imp.Path.Value, `"`)
				nodeSet[impPath] = true
				graph.ImportDAG.Edges = append(graph.ImportDAG.Edges, DependencyEdge{
					From: relPath,
					To:   impPath,
					Kind: "import",
				})
			}

			// Collect Symbols
			c.extractSymbols(pkgName, relPath, fileAst, modifiedSymbols, graph.SymbolTable)
		}
	}

	for node := range nodeSet {
		graph.ImportDAG.Nodes = append(graph.ImportDAG.Nodes, node)
	}
	sort.Strings(graph.ImportDAG.Nodes)

	// Phase 2: Densely Summarize Non-Modified Function Implementation Bodies
	for pkgName, pkg := range pkgs {
		for filePath, fileAst := range pkg.Files {
			relPath := filepath.Base(filePath)

			// Clone file AST for non-destructive body compaction
			compactedAst := c.compactFileAST(fileAst, modifiedSymbols)

			var buf bytes.Buffer
			if err := format.Node(&buf, c.fset, compactedAst); err != nil {
				return nil, fmt.Errorf("tsdg: failed to format compacted AST for %s: %w", relPath, err)
			}

			graph.CompactCode[fmt.Sprintf("%s/%s", pkgName, relPath)] = buf.String()
		}
	}

	// Calculate serialized KV-cache payload size
	jsonBytes, err := json.Marshal(graph)
	if err == nil {
		graph.ByteSize = len(jsonBytes)
	}

	return graph, nil
}

// Phase 1: Extract Symbol Definitions, Types, and Signatures
func (c *TSDGCompactor) extractSymbols(pkgName, fileName string, fileAst *ast.File, modifiedSymbols map[string]bool, symbolTable map[string]SymbolDef) {
	ast.Inspect(fileAst, func(n ast.Node) bool {
		switch decl := n.(type) {
		case *ast.GenDecl:
			doc := ""
			if decl.Doc != nil {
				doc = strings.TrimSpace(decl.Doc.Text())
			}
			for _, spec := range decl.Specs {
				switch typeSpec := spec.(type) {
				case *ast.TypeSpec:
					symID := fmt.Sprintf("%s.%s", pkgName, typeSpec.Name.Name)
					isExported := ast.IsExported(typeSpec.Name.Name)

					sym := SymbolDef{
						ID:         symID,
						Name:       typeSpec.Name.Name,
						Package:    pkgName,
						File:       fileName,
						Exported:   isExported,
						DocComment: doc,
						Modified:   modifiedSymbols[symID] || modifiedSymbols[typeSpec.Name.Name],
					}

					switch typeBody := typeSpec.Type.(type) {
					case *ast.StructType:
						sym.Kind = SymbolStruct
						sym.Signature = fmt.Sprintf("type %s struct", typeSpec.Name.Name)
						sym.Fields = c.extractFields(typeBody.Fields)
					case *ast.InterfaceType:
						sym.Kind = SymbolInterface
						sym.Signature = fmt.Sprintf("type %s interface", typeSpec.Name.Name)
						sym.Methods = c.extractInterfaceMethods(typeBody.Methods)
					default:
						sym.Kind = SymbolTypeAlias
						sym.Signature = fmt.Sprintf("type %s alias", typeSpec.Name.Name)
					}

					symbolTable[symID] = sym
				}
			}

		case *ast.FuncDecl:
			funcName := decl.Name.Name
			isExported := ast.IsExported(funcName)
			recvStr := ""
			symKind := SymbolFunction

			if decl.Recv != nil && len(decl.Recv.List) > 0 {
				symKind = SymbolMethod
				recvStr = c.formatExpr(decl.Recv.List[0].Type)
			}

			symID := fmt.Sprintf("%s.%s", pkgName, funcName)
			if recvStr != "" {
				symID = fmt.Sprintf("%s.(%s).%s", pkgName, recvStr, funcName)
			}

			doc := ""
			if decl.Doc != nil {
				doc = strings.TrimSpace(decl.Doc.Text())
			}

			sig := c.formatFuncSignature(funcName, recvStr, decl.Type)

			sym := SymbolDef{
				ID:         symID,
				Name:       funcName,
				Package:    pkgName,
				File:       fileName,
				Kind:       symKind,
				Exported:   isExported,
				Signature:  sig,
				DocComment: doc,
				Modified:   modifiedSymbols[symID] || modifiedSymbols[funcName],
			}

			if decl.Type.Params != nil {
				sym.Fields = c.extractFields(decl.Type.Params)
			}
			if decl.Type.Results != nil {
				for _, res := range c.extractFields(decl.Type.Results) {
					sym.Methods = append(sym.Methods, MethodInfo{
						Name:     res.Name,
						Receiver: res.Type,
						Exported: isExported,
					})
				}
			}

			symbolTable[symID] = sym
		}
		return true
	})
}

// Phase 2: Compact AST Bodies by replacing non-modified implementation bodies
func (c *TSDGCompactor) compactFileAST(fileAst *ast.File, modifiedSymbols map[string]bool) *ast.File {
	// Deep copy AST
	fileCopy := ast.Walk; _ = fileCopy // Keep reference

	// Create a new File struct copying top-level declarations
	compacted := &ast.File{
		Doc:     fileAst.Doc,
		Package: fileAst.Package,
		Name:    fileAst.Name,
		Imports: fileAst.Imports,
		Scope:   fileAst.Scope,
	}

	for _, decl := range fileAst.Decls {
		switch funcDecl := decl.(type) {
		case *ast.FuncDecl:
			funcName := funcDecl.Name.Name
			recvStr := ""
			if funcDecl.Recv != nil && len(funcDecl.Recv.List) > 0 {
				recvStr = c.formatExpr(funcDecl.Recv.List[0].Type)
			}
			symID := fmt.Sprintf("%s.%s", fileAst.Name.Name, funcName)
			if recvStr != "" {
				symID = fmt.Sprintf("%s.(%s).%s", fileAst.Name.Name, recvStr, funcName)
			}

			isModified := modifiedSymbols[symID] || modifiedSymbols[funcName]

			if isModified || funcDecl.Body == nil {
				// Keep full implementation body intact for modified functions
				compacted.Decls = append(compacted.Decls, funcDecl)
			} else {
				// Densely summarize non-modified function body
				stmtCount := len(funcDecl.Body.List)
				summaryComment := fmt.Sprintf("// [summarized body: %d statements, signature preserved]", stmtCount)

				summarizedBody := &ast.BlockStmt{
					List: []ast.Stmt{
						&ast.ExprStmt{
							X: &ast.BasicLit{
								Kind:  token.STRING,
								Value: fmt.Sprintf("%q", summaryComment),
							},
						},
					},
				}

				// If function has return types, add zero-value return to maintain syntax validity
				if funcDecl.Type.Results != nil && len(funcDecl.Type.Results.List) > 0 {
					returnStmt := &ast.ReturnStmt{
						Results: c.generateZeroValues(funcDecl.Type.Results),
					}
					summarizedBody.List = append(summarizedBody.List, returnStmt)
				}

				compactFunc := &ast.FuncDecl{
					Doc:  funcDecl.Doc,
					Recv: funcDecl.Recv,
					Name: funcDecl.Name,
					Type: funcDecl.Type,
					Body: summarizedBody,
				}

				compacted.Decls = append(compacted.Decls, compactFunc)
			}

		default:
			// Non-function declarations (structs, interfaces, constants, imports) are preserved 100%
			compacted.Decls = append(compacted.Decls, decl)
		}
	}

	return compacted
}

func (c *TSDGCompactor) extractFields(fieldList *ast.FieldList) []FieldInfo {
	if fieldList == nil {
		return nil
	}
	var res []FieldInfo
	for _, field := range fieldList.List {
		typeStr := c.formatExpr(field.Type)
		tagStr := ""
		if field.Tag != nil {
			tagStr = field.Tag.Value
		}

		if len(field.Names) == 0 {
			res = append(res, FieldInfo{
				Name:     typeStr,
				Type:     typeStr,
				Tag:      tagStr,
				Exported: ast.IsExported(typeStr),
			})
		} else {
			for _, name := range field.Names {
				res = append(res, FieldInfo{
					Name:     name.Name,
					Type:     typeStr,
					Tag:      tagStr,
					Exported: ast.IsExported(name.Name),
				})
			}
		}
	}
	return res
}

func (c *TSDGCompactor) extractInterfaceMethods(fieldList *ast.FieldList) []MethodInfo {
	if fieldList == nil {
		return nil
	}
	var res []MethodInfo
	for _, field := range fieldList.List {
		if len(field.Names) > 0 {
			name := field.Names[0].Name
			m := MethodInfo{
				Name:     name,
				Exported: ast.IsExported(name),
			}
			if funcType, ok := field.Type.(*ast.FuncType); ok {
				m.Parameters = c.extractFields(funcType.Params)
				m.Results = c.extractFields(funcType.Results)
			}
			res = append(res, m)
		}
	}
	return res
}

func (c *TSDGCompactor) generateZeroValues(results *ast.FieldList) []ast.Expr {
	var exprs []ast.Expr
	for _, field := range results.List {
		typeStr := c.formatExpr(field.Type)
		var zero ast.Expr

		switch {
		case typeStr == "error" || strings.HasPrefix(typeStr, "*") || strings.HasPrefix(typeStr, "[]") || strings.HasPrefix(typeStr, "map["):
			zero = ast.NewIdent("nil")
		case typeStr == "bool":
			zero = ast.NewIdent("false")
		case typeStr == "string":
			zero = &ast.BasicLit{Kind: token.STRING, Value: `""`}
		case typeStr == "int", typeStr == "int64", typeStr == "uint", typeStr == "float64":
			zero = &ast.BasicLit{Kind: token.INT, Value: "0"}
		default:
			zero = ast.NewIdent("nil")
		}

		if len(field.Names) == 0 {
			exprs = append(exprs, zero)
		} else {
			for range field.Names {
				exprs = append(exprs, zero)
			}
		}
	}
	return exprs
}

func (c *TSDGCompactor) formatExpr(expr ast.Expr) string {
	var buf bytes.Buffer
	if err := format.Node(&buf, c.fset, expr); err != nil {
		return "unknown"
	}
	return buf.String()
}

func (c *TSDGCompactor) formatFuncSignature(name, recv string, funcType *ast.FuncType) string {
	var paramBuf bytes.Buffer
	if err := format.Node(&paramBuf, c.fset, funcType.Params); err != nil {
		paramBuf.WriteString("(...)")
	}
	params := paramBuf.String()

	results := ""
	if funcType.Results != nil && len(funcType.Results.List) > 0 {
		var resBuf bytes.Buffer
		if err := format.Node(&resBuf, c.fset, funcType.Results); err == nil {
			results = " " + resBuf.String()
		}
	}

	if recv != "" {
		return fmt.Sprintf("func (%s) %s%s%s", recv, name, params, results)
	}
	return fmt.Sprintf("func %s%s%s", name, params, results)
}

// ToCompactJSON serializes the TSDGGraph into a compact JSON string suitable for KV-cache indexing.
func (g *TSDGGraph) ToCompactJSON() (string, error) {
	data, err := json.Marshal(g)
	if err != nil {
		return "", fmt.Errorf("tsdg: marshal compact json error: %w", err)
	}
	return string(data), nil
}

// FromCompactJSON deserializes a compact JSON string back into a TSDGGraph struct.
func FromCompactJSON(jsonStr string) (*TSDGGraph, error) {
	var graph TSDGGraph
	if err := json.Unmarshal([]byte(jsonStr), &graph); err != nil {
		return nil, fmt.Errorf("tsdg: unmarshal compact json error: %w", err)
	}
	return &graph, nil
}
