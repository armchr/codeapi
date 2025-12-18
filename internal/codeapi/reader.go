package codeapi

import (
	"context"

	"github.com/armchr/codeapi/internal/model/ast"
)

// CodeReader provides repository-scoped access to code entities.
// It follows a hierarchical pattern: CodeReader → RepoReader → FileReader
type CodeReader interface {
	// Repo returns a reader scoped to a specific repository
	Repo(name string) RepoReader

	// ListRepos returns all available repository names
	ListRepos(ctx context.Context) ([]string, error)
}

// RepoReader provides access to code entities within a repository
type RepoReader interface {
	// Name returns the repository name
	Name() string

	// --- File Operations ---

	// ListFiles returns files in the repository with pagination
	ListFiles(ctx context.Context, limit, offset int) ([]*FileInfo, error)

	// FindFiles returns files matching the filter
	FindFiles(ctx context.Context, filter FileFilter) ([]*FileInfo, error)

	// GetFile returns a file by its ID
	GetFile(ctx context.Context, id ast.NodeID) (*FileInfo, error)

	// GetFileByPath returns a file by its path
	GetFileByPath(ctx context.Context, path string) (*FileInfo, error)

	// File returns a reader scoped to a specific file
	File(path string) FileReader

	// FileByID returns a reader scoped to a specific file by ID
	FileByID(id int32) FileReader

	// --- Class Operations ---

	// ListClasses returns classes in the repository with pagination
	ListClasses(ctx context.Context, limit, offset int) ([]*ClassInfo, error)

	// FindClasses returns classes matching the filter
	FindClasses(ctx context.Context, filter ClassFilter) ([]*ClassInfo, error)

	// GetClass returns a class by its ID
	GetClass(ctx context.Context, id ast.NodeID) (*ClassInfo, error)

	// GetClassFull returns a class with methods and fields populated
	GetClassFull(ctx context.Context, id ast.NodeID, opts LoadOptions) (*ClassInfo, error)

	// FindClassByName finds a class by name (returns first match)
	FindClassByName(ctx context.Context, name string) (*ClassInfo, error)

	// --- Method/Function Operations ---

	// ListMethods returns methods in the repository with pagination
	ListMethods(ctx context.Context, limit, offset int) ([]*MethodInfo, error)

	// ListFunctions returns top-level functions (not class methods) with pagination
	ListFunctions(ctx context.Context, limit, offset int) ([]*MethodInfo, error)

	// FindMethods returns methods matching the filter
	FindMethods(ctx context.Context, filter MethodFilter) ([]*MethodInfo, error)

	// GetMethod returns a method by its ID
	GetMethod(ctx context.Context, id ast.NodeID) (*MethodInfo, error)

	// FindMethodByName finds a method by name and optional class
	FindMethodByName(ctx context.Context, methodName string, className string) (*MethodInfo, error)

	// --- Field Operations ---

	// FindFields returns fields matching the filter
	FindFields(ctx context.Context, filter FieldFilter) ([]*FieldInfo, error)

	// GetField returns a field by its ID
	GetField(ctx context.Context, id ast.NodeID) (*FieldInfo, error)

	// --- Relationship Queries ---

	// GetClassMethods returns all methods belonging to a class
	GetClassMethods(ctx context.Context, classID ast.NodeID) ([]*MethodInfo, error)

	// GetClassFields returns all fields belonging to a class
	GetClassFields(ctx context.Context, classID ast.NodeID) ([]*FieldInfo, error)

	// GetMethodClass returns the class containing a method (nil if top-level function)
	GetMethodClass(ctx context.Context, methodID ast.NodeID) (*ClassInfo, error)
}

// FileReader provides access to code entities within a specific file
type FileReader interface {
	// Path returns the file path
	Path() string

	// FileID returns the file ID (0 if not yet resolved)
	FileID() int32

	// Info returns the file info
	Info(ctx context.Context) (*FileInfo, error)

	// --- Class Operations ---

	// ListClasses returns all classes in this file
	ListClasses(ctx context.Context) ([]*ClassInfo, error)

	// FindClassByName finds a class by name in this file
	FindClassByName(ctx context.Context, name string) (*ClassInfo, error)

	// --- Method/Function Operations ---

	// ListMethods returns all methods (class methods) in this file
	ListMethods(ctx context.Context) ([]*MethodInfo, error)

	// ListFunctions returns all top-level functions in this file
	ListFunctions(ctx context.Context) ([]*MethodInfo, error)

	// FindMethodByName finds a method/function by name in this file
	FindMethodByName(ctx context.Context, name string) (*MethodInfo, error)

	// FindMethodInClass finds a method by name within a specific class
	FindMethodInClass(ctx context.Context, methodName, className string) (*MethodInfo, error)

	// --- Field Operations ---

	// ListFields returns all fields in this file
	ListFields(ctx context.Context) ([]*FieldInfo, error)

	// FindFieldByName finds a field by name, optionally scoped to a class
	FindFieldByName(ctx context.Context, fieldName, className string) (*FieldInfo, error)
}
