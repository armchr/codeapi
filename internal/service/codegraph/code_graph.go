package codegraph

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/armchr/codeapi/internal/config"
	"github.com/armchr/codeapi/internal/model/ast"
	"github.com/armchr/codeapi/pkg/lsp/base"

	"go.uber.org/zap"
)

type Buffer struct {
	Nodes     []*ast.Node
	Relations []RelationSpec
}

type CodeGraph struct {
	db          GraphDatabase
	config      *config.Config
	logger      *zap.Logger
	fileIDCache map[int32]string
	// Batch writing support - file-level buffers for parallel processing
	enableBatchWrites bool
	batchSize         int
	buffers           map[int32]*Buffer // Map: fileID -> buffer
	bufferMutex       sync.Mutex        // Protects buffer maps
}

func NewCodeGraph(uri, username, password string, config *config.Config, logger *zap.Logger) (*CodeGraph, error) {
	db, err := NewNeo4jDatabase(uri, username, password, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create Neo4j database: %w", err)
	}

	err = db.VerifyConnectivity(context.Background())
	if err != nil {
		db.Close(context.Background())
		return nil, fmt.Errorf("failed to verify database connectivity: %w", err)
	}

	// Initialize batch writing configuration
	enableBatch := config.CodeGraph.EnableBatchWrites
	batchSize := config.CodeGraph.BatchSize
	if batchSize == 0 {
		batchSize = 100 // default
	}

	return &CodeGraph{
		db:                db,
		config:            config,
		logger:            logger,
		fileIDCache:       make(map[int32]string),
		enableBatchWrites: enableBatch,
		batchSize:         batchSize,
		buffers:           make(map[int32]*Buffer),
	}, nil
}

func (cg *CodeGraph) Close(ctx context.Context) error {
	return cg.db.Close(ctx)
}

// InitializeFileBuffers initializes buffers for a file before processing starts
// This reduces lock contention during writeNode/CreateRelation calls
func (cg *CodeGraph) InitializeFileBuffers(fileID int32) {
	if !cg.enableBatchWrites {
		return
	}

	cg.bufferMutex.Lock()
	//cg.logger.Debug("Acquired bufferMutex lock in InitializeFileBuffers", zap.Int32("fileID", fileID))
	defer func() {
		//cg.logger.Debug("Releasing bufferMutex lock in InitializeFileBuffers", zap.Int32("fileID", fileID))
		cg.bufferMutex.Unlock()
	}()

	// Initialize buffers for this file
	cg.buffers[fileID] = &Buffer{
		Nodes:     make([]*ast.Node, 0, cg.batchSize),
		Relations: make([]RelationSpec, 0, cg.batchSize),
	}
}

// CleanupFileBuffers flushes and removes buffers for a file after processing completes
// This frees memory and ensures data is written to database
func (cg *CodeGraph) CleanupFileBuffers(ctx context.Context, fileID int32) error {
	if !cg.enableBatchWrites {
		return nil
	}

	// Flush any remaining data for this file
	if err := cg.Flush(ctx, &fileID); err != nil {
		return err
	}

	// Remove buffers to free memory
	cg.bufferMutex.Lock()
	//cg.logger.Debug("Acquired bufferMutex lock in CleanupFileBuffers", zap.Int32("fileID", fileID))
	defer func() {
		//cg.logger.Debug("Releasing bufferMutex lock in CleanupFileBuffers", zap.Int32("fileID", fileID))
		cg.bufferMutex.Unlock()
	}()

	delete(cg.buffers, fileID)

	return nil
}

// FlushNodes writes buffered nodes to the database
// If fileID is provided, only flushes nodes for that file
// If fileID is nil, flushes all buffered nodes
func (cg *CodeGraph) FlushNodes(ctx context.Context, fileID *int32) error {
	if !cg.enableBatchWrites {
		return nil // No-op if batch writes not enabled
	}

	if fileID != nil {
		cg.bufferMutex.Lock()
		buffers := cg.buffers[*fileID]
		cg.bufferMutex.Unlock()

		if buffers == nil {
			return nil
		}

		nodes := make([]*ast.Node, len(buffers.Nodes))
		copy(nodes, buffers.Nodes)

		buffers.Nodes = make([]*ast.Node, 0, cg.batchSize)

		if len(nodes) == 0 {
			cg.logger.Debug("Flushing node buffer for file",
				zap.Int32("file_id", *fileID),
				zap.Int("count", 0))
			return nil
		}

		cg.logger.Debug("Flushing node buffer for file",
			zap.Int32("file_id", *fileID),
			zap.Int("count", len(nodes)))

		err := cg.BatchWriteNodes(ctx, nodes)
		if err != nil {
			return fmt.Errorf("failed to flush nodes for file %d: %w", *fileID, err)
		}
	} else {
		cg.bufferMutex.Lock()
		//cg.logger.Debug("Acquired bufferMutex lock in FlushNodes (flush all)")
		defer func() {
			//cg.logger.Debug("Releasing bufferMutex lock in FlushNodes (flush all)")
			cg.bufferMutex.Unlock()
		}()

		// Flush all files' nodes
		totalCount := 0
		for fid := range cg.buffers {
			totalCount += len(cg.buffers[fid].Nodes)
		}

		if totalCount == 0 {
			return nil
		}

		cg.logger.Debug("Flushing all node buffers", zap.Int("count", totalCount))

		// Collect all nodes from all files
		allNodes := make([]*ast.Node, 0, totalCount)
		for _, buffers := range cg.buffers {
			allNodes = append(allNodes, buffers.Nodes...)
		}

		err := cg.BatchWriteNodes(ctx, allNodes)
		if err != nil {
			return fmt.Errorf("failed to flush all nodes: %w", err)
		}

		// Clear all buffers
		//cg.nodeBuffers = make(map[int32][]*ast.Node)
	}

	return nil
}

// FlushRelations writes buffered relations to the database
// If fileID is provided, only flushes relations for that file
// If fileID is nil, flushes all buffered relations
func (cg *CodeGraph) FlushRelations(ctx context.Context, fileID *int32) error {
	if !cg.enableBatchWrites {
		return nil // No-op if batch writes not enabled
	}

	if fileID != nil {
		cg.bufferMutex.Lock()
		buffers := cg.buffers[*fileID]
		cg.bufferMutex.Unlock()
		if buffers == nil {
			return nil
		}

		relations := make([]RelationSpec, len(buffers.Relations))
		copy(relations, buffers.Relations)

		buffers.Relations = make([]RelationSpec, 0, cg.batchSize)

		if len(relations) == 0 {
			cg.logger.Debug("Flushing relation buffer for file",
				zap.Int32("file_id", *fileID),
				zap.Int("count", 0))
			return nil
		}

		cg.logger.Debug("Flushing relation buffer for file",
			zap.Int32("file_id", *fileID),
			zap.Int("count", len(relations)))

		err := cg.BatchCreateRelations(ctx, relations)
		if err != nil {
			return fmt.Errorf("failed to flush relations for file %d: %w", *fileID, err)
		}
	} else {
		cg.bufferMutex.Lock()
		//cg.logger.Debug("Acquired bufferMutex lock in FlushRelations (flush all)")
		defer func() {
			//cg.logger.Debug("Releasing bufferMutex lock in FlushRelations (flush all)")
			cg.bufferMutex.Unlock()
		}()

		// Flush all files' relations
		totalCount := 0
		for fid := range cg.buffers {
			totalCount += len(cg.buffers[fid].Relations)
		}

		if totalCount == 0 {
			return nil
		}

		cg.logger.Debug("Flushing all relation buffers", zap.Int("count", totalCount))

		// Collect all relations from all files
		allRelations := make([]RelationSpec, 0, totalCount)
		for _, buffers := range cg.buffers {
			allRelations = append(allRelations, buffers.Relations...)
		}

		err := cg.BatchCreateRelations(ctx, allRelations)
		if err != nil {
			return fmt.Errorf("failed to flush all relations: %w", err)
		}

		// Clear all buffers
		//cg.relationBuffers = make(map[int32][]RelationSpec)
	}

	return nil
}

// Flush writes buffered nodes and relations to the database
// If fileID is provided, only flushes buffers for that file
// If fileID is nil, flushes all buffers
// IMPORTANT: Nodes are flushed BEFORE relations to ensure they exist in the database
func (cg *CodeGraph) Flush(ctx context.Context, fileID *int32) error {
	if !cg.enableBatchWrites {
		return nil // No-op if batch writes not enabled
	}

	// Flush nodes first (required for relations to reference them)
	if err := cg.FlushNodes(ctx, fileID); err != nil {
		return err
	}

	// Then flush relations
	if err := cg.FlushRelations(ctx, fileID); err != nil {
		return err
	}

	return nil
}

func (cg *CodeGraph) dbRecordToNode(record GraphNode) (*ast.Node, error) {
	recordMap := make(map[string]any)
	for key, value := range record.GetProperties() {
		recordMap[key] = value
	}

	return cg.recordToNode(recordMap)
}

func (cg *CodeGraph) recordToNode(record map[string]any) (*ast.Node, error) {
	id := record["id"]
	nodeType := record["nodeType"]
	fileID := record["fileId"]
	name := record["name"]
	rangeStr := record["range"]
	version := record["version"]
	scopeID := record["scopeId"]

	newMetadata := make(map[string]any)
	for key, value := range record {
		if cg.isFirstClassMetadata(key) {
			newMetadata[key] = value
		} else if strings.HasPrefix(key, "md_") {
			newMetadata[key[3:]] = value
		}
	}

	node := &ast.Node{
		ID:       ast.NodeID(cg.convertToInt64(id)),
		NodeType: ast.NodeType(cg.convertToInt64(nodeType)),
		FileID:   cg.convertToInt32(fileID),
		Name:     name.(string),
		Version:  cg.convertToInt32(version),
		ScopeID:  ast.NodeID(cg.convertToInt64(scopeID)),
	}

	if rangeStr != nil {
		node.Range = strToRange(rangeStr.(string))
	}

	if len(newMetadata) > 0 {
		node.MetaData = newMetadata
	}

	return node, nil
}

/*
func parseRange(rangeMap map[string]any) base.Range {
	var rng base.Range

	if startMap, ok := rangeMap["start"].(map[string]any); ok {
		if line, ok := startMap["line"].(int64); ok {
			rng.Start.Line = int(line)
		}
		if char, ok := startMap["character"].(int64); ok {
			rng.Start.Character = int(char)
		}
	}

	if endMap, ok := rangeMap["end"].(map[string]any); ok {
		if line, ok := endMap["line"].(int64); ok {
			rng.End.Line = int(line)
		}
		if char, ok := endMap["character"].(int64); ok {
			rng.End.Character = int(char)
		}
	}

	return rng
}
*/

func (cg *CodeGraph) getNodeLabel(nodeType ast.NodeType) string {
	switch nodeType {
	case ast.NodeTypeModuleScope:
		return "ModuleScope"
	case ast.NodeTypeFileScope:
		return "FileScope"
	case ast.NodeTypeBlock:
		return "Block"
	case ast.NodeTypeVariable:
		return "Variable"
	case ast.NodeTypeExpression:
		return "Expression"
	case ast.NodeTypeConditional:
		return "Conditional"
	case ast.NodeTypeFunction:
		return "Function"
	case ast.NodeTypeClass:
		return "Class"
	case ast.NodeTypeField:
		return "Field"
	case ast.NodeTypeFunctionCall:
		return "FunctionCall"
	case ast.NodeTypeFileNumber:
		return "FileNumber"
	case ast.NodeTypeLoop:
		return "Loop"
	case ast.NodeTypeImport:
		return "Import"
	default:
		return "Node"
	}
}

func (cg *CodeGraph) CreateFunction(ctx context.Context, node *ast.Node) error {
	if node.NodeType != ast.NodeTypeFunction {
		return fmt.Errorf("invalid node type: expected %d, got %d", ast.NodeTypeFunction, node.NodeType)
	}
	err := cg.writeNode(ctx, node)
	if err != nil {
		return fmt.Errorf("failed to create function node: %w", err)
	}

	return nil
}

func (cg *CodeGraph) ReadFunction(ctx context.Context, nodeID ast.NodeID) (*ast.Node, error) {
	return cg.readNodeByType(ctx, nodeID, ast.NodeTypeFunction)
}

/*
func (cg *CodeGraph) ReadFunctionArgs(ctx context.Context, functionNodeID ast.NodeID) ([]*ast.Node, error) {
	session := cg.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	nodesResult, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := `
			MATCH (f:Function {id: $functionId})-[:FUNCTION_ARG]->(arg)
			RETURN arg
			ORDER BY arg.position
		`
		parameters := map[string]any{
			"functionId": int64(functionNodeID),
		}

		result, err := tx.Run(ctx, query, parameters)
		if err != nil {
			return nil, err
		}

		var nodes []*ast.Node
		for result.Next(ctx) {
			record := result.Record()
			node, err := cg.recordToNode(record)
			if err != nil {
				return nil, err
			}
			nodes = append(nodes, node)
		}

		if err = result.Err(); err != nil {
			return nil, err
		}

		return nodes, nil
	})

	if err != nil {
		cg.logger.Error("Failed to read function arguments", zap.Int64("functionId", int64(functionNodeID)), zap.Error(err))
		return nil, fmt.Errorf("failed to read function arguments: %w", err)
	}

	return nodesResult.([]*ast.Node), nil
}
*/

func (cg *CodeGraph) CreateFileScope(ctx context.Context, node *ast.Node) error {
	if node.NodeType != ast.NodeTypeFileScope {
		return fmt.Errorf("invalid node type: expected %d, got %d", ast.NodeTypeFileScope, node.NodeType)
	}
	return cg.writeNode(ctx, node)
}

func (cg *CodeGraph) ReadFileScope(ctx context.Context, nodeID ast.NodeID) (*ast.Node, error) {
	return cg.readNodeByType(ctx, nodeID, ast.NodeTypeFileScope)
}

func (cg *CodeGraph) GetFilePath(ctx context.Context, fileID int32) string {
	if path, ok := cg.fileIDCache[fileID]; ok {
		return path
	}

	fs, err := cg.ReadFileScope(ctx, ast.NodeID(fileID))
	if err != nil {
		return ""
	}
	path, ok := fs.MetaData["path"].(string)
	if !ok {
		return ""
	}
	cg.fileIDCache[fileID] = path
	return path
}

func (cg *CodeGraph) FindFileScopes(ctx context.Context, repoName, filePath string) ([]*ast.Node, error) {
	params := map[string]any{
		"repo": repoName,
	}

	if filePath != "" {
		params["path"] = filePath
	}
	nodes, err := cg.readNodes(ctx, ast.NodeTypeFileScope, params)
	if err != nil {
		return nil, err
	}
	return nodes, nil
}

func (cg *CodeGraph) CreateClass(ctx context.Context, node *ast.Node) error {
	if node.NodeType != ast.NodeTypeClass {
		return fmt.Errorf("invalid node type: expected %d, got %d", ast.NodeTypeClass, node.NodeType)
	}
	return cg.writeNode(ctx, node)
}

func (cg *CodeGraph) ReadClass(ctx context.Context, nodeID ast.NodeID) (*ast.Node, error) {
	return cg.readNodeByType(ctx, nodeID, ast.NodeTypeClass)
}

func (cg *CodeGraph) CreateVariable(ctx context.Context, node *ast.Node) error {
	if node.NodeType != ast.NodeTypeVariable {
		return fmt.Errorf("invalid node type: expected %d, got %d", ast.NodeTypeVariable, node.NodeType)
	}
	return cg.writeNode(ctx, node)
}

func (cg *CodeGraph) ReadVariable(ctx context.Context, nodeID ast.NodeID) (*ast.Node, error) {
	return cg.readNodeByType(ctx, nodeID, ast.NodeTypeVariable)
}

func (cg *CodeGraph) CreateBlock(ctx context.Context, node *ast.Node) error {
	if node.NodeType != ast.NodeTypeBlock {
		return fmt.Errorf("invalid node type: expected %d, got %d", ast.NodeTypeBlock, node.NodeType)
	}
	return cg.writeNode(ctx, node)
}

func (cg *CodeGraph) ReadBlock(ctx context.Context, nodeID ast.NodeID) (*ast.Node, error) {
	return cg.readNodeByType(ctx, nodeID, ast.NodeTypeBlock)
}

func (cg *CodeGraph) CreateExpression(ctx context.Context, node *ast.Node) error {
	if node.NodeType != ast.NodeTypeExpression {
		return fmt.Errorf("invalid node type: expected %d, got %d", ast.NodeTypeExpression, node.NodeType)
	}
	return cg.writeNode(ctx, node)
}

func (cg *CodeGraph) ReadExpression(ctx context.Context, nodeID ast.NodeID) (*ast.Node, error) {
	return cg.readNodeByType(ctx, nodeID, ast.NodeTypeExpression)
}

func (cg *CodeGraph) CreateConditional(ctx context.Context, node *ast.Node) error {
	if node.NodeType != ast.NodeTypeConditional {
		return fmt.Errorf("invalid node type: expected %d, got %d", ast.NodeTypeConditional, node.NodeType)
	}
	return cg.writeNode(ctx, node)
}

func (cg *CodeGraph) ReadConditional(ctx context.Context, nodeID ast.NodeID) (*ast.Node, error) {
	return cg.readNodeByType(ctx, nodeID, ast.NodeTypeConditional)
}

func (cg *CodeGraph) CreateLoop(ctx context.Context, node *ast.Node) error {
	if node.NodeType != ast.NodeTypeLoop {
		return fmt.Errorf("invalid node type: expected %d, got %d", ast.NodeTypeLoop, node.NodeType)
	}
	return cg.writeNode(ctx, node)
}

func (cg *CodeGraph) CreateField(ctx context.Context, node *ast.Node) error {
	if node.NodeType != ast.NodeTypeField {
		return fmt.Errorf("invalid node type: expected %d, got %d", ast.NodeTypeField, node.NodeType)
	}
	return cg.writeNode(ctx, node)
}

func (cg *CodeGraph) ReadField(ctx context.Context, nodeID ast.NodeID) (*ast.Node, error) {
	return cg.readNodeByType(ctx, nodeID, ast.NodeTypeField)
}

func (cg *CodeGraph) CreateImport(ctx context.Context, node *ast.Node) error {
	if node.NodeType != ast.NodeTypeImport {
		return fmt.Errorf("invalid node type: expected %d, got %d", ast.NodeTypeImport, node.NodeType)
	}
	return cg.writeNode(ctx, node)
}

func (cg *CodeGraph) ReadImport(ctx context.Context, nodeID ast.NodeID) (*ast.Node, error) {
	return cg.readNodeByType(ctx, nodeID, ast.NodeTypeImport)
}

func (cg *CodeGraph) CreateFunctionCall(ctx context.Context, node *ast.Node) error {
	if node.NodeType != ast.NodeTypeFunctionCall {
		return fmt.Errorf("invalid node type: expected %d, got %d", ast.NodeTypeFunctionCall, node.NodeType)
	}
	return cg.writeNode(ctx, node)
}

func (cg *CodeGraph) ReadFunctionCall(ctx context.Context, nodeID ast.NodeID) (*ast.Node, error) {
	return cg.readNodeByType(ctx, nodeID, ast.NodeTypeFunctionCall)
}

func (cg *CodeGraph) CreateModuleScope(ctx context.Context, node *ast.Node) error {
	if node.NodeType != ast.NodeTypeModuleScope {
		return fmt.Errorf("invalid node type: expected %d, got %d", ast.NodeTypeModuleScope, node.NodeType)
	}
	return cg.writeNode(ctx, node)
}

func (cg *CodeGraph) ReadModuleScope(ctx context.Context, nodeID ast.NodeID) (*ast.Node, error) {
	return cg.readNodeByType(ctx, nodeID, ast.NodeTypeModuleScope)
}

func rangeToString(rng base.Range) string {
	return fmt.Sprintf("(%d,%d)-(%d,%d)", rng.Start.Line, rng.Start.Character, rng.End.Line, rng.End.Character)
}

func strToRange(s string) base.Range {
	var rng base.Range
	_, err := fmt.Sscanf(s, "(%d,%d)-(%d,%d)", &rng.Start.Line, &rng.Start.Character, &rng.End.Line, &rng.End.Character)
	if err != nil {
		return base.Range{}
	}
	return rng
}

var (
	FirstClassMetadata = map[string]bool{
		"fake":           true,
		"nameID":         true,
		"return":         true,
		"repo":           true,
		"path":           true,
		"language":       true,
		"is_constructor": true,
	}
)

func (cg *CodeGraph) isFirstClassMetadata(key string) bool {
	return FirstClassMetadata[key]
}

func (cg *CodeGraph) populateFirstClassMetadata(metadata map[string]any,
	param map[string]any,
	newMetadata map[string]any) {
	for key, value := range metadata {
		if cg.isFirstClassMetadata(key) {
			param[key] = value
		} else {
			newMetadata[key] = value
		}
	}
}

func (cg *CodeGraph) mapToSetParamString(m map[string]any, varName string) string {
	if len(m) == 0 {
		return ""
	}

	setClauses := ""
	for key := range m {
		if setClauses != "" {
			setClauses += ",\n"
		}
		setClauses += fmt.Sprintf("%s.%s = $%s", varName, key, key)
	}
	return setClauses
}

func (cg *CodeGraph) flattenMetadata(metadata map[string]any, param map[string]any) {
	for key, value := range metadata {
		param["md_"+key] = value
	}
}

func (cg *CodeGraph) writeNodeReal(ctx context.Context, node *ast.Node) error {
	// Original immediate write logic (when batch writes disabled)
	nodeLabel := cg.getNodeLabel(node.NodeType)
	parameters := map[string]any{
		"id":       int64(node.ID),
		"nodeType": int64(node.NodeType),
		"fileId":   int64(node.FileID),
		"name":     node.Name,
		"range":    rangeToString(node.Range),
		"version":  int64(node.Version),
		"scopeId":  int64(node.ScopeID),
	}

	if node.MetaData != nil {
		newMetadata := make(map[string]any)
		cg.populateFirstClassMetadata(node.MetaData, parameters, newMetadata)
		if len(newMetadata) > 0 {
			cg.flattenMetadata(newMetadata, parameters)
			//parameters["metaData"] = newMetadata
		}
	}

	// cg.logger.Debug("Writing node", zap.Int64("nodeId", int64(node.ID)), zap.Any("parameters", parameters))

	setQ := cg.mapToSetParamString(parameters, "n")
	query := fmt.Sprintf(`
		MERGE (n:%s {id: $id})
		SET %s
		RETURN n
	`, nodeLabel, setQ)

	_, err := cg.db.ExecuteWrite(ctx, query, parameters)
	if err != nil {
		cg.logger.Error("Failed to write node", zap.Int64("nodeId", int64(node.ID)), zap.Error(err))
		return fmt.Errorf("failed to write node: %w", err)
	}

	return nil
}

func (cg *CodeGraph) writeNode(ctx context.Context, node *ast.Node) error {
	// If batch writes are enabled, buffer the node instead of writing immediately
	if cg.enableBatchWrites {
		fileID := node.FileID

		// Only lock for map access - Go maps are not safe for concurrent reads/writes
		cg.bufferMutex.Lock()
		buffers := cg.buffers[fileID]
		cg.bufferMutex.Unlock()

		if buffers != nil {
			// These operations are safe without lock since each file is processed by a single thread
			buffers.Nodes = append(buffers.Nodes, node)
			shouldFlush := len(buffers.Nodes) >= cg.batchSize

			// Flush if this file's buffer is full
			if shouldFlush {
				// Flush both nodes and relations to maintain referential integrity
				err := cg.Flush(ctx, &fileID)
				if err != nil {
					return err
				}
			}

			return nil
		}
	}

	return cg.writeNodeReal(ctx, node)
}

// BatchWriteNodes writes multiple nodes in a single database transaction using UNWIND
// This is much faster than individual writeNode calls for bulk operations
func (cg *CodeGraph) BatchWriteNodes(ctx context.Context, nodes []*ast.Node) error {
	if len(nodes) == 0 {
		return nil
	}

	cg.logger.Debug("Batch writing nodes", zap.Int("count", len(nodes)))

	// Group nodes by label for efficient batch operations
	nodesByLabel := make(map[string][]map[string]any)
	astNodesByLabel := make(map[string][]*ast.Node)
	for _, node := range nodes {
		label := cg.getNodeLabel(node.NodeType)
		astNodesByLabel[label] = append(astNodesByLabel[label], node)

		// Convert node to parameters
		parameters := map[string]any{
			"id":       int64(node.ID),
			"nodeType": int64(node.NodeType),
			"fileId":   int64(node.FileID),
			"name":     node.Name,
			"range":    rangeToString(node.Range),
			"version":  int64(node.Version),
			"scopeId":  int64(node.ScopeID),
		}

		if node.MetaData != nil {
			newMetadata := make(map[string]any)
			cg.populateFirstClassMetadata(node.MetaData, parameters, newMetadata)
			if len(newMetadata) > 0 {
				cg.flattenMetadata(newMetadata, parameters)
			}
		}

		nodesByLabel[label] = append(nodesByLabel[label], parameters)
	}

	// Write each label group in batch
	for label, nodeParams := range nodesByLabel {
		// Build dynamic SET clause from first node's properties
		if len(nodeParams) == 0 {
			continue
		}

		// if len(nodeParams) == 1, use regular writeNode instead
		if len(nodeParams) == 1 {
			err := cg.writeNodeReal(ctx, astNodesByLabel[label][0])
			if err != nil {
				return fmt.Errorf("failed to write single node for label %s: %w", label, err)
			}
			continue
		}

		setClause := ""
		first := true
		for key := range nodeParams[0] {
			if !first {
				setClause += ",\n  "
			}
			setClause += fmt.Sprintf("n.%s = nodeData.%s", key, key)
			first = false
		}

		query := fmt.Sprintf(`
			UNWIND $nodes AS nodeData
			MERGE (n:%s {id: nodeData.id})
			SET %s
			RETURN count(n) as created
		`, label, setClause)

		_, err := cg.db.ExecuteWrite(ctx, query, map[string]any{"nodes": nodeParams})
		if err != nil {
			cg.logger.Error("Failed to batch write nodes",
				zap.String("label", label),
				zap.Int("count", len(nodeParams)),
				zap.Error(err))
			return fmt.Errorf("failed to batch write nodes for label %s: %w", label, err)
		}

		cg.logger.Debug("Batch wrote nodes",
			zap.String("label", label),
			zap.Int("count", len(nodeParams)))
	}

	return nil
}

// RelationSpec specifies a relationship to be created
type RelationSpec struct {
	ParentID ast.NodeID
	ChildID  ast.NodeID
	Label    string
	Metadata map[string]any
	FileID   int32 // File ID for buffer management (can be from parent or child node)
}

// BatchCreateRelations creates multiple relationships in a single database transaction
// This is much faster than individual CreateRelation calls for bulk operations
func (cg *CodeGraph) BatchCreateRelations(ctx context.Context, relations []RelationSpec) error {
	if len(relations) == 0 {
		return nil
	}

	cg.logger.Debug("Batch creating relations", zap.Int("count", len(relations)))

	// Group relations by label for efficient processing
	relationsByLabel := make(map[string][]map[string]any)
	for _, rel := range relations {
		relData := map[string]any{
			"parentId": int64(rel.ParentID),
			"childId":  int64(rel.ChildID),
		}

		// Add metadata if present
		if rel.Metadata != nil {
			newMetadata := make(map[string]any)
			cg.flattenMetadata(rel.Metadata, newMetadata)
			for key, value := range newMetadata {
				relData[key] = value
			}
		}

		relationsByLabel[rel.Label] = append(relationsByLabel[rel.Label], relData)
	}

	// Write each label group in batch
	for label, relParams := range relationsByLabel {
		// Build SET clause for metadata (if any)
		setClause := ""
		if len(relParams) > 0 && len(relParams[0]) > 2 { // More than just parentId and childId
			first := true
			for key := range relParams[0] {
				if key == "parentId" || key == "childId" {
					continue
				}
				if !first {
					setClause += ",\n  "
				}
				setClause += fmt.Sprintf("r.%s = relData.%s", key, key)
				first = false
			}
			if setClause != "" {
				setClause = "SET " + setClause
			}
		}

		query := fmt.Sprintf(`
			UNWIND $relations AS relData
			MATCH (parent {id: relData.parentId}), (child {id: relData.childId})
			MERGE (parent)-[r:%s]->(child)
			%s
			RETURN count(r) as created
		`, label, setClause)

		_, err := cg.db.ExecuteWrite(ctx, query, map[string]any{"relations": relParams})
		if err != nil {
			cg.logger.Error("Failed to batch create relations",
				zap.String("label", label),
				zap.Int("count", len(relParams)),
				zap.Error(err))
			return fmt.Errorf("failed to batch create relations for label %s: %w", label, err)
		}

		cg.logger.Debug("Batch created relations",
			zap.String("label", label),
			zap.Int("count", len(relParams)))
	}

	return nil
}

func (cg *CodeGraph) readNodesByQuery(ctx context.Context, nodeVarName string, query string, params map[string]any) ([]*ast.Node, error) {
	records, err := cg.db.ExecuteRead(ctx, query, params)
	if err != nil {
		cg.logger.Error("Failed to read nodes",
			zap.String("Raw Query", query),
			zap.Any("Parameters", params),
			zap.Error(err))
		return nil, fmt.Errorf("failed to read node: %w", err)
	}

	if len(records) == 0 {
		return nil, nil
	}

	var results []*ast.Node
	for _, record := range records {
		nodeData, ok := record[nodeVarName]
		if !ok || nodeData == nil {
			continue
		}

		// Convert map to our GraphNode interface and then to ast.Node
		nodeMap, ok := nodeData.(map[string]any)
		if !ok {
			continue
		}

		node, err := cg.recordToNode(nodeMap)
		if err != nil {
			return nil, err
		}

		results = append(results, node)
	}

	return results, nil
}

func (cg *CodeGraph) readNodes(ctx context.Context, nodeType ast.NodeType, query map[string]any) ([]*ast.Node, error) {
	nodeLabel := cg.getNodeLabel(nodeType)
	q := ""
	if len(query) > 0 {
		q = "WHERE "
		i := 0
		for key := range query {
			if i > 0 {
				q += " AND\n"
			}
			q += fmt.Sprintf("n.%s = $%s", key, key)
			i++
		}
	}

	// For Kuzu, we need to handle the query differently since it doesn't use labels in MATCH the same way
	fullQuery := fmt.Sprintf(`
		MATCH (n:%s)
		%s
		RETURN n
	`, nodeLabel, q)
	return cg.readNodesByQuery(ctx, "n", fullQuery, query)

	/*
		records, err := cg.db.ExecuteRead(ctx, fullQuery, query)
		if err != nil {
			cg.logger.Error("Failed to read node",
				zap.Int64("nodeType", int64(nodeType)),
				zap.Error(err))
			return nil, fmt.Errorf("failed to read node: %w", err)
		}

		if len(records) == 0 {
			return nil, fmt.Errorf("node query with type %d not found", nodeType)
		}

		var results []*ast.Node
		for _, record := range records {
			nodeData, ok := record["n"]
			if !ok || nodeData == nil {
				continue
			}

			// Convert map to our GraphNode interface and then to ast.Node
			nodeMap, ok := nodeData.(map[string]any)
			if !ok {
				continue
			}

			node, err := cg.recordToNode(nodeMap)
			if err != nil {
				return nil, err
			}

			results = append(results, node)
		}

		return results, nil
	*/
}

func (cg *CodeGraph) readNodeByType(ctx context.Context, nodeID ast.NodeID, nodeType ast.NodeType) (*ast.Node, error) {
	nodes, err := cg.readNodes(ctx, nodeType, map[string]any{"id": int64(nodeID)})
	if err != nil {
		return nil, err
	}
	if len(nodes) != 1 {
		return nil, fmt.Errorf("node with id %d and type %d found - expected 1 but got %d", nodeID, nodeType, len(nodes))
	}
	return nodes[0], nil
}

func (cg *CodeGraph) FindNodesByNameAndTypeInFile(ctx context.Context, name string, nodeType ast.NodeType, fileID int32) ([]*ast.Node, error) {
	return cg.readNodes(ctx, nodeType, map[string]any{
		"name":   name,
		"fileId": int64(fileID),
	})
}

// GetNodesByTypeAndFileID returns all nodes of a given type in a specific file
func (cg *CodeGraph) GetNodesByTypeAndFileID(ctx context.Context, nodeType ast.NodeType, fileID int32) ([]*ast.Node, error) {
	return cg.readNodes(ctx, nodeType, map[string]any{
		"fileId": int64(fileID),
	})
}

func (cg *CodeGraph) CreateRelationReal(ctx context.Context, parentNodeID, childNodeID ast.NodeID,
	relationLabel string, metaData map[string]any, fileID int32) error {
	parameters := map[string]any{
		"parentId": int64(parentNodeID),
		"childId":  int64(childNodeID),
	}

	setMetaDataQ := ""
	if metaData != nil {
		//parameters["metaData"] = metaData
		//setMetaDataQ = "SET r.metaData = $metaData"
		newMetadata := make(map[string]any)
		cg.flattenMetadata(metaData, newMetadata)
		setMetaDataQ = cg.mapToSetParamString(newMetadata, "r")
		if setMetaDataQ != "" {
			setMetaDataQ = "SET " + setMetaDataQ
		}
		// append newMetadata to parameters
		for key, value := range newMetadata {
			parameters[key] = value
		}
	}

	query := fmt.Sprintf(`
		MATCH (parent {id: $parentId}), (child {id: $childId})
		MERGE (parent)-[r:%s]->(child)
		%s
		RETURN parent, child
	`, relationLabel, setMetaDataQ)

	_, err := cg.db.ExecuteWrite(ctx, query, parameters)
	if err != nil {
		cg.logger.Error("Failed to create relation",
			zap.Int64("parentId", int64(parentNodeID)),
			zap.Int64("childId", int64(childNodeID)),
			zap.String("relationLabel", relationLabel),
			zap.Error(err))
		return fmt.Errorf("failed to create relation: %w", err)
	}

	return nil
}

func (cg *CodeGraph) CreateRelation(ctx context.Context, parentNodeID, childNodeID ast.NodeID,
	relationLabel string, metaData map[string]any, fileID int32) error {

	// If batch writes are enabled, buffer the relation instead of writing immediately
	if cg.enableBatchWrites {
		// Only lock for map access - Go maps are not safe for concurrent reads/writes
		cg.bufferMutex.Lock()
		buffers := cg.buffers[fileID]
		cg.bufferMutex.Unlock()

		if buffers != nil {
			// These operations are safe without lock since each file is processed by a single thread
			relSpec := RelationSpec{
				ParentID: parentNodeID,
				ChildID:  childNodeID,
				Label:    relationLabel,
				Metadata: metaData,
				FileID:   fileID,
			}
			buffers.Relations = append(buffers.Relations, relSpec)
			shouldFlush := len(buffers.Relations) >= cg.batchSize

			// Flush if this file's buffer is full
			if shouldFlush {
				// Flush both nodes and relations to maintain referential integrity
				// Nodes must be flushed first so relations can reference them
				err := cg.Flush(ctx, &fileID)
				if err != nil {
					return err
				}
			}

			return nil
		}
	}

	return cg.CreateRelationReal(ctx, parentNodeID, childNodeID, relationLabel, metaData, fileID)
}

func (cg *CodeGraph) CreateContainsRelation(ctx context.Context, parentNodeID, childNodeID ast.NodeID, fileID int32) error {
	return cg.CreateRelation(ctx, parentNodeID, childNodeID, "CONTAINS", nil, fileID)
}

func (cg *CodeGraph) CreateHasFieldRelation(ctx context.Context, parentNodeID, childNodeID ast.NodeID, fileID int32) error {
	return cg.CreateRelation(ctx, parentNodeID, childNodeID, "HAS_FIELD", nil, fileID)
}
func (cg *CodeGraph) CreateCallsRelation(ctx context.Context, callerNodeID, calleeNodeID ast.NodeID, fileID int32) error {
	return cg.CreateRelation(ctx, callerNodeID, calleeNodeID, "CALLS", nil, fileID)
}

/*
func (cg *CodeGraph) CreateContainedByRelation(ctx context.Context, parentNodeID, childNodeID ast.NodeID) error {
	return cg.CreateRelation(ctx, parentNodeID, childNodeID, "CONTAINED_BY", nil)
}
*/

func (cg *CodeGraph) CreateInheritsRelation(ctx context.Context, parentNodeID, childNodeID ast.NodeID, fileID int32) error {
	return cg.CreateRelation(ctx, parentNodeID, childNodeID, "INHERITS", nil, fileID)
}

func (cg *CodeGraph) CreateCallsFunctionRelation(ctx context.Context, callerNodeID, calleeNodeID ast.NodeID, fileID int32) error {
	return cg.CreateRelation(ctx, callerNodeID, calleeNodeID, "CALLS_FUNCTION", nil, fileID)
}

// GetNodesByName returns all nodes with a given name and type
func (cg *CodeGraph) GetNodesByName(ctx context.Context, name string, nodeType ast.NodeType) ([]*ast.Node, error) {
	return cg.readNodes(ctx, nodeType, map[string]any{"name": name})
}

// GetNodesByType returns all nodes of a given type
func (cg *CodeGraph) GetNodesByType(ctx context.Context, nodeType ast.NodeType) ([]*ast.Node, error) {
	return cg.readNodes(ctx, nodeType, map[string]any{})
}

// GetNodeByID returns a node by its ID
func (cg *CodeGraph) GetNodeByID(ctx context.Context, nodeID ast.NodeID) (*ast.Node, error) {
	// Try each node type until we find the node
	nodeTypes := []ast.NodeType{
		ast.NodeTypeClass,
		ast.NodeTypeFunction,
		ast.NodeTypeField,
		ast.NodeTypeVariable,
		ast.NodeTypeBlock,
		ast.NodeTypeFileScope,
	}

	for _, nodeType := range nodeTypes {
		nodes, err := cg.readNodes(ctx, nodeType, map[string]any{"id": int64(nodeID)})
		if err == nil && len(nodes) > 0 {
			return nodes[0], nil
		}
	}

	return nil, fmt.Errorf("node with id %d not found", nodeID)
}

// RelationInfo represents a relationship between nodes
type RelationInfo struct {
	FromNodeID ast.NodeID
	ToNodeID   ast.NodeID
	Label      string
}

// GetChildNodes returns all child nodes of a given parent via a relationship
func (cg *CodeGraph) GetChildNodes(ctx context.Context, parentID ast.NodeID, relationLabel string, childType ast.NodeType) ([]*ast.Node, error) {
	childLabel := cg.getNodeLabel(childType)

	query := fmt.Sprintf(`
		MATCH (parent {id: $parentId})-[:%s]->(child:%s)
		RETURN child
	`, relationLabel, childLabel)

	records, err := cg.db.ExecuteRead(ctx, query, map[string]any{"parentId": int64(parentID)})
	if err != nil {
		return nil, fmt.Errorf("failed to get child nodes: %w", err)
	}

	var results []*ast.Node
	for _, record := range records {
		childData, ok := record["child"]
		if !ok || childData == nil {
			continue
		}

		childMap, ok := childData.(map[string]any)
		if !ok {
			continue
		}

		node, err := cg.recordToNode(childMap)
		if err != nil {
			continue
		}

		results = append(results, node)
	}

	return results, nil
}

// GetOutgoingRelations returns all outgoing relationships from a node
func (cg *CodeGraph) GetOutgoingRelations(ctx context.Context, fromNodeID ast.NodeID, relationLabel string) ([]RelationInfo, error) {
	query := fmt.Sprintf(`
		MATCH (from {id: $fromId})-[r:%s]->(to)
		RETURN to.id as toId
	`, relationLabel)

	records, err := cg.db.ExecuteRead(ctx, query, map[string]any{"fromId": int64(fromNodeID)})
	if err != nil {
		return nil, fmt.Errorf("failed to get outgoing relations: %w", err)
	}

	var results []RelationInfo
	for _, record := range records {
		toID, ok := record["toId"]
		if !ok {
			continue
		}

		toNodeID := cg.convertToInt64(toID)
		results = append(results, RelationInfo{
			FromNodeID: fromNodeID,
			ToNodeID:   ast.NodeID(toNodeID),
			Label:      relationLabel,
		})
	}

	return results, nil
}

// GetIncomingRelations returns all incoming relationships to a node
func (cg *CodeGraph) GetIncomingRelations(ctx context.Context, toNodeID ast.NodeID, relationLabel string) ([]RelationInfo, error) {
	query := fmt.Sprintf(`
		MATCH (from)-[r:%s]->(to {id: $toId})
		RETURN from.id as fromId
	`, relationLabel)

	records, err := cg.db.ExecuteRead(ctx, query, map[string]any{"toId": int64(toNodeID)})
	if err != nil {
		return nil, fmt.Errorf("failed to get incoming relations: %w", err)
	}

	var results []RelationInfo
	for _, record := range records {
		fromID, ok := record["fromId"]
		if !ok {
			continue
		}

		fromNodeID := cg.convertToInt64(fromID)
		results = append(results, RelationInfo{
			FromNodeID: ast.NodeID(fromNodeID),
			ToNodeID:   toNodeID,
			Label:      relationLabel,
		})
	}

	return results, nil
}

func (cg *CodeGraph) CreateUsesVariableRelation(ctx context.Context, userNodeID, variableNodeID ast.NodeID, fileID int32) error {
	return cg.CreateRelation(ctx, userNodeID, variableNodeID, "USES_VARIABLE", nil, fileID)
}

func (cg *CodeGraph) CreateImportsRelation(ctx context.Context, importerNodeID, importedNodeID ast.NodeID, fileID int32) error {
	return cg.CreateRelation(ctx, importerNodeID, importedNodeID, "IMPORTS", nil, fileID)
}

func (cg *CodeGraph) CreateBodyRelation(ctx context.Context, parentNodeID, bodyNodeID ast.NodeID, fileID int32) error {
	return cg.CreateRelation(ctx, parentNodeID, bodyNodeID, "BODY", nil, fileID)
}

func (cg *CodeGraph) CreateAnnotationRelation(ctx context.Context, parentNodeID, annotationNodeID ast.NodeID, fileID int32) error {
	return cg.CreateRelation(ctx, parentNodeID, annotationNodeID, "ANNOTATION", nil, fileID)
}

func (cg *CodeGraph) CreateFunctionArgRelation(ctx context.Context, functionNodeID, argNodeID ast.NodeID,
	position int, fileID int32) error {
	return cg.CreateRelation(ctx, functionNodeID, argNodeID, "FUNCTION_ARG", map[string]any{
		"position": position,
	}, fileID)
}

func (cg *CodeGraph) CreateFromRelation(ctx context.Context, fromNodeID, toNodeID ast.NodeID, fileID int32) error {
	return cg.CreateRelation(ctx, fromNodeID, toNodeID, "FROM", nil, fileID)
}

func (cg *CodeGraph) CreateDataFlowRelation(ctx context.Context, sourceNodeID, targetNodeID ast.NodeID, fileID int32) error {
	return cg.CreateRelation(ctx, sourceNodeID, targetNodeID, "DATA_FLOW", nil, fileID)
}

func (cg *CodeGraph) CreateFunctionCallArgRelation(ctx context.Context, callNodeID, argNodeID ast.NodeID,
	position int, fileID int32) error {
	return cg.CreateRelation(ctx, callNodeID, argNodeID, "FUNCTION_CALL_ARG", map[string]any{
		"position": position,
	}, fileID)
}

func (cg *CodeGraph) CreateReturnsRelation(ctx context.Context, functionNodeID, returnNodeID ast.NodeID, fileID int32) error {
	return cg.CreateRelation(ctx, functionNodeID, returnNodeID, "RETURNS", nil, fileID)
}

func (cg *CodeGraph) CreateAliasRelation(ctx context.Context, aliasNodeID, originalNodeID ast.NodeID, fileID int32) error {
	return cg.CreateRelation(ctx, aliasNodeID, originalNodeID, "ALIAS", nil, fileID)
}

func (cg *CodeGraph) CreateConditionalRelation(ctx context.Context, condNodeID,
	branchNodeID ast.NodeID, position int, conditionID ast.NodeID, fileID int32) error {
	return cg.CreateRelation(ctx, condNodeID, branchNodeID, "BRANCH", map[string]any{
		"position":  position,
		"condition": conditionID,
	}, fileID)
}

/*func (cg *CodeGraph) GetOrCreateNextFileID(ctx context.Context) (int32, error) {
	query := `
		MERGE (fn:FileNumber {id: -1})
		ON CREATE SET fn.max_file_id = 1
		ON MATCH SET fn.max_file_id = fn.max_file_id + 1
		RETURN fn.max_file_id as next_file_id
	`

	record, err := cg.db.ExecuteWriteSingle(ctx, query, nil)
	if err != nil {
		cg.logger.Error("Failed to get or create next file ID", zap.Error(err))
		return 0, fmt.Errorf("failed to get or create next file ID: %w", err)
	}

	nextFileID, ok := record["next_file_id"]
	if !ok {
		return 0, fmt.Errorf("failed to get next_file_id from result")
	}

	// Handle different numeric types from different database backends
	switch v := nextFileID.(type) {
	case int32:
		return v, nil
	case int64:
		return int32(v), nil
	case int:
		return int32(v), nil
	default:
		return 0, fmt.Errorf("unexpected type for next_file_id: %T", nextFileID)
	}
}
*/

func (cg *CodeGraph) FindFunctionCalls(ctx context.Context, fileID ast.NodeID) (map[ast.NodeID][]*ast.Node, error) {
	query := `
		MATCH (fc:FunctionCall {fileId: $fileId})<-[:CONTAINS*]-(f:Function {fileId: $fileId})
		RETURN fc, f.id AS functionId
	`

	parameters := map[string]any{
		"fileId": int64(fileID),
	}

	records, err := cg.db.ExecuteRead(ctx, query, parameters)
	if err != nil {
		cg.logger.Error("Failed to find function calls", zap.Error(err))
		return nil, fmt.Errorf("failed to find function calls: %w", err)
	}

	functionCalls := make(map[ast.NodeID][]*ast.Node)
	for _, record := range records {
		fcData, ok := record["fc"]
		if !ok || fcData == nil {
			continue
		}

		fcMap, ok := fcData.(map[string]any)
		if !ok {
			continue
		}

		node, err := cg.recordToNode(fcMap)
		if err != nil {
			return nil, fmt.Errorf("failed to convert record to node: %w", err)
		}

		functionId, ok := record["functionId"]
		if !ok {
			continue
		}

		functionCalls[ast.NodeID(functionId.(int64))] =
			append(functionCalls[ast.NodeID(functionId.(int64))], node)
	}

	return functionCalls, nil
}

func (cg *CodeGraph) FindFunctionsByName(ctx context.Context, fileID int, name string) ([]*ast.Node, error) {
	return cg.readNodes(ctx, ast.NodeTypeFunction, map[string]any{
		"name":   name,
		"fileId": fileID,
	})
}

// convertToInt64 safely converts various integer types to int64
func (cg *CodeGraph) convertToInt64(value any) int64 {
	switch v := value.(type) {
	case int64:
		return v
	case int32:
		return int64(v)
	case int:
		return int64(v)
	case uint64:
		return int64(v)
	case uint32:
		return int64(v)
	case uint:
		return int64(v)
	default:
		cg.logger.Warn("Unexpected type for int64 conversion", zap.Any("value", value), zap.String("type", fmt.Sprintf("%T", value)))
		return 0
	}
}

// convertToInt32 safely converts various integer types to int32
func (cg *CodeGraph) convertToInt32(value any) int32 {
	switch v := value.(type) {
	case int32:
		return v
	case int64:
		return int32(v)
	case int:
		return int32(v)
	case uint32:
		return int32(v)
	case uint64:
		return int32(v)
	case uint:
		return int32(v)
	default:
		cg.logger.Warn("Unexpected type for int32 conversion", zap.Any("value", value), zap.String("type", fmt.Sprintf("%T", value)))
		return 0
	}
}

// UpdateNodeMetaData updates the metadata of an existing node
// Works in both batch and non-batch write modes
// In batch mode: updates the buffered node if it exists, otherwise performs immediate update
// In non-batch mode: performs immediate update to database
func (cg *CodeGraph) UpdateNodeMetaData(ctx context.Context, nodeID ast.NodeID, fileID int32, metadata map[string]any) error {
	if len(metadata) == 0 {
		return fmt.Errorf("metadata cannot be nil or empty")
	}

	// If batch writes are enabled, try to update the buffered node first
	if cg.enableBatchWrites {
		cg.bufferMutex.Lock()
		buffer := cg.buffers[fileID]
		cg.bufferMutex.Unlock()

		if buffer != nil {
			// Try to find the node in the buffer
			for _, node := range buffer.Nodes {
				if node.ID == nodeID {
					// Update the node's metadata in the buffer
					if node.MetaData == nil {
						node.MetaData = make(map[string]any)
					}
					for key, value := range metadata {
						node.MetaData[key] = value
					}
					cg.logger.Debug("Updated node metadata in buffer",
						zap.Int64("nodeId", int64(nodeID)),
						zap.Int32("fileId", fileID))
					return nil
				}
			}
			// Node not found in buffer, fall through to immediate update
		}
	}

	// Perform immediate update (either batch mode disabled, or node not in buffer)
	return cg.updateNodeMetaDataReal(ctx, nodeID, metadata)
}

// updateNodeMetaDataReal performs the actual database update
func (cg *CodeGraph) updateNodeMetaDataReal(ctx context.Context, nodeID ast.NodeID, metadata map[string]any) error {
	// Prepare parameters for the update
	parameters := make(map[string]any)
	newMetadata := make(map[string]any)

	// Process metadata to separate first-class properties from nested metadata
	cg.populateFirstClassMetadata(metadata, parameters, newMetadata)

	// Flatten remaining metadata
	if len(newMetadata) > 0 {
		cg.flattenMetadata(newMetadata, parameters)
	}

	if len(parameters) == 0 {
		return fmt.Errorf("no valid metadata to update")
	}

	// Build SET clause
	setQ := cg.mapToSetParamString(parameters, "n")

	// Add node ID to parameters
	parameters["id"] = int64(nodeID)

	// Remove the id from SET clause since we use it in MATCH
	/*
		setQ = strings.Replace(setQ, "n.id = $id, ", "", 1)
		setQ = strings.Replace(setQ, ", n.id = $id", "", 1)
		setQ = strings.Replace(setQ, "n.id = $id", "", 1)
	*/

	if strings.TrimSpace(setQ) == "" {
		return fmt.Errorf("no properties to update after processing")
	}

	// Update the node using MATCH + SET
	query := fmt.Sprintf(`
		MATCH (n {id: $id})
		SET %s
		RETURN n
	`, setQ)

	records, err := cg.db.ExecuteWrite(ctx, query, parameters)
	if err != nil {
		cg.logger.Error("Failed to update node metadata",
			zap.Int64("nodeId", int64(nodeID)),
			zap.Error(err))
		return fmt.Errorf("failed to update node metadata: %w", err)
	}

	if len(records) == 0 {
		return fmt.Errorf("node with id %d not found", nodeID)
	}

	cg.logger.Debug("Updated node metadata in database",
		zap.Int64("nodeId", int64(nodeID)),
		zap.Int("propertiesUpdated", len(parameters)-1))

	return nil
}

// BatchUpdateNodeMetaData updates metadata for multiple nodes in a single transaction
// This is more efficient than calling UpdateNodeMetaData multiple times
func (cg *CodeGraph) BatchUpdateNodeMetaData(ctx context.Context, updates map[ast.NodeID]map[string]any) error {
	if len(updates) == 0 {
		return nil
	}

	cg.logger.Debug("Batch updating node metadata", zap.Int("count", len(updates)))

	// If batch writes are enabled, try to update buffered nodes first
	if cg.enableBatchWrites {
		remainingUpdates := make(map[ast.NodeID]map[string]any)

		// Group updates by fileID for efficient buffer lookup
		for nodeID, metadata := range updates {
			updated := false

			// Check all buffers (we don't know which file each node belongs to)
			cg.bufferMutex.Lock()
			for _, buffer := range cg.buffers {
				for _, node := range buffer.Nodes {
					if node.ID == nodeID {
						// Update the node's metadata in the buffer
						if node.MetaData == nil {
							node.MetaData = make(map[string]any)
						}
						for key, value := range metadata {
							node.MetaData[key] = value
						}
						updated = true
						break
					}
				}
				if updated {
					break
				}
			}
			cg.bufferMutex.Unlock()

			if !updated {
				remainingUpdates[nodeID] = metadata
			}
		}

		// If all nodes were updated in buffers, we're done
		if len(remainingUpdates) == 0 {
			cg.logger.Debug("All nodes updated in buffers", zap.Int("count", len(updates)))
			return nil
		}

		// Update remaining nodes in database
		updates = remainingUpdates
	}

	// Perform batch update in database
	// Build a single query with UNWIND for all updates
	var updateItems []map[string]any
	for nodeID, metadata := range updates {
		parameters := make(map[string]any)
		newMetadata := make(map[string]any)

		cg.populateFirstClassMetadata(metadata, parameters, newMetadata)
		if len(newMetadata) > 0 {
			cg.flattenMetadata(newMetadata, parameters)
		}

		if len(parameters) > 0 {
			parameters["id"] = int64(nodeID)
			updateItems = append(updateItems, parameters)
		}
	}

	if len(updateItems) == 0 {
		return fmt.Errorf("no valid metadata to update")
	}

	// Use UNWIND to process all updates
	query := `
		UNWIND $updates as update
		MATCH (n {id: update.id})
		SET n += update
		RETURN count(n) as updated
	`

	records, err := cg.db.ExecuteWrite(ctx, query, map[string]any{"updates": updateItems})
	if err != nil {
		cg.logger.Error("Failed to batch update node metadata", zap.Error(err))
		return fmt.Errorf("failed to batch update node metadata: %w", err)
	}

	updatedCount := int64(0)
	if len(records) > 0 {
		if count, ok := records[0]["updated"]; ok {
			updatedCount = cg.convertToInt64(count)
		}
	}

	cg.logger.Debug("Batch updated node metadata in database",
		zap.Int64("nodesUpdated", updatedCount),
		zap.Int("requested", len(updates)))

	return nil
}

func (cg *CodeGraph) FindClassInModule(ctx context.Context, name string, moduleName string) ([]*ast.Node, error) {
	q := `MATCH (n:Class {name: $name})
	MATCH (m: ModuleScope {name: $moduleName})
	WHERE (m)-[:CONTAINS]->(n)
	RETURN n
	`

	return cg.readNodesByQuery(ctx, "n", q, map[string]any{
		"name":       name,
		"moduleName": moduleName,
	})
}

// FindClassesByNameInRepo finds all classes with the given name in a repository.
// This is useful for inheritance resolution where a class may extend another class in a different file.
func (cg *CodeGraph) FindClassesByNameInRepo(ctx context.Context, name string, repoName string) ([]*ast.Node, error) {
	q := `MATCH (f:FileScope {repo: $repo})-[:CONTAINS]->(m:ModuleScope)-[:CONTAINS]->(c:Class {name: $name})
	RETURN c
	`

	return cg.readNodesByQuery(ctx, "c", q, map[string]any{
		"name": name,
		"repo": repoName,
	})
}

// FindAllClassesWithInheritance returns all classes in a repository that have extends or implements metadata.
func (cg *CodeGraph) FindAllClassesWithInheritance(ctx context.Context, repoName string) ([]*ast.Node, error) {
	q := `MATCH (f:FileScope {repo: $repo})-[:CONTAINS]->(m:ModuleScope)-[:CONTAINS]->(c:Class)
	WHERE c.extends IS NOT NULL OR c.implements IS NOT NULL
	RETURN c
	`

	return cg.readNodesByQuery(ctx, "c", q, map[string]any{
		"repo": repoName,
	})
}

// FindAllClassesInFile returns all classes in a file.
func (cg *CodeGraph) FindAllClassesInFile(ctx context.Context, fileID int32) ([]*ast.Node, error) {
	q := `MATCH (f:FileScope {id: $fileId})-[:CONTAINS]->(m:ModuleScope)-[:CONTAINS]->(c:Class)
	RETURN c
	`

	return cg.readNodesByQuery(ctx, "c", q, map[string]any{
		"fileId": fileID,
	})
}

// FindConstructorCallsInFile returns all constructor calls (new expressions) in a file.
// These are FunctionCall nodes with is_constructor=true metadata.
func (cg *CodeGraph) FindConstructorCallsInFile(ctx context.Context, fileID int32) ([]*ast.Node, error) {
	q := `MATCH (fc:FunctionCall {fileId: $fileId})
	WHERE fc.is_constructor = true
	RETURN fc
	`

	return cg.readNodesByQuery(ctx, "fc", q, map[string]any{
		"fileId": fileID,
	})
}

// GetConstructorsOfClass returns all constructors of a class.
// These are Function nodes with is_constructor=true contained by the class.
func (cg *CodeGraph) GetConstructorsOfClass(ctx context.Context, classID ast.NodeID) ([]*ast.Node, error) {
	query := `
		MATCH (c:Class {id: $classId})-[:CONTAINS]->(f:Function)
		WHERE f.is_constructor = true
		RETURN f
	`
	return cg.readNodesByQuery(ctx, "f", query, map[string]any{"classId": int64(classID)})
}

/*
func (cg *CodeGraph) FindClassInFile(ctx context.Context, name string, fileId int32) ([]*ast.Node, error) {
	return cg.readNodeByType(ctx)
}
*/

func (t *CodeGraph) MarkThis(ctx context.Context, fileID int32, thisNodeId ast.NodeID, classNodeId ast.NodeID) {
	_ = t.CreateRelation(ctx, thisNodeId, classNodeId, "THIS", nil, fileID)
}

// GetMethodsOfClass returns all methods (functions) contained by a class
func (cg *CodeGraph) GetMethodsOfClass(ctx context.Context, classID ast.NodeID) ([]*ast.Node, error) {
	query := `
		MATCH (c:Class {id: $classId})-[:CONTAINS]->(m:Function)
		RETURN m
	`
	return cg.readNodesByQuery(ctx, "m", query, map[string]any{"classId": int64(classID)})
}

// GetFieldsOfClass returns all fields contained by a class
func (cg *CodeGraph) GetFieldsOfClass(ctx context.Context, classID ast.NodeID) ([]*ast.Node, error) {
	query := `
		MATCH (c:Class {id: $classId})-[:CONTAINS]->(f:Field)
		RETURN f
	`
	return cg.readNodesByQuery(ctx, "f", query, map[string]any{"classId": int64(classID)})
}

// GetFieldsAccessedByMethod returns all Field nodes contained within a method (directly or nested)
func (cg *CodeGraph) GetFieldsAccessedByMethod(ctx context.Context, methodID ast.NodeID) ([]*ast.Node, error) {
	query := `
		MATCH (m:Function {id: $methodId})-[:CONTAINS*]->(f:Field)
		RETURN DISTINCT f
	`
	return cg.readNodesByQuery(ctx, "f", query, map[string]any{"methodId": int64(methodID)})
}

// GetFieldsAccessedViaThis returns fields accessed through the "this" receiver in a method
// Pattern: (method)-[:CONTAINS*]->(thisVar)-[:THIS]->(class), (thisVar)-[:HAS_FIELD*]->(field)
func (cg *CodeGraph) GetFieldsAccessedViaThis(ctx context.Context, methodID ast.NodeID) ([]*ast.Node, error) {
	query := `
		MATCH (m:Function {id: $methodId})-[:CONTAINS*]->(thisVar)-[:THIS]->(c:Class)
		MATCH (thisVar)-[:HAS_FIELD*]->(f:Field)
		RETURN DISTINCT f
	`
	return cg.readNodesByQuery(ctx, "f", query, map[string]any{"methodId": int64(methodID)})
}

// GetThisClassForMethod returns the class that the method's receiver (this) points to
func (cg *CodeGraph) GetThisClassForMethod(ctx context.Context, methodID ast.NodeID) (*ast.Node, error) {
	query := `
		MATCH (m:Function {id: $methodId})-[:CONTAINS*]->(thisVar)-[:THIS]->(c:Class)
		RETURN c
		LIMIT 1
	`
	nodes, err := cg.readNodesByQuery(ctx, "c", query, map[string]any{"methodId": int64(methodID)})
	if err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return nil, nil
	}
	return nodes[0], nil
}

// GetContainingClass returns the class that contains a method
func (cg *CodeGraph) GetContainingClass(ctx context.Context, methodID ast.NodeID) (*ast.Node, error) {
	query := `
		MATCH (c:Class)-[:CONTAINS]->(m:Function {id: $methodId})
		RETURN c
		LIMIT 1
	`
	nodes, err := cg.readNodesByQuery(ctx, "c", query, map[string]any{"methodId": int64(methodID)})
	if err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return nil, nil
	}
	return nodes[0], nil
}

// GetFieldOwnerClass returns the class that owns a field
func (cg *CodeGraph) GetFieldOwnerClass(ctx context.Context, fieldID ast.NodeID) (*ast.Node, error) {
	query := `
		MATCH (c:Class)-[:CONTAINS]->(f:Field {id: $fieldId})
		RETURN c
		LIMIT 1
	`
	nodes, err := cg.readNodesByQuery(ctx, "c", query, map[string]any{"fieldId": int64(fieldID)})
	if err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return nil, nil
	}
	return nodes[0], nil
}

func (cg *CodeGraph) GetModuleName(ctx context.Context, fileId int32) (string, error) {
	// Query the database (either batch mode disabled, or module not in buffer)
	query := `
		MATCH (f:FileScope {id: $fileId})-[:CONTAINS]->(m:ModuleScope)
		RETURN m.name AS moduleName
	`

	parameters := map[string]any{
		"fileId": fileId,
	}

	records, err := cg.db.ExecuteRead(ctx, query, parameters)
	if err != nil {
		cg.logger.Error("Failed to get module name", zap.Error(err))
		return "", fmt.Errorf("failed to get module name: %w", err)
	}

	if len(records) == 0 {
		return "", fmt.Errorf("module not found for file node ID %d", fileId)
	}

	moduleName, ok := records[0]["moduleName"]
	if !ok {
		return "", fmt.Errorf("moduleName not found in query result")
	}

	return moduleName.(string), nil
}

func (cg *CodeGraph) UpdateFakeClasses(ctx context.Context, fileID int32) error {
	// find all the modules in the given file scope
	moduleQuery := `
		MATCH(m:ModuleScope {fileId: $fileID})
		RETURN m
	`

	moduleParameters := map[string]any{
		"fileID": fileID,
	}

	moduleRecords, err := cg.readNodesByQuery(ctx, "m", moduleQuery, moduleParameters)
	if err != nil {
		return fmt.Errorf("failed to read modules: %w", err)
	}

	if len(moduleRecords) != 1 {
		return fmt.Errorf("expected exactly one module in file %d, found %d", fileID, len(moduleRecords))
	}

	moduleNode := moduleRecords[0]

	// find all fake classes in the given file scope
	query := `
		MATCH (c:Class {fileId: $fileID, is_fake: true})
		RETURN c
	`

	parameters := map[string]any{
		"fileID": fileID,
	}

	records, err := cg.readNodesByQuery(ctx, "c", query, parameters)
	if err != nil {
		return fmt.Errorf("failed to read fake classes: %w", err)
	}

	for _, fakeClass := range records {
		// find actual class in module with same name
		actualClasses, err := cg.FindClassInModule(ctx, fakeClass.Name, moduleNode.Name)
		if err != nil {
			return fmt.Errorf("failed to find actual class in module: %w", err)
		}

		if len(actualClasses) == 1 {
			// move all children of fake class to actual class
			moveQuery := `
				MATCH (fake:Class {id: $fakeClassID})-[r:CONTAINS]->(child)
				MATCH (actual:Class {id: $actualClassID})
				MERGE (actual)-[:CONTAINS]->(child)
				DELETE r
			`
			moveParameters := map[string]any{
				"fakeClassID":   int64(fakeClass.ID),
				"actualClassID": int64(actualClasses[0].ID),
			}
			_, err := cg.db.ExecuteWrite(ctx, moveQuery, moveParameters)
			if err != nil {
				return fmt.Errorf("failed to move children from fake class to actual class: %w", err)
			}

			// delete fake class
			deleteQuery := `
				MATCH (fake:Class {id: $fakeClassID})
				DETACH DELETE fake
			`
			deleteParameters := map[string]any{
				"fakeClassID": int64(fakeClass.ID),
			}
			_, err = cg.db.ExecuteWrite(ctx, deleteQuery, deleteParameters)
			if err != nil {
				return fmt.Errorf("failed to delete fake class: %w", err)
			}

			cg.logger.Debug("Replaced fake class with actual class",
				zap.String("className", fakeClass.Name),
				zap.Int64("fakeClassID", int64(fakeClass.ID)),
				zap.Int64("actualClassID", int64(actualClasses[0].ID)))
		}
	}
	return nil
}

// IsFieldWrittenInMethod checks if a field has an incoming DATA_FLOW relationship
// within the scope of a method, indicating the field is being written to
func (cg *CodeGraph) IsFieldWrittenInMethod(ctx context.Context, methodID ast.NodeID, fieldID ast.NodeID) (bool, error) {
	// Check if there's a DATA_FLOW relationship targeting this field
	// within the method's scope
	query := `
		MATCH (m:Function {id: $methodId})-[:CONTAINS*]->(source)-[:DATA_FLOW]->(f:Field {id: $fieldId})
		RETURN count(f) > 0 AS isWritten
	`
	parameters := map[string]any{
		"methodId": int64(methodID),
		"fieldId":  int64(fieldID),
	}

	results, err := cg.db.ExecuteRead(ctx, query, parameters)
	if err != nil {
		return false, fmt.Errorf("failed to check field write: %w", err)
	}

	if len(results) > 0 && results[0]["isWritten"] != nil {
		if isWritten, ok := results[0]["isWritten"].(bool); ok {
			return isWritten, nil
		}
	}
	return false, nil
}

// GetFieldsWrittenByMethod returns all fields that are written to within a method
// (fields that are targets of DATA_FLOW relationships)
func (cg *CodeGraph) GetFieldsWrittenByMethod(ctx context.Context, methodID ast.NodeID) ([]*ast.Node, error) {
	query := `
		MATCH (m:Function {id: $methodId})-[:CONTAINS*]->(source)-[:DATA_FLOW]->(f:Field)
		RETURN DISTINCT f
	`
	return cg.readNodesByQuery(ctx, "f", query, map[string]any{"methodId": int64(methodID)})
}

// DumpToFile dumps the code graph for the specified repositories to a file.
// FileScopes are output in alphabetical order by their path.
// For each FileScope, all nodes and relations within that file are dumped.
func (cg *CodeGraph) DumpToFile(ctx context.Context, filePath string, repoNames []string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create dump file: %w", err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	defer writer.Flush()

	// Write header
	fmt.Fprintf(writer, "# Code Graph Dump\n")
	fmt.Fprintf(writer, "# Repositories: %s\n", strings.Join(repoNames, ", "))
	fmt.Fprintf(writer, "# Generated at: %s\n\n", time.Now().Format(time.RFC3339))

	// For each repository
	for _, repoName := range repoNames {
		fmt.Fprintf(writer, "================================================================================\n")
		fmt.Fprintf(writer, "REPOSITORY: %s\n", repoName)
		fmt.Fprintf(writer, "================================================================================\n\n")

		// Get all FileScopes for this repository
		fileScopes, err := cg.FindFileScopes(ctx, repoName, "")
		if err != nil {
			cg.logger.Error("Failed to find file scopes", zap.String("repo", repoName), zap.Error(err))
			fmt.Fprintf(writer, "ERROR: Failed to find file scopes: %v\n\n", err)
			continue
		}

		if len(fileScopes) == 0 {
			fmt.Fprintf(writer, "No file scopes found for repository.\n\n")
			continue
		}

		// Sort FileScopes by path alphabetically
		sort.Slice(fileScopes, func(i, j int) bool {
			pathI := ""
			pathJ := ""
			if fileScopes[i].MetaData != nil {
				if p, ok := fileScopes[i].MetaData["path"].(string); ok {
					pathI = p
				}
			}
			if fileScopes[j].MetaData != nil {
				if p, ok := fileScopes[j].MetaData["path"].(string); ok {
					pathJ = p
				}
			}
			return pathI < pathJ
		})

		fmt.Fprintf(writer, "Total files: %d\n\n", len(fileScopes))

		// For each FileScope, dump all nodes and relations
		for _, fs := range fileScopes {
			filePath := ""
			if fs.MetaData != nil {
				if p, ok := fs.MetaData["path"].(string); ok {
					filePath = p
				}
			}

			fmt.Fprintf(writer, "--------------------------------------------------------------------------------\n")
			fmt.Fprintf(writer, "FILE: %s (FileID: %d)\n", filePath, fs.FileID)
			fmt.Fprintf(writer, "--------------------------------------------------------------------------------\n\n")

			// Dump the FileScope node itself
			fmt.Fprintf(writer, "## Nodes\n\n")
			cg.writeNodeToFile(writer, fs, 0)

			// Get all nodes in this file
			nodesInFile, err := cg.getAllNodesInFile(ctx, fs.FileID)
			if err != nil {
				cg.logger.Error("Failed to get nodes in file", zap.Int32("fileId", fs.FileID), zap.Error(err))
				fmt.Fprintf(writer, "ERROR: Failed to get nodes: %v\n\n", err)
				continue
			}

			// Sort nodes by ID for consistent output
			sort.Slice(nodesInFile, func(i, j int) bool {
				return nodesInFile[i].ID < nodesInFile[j].ID
			})

			for _, node := range nodesInFile {
				cg.writeNodeToFile(writer, node, 1)
			}

			// Get all relations for this file
			fmt.Fprintf(writer, "\n## Relations\n\n")
			relations, err := cg.getAllRelationsInFile(ctx, fs.FileID)
			if err != nil {
				cg.logger.Error("Failed to get relations in file", zap.Int32("fileId", fs.FileID), zap.Error(err))
				fmt.Fprintf(writer, "ERROR: Failed to get relations: %v\n\n", err)
				continue
			}

			// Sort relations for consistent output
			sort.Slice(relations, func(i, j int) bool {
				if relations[i].fromID != relations[j].fromID {
					return relations[i].fromID < relations[j].fromID
				}
				if relations[i].relType != relations[j].relType {
					return relations[i].relType < relations[j].relType
				}
				return relations[i].toID < relations[j].toID
			})

			for _, rel := range relations {
				fmt.Fprintf(writer, "  (%d) -[%s]-> (%d)\n", rel.fromID, rel.relType, rel.toID)
			}

			fmt.Fprintf(writer, "\nTotal nodes in file: %d\n", len(nodesInFile)+1) // +1 for FileScope
			fmt.Fprintf(writer, "Total relations in file: %d\n\n", len(relations))
		}
	}

	return nil
}

// relationInfo holds information about a relationship for dumping
type relationInfo struct {
	fromID  int64
	toID    int64
	relType string
}

// writeNodeToFile writes a single node to the dump file
func (cg *CodeGraph) writeNodeToFile(writer *bufio.Writer, node *ast.Node, indent int) {
	indentStr := strings.Repeat("  ", indent)
	nodeTypeName := cg.getNodeLabel(node.NodeType)

	fmt.Fprintf(writer, "%s[%s] ID:%d Name:%q Range:%s\n",
		indentStr, nodeTypeName, node.ID, node.Name, rangeToString(node.Range))

	// Print metadata if present
	if node.MetaData != nil && len(node.MetaData) > 0 {
		// Sort metadata keys for consistent output
		keys := make([]string, 0, len(node.MetaData))
		for k := range node.MetaData {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, k := range keys {
			v := node.MetaData[k]
			fmt.Fprintf(writer, "%s    %s: %v\n", indentStr, k, v)
		}
	}
}

// getAllNodesInFile retrieves all nodes (except FileScope) that belong to a specific file
func (cg *CodeGraph) getAllNodesInFile(ctx context.Context, fileID int32) ([]*ast.Node, error) {
	// Query all node types except FileScope and FileNumber
	query := `
		MATCH (n)
		WHERE n.fileId = $fileId
		  AND n.nodeType <> $fileScopeType
		  AND n.nodeType <> $fileNumberType
		RETURN n
	`
	params := map[string]any{
		"fileId":         int64(fileID),
		"fileScopeType":  int64(ast.NodeTypeFileScope),
		"fileNumberType": int64(ast.NodeTypeFileNumber),
	}

	return cg.readNodesByQuery(ctx, "n", query, params)
}

// getAllRelationsInFile retrieves all relationships where either the source or target is in the file
func (cg *CodeGraph) getAllRelationsInFile(ctx context.Context, fileID int32) ([]relationInfo, error) {
	query := `
		MATCH (from)-[r]->(to)
		WHERE from.fileId = $fileId OR to.fileId = $fileId
		RETURN from.id as fromId, type(r) as relType, to.id as toId
	`
	params := map[string]any{
		"fileId": int64(fileID),
	}

	records, err := cg.db.ExecuteRead(ctx, query, params)
	if err != nil {
		return nil, fmt.Errorf("failed to get relations: %w", err)
	}

	var relations []relationInfo
	for _, record := range records {
		fromID := cg.convertToInt64(record["fromId"])
		toID := cg.convertToInt64(record["toId"])
		relType, _ := record["relType"].(string)

		relations = append(relations, relationInfo{
			fromID:  fromID,
			toID:    toID,
			relType: relType,
		})
	}

	return relations, nil
}

// CleanRepository deletes all nodes and relationships for a specific repository from Neo4j.
// This includes all FileScopes and their descendant nodes (functions, classes, variables, etc.)
func (cg *CodeGraph) CleanRepository(ctx context.Context, repoName string) error {
	cg.logger.Info("Starting Neo4j cleanup for repository", zap.String("repo", repoName))

	// First, get count of nodes to be deleted for logging
	countQuery := `
		MATCH (fs:FileScope {repo: $repo})
		OPTIONAL MATCH (fs)-[:CONTAINS*]->(descendant)
		RETURN count(DISTINCT fs) as fileScopeCount, count(DISTINCT descendant) as descendantCount
	`
	countResult, err := cg.db.ExecuteReadSingle(ctx, countQuery, map[string]any{"repo": repoName})
	if err != nil {
		cg.logger.Warn("Failed to count nodes for deletion", zap.Error(err))
	} else {
		fsCount := cg.convertToInt64(countResult["fileScopeCount"])
		descCount := cg.convertToInt64(countResult["descendantCount"])
		cg.logger.Info("Nodes to be deleted",
			zap.String("repo", repoName),
			zap.Int64("file_scopes", fsCount),
			zap.Int64("descendants", descCount))
	}

	// Delete all descendant nodes first (nodes connected via CONTAINS relationships)
	// Using DETACH DELETE to also remove all relationships
	deleteDescendantsQuery := `
		MATCH (fs:FileScope {repo: $repo})-[:CONTAINS*]->(descendant)
		DETACH DELETE descendant
	`
	_, err = cg.db.ExecuteWrite(ctx, deleteDescendantsQuery, map[string]any{"repo": repoName})
	if err != nil {
		return fmt.Errorf("failed to delete descendant nodes: %w", err)
	}
	cg.logger.Debug("Deleted descendant nodes", zap.String("repo", repoName))

	// Now delete the FileScope nodes themselves
	deleteFileScopesQuery := `
		MATCH (fs:FileScope {repo: $repo})
		DETACH DELETE fs
	`
	_, err = cg.db.ExecuteWrite(ctx, deleteFileScopesQuery, map[string]any{"repo": repoName})
	if err != nil {
		return fmt.Errorf("failed to delete FileScope nodes: %w", err)
	}
	cg.logger.Debug("Deleted FileScope nodes", zap.String("repo", repoName))

	cg.logger.Info("Neo4j cleanup completed for repository", zap.String("repo", repoName))
	return nil
}

// ExecuteRead executes a read-only Cypher query and returns the raw records.
// This is exposed for use by higher-level query APIs (e.g., codeapi package).
func (cg *CodeGraph) ExecuteRead(ctx context.Context, query string, params map[string]any) ([]map[string]any, error) {
	return cg.db.ExecuteRead(ctx, query, params)
}

// ExecuteReadSingle executes a read-only Cypher query expecting a single record.
// Returns error if no records found.
func (cg *CodeGraph) ExecuteReadSingle(ctx context.Context, query string, params map[string]any) (map[string]any, error) {
	return cg.db.ExecuteReadSingle(ctx, query, params)
}

// ExecuteWrite executes a write Cypher query and returns the raw records.
// This is exposed for use by higher-level query APIs (e.g., codeapi package).
func (cg *CodeGraph) ExecuteWrite(ctx context.Context, query string, params map[string]any) ([]map[string]any, error) {
	return cg.db.ExecuteWrite(ctx, query, params)
}

// -----------------------------------------------------------------------------
// On-Demand Summary Support Methods
// -----------------------------------------------------------------------------

// FindFunctionByName finds a function by name within a file path
func (cg *CodeGraph) FindFunctionByName(ctx context.Context, filePath string, functionName string) (*ast.Node, error) {
	query := `
		MATCH (f:FileScope {filePath: $filePath})-[:CONTAINS*]->(fn:Function {name: $name})
		RETURN fn
		LIMIT 1
	`
	nodes, err := cg.readNodesByQuery(ctx, "fn", query, map[string]any{
		"filePath": filePath,
		"name":     functionName,
	})
	if err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return nil, nil
	}
	return nodes[0], nil
}

// FindClassByName finds a class by name within a file path
func (cg *CodeGraph) FindClassByName(ctx context.Context, filePath string, className string) (*ast.Node, error) {
	query := `
		MATCH (f:FileScope {filePath: $filePath})-[:CONTAINS*]->(c:Class {name: $name})
		RETURN c
		LIMIT 1
	`
	nodes, err := cg.readNodesByQuery(ctx, "c", query, map[string]any{
		"filePath": filePath,
		"name":     className,
	})
	if err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return nil, nil
	}
	return nodes[0], nil
}

// FindFileByPath finds a file node by its path in a repository
func (cg *CodeGraph) FindFileByPath(ctx context.Context, repoName string, filePath string) (*ast.Node, error) {
	query := `
		MATCH (f:FileScope {repo: $repo, path: $path})
		RETURN f
		LIMIT 1
	`
	nodes, err := cg.readNodesByQuery(ctx, "f", query, map[string]any{
		"repo": repoName,
		"path": filePath,
	})
	if err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return nil, nil
	}
	return nodes[0], nil
}

// GetClassMethods returns all method nodes for a class
func (cg *CodeGraph) GetClassMethods(ctx context.Context, classID ast.NodeID) ([]*ast.Node, error) {
	return cg.GetMethodsOfClass(ctx, classID)
}
