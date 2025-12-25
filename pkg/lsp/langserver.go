package lsp

import (
	"github.com/armchr/codeapi/internal/config"
	"github.com/armchr/codeapi/pkg/lsp/base"
	"fmt"
	"strings"

	"go.uber.org/zap"
)

func NewLSPLanguageServer(config *config.Config, language, rootPath string, logger *zap.Logger) (base.LSPClient, error) {
	switch strings.ToLower(language) {
	case "go", "golang":
		return NewGoLanguageServerClient(config, rootPath, logger)
	case "java", "kotlin":
		return nil, fmt.Errorf("Java/Kotlin language server not implemented yet")
	case "csharp", "c#":
		return NewCSharpLanguageServerClient(config, rootPath, logger)
	case "ruby":
		return nil, fmt.Errorf("Ruby language server not implemented yet")
	case "php":
		return nil, fmt.Errorf("PHP language server not implemented yet")
	case "rust":
		return nil, fmt.Errorf("Rust language server not implemented yet")
	case "c", "cpp", "c++":
		return nil, fmt.Errorf("C/C++ language server not implemented yet")
	case "swift":
		return nil, fmt.Errorf("Swift language server not implemented yet")
	case "python", "py":
		return NewPythonLanguageServerClient(config, rootPath, logger)
	case "javascript", "js", "typescript", "ts":
		return NewTypeScriptLanguageServerClient(rootPath, logger)
	default:
		return nil, fmt.Errorf("unsupported language: %s", language)
	}
}
