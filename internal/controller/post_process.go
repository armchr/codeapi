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

	// dependency name ends with call node name
	return strings.HasSuffix(callNode.Name, dependency.Definition.Name)
}
