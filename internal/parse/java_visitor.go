package parse

import (
	"context"
	"encoding/json"

	"github.com/armchr/codeapi/internal/model/ast"
	tree_sitter "github.com/tree-sitter/go-tree-sitter"
	"go.uber.org/zap"
)

type JavaVisitor struct {
	translate *TranslateFromSyntaxTree
	logger    *zap.Logger
}

func NewJavaVisitor(logger *zap.Logger, ts *TranslateFromSyntaxTree) *JavaVisitor {
	return &JavaVisitor{
		translate: ts,
		logger:    logger,
	}
}

// extractAnnotations extracts annotations from a modifiers node and returns them as JSON strings
// Each annotation is serialized to JSON for Neo4j compatibility (Neo4j can't store nested maps)
func (jv *JavaVisitor) extractAnnotations(tsNode *tree_sitter.Node) []string {
	modifiers := jv.translate.TreeChildByKind(tsNode, "modifiers")
	if modifiers == nil {
		return nil
	}

	var annotations []string

	// Find all marker_annotation and annotation nodes
	for i := uint(0); i < modifiers.ChildCount(); i++ {
		child := modifiers.Child(i)
		kind := child.Kind()

		if kind == "marker_annotation" || kind == "annotation" {
			annotation := make(map[string]any)

			// Get the annotation name from the identifier child
			nameNode := jv.translate.TreeChildByKind(child, "identifier")
			if nameNode != nil {
				annotation["name"] = jv.translate.String(nameNode)
			}

			// For annotations with arguments, extract the argument list
			if kind == "annotation" {
				argList := jv.translate.TreeChildByKind(child, "annotation_argument_list")
				if argList != nil {
					args := jv.extractAnnotationArguments(argList)
					if len(args) > 0 {
						annotation["arguments"] = args
					}
				}
			}

			if annotation["name"] != nil {
				// Serialize to JSON string for Neo4j compatibility
				jsonBytes, err := json.Marshal(annotation)
				if err == nil {
					annotations = append(annotations, string(jsonBytes))
				}
			}
		}
	}

	return annotations
}

// extractAnnotationArguments extracts arguments from an annotation_argument_list node
func (jv *JavaVisitor) extractAnnotationArguments(argList *tree_sitter.Node) map[string]string {
	args := make(map[string]string)

	for i := uint(0); i < argList.ChildCount(); i++ {
		child := argList.Child(i)
		kind := child.Kind()

		switch kind {
		case "string_literal":
			// Single value annotation like @GetMapping("/path")
			// Extract the string content (without quotes)
			stringFragment := jv.translate.TreeChildByKind(child, "string_fragment")
			if stringFragment != nil {
				args["value"] = jv.translate.String(stringFragment)
			}
		case "element_value_pair":
			// Named argument like @Size(min = 1, max = 50)
			keyNode := jv.translate.TreeChildByKind(child, "identifier")
			if keyNode != nil {
				key := jv.translate.String(keyNode)
				// Try to get the value - could be string_literal, decimal_integer_literal, etc.
				for j := uint(0); j < child.ChildCount(); j++ {
					valChild := child.Child(j)
					valKind := valChild.Kind()
					if valKind == "string_literal" {
						stringFragment := jv.translate.TreeChildByKind(valChild, "string_fragment")
						if stringFragment != nil {
							args[key] = jv.translate.String(stringFragment)
						}
					} else if valKind == "decimal_integer_literal" || valKind == "true" || valKind == "false" {
						args[key] = jv.translate.String(valChild)
					}
				}
			}
		}
	}

	return args
}

func (jv *JavaVisitor) TraverseNode(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	if tsNode == nil {
		return ast.InvalidNodeID
	}

	switch tsNode.Kind() {
	case "program":
		return jv.handleProgram(ctx, tsNode)
	case "package_declaration":
		return jv.handlePackageDeclaration(ctx, tsNode, scopeID)
	case "class_declaration":
		return jv.handleClassDeclaration(ctx, tsNode, scopeID)
	case "interface_declaration":
		return jv.handleInterfaceDeclaration(ctx, tsNode, scopeID)
	case "record_declaration":
		return jv.handleRecordDeclaration(ctx, tsNode, scopeID)
	case "enum_declaration":
		return jv.handleEnumDeclaration(ctx, tsNode, scopeID)
	case "method_declaration":
		return jv.handleMethodDeclaration(ctx, tsNode, scopeID)
	case "constructor_declaration":
		return jv.handleConstructorDeclaration(ctx, tsNode, scopeID)
	case "field_declaration":
		return jv.handleFieldDeclaration(ctx, tsNode, scopeID)
	case "block":
		return jv.translate.HandleBlock(ctx, tsNode, scopeID)
	case "local_variable_declaration":
		return jv.handleLocalVariableDeclaration(ctx, tsNode, scopeID)
	case "expression_statement":
		return jv.handleExpressionStatement(ctx, tsNode, scopeID)
	case "return_statement":
		return jv.handleReturnStatement(ctx, tsNode, scopeID)
	case "if_statement":
		return jv.handleIfStatement(ctx, tsNode, scopeID)
	case "for_statement":
		return jv.handleForStatement(ctx, tsNode, scopeID)
	case "enhanced_for_statement":
		return jv.handleEnhancedForStatement(ctx, tsNode, scopeID)
	case "while_statement":
		return jv.handleWhileStatement(ctx, tsNode, scopeID)
	case "do_statement":
		return jv.handleDoStatement(ctx, tsNode, scopeID)
	case "switch_expression", "switch_statement":
		return jv.handleSwitchExpression(ctx, tsNode, scopeID)
	case "try_statement":
		return jv.handleTryStatement(ctx, tsNode, scopeID)
	case "try_with_resources_statement":
		return jv.handleTryWithResourcesStatement(ctx, tsNode, scopeID)
	case "throw_statement":
		return jv.handleThrowStatement(ctx, tsNode, scopeID)
	case "method_invocation":
		return jv.handleMethodInvocation(ctx, tsNode, scopeID)
	case "object_creation_expression":
		return jv.handleObjectCreationExpression(ctx, tsNode, scopeID)
	case "assignment_expression":
		return jv.handleAssignmentExpression(ctx, tsNode, scopeID)
	case "field_access":
		return jv.handleFieldAccess(ctx, tsNode, scopeID)
	case "array_access":
		return jv.handleArrayAccess(ctx, tsNode, scopeID)
	case "identifier":
		return jv.translate.HandleIdentifier(ctx, tsNode, scopeID)
	case "type_identifier":
		return jv.handleTypeIdentifier(ctx, tsNode, scopeID)
	case "scoped_identifier":
		return jv.handleScopedIdentifier(ctx, tsNode, scopeID)
	case "class_literal":
		return jv.handleClassLiteral(ctx, tsNode, scopeID)
	case "this":
		return jv.handleThis(ctx, tsNode, scopeID)
	case "super":
		return jv.handleSuper(ctx, tsNode, scopeID)
	case "import_declaration":
		return jv.handleImportDeclaration(ctx, tsNode, scopeID)
	case "lambda_expression":
		return jv.handleLambdaExpression(ctx, tsNode, scopeID)
	case "ternary_expression":
		return jv.handleTernaryExpression(ctx, tsNode, scopeID)
	case "binary_expression":
		return jv.handleBinaryExpression(ctx, tsNode, scopeID)
	case "unary_expression":
		return jv.handleUnaryExpression(ctx, tsNode, scopeID)
	case "cast_expression":
		return jv.handleCastExpression(ctx, tsNode, scopeID)
	case "parenthesized_expression":
		return jv.handleParenthesizedExpression(ctx, tsNode, scopeID)
	case "update_expression":
		return jv.handleUpdateExpression(ctx, tsNode, scopeID)
	case "generic_type":
		return jv.handleGenericType(ctx, tsNode, scopeID)
	case "annotation", "marker_annotation":
		return jv.handleAnnotation(ctx, tsNode, scopeID)
	case "string_literal":
		// String literals are leaf nodes, no traversal needed
		return ast.InvalidNodeID
	case "block_comment", "line_comment":
		// Skip comments
		return ast.InvalidNodeID
	case "modifiers":
		// Traverse modifiers to process annotations
		jv.translate.TraverseChildren(ctx, tsNode, scopeID)
		return ast.InvalidNodeID
	case "boolean_type", "void_type", "integral_type", "floating_point_type":
		// Primitive types are leaf nodes
		return ast.InvalidNodeID
	case "type_arguments", "type_list", "extends_interfaces", "implements_interfaces", "superclass":
		// Type-related nodes - traverse children for type references
		jv.translate.TraverseChildren(ctx, tsNode, scopeID)
		return ast.InvalidNodeID
	case "annotation_argument_list", "element_value_pair", "element_value_array_initializer":
		// Annotation arguments - traverse for any expressions
		jv.translate.TraverseChildren(ctx, tsNode, scopeID)
		return ast.InvalidNodeID
	case "array_creation_expression":
		return jv.handleArrayCreationExpression(ctx, tsNode, scopeID)
	case "array_initializer":
		return jv.handleArrayInitializer(ctx, tsNode, scopeID)
	case "instanceof_expression":
		return jv.handleInstanceofExpression(ctx, tsNode, scopeID)
	default:
		jv.translate.TraverseChildren(ctx, tsNode, scopeID)
		return ast.InvalidNodeID
	}
}

func (jv *JavaVisitor) handleProgram(ctx context.Context, tsNode *tree_sitter.Node) ast.NodeID {
	// Look for package declaration first
	packageDecl := jv.translate.TreeChildByKind(tsNode, "package_declaration")
	var moduleNodeID ast.NodeID

	if packageDecl != nil {
		moduleNodeID = jv.handlePackageDeclaration(ctx, packageDecl, ast.NodeID(jv.translate.FileID))
	} else {
		// Create a default module scope for files without package declaration
		moduleNode := ast.NewNode(
			jv.translate.NextNodeID(), ast.NodeTypeModuleScope, jv.translate.FileID,
			"default", jv.translate.ToRange(tsNode), jv.translate.Version,
			ast.NodeID(jv.translate.FileID),
		)
		jv.translate.CodeGraph.CreateModuleScope(ctx, moduleNode)
		moduleNodeID = moduleNode.ID
	}

	jv.translate.PushScope(false)
	defer jv.translate.PopScope(ctx, moduleNodeID)

	childNodes := jv.translate.TraverseChildren(ctx, tsNode, moduleNodeID)
	if len(childNodes) > 0 {
		jv.translate.CreateContainsRelations(ctx, moduleNodeID, childNodes)
	}
	return moduleNodeID
}

func (jv *JavaVisitor) handlePackageDeclaration(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	// Get the scoped_identifier which contains the package name
	nameNode := jv.translate.TreeChildByKind(tsNode, "scoped_identifier")
	if nameNode == nil {
		nameNode = jv.translate.TreeChildByKind(tsNode, "identifier")
	}

	packageName := ""
	if nameNode != nil {
		packageName = jv.translate.String(nameNode)
	}

	moduleNode := ast.NewNode(
		jv.translate.NextNodeID(), ast.NodeTypeModuleScope, jv.translate.FileID,
		packageName, jv.translate.ToRange(tsNode), jv.translate.Version,
		ast.NodeID(jv.translate.FileID),
	)
	jv.translate.CodeGraph.CreateModuleScope(ctx, moduleNode)
	return moduleNode.ID
}

func (jv *JavaVisitor) handleClassDeclaration(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	nameNode := jv.translate.TreeChildByFieldName(tsNode, "name")
	className := ""
	if nameNode != nil {
		className = jv.translate.String(nameNode)
	}

	// Use TreeChildByKind since Java grammar uses "class_body" as node kind, not "body" as field name
	classBody := jv.translate.TreeChildByKind(tsNode, "class_body")
	var methods []*tree_sitter.Node
	var fields []*tree_sitter.Node

	if classBody != nil {
		methods = jv.translate.TreeChildrenByKind(classBody, "method_declaration")
		constructors := jv.translate.TreeChildrenByKind(classBody, "constructor_declaration")
		methods = append(methods, constructors...)
		fields = jv.translate.TreeChildrenByKind(classBody, "field_declaration")
	}

	// Extract annotations from modifiers
	metadata := make(map[string]any)
	annotations := jv.extractAnnotations(tsNode)
	if len(annotations) > 0 {
		metadata["annotations"] = annotations
	}

	// Extract superclass (extends)
	superclassNode := jv.translate.TreeChildByKind(tsNode, "superclass")
	if superclassNode != nil {
		superclassName := jv.extractTypeName(superclassNode)
		if superclassName != "" {
			metadata["extends"] = superclassName
		}
	}

	// Extract implemented interfaces
	interfacesNode := jv.translate.TreeChildByKind(tsNode, "super_interfaces")
	if interfacesNode != nil {
		interfaces := jv.extractTypeList(interfacesNode)
		if len(interfaces) > 0 {
			metadata["implements"] = interfaces
		}
	}

	// Pass nil for fields - we'll handle field_declarations separately
	// because they have a different structure (variable_declarator children)
	classNodeID := jv.translate.HandleClassWithMetadata(ctx, scopeID, tsNode, className, methods, nil, metadata)

	// Handle field declarations within the class scope
	if classNodeID != ast.InvalidNodeID {
		for _, field := range fields {
			jv.handleFieldDeclaration(ctx, field, classNodeID)
		}
	}

	return classNodeID
}

func (jv *JavaVisitor) handleInterfaceDeclaration(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	nameNode := jv.translate.TreeChildByFieldName(tsNode, "name")
	interfaceName := ""
	if nameNode != nil {
		interfaceName = jv.translate.String(nameNode)
	}

	interfaceBody := jv.translate.TreeChildByKind(tsNode, "interface_body")
	var methods []*tree_sitter.Node

	if interfaceBody != nil {
		methods = jv.translate.TreeChildrenByKind(interfaceBody, "method_declaration")
	}

	// Extract annotations and mark as interface
	metadata := map[string]any{"is_interface": true}
	annotations := jv.extractAnnotations(tsNode)
	if len(annotations) > 0 {
		metadata["annotations"] = annotations
	}

	// Extract extended interfaces (interface Foo extends Bar, Baz)
	extendsNode := jv.translate.TreeChildByKind(tsNode, "extends_interfaces")
	if extendsNode != nil {
		extendedInterfaces := jv.extractTypeList(extendsNode)
		if len(extendedInterfaces) > 0 {
			metadata["extends"] = extendedInterfaces
		}
	}

	return jv.translate.HandleClassWithMetadata(ctx, scopeID, tsNode, interfaceName, methods, nil, metadata)
}

func (jv *JavaVisitor) handleRecordDeclaration(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	nameNode := jv.translate.TreeChildByFieldName(tsNode, "name")
	recordName := ""
	if nameNode != nil {
		recordName = jv.translate.String(nameNode)
	}

	// Record parameters become fields
	paramList := jv.translate.TreeChildByKind(tsNode, "formal_parameters")
	var fields []*tree_sitter.Node
	if paramList != nil {
		fields = jv.translate.TreeChildrenByKind(paramList, "formal_parameter")
	}

	recordBody := jv.translate.TreeChildByKind(tsNode, "class_body")
	var methods []*tree_sitter.Node
	if recordBody != nil {
		methods = jv.translate.TreeChildrenByKind(recordBody, "method_declaration")
	}

	// Extract annotations and mark as record
	var metadata map[string]any
	annotations := jv.extractAnnotations(tsNode)
	if len(annotations) > 0 {
		metadata = map[string]any{"annotations": annotations, "is_record": true}
	} else {
		metadata = map[string]any{"is_record": true}
	}

	return jv.translate.HandleClassWithMetadata(ctx, scopeID, tsNode, recordName, methods, fields, metadata)
}

func (jv *JavaVisitor) handleEnumDeclaration(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	nameNode := jv.translate.TreeChildByFieldName(tsNode, "name")
	enumName := ""
	if nameNode != nil {
		enumName = jv.translate.String(nameNode)
	}

	enumBody := jv.translate.TreeChildByKind(tsNode, "enum_body")
	var methods []*tree_sitter.Node
	var fields []*tree_sitter.Node

	if enumBody != nil {
		methods = jv.translate.TreeChildrenByKind(enumBody, "method_declaration")
		fields = jv.translate.TreeChildrenByKind(enumBody, "enum_constant")
	}

	// Extract annotations and mark as enum
	var metadata map[string]any
	annotations := jv.extractAnnotations(tsNode)
	if len(annotations) > 0 {
		metadata = map[string]any{"annotations": annotations, "is_enum": true}
	} else {
		metadata = map[string]any{"is_enum": true}
	}

	return jv.translate.HandleClassWithMetadata(ctx, scopeID, tsNode, enumName, methods, fields, metadata)
}

func (jv *JavaVisitor) handleMethodDeclaration(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	nameNode := jv.translate.TreeChildByFieldName(tsNode, "name")
	methodName := ""
	if nameNode != nil {
		methodName = jv.translate.String(nameNode)
	}

	paramsNode := jv.translate.TreeChildByFieldName(tsNode, "parameters")
	bodyNode := jv.translate.TreeChildByFieldName(tsNode, "body")

	var params []*tree_sitter.Node
	if paramsNode != nil {
		params = jv.translate.TreeChildrenByKind(paramsNode, "formal_parameter")
		spreadParams := jv.translate.TreeChildrenByKind(paramsNode, "spread_parameter")
		params = append(params, spreadParams...)
	}

	// Extract annotations from modifiers
	var metadata map[string]any
	annotations := jv.extractAnnotations(tsNode)
	if len(annotations) > 0 {
		metadata = map[string]any{"annotations": annotations}
	}

	return jv.translate.CreateFunctionWithMetadata(ctx, scopeID, tsNode, methodName, params, bodyNode, metadata)
}

func (jv *JavaVisitor) handleConstructorDeclaration(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	nameNode := jv.translate.TreeChildByFieldName(tsNode, "name")
	constructorName := ""
	if nameNode != nil {
		constructorName = jv.translate.String(nameNode)
	}

	paramsNode := jv.translate.TreeChildByFieldName(tsNode, "parameters")
	bodyNode := jv.translate.TreeChildByKind(tsNode, "constructor_body")

	var params []*tree_sitter.Node
	if paramsNode != nil {
		params = jv.translate.TreeChildrenByKind(paramsNode, "formal_parameter")
	}

	// Extract annotations and mark as constructor
	metadata := map[string]any{"is_constructor": true}
	annotations := jv.extractAnnotations(tsNode)
	if len(annotations) > 0 {
		metadata["annotations"] = annotations
	}

	return jv.translate.CreateFunctionWithMetadata(ctx, scopeID, tsNode, constructorName, params, bodyNode, metadata)
}

func (jv *JavaVisitor) handleFieldDeclaration(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	declarators := jv.translate.TreeChildrenByKind(tsNode, "variable_declarator")
	var firstFieldID ast.NodeID = ast.InvalidNodeID

	for _, declarator := range declarators {
		nameNode := jv.translate.TreeChildByFieldName(declarator, "name")
		if nameNode != nil {
			fieldNodeID := jv.translate.HandleVariable(ctx, nameNode, scopeID)
			if fieldNodeID != ast.InvalidNodeID {
				if firstFieldID == ast.InvalidNodeID {
					firstFieldID = fieldNodeID
				}
				// Create CONTAINS and HAS_FIELD relations for class fields
				jv.translate.CreateContainsRelation(ctx, scopeID, fieldNodeID, jv.translate.FileID)
				jv.translate.CodeGraph.CreateHasFieldRelation(ctx, scopeID, fieldNodeID, jv.translate.FileID)
			}
		}

		// Handle initialization if present
		valueNode := jv.translate.TreeChildByFieldName(declarator, "value")
		if nameNode != nil && valueNode != nil {
			jv.translate.HandleAssignment(ctx, declarator, nameNode, valueNode, scopeID)
		}
	}
	return firstFieldID
}

func (jv *JavaVisitor) handleLocalVariableDeclaration(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	declarators := jv.translate.TreeChildrenByKind(tsNode, "variable_declarator")
	for _, declarator := range declarators {
		nameNode := jv.translate.TreeChildByFieldName(declarator, "name")
		valueNode := jv.translate.TreeChildByFieldName(declarator, "value")

		if nameNode != nil {
			jv.translate.HandleVariable(ctx, nameNode, scopeID)
		}

		if nameNode != nil && valueNode != nil {
			jv.translate.HandleAssignment(ctx, declarator, nameNode, valueNode, scopeID)
		}
	}
	return ast.InvalidNodeID
}

func (jv *JavaVisitor) handleExpressionStatement(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	if tsNode.ChildCount() > 0 {
		return jv.TraverseNode(ctx, tsNode.Child(0), scopeID)
	}
	return ast.InvalidNodeID
}

func (jv *JavaVisitor) handleReturnStatement(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	// Return statement may have an expression child
	for i := uint(0); i < tsNode.ChildCount(); i++ {
		child := tsNode.Child(i)
		if child.IsNamed() {
			return jv.translate.HandleReturn(ctx, child, scopeID)
		}
	}
	return ast.InvalidNodeID
}

func (jv *JavaVisitor) handleIfStatement(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	conditionNode := jv.translate.TreeChildByFieldName(tsNode, "condition")
	consequenceNode := jv.translate.TreeChildByFieldName(tsNode, "consequence")
	alternativeNode := jv.translate.TreeChildByFieldName(tsNode, "alternative")

	conditions := []*tree_sitter.Node{conditionNode}
	branches := []*tree_sitter.Node{consequenceNode}

	if alternativeNode != nil {
		if alternativeNode.Kind() == "if_statement" {
			altCondition := jv.translate.TreeChildByFieldName(alternativeNode, "condition")
			altConsequence := jv.translate.TreeChildByFieldName(alternativeNode, "consequence")
			conditions = append(conditions, altCondition)
			branches = append(branches, altConsequence)

			altAlternative := jv.translate.TreeChildByFieldName(alternativeNode, "alternative")
			if altAlternative != nil {
				branches = append(branches, altAlternative)
			}
		} else {
			branches = append(branches, alternativeNode)
		}
	}

	return jv.translate.HandleConditional(ctx, tsNode, conditions, branches, scopeID)
}

func (jv *JavaVisitor) handleForStatement(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	initNode := jv.translate.TreeChildByFieldName(tsNode, "init")
	conditionNode := jv.translate.TreeChildByFieldName(tsNode, "condition")
	updateNode := jv.translate.TreeChildByFieldName(tsNode, "update")
	bodyNode := jv.translate.TreeChildByFieldName(tsNode, "body")

	jv.translate.PushScope(false)
	defer jv.translate.PopScope(ctx, ast.InvalidNodeID)

	var inits []*tree_sitter.Node
	if initNode != nil {
		inits = append(inits, initNode)
	}
	if conditionNode != nil {
		inits = append(inits, conditionNode)
	}

	initCondID := ast.InvalidNodeID
	if len(inits) > 0 {
		initCondID = jv.translate.HandleRhsExprsWithFakeVariable(ctx, "__init__", inits, scopeID, nil)
	}

	updateID := ast.InvalidNodeID
	if updateNode != nil {
		updateID = jv.translate.HandleRhsWithFakeVariable(ctx, "__update__", updateNode, scopeID, nil)
	}

	if bodyNode == nil {
		return ast.InvalidNodeID
	}
	return jv.translate.HandleLoop(ctx, tsNode, updateID, initCondID, bodyNode, scopeID)
}

func (jv *JavaVisitor) handleEnhancedForStatement(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	// for (Type var : iterable) { body }
	valueNode := jv.translate.TreeChildByFieldName(tsNode, "value")
	bodyNode := jv.translate.TreeChildByFieldName(tsNode, "body")

	jv.translate.PushScope(false)
	defer jv.translate.PopScope(ctx, ast.InvalidNodeID)

	condID := ast.InvalidNodeID
	if valueNode != nil {
		condID = jv.translate.HandleRhsWithFakeVariable(ctx, "__iter__", valueNode, scopeID, nil)
	}

	if bodyNode == nil {
		return ast.InvalidNodeID
	}
	return jv.translate.HandleLoop(ctx, tsNode, ast.InvalidNodeID, condID, bodyNode, scopeID)
}

func (jv *JavaVisitor) handleWhileStatement(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	conditionNode := jv.translate.TreeChildByFieldName(tsNode, "condition")
	bodyNode := jv.translate.TreeChildByFieldName(tsNode, "body")

	condID := ast.InvalidNodeID
	if conditionNode != nil {
		condID = jv.translate.HandleRhsWithFakeVariable(ctx, "__cond__", conditionNode, scopeID, nil)
	}

	if bodyNode == nil {
		return ast.InvalidNodeID
	}
	return jv.translate.HandleLoop(ctx, tsNode, ast.InvalidNodeID, condID, bodyNode, scopeID)
}

func (jv *JavaVisitor) handleSwitchExpression(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	conditionNode := jv.translate.TreeChildByFieldName(tsNode, "condition")
	bodyNode := jv.translate.TreeChildByFieldName(tsNode, "body")

	var conditions []*tree_sitter.Node
	var branches []*tree_sitter.Node

	if conditionNode != nil {
		conditions = append(conditions, conditionNode)
	}

	if bodyNode != nil {
		switchBlocks := jv.translate.TreeChildrenByKind(bodyNode, "switch_block_statement_group")
		for _, block := range switchBlocks {
			labelNode := jv.translate.TreeChildByKind(block, "switch_label")
			if labelNode != nil {
				conditions = append(conditions, labelNode)
			}
			branches = append(branches, block)
		}

		// Handle switch rules (arrow syntax)
		switchRules := jv.translate.TreeChildrenByKind(bodyNode, "switch_rule")
		for _, rule := range switchRules {
			labelNode := jv.translate.TreeChildByKind(rule, "switch_label")
			if labelNode != nil {
				conditions = append(conditions, labelNode)
			}
			branches = append(branches, rule)
		}
	}

	return jv.translate.HandleConditional(ctx, tsNode, conditions, branches, scopeID)
}

func (jv *JavaVisitor) handleTryStatement(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	bodyNode := jv.translate.TreeChildByFieldName(tsNode, "body")
	if bodyNode != nil {
		jv.TraverseNode(ctx, bodyNode, scopeID)
	}

	// Handle catch clauses
	catchClauses := jv.translate.TreeChildrenByKind(tsNode, "catch_clause")
	for _, clause := range catchClauses {
		catchBody := jv.translate.TreeChildByFieldName(clause, "body")
		if catchBody != nil {
			jv.TraverseNode(ctx, catchBody, scopeID)
		}
	}

	// Handle finally clause
	finallyClause := jv.translate.TreeChildByKind(tsNode, "finally_clause")
	if finallyClause != nil {
		finallyBody := jv.translate.TreeChildByKind(finallyClause, "block")
		if finallyBody != nil {
			jv.TraverseNode(ctx, finallyBody, scopeID)
		}
	}

	return ast.InvalidNodeID
}

func (jv *JavaVisitor) handleThrowStatement(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	for i := uint(0); i < tsNode.ChildCount(); i++ {
		child := tsNode.Child(i)
		if child.IsNamed() {
			return jv.translate.HandleRhsWithFakeVariable(ctx, "__throw__", child, scopeID, nil)
		}
	}
	return ast.InvalidNodeID
}

func (jv *JavaVisitor) handleMethodInvocation(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	objectNode := jv.translate.TreeChildByFieldName(tsNode, "object")
	nameNode := jv.translate.TreeChildByFieldName(tsNode, "name")
	argumentsNode := jv.translate.TreeChildByFieldName(tsNode, "arguments")

	// For chained method calls like obj.method1().method2(), the objectNode is another method_invocation.
	// We need to traverse it first to create the FunctionCall node for the inner call,
	// rather than trying to resolve it as a name chain (which doesn't work for method invocations).
	if objectNode != nil && objectNode.Kind() == "method_invocation" {
		// Traverse the inner method invocation first to create its FunctionCall node
		jv.TraverseNode(ctx, objectNode, scopeID)
	}

	// Build name chain only for simple identifiers/field accesses, not for method invocations
	var nameChain []*tree_sitter.Node
	if objectNode != nil && objectNode.Kind() != "method_invocation" {
		nameChain = append(nameChain, objectNode)
	}
	if nameNode != nil {
		nameChain = append(nameChain, nameNode)
	}

	fnNameNodeID := ast.InvalidNodeID
	if len(nameChain) > 0 {
		fnNameNodeID = jv.translate.ResolveNameChain(ctx, nameChain, scopeID)
	}

	if fnNameNodeID == ast.InvalidNodeID && nameNode != nil {
		fnNameNodeID = jv.translate.HandleRhsWithFakeVariable(ctx, "__fn__", nameNode, scopeID, nil)
	}

	var args []*tree_sitter.Node
	if argumentsNode != nil {
		args = jv.translate.NamedChildren(argumentsNode)
	}

	return jv.translate.HandleCall(ctx, fnNameNodeID, args, scopeID, jv.translate.ToRange(tsNode))
}

func (jv *JavaVisitor) handleObjectCreationExpression(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	typeNode := jv.translate.TreeChildByFieldName(tsNode, "type")
	argumentsNode := jv.translate.TreeChildByFieldName(tsNode, "arguments")

	fnNameNodeID := ast.InvalidNodeID
	if typeNode != nil {
		fnNameNodeID = jv.translate.HandleRhsWithFakeVariable(ctx, "__new__", typeNode, scopeID, nil)
	}

	var args []*tree_sitter.Node
	if argumentsNode != nil {
		args = jv.translate.NamedChildren(argumentsNode)
	}

	// Mark this call as a constructor call for post-processing
	metadata := map[string]any{
		"is_constructor": true,
	}

	return jv.translate.HandleCallWithMetadata(ctx, fnNameNodeID, args, scopeID, jv.translate.ToRange(tsNode), metadata)
}

func (jv *JavaVisitor) handleAssignmentExpression(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	leftNode := jv.translate.TreeChildByFieldName(tsNode, "left")
	rightNode := jv.translate.TreeChildByFieldName(tsNode, "right")

	if leftNode == nil || rightNode == nil {
		return ast.InvalidNodeID
	}

	return jv.translate.HandleAssignment(ctx, tsNode, leftNode, rightNode, scopeID)
}

func (jv *JavaVisitor) handleFieldAccess(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	objectNode := jv.translate.TreeChildByFieldName(tsNode, "object")
	fieldNode := jv.translate.TreeChildByFieldName(tsNode, "field")

	var names []*tree_sitter.Node
	if objectNode != nil {
		names = append(names, objectNode)
	}
	if fieldNode != nil {
		names = append(names, fieldNode)
	}

	resolvedNodeId := jv.translate.ResolveNameChain(ctx, names, scopeID)
	if jv.translate.CurrentScope.IsRhs() && resolvedNodeId != ast.InvalidNodeID {
		jv.translate.CurrentScope.AddRhsVar(resolvedNodeId)
	}
	return resolvedNodeId
}

func (jv *JavaVisitor) handleImportDeclaration(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	// Get the scoped_identifier which contains the import path
	nameNode := jv.translate.TreeChildByKind(tsNode, "scoped_identifier")
	if nameNode == nil {
		return ast.InvalidNodeID
	}

	importPath := jv.translate.String(nameNode)
	if importPath == "" {
		return ast.InvalidNodeID
	}

	// Extract the simple name (last component)
	symbolName := jv.getSimpleNameFromImport(importPath)
	if symbolName == "" || symbolName == "*" {
		return ast.InvalidNodeID
	}

	importNode := ast.NewNode(
		jv.translate.NextNodeID(),
		ast.NodeTypeImport,
		jv.translate.FileID,
		symbolName,
		jv.translate.ToRange(tsNode),
		jv.translate.Version,
		scopeID,
	)

	importNode.MetaData = map[string]any{
		"importPath": importPath,
	}

	jv.translate.CodeGraph.CreateImport(ctx, importNode)
	jv.translate.CurrentScope.AddSymbol(NewSymbol(importNode))
	jv.translate.Nodes[importNode.ID] = importNode

	return importNode.ID
}

func (jv *JavaVisitor) handleLambdaExpression(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	paramsNode := jv.translate.TreeChildByFieldName(tsNode, "parameters")
	bodyNode := jv.translate.TreeChildByFieldName(tsNode, "body")

	var params []*tree_sitter.Node
	if paramsNode != nil {
		// Could be identifier (single param) or formal_parameters
		if paramsNode.Kind() == "identifier" {
			params = []*tree_sitter.Node{paramsNode}
		} else if paramsNode.Kind() == "inferred_parameters" {
			params = jv.translate.NamedChildren(paramsNode)
		} else {
			params = jv.translate.TreeChildrenByKind(paramsNode, "formal_parameter")
		}
	}

	return jv.translate.CreateFunction(ctx, scopeID, tsNode, "__lambda__", params, bodyNode)
}

// getSimpleNameFromImport extracts the simple name from a fully qualified import
// e.g., "com.example.petclinic.model.Pet" -> "Pet"
func (jv *JavaVisitor) getSimpleNameFromImport(importPath string) string {
	if importPath == "" {
		return ""
	}

	lastDot := -1
	for i := len(importPath) - 1; i >= 0; i-- {
		if importPath[i] == '.' {
			lastDot = i
			break
		}
	}

	if lastDot == -1 {
		return importPath
	}
	return importPath[lastDot+1:]
}

func (jv *JavaVisitor) handleDoStatement(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	bodyNode := jv.translate.TreeChildByFieldName(tsNode, "body")
	conditionNode := jv.translate.TreeChildByFieldName(tsNode, "condition")

	condID := ast.InvalidNodeID
	if conditionNode != nil {
		condID = jv.translate.HandleRhsWithFakeVariable(ctx, "__cond__", conditionNode, scopeID, nil)
	}

	if bodyNode == nil {
		return ast.InvalidNodeID
	}
	return jv.translate.HandleLoop(ctx, tsNode, ast.InvalidNodeID, condID, bodyNode, scopeID)
}

func (jv *JavaVisitor) handleTryWithResourcesStatement(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	// Handle resources
	resourcesNode := jv.translate.TreeChildByFieldName(tsNode, "resources")
	if resourcesNode != nil {
		jv.translate.TraverseChildren(ctx, resourcesNode, scopeID)
	}

	// Handle body
	bodyNode := jv.translate.TreeChildByFieldName(tsNode, "body")
	if bodyNode != nil {
		jv.TraverseNode(ctx, bodyNode, scopeID)
	}

	// Handle catch clauses
	catchClauses := jv.translate.TreeChildrenByKind(tsNode, "catch_clause")
	for _, clause := range catchClauses {
		catchBody := jv.translate.TreeChildByFieldName(clause, "body")
		if catchBody != nil {
			jv.TraverseNode(ctx, catchBody, scopeID)
		}
	}

	// Handle finally clause
	finallyClause := jv.translate.TreeChildByKind(tsNode, "finally_clause")
	if finallyClause != nil {
		finallyBody := jv.translate.TreeChildByKind(finallyClause, "block")
		if finallyBody != nil {
			jv.TraverseNode(ctx, finallyBody, scopeID)
		}
	}

	return ast.InvalidNodeID
}

func (jv *JavaVisitor) handleArrayAccess(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	arrayNode := jv.translate.TreeChildByFieldName(tsNode, "array")
	indexNode := jv.translate.TreeChildByFieldName(tsNode, "index")

	var nodes []*tree_sitter.Node
	if arrayNode != nil {
		nodes = append(nodes, arrayNode)
	}
	if indexNode != nil {
		nodes = append(nodes, indexNode)
	}

	if len(nodes) > 0 {
		return jv.translate.HandleRhsExprsWithFakeVariable(ctx, "__array_access__", nodes, scopeID, nil)
	}
	return ast.InvalidNodeID
}

func (jv *JavaVisitor) handleTypeIdentifier(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	// Type identifiers are class/interface names used as types
	typeName := jv.translate.String(tsNode)
	if typeName == "" {
		return ast.InvalidNodeID
	}

	// Try to resolve the type from imports or current scope
	if sym := jv.translate.CurrentScope.Resolve(typeName); sym != nil {
		if jv.translate.CurrentScope.IsRhs() {
			jv.translate.CurrentScope.AddRhsVar(sym.Node.ID)
		}
		return sym.Node.ID
	}

	// Create a reference to the type
	typeNode := jv.translate.NewNode(
		ast.NodeTypeVariable, typeName, jv.translate.ToRange(tsNode), scopeID,
	)
	typeNode.MetaData = map[string]any{
		"is_type": true,
	}
	jv.translate.CodeGraph.CreateVariable(ctx, typeNode)

	if jv.translate.CurrentScope.IsRhs() {
		jv.translate.CurrentScope.AddRhsVar(typeNode.ID)
	}
	return typeNode.ID
}

func (jv *JavaVisitor) handleScopedIdentifier(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	// Scoped identifier like com.example.ClassName
	// Collect all identifier parts
	var names []*tree_sitter.Node
	jv.collectScopedIdentifierParts(tsNode, &names)

	if len(names) == 0 {
		return ast.InvalidNodeID
	}

	resolvedNodeId := jv.translate.ResolveNameChain(ctx, names, scopeID)
	if jv.translate.CurrentScope.IsRhs() && resolvedNodeId != ast.InvalidNodeID {
		jv.translate.CurrentScope.AddRhsVar(resolvedNodeId)
	}
	return resolvedNodeId
}

func (jv *JavaVisitor) collectScopedIdentifierParts(tsNode *tree_sitter.Node, parts *[]*tree_sitter.Node) {
	if tsNode == nil {
		return
	}

	if tsNode.Kind() == "identifier" || tsNode.Kind() == "type_identifier" {
		*parts = append(*parts, tsNode)
		return
	}

	if tsNode.Kind() == "scoped_identifier" {
		// Recursively collect from nested scoped_identifier
		for i := uint(0); i < tsNode.ChildCount(); i++ {
			child := tsNode.Child(i)
			if child.IsNamed() {
				jv.collectScopedIdentifierParts(child, parts)
			}
		}
	}
}

func (jv *JavaVisitor) handleClassLiteral(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	// Handles ClassName.class expressions
	typeNode := jv.translate.TreeChildByKind(tsNode, "type_identifier")
	if typeNode == nil {
		typeNode = jv.translate.TreeChildByKind(tsNode, "scoped_identifier")
	}
	if typeNode == nil {
		typeNode = jv.translate.TreeChildByKind(tsNode, "identifier")
	}

	if typeNode != nil {
		return jv.translate.HandleRhsWithFakeVariable(ctx, "__class__", typeNode, scopeID, map[string]any{
			"class_literal": true,
		})
	}
	return ast.InvalidNodeID
}

func (jv *JavaVisitor) handleThis(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	// Handle 'this' keyword
	if sym := jv.translate.CurrentScope.Resolve("this"); sym != nil {
		if jv.translate.CurrentScope.IsRhs() {
			jv.translate.CurrentScope.AddRhsVar(sym.Node.ID)
		}
		return sym.Node.ID
	}

	thisNode := jv.translate.NewNode(
		ast.NodeTypeVariable, "this", jv.translate.ToRange(tsNode), scopeID,
	)
	thisNode.MetaData = map[string]any{
		"is_this": true,
	}
	jv.translate.CodeGraph.CreateVariable(ctx, thisNode)
	jv.translate.CurrentScope.AddSymbol(NewSymbol(thisNode))

	if jv.translate.CurrentScope.IsRhs() {
		jv.translate.CurrentScope.AddRhsVar(thisNode.ID)
	}
	return thisNode.ID
}

func (jv *JavaVisitor) handleSuper(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	// Handle 'super' keyword
	superNode := jv.translate.NewNode(
		ast.NodeTypeVariable, "super", jv.translate.ToRange(tsNode), scopeID,
	)
	superNode.MetaData = map[string]any{
		"is_super": true,
	}
	jv.translate.CodeGraph.CreateVariable(ctx, superNode)

	if jv.translate.CurrentScope.IsRhs() {
		jv.translate.CurrentScope.AddRhsVar(superNode.ID)
	}
	return superNode.ID
}

func (jv *JavaVisitor) handleTernaryExpression(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	conditionNode := jv.translate.TreeChildByFieldName(tsNode, "condition")
	consequenceNode := jv.translate.TreeChildByFieldName(tsNode, "consequence")
	alternativeNode := jv.translate.TreeChildByFieldName(tsNode, "alternative")

	var exprs []*tree_sitter.Node
	if conditionNode != nil {
		exprs = append(exprs, conditionNode)
	}
	if consequenceNode != nil {
		exprs = append(exprs, consequenceNode)
	}
	if alternativeNode != nil {
		exprs = append(exprs, alternativeNode)
	}

	if len(exprs) > 0 {
		return jv.translate.HandleRhsExprsWithFakeVariable(ctx, "__ternary__", exprs, scopeID, nil)
	}
	return ast.InvalidNodeID
}

func (jv *JavaVisitor) handleBinaryExpression(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	leftNode := jv.translate.TreeChildByFieldName(tsNode, "left")
	rightNode := jv.translate.TreeChildByFieldName(tsNode, "right")

	var exprs []*tree_sitter.Node
	if leftNode != nil {
		exprs = append(exprs, leftNode)
	}
	if rightNode != nil {
		exprs = append(exprs, rightNode)
	}

	if len(exprs) > 0 {
		return jv.translate.HandleRhsExprsWithFakeVariable(ctx, "__binary__", exprs, scopeID, nil)
	}
	return ast.InvalidNodeID
}

func (jv *JavaVisitor) handleUnaryExpression(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	operandNode := jv.translate.TreeChildByFieldName(tsNode, "operand")
	if operandNode != nil {
		return jv.translate.HandleRhsWithFakeVariable(ctx, "__unary__", operandNode, scopeID, nil)
	}
	return ast.InvalidNodeID
}

func (jv *JavaVisitor) handleCastExpression(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	valueNode := jv.translate.TreeChildByFieldName(tsNode, "value")
	if valueNode != nil {
		return jv.translate.HandleRhsWithFakeVariable(ctx, "__cast__", valueNode, scopeID, nil)
	}
	return ast.InvalidNodeID
}

func (jv *JavaVisitor) handleParenthesizedExpression(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	// Just traverse the inner expression
	for i := uint(0); i < tsNode.ChildCount(); i++ {
		child := tsNode.Child(i)
		if child.IsNamed() {
			return jv.TraverseNode(ctx, child, scopeID)
		}
	}
	return ast.InvalidNodeID
}

func (jv *JavaVisitor) handleUpdateExpression(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	// Handles i++ or ++i
	for i := uint(0); i < tsNode.ChildCount(); i++ {
		child := tsNode.Child(i)
		if child.IsNamed() {
			return jv.TraverseNode(ctx, child, scopeID)
		}
	}
	return ast.InvalidNodeID
}

func (jv *JavaVisitor) handleGenericType(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	// Generic type like List<Owner> or Map<String, Object>
	// Get the base type identifier
	typeIdNode := jv.translate.TreeChildByKind(tsNode, "type_identifier")
	if typeIdNode == nil {
		typeIdNode = jv.translate.TreeChildByKind(tsNode, "scoped_identifier")
	}

	if typeIdNode != nil {
		baseTypeID := jv.TraverseNode(ctx, typeIdNode, scopeID)

		// Also traverse type arguments to resolve referenced types
		typeArgsNode := jv.translate.TreeChildByKind(tsNode, "type_arguments")
		if typeArgsNode != nil {
			jv.translate.TraverseChildren(ctx, typeArgsNode, scopeID)
		}

		return baseTypeID
	}
	return ast.InvalidNodeID
}

func (jv *JavaVisitor) handleAnnotation(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	// Handle annotations like @Query("...") or @Repository
	nameNode := jv.translate.TreeChildByKind(tsNode, "identifier")
	if nameNode == nil {
		nameNode = jv.translate.TreeChildByKind(tsNode, "scoped_identifier")
	}

	if nameNode != nil {
		annotationName := jv.translate.String(nameNode)

		// Try to resolve annotation type from imports
		if sym := jv.translate.CurrentScope.Resolve(annotationName); sym != nil {
			if jv.translate.CurrentScope.IsRhs() {
				jv.translate.CurrentScope.AddRhsVar(sym.Node.ID)
			}
			return sym.Node.ID
		}
	}

	// Traverse annotation arguments for any expressions
	argsNode := jv.translate.TreeChildByKind(tsNode, "annotation_argument_list")
	if argsNode != nil {
		jv.translate.TraverseChildren(ctx, argsNode, scopeID)
	}

	return ast.InvalidNodeID
}

func (jv *JavaVisitor) handleArrayCreationExpression(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	// new Type[size] or new Type[] { elements }
	typeNode := jv.translate.TreeChildByFieldName(tsNode, "type")
	if typeNode != nil {
		jv.TraverseNode(ctx, typeNode, scopeID)
	}

	// Handle dimensions with expressions
	dimensionsNode := jv.translate.TreeChildByKind(tsNode, "dimensions_expr")
	if dimensionsNode != nil {
		return jv.translate.HandleRhsWithFakeVariable(ctx, "__array_new__", dimensionsNode, scopeID, nil)
	}

	// Handle array initializer
	initNode := jv.translate.TreeChildByKind(tsNode, "array_initializer")
	if initNode != nil {
		return jv.handleArrayInitializer(ctx, initNode, scopeID)
	}

	return ast.InvalidNodeID
}

func (jv *JavaVisitor) handleArrayInitializer(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	// { element1, element2, ... }
	var elements []*tree_sitter.Node
	for i := uint(0); i < tsNode.ChildCount(); i++ {
		child := tsNode.Child(i)
		if child.IsNamed() {
			elements = append(elements, child)
		}
	}

	if len(elements) > 0 {
		return jv.translate.HandleRhsExprsWithFakeVariable(ctx, "__array_init__", elements, scopeID, nil)
	}
	return ast.InvalidNodeID
}

func (jv *JavaVisitor) handleInstanceofExpression(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	// expr instanceof Type
	leftNode := jv.translate.TreeChildByFieldName(tsNode, "left")
	rightNode := jv.translate.TreeChildByFieldName(tsNode, "right")

	var exprs []*tree_sitter.Node
	if leftNode != nil {
		exprs = append(exprs, leftNode)
	}
	if rightNode != nil {
		exprs = append(exprs, rightNode)
	}

	if len(exprs) > 0 {
		return jv.translate.HandleRhsExprsWithFakeVariable(ctx, "__instanceof__", exprs, scopeID, nil)
	}
	return ast.InvalidNodeID
}

// extractTypeName extracts a type name from a tree-sitter node.
// This handles superclass nodes, type_identifier, generic_type, and scoped_identifier.
func (jv *JavaVisitor) extractTypeName(tsNode *tree_sitter.Node) string {
	if tsNode == nil {
		return ""
	}

	kind := tsNode.Kind()
	switch kind {
	case "superclass", "extends_interfaces":
		// For superclass node, look for the type child
		typeNode := jv.translate.TreeChildByKind(tsNode, "type_identifier")
		if typeNode != nil {
			return jv.translate.String(typeNode)
		}
		// Try generic_type (e.g., extends List<T>)
		genericNode := jv.translate.TreeChildByKind(tsNode, "generic_type")
		if genericNode != nil {
			return jv.extractTypeNameFromGeneric(genericNode)
		}
		// Try scoped_identifier (e.g., extends com.example.BaseClass)
		scopedNode := jv.translate.TreeChildByKind(tsNode, "scoped_identifier")
		if scopedNode != nil {
			return jv.translate.String(scopedNode)
		}
	case "type_identifier", "identifier":
		return jv.translate.String(tsNode)
	case "generic_type":
		return jv.extractTypeNameFromGeneric(tsNode)
	case "scoped_identifier":
		return jv.translate.String(tsNode)
	}

	return ""
}

// extractTypeNameFromGeneric extracts the base type name from a generic_type node.
// e.g., List<Owner> -> "List", Map<String, Object> -> "Map"
func (jv *JavaVisitor) extractTypeNameFromGeneric(tsNode *tree_sitter.Node) string {
	if tsNode == nil {
		return ""
	}

	// Get the base type identifier
	typeIdNode := jv.translate.TreeChildByKind(tsNode, "type_identifier")
	if typeIdNode != nil {
		return jv.translate.String(typeIdNode)
	}

	// Try scoped identifier for qualified generic types
	scopedNode := jv.translate.TreeChildByKind(tsNode, "scoped_identifier")
	if scopedNode != nil {
		return jv.translate.String(scopedNode)
	}

	return ""
}

// extractTypeList extracts a list of type names from a super_interfaces or extends_interfaces node.
// Java: class Foo implements Bar, Baz -> ["Bar", "Baz"]
// Java: interface Foo extends Bar, Baz -> ["Bar", "Baz"]
func (jv *JavaVisitor) extractTypeList(tsNode *tree_sitter.Node) []string {
	if tsNode == nil {
		return nil
	}

	var types []string

	// Look for type_list child which contains the actual types
	typeListNode := jv.translate.TreeChildByKind(tsNode, "type_list")
	if typeListNode != nil {
		tsNode = typeListNode
	}

	// Iterate through children looking for type nodes
	for i := uint(0); i < tsNode.ChildCount(); i++ {
		child := tsNode.Child(i)
		if !child.IsNamed() {
			continue
		}

		kind := child.Kind()
		var typeName string

		switch kind {
		case "type_identifier", "identifier":
			typeName = jv.translate.String(child)
		case "generic_type":
			typeName = jv.extractTypeNameFromGeneric(child)
		case "scoped_identifier":
			typeName = jv.translate.String(child)
		}

		if typeName != "" {
			types = append(types, typeName)
		}
	}

	return types
}

// HasSpecialName returns false for Java - no special naming conventions like C#
func (jv *JavaVisitor) HasSpecialName(kind string) bool {
	return false
}

// GetName is not implemented for Java visitor
func (jv *JavaVisitor) GetName(tsNode *tree_sitter.Node) string {
	jv.logger.Error("GetName not implemented for Java visitor")
	return ""
}
