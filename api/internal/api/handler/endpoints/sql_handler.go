package endpoints

import (
	"api"
	"api/internal/api/handler/middleware"
	"api/internal/api/handler/request"
	"api/internal/api/handler/response"
	"api/internal/api/service"
	"api/pkg"
	"net/http"

	"github.com/gin-contrib/graceful"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

type sqlHandler struct {
	logger     zerolog.Logger
	config     api.AppConfig
	sqlService *service.SqlService
}

func newSqlHandler() *sqlHandler {
	return &sqlHandler{
		logger:     api.Logger,
		config:     api.GetConfig(),
		sqlService: service.NewSqlService(),
	}
}

func SqlHandler(router *graceful.Graceful) {
	h := newSqlHandler()

	routes := router.Group("/api/v1/sql")
	routes.Use(middleware.AuthMiddleware(h.config))
	{
		routes.POST("/guess-query", h.guessQuery)
		routes.POST("/optimize-query", h.optimizeQuery)
		routes.POST("/introspect/test-connection", h.testConnection)
		routes.POST("/introspect/tables", h.getTables)
		routes.POST("/introspect/columns", h.getColumns)
	}
}

func (slf *sqlHandler) guessQuery(c *gin.Context) {
	var req request.GuessQueryRequest
	if err := pkg.ParseAndValidate(c, &req); err != nil {
		slf.logger.Error().Err(err).Msg("Failed to parse guess query request")
		c.JSON(http.StatusBadRequest, response.APIError{Message: err.Error()})
		return
	}

	query, err := slf.sqlService.GuessQuery(req.Prompt, req.SchemaOptimizationNeeded, req.ConnectionID, req.PreviousMessages)
	if err != nil {
		slf.logger.Error().Err(err).Msg("Failed to guess query")
		c.JSON(http.StatusInternalServerError, response.APIError{Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, response.GuessQueryResponse{Query: query})
}

func (slf *sqlHandler) optimizeQuery(c *gin.Context) {
	var req request.OptimizeQueryRequest
	if err := pkg.ParseAndValidate(c, &req); err != nil {
		slf.logger.Error().Err(err).Msg("Failed to parse optimize query request")
		c.JSON(http.StatusBadRequest, response.APIError{Message: err.Error()})
		return
	}

	optimized, explanation, err := slf.sqlService.OptimizeQuery(req.Query, req.ConnectionID)
	if err != nil {
		slf.logger.Error().Err(err).Msg("Failed to optimize query")
		c.JSON(http.StatusInternalServerError, response.APIError{Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, response.OptimizeQueryResponse{
		OptimizedQuery: optimized,
		Explanation:    explanation,
	})
}

func (slf *sqlHandler) testConnection(c *gin.Context) {
	var req request.TestDatabaseConnection
	if err := pkg.ParseAndValidate(c, &req); err != nil {
		slf.logger.Error().Err(err).Msg("Failed to parse test connection request")
		c.JSON(http.StatusBadRequest, response.APIError{Message: err.Error()})
		return
	}

	result := service.TestDatabaseConnection(req.Connection)
	c.JSON(http.StatusOK, result)
}

func (slf *sqlHandler) getTables(c *gin.Context) {
	var req request.IntrospectDatabase
	if err := pkg.ParseAndValidate(c, &req); err != nil {
		slf.logger.Error().Err(err).Msg("Failed to parse introspect request")
		c.JSON(http.StatusBadRequest, response.APIError{Message: err.Error()})
		return
	}

	tables, err := service.IntrospectTables(req.MetadataDatabaseID, req.Connection)
	if err != nil {
		slf.logger.Error().Err(err).Msg("Failed to introspect tables")
		c.JSON(http.StatusBadRequest, response.APIError{Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, response.DatabaseIntrospection{Tables: tables})
}

func (slf *sqlHandler) getColumns(c *gin.Context) {
	var req struct {
		request.IntrospectDatabase
		TableName string `json:"tableName" validate:"required"`
	}
	if err := pkg.ParseAndValidate(c, &req); err != nil {
		slf.logger.Error().Err(err).Msg("Failed to parse introspect columns request")
		c.JSON(http.StatusBadRequest, response.APIError{Message: err.Error()})
		return
	}

	columns, err := service.IntrospectColumns(req.MetadataDatabaseID, req.Connection, req.TableName)
	if err != nil {
		slf.logger.Error().Err(err).Msg("Failed to introspect columns")
		c.JSON(http.StatusBadRequest, response.APIError{Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, response.DatabaseIntrospection{Columns: columns})
}
