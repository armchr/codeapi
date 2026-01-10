package codeapi

import (
	"context"
	"fmt"

	"github.com/armchr/codeapi/internal/model/ast"
	"github.com/armchr/codeapi/internal/service/codegraph"

	"go.uber.org/zap"
)

// graphAnalyzerImpl implements GraphAnalyzer
type graphAnalyzerImpl struct {
	graph  *codegraph.CodeGraph
	logger *zap.Logger
}

func newGraphAnalyzerImpl(graph *codegraph.CodeGraph, logger *zap.Logger) *graphAnalyzerImpl {
	return &graphAnalyzerImpl{
		graph:  graph,
		logger: logger,
	}
}

// -----------------------------------------------------------------------------
// Call Graph Operations
// -----------------------------------------------------------------------------

func (a *graphAnalyzerImpl) GetCallGraph(ctx context.Context, functionID ast.NodeID, opts CallGraphOptions) (*CallGraph, error) {
	if opts.MaxDepth <= 0 {
		opts.MaxDepth = 3
	}

	result := &CallGraph{
		Nodes:     make(map[ast.NodeID]*CallNode),
		Edges:     make([]*CallEdge, 0),
		Direction: opts.Direction,
		MaxDepth:  opts.MaxDepth,
	}

	// Get the root function
	rootNode, err := a.getFunctionAsCallNode(ctx, functionID, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to get root function: %w", err)
	}
	result.Root = rootNode
	result.Nodes[functionID] = rootNode

	// Traverse based on direction
	visited := make(map[ast.NodeID]bool)
	visited[functionID] = true

	switch opts.Direction {
	case DirectionOutgoing:
		err = a.traverseCallees(ctx, functionID, 1, opts.MaxDepth, result, visited, opts)
	case DirectionIncoming:
		err = a.traverseCallers(ctx, functionID, 1, opts.MaxDepth, result, visited, opts)
	case DirectionBoth:
		err = a.traverseCallees(ctx, functionID, 1, opts.MaxDepth, result, visited, opts)
		if err == nil {
			err = a.traverseCallers(ctx, functionID, 1, opts.MaxDepth, result, visited, opts)
		}
	}

	if err != nil {
		return nil, err
	}

	return result, nil
}

func (a *graphAnalyzerImpl) GetCallGraphByName(ctx context.Context, repoName, filePath, className, functionName string, opts CallGraphOptions) (*CallGraph, error) {
	// Find the function
	functionID, err := a.findFunctionID(ctx, repoName, filePath, className, functionName)
	if err != nil {
		return nil, err
	}

	return a.GetCallGraph(ctx, functionID, opts)
}

func (a *graphAnalyzerImpl) GetCallers(ctx context.Context, functionID ast.NodeID, maxDepth int) (*CallGraph, error) {
	return a.GetCallGraph(ctx, functionID, CallGraphOptions{
		Direction: DirectionIncoming,
		MaxDepth:  maxDepth,
	})
}

func (a *graphAnalyzerImpl) GetCallees(ctx context.Context, functionID ast.NodeID, maxDepth int) (*CallGraph, error) {
	return a.GetCallGraph(ctx, functionID, CallGraphOptions{
		Direction: DirectionOutgoing,
		MaxDepth:  maxDepth,
	})
}

func (a *graphAnalyzerImpl) traverseCallees(ctx context.Context, functionID ast.NodeID, depth, maxDepth int, result *CallGraph, visited map[ast.NodeID]bool, opts CallGraphOptions) error {
	if depth > maxDepth {
		result.Truncated = true
		return nil
	}

	// Query: function -[:CONTAINS]-> functionCall -[:CALLS_FUNCTION]-> callee
	query := `
		MATCH (f:Function {id: $functionId})-[:CONTAINS*]->(fc:FunctionCall)-[:CALLS_FUNCTION]->(callee:Function)
		RETURN DISTINCT callee.id AS calleeId, callee.name AS calleeName,
		       callee.fileId AS fileId, callee.range AS range,
		       fc.id AS callSiteId, fc.range AS callSiteRange
	`
	records, err := a.graph.ExecuteRead(ctx, query, map[string]any{"functionId": int64(functionID)})
	if err != nil {
		return fmt.Errorf("failed to query callees: %w", err)
	}

	for _, record := range records {
		calleeID := ast.NodeID(toInt64(record["calleeId"]))

		// Add edge
		result.Edges = append(result.Edges, &CallEdge{
			CallerID: functionID,
			CalleeID: calleeID,
			CallSite: &Location{
				FileID: int32(toInt64(record["fileId"])),
				Range:  parseRange(toString(record["callSiteRange"])),
			},
		})

		// Skip if already visited
		if visited[calleeID] {
			continue
		}
		visited[calleeID] = true

		// Add node
		node := &CallNode{
			ID:       calleeID,
			Name:     toString(record["calleeName"]),
			FileID:   int32(toInt64(record["fileId"])),
			Depth:    depth,
		}
		if rangeStr := toString(record["range"]); rangeStr != "" {
			node.Range = parseRange(rangeStr)
		}
		result.Nodes[calleeID] = node

		// Recurse
		if err := a.traverseCallees(ctx, calleeID, depth+1, maxDepth, result, visited, opts); err != nil {
			return err
		}
	}

	return nil
}

func (a *graphAnalyzerImpl) traverseCallers(ctx context.Context, functionID ast.NodeID, depth, maxDepth int, result *CallGraph, visited map[ast.NodeID]bool, opts CallGraphOptions) error {
	if depth > maxDepth {
		result.Truncated = true
		return nil
	}

	// Query: caller -[:CONTAINS]-> functionCall -[:CALLS_FUNCTION]-> function
	query := `
		MATCH (caller:Function)-[:CONTAINS*]->(fc:FunctionCall)-[:CALLS_FUNCTION]->(f:Function {id: $functionId})
		RETURN DISTINCT caller.id AS callerId, caller.name AS callerName,
		       caller.fileId AS fileId, caller.range AS range,
		       fc.id AS callSiteId, fc.range AS callSiteRange
	`
	records, err := a.graph.ExecuteRead(ctx, query, map[string]any{"functionId": int64(functionID)})
	if err != nil {
		return fmt.Errorf("failed to query callers: %w", err)
	}

	for _, record := range records {
		callerID := ast.NodeID(toInt64(record["callerId"]))

		// Add edge
		result.Edges = append(result.Edges, &CallEdge{
			CallerID: callerID,
			CalleeID: functionID,
			CallSite: &Location{
				FileID: int32(toInt64(record["fileId"])),
				Range:  parseRange(toString(record["callSiteRange"])),
			},
		})

		// Skip if already visited
		if visited[callerID] {
			continue
		}
		visited[callerID] = true

		// Add node
		node := &CallNode{
			ID:       callerID,
			Name:     toString(record["callerName"]),
			FileID:   int32(toInt64(record["fileId"])),
			Depth:    -depth, // negative depth for callers
		}
		if rangeStr := toString(record["range"]); rangeStr != "" {
			node.Range = parseRange(rangeStr)
		}
		result.Nodes[callerID] = node

		// Recurse
		if err := a.traverseCallers(ctx, callerID, depth+1, maxDepth, result, visited, opts); err != nil {
			return err
		}
	}

	return nil
}

// -----------------------------------------------------------------------------
// Data Flow Operations
// -----------------------------------------------------------------------------

func (a *graphAnalyzerImpl) GetDataDependents(ctx context.Context, nodeID ast.NodeID, opts DependencyOptions) (*DependencyGraph, error) {
	if opts.MaxDepth == 0 {
		opts.MaxDepth = -1 // unlimited by default
	}

	result := &DependencyGraph{
		Nodes:     make(map[ast.NodeID]*DependencyNode),
		Edges:     make([]*DependencyEdge, 0),
		Direction: DirectionOutgoing,
	}

	// Get the root node
	rootNode, err := a.getNodeAsDependencyNode(ctx, nodeID, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to get root node: %w", err)
	}
	result.Root = rootNode
	result.Nodes[nodeID] = rootNode

	// Traverse data flow edges outward
	visited := make(map[ast.NodeID]bool)
	visited[nodeID] = true

	err = a.traverseDataFlow(ctx, nodeID, 1, opts.MaxDepth, DirectionOutgoing, result, visited, opts)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (a *graphAnalyzerImpl) GetDataSources(ctx context.Context, nodeID ast.NodeID, opts DependencyOptions) (*DependencyGraph, error) {
	if opts.MaxDepth == 0 {
		opts.MaxDepth = -1
	}

	result := &DependencyGraph{
		Nodes:     make(map[ast.NodeID]*DependencyNode),
		Edges:     make([]*DependencyEdge, 0),
		Direction: DirectionIncoming,
	}

	rootNode, err := a.getNodeAsDependencyNode(ctx, nodeID, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to get root node: %w", err)
	}
	result.Root = rootNode
	result.Nodes[nodeID] = rootNode

	visited := make(map[ast.NodeID]bool)
	visited[nodeID] = true

	err = a.traverseDataFlow(ctx, nodeID, 1, opts.MaxDepth, DirectionIncoming, result, visited, opts)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (a *graphAnalyzerImpl) GetVariableDependents(ctx context.Context, repoName, filePath, variableName string, opts DependencyOptions) (*DependencyGraph, error) {
	// Find the variable
	query := `
		MATCH (v:Variable {name: $name})
		WHERE v.repo = $repo
	`
	params := map[string]any{"name": variableName, "repo": repoName}

	if filePath != "" {
		query += " AND v.path = $path"
		params["path"] = filePath
	}

	query += " RETURN v.id AS id LIMIT 1"

	records, err := a.graph.ExecuteRead(ctx, query, params)
	if err != nil {
		return nil, fmt.Errorf("failed to find variable: %w", err)
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("variable not found: %s", variableName)
	}

	varID := ast.NodeID(toInt64(records[0]["id"]))
	return a.GetDataDependents(ctx, varID, opts)
}

func (a *graphAnalyzerImpl) traverseDataFlow(ctx context.Context, nodeID ast.NodeID, depth, maxDepth int, direction Direction, result *DependencyGraph, visited map[ast.NodeID]bool, opts DependencyOptions) error {
	if maxDepth > 0 && depth > maxDepth {
		result.Truncated = true
		return nil
	}

	var query string
	if direction == DirectionOutgoing {
		query = `
			MATCH (source {id: $nodeId})-[:DATA_FLOW]->(target)
			RETURN target.id AS targetId, target.name AS name, target.nodeType AS nodeType,
			       target.fileId AS fileId
		`
	} else {
		query = `
			MATCH (source)-[:DATA_FLOW]->(target {id: $nodeId})
			RETURN source.id AS targetId, source.name AS name, source.nodeType AS nodeType,
			       source.fileId AS fileId
		`
	}

	records, err := a.graph.ExecuteRead(ctx, query, map[string]any{"nodeId": int64(nodeID)})
	if err != nil {
		return fmt.Errorf("failed to query data flow: %w", err)
	}

	for _, record := range records {
		targetID := ast.NodeID(toInt64(record["targetId"]))

		// Add edge
		if direction == DirectionOutgoing {
			result.Edges = append(result.Edges, &DependencyEdge{
				SourceID: nodeID,
				TargetID: targetID,
				FlowType: "data_flow",
			})
		} else {
			result.Edges = append(result.Edges, &DependencyEdge{
				SourceID: targetID,
				TargetID: nodeID,
				FlowType: "data_flow",
			})
		}

		if visited[targetID] {
			continue
		}
		visited[targetID] = true

		// Filter by node type if specified
		nodeType := ast.NodeType(toInt64(record["nodeType"]))
		if len(opts.FilterTypes) > 0 {
			found := false
			for _, t := range opts.FilterTypes {
				if t == nodeType {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		// Add node
		node := &DependencyNode{
			ID:       targetID,
			Name:     toString(record["name"]),
			NodeType: nodeType,
			FileID:   int32(toInt64(record["fileId"])),
			Depth:    depth,
		}
		result.Nodes[targetID] = node

		// Recurse if including indirect
		if opts.IncludeIndirect {
			if err := a.traverseDataFlow(ctx, targetID, depth+1, maxDepth, direction, result, visited, opts); err != nil {
				return err
			}
		}
	}

	return nil
}

// -----------------------------------------------------------------------------
// Field Access Operations
// -----------------------------------------------------------------------------

func (a *graphAnalyzerImpl) GetFieldAccessors(ctx context.Context, fieldID ast.NodeID) (*FieldAccessResult, error) {
	// Get the field info
	fieldQuery := `
		MATCH (f:Field {id: $fieldId})
		RETURN f.name AS name, f.type AS type
	`
	fieldRecords, err := a.graph.ExecuteRead(ctx, fieldQuery, map[string]any{"fieldId": int64(fieldID)})
	if err != nil {
		return nil, fmt.Errorf("failed to get field: %w", err)
	}
	if len(fieldRecords) == 0 {
		return nil, fmt.Errorf("field not found: %d", fieldID)
	}

	result := &FieldAccessResult{
		Field: &FieldInfo{
			ID:   fieldID,
			Name: toString(fieldRecords[0]["name"]),
			Type: toString(fieldRecords[0]["type"]),
		},
		Readers: make([]*MethodAccessInfo, 0),
		Writers: make([]*MethodAccessInfo, 0),
	}

	// Find methods that read this field (via HAS_FIELD)
	readerQuery := `
		MATCH (m:Function)-[:CONTAINS*]->(accessor)-[:HAS_FIELD]->(f:Field {id: $fieldId})
		WHERE NOT EXISTS { (accessor)-[:DATA_FLOW]->(f) }
		RETURN DISTINCT m.id AS methodId, m.name AS methodName,
		       m.fileId AS fileId, count(*) AS accessCount
	`
	readerRecords, err := a.graph.ExecuteRead(ctx, readerQuery, map[string]any{"fieldId": int64(fieldID)})
	if err != nil {
		a.logger.Warn("Failed to query field readers", zap.Error(err))
	} else {
		for _, record := range readerRecords {
			result.Readers = append(result.Readers, &MethodAccessInfo{
				Method: &MethodInfo{
					ID:     ast.NodeID(toInt64(record["methodId"])),
					Name:   toString(record["methodName"]),
					FileID: int32(toInt64(record["fileId"])),
				},
				AccessCount: int(toInt64(record["accessCount"])),
			})
		}
	}

	// Find methods that write this field (via DATA_FLOW)
	writerQuery := `
		MATCH (m:Function)-[:CONTAINS*]->(source)-[:DATA_FLOW]->(f:Field {id: $fieldId})
		RETURN DISTINCT m.id AS methodId, m.name AS methodName,
		       m.fileId AS fileId, count(*) AS accessCount
	`
	writerRecords, err := a.graph.ExecuteRead(ctx, writerQuery, map[string]any{"fieldId": int64(fieldID)})
	if err != nil {
		a.logger.Warn("Failed to query field writers", zap.Error(err))
	} else {
		for _, record := range writerRecords {
			result.Writers = append(result.Writers, &MethodAccessInfo{
				Method: &MethodInfo{
					ID:     ast.NodeID(toInt64(record["methodId"])),
					Name:   toString(record["methodName"]),
					FileID: int32(toInt64(record["fileId"])),
				},
				AccessCount: int(toInt64(record["accessCount"])),
			})
		}
	}

	return result, nil
}

func (a *graphAnalyzerImpl) GetFieldAccessorsByName(ctx context.Context, repoName, className, fieldName string) (*FieldAccessResult, error) {
	query := `
		MATCH (c:Class {name: $className})-[:CONTAINS]->(f:Field {name: $fieldName})
		WHERE c.repo = $repo
		RETURN f.id AS fieldId
		LIMIT 1
	`
	records, err := a.graph.ExecuteRead(ctx, query, map[string]any{
		"className": className,
		"fieldName": fieldName,
		"repo":      repoName,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to find field: %w", err)
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("field not found: %s.%s", className, fieldName)
	}

	fieldID := ast.NodeID(toInt64(records[0]["fieldId"]))
	return a.GetFieldAccessors(ctx, fieldID)
}

// -----------------------------------------------------------------------------
// Inheritance Operations
// -----------------------------------------------------------------------------

func (a *graphAnalyzerImpl) GetInheritanceTree(ctx context.Context, classID ast.NodeID) (*InheritanceTree, error) {
	result := &InheritanceTree{
		Nodes: make(map[ast.NodeID]*InheritanceNode),
	}

	// Get root class
	rootQuery := `
		MATCH (c:Class {id: $classId})
		RETURN c.id AS id, c.name AS name, c.path AS path
	`
	rootRecords, err := a.graph.ExecuteRead(ctx, rootQuery, map[string]any{"classId": int64(classID)})
	if err != nil {
		return nil, fmt.Errorf("failed to get class: %w", err)
	}
	if len(rootRecords) == 0 {
		return nil, fmt.Errorf("class not found: %d", classID)
	}

	rootNode := &InheritanceNode{
		ID:       classID,
		Name:     toString(rootRecords[0]["name"]),
		FilePath: toString(rootRecords[0]["path"]),
		Depth:    0,
		Parents:  make([]*InheritanceNode, 0),
		Children: make([]*InheritanceNode, 0),
	}
	result.Root = rootNode
	result.Nodes[classID] = rootNode

	// Get parent classes (ancestors)
	visited := make(map[ast.NodeID]bool)
	visited[classID] = true
	a.collectParents(ctx, classID, rootNode, 1, result, visited)

	// Get child classes (descendants)
	visited = make(map[ast.NodeID]bool)
	visited[classID] = true
	a.collectChildren(ctx, classID, rootNode, 1, result, visited)

	return result, nil
}

func (a *graphAnalyzerImpl) collectParents(ctx context.Context, classID ast.NodeID, node *InheritanceNode, depth int, result *InheritanceTree, visited map[ast.NodeID]bool) {
	query := `
		MATCH (c:Class {id: $classId})-[:INHERITS]->(parent:Class)
		RETURN parent.id AS id, parent.name AS name, parent.path AS path
	`
	records, err := a.graph.ExecuteRead(ctx, query, map[string]any{"classId": int64(classID)})
	if err != nil {
		a.logger.Warn("Failed to get parent classes", zap.Error(err))
		return
	}

	for _, record := range records {
		parentID := ast.NodeID(toInt64(record["id"]))
		if visited[parentID] {
			continue
		}
		visited[parentID] = true

		parentNode := &InheritanceNode{
			ID:       parentID,
			Name:     toString(record["name"]),
			FilePath: toString(record["path"]),
			Depth:    -depth,
			Parents:  make([]*InheritanceNode, 0),
			Children: []*InheritanceNode{node},
		}
		result.Nodes[parentID] = parentNode
		node.Parents = append(node.Parents, parentNode)

		if depth > result.MaxDepth {
			result.MaxDepth = depth
		}

		// Recurse
		a.collectParents(ctx, parentID, parentNode, depth+1, result, visited)
	}
}

func (a *graphAnalyzerImpl) collectChildren(ctx context.Context, classID ast.NodeID, node *InheritanceNode, depth int, result *InheritanceTree, visited map[ast.NodeID]bool) {
	query := `
		MATCH (child:Class)-[:INHERITS]->(c:Class {id: $classId})
		RETURN child.id AS id, child.name AS name, child.path AS path
	`
	records, err := a.graph.ExecuteRead(ctx, query, map[string]any{"classId": int64(classID)})
	if err != nil {
		a.logger.Warn("Failed to get child classes", zap.Error(err))
		return
	}

	for _, record := range records {
		childID := ast.NodeID(toInt64(record["id"]))
		if visited[childID] {
			continue
		}
		visited[childID] = true

		childNode := &InheritanceNode{
			ID:       childID,
			Name:     toString(record["name"]),
			FilePath: toString(record["path"]),
			Depth:    depth,
			Parents:  []*InheritanceNode{node},
			Children: make([]*InheritanceNode, 0),
		}
		result.Nodes[childID] = childNode
		node.Children = append(node.Children, childNode)

		if depth > result.MaxDepth {
			result.MaxDepth = depth
		}

		// Recurse
		a.collectChildren(ctx, childID, childNode, depth+1, result, visited)
	}
}

func (a *graphAnalyzerImpl) GetParentClasses(ctx context.Context, classID ast.NodeID, maxDepth int) ([]*ClassInfo, error) {
	tree, err := a.GetInheritanceTree(ctx, classID)
	if err != nil {
		return nil, err
	}

	parents := make([]*ClassInfo, 0)
	for _, node := range tree.Nodes {
		if node.Depth < 0 && (maxDepth < 0 || -node.Depth <= maxDepth) {
			parents = append(parents, &ClassInfo{
				ID:       node.ID,
				Name:     node.Name,
				FilePath: node.FilePath,
			})
		}
	}
	return parents, nil
}

func (a *graphAnalyzerImpl) GetChildClasses(ctx context.Context, classID ast.NodeID, maxDepth int) ([]*ClassInfo, error) {
	tree, err := a.GetInheritanceTree(ctx, classID)
	if err != nil {
		return nil, err
	}

	children := make([]*ClassInfo, 0)
	for _, node := range tree.Nodes {
		if node.Depth > 0 && (maxDepth < 0 || node.Depth <= maxDepth) {
			children = append(children, &ClassInfo{
				ID:       node.ID,
				Name:     node.Name,
				FilePath: node.FilePath,
			})
		}
	}
	return children, nil
}

// -----------------------------------------------------------------------------
// Impact Analysis
// -----------------------------------------------------------------------------

func (a *graphAnalyzerImpl) GetImpact(ctx context.Context, nodeID ast.NodeID, opts ImpactOptions) (*ImpactResult, error) {
	if opts.MaxDepth <= 0 {
		opts.MaxDepth = 3
	}

	result := &ImpactResult{
		AffectedNodes:       make([]*ImpactNode, 0),
		AffectedByCallGraph: make([]*ImpactNode, 0),
		AffectedByDataFlow:  make([]*ImpactNode, 0),
	}

	// Get source node info
	sourceNode, err := a.getNodeAsImpactNode(ctx, nodeID, 0, ImpactTypeDirect)
	if err != nil {
		return nil, err
	}
	result.Source = sourceNode

	seen := make(map[ast.NodeID]bool)
	seen[nodeID] = true

	// Collect call graph impact
	if opts.IncludeCallGraph {
		callGraph, err := a.GetCallers(ctx, nodeID, opts.MaxDepth)
		if err == nil && callGraph != nil {
			for id, node := range callGraph.Nodes {
				if seen[id] {
					continue
				}
				seen[id] = true

				impactNode := &ImpactNode{
					ID:       id,
					Name:     node.Name,
					NodeType: ast.NodeTypeFunction,
					FilePath: node.FilePath,
					FileID:   node.FileID,
					Depth:    node.Depth,
					Impact:   ImpactTypeCallGraph,
				}
				result.AffectedByCallGraph = append(result.AffectedByCallGraph, impactNode)
				result.AffectedNodes = append(result.AffectedNodes, impactNode)
			}
		}
	}

	// Collect data flow impact
	if opts.IncludeDataFlow {
		dataGraph, err := a.GetDataDependents(ctx, nodeID, DependencyOptions{
			MaxDepth:        opts.MaxDepth,
			IncludeIndirect: true,
		})
		if err == nil && dataGraph != nil {
			for id, node := range dataGraph.Nodes {
				if seen[id] {
					continue
				}
				seen[id] = true

				impactNode := &ImpactNode{
					ID:       id,
					Name:     node.Name,
					NodeType: node.NodeType,
					FilePath: node.FilePath,
					FileID:   node.FileID,
					Depth:    node.Depth,
					Impact:   ImpactTypeDataFlow,
				}
				result.AffectedByDataFlow = append(result.AffectedByDataFlow, impactNode)
				result.AffectedNodes = append(result.AffectedNodes, impactNode)
			}
		}
	}

	result.TotalAffected = len(result.AffectedNodes)

	return result, nil
}

func (a *graphAnalyzerImpl) GetImpactByName(ctx context.Context, repoName, filePath, name string, nodeType ast.NodeType, opts ImpactOptions) (*ImpactResult, error) {
	// Find the node
	var query string
	params := map[string]any{"name": name, "repo": repoName}

	switch nodeType {
	case ast.NodeTypeFunction:
		query = "MATCH (n:Function {name: $name}) WHERE n.repo = $repo"
	case ast.NodeTypeClass:
		query = "MATCH (n:Class {name: $name}) WHERE n.repo = $repo"
	case ast.NodeTypeField:
		query = "MATCH (n:Field {name: $name}) WHERE n.repo = $repo"
	case ast.NodeTypeVariable:
		query = "MATCH (n:Variable {name: $name}) WHERE n.repo = $repo"
	default:
		return nil, fmt.Errorf("unsupported node type: %d", nodeType)
	}

	if filePath != "" {
		query += " AND n.path = $path"
		params["path"] = filePath
	}

	query += " RETURN n.id AS id LIMIT 1"

	records, err := a.graph.ExecuteRead(ctx, query, params)
	if err != nil {
		return nil, fmt.Errorf("failed to find node: %w", err)
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("node not found: %s", name)
	}

	nodeID := ast.NodeID(toInt64(records[0]["id"]))
	return a.GetImpact(ctx, nodeID, opts)
}

// -----------------------------------------------------------------------------
// Helper Methods
// -----------------------------------------------------------------------------

func (a *graphAnalyzerImpl) getFunctionAsCallNode(ctx context.Context, functionID ast.NodeID, depth int) (*CallNode, error) {
	query := `
		MATCH (f:Function {id: $id})
		RETURN f.name AS name, f.fileId AS fileId, f.range AS range
	`
	records, err := a.graph.ExecuteRead(ctx, query, map[string]any{"id": int64(functionID)})
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("function not found: %d", functionID)
	}

	record := records[0]
	node := &CallNode{
		ID:     functionID,
		Name:   toString(record["name"]),
		FileID: int32(toInt64(record["fileId"])),
		Depth:  depth,
	}
	if rangeStr := toString(record["range"]); rangeStr != "" {
		node.Range = parseRange(rangeStr)
	}

	return node, nil
}

func (a *graphAnalyzerImpl) getNodeAsDependencyNode(ctx context.Context, nodeID ast.NodeID, depth int) (*DependencyNode, error) {
	query := `
		MATCH (n {id: $id})
		RETURN n.name AS name, n.nodeType AS nodeType, n.fileId AS fileId
	`
	records, err := a.graph.ExecuteRead(ctx, query, map[string]any{"id": int64(nodeID)})
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("node not found: %d", nodeID)
	}

	record := records[0]
	return &DependencyNode{
		ID:       nodeID,
		Name:     toString(record["name"]),
		NodeType: ast.NodeType(toInt64(record["nodeType"])),
		FileID:   int32(toInt64(record["fileId"])),
		Depth:    depth,
	}, nil
}

func (a *graphAnalyzerImpl) getNodeAsImpactNode(ctx context.Context, nodeID ast.NodeID, depth int, impactType ImpactType) (*ImpactNode, error) {
	query := `
		MATCH (n {id: $id})
		RETURN n.name AS name, n.nodeType AS nodeType, n.fileId AS fileId, n.path AS path
	`
	records, err := a.graph.ExecuteRead(ctx, query, map[string]any{"id": int64(nodeID)})
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("node not found: %d", nodeID)
	}

	record := records[0]
	return &ImpactNode{
		ID:       nodeID,
		Name:     toString(record["name"]),
		NodeType: ast.NodeType(toInt64(record["nodeType"])),
		FilePath: toString(record["path"]),
		FileID:   int32(toInt64(record["fileId"])),
		Depth:    depth,
		Impact:   impactType,
	}, nil
}

func (a *graphAnalyzerImpl) findFunctionID(ctx context.Context, repoName, filePath, className, functionName string) (ast.NodeID, error) {
	var query string
	params := map[string]any{"name": functionName, "repo": repoName}

	if className != "" {
		// Traverse from FileScope (which has repo) through to Class and then Function
		query = `
			MATCH (fs:FileScope {repo: $repo})-[:CONTAINS*]->(c:Class {name: $className})-[:CONTAINS*]->(f:Function {name: $name})
		`
		params["className"] = className
		if filePath != "" {
			query += " WHERE fs.filePath = $path"
			params["path"] = filePath
		}
	} else {
		// Search for function directly under FileScope
		query = `
			MATCH (fs:FileScope {repo: $repo})-[:CONTAINS*]->(f:Function {name: $name})
		`
		if filePath != "" {
			query += " WHERE fs.filePath = $path"
			params["path"] = filePath
		}
	}

	query += " RETURN f.id AS id LIMIT 1"

	records, err := a.graph.ExecuteRead(ctx, query, params)
	if err != nil {
		return 0, fmt.Errorf("failed to find function: %w", err)
	}
	if len(records) == 0 {
		return 0, fmt.Errorf("function not found: %s", functionName)
	}

	return ast.NodeID(toInt64(records[0]["id"])), nil
}
