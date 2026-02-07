package endpoints

import (
	"api"
	"api/internal/api/handler/mapper"
	"api/internal/api/handler/middleware"
	"api/internal/api/handler/request"
	"api/internal/api/handler/response"
	"api/internal/api/models"
	"api/internal/api/service"
	"api/pkg"
	"net/http"
	"strconv"

	"github.com/gin-contrib/graceful"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

type dbMetadataHandler struct {
	metadataService *service.MetadataService
	logger          zerolog.Logger
	config          api.AppConfig
	metadataMapper  mapper.MetadataMapper
}

type sftpMetadataHandler struct {
	sftpService    *service.SftpMetadataService
	logger         zerolog.Logger
	config         api.AppConfig
	metadataMapper mapper.MetadataMapper
}

func newDbMetadataHandler() *dbMetadataHandler {
	return &dbMetadataHandler{
		metadataService: service.NewMetadataService(),
		logger:          api.Logger,
		config:          api.GetConfig(),
		metadataMapper:  mapper.NewMetadataMapper(),
	}
}

func newSftpMetadataHandler() *sftpMetadataHandler {
	return &sftpMetadataHandler{
		sftpService:    service.NewSftpMetadataService(),
		logger:         api.Logger,
		config:         api.GetConfig(),
		metadataMapper: mapper.NewMetadataMapper(),
	}
}

// DbMetadataHandler sets up DB metadata routes
func DbMetadataHandler(router *graceful.Graceful) {
	dbHandler := newDbMetadataHandler()
	sftpHandler := newSftpMetadataHandler()

	routes := router.Group("/api/v1/metadata")
	routes.Use(middleware.AuthMiddleware(dbHandler.config))

	db := routes.Group("/db")
	{
		db.GET("", dbHandler.getAll)
		db.GET("/:id", dbHandler.getByID)
		db.POST("", dbHandler.create)
		db.PUT("/:id", dbHandler.update)
		db.DELETE("/:id", dbHandler.delete)
		db.POST("/test-connection", dbHandler.testConnection)
	}

	sftp := routes.Group("/sftp")
	{
		sftp.GET("", sftpHandler.getAll)
		sftp.GET("/:id", sftpHandler.getByID)
		sftp.POST("", sftpHandler.create)
		sftp.PUT("/:id", sftpHandler.update)
		sftp.DELETE("/:id", sftpHandler.delete)
	}
}

// getAll returns all database metadata entries
func (slf *dbMetadataHandler) getAll(c *gin.Context) {
	metadataList, err := slf.metadataService.FindAll()
	if err != nil {
		slf.logger.Error().Err(err).Msg("Failed to get all db metadata")
		c.JSON(http.StatusInternalServerError, response.APIError{Message: "Failed to retrieve metadata"})
		return
	}

	c.JSON(http.StatusOK, slf.metadataMapper.ToMetadataResponses(metadataList))
}

// getByID returns a single database metadata entry by ID
func (slf *dbMetadataHandler) getByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.APIError{Message: "Invalid ID"})
		return
	}

	metadata, err := slf.metadataService.FindByID(uint(id))
	if err != nil {
		slf.logger.Error().Err(err).Uint64("id", id).Msg("Failed to get db metadata")
		c.JSON(http.StatusNotFound, response.APIError{Message: "Metadata not found"})
		return
	}

	c.JSON(http.StatusOK, slf.metadataMapper.ToMetadataResponse(*metadata))
}

// create creates a new database metadata entry
func (slf *dbMetadataHandler) create(c *gin.Context) {
	var req request.CreateMetadata
	if err := pkg.ParseAndValidate(c, &req); err != nil {
		slf.logger.Error().Err(err).Msg("Failed to parse create metadata request")
		c.JSON(http.StatusBadRequest, response.APIError{Message: err.Error()})
		return
	}

	metadata := slf.metadataMapper.CreateDbMetadata(req)
	created, err := slf.metadataService.Create(metadata)
	if err != nil {
		slf.logger.Error().Err(err).Msg("Failed to create db metadata")
		c.JSON(http.StatusInternalServerError, response.APIError{Message: "Failed to create metadata"})
		return
	}

	c.JSON(http.StatusCreated, slf.metadataMapper.ToMetadataResponse(*created))
}

// update updates an existing database metadata entry
func (slf *dbMetadataHandler) update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.APIError{Message: "Invalid ID"})
		return
	}

	var req request.UpdateMetadata
	if err := pkg.ParseAndValidate(c, &req); err != nil {
		slf.logger.Error().Err(err).Msg("Failed to parse update metadata request")
		c.JSON(http.StatusBadRequest, response.APIError{Message: err.Error()})
		return
	}

	patch := slf.metadataMapper.PatchDbMetadata(req)
	updated, err := slf.metadataService.Update(uint(id), patch)
	if err != nil {
		slf.logger.Error().Err(err).Uint64("id", id).Msg("Failed to update db metadata")
		c.JSON(http.StatusInternalServerError, response.APIError{Message: "Failed to update metadata"})
		return
	}

	c.JSON(http.StatusOK, slf.metadataMapper.ToMetadataResponse(*updated))
}

// delete removes a database metadata entry
func (slf *dbMetadataHandler) delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.APIError{Message: "Invalid ID"})
		return
	}

	if err := slf.metadataService.Delete(uint(id)); err != nil {
		slf.logger.Error().Err(err).Uint64("id", id).Msg("Failed to delete db metadata")
		c.JSON(http.StatusInternalServerError, response.APIError{Message: "Failed to delete metadata"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"id": id, "deleted": true})
}

// testConnection tests a database connection from metadata form values
func (slf *dbMetadataHandler) testConnection(c *gin.Context) {
	var req request.CreateMetadata
	if err := pkg.ParseAndValidate(c, &req); err != nil {
		slf.logger.Error().Err(err).Msg("Failed to parse test connection request")
		c.JSON(http.StatusBadRequest, response.APIError{Message: err.Error()})
		return
	}

	cfg := models.DBConnectionConfig{
		Type:     models.DBType(req.DbType),
		Host:     req.Host,
		Port:     req.Port,
		Database: req.DatabaseName,
		Username: req.User,
		Password: req.Password,
		SSLMode:  req.SSLMode,
	}

	result := service.TestDatabaseConnection(cfg)
	c.JSON(http.StatusOK, result)
}

// getAll returns all SFTP metadata entries
func (slf *sftpMetadataHandler) getAll(c *gin.Context) {
	metadataList, err := slf.sftpService.FindAll()
	if err != nil {
		slf.logger.Error().Err(err).Msg("Failed to get all sftp metadata")
		c.JSON(http.StatusInternalServerError, response.APIError{Message: "Failed to retrieve metadata"})
		return
	}

	c.JSON(http.StatusOK, slf.metadataMapper.ToSftpMetadataResponses(metadataList))
}

// getByID returns a single SFTP metadata entry by ID
func (slf *sftpMetadataHandler) getByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.APIError{Message: "Invalid ID"})
		return
	}

	metadata, err := slf.sftpService.FindByID(uint(id))
	if err != nil {
		slf.logger.Error().Err(err).Uint64("id", id).Msg("Failed to get sftp metadata")
		c.JSON(http.StatusNotFound, response.APIError{Message: "Metadata not found"})
		return
	}

	c.JSON(http.StatusOK, slf.metadataMapper.ToSftpMetadataResponse(*metadata))
}

// create creates a new SFTP metadata entry
func (slf *sftpMetadataHandler) create(c *gin.Context) {
	var req request.CreateSftpMetadata
	if err := pkg.ParseAndValidate(c, &req); err != nil {
		slf.logger.Error().Err(err).Msg("Failed to parse create sftp metadata request")
		c.JSON(http.StatusBadRequest, response.APIError{Message: err.Error()})
		return
	}

	metadata := slf.metadataMapper.CreateSftpMetadata(req)
	created, err := slf.sftpService.Create(metadata)
	if err != nil {
		slf.logger.Error().Err(err).Msg("Failed to create sftp metadata")
		c.JSON(http.StatusInternalServerError, response.APIError{Message: "Failed to create metadata"})
		return
	}

	c.JSON(http.StatusCreated, slf.metadataMapper.ToSftpMetadataResponse(*created))
}

// update updates an existing SFTP metadata entry
func (slf *sftpMetadataHandler) update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.APIError{Message: "Invalid ID"})
		return
	}

	var req request.UpdateSftpMetadata
	if err := pkg.ParseAndValidate(c, &req); err != nil {
		slf.logger.Error().Err(err).Msg("Failed to parse update sftp metadata request")
		c.JSON(http.StatusBadRequest, response.APIError{Message: err.Error()})
		return
	}

	patch := slf.metadataMapper.PatchSftpMetadata(req)
	updated, err := slf.sftpService.Update(uint(id), patch)
	if err != nil {
		slf.logger.Error().Err(err).Uint64("id", id).Msg("Failed to update sftp metadata")
		c.JSON(http.StatusInternalServerError, response.APIError{Message: "Failed to update metadata"})
		return
	}

	c.JSON(http.StatusOK, slf.metadataMapper.ToSftpMetadataResponse(*updated))
}

// delete removes a SFTP metadata entry
func (slf *sftpMetadataHandler) delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.APIError{Message: "Invalid ID"})
		return
	}

	if err := slf.sftpService.Delete(uint(id)); err != nil {
		slf.logger.Error().Err(err).Uint64("id", id).Msg("Failed to delete sftp metadata")
		c.JSON(http.StatusInternalServerError, response.APIError{Message: "Failed to delete metadata"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"id": id, "deleted": true})
}
