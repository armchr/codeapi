package parse

import (
	"github.com/armchr/codeapi/internal/config"
	"github.com/armchr/codeapi/internal/model/ast"
	"github.com/armchr/codeapi/internal/service/codegraph"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"
	golang "github.com/tree-sitter/tree-sitter-go/bindings/go"
	java "github.com/tree-sitter/tree-sitter-java/bindings/go"
	javascript "github.com/tree-sitter/tree-sitter-javascript/bindings/go"
	python "github.com/tree-sitter/tree-sitter-python/bindings/go"
	typescript "github.com/tree-sitter/tree-sitter-typescript/bindings/go"
	"go.uber.org/zap"
)

type LanguageType int

const (
	Go LanguageType = iota
	JavaScript
	TypeScript
	Python
	Java
	Unknown
)

type FileParser struct {
	parser    *tree_sitter.Parser
	CodeGraph *codegraph.CodeGraph
	logger    *zap.Logger
	Config    *config.Config
}

func (lt LanguageType) String() string {
	switch lt {
	case Go:
		return "go"
	case JavaScript:
		return "javascript"
	case TypeScript:
		return "typescript"
	case Python:
		return "python"
	case Java:
		return "java"
	default:
		return "unknown"
	}
}

func NewLanguageTypeFromString(lang string) LanguageType {
	switch strings.ToLower(lang) {
	case "go":
		return Go
	case "javascript":
		return JavaScript
	case "typescript":
		return TypeScript
	case "python":
		return Python
	case "java":
		return Java
	default:
		return Unknown
	}
}

func NewFileParser(logger *zap.Logger, cg *codegraph.CodeGraph, cfg *config.Config) *FileParser {
	return &FileParser{
		parser:    tree_sitter.NewParser(),
		CodeGraph: cg,
		logger:    logger,
		Config:    cfg,
	}
}

func (fp *FileParser) DetectLanguage(filePath string) LanguageType {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".go":
		return Go
	case ".js", ".jsx", ".mjs":
		return JavaScript
	case ".ts", ".tsx":
		return TypeScript
	case ".py", ".pyw":
		return Python
	case ".java":
		return Java
	default:
		return Unknown
	}
}

func (fp *FileParser) GetLanguageParser(langType LanguageType) (*tree_sitter.Language, error) {
	switch langType {
	case Go:
		return tree_sitter.NewLanguage(golang.Language()), nil
	case JavaScript:
		return tree_sitter.NewLanguage(javascript.Language()), nil
	case TypeScript:
		return tree_sitter.NewLanguage(typescript.LanguageTypescript()), nil
	case Python:
		return tree_sitter.NewLanguage(python.Language()), nil
	case Java:
		return tree_sitter.NewLanguage(java.Language()), nil
	default:
		return nil, fmt.Errorf("unsupported language type: %v", langType)
	}
}

func (fp *FileParser) GetLanguageVisitor(langType LanguageType, ts *TranslateFromSyntaxTree) (SyntaxTreeVisitor, error) {
	switch langType {
	case Go:
		return NewGoVisitor(fp.logger, ts), nil
		//return NewPrintVisitor(fp.logger, ts), nil
	/*
		case JavaScript:
				return NewJavaScriptVisitor(ts), nil
			case TypeScript:
				return NewTypeScriptVisitor(ts), nil
				case Java:
				return NewJavaVisitor(ts), nil
	*/
	case Python:
		return NewPythonVisitor(fp.logger, ts), nil
		//return NewPrintVisitor(fp.logger, ts), nil

	case JavaScript, TypeScript:
		return NewPrintVisitor(ts), nil

	default:
		return nil, fmt.Errorf("unsupported language type: %v", langType)
	}
}

func (fp *FileParser) CreateTranslator(ctx context.Context, filePath string, fileID int32, langType LanguageType, version int32) (*tree_sitter.Tree, *TranslateFromSyntaxTree, error) {
	content, err := fp.ReadFile(filePath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	return fp.CreateTranslatorWithContent(ctx, filePath, fileID, langType, version, content)
}

func (fp *FileParser) CreateTranslatorWithContent(ctx context.Context, filePath string, fileID int32, langType LanguageType, version int32, content []byte) (*tree_sitter.Tree, *TranslateFromSyntaxTree, error) {
	language, err := fp.GetLanguageParser(langType)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get language parser: %w", err)
	}

	err = fp.parser.SetLanguage(language)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to set parser language: %w", err)
	}

	tree := fp.parser.Parse(content, nil)
	if tree == nil {
		return nil, nil, fmt.Errorf("failed to parse file: %s", filePath)
	}

	translator := NewTranslateFromSyntaxTree(fileID, version, fp.CodeGraph, content, fp.logger)
	return tree, translator, nil
}

func (fp *FileParser) ReadFile(filePath string) ([]byte, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	return content, nil
}

func (fp *FileParser) relativePath(repo *config.Repository, fullPath string) string {
	relPath, err := filepath.Rel(repo.Path, fullPath)
	if err != nil {
		return fullPath
	}
	return relPath
}

/*
func (fp *FileParser) ParseAndTraverse(ctx context.Context, repo *config.Repository, info os.FileInfo, filePath string, fileID int32, version int32) error {
	content, err := fp.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", filePath, err)
	}
	return fp.ParseAndTraverseWithContent(ctx, repo, info, filePath, fileID, version, content)
}
*/

func (fp *FileParser) ParseAndTraverseWithContent(ctx context.Context, repo *config.Repository, info os.FileInfo, filePath string, fileID int32, version int32, content []byte) error {
	languageType := fp.DetectLanguage(filePath)
	if languageType == Unknown {
		return fmt.Errorf("unsupported file type for file: %s", filePath)
	}
	tree, translator, err := fp.CreateTranslatorWithContent(ctx, filePath, fileID, languageType, version, content)
	if err != nil {
		return err
	}
	defer tree.Close()

	rootNode := tree.RootNode()
	if rootNode == nil {
		return fmt.Errorf("no root node found in parsed tree")
	}

	visitor, err := fp.GetLanguageVisitor(languageType, translator)
	if err != nil {
		return err
	}
	translator.Visitor = visitor

	fileScope := ast.NewNode(
		ast.NodeID(fileID), ast.NodeTypeFileScope,
		translator.FileID,
		translator.GetTreeNodeName(rootNode),
		translator.ToRange(rootNode),
		translator.Version, ast.InvalidNodeID,
	)

	fileScope.MetaData = map[string]any{
		"repo":     repo.Name,
		"path":     fp.relativePath(repo, filePath),
		"modified": info.ModTime().Unix(),
		"language": languageType.String(),
	}

	fp.CodeGraph.CreateFileScope(ctx, fileScope)

	rootNodeId := visitor.TraverseNode(ctx, rootNode, fileScope.ID)
	if rootNodeId != ast.InvalidNodeID {
		fp.CodeGraph.CreateContainsRelation(ctx, fileScope.ID, rootNodeId, fileID)
	}

	if fp.Config.CodeGraph.PrintParseTree {
		content := PrintSyntaxTree(ctx, rootNode, translator.FileContent)
		fp.logger.Info("Syntax Tree: " + filePath + "\n" + content)
	}
	return nil
}

func (fp *FileParser) ShouldSkipFile(ctx context.Context, repo *config.Repository, info os.FileInfo, filePath string) bool {
	// Skip common directories and files that shouldn't be parsed
	skipPaths := []string{
		".git", "node_modules", ".vscode", ".idea", "vendor", "target",
		"build", "dist", "__pycache__", ".pytest_cache", "coverage",
		"site-packages",
	}

	for _, skipPath := range skipPaths {
		if strings.Contains(filePath, skipPath) {
			return true
		}
	}

	if repo.Test != "" && !strings.Contains(filePath, repo.Test) {
		return true
	}

	languageType := fp.DetectLanguage(filePath)

	if languageType == Unknown {
		fp.logger.Debug("Skipping unsupported file", zap.String("path", filePath))
		return true
	}

	if !fp.isAllowedFileExtensionsInRepo(repo, languageType) {
		fp.logger.Debug("Skipping file due to unsupported language for repository", zap.String("path", filePath), zap.String("repo_language", repo.Language))
		return true
	}

	fileScopes, err := fp.CodeGraph.FindFileScopes(ctx, repo.Name, fp.relativePath(repo, filePath))
	if err != nil {
		//fp.logger.Error("Failed to find file scopes", zap.String("path", filePath), zap.Error(err))
		return false
	}

	if len(fileScopes) > 0 {
		for _, fs := range fileScopes {
			if modTime, ok := fs.MetaData["modified"]; ok {
				if modTimeInt, ok := modTime.(int64); ok {
					if modTimeInt == info.ModTime().Unix() {
						fp.logger.Info("Skipping unmodified file", zap.String("path", filePath))
						return true
					}
				}
			}
		}
	}

	return false
}

func (fp *FileParser) isAllowedFileExtensionsInRepo(repo *config.Repository, languageType LanguageType) bool {
	switch repo.Language {
	case "python":
		return languageType == Python
	case "javascript":
		return languageType == JavaScript || languageType == TypeScript
	case "typescript":
		return languageType == TypeScript
	case "go":
		return languageType == Go
	case "java":
		return languageType == Java
	default:
		return false
	}
}
