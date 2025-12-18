// Package codeapi provides a clean read interface for querying the code graph.
// It offers two main interfaces:
//   - CodeReader: Repository-scoped entity queries (find classes, methods, fields)
//   - GraphAnalyzer: Graph traversals (call graphs, data dependencies)
//
// These are combined in the CodeAPI facade for convenient access.
package codeapi

import (
	"github.com/armchr/codeapi/internal/model/ast"
	"github.com/armchr/codeapi/pkg/lsp/base"
)

// Direction specifies traversal direction for graph queries
type Direction string

const (
	DirectionOutgoing Direction = "outgoing" // follow outgoing edges (e.g., callees)
	DirectionIncoming Direction = "incoming" // follow incoming edges (e.g., callers)
	DirectionBoth     Direction = "both"     // follow both directions
)

// Visibility represents field/method visibility
type Visibility string

const (
	VisibilityPublic    Visibility = "public"
	VisibilityPrivate   Visibility = "private"
	VisibilityProtected Visibility = "protected"
	VisibilityPackage   Visibility = "package" // Go's default, Java's package-private
)

// Location represents a position in source code
type Location struct {
	FilePath string
	FileID   int32
	Range    base.Range
}

// -----------------------------------------------------------------------------
// Entity Types - Rich representations of code elements
// -----------------------------------------------------------------------------

// ClassInfo contains information about a class/struct
type ClassInfo struct {
	ID       ast.NodeID
	Name     string
	FilePath string
	FileID   int32
	Range    base.Range
	Language string

	// Populated on demand
	Methods []*MethodInfo
	Fields  []*FieldInfo

	// Inheritance
	ParentClasses []ast.NodeID
	ChildClasses  []ast.NodeID
}

// MethodInfo contains information about a method/function
type MethodInfo struct {
	ID       ast.NodeID
	Name     string
	FilePath string
	FileID   int32
	Range    base.Range

	// Context
	ClassName   string
	ClassID     ast.NodeID
	IsMethod    bool // true if belongs to a class, false if top-level function

	// Signature
	Parameters []*ParameterInfo
	ReturnType string

	// Flags
	IsAccessor    bool
	IsConstructor bool
	IsStatic      bool
	Visibility    Visibility
}

// FieldInfo contains information about a class field
type FieldInfo struct {
	ID         ast.NodeID
	Name       string
	Type       string
	ClassID    ast.NodeID
	Range      base.Range
	Visibility Visibility
	IsStatic   bool
}

// ParameterInfo contains information about a function parameter
type ParameterInfo struct {
	ID       ast.NodeID
	Name     string
	Type     string
	Position int
}

// FileInfo contains information about a source file
type FileInfo struct {
	ID       ast.NodeID
	Path     string
	Language string
	FileID   int32
	RepoName string

	// Populated on demand
	Classes   []*ClassInfo
	Functions []*MethodInfo // top-level functions
}

// -----------------------------------------------------------------------------
// Filter Types - For querying entities
// -----------------------------------------------------------------------------

// ClassFilter specifies criteria for finding classes
type ClassFilter struct {
	Name     string // exact match
	NameLike string // pattern match (e.g., "*Service")
	FilePath string // exact file path
	FileID   *int32

	Limit  int
	Offset int
}

// MethodFilter specifies criteria for finding methods
type MethodFilter struct {
	Name      string
	NameLike  string
	ClassName string
	ClassID   *ast.NodeID
	FilePath  string
	FileID    *int32

	IsMethod   *bool // nil = any, true = methods only, false = functions only
	IsAccessor *bool

	Limit  int
	Offset int
}

// FieldFilter specifies criteria for finding fields
type FieldFilter struct {
	Name       string
	NameLike   string
	ClassName  string
	ClassID    *ast.NodeID
	Visibility *Visibility

	Limit  int
	Offset int
}

// FileFilter specifies criteria for finding files
type FileFilter struct {
	Path     string
	PathLike string // pattern match
	Language string

	Limit  int
	Offset int
}

// -----------------------------------------------------------------------------
// Graph Result Types - For traversal queries
// -----------------------------------------------------------------------------

// CallGraph represents a function call graph
type CallGraph struct {
	Root      *CallNode
	Nodes     map[ast.NodeID]*CallNode
	Edges     []*CallEdge
	Direction Direction
	MaxDepth  int
	Truncated bool // true if results were limited
}

// CallNode represents a function in the call graph
type CallNode struct {
	ID        ast.NodeID
	Name      string
	ClassName string // empty if top-level function
	FilePath  string
	FileID    int32
	Depth     int // distance from root
	Range     base.Range
}

// CallEdge represents a call relationship
type CallEdge struct {
	CallerID ast.NodeID
	CalleeID ast.NodeID
	CallSite *Location // where the call occurs
}

// DependencyGraph represents data dependencies
type DependencyGraph struct {
	Root      *DependencyNode
	Nodes     map[ast.NodeID]*DependencyNode
	Edges     []*DependencyEdge
	Direction Direction
	Truncated bool
}

// DependencyNode represents a node in the dependency graph
type DependencyNode struct {
	ID       ast.NodeID
	Name     string
	NodeType ast.NodeType
	FilePath string
	FileID   int32
	Depth    int
}

// DependencyEdge represents a data flow relationship
type DependencyEdge struct {
	SourceID ast.NodeID
	TargetID ast.NodeID
	FlowType string // "assignment", "parameter", "return", etc.
}

// InheritanceTree represents class inheritance relationships
type InheritanceTree struct {
	Root     *InheritanceNode
	Nodes    map[ast.NodeID]*InheritanceNode
	MaxDepth int
}

// InheritanceNode represents a class in the inheritance tree
type InheritanceNode struct {
	ID       ast.NodeID
	Name     string
	FilePath string
	Parents  []*InheritanceNode
	Children []*InheritanceNode
	Depth    int
}

// -----------------------------------------------------------------------------
// Options Types - For controlling query behavior
// -----------------------------------------------------------------------------

// LoadOptions controls what data to load with entities
type LoadOptions struct {
	IncludeMethods     bool
	IncludeFields      bool
	IncludeInheritance bool
	IncludeParameters  bool
}

// CallGraphOptions controls call graph traversal
type CallGraphOptions struct {
	Direction       Direction
	MaxDepth        int
	IncludeExternal bool         // include calls to external packages
	IncludeTests    bool         // include test files
	StopAt          []ast.NodeID // don't traverse past these nodes
}

// DefaultCallGraphOptions returns sensible defaults
func DefaultCallGraphOptions() CallGraphOptions {
	return CallGraphOptions{
		Direction:       DirectionOutgoing,
		MaxDepth:        3,
		IncludeExternal: false,
		IncludeTests:    false,
	}
}

// DependencyOptions controls dependency graph traversal
type DependencyOptions struct {
	MaxDepth        int
	IncludeIndirect bool           // transitive dependencies
	FilterTypes     []ast.NodeType // only return these node types
}

// DefaultDependencyOptions returns sensible defaults
func DefaultDependencyOptions() DependencyOptions {
	return DependencyOptions{
		MaxDepth:        -1, // unlimited
		IncludeIndirect: true,
	}
}
