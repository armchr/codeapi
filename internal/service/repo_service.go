package service

import (
	"context"

	"github.com/armchr/codeapi/internal/config"
	"github.com/armchr/codeapi/internal/model"
	"github.com/armchr/codeapi/pkg/lsp"

	"go.uber.org/zap"
)

type RepoService struct {
	config     *config.Config
	logger     *zap.Logger
	lspService *lsp.LspService
}

func NewRepoService(config *config.Config, logger *zap.Logger) *RepoService {
	return &RepoService{
		config:     config,
		logger:     logger,
		lspService: lsp.NewLspService(config, logger),
	}
}

func (rs *RepoService) GetLspService() *lsp.LspService {
	return rs.lspService
}

func (rs *RepoService) GetConfig() *config.Config {
	return rs.config
}

func (rs *RepoService) GetFunctionDetails(repoName, relativePath, functionName string) (*model.GetFunctionDetailsResponse, error) {
	return nil, nil
}

func (rs *RepoService) GetFunctionDependencies(ctx context.Context, repoName, relativePath, functionName string, depth int) (*model.CallGraph, error) {
	return rs.lspService.GetFunctionDependencies(ctx, repoName, relativePath, functionName, depth)
}

func (rs *RepoService) GetFunctionHovers(ctx context.Context, repoName string, functions []model.FunctionDefinition) ([]string, error) {
	return rs.lspService.GetFunctionHovers(ctx, repoName, functions)
}

func (rs *RepoService) GetFunctionCallers(ctx context.Context, repoName, relativePath, functionName string, depth int) (*model.CallGraph, error) {
	return rs.lspService.GetFunctionCallers(ctx, repoName, relativePath, functionName, depth)
}
