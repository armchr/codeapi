package parse

import (
	"context"

	"github.com/armchr/codeapi/internal/model/ast"
	tree_sitter "github.com/tree-sitter/go-tree-sitter"
	"go.uber.org/zap"
)

// CSharpVisitor handles traversal of C# syntax trees
type CSharpVisitor struct {
	translate *TranslateFromSyntaxTree
	logger    *zap.Logger
}

// NewCSharpVisitor creates a new C# visitor instance
func NewCSharpVisitor(logger *zap.Logger, ts *TranslateFromSyntaxTree) *CSharpVisitor {
	return &CSharpVisitor{
		translate: ts,
		logger:    logger,
	}
}

func (cv *CSharpVisitor) HasSpecialName(kind string) bool {
	switch kind {
	case "parameter":
		return true
	}

	return false
}

func (cv *CSharpVisitor) GetName(tsNode *tree_sitter.Node) string {
	if tsNode == nil {
		return ""
	}
	switch tsNode.Kind() {
	case "parameter":
		// the first identifier is the parameter name
		identNode := cv.translate.TreeChildByKind(tsNode, "identifier")
		if identNode != nil {
			return cv.translate.GetTreeNodeName(identNode)
		}
	}

	return cv.translate.String(tsNode)
}

// TraverseNode traverses a C# syntax tree node and returns the created AST node ID
func (cv *CSharpVisitor) TraverseNode(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	if tsNode == nil {
		return ast.InvalidNodeID
	}

	switch tsNode.Kind() {
	// Compilation unit (root)
	case "compilation_unit":
		return cv.handleCompilationUnit(ctx, tsNode)
	// Namespace declarations
	case "namespace_declaration", "file_scoped_namespace_declaration":
		return cv.handleNamespaceDeclaration(ctx, tsNode, scopeID)
	// Using directives (imports)
	case "using_directive":
		return cv.handleUsingDirective(ctx, tsNode, scopeID)
	// Type declarations
	case "class_declaration":
		return cv.handleClassDeclaration(ctx, tsNode, scopeID)
	case "struct_declaration":
		return cv.handleClassDeclaration(ctx, tsNode, scopeID)
	case "interface_declaration":
		return cv.handleInterfaceDeclaration(ctx, tsNode, scopeID)
	case "record_declaration":
		return cv.handleClassDeclaration(ctx, tsNode, scopeID)
	// Member declarations
	case "method_declaration":
		return cv.handleMethodDeclaration(ctx, tsNode, scopeID)
	case "constructor_declaration":
		return cv.handleConstructorDeclaration(ctx, tsNode, scopeID)
	// Block
	case "block":
		return cv.translate.HandleBlock(ctx, tsNode, scopeID)
	// Statements
	case "if_statement":
		return cv.handleIfStatement(ctx, tsNode, scopeID)
	case "switch_statement":
		return cv.handleSwitchStatement(ctx, tsNode, scopeID)
	case "for_statement":
		return cv.handleForStatement(ctx, tsNode, scopeID)
	case "foreach_statement":
		return cv.handleForeachStatement(ctx, tsNode, scopeID)
	case "while_statement":
		return cv.handleWhileStatement(ctx, tsNode, scopeID)
	case "do_statement":
		return cv.handleDoStatement(ctx, tsNode, scopeID)
	case "return_statement":
		return cv.handleReturnStatement(ctx, tsNode, scopeID)
	case "local_declaration_statement":
		return cv.handleLocalDeclarationStatement(ctx, tsNode, scopeID)
	case "expression_statement":
		return cv.handleExpressionStatement(ctx, tsNode, scopeID)
	// Expressions
	case "invocation_expression":
		return cv.handleInvocationExpression(ctx, tsNode, scopeID)
	case "member_access_expression":
		return cv.handleMemberAccessExpression(ctx, tsNode, scopeID)
	case "assignment_expression":
		return cv.handleAssignmentExpression(ctx, tsNode, scopeID)
	case "object_creation_expression":
		return cv.handleObjectCreationExpression(ctx, tsNode, scopeID)
	case "identifier":
		return cv.translate.HandleIdentifier(ctx, tsNode, scopeID)

	default:
		// For unhandled node types, traverse children
		cv.translate.TraverseChildren(ctx, tsNode, scopeID)
		return ast.InvalidNodeID
	}
}

// handleCompilationUnit handles the root compilation_unit node
func (cv *CSharpVisitor) handleCompilationUnit(ctx context.Context, tsNode *tree_sitter.Node) ast.NodeID {
	// Find namespace declaration to get the module name
	namespaceName := ""
	namespaceNode := cv.translate.TreeChildByKind(tsNode, "file_scoped_namespace_declaration")
	if namespaceNode == nil {
		namespaceNode = cv.translate.TreeChildByKind(tsNode, "namespace_declaration")
	}

	if namespaceNode != nil {
		qualifiedName := cv.translate.TreeChildByKind(namespaceNode, "qualified_name")
		if qualifiedName != nil {
			namespaceName = cv.getQualifiedName(qualifiedName)
		} else {
			identNode := cv.translate.TreeChildByKind(namespaceNode, "identifier")
			if identNode != nil {
				namespaceName = cv.translate.GetTreeNodeName(identNode)
			}
		}
	}

	// Create module scope node
	moduleNode := ast.NewNode(
		cv.translate.NextNodeID(), ast.NodeTypeModuleScope, cv.translate.FileID,
		namespaceName, cv.translate.ToRange(tsNode), cv.translate.Version,
		ast.NodeID(cv.translate.FileID),
	)
	cv.translate.CodeGraph.CreateModuleScope(ctx, moduleNode)

	cv.translate.PushScope(false)
	defer cv.translate.PopScope(ctx, moduleNode.ID)

	// Traverse all children to handle using directives, type declarations, etc.
	childNodes := cv.translate.TraverseChildren(ctx, tsNode, moduleNode.ID)
	if len(childNodes) > 0 {
		cv.translate.CreateContainsRelations(ctx, moduleNode.ID, childNodes)
	}

	return moduleNode.ID
}

// handleNamespaceDeclaration handles namespace_declaration and file_scoped_namespace_declaration
func (cv *CSharpVisitor) handleNamespaceDeclaration(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	// For file-scoped namespaces, the members are siblings in compilation_unit
	// For block-scoped namespaces, traverse the declaration_list
	declList := cv.translate.TreeChildByKind(tsNode, "declaration_list")
	if declList != nil {
		cv.translate.TraverseChildren(ctx, declList, scopeID)
	}
	return ast.InvalidNodeID
}

// handleUsingDirective handles using directives (imports)
func (cv *CSharpVisitor) handleUsingDirective(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	// Get the namespace being imported
	qualifiedName := cv.translate.TreeChildByKind(tsNode, "qualified_name")
	importPath := ""
	symbolName := ""

	if qualifiedName != nil {
		importPath = cv.getQualifiedName(qualifiedName)
		// Symbol name is the last part of the qualified name
		symbolName = cv.getLastIdentifier(qualifiedName)
	} else {
		identNode := cv.translate.TreeChildByKind(tsNode, "identifier")
		if identNode != nil {
			importPath = cv.translate.GetTreeNodeName(identNode)
			symbolName = importPath
		}
	}

	if importPath == "" {
		return ast.InvalidNodeID
	}

	// Check for alias: using Alias = Namespace.Type
	aliasNode := cv.translate.TreeChildByKind(tsNode, "name_equals")
	if aliasNode != nil {
		aliasIdent := cv.translate.TreeChildByKind(aliasNode, "identifier")
		if aliasIdent != nil {
			symbolName = cv.translate.GetTreeNodeName(aliasIdent)
		}
	}

	// Create Import node
	importNode := ast.NewNode(
		cv.translate.NextNodeID(),
		ast.NodeTypeImport,
		cv.translate.FileID,
		symbolName,
		cv.translate.ToRange(tsNode),
		cv.translate.Version,
		scopeID,
	)

	importNode.MetaData = map[string]any{
		"importPath": importPath,
	}

	cv.translate.CodeGraph.CreateImport(ctx, importNode)
	cv.translate.CurrentScope.AddSymbol(NewSymbol(importNode))
	cv.translate.Nodes[importNode.ID] = importNode

	return importNode.ID
}

// handleClassDeclaration handles class, struct, and record declarations
func (cv *CSharpVisitor) handleClassDeclaration(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	nameNode := cv.translate.TreeChildByKind(tsNode, "identifier")
	className := ""
	if nameNode != nil {
		className = cv.translate.GetTreeNodeName(nameNode)
	}

	declList := cv.translate.TreeChildByKind(tsNode, "declaration_list")
	var members []*tree_sitter.Node
	var fields []*tree_sitter.Node

	if declList != nil {
		// Collect methods and fields from declaration_list
		for i := uint(0); i < declList.ChildCount(); i++ {
			child := declList.Child(i)
			if child == nil {
				continue
			}
			switch child.Kind() {
			case "method_declaration", "constructor_declaration":
				members = append(members, child)
			case "field_declaration":
				fields = append(fields, child)
			}
		}
	}

	return cv.translate.HandleClass(ctx, scopeID, tsNode, className, members, fields)
}

// handleInterfaceDeclaration handles interface declarations
func (cv *CSharpVisitor) handleInterfaceDeclaration(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	nameNode := cv.translate.TreeChildByKind(tsNode, "identifier")
	interfaceName := ""
	if nameNode != nil {
		interfaceName = cv.translate.GetTreeNodeName(nameNode)
	}

	declList := cv.translate.TreeChildByKind(tsNode, "declaration_list")
	var methods []*tree_sitter.Node

	if declList != nil {
		// Collect method declarations from declaration_list
		methods = cv.translate.TreeChildrenByKind(declList, "method_declaration")
	}

	return cv.translate.HandleClass(ctx, scopeID, tsNode, interfaceName, methods, nil)
}

// handleMethodDeclaration handles method declarations
func (cv *CSharpVisitor) handleMethodDeclaration(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	nameNode := cv.translate.TreeChildByKind(tsNode, "identifier")
	methodName := ""
	if nameNode != nil {
		methodName = cv.translate.GetTreeNodeName(nameNode)
	}

	paramListNode := cv.translate.TreeChildByKind(tsNode, "parameter_list")
	bodyNode := cv.translate.TreeChildByKind(tsNode, "block")

	var params []*tree_sitter.Node
	if paramListNode != nil {
		params = cv.translate.TreeChildrenByKind(paramListNode, "parameter")
	}

	// For interface methods without body, bodyNode will be nil
	return cv.translate.CreateFunction(ctx, scopeID, tsNode, methodName, params, bodyNode)
}

// handleConstructorDeclaration handles constructor declarations
func (cv *CSharpVisitor) handleConstructorDeclaration(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	nameNode := cv.translate.TreeChildByKind(tsNode, "identifier")
	ctorName := ""
	if nameNode != nil {
		ctorName = cv.translate.GetTreeNodeName(nameNode)
	}

	paramListNode := cv.translate.TreeChildByKind(tsNode, "parameter_list")
	bodyNode := cv.translate.TreeChildByKind(tsNode, "block")

	var params []*tree_sitter.Node
	if paramListNode != nil {
		params = cv.translate.TreeChildrenByKind(paramListNode, "parameter")
	}

	return cv.translate.CreateFunction(ctx, scopeID, tsNode, ctorName, params, bodyNode)
}

// handleIfStatement handles if statements
func (cv *CSharpVisitor) handleIfStatement(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	var conditions []*tree_sitter.Node
	var branches []*tree_sitter.Node

	// Get condition - it's inside parentheses after 'if'
	// The condition is typically a named child (expression)
	for i := uint(0); i < tsNode.ChildCount(); i++ {
		child := tsNode.Child(i)
		if child == nil {
			continue
		}
		kind := child.Kind()
		// Skip keywords and punctuation
		if kind == "if" || kind == "(" || kind == ")" || kind == "else" {
			continue
		}
		// First expression-like child is the condition
		if len(conditions) == 0 && child.IsNamed() && kind != "block" && kind != "if_statement" {
			conditions = append(conditions, child)
			continue
		}
		// Block or statement is a branch
		if kind == "block" {
			branches = append(branches, child)
		}
		// Handle else if
		if kind == "if_statement" {
			// Recursively get conditions and branches from nested if
			nestedConditions, nestedBranches := cv.getIfConditionsAndBranches(child)
			conditions = append(conditions, nestedConditions...)
			branches = append(branches, nestedBranches...)
		}
	}

	return cv.translate.HandleConditional(ctx, tsNode, conditions, branches, scopeID)
}

// getIfConditionsAndBranches extracts conditions and branches from an if statement
func (cv *CSharpVisitor) getIfConditionsAndBranches(tsNode *tree_sitter.Node) ([]*tree_sitter.Node, []*tree_sitter.Node) {
	var conditions []*tree_sitter.Node
	var branches []*tree_sitter.Node

	for i := uint(0); i < tsNode.ChildCount(); i++ {
		child := tsNode.Child(i)
		if child == nil {
			continue
		}
		kind := child.Kind()
		if kind == "if" || kind == "(" || kind == ")" || kind == "else" {
			continue
		}
		if len(conditions) == 0 && child.IsNamed() && kind != "block" && kind != "if_statement" {
			conditions = append(conditions, child)
			continue
		}
		if kind == "block" {
			branches = append(branches, child)
		}
		if kind == "if_statement" {
			nestedConditions, nestedBranches := cv.getIfConditionsAndBranches(child)
			conditions = append(conditions, nestedConditions...)
			branches = append(branches, nestedBranches...)
		}
	}

	return conditions, branches
}

// handleSwitchStatement handles switch statements
func (cv *CSharpVisitor) handleSwitchStatement(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	var conditions []*tree_sitter.Node
	var branches []*tree_sitter.Node

	// Get the switch expression (value being switched on)
	for i := uint(0); i < tsNode.ChildCount(); i++ {
		child := tsNode.Child(i)
		if child == nil {
			continue
		}
		kind := child.Kind()
		if kind == "switch" || kind == "(" || kind == ")" {
			continue
		}
		// First expression is the switch value
		if len(conditions) == 0 && child.IsNamed() && kind != "switch_body" {
			conditions = append(conditions, child)
			continue
		}
		// Switch body contains cases
		if kind == "switch_body" {
			caseSections := cv.translate.TreeChildrenByKind(child, "switch_section")
			for _, section := range caseSections {
				// Get case labels as conditions
				labels := cv.translate.TreeChildrenByKind(section, "case_switch_label")
				for _, label := range labels {
					conditions = append(conditions, label)
				}
				// Get statement list as branch
				stmtList := cv.translate.TreeChildrenByKind(section, "statement")
				if len(stmtList) > 0 {
					// Use the section itself as branch container
					branches = append(branches, section)
				}
			}
		}
	}

	return cv.translate.HandleConditional(ctx, tsNode, conditions, branches, scopeID)
}

// handleForStatement handles for statements
func (cv *CSharpVisitor) handleForStatement(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	var inits []*tree_sitter.Node

	// Get initializer, condition, and incrementor
	for i := uint(0); i < tsNode.ChildCount(); i++ {
		child := tsNode.Child(i)
		if child == nil {
			continue
		}
		kind := child.Kind()
		if kind == "for" || kind == "(" || kind == ")" || kind == ";" {
			continue
		}
		if kind == "block" {
			continue
		}
		// Collect initialization, condition, and update as inits
		if child.IsNamed() {
			inits = append(inits, child)
		}
	}

	bodyNode := cv.translate.TreeChildByKind(tsNode, "block")

	cv.translate.PushScope(false)
	defer cv.translate.PopScope(ctx, ast.InvalidNodeID)

	initCondID := ast.InvalidNodeID
	if len(inits) > 0 {
		initCondID = cv.translate.HandleRhsExprsWithFakeVariable(ctx, "__init__", inits, scopeID, nil)
	}

	if bodyNode == nil {
		return ast.InvalidNodeID
	}
	return cv.translate.HandleLoop(ctx, tsNode, ast.InvalidNodeID, initCondID, bodyNode, scopeID)
}

// handleForeachStatement handles foreach statements
func (cv *CSharpVisitor) handleForeachStatement(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	// Get the iteration variable
	varIdent := cv.translate.TreeChildByKind(tsNode, "identifier")
	// Get the collection expression (after 'in')
	var collectionNode *tree_sitter.Node
	inFound := false
	for i := uint(0); i < tsNode.ChildCount(); i++ {
		child := tsNode.Child(i)
		if child == nil {
			continue
		}
		if child.Kind() == "in" {
			inFound = true
			continue
		}
		if inFound && child.IsNamed() && child.Kind() != "block" {
			collectionNode = child
			break
		}
	}

	bodyNode := cv.translate.TreeChildByKind(tsNode, "block")

	var inits []*tree_sitter.Node
	if varIdent != nil {
		inits = append(inits, varIdent)
	}
	if collectionNode != nil {
		inits = append(inits, collectionNode)
	}

	cv.translate.PushScope(false)
	defer cv.translate.PopScope(ctx, ast.InvalidNodeID)

	initCondID := ast.InvalidNodeID
	if len(inits) > 0 {
		initCondID = cv.translate.HandleRhsExprsWithFakeVariable(ctx, "__foreach__", inits, scopeID, nil)
	}

	if bodyNode == nil {
		return ast.InvalidNodeID
	}
	return cv.translate.HandleLoop(ctx, tsNode, ast.InvalidNodeID, initCondID, bodyNode, scopeID)
}

// handleWhileStatement handles while statements
func (cv *CSharpVisitor) handleWhileStatement(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	var conditionNode *tree_sitter.Node
	for i := uint(0); i < tsNode.ChildCount(); i++ {
		child := tsNode.Child(i)
		if child == nil {
			continue
		}
		kind := child.Kind()
		if kind == "while" || kind == "(" || kind == ")" {
			continue
		}
		if child.IsNamed() && kind != "block" {
			conditionNode = child
			break
		}
	}

	bodyNode := cv.translate.TreeChildByKind(tsNode, "block")

	cv.translate.PushScope(false)
	defer cv.translate.PopScope(ctx, ast.InvalidNodeID)

	condID := ast.InvalidNodeID
	if conditionNode != nil {
		condID = cv.translate.HandleRhsWithFakeVariable(ctx, "__while__", conditionNode, scopeID, nil)
	}

	if bodyNode == nil {
		return ast.InvalidNodeID
	}
	return cv.translate.HandleLoop(ctx, tsNode, ast.InvalidNodeID, condID, bodyNode, scopeID)
}

// handleDoStatement handles do-while statements
func (cv *CSharpVisitor) handleDoStatement(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	bodyNode := cv.translate.TreeChildByKind(tsNode, "block")

	// Get condition after 'while'
	var conditionNode *tree_sitter.Node
	whileFound := false
	for i := uint(0); i < tsNode.ChildCount(); i++ {
		child := tsNode.Child(i)
		if child == nil {
			continue
		}
		if child.Kind() == "while" {
			whileFound = true
			continue
		}
		if whileFound && child.IsNamed() {
			conditionNode = child
			break
		}
	}

	cv.translate.PushScope(false)
	defer cv.translate.PopScope(ctx, ast.InvalidNodeID)

	condID := ast.InvalidNodeID
	if conditionNode != nil {
		condID = cv.translate.HandleRhsWithFakeVariable(ctx, "__do_while__", conditionNode, scopeID, nil)
	}

	if bodyNode == nil {
		return ast.InvalidNodeID
	}
	return cv.translate.HandleLoop(ctx, tsNode, ast.InvalidNodeID, condID, bodyNode, scopeID)
}

// handleReturnStatement handles return statements
func (cv *CSharpVisitor) handleReturnStatement(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	// Get the return expression (if any)
	for i := uint(0); i < tsNode.ChildCount(); i++ {
		child := tsNode.Child(i)
		if child == nil {
			continue
		}
		if child.Kind() == "return" || child.Kind() == ";" {
			continue
		}
		if child.IsNamed() {
			return cv.translate.HandleReturn(ctx, child, scopeID)
		}
	}
	return ast.InvalidNodeID
}

// handleLocalDeclarationStatement handles local variable declarations
func (cv *CSharpVisitor) handleLocalDeclarationStatement(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	varDecl := cv.translate.TreeChildByKind(tsNode, "variable_declaration")
	if varDecl == nil {
		return ast.InvalidNodeID
	}

	declarators := cv.translate.TreeChildrenByKind(varDecl, "variable_declarator")
	for _, decl := range declarators {
		nameNode := cv.translate.TreeChildByKind(decl, "identifier")
		// Get initializer (after '=')
		var initNode *tree_sitter.Node
		for i := uint(0); i < decl.ChildCount(); i++ {
			child := decl.Child(i)
			if child == nil {
				continue
			}
			if child.Kind() == "=" {
				// Next named child is the initializer
				for j := i + 1; j < decl.ChildCount(); j++ {
					nextChild := decl.Child(j)
					if nextChild != nil && nextChild.IsNamed() {
						initNode = nextChild
						break
					}
				}
				break
			}
			// Also check for equals_value_clause
			if child.Kind() == "equals_value_clause" {
				for k := uint(0); k < child.ChildCount(); k++ {
					grandChild := child.Child(k)
					if grandChild != nil && grandChild.IsNamed() && grandChild.Kind() != "=" {
						initNode = grandChild
						break
					}
				}
				break
			}
		}

		if nameNode != nil && initNode != nil {
			cv.translate.HandleAssignment(ctx, decl, nameNode, initNode, scopeID)
		}
	}

	return ast.InvalidNodeID
}

// handleExpressionStatement handles expression statements
func (cv *CSharpVisitor) handleExpressionStatement(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	// Get the expression child and traverse it
	for i := uint(0); i < tsNode.ChildCount(); i++ {
		child := tsNode.Child(i)
		if child != nil && child.IsNamed() {
			return cv.TraverseNode(ctx, child, scopeID)
		}
	}
	return ast.InvalidNodeID
}

// handleInvocationExpression handles method/function invocations
func (cv *CSharpVisitor) handleInvocationExpression(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	// Get the function/method being called (first child)
	var functionNode *tree_sitter.Node
	var argumentsNode *tree_sitter.Node

	for i := uint(0); i < tsNode.ChildCount(); i++ {
		child := tsNode.Child(i)
		if child == nil {
			continue
		}
		kind := child.Kind()
		if kind == "argument_list" {
			argumentsNode = child
		} else if child.IsNamed() && functionNode == nil {
			functionNode = child
		}
	}

	var args []*tree_sitter.Node
	if argumentsNode != nil {
		arguments := cv.translate.TreeChildrenByKind(argumentsNode, "argument")
		for _, arg := range arguments {
			// Get the expression from the argument
			for j := uint(0); j < arg.ChildCount(); j++ {
				argChild := arg.Child(j)
				if argChild != nil && argChild.IsNamed() {
					args = append(args, argChild)
					break
				}
			}
		}
	}

	fnNameNodeID := cv.translate.HandleRhsWithFakeVariable(ctx, "__fn__", functionNode, scopeID, nil)
	return cv.translate.HandleCall(ctx, fnNameNodeID, args, scopeID, cv.translate.ToRange(tsNode))
}

// handleMemberAccessExpression handles member access like obj.Member
func (cv *CSharpVisitor) handleMemberAccessExpression(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	var names []*tree_sitter.Node

	// Collect all identifiers in the chain
	cv.collectMemberAccessNames(tsNode, &names)

	resolvedNodeId := cv.translate.ResolveNameChain(ctx, names, scopeID)
	if cv.translate.CurrentScope.IsRhs() && resolvedNodeId != ast.InvalidNodeID {
		cv.translate.CurrentScope.AddRhsVar(resolvedNodeId)
	}
	return resolvedNodeId
}

// collectMemberAccessNames recursively collects names from member access expressions
func (cv *CSharpVisitor) collectMemberAccessNames(tsNode *tree_sitter.Node, names *[]*tree_sitter.Node) {
	if tsNode == nil {
		return
	}

	if tsNode.Kind() == "identifier" {
		*names = append(*names, tsNode)
		return
	}

	if tsNode.Kind() == "member_access_expression" {
		for i := uint(0); i < tsNode.ChildCount(); i++ {
			child := tsNode.Child(i)
			if child == nil {
				continue
			}
			if child.Kind() == "." {
				continue
			}
			cv.collectMemberAccessNames(child, names)
		}
	}
}

// handleAssignmentExpression handles assignment expressions
func (cv *CSharpVisitor) handleAssignmentExpression(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	var leftNode *tree_sitter.Node
	var rightNode *tree_sitter.Node

	for i := uint(0); i < tsNode.ChildCount(); i++ {
		child := tsNode.Child(i)
		if child == nil {
			continue
		}
		if child.Kind() == "=" || child.Kind() == "+=" || child.Kind() == "-=" ||
			child.Kind() == "*=" || child.Kind() == "/=" {
			continue
		}
		if child.IsNamed() {
			if leftNode == nil {
				leftNode = child
			} else if rightNode == nil {
				rightNode = child
				break
			}
		}
	}

	if leftNode == nil || rightNode == nil {
		return ast.InvalidNodeID
	}

	return cv.translate.HandleAssignment(ctx, tsNode, leftNode, rightNode, scopeID)
}

// handleObjectCreationExpression handles new Type(args) expressions
func (cv *CSharpVisitor) handleObjectCreationExpression(ctx context.Context, tsNode *tree_sitter.Node, scopeID ast.NodeID) ast.NodeID {
	// Get the type being instantiated
	var typeNode *tree_sitter.Node
	var argumentsNode *tree_sitter.Node

	for i := uint(0); i < tsNode.ChildCount(); i++ {
		child := tsNode.Child(i)
		if child == nil {
			continue
		}
		kind := child.Kind()
		if kind == "new" {
			continue
		}
		if kind == "argument_list" {
			argumentsNode = child
		} else if child.IsNamed() && typeNode == nil {
			typeNode = child
		}
	}

	var args []*tree_sitter.Node
	if argumentsNode != nil {
		arguments := cv.translate.TreeChildrenByKind(argumentsNode, "argument")
		for _, arg := range arguments {
			for j := uint(0); j < arg.ChildCount(); j++ {
				argChild := arg.Child(j)
				if argChild != nil && argChild.IsNamed() {
					args = append(args, argChild)
					break
				}
			}
		}
	}

	// Treat object creation as a call to the constructor
	fnNameNodeID := cv.translate.HandleRhsWithFakeVariable(ctx, "__new__", typeNode, scopeID, nil)
	return cv.translate.HandleCall(ctx, fnNameNodeID, args, scopeID, cv.translate.ToRange(tsNode))
}

// Helper functions

// getQualifiedName extracts the full qualified name as a string
func (cv *CSharpVisitor) getQualifiedName(tsNode *tree_sitter.Node) string {
	if tsNode == nil {
		return ""
	}
	return cv.translate.String(tsNode)
}

// getLastIdentifier gets the last identifier from a qualified name
func (cv *CSharpVisitor) getLastIdentifier(tsNode *tree_sitter.Node) string {
	if tsNode == nil {
		return ""
	}

	// For qualified_name, find the rightmost identifier
	if tsNode.Kind() == "qualified_name" {
		// The last named child should be an identifier
		for i := int(tsNode.ChildCount()) - 1; i >= 0; i-- {
			child := tsNode.Child(uint(i))
			if child != nil && child.Kind() == "identifier" {
				return cv.translate.GetTreeNodeName(child)
			}
		}
	}

	if tsNode.Kind() == "identifier" {
		return cv.translate.GetTreeNodeName(tsNode)
	}

	return ""
}

// HasSpecialName returns true for C# node kinds that have special naming conventions
func (cv *CSharpVisitor) HasSpecialName(kind string) bool {
	// C# has special naming for method declarations where the first identifier
	// may be the return type, not the method name
	switch kind {
	case "method_declaration", "constructor_declaration":
		return true
	default:
		return false
	}
}

// GetName extracts the proper name from a C# node considering special naming conventions
func (cv *CSharpVisitor) GetName(tsNode *tree_sitter.Node) string {
	if tsNode == nil {
		return ""
	}

	kind := tsNode.Kind()
	switch kind {
	case "method_declaration":
		// In C# method_declaration, the name is the identifier after the return type
		// Structure: [modifiers] return_type identifier parameter_list [block]
		identifiers := cv.translate.TreeChildrenByKind(tsNode, "identifier")
		if len(identifiers) >= 2 {
			// Second identifier is the method name (first is often return type)
			return cv.translate.GetTreeNodeName(identifiers[1])
		} else if len(identifiers) == 1 {
			return cv.translate.GetTreeNodeName(identifiers[0])
		}
	case "constructor_declaration":
		// Constructor name is the identifier
		nameNode := cv.translate.TreeChildByKind(tsNode, "identifier")
		if nameNode != nil {
			return cv.translate.GetTreeNodeName(nameNode)
		}
	}

	return ""
}
