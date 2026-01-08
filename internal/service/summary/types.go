package summary

import "time"

// SummaryLevel represents the hierarchical level of a summary
type SummaryLevel int

const (
	LevelFunction SummaryLevel = iota + 1
	LevelClass
	LevelFile
	LevelFolder
	LevelProject
)

// String returns the string representation of the level
func (l SummaryLevel) String() string {
	switch l {
	case LevelFunction:
		return "function"
	case LevelClass:
		return "class"
	case LevelFile:
		return "file"
	case LevelFolder:
		return "folder"
	case LevelProject:
		return "project"
	default:
		return "unknown"
	}
}

// ParseSummaryLevel parses a string into a SummaryLevel
func ParseSummaryLevel(s string) SummaryLevel {
	switch s {
	case "function":
		return LevelFunction
	case "class":
		return LevelClass
	case "file":
		return LevelFile
	case "folder":
		return LevelFolder
	case "project":
		return LevelProject
	default:
		return 0
	}
}

// CodeSummary represents a generated summary for a code entity
type CodeSummary struct {
	ID           int64        `json:"id" db:"id"`
	EntityID     string       `json:"entity_id" db:"entity_id"`         // AST NodeID or path
	EntityType   SummaryLevel `json:"entity_type" db:"entity_type"`     // function, class, file, folder, project
	EntityName   string       `json:"entity_name" db:"entity_name"`
	FilePath     string       `json:"file_path" db:"file_path"`
	Summary      string       `json:"summary" db:"summary"`
	ContextHash  string       `json:"context_hash" db:"context_hash"`   // Hash of input context
	LLMProvider  string       `json:"llm_provider" db:"llm_provider"`
	LLMModel     string       `json:"llm_model" db:"llm_model"`
	PromptTokens int          `json:"prompt_tokens" db:"prompt_tokens"`
	OutputTokens int          `json:"output_tokens" db:"output_tokens"`
	CreatedAt    time.Time    `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time    `json:"updated_at" db:"updated_at"`
}

// FunctionContext holds context for function-level summarization
type FunctionContext struct {
	Name        string            `json:"name"`
	Signature   string            `json:"signature"`
	Docstring   string            `json:"docstring"`
	SourceCode  string            `json:"source_code"`
	Parameters  []ParameterInfo   `json:"parameters"`
	ReturnType  string            `json:"return_type"`
	Language    string            `json:"language"`
	FilePath    string            `json:"file_path"`
	ClassName   string            `json:"class_name"` // If it's a method
	Annotations []string          `json:"annotations"`
	Modifiers   []string          `json:"modifiers"` // public, private, static, etc.
}

// ParameterInfo holds information about a function parameter
type ParameterInfo struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// ClassContext holds context for class-level summarization
type ClassContext struct {
	Name            string          `json:"name"`
	Docstring       string          `json:"docstring"`
	Inheritance     []string        `json:"inheritance"` // Parent classes/interfaces
	Implements      []string        `json:"implements"`
	Fields          []FieldInfo     `json:"fields"`
	MethodSummaries []EntitySummary `json:"method_summaries"`
	Language        string          `json:"language"`
	FilePath        string          `json:"file_path"`
	Annotations     []string        `json:"annotations"`
	Modifiers       []string        `json:"modifiers"`
}

// FieldInfo holds information about a class field
type FieldInfo struct {
	Name      string   `json:"name"`
	Type      string   `json:"type"`
	Modifiers []string `json:"modifiers"`
}

// FileContext holds context for file-level summarization
type FileContext struct {
	FilePath         string          `json:"file_path"`
	FileName         string          `json:"file_name"`
	Language         string          `json:"language"`
	Imports          []string        `json:"imports"`
	ClassSummaries   []EntitySummary `json:"class_summaries"`
	FunctionSummaries []EntitySummary `json:"function_summaries"`
	PackageName      string          `json:"package_name"`
	ModuleName       string          `json:"module_name"`
}

// FolderContext holds context for folder-level summarization
type FolderContext struct {
	FolderPath        string          `json:"folder_path"`
	FolderName        string          `json:"folder_name"`
	FileSummaries     []EntitySummary `json:"file_summaries"`
	SubfolderSummaries []EntitySummary `json:"subfolder_summaries"`
	Languages         []string        `json:"languages"`
}

// ProjectContext holds context for project-level summarization
type ProjectContext struct {
	ProjectName       string          `json:"project_name"`
	Languages         []string        `json:"languages"`
	TopLevelSummaries []EntitySummary `json:"top_level_summaries"`
	EntryPoints       []string        `json:"entry_points"`
	TotalFiles        int             `json:"total_files"`
	TotalClasses      int             `json:"total_classes"`
	TotalFunctions    int             `json:"total_functions"`
}

// EntitySummary is a lightweight summary reference used in contexts
type EntitySummary struct {
	Name     string `json:"name"`
	Summary  string `json:"summary"`
	FilePath string `json:"file_path,omitempty"`
}
