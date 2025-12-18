package codeapi

import (
	"context"

	"github.com/armchr/codeapi/internal/model/ast"
)

// GraphAnalyzer provides graph traversal operations on the code graph.
// It supports call graph analysis, data flow tracking, and inheritance queries.
type GraphAnalyzer interface {
	// --- Call Graph Operations ---

	// GetCallGraph returns the call graph starting from a function.
	// Use opts.Direction to control whether to get callees, callers, or both.
	GetCallGraph(ctx context.Context, functionID ast.NodeID, opts CallGraphOptions) (*CallGraph, error)

	// GetCallGraphByName finds a function by name and returns its call graph.
	// If className is empty, searches for top-level functions.
	// If filePath is empty, searches across all files in the repo.
	GetCallGraphByName(ctx context.Context, repoName, filePath, className, functionName string, opts CallGraphOptions) (*CallGraph, error)

	// GetCallers returns functions that call the specified function (convenience method).
	// Equivalent to GetCallGraph with Direction=Incoming.
	GetCallers(ctx context.Context, functionID ast.NodeID, maxDepth int) (*CallGraph, error)

	// GetCallees returns functions called by the specified function (convenience method).
	// Equivalent to GetCallGraph with Direction=Outgoing.
	GetCallees(ctx context.Context, functionID ast.NodeID, maxDepth int) (*CallGraph, error)

	// --- Data Flow Operations ---

	// GetDataDependents returns nodes that depend on the value of the specified node.
	// Follows DATA_FLOW edges outward to find what uses this value.
	GetDataDependents(ctx context.Context, nodeID ast.NodeID, opts DependencyOptions) (*DependencyGraph, error)

	// GetDataSources returns nodes that contribute to the value of the specified node.
	// Follows DATA_FLOW edges backward to find where values come from.
	GetDataSources(ctx context.Context, nodeID ast.NodeID, opts DependencyOptions) (*DependencyGraph, error)

	// GetVariableDependents returns functions/methods that depend on a variable's value.
	// This is a higher-level query that finds the variable and traces its usage.
	GetVariableDependents(ctx context.Context, repoName, filePath, variableName string, opts DependencyOptions) (*DependencyGraph, error)

	// --- Field Access Operations ---

	// GetFieldAccessors returns methods that read or write a specific field.
	GetFieldAccessors(ctx context.Context, fieldID ast.NodeID) (*FieldAccessResult, error)

	// GetFieldAccessorsByName finds a field by name and returns its accessors.
	GetFieldAccessorsByName(ctx context.Context, repoName, className, fieldName string) (*FieldAccessResult, error)

	// --- Inheritance Operations ---

	// GetInheritanceTree returns the inheritance hierarchy for a class.
	// Includes both ancestors (parents) and descendants (children).
	GetInheritanceTree(ctx context.Context, classID ast.NodeID) (*InheritanceTree, error)

	// GetParentClasses returns direct and indirect parent classes.
	GetParentClasses(ctx context.Context, classID ast.NodeID, maxDepth int) ([]*ClassInfo, error)

	// GetChildClasses returns direct and indirect child classes.
	GetChildClasses(ctx context.Context, classID ast.NodeID, maxDepth int) ([]*ClassInfo, error)

	// --- Impact Analysis ---

	// GetImpact returns all code elements that could be affected by changes to the specified node.
	// This combines call graph and data flow analysis.
	GetImpact(ctx context.Context, nodeID ast.NodeID, opts ImpactOptions) (*ImpactResult, error)

	// GetImpactByName is a convenience method for impact analysis by name.
	GetImpactByName(ctx context.Context, repoName, filePath, name string, nodeType ast.NodeType, opts ImpactOptions) (*ImpactResult, error)
}

// FieldAccessResult contains methods that access a field
type FieldAccessResult struct {
	Field   *FieldInfo
	Readers []*MethodAccessInfo // methods that read this field
	Writers []*MethodAccessInfo // methods that write this field
}

// MethodAccessInfo contains info about a method that accesses a field
type MethodAccessInfo struct {
	Method      *MethodInfo
	AccessCount int        // number of times the field is accessed
	Locations   []Location // where in the method the access occurs
}

// ImpactOptions controls impact analysis behavior
type ImpactOptions struct {
	MaxDepth         int  // max traversal depth (-1 for unlimited)
	IncludeCallGraph bool // include callers in impact
	IncludeDataFlow  bool // include data dependents in impact
	IncludeTests     bool // include test files
	Scope            ImpactScope
}

// ImpactScope defines the boundary for impact analysis
type ImpactScope string

const (
	ImpactScopeFile    ImpactScope = "file"    // only within the same file
	ImpactScopePackage ImpactScope = "package" // within the same package/module
	ImpactScopeRepo    ImpactScope = "repo"    // within the repository
	ImpactScopeAll     ImpactScope = "all"     // including external dependencies
)

// DefaultImpactOptions returns sensible defaults for impact analysis
func DefaultImpactOptions() ImpactOptions {
	return ImpactOptions{
		MaxDepth:         3,
		IncludeCallGraph: true,
		IncludeDataFlow:  true,
		IncludeTests:     false,
		Scope:            ImpactScopeRepo,
	}
}

// ImpactResult contains the result of impact analysis
type ImpactResult struct {
	// Source is the node being analyzed
	Source *ImpactNode

	// AffectedNodes are nodes that could be affected by changes
	AffectedNodes []*ImpactNode

	// AffectedByCallGraph are nodes affected via call relationships
	AffectedByCallGraph []*ImpactNode

	// AffectedByDataFlow are nodes affected via data dependencies
	AffectedByDataFlow []*ImpactNode

	// Summary statistics
	TotalAffected   int
	MaxDepthReached int
	Truncated       bool
}

// ImpactNode represents a node in the impact analysis
type ImpactNode struct {
	ID       ast.NodeID
	Name     string
	NodeType ast.NodeType
	FilePath string
	FileID   int32
	Depth    int
	Impact   ImpactType // how this node is affected
}

// ImpactType describes how a node is affected
type ImpactType string

const (
	ImpactTypeDirect   ImpactType = "direct"   // directly uses the source
	ImpactTypeTransitive ImpactType = "transitive" // indirectly affected
	ImpactTypeCallGraph  ImpactType = "call_graph" // affected via call relationship
	ImpactTypeDataFlow   ImpactType = "data_flow"  // affected via data dependency
)
