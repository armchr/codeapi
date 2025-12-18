package base

import (
	"context"
)

type LSPClient interface {
	GetRootPath() string
	LanguageID(uri string) string
	IsExternalModule(uri string) bool

	MatchSymbolByName(name, nameInFile string) bool
	SymbolPartToMatch(name string) string

	Initialize(ctx context.Context) (*InitializeResult, error)
	Shutdown(ctx context.Context) error
	Close() error

	DidOpenFile(ctx context.Context, uri string) error
	GetDocumentSymbols(ctx context.Context, uri string) ([]interface{}, error)
	GetCallHierarchy(ctx context.Context, uri string, fnName string, position Position, inbound bool) (*CallHierarchyIncomingOrgoingCalls, error)
	GetHover(ctx context.Context, uri string, position Position) (*Hover, error)
	//GetFunctionsInFile(ctx context.Context, uri string) ([]model.Function, error)

	/*
		GetDefinition(ctx context.Context, uri, text string, position model.Position) (*model.Location, error)
		GetReferences(ctx context.Context, uri, text string, position model.Position) ([]model.Location, error)

		GetFunctionsInFile(ctx context.Context, request model.GetFunctionsInFileRequest) (*model.GetFunctionsInFileResponse, error)
	*/
}
