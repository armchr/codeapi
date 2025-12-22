package parse

import (
	"github.com/armchr/codeapi/internal/model/ast"
	"github.com/armchr/codeapi/internal/service/codegraph"
	"github.com/armchr/codeapi/pkg/lsp/base"
	"context"
	"fmt"
	"maps"
	"strings"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"
	"go.uber.org/zap"
)

type Symbol struct {
	Node   *ast.Node
	Fields map[string]*Symbol
}

func NewSymbol(node *ast.Node) *Symbol {
	return &Symbol{
		Node:   node,
		Fields: make(map[string]*Symbol),
	}
}

func (s *Symbol) GetField(fieldName string) *Symbol {
	if f, ok := s.Fields[fieldName]; ok {
		return f
	}
	return nil
}

func (s *Symbol) AddField(field *Symbol) error {
	if _, exists := s.Fields[field.Node.Name]; exists {
		return fmt.Errorf("field already exists: %s", field.Node.Name)
	}
	s.Fields[field.Node.Name] = field
	return nil
}

type Scope struct {
	symbols           map[string]*Symbol
	Parent            *Scope
	rhsVars           map[ast.NodeID]bool
	notContainedNodes map[ast.NodeID]bool
}

func (s *Scope) AddSymbol(sym *Symbol) error {
	if _, exists := s.symbols[sym.Node.Name]; exists {
		return fmt.Errorf("symbol already exists: %s", sym.Node.Name)
	}
	s.symbols[sym.Node.Name] = sym
	return nil
}

func (s *Scope) GetSymbol(name string) *Symbol {
	if sym, ok := s.symbols[name]; ok {
		return sym
	}
	return nil
}

func (s *Scope) GetAllNotContainedNodes() []ast.NodeID {
	var notContained []ast.NodeID
	for id := range s.notContainedNodes {
		notContained = append(notContained, ast.NodeID(id))
	}
	return notContained
}

func (s *Scope) AddNotContainedNode(nodeID ast.NodeID) {
	s.notContainedNodes[nodeID] = true
}

func (s *Scope) IsNotContainedNode(nodeID ast.NodeID) bool {
	return s.notContainedNodes[nodeID]
}

func (s *Scope) RemoveNotContainedNode(nodeID ast.NodeID) {
	delete(s.notContainedNodes, nodeID)
}

func (s *Scope) IsRhs() bool {
	return s.rhsVars != nil
}

func (s *Scope) AddRhsVar(nodeID ast.NodeID) {
	if s.rhsVars == nil {
		return
	}
	s.rhsVars[nodeID] = true
}

func (s *Scope) GetRhsVars() []ast.NodeID {
	if s.rhsVars == nil {
		return nil
	}
	var vars []ast.NodeID
	for id := range s.rhsVars {
		vars = append(vars, ast.NodeID(id))
	}
	return vars
}

func (s *Scope) Resolve(name string) *Symbol {
	if sym := s.GetSymbol(name); sym != nil {
		return sym
	}
	if s.Parent != nil {
		return s.Parent.Resolve(name)
	}
	return nil
}

func NewScope(parent *Scope, isRhs bool) *Scope {
	var rhsVars map[ast.NodeID]bool = nil
	if isRhs {
		rhsVars = make(map[ast.NodeID]bool)
	}
	return &Scope{
		symbols:           make(map[string]*Symbol),
		Parent:            parent,
		rhsVars:           rhsVars,
		notContainedNodes: make(map[ast.NodeID]bool),
	}
}

type SyntaxTreeVisitor interface {
	TraverseNode(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID
	HasSpecialName(kind string) bool
	GetName(tsNode *tree_sitter.Node) string
}

type TranslateFromSyntaxTree struct {
	ScopeStack   []*Scope
	CurrentScope *Scope
	FileID       int32
	Version      int32
	NodeIDSeq    uint32
	CodeGraph    *codegraph.CodeGraph
	FileContent  []byte
	Visitor      SyntaxTreeVisitor
	Logger       *zap.Logger
	Nodes        map[ast.NodeID]*ast.Node
	// Batch writing support
	EnableBatchWrites bool
	BatchSize         int
	nodeBuffer        []*ast.Node
	relationBuffer    []codegraph.RelationSpec
}

func NewTranslateFromSyntaxTree(fileID int32, version int32, codeGraph *codegraph.CodeGraph,
	fileContent []byte,
	logger *zap.Logger) *TranslateFromSyntaxTree {
	globalScope := NewScope(nil, false)
	return &TranslateFromSyntaxTree{
		ScopeStack:   []*Scope{globalScope},
		CurrentScope: globalScope,
		FileID:       fileID,
		Version:      version,
		NodeIDSeq:    1,
		CodeGraph:    codeGraph,
		FileContent:  fileContent,
		Logger:       logger,
		Nodes:        make(map[ast.NodeID]*ast.Node),
	}
}

func (t *TranslateFromSyntaxTree) NewNode(nodeType ast.NodeType, name string, rng base.Range, parentID ast.NodeID) *ast.Node {
	node := ast.NewNode(t.NextNodeID(), nodeType, t.FileID, name, rng, t.Version, parentID)
	t.Nodes[node.ID] = node
	t.CurrentScope.AddNotContainedNode(node.ID)
	return node
}

func (t *TranslateFromSyntaxTree) PushScope(rhs bool) {
	newScope := NewScope(t.CurrentScope, rhs)
	t.ScopeStack = append(t.ScopeStack, newScope)
	t.CurrentScope = newScope
}

func (t *TranslateFromSyntaxTree) PopScope(ctx context.Context, closingScopeId ast.NodeID) {
	if len(t.ScopeStack) == 0 {
		t.Logger.Error("Scope stack underflow")
		return
	}

	curScope := t.CurrentScope
	if curScope == nil {
		t.Logger.Error("Current scope is nil")
		return
	}
	parentScope := curScope.Parent

	if closingScopeId == ast.InvalidNodeID {
		// move all not contained nodes to parent scope
		for childID := range curScope.notContainedNodes {
			if parentScope != nil {
				parentScope.AddNotContainedNode(childID)
			}
		}
	} else {
		// create contains relations for all not contained nodes
		for childID := range curScope.notContainedNodes {
			t.CreateContainsRelation(ctx, closingScopeId, childID, t.FileID)
		}
	}

	t.ScopeStack = t.ScopeStack[:len(t.ScopeStack)-1]
	t.CurrentScope = parentScope
}

func (t *TranslateFromSyntaxTree) NextNodeID() ast.NodeID {
	id := t.NodeIDSeq
	t.NodeIDSeq++
	newId := ast.NodeID(t.FileID)
	newId = (newId << 32) | ast.NodeID(id)
	return newId
}

func (t *TranslateFromSyntaxTree) TreeChildByKind(node *tree_sitter.Node, kind string) *tree_sitter.Node {
	for i := uint(0); i < uint(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Kind() == kind {
			return child
		}
	}
	return nil
}

func (t *TranslateFromSyntaxTree) TreeChildrenByKind(node *tree_sitter.Node, kind string) []*tree_sitter.Node {
	var children []*tree_sitter.Node
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child.Kind() == kind {
			children = append(children, child)
		}
	}
	return children
}
func (t *TranslateFromSyntaxTree) TreeChildByFieldName(node *tree_sitter.Node, fieldName string) *tree_sitter.Node {
	for i := uint(0); i < node.ChildCount(); i++ {
		if node.FieldNameForChild(uint32(i)) == fieldName {
			return node.Child(i)
		}
	}
	return nil
}

func (t *TranslateFromSyntaxTree) SubtreeNodeByKind(node *tree_sitter.Node, kind string) *tree_sitter.Node {
	if node == nil {
		return nil
	}
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		childKind := child.Kind()
		if childKind == kind {
			return child
		}
		result := t.SubtreeNodeByKind(child, kind)
		if result != nil {
			return result
		}
	}
	return nil
}

func (t *TranslateFromSyntaxTree) String(node *tree_sitter.Node) string {
	if node == nil {
		return ""
	}
	return t.getNodeText(node, []byte(t.FileContent))
}

func (t *TranslateFromSyntaxTree) getNodeText(node *tree_sitter.Node, sourceCode []byte) string {
	return string(sourceCode[node.StartByte():node.EndByte()])
}

func (t *TranslateFromSyntaxTree) GetAstNodeText(node *ast.Node) string {
	if node == nil {
		return ""
	}
	return t.getTextFromSourceAndRange(t.FileContent, node.Range)
}

func (t *TranslateFromSyntaxTree) getTextFromSourceAndRange(sourceCode []byte, rng base.Range) string {
	// iterate over the source code to get the text in the range
	// for every /n increment line count
	var currentLine, currentChar int
	var startByte, endByte int
	for i := 0; i < len(sourceCode); i++ {
		if currentLine == rng.Start.Line && currentChar == rng.Start.Character {
			startByte = i
		}
		if currentLine == rng.End.Line && currentChar == rng.End.Character {
			endByte = i
			break
		}
		if sourceCode[i] == '\n' {
			currentLine++
			currentChar = 0
		} else {
			currentChar++
		}
	}
	return string(sourceCode[startByte:endByte])
}

func (t *TranslateFromSyntaxTree) Chindren(node *tree_sitter.Node) []*tree_sitter.Node {
	var children []*tree_sitter.Node
	for i := uint(0); i < node.ChildCount(); i++ {
		children = append(children, node.Child(i))
	}
	return children
}

func (t *TranslateFromSyntaxTree) NamedChildren(node *tree_sitter.Node) []*tree_sitter.Node {
	var children []*tree_sitter.Node
	for i := uint(0); i < node.NamedChildCount(); i++ {
		children = append(children, node.NamedChild(i))
	}
	return children
}

func (t *TranslateFromSyntaxTree) ToRange(node *tree_sitter.Node) base.Range {
	if node == nil {
		return base.Range{}
	}
	startPos := node.StartPosition()
	endPos := node.EndPosition()
	return base.Range{
		Start: base.Position{
			Line:      int(startPos.Row),
			Character: int(startPos.Column),
		},
		End: base.Position{
			Line:      int(endPos.Row),
			Character: int(endPos.Column),
		},
	}
}

func (t *TranslateFromSyntaxTree) GetTreeNodeName(node *tree_sitter.Node) string {
	kind := node.Kind()
	if t.Visitor.HasSpecialName(kind) {
		return t.Visitor.GetName(node)
	}
	if kind == "scoped_identifier" ||
		kind == "identifier" ||
		kind == "property_identifier" ||
		kind == "type_identifier" ||
		kind == "shorthand_property_identifier_pattern" ||
		kind == "field_identifier" ||
		kind == "type_spec" ||
		strings.HasSuffix(kind, "_identifier") {
		return t.String(node)
	}

	idNode := t.TreeChildByKind(node, "scoped_identifier")
	if idNode == nil {
		idNode = t.TreeChildByKind(node, "identifier")
	}

	if idNode == nil {
		idNode = t.TreeChildByKind(node, "property_identifier")
	}

	/*
		if idNode == nil {
			idNode = t.TreeChildByKind(node, "type_identifier")
		}
	*/

	if kind == "method_elem" && idNode == nil {
		idNode = t.TreeChildByKind(node, "field_identifier")
	}

	if idNode == nil {
		return ""
	}

	return t.String(idNode)
}

func (t *TranslateFromSyntaxTree) TraverseChildren(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) []ast.NodeID {
	if tsNode == nil {
		return nil
	}
	var childIDs []ast.NodeID
	for i := uint(0); i < tsNode.ChildCount(); i++ {
		child := tsNode.Child(i)
		childID := t.Visitor.TraverseNode(ctx, child, scopeID)
		if childID != ast.InvalidNodeID {
			childIDs = append(childIDs, childID)
		}
	}
	return childIDs
}

func (t *TranslateFromSyntaxTree) CreateContainsRelation(ctx context.Context, parentID ast.NodeID, childID ast.NodeID, fileID int32) {
	err := t.CodeGraph.CreateContainsRelation(ctx, parentID, childID, fileID)
	if err != nil {
		t.Logger.Error("Failed to create contains relation", zap.Int64("parentID", int64(parentID)), zap.Int64("childID", int64(childID)), zap.Error(err))
	}
	t.CurrentScope.RemoveNotContainedNode(childID)
}

func (t *TranslateFromSyntaxTree) CreateContainsRelations(ctx context.Context, parentID ast.NodeID, childIDs []ast.NodeID) {
	for _, childID := range childIDs {
		if childID == ast.InvalidNodeID {
			continue
		}
		t.CreateContainsRelation(ctx, parentID, childID, t.FileID)
	}
}

func (t *TranslateFromSyntaxTree) CreateFunction(ctx context.Context,
	scopeID ast.NodeID,
	fn *tree_sitter.Node,
	fnName string,
	params []*tree_sitter.Node, body *tree_sitter.Node) ast.NodeID {
	funcName := fnName
	if funcName == "" {
		funcName = t.GetTreeNodeName(fn)
	}
	if funcName == "" {
		return ast.InvalidNodeID
	}

	funcNode := t.NewNode(
		ast.NodeTypeFunction, funcName, t.ToRange(fn), scopeID,
	)
	t.CodeGraph.CreateFunction(ctx, funcNode)

	t.PushScope(false)
	defer t.PopScope(ctx, funcNode.ID)

	// Handle parameters
	for idx, param := range params {
		paramNodeID := t.HandleVariable(ctx, param, funcNode.ID)
		t.CreateContainsRelation(ctx, funcNode.ID, paramNodeID, t.FileID)
		t.CodeGraph.CreateFunctionArgRelation(ctx, funcNode.ID, paramNodeID, idx, t.FileID)
	}

	if body != nil {
		bodyNodeID := t.Visitor.TraverseNode(ctx, body, funcNode.ID)
		if bodyNodeID != ast.InvalidNodeID {
			t.CreateContainsRelation(ctx, funcNode.ID, bodyNodeID, t.FileID)
			t.CodeGraph.CreateBodyRelation(ctx, funcNode.ID, bodyNodeID, t.FileID)
		}
	}

	return funcNode.ID
}

func (t *TranslateFromSyntaxTree) HandleBlock(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	blockNode := t.NewNode(
		ast.NodeTypeBlock, "", t.ToRange(tsNode), scopeID,
	)
	t.CodeGraph.CreateBlock(ctx, blockNode)
	t.PushScope(false)
	defer t.PopScope(ctx, blockNode.ID)

	childNodes := t.TraverseChildren(ctx, tsNode, blockNode.ID)
	if len(childNodes) > 0 {
		t.CreateContainsRelations(ctx, blockNode.ID, childNodes)
	}
	return blockNode.ID
}

func (t *TranslateFromSyntaxTree) CreateFakeVariable(ctx context.Context, scopeID ast.NodeID, prefix string, rng base.Range, additionalMetadata map[string]any) ast.NodeID {
	varName := fmt.Sprintf("%s_%d", prefix, t.NextNodeID())
	varNode := t.NewNode(
		ast.NodeTypeVariable, varName, rng, scopeID,
	)

	varNode.MetaData = map[string]any{
		"fake": true,
	}

	if additionalMetadata != nil {
		maps.Copy(varNode.MetaData, additionalMetadata)
	}
	t.CodeGraph.CreateVariable(ctx, varNode)
	return varNode.ID
}

func (t *TranslateFromSyntaxTree) HandleVariable(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	varName := t.GetTreeNodeName(tsNode)
	if varName == "" {
		return ast.InvalidNodeID
	}

	varNode := t.NewNode(
		ast.NodeTypeVariable, varName, t.ToRange(tsNode), scopeID,
	)

	typeId := t.TreeChildByFieldName(tsNode, "type_identifier")
	if typeId != nil {
		typeName := t.GetTreeNodeName(typeId)
		varNode.MetaData = map[string]any{
			"type": typeName,
		}
	}

	t.CodeGraph.CreateVariable(ctx, varNode)
	t.CurrentScope.AddSymbol(NewSymbol(varNode))

	return varNode.ID
}

func (t *TranslateFromSyntaxTree) ResolveNameChain(ctx context.Context, nameChain []*tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	var sym *Symbol = nil
	for _, nameNode := range nameChain {
		varName := t.GetTreeNodeName(nameNode)

		if varName == "" {
			//t.GetTreeNodeName(nameNode) // just for debugging
			if sym != nil {
				//t.Logger.Error("Empty name not the first on in list of names", zap.String("prev", sym.Node.Name))
				debugName := PrintSyntaxTree(ctx, nameNode, t.FileContent)
				t.Logger.Info("Node with empty name", zap.String("node", debugName))
			}
			fakeVarID := t.HandleRhsWithFakeVariable(ctx, "__name__", nameNode, scopeID, nil)
			varNode := t.Nodes[fakeVarID]
			sym = NewSymbol(varNode)
		} else {
			if sym == nil {
				sym = t.CurrentScope.Resolve(varName)
			} else {
				newSym := sym.GetField(varName)

				if newSym == nil {
					varNode := t.NewNode(
						ast.NodeTypeField, varName, t.ToRange(nameNode), scopeID,
					)
					t.CodeGraph.CreateField(ctx, varNode)
					newSym = NewSymbol(varNode)
				}
				sym.AddField(newSym)

				if newSym.Node.ID != ast.InvalidNodeID {
					t.CodeGraph.CreateHasFieldRelation(ctx, sym.Node.ID, newSym.Node.ID, t.FileID)
				}
				// must be at the end of this block
				sym = newSym
			}
		}
	}

	if sym == nil {
		return ast.InvalidNodeID
	}
	return sym.Node.ID
}

func (t *TranslateFromSyntaxTree) HandleClass(ctx context.Context,
	scopeID ast.NodeID,
	cls *tree_sitter.Node,
	name string,
	methods []*tree_sitter.Node,
	fields []*tree_sitter.Node) ast.NodeID {
	className := name
	if className == "" {
		className = t.GetTreeNodeName(cls)
	}
	if className == "" {
		return ast.InvalidNodeID
	}

	classNode := t.NewNode(
		ast.NodeTypeClass, className, t.ToRange(cls), scopeID,
	)
	t.CodeGraph.CreateClass(ctx, classNode)

	t.PushScope(false)
	defer t.PopScope(ctx, classNode.ID)

	for _, field := range fields {
		fieldNodeID := t.HandleVariable(ctx, field, classNode.ID)
		if fieldNodeID != ast.InvalidNodeID {
			t.CreateContainsRelation(ctx, classNode.ID, fieldNodeID, t.FileID)
			t.CodeGraph.CreateHasFieldRelation(ctx, classNode.ID, fieldNodeID, t.FileID)
		}
	}

	for _, method := range methods {
		methodNodeID := t.Visitor.TraverseNode(ctx, method, classNode.ID)
		if methodNodeID != ast.InvalidNodeID {
			t.CreateContainsRelation(ctx, classNode.ID, methodNodeID, t.FileID)
			t.CodeGraph.CreateHasFieldRelation(ctx, classNode.ID, methodNodeID, t.FileID)
		}
	}

	return classNode.ID
}

func (t *TranslateFromSyntaxTree) HandleRhsWithFakeVariable(ctx context.Context, lhsPrefix string, rhs *tree_sitter.Node, scopeID ast.NodeID, additionalMetadata map[string]any) ast.NodeID {
	rhsVarIds, rhsNodeId := t.HandleRhs(ctx, rhs, scopeID)

	// if the rhs is a single variable, return it directly
	if len(rhsVarIds) == 1 && rhsVarIds[0] == rhsNodeId && rhsNodeId != ast.InvalidNodeID {
		return rhsNodeId
	}

	retVarID := t.CreateFakeVariable(ctx, scopeID, lhsPrefix, t.ToRange(rhs), additionalMetadata)

	for _, rhsVarID := range rhsVarIds {
		t.CodeGraph.CreateDataFlowRelation(ctx, rhsVarID, retVarID, t.FileID)
	}
	return retVarID
}

func (t *TranslateFromSyntaxTree) HandleRhsExprsWithFakeVariable(ctx context.Context, lhsPrefix string, rhsExprs []*tree_sitter.Node, scopeID ast.NodeID, additionalMetadata map[string]any) ast.NodeID {
	if len(rhsExprs) == 0 {
		return ast.InvalidNodeID
	}
	if len(rhsExprs) == 1 {
		return t.HandleRhsWithFakeVariable(ctx, lhsPrefix, rhsExprs[0], scopeID, nil)
	}

	var allRhsVarIds []ast.NodeID
	for _, rhs := range rhsExprs {
		rhsVarIds, _ := t.HandleRhs(ctx, rhs, scopeID)
		allRhsVarIds = append(allRhsVarIds, rhsVarIds...)
	}
	if len(allRhsVarIds) == 0 {
		return ast.InvalidNodeID
	}
	retVarID := t.CreateFakeVariable(ctx, scopeID, lhsPrefix, t.ToRange(rhsExprs[0]), additionalMetadata)
	for _, rhsVarID := range allRhsVarIds {
		t.CodeGraph.CreateDataFlowRelation(ctx, rhsVarID, retVarID, t.FileID)
	}
	return retVarID
}

func (t *TranslateFromSyntaxTree) HandleReturn(ctx context.Context, rhs *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	if rhs == nil {
		return ast.InvalidNodeID
	}

	return t.HandleRhsWithFakeVariable(ctx, "__ret_value__", rhs, scopeID, map[string]any{"return": true})
}

func (t *TranslateFromSyntaxTree) HandleRhs(ctx context.Context, rhs *tree_sitter.Node, scopeID ast.NodeID) ([]ast.NodeID, ast.NodeID) {
	// returns a list of "variables" that are referenced in the rhs
	if rhs == nil {
		return nil, ast.InvalidNodeID
	}
	t.PushScope(true)
	defer t.PopScope(ctx, ast.InvalidNodeID)

	nodeID := t.Visitor.TraverseNode(ctx, rhs, scopeID)
	return t.CurrentScope.GetRhsVars(), nodeID
}

func (t *TranslateFromSyntaxTree) HandleCall(ctx context.Context, nameID ast.NodeID, args []*tree_sitter.Node, scopeID ast.NodeID, rng base.Range) ast.NodeID {
	if nameID == ast.InvalidNodeID {
		return ast.InvalidNodeID
	}

	fnNameNode := t.Nodes[nameID]
	if fnNameNode == nil {
		return ast.InvalidNodeID
	}

	fnName := fnNameNode.Name
	if fnNameNode.MetaData["fake"] == true {
		fnName = t.GetAstNodeText(fnNameNode)
	}

	callNode := t.NewNode(
		ast.NodeTypeFunctionCall, fnName, rng, scopeID,
	)
	callNode.MetaData = map[string]any{
		"nameID": fnNameNode.ID,
	}
	t.CodeGraph.CreateFunctionCall(ctx, callNode)

	for idx, arg := range args {
		argNodeID := t.HandleRhsWithFakeVariable(ctx, fmt.Sprintf("__arg_%d__", idx), arg, scopeID, nil)
		t.CodeGraph.CreateFunctionCallArgRelation(ctx, callNode.ID, argNodeID, idx, t.FileID)
	}

	t.CurrentScope.AddRhsVar(callNode.ID)

	return callNode.ID
}

func (t *TranslateFromSyntaxTree) HandleIdentifier(ctx context.Context, idNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	if idNode == nil {
		return ast.InvalidNodeID
	}

	name := t.GetTreeNodeName(idNode)
	if name == "" {
		return ast.InvalidNodeID
	}

	varId := ast.InvalidNodeID
	// Check if the identifier is already in the current scope
	if sym := t.CurrentScope.Resolve(name); sym != nil {
		varId = sym.Node.ID
	} else {
		varNode := t.NewNode(
			ast.NodeTypeVariable, name, t.ToRange(idNode), scopeID,
		)
		t.CodeGraph.CreateVariable(ctx, varNode)
		t.CurrentScope.AddSymbol(NewSymbol(varNode))
		varId = varNode.ID
	}

	if t.CurrentScope.IsRhs() && varId != ast.InvalidNodeID {
		t.CurrentScope.AddRhsVar(varId)
	}

	return varId
}

func (t *TranslateFromSyntaxTree) HandleConditional(ctx context.Context, conditionalNode *tree_sitter.Node, conditions []*tree_sitter.Node, branches []*tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	condNode := t.NewNode(
		ast.NodeTypeConditional, "", t.ToRange(conditions[0]), scopeID,
	)
	t.CodeGraph.CreateConditional(ctx, condNode)

	var conditionIDs []ast.NodeID
	for _, cond := range conditions {
		conditionID := t.HandleRhsWithFakeVariable(ctx, "__cond__", cond, condNode.ID, nil)
		if conditionID == ast.InvalidNodeID {
			return ast.InvalidNodeID
		}
		conditionIDs = append(conditionIDs, conditionID)
	}
	t.CreateContainsRelations(ctx, condNode.ID, conditionIDs)

	var branchNodeIDs []ast.NodeID
	for _, branch := range branches {
		branchNodeID := t.Visitor.TraverseNode(ctx, branch, condNode.ID)
		if branchNodeID == ast.InvalidNodeID {
			return ast.InvalidNodeID
		}
		branchNodeIDs = append(branchNodeIDs, branchNodeID)
	}
	t.CreateContainsRelations(ctx, condNode.ID, branchNodeIDs)

	for idx, branchNodeID := range branchNodeIDs {
		conditionId := ast.InvalidNodeID
		if idx < len(conditionIDs) {
			conditionId = conditionIDs[idx]
		}
		t.CodeGraph.CreateConditionalRelation(ctx, condNode.ID, branchNodeID, idx, conditionId, t.FileID)
	}
	return condNode.ID
}

func (t *TranslateFromSyntaxTree) HandleLoop(ctx context.Context, loopNode *tree_sitter.Node,
	initID ast.NodeID, conditionID ast.NodeID, body *tree_sitter.Node,
	scopeID ast.NodeID) ast.NodeID {
	node := t.NewNode(ast.NodeTypeLoop, "", t.ToRange(loopNode), scopeID)
	bodyID := t.Visitor.TraverseNode(ctx, body, node.ID)

	node.MetaData = map[string]any{
		"condition": conditionID,
		//"body":      bodyID,
	}
	if initID != ast.InvalidNodeID {
		node.MetaData["init"] = initID
	}
	t.CodeGraph.CreateLoop(ctx, node)

	t.CreateContainsRelations(ctx, node.ID, []ast.NodeID{conditionID, bodyID})
	t.CodeGraph.CreateBodyRelation(ctx, node.ID, bodyID, t.FileID)

	if initID != ast.InvalidNodeID {
		t.CreateContainsRelation(ctx, node.ID, initID, t.FileID)
	}
	return node.ID
}

func (t *TranslateFromSyntaxTree) HandleAssignment(ctx context.Context, assignNode *tree_sitter.Node, lhs *tree_sitter.Node, rhs *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	if lhs == nil || rhs == nil {
		return ast.InvalidNodeID
	}

	lhsID := t.Visitor.TraverseNode(ctx, lhs, scopeID)
	rhsID := t.HandleRhsWithFakeVariable(ctx, "__rhs__", rhs, scopeID, nil)

	if lhsID == ast.InvalidNodeID || rhsID == ast.InvalidNodeID {
		return ast.InvalidNodeID
	}

	t.CodeGraph.CreateDataFlowRelation(ctx, rhsID, lhsID, t.FileID)
	return lhsID
}
