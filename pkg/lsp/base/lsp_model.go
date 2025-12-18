package base

import "fmt"

type JSONRPCMessage struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      *int        `json:"id,omitempty"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
	Result  interface{} `json:"result,omitempty"`
	Error   *RPCError   `json:"error,omitempty"`
}

type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type InitializeParams struct {
	ProcessID    *int               `json:"processId"`
	RootPath     *string            `json:"rootPath,omitempty"`
	RootURI      *string            `json:"rootUri,omitempty"`
	Capabilities ClientCapabilities `json:"capabilities"`
}

type ClientCapabilities struct {
	TextDocument TextDocumentClientCapabilities `json:"textDocument"`
	Workspace    WorkspaceClientCapabilities    `json:"workspace"`
}

type ServerCapabilities struct {
	Capabilities map[string]interface{} `json:"capabilities"`
	// TextDocumentSync        TextDocumentSyncOptions `json:"textDocumentSync,omitempty"`
	TextDocumentSync        interface{} `json:"textDocumentSync,omitempty"`
	CompletionProvider      interface{} `json:"completionProvider,omitempty"`
	HoverProvider           bool        `json:"hoverProvider,omitempty"`
	SignatureHelpProvider   interface{} `json:"signatureHelpProvider,omitempty"`
	DefinitionProvider      bool        `json:"definitionProvider,omitempty"`
	ReferencesProvider      bool        `json:"referencesProvider,omitempty"`
	DocumentSymbolProvider  bool        `json:"documentSymbolProvider,omitempty"`
	WorkspaceSymbolProvider bool        `json:"workspaceSymbolProvider,omitempty"`
	CallHierarchyProvider   bool        `json:"callHierarchyProvider,omitempty"`
	ExecuteCommandProvider  interface{} `json:"executeCommandProvider,omitempty"`
	Workspace               interface{} `json:"workspace,omitempty"`
	// WorkspaceEdit          interface{} `json:"workspaceEdit,omitempty"`
	// CodeActionProvider     interface{} `json:"codeActionProvider,omitempty"`
	// CodeLensProvider       interface{} `json:"codeLensProvider,omitempty"`
	// DocumentFormattingProvider interface{} `json:"documentFormattingProvider,omitempty"`
	// DocumentRangeFormattingProvider interface{} `json:"documentRangeFormattingProvider,omitempty"`
	// DocumentOnTypeFormattingProvider interface{} `json:"documentOnTypeFormattingProvider,omitempty"`
	// RenameProvider         interface{} `json:"renameProvider,omitempty"`
	// FoldingRangeProvider   interface{} `json:"foldingRangeProvider,omitempty"`
	// SelectionRangeProvider interface{} `json:"selectionRangeProvider,omitempty"`
	// LinkedEditingRangeProvider interface{} `json:"linkedEditingRangeProvider,omitempty"`
	// ColorProvider          interface{} `json:"colorProvider,omitempty"`
	// DocumentLinkProvider   interface{} `json:"documentLinkProvider,omitempty"`
	// SemanticTokensProvider interface{} `json:"semanticTokensProvider,omitempty"`
	// MonikerProvider        interface{} `json:"monikerProvider,omitempty"`
	// InlineValueProvider    interface{} `json:"inlineValueProvider,omitempty"`
	// DiagnosticProvider     interface{} `json:"diagnosticProvider,omitempty"`
	// TypeHierarchyProvider  interface{} `json:"typeHierarchyProvider,omitempty"`
	// InlineCompletionProvider interface{} `json:"inlineCompletionProvider,omitempty"`
	// SemanticTokensOptions  interface{} `json:"semanticTokensOptions,omitempty"`
	// LinkedEditingRangeOptions interface{} `json:"linkedEditingRangeOptions,omitempty"`
	// ColorOptions           interface{} `json:"colorOptions,omitempty"`
	// DocumentLinkOptions    interface{} `json:"documentLinkOptions,omitempty"`
	// FoldingRangeOptions    interface{} `json:"foldingRangeOptions,omitempty"`
	// SelectionRangeOptions  interface{} `json:"selectionRangeOptions,omitempty"`
	// MonikerOptions         interface{} `json:"monikerOptions,omitempty"`
	// InlineValueOptions     interface{} `json:"inlineValueOptions,omitempty"`
	// TypeHierarchyOptions   interface{} `json:"typeHierarchyOptions,omitempty"`
	// InlineCompletionOptions interface{} `json:"inlineCompletionOptions,omitempty"`
}

type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type InitializeResult struct {
	Capabilities ServerCapabilities `json:"capabilities"`
	ServerInfo   *ServerInfo        `json:"serverInfo,omitempty"`
}

type TextDocumentClientCapabilities struct {
	DocumentSymbol DocumentSymbolClientCapabilities `json:"documentSymbol"`
	Hover          HoverClientCapabilities          `json:"hover"`
	SignatureHelp  SignatureHelpClientCapabilities  `json:"signatureHelp"`
	Definition     DefinitionClientCapabilities     `json:"definition"`
	References     ReferenceClientCapabilities      `json:"references"`
	CallHierarchy  CallHierarchyClientCapabilities  `json:"callHierarchy"`
}

type HoverClientCapabilities struct {
	DynamicRegistration bool     `json:"dynamicRegistration"`
	ContentFormat       []string `json:"contentFormat,omitempty"`
}

type SignatureHelpClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration"`
}

type DefinitionClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration"`
	LinkSupport         bool `json:"linkSupport"`
}

type ReferenceClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration"`
}

type CallHierarchyClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration"`
	IncomingCalls       bool `json:"incomingCalls"`
	OutgoingCalls       bool `json:"outgoingCalls"`
}

type DocumentSymbolClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration"`
}

type WorkspaceClientCapabilities struct {
	ApplyEdit              bool                                     `json:"applyEdit"`
	WorkspaceEdit          WorkspaceEditClientCapabilities          `json:"workspaceEdit"`
	DidChangeConfiguration DidChangeConfigurationClientCapabilities `json:"didChangeConfiguration"`
	DidChangeWatchedFiles  DidChangeWatchedFilesClientCapabilities  `json:"didChangeWatchedFiles"`
	Symbol                 WorkspaceSymbolClientCapabilities        `json:"symbol"`
	Configuration          bool                                     `json:"configuration"`
}

type WorkspaceEditClientCapabilities struct {
	DocumentChanges    bool     `json:"documentChanges"`
	ResourceOperations []string `json:"resourceOperations"`
}

type DidChangeConfigurationClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration"`
}

type DidChangeWatchedFilesClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration"`
}

type WorkspaceSymbolClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration"`
}

type DocumentSymbolParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}

type TextDocumentIdentifier struct {
	URI string `json:"uri"`
}

type CallHierarchyParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
}

type CallHierarchyItem struct {
	Name           string `json:"name"`
	Kind           int    `json:"kind"`
	URI            string `json:"uri"`
	Range          Range  `json:"range"`
	SelectionRange Range  `json:"selectionRange"`
}

type CallHierarchyIncomingCallsParams struct {
	Item CallHierarchyItem `json:"item"`
	//Item interface{} `json:"item"`
}

type CallHierarchyIncomingCall struct {
	From       CallHierarchyItem `json:"from"`
	FromRanges []Range           `json:"fromRanges"`
}

type CallHierarchyOutgoingCall struct {
	To         CallHierarchyItem `json:"to"`
	FromRanges []Range           `json:"FromRanges"`
}

type CallHierarchyIncomingOrgoingCalls struct {
	IncomingCalls []CallHierarchyIncomingCall `json:"incomingCalls,omitempty"`
	OutgoingCalls []CallHierarchyOutgoingCall `json:"outgoingCalls,omitempty"`
}

type DidOpenTextDocumentParams struct {
	TextDocument TextDocumentItem `json:"textDocument"`
}

type TextDocumentItem struct {
	URI        string `json:"uri"`
	LanguageId string `json:"languageId"`
	Version    int    `json:"version"`
	Text       string `json:"text"`
}

type TextDocumentPositionParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
}

type Position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

type Location struct {
	URI   string `json:"uri"`
	Range Range  `json:"range"`
}

type HoverParams struct {
	TextDocumentPositionParams
}

type MarkupContent struct {
	Kind  string `json:"kind"`
	Value string `json:"value"`
}

type Hover struct {
	Contents interface{} `json:"contents"`
	Range    *Range      `json:"range,omitempty"`
}

type SignatureHelpParams struct {
	TextDocumentPositionParams
}

type DefinitionParams struct {
	TextDocumentPositionParams
}

type ReferenceParams struct {
	TextDocumentPositionParams
	Context ReferenceContext `json:"context"`
}

type ReferenceContext struct {
	IncludeDeclaration bool `json:"includeDeclaration"`
}

type WorkspaceSymbolParams struct {
	Query string `json:"query"`
}

const (
	// list the different symbol kinds
	SymbolKindFile          = 1
	SymbolKindModule        = 2
	SymbolKindNamespace     = 3
	SymbolKindPackage       = 4
	SymbolKindClass         = 5
	SymbolKindMethod        = 6
	SymbolKindProperty      = 7
	SymbolKindField         = 8
	SymbolKindConstructor   = 9
	SymbolKindEnum          = 10
	SymbolKindInterface     = 11
	SymbolKindFunction      = 12
	SymbolKindVariable      = 13
	SymbolKindConstant      = 14
	SymbolKindString        = 15
	SymbolKindNumber        = 16
	SymbolKindBoolean       = 17
	SymbolKindArray         = 18
	SymbolKindObject        = 19
	SymbolKindKey           = 20
	SymbolKindNull          = 21
	SymbolKindEnumMember    = 22
	SymbolKindStruct        = 23
	SymbolKindEvent         = 24
	SymbolKindOperator      = 25
	SymbolKindTypeParameter = 26
	/*
		SymbolKindUser          = 27
		SymbolKindIssue         = 28
		SymbolKindSnippet       = 29
		SymbolKindFileSystem    = 30
		SymbolKindFolder        = 31
		SymbolKindLink          = 32
		SymbolKindSymbol        = 33
		SymbolKindText          = 34
		SymbolKindImage         = 35
		SymbolKindAudio         = 36
		SymbolKindVideo         = 37
		SymbolKindArchive       = 38
		SymbolKindExecutable    = 39
		SymbolKindConfiguration = 40
		SymbolKindLicense       = 41
		SymbolKindDocumentation = 42
		SymbolKindTest          = 43
		SymbolKindBenchmark     = 44
		SymbolKindDebug         = 45
		SymbolKindPerformance   = 46
		SymbolKindSecurity      = 47
		SymbolKindCompliance    = 48
		SymbolKindDeployment    = 49
		SymbolKindMonitoring    = 50
		SymbolKindAnalytics     = 51
	*/
)

type DocumentSymbol struct {
	Name           string           `json:"name"`
	Detail         string           `json:"detail,omitempty"`
	Kind           int              `json:"kind"`
	Range          Range            `json:"range"`
	SelectionRange Range            `json:"selectionRange"`
	Children       []DocumentSymbol `json:"children,omitempty"`
}

type SymbolInformation struct {
	Name     string   `json:"name"`
	Kind     int      `json:"kind"`
	Location Location `json:"location"`
}

func (r *Range) Contains(pos Position) bool {
	if pos.Line < r.Start.Line || pos.Line > r.End.Line {
		return false
	}
	if pos.Line == r.Start.Line && pos.Character < r.Start.Character {
		return false
	}
	if pos.Line == r.End.Line && pos.Character > r.End.Character {
		return false
	}
	return true
}

func (r *Range) ContainsRange(other *Range) bool {
	if r.Contains(other.Start) && r.Contains(other.End) {
		return true
	}
	return false
}

func MapToInitializeResult(data map[string]interface{}) (*InitializeResult, error) {
	capabilities, ok := data["capabilities"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid capabilities format")
	}

	serverInfoData, ok := data["serverInfo"].(map[string]interface{})
	var serverInfo *ServerInfo
	if ok {
		serverInfo = &ServerInfo{
			Name:    serverInfoData["name"].(string),
			Version: serverInfoData["version"].(string),
		}
	}

	return &InitializeResult{
		Capabilities: ServerCapabilities{Capabilities: capabilities},
		ServerInfo:   serverInfo,
	}, nil
}

func MapToDocumentSymbol(data map[string]interface{}) (*DocumentSymbol, error) {
	name, ok := data["name"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid name format")
	}

	detail, _ := data["detail"].(string)
	kind, ok := data["kind"].(float64)
	if !ok {
		return nil, fmt.Errorf("invalid kind format")
	}

	rangeData, ok := data["range"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid range format")
	}
	startData, ok := rangeData["start"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid start position format")
	}
	endData, ok := rangeData["end"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid end position format")
	}

	symbol := &DocumentSymbol{
		Name:   name,
		Detail: detail,
		Kind:   int(kind),
		Range: Range{
			Start: Position{
				Line:      int(startData["line"].(float64)),
				Character: int(startData["character"].(float64)),
			},
			End: Position{
				Line:      int(endData["line"].(float64)),
				Character: int(endData["character"].(float64)),
			},
		},
	}

	childrenData, ok := data["children"].([]interface{})
	if ok {
		for _, child := range childrenData {
			childMap, ok := child.(map[string]interface{})
			if ok {
				childSymbol, err := MapToDocumentSymbol(childMap)
				if err != nil {
					return nil, err
				}
				symbol.Children = append(symbol.Children, *childSymbol)
			}
		}
	}

	return symbol, nil
}

func MapToSymbolInformation(data map[string]interface{}) (*SymbolInformation, error) {
	name, ok := data["name"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid name format")
	}

	kind, ok := data["kind"].(float64)
	if !ok {
		return nil, fmt.Errorf("invalid kind format")
	}

	locationData, ok := data["location"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid location format")
	}

	uri, ok := locationData["uri"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid URI format")
	}

	rangeData, ok := locationData["range"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid range format")
	}
	startData, ok := rangeData["start"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid start position format")
	}
	endData, ok := rangeData["end"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid end position format")
	}

	return &SymbolInformation{
		Name: name,
		Kind: int(kind),
		Location: Location{
			URI: uri,
			Range: Range{
				Start: Position{
					Line:      int(startData["line"].(float64)),
					Character: int(startData["character"].(float64)),
				},
				End: Position{
					Line:      int(endData["line"].(float64)),
					Character: int(endData["character"].(float64)),
				},
			},
		},
	}, nil
}

func MapToDocumentSymbolOrSymbolInformation(data map[string]interface{}) (interface{}, error) {
	if _, ok := data["location"]; ok {
		return MapToSymbolInformation(data)
	} else {
		return MapToDocumentSymbol(data)
	}
}

func MapToCallHierarchyItem(data map[string]interface{}) (*CallHierarchyItem, error) {
	name, ok := data["name"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid name format")
	}

	kind, ok := data["kind"].(float64)
	if !ok {
		return nil, fmt.Errorf("invalid kind format")
	}

	uri, ok := data["uri"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid URI format")
	}

	rangeData, ok := data["range"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid range format")
	}
	startData, ok := rangeData["start"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid start position format")
	}
	endData, ok := rangeData["end"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid end position format")
	}

	selectionRangeData, ok := data["selectionRange"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid selectionRange format")
	}
	selStartData, ok := selectionRangeData["start"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid selectionRange start position format")
	}
	selEndData, ok := selectionRangeData["end"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid selectionRange end position format")
	}

	return &CallHierarchyItem{
		Name: name,
		Kind: int(kind),
		URI:  uri,
		Range: Range{
			Start: Position{
				Line:      int(startData["line"].(float64)),
				Character: int(startData["character"].(float64)),
			},
			End: Position{
				Line:      int(endData["line"].(float64)),
				Character: int(endData["character"].(float64)),
			},
		},
		SelectionRange: Range{
			Start: Position{
				Line:      int(selStartData["line"].(float64)),
				Character: int(selStartData["character"].(float64)),
			},
			End: Position{
				Line:      int(selEndData["line"].(float64)),
				Character: int(selEndData["character"].(float64)),
			},
		},
	}, nil
}

func MapToCallHierarchyIncomingCall(data map[string]interface{}) (*CallHierarchyIncomingCall, error) {
	fromData, ok := data["from"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid from format")
	}

	fromItem, err := MapToCallHierarchyItem(fromData)
	if err != nil {
		return nil, err
	}

	fromRangesData, ok := data["fromRanges"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid fromRanges format")
	}

	var fromRanges []Range
	for _, r := range fromRangesData {
		rangeData, ok := r.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid range format in fromRanges")
		}
		startData, ok := rangeData["start"].(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid start position format in fromRanges")
		}
		endData, ok := rangeData["end"].(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid end position format in fromRanges")
		}
		fromRanges = append(fromRanges, Range{
			Start: Position{
				Line:      int(startData["line"].(float64)),
				Character: int(startData["character"].(float64)),
			},
			End: Position{
				Line:      int(endData["line"].(float64)),
				Character: int(endData["character"].(float64)),
			},
		})
	}

	return &CallHierarchyIncomingCall{
		From:       *fromItem,
		FromRanges: fromRanges,
	}, nil
}

func MapToCallHierarchyOutgoingCall(data map[string]interface{}) (*CallHierarchyOutgoingCall, error) {
	toData, ok := data["to"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid to format")
	}

	toItem, err := MapToCallHierarchyItem(toData)
	if err != nil {
		return nil, err
	}

	fromRangesData, ok := data["fromRanges"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid fromRanges format")
	}

	var fromRanges []Range
	for _, r := range fromRangesData {
		rangeData, ok := r.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid range format in fromRanges")
		}
		startData, ok := rangeData["start"].(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid start position format in fromRanges")
		}
		endData, ok := rangeData["end"].(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid end position format in fromRanges")
		}
		fromRanges = append(fromRanges, Range{
			Start: Position{
				Line:      int(startData["line"].(float64)),
				Character: int(startData["character"].(float64)),
			},
			End: Position{
				Line:      int(endData["line"].(float64)),
				Character: int(endData["character"].(float64)),
			},
		})
	}

	return &CallHierarchyOutgoingCall{
		To:         *toItem,
		FromRanges: fromRanges,
	}, nil
}

func LocationToKey(loc *Location) string {
	return fmt.Sprintf("%s:%d:%d-%d:%d", loc.URI, loc.Range.Start.Line, loc.Range.Start.Character, loc.Range.End.Line, loc.Range.End.Character)
}
