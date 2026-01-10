package controller

import (
	"bufio"
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/armchr/codeapi/internal/config"
	"github.com/armchr/codeapi/internal/db"
	"github.com/armchr/codeapi/internal/model/ast"
	"github.com/armchr/codeapi/internal/service/codegraph"
	"github.com/armchr/codeapi/internal/service/llm"
	"github.com/armchr/codeapi/internal/service/summary"
	"github.com/armchr/codeapi/pkg/lsp/base"

	"go.uber.org/zap"
)

// SummaryProcessor generates hierarchical code summaries
type SummaryProcessor struct {
	llmService    llm.LLMService
	promptManager *summary.PromptManager
	codeGraph     *codegraph.CodeGraph
	mysqlDB       *sql.DB // For creating per-repo summary stores
	config        *SummaryProcessorConfig
	logger        *zap.Logger

	// Per-repo summary stores (created in Init)
	storesMu     sync.RWMutex
	stores       map[string]*db.SummaryStore
	currentStore *db.SummaryStore // Store for the current repository being processed
}

// SummaryProcessorConfig holds configuration for the summary processor
type SummaryProcessorConfig struct {
	Enabled      bool
	WorkerCount  int
	SkipIfExists bool // Skip if summary exists and context unchanged
	BatchSize    int
}

// NewSummaryProcessor creates a new summary processor
func NewSummaryProcessor(
	llmService llm.LLMService,
	promptManager *summary.PromptManager,
	mysqlDB *sql.DB, // For creating per-repo summary stores
	codeGraph *codegraph.CodeGraph,
	config *SummaryProcessorConfig,
	logger *zap.Logger,
) *SummaryProcessor {
	if config == nil {
		config = &SummaryProcessorConfig{
			Enabled:      true,
			WorkerCount:  4,
			SkipIfExists: true,
			BatchSize:    50,
		}
	}

	if config.WorkerCount <= 0 {
		config.WorkerCount = 4
	}

	return &SummaryProcessor{
		llmService:    llmService,
		promptManager: promptManager,
		mysqlDB:       mysqlDB,
		codeGraph:     codeGraph,
		config:        config,
		logger:        logger,
		stores:        make(map[string]*db.SummaryStore),
	}
}

// Name returns the processor name
func (p *SummaryProcessor) Name() string {
	return "SummaryProcessor"
}

// Init initializes the summary store for the repository
func (p *SummaryProcessor) Init(ctx context.Context, repo *config.Repository) error {
	if !p.config.Enabled {
		return nil
	}

	store, err := p.getOrCreateStore(repo.Name)
	if err != nil {
		return err
	}
	p.currentStore = store
	p.logger.Info("Initialized SummaryProcessor for repository", zap.String("repo", repo.Name))
	return nil
}

// getOrCreateStore returns the summary store for a repository, creating it if needed
func (p *SummaryProcessor) getOrCreateStore(repoName string) (*db.SummaryStore, error) {
	// Fast path: check if store already exists
	p.storesMu.RLock()
	store, exists := p.stores[repoName]
	p.storesMu.RUnlock()
	if exists {
		return store, nil
	}

	// Slow path: create store with write lock
	p.storesMu.Lock()
	defer p.storesMu.Unlock()

	// Double-check after acquiring write lock
	if store, exists = p.stores[repoName]; exists {
		return store, nil
	}

	if p.mysqlDB == nil {
		return nil, fmt.Errorf("MySQL database connection required for summary storage")
	}

	var err error
	store, err = db.NewSummaryStore(p.mysqlDB, repoName, p.logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create summary store for %s: %w", repoName, err)
	}

	p.stores[repoName] = store
	p.logger.Info("Created summary store for repository", zap.String("repo", repoName))
	return store, nil
}

// ProcessFile generates summaries for functions, classes, and the file itself
// This runs after CodeGraphProcessor has already populated the code graph for this file
func (p *SummaryProcessor) ProcessFile(ctx context.Context, repo *config.Repository, fileCtx *FileContext) error {
	if !p.config.Enabled {
		return nil
	}

	if p.currentStore == nil {
		return fmt.Errorf("SummaryProcessor not initialized - Init must be called before ProcessFile")
	}

	// Skip files without parser support (e.g., .classpath, .project, pom.xml, ruby files)
	if !isSupportedForSummary(fileCtx.RelativePath) {
		p.logger.Debug("Skipping unsupported file for summarization",
			zap.String("file", fileCtx.RelativePath))
		return nil
	}

	p.logger.Debug("Processing file for summaries",
		zap.String("file", fileCtx.RelativePath),
		zap.Int32("fileID", fileCtx.FileID))

	// Step 1: Summarize all functions in this file
	functions, err := p.codeGraph.GetNodesByTypeAndFileID(ctx, ast.NodeTypeFunction, fileCtx.FileID)
	if err != nil {
		p.logger.Error("Failed to get functions for file", zap.Error(err))
		// Continue - we can still try to process other entities
	} else {
		for _, fn := range functions {
			if err := p.summarizeFunction(ctx, fn, repo, p.currentStore); err != nil {
				p.logger.Error("Failed to summarize function",
					zap.String("function", fn.Name),
					zap.Error(err))
				// Continue with other functions
			}
		}
	}

	// Step 2: Summarize all classes in this file (using function summaries)
	classes, err := p.codeGraph.GetNodesByTypeAndFileID(ctx, ast.NodeTypeClass, fileCtx.FileID)
	if err != nil {
		p.logger.Error("Failed to get classes for file", zap.Error(err))
	} else {
		for _, cls := range classes {
			if err := p.summarizeClass(ctx, cls, repo, p.currentStore); err != nil {
				p.logger.Error("Failed to summarize class",
					zap.String("class", cls.Name),
					zap.Error(err))
			}
		}
	}

	// Step 3: Summarize the file itself (using function and class summaries)
	if err := p.summarizeFile(ctx, fileCtx, repo, p.currentStore); err != nil {
		p.logger.Error("Failed to summarize file",
			zap.String("file", fileCtx.RelativePath),
			zap.Error(err))
		return err
	}

	return nil
}

// PostProcess generates folder and project level summaries
// These require all files to be processed first
func (p *SummaryProcessor) PostProcess(ctx context.Context, repo *config.Repository) error {
	if !p.config.Enabled {
		p.logger.Info("Summary processor is disabled, skipping")
		return nil
	}

	if p.currentStore == nil {
		return fmt.Errorf("SummaryProcessor not initialized - Init must be called before PostProcess")
	}

	p.logger.Info("Starting folder and project summary generation", zap.String("repo", repo.Name))

	// Level 4: Folders (bottom-up)
	if err := p.summarizeFolders(ctx, repo, p.currentStore); err != nil {
		p.logger.Error("Failed to summarize folders", zap.Error(err))
		return err
	}

	// Level 5: Project
	if err := p.summarizeProject(ctx, repo, p.currentStore); err != nil {
		p.logger.Error("Failed to summarize project", zap.Error(err))
		return err
	}

	p.logger.Info("Completed folder and project summary generation", zap.String("repo", repo.Name))
	return nil
}

// summarizeFunction generates a summary for a single function
func (p *SummaryProcessor) summarizeFunction(
	ctx context.Context,
	node *ast.Node,
	repo *config.Repository,
	store *db.SummaryStore,
) error {
	entityID := strconv.FormatInt(int64(node.ID), 10)
	contextBuilder := summary.NewContextBuilder(4000)
	fnCtx := p.buildFunctionContext(ctx, node, repo)
	contextHash := contextBuilder.HashContext(fnCtx)

	// Check if update needed
	if p.config.SkipIfExists {
		needsUpdate, err := store.NeedsUpdate(entityID, summary.LevelFunction, contextHash)
		if err != nil {
			return err
		}
		if !needsUpdate {
			p.logger.Debug("Skipping function - unchanged", zap.String("name", node.Name))
			return nil
		}
	}

	// Generate summary
	systemPrompt, userPrompt, err := p.promptManager.RenderPrompt(summary.LevelFunction, fnCtx)
	if err != nil {
		return fmt.Errorf("failed to render prompt: %w", err)
	}

	tmpl, _ := p.promptManager.GetTemplate(summary.LevelFunction)
	opts := llm.GenerateOptions{
		MaxTokens:   tmpl.MaxTokens,
		Temperature: tmpl.Temperature,
	}

	resp, err := p.llmService.GenerateWithSystem(ctx, systemPrompt, userPrompt, opts)
	if err != nil {
		return fmt.Errorf("failed to generate summary: %w", err)
	}

	// Get file path
	filePath := p.codeGraph.GetFilePath(ctx, node.FileID)

	// Store summary
	cs := &summary.CodeSummary{
		EntityID:     entityID,
		EntityType:   summary.LevelFunction,
		EntityName:   node.Name,
		FilePath:     filePath,
		Summary:      resp.Content,
		ContextHash:  contextHash,
		LLMProvider:  p.llmService.Name(),
		LLMModel:     p.llmService.ModelName(),
		PromptTokens: resp.PromptTokens,
		OutputTokens: resp.OutputTokens,
	}

	return store.SaveSummary(cs)
}

// summarizeClass generates a summary for a single class using method summaries
func (p *SummaryProcessor) summarizeClass(
	ctx context.Context,
	node *ast.Node,
	repo *config.Repository,
	store *db.SummaryStore,
) error {
	entityID := strconv.FormatInt(int64(node.ID), 10)
	contextBuilder := summary.NewContextBuilder(8000)
	clsCtx := p.buildClassContext(ctx, node, repo, store)
	contextHash := contextBuilder.HashContext(clsCtx)

	// Check if update needed
	if p.config.SkipIfExists {
		needsUpdate, err := store.NeedsUpdate(entityID, summary.LevelClass, contextHash)
		if err != nil {
			return err
		}
		if !needsUpdate {
			p.logger.Debug("Skipping class - unchanged", zap.String("name", node.Name))
			return nil
		}
	}

	// Generate summary
	systemPrompt, userPrompt, err := p.promptManager.RenderPrompt(summary.LevelClass, clsCtx)
	if err != nil {
		return fmt.Errorf("failed to render prompt: %w", err)
	}

	tmpl, _ := p.promptManager.GetTemplate(summary.LevelClass)
	opts := llm.GenerateOptions{
		MaxTokens:   tmpl.MaxTokens,
		Temperature: tmpl.Temperature,
	}

	resp, err := p.llmService.GenerateWithSystem(ctx, systemPrompt, userPrompt, opts)
	if err != nil {
		return fmt.Errorf("failed to generate summary: %w", err)
	}

	// Get file path
	filePath := p.codeGraph.GetFilePath(ctx, node.FileID)

	// Store summary
	cs := &summary.CodeSummary{
		EntityID:     entityID,
		EntityType:   summary.LevelClass,
		EntityName:   node.Name,
		FilePath:     filePath,
		Summary:      resp.Content,
		ContextHash:  contextHash,
		LLMProvider:  p.llmService.Name(),
		LLMModel:     p.llmService.ModelName(),
		PromptTokens: resp.PromptTokens,
		OutputTokens: resp.OutputTokens,
	}

	return store.SaveSummary(cs)
}

// summarizeFile generates a summary for a file using class and function summaries
func (p *SummaryProcessor) summarizeFile(
	ctx context.Context,
	fileCtx *FileContext,
	repo *config.Repository,
	store *db.SummaryStore,
) error {
	// Use relative path as entity ID for files
	entityID := fileCtx.RelativePath
	contextBuilder := summary.NewContextBuilder(8000)
	fileSummaryCtx := p.buildFileContextFromFileCtx(ctx, fileCtx, repo, store)
	contextHash := contextBuilder.HashContext(fileSummaryCtx)

	// Check if update needed
	if p.config.SkipIfExists {
		needsUpdate, err := store.NeedsUpdate(entityID, summary.LevelFile, contextHash)
		if err != nil {
			return err
		}
		if !needsUpdate {
			p.logger.Debug("Skipping file - unchanged", zap.String("file", fileCtx.RelativePath))
			return nil
		}
	}

	// Generate summary
	systemPrompt, userPrompt, err := p.promptManager.RenderPrompt(summary.LevelFile, fileSummaryCtx)
	if err != nil {
		return fmt.Errorf("failed to render prompt: %w", err)
	}

	tmpl, _ := p.promptManager.GetTemplate(summary.LevelFile)
	opts := llm.GenerateOptions{
		MaxTokens:   tmpl.MaxTokens,
		Temperature: tmpl.Temperature,
	}

	resp, err := p.llmService.GenerateWithSystem(ctx, systemPrompt, userPrompt, opts)
	if err != nil {
		return fmt.Errorf("failed to generate summary: %w", err)
	}

	// Store summary
	cs := &summary.CodeSummary{
		EntityID:     entityID,
		EntityType:   summary.LevelFile,
		EntityName:   filepath.Base(fileCtx.RelativePath),
		FilePath:     fileCtx.RelativePath,
		Summary:      resp.Content,
		ContextHash:  contextHash,
		LLMProvider:  p.llmService.Name(),
		LLMModel:     p.llmService.ModelName(),
		PromptTokens: resp.PromptTokens,
		OutputTokens: resp.OutputTokens,
	}

	p.logger.Debug("Generated file summary",
		zap.String("file", fileCtx.RelativePath),
		zap.Int("prompt_tokens", resp.PromptTokens),
		zap.Int("output_tokens", resp.OutputTokens))

	return store.SaveSummary(cs)
}

// summarizeFolders generates summaries for folders bottom-up
func (p *SummaryProcessor) summarizeFolders(ctx context.Context, repo *config.Repository, store *db.SummaryStore) error {
	p.logger.Info("Summarizing folders", zap.String("repo", repo.Name))

	// Get all file summaries to build folder hierarchy
	fileSummaries, err := store.GetSummariesByType(summary.LevelFile)
	if err != nil {
		return fmt.Errorf("failed to get file summaries: %w", err)
	}

	// Build folder hierarchy
	folderFiles := make(map[string][]summary.EntitySummary)
	allFolders := make(map[string]bool)

	for _, fs := range fileSummaries {
		dir := filepath.Dir(fs.FilePath)
		folderFiles[dir] = append(folderFiles[dir], summary.EntitySummary{
			Name:     filepath.Base(fs.FilePath),
			Summary:  fs.Summary,
			FilePath: fs.FilePath,
		})

		// Track all folder paths up to root
		for d := dir; d != "." && d != "/" && d != ""; d = filepath.Dir(d) {
			allFolders[d] = true
		}
	}

	// Sort folders by depth (deepest first for bottom-up processing)
	sortedFolders := make([]string, 0, len(allFolders))
	for folder := range allFolders {
		sortedFolders = append(sortedFolders, folder)
	}
	sort.Slice(sortedFolders, func(i, j int) bool {
		return strings.Count(sortedFolders[i], string(filepath.Separator)) >
			strings.Count(sortedFolders[j], string(filepath.Separator))
	})

	p.logger.Info("Found folders to summarize",
		zap.Int("count", len(sortedFolders)),
		zap.String("repo", repo.Name))

	// Process folders bottom-up (deepest first)
	for _, folder := range sortedFolders {
		if err := p.summarizeFolder(ctx, folder, folderFiles, repo, store); err != nil {
			p.logger.Error("Failed to summarize folder",
				zap.String("folder", folder),
				zap.Error(err))
			continue // Continue with other folders
		}
	}

	return nil
}

// summarizeFolder generates a summary for a single folder
func (p *SummaryProcessor) summarizeFolder(
	ctx context.Context,
	folderPath string,
	folderFiles map[string][]summary.EntitySummary,
	repo *config.Repository,
	store *db.SummaryStore,
) error {
	// Get file summaries for this folder
	fileSummaries := folderFiles[folderPath]

	// Get subfolder summaries
	var subfolderSummaries []summary.EntitySummary
	for subFolder := range folderFiles {
		if filepath.Dir(subFolder) == folderPath {
			// Get the summary for this subfolder
			existing, err := store.GetSummary(subFolder, summary.LevelFolder)
			if err == nil && existing != nil {
				subfolderSummaries = append(subfolderSummaries, summary.EntitySummary{
					Name:    filepath.Base(subFolder),
					Summary: existing.Summary,
				})
			}
		}
	}

	// Build context
	contextBuilder := summary.NewContextBuilder(12000)
	folderCtx := contextBuilder.BuildFolderContext(
		folderPath,
		fileSummaries,
		subfolderSummaries,
		[]string{repo.Language},
	)

	// Check if update needed
	contextHash := contextBuilder.HashContext(folderCtx)
	if p.config.SkipIfExists {
		needsUpdate, err := store.NeedsUpdate(folderPath, summary.LevelFolder, contextHash)
		if err != nil {
			return err
		}
		if !needsUpdate {
			p.logger.Debug("Skipping folder - unchanged", zap.String("folder", folderPath))
			return nil
		}
	}

	// Generate summary
	systemPrompt, userPrompt, err := p.promptManager.RenderPrompt(summary.LevelFolder, folderCtx)
	if err != nil {
		return fmt.Errorf("failed to render prompt: %w", err)
	}

	tmpl, _ := p.promptManager.GetTemplate(summary.LevelFolder)
	opts := llm.GenerateOptions{
		MaxTokens:   tmpl.MaxTokens,
		Temperature: tmpl.Temperature,
	}

	resp, err := p.llmService.GenerateWithSystem(ctx, systemPrompt, userPrompt, opts)
	if err != nil {
		return fmt.Errorf("failed to generate summary: %w", err)
	}

	// Store summary
	cs := &summary.CodeSummary{
		EntityID:     folderPath,
		EntityType:   summary.LevelFolder,
		EntityName:   filepath.Base(folderPath),
		FilePath:     folderPath,
		Summary:      resp.Content,
		ContextHash:  contextHash,
		LLMProvider:  p.llmService.Name(),
		LLMModel:     p.llmService.ModelName(),
		PromptTokens: resp.PromptTokens,
		OutputTokens: resp.OutputTokens,
	}

	return store.SaveSummary(cs)
}

// summarizeProject generates a project-level summary
func (p *SummaryProcessor) summarizeProject(ctx context.Context, repo *config.Repository, store *db.SummaryStore) error {
	p.logger.Info("Summarizing project", zap.String("repo", repo.Name))

	// Get top-level folder summaries
	folderSummaries, err := store.GetSummariesByType(summary.LevelFolder)
	if err != nil {
		return fmt.Errorf("failed to get folder summaries: %w", err)
	}

	// Find top-level folders (those with shortest path depth)
	var topLevelSummaries []summary.EntitySummary
	minDepth := -1
	for _, fs := range folderSummaries {
		depth := strings.Count(fs.FilePath, string(filepath.Separator))
		if minDepth == -1 || depth < minDepth {
			minDepth = depth
		}
	}
	for _, fs := range folderSummaries {
		depth := strings.Count(fs.FilePath, string(filepath.Separator))
		if depth == minDepth {
			topLevelSummaries = append(topLevelSummaries, summary.EntitySummary{
				Name:    filepath.Base(fs.FilePath),
				Summary: fs.Summary,
			})
		}
	}

	// Get statistics
	fileSummaries, _ := store.GetSummariesByType(summary.LevelFile)
	classSummaries, _ := store.GetSummariesByType(summary.LevelClass)
	functionSummaries, _ := store.GetSummariesByType(summary.LevelFunction)

	// Find entry points (main files)
	var entryPoints []string
	for _, fs := range fileSummaries {
		name := strings.ToLower(filepath.Base(fs.FilePath))
		if strings.Contains(name, "main") || strings.HasPrefix(name, "index") ||
			strings.HasPrefix(name, "app.") {
			entryPoints = append(entryPoints, fs.FilePath)
		}
	}

	// Build context
	contextBuilder := summary.NewContextBuilder(16000)
	projectCtx := contextBuilder.BuildProjectContext(
		repo.Name,
		[]string{repo.Language},
		topLevelSummaries,
		entryPoints,
		len(fileSummaries),
		len(classSummaries),
		len(functionSummaries),
	)

	// Check if update needed
	contextHash := contextBuilder.HashContext(projectCtx)
	if p.config.SkipIfExists {
		needsUpdate, err := store.NeedsUpdate(repo.Name, summary.LevelProject, contextHash)
		if err != nil {
			return err
		}
		if !needsUpdate {
			p.logger.Debug("Skipping project - unchanged", zap.String("repo", repo.Name))
			return nil
		}
	}

	// Generate summary
	systemPrompt, userPrompt, err := p.promptManager.RenderPrompt(summary.LevelProject, projectCtx)
	if err != nil {
		return fmt.Errorf("failed to render prompt: %w", err)
	}

	tmpl, _ := p.promptManager.GetTemplate(summary.LevelProject)
	opts := llm.GenerateOptions{
		MaxTokens:   tmpl.MaxTokens,
		Temperature: tmpl.Temperature,
	}

	resp, err := p.llmService.GenerateWithSystem(ctx, systemPrompt, userPrompt, opts)
	if err != nil {
		return fmt.Errorf("failed to generate summary: %w", err)
	}

	// Store summary
	cs := &summary.CodeSummary{
		EntityID:     repo.Name,
		EntityType:   summary.LevelProject,
		EntityName:   repo.Name,
		FilePath:     repo.Path,
		Summary:      resp.Content,
		ContextHash:  contextHash,
		LLMProvider:  p.llmService.Name(),
		LLMModel:     p.llmService.ModelName(),
		PromptTokens: resp.PromptTokens,
		OutputTokens: resp.OutputTokens,
	}

	return store.SaveSummary(cs)
}

// buildFunctionContext builds context for function summarization
func (p *SummaryProcessor) buildFunctionContext(ctx context.Context, node *ast.Node, repo *config.Repository) *summary.FunctionContext {
	// Extract metadata
	var docstring, signature, returnType, className string
	var annotations, modifiers []string
	var params []summary.ParameterInfo

	if node.MetaData != nil {
		if ds, ok := node.MetaData["docstring"].(string); ok {
			docstring = ds
		}
		if sig, ok := node.MetaData["signature"].(string); ok {
			signature = sig
		}
		if ret, ok := node.MetaData["return"].(string); ok {
			returnType = ret
		}
		if ann, ok := node.MetaData["annotations"].([]string); ok {
			annotations = ann
		}
		if mod, ok := node.MetaData["modifiers"].([]string); ok {
			modifiers = mod
		}
	}

	// Get containing class if this is a method
	containingClass, _ := p.codeGraph.GetContainingClass(ctx, node.ID)
	if containingClass != nil {
		className = containingClass.Name
	}

	// Get file path and extract source code
	filePath := p.codeGraph.GetFilePath(ctx, node.FileID)
	sourceCode := p.extractSourceCode(repo.Path, filePath, node.Range)

	return &summary.FunctionContext{
		Name:        node.Name,
		Signature:   signature,
		Docstring:   docstring,
		SourceCode:  sourceCode,
		Parameters:  params,
		ReturnType:  returnType,
		Language:    repo.Language,
		FilePath:    filePath,
		ClassName:   className,
		Annotations: annotations,
		Modifiers:   modifiers,
	}
}

// buildClassContext builds context for class summarization using method summaries from store
func (p *SummaryProcessor) buildClassContext(
	ctx context.Context,
	node *ast.Node,
	repo *config.Repository,
	store *db.SummaryStore,
) *summary.ClassContext {
	var docstring string
	var inheritance, implements, annotations, modifiers []string

	if node.MetaData != nil {
		if ds, ok := node.MetaData["docstring"].(string); ok {
			docstring = ds
		}
		if inh, ok := node.MetaData["extends"].([]string); ok {
			inheritance = inh
		}
		if impl, ok := node.MetaData["implements"].([]string); ok {
			implements = impl
		}
		if ann, ok := node.MetaData["annotations"].([]string); ok {
			annotations = ann
		}
		if mod, ok := node.MetaData["modifiers"].([]string); ok {
			modifiers = mod
		}
	}

	// Get methods and their summaries
	methods, _ := p.codeGraph.GetMethodsOfClass(ctx, node.ID)
	var methodSummaries []summary.EntitySummary
	for _, method := range methods {
		methodID := strconv.FormatInt(int64(method.ID), 10)
		existing, err := store.GetSummary(methodID, summary.LevelFunction)
		if err == nil && existing != nil {
			methodSummaries = append(methodSummaries, summary.EntitySummary{
				Name:    method.Name,
				Summary: existing.Summary,
			})
		}
	}

	// Get fields
	fieldsNodes, _ := p.codeGraph.GetFieldsOfClass(ctx, node.ID)
	var fields []summary.FieldInfo
	for _, field := range fieldsNodes {
		fieldType := ""
		var fieldModifiers []string
		if field.MetaData != nil {
			if ft, ok := field.MetaData["type"].(string); ok {
				fieldType = ft
			}
			if fm, ok := field.MetaData["modifiers"].([]string); ok {
				fieldModifiers = fm
			}
		}
		fields = append(fields, summary.FieldInfo{
			Name:      field.Name,
			Type:      fieldType,
			Modifiers: fieldModifiers,
		})
	}

	filePath := p.codeGraph.GetFilePath(ctx, node.FileID)

	return &summary.ClassContext{
		Name:            node.Name,
		Docstring:       docstring,
		Inheritance:     inheritance,
		Implements:      implements,
		Fields:          fields,
		MethodSummaries: methodSummaries,
		Language:        repo.Language,
		FilePath:        filePath,
		Annotations:     annotations,
		Modifiers:       modifiers,
	}
}

// buildFileContextFromFileCtx builds context for file summarization using FileContext
func (p *SummaryProcessor) buildFileContextFromFileCtx(
	ctx context.Context,
	fileCtx *FileContext,
	repo *config.Repository,
	store *db.SummaryStore,
) *summary.FileContext {
	// Get classes in file and their summaries
	classes, _ := p.codeGraph.GetNodesByTypeAndFileID(ctx, ast.NodeTypeClass, fileCtx.FileID)
	var classSummaries []summary.EntitySummary
	for _, cls := range classes {
		clsID := strconv.FormatInt(int64(cls.ID), 10)
		existing, err := store.GetSummary(clsID, summary.LevelClass)
		if err == nil && existing != nil {
			classSummaries = append(classSummaries, summary.EntitySummary{
				Name:    cls.Name,
				Summary: existing.Summary,
			})
		}
	}

	// Get top-level functions and their summaries
	functions, _ := p.codeGraph.GetNodesByTypeAndFileID(ctx, ast.NodeTypeFunction, fileCtx.FileID)
	var functionSummaries []summary.EntitySummary
	for _, fn := range functions {
		// Skip methods (functions inside classes)
		containingClass, _ := p.codeGraph.GetContainingClass(ctx, fn.ID)
		if containingClass != nil {
			continue // Skip methods, only include top-level functions
		}

		fnID := strconv.FormatInt(int64(fn.ID), 10)
		existing, err := store.GetSummary(fnID, summary.LevelFunction)
		if err == nil && existing != nil {
			functionSummaries = append(functionSummaries, summary.EntitySummary{
				Name:    fn.Name,
				Summary: existing.Summary,
			})
		}
	}

	// Get imports
	imports, _ := p.codeGraph.GetNodesByTypeAndFileID(ctx, ast.NodeTypeImport, fileCtx.FileID)
	var importNames []string
	for _, imp := range imports {
		importNames = append(importNames, imp.Name)
	}

	// Get module name from code graph
	moduleName, _ := p.codeGraph.GetModuleName(ctx, fileCtx.FileID)

	return &summary.FileContext{
		FilePath:          fileCtx.RelativePath,
		FileName:          filepath.Base(fileCtx.RelativePath),
		Language:          repo.Language,
		Imports:           importNames,
		ClassSummaries:    classSummaries,
		FunctionSummaries: functionSummaries,
		PackageName:       "",
		ModuleName:        moduleName,
	}
}

// isSupportedForSummary checks if a file has parser support for summarization
// Files without parsers (like .classpath, .project, pom.xml, etc.) should be skipped
func isSupportedForSummary(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".go", ".js", ".jsx", ".mjs", ".ts", ".tsx", ".py", ".pyw", ".java", ".cs":
		return true
	default:
		return false
	}
}

// extractSourceCode reads source code from a file for a given range.
// It returns the source code lines from start to end (inclusive).
// If the range is invalid or file cannot be read, returns empty string.
func (p *SummaryProcessor) extractSourceCode(repoPath, relativePath string, rng base.Range) string {
	if relativePath == "" {
		return ""
	}

	fullPath := filepath.Join(repoPath, relativePath)

	file, err := os.Open(fullPath)
	if err != nil {
		p.logger.Debug("Failed to open file for source extraction",
			zap.String("path", fullPath),
			zap.Error(err))
		return ""
	}
	defer file.Close()

	// LSP positions are 0-indexed, so line 0 is the first line
	startLine := rng.Start.Line
	endLine := rng.End.Line

	// Sanity check
	if startLine < 0 || endLine < startLine {
		return ""
	}

	var lines []string
	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		if lineNum >= startLine && lineNum <= endLine {
			lines = append(lines, scanner.Text())
		}
		if lineNum > endLine {
			break
		}
		lineNum++
	}

	if err := scanner.Err(); err != nil {
		p.logger.Debug("Error reading file for source extraction",
			zap.String("path", fullPath),
			zap.Error(err))
		return ""
	}

	return strings.Join(lines, "\n")
}

// -----------------------------------------------------------------------------
// On-Demand Summary Generation (for API requests)
// -----------------------------------------------------------------------------

// GenerateFunctionSummaryOnDemand generates a summary for a function by name
// This is used when the API is called but no summary exists
// Note: This does not check p.config.Enabled since on-demand generation should always work
func (p *SummaryProcessor) GenerateFunctionSummaryOnDemand(
	ctx context.Context,
	repo *config.Repository,
	filePath string,
	functionName string,
) (*summary.CodeSummary, error) {
	store, err := p.getOrCreateStore(repo.Name)
	if err != nil {
		return nil, err
	}

	// Find the function node in the code graph
	node, err := p.codeGraph.FindFunctionByName(ctx, filePath, functionName)
	if err != nil {
		return nil, fmt.Errorf("failed to find function %s in %s: %w", functionName, filePath, err)
	}
	if node == nil {
		return nil, fmt.Errorf("function %s not found in %s", functionName, filePath)
	}

	// Generate the summary
	if err := p.summarizeFunction(ctx, node, repo, store); err != nil {
		return nil, fmt.Errorf("failed to generate function summary: %w", err)
	}

	// Retrieve and return the generated summary
	entityID := strconv.FormatInt(int64(node.ID), 10)
	return store.GetSummary(entityID, summary.LevelFunction)
}

// GenerateClassSummaryOnDemand generates a summary for a class by name
// This is used when the API is called but no summary exists
// Note: This does not check p.config.Enabled since on-demand generation should always work
func (p *SummaryProcessor) GenerateClassSummaryOnDemand(
	ctx context.Context,
	repo *config.Repository,
	filePath string,
	className string,
) (*summary.CodeSummary, error) {
	store, err := p.getOrCreateStore(repo.Name)
	if err != nil {
		return nil, err
	}

	// Find the class node in the code graph
	node, err := p.codeGraph.FindClassByName(ctx, filePath, className)
	if err != nil {
		return nil, fmt.Errorf("failed to find class %s in %s: %w", className, filePath, err)
	}
	if node == nil {
		return nil, fmt.Errorf("class %s not found in %s", className, filePath)
	}

	// First, ensure all methods in the class have summaries (for hierarchical summarization)
	methods, _ := p.codeGraph.GetClassMethods(ctx, node.ID)
	for _, method := range methods {
		// Check if method summary exists
		methodEntityID := strconv.FormatInt(int64(method.ID), 10)
		existing, _ := store.GetSummary(methodEntityID, summary.LevelFunction)
		if existing == nil {
			// Generate method summary first
			_ = p.summarizeFunction(ctx, method, repo, store)
		}
	}

	// Generate the class summary
	if err := p.summarizeClass(ctx, node, repo, store); err != nil {
		return nil, fmt.Errorf("failed to generate class summary: %w", err)
	}

	// Retrieve and return the generated summary
	entityID := strconv.FormatInt(int64(node.ID), 10)
	return store.GetSummary(entityID, summary.LevelClass)
}

// GenerateFileSummaryOnDemand generates a summary for a file by path
// This is used when the API is called but no summary exists
// Note: This does not check p.config.Enabled since on-demand generation should always work
func (p *SummaryProcessor) GenerateFileSummaryOnDemand(
	ctx context.Context,
	repo *config.Repository,
	filePath string,
) (*summary.CodeSummary, error) {
	// Skip unsupported files
	if !isSupportedForSummary(filePath) {
		return nil, fmt.Errorf("file type not supported for summarization: %s", filePath)
	}

	store, err := p.getOrCreateStore(repo.Name)
	if err != nil {
		return nil, err
	}

	// Find the file in the code graph
	fileNode, err := p.codeGraph.FindFileByPath(ctx, repo.Name, filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to find file %s: %w", filePath, err)
	}
	if fileNode == nil {
		return nil, fmt.Errorf("file %s not found in code graph", filePath)
	}

	// First, generate summaries for all functions and classes in the file
	functions, _ := p.codeGraph.GetNodesByTypeAndFileID(ctx, ast.NodeTypeFunction, fileNode.FileID)
	for _, fn := range functions {
		fnEntityID := strconv.FormatInt(int64(fn.ID), 10)
		existing, _ := store.GetSummary(fnEntityID, summary.LevelFunction)
		if existing == nil {
			_ = p.summarizeFunction(ctx, fn, repo, store)
		}
	}

	classes, _ := p.codeGraph.GetNodesByTypeAndFileID(ctx, ast.NodeTypeClass, fileNode.FileID)
	for _, cls := range classes {
		clsEntityID := strconv.FormatInt(int64(cls.ID), 10)
		existing, _ := store.GetSummary(clsEntityID, summary.LevelClass)
		if existing == nil {
			// Generate method summaries first
			methods, _ := p.codeGraph.GetClassMethods(ctx, cls.ID)
			for _, method := range methods {
				methodEntityID := strconv.FormatInt(int64(method.ID), 10)
				methodExisting, _ := store.GetSummary(methodEntityID, summary.LevelFunction)
				if methodExisting == nil {
					_ = p.summarizeFunction(ctx, method, repo, store)
				}
			}
			_ = p.summarizeClass(ctx, cls, repo, store)
		}
	}

	// Create FileContext for the file summary
	fileCtx := &FileContext{
		FileID:       fileNode.FileID,
		FilePath:     filepath.Join(repo.Path, filePath),
		RelativePath: filePath,
	}

	// Generate the file summary
	if err := p.summarizeFile(ctx, fileCtx, repo, store); err != nil {
		return nil, fmt.Errorf("failed to generate file summary: %w", err)
	}

	// Retrieve and return the generated summary
	return store.GetFileSummary(filePath)
}

// GenerateFileSummariesOnDemand generates summaries for all entities in a file.
// If entityType is specified (non-zero), only generates summaries for that type.
// Returns the list of generated summaries.
// Note: This does not check p.config.Enabled since on-demand generation should always work
func (p *SummaryProcessor) GenerateFileSummariesOnDemand(
	ctx context.Context,
	repo *config.Repository,
	filePath string,
	entityType summary.SummaryLevel,
) ([]*summary.CodeSummary, error) {
	// Skip unsupported files
	if !isSupportedForSummary(filePath) {
		return nil, fmt.Errorf("file type not supported for summarization: %s", filePath)
	}

	store, err := p.getOrCreateStore(repo.Name)
	if err != nil {
		return nil, err
	}

	// Find the file in the code graph
	p.logger.Debug("Looking up file in code graph",
		zap.String("repo", repo.Name),
		zap.String("filePath", filePath))
	fileNode, err := p.codeGraph.FindFileByPath(ctx, repo.Name, filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to find file %s: %w", filePath, err)
	}
	if fileNode == nil {
		return nil, fmt.Errorf("file %s not found in code graph", filePath)
	}
	p.logger.Debug("Found file in code graph",
		zap.Int32("fileID", fileNode.FileID),
		zap.Int64("nodeID", int64(fileNode.ID)))

	var generatedSummaries []*summary.CodeSummary

	// Generate function summaries if requested or if no filter
	if entityType == 0 || entityType == summary.LevelFunction {
		functions, _ := p.codeGraph.GetNodesByTypeAndFileID(ctx, ast.NodeTypeFunction, fileNode.FileID)
		for _, fn := range functions {
			fnEntityID := strconv.FormatInt(int64(fn.ID), 10)
			existing, _ := store.GetSummary(fnEntityID, summary.LevelFunction)
			if existing == nil {
				if err := p.summarizeFunction(ctx, fn, repo, store); err != nil {
					p.logger.Debug("Failed to generate function summary",
						zap.String("function", fn.Name),
						zap.Error(err))
					continue
				}
				// Retrieve the generated summary
				if generated, _ := store.GetSummary(fnEntityID, summary.LevelFunction); generated != nil {
					generatedSummaries = append(generatedSummaries, generated)
				}
			} else {
				generatedSummaries = append(generatedSummaries, existing)
			}
		}
	}

	// Generate class summaries if requested or if no filter
	if entityType == 0 || entityType == summary.LevelClass {
		classes, _ := p.codeGraph.GetNodesByTypeAndFileID(ctx, ast.NodeTypeClass, fileNode.FileID)
		for _, cls := range classes {
			clsEntityID := strconv.FormatInt(int64(cls.ID), 10)
			existing, _ := store.GetSummary(clsEntityID, summary.LevelClass)
			if existing == nil {
				// First ensure all methods have summaries
				methods, _ := p.codeGraph.GetClassMethods(ctx, cls.ID)
				for _, method := range methods {
					methodEntityID := strconv.FormatInt(int64(method.ID), 10)
					methodExisting, _ := store.GetSummary(methodEntityID, summary.LevelFunction)
					if methodExisting == nil {
						_ = p.summarizeFunction(ctx, method, repo, store)
					}
				}
				if err := p.summarizeClass(ctx, cls, repo, store); err != nil {
					p.logger.Debug("Failed to generate class summary",
						zap.String("class", cls.Name),
						zap.Error(err))
					continue
				}
				// Retrieve the generated summary
				if generated, _ := store.GetSummary(clsEntityID, summary.LevelClass); generated != nil {
					generatedSummaries = append(generatedSummaries, generated)
				}
			} else {
				generatedSummaries = append(generatedSummaries, existing)
			}
		}
	}

	return generatedSummaries, nil
}
