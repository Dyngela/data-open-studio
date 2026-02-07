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

type triggerHandler struct {
	triggerService *service.TriggerService
	triggerMapper  mapper.TriggerMapper
	config         api.AppConfig
	logger         zerolog.Logger
}

func newTriggerHandler() *triggerHandler {
	return &triggerHandler{
		triggerService: service.NewTriggerService(),
		triggerMapper:  mapper.NewTriggerMapper(),
		config:         api.GetConfig(),
		logger:         api.Logger,
	}
}

func TriggerHandler(router *graceful.Graceful) {
	h := newTriggerHandler()

	routes := router.Group("/api/v1/triggers")
	routes.Use(middleware.AuthMiddleware(h.config))
	{
		// CRUD operations
		routes.GET("", h.getAll)
		routes.GET("/:id", h.getByID)
		routes.POST("", h.create)
		routes.PUT("/:id", h.update)
		routes.DELETE("/:id", h.delete)

		// Status operations
		routes.POST("/:id/activate", h.activate)
		routes.POST("/:id/pause", h.pause)

		// Rule operations
		routes.POST("/:id/rules", h.addRule)
		routes.PUT("/:id/rules/:ruleId", h.updateRule)
		routes.DELETE("/:id/rules/:ruleId", h.deleteRule)

		// Job linking operations
		routes.POST("/:id/jobs", h.linkJob)
		routes.DELETE("/:id/jobs/:jobId", h.unlinkJob)

		// Execution history
		routes.GET("/:id/executions", h.getExecutions)

		// Database introspection (for UI wizard)
		routes.POST("/introspect/test-connection", h.testConnection)
		routes.POST("/introspect/tables", h.getTables)
		routes.POST("/introspect/columns", h.getColumns)
	}
}

// checkAccess verifies if the user can access the trigger
func (slf *triggerHandler) checkAccess(c *gin.Context, triggerID, userID uint) bool {
	canAccess, err := slf.triggerService.CanUserAccess(triggerID, userID)
	if err != nil {
		slf.logger.Error().Err(err).Uint("triggerID", triggerID).Msg("Failed to check trigger access")
		c.JSON(http.StatusNotFound, response.APIError{Message: "Trigger not found"})
		return false
	}

	if !canAccess {
		c.JSON(http.StatusForbidden, response.APIError{Message: "You don't have access to this trigger"})
		return false
	}

	return true
}

// getAll returns all triggers for the current user
func (slf *triggerHandler) getAll(c *gin.Context) {
	userID, ok := pkg.GetUserID(c)
	if !ok {
		return
	}

	triggers, err := slf.triggerService.FindAllForUser(userID)
	if err != nil {
		slf.logger.Error().Err(err).Msg("Failed to get triggers")
		c.JSON(http.StatusInternalServerError, response.APIError{Message: "Failed to retrieve triggers"})
		return
	}

	c.JSON(http.StatusOK, slf.triggerMapper.ToTriggerResponses(triggers))
}

// getByID returns a single trigger with full details
func (slf *triggerHandler) getByID(c *gin.Context) {
	userID, ok := pkg.GetUserID(c)
	if !ok {
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.APIError{Message: "Invalid ID"})
		return
	}

	if !slf.checkAccess(c, uint(id), userID) {
		return
	}

	trigger, err := slf.triggerService.FindByID(uint(id))
	if err != nil {
		slf.logger.Error().Err(err).Uint64("id", id).Msg("Failed to get trigger")
		c.JSON(http.StatusNotFound, response.APIError{Message: "Trigger not found"})
		return
	}

	c.JSON(http.StatusOK, slf.triggerMapper.ToTriggerWithDetails(*trigger))
}

// create creates a new trigger
func (slf *triggerHandler) create(c *gin.Context) {
	userID, ok := pkg.GetUserID(c)
	if !ok {
		return
	}

	var req request.CreateTrigger
	if err := pkg.ParseAndValidate(c, &req); err != nil {
		slf.logger.Error().Err(err).Msg("Failed to parse create trigger request")
		c.JSON(http.StatusBadRequest, response.APIError{Message: err.Error()})
		return
	}

	trigger := slf.triggerMapper.CreateTrigger(req, userID)
	created, err := slf.triggerService.Create(trigger)
	if err != nil {
		slf.logger.Error().Err(err).Msg("Failed to create trigger")
		c.JSON(http.StatusBadRequest, response.APIError{Message: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, slf.triggerMapper.ToTriggerWithDetails(*created))
}

// update updates an existing trigger
func (slf *triggerHandler) update(c *gin.Context) {
	userID, ok := pkg.GetUserID(c)
	if !ok {
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.APIError{Message: "Invalid ID"})
		return
	}

	if !slf.checkAccess(c, uint(id), userID) {
		return
	}

	var req request.UpdateTrigger
	if err := pkg.ParseAndValidate(c, &req); err != nil {
		slf.logger.Error().Err(err).Msg("Failed to parse update trigger request")
		c.JSON(http.StatusBadRequest, response.APIError{Message: err.Error()})
		return
	}

	// Handle config update separately if provided
	if req.Config != nil {
		updated, err := slf.triggerService.UpdateConfig(uint(id), *req.Config)
		if err != nil {
			slf.logger.Error().Err(err).Uint64("id", id).Msg("Failed to update trigger config")
			c.JSON(http.StatusBadRequest, response.APIError{Message: err.Error()})
			return
		}
		c.JSON(http.StatusOK, slf.triggerMapper.ToTriggerWithDetails(*updated))
		return
	}

	patch := slf.triggerMapper.PatchTrigger(req)
	updated, err := slf.triggerService.Update(uint(id), patch)
	if err != nil {
		slf.logger.Error().Err(err).Uint64("id", id).Msg("Failed to update trigger")
		c.JSON(http.StatusInternalServerError, response.APIError{Message: "Failed to update trigger"})
		return
	}

	c.JSON(http.StatusOK, slf.triggerMapper.ToTriggerWithDetails(*updated))
}

// delete removes a trigger
func (slf *triggerHandler) delete(c *gin.Context) {
	userID, ok := pkg.GetUserID(c)
	if !ok {
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.APIError{Message: "Invalid ID"})
		return
	}

	if !slf.checkAccess(c, uint(id), userID) {
		return
	}

	if err := slf.triggerService.Delete(uint(id)); err != nil {
		slf.logger.Error().Err(err).Uint64("id", id).Msg("Failed to delete trigger")
		c.JSON(http.StatusInternalServerError, response.APIError{Message: "Failed to delete trigger"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"id": id, "deleted": true})
}

// activate starts a trigger (begins polling)
func (slf *triggerHandler) activate(c *gin.Context) {
	userID, ok := pkg.GetUserID(c)
	if !ok {
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.APIError{Message: "Invalid ID"})
		return
	}

	if !slf.checkAccess(c, uint(id), userID) {
		return
	}

	trigger, err := slf.triggerService.Activate(uint(id))
	if err != nil {
		slf.logger.Error().Err(err).Uint64("id", id).Msg("Failed to activate trigger")
		c.JSON(http.StatusBadRequest, response.APIError{Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, slf.triggerMapper.ToTriggerWithDetails(*trigger))
}

// pause stops a trigger (stops polling)
func (slf *triggerHandler) pause(c *gin.Context) {
	userID, ok := pkg.GetUserID(c)
	if !ok {
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.APIError{Message: "Invalid ID"})
		return
	}

	if !slf.checkAccess(c, uint(id), userID) {
		return
	}

	trigger, err := slf.triggerService.Pause(uint(id))
	if err != nil {
		slf.logger.Error().Err(err).Uint64("id", id).Msg("Failed to pause trigger")
		c.JSON(http.StatusInternalServerError, response.APIError{Message: "Failed to pause trigger"})
		return
	}

	c.JSON(http.StatusOK, slf.triggerMapper.ToTriggerWithDetails(*trigger))
}

// addRule adds a rule to a trigger
func (slf *triggerHandler) addRule(c *gin.Context) {
	userID, ok := pkg.GetUserID(c)
	if !ok {
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.APIError{Message: "Invalid ID"})
		return
	}

	if !slf.checkAccess(c, uint(id), userID) {
		return
	}

	var req request.CreateTriggerRule
	if err := pkg.ParseAndValidate(c, &req); err != nil {
		slf.logger.Error().Err(err).Msg("Failed to parse create rule request")
		c.JSON(http.StatusBadRequest, response.APIError{Message: err.Error()})
		return
	}

	rule := models.TriggerRule{
		TriggerID:  uint(id),
		Name:       req.Name,
		Conditions: req.Conditions,
	}

	created, err := slf.triggerService.AddRule(uint(id), rule)
	if err != nil {
		slf.logger.Error().Err(err).Uint64("id", id).Msg("Failed to add rule")
		c.JSON(http.StatusInternalServerError, response.APIError{Message: "Failed to add rule"})
		return
	}

	c.JSON(http.StatusCreated, slf.triggerMapper.ToTriggerRuleResponse(*created))
}

// updateRule updates an existing rule
func (slf *triggerHandler) updateRule(c *gin.Context) {
	userID, ok := pkg.GetUserID(c)
	if !ok {
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.APIError{Message: "Invalid trigger ID"})
		return
	}

	ruleID, err := strconv.ParseUint(c.Param("ruleId"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.APIError{Message: "Invalid rule ID"})
		return
	}

	if !slf.checkAccess(c, uint(id), userID) {
		return
	}

	var req request.UpdateTriggerRule
	if err := pkg.ParseAndValidate(c, &req); err != nil {
		slf.logger.Error().Err(err).Msg("Failed to parse update rule request")
		c.JSON(http.StatusBadRequest, response.APIError{Message: err.Error()})
		return
	}

	rule := models.TriggerRule{ID: uint(ruleID), TriggerID: uint(id)}
	if req.Name != nil {
		rule.Name = *req.Name
	}
	if req.Conditions != nil {
		rule.Conditions = *req.Conditions
	}

	updated, err := slf.triggerService.UpdateRule(rule)
	if err != nil {
		slf.logger.Error().Err(err).Uint64("ruleId", ruleID).Msg("Failed to update rule")
		c.JSON(http.StatusInternalServerError, response.APIError{Message: "Failed to update rule"})
		return
	}

	c.JSON(http.StatusOK, slf.triggerMapper.ToTriggerRuleResponse(*updated))
}

// deleteRule removes a rule from a trigger
func (slf *triggerHandler) deleteRule(c *gin.Context) {
	userID, ok := pkg.GetUserID(c)
	if !ok {
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.APIError{Message: "Invalid trigger ID"})
		return
	}

	ruleID, err := strconv.ParseUint(c.Param("ruleId"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.APIError{Message: "Invalid rule ID"})
		return
	}

	if !slf.checkAccess(c, uint(id), userID) {
		return
	}

	if err := slf.triggerService.DeleteRule(uint(ruleID)); err != nil {
		slf.logger.Error().Err(err).Uint64("ruleId", ruleID).Msg("Failed to delete rule")
		c.JSON(http.StatusInternalServerError, response.APIError{Message: "Failed to delete rule"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"id": ruleID, "deleted": true})
}

// linkJob links a job to a trigger
func (slf *triggerHandler) linkJob(c *gin.Context) {
	userID, ok := pkg.GetUserID(c)
	if !ok {
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.APIError{Message: "Invalid ID"})
		return
	}

	if !slf.checkAccess(c, uint(id), userID) {
		return
	}

	var req request.LinkJob
	if err := pkg.ParseAndValidate(c, &req); err != nil {
		slf.logger.Error().Err(err).Msg("Failed to parse link job request")
		c.JSON(http.StatusBadRequest, response.APIError{Message: err.Error()})
		return
	}

	link, err := slf.triggerService.LinkJob(uint(id), req.JobID, req.Priority, req.PassEventData)
	if err != nil {
		slf.logger.Error().Err(err).Uint64("id", id).Msg("Failed to link job")
		c.JSON(http.StatusBadRequest, response.APIError{Message: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":            link.ID,
		"triggerId":     link.TriggerID,
		"jobId":         link.JobID,
		"priority":      link.Priority,
		"active":        link.Active,
		"passEventData": link.PassEventData,
	})
}

// unlinkJob removes a job from a trigger
func (slf *triggerHandler) unlinkJob(c *gin.Context) {
	userID, ok := pkg.GetUserID(c)
	if !ok {
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.APIError{Message: "Invalid trigger ID"})
		return
	}

	jobID, err := strconv.ParseUint(c.Param("jobId"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.APIError{Message: "Invalid job ID"})
		return
	}

	if !slf.checkAccess(c, uint(id), userID) {
		return
	}

	if err := slf.triggerService.UnlinkJob(uint(id), uint(jobID)); err != nil {
		slf.logger.Error().Err(err).Uint64("id", id).Uint64("jobId", jobID).Msg("Failed to unlink job")
		c.JSON(http.StatusInternalServerError, response.APIError{Message: "Failed to unlink job"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"triggerId": id, "jobId": jobID, "unlinked": true})
}

// getExecutions returns recent execution history for a trigger
func (slf *triggerHandler) getExecutions(c *gin.Context) {
	userID, ok := pkg.GetUserID(c)
	if !ok {
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.APIError{Message: "Invalid ID"})
		return
	}

	if !slf.checkAccess(c, uint(id), userID) {
		return
	}

	limit := 20
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	executions, err := slf.triggerService.GetRecentExecutions(uint(id), limit)
	if err != nil {
		slf.logger.Error().Err(err).Uint64("id", id).Msg("Failed to get executions")
		c.JSON(http.StatusInternalServerError, response.APIError{Message: "Failed to retrieve executions"})
		return
	}

	c.JSON(http.StatusOK, slf.triggerMapper.ToTriggerExecutionResponses(executions))
}

// testConnection tests a database connection
func (slf *triggerHandler) testConnection(c *gin.Context) {
	var req request.TestDatabaseConnection
	if err := pkg.ParseAndValidate(c, &req); err != nil {
		slf.logger.Error().Err(err).Msg("Failed to parse test connection request")
		c.JSON(http.StatusBadRequest, response.APIError{Message: err.Error()})
		return
	}

	result := service.TestDatabaseConnection(req.Connection)
	c.JSON(http.StatusOK, result)
}

// getTables returns tables from a database for introspection
func (slf *triggerHandler) getTables(c *gin.Context) {
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

// getColumns returns columns for a specific table
func (slf *triggerHandler) getColumns(c *gin.Context) {
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
