package parse

import (
	"github.com/armchr/codeapi/internal/model/ast"
	"context"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"
	"go.uber.org/zap"
)

type PythonVisitor struct {
	translate *TranslateFromSyntaxTree
	logger    *zap.Logger
}

func NewPythonVisitor(logger *zap.Logger, ts *TranslateFromSyntaxTree) *PythonVisitor {
	return &PythonVisitor{
		translate: ts,
		logger:    logger,
	}
}

func (pv *PythonVisitor) TraverseNode(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	if tsNode == nil {
		return ast.InvalidNodeID
	}

	switch tsNode.Kind() {
	case "module":
		return pv.handleModule(ctx, tsNode)
	case "function_definition":
		return pv.handleFunctionDefinition(ctx, tsNode, scopeID)
	case "block":
		return pv.translate.HandleBlock(ctx, tsNode, scopeID)
	case "class_definition":
		return pv.handleClassDefinition(ctx, tsNode, scopeID)
	case "return_statement":
		return pv.handleReturnStatement(ctx, tsNode, scopeID)
	case "call":
		return pv.handleCall(ctx, tsNode, scopeID)
	case "attribute":
		return pv.handleAttribute(ctx, tsNode, scopeID)
	case "identifier":
		return pv.translate.HandleIdentifier(ctx, tsNode, scopeID)
	case "if_statement":
		return pv.handleIfStatement(ctx, tsNode, scopeID)
	case "for_statement":
		return pv.handleForStatement(ctx, tsNode, scopeID)
	case "while_statement":
		return pv.handleWhileStatement(ctx, tsNode, scopeID)
	case "assignment":
		return pv.handleAssignment(ctx, tsNode, scopeID)
	/*

		case "expression_statement":
			return pv.handleExpressionStatement(ctx, tsNode, scopeID)
		case "if_statement":
			return pv.handleIfStatement(ctx, tsNode, scopeID)



	*/
	// Add more cases as needed for other node types
	default:
		// For unhandled node types, we can choose to log or ignore
		// fmt.Printf("Unhandled node type: %s\n", tsNode.Type())
		pv.translate.TraverseChildren(ctx, tsNode, scopeID)
		return ast.InvalidNodeID
	}
}

func (pv *PythonVisitor) handleModule(ctx context.Context, tsNode *tree_sitter.Node) ast.NodeID {
	// Handle module-level constructs if needed
	moduleNode := ast.NewNode(
		pv.translate.NextNodeID(), ast.NodeTypeModuleScope, pv.translate.FileID,
		pv.translate.GetTreeNodeName(tsNode), pv.translate.ToRange(tsNode), pv.translate.Version,
		ast.NodeID(pv.translate.FileID),
	)
	pv.translate.CodeGraph.CreateModuleScope(ctx, moduleNode)
	pv.translate.PushScope(false)
	defer pv.translate.PopScope(ctx, moduleNode.ID)
	childNodes := pv.translate.TraverseChildren(ctx, tsNode, moduleNode.ID)
	if len(childNodes) > 0 {
		pv.translate.CreateContainsRelations(ctx, moduleNode.ID, childNodes)
	}
	return moduleNode.ID
}

func (pv *PythonVisitor) handleFunctionDefinition(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	paramsNode := pv.translate.TreeChildByFieldName(tsNode, "parameters")
	bodyNode := pv.translate.TreeChildByFieldName(tsNode, "body")

	return pv.translate.CreateFunction(ctx, scopeID, tsNode, "", pv.translate.NamedChildren(paramsNode), bodyNode)
}

func (pv *PythonVisitor) handleClassDefinition(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	body := pv.translate.TreeChildByFieldName(tsNode, "body")
	var methods []*tree_sitter.Node
	if body != nil {
		methods = pv.translate.TreeChildrenByKind(body, "function_definition")
	}
	return pv.translate.HandleClass(ctx, scopeID, tsNode, "", methods, nil)
}

func (pv *PythonVisitor) handleReturnStatement(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	if tsNode.ChildCount() < 2 {
		return ast.InvalidNodeID
	}
	rhsNode := tsNode.Child(1)
	rhs := pv.translate.HandleReturn(ctx, rhsNode, scopeID)
	return rhs
}

func (pv *PythonVisitor) handleCall(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	argList := pv.translate.TreeChildByKind(tsNode, "argument_list")
	var args []*tree_sitter.Node
	if argList != nil {
		args = pv.translate.NamedChildren(argList)
	}
	fnNameNodeID := pv.translate.HandleRhsWithFakeVariable(ctx, "__fn__", tsNode.Child(0), scopeID, nil)
	return pv.translate.HandleCall(ctx, fnNameNodeID, args, scopeID, pv.translate.ToRange(tsNode))
}

func (pv *PythonVisitor) handleAttribute(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	var names []*tree_sitter.Node

	for i := uint(0); i < tsNode.ChildCount(); i++ {
		child := tsNode.Child(i)
		if child.Kind() == "." {
			continue
		}
		names = append(names, child)
	}
	resolvedNodeId := pv.translate.ResolveNameChain(ctx, names, scopeID)
	if pv.translate.CurrentScope.IsRhs() && resolvedNodeId != ast.InvalidNodeID {
		pv.translate.CurrentScope.AddRhsVar(resolvedNodeId)
	}
	return resolvedNodeId
}

func (pv *PythonVisitor) handleIfStatement(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	conditionNode := tsNode.Child(1) // if is 0, condition is 1
	branch := tsNode.Child(3)        // body is 3

	conditions := []*tree_sitter.Node{conditionNode}
	branches := []*tree_sitter.Node{branch}

	elifNodes := pv.translate.TreeChildrenByKind(tsNode, "elif_clause")
	for _, elif := range elifNodes {
		cond := elif.Child(1) // elif is 0, condition is 1
		br := elif.Child(3)   // body is 3
		conditions = append(conditions, cond)
		branches = append(branches, br)
	}

	elseNode := pv.translate.TreeChildByKind(tsNode, "else_clause")
	if elseNode != nil {
		branches = append(branches, elseNode)
	}
	return pv.translate.HandleConditional(ctx, tsNode, conditions, branches, scopeID)
}

func (pv *PythonVisitor) handleForStatement(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	inits := make([]*tree_sitter.Node, 0)
	initVars := tsNode.Child(1) // target is 1
	if initVars != nil {
		inits = append(inits, initVars)
	}
	initCond := tsNode.Child(3)
	if initCond == nil {
		return ast.InvalidNodeID
	}
	inits = append(inits, initCond)

	pv.translate.PushScope(false)
	defer pv.translate.PopScope(ctx, ast.InvalidNodeID)

	initCondID := pv.translate.HandleRhsExprsWithFakeVariable(ctx, "__init__", inits, scopeID, nil)
	body := tsNode.Child(5) // body is 5
	if body == nil {
		return ast.InvalidNodeID
	}
	return pv.translate.HandleLoop(ctx, tsNode, ast.InvalidNodeID, initCondID, body, scopeID)
}

func (pv *PythonVisitor) handleWhileStatement(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	conditionNode := tsNode.Child(1) // while is 0, condition is 1
	if conditionNode == nil {
		return ast.InvalidNodeID
	}
	conditionID := pv.translate.HandleRhsWithFakeVariable(ctx, "__cond__", conditionNode, scopeID, nil)
	body := tsNode.Child(3) // body is 3
	if body == nil {
		return ast.InvalidNodeID
	}
	return pv.translate.HandleLoop(ctx, tsNode, ast.InvalidNodeID, conditionID, body, scopeID)
}

func (pv *PythonVisitor) handleAssignment(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	if tsNode.ChildCount() < 3 {
		return ast.InvalidNodeID
	}
	lhsNode := tsNode.Child(0)
	rhsNode := tsNode.Child(2)

	return pv.translate.HandleAssignment(ctx, tsNode, lhsNode, rhsNode, scopeID)
}

// HasSpecialName returns false for Python - no special naming conventions
func (pv *PythonVisitor) HasSpecialName(kind string) bool {
	return false
}

// GetName is not implemented for Python visitor
func (pv *PythonVisitor) GetName(tsNode *tree_sitter.Node) string {
	pv.logger.Error("GetName not implemented for Python visitor")
	return ""
}
