package handler

import (
	"bytes"
	"io"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/armchr/codeapi/internal/config"
	"github.com/armchr/codeapi/internal/controller"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// responseWriter wraps gin.ResponseWriter to capture the response body
type responseWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *responseWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func SetupRouter(repoController *controller.RepoController, codeAPIController *controller.CodeAPIController, summaryController *controller.SummaryController, cfg *config.Config, logger *zap.Logger) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()
	router.Use(CustomRecoveryMiddleware(logger))
	router.Use(LoggerMiddleware(cfg.App.DebugHTTP, logger))

	v1 := router.Group("/api/v1")
	{
		v1.POST("/buildIndex", repoController.BuildIndex)
		//v1.POST("/getFunctionsInFile", repoController.GetFunctionsInFile)
		//v1.POST("/getFunctionDetails", repoController.GetFunctionDetails)
		v1.POST("/functionDependencies", repoController.GetFunctionDependencies)
		v1.POST("/processDirectory", repoController.ProcessDirectory)
		v1.POST("/searchSimilarCode", repoController.SearchSimilarCode)

		// Semantic signature search endpoint
		v1.POST("/searchMethodsBySignature", repoController.SearchMethodsBySignature)

		// Index building endpoints
		v1.POST("/indexFile", repoController.IndexFile)

		v1.GET("/health", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"status": "healthy",
			})
		})
	}

	// CodeAPI routes
	if codeAPIController != nil {
		codeAPI := router.Group("/codeapi/v1")
		{
			// Reader endpoints
			codeAPI.GET("/repos", codeAPIController.ListRepos)
			codeAPI.POST("/files", codeAPIController.ListFiles)
			codeAPI.POST("/classes", codeAPIController.ListClasses)
			codeAPI.POST("/methods", codeAPIController.ListMethods)
			codeAPI.POST("/functions", codeAPIController.ListFunctions)
			codeAPI.POST("/classes/find", codeAPIController.FindClasses)
			codeAPI.POST("/methods/find", codeAPIController.FindMethods)
			codeAPI.POST("/class", codeAPIController.GetClass)
			codeAPI.POST("/method", codeAPIController.GetMethod)
			codeAPI.POST("/class/methods", codeAPIController.GetClassMethods)
			codeAPI.POST("/class/fields", codeAPIController.GetClassFields)

			// Analyzer endpoints
			codeAPI.POST("/callgraph", codeAPIController.GetCallGraph)
			codeAPI.POST("/callers", codeAPIController.GetCallers)
			codeAPI.POST("/callees", codeAPIController.GetCallees)
			codeAPI.POST("/data/dependents", codeAPIController.GetDataDependents)
			codeAPI.POST("/data/sources", codeAPIController.GetDataSources)
			codeAPI.POST("/impact", codeAPIController.GetImpact)
			codeAPI.POST("/inheritance", codeAPIController.GetInheritanceTree)
			codeAPI.POST("/field/accessors", codeAPIController.GetFieldAccessors)

			// Raw Cypher endpoints
			codeAPI.POST("/cypher", codeAPIController.ExecuteCypher)
			codeAPI.POST("/cypher/write", codeAPIController.ExecuteCypherWrite)

			// Code snippet endpoint
			codeAPI.POST("/snippet", codeAPIController.GetCodeSnippet)

			// Health check
			codeAPI.GET("/health", func(c *gin.Context) {
				c.JSON(200, gin.H{"status": "healthy"})
			})
		}
	}

	// Summary query routes
	if summaryController != nil {
		summaryAPI := router.Group("/codeapi/v1/summaries")
		{
			// Get all summaries for a file (optionally filtered by entity_type)
			summaryAPI.POST("/file", summaryController.GetFileSummaries)

			// Get file-level summary
			summaryAPI.POST("/file/summary", summaryController.GetFileSummary)

			// Get a specific function or class summary
			summaryAPI.POST("/entity", summaryController.GetEntitySummary)

			// Get summary statistics for a repository
			summaryAPI.POST("/stats", summaryController.GetSummaryStats)
		}
	}

	return router
}

func LoggerMiddleware(debugHTTP bool, logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		var requestBody []byte
		var responseBody *bytes.Buffer

		if debugHTTP {
			// Read request body for debug logging
			if c.Request.Body != nil {
				requestBody, _ = io.ReadAll(c.Request.Body)
				// Restore the body for the handler
				c.Request.Body = io.NopCloser(bytes.NewBuffer(requestBody))
			}

			// Log request with body
			requestFields := []zap.Field{
				zap.String("method", c.Request.Method),
				zap.String("path", c.Request.URL.Path),
				zap.String("client_ip", c.ClientIP()),
			}
			if len(requestBody) > 0 && len(requestBody) <= 10000 {
				requestFields = append(requestFields, zap.String("request_body", string(requestBody)))
			} else if len(requestBody) > 10000 {
				requestFields = append(requestFields, zap.String("request_body", string(requestBody[:10000])+"... (truncated)"))
			}
			logger.Info("HTTP Request", requestFields...)

			// Wrap response writer to capture response body
			responseBody = &bytes.Buffer{}
			writer := &responseWriter{
				ResponseWriter: c.Writer,
				body:           responseBody,
			}
			c.Writer = writer
		} else {
			// Basic request logging without body
			logger.Info("HTTP Request",
				zap.String("method", c.Request.Method),
				zap.String("path", c.Request.URL.Path),
				zap.String("client_ip", c.ClientIP()),
			)
		}

		// Process request
		c.Next()

		// Calculate duration
		duration := time.Since(start)

		if debugHTTP {
			// Log response with body
			responseFields := []zap.Field{
				zap.String("method", c.Request.Method),
				zap.String("path", c.Request.URL.Path),
				zap.Int("status", c.Writer.Status()),
				zap.Duration("duration", duration),
			}
			if responseBody != nil && responseBody.Len() > 0 && responseBody.Len() <= 10000 {
				responseFields = append(responseFields, zap.String("response_body", responseBody.String()))
			} else if responseBody != nil && responseBody.Len() > 10000 {
				responseFields = append(responseFields, zap.String("response_body", responseBody.String()[:10000]+"... (truncated)"))
			}
			logger.Info("HTTP Response", responseFields...)
		} else {
			// Basic response logging without body
			logger.Info("HTTP Response",
				zap.String("method", c.Request.Method),
				zap.String("path", c.Request.URL.Path),
				zap.Int("status", c.Writer.Status()),
				zap.Duration("duration", duration),
			)
		}
	}
}

func CustomRecoveryMiddleware(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				logger.Error("Panic recovered",
					zap.Any("error", err),
					zap.String("stack", string(debug.Stack())),
					zap.String("path", c.Request.URL.Path),
					zap.String("method", c.Request.Method),
				)
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "Internal server error",
				})
				c.Abort()
			}
		}()
		c.Next()
	}
}
