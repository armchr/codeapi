package lsp

import (
	"github.com/armchr/codeapi/internal/config"
	"github.com/armchr/codeapi/internal/model"
	"github.com/armchr/codeapi/internal/util"
	"github.com/armchr/codeapi/pkg/lsp/base"
	"context"
	"fmt"

	"go.uber.org/zap"
)

type LspService struct {
	config     *config.Config
	logger     *zap.Logger
	lspClients *util.SafeMap[base.LSPClient]
}

func NewLspService(config *config.Config, logger *zap.Logger) *LspService {
	return &LspService{
		config:     config,
		logger:     logger,
		lspClients: util.NewSafeMap[base.LSPClient](),
	}
}

func (rs *LspService) prepareLanguageServer(repoName string) (base.LSPClient, error) {
	rs.logger.Info("Preparing language server", zap.String("repo_name", repoName))

	repo, err := rs.config.GetRepository(repoName)
	if err != nil {
		rs.logger.Error("Failed to get repository config", zap.String("repo_name", repoName), zap.Error(err))
		return nil, fmt.Errorf("failed to get repository config: %w", err)
	}

	languageServer, err := NewLSPLanguageServer(rs.config, repo.Language, repo.Path, rs.logger)
	if err != nil {
		rs.logger.Error("Failed to create language server", zap.String("language", repo.Language), zap.Error(err))
		return nil, fmt.Errorf("failed to create language server: %w", err)
	}

	_, err = languageServer.Initialize(context.Background())
	if err != nil {
		rs.logger.Error("Failed to initialize language server", zap.String("repo_name", repoName), zap.Error(err))
		return nil, fmt.Errorf("failed to initialize language server: %w", err)
	}

	return languageServer, nil
}

func (rs *LspService) getLanguageServerClient(repoName string) (base.LSPClient, error) {
	rs.logger.Info("Getting language server client", zap.String("repo_name", repoName))

	//var client lsp.LSPClient
	client, exists := rs.lspClients.Get(repoName)
	if exists {
		return client, nil
	}

	client, err := rs.prepareLanguageServer(repoName)

	if err != nil {
		rs.logger.Error("Failed to prepare language server", zap.String("repo_name", repoName), zap.Error(err))
		return nil, fmt.Errorf("failed to prepare language server: %w", err)
	}
	rs.lspClients.Set(repoName, client)
	return client, nil
}

func (rs *LspService) getSymbolsOfType(ctx context.Context, lspClient base.LSPClient, fileUri string, symType int) ([]interface{}, error) {
	lspClient.DidOpenFile(ctx, fileUri)

	symbols, err := lspClient.GetDocumentSymbols(ctx, fileUri)
	if err != nil {
		return nil, fmt.Errorf("failed to get document symbols: %w", err)
	}

	matched := make([]interface{}, 0)

	for _, sym := range symbols {
		switch s := sym.(type) {
		case *base.SymbolInformation:
			if s.Kind == symType {
				matched = append(matched, s)
			}
		case *base.DocumentSymbol:
			if s.Kind == symType {
				matched = append(matched, s)
			}
		default:
			return nil, fmt.Errorf("unexpected symbol type: %T", sym)
		}
	}

	if len(matched) == 0 {
		return nil, fmt.Errorf("no symbols of type %d found in file %s",
			symType, fileUri)
	}

	return matched, nil
}

func (rs *LspService) getSymbolsOfTypes(ctx context.Context, lspClient base.LSPClient, fileUri string, symTypes []int) ([]interface{}, error) {
	var matched []interface{}
	for _, t := range symTypes {
		symbols, err := rs.getSymbolsOfType(ctx, lspClient, fileUri, t)
		if err != nil {
			return nil, fmt.Errorf("failed to get symbols of type %d: %w", t, err)
		}
		matched = append(matched, symbols...)
	}
	return matched, nil
}

func (rs *LspService) getFunctionDefinitions(ctx context.Context,
	lspClient base.LSPClient,
	uri string,
	functionName string) ([]model.FunctionDefinition, error) {
	// Get both functions and methods
	fns, err := rs.getSymbolsOfTypes(ctx, lspClient, uri, []int{base.SymbolKindFunction, base.SymbolKindMethod})
	if err != nil {
		return nil, fmt.Errorf("failed to get functions and methods in file: %w", err)
	}

	targetFunctions := make([]model.FunctionDefinition, 0)
	for _, fn := range fns {
		switch s := fn.(type) {
		case *base.DocumentSymbol:
			if lspClient.MatchSymbolByName(s.Name, functionName) {
				targetFunctions = append(targetFunctions, model.MapToFunctionFromDocumentSymbol(uri, s))
			}
		case *base.SymbolInformation:
			if lspClient.MatchSymbolByName(s.Name, functionName) {
				targetFunctions = append(targetFunctions, model.MapToFunctionFromSymbolInformation(uri, s))
			}
		default:
			return nil, fmt.Errorf("unexpected symbol type: %T", fn)
		}
	}

	if len(targetFunctions) == 0 {
		return nil, fmt.Errorf("function '%s' not found in file '%s'", functionName, uri)
	}

	return targetFunctions, nil
}

func (rs *LspService) extractSignature(sig map[string]interface{}) string {
	if label, ok := sig["label"].(string); ok {
		return label
	}
	return ""
}

func (rs *LspService) GetFunctionCallsAndDefinitions(ctx context.Context,
	repoName string,
	targetFunction *model.FunctionDefinition) ([]model.FunctionDependency, error) {
	lspClient, err := rs.getLanguageServerClient(repoName)
	if err != nil {
		return nil, fmt.Errorf("failed to get language server client: %w", err)
	}

	lspClient.DidOpenFile(ctx, targetFunction.Location.URI)

	return rs.getFunctionCallsAndDefinitions(ctx, lspClient, targetFunction)
}

func (rs *LspService) getFunctionCallsAndDefinitions(ctx context.Context,
	lspClient base.LSPClient,
	targetFunction *model.FunctionDefinition) ([]model.FunctionDependency, error) {

	// Analyze function calls within the target function
	inOrOutCalls, err := lspClient.GetCallHierarchy(ctx, targetFunction.Location.URI, targetFunction.Name, base.Position{
		Line:      targetFunction.Location.Range.Start.Line,
		Character: targetFunction.Location.Range.Start.Character,
	}, false)
	if err != nil {
		return nil, fmt.Errorf("failed to get call hierarchy: %w", err)
	}

	calls := []model.FunctionDependency{}

	// GetCallHierarchy can return (nil, nil) when no call hierarchy items are found
	if inOrOutCalls == nil {
		return calls, nil
	}

	for _, call := range inOrOutCalls.OutgoingCalls {
		dependency := model.MapToFunctionDependency(call, lspClient)
		calls = append(calls, dependency)
	}

	return calls, nil
}

func (rs *LspService) buildCallGraphWithFunction(
	ctx context.Context,
	lspClient base.LSPClient,
	callGraph *model.CallGraph,
	fn *model.FunctionDefinition,
	fnCache map[string]model.FunctionDefinition,
	depth int) error {
	deps, err := rs.getFunctionCallsAndDefinitions(ctx, lspClient, fn)
	if err != nil {
		return fmt.Errorf("failed to get function dependencies: %w", err)
	}
	// Process the function dependencies
	for _, dep := range deps {
		callGraph.AddFunctionDependency(fn, &dep)
		_, ok := fnCache[dep.Definition.ToKey()]
		if !ok {
			// If we have a cached function, use it
			fnCache[dep.Definition.ToKey()] = dep.Definition
			if depth > 1 && !lspClient.IsExternalModule(dep.Definition.Location.URI) {
				lspClient.DidOpenFile(ctx, dep.Definition.Location.URI)
				err := rs.buildCallGraphWithFunction(ctx, lspClient,
					callGraph,
					&dep.Definition, fnCache, depth-1)
				if err != nil {
					return fmt.Errorf("failed to get call graph: %w", err)
				}
			}
		}
	}

	return nil
}

func (rs *LspService) populateCallGraph(
	ctx context.Context,
	lspClient base.LSPClient,
	callGraph *model.CallGraph,
	uri string,
	functionName string,
	depth int) ([]model.FunctionDefinition, error) {

	fnCache := make(map[string]model.FunctionDefinition)

	fns, err := rs.getFunctionDefinitions(ctx, lspClient, uri, functionName)
	if err != nil {
		return nil, fmt.Errorf("failed to get functions in file: %w", err)
	}

	for _, fn := range fns {
		if _, ok := fnCache[fn.ToKey()]; !ok {
			fnCache[fn.ToKey()] = fn
			err := rs.buildCallGraphWithFunction(ctx, lspClient, callGraph, &fn, fnCache, depth)
			if err != nil {
				return nil, fmt.Errorf("failed to get call graph: %w", err)
			}
		}
	}
	return fns, nil
}

func (rs *LspService) PopulateCallGraphForFunction(
	ctx context.Context,
	repoName string,
	fn *model.FunctionDefinition,
	depth int) (*model.CallGraph, error) {
	lspClient, err := rs.getLanguageServerClient(repoName)
	if err != nil {
		return nil, fmt.Errorf("failed to get language server client: %w", err)
	}

	fnCache := make(map[string]model.FunctionDefinition)
	callGraph := model.NewCallGraph()
	err = rs.buildCallGraphWithFunction(ctx, lspClient, callGraph, fn, fnCache, depth)
	if err != nil {
		return nil, fmt.Errorf("failed to get call graph: %w", err)
	}
	return callGraph, nil
}

func (rs *LspService) GetFunctionDependencies(ctx context.Context, repoName, relativePath, functionName string, depth int) (*model.CallGraph, error) {
	lspClient, err := rs.getLanguageServerClient(repoName)
	if err != nil {
		return nil, fmt.Errorf("failed to get language server client: %w", err)
	}

	rootPath := lspClient.GetRootPath()
	uri, _ := util.ToUri(relativePath, rootPath)
	callGraph := model.NewCallGraph()
	roots, err := rs.populateCallGraph(ctx, lspClient, callGraph, uri, functionName, depth)
	if err != nil {
		return nil, fmt.Errorf("failed to get call graph: %w", err)
	}

	callGraph.Roots = roots

	return callGraph, nil
}

func (rs *LspService) GetFunctionHovers(ctx context.Context, repoName string, functions []model.FunctionDefinition) ([]string, error) {
	lspClient, err := rs.getLanguageServerClient(repoName)
	if err != nil {
		return nil, fmt.Errorf("failed to get language server client: %w", err)
	}

	hovers := make([]string, len(functions))

	for i, fn := range functions {
		// Ensure the file is open in the LSP client
		err := lspClient.DidOpenFile(ctx, fn.Location.URI)
		if err != nil {
			rs.logger.Warn("Failed to open file for hover",
				zap.String("uri", fn.Location.URI),
				zap.String("function", fn.Name),
				zap.Error(err))
			hovers[i] = ""
			continue
		}

		// Get hover information at the function's start position
		hoverInfo, err := lspClient.GetHover(ctx, fn.Location.URI, fn.Location.Range.Start)
		if err != nil {
			rs.logger.Warn("Failed to get hover information",
				zap.String("function", fn.Name),
				zap.String("uri", fn.Location.URI),
				zap.Error(err))
			hovers[i] = ""
			continue
		}

		if hoverInfo == nil {
			hovers[i] = ""
			continue
		}

		// Convert hover contents to string
		hoverString := rs.extractHoverContent(hoverInfo.Contents)
		hovers[i] = hoverString

		rs.logger.Debug("Retrieved hover for function",
			zap.String("function", fn.Name),
			zap.String("hover", hoverString))
	}

	return hovers, nil
}

func (rs *LspService) extractHoverContent(contents interface{}) string {
	if contents == nil {
		return ""
	}

	switch c := contents.(type) {
	case string:
		return c
	case map[string]interface{}:
		// Handle MarkupContent structure
		if value, ok := c["value"].(string); ok {
			return value
		}
		// Fallback to extracting any string field
		for _, v := range c {
			if str, ok := v.(string); ok {
				return str
			}
		}
	case []interface{}:
		// Handle array of content
		var result string
		for _, item := range c {
			if itemContent := rs.extractHoverContent(item); itemContent != "" {
				if result != "" {
					result += "\n"
				}
				result += itemContent
			}
		}
		return result
	}

	return ""
}

func (rs *LspService) getFunctionCallers(ctx context.Context,
	lspClient base.LSPClient,
	targetFunction *model.FunctionDefinition) ([]model.FunctionDependency, error) {

	// Analyze functions that call the target function (inbound calls)
	inOrOutCalls, err := lspClient.GetCallHierarchy(ctx, targetFunction.Location.URI, targetFunction.Name, base.Position{
		Line:      targetFunction.Location.Range.Start.Line,
		Character: targetFunction.Location.Range.Start.Character,
	}, true)
	if err != nil {
		return nil, fmt.Errorf("failed to get call hierarchy: %w", err)
	}

	callers := []model.FunctionDependency{}

	// GetCallHierarchy can return (nil, nil) when no call hierarchy items are found
	if inOrOutCalls == nil {
		return callers, nil
	}

	for _, call := range inOrOutCalls.IncomingCalls {
		// Convert incoming call to function dependency
		callLocations := make([]base.Location, 0, 1)
		for _, r := range call.FromRanges {
			callLocations = append(callLocations, base.Location{
				URI: call.From.URI,
				Range: base.Range{
					Start: base.Position{
						Line:      r.Start.Line,
						Character: r.Start.Character,
					},
					End: base.Position{
						Line:      r.End.Line,
						Character: r.End.Character,
					},
				},
			})
		}

		caller := model.FunctionDependency{
			Name:          call.From.Name,
			CallLocations: callLocations,
			Definition: model.FunctionDefinition{
				Name: call.From.Name,
				Location: base.Location{
					URI: call.From.URI,
					Range: base.Range{
						Start: base.Position{
							Line:      call.From.Range.Start.Line,
							Character: call.From.Range.Start.Character,
						},
						End: base.Position{
							Line:      call.From.Range.End.Line,
							Character: call.From.Range.End.Character,
						},
					},
				},
				IsExternal: lspClient.IsExternalModule(call.From.URI),
				Module:     "",
			},
		}
		callers = append(callers, caller)
	}

	return callers, nil
}

func (rs *LspService) buildCallGraphWithCallers(
	ctx context.Context,
	lspClient base.LSPClient,
	callGraph *model.CallGraph,
	fn *model.FunctionDefinition,
	fnCache map[string]model.FunctionDefinition,
	depth int) error {

	callers, err := rs.getFunctionCallers(ctx, lspClient, fn)
	if err != nil {
		return fmt.Errorf("failed to get function callers: %w", err)
	}

	// Process the function callers (reverse direction from dependencies)
	for _, caller := range callers {
		// Add caller as dependency where the edge points from caller to target function
		callGraph.AddFunctionDependency(&caller.Definition, &model.FunctionDependency{
			Name:          fn.Name,
			CallLocations: caller.CallLocations,
			Definition:    *fn,
		})

		_, ok := fnCache[caller.Definition.ToKey()]
		if !ok {
			fnCache[caller.Definition.ToKey()] = caller.Definition
			if depth > 1 && !lspClient.IsExternalModule(caller.Definition.Location.URI) {
				lspClient.DidOpenFile(ctx, caller.Definition.Location.URI)
				err := rs.buildCallGraphWithCallers(ctx, lspClient,
					callGraph,
					&caller.Definition, fnCache, depth-1)
				if err != nil {
					return fmt.Errorf("failed to get caller graph: %w", err)
				}
			}
		}
	}

	return nil
}

func (rs *LspService) populateCallerGraph(
	ctx context.Context,
	lspClient base.LSPClient,
	callGraph *model.CallGraph,
	uri string,
	functionName string,
	depth int) ([]model.FunctionDefinition, error) {

	fnCache := make(map[string]model.FunctionDefinition)

	fns, err := rs.getFunctionDefinitions(ctx, lspClient, uri, functionName)
	if err != nil {
		return nil, fmt.Errorf("failed to get functions in file: %w", err)
	}

	for _, fn := range fns {
		if _, ok := fnCache[fn.ToKey()]; !ok {
			fnCache[fn.ToKey()] = fn
			err := rs.buildCallGraphWithCallers(ctx, lspClient, callGraph, &fn, fnCache, depth)
			if err != nil {
				return nil, fmt.Errorf("failed to get caller graph: %w", err)
			}
		}
	}
	return fns, nil
}

func (rs *LspService) GetFunctionCallers(ctx context.Context, repoName, relativePath, functionName string, depth int) (*model.CallGraph, error) {
	lspClient, err := rs.getLanguageServerClient(repoName)
	if err != nil {
		return nil, fmt.Errorf("failed to get language server client: %w", err)
	}

	rootPath := lspClient.GetRootPath()
	uri, _ := util.ToUri(relativePath, rootPath)
	callGraph := model.NewCallGraph()
	roots, err := rs.populateCallerGraph(ctx, lspClient, callGraph, uri, functionName, depth)
	if err != nil {
		return nil, fmt.Errorf("failed to get caller graph: %w", err)
	}

	callGraph.Roots = roots

	return callGraph, nil
}
