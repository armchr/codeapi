package main

import (
	"github.com/armchr/codeapi/internal/config"
	"github.com/armchr/codeapi/internal/service"
	"github.com/armchr/codeapi/internal/util"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// EmbeddingAdapter adapts service.EmbeddingModel to util.EmbeddingGenerator interface
type EmbeddingAdapter struct {
	model service.EmbeddingModel
}

func (e *EmbeddingAdapter) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	embeddings, err := e.model.GenerateEmbeddings(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	if len(embeddings) == 0 {
		return nil, fmt.Errorf("no embeddings generated")
	}
	return embeddings[0], nil
}

// EvalResult represents the evaluation results for all test cases
type EvalResult struct {
	TestCases []TestCaseResult `json:"test_cases"`
	Summary   Summary          `json:"summary"`
}

// TestCaseResult represents results for a single test case
type TestCaseResult struct {
	TestCaseDir  string                   `json:"test_case_dir"`
	Metadata     *util.EvalMetadata       `json:"metadata"`
	Similarities []util.SnippetSimilarity `json:"similarities"`
}

// Summary provides aggregate statistics
type Summary struct {
	TotalTestCases      int     `json:"total_test_cases"`
	SimilarTestCases    int     `json:"similar_test_cases"`
	DifferentTestCases  int     `json:"different_test_cases"`
	AvgSimilarScore     float64 `json:"avg_similar_score"`
	AvgDifferentScore   float64 `json:"avg_different_score"`
	HighSimilarityCount int     `json:"high_similarity_count"` // > 0.85
	LowSimilarityCount  int     `json:"low_similarity_count"`  // < 0.5
}

func main() {
	// Command line flags
	testPath := flag.String("test", "", "Path to test case directory or directory containing multiple test cases")
	outputFile := flag.String("output", "eval_results.json", "Output file for evaluation results")
	appConfig := flag.String("app", "config/app.yaml", "Path to app config file")
	sourceConfig := flag.String("source", "config/source.yaml", "Path to source config file")
	flag.Parse()

	if *testPath == "" {
		fmt.Println("Error: -test flag is required")
		flag.Usage()
		os.Exit(1)
	}

	// Initialize logger
	loggerConfig := zap.NewProductionConfig()
	loggerConfig.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	logger, err := loggerConfig.Build()
	if err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	logger.Info("Starting evaluation",
		zap.String("test_path", *testPath),
		zap.String("output_file", *outputFile))

	// Load configuration
	cfg, err := config.LoadConfig(*appConfig, *sourceConfig)
	if err != nil {
		logger.Fatal("Failed to load config", zap.Error(err))
	}

	// Initialize embedding model (same as in main.go)
	embeddingModel, err := service.NewOllamaEmbedding(service.OllamaEmbeddingConfig{
		APIURL:    cfg.Ollama.URL,
		APIKey:    cfg.Ollama.APIKey,
		Model:     cfg.Ollama.Model,
		Dimension: cfg.Ollama.Dimension,
	}, logger)
	if err != nil {
		logger.Fatal("Failed to initialize Ollama embedding model", zap.Error(err))
	}

	// Create adapter for util.EmbeddingGenerator interface
	embeddingGen := &EmbeddingAdapter{model: embeddingModel}

	ctx := context.Background()

	// Determine if testPath is a single test case or directory of test cases
	metadataPath := filepath.Join(*testPath, "metadata.json")
	isSingleTest := false
	if _, err := os.Stat(metadataPath); err == nil {
		isSingleTest = true
	}

	var results map[string][]util.SnippetSimilarity
	var err2 error

	var basePath string
	if isSingleTest {
		logger.Info("Evaluating single test case", zap.String("path", *testPath))
		similarities, err := util.EvaluateTestCase(ctx, *testPath, embeddingGen, logger)
		if err != nil {
			logger.Fatal("Failed to evaluate test case", zap.Error(err))
		}
		results = map[string][]util.SnippetSimilarity{
			".": similarities, // Use "." for current directory since basePath will be the test case dir
		}
		basePath = *testPath
	} else {
		logger.Info("Evaluating all test cases in directory", zap.String("path", *testPath))
		results, err2 = util.EvaluateAllTestCases(ctx, *testPath, embeddingGen, logger)
		if err2 != nil {
			logger.Fatal("Failed to evaluate test cases", zap.Error(err2))
		}
		basePath = *testPath
	}

	// Build evaluation results with metadata
	evalResult := buildEvalResult(basePath, results, isSingleTest, logger)

	// Write results to output file
	outputData, err := json.MarshalIndent(evalResult, "", "  ")
	if err != nil {
		logger.Fatal("Failed to marshal results", zap.Error(err))
	}

	if err := os.WriteFile(*outputFile, outputData, 0644); err != nil {
		logger.Fatal("Failed to write output file", zap.Error(err))
	}

	logger.Info("Evaluation complete",
		zap.String("output_file", *outputFile),
		zap.Int("total_test_cases", len(results)))

	// Print summary to console
	printSummary(evalResult, logger)
}

// buildEvalResult constructs the full evaluation result with metadata
func buildEvalResult(basePath string, results map[string][]util.SnippetSimilarity, isSingleTest bool, logger *zap.Logger) EvalResult {
	testCaseResults := make([]TestCaseResult, 0, len(results))

	var totalSimilarScore, totalDifferentScore float64
	var similarCount, differentCount int
	var highSimilarityCount, lowSimilarityCount int

	// Sort test case names for consistent output
	testCaseNames := make([]string, 0, len(results))
	for name := range results {
		testCaseNames = append(testCaseNames, name)
	}
	sort.Strings(testCaseNames)

	for _, testCaseName := range testCaseNames {
		similarities := results[testCaseName]

		// Read metadata for this test case
		var testCaseDir string
		var displayName string
		if isSingleTest {
			testCaseDir = basePath
			displayName = filepath.Base(basePath)
		} else {
			testCaseDir = filepath.Join(basePath, testCaseName)
			displayName = testCaseName
		}
		metadataPath := filepath.Join(testCaseDir, "metadata.json")

		data, err := os.ReadFile(metadataPath)
		if err != nil {
			logger.Warn("Failed to read metadata",
				zap.String("test_case", testCaseName),
				zap.Error(err))
			continue
		}

		var metadata util.EvalMetadata
		if err := json.Unmarshal(data, &metadata); err != nil {
			logger.Warn("Failed to parse metadata",
				zap.String("test_case", testCaseName),
				zap.Error(err))
			continue
		}

		testCaseResults = append(testCaseResults, TestCaseResult{
			TestCaseDir:  displayName,
			Metadata:     &metadata,
			Similarities: similarities,
		})

		// Calculate statistics (skip first similarity as it's self-similarity = 1.0)
		if len(similarities) > 1 {
			for i := 1; i < len(similarities); i++ {
				score := similarities[i].Similarity

				if metadata.Type == "similar" {
					totalSimilarScore += score
					similarCount++
					if score > 0.85 {
						highSimilarityCount++
					}
				} else if metadata.Type == "different" {
					totalDifferentScore += score
					differentCount++
					if score < 0.5 {
						lowSimilarityCount++
					}
				}
			}
		}
	}

	summary := Summary{
		TotalTestCases:      len(testCaseResults),
		SimilarTestCases:    similarCount,
		DifferentTestCases:  differentCount,
		HighSimilarityCount: highSimilarityCount,
		LowSimilarityCount:  lowSimilarityCount,
	}

	if similarCount > 0 {
		summary.AvgSimilarScore = totalSimilarScore / float64(similarCount)
	}
	if differentCount > 0 {
		summary.AvgDifferentScore = totalDifferentScore / float64(differentCount)
	}

	return EvalResult{
		TestCases: testCaseResults,
		Summary:   summary,
	}
}

// printSummary prints a human-readable summary to console
func printSummary(result EvalResult, logger *zap.Logger) {
	fmt.Println("\n" + string(make([]byte, 80)) + "\n")
	fmt.Println("=== EVALUATION SUMMARY ===")
	fmt.Println()
	fmt.Printf("Total Test Cases:     %d\n", result.Summary.TotalTestCases)
	fmt.Printf("Similar Test Cases:   %d\n", result.Summary.SimilarTestCases)
	fmt.Printf("Different Test Cases: %d\n", result.Summary.DifferentTestCases)
	fmt.Println()
	fmt.Printf("Average Similarity Score (Similar):   %.4f\n", result.Summary.AvgSimilarScore)
	fmt.Printf("Average Similarity Score (Different): %.4f\n", result.Summary.AvgDifferentScore)
	fmt.Println()
	fmt.Printf("High Similarity Count (>0.85): %d\n", result.Summary.HighSimilarityCount)
	fmt.Printf("Low Similarity Count (<0.50):  %d\n", result.Summary.LowSimilarityCount)
	fmt.Println()

	fmt.Println("=== TEST CASE DETAILS ===")
	for _, tc := range result.TestCases {
		fmt.Printf("\n%s (%s):\n", tc.Metadata.Name, tc.Metadata.Type)
		for _, sim := range tc.Similarities {
			fmt.Printf("  %-40s %.4f\n", sim.SnippetName, sim.Similarity)
		}
	}
	fmt.Println()
}
