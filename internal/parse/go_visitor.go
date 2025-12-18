package parse

import (
	"github.com/armchr/codeapi/internal/model/ast"
	"github.com/armchr/codeapi/pkg/lsp/base"
	"context"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"
	"go.uber.org/zap"
)

type GoVisitor struct {
	translate *TranslateFromSyntaxTree
	logger    *zap.Logger
}

func NewGoVisitor(logger *zap.Logger, ts *TranslateFromSyntaxTree) *GoVisitor {
	return &GoVisitor{
		translate: ts,
		logger:    logger,
	}
}

func (gv *GoVisitor) TraverseNode(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	if tsNode == nil {
		return ast.InvalidNodeID
	}

	switch tsNode.Kind() {
	case "source_file":
		return gv.handleSourceFile(ctx, tsNode)
	case "function_declaration":
		return gv.handleFunctionDeclaration(ctx, tsNode, scopeID)
	case "method_declaration":
		return gv.handleMethodDeclaration(ctx, tsNode, scopeID)
	case "block":
		return gv.translate.HandleBlock(ctx, tsNode, scopeID)
	case "type_declaration":
		return gv.handleTypeDeclaration(ctx, tsNode, scopeID)
	case "method_elem":
		return gv.handleMethodElem(ctx, tsNode, scopeID)
	//case "struct_type":
	//	return gv.handleStructType(ctx, tsNode, scopeID)
	//case "interface_type":
	//	return gv.handleInterfaceType(ctx, tsNode, scopeID)
	case "return_statement":
		return gv.handleReturnStatement(ctx, tsNode, scopeID)
	case "call_expression":
		return gv.handleCallExpression(ctx, tsNode, scopeID)
	case "selector_expression":
		return gv.handleSelectorExpression(ctx, tsNode, scopeID)
	case "identifier":
		return gv.translate.HandleIdentifier(ctx, tsNode, scopeID)
	case "if_statement":
		return gv.handleIfStatement(ctx, tsNode, scopeID)
	case "for_statement":
		return gv.handleForStatement(ctx, tsNode, scopeID)
	case "range_clause":
		return gv.handleRangeClause(ctx, tsNode, scopeID)
	case "assignment_statement":
		return gv.handleAssignmentStatement(ctx, tsNode, scopeID)
	case "short_var_declaration":
		return gv.handleShortVarDeclaration(ctx, tsNode, scopeID)
	case "var_declaration":
		return gv.handleVarDeclaration(ctx, tsNode, scopeID)
	case "const_declaration":
		return gv.handleConstDeclaration(ctx, tsNode, scopeID)
	case "switch_statement":
		return gv.handleSwitchStatement(ctx, tsNode, scopeID)
	case "expression_switch_statement":
		return gv.handleSwitchStatement(ctx, tsNode, scopeID)
	case "type_switch_statement":
		return gv.handleTypeSwitchStatement(ctx, tsNode, scopeID)
	case "go_statement":
		return gv.handleGoStatement(ctx, tsNode, scopeID)
	case "defer_statement":
		return gv.handleDeferStatement(ctx, tsNode, scopeID)
	case "select_statement":
		return gv.handleSelectStatement(ctx, tsNode, scopeID)
	case "import_declaration":
		return gv.handleImportDeclaration(ctx, tsNode, scopeID)
	default:
		gv.translate.TraverseChildren(ctx, tsNode, scopeID)
		return ast.InvalidNodeID
	}
}

func (gv *GoVisitor) handlePackage(ctx context.Context, tsNode *tree_sitter.Node) ast.NodeID {
	nameNode := gv.translate.TreeChildByKind(tsNode, "package_identifier")
	moduleNode := ast.NewNode(
		gv.translate.NextNodeID(), ast.NodeTypeModuleScope, gv.translate.FileID,
		gv.translate.GetTreeNodeName(nameNode), gv.translate.ToRange(tsNode), gv.translate.Version,
		ast.NodeID(gv.translate.FileID),
	)
	gv.translate.CodeGraph.CreateModuleScope(ctx, moduleNode)
	return moduleNode.ID
}

func (gv *GoVisitor) handleSourceFile(ctx context.Context, tsNode *tree_sitter.Node) ast.NodeID {
	packageClause := gv.translate.TreeChildByKind(tsNode, "package_clause")
	moduleNodeID := gv.handlePackage(ctx, packageClause)
	gv.translate.PushScope(false)
	defer gv.translate.PopScope(ctx, moduleNodeID)
	childNodes := gv.translate.TraverseChildren(ctx, tsNode, moduleNodeID)
	if len(childNodes) > 0 {
		gv.translate.CreateContainsRelations(ctx, moduleNodeID, childNodes)
	}
	return moduleNodeID
}

func (gv *GoVisitor) handleFunctionDeclaration(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	funcName := ""
	nameNode := gv.translate.TreeChildByKind(tsNode, "identifier")
	if nameNode != nil {
		funcName = gv.translate.GetTreeNodeName(nameNode)
	}
	paramsNode := gv.translate.TreeChildByFieldName(tsNode, "parameters")
	bodyNode := gv.translate.TreeChildByFieldName(tsNode, "body")

	return gv.translate.CreateFunction(ctx, scopeID, tsNode, funcName, gv.translate.NamedChildren(paramsNode), bodyNode)
}

func (gv *GoVisitor) createFakeClass(ctx context.Context, className string, fileID int32, scopeID ast.NodeID) *ast.Node {
	classNode := ast.NewNode(
		gv.translate.NextNodeID(), ast.NodeTypeClass, fileID,
		className, base.Range{}, gv.translate.Version,
		scopeID,
	)
	classNode.MetaData = map[string]any{
		"is_fake": true,
	}
	gv.translate.CodeGraph.CreateClass(ctx, classNode)
	return classNode
}

func (gv *GoVisitor) handleMethodDeclaration(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	nameNode := gv.translate.TreeChildByKind(tsNode, "field_identifier")
	methodName := ""
	if nameNode != nil {
		methodName = gv.translate.GetTreeNodeName(nameNode)
	}

	paramLists := gv.translate.TreeChildrenByKind(tsNode, "parameter_list")

	if len(paramLists) < 2 {
		gv.logger.Warn("method_declaration missing parameter_list",
			zap.Int("child_count", len(paramLists)),
			zap.String("node_text", gv.translate.GetTreeNodeName(nameNode)),
		)
		return ast.InvalidNodeID
	}

	receiverNode := paramLists[0]
	paramsNode := paramLists[1]

	classNameNode := gv.translate.SubtreeNodeByKind(receiverNode, "type_identifier")
	if classNameNode == nil {
		gv.logger.Error("classNameNode is nil")
		return ast.InvalidNodeID
	}

	className := gv.translate.GetTreeNodeName(classNameNode)
	classNodes, err := gv.translate.CodeGraph.FindNodesByNameAndTypeInFile(ctx, className, ast.NodeTypeClass, gv.translate.FileID)
	if err != nil {
		gv.logger.Error("Error in find class for method",
			zap.String("class_name", className),
			zap.Int32("file_id", gv.translate.FileID),
			zap.Error(err))
		return ast.InvalidNodeID
	}

	classNode := &ast.Node{}
	if len(classNodes) > 0 {
		classNode = classNodes[0]
	} else {
		classNode = gv.createFakeClass(ctx, className, gv.translate.FileID, scopeID)
	}

	//receiverNode := gv.translate.TreeChildByFieldName(tsNode, "receiver")
	//paramsNode := gv.translate.TreeChildByFieldName(tsNode, "parameters")
	bodyNode := gv.translate.TreeChildByFieldName(tsNode, "block")

	var allParams []*tree_sitter.Node
	if paramsNode != nil {
		allParams = append(allParams,
			gv.translate.TreeChildrenByKind(paramsNode, "parameter_declaration")...)
	}

	functionId := gv.translate.CreateFunction(ctx, classNode.ID, tsNode, methodName, allParams, bodyNode)

	// TODO: bad design. ideally this function should return the functionId. But that will end up adding functionID
	// as a CONTAINS in the module.
	if functionId != ast.InvalidNodeID {
		gv.translate.CreateContainsRelation(ctx, classNode.ID, functionId, gv.translate.FileID)

		thisParamDecl := gv.translate.TreeChildByKind(receiverNode, "parameter_declaration")
		if thisParamDecl != nil {
			thisNode := gv.translate.TreeChildByKind(thisParamDecl, "identifier")
			if thisNode != nil {
				// push scope to create "this" in function scope instead of higher scope
				gv.translate.PushScope(false)
				thisNodeId := gv.TraverseNode(ctx, thisNode, functionId)
				if thisNodeId != ast.InvalidNodeID {
					gv.translate.CodeGraph.MarkThis(ctx, gv.translate.FileID, thisNodeId, classNode.ID)
				}
				gv.translate.PopScope(ctx, functionId)
			}
		}
	}
	return ast.InvalidNodeID
}

func (gv *GoVisitor) handleMethodElem(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	methodName := ""
	nameNode := gv.translate.TreeChildByKind(tsNode, "field_identifier")
	if nameNode != nil {
		methodName = gv.translate.GetTreeNodeName(nameNode)
	}
	paramList := gv.translate.TreeChildByKind(tsNode, "parameter_list")
	params := []*tree_sitter.Node{}
	if paramList != nil {
		params = gv.translate.TreeChildrenByKind(paramList, "parameter_declaration")
	}

	return gv.translate.CreateFunction(ctx, scopeID, tsNode, methodName, params, nil)
}

func (gv *GoVisitor) handleTypeDeclaration(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	typeSpecs := gv.translate.TreeChildrenByKind(tsNode, "type_spec")
	var childNodes []ast.NodeID
	for _, typeSpec := range typeSpecs {
		structType := gv.translate.TreeChildByKind(typeSpec, "struct_type")
		if structType != nil {
			childID := gv.handleStructTypeSpec(ctx, typeSpec, scopeID)
			if childID != ast.InvalidNodeID {
				childNodes = append(childNodes, childID)
			}
			continue
		}
		interfaceType := gv.translate.TreeChildByKind(typeSpec, "interface_type")
		if interfaceType != nil {
			childID := gv.handleInterfaceType(ctx, typeSpec, scopeID)

			/*
				structuTypeId := gv.TraverseNode(ctx, typeSpec, scopeID)
				if childID != ast.InvalidNodeID {
					childNodes = append(childNodes, childID)
				}
			*/
			if childID != ast.InvalidNodeID {
				childNodes = append(childNodes, childID)
			}
			continue
		}
	}
	return ast.InvalidNodeID
}

func (gv *GoVisitor) handleStructTypeSpec(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	structType := gv.translate.TreeChildByKind(tsNode, "struct_type")
	fieldDeclList := gv.translate.TreeChildByKind(structType, "field_declaration_list")
	if fieldDeclList == nil {
		return ast.InvalidNodeID
	}
	fieldDecls := gv.translate.TreeChildrenByKind(fieldDeclList, "field_declaration")

	typeId := gv.translate.TreeChildByKind(tsNode, "type_identifier")

	clsName := ""
	if typeId != nil {
		clsName = gv.translate.GetTreeNodeName(typeId)
	}

	/*
		var fields []*tree_sitter.Node
		if fieldDecls != nil {
			field := gv.translate.TreeChildByKind(fieldDecls[0], "field_identifier")
			fields = append(fields, field)
		}
	*/
	return gv.translate.HandleClass(ctx, scopeID, tsNode, clsName, nil, fieldDecls)
}

func (gv *GoVisitor) handleInterfaceType(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	interfaceType := gv.translate.TreeChildByKind(tsNode, "interface_type")
	methods := gv.translate.TreeChildrenByKind(interfaceType, "method_elem")
	typeId := gv.translate.TreeChildByKind(tsNode, "type_identifier")
	clsName := ""
	if typeId != nil {
		clsName = gv.translate.GetTreeNodeName(typeId)
	}

	return gv.translate.HandleClass(ctx, scopeID, tsNode, clsName, methods, nil)
}

func (gv *GoVisitor) handleReturnStatement(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	if tsNode.ChildCount() < 2 {
		return ast.InvalidNodeID
	}
	rhsNode := tsNode.Child(1)
	rhs := gv.translate.HandleReturn(ctx, rhsNode, scopeID)
	return rhs
}

func (gv *GoVisitor) handleCallExpression(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	functionNode := gv.translate.TreeChildByFieldName(tsNode, "function")
	argumentsNode := gv.translate.TreeChildByFieldName(tsNode, "arguments")

	var args []*tree_sitter.Node
	if argumentsNode != nil {
		args = gv.translate.NamedChildren(argumentsNode)
	}

	fnNameNodeID := gv.translate.HandleRhsWithFakeVariable(ctx, "__fn__", functionNode, scopeID, nil)
	return gv.translate.HandleCall(ctx, fnNameNodeID, args, scopeID, gv.translate.ToRange(tsNode))
}

func (gv *GoVisitor) handleSelectorExpression(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	debugName := PrintSyntaxTree(ctx, tsNode, gv.translate.FileContent)
	gv.logger.Debug("Handling selector expression", zap.String("node_text", debugName))
	operandNode := gv.translate.TreeChildByFieldName(tsNode, "operand")
	fieldNode := gv.translate.TreeChildByFieldName(tsNode, "field")

	var names []*tree_sitter.Node
	if operandNode != nil {
		names = append(names, operandNode)
	}
	if fieldNode != nil {
		names = append(names, fieldNode)
	}

	resolvedNodeId := gv.translate.ResolveNameChain(ctx, names, scopeID)
	if gv.translate.CurrentScope.IsRhs() && resolvedNodeId != ast.InvalidNodeID {
		gv.translate.CurrentScope.AddRhsVar(resolvedNodeId)
	}
	return resolvedNodeId
}

func (gv *GoVisitor) handleIfStatement(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	conditionNode := gv.translate.TreeChildByFieldName(tsNode, "condition")
	consequenceNode := gv.translate.TreeChildByFieldName(tsNode, "consequence")
	alternativeNode := gv.translate.TreeChildByFieldName(tsNode, "alternative")

	conditions := []*tree_sitter.Node{conditionNode}
	branches := []*tree_sitter.Node{consequenceNode}

	if alternativeNode != nil {
		if alternativeNode.Kind() == "if_statement" {
			altCondition := gv.translate.TreeChildByFieldName(alternativeNode, "condition")
			altConsequence := gv.translate.TreeChildByFieldName(alternativeNode, "consequence")
			conditions = append(conditions, altCondition)
			branches = append(branches, altConsequence)

			altAlternative := gv.translate.TreeChildByFieldName(alternativeNode, "alternative")
			if altAlternative != nil {
				branches = append(branches, altAlternative)
			}
		} else {
			branches = append(branches, alternativeNode)
		}
	}

	return gv.translate.HandleConditional(ctx, tsNode, conditions, branches, scopeID)
}

func (gv *GoVisitor) handleForStatement(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	initNode := gv.translate.TreeChildByFieldName(tsNode, "initializer")
	conditionNode := gv.translate.TreeChildByFieldName(tsNode, "condition")
	updateNode := gv.translate.TreeChildByFieldName(tsNode, "update")
	bodyNode := gv.translate.TreeChildByFieldName(tsNode, "body")

	var inits []*tree_sitter.Node
	if initNode != nil {
		inits = append(inits, initNode)
	}
	if conditionNode != nil {
		inits = append(inits, conditionNode)
	}

	gv.translate.PushScope(false)
	defer gv.translate.PopScope(ctx, ast.InvalidNodeID)

	initCondID := ast.InvalidNodeID
	if len(inits) > 0 {
		initCondID = gv.translate.HandleRhsExprsWithFakeVariable(ctx, "__init__", inits, scopeID, nil)
	}

	updateID := ast.InvalidNodeID
	if updateNode != nil {
		updateID = gv.translate.HandleRhsWithFakeVariable(ctx, "__update__", updateNode, scopeID, nil)
	}

	if bodyNode == nil {
		return ast.InvalidNodeID
	}
	return gv.translate.HandleLoop(ctx, tsNode, updateID, initCondID, bodyNode, scopeID)
}

func (gv *GoVisitor) handleRangeClause(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	leftNode := gv.translate.TreeChildByFieldName(tsNode, "left")
	rightNode := gv.translate.TreeChildByFieldName(tsNode, "right")

	var inits []*tree_sitter.Node
	if leftNode != nil {
		inits = append(inits, leftNode)
	}
	if rightNode != nil {
		inits = append(inits, rightNode)
	}

	if len(inits) > 0 {
		return gv.translate.HandleRhsExprsWithFakeVariable(ctx, "__range__", inits, scopeID, nil)
	}
	return ast.InvalidNodeID
}

func (gv *GoVisitor) handleAssignmentStatement(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	leftNode := gv.translate.TreeChildByFieldName(tsNode, "left")
	rightNode := gv.translate.TreeChildByFieldName(tsNode, "right")

	if leftNode == nil || rightNode == nil {
		return ast.InvalidNodeID
	}

	return gv.translate.HandleAssignment(ctx, tsNode, leftNode, rightNode, scopeID)
}

func (gv *GoVisitor) handleShortVarDeclaration(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	leftNode := gv.translate.TreeChildByFieldName(tsNode, "left")
	rightNode := gv.translate.TreeChildByFieldName(tsNode, "right")

	if leftNode == nil || rightNode == nil {
		return ast.InvalidNodeID
	}

	return gv.translate.HandleAssignment(ctx, tsNode, leftNode, rightNode, scopeID)
}

func (gv *GoVisitor) handleVarDeclaration(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	specs := gv.translate.TreeChildrenByKind(tsNode, "var_spec")
	for _, spec := range specs {
		nameNode := gv.translate.TreeChildByFieldName(spec, "name")
		valueNode := gv.translate.TreeChildByFieldName(spec, "value")
		if nameNode != nil && valueNode != nil {
			gv.translate.HandleAssignment(ctx, spec, nameNode, valueNode, scopeID)
		}
	}
	return ast.InvalidNodeID
}

func (gv *GoVisitor) handleConstDeclaration(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	specs := gv.translate.TreeChildrenByKind(tsNode, "const_spec")
	for _, spec := range specs {
		nameNode := gv.translate.TreeChildByFieldName(spec, "name")
		valueNode := gv.translate.TreeChildByFieldName(spec, "value")
		if nameNode != nil && valueNode != nil {
			gv.translate.HandleAssignment(ctx, spec, nameNode, valueNode, scopeID)
		}
	}
	return ast.InvalidNodeID
}

func (gv *GoVisitor) handleSwitchStatement(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	valueNode := gv.translate.TreeChildByFieldName(tsNode, "value")
	bodyNode := gv.translate.TreeChildByFieldName(tsNode, "body")

	var conditions []*tree_sitter.Node
	var branches []*tree_sitter.Node

	if valueNode != nil {
		conditions = append(conditions, valueNode)
	}

	if bodyNode != nil {
		caseClauses := gv.translate.TreeChildrenByKind(bodyNode, "expression_case")
		defaultClauses := gv.translate.TreeChildrenByKind(bodyNode, "default_case")

		for _, clause := range caseClauses {
			valueList := gv.translate.TreeChildByFieldName(clause, "value")
			if valueList != nil {
				conditions = append(conditions, valueList)
			}
			consequence := gv.translate.TreeChildByFieldName(clause, "consequence")
			if consequence != nil {
				branches = append(branches, consequence)
			}
		}

		for _, clause := range defaultClauses {
			consequence := gv.translate.TreeChildByFieldName(clause, "consequence")
			if consequence != nil {
				branches = append(branches, consequence)
			}
		}
	}

	return gv.translate.HandleConditional(ctx, tsNode, conditions, branches, scopeID)
}

/*
func (gv *GoVisitor) handleExpressionSwitchStatement(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	var conditions []*tree_sitter.Node
	var branches []*tree_sitter.Node

	coreCondition := gv.translate.TreeChildByKind(tsNode, "selector_expression")

	caseClauses := gv.translate.TreeChildrenByKind(tsNode, "expression_case")
	defaultClauses := gv.translate.TreeChildrenByKind(tsNode, "default_case")

	for _, clause := range caseClauses {
		valueList := gv.translate.TreeChildByFieldName(clause, "value")
		if valueList != nil {
			conditions = append(conditions, valueList)
		}
		consequence := gv.translate.TreeChildByFieldName(clause, "consequence")
		if consequence != nil {
			branches = append(branches, consequence)
		}
	}

	for _, clause := range defaultClauses {
		consequence := gv.translate.TreeChildByFieldName(clause, "consequence")
		if consequence != nil {
			branches = append(branches, consequence)
		}
	}

	return gv.translate.HandleConditional(ctx, tsNode, conditions, branches, scopeID)
}
*/

func (gv *GoVisitor) handleTypeSwitchStatement(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	aliasNode := gv.translate.TreeChildByFieldName(tsNode, "alias")
	valueNode := gv.translate.TreeChildByFieldName(tsNode, "value")
	bodyNode := gv.translate.TreeChildByFieldName(tsNode, "body")

	var conditions []*tree_sitter.Node
	var branches []*tree_sitter.Node

	if aliasNode != nil {
		conditions = append(conditions, aliasNode)
	}
	if valueNode != nil {
		conditions = append(conditions, valueNode)
	}

	if bodyNode != nil {
		caseClauses := gv.translate.TreeChildrenByKind(bodyNode, "type_case")
		defaultClauses := gv.translate.TreeChildrenByKind(bodyNode, "default_case")

		for _, clause := range caseClauses {
			typeList := gv.translate.TreeChildByFieldName(clause, "type")
			if typeList != nil {
				conditions = append(conditions, typeList)
			}
			consequence := gv.translate.TreeChildByFieldName(clause, "consequence")
			if consequence != nil {
				branches = append(branches, consequence)
			}
		}

		for _, clause := range defaultClauses {
			consequence := gv.translate.TreeChildByFieldName(clause, "consequence")
			if consequence != nil {
				branches = append(branches, consequence)
			}
		}
	}

	return gv.translate.HandleConditional(ctx, tsNode, conditions, branches, scopeID)
}

func (gv *GoVisitor) handleGoStatement(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	callNode := gv.translate.TreeChildByFieldName(tsNode, "call")
	if callNode != nil {
		return gv.TraverseNode(ctx, callNode, scopeID)
	}
	return ast.InvalidNodeID
}

func (gv *GoVisitor) handleDeferStatement(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	callNode := gv.translate.TreeChildByFieldName(tsNode, "call")
	if callNode != nil {
		return gv.TraverseNode(ctx, callNode, scopeID)
	}
	return ast.InvalidNodeID
}

func (gv *GoVisitor) handleSelectStatement(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	bodyNode := gv.translate.TreeChildByFieldName(tsNode, "body")

	var conditions []*tree_sitter.Node
	var branches []*tree_sitter.Node

	if bodyNode != nil {
		communicationClauses := gv.translate.TreeChildrenByKind(bodyNode, "communication_case")
		defaultClauses := gv.translate.TreeChildrenByKind(bodyNode, "default_case")

		for _, clause := range communicationClauses {
			communication := gv.translate.TreeChildByFieldName(clause, "communication")
			if communication != nil {
				conditions = append(conditions, communication)
			}
			consequence := gv.translate.TreeChildByFieldName(clause, "consequence")
			if consequence != nil {
				branches = append(branches, consequence)
			}
		}

		for _, clause := range defaultClauses {
			consequence := gv.translate.TreeChildByFieldName(clause, "consequence")
			if consequence != nil {
				branches = append(branches, consequence)
			}
		}
	}

	return gv.translate.HandleConditional(ctx, tsNode, conditions, branches, scopeID)
}

// handleImportDeclaration processes Go import declarations
// For imports like:
//
//	import "fmt"
//	import (
//	    "context"
//	    "github.com/example/counter-app/pkg/counter"
//	)
//
// Creates Import nodes and adds symbols to scope so they can be resolved
// when used in selector expressions like `counter.New()` or `fmt.Println()`
func (gv *GoVisitor) handleImportDeclaration(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	// Get import_spec_list for grouped imports, or single import_spec
	importSpecList := gv.translate.TreeChildByKind(tsNode, "import_spec_list")

	var importSpecs []*tree_sitter.Node
	if importSpecList != nil {
		// Grouped imports: import ( ... )
		importSpecs = gv.translate.TreeChildrenByKind(importSpecList, "import_spec")
	} else {
		// Single import: import "fmt"
		singleSpec := gv.translate.TreeChildByKind(tsNode, "import_spec")
		if singleSpec != nil {
			importSpecs = []*tree_sitter.Node{singleSpec}
		}
	}

	for _, spec := range importSpecs {
		gv.handleImportSpec(ctx, spec, scopeID)
	}

	return ast.InvalidNodeID
}

// handleImportSpec processes a single import spec
// Handles:
//   - Regular imports: "fmt" -> symbol name is "fmt"
//   - Path imports: "github.com/example/pkg/counter" -> symbol name is "counter"
//   - Aliased imports: alias "path/to/pkg" -> symbol name is "alias"
//   - Dot imports: . "path/to/pkg" -> no symbol (imports into current namespace)
//   - Blank imports: _ "path/to/pkg" -> no symbol (side effects only)
func (gv *GoVisitor) handleImportSpec(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	// Check for alias (named import like: alias "path/to/pkg")
	aliasNode := gv.translate.TreeChildByKind(tsNode, "package_identifier")
	// Check for blank identifier or dot import
	blankNode := gv.translate.TreeChildByKind(tsNode, "blank_identifier")
	dotNode := gv.translate.TreeChildByKind(tsNode, "dot")

	// Get the import path string
	pathNode := gv.translate.TreeChildByKind(tsNode, "interpreted_string_literal")
	if pathNode == nil {
		return ast.InvalidNodeID
	}

	// Get the actual path content (without quotes)
	pathContentNode := gv.translate.TreeChildByKind(pathNode, "interpreted_string_literal_content")
	importPath := ""
	if pathContentNode != nil {
		importPath = gv.translate.String(pathContentNode)
	} else {
		// Fallback: get full string and strip quotes
		importPath = gv.translate.String(pathNode)
		if len(importPath) >= 2 {
			importPath = importPath[1 : len(importPath)-1]
		}
	}

	if importPath == "" {
		return ast.InvalidNodeID
	}

	// Skip blank imports and dot imports - they don't create named symbols
	if blankNode != nil || dotNode != nil {
		return ast.InvalidNodeID
	}

	// Determine the symbol name
	var symbolName string
	if aliasNode != nil {
		// Aliased import: use the alias
		symbolName = gv.translate.String(aliasNode)
	} else {
		// Regular import: use the last component of the path
		symbolName = gv.getPackageNameFromPath(importPath)
	}

	if symbolName == "" {
		return ast.InvalidNodeID
	}

	// Create the Import node
	importNode := ast.NewNode(
		gv.translate.NextNodeID(),
		ast.NodeTypeImport,
		gv.translate.FileID,
		symbolName,
		gv.translate.ToRange(tsNode),
		gv.translate.Version,
		scopeID,
	)

	// Store the full import path in metadata
	importNode.MetaData = map[string]any{
		"importPath": importPath,
	}

	// Write node to code graph
	gv.translate.CodeGraph.CreateImport(ctx, importNode)

	// Add to current scope so it can be resolved when used
	gv.translate.CurrentScope.AddSymbol(NewSymbol(importNode))

	// Track in translator's node map
	gv.translate.Nodes[importNode.ID] = importNode

	return importNode.ID
}

// getPackageNameFromPath extracts the package name from an import path
// For "github.com/example/pkg/counter", returns "counter"
// For "fmt", returns "fmt"
func (gv *GoVisitor) getPackageNameFromPath(importPath string) string {
	if importPath == "" {
		return ""
	}

	// Find the last "/" and take everything after it
	lastSlash := -1
	for i := len(importPath) - 1; i >= 0; i-- {
		if importPath[i] == '/' {
			lastSlash = i
			break
		}
	}

	if lastSlash == -1 {
		return importPath
	}
	return importPath[lastSlash+1:]
}
