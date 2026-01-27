package endpoints

import (
	"api"
	"api/internal/api/handler/mapper"
	"api/internal/api/handler/middleware"
	"api/internal/api/handler/request"
	"api/internal/api/handler/response"
	"api/pkg"
	"net/http"

	"github.com/gin-contrib/graceful"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

type dbNodeHandler struct {
	logger     zerolog.Logger
	config     api.AppConfig
	nodeMapper mapper.NodeMapper
}

func newDbNodeHandler() *dbNodeHandler {
	return &dbNodeHandler{
		logger:     api.Logger,
		config:     api.GetConfig(),
		nodeMapper: mapper.NewNodeMapper(),
	}
}

// DbNodeHandler sets up DB node routes
func DbNodeHandler(router *graceful.Graceful) {
	h := newDbNodeHandler()

	routes := router.Group("/api/v1/db-node")
	routes.Use(middleware.AuthMiddleware(h.config))
	{
		routes.POST("/guess-schema", h.guessSchema)
	}
}

// guessSchema introspects a database query and returns the schema/data model
func (slf *dbNodeHandler) guessSchema(c *gin.Context) {
	var req request.GuessSchemaRequest
	if err := pkg.ParseAndValidate(c, &req); err != nil {
		slf.logger.Error().Err(err).Msg("Failed to parse guess schema request")
		c.JSON(http.StatusBadRequest, response.APIError{Message: err.Error()})
		return
	}

	slf.logger.Debug().
		Str("query", req.Query).
		Str("dbType", string(req.DbType)).
		Str("host", req.Host).
		Msg("Guessing schema for query")

	// build the DB input config
	node := slf.nodeMapper.GuessSchemaRequestToDBInputConfig(req)

	// execute schema introspection
	if err := node.FillDataModels(); err != nil {
		slf.logger.Error().Err(err).Msg("Failed to guess data model")
		c.JSON(http.StatusInternalServerError, response.APIError{Message: "Failed to guess schema: " + err.Error()})
		return
	}

	slf.logger.Info().
		Int("columnsFound", len(node.DataModels)).
		Msg("Successfully guessed schema")

	c.JSON(http.StatusOK, response.GuessSchemaResponse{
		NodeID:     req.NodeID,
		DataModels: node.DataModels,
	})
}
