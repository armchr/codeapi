package lsp

import (
	"github.com/armchr/codeapi/internal/config"
	"github.com/armchr/codeapi/pkg/lsp/base"
	"strings"

	"go.uber.org/zap"
)

type PythonLanguageServerClient struct {
	*BaseClient
	rootPath string
	logger   *zap.Logger
}

func NewPythonLanguageServerClient(config *config.Config, rootPath string, logger *zap.Logger) (*PythonLanguageServerClient, error) {
	logger.Info("Creating new Python language server client")
	base, err := NewBaseClient(config.App.Python, logger)
	if err != nil {
		return nil, err
	}

	t := &PythonLanguageServerClient{BaseClient: base, rootPath: rootPath, logger: logger}
	t.client = t
	return t, nil
}

func (t *PythonLanguageServerClient) GetRootPath() string {
	return t.rootPath
}

func (t *PythonLanguageServerClient) LanguageID(uri string) string {
	if strings.HasSuffix(uri, ".py") {
		return "python"
	} else {
		return "unknown"
	}
}

func (t *PythonLanguageServerClient) IsExternalModule(uri string) bool {
	if strings.Contains(uri, "site-packages/") || strings.Contains(uri, "dist-packages/") ||
		strings.Contains(uri, ".venv/") || strings.Contains(uri, "node_modules/") {
		return true
	}
	return false
}

func (t *PythonLanguageServerClient) MatchSymbolByName(name, nameInFile string) bool {
	return base.MatchExact(name, nameInFile)
}

func (t *PythonLanguageServerClient) SymbolPartToMatch(name string) string {
	return base.LastSegment(name)
}

// GetCallHierarchy implements call hierarchy using textDocument/references as fallback
// since pylsp doesn't support textDocument/prepareCallHierarchy
/*
func (t *PythonLanguageServerClient) GetCallHierarchy(ctx context.Context, uri string, fnName string, position base.Position, inbound bool) (*base.CallHierarchyIncomingOrgoingCalls, error) {
	t.logger.Info("Getting call hierarchy from Python language server using references fallback", zap.String("uri", uri), zap.String("function", fnName), zap.Bool("inbound", inbound))

	if !t.initialized {
		t.logger.Error("Python language server client not initialized", zap.String("uri", uri))
		return nil, fmt.Errorf("client not initialized")
	}

	fileHolder := t.fileHolders[uri]
	if fileHolder == nil {
		t.logger.Error("file not opened in Python language server", zap.String("uri", uri))
		return nil, fmt.Errorf("file not opened in language server")
	}

	// If fnName is provided, try to find its position in the file
	if fnName != "" {
		line := position.Line
		character := fileHolder.FindNameInNextLines(fnName, line, 50)
		if character == -1 {
			t.logger.Error("function name not found in file", zap.String("uri", uri), zap.String("function_name", fnName))
			return nil, fmt.Errorf("function name not found in file")
		}
		position.Character = character
	}

	// Use textDocument/references instead of prepareCallHierarchy
	params := base.ReferenceParams{
		TextDocumentPositionParams: base.TextDocumentPositionParams{
			TextDocument: base.TextDocumentIdentifier{URI: uri},
			Position:     position,
		},
		Context: base.ReferenceContext{
			IncludeDeclaration: true,
		},
	}

	t.logger.Debug("Requesting references from Python language server", zap.String("uri", uri))
	resp, err := t.sendRequest(ctx, "textDocument/references", params)
	if err != nil {
		t.logger.Error("Failed to get references from Python language server", zap.String("uri", uri), zap.Error(err))
		return nil, fmt.Errorf("failed to get references: %w", err)
	}

	// Convert references to call hierarchy format
	var result *base.CallHierarchyIncomingOrgoingCalls
	if resp.Result == nil {
		t.logger.Warn("No references found for function", zap.String("uri", uri), zap.String("function", fnName))
		return nil, nil
	}

	if references, ok := resp.Result.([]interface{}); ok {
		t.logger.Debug("References retrieved", zap.String("uri", uri), zap.Int("count", len(references)))

		if inbound {
			// For inbound calls, references are the callers
			var incomingCalls []base.CallHierarchyIncomingCall
			for _, ref := range references {
				if refMap, ok := ref.(map[string]interface{}); ok {
					refUri, _ := refMap["uri"].(string)

					// Skip the declaration itself (same URI and position)
					if refUri == uri {
						if rangeMap, ok := refMap["range"].(map[string]interface{}); ok {
							if startMap, ok := rangeMap["start"].(map[string]interface{}); ok {
								refLine := int(startMap["line"].(float64))
								if refLine == position.Line {
									continue // Skip declaration
								}
							}
						}
					}

					// Create a caller item
					if rangeMap, ok := refMap["range"].(map[string]interface{}); ok {
						if startMap, ok := rangeMap["start"].(map[string]interface{}); ok {
							if endMap, ok := rangeMap["end"].(map[string]interface{}); ok {
								refRange := base.Range{
									Start: base.Position{
										Line:      int(startMap["line"].(float64)),
										Character: int(startMap["character"].(float64)),
									},
									End: base.Position{
										Line:      int(endMap["line"].(float64)),
										Character: int(endMap["character"].(float64)),
									},
								}

								callerItem := base.CallHierarchyItem{
									Name: "caller", // We don't have the exact function name from references
									Kind: base.SymbolKindFunction,
									URI:  refUri,
									Range: refRange,
									SelectionRange: refRange,
								}

								incomingCall := base.CallHierarchyIncomingCall{
									From:       callerItem,
									FromRanges: []base.Range{refRange},
								}
								incomingCalls = append(incomingCalls, incomingCall)
							}
						}
					}
				}
			}

			result = &base.CallHierarchyIncomingOrgoingCalls{
				IncomingCalls: incomingCalls,
			}
		} else {
			// For outgoing calls, we'd need to parse the function body to find calls
			// This is more complex with just references, so we'll return empty for now
			t.logger.Info("Outgoing calls not fully supported with references fallback", zap.String("uri", uri))
			result = &base.CallHierarchyIncomingOrgoingCalls{
				OutgoingCalls: []base.CallHierarchyOutgoingCall{},
			}
		}
	} else {
		t.logger.Error("Unexpected response type for references", zap.String("uri", uri), zap.Any("result", resp.Result))
		return nil, fmt.Errorf("unexpected response type for references: %T", resp.Result)
	}

	t.logger.Info("Call hierarchy retrieved using references fallback", zap.String("uri", uri), zap.String("function", fnName))
	return result, nil
}
*/

/*func (t *TypeScriptLanguageServerClient) GetDocumentSymbols(ctx context.Context, uri, text string) ([]model.Function, error) {
	t.logger.Info("Getting document symbols from TypeScript language server", zap.String("uri", uri))

	if !t.initialized {
		t.logger.Error("TypeScript language server client not initialized", zap.String("uri", uri))
		return nil, fmt.Errorf("client not initialized")
	}

	// Read file content if not provided
	var fileContent string
	if text == "" {
		filePath := strings.TrimPrefix(uri, "file://")
		content, err := ioutil.ReadFile(filePath)
		if err != nil {
			t.logger.Error("Failed to read file", zap.String("file_path", filePath), zap.Error(err))
			return nil, fmt.Errorf("failed to read file: %w", err)
		}
		fileContent = string(content)
	} else {
		fileContent = text
	}

	// Determine language ID from file extension
	var languageId string
	if strings.HasSuffix(uri, ".ts") {
		languageId = "typescript"
	} else if strings.HasSuffix(uri, ".tsx") {
		languageId = "typescriptreact"
	} else {
		languageId = "javascript"
	}

	// First, open the document
	didOpenParams := DidOpenTextDocumentParams{
		TextDocument: TextDocumentItem{
			URI:        uri,
			LanguageId: languageId,
			Version:    1,
			Text:       fileContent,
		},
	}

	t.logger.Debug("Opening document in TypeScript language server", zap.String("uri", uri))
	if err := t.SendNotification("textDocument/didOpen", didOpenParams); err != nil {
		t.logger.Error("Failed to open document in TypeScript language server", zap.String("uri", uri), zap.Error(err))
		return nil, fmt.Errorf("failed to open document: %w", err)
	}

	// Now request document symbols
	params := DocumentSymbolParams{
		TextDocument: TextDocumentIdentifier{
			URI: uri,
		},
	}

	t.logger.Debug("Requesting document symbols from TypeScript language server", zap.String("uri", uri))
	resp, err := t.sendRequest(ctx, "textDocument/documentSymbol", params)
	if err != nil {
		t.logger.Error("Failed to get document symbols from TypeScript language server", zap.String("uri", uri), zap.Error(err))
		return nil, fmt.Errorf("failed to get document symbols: %w", err)
	}

	t.logger.Debug("Raw LSP response", zap.String("uri", uri), zap.Any("result", resp.Result))
	functions := convertSymbolsToFunctions(resp.Result, uri)
	t.logger.Debug("Converted functions", zap.String("uri", uri), zap.Int("count", len(functions)), zap.Any("functions", functions))
	t.logger.Info("Successfully extracted functions from TypeScript language server", zap.String("uri", uri), zap.Int("function_count", len(functions)))
	return functions, nil
}
*/

/*
func (t *TypeScriptLanguageServerClient) GetHover(ctx context.Context, uri string, position Position) (interface{}, error) {
	t.logger.Debug("Getting hover information", zap.String("uri", uri), zap.Int("line", position.Line), zap.Int("character", position.Character))

	if !t.initialized {
		return nil, fmt.Errorf("client not initialized")
	}

	params := HoverParams{
		TextDocumentPositionParams: TextDocumentPositionParams{
			TextDocument: TextDocumentIdentifier{URI: uri},
			Position:     position,
		},
	}

	resp, err := t.sendRequest(ctx, "textDocument/hover", params)
	if err != nil {
		t.logger.Error("Failed to get hover information", zap.String("uri", uri), zap.Error(err))
		return nil, fmt.Errorf("failed to get hover: %w", err)
	}

	return resp.Result, nil
}
*/

/*
func (t *TypeScriptLanguageServerClient) GetSignatureHelp(ctx context.Context, uri string, position Position) (interface{}, error) {
	t.logger.Debug("Getting signature help", zap.String("uri", uri), zap.Int("line", position.Line), zap.Int("character", position.Character))

	if !t.initialized {
		return nil, fmt.Errorf("client not initialized")
	}

	params := SignatureHelpParams{
		TextDocumentPositionParams: TextDocumentPositionParams{
			TextDocument: TextDocumentIdentifier{URI: uri},
			Position:     position,
		},
	}

	resp, err := t.sendRequest(ctx, "textDocument/signatureHelp", params)
	if err != nil {
		t.logger.Error("Failed to get signature help", zap.String("uri", uri), zap.Error(err))
		return nil, fmt.Errorf("failed to get signature help: %w", err)
	}

	return resp.Result, nil
}
*/

/*
func (t *TypeScriptLanguageServerClient) GetDefinition(ctx context.Context, uri string, position base.Position) (*base.Location, error) {
	t.logger.Info("Sending definition request to TypeScript LSP",
		zap.String("uri", uri),
		zap.Int("line", position.Line),
		zap.Int("character", position.Character))

	if !t.initialized {
		t.logger.Error("TypeScript LSP client not initialized for definition request")
		return nil, fmt.Errorf("client not initialized")
	}

	params := map[string]interface{}{
		"command": "_typescript.goToSourceDefinition",
		"arguments": []interface{}{
			uri,
			map[string]interface{}{
				"line":      position.Line,
				"character": position.Character,
			},
		},
	}

	t.logger.Debug("Definition request params",
		zap.Any("params", params))

	resp, _ := t.sendRequest(ctx, "workspace/executeCommand", params)

	t.logger.Info("TypeScript LSP definition response received",
		zap.String("uri", uri),
		zap.Int("line", position.Line),
		zap.Int("character", position.Character),
		zap.String("response_type", fmt.Sprintf("%T", resp.Result)),
		zap.Any("raw_result", resp.Result))

	// Log additional details about the response structure
	if resp.Result == nil {
		t.logger.Warn("TypeScript LSP returned nil definition result",
			zap.String("uri", uri))
	} else {
		switch r := resp.Result.(type) {
		case []interface{}:
			t.logger.Info("TypeScript LSP returned array result",
				zap.String("uri", uri),
				zap.Int("length", len(r)))
		case map[string]interface{}:
			keys := make([]string, 0, len(r))
			for k := range r {
				keys = append(keys, k)
			}
			t.logger.Info("TypeScript LSP returned map result",
				zap.String("uri", uri),
				zap.Strings("keys", keys))
		default:
			t.logger.Info("TypeScript LSP returned other result type",
				zap.String("uri", uri),
				zap.String("type", fmt.Sprintf("%T", resp.Result)))
		}
	}

	if locations, ok := resp.Result.([]interface{}); ok && len(locations) > 0 {
		if loc, ok := locations[0].(map[string]interface{}); ok {
			return &base.Location{
				URI: uri,
				Range: base.Range{
					Start: base.Position{
						Line:      int(loc["start"].(map[string]interface{})["line"].(float64)),
						Character: int(loc["start"].(map[string]interface{})["character"].(float64)),
					},
					End: base.Position{
						Line:      int(loc["end"].(map[string]interface{})["line"].(float64)),
						Character: int(loc["end"].(map[string]interface{})["character"].(float64)),
					},
				},
			}, nil
		}
		t.logger.Warn("TypeScript LSP returned unexpected location format",
			zap.String("uri", uri),
			zap.Any("locations", locations))
		return nil, fmt.Errorf("unexpected location format from TypeScript LSP")
	}
	return nil, nil
}
*/

/*func (t *TypeScriptLanguageServerClient) GetReferences(ctx context.Context, uri string, position Position, includeDeclaration bool) (interface{}, error) {
	t.logger.Debug("Getting references", zap.String("uri", uri), zap.Int("line", position.Line), zap.Int("character", position.Character))

	if !t.initialized {
		return nil, fmt.Errorf("client not initialized")
	}

	params := ReferenceParams{
		TextDocumentPositionParams: TextDocumentPositionParams{
			TextDocument: TextDocumentIdentifier{URI: uri},
			Position:     position,
		},
		Context: ReferenceContext{
			IncludeDeclaration: includeDeclaration,
		},
	}

	resp, err := t.sendRequest(ctx, "textDocument/references", params)
	if err != nil {
		t.logger.Error("Failed to get references", zap.String("uri", uri), zap.Error(err))
		return nil, fmt.Errorf("failed to get references: %w", err)
	}

	return resp.Result, nil
}*/

/*func (t *TypeScriptLanguageServerClient) GetWorkspaceSymbols(ctx context.Context, query string) (interface{}, error) {
	t.logger.Info("Getting workspace symbols", zap.String("query", query))

	if !t.initialized {
		return nil, fmt.Errorf("client not initialized")
	}

	params := WorkspaceSymbolParams{
		Query: query,
	}

	resp, err := t.sendRequest(ctx, "workspace/symbol", params)
	if err != nil {
		t.logger.Error("Failed to get workspace symbols", zap.String("query", query), zap.Error(err))
		return nil, fmt.Errorf("failed to get workspace symbols: %w", err)
	}

	t.logger.Info("Workspace symbols response received",
		zap.String("query", query),
		zap.String("response_type", fmt.Sprintf("%T", resp.Result)))

	return resp.Result, nil
}*/

/*
type LSPJavaScriptLanguageServer struct {
	BaseClient
	rootPath string
	logger   *zap.Logger
}

func NewLSPJavaScriptLanguageServer(rootPath string, logger *zap.Logger) *LSPJavaScriptLanguageServer {
	return &LSPJavaScriptLanguageServer{rootPath: rootPath, logger: logger}
}

/*
func (j *LSPJavaScriptLanguageServer) ParseRepository(repoPath string) ([]model.FileInfo, []model.Function, error) {
	var files []model.FileInfo
	var functions []model.Function

	client, err := NewTypeScriptLanguageServerClient(j.logger)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create typescript-language-server client: %w", err)
	}
	defer client.Close()

	ctx := context.Background()
	if err := client.Initialize(ctx, repoPath); err != nil {
		return nil, nil, fmt.Errorf("failed to initialize typescript-language-server: %w", err)
	}
	defer client.Shutdown(ctx)

	err = filepath.Walk(repoPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if (!strings.HasSuffix(path, ".js") && !strings.HasSuffix(path, ".ts")) ||
			strings.Contains(path, "node_modules/") {
			return nil
		}

		language := "javascript"
		if strings.HasSuffix(path, ".ts") {
			language = "typescript"
		}

		files = append(files, model.FileInfo{
			Path:     path,
			Language: language,
		})

		uri := "file://" + path
		fileFunctions, parseErr := client.GetDocumentSymbols(ctx, uri, "")
		if parseErr != nil {
			return parseErr
		}

		functions = append(functions, fileFunctions...)
		return nil
	})

	return files, functions, err
}
*/
