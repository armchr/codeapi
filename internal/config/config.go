package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"regexp"

	"gopkg.in/yaml.v2"
)

type SourceConfig struct {
	Repositories []Repository `yaml:"repositories"`
}

type Repository struct {
	Name               string `yaml:"name"`
	Path               string `yaml:"path"`
	Test               string `yaml:"test,omitempty"`
	Language           string `yaml:"language"`
	Disabled           bool   `yaml:"disabled,omitempty"`
	SkipOtherLanguages bool   `yaml:"skip_other_languages,omitempty"`
}

type App struct {
	Port                        int    `yaml:"port"`
	CodeGraph                   bool   `yaml:"codegraph"`
	WorkDir                     string `yaml:"workdir,omitempty"`
	GCThreshold                 int64  `yaml:"gc_threshold,omitempty"`
	NumFileThreads              int    `yaml:"num_file_threads,omitempty"`
	MaxConcurrentFileProcessing int    `yaml:"max_concurrent_file_processing,omitempty"`
	DebugHTTP                   bool   `yaml:"debug_http,omitempty"` // Log full request/response bodies
	LogLevel                    string `yaml:"log_level,omitempty"` // debug, info, warn, error (default: info)
}

// LanguageServersConfig holds paths to language server executables
// Keys are language names (e.g., "go", "python", "csharp"), values are paths to LSP executables
type LanguageServersConfig map[string]string

// GetLSPPath returns the path to the language server for the given language
// Returns empty string if no LSP is configured for the language
func (lsc LanguageServersConfig) GetLSPPath(language string) string {
	if lsc == nil {
		return ""
	}
	return lsc[language]
}

type Neo4jConfig struct {
	URI      string `yaml:"uri"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type QdrantConfig struct {
	Host   string `yaml:"host"`
	Port   int    `yaml:"port"`
	APIKey string `yaml:"apikey"`
}

type OllamaConfig struct {
	URL       string `yaml:"url"`
	APIKey    string `yaml:"apikey"`
	Model     string `yaml:"model"`
	Dimension int    `yaml:"dimension"`
}

type ChunkingConfig struct {
	MinConditionalLines int `yaml:"min_conditional_lines"`
	MinLoopLines        int `yaml:"min_loop_lines"`
}

type BloomFilterConfig struct {
	Enabled           bool    `yaml:"enabled"`
	StorageDir        string  `yaml:"storage_dir"`
	ExpectedItems     uint    `yaml:"expected_items"`
	FalsePositiveRate float64 `yaml:"false_positive_rate"`
}

type IndexBuildingConfig struct {
	EnableCodeGraph  bool `yaml:"enable_code_graph"`
	EnableEmbeddings bool `yaml:"enable_embeddings"`
	EnableSummary    bool `yaml:"enable_summary"`
}

type MySQLConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Database string `yaml:"database"`
}

type CodeGraphConfig struct {
	EnableBatchWrites bool `yaml:"enable_batch_writes"`
	BatchSize         int  `yaml:"batch_size"` // Number of nodes/relations to batch before writing
	PrintParseTree    bool `yaml:"print_parse_tree"`
}

// GitAnalysisMode defines how git analysis is performed
type GitAnalysisMode string

const (
	GitAnalysisModeOnDemand   GitAnalysisMode = "ondemand"
	GitAnalysisModePrecompute GitAnalysisMode = "precompute"
)

type GitAnalysisConfig struct {
	Enabled         bool            `yaml:"enabled"`
	Mode            GitAnalysisMode `yaml:"mode"`              // "ondemand" or "precompute"
	LookbackCommits int             `yaml:"lookback_commits"`  // How many commits to analyze (default: 1000)
}

// SummaryConfig holds configuration for hierarchical code summarization
type SummaryConfig struct {
	LLMProvider  string `yaml:"llm_provider"`   // ollama, claude, openai
	LLMModel     string `yaml:"llm_model"`      // Model name (e.g., llama3.2, claude-3-5-haiku-20241022)
	PromptsFile  string `yaml:"prompts_file"`   // Path to prompts YAML config
	WorkerCount  int    `yaml:"worker_count"`   // Parallel workers for summarization
	BatchSize    int    `yaml:"batch_size"`     // Batch size for DB writes
	SkipIfExists bool   `yaml:"skip_if_exists"` // Skip if summary exists and context unchanged

	// Provider-specific
	OllamaURL     string `yaml:"ollama_url"`     // Ollama API URL
	ClaudeAPIKey  string `yaml:"claude_api_key"` // Or use ANTHROPIC_API_KEY env var
	OpenAIAPIKey  string `yaml:"openai_api_key"` // Or use OPENAI_API_KEY env var
	OpenAIBaseURL string `yaml:"openai_base_url"` // For API-compatible services
}

// GitChurnConfig holds configuration for git churn analysis
type GitChurnConfig struct {
	// Enabled enables git churn analysis
	Enabled bool `yaml:"enabled"`

	// TimeWindowDays is the lookback period in days (default: 180)
	TimeWindowDays int `yaml:"time_window_days"`

	// EnableFileLevel enables file-level churn metrics (default: true)
	EnableFileLevel bool `yaml:"enable_file_level"`

	// EnableFunctionLevel enables function-level churn metrics (default: true)
	EnableFunctionLevel bool `yaml:"enable_function_level"`

	// Weights for churn score calculation
	Weights ChurnWeights `yaml:"weights"`

	// ExcludePatterns are glob patterns for files to exclude (e.g., "vendor/**", "**/*_test.go")
	ExcludePatterns []string `yaml:"exclude_patterns"`

	// ExcludeAuthors are author names to exclude (e.g., "dependabot[bot]")
	ExcludeAuthors []string `yaml:"exclude_authors"`

	// ExcludeMerges excludes merge commits from analysis (default: true)
	ExcludeMerges bool `yaml:"exclude_merges"`

	// MaxConcurrency is the maximum number of concurrent file processors (default: 4)
	MaxConcurrency int `yaml:"max_concurrency"`

	// FunctionChurnThreshold is the percentile threshold for function-level analysis
	// Only files in the top N% by churn will get function-level analysis (default: 10)
	FunctionChurnThreshold float64 `yaml:"function_churn_threshold"`
}

// ChurnWeights holds the weights for churn score calculation
type ChurnWeights struct {
	LinesChanged float64 `yaml:"lines_changed"` // Default: 0.5
	CommitCount  float64 `yaml:"commit_count"`  // Default: 0.3
	AuthorCount  float64 `yaml:"author_count"`  // Default: 0.2
}

// GetDefaults returns GitChurnConfig with default values applied
func (c *GitChurnConfig) GetDefaults() GitChurnConfig {
	result := *c
	if result.TimeWindowDays == 0 {
		result.TimeWindowDays = 180
	}
	if !result.EnableFileLevel && !result.EnableFunctionLevel {
		result.EnableFileLevel = true
		result.EnableFunctionLevel = true
	}
	if result.Weights.LinesChanged == 0 && result.Weights.CommitCount == 0 && result.Weights.AuthorCount == 0 {
		result.Weights.LinesChanged = 0.5
		result.Weights.CommitCount = 0.3
		result.Weights.AuthorCount = 0.2
	}
	if result.MaxConcurrency == 0 {
		result.MaxConcurrency = 4
	}
	if result.FunctionChurnThreshold == 0 {
		result.FunctionChurnThreshold = 10.0
	}
	return result
}

type Config struct {
	Source          SourceConfig          `yaml:"source"`
	Neo4j           Neo4jConfig           `yaml:"neo4j"`
	Qdrant          QdrantConfig          `yaml:"qdrant"`
	Chunking        ChunkingConfig        `yaml:"chunking"`
	Ollama          OllamaConfig          `yaml:"ollama"`
	BloomFilter     BloomFilterConfig     `yaml:"bloom_filter"`
	IndexBuilding   IndexBuildingConfig   `yaml:"index_building"`
	MySQL           MySQLConfig           `yaml:"mysql"`
	CodeGraph       CodeGraphConfig       `yaml:"code_graph"`
	GitAnalysis     GitAnalysisConfig     `yaml:"git_analysis"`
	GitChurn        GitChurnConfig        `yaml:"git_churn"`
	Summary         SummaryConfig         `yaml:"summary"`
	LanguageServers LanguageServersConfig `yaml:"language_servers"`
	App             App                   `yaml:"app"`
}

// expandEnvVars expands environment variables in the given string
// Supports formats: ${VAR}, $VAR, ${VAR:-default}
func expandEnvVars(s string) string {
	// Pattern for ${VAR:-default} or ${VAR}
	reBraces := regexp.MustCompile(`\$\{([^}:]+)(:-([^}]*))?\}`)
	s = reBraces.ReplaceAllStringFunc(s, func(match string) string {
		parts := reBraces.FindStringSubmatch(match)
		if len(parts) >= 2 {
			varName := parts[1]
			defaultValue := ""
			if len(parts) >= 4 {
				defaultValue = parts[3]
			}
			if val, ok := os.LookupEnv(varName); ok {
				return val
			}
			return defaultValue
		}
		return match
	})

	// Pattern for $VAR (without braces)
	reSimple := regexp.MustCompile(`\$([A-Za-z_][A-Za-z0-9_]*)`)
	s = reSimple.ReplaceAllStringFunc(s, func(match string) string {
		parts := reSimple.FindStringSubmatch(match)
		if len(parts) >= 2 {
			varName := parts[1]
			if val, ok := os.LookupEnv(varName); ok {
				return val
			}
			return match
		}
		return match
	})

	return s
}

func LoadConfig(appConfigPath string, sourceConfigPath string) (*Config, error) {
	if _, err := os.Stat(appConfigPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("app config file does not exist: %s", appConfigPath)
	}
	if _, err := os.Stat(sourceConfigPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("source config file does not exist: %s", sourceConfigPath)
	}

	dataApp, err := ioutil.ReadFile(appConfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read app config file: %w", err)
	}

	dataSource, err := ioutil.ReadFile(sourceConfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read source config file: %w", err)
	}

	// Expand environment variables in both config files
	dataApp = []byte(expandEnvVars(string(dataApp)))
	dataSource = []byte(expandEnvVars(string(dataSource)))

	var configApp Config
	if err := yaml.Unmarshal(dataApp, &configApp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal app config: %w", err)
	}

	var configSource Config
	if err := yaml.Unmarshal(dataSource, &configSource); err != nil {
		return nil, fmt.Errorf("failed to unmarshal source config: %w", err)
	}

	// Merge SourceConfig into configApp
	configApp.Source = configSource.Source

	// Validate repository configurations
	if err := validateRepositories(&configApp); err != nil {
		return nil, fmt.Errorf("invalid repository configuration: %w", err)
	}

	if configSource.Neo4j.URI != "" {
		configApp.Neo4j = configSource.Neo4j
	}

	if configSource.Qdrant.Host != "" {
		configApp.Qdrant = configSource.Qdrant
	}

	if configSource.Ollama.URL != "" {
		configApp.Ollama = configSource.Ollama
	}

	return &configApp, nil
}

func (c *Config) GetRepository(name string) (*Repository, error) {
	for _, repo := range c.Source.Repositories {
		if repo.Name == name {
			return &repo, nil
		}
	}
	return nil, fmt.Errorf("repository not found: %s", name)
}

// validateRepositories validates repository configurations
func validateRepositories(config *Config) error {
	for _, repo := range config.Source.Repositories {
		// If skip_other_languages is true, language must be specified
		if repo.SkipOtherLanguages && repo.Language == "" {
			return fmt.Errorf("repository '%s': skip_other_languages is true but language is not specified", repo.Name)
		}
	}
	return nil
}
