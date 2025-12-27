package controller

import (
	"net/http"

	"github.com/armchr/codeapi/internal/codeapi"
	"github.com/armchr/codeapi/internal/model/ast"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// CodeAPIController handles HTTP requests for the CodeAPI
type CodeAPIController struct {
	api    codeapi.CodeAPI
	logger *zap.Logger
}

// NewCodeAPIController creates a new CodeAPIController
func NewCodeAPIController(api codeapi.CodeAPI, logger *zap.Logger) *CodeAPIController {
	return &CodeAPIController{
		api:    api,
		logger: logger,
	}
}

// -----------------------------------------------------------------------------
// Request/Response Types
// -----------------------------------------------------------------------------

// ListReposResponse is the response for listing repositories
type ListReposResponse struct {
	Repos []string `json:"repos"`
}

// ListFilesRequest is the request for listing files
type ListFilesRequest struct {
	RepoName string `json:"repo_name" binding:"required"`
	Limit    int    `json:"limit"`
	Offset   int    `json:"offset"`
}

// ListClassesRequest is the request for listing classes
type ListClassesRequest struct {
	RepoName string `json:"repo_name" binding:"required"`
	Limit    int    `json:"limit"`
	Offset   int    `json:"offset"`
}

// ListMethodsRequest is the request for listing methods
type ListMethodsRequest struct {
	RepoName string `json:"repo_name" binding:"required"`
	Limit    int    `json:"limit"`
	Offset   int    `json:"offset"`
}

// FindClassesRequest is the request for finding classes
type FindClassesRequest struct {
	RepoName string `json:"repo_name" binding:"required"`
	Name     string `json:"name"`
	NameLike string `json:"name_like"`
	FilePath string `json:"file_path"`
	FileID   *int32 `json:"file_id"`
	Limit    int    `json:"limit"`
	Offset   int    `json:"offset"`
}

// FindMethodsRequest is the request for finding methods
type FindMethodsRequest struct {
	RepoName  string      `json:"repo_name" binding:"required"`
	Name      string      `json:"name"`
	NameLike  string      `json:"name_like"`
	ClassName string      `json:"class_name"`
	ClassID   *ast.NodeID `json:"class_id"`
	FilePath  string      `json:"file_path"`
	FileID    *int32      `json:"file_id"`
	Limit     int         `json:"limit"`
	Offset    int         `json:"offset"`
}

// GetClassRequest is the request for getting a class by ID
type GetClassRequest struct {
	RepoName       string `json:"repo_name" binding:"required"`
	ClassID        int64  `json:"class_id" binding:"required"`
	IncludeMethods bool   `json:"include_methods"`
	IncludeFields  bool   `json:"include_fields"`
}

// GetMethodRequest is the request for getting a method by ID
type GetMethodRequest struct {
	RepoName string `json:"repo_name" binding:"required"`
	MethodID int64  `json:"method_id" binding:"required"`
}

// GetCallGraphRequest is the request for getting a call graph
type GetCallGraphRequest struct {
	RepoName        string `json:"repo_name" binding:"required"`
	FunctionID      int64  `json:"function_id"`
	FunctionName    string `json:"function_name"`
	ClassName       string `json:"class_name"`
	FilePath        string `json:"file_path"`
	Direction       string `json:"direction"` // "outgoing", "incoming", "both"
	MaxDepth        int    `json:"max_depth"`
	IncludeExternal bool   `json:"include_external"`
}

// GetDataDependentsRequest is the request for getting data dependents
type GetDataDependentsRequest struct {
	RepoName        string `json:"repo_name" binding:"required"`
	NodeID          int64  `json:"node_id"`
	VariableName    string `json:"variable_name"`
	FilePath        string `json:"file_path"`
	MaxDepth        int    `json:"max_depth"`
	IncludeIndirect bool   `json:"include_indirect"`
}

// GetImpactRequest is the request for impact analysis
type GetImpactRequest struct {
	RepoName         string `json:"repo_name" binding:"required"`
	NodeID           int64  `json:"node_id"`
	Name             string `json:"name"`
	NodeType         string `json:"node_type"` // "function", "class", "field", "variable"
	FilePath         string `json:"file_path"`
	MaxDepth         int    `json:"max_depth"`
	IncludeCallGraph bool   `json:"include_call_graph"`
	IncludeDataFlow  bool   `json:"include_data_flow"`
}

// ExecuteCypherRequest is the request for executing raw Cypher
type ExecuteCypherRequest struct {
	Query  string         `json:"query" binding:"required"`
	Params map[string]any `json:"params"`
}

// -----------------------------------------------------------------------------
// Reader Endpoints
// -----------------------------------------------------------------------------

// ListRepos returns all available repositories
func (c *CodeAPIController) ListRepos(ctx *gin.Context) {
	repos, err := c.api.Reader().ListRepos(ctx.Request.Context())
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, ListReposResponse{Repos: repos})
}

// ListFiles returns files in a repository
func (c *CodeAPIController) ListFiles(ctx *gin.Context) {
	var req ListFilesRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	files, err := c.api.Reader().Repo(req.RepoName).ListFiles(ctx.Request.Context(), req.Limit, req.Offset)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"files": files})
}

// ListClasses returns classes in a repository
func (c *CodeAPIController) ListClasses(ctx *gin.Context) {
	var req ListClassesRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	classes, err := c.api.Reader().Repo(req.RepoName).ListClasses(ctx.Request.Context(), req.Limit, req.Offset)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"classes": classes})
}

// ListMethods returns methods in a repository
func (c *CodeAPIController) ListMethods(ctx *gin.Context) {
	var req ListMethodsRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	methods, err := c.api.Reader().Repo(req.RepoName).ListMethods(ctx.Request.Context(), req.Limit, req.Offset)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"methods": methods})
}

// ListFunctions returns top-level functions in a repository
func (c *CodeAPIController) ListFunctions(ctx *gin.Context) {
	var req ListMethodsRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.logger.Debug("ListFunctions called",
		zap.String("repo_name", req.RepoName),
		zap.Int("limit", req.Limit),
		zap.Int("offset", req.Offset))

	functions, err := c.api.Reader().Repo(req.RepoName).ListFunctions(ctx.Request.Context(), req.Limit, req.Offset)
	if err != nil {
		c.logger.Error("ListFunctions failed", zap.Error(err))
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.logger.Debug("ListFunctions completed",
		zap.String("repo_name", req.RepoName),
		zap.Int("result_count", len(functions)))

	ctx.JSON(http.StatusOK, gin.H{"functions": functions})
}

// FindClasses finds classes matching criteria
func (c *CodeAPIController) FindClasses(ctx *gin.Context) {
	var req FindClassesRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	filter := codeapi.ClassFilter{
		Name:     req.Name,
		NameLike: req.NameLike,
		FilePath: req.FilePath,
		FileID:   req.FileID,
		Limit:    req.Limit,
		Offset:   req.Offset,
	}

	classes, err := c.api.Reader().Repo(req.RepoName).FindClasses(ctx.Request.Context(), filter)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"classes": classes})
}

// FindMethods finds methods matching criteria
func (c *CodeAPIController) FindMethods(ctx *gin.Context) {
	var req FindMethodsRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	filter := codeapi.MethodFilter{
		Name:      req.Name,
		NameLike:  req.NameLike,
		ClassName: req.ClassName,
		ClassID:   req.ClassID,
		FilePath:  req.FilePath,
		FileID:    req.FileID,
		Limit:     req.Limit,
		Offset:    req.Offset,
	}

	methods, err := c.api.Reader().Repo(req.RepoName).FindMethods(ctx.Request.Context(), filter)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"methods": methods})
}

// GetClass returns a class by ID
func (c *CodeAPIController) GetClass(ctx *gin.Context) {
	var req GetClassRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	repo := c.api.Reader().Repo(req.RepoName)
	var class *codeapi.ClassInfo
	var err error

	if req.IncludeMethods || req.IncludeFields {
		class, err = repo.GetClassFull(ctx.Request.Context(), ast.NodeID(req.ClassID), codeapi.LoadOptions{
			IncludeMethods: req.IncludeMethods,
			IncludeFields:  req.IncludeFields,
		})
	} else {
		class, err = repo.GetClass(ctx.Request.Context(), ast.NodeID(req.ClassID))
	}

	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"class": class})
}

// GetMethod returns a method by ID
func (c *CodeAPIController) GetMethod(ctx *gin.Context) {
	var req GetMethodRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	method, err := c.api.Reader().Repo(req.RepoName).GetMethod(ctx.Request.Context(), ast.NodeID(req.MethodID))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"method": method})
}

// GetClassMethods returns methods belonging to a class
func (c *CodeAPIController) GetClassMethods(ctx *gin.Context) {
	var req GetClassRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	methods, err := c.api.Reader().Repo(req.RepoName).GetClassMethods(ctx.Request.Context(), ast.NodeID(req.ClassID))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"methods": methods})
}

// GetClassFields returns fields belonging to a class
func (c *CodeAPIController) GetClassFields(ctx *gin.Context) {
	var req GetClassRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	fields, err := c.api.Reader().Repo(req.RepoName).GetClassFields(ctx.Request.Context(), ast.NodeID(req.ClassID))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"fields": fields})
}

// -----------------------------------------------------------------------------
// Analyzer Endpoints
// -----------------------------------------------------------------------------

// GetCallGraph returns the call graph for a function
func (c *CodeAPIController) GetCallGraph(ctx *gin.Context) {
	var req GetCallGraphRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Set defaults
	if req.MaxDepth <= 0 {
		req.MaxDepth = 3
	}
	direction := codeapi.DirectionOutgoing
	switch req.Direction {
	case "incoming":
		direction = codeapi.DirectionIncoming
	case "both":
		direction = codeapi.DirectionBoth
	}

	opts := codeapi.CallGraphOptions{
		Direction:       direction,
		MaxDepth:        req.MaxDepth,
		IncludeExternal: req.IncludeExternal,
	}

	var callGraph *codeapi.CallGraph
	var err error

	if req.FunctionID != 0 {
		callGraph, err = c.api.Analyzer().GetCallGraph(ctx.Request.Context(), ast.NodeID(req.FunctionID), opts)
	} else if req.FunctionName != "" {
		callGraph, err = c.api.Analyzer().GetCallGraphByName(
			ctx.Request.Context(),
			req.RepoName, req.FilePath, req.ClassName, req.FunctionName,
			opts,
		)
	} else {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "either function_id or function_name is required"})
		return
	}

	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"call_graph": callGraph})
}

// GetCallers returns functions that call the specified function
func (c *CodeAPIController) GetCallers(ctx *gin.Context) {
	var req GetCallGraphRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.FunctionID == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "function_id is required"})
		return
	}

	if req.MaxDepth <= 0 {
		req.MaxDepth = 3
	}

	callGraph, err := c.api.Analyzer().GetCallers(ctx.Request.Context(), ast.NodeID(req.FunctionID), req.MaxDepth)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"call_graph": callGraph})
}

// GetCallees returns functions called by the specified function
func (c *CodeAPIController) GetCallees(ctx *gin.Context) {
	var req GetCallGraphRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.FunctionID == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "function_id is required"})
		return
	}

	if req.MaxDepth <= 0 {
		req.MaxDepth = 3
	}

	callGraph, err := c.api.Analyzer().GetCallees(ctx.Request.Context(), ast.NodeID(req.FunctionID), req.MaxDepth)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"call_graph": callGraph})
}

// GetDataDependents returns nodes that depend on a value
func (c *CodeAPIController) GetDataDependents(ctx *gin.Context) {
	var req GetDataDependentsRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	opts := codeapi.DependencyOptions{
		MaxDepth:        req.MaxDepth,
		IncludeIndirect: req.IncludeIndirect,
	}

	var graph *codeapi.DependencyGraph
	var err error

	if req.NodeID != 0 {
		graph, err = c.api.Analyzer().GetDataDependents(ctx.Request.Context(), ast.NodeID(req.NodeID), opts)
	} else if req.VariableName != "" {
		graph, err = c.api.Analyzer().GetVariableDependents(
			ctx.Request.Context(),
			req.RepoName, req.FilePath, req.VariableName,
			opts,
		)
	} else {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "either node_id or variable_name is required"})
		return
	}

	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"dependency_graph": graph})
}

// GetDataSources returns nodes that contribute to a value
func (c *CodeAPIController) GetDataSources(ctx *gin.Context) {
	var req GetDataDependentsRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.NodeID == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "node_id is required"})
		return
	}

	opts := codeapi.DependencyOptions{
		MaxDepth:        req.MaxDepth,
		IncludeIndirect: req.IncludeIndirect,
	}

	graph, err := c.api.Analyzer().GetDataSources(ctx.Request.Context(), ast.NodeID(req.NodeID), opts)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"dependency_graph": graph})
}

// GetImpact returns impact analysis for a node
func (c *CodeAPIController) GetImpact(ctx *gin.Context) {
	var req GetImpactRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.MaxDepth <= 0 {
		req.MaxDepth = 3
	}

	opts := codeapi.ImpactOptions{
		MaxDepth:         req.MaxDepth,
		IncludeCallGraph: req.IncludeCallGraph,
		IncludeDataFlow:  req.IncludeDataFlow,
	}

	var impact *codeapi.ImpactResult
	var err error

	if req.NodeID != 0 {
		impact, err = c.api.Analyzer().GetImpact(ctx.Request.Context(), ast.NodeID(req.NodeID), opts)
	} else if req.Name != "" {
		nodeType := ast.NodeTypeFunction
		switch req.NodeType {
		case "class":
			nodeType = ast.NodeTypeClass
		case "field":
			nodeType = ast.NodeTypeField
		case "variable":
			nodeType = ast.NodeTypeVariable
		}
		impact, err = c.api.Analyzer().GetImpactByName(
			ctx.Request.Context(),
			req.RepoName, req.FilePath, req.Name, nodeType,
			opts,
		)
	} else {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "either node_id or name is required"})
		return
	}

	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"impact": impact})
}

// GetInheritanceTree returns the inheritance hierarchy for a class
func (c *CodeAPIController) GetInheritanceTree(ctx *gin.Context) {
	var req GetClassRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tree, err := c.api.Analyzer().GetInheritanceTree(ctx.Request.Context(), ast.NodeID(req.ClassID))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"inheritance_tree": tree})
}

// GetFieldAccessors returns methods that access a field
func (c *CodeAPIController) GetFieldAccessors(ctx *gin.Context) {
	type FieldAccessorsRequest struct {
		RepoName  string `json:"repo_name" binding:"required"`
		FieldID   int64  `json:"field_id"`
		ClassName string `json:"class_name"`
		FieldName string `json:"field_name"`
	}

	var req FieldAccessorsRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var result *codeapi.FieldAccessResult
	var err error

	if req.FieldID != 0 {
		result, err = c.api.Analyzer().GetFieldAccessors(ctx.Request.Context(), ast.NodeID(req.FieldID))
	} else if req.ClassName != "" && req.FieldName != "" {
		result, err = c.api.Analyzer().GetFieldAccessorsByName(
			ctx.Request.Context(),
			req.RepoName, req.ClassName, req.FieldName,
		)
	} else {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "either field_id or (class_name and field_name) is required"})
		return
	}

	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"field_accessors": result})
}

// -----------------------------------------------------------------------------
// Raw Cypher Endpoints
// -----------------------------------------------------------------------------

// ExecuteCypher executes a raw read-only Cypher query
func (c *CodeAPIController) ExecuteCypher(ctx *gin.Context) {
	var req ExecuteCypherRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	results, err := c.api.ExecuteCypher(ctx.Request.Context(), req.Query, req.Params)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"results": results})
}

// ExecuteCypherWrite executes a raw write Cypher query
func (c *CodeAPIController) ExecuteCypherWrite(ctx *gin.Context) {
	var req ExecuteCypherRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	results, err := c.api.ExecuteCypherWrite(ctx.Request.Context(), req.Query, req.Params)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"results": results})
}
