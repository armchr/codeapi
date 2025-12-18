package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/armchr/codeapi/internal/codeapi"
	"github.com/armchr/codeapi/internal/config"
	"github.com/armchr/codeapi/internal/controller"
	"github.com/armchr/codeapi/internal/db"
	"github.com/armchr/codeapi/internal/handler"
	init_services "github.com/armchr/codeapi/internal/init"
	"github.com/armchr/codeapi/internal/util"
	"github.com/armchr/codeapi/pkg/lsp"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// stringSliceFlag is a custom flag type that allows multiple values
type stringSliceFlag []string

func (s *stringSliceFlag) String() string {
	return strings.Join(*s, ",")
}

func (s *stringSliceFlag) Set(value string) error {
	*s = append(*s, value)
	return nil
}

func main() {
	var sourceConfigPath = flag.String("source", "source.yaml", "Path to source configuration file")
	var appConfigPath = flag.String("app", "app.yaml", "Path to app configuration file")
	var workDir = flag.String("workdir", "", "Working directory to store files")
	//var port = flag.String("port", "8080", "Server port")
	var test = flag.Bool("test", false, "Run in test mode")
	var buildIndex stringSliceFlag
	flag.Var(&buildIndex, "build-index", "Repository name to build index for (can be specified multiple times)")
	var useHead = flag.Bool("head", false, "Use git HEAD version instead of working directory (only valid with --build-index)")
	var testDump = flag.String("test-dump", "", "Path to output file for dumping code graph after index building (only valid with --build-index)")
	var clean = flag.Bool("clean", false, "Clean up all DB entries (MySQL, Neo4j, Qdrant) for the repository after processing (only valid with --build-index)")
	flag.Parse()

	//logger, err := zap.NewProduction()
	cfgZap := zap.NewProductionConfig()
	//cfgZap.Level.SetLevel(zapcore.DebugLevel)
	cfgZap.Level.SetLevel(zapcore.InfoLevel)
	cfgZap.OutputPaths = []string{"stdout", "all.log"}
	logger, err := cfgZap.Build()
	if err != nil {
		log.Fatal("Failed to initialize logger:", err)
	}

	defer logger.Sync()

	cfg, err := config.LoadConfig(*appConfigPath, *sourceConfigPath)
	if err != nil {
		logger.Fatal("Failed to load configuration", zap.Error(err))
	}

	// Override workdir from command line if provided
	if *workDir != "" {
		cfg.App.WorkDir = *workDir
	}

	logger.Info("Configuration loaded successfully", zap.Any("config", cfg))

	if test != nil && *test {
		logger.Info("Running in test mode")
		LSPTest(cfg, logger)
		return
	}

	// Check if we're in CLI mode (build-index specified)
	if len(buildIndex) > 0 {
		logger.Info("Running in CLI mode - build-index")
		BuildIndexCommand(cfg, logger, buildIndex, *useHead, *testDump, *clean)
		return
	}

	// Validate --test-dump flag usage
	if *testDump != "" {
		logger.Fatal("--test-dump flag is only valid with --build-index")
	}

	// Validate --clean flag usage
	if *clean {
		logger.Fatal("--clean flag is only valid with --build-index")
	}

	// Validate --head flag usage
	if *useHead {
		logger.Fatal("--head flag is only valid with --build-index")
	}

	// Initialize all services using the new initialization module
	opts := init_services.GetServerModeOptions(cfg)
	container, err := init_services.NewServiceContainer(cfg, opts, logger)
	if err != nil {
		logger.Fatal("Failed to initialize services", zap.Error(err))
	}
	defer container.Close(context.Background())

	// Initialize processors and index builder
	if err := container.InitProcessors(cfg); err != nil {
		logger.Fatal("Failed to initialize processors", zap.Error(err))
	}

	// Start CodeGraph processing in background if enabled
	/*
		if container.CodeGraph != nil {
			CodeGraphEntry(cfg, logger, container)
		}
	*/

	repoController := controller.NewRepoController(container.RepoService, container.ChunkService, container.Processors, container.MySQLConn, cfg, logger)

	// Initialize CodeAPI controller if CodeGraph is available
	var codeAPIController *controller.CodeAPIController
	if container.CodeGraph != nil {
		codeAPI := codeapi.NewCodeAPI(container.CodeGraph, logger)
		codeAPIController = controller.NewCodeAPIController(codeAPI, logger)
	}

	router := handler.SetupRouter(repoController, codeAPIController, logger)

	logger.Info("Starting server", zap.Int("port", cfg.App.Port))
	if err := http.ListenAndServe(fmt.Sprintf(":%d", cfg.App.Port), router); err != nil {
		logger.Fatal("Failed to start server", zap.Error(err))
	}
}

func LSPTest(cfg *config.Config, logger *zap.Logger) {
	logger.Info("Testing LSP client")
	repo, _ := cfg.GetRepository("mcp-server")

	// Initialize the LSP client
	ls, err := lsp.NewLSPLanguageServer(cfg, repo.Language, repo.Path, logger)
	if err != nil {
		logger.Fatal("Failed to create LSP client", zap.Error(err))
	}

	// Create a context for the LSP operations
	ctx := context.Background()

	defer ls.Shutdown(ctx)

	// Initialize the LSP client

	baseClient := ls.(*lsp.TypeScriptLanguageServerClient).BaseClient

	baseClient.TestCommand(ctx)
}

func BuildIndexCommand(cfg *config.Config, logger *zap.Logger, repoNames []string, useHead bool, testDumpPath string, clean bool) {
	ctx := context.Background()

	logger.Info("Build index command started",
		zap.Strings("repositories", repoNames),
		zap.Bool("use_head", useHead),
		zap.String("test_dump_path", testDumpPath),
		zap.Bool("clean", clean),
		zap.Bool("code_graph_enabled", cfg.IndexBuilding.EnableCodeGraph),
		zap.Bool("embeddings_enabled", cfg.IndexBuilding.EnableEmbeddings))

	// Initialize all services using the new initialization module
	opts := init_services.GetIndexBuildingOptions(cfg)
	container, err := init_services.NewServiceContainer(cfg, opts, logger)
	if err != nil {
		logger.Fatal("Failed to initialize services", zap.Error(err))
		return
	}
	defer container.Close(ctx)

	// Initialize processors based on configuration
	if err := container.InitProcessors(cfg); err != nil {
		logger.Fatal("Failed to initialize processors", zap.Error(err))
		return
	}

	// Process each repository
	for _, repoName := range repoNames {
		logger.Info("Processing repository for index building",
			zap.String("repo_name", repoName))

		// Validate repository exists in config
		repo, err := cfg.GetRepository(repoName)
		if err != nil {
			logger.Error("Repository not found in configuration",
				zap.String("repo_name", repoName),
				zap.Error(err))
			continue
		}

		logger.Info("Building indexes for repository",
			zap.String("repo_name", repo.Name),
			zap.String("path", repo.Path),
			zap.String("language", repo.Language))

		// Create FileVersionRepository for this repository
		fileVersionRepo, err := db.NewFileVersionRepository(container.MySQLConn.GetDB(), repo.Name, logger)
		if err != nil {
			logger.Error("Failed to create file version repository",
				zap.String("repo_name", repo.Name),
				zap.Error(err))
			continue
		}

		// Create index builder with FileVersionRepository for this specific repo
		indexBuilder := controller.NewIndexBuilder(cfg, container.Processors, fileVersionRepo, logger)

		// Get git info if using HEAD mode
		var gitInfo *util.GitInfo
		if useHead {
			gitInfo, err = util.GetGitInfo(repo.Path)
			if err != nil {
				logger.Error("Failed to get git info",
					zap.String("repo_name", repo.Name),
					zap.Error(err))
				continue
			}
			if !gitInfo.IsGitRepo {
				logger.Error("Repository is not a git repository, cannot use --head flag",
					zap.String("repo_name", repo.Name),
					zap.String("path", repo.Path))
				continue
			}
		}

		// Build all indexes using the unified index builder
		if err := indexBuilder.BuildIndexWithGitInfo(ctx, repo, useHead, gitInfo); err != nil {
			logger.Error("Failed to build indexes for repository",
				zap.String("repo_name", repo.Name),
				zap.Error(err))
			continue
		}

		logger.Info("Completed index building for repository",
			zap.String("repo_name", repo.Name))
	}

	// If test-dump is specified, dump the code graph after all processing is complete
	if testDumpPath != "" && container.CodeGraph != nil {
		logger.Info("Dumping code graph to file", zap.String("path", testDumpPath))
		if err := container.CodeGraph.DumpToFile(ctx, testDumpPath, repoNames); err != nil {
			logger.Error("Failed to dump code graph", zap.Error(err))
		} else {
			logger.Info("Code graph dumped successfully", zap.String("path", testDumpPath))
		}
	} else if testDumpPath != "" && container.CodeGraph == nil {
		logger.Warn("Cannot dump code graph: CodeGraph is not enabled")
	}

	// If clean is specified, clean up all DB entries for each repository
	if clean {
		logger.Info("Starting cleanup phase for all repositories")
		for _, repoName := range repoNames {
			logger.Info("Cleaning up repository data", zap.String("repo_name", repoName))

			// Clean Neo4j (CodeGraph)
			if container.CodeGraph != nil {
				logger.Info("Cleaning Neo4j data", zap.String("repo_name", repoName))
				if err := container.CodeGraph.CleanRepository(ctx, repoName); err != nil {
					logger.Error("Failed to clean Neo4j data",
						zap.String("repo_name", repoName),
						zap.Error(err))
				} else {
					logger.Info("Neo4j data cleaned successfully", zap.String("repo_name", repoName))
				}
			}

			// Clean Qdrant (Vector DB)
			if container.VectorDB != nil {
				logger.Info("Cleaning Qdrant collection", zap.String("repo_name", repoName))
				// Use repo name as collection name (default convention)
				if err := container.VectorDB.DeleteCollection(ctx, repoName); err != nil {
					logger.Error("Failed to clean Qdrant collection",
						zap.String("repo_name", repoName),
						zap.Error(err))
				} else {
					logger.Info("Qdrant collection cleaned successfully", zap.String("repo_name", repoName))
				}
			}

			// Clean MySQL (FileVersionRepository)
			if container.MySQLConn != nil {
				logger.Info("Cleaning MySQL table", zap.String("repo_name", repoName))
				fileVersionRepo, err := db.NewFileVersionRepository(container.MySQLConn.GetDB(), repoName, logger)
				if err != nil {
					logger.Error("Failed to create file version repository for cleanup",
						zap.String("repo_name", repoName),
						zap.Error(err))
				} else {
					if err := fileVersionRepo.DropTable(); err != nil {
						logger.Error("Failed to drop MySQL table",
							zap.String("repo_name", repoName),
							zap.Error(err))
					} else {
						logger.Info("MySQL table dropped successfully", zap.String("repo_name", repoName))
					}
				}
			}

			logger.Info("Cleanup completed for repository", zap.String("repo_name", repoName))
		}
		logger.Info("Cleanup phase completed for all repositories")
	}

	logger.Info("Build index command completed")
}

func CodeGraphEntry(cfg *config.Config, logger *zap.Logger, container *init_services.ServiceContainer) {
	if !cfg.App.CodeGraph {
		logger.Info("CodeGraph is disabled in the configuration")
		return
	}
	ctx := context.Background()

	// Initialize processors for CodeGraph-only mode
	if err := container.InitProcessors(cfg); err != nil {
		logger.Fatal("Failed to initialize processors", zap.Error(err))
		return
	}

	// Start processing repositories in a goroutine
	go func() {
		logger.Info("Starting repository processing thread")

		for _, repo := range cfg.Source.Repositories {
			if repo.Disabled {
				logger.Info("Skipping disabled repository", zap.String("name", repo.Name))
				continue
			}

			logger.Info("Processing repository", zap.String("name", repo.Name))

			// Create FileVersionRepository for this repository if MySQL is available
			var fileVersionRepo *db.FileVersionRepository
			var err error
			if container.MySQLConn != nil {
				fileVersionRepo, err = db.NewFileVersionRepository(container.MySQLConn.GetDB(), repo.Name, logger)
				if err != nil {
					logger.Error("Failed to create file version repository, will process without FileID tracking",
						zap.String("name", repo.Name),
						zap.Error(err))
					fileVersionRepo = nil
				}
			}

			// Create index builder for this repository
			// If fileVersionRepo is nil, IndexBuilder will fail - this is intentional to enforce MySQL requirement
			if fileVersionRepo == nil {
				logger.Error("Skipping repository - MySQL FileID tracking is required",
					zap.String("name", repo.Name))
				continue
			}

			indexBuilder := controller.NewIndexBuilder(cfg, container.Processors, fileVersionRepo, logger)

			err = indexBuilder.BuildIndex(ctx, &repo)
			if err != nil {
				logger.Error("Failed to process repository",
					zap.String("name", repo.Name),
					zap.Error(err))
				continue
			}
			logger.Info("Completed processing repository", zap.String("name", repo.Name))
		}

		logger.Info("Repository processing thread completed")
	}()
}
