package codeapi

import (
	"context"
	"fmt"

	"github.com/armchr/codeapi/internal/model/ast"
	"github.com/armchr/codeapi/internal/service/codegraph"
	"github.com/armchr/codeapi/pkg/lsp/base"

	"go.uber.org/zap"
)

// -----------------------------------------------------------------------------
// CodeReader Implementation
// -----------------------------------------------------------------------------

type codeReaderImpl struct {
	graph  *codegraph.CodeGraph
	logger *zap.Logger
}

func newCodeReaderImpl(graph *codegraph.CodeGraph, logger *zap.Logger) *codeReaderImpl {
	return &codeReaderImpl{
		graph:  graph,
		logger: logger,
	}
}

func (r *codeReaderImpl) Repo(name string) RepoReader {
	return &repoReaderImpl{
		repoName: name,
		graph:    r.graph,
		logger:   r.logger,
	}
}

func (r *codeReaderImpl) ListRepos(ctx context.Context) ([]string, error) {
	// Query all distinct repo names from FileScope nodes
	query := `
		MATCH (f:FileScope)
		RETURN DISTINCT f.repo AS repo
		ORDER BY repo
	`
	records, err := r.graph.ExecuteRead(ctx, query, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list repos: %w", err)
	}

	repos := make([]string, 0, len(records))
	for _, record := range records {
		if repo, ok := record["repo"].(string); ok {
			repos = append(repos, repo)
		}
	}
	return repos, nil
}

// -----------------------------------------------------------------------------
// RepoReader Implementation
// -----------------------------------------------------------------------------

type repoReaderImpl struct {
	repoName string
	graph    *codegraph.CodeGraph
	logger   *zap.Logger
}

func (r *repoReaderImpl) Name() string {
	return r.repoName
}

// --- File Operations ---

func (r *repoReaderImpl) ListFiles(ctx context.Context, limit, offset int) ([]*FileInfo, error) {
	return r.FindFiles(ctx, FileFilter{Limit: limit, Offset: offset})
}

func (r *repoReaderImpl) FindFiles(ctx context.Context, filter FileFilter) ([]*FileInfo, error) {
	query := `
		MATCH (f:FileScope {repo: $repo})
	`
	params := map[string]any{"repo": r.repoName}

	// Add filter conditions
	conditions := []string{}
	if filter.Path != "" {
		conditions = append(conditions, "f.path = $path")
		params["path"] = filter.Path
	}
	if filter.PathLike != "" {
		conditions = append(conditions, "f.path CONTAINS $pathLike")
		params["pathLike"] = filter.PathLike
	}
	if filter.Language != "" {
		conditions = append(conditions, "f.language = $language")
		params["language"] = filter.Language
	}

	if len(conditions) > 0 {
		query += " WHERE "
		for i, cond := range conditions {
			if i > 0 {
				query += " AND "
			}
			query += cond
		}
	}

	query += " RETURN f ORDER BY f.path"

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", filter.Limit)
	}
	if filter.Offset > 0 {
		query += fmt.Sprintf(" SKIP %d", filter.Offset)
	}

	records, err := r.graph.ExecuteRead(ctx, query, params)
	if err != nil {
		return nil, fmt.Errorf("failed to find files: %w", err)
	}

	return r.recordsToFileInfos(records)
}

func (r *repoReaderImpl) GetFile(ctx context.Context, id ast.NodeID) (*FileInfo, error) {
	query := `
		MATCH (f:FileScope {id: $id, repo: $repo})
		RETURN f
	`
	records, err := r.graph.ExecuteRead(ctx, query, map[string]any{
		"id":   int64(id),
		"repo": r.repoName,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get file: %w", err)
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("file not found: %d", id)
	}

	files, err := r.recordsToFileInfos(records)
	if err != nil {
		return nil, err
	}
	return files[0], nil
}

func (r *repoReaderImpl) GetFileByPath(ctx context.Context, path string) (*FileInfo, error) {
	files, err := r.FindFiles(ctx, FileFilter{Path: path, Limit: 1})
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("file not found: %s", path)
	}
	return files[0], nil
}

func (r *repoReaderImpl) File(path string) FileReader {
	return &fileReaderImpl{
		repoName: r.repoName,
		filePath: path,
		fileID:   0, // will be resolved lazily
		graph:    r.graph,
		logger:   r.logger,
	}
}

func (r *repoReaderImpl) FileByID(id int32) FileReader {
	return &fileReaderImpl{
		repoName: r.repoName,
		filePath: "", // will be resolved lazily
		fileID:   id,
		graph:    r.graph,
		logger:   r.logger,
	}
}

// --- Class Operations ---

func (r *repoReaderImpl) ListClasses(ctx context.Context, limit, offset int) ([]*ClassInfo, error) {
	return r.FindClasses(ctx, ClassFilter{Limit: limit, Offset: offset})
}

func (r *repoReaderImpl) FindClasses(ctx context.Context, filter ClassFilter) ([]*ClassInfo, error) {
	// Join through FileScope to filter by repo (only FileScope has repo field)
	query := `
		MATCH (file:FileScope {repo: $repo})
		MATCH (c:Class {fileId: file.fileId})
	`
	params := map[string]any{"repo": r.repoName}

	conditions := []string{}
	if filter.Name != "" {
		conditions = append(conditions, "c.name = $name")
		params["name"] = filter.Name
	}
	if filter.NameLike != "" {
		conditions = append(conditions, "c.name CONTAINS $nameLike")
		params["nameLike"] = filter.NameLike
	}
	if filter.FileID != nil {
		conditions = append(conditions, "c.fileId = $fileId")
		params["fileId"] = *filter.FileID
	}

	if len(conditions) > 0 {
		query += " WHERE "
		for i, cond := range conditions {
			if i > 0 {
				query += " AND "
			}
			query += cond
		}
	}

	query += " RETURN c ORDER BY c.name"

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", filter.Limit)
	}
	if filter.Offset > 0 {
		query += fmt.Sprintf(" SKIP %d", filter.Offset)
	}

	records, err := r.graph.ExecuteRead(ctx, query, params)
	if err != nil {
		return nil, fmt.Errorf("failed to find classes: %w", err)
	}

	return r.recordsToClassInfos(records, "c")
}

func (r *repoReaderImpl) GetClass(ctx context.Context, id ast.NodeID) (*ClassInfo, error) {
	query := `
		MATCH (c:Class {id: $id})
		RETURN c
	`
	records, err := r.graph.ExecuteRead(ctx, query, map[string]any{"id": int64(id)})
	if err != nil {
		return nil, fmt.Errorf("failed to get class: %w", err)
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("class not found: %d", id)
	}

	classes, err := r.recordsToClassInfos(records, "c")
	if err != nil {
		return nil, err
	}
	return classes[0], nil
}

func (r *repoReaderImpl) GetClassFull(ctx context.Context, id ast.NodeID, opts LoadOptions) (*ClassInfo, error) {
	class, err := r.GetClass(ctx, id)
	if err != nil {
		return nil, err
	}

	if opts.IncludeMethods {
		methods, err := r.GetClassMethods(ctx, id)
		if err == nil {
			class.Methods = methods
		}
	}

	if opts.IncludeFields {
		fields, err := r.GetClassFields(ctx, id)
		if err == nil {
			class.Fields = fields
		}
	}

	if opts.IncludeInheritance {
		// TODO: Load parent/child classes
	}

	return class, nil
}

func (r *repoReaderImpl) FindClassByName(ctx context.Context, name string) (*ClassInfo, error) {
	classes, err := r.FindClasses(ctx, ClassFilter{Name: name, Limit: 1})
	if err != nil {
		return nil, err
	}
	if len(classes) == 0 {
		return nil, fmt.Errorf("class not found: %s", name)
	}
	return classes[0], nil
}

// --- Method Operations ---

func (r *repoReaderImpl) ListMethods(ctx context.Context, limit, offset int) ([]*MethodInfo, error) {
	return r.FindMethods(ctx, MethodFilter{Limit: limit, Offset: offset})
}

func (r *repoReaderImpl) ListFunctions(ctx context.Context, limit, offset int) ([]*MethodInfo, error) {
	r.logger.Debug("ListFunctions called",
		zap.String("repo", r.repoName),
		zap.Int("limit", limit),
		zap.Int("offset", offset))

	isMethod := false
	results, err := r.FindMethods(ctx, MethodFilter{IsMethod: &isMethod, Limit: limit, Offset: offset})

	r.logger.Debug("ListFunctions completed",
		zap.String("repo", r.repoName),
		zap.Int("result_count", len(results)),
		zap.Error(err))

	return results, err
}

func (r *repoReaderImpl) FindMethods(ctx context.Context, filter MethodFilter) ([]*MethodInfo, error) {
	r.logger.Debug("FindMethods called",
		zap.String("repo", r.repoName),
		zap.Any("filter", filter))

	// Join through FileScope to filter by repo (only FileScope has repo field)
	query := `
		MATCH (file:FileScope {repo: $repo})
		MATCH (m:Function {fileId: file.fileId})
	`
	params := map[string]any{"repo": r.repoName}

	conditions := []string{}
	if filter.Name != "" {
		conditions = append(conditions, "m.name = $name")
		params["name"] = filter.Name
	}
	if filter.NameLike != "" {
		conditions = append(conditions, "m.name CONTAINS $nameLike")
		params["nameLike"] = filter.NameLike
	}
	if filter.FileID != nil {
		conditions = append(conditions, "m.fileId = $fileId")
		params["fileId"] = *filter.FileID
	}
	if filter.ClassID != nil {
		conditions = append(conditions, "EXISTS { MATCH (c:Class {id: $classId})-[:CONTAINS]->(m) }")
		params["classId"] = int64(*filter.ClassID)
	}
	// Filter by IsMethod: true = methods only (contained by Class), false = functions only (not contained by Class)
	if filter.IsMethod != nil {
		if *filter.IsMethod {
			conditions = append(conditions, "EXISTS { MATCH (c:Class)-[:CONTAINS]->(m) }")
		}/* else {
			conditions = append(conditions, "NOT EXISTS { MATCH (c:Class)-[:CONTAINS]->(m) }")
		}*/
	}

	if len(conditions) > 0 {
		query += " WHERE "
		for i, cond := range conditions {
			if i > 0 {
				query += " AND "
			}
			query += cond
		}
	}

	query += " RETURN m ORDER BY m.name"

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", filter.Limit)
	}

	r.logger.Debug("FindMethods executing query",
		zap.String("query", query),
		zap.Any("params", params))

	records, err := r.graph.ExecuteRead(ctx, query, params)
	if err != nil {
		r.logger.Error("FindMethods query failed", zap.Error(err))
		return nil, fmt.Errorf("failed to find methods: %w", err)
	}

	r.logger.Debug("FindMethods query completed",
		zap.Int("record_count", len(records)))

	return r.recordsToMethodInfos(records, "m")
}

func (r *repoReaderImpl) GetMethod(ctx context.Context, id ast.NodeID) (*MethodInfo, error) {
	query := `
		MATCH (m:Function {id: $id})
		RETURN m
	`
	records, err := r.graph.ExecuteRead(ctx, query, map[string]any{"id": int64(id)})
	if err != nil {
		return nil, fmt.Errorf("failed to get method: %w", err)
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("method not found: %d", id)
	}

	methods, err := r.recordsToMethodInfos(records, "m")
	if err != nil {
		return nil, err
	}
	return methods[0], nil
}

func (r *repoReaderImpl) FindMethodByName(ctx context.Context, methodName string, className string) (*MethodInfo, error) {
	filter := MethodFilter{Name: methodName, Limit: 1}
	if className != "" {
		filter.ClassName = className
	}

	methods, err := r.FindMethods(ctx, filter)
	if err != nil {
		return nil, err
	}
	if len(methods) == 0 {
		return nil, fmt.Errorf("method not found: %s", methodName)
	}
	return methods[0], nil
}

// --- Field Operations ---

func (r *repoReaderImpl) FindFields(ctx context.Context, filter FieldFilter) ([]*FieldInfo, error) {
	// Join through FileScope to filter by repo (only FileScope has repo field)
	query := `
		MATCH (file:FileScope {repo: $repo})
		MATCH (f:Field {fileId: file.fileId})
	`
	params := map[string]any{"repo": r.repoName}

	conditions := []string{}
	if filter.Name != "" {
		conditions = append(conditions, "f.name = $name")
		params["name"] = filter.Name
	}
	if filter.ClassID != nil {
		conditions = append(conditions, "EXISTS { MATCH (c:Class {id: $classId})-[:CONTAINS]->(f) }")
		params["classId"] = int64(*filter.ClassID)
	}

	if len(conditions) > 0 {
		query += " WHERE "
		for i, cond := range conditions {
			if i > 0 {
				query += " AND "
			}
			query += cond
		}
	}

	query += " RETURN f ORDER BY f.name"

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", filter.Limit)
	}

	records, err := r.graph.ExecuteRead(ctx, query, params)
	if err != nil {
		return nil, fmt.Errorf("failed to find fields: %w", err)
	}

	return r.recordsToFieldInfos(records, "f")
}

func (r *repoReaderImpl) GetField(ctx context.Context, id ast.NodeID) (*FieldInfo, error) {
	query := `
		MATCH (f:Field {id: $id})
		RETURN f
	`
	records, err := r.graph.ExecuteRead(ctx, query, map[string]any{"id": int64(id)})
	if err != nil {
		return nil, fmt.Errorf("failed to get field: %w", err)
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("field not found: %d", id)
	}

	fields, err := r.recordsToFieldInfos(records, "f")
	if err != nil {
		return nil, err
	}
	return fields[0], nil
}

// --- Relationship Queries ---

func (r *repoReaderImpl) GetClassMethods(ctx context.Context, classID ast.NodeID) ([]*MethodInfo, error) {
	query := `
		MATCH (c:Class {id: $classId})-[:CONTAINS]->(m:Function)
		RETURN m
		ORDER BY m.name
	`
	records, err := r.graph.ExecuteRead(ctx, query, map[string]any{"classId": int64(classID)})
	if err != nil {
		return nil, fmt.Errorf("failed to get class methods: %w", err)
	}

	return r.recordsToMethodInfos(records, "m")
}

func (r *repoReaderImpl) GetClassFields(ctx context.Context, classID ast.NodeID) ([]*FieldInfo, error) {
	query := `
		MATCH (c:Class {id: $classId})-[:CONTAINS]->(f:Field)
		RETURN f
		ORDER BY f.name
	`
	records, err := r.graph.ExecuteRead(ctx, query, map[string]any{"classId": int64(classID)})
	if err != nil {
		return nil, fmt.Errorf("failed to get class fields: %w", err)
	}

	return r.recordsToFieldInfos(records, "f")
}

func (r *repoReaderImpl) GetMethodClass(ctx context.Context, methodID ast.NodeID) (*ClassInfo, error) {
	query := `
		MATCH (c:Class)-[:CONTAINS]->(m:Function {id: $methodId})
		RETURN c
	`
	records, err := r.graph.ExecuteRead(ctx, query, map[string]any{"methodId": int64(methodID)})
	if err != nil {
		return nil, fmt.Errorf("failed to get method class: %w", err)
	}
	if len(records) == 0 {
		return nil, nil // top-level function
	}

	classes, err := r.recordsToClassInfos(records, "c")
	if err != nil {
		return nil, err
	}
	return classes[0], nil
}

// -----------------------------------------------------------------------------
// Record Conversion Helpers
// -----------------------------------------------------------------------------

func (r *repoReaderImpl) recordsToFileInfos(records []map[string]any) ([]*FileInfo, error) {
	files := make([]*FileInfo, 0, len(records))
	for _, record := range records {
		nodeData, ok := record["f"].(map[string]any)
		if !ok {
			continue
		}

		file := &FileInfo{
			ID:       ast.NodeID(toInt64(nodeData["id"])),
			FileID:   int32(toInt64(nodeData["fileId"])),
			RepoName: toString(nodeData["repo"]),
		}
		if path, ok := nodeData["path"].(string); ok {
			file.Path = path
		}
		if lang, ok := nodeData["language"].(string); ok {
			file.Language = lang
		}
		// Extract metadata from md_ prefixed properties
		file.Metadata = extractMetadata(nodeData)

		files = append(files, file)
	}
	return files, nil
}

func (r *repoReaderImpl) recordsToClassInfos(records []map[string]any, varName string) ([]*ClassInfo, error) {
	classes := make([]*ClassInfo, 0, len(records))
	for _, record := range records {
		nodeData, ok := record[varName].(map[string]any)
		if !ok {
			continue
		}

		class := &ClassInfo{
			ID:     ast.NodeID(toInt64(nodeData["id"])),
			FileID: int32(toInt64(nodeData["fileId"])),
			Name:   toString(nodeData["name"]),
		}
		if path, ok := nodeData["path"].(string); ok {
			class.FilePath = path
		}
		// Parse range if present
		if rangeStr, ok := nodeData["range"].(string); ok {
			class.Range = parseRange(rangeStr)
		}
		// Extract metadata from md_ prefixed properties
		class.Metadata = extractMetadata(nodeData)

		classes = append(classes, class)
	}
	return classes, nil
}

func (r *repoReaderImpl) recordsToMethodInfos(records []map[string]any, varName string) ([]*MethodInfo, error) {
	methods := make([]*MethodInfo, 0, len(records))
	for _, record := range records {
		nodeData, ok := record[varName].(map[string]any)
		if !ok {
			continue
		}

		method := &MethodInfo{
			ID:     ast.NodeID(toInt64(nodeData["id"])),
			FileID: int32(toInt64(nodeData["fileId"])),
			Name:   toString(nodeData["name"]),
		}
		if path, ok := nodeData["path"].(string); ok {
			method.FilePath = path
		}
		if rangeStr, ok := nodeData["range"].(string); ok {
			method.Range = parseRange(rangeStr)
		}
		// Extract metadata from md_ prefixed properties
		method.Metadata = extractMetadata(nodeData)

		methods = append(methods, method)
	}
	return methods, nil
}

func (r *repoReaderImpl) recordsToFieldInfos(records []map[string]any, varName string) ([]*FieldInfo, error) {
	fields := make([]*FieldInfo, 0, len(records))
	for _, record := range records {
		nodeData, ok := record[varName].(map[string]any)
		if !ok {
			continue
		}

		field := &FieldInfo{
			ID:   ast.NodeID(toInt64(nodeData["id"])),
			Name: toString(nodeData["name"]),
		}
		if typeStr, ok := nodeData["type"].(string); ok {
			field.Type = typeStr
		}
		if rangeStr, ok := nodeData["range"].(string); ok {
			field.Range = parseRange(rangeStr)
		}
		// Extract metadata from md_ prefixed properties
		field.Metadata = extractMetadata(nodeData)

		fields = append(fields, field)
	}
	return fields, nil
}

// -----------------------------------------------------------------------------
// FileReader Implementation
// -----------------------------------------------------------------------------

type fileReaderImpl struct {
	repoName string
	filePath string
	fileID   int32
	graph    *codegraph.CodeGraph
	logger   *zap.Logger
}

func (f *fileReaderImpl) Path() string {
	return f.filePath
}

func (f *fileReaderImpl) FileID() int32 {
	return f.fileID
}

func (f *fileReaderImpl) Info(ctx context.Context) (*FileInfo, error) {
	repo := &repoReaderImpl{repoName: f.repoName, graph: f.graph, logger: f.logger}
	if f.filePath != "" {
		return repo.GetFileByPath(ctx, f.filePath)
	}
	return repo.GetFile(ctx, ast.NodeID(f.fileID))
}

func (f *fileReaderImpl) ListClasses(ctx context.Context) ([]*ClassInfo, error) {
	fileID, err := f.resolveFileID(ctx)
	if err != nil {
		return nil, err
	}

	repo := &repoReaderImpl{repoName: f.repoName, graph: f.graph, logger: f.logger}
	return repo.FindClasses(ctx, ClassFilter{FileID: &fileID})
}

func (f *fileReaderImpl) FindClassByName(ctx context.Context, name string) (*ClassInfo, error) {
	fileID, err := f.resolveFileID(ctx)
	if err != nil {
		return nil, err
	}

	repo := &repoReaderImpl{repoName: f.repoName, graph: f.graph, logger: f.logger}
	classes, err := repo.FindClasses(ctx, ClassFilter{Name: name, FileID: &fileID, Limit: 1})
	if err != nil {
		return nil, err
	}
	if len(classes) == 0 {
		return nil, fmt.Errorf("class not found: %s in file %s", name, f.filePath)
	}
	return classes[0], nil
}

func (f *fileReaderImpl) ListMethods(ctx context.Context) ([]*MethodInfo, error) {
	fileID, err := f.resolveFileID(ctx)
	if err != nil {
		return nil, err
	}

	repo := &repoReaderImpl{repoName: f.repoName, graph: f.graph, logger: f.logger}
	return repo.FindMethods(ctx, MethodFilter{FileID: &fileID})
}

func (f *fileReaderImpl) ListFunctions(ctx context.Context) ([]*MethodInfo, error) {
	fileID, err := f.resolveFileID(ctx)
	if err != nil {
		return nil, err
	}

	isMethod := false
	repo := &repoReaderImpl{repoName: f.repoName, graph: f.graph, logger: f.logger}
	return repo.FindMethods(ctx, MethodFilter{FileID: &fileID, IsMethod: &isMethod})
}

func (f *fileReaderImpl) FindMethodByName(ctx context.Context, name string) (*MethodInfo, error) {
	fileID, err := f.resolveFileID(ctx)
	if err != nil {
		return nil, err
	}

	repo := &repoReaderImpl{repoName: f.repoName, graph: f.graph, logger: f.logger}
	methods, err := repo.FindMethods(ctx, MethodFilter{Name: name, FileID: &fileID, Limit: 1})
	if err != nil {
		return nil, err
	}
	if len(methods) == 0 {
		return nil, fmt.Errorf("method not found: %s in file %s", name, f.filePath)
	}
	return methods[0], nil
}

func (f *fileReaderImpl) FindMethodInClass(ctx context.Context, methodName, className string) (*MethodInfo, error) {
	// First find the class
	class, err := f.FindClassByName(ctx, className)
	if err != nil {
		return nil, err
	}

	// Then find the method in that class
	repo := &repoReaderImpl{repoName: f.repoName, graph: f.graph, logger: f.logger}
	methods, err := repo.FindMethods(ctx, MethodFilter{Name: methodName, ClassID: &class.ID, Limit: 1})
	if err != nil {
		return nil, err
	}
	if len(methods) == 0 {
		return nil, fmt.Errorf("method not found: %s.%s", className, methodName)
	}
	return methods[0], nil
}

func (f *fileReaderImpl) ListFields(ctx context.Context) ([]*FieldInfo, error) {
	fileID, err := f.resolveFileID(ctx)
	if err != nil {
		return nil, err
	}

	query := `
		MATCH (f:Field {fileId: $fileId})
		RETURN f
		ORDER BY f.name
	`
	records, err := f.graph.ExecuteRead(ctx, query, map[string]any{"fileId": fileID})
	if err != nil {
		return nil, fmt.Errorf("failed to list fields: %w", err)
	}

	repo := &repoReaderImpl{repoName: f.repoName, graph: f.graph, logger: f.logger}
	return repo.recordsToFieldInfos(records, "f")
}

func (f *fileReaderImpl) FindFieldByName(ctx context.Context, fieldName, className string) (*FieldInfo, error) {
	fileID, err := f.resolveFileID(ctx)
	if err != nil {
		return nil, err
	}

	query := `
		MATCH (f:Field {name: $name, fileId: $fileId})
		RETURN f
		LIMIT 1
	`
	params := map[string]any{"name": fieldName, "fileId": fileID}

	if className != "" {
		query = `
			MATCH (c:Class {name: $className})-[:CONTAINS]->(f:Field {name: $name, fileId: $fileId})
			RETURN f
			LIMIT 1
		`
		params["className"] = className
	}

	records, err := f.graph.ExecuteRead(ctx, query, params)
	if err != nil {
		return nil, fmt.Errorf("failed to find field: %w", err)
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("field not found: %s", fieldName)
	}

	repo := &repoReaderImpl{repoName: f.repoName, graph: f.graph, logger: f.logger}
	fields, err := repo.recordsToFieldInfos(records, "f")
	if err != nil {
		return nil, err
	}
	return fields[0], nil
}

func (f *fileReaderImpl) resolveFileID(ctx context.Context) (int32, error) {
	if f.fileID != 0 {
		return f.fileID, nil
	}

	// Resolve file ID from path
	query := `
		MATCH (file:FileScope {path: $path, repo: $repo})
		RETURN file.fileId AS fileId
		LIMIT 1
	`
	records, err := f.graph.ExecuteRead(ctx, query, map[string]any{
		"path": f.filePath,
		"repo": f.repoName,
	})
	if err != nil {
		return 0, fmt.Errorf("failed to resolve file ID: %w", err)
	}
	if len(records) == 0 {
		return 0, fmt.Errorf("file not found: %s", f.filePath)
	}

	f.fileID = int32(toInt64(records[0]["fileId"]))
	return f.fileID, nil
}

// -----------------------------------------------------------------------------
// Utility Functions
// -----------------------------------------------------------------------------

func toInt64(v any) int64 {
	switch val := v.(type) {
	case int64:
		return val
	case int32:
		return int64(val)
	case int:
		return int64(val)
	case float64:
		return int64(val)
	default:
		return 0
	}
}

func toString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func parseRange(s string) base.Range {
	var r base.Range
	fmt.Sscanf(s, "(%d,%d)-(%d,%d)",
		&r.Start.Line, &r.Start.Character,
		&r.End.Line, &r.End.Character)
	return r
}

// extractMetadata extracts metadata from Neo4j node properties.
// Metadata is stored with "md_" prefix in Neo4j (e.g., "md_annotations" -> "annotations").
func extractMetadata(nodeData map[string]any) map[string]any {
	metadata := make(map[string]any)
	for key, value := range nodeData {
		if len(key) > 3 && key[:3] == "md_" {
			metadata[key[3:]] = value
		}
	}
	if len(metadata) == 0 {
		return nil
	}
	return metadata
}
