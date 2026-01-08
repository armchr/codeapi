package controller

import (
	"github.com/armchr/codeapi/internal/config"
	"github.com/armchr/codeapi/internal/model"
	"github.com/armchr/codeapi/internal/model/ast"
	"github.com/armchr/codeapi/internal/parse"
	"github.com/armchr/codeapi/internal/service/codegraph"
	"github.com/armchr/codeapi/internal/util"
	"github.com/armchr/codeapi/pkg/lsp"
	"github.com/armchr/codeapi/pkg/lsp/base"
	"context"
	"fmt"
	"strings"

	"go.uber.org/zap"
)

type PostProcessor struct {
	codeGraph  *codegraph.CodeGraph
	lspService *lsp.LspService
	logger     *zap.Logger
}

func NewPostProcessor(codeGraph *codegraph.CodeGraph, lspService *lsp.LspService, logger *zap.Logger) *PostProcessor {
	return &PostProcessor{
		codeGraph:  codeGraph,
		lspService: lspService,
		logger:     logger,
	}
}

func (pp *PostProcessor) ProcessFakeClasses(ctx context.Context, fileScope *ast.Node) error {
	return pp.codeGraph.UpdateFakeClasses(ctx, fileScope.FileID)
}

func (pp *PostProcessor) PostProcessRepository(ctx context.Context, repo *config.Repository) error {
	pp.logger.Info("Starting post-processing for repository", zap.String("name", repo.Name))

	fileScopes, err := pp.codeGraph.FindFileScopes(ctx, repo.Name, "")
	if err != nil {
		return fmt.Errorf("failed to find file scopes: %w", err)
	}

	pp.logger.Info("Found file scopes", zap.Int("count", len(fileScopes)))

	for _, fileScope := range fileScopes {
		pp.logger.Info("Post-processing file", zap.String("path", fileScope.MetaData["path"].(string)), zap.Int64("fileId", int64(fileScope.ID)))

		if err := pp.processOneFile(ctx, repo, fileScope); err != nil {
			pp.logger.Error("Failed to post-process file", zap.String("path", fileScope.MetaData["path"].(string)), zap.Int64("fileId", int64(fileScope.ID)), zap.Error(err))
			continue
		}

		pp.logger.Info("Completed post-processing for file", zap.String("path", fileScope.MetaData["path"].(string)), zap.Int64("fileId", int64(fileScope.ID)))
	}

	pp.logger.Info("Completed post-processing for repository", zap.String("name", repo.Name))

	return nil
}

func (pp *PostProcessor) processOneFile(ctx context.Context, repo *config.Repository, fileScope *ast.Node) error {
	language := fileScope.MetaData["language"].(string)
	langType := parse.NewLanguageTypeFromString(language)
	if langType == parse.Go {
		if err := pp.ProcessFakeClasses(ctx, fileScope); err != nil {
			pp.logger.Error("Failed to process fake classes", zap.Error(err))
		}
	}

	if err := pp.processFunctionCalls(ctx, repo, fileScope); err != nil {
		return fmt.Errorf("failed to process function calls: %w", err)
	}

	// Process inheritance for Java files
	if langType == parse.Java {
		if err := pp.processInheritance(ctx, repo, fileScope); err != nil {
			pp.logger.Error("Failed to process inheritance", zap.Error(err))
		}

		if err := pp.processConstructorCalls(ctx, repo, fileScope); err != nil {
			pp.logger.Error("Failed to process constructor calls", zap.Error(err))
		}
	}

	return nil
}

func (pp *PostProcessor) processFunctionCalls(ctx context.Context, repo *config.Repository, fileScope *ast.Node) error {
	functionCallsInFunction, err := pp.codeGraph.FindFunctionCalls(ctx, fileScope.ID)
	if err != nil {
		return fmt.Errorf("failed to find orphan function calls: %w", err)
	}

	pp.logger.Info("Found orphan function calls", zap.Int("count", len(functionCallsInFunction)))

	fileUri, _ := util.ToUri(fileScope.MetaData["path"].(string), repo.Path)

	for containerFunctionId, fnCalls := range functionCallsInFunction {
		pp.processFunctionCallsInContainerFunction(ctx, repo, fileUri, containerFunctionId, fnCalls)
	}

	return nil
}

func (pp *PostProcessor) nodeToFunctionDefinition(ctx context.Context, fileUri string, functionNode *ast.Node) *model.FunctionDefinition {
	return &model.FunctionDefinition{
		Name: functionNode.Name,
		Location: base.Location{
			URI: fileUri,
			Range: base.Range{
				Start: base.Position{
					Line:      functionNode.Range.Start.Line,
					Character: functionNode.Range.Start.Character,
				},
				End: base.Position{
					Line:      functionNode.Range.End.Line,
					Character: functionNode.Range.End.Character,
				},
			},
		},
	}
}

func (pp *PostProcessor) processFunctionCallsInContainerFunction(ctx context.Context,
	repo *config.Repository,
	fileUri string,
	containerFunctionID ast.NodeID,
	fnCalls []*ast.Node,
) error {
	containingFunction, err := pp.codeGraph.ReadFunction(ctx, containerFunctionID)
	if err != nil {
		return fmt.Errorf("failed to find containing function: %w", err)
	}
	if containingFunction == nil {
		return fmt.Errorf("no function found for call node id %d", containerFunctionID)
	}

	// Skip lambda functions - they don't have real names in the source file
	// and LSP can't find them. Their calls will be processed by the parent function.
	if strings.HasPrefix(containingFunction.Name, "__lambda__") {
		pp.logger.Debug("Skipping lambda container function",
			zap.String("functionName", containingFunction.Name),
			zap.Int("callCount", len(fnCalls)))
		return nil
	}

	containingFnDefn := pp.nodeToFunctionDefinition(ctx, fileUri, containingFunction)

	deps, err := pp.lspService.GetFunctionCallsAndDefinitions(ctx, repo.Name, containingFnDefn)
	if err != nil {
		return fmt.Errorf("failed to get function dependencies: %w", err)
	}

	if len(deps) == 0 {
		pp.logger.Info("No dependencies found for containing function",
			zap.String("functionName", containingFnDefn.Name),
			zap.String("functionPath", containingFnDefn.Location.URI))
		return nil
	}

	err = pp.createCallsRelations(ctx, repo, fnCalls, deps)
	if err != nil {
		pp.logger.Error("Failed to create calls relations",
			zap.Error(err))
	}

	return nil
}

/*
func (pp *PostProcessor) getFunctionPath(functionNode *ast.Node) (string, error) {
	if functionNode.MetaData == nil {
		return "", fmt.Errorf("function node %d has no metadata", functionNode.ID)
	}

	path, ok := functionNode.MetaData["path"].(string)
	if !ok {
		return "", fmt.Errorf("function node %d has no path in metadata", functionNode.ID)
	}

	return path, nil
}
*/

func (pp *PostProcessor) findCallInDependency(call *ast.Node, dependencies []model.FunctionDependency) *model.FunctionDependency {
	for _, dep := range dependencies {
		if pp.matchesFunctionCall(call, &dep) {
			return &dep
		}
	}
	return nil
}

func (pp *PostProcessor) createCallsRelations(ctx context.Context, repo *config.Repository, calls []*ast.Node, dependencies []model.FunctionDependency) error {
	for _, call := range calls {
		// Skip constructor calls - they are processed separately by processConstructorCalls
		if call.MetaData != nil {
			if isConstructor, ok := call.MetaData["is_constructor"].(bool); ok && isConstructor {
				continue
			}
		}

		dep := pp.findCallInDependency(call, dependencies)
		if dep == nil {
			pp.logger.Warn("No matching dependency found for function call",
				zap.Int64("callNodeId", int64(call.ID)),
				zap.String("callName", call.Name))
			continue
		}

		// Get target function node from CodeGraph
		if dep.Definition.IsExternal {
			if call.MetaData == nil {
				call.MetaData = make(map[string]any)
			}
			call.MetaData["external"] = true
			pp.codeGraph.CreateFunctionCall(ctx, call)
			continue
		}

		targetFileRelPath := util.ToRelativePath(repo.Path, util.ExtractPathFromURI(dep.Definition.Location.URI))
		fileScopes, err := pp.codeGraph.FindFileScopes(ctx, repo.Name, targetFileRelPath)
		if err != nil || len(fileScopes) == 0 {
			pp.logger.Error("Failed to find file scopes for dependency",
				zap.String("functionName", dep.Definition.Name),
				zap.String("functionPath", dep.Definition.Location.URI),
				zap.Error(err))
			continue
		}

		targetFileScope := fileScopes[0]
		targetDefns, err := pp.codeGraph.FindFunctionsByName(ctx, int(targetFileScope.FileID), dep.Definition.Name)
		if err != nil || len(targetDefns) == 0 {
			pp.logger.Error("Failed to find target function for dependency",
				zap.String("functionName", dep.Definition.Name),
				zap.String("functionPath", dep.Definition.Location.URI),
				zap.Error(err))
			continue
		}

		targetDefnID := ast.InvalidNodeID

		for _, fn := range targetDefns {
			if base.RangeInRange(fn.Range, dep.Definition.Location.Range) ||
				base.RangeInRange(dep.Definition.Location.Range, fn.Range) {
				targetDefnID = fn.ID
				break
			}
		}

		if targetDefnID != ast.InvalidNodeID {
			pp.codeGraph.CreateCallsFunctionRelation(ctx, call.ID, targetDefnID, call.FileID)
			// log
			pp.logger.Info("Created CALLS_FUNCTION relation",
				zap.Int64("callNodeId", int64(call.ID)),
				zap.String("callName", call.Name),
				zap.Int64("targetFunctionId", int64(targetDefnID)),
				zap.String("targetFunctionName", dep.Definition.Name))
		}
	}

	return nil
}

/*
func (pp *PostProcessor) getDependenciesFromCallGraph(callGraph *model.CallGraph, root model.FunctionDefinition) []model.FunctionDependency {
	var dependencies []model.FunctionDependency

	for _, edge := range callGraph.Edges {
		if edge.From != nil && edge.From.ToKey() == root.ToKey() {
			dep := model.FunctionDependency{
				Name:       edge.To.Name,
				Definition: *edge.To,
			}
			dependencies = append(dependencies, dep)
		}
	}

	return dependencies
}
*/

func (pp *PostProcessor) matchesFunctionCall(callNode *ast.Node, dependency *model.FunctionDependency) bool {
	if !dependency.IsIn(&callNode.Range) {
		return false
	}

	callName := callNode.Name
	depName := dependency.Definition.Name

	// Direct match (most common case after name extraction fix)
	if callName == depName {
		return true
	}

	// Handle qualified call names like "this.method" or "object.method"
	// where tree-sitter might include the receiver
	if strings.HasSuffix(callName, "."+depName) {
		return true
	}

	// Handle case where dependency name might be qualified
	if strings.HasSuffix(depName, "."+callName) {
		return true
	}

	return false
}

// processInheritance resolves inheritance relationships for classes in a file.
// It looks at the extends/implements metadata captured during parsing and creates
// INHERITS relationships to the resolved parent classes/interfaces.
func (pp *PostProcessor) processInheritance(ctx context.Context, repo *config.Repository, fileScope *ast.Node) error {
	// Find all classes in this file
	classes, err := pp.codeGraph.FindAllClassesInFile(ctx, fileScope.FileID)
	if err != nil {
		return fmt.Errorf("failed to find classes in file: %w", err)
	}

	pp.logger.Info("Processing inheritance for classes",
		zap.Int("count", len(classes)),
		zap.String("file", fileScope.MetaData["path"].(string)))

	for _, class := range classes {
		if class.MetaData == nil {
			continue
		}

		// Process extends (single parent class or interface extends)
		if extends, ok := class.MetaData["extends"]; ok {
			pp.resolveAndCreateInheritance(ctx, repo, class, extends)
		}

		// Process implements (multiple interfaces)
		if implements, ok := class.MetaData["implements"]; ok {
			pp.resolveAndCreateInheritance(ctx, repo, class, implements)
		}
	}

	return nil
}

// resolveAndCreateInheritance resolves parent type names and creates INHERITS relationships.
// The parentTypes parameter can be a string (single parent) or []string (multiple parents).
func (pp *PostProcessor) resolveAndCreateInheritance(ctx context.Context, repo *config.Repository, childClass *ast.Node, parentTypes any) {
	var typeNames []string

	switch v := parentTypes.(type) {
	case string:
		typeNames = []string{v}
	case []string:
		typeNames = v
	case []any:
		// Handle case where it comes from JSON/Neo4j as []any
		for _, item := range v {
			if s, ok := item.(string); ok {
				typeNames = append(typeNames, s)
			}
		}
	default:
		pp.logger.Warn("Unexpected type for parent types",
			zap.String("childClass", childClass.Name),
			zap.Any("parentTypes", parentTypes))
		return
	}

	for _, typeName := range typeNames {
		// Extract simple name if it's a qualified name
		simpleName := extractSimpleName(typeName)

		// Try to find the parent class/interface in the repo
		parentClasses, err := pp.codeGraph.FindClassesByNameInRepo(ctx, simpleName, repo.Name)
		if err != nil {
			pp.logger.Warn("Failed to find parent class",
				zap.String("childClass", childClass.Name),
				zap.String("parentName", simpleName),
				zap.Error(err))
			continue
		}

		if len(parentClasses) == 0 {
			pp.logger.Debug("Parent class not found in repo (may be external)",
				zap.String("childClass", childClass.Name),
				zap.String("parentName", simpleName))
			continue
		}

		// If multiple matches, try to pick the best one (same package if possible)
		parentClass := pp.selectBestParentMatch(ctx, childClass, parentClasses)
		if parentClass == nil {
			parentClass = parentClasses[0] // Default to first match
		}

		// Create INHERITS relationship: childClass INHERITS parentClass
		err = pp.codeGraph.CreateInheritsRelation(ctx, parentClass.ID, childClass.ID, childClass.FileID)
		if err != nil {
			pp.logger.Error("Failed to create INHERITS relation",
				zap.String("childClass", childClass.Name),
				zap.String("parentClass", parentClass.Name),
				zap.Error(err))
			continue
		}

		pp.logger.Info("Created INHERITS relation",
			zap.String("childClass", childClass.Name),
			zap.Int64("childClassId", int64(childClass.ID)),
			zap.String("parentClass", parentClass.Name),
			zap.Int64("parentClassId", int64(parentClass.ID)))
	}
}

// selectBestParentMatch selects the best matching parent class when multiple classes
// with the same name exist. Prefers classes in the same package/module.
func (pp *PostProcessor) selectBestParentMatch(ctx context.Context, childClass *ast.Node, parentClasses []*ast.Node) *ast.Node {
	if len(parentClasses) == 1 {
		return parentClasses[0]
	}

	// Get the child's module name for comparison
	childModuleName, err := pp.codeGraph.GetModuleName(ctx, childClass.FileID)
	if err != nil {
		return nil
	}

	// Prefer parent in the same module/package
	for _, parent := range parentClasses {
		parentModuleName, err := pp.codeGraph.GetModuleName(ctx, parent.FileID)
		if err != nil {
			continue
		}
		if parentModuleName == childModuleName {
			return parent
		}
	}

	return nil
}

// extractSimpleName extracts the simple class name from a potentially qualified name.
// e.g., "com.example.MyClass" -> "MyClass"
func extractSimpleName(name string) string {
	lastDot := strings.LastIndex(name, ".")
	if lastDot == -1 {
		return name
	}
	return name[lastDot+1:]
}

// processConstructorCalls resolves constructor calls (new expressions) to their constructor definitions.
// This uses the code graph to find matching classes and their constructors without LSP calls.
func (pp *PostProcessor) processConstructorCalls(ctx context.Context, repo *config.Repository, fileScope *ast.Node) error {
	// Find all constructor calls in this file
	constructorCalls, err := pp.codeGraph.FindConstructorCallsInFile(ctx, fileScope.FileID)
	if err != nil {
		return fmt.Errorf("failed to find constructor calls: %w", err)
	}

	if len(constructorCalls) == 0 {
		return nil
	}

	pp.logger.Info("Processing constructor calls",
		zap.Int("count", len(constructorCalls)),
		zap.String("file", fileScope.MetaData["path"].(string)))

	for _, call := range constructorCalls {
		pp.resolveConstructorCall(ctx, repo, call)
	}

	return nil
}

// resolveConstructorCall resolves a single constructor call to its definition.
func (pp *PostProcessor) resolveConstructorCall(ctx context.Context, repo *config.Repository, call *ast.Node) {
	// The call name is the class name being constructed (e.g., "Pet" from "new Pet()")
	className := call.Name
	if className == "" {
		return
	}

	// Extract simple name if qualified
	simpleName := extractSimpleName(className)

	// Find classes with this name in the repository
	classes, err := pp.codeGraph.FindClassesByNameInRepo(ctx, simpleName, repo.Name)
	if err != nil {
		pp.logger.Warn("Failed to find class for constructor call",
			zap.String("className", simpleName),
			zap.Error(err))
		return
	}

	if len(classes) == 0 {
		// Class not found - likely external (e.g., java.util.ArrayList)
		pp.logger.Debug("Class not found for constructor call (likely external)",
			zap.String("className", simpleName),
			zap.Int64("callId", int64(call.ID)))

		// Mark as external
		if call.MetaData == nil {
			call.MetaData = make(map[string]any)
		}
		call.MetaData["external"] = true
		pp.codeGraph.CreateFunctionCall(ctx, call)
		return
	}

	// Select the best matching class (prefer same package)
	targetClass := pp.selectBestClassMatch(ctx, call, classes)
	if targetClass == nil {
		targetClass = classes[0]
	}

	// Find constructors of the target class
	constructors, err := pp.codeGraph.GetConstructorsOfClass(ctx, targetClass.ID)
	if err != nil {
		pp.logger.Warn("Failed to find constructors of class",
			zap.String("className", targetClass.Name),
			zap.Error(err))
		return
	}

	if len(constructors) == 0 {
		pp.logger.Debug("No constructors found for class",
			zap.String("className", targetClass.Name))
		return
	}

	// For now, link to the first constructor (future: match by parameter count)
	// TODO: Match constructor by parameter count for overloaded constructors
	constructor := constructors[0]

	// Create CALLS_FUNCTION relationship
	err = pp.codeGraph.CreateCallsFunctionRelation(ctx, call.ID, constructor.ID, call.FileID)
	if err != nil {
		pp.logger.Error("Failed to create CALLS_FUNCTION relation for constructor",
			zap.Int64("callId", int64(call.ID)),
			zap.Int64("constructorId", int64(constructor.ID)),
			zap.Error(err))
		return
	}

	pp.logger.Info("Resolved constructor call",
		zap.String("className", className),
		zap.Int64("callId", int64(call.ID)),
		zap.String("constructorName", constructor.Name),
		zap.Int64("constructorId", int64(constructor.ID)))
}

// selectBestClassMatch selects the best matching class when multiple classes
// with the same name exist. Prefers class in the same package/module as the caller.
func (pp *PostProcessor) selectBestClassMatch(ctx context.Context, call *ast.Node, classes []*ast.Node) *ast.Node {
	if len(classes) == 1 {
		return classes[0]
	}

	// Get the caller's module name
	callerModuleName, err := pp.codeGraph.GetModuleName(ctx, call.FileID)
	if err != nil {
		return nil
	}

	// Prefer class in the same module/package
	for _, class := range classes {
		classModuleName, err := pp.codeGraph.GetModuleName(ctx, class.FileID)
		if err != nil {
			continue
		}
		if classModuleName == callerModuleName {
			return class
		}
	}

	return nil
}
