package lsp

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/armchr/codeapi/internal/util"
	"github.com/armchr/codeapi/pkg/lsp/base"

	"go.uber.org/zap"
)

func (t *BaseClient) Initialize(ctx context.Context) (*base.InitializeResult, error) {
	rootPath := t.client.GetRootPath()
	t.logger.Info("Initializing language server client", zap.String("root_path", rootPath))

	params := base.InitializeParams{
		ProcessID: util.Ptr(os.Getpid()),
		RootPath:  util.Ptr(rootPath),
		RootURI:   util.Ptr("file://" + rootPath),
		Capabilities: base.ClientCapabilities{
			TextDocument: base.TextDocumentClientCapabilities{
				DocumentSymbol: base.DocumentSymbolClientCapabilities{
					DynamicRegistration: false,
				},
				Hover: base.HoverClientCapabilities{
					DynamicRegistration: false,
					ContentFormat:       []string{"markdown", "plaintext"},
				},
				/*SignatureHelp: SignatureHelpClientCapabilities{
					DynamicRegistration: false,
				},
				Definition: DefinitionClientCapabilities{
					DynamicRegistration: false,
					LinkSupport:         true,
				},
				References: ReferenceClientCapabilities{
					DynamicRegistration: false,
				},*/
				CallHierarchy: base.CallHierarchyClientCapabilities{
					DynamicRegistration: false,
				},
			},
			Workspace: base.WorkspaceClientCapabilities{
				ApplyEdit: false,
				WorkspaceEdit: base.WorkspaceEditClientCapabilities{
					DocumentChanges:    true,
					ResourceOperations: []string{"create", "rename", "delete"},
				},
				DidChangeConfiguration: base.DidChangeConfigurationClientCapabilities{
					DynamicRegistration: false,
				},
				DidChangeWatchedFiles: base.DidChangeWatchedFilesClientCapabilities{
					DynamicRegistration: false,
				},
				Symbol: base.WorkspaceSymbolClientCapabilities{
					DynamicRegistration: false,
				},
				Configuration: false,
			},
		},
	}

	t.logger.Debug("Sending initialize request to language server")
	resp, err := t.sendRequest(ctx, "initialize", params)
	if err != nil {
		t.logger.Error("Failed to initialize language server", zap.String("root_path", rootPath), zap.Error(err))
		return nil, fmt.Errorf("failed to initialize language server: %w", err)
	}

	// Send initialized notification to complete the handshake
	t.logger.Debug("Sending initialized notification to language server")
	if err := t.SendNotification("initialized", struct{}{}); err != nil {
		t.logger.Error("Failed to send initialized notification", zap.Error(err))
		return nil, fmt.Errorf("failed to send initialized notification: %w", err)
	}

	initResult, err := base.MapToInitializeResult(resp.Result.(map[string]interface{}))
	if err != nil {
		t.logger.Error("Language server did not return capabilities", zap.String("root_path", rootPath))
		return nil, fmt.Errorf("language server did not return capabilities")
	}

	t.initialized = true
	t.logger.Info("TypeScript language server initialized successfully", zap.String("root_path", rootPath))
	return initResult, nil
}

func (t *BaseClient) DidOpenFile(ctx context.Context, uri string) error {
	t.logger.Info("Opening file in TypeScript language server", zap.String("uri", uri))

	if !t.initialized {
		t.logger.Error("language server client not initialized", zap.String("uri", uri))
		return fmt.Errorf("client not initialized")
	}

	// Read file content
	defUri, _ := util.ToUri(uri, t.client.GetRootPath())
	filePath := strings.TrimPrefix(defUri, "file://")
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		t.logger.Error("Failed to read file", zap.String("file_path", filePath), zap.Error(err))
		return fmt.Errorf("failed to read file: %w", err)
	}

	fileHolder := base.NewFileHolder(uri, string(content))
	t.fileHolders[uri] = fileHolder

	params := base.DidOpenTextDocumentParams{
		TextDocument: base.TextDocumentItem{
			URI:        uri,
			LanguageId: t.client.LanguageID(uri),
			Version:    1,
			Text:       string(content),
		},
	}

	t.logger.Debug("Sending didOpen notification to language server", zap.String("uri", uri))
	if err := t.SendNotification("textDocument/didOpen", params); err != nil {
		t.logger.Error("Failed to send didOpen notification", zap.String("uri", uri), zap.Error(err))
		return fmt.Errorf("failed to send didOpen notification: %w", err)
	}

	t.logger.Info("File opened successfully in language server", zap.String("uri", uri))
	return nil
}

func (t *BaseClient) GetDocumentSymbols(ctx context.Context, uri string) ([]interface{}, error) {
	t.logger.Info("Getting document symbols from language server", zap.String("uri", uri))

	if !t.initialized {
		t.logger.Error("language server client not initialized", zap.String("uri", uri))
		return nil, fmt.Errorf("client not initialized")
	}

	// Read file content if not provided
	//t.DidOpenFile(ctx, uri)

	// Now request document symbols
	params := base.DocumentSymbolParams{
		TextDocument: base.TextDocumentIdentifier{
			URI: uri,
		},
	}

	t.logger.Debug("Requesting document symbols from language server", zap.String("uri", uri))
	resp, err := t.sendRequest(ctx, "textDocument/documentSymbol", params)
	if err != nil {
		t.logger.Error("Failed to get document symbols from language server", zap.String("uri", uri), zap.Error(err))
		return nil, fmt.Errorf("failed to get document symbols: %w", err)
	}

	t.logger.Debug("Raw LSP response", zap.String("uri", uri), zap.Any("result", resp.Result))
	/*
		functions := convertSymbolsToFunctions(resp.Result, uri)
		t.logger.Debug("Converted functions", zap.String("uri", uri), zap.Int("count", len(functions)), zap.Any("functions", functions))
		t.logger.Info("Successfully extracted functions from TypeScript language server", zap.String("uri", uri), zap.Int("function_count", len(functions)))
	*/

	// resp.Result is an array of DocumentSymbol or SymbolInformation
	if resp.Result == nil {
		t.logger.Warn("No symbols found in document", zap.String("uri", uri))
		return nil, nil
	}
	if symbols, ok := resp.Result.([]interface{}); ok {
		t.logger.Debug("Document symbols retrieved", zap.String("uri", uri), zap.Int("count", len(symbols)))
	} else {
		t.logger.Error("Unexpected response type for document symbols", zap.String("uri", uri), zap.Any("result", resp.Result))
		return nil, fmt.Errorf("unexpected response type for document symbols: %T", resp.Result)
	}

	// loop over symbols and convert to array of document symbols or symbol information
	var documentSymbols []interface{}
	documentSymbols = make([]interface{}, 0, len(resp.Result.([]interface{})))
	for _, sym := range resp.Result.([]interface{}) {
		mappedSym, _ := base.MapToDocumentSymbolOrSymbolInformation(sym.(map[string]interface{}))
		documentSymbols = append(documentSymbols, mappedSym)
	}
	return documentSymbols, nil
}

func (t *BaseClient) GetCallHierarchy(ctx context.Context, uri string, fnName string, position base.Position, inbound bool) (*base.CallHierarchyIncomingOrgoingCalls, error) {
	t.logger.Info("Getting call hierarchy from language server", zap.String("uri", uri))

	if !t.initialized {
		t.logger.Error("language server client not initialized", zap.String("uri", uri))
		return nil, fmt.Errorf("client not initialized")
	}

	fileHolder := t.fileHolders[uri]
	if fileHolder == nil {
		t.logger.Error("file not opened in language server", zap.String("uri", uri))
		return nil, fmt.Errorf("file not opened in language server")
	}

	// If fnName is provided, try to find its position in the file
	if fnName != "" {
		startLine := position.Line
		//TODO this is flawed, as the function name might just be used in a comment or string before its actual definition
		// ideally we should parse the file to find the actual function definition
		// but for now, we will just search in the next 50 lines from the given line
		foundLine, foundChar := fileHolder.FindNameInNextLines(t.client, fnName, startLine, 50)
		if foundLine == -1 {
			t.logger.Error("function name not found in file", zap.String("uri", uri), zap.String("function_name", fnName))
			return nil, fmt.Errorf("function name not found in file")
		}
		position.Line = foundLine
		position.Character = foundChar
	}

	params := base.CallHierarchyParams{
		TextDocument: base.TextDocumentIdentifier{
			URI: uri,
		},
		Position: position,
	}

	t.logger.Debug("Requesting call hierarchy from TypeScript language server", zap.String("uri", uri))
	resp, err := t.sendRequest(ctx, "textDocument/prepareCallHierarchy", params)
	if err != nil {
		t.logger.Error("Failed to get call hierarchy from TypeScript language server", zap.String("uri", uri), zap.Error(err))
		return nil, fmt.Errorf("failed to get call hierarchy: %w", err)
	}

	// resp.Result is an array of maps that needs to be converted to array of CallHierarchyItem
	var callItems []base.CallHierarchyItem
	if resp.Result == nil {
		t.logger.Warn("No call hierarchy items found in document", zap.String("uri", uri))
		return nil, nil
	}
	if items, ok := resp.Result.([]interface{}); ok {
		t.logger.Debug("Call hierarchy items retrieved", zap.String("uri", uri), zap.Int("count", len(items)))
		callItems = make([]base.CallHierarchyItem, 0, len(items))
		for _, item := range items {
			if itemMap, ok := item.(map[string]interface{}); ok {
				if chi, err := base.MapToCallHierarchyItem(itemMap); err == nil {
					callItems = append(callItems, *chi)
				} else {
					t.logger.Error("Failed to map call hierarchy item", zap.String("uri", uri), zap.Any("item", item), zap.Error(err))
				}
			} else {
				t.logger.Error("Unexpected item type in call hierarchy items", zap.String("uri", uri), zap.Any("item", item))
			}
		}
	} else {
		t.logger.Error("Unexpected response type for call hierarchy items", zap.String("uri", uri), zap.Any("result", resp.Result))
		return nil, fmt.Errorf("unexpected response type for call hierarchy items: %T", resp.Result)
	}

	var command string
	if inbound {
		command = "callHierarchy/incomingCalls"
	} else {
		command = "callHierarchy/outgoingCalls"
	}

	var callsParams []base.CallHierarchyIncomingCallsParams
	// for each call item, create a CallHierarchyIncomingCallsParams
	callsParams = make([]base.CallHierarchyIncomingCallsParams, 0, len(callItems))
	for _, item := range callItems {
		callsParams = append(callsParams, base.CallHierarchyIncomingCallsParams{
			Item: item,
		})
	}

	results := util.DoWorkList(callsParams, func(param base.CallHierarchyIncomingCallsParams) interface{} {
		callsResp, err := t.sendRequest(ctx, command, param)
		if err != nil {
			t.logger.Error("Failed to get calls from language server", zap.String("uri", uri), zap.Any("item", param.Item), zap.Error(err))
			return nil
		}
		return callsResp.Result
	})

	if len(results) == 0 {
		return nil, nil
	}

	// flatten results
	var refList []interface{}
	for _, res := range results {
		if res != nil {
			if refs, ok := res.([]interface{}); ok {
				refList = append(refList, refs...)
			} else {
				t.logger.Error("Unexpected result type for calls", zap.String("uri", uri), zap.Any("result", res))
			}
		}
	}

	var incommingCalls []base.CallHierarchyIncomingCall = nil
	var outgoingCalls []base.CallHierarchyOutgoingCall = nil

	if inbound {
		incommingCalls = make([]base.CallHierarchyIncomingCall, 0, len(refList))
		for _, ref := range refList {
			if refMap, ok := ref.(map[string]interface{}); ok {
				if call, err := base.MapToCallHierarchyIncomingCall(refMap); err == nil {
					incommingCalls = append(incommingCalls, *call)
				} else {
					t.logger.Error("Failed to map incoming call", zap.String("uri", uri), zap.Any("ref", ref), zap.Error(err))
				}
			} else {
				t.logger.Error("Unexpected reference type in incoming calls", zap.String("uri", uri), zap.Any("ref", ref))
			}
		}
		t.logger.Debug("Incoming calls retrieved", zap.String("uri", uri), zap.Int("count", len(incommingCalls)))
	} else {
		outgoingCalls = make([]base.CallHierarchyOutgoingCall, 0, len(refList))
		for _, ref := range refList {

			if refMap, ok := ref.(map[string]interface{}); ok {
				if call, err := base.MapToCallHierarchyOutgoingCall(refMap); err == nil {
					outgoingCalls = append(outgoingCalls, *call)
				} else {
					t.logger.Error("Failed to map outgoing call", zap.String("uri", uri), zap.Any("ref", ref), zap.Error(err))
				}
			} else {
				t.logger.Error("Unexpected reference type in outgoing calls", zap.String("uri", uri), zap.Any("ref", ref))
			}
		}
		t.logger.Debug("Outgoing calls retrieved", zap.String("uri", uri), zap.Int("count", len(outgoingCalls)))
	}
	t.logger.Debug("Raw LSP response for call hierarchy", zap.String("uri", uri))
	return &base.CallHierarchyIncomingOrgoingCalls{
		IncomingCalls: incommingCalls,
		OutgoingCalls: outgoingCalls,
	}, nil
}

func (t *BaseClient) GetHover(ctx context.Context, uri string, position base.Position) (*base.Hover, error) {
	t.logger.Info("Getting hover information from language server", zap.String("uri", uri))

	if !t.initialized {
		t.logger.Error("language server client not initialized", zap.String("uri", uri))
		return nil, fmt.Errorf("client not initialized")
	}

	params := base.HoverParams{
		TextDocumentPositionParams: base.TextDocumentPositionParams{
			TextDocument: base.TextDocumentIdentifier{
				URI: uri,
			},
			Position: position,
		},
	}

	t.logger.Debug("Requesting hover information from language server", zap.String("uri", uri))
	resp, err := t.sendRequest(ctx, "textDocument/hover", params)
	if err != nil {
		t.logger.Error("Failed to get hover information from language server", zap.String("uri", uri), zap.Error(err))
		return nil, fmt.Errorf("failed to get hover information: %w", err)
	}

	if resp.Result == nil {
		t.logger.Debug("No hover information found", zap.String("uri", uri))
		return nil, nil
	}

	resultMap, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.logger.Error("Unexpected response type for hover", zap.String("uri", uri), zap.Any("result", resp.Result))
		return nil, fmt.Errorf("unexpected response type for hover: %T", resp.Result)
	}

	hover := &base.Hover{
		Contents: resultMap["contents"],
	}

	if rangeData, ok := resultMap["range"].(map[string]interface{}); ok {
		startData, _ := rangeData["start"].(map[string]interface{})
		endData, _ := rangeData["end"].(map[string]interface{})

		if startData != nil && endData != nil {
			hover.Range = &base.Range{
				Start: base.Position{
					Line:      int(startData["line"].(float64)),
					Character: int(startData["character"].(float64)),
				},
				End: base.Position{
					Line:      int(endData["line"].(float64)),
					Character: int(endData["character"].(float64)),
				},
			}
		}
	}

	t.logger.Debug("Hover information retrieved successfully", zap.String("uri", uri))
	return hover, nil
}
