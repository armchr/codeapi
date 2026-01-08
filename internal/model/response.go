package model

import (
	"github.com/armchr/codeapi/pkg/lsp/base"
)

type ProcessRepoResponse struct {
	RepoName  string               `json:"repo_name"`
	Files     []FileInfo           `json:"files"`
	Functions []FunctionDefinition `json:"functions"`
}

type GetFunctionsInFileRequest struct {
	RepoName     string `json:"repo_name" binding:"required"`
	RelativePath string `json:"relative_path" binding:"required"`
}

type GetFunctionsInFileResponse struct {
	RepoName  string               `json:"repo_name"`
	FilePath  string               `json:"file_path"`
	Functions []FunctionDefinition `json:"functions"`
}

type GetFunctionDetailsRequest struct {
	RepoName     string `json:"repo_name" binding:"required"`
	RelativePath string `json:"relative_path" binding:"required"`
	FunctionName string `json:"function_name" binding:"required"`
}

type GetFunctionDetailsResponse struct {
	RepoName     string          `json:"repo_name"`
	FilePath     string          `json:"file_path"`
	FunctionName string          `json:"function_name"`
	Details      FunctionDetails `json:"details"`
}

type FunctionDetails struct {
	Name          string        `json:"name"`
	Signature     string        `json:"signature"`
	Parameters    []Parameter   `json:"parameters"`
	ReturnType    string        `json:"return_type"`
	IsAsync       bool          `json:"is_async"`
	Documentation string        `json:"documentation"`
	Location      base.Location `json:"location"`
}

type Parameter struct {
	Name          string `json:"name"`
	Type          string `json:"type"`
	Optional      bool   `json:"optional"`
	Documentation string `json:"documentation"`
}

type GetFunctionDependenciesRequest struct {
	RepoName     string `json:"repo_name" binding:"required"`
	RelativePath string `json:"relative_path" binding:"required"`
	FunctionName string `json:"function_name" binding:"required"`
	Depth        int    `json:"depth"`
}

type GetFunctionDependenciesResponse struct {
	RepoName     string               `json:"repo_name"`
	FilePath     string               `json:"file_path"`
	FunctionName string               `json:"function_name"`
	Dependencies []FunctionDependency `json:"dependencies"`
}

type FunctionDependency struct {
	Name          string             `json:"name"`
	CallLocations []base.Location    `json:"call_locations"`
	Definition    FunctionDefinition `json:"definition"`
}

type FunctionDefinition struct {
	Name       string        `json:"name"`
	Location   base.Location `json:"location"`
	IsExternal bool          `json:"is_external"`
	Module     string        `json:"module,omitempty"`
	Params     string        `json:"params"`
	Returns    string        `json:"returns"`
}

type CallGraph struct {
	Roots        []FunctionDefinition           `json:"roots"`
	Functions    []FunctionDefinition           `json:"functions"`
	Edges        []CallEdge                     `json:"edges"`
	functionsMap map[string]*FunctionDefinition `json:"-"`
	edgesMap     map[string]*CallEdge           `json:"-"`
}

type CallEdge struct {
	From *FunctionDefinition `json:"from"`
	To   *FunctionDefinition `json:"to"`
}

type FileInfo struct {
	Path     string `json:"path"`
	Language string `json:"language"`
}

func NewCallGraph() *CallGraph {
	return &CallGraph{
		Roots:        []FunctionDefinition{},
		Functions:    []FunctionDefinition{},
		Edges:        []CallEdge{},
		functionsMap: make(map[string]*FunctionDefinition),
		edgesMap:     make(map[string]*CallEdge),
	}
}

func (fn *FunctionDefinition) ToKey() string {
	return fn.Name + ":" + base.LocationToKey(&fn.Location)
}

func (e CallEdge) ToKey() string {
	var fromKey, toKey string
	if e.From != nil {
		fromKey = e.From.ToKey()
	}
	if e.To != nil {
		toKey = e.To.ToKey()
	}
	return fromKey + "->" + toKey
}

/*
func (cg *CallGraph) Merge(other *CallGraph) {
	// Merge functions, ensuring no duplicates using ToKey
	existingKeys := make(map[string]FunctionDefinition)
	for _, fn := range cg.Functions {
		existingKeys[fn.ToKey()] = fn
	}
	for _, fn := range other.Functions {
		if _, exists := existingKeys[fn.ToKey()]; !exists {
			cg.Functions = append(cg.Functions, fn)
			existingKeys[fn.ToKey()] = fn
		}
	}

	// Merge edges, ensuring no duplicates using From/To ToKey
	existingEdges := make(map[string]*CallEdge)
	for _, e := range cg.Edges {
		existingEdges[e.ToKey()] = &e
	}
	for _, e := range other.Edges {
		k := e.ToKey()
		if _, exists := existingEdges[k]; !exists {
			cg.Edges = append(cg.Edges, e)
			existingEdges[k] = &e
		}
	}
}
*/

func (cg *CallGraph) AddFunctionDependency(caller *FunctionDefinition, dep *FunctionDependency) {
	// Add the function definition if not already present
	defKey := dep.Definition.ToKey()
	if _, exists := cg.functionsMap[defKey]; !exists {
		cg.Functions = append(cg.Functions, dep.Definition)
		cg.functionsMap[defKey] = &dep.Definition
	}

	edge := CallEdge{
		From: caller, // If you have the caller function, set it here
		To:   &dep.Definition,
	}
	edgeKey := edge.ToKey()
	if _, exists := cg.edgesMap[edgeKey]; !exists {
		cg.Edges = append(cg.Edges, edge)
		cg.edgesMap[edgeKey] = &edge
	}
	// Add edges for each call location, ensuring uniqueness
	/*for _, loc := range dep.CallLocations {
	}*/
}

func MapToFunctionFromSymbolInformation(uri string, sym *base.SymbolInformation) FunctionDefinition {
	return FunctionDefinition{
		Name: sym.Name,
		Location: base.Location{
			URI: uri,
			Range: base.Range{
				Start: base.Position{
					Line:      sym.Location.Range.Start.Line,
					Character: sym.Location.Range.Start.Character,
				},
				End: base.Position{
					Line:      sym.Location.Range.End.Line,
					Character: sym.Location.Range.End.Character,
				},
			},
		},
	}
}

func MapToFunctionFromDocumentSymbol(uri string, sym *base.DocumentSymbol) FunctionDefinition {
	return FunctionDefinition{
		Name: sym.Name,
		Location: base.Location{
			URI: uri,
			Range: base.Range{
				Start: base.Position{
					Line:      sym.Range.Start.Line,
					Character: sym.Range.Start.Character,
				},
				End: base.Position{
					Line:      sym.Range.End.Line,
					Character: sym.Range.End.Character,
				},
			},
		},
	}
}

func MapToFunctionDependency(call base.CallHierarchyOutgoingCall, lspClient base.LSPClient) FunctionDependency {
	callLocations := make([]base.Location, 0, 1)
	// call.FromRanges to callLocations
	for _, r := range call.FromRanges {
		callLocations = append(callLocations, base.Location{
			URI: "",
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

	// Extract clean method name from LSP signature
	// Java LSP returns signatures like "findByOwnerId(Long) : List<PetDto>"
	// We need just "findByOwnerId" to match against tree-sitter parsed names
	methodName := base.ExtractJavaMethodName(call.To.Name)

	return FunctionDependency{
		Name:          methodName,
		CallLocations: callLocations,
		Definition: FunctionDefinition{
			Name: methodName,
			Location: base.Location{
				URI: call.To.URI,
				Range: base.Range{
					Start: base.Position{
						Line:      call.To.Range.Start.Line,
						Character: call.To.Range.Start.Character,
					},
					End: base.Position{
						Line:      call.To.Range.End.Line,
						Character: call.To.Range.End.Character,
					},
				},
			},
			IsExternal: lspClient.IsExternalModule(call.To.URI), // This could be determined with more context
			Module:     "",                                      // This could be filled in with more context
		},
	}
}

func FunctionDefinitionToKey(def FunctionDefinition) string {
	return def.Location.URI + ":" + def.Name + ":" + base.LocationToKey(&def.Location)
}

type ProcessDirectoryRequest struct {
	RepoName       string `json:"repo_name" binding:"required"`
	CollectionName string `json:"collection_name"`
}

type ProcessDirectoryResponse struct {
	RepoName       string `json:"repo_name"`
	CollectionName string `json:"collection_name"`
	TotalChunks    int    `json:"total_chunks"`
	Success        bool   `json:"success"`
	Message        string `json:"message,omitempty"`
}

type SearchSimilarCodeRequest struct {
	RepoName       string `json:"repo_name" binding:"required"`
	CollectionName string `json:"collection_name"`
	CodeSnippet    string `json:"code_snippet" binding:"required"`
	Language       string `json:"language" binding:"required"`
	Limit          int    `json:"limit"`
	IncludeCode    bool   `json:"include_code"`
}

type SearchSimilarCodeResponse struct {
	RepoName       string              `json:"repo_name"`
	CollectionName string              `json:"collection_name"`
	Query          QueryInfo           `json:"query"`
	Results        []SimilarCodeResult `json:"results"`
	Success        bool                `json:"success"`
	Message        string              `json:"message,omitempty"`
}

type QueryInfo struct {
	CodeSnippet string       `json:"code_snippet"`
	Language    string       `json:"language"`
	ChunksFound int          `json:"chunks_found"`
	Chunks      []*CodeChunk `json:"chunks"` // The parsed chunks from the input snippet
}

type SimilarCodeResult struct {
	Chunk           *CodeChunk `json:"chunk"`
	Score           float32    `json:"score"`
	QueryChunkIndex int        `json:"query_chunk_index"` // Index of the input chunk that matched this result (0-based)
	Code            string     `json:"code,omitempty"`    // Actual code content from file (if include_code is true)
}

func (fd *FunctionDependency) IsIn(rng *base.Range) bool {
	for _, loc := range fd.CallLocations {
		if rng.ContainsRange(&loc.Range) {
			return true
		}
	}
	return false
}
