package summary

import (
	"crypto/sha256"
	"encoding/hex"
	"path/filepath"
	"sort"
	"strings"
)

// ContextBuilder builds context objects for different summary levels
type ContextBuilder struct {
	maxContextChars int
}

// NewContextBuilder creates a new context builder
func NewContextBuilder(maxContextChars int) *ContextBuilder {
	if maxContextChars <= 0 {
		maxContextChars = 4000
	}
	return &ContextBuilder{maxContextChars: maxContextChars}
}

// BuildFunctionContext builds context for function-level summarization
func (cb *ContextBuilder) BuildFunctionContext(
	name, signature, docstring, sourceCode, language, filePath, className string,
	parameters []ParameterInfo,
	returnType string,
	annotations, modifiers []string,
) *FunctionContext {
	// Truncate source code if too long
	truncatedCode := cb.truncateText(sourceCode, cb.maxContextChars)

	return &FunctionContext{
		Name:        name,
		Signature:   signature,
		Docstring:   docstring,
		SourceCode:  truncatedCode,
		Parameters:  parameters,
		ReturnType:  returnType,
		Language:    language,
		FilePath:    filePath,
		ClassName:   className,
		Annotations: annotations,
		Modifiers:   modifiers,
	}
}

// BuildClassContext builds context for class-level summarization
func (cb *ContextBuilder) BuildClassContext(
	name, docstring, language, filePath string,
	inheritance, implements []string,
	fields []FieldInfo,
	methodSummaries []EntitySummary,
	annotations, modifiers []string,
) *ClassContext {
	return &ClassContext{
		Name:            name,
		Docstring:       docstring,
		Inheritance:     inheritance,
		Implements:      implements,
		Fields:          fields,
		MethodSummaries: methodSummaries,
		Language:        language,
		FilePath:        filePath,
		Annotations:     annotations,
		Modifiers:       modifiers,
	}
}

// BuildFileContext builds context for file-level summarization
func (cb *ContextBuilder) BuildFileContext(
	filePath, language, packageName, moduleName string,
	imports []string,
	classSummaries, functionSummaries []EntitySummary,
) *FileContext {
	// Limit imports to most relevant ones
	limitedImports := cb.limitImports(imports, 20)

	return &FileContext{
		FilePath:          filePath,
		FileName:          filepath.Base(filePath),
		Language:          language,
		Imports:           limitedImports,
		ClassSummaries:    classSummaries,
		FunctionSummaries: functionSummaries,
		PackageName:       packageName,
		ModuleName:        moduleName,
	}
}

// BuildFolderContext builds context for folder-level summarization
func (cb *ContextBuilder) BuildFolderContext(
	folderPath string,
	fileSummaries, subfolderSummaries []EntitySummary,
	languages []string,
) *FolderContext {
	return &FolderContext{
		FolderPath:         folderPath,
		FolderName:         filepath.Base(folderPath),
		FileSummaries:      fileSummaries,
		SubfolderSummaries: subfolderSummaries,
		Languages:          languages,
	}
}

// BuildProjectContext builds context for project-level summarization
func (cb *ContextBuilder) BuildProjectContext(
	projectName string,
	languages []string,
	topLevelSummaries []EntitySummary,
	entryPoints []string,
	totalFiles, totalClasses, totalFunctions int,
) *ProjectContext {
	return &ProjectContext{
		ProjectName:       projectName,
		Languages:         languages,
		TopLevelSummaries: topLevelSummaries,
		EntryPoints:       entryPoints,
		TotalFiles:        totalFiles,
		TotalClasses:      totalClasses,
		TotalFunctions:    totalFunctions,
	}
}

// HashContext generates a hash of the context for caching
func (cb *ContextBuilder) HashContext(context any) string {
	var builder strings.Builder

	switch ctx := context.(type) {
	case *FunctionContext:
		builder.WriteString(ctx.Name)
		builder.WriteString(ctx.Signature)
		builder.WriteString(ctx.Docstring)
		builder.WriteString(ctx.SourceCode)
		builder.WriteString(ctx.Language)
	case *ClassContext:
		builder.WriteString(ctx.Name)
		builder.WriteString(ctx.Docstring)
		for _, m := range ctx.MethodSummaries {
			builder.WriteString(m.Name)
			builder.WriteString(m.Summary)
		}
	case *FileContext:
		builder.WriteString(ctx.FilePath)
		for _, c := range ctx.ClassSummaries {
			builder.WriteString(c.Name)
			builder.WriteString(c.Summary)
		}
		for _, f := range ctx.FunctionSummaries {
			builder.WriteString(f.Name)
			builder.WriteString(f.Summary)
		}
	case *FolderContext:
		builder.WriteString(ctx.FolderPath)
		for _, f := range ctx.FileSummaries {
			builder.WriteString(f.Name)
			builder.WriteString(f.Summary)
		}
		for _, s := range ctx.SubfolderSummaries {
			builder.WriteString(s.Name)
			builder.WriteString(s.Summary)
		}
	case *ProjectContext:
		builder.WriteString(ctx.ProjectName)
		for _, t := range ctx.TopLevelSummaries {
			builder.WriteString(t.Name)
			builder.WriteString(t.Summary)
		}
	}

	hash := sha256.Sum256([]byte(builder.String()))
	return hex.EncodeToString(hash[:])
}

// truncateText truncates text to a maximum length, trying to break at word boundaries
func (cb *ContextBuilder) truncateText(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}

	// Find a good break point (newline or space)
	truncated := text[:maxLen]
	if lastNewline := strings.LastIndex(truncated, "\n"); lastNewline > maxLen*3/4 {
		return truncated[:lastNewline] + "\n... (truncated)"
	}
	if lastSpace := strings.LastIndex(truncated, " "); lastSpace > maxLen*3/4 {
		return truncated[:lastSpace] + " ... (truncated)"
	}
	return truncated + "... (truncated)"
}

// limitImports limits the number of imports, prioritizing local/project imports
func (cb *ContextBuilder) limitImports(imports []string, max int) []string {
	if len(imports) <= max {
		return imports
	}

	// Sort to prioritize relative/local imports
	sorted := make([]string, len(imports))
	copy(sorted, imports)
	sort.Slice(sorted, func(i, j int) bool {
		iLocal := strings.HasPrefix(sorted[i], ".") || strings.HasPrefix(sorted[i], "/")
		jLocal := strings.HasPrefix(sorted[j], ".") || strings.HasPrefix(sorted[j], "/")
		if iLocal != jLocal {
			return iLocal
		}
		return sorted[i] < sorted[j]
	})

	return sorted[:max]
}

// TruncateSummaries truncates a list of summaries to fit within context limits
func (cb *ContextBuilder) TruncateSummaries(summaries []EntitySummary, maxTotal int) []EntitySummary {
	if len(summaries) == 0 {
		return summaries
	}

	// Calculate total length
	totalLen := 0
	for _, s := range summaries {
		totalLen += len(s.Name) + len(s.Summary) + 10 // 10 for formatting
	}

	if totalLen <= maxTotal {
		return summaries
	}

	// Calculate average allowed per summary
	avgAllowed := maxTotal / len(summaries)
	if avgAllowed < 50 {
		// Too many summaries, truncate the list
		maxItems := maxTotal / 50
		if maxItems < 1 {
			maxItems = 1
		}
		return summaries[:maxItems]
	}

	// Truncate individual summaries
	result := make([]EntitySummary, len(summaries))
	for i, s := range summaries {
		result[i] = EntitySummary{
			Name:     s.Name,
			Summary:  cb.truncateText(s.Summary, avgAllowed-len(s.Name)-10),
			FilePath: s.FilePath,
		}
	}
	return result
}
