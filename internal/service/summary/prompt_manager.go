package summary

import (
	"bytes"
	"fmt"
	"os"
	"text/template"

	"gopkg.in/yaml.v2"
)

// PromptManager manages prompt templates for different summary levels
type PromptManager struct {
	templates     map[SummaryLevel]*PromptTemplate
	defaults      PromptDefaults
}

// PromptDefaults holds default settings for all prompts
type PromptDefaults struct {
	MaxTokens   int     `yaml:"max_tokens"`
	Temperature float64 `yaml:"temperature"`
}

// PromptTemplate holds a parsed prompt template
type PromptTemplate struct {
	Level           SummaryLevel
	SystemPrompt    string
	UserPromptTmpl  *template.Template
	ContextFields   []string
	MaxContextChars int
	MaxTokens       int
	Temperature     float64
}

// promptConfigFile represents the structure of the YAML config file
type promptConfigFile struct {
	Defaults PromptDefaults              `yaml:"defaults"`
	Levels   map[string]promptLevelConfig `yaml:"levels"`
}

type promptLevelConfig struct {
	SystemPrompt    string   `yaml:"system_prompt"`
	UserPrompt      string   `yaml:"user_prompt"`
	ContextFields   []string `yaml:"context_fields"`
	MaxContextChars int      `yaml:"max_context_chars"`
	MaxTokens       int      `yaml:"max_tokens"`
	Temperature     float64  `yaml:"temperature"`
}

// NewPromptManager creates a new prompt manager from a YAML config file
func NewPromptManager(configPath string) (*PromptManager, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read prompt config: %w", err)
	}

	return NewPromptManagerFromBytes(data)
}

// NewPromptManagerFromBytes creates a new prompt manager from YAML bytes
func NewPromptManagerFromBytes(data []byte) (*PromptManager, error) {
	var config promptConfigFile
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse prompt config: %w", err)
	}

	// Set defaults if not specified
	if config.Defaults.MaxTokens == 0 {
		config.Defaults.MaxTokens = 500
	}
	if config.Defaults.Temperature == 0 {
		config.Defaults.Temperature = 0.3
	}

	pm := &PromptManager{
		templates: make(map[SummaryLevel]*PromptTemplate),
		defaults:  config.Defaults,
	}

	// Parse each level
	for levelName, levelConfig := range config.Levels {
		level := ParseSummaryLevel(levelName)
		if level == 0 {
			return nil, fmt.Errorf("unknown level: %s", levelName)
		}

		tmpl, err := template.New(levelName).Parse(levelConfig.UserPrompt)
		if err != nil {
			return nil, fmt.Errorf("failed to parse template for %s: %w", levelName, err)
		}

		// Apply defaults where not specified
		maxTokens := levelConfig.MaxTokens
		if maxTokens == 0 {
			maxTokens = config.Defaults.MaxTokens
		}
		temperature := levelConfig.Temperature
		if temperature == 0 {
			temperature = config.Defaults.Temperature
		}
		maxContextChars := levelConfig.MaxContextChars
		if maxContextChars == 0 {
			maxContextChars = 4000
		}

		pm.templates[level] = &PromptTemplate{
			Level:           level,
			SystemPrompt:    levelConfig.SystemPrompt,
			UserPromptTmpl:  tmpl,
			ContextFields:   levelConfig.ContextFields,
			MaxContextChars: maxContextChars,
			MaxTokens:       maxTokens,
			Temperature:     temperature,
		}
	}

	return pm, nil
}

// NewPromptManagerWithDefaults creates a prompt manager with default prompts
func NewPromptManagerWithDefaults() (*PromptManager, error) {
	return NewPromptManagerFromBytes([]byte(defaultPromptConfig))
}

// GetTemplate returns the prompt template for a given level
func (pm *PromptManager) GetTemplate(level SummaryLevel) (*PromptTemplate, error) {
	tmpl, ok := pm.templates[level]
	if !ok {
		return nil, fmt.Errorf("no template for level: %s", level.String())
	}
	return tmpl, nil
}

// RenderPrompt renders a prompt for the given level and context
func (pm *PromptManager) RenderPrompt(level SummaryLevel, context any) (systemPrompt, userPrompt string, err error) {
	tmpl, err := pm.GetTemplate(level)
	if err != nil {
		return "", "", err
	}

	var buf bytes.Buffer
	if err := tmpl.UserPromptTmpl.Execute(&buf, context); err != nil {
		return "", "", fmt.Errorf("failed to render template: %w", err)
	}

	return tmpl.SystemPrompt, buf.String(), nil
}

// GetDefaults returns the default prompt settings
func (pm *PromptManager) GetDefaults() PromptDefaults {
	return pm.defaults
}

// default prompt configuration
const defaultPromptConfig = `
defaults:
  max_tokens: 500
  temperature: 0.3

levels:
  function:
    system_prompt: |
      You are a code documentation expert. Generate concise, accurate summaries of code functions and methods.
      Focus on:
      - What the function does (its purpose)
      - Key parameters and their roles
      - What it returns
      - Important side effects or behaviors
      Keep summaries to 2-3 sentences maximum.
    user_prompt: |
      Summarize this {{.Language}} function:

      Name: {{.Name}}
      {{if .ClassName}}Class: {{.ClassName}}{{end}}
      {{if .Signature}}Signature: {{.Signature}}{{end}}
      {{if .Docstring}}Existing Docstring: {{.Docstring}}{{end}}
      {{if .Annotations}}Annotations: {{range .Annotations}}@{{.}} {{end}}{{end}}

      Code:
      ` + "```{{.Language}}" + `
      {{.SourceCode}}
      ` + "```" + `
    context_fields: [name, signature, docstring, source_code, parameters, return_type, annotations]
    max_context_chars: 4000

  class:
    system_prompt: |
      You are a code documentation expert. Generate concise, accurate summaries of classes.
      Focus on:
      - The class's primary responsibility
      - Key public methods and their purposes
      - Important relationships (inheritance, interfaces)
      Keep summaries to 2-4 sentences maximum.
    user_prompt: |
      Summarize this {{.Language}} class based on its structure and method summaries:

      Name: {{.Name}}
      {{if .Docstring}}Existing Docstring: {{.Docstring}}{{end}}
      {{if .Inheritance}}Inherits from: {{range $i, $v := .Inheritance}}{{if $i}}, {{end}}{{$v}}{{end}}{{end}}
      {{if .Implements}}Implements: {{range $i, $v := .Implements}}{{if $i}}, {{end}}{{$v}}{{end}}{{end}}
      {{if .Annotations}}Annotations: {{range .Annotations}}@{{.}} {{end}}{{end}}

      Fields:
      {{range .Fields}}- {{.Name}}: {{.Type}}
      {{end}}

      Method Summaries:
      {{range .MethodSummaries}}- {{.Name}}: {{.Summary}}
      {{end}}
    context_fields: [name, docstring, inheritance, implements, fields, method_summaries, annotations]
    max_context_chars: 8000

  file:
    system_prompt: |
      You are a code documentation expert. Generate concise, accurate summaries of source files.
      Focus on:
      - The file's primary purpose
      - Key classes, functions, or exports it provides
      - Its role in the larger codebase
      Keep summaries to 2-4 sentences maximum.
    user_prompt: |
      Summarize this {{.Language}} source file:

      Path: {{.FilePath}}
      {{if .PackageName}}Package: {{.PackageName}}{{end}}
      {{if .ModuleName}}Module: {{.ModuleName}}{{end}}

      {{if .Imports}}Key Imports:
      {{range .Imports}}- {{.}}
      {{end}}{{end}}

      Contents:
      {{if .ClassSummaries}}Classes:
      {{range .ClassSummaries}}- {{.Name}}: {{.Summary}}
      {{end}}{{end}}
      {{if .FunctionSummaries}}Functions:
      {{range .FunctionSummaries}}- {{.Name}}: {{.Summary}}
      {{end}}{{end}}
    context_fields: [file_path, package_name, imports, class_summaries, function_summaries]
    max_context_chars: 8000

  folder:
    system_prompt: |
      You are a code documentation expert. Generate concise summaries of code modules/packages.
      Focus on:
      - The module's primary purpose and responsibility
      - Key components it provides
      - Its role in the system architecture
      Keep summaries to 2-4 sentences maximum.
    user_prompt: |
      Summarize this code folder/module:

      Path: {{.FolderPath}}
      {{if .Languages}}Languages: {{range $i, $v := .Languages}}{{if $i}}, {{end}}{{$v}}{{end}}{{end}}

      Files:
      {{range .FileSummaries}}- {{.Name}}: {{.Summary}}
      {{end}}

      {{if .SubfolderSummaries}}Submodules:
      {{range .SubfolderSummaries}}- {{.Name}}: {{.Summary}}
      {{end}}{{end}}
    context_fields: [folder_path, file_summaries, subfolder_summaries, languages]
    max_context_chars: 12000

  project:
    system_prompt: |
      You are a code documentation expert. Generate a high-level project overview.
      Focus on:
      - What the project does
      - Key components and their roles
      - Technologies and patterns used
      Keep the overview to 4-6 sentences maximum.
    user_prompt: |
      Provide a high-level summary of this project:

      Name: {{.ProjectName}}
      Languages: {{range $i, $v := .Languages}}{{if $i}}, {{end}}{{$v}}{{end}}
      Total Files: {{.TotalFiles}}
      Total Classes: {{.TotalClasses}}
      Total Functions: {{.TotalFunctions}}

      Top-level Structure:
      {{range .TopLevelSummaries}}- {{.Name}}: {{.Summary}}
      {{end}}

      {{if .EntryPoints}}Entry Points:
      {{range .EntryPoints}}- {{.}}
      {{end}}{{end}}
    context_fields: [project_name, languages, top_level_summaries, entry_points, total_files]
    max_context_chars: 16000
`
