package endpoints

import (
	"api"
	"api/internal/api/handler/mapper"
	"api/internal/api/handler/middleware"
	"api/internal/api/handler/request"
	"api/internal/api/handler/response"
	"api/internal/api/service"
	"api/pkg"
	"net/http"
	"strconv"

	"github.com/gin-contrib/graceful"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

type datasetHandler struct {
	datasetService *service.DatasetService
	datasetMapper  mapper.DatasetMapper
	config         api.AppConfig
	logger         zerolog.Logger
}

func newDatasetHandler() *datasetHandler {
	return &datasetHandler{
		datasetService: service.NewDatasetService(),
		datasetMapper:  mapper.NewDatasetMapper(),
		config:         api.GetConfig(),
		logger:         api.Logger,
	}
}

func DatasetHandler(router *graceful.Graceful) {
	h := newDatasetHandler()

	routes := router.Group("/api/v1/datasets")
	routes.Use(middleware.AuthMiddleware(h.config))
	{
		routes.GET("", h.getAll)
		routes.GET("/:id", h.getByID)
		routes.POST("", h.create)
		routes.PUT("/:id", h.update)
		routes.DELETE("/:id", h.delete)
		routes.POST("/:id/refresh", h.refresh)
		routes.POST("/:id/preview", h.preview)
		routes.POST("/:id/query", h.query)
	}
}

func (h *datasetHandler) checkAccess(c *gin.Context, datasetID, userID uint) bool {
	canAccess, err := h.datasetService.CanUserAccess(datasetID, userID)
	if err != nil {
		h.logger.Error().Err(err).Uint("datasetID", datasetID).Msg("Failed to check dataset access")
		c.JSON(http.StatusNotFound, response.APIError{Message: "Dataset not found"})
		return false
	}
	if !canAccess {
		c.JSON(http.StatusForbidden, response.APIError{Message: "You don't have access to this dataset"})
		return false
	}
	return true
}

func (h *datasetHandler) getAll(c *gin.Context) {
	userID, ok := pkg.GetUserID(c)
	if !ok {
		return
	}

	datasets, err := h.datasetService.FindAllForUser(userID)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to get datasets")
		c.JSON(http.StatusInternalServerError, response.APIError{Message: "Failed to retrieve datasets"})
		return
	}

	c.JSON(http.StatusOK, h.datasetMapper.ToDatasetSummaries(datasets))
}

func (h *datasetHandler) getByID(c *gin.Context) {
	userID, ok := pkg.GetUserID(c)
	if !ok {
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.APIError{Message: "Invalid ID"})
		return
	}

	if !h.checkAccess(c, uint(id), userID) {
		return
	}

	dataset, err := h.datasetService.FindByID(uint(id))
	if err != nil {
		h.logger.Error().Err(err).Uint64("id", id).Msg("Failed to get dataset")
		c.JSON(http.StatusNotFound, response.APIError{Message: "Dataset not found"})
		return
	}

	c.JSON(http.StatusOK, h.datasetMapper.ToDatasetWithDetails(*dataset))
}

func (h *datasetHandler) create(c *gin.Context) {
	userID, ok := pkg.GetUserID(c)
	if !ok {
		return
	}

	var req request.CreateDataset
	if err := pkg.ParseAndValidate(c, &req); err != nil {
		h.logger.Error().Err(err).Msg("Failed to parse create dataset request")
		c.JSON(http.StatusBadRequest, response.APIError{Message: err.Error()})
		return
	}

	dataset := h.datasetMapper.ToDataset(req, userID)
	created, err := h.datasetService.Create(dataset)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to create dataset")
		c.JSON(http.StatusBadRequest, response.APIError{Message: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, h.datasetMapper.ToDatasetWithDetails(*created))
}

func (h *datasetHandler) update(c *gin.Context) {
	userID, ok := pkg.GetUserID(c)
	if !ok {
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.APIError{Message: "Invalid ID"})
		return
	}

	if !h.checkAccess(c, uint(id), userID) {
		return
	}

	var req request.UpdateDataset
	if err := pkg.ParseAndValidate(c, &req); err != nil {
		h.logger.Error().Err(err).Msg("Failed to parse update dataset request")
		c.JSON(http.StatusBadRequest, response.APIError{Message: err.Error()})
		return
	}

	patch := h.datasetMapper.ToDatasetPatch(req)
	updated, err := h.datasetService.Update(uint(id), patch)
	if err != nil {
		h.logger.Error().Err(err).Uint64("id", id).Msg("Failed to update dataset")
		c.JSON(http.StatusBadRequest, response.APIError{Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, h.datasetMapper.ToDatasetWithDetails(*updated))
}

func (h *datasetHandler) delete(c *gin.Context) {
	userID, ok := pkg.GetUserID(c)
	if !ok {
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.APIError{Message: "Invalid ID"})
		return
	}

	if !h.checkAccess(c, uint(id), userID) {
		return
	}

	if err := h.datasetService.Delete(uint(id)); err != nil {
		h.logger.Error().Err(err).Uint64("id", id).Msg("Failed to delete dataset")
		c.JSON(http.StatusInternalServerError, response.APIError{Message: "Failed to delete dataset"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"id": id, "deleted": true})
}

func (h *datasetHandler) refresh(c *gin.Context) {
	userID, ok := pkg.GetUserID(c)
	if !ok {
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.APIError{Message: "Invalid ID"})
		return
	}

	if !h.checkAccess(c, uint(id), userID) {
		return
	}

	dataset, err := h.datasetService.Refresh(uint(id))
	if err != nil {
		h.logger.Warn().Err(err).Uint64("id", id).Msg("Schema refresh returned error")
		// Still return the dataset even if refresh failed (status will be "error")
		if dataset != nil {
			c.JSON(http.StatusOK, h.datasetMapper.ToDatasetWithDetails(*dataset))
			return
		}
		c.JSON(http.StatusBadRequest, response.APIError{Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, h.datasetMapper.ToDatasetWithDetails(*dataset))
}

func (h *datasetHandler) preview(c *gin.Context) {
	userID, ok := pkg.GetUserID(c)
	if !ok {
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.APIError{Message: "Invalid ID"})
		return
	}

	if !h.checkAccess(c, uint(id), userID) {
		return
	}

	var req request.DatasetPreview
	// Ignore parse errors - all fields are optional
	_ = pkg.ParseAndValidate(c, &req)

	result, err := h.datasetService.Preview(uint(id), req.Limit)
	if err != nil {
		h.logger.Error().Err(err).Uint64("id", id).Msg("Failed to preview dataset")
		c.JSON(http.StatusBadRequest, response.APIError{Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *datasetHandler) query(c *gin.Context) {
	userID, ok := pkg.GetUserID(c)
	if !ok {
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.APIError{Message: "Invalid ID"})
		return
	}

	if !h.checkAccess(c, uint(id), userID) {
		return
	}

	var req request.DatasetQuery
	_ = pkg.ParseAndValidate(c, &req)

	filters := h.datasetMapper.ToQueryFilters(req.Filters)
	result, err := h.datasetService.Query(uint(id), filters, req.Limit)
	if err != nil {
		h.logger.Error().Err(err).Uint64("id", id).Msg("Failed to query dataset")
		c.JSON(http.StatusBadRequest, response.APIError{Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}
