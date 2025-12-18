package chunk

import (
	"github.com/armchr/codeapi/internal/model"
	"github.com/armchr/codeapi/pkg/lsp/base"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"
	"go.uber.org/zap"
)

// ChunkVisitor implements SyntaxTreeVisitor for hierarchical code chunking
// It creates chunks at file, class, function, and block levels
type ChunkVisitor struct {
	logger              *zap.Logger
	language            string
	filePath            string
	sourceCode          []byte
	chunks              []*model.CodeChunk
	currentFile         *model.CodeChunk
	currentClass        *model.CodeChunk
	moduleName          string
	minConditionalLines int
	minLoopLines        int
}

// NewChunkVisitor creates a new chunk visitor
func NewChunkVisitor(logger *zap.Logger, language, filePath string, sourceCode []byte, minConditionalLines, minLoopLines int) *ChunkVisitor {
	return &ChunkVisitor{
		logger:              logger,
		language:            language,
		filePath:            filePath,
		sourceCode:          sourceCode,
		chunks:              make([]*model.CodeChunk, 0),
		minConditionalLines: minConditionalLines,
		minLoopLines:        minLoopLines,
	}
}

// GetChunks returns all collected code chunks
func (cv *ChunkVisitor) GetChunks() []*model.CodeChunk {
	return cv.chunks
}

// TraverseNode is the main entry point for traversing syntax tree nodes
func (cv *ChunkVisitor) TraverseNode(ctx context.Context, tsNode *tree_sitter.Node, scopeID any) any {
	if tsNode == nil {
		return nil
	}

	kind := tsNode.Kind()

	// Dispatch to language-specific handlers
	switch cv.language {
	case "go":
		return cv.traverseGoNode(ctx, tsNode, kind)
	case "python":
		return cv.traversePythonNode(ctx, tsNode, kind)
	case "java":
		return cv.traverseJavaNode(ctx, tsNode, kind)
	case "javascript", "typescript":
		return cv.traverseJavaScriptNode(ctx, tsNode, kind)
	default:
		// Fallback: traverse children
		cv.traverseChildren(ctx, tsNode)
		return nil
	}
}

// Go-specific node handling
func (cv *ChunkVisitor) traverseGoNode(ctx context.Context, tsNode *tree_sitter.Node, kind string) any {
	// Log every node kind we encounter for debugging
	/*
		if kind == "if_statement" || kind == "for_statement" || kind == "switch_statement" || kind == "type_switch_statement" {
			cv.logger.Debug("traverseGoNode encountered control flow",
				zap.String("kind", kind),
				zap.String("file", cv.filePath),
				zap.Int("line", int(tsNode.StartPosition().Row)))
		}
	*/

	switch kind {
	case "source_file":
		return cv.handleSourceFile(ctx, tsNode)
	case "package_clause":
		cv.extractPackageName(tsNode)
	case "function_declaration":
		return cv.handleFunctionDeclaration(ctx, tsNode, false)
	case "method_declaration":
		return cv.handleFunctionDeclaration(ctx, tsNode, true)
	case "type_declaration":
		cv.handleTypeDeclaration(ctx, tsNode)
	case "if_statement":
		return cv.handleConditional(ctx, tsNode, "if")
	case "switch_statement", "type_switch_statement":
		return cv.handleConditional(ctx, tsNode, "switch")
	case "for_statement":
		return cv.handleLoop(ctx, tsNode, "for")
	}

	cv.traverseChildren(ctx, tsNode)
	return nil
}

// Python-specific node handling
func (cv *ChunkVisitor) traversePythonNode(ctx context.Context, tsNode *tree_sitter.Node, kind string) any {
	switch kind {
	case "module":
		return cv.handleSourceFile(ctx, tsNode)
	case "class_definition":
		return cv.handleClassDefinition(ctx, tsNode)
	case "function_definition":
		return cv.handlePythonFunction(ctx, tsNode)
	case "if_statement":
		return cv.handleConditional(ctx, tsNode, "if")
	case "match_statement":
		return cv.handleConditional(ctx, tsNode, "match")
	case "for_statement":
		return cv.handleLoop(ctx, tsNode, "for")
	case "while_statement":
		return cv.handleLoop(ctx, tsNode, "while")
	}

	cv.traverseChildren(ctx, tsNode)
	return nil
}

// Java-specific node handling
func (cv *ChunkVisitor) traverseJavaNode(ctx context.Context, tsNode *tree_sitter.Node, kind string) any {
	switch kind {
	case "program":
		return cv.handleSourceFile(ctx, tsNode)
	case "package_declaration":
		cv.extractJavaPackageName(tsNode)
	case "class_declaration", "interface_declaration":
		return cv.handleJavaClass(ctx, tsNode)
	case "method_declaration":
		return cv.handleJavaMethod(ctx, tsNode)
	case "if_statement":
		return cv.handleConditional(ctx, tsNode, "if")
	case "switch_expression":
		return cv.handleConditional(ctx, tsNode, "switch")
	case "for_statement", "enhanced_for_statement":
		return cv.handleLoop(ctx, tsNode, "for")
	case "while_statement":
		return cv.handleLoop(ctx, tsNode, "while")
	case "do_statement":
		return cv.handleLoop(ctx, tsNode, "do-while")
	}

	cv.traverseChildren(ctx, tsNode)
	return nil
}

// JavaScript/TypeScript-specific node handling
func (cv *ChunkVisitor) traverseJavaScriptNode(ctx context.Context, tsNode *tree_sitter.Node, kind string) any {
	switch kind {
	case "program":
		return cv.handleSourceFile(ctx, tsNode)
	case "class_declaration":
		return cv.handleJSClass(ctx, tsNode)
	case "function_declaration":
		return cv.handleJSFunction(ctx, tsNode)
	case "method_definition":
		return cv.handleJSMethod(ctx, tsNode)
	case "if_statement":
		return cv.handleConditional(ctx, tsNode, "if")
	case "switch_statement":
		return cv.handleConditional(ctx, tsNode, "switch")
	case "for_statement", "for_in_statement":
		return cv.handleLoop(ctx, tsNode, "for")
	case "while_statement":
		return cv.handleLoop(ctx, tsNode, "while")
	case "do_statement":
		return cv.handleLoop(ctx, tsNode, "do-while")
	}

	cv.traverseChildren(ctx, tsNode)
	return nil
}

// handleSourceFile creates a file-level chunk
func (cv *ChunkVisitor) handleSourceFile(ctx context.Context, tsNode *tree_sitter.Node) any {
	content := cv.getNodeText(tsNode)
	rng := cv.toRange(tsNode)

	chunkID := cv.generateChunkID(cv.filePath, "file", 0)

	chunk := model.NewCodeChunk(
		chunkID,
		model.ChunkTypeFile,
		1,
		content,
		cv.language,
		cv.filePath,
		rng,
	).WithName(cv.filePath)

	cv.currentFile = chunk
	cv.chunks = append(cv.chunks, chunk)

	cv.traverseChildren(ctx, tsNode)
	return chunk
}

// handleFunctionDeclaration handles Go functions and methods
func (cv *ChunkVisitor) handleFunctionDeclaration(ctx context.Context, tsNode *tree_sitter.Node, isMethod bool) any {
	nameNode := cv.getChildByFieldName(tsNode, "name")
	if nameNode == nil {
		return nil
	}

	name := cv.getNodeText(nameNode)
	content := cv.getNodeText(tsNode)
	signature := cv.extractGoFunctionSignature(tsNode)
	docstring := cv.extractGoDocstring(tsNode)

	chunkID := cv.generateChunkID(cv.filePath, name, tsNode.StartPosition().Row)

	parentID := ""
	className := ""
	if cv.currentClass != nil {
		parentID = cv.currentClass.ID
		className = cv.currentClass.Name
	} else if cv.currentFile != nil {
		parentID = cv.currentFile.ID
	}

	chunk := model.NewCodeChunk(
		chunkID,
		model.ChunkTypeFunction,
		3,
		content,
		cv.language,
		cv.filePath,
		cv.toRange(tsNode),
	).WithParent(parentID).
		WithName(name).
		WithSignature(signature).
		WithDocstring(docstring).
		WithContext(cv.moduleName, className)

	cv.chunks = append(cv.chunks, chunk)

	// Traverse function body to find conditionals and loops
	cv.traverseChildren(ctx, tsNode)

	return chunk
}

// handleClassDefinition handles Python class definitions
func (cv *ChunkVisitor) handleClassDefinition(ctx context.Context, tsNode *tree_sitter.Node) any {
	nameNode := cv.getChildByFieldName(tsNode, "name")
	if nameNode == nil {
		return nil
	}

	name := cv.getNodeText(nameNode)
	content := cv.getNodeText(tsNode)
	docstring := cv.extractPythonDocstring(tsNode)

	chunkID := cv.generateChunkID(cv.filePath, name, tsNode.StartPosition().Row)

	parentID := ""
	if cv.currentFile != nil {
		parentID = cv.currentFile.ID
	}

	chunk := model.NewCodeChunk(
		chunkID,
		model.ChunkTypeClass,
		2,
		content,
		cv.language,
		cv.filePath,
		cv.toRange(tsNode),
	).WithParent(parentID).
		WithName(name).
		WithDocstring(docstring).
		WithContext(cv.moduleName, "")

	oldClass := cv.currentClass
	cv.currentClass = chunk
	cv.chunks = append(cv.chunks, chunk)

	cv.traverseChildren(ctx, tsNode)

	cv.currentClass = oldClass
	return chunk
}

// handlePythonFunction handles Python function/method definitions
func (cv *ChunkVisitor) handlePythonFunction(ctx context.Context, tsNode *tree_sitter.Node) any {
	nameNode := cv.getChildByFieldName(tsNode, "name")
	if nameNode == nil {
		return nil
	}

	name := cv.getNodeText(nameNode)
	content := cv.getNodeText(tsNode)
	signature := cv.extractPythonFunctionSignature(tsNode)
	docstring := cv.extractPythonDocstring(tsNode)

	chunkID := cv.generateChunkID(cv.filePath, name, tsNode.StartPosition().Row)

	parentID := ""
	className := ""
	if cv.currentClass != nil {
		parentID = cv.currentClass.ID
		className = cv.currentClass.Name
	} else if cv.currentFile != nil {
		parentID = cv.currentFile.ID
	}

	chunk := model.NewCodeChunk(
		chunkID,
		model.ChunkTypeFunction,
		3,
		content,
		cv.language,
		cv.filePath,
		cv.toRange(tsNode),
	).WithParent(parentID).
		WithName(name).
		WithSignature(signature).
		WithDocstring(docstring).
		WithContext(cv.moduleName, className)

	cv.chunks = append(cv.chunks, chunk)

	// Traverse function body to find conditionals and loops
	cv.traverseChildren(ctx, tsNode)

	return chunk
}

// handleJavaClass handles Java class/interface declarations
func (cv *ChunkVisitor) handleJavaClass(ctx context.Context, tsNode *tree_sitter.Node) any {
	nameNode := cv.getChildByFieldName(tsNode, "name")
	if nameNode == nil {
		return nil
	}

	name := cv.getNodeText(nameNode)
	content := cv.getNodeText(tsNode)

	chunkID := cv.generateChunkID(cv.filePath, name, tsNode.StartPosition().Row)

	parentID := ""
	if cv.currentFile != nil {
		parentID = cv.currentFile.ID
	}

	chunk := model.NewCodeChunk(
		chunkID,
		model.ChunkTypeClass,
		2,
		content,
		cv.language,
		cv.filePath,
		cv.toRange(tsNode),
	).WithParent(parentID).
		WithName(name).
		WithContext(cv.moduleName, "")

	oldClass := cv.currentClass
	cv.currentClass = chunk
	cv.chunks = append(cv.chunks, chunk)

	cv.traverseChildren(ctx, tsNode)

	cv.currentClass = oldClass
	return chunk
}

// handleJavaMethod handles Java method declarations
func (cv *ChunkVisitor) handleJavaMethod(ctx context.Context, tsNode *tree_sitter.Node) any {
	nameNode := cv.getChildByFieldName(tsNode, "name")
	if nameNode == nil {
		return nil
	}

	name := cv.getNodeText(nameNode)
	content := cv.getNodeText(tsNode)
	signature := cv.extractJavaMethodSignature(tsNode)

	chunkID := cv.generateChunkID(cv.filePath, name, tsNode.StartPosition().Row)

	parentID := ""
	className := ""
	if cv.currentClass != nil {
		parentID = cv.currentClass.ID
		className = cv.currentClass.Name
	} else if cv.currentFile != nil {
		parentID = cv.currentFile.ID
	}

	chunk := model.NewCodeChunk(
		chunkID,
		model.ChunkTypeFunction,
		3,
		content,
		cv.language,
		cv.filePath,
		cv.toRange(tsNode),
	).WithParent(parentID).
		WithName(name).
		WithSignature(signature).
		WithContext(cv.moduleName, className)

	cv.chunks = append(cv.chunks, chunk)

	// Traverse body to find conditionals and loops
	cv.traverseChildren(ctx, tsNode)

	return chunk
}

// handleJSClass handles JavaScript/TypeScript class declarations
func (cv *ChunkVisitor) handleJSClass(ctx context.Context, tsNode *tree_sitter.Node) any {
	nameNode := cv.getChildByFieldName(tsNode, "name")
	if nameNode == nil {
		return nil
	}

	name := cv.getNodeText(nameNode)
	content := cv.getNodeText(tsNode)

	chunkID := cv.generateChunkID(cv.filePath, name, tsNode.StartPosition().Row)

	parentID := ""
	if cv.currentFile != nil {
		parentID = cv.currentFile.ID
	}

	chunk := model.NewCodeChunk(
		chunkID,
		model.ChunkTypeClass,
		2,
		content,
		cv.language,
		cv.filePath,
		cv.toRange(tsNode),
	).WithParent(parentID).
		WithName(name).
		WithContext(cv.moduleName, "")

	oldClass := cv.currentClass
	cv.currentClass = chunk
	cv.chunks = append(cv.chunks, chunk)

	cv.traverseChildren(ctx, tsNode)

	cv.currentClass = oldClass
	return chunk
}

// handleJSFunction handles JavaScript/TypeScript function declarations
func (cv *ChunkVisitor) handleJSFunction(ctx context.Context, tsNode *tree_sitter.Node) any {
	nameNode := cv.getChildByFieldName(tsNode, "name")
	if nameNode == nil {
		return nil
	}

	name := cv.getNodeText(nameNode)
	content := cv.getNodeText(tsNode)
	signature := cv.extractJSFunctionSignature(tsNode)

	chunkID := cv.generateChunkID(cv.filePath, name, tsNode.StartPosition().Row)

	parentID := ""
	if cv.currentFile != nil {
		parentID = cv.currentFile.ID
	}

	chunk := model.NewCodeChunk(
		chunkID,
		model.ChunkTypeFunction,
		3,
		content,
		cv.language,
		cv.filePath,
		cv.toRange(tsNode),
	).WithParent(parentID).
		WithName(name).
		WithSignature(signature).
		WithContext(cv.moduleName, "")

	cv.chunks = append(cv.chunks, chunk)

	// Traverse body to find conditionals and loops
	cv.traverseChildren(ctx, tsNode)

	return chunk
}

// handleJSMethod handles JavaScript/TypeScript method definitions
func (cv *ChunkVisitor) handleJSMethod(ctx context.Context, tsNode *tree_sitter.Node) any {
	nameNode := cv.getChildByFieldName(tsNode, "name")
	if nameNode == nil {
		return nil
	}

	name := cv.getNodeText(nameNode)
	content := cv.getNodeText(tsNode)
	signature := cv.extractJSFunctionSignature(tsNode)

	chunkID := cv.generateChunkID(cv.filePath, name, tsNode.StartPosition().Row)

	parentID := ""
	className := ""
	if cv.currentClass != nil {
		parentID = cv.currentClass.ID
		className = cv.currentClass.Name
	} else if cv.currentFile != nil {
		parentID = cv.currentFile.ID
	}

	chunk := model.NewCodeChunk(
		chunkID,
		model.ChunkTypeFunction,
		3,
		content,
		cv.language,
		cv.filePath,
		cv.toRange(tsNode),
	).WithParent(parentID).
		WithName(name).
		WithSignature(signature).
		WithContext(cv.moduleName, className)

	cv.chunks = append(cv.chunks, chunk)

	// Traverse body to find conditionals and loops
	cv.traverseChildren(ctx, tsNode)

	return chunk
}

// handleTypeDeclaration handles Go type declarations
func (cv *ChunkVisitor) handleTypeDeclaration(ctx context.Context, tsNode *tree_sitter.Node) {
	for i := uint(0); i < tsNode.ChildCount(); i++ {
		child := tsNode.Child(i)
		if child.Kind() == "type_spec" {
			nameNode := cv.getChildByFieldName(child, "name")
			typeNode := cv.getChildByFieldName(child, "type")

			if nameNode != nil && typeNode != nil {
				if typeNode.Kind() == "struct_type" || typeNode.Kind() == "interface_type" {
					cv.handleGoTypeSpec(ctx, child, nameNode, typeNode)
				}
			}
		}
	}
}

// handleGoTypeSpec handles Go struct/interface type specifications
func (cv *ChunkVisitor) handleGoTypeSpec(ctx context.Context, tsNode, nameNode, typeNode *tree_sitter.Node) {
	name := cv.getNodeText(nameNode)
	content := cv.getNodeText(tsNode)

	chunkID := cv.generateChunkID(cv.filePath, name, tsNode.StartPosition().Row)

	parentID := ""
	if cv.currentFile != nil {
		parentID = cv.currentFile.ID
	}

	chunk := model.NewCodeChunk(
		chunkID,
		model.ChunkTypeClass,
		2,
		content,
		cv.language,
		cv.filePath,
		cv.toRange(tsNode),
	).WithParent(parentID).
		WithName(name).
		WithContext(cv.moduleName, "")

	cv.chunks = append(cv.chunks, chunk)
}

// Helper methods

func (cv *ChunkVisitor) traverseChildren(ctx context.Context, tsNode *tree_sitter.Node) {
	for i := uint(0); i < tsNode.ChildCount(); i++ {
		child := tsNode.Child(i)
		cv.TraverseNode(ctx, child, nil)
	}
}

func (cv *ChunkVisitor) getChildByFieldName(tsNode *tree_sitter.Node, fieldName string) *tree_sitter.Node {
	return tsNode.ChildByFieldName(fieldName)
}

func (cv *ChunkVisitor) getNodeText(tsNode *tree_sitter.Node) string {
	startByte := tsNode.StartByte()
	endByte := tsNode.EndByte()

	if endByte > uint(len(cv.sourceCode)) {
		endByte = uint(len(cv.sourceCode))
	}

	return string(cv.sourceCode[startByte:endByte])
}

func (cv *ChunkVisitor) toRange(tsNode *tree_sitter.Node) base.Range {
	return base.Range{
		Start: base.Position{
			Line:      int(tsNode.StartPosition().Row),
			Character: int(tsNode.StartPosition().Column),
		},
		End: base.Position{
			Line:      int(tsNode.EndPosition().Row),
			Character: int(tsNode.EndPosition().Column),
		},
	}
}

func (cv *ChunkVisitor) generateChunkID(filePath, name string, line uint) string {
	// Generate a unique ID based on file path, name, and line number
	input := fmt.Sprintf("%s:%s:%d", filePath, name, line)
	hash := sha256.Sum256([]byte(input))
	hashStr := hex.EncodeToString(hash[:])

	// Convert hash to UUID format (8-4-4-4-12)
	// Qdrant requires valid UUID format
	return fmt.Sprintf("%s-%s-%s-%s-%s",
		hashStr[0:8],
		hashStr[8:12],
		hashStr[12:16],
		hashStr[16:20],
		hashStr[20:32],
	)
}

func (cv *ChunkVisitor) extractPackageName(tsNode *tree_sitter.Node) {
	nameNode := cv.getChildByFieldName(tsNode, "name")
	if nameNode != nil {
		cv.moduleName = cv.getNodeText(nameNode)
	}
}

func (cv *ChunkVisitor) extractJavaPackageName(tsNode *tree_sitter.Node) {
	for i := uint(0); i < tsNode.ChildCount(); i++ {
		child := tsNode.Child(i)
		if child.Kind() == "scoped_identifier" || child.Kind() == "identifier" {
			cv.moduleName = cv.getNodeText(child)
			break
		}
	}
}

func (cv *ChunkVisitor) extractGoFunctionSignature(tsNode *tree_sitter.Node) string {
	nameNode := cv.getChildByFieldName(tsNode, "name")
	paramsNode := cv.getChildByFieldName(tsNode, "parameters")
	resultNode := cv.getChildByFieldName(tsNode, "result")

	sig := ""
	if nameNode != nil {
		sig = cv.getNodeText(nameNode)
	}
	if paramsNode != nil {
		sig += cv.getNodeText(paramsNode)
	}
	if resultNode != nil {
		sig += " " + cv.getNodeText(resultNode)
	}

	return sig
}

func (cv *ChunkVisitor) extractPythonFunctionSignature(tsNode *tree_sitter.Node) string {
	nameNode := cv.getChildByFieldName(tsNode, "name")
	paramsNode := cv.getChildByFieldName(tsNode, "parameters")
	returnNode := cv.getChildByFieldName(tsNode, "return_type")

	sig := ""
	if nameNode != nil {
		sig = cv.getNodeText(nameNode)
	}
	if paramsNode != nil {
		sig += cv.getNodeText(paramsNode)
	}
	if returnNode != nil {
		sig += " -> " + cv.getNodeText(returnNode)
	}

	return sig
}

func (cv *ChunkVisitor) extractJavaMethodSignature(tsNode *tree_sitter.Node) string {
	// Get method signature including modifiers, return type, name, and parameters
	parts := []string{}

	nameNode := cv.getChildByFieldName(tsNode, "name")
	paramsNode := cv.getChildByFieldName(tsNode, "parameters")

	for i := uint(0); i < tsNode.ChildCount(); i++ {
		child := tsNode.Child(i)
		kind := child.Kind()

		if kind == "modifiers" || kind == "type_identifier" || kind == "void_type" {
			parts = append(parts, cv.getNodeText(child))
		}
	}

	if nameNode != nil {
		parts = append(parts, cv.getNodeText(nameNode))
	}
	if paramsNode != nil {
		parts = append(parts, cv.getNodeText(paramsNode))
	}

	return strings.Join(parts, " ")
}

func (cv *ChunkVisitor) extractJSFunctionSignature(tsNode *tree_sitter.Node) string {
	nameNode := cv.getChildByFieldName(tsNode, "name")
	paramsNode := cv.getChildByFieldName(tsNode, "parameters")
	returnNode := cv.getChildByFieldName(tsNode, "return_type")

	sig := ""
	if nameNode != nil {
		sig = cv.getNodeText(nameNode)
	}
	if paramsNode != nil {
		sig += cv.getNodeText(paramsNode)
	}
	if returnNode != nil {
		sig += ": " + cv.getNodeText(returnNode)
	}

	return sig
}

func (cv *ChunkVisitor) extractGoDocstring(tsNode *tree_sitter.Node) string {
	// Go docstrings are comments immediately before the function
	// This is a simplified implementation
	return ""
}

func (cv *ChunkVisitor) extractPythonDocstring(tsNode *tree_sitter.Node) string {
	bodyNode := cv.getChildByFieldName(tsNode, "body")
	if bodyNode == nil {
		return ""
	}

	// Look for first string expression in body
	for i := uint(0); i < bodyNode.ChildCount(); i++ {
		child := bodyNode.Child(i)
		if child.Kind() == "expression_statement" {
			for j := uint(0); j < child.ChildCount(); j++ {
				grandchild := child.Child(j)
				if grandchild.Kind() == "string" {
					docstring := cv.getNodeText(grandchild)
					// Remove quotes
					docstring = strings.Trim(docstring, `"'`)
					return docstring
				}
			}
		}
	}

	return ""
}

// handleConditional creates a chunk for conditional statements (if, switch, etc.)
func (cv *ChunkVisitor) handleConditional(ctx context.Context, tsNode *tree_sitter.Node, condType string) any {
	content := cv.getNodeText(tsNode)
	rng := cv.toRange(tsNode)
	/*
		cv.logger.Debug("handleConditional called",
			zap.String("type", condType),
			zap.String("file", cv.filePath),
			zap.Int("line", int(tsNode.StartPosition().Row)),
			zap.Int("threshold", cv.minConditionalLines))
	*/

	// Calculate line count
	lineCount := int(tsNode.EndPosition().Row - tsNode.StartPosition().Row + 1)

	// Skip if below threshold
	if lineCount < cv.minConditionalLines {
		/*
			cv.logger.Debug("Skipping small conditional",
				zap.String("type", condType),
				zap.String("file", cv.filePath),
				zap.Int("line", int(tsNode.StartPosition().Row)),
				zap.Int("lines", lineCount),
				zap.Int("threshold", cv.minConditionalLines))
		*/
		cv.traverseChildren(ctx, tsNode)
		return nil
	}

	// Extract condition expression
	/*
		var condition string
		conditionNode := cv.getChildByFieldName(tsNode, "condition")
		if conditionNode != nil {
			condition = cv.getNodeText(conditionNode)
		} else if tsNode.ChildCount() > 0 {
			// For switch statements, try to get the value being switched on
			for i := uint(0); i < tsNode.ChildCount(); i++ {
				child := tsNode.Child(i)
				if child.Kind() == "parenthesized_expression" || child.Kind() == "expression" {
					condition = cv.getNodeText(child)
					break
				}
			}
		}
	*/

	chunkID := cv.generateChunkID(cv.filePath, condType, tsNode.StartPosition().Row)

	parentID := ""
	if cv.currentFile != nil {
		parentID = cv.currentFile.ID
	}

	chunk := model.NewCodeChunk(
		chunkID,
		model.ChunkTypeConditional,
		4,
		content,
		cv.language,
		cv.filePath,
		rng,
	).WithParent(parentID).
		WithName(condType).
		//WithSignature(condition).
		WithContext(cv.moduleName, "")

	cv.chunks = append(cv.chunks, chunk)
	cv.traverseChildren(ctx, tsNode)
	return chunk
}

// handleLoop creates a chunk for loop statements (for, while, etc.)
func (cv *ChunkVisitor) handleLoop(ctx context.Context, tsNode *tree_sitter.Node, loopType string) any {
	content := cv.getNodeText(tsNode)
	rng := cv.toRange(tsNode)

	/*
		cv.logger.Debug("handleLoop called",
			zap.String("type", loopType),
			zap.String("file", cv.filePath),
			zap.Int("line", int(tsNode.StartPosition().Row)),
			zap.Int("threshold", cv.minLoopLines))

	*/

	// Calculate line count
	lineCount := int(tsNode.EndPosition().Row - tsNode.StartPosition().Row + 1)

	// Skip if below threshold
	if lineCount < cv.minLoopLines {
		/*
			cv.logger.Debug("Skipping small loop",
				zap.String("type", loopType),
				zap.String("file", cv.filePath),
				zap.Int("line", int(tsNode.StartPosition().Row)),
				zap.Int("lines", lineCount),
				zap.Int("threshold", cv.minLoopLines))
		*/
		cv.traverseChildren(ctx, tsNode)
		return nil
	}

	// Extract loop condition/range
	/*
		var condition string

		// Try different field names for different loop types
		conditionNode := cv.getChildByFieldName(tsNode, "condition")
		if conditionNode == nil {
			conditionNode = cv.getChildByFieldName(tsNode, "left")
		}
		if conditionNode == nil {
			conditionNode = cv.getChildByFieldName(tsNode, "right")
		}
		if conditionNode != nil {
			condition = cv.getNodeText(conditionNode)
		}
	*/

	chunkID := cv.generateChunkID(cv.filePath, loopType, tsNode.StartPosition().Row)

	parentID := ""
	if cv.currentFile != nil {
		parentID = cv.currentFile.ID
	}

	chunk := model.NewCodeChunk(
		chunkID,
		model.ChunkTypeLoop,
		4,
		content,
		cv.language,
		cv.filePath,
		rng,
	).WithParent(parentID).
		WithName(loopType).
		//WithSignature(condition).
		WithContext(cv.moduleName, "")

	cv.chunks = append(cv.chunks, chunk)
	cv.traverseChildren(ctx, tsNode)
	return chunk
}
