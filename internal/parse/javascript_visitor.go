package parse

import (
	"github.com/armchr/codeapi/internal/model/ast"
	"context"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"
	"go.uber.org/zap"
)

type JavaScriptVisitor struct {
	translate *TranslateFromSyntaxTree
	logger    *zap.Logger
}

func NewJavaScriptVisitor(logger *zap.Logger, ts *TranslateFromSyntaxTree) *JavaScriptVisitor {
	return &JavaScriptVisitor{
		translate: ts,
		logger:    logger,
	}
}

func (jsv *JavaScriptVisitor) TraverseNode(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	if tsNode == nil {
		return ast.InvalidNodeID
	}

	switch tsNode.Kind() {
	case "program":
		return jsv.handleProgram(ctx, tsNode)
	case "function_declaration":
		return jsv.handleFunctionDeclaration(ctx, tsNode, scopeID)
	case "arrow_function":
		return jsv.handleArrowFunction(ctx, tsNode, scopeID)
	case "function_expression":
		return jsv.handleFunctionExpression(ctx, tsNode, scopeID)
	case "method_definition":
		return jsv.handleMethodDefinition(ctx, tsNode, scopeID)
	case "class_declaration":
		return jsv.handleClassDeclaration(ctx, tsNode, scopeID)
	case "class_expression":
		return jsv.handleClassExpression(ctx, tsNode, scopeID)
	case "statement_block":
		return jsv.translate.HandleBlock(ctx, tsNode, scopeID)
	case "return_statement":
		return jsv.handleReturnStatement(ctx, tsNode, scopeID)
	case "call_expression":
		return jsv.handleCallExpression(ctx, tsNode, scopeID)
	case "new_expression":
		return jsv.handleNewExpression(ctx, tsNode, scopeID)
	case "member_expression":
		return jsv.handleMemberExpression(ctx, tsNode, scopeID)
	case "subscript_expression":
		return jsv.handleSubscriptExpression(ctx, tsNode, scopeID)
	case "identifier":
		return jsv.translate.HandleIdentifier(ctx, tsNode, scopeID)
	case "if_statement":
		return jsv.handleIfStatement(ctx, tsNode, scopeID)
	case "for_statement":
		return jsv.handleForStatement(ctx, tsNode, scopeID)
	case "for_in_statement":
		return jsv.handleForInStatement(ctx, tsNode, scopeID)
	case "for_of_statement":
		return jsv.handleForOfStatement(ctx, tsNode, scopeID)
	case "while_statement":
		return jsv.handleWhileStatement(ctx, tsNode, scopeID)
	case "do_statement":
		return jsv.handleDoStatement(ctx, tsNode, scopeID)
	case "assignment_expression":
		return jsv.handleAssignmentExpression(ctx, tsNode, scopeID)
	case "variable_declaration":
		return jsv.handleVariableDeclaration(ctx, tsNode, scopeID)
	case "lexical_declaration":
		return jsv.handleLexicalDeclaration(ctx, tsNode, scopeID)
	case "switch_statement":
		return jsv.handleSwitchStatement(ctx, tsNode, scopeID)
	case "try_statement":
		return jsv.handleTryStatement(ctx, tsNode, scopeID)
	case "throw_statement":
		return jsv.handleThrowStatement(ctx, tsNode, scopeID)
	case "expression_statement":
		return jsv.handleExpressionStatement(ctx, tsNode, scopeID)
	case "import_statement":
		return jsv.handleImportStatement(ctx, tsNode, scopeID)
	case "export_statement":
		return jsv.handleExportStatement(ctx, tsNode, scopeID)
	case "await_expression":
		return jsv.handleAwaitExpression(ctx, tsNode, scopeID)
	case "yield_expression":
		return jsv.handleYieldExpression(ctx, tsNode, scopeID)
	case "conditional_expression":
		return jsv.handleConditionalExpression(ctx, tsNode, scopeID)
	case "object_expression":
		return jsv.handleObjectExpression(ctx, tsNode, scopeID)
	case "array_expression":
		return jsv.handleArrayExpression(ctx, tsNode, scopeID)
	case "template_string":
		return jsv.handleTemplateString(ctx, tsNode, scopeID)
	default:
		jsv.translate.TraverseChildren(ctx, tsNode, scopeID)
		return ast.InvalidNodeID
	}
}

func (jsv *JavaScriptVisitor) handleProgram(ctx context.Context, tsNode *tree_sitter.Node) ast.NodeID {
	moduleNode := ast.NewNode(
		jsv.translate.NextNodeID(), ast.NodeTypeModuleScope, jsv.translate.FileID,
		jsv.translate.GetTreeNodeName(tsNode), jsv.translate.ToRange(tsNode), jsv.translate.Version,
		ast.NodeID(jsv.translate.FileID),
	)
	jsv.translate.CodeGraph.CreateModuleScope(ctx, moduleNode)
	jsv.translate.PushScope(false)
	defer jsv.translate.PopScope(ctx, moduleNode.ID)
	childNodes := jsv.translate.TraverseChildren(ctx, tsNode, moduleNode.ID)
	if len(childNodes) > 0 {
		jsv.translate.CreateContainsRelations(ctx, moduleNode.ID, childNodes)
	}
	return moduleNode.ID
}

func (jsv *JavaScriptVisitor) handleFunctionDeclaration(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	paramsNode := jsv.translate.TreeChildByFieldName(tsNode, "parameters")
	bodyNode := jsv.translate.TreeChildByFieldName(tsNode, "body")

	return jsv.translate.CreateFunction(ctx, scopeID, tsNode, "", jsv.translate.NamedChildren(paramsNode), bodyNode)
}

func (jsv *JavaScriptVisitor) handleArrowFunction(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	paramsNode := jsv.translate.TreeChildByFieldName(tsNode, "parameters")
	if paramsNode == nil {
		paramsNode = jsv.translate.TreeChildByFieldName(tsNode, "parameter")
	}
	bodyNode := jsv.translate.TreeChildByFieldName(tsNode, "body")

	return jsv.translate.CreateFunction(ctx, scopeID, tsNode, "", jsv.translate.NamedChildren(paramsNode), bodyNode)
}

func (jsv *JavaScriptVisitor) handleFunctionExpression(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	paramsNode := jsv.translate.TreeChildByFieldName(tsNode, "parameters")
	bodyNode := jsv.translate.TreeChildByFieldName(tsNode, "body")

	return jsv.translate.CreateFunction(ctx, scopeID, tsNode, "", jsv.translate.NamedChildren(paramsNode), bodyNode)
}

func (jsv *JavaScriptVisitor) handleMethodDefinition(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	paramsNode := jsv.translate.TreeChildByFieldName(tsNode, "parameters")
	bodyNode := jsv.translate.TreeChildByFieldName(tsNode, "body")

	return jsv.translate.CreateFunction(ctx, scopeID, tsNode, "", jsv.translate.NamedChildren(paramsNode), bodyNode)
}

func (jsv *JavaScriptVisitor) handleClassDeclaration(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	bodyNode := jsv.translate.TreeChildByFieldName(tsNode, "body")
	var methods []*tree_sitter.Node
	if bodyNode != nil {
		methods = jsv.translate.TreeChildrenByKind(bodyNode, "method_definition")
	}
	return jsv.translate.HandleClass(ctx, scopeID, tsNode, "", methods, nil)
}

func (jsv *JavaScriptVisitor) handleClassExpression(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	bodyNode := jsv.translate.TreeChildByFieldName(tsNode, "body")
	var methods []*tree_sitter.Node
	if bodyNode != nil {
		methods = jsv.translate.TreeChildrenByKind(bodyNode, "method_definition")
	}
	return jsv.translate.HandleClass(ctx, scopeID, tsNode, "", methods, nil)
}

func (jsv *JavaScriptVisitor) handleReturnStatement(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	if tsNode.ChildCount() < 2 {
		return ast.InvalidNodeID
	}
	rhsNode := tsNode.Child(1)
	rhs := jsv.translate.HandleReturn(ctx, rhsNode, scopeID)
	return rhs
}

func (jsv *JavaScriptVisitor) handleCallExpression(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	functionNode := jsv.translate.TreeChildByFieldName(tsNode, "function")
	argumentsNode := jsv.translate.TreeChildByFieldName(tsNode, "arguments")

	var args []*tree_sitter.Node
	if argumentsNode != nil {
		args = jsv.translate.NamedChildren(argumentsNode)
	}

	fnNameNodeID := jsv.translate.HandleRhsWithFakeVariable(ctx, "__fn__", functionNode, scopeID, nil)
	return jsv.translate.HandleCall(ctx, fnNameNodeID, args, scopeID, jsv.translate.ToRange(tsNode))
}

func (jsv *JavaScriptVisitor) handleNewExpression(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	constructorNode := jsv.translate.TreeChildByFieldName(tsNode, "constructor")
	argumentsNode := jsv.translate.TreeChildByFieldName(tsNode, "arguments")

	var args []*tree_sitter.Node
	if argumentsNode != nil {
		args = jsv.translate.NamedChildren(argumentsNode)
	}

	fnNameNodeID := jsv.translate.HandleRhsWithFakeVariable(ctx, "__constructor__", constructorNode, scopeID, nil)
	return jsv.translate.HandleCall(ctx, fnNameNodeID, args, scopeID, jsv.translate.ToRange(tsNode))
}

func (jsv *JavaScriptVisitor) handleMemberExpression(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	objectNode := jsv.translate.TreeChildByFieldName(tsNode, "object")
	propertyNode := jsv.translate.TreeChildByFieldName(tsNode, "property")

	var names []*tree_sitter.Node
	if objectNode != nil {
		names = append(names, objectNode)
	}
	if propertyNode != nil {
		names = append(names, propertyNode)
	}

	resolvedNodeId := jsv.translate.ResolveNameChain(ctx, names, scopeID)
	if jsv.translate.CurrentScope.IsRhs() && resolvedNodeId != ast.InvalidNodeID {
		jsv.translate.CurrentScope.AddRhsVar(resolvedNodeId)
	}
	return resolvedNodeId
}

func (jsv *JavaScriptVisitor) handleSubscriptExpression(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	objectNode := jsv.translate.TreeChildByFieldName(tsNode, "object")
	indexNode := jsv.translate.TreeChildByFieldName(tsNode, "index")

	var names []*tree_sitter.Node
	if objectNode != nil {
		names = append(names, objectNode)
	}
	if indexNode != nil {
		names = append(names, indexNode)
	}

	resolvedNodeId := jsv.translate.ResolveNameChain(ctx, names, scopeID)
	if jsv.translate.CurrentScope.IsRhs() && resolvedNodeId != ast.InvalidNodeID {
		jsv.translate.CurrentScope.AddRhsVar(resolvedNodeId)
	}
	return resolvedNodeId
}

func (jsv *JavaScriptVisitor) handleIfStatement(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	conditionNode := jsv.translate.TreeChildByFieldName(tsNode, "condition")
	consequenceNode := jsv.translate.TreeChildByFieldName(tsNode, "consequence")
	alternativeNode := jsv.translate.TreeChildByFieldName(tsNode, "alternative")

	conditions := []*tree_sitter.Node{conditionNode}
	branches := []*tree_sitter.Node{consequenceNode}

	if alternativeNode != nil {
		if alternativeNode.Kind() == "if_statement" {
			altCondition := jsv.translate.TreeChildByFieldName(alternativeNode, "condition")
			altConsequence := jsv.translate.TreeChildByFieldName(alternativeNode, "consequence")
			conditions = append(conditions, altCondition)
			branches = append(branches, altConsequence)

			altAlternative := jsv.translate.TreeChildByFieldName(alternativeNode, "alternative")
			if altAlternative != nil {
				branches = append(branches, altAlternative)
			}
		} else {
			branches = append(branches, alternativeNode)
		}
	}

	return jsv.translate.HandleConditional(ctx, tsNode, conditions, branches, scopeID)
}

func (jsv *JavaScriptVisitor) handleForStatement(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	initNode := jsv.translate.TreeChildByFieldName(tsNode, "initializer")
	conditionNode := jsv.translate.TreeChildByFieldName(tsNode, "condition")
	updateNode := jsv.translate.TreeChildByFieldName(tsNode, "increment")
	bodyNode := jsv.translate.TreeChildByFieldName(tsNode, "body")

	var inits []*tree_sitter.Node
	if initNode != nil {
		inits = append(inits, initNode)
	}
	if conditionNode != nil {
		inits = append(inits, conditionNode)
	}

	jsv.translate.PushScope(false)
	defer jsv.translate.PopScope(ctx, ast.InvalidNodeID)

	initCondID := ast.InvalidNodeID
	if len(inits) > 0 {
		initCondID = jsv.translate.HandleRhsExprsWithFakeVariable(ctx, "__init__", inits, scopeID, nil)
	}

	updateID := ast.InvalidNodeID
	if updateNode != nil {
		updateID = jsv.translate.HandleRhsWithFakeVariable(ctx, "__update__", updateNode, scopeID, nil)
	}

	if bodyNode == nil {
		return ast.InvalidNodeID
	}
	return jsv.translate.HandleLoop(ctx, tsNode, updateID, initCondID, bodyNode, scopeID)
}

func (jsv *JavaScriptVisitor) handleForInStatement(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	leftNode := jsv.translate.TreeChildByFieldName(tsNode, "left")
	rightNode := jsv.translate.TreeChildByFieldName(tsNode, "right")
	bodyNode := jsv.translate.TreeChildByFieldName(tsNode, "body")

	var inits []*tree_sitter.Node
	if leftNode != nil {
		inits = append(inits, leftNode)
	}
	if rightNode != nil {
		inits = append(inits, rightNode)
	}

	jsv.translate.PushScope(false)
	defer jsv.translate.PopScope(ctx, ast.InvalidNodeID)

	initCondID := ast.InvalidNodeID
	if len(inits) > 0 {
		initCondID = jsv.translate.HandleRhsExprsWithFakeVariable(ctx, "__for_in__", inits, scopeID, nil)
	}

	if bodyNode == nil {
		return ast.InvalidNodeID
	}
	return jsv.translate.HandleLoop(ctx, tsNode, ast.InvalidNodeID, initCondID, bodyNode, scopeID)
}

func (jsv *JavaScriptVisitor) handleForOfStatement(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	leftNode := jsv.translate.TreeChildByFieldName(tsNode, "left")
	rightNode := jsv.translate.TreeChildByFieldName(tsNode, "right")
	bodyNode := jsv.translate.TreeChildByFieldName(tsNode, "body")

	var inits []*tree_sitter.Node
	if leftNode != nil {
		inits = append(inits, leftNode)
	}
	if rightNode != nil {
		inits = append(inits, rightNode)
	}

	jsv.translate.PushScope(false)
	defer jsv.translate.PopScope(ctx, ast.InvalidNodeID)

	initCondID := ast.InvalidNodeID
	if len(inits) > 0 {
		initCondID = jsv.translate.HandleRhsExprsWithFakeVariable(ctx, "__for_of__", inits, scopeID, nil)
	}

	if bodyNode == nil {
		return ast.InvalidNodeID
	}
	return jsv.translate.HandleLoop(ctx, tsNode, ast.InvalidNodeID, initCondID, bodyNode, scopeID)
}

func (jsv *JavaScriptVisitor) handleWhileStatement(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	conditionNode := jsv.translate.TreeChildByFieldName(tsNode, "condition")
	bodyNode := jsv.translate.TreeChildByFieldName(tsNode, "body")

	if conditionNode == nil || bodyNode == nil {
		return ast.InvalidNodeID
	}

	conditionID := jsv.translate.HandleRhsWithFakeVariable(ctx, "__cond__", conditionNode, scopeID, nil)
	return jsv.translate.HandleLoop(ctx, tsNode, ast.InvalidNodeID, conditionID, bodyNode, scopeID)
}

func (jsv *JavaScriptVisitor) handleDoStatement(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	bodyNode := jsv.translate.TreeChildByFieldName(tsNode, "body")
	conditionNode := jsv.translate.TreeChildByFieldName(tsNode, "condition")

	if conditionNode == nil || bodyNode == nil {
		return ast.InvalidNodeID
	}

	conditionID := jsv.translate.HandleRhsWithFakeVariable(ctx, "__cond__", conditionNode, scopeID, nil)
	return jsv.translate.HandleLoop(ctx, tsNode, ast.InvalidNodeID, conditionID, bodyNode, scopeID)
}

func (jsv *JavaScriptVisitor) handleAssignmentExpression(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	leftNode := jsv.translate.TreeChildByFieldName(tsNode, "left")
	rightNode := jsv.translate.TreeChildByFieldName(tsNode, "right")

	if leftNode == nil || rightNode == nil {
		return ast.InvalidNodeID
	}

	return jsv.translate.HandleAssignment(ctx, tsNode, leftNode, rightNode, scopeID)
}

func (jsv *JavaScriptVisitor) handleVariableDeclaration(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	declarators := jsv.translate.TreeChildrenByKind(tsNode, "variable_declarator")
	for _, declarator := range declarators {
		nameNode := jsv.translate.TreeChildByFieldName(declarator, "name")
		valueNode := jsv.translate.TreeChildByFieldName(declarator, "value")
		if nameNode != nil && valueNode != nil {
			jsv.translate.HandleAssignment(ctx, declarator, nameNode, valueNode, scopeID)
		}
	}
	return ast.InvalidNodeID
}

func (jsv *JavaScriptVisitor) handleLexicalDeclaration(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	declarators := jsv.translate.TreeChildrenByKind(tsNode, "variable_declarator")
	for _, declarator := range declarators {
		nameNode := jsv.translate.TreeChildByFieldName(declarator, "name")
		valueNode := jsv.translate.TreeChildByFieldName(declarator, "value")
		if nameNode != nil && valueNode != nil {
			jsv.translate.HandleAssignment(ctx, declarator, nameNode, valueNode, scopeID)
		}
	}
	return ast.InvalidNodeID
}

func (jsv *JavaScriptVisitor) handleSwitchStatement(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	discriminantNode := jsv.translate.TreeChildByFieldName(tsNode, "value")
	bodyNode := jsv.translate.TreeChildByFieldName(tsNode, "body")

	var conditions []*tree_sitter.Node
	var branches []*tree_sitter.Node

	if discriminantNode != nil {
		conditions = append(conditions, discriminantNode)
	}

	if bodyNode != nil {
		caseClauses := jsv.translate.TreeChildrenByKind(bodyNode, "switch_case")
		defaultClauses := jsv.translate.TreeChildrenByKind(bodyNode, "switch_default")

		for _, clause := range caseClauses {
			valueNode := jsv.translate.TreeChildByFieldName(clause, "value")
			if valueNode != nil {
				conditions = append(conditions, valueNode)
			}
			consequence := jsv.translate.TreeChildByFieldName(clause, "body")
			if consequence != nil {
				branches = append(branches, consequence)
			}
		}

		for _, clause := range defaultClauses {
			consequence := jsv.translate.TreeChildByFieldName(clause, "body")
			if consequence != nil {
				branches = append(branches, consequence)
			}
		}
	}

	return jsv.translate.HandleConditional(ctx, tsNode, conditions, branches, scopeID)
}

func (jsv *JavaScriptVisitor) handleTryStatement(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	bodyNode := jsv.translate.TreeChildByFieldName(tsNode, "body")
	handlerNode := jsv.translate.TreeChildByFieldName(tsNode, "handler")
	finalizerNode := jsv.translate.TreeChildByFieldName(tsNode, "finalizer")

	var conditions []*tree_sitter.Node
	var branches []*tree_sitter.Node

	if bodyNode != nil {
		branches = append(branches, bodyNode)
	}

	if handlerNode != nil {
		paramNode := jsv.translate.TreeChildByFieldName(handlerNode, "parameter")
		if paramNode != nil {
			conditions = append(conditions, paramNode)
		}
		handlerBodyNode := jsv.translate.TreeChildByFieldName(handlerNode, "body")
		if handlerBodyNode != nil {
			branches = append(branches, handlerBodyNode)
		}
	}

	if finalizerNode != nil {
		branches = append(branches, finalizerNode)
	}

	return jsv.translate.HandleConditional(ctx, tsNode, conditions, branches, scopeID)
}

func (jsv *JavaScriptVisitor) handleThrowStatement(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	if tsNode.ChildCount() < 2 {
		return ast.InvalidNodeID
	}
	rhsNode := tsNode.Child(1)
	rhs := jsv.translate.HandleReturn(ctx, rhsNode, scopeID)
	return rhs
}

func (jsv *JavaScriptVisitor) handleExpressionStatement(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	if tsNode.ChildCount() < 1 {
		return ast.InvalidNodeID
	}
	exprNode := tsNode.Child(0)
	return jsv.TraverseNode(ctx, exprNode, scopeID)
}

func (jsv *JavaScriptVisitor) handleImportStatement(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	clauseNode := jsv.translate.TreeChildByFieldName(tsNode, "import")
	sourceNode := jsv.translate.TreeChildByFieldName(tsNode, "source")

	if clauseNode != nil && sourceNode != nil {
		return jsv.translate.HandleAssignment(ctx, tsNode, clauseNode, sourceNode, scopeID)
	}
	return ast.InvalidNodeID
}

func (jsv *JavaScriptVisitor) handleExportStatement(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	declarationNode := jsv.translate.TreeChildByFieldName(tsNode, "declaration")
	if declarationNode != nil {
		return jsv.TraverseNode(ctx, declarationNode, scopeID)
	}
	return ast.InvalidNodeID
}

func (jsv *JavaScriptVisitor) handleAwaitExpression(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	argumentNode := jsv.translate.TreeChildByFieldName(tsNode, "argument")
	if argumentNode != nil {
		return jsv.TraverseNode(ctx, argumentNode, scopeID)
	}
	return ast.InvalidNodeID
}

func (jsv *JavaScriptVisitor) handleYieldExpression(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	argumentNode := jsv.translate.TreeChildByFieldName(tsNode, "argument")
	if argumentNode != nil {
		return jsv.TraverseNode(ctx, argumentNode, scopeID)
	}
	return ast.InvalidNodeID
}

func (jsv *JavaScriptVisitor) handleConditionalExpression(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	conditionNode := jsv.translate.TreeChildByFieldName(tsNode, "condition")
	consequenceNode := jsv.translate.TreeChildByFieldName(tsNode, "consequence")
	alternativeNode := jsv.translate.TreeChildByFieldName(tsNode, "alternative")

	conditions := []*tree_sitter.Node{conditionNode}
	branches := []*tree_sitter.Node{consequenceNode}

	if alternativeNode != nil {
		branches = append(branches, alternativeNode)
	}

	return jsv.translate.HandleConditional(ctx, tsNode, conditions, branches, scopeID)
}

func (jsv *JavaScriptVisitor) handleObjectExpression(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	properties := jsv.translate.NamedChildren(tsNode)
	for _, property := range properties {
		jsv.TraverseNode(ctx, property, scopeID)
	}
	return ast.InvalidNodeID
}

func (jsv *JavaScriptVisitor) handleArrayExpression(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	elements := jsv.translate.NamedChildren(tsNode)
	for _, element := range elements {
		jsv.TraverseNode(ctx, element, scopeID)
	}
	return ast.InvalidNodeID
}

func (jsv *JavaScriptVisitor) handleTemplateString(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	substitutions := jsv.translate.TreeChildrenByKind(tsNode, "template_substitution")
	for _, substitution := range substitutions {
		jsv.TraverseNode(ctx, substitution, scopeID)
	}
	return ast.InvalidNodeID
}
