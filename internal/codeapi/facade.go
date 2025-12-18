package codeapi

import (
	"context"

	"github.com/armchr/codeapi/internal/model/ast"
	"github.com/armchr/codeapi/internal/service/codegraph"

	"go.uber.org/zap"
)

// CodeAPI is the main entry point for querying the code graph.
// It provides access to both entity queries (CodeReader) and graph traversals (GraphAnalyzer).
type CodeAPI interface {
	// Reader returns the CodeReader for entity queries
	Reader() CodeReader

	// Analyzer returns the GraphAnalyzer for graph traversals
	Analyzer() GraphAnalyzer

	// --- Raw Query Access ---

	// ExecuteCypher executes a raw Cypher query and returns the results.
	// Use this for complex queries not covered by Reader or Analyzer.
	// The query should be read-only; use ExecuteCypherWrite for mutations.
	ExecuteCypher(ctx context.Context, query string, params map[string]any) ([]map[string]any, error)

	// ExecuteCypherWrite executes a write Cypher query (create, update, delete).
	// Returns the result records if any.
	ExecuteCypherWrite(ctx context.Context, query string, params map[string]any) ([]map[string]any, error)

	// --- Convenience Methods ---
	// These combine Reader and Analyzer for common use cases

	// GetClassWithCallGraph returns a class with its methods' call graphs
	//GetClassWithCallGraph(ctx context.Context, repoName string, classID ast.NodeID, callDepth int) (*ClassWithCallGraph, error)

	// GetMethodWithContext returns a method with its class context and call graph
	//GetMethodWithContext(ctx context.Context, repoName string, methodID ast.NodeID, callDepth int) (*MethodWithContext, error)

	// FindAndAnalyze finds an entity by name and returns it with analysis
	//FindAndAnalyze(ctx context.Context, req FindAndAnalyzeRequest) (*FindAndAnalyzeResult, error)
}

// ClassWithCallGraph combines class info with call graphs for its methods
type ClassWithCallGraph struct {
	Class       *ClassInfo
	MethodCalls map[ast.NodeID]*CallGraph // methodID -> call graph
}

// MethodWithContext provides method info with surrounding context
type MethodWithContext struct {
	Method  *MethodInfo
	Class   *ClassInfo // nil if top-level function
	File    *FileInfo
	Callers *CallGraph
	Callees *CallGraph
}

// FindAndAnalyzeRequest specifies what to find and analyze
type FindAndAnalyzeRequest struct {
	RepoName string
	FilePath string // optional, narrows search
	Name     string
	NodeType ast.NodeType // Function, Class, Field, Variable

	// Analysis options
	IncludeCallGraph bool
	CallGraphDepth   int
	IncludeDataFlow  bool
	DataFlowDepth    int
}

// FindAndAnalyzeResult contains the found entity with analysis
type FindAndAnalyzeResult struct {
	// The found entity (one of these will be set)
	Class    *ClassInfo
	Method   *MethodInfo
	Field    *FieldInfo
	Variable *VariableInfo

	// Analysis results
	CallGraph *CallGraph
	DataFlow  *DependencyGraph
	Impact    *ImpactResult
}

// VariableInfo contains information about a variable
type VariableInfo struct {
	ID       ast.NodeID
	Name     string
	Type     string
	FilePath string
	FileID   int32
	Range    Location
	Scope    string // "local", "parameter", "global"
}

// -----------------------------------------------------------------------------
// Implementation
// -----------------------------------------------------------------------------

// codeAPIImpl implements CodeAPI
type codeAPIImpl struct {
	reader   CodeReader
	analyzer GraphAnalyzer
	graph    *codegraph.CodeGraph
	logger   *zap.Logger
}

// NewCodeAPI creates a new CodeAPI instance backed by the given CodeGraph
func NewCodeAPI(graph *codegraph.CodeGraph, logger *zap.Logger) CodeAPI {
	reader := newCodeReaderImpl(graph, logger)
	analyzer := newGraphAnalyzerImpl(graph, logger)

	return &codeAPIImpl{
		reader:   reader,
		analyzer: analyzer,
		graph:    graph,
		logger:   logger,
	}
}

// Reader returns the CodeReader
func (api *codeAPIImpl) Reader() CodeReader {
	return api.reader
}

// Analyzer returns the GraphAnalyzer
func (api *codeAPIImpl) Analyzer() GraphAnalyzer {
	return api.analyzer
}

// ExecuteCypher executes a raw read-only Cypher query
func (api *codeAPIImpl) ExecuteCypher(ctx context.Context, query string, params map[string]any) ([]map[string]any, error) {
	return api.graph.ExecuteRead(ctx, query, params)
}

// ExecuteCypherWrite executes a raw write Cypher query
func (api *codeAPIImpl) ExecuteCypherWrite(ctx context.Context, query string, params map[string]any) ([]map[string]any, error) {
	return api.graph.ExecuteWrite(ctx, query, params)
}

// GetClassWithCallGraph returns a class with call graphs for all its methods
func (api *codeAPIImpl) GetClassWithCallGraph(ctx context.Context, repoName string, classID ast.NodeID, callDepth int) (*ClassWithCallGraph, error) {
	// Get the class with methods
	repo := api.reader.Repo(repoName)
	class, err := repo.GetClassFull(ctx, classID, LoadOptions{
		IncludeMethods: true,
		IncludeFields:  true,
	})
	if err != nil {
		return nil, err
	}

	// Build call graphs for each method
	methodCalls := make(map[ast.NodeID]*CallGraph)
	opts := CallGraphOptions{
		Direction: DirectionBoth,
		MaxDepth:  callDepth,
	}

	for _, method := range class.Methods {
		callGraph, err := api.analyzer.GetCallGraph(ctx, method.ID, opts)
		if err != nil {
			api.logger.Warn("Failed to get call graph for method",
				zap.String("method", method.Name),
				zap.Error(err))
			continue
		}
		methodCalls[method.ID] = callGraph
	}

	return &ClassWithCallGraph{
		Class:       class,
		MethodCalls: methodCalls,
	}, nil
}

// GetMethodWithContext returns a method with its surrounding context
func (api *codeAPIImpl) GetMethodWithContext(ctx context.Context, repoName string, methodID ast.NodeID, callDepth int) (*MethodWithContext, error) {
	repo := api.reader.Repo(repoName)

	// Get the method
	method, err := repo.GetMethod(ctx, methodID)
	if err != nil {
		return nil, err
	}

	result := &MethodWithContext{
		Method: method,
	}

	// Get the containing class (if any)
	if method.IsMethod && method.ClassID != 0 {
		class, err := repo.GetClass(ctx, method.ClassID)
		if err == nil {
			result.Class = class
		}
	}

	// Get the file
	file, err := repo.GetFileByPath(ctx, method.FilePath)
	if err == nil {
		result.File = file
	}

	// Get call graphs
	if callDepth > 0 {
		callers, _ := api.analyzer.GetCallers(ctx, methodID, callDepth)
		result.Callers = callers

		callees, _ := api.analyzer.GetCallees(ctx, methodID, callDepth)
		result.Callees = callees
	}

	return result, nil
}

// FindAndAnalyze finds an entity and runs analysis on it
func (api *codeAPIImpl) FindAndAnalyze(ctx context.Context, req FindAndAnalyzeRequest) (*FindAndAnalyzeResult, error) {
	repo := api.reader.Repo(req.RepoName)
	result := &FindAndAnalyzeResult{}

	var nodeID ast.NodeID

	// Find the entity based on type
	switch req.NodeType {
	case ast.NodeTypeClass:
		var class *ClassInfo
		var err error
		if req.FilePath != "" {
			class, err = repo.File(req.FilePath).FindClassByName(ctx, req.Name)
		} else {
			class, err = repo.FindClassByName(ctx, req.Name)
		}
		if err != nil {
			return nil, err
		}
		result.Class = class
		nodeID = class.ID

	case ast.NodeTypeFunction:
		var method *MethodInfo
		var err error
		if req.FilePath != "" {
			method, err = repo.File(req.FilePath).FindMethodByName(ctx, req.Name)
		} else {
			method, err = repo.FindMethodByName(ctx, req.Name, "")
		}
		if err != nil {
			return nil, err
		}
		result.Method = method
		nodeID = method.ID

	case ast.NodeTypeField:
		fields, err := repo.FindFields(ctx, FieldFilter{Name: req.Name})
		if err != nil {
			return nil, err
		}
		if len(fields) > 0 {
			result.Field = fields[0]
			nodeID = fields[0].ID
		}
	}

	// Run requested analysis
	if req.IncludeCallGraph && nodeID != 0 {
		opts := CallGraphOptions{
			Direction: DirectionBoth,
			MaxDepth:  req.CallGraphDepth,
		}
		callGraph, _ := api.analyzer.GetCallGraph(ctx, nodeID, opts)
		result.CallGraph = callGraph
	}

	if req.IncludeDataFlow && nodeID != 0 {
		opts := DependencyOptions{
			MaxDepth:        req.DataFlowDepth,
			IncludeIndirect: true,
		}
		dataFlow, _ := api.analyzer.GetDataDependents(ctx, nodeID, opts)
		result.DataFlow = dataFlow
	}

	return result, nil
}
