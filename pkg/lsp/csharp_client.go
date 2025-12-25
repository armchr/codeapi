package lsp

import (
	"fmt"
	"strings"

	"github.com/armchr/codeapi/internal/config"
	"github.com/armchr/codeapi/pkg/lsp/base"

	"go.uber.org/zap"
)

// CSharpLanguageServerClient wraps the base LSP client for C# specific functionality
type CSharpLanguageServerClient struct {
	*BaseClient
	rootPath string
	logger   *zap.Logger
}

// NewCSharpLanguageServerClient creates a new C# language server client
func NewCSharpLanguageServerClient(config *config.Config, rootPath string, logger *zap.Logger) (*CSharpLanguageServerClient, error) {
	logger.Info("Creating new C# language server client")
	lspPath := config.LanguageServers.GetLSPPath("csharp")
	if lspPath == "" {
		return nil, fmt.Errorf("no language server configured for C#")
	}
	baseClient, err := NewBaseClient(lspPath, logger)
	if err != nil {
		return nil, err
	}

	t := &CSharpLanguageServerClient{BaseClient: baseClient, rootPath: rootPath, logger: logger}
	t.client = t
	return t, nil
}

// GetRootPath returns the root path for the C# project
func (t *CSharpLanguageServerClient) GetRootPath() string {
	return t.rootPath
}

// LanguageID returns the language identifier for LSP based on file extension
func (t *CSharpLanguageServerClient) LanguageID(uri string) string {
	if strings.HasSuffix(uri, ".cs") {
		return "csharp"
	}
	return "unknown"
}

// IsExternalModule checks if the given URI points to an external module
// For C#, this includes NuGet packages and .NET SDK assemblies
func (t *CSharpLanguageServerClient) IsExternalModule(uri string) bool {
	// NuGet packages cache locations
	if strings.Contains(uri, ".nuget/packages/") ||
		strings.Contains(uri, "/.nuget/") ||
		strings.Contains(uri, "/packages/") {
		return true
	}

	// .NET SDK and runtime locations
	if strings.Contains(uri, "/dotnet/") ||
		strings.Contains(uri, "/Microsoft.NETCore.App/") ||
		strings.Contains(uri, "/Microsoft.AspNetCore.App/") {
		return true
	}

	// Check if file is outside the root path
	if strings.HasPrefix(uri, "file://") && !strings.HasPrefix(uri, "file://"+t.rootPath) {
		return true
	}

	return false
}

// MatchSymbolByName matches C# symbol names
// In C#, fully qualified names use dot notation (e.g., Namespace.Class.Method)
func (t *CSharpLanguageServerClient) MatchSymbolByName(name, nameInFile string) bool {
	return base.MatchLastSegment(name, nameInFile, ".")
}

// SymbolPartToMatch returns the part of the symbol name to use for matching
func (t *CSharpLanguageServerClient) SymbolPartToMatch(name string) string {
	return base.LastSegment(name)
}
