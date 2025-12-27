package lsp

import (
	"fmt"
	"strings"

	"github.com/armchr/codeapi/internal/config"
	"github.com/armchr/codeapi/pkg/lsp/base"

	"go.uber.org/zap"
)

// JavaLanguageServerClient wraps the base LSP client for Java specific functionality
type JavaLanguageServerClient struct {
	*BaseClient
	rootPath string
	logger   *zap.Logger
}

// NewJavaLanguageServerClient creates a new Java language server client (Eclipse JDT.LS)
func NewJavaLanguageServerClient(config *config.Config, rootPath string, logger *zap.Logger) (*JavaLanguageServerClient, error) {
	logger.Info("Creating new Java language server client (Eclipse JDT.LS)")
	lspPath := config.LanguageServers.GetLSPPath("java")
	if lspPath == "" {
		return nil, fmt.Errorf("no language server configured for Java")
	}
	baseClient, err := NewBaseClient(lspPath, logger)
	if err != nil {
		return nil, err
	}

	t := &JavaLanguageServerClient{BaseClient: baseClient, rootPath: rootPath, logger: logger}
	t.client = t
	return t, nil
}

// GetRootPath returns the root path for the Java project
func (t *JavaLanguageServerClient) GetRootPath() string {
	return t.rootPath
}

// LanguageID returns the language identifier for LSP based on file extension
func (t *JavaLanguageServerClient) LanguageID(uri string) string {
	if strings.HasSuffix(uri, ".java") {
		return "java"
	}
	if strings.HasSuffix(uri, ".kt") || strings.HasSuffix(uri, ".kts") {
		return "kotlin"
	}
	return "unknown"
}

// IsExternalModule checks if the given URI points to an external module
// For Java, this includes Maven/Gradle dependencies and JDK classes
func (t *JavaLanguageServerClient) IsExternalModule(uri string) bool {
	// Maven local repository
	if strings.Contains(uri, ".m2/repository/") ||
		strings.Contains(uri, "/.m2/") {
		return true
	}

	// Gradle cache locations
	if strings.Contains(uri, ".gradle/caches/") ||
		strings.Contains(uri, "/.gradle/") {
		return true
	}

	// Build output directories (compiled dependencies)
	if strings.Contains(uri, "/target/") ||
		strings.Contains(uri, "/build/") ||
		strings.Contains(uri, "/out/") {
		return true
	}

	// JDK/JRE locations
	if strings.Contains(uri, "/jdk") ||
		strings.Contains(uri, "/jre") ||
		strings.Contains(uri, "/java/") ||
		strings.Contains(uri, "rt.jar") {
		return true
	}

	// Eclipse JDT.LS workspace/cache
	if strings.Contains(uri, "jdt.ls") ||
		strings.Contains(uri, "jdt-language-server") {
		return true
	}

	// Check if file is outside the root path
	if strings.HasPrefix(uri, "file://") && !strings.HasPrefix(uri, "file://"+t.rootPath) {
		return true
	}

	return false
}

// MatchSymbolByName matches Java symbol names
// In Java, fully qualified names use dot notation (e.g., com.example.MyClass.myMethod)
func (t *JavaLanguageServerClient) MatchSymbolByName(name, nameInFile string) bool {
	return base.MatchLastSegment(name, nameInFile, ".")
}

// SymbolPartToMatch returns the part of the symbol name to use for matching
func (t *JavaLanguageServerClient) SymbolPartToMatch(name string) string {
	return base.LastSegment(name)
}
