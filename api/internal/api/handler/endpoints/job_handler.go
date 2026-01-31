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

type jobHandler struct {
	jobService *service.JobService
	jobMapper  mapper.JobMapper
	config     api.AppConfig
	logger     zerolog.Logger
}

func newJobHandler() *jobHandler {
	return &jobHandler{
		jobService: service.NewJobService(),
		jobMapper:  mapper.NewJobMapper(),
		config:     api.GetConfig(),
		logger:     api.Logger,
	}
}

func JobHandler(router *graceful.Graceful) {
	h := newJobHandler()

	routes := router.Group("/api/v1/jobs")
	routes.Use(middleware.AuthMiddleware(h.config))
	{
		routes.GET("", h.getAll)
		routes.GET("/:id", h.getByID)
		routes.POST("", h.create)
		routes.PUT("/:id", h.update)
		routes.DELETE("/:id", h.delete)

		// Sharing endpoints
		routes.POST("/:id/share", h.share)
		routes.DELETE("/:id/share", h.unshare)

		routes.POST("/:id/execute", h.execute)
		routes.POST("/:id/print-code", h.printCode)
		routes.POST("/:id/stop", h.stop)
	}
}

// getUserID extracts the user ID from the JWT context

// checkAccess verifies if the user can access the job and returns the role
func (slf *jobHandler) checkAccess(c *gin.Context, jobID, userID uint, requiredRole models.OwningJob) bool {
	canAccess, role, err := slf.jobService.CanUserAccess(jobID, userID)
	if err != nil {
		slf.logger.Error().Err(err).Uint("jobID", jobID).Msg("Failed to check job access")
		c.JSON(http.StatusNotFound, response.APIError{Message: "Job not found"})
		return false
	}

	if !canAccess {
		c.JSON(http.StatusForbidden, response.APIError{Message: "You don't have access to this job"})
		return false
	}

	// Check if role is sufficient
	if requiredRole == models.Owner && role != models.Owner {
		c.JSON(http.StatusForbidden, response.APIError{Message: "Only the owner can perform this action"})
		return false
	}
	if requiredRole == models.Editor && role != models.Owner && role != models.Editor {
		c.JSON(http.StatusForbidden, response.APIError{Message: "You don't have edit permissions for this job"})
		return false
	}

	return true
}

// getAll returns all jobs visible to the current user (optionally filtered by filePath)
func (slf *jobHandler) getAll(c *gin.Context) {
	userID, ok := pkg.GetUserID(c)
	if !ok {
		return
	}

	filePath := c.Query("filePath")

	var jobs []response.Job
	var entities []models.Job
	var err error

	if filePath != "" {
		entities, err = slf.jobService.FindByFilePathForUser(filePath, userID)
		if err != nil {
			slf.logger.Error().Err(err).Str("filePath", filePath).Msg("Failed to get jobs by file path")
			c.JSON(http.StatusInternalServerError, response.APIError{Message: "Failed to retrieve jobs"})
			return
		}
	} else {
		entities, err = slf.jobService.FindAllForUser(userID)
		if err != nil {
			slf.logger.Error().Err(err).Msg("Failed to get all jobs")
			c.JSON(http.StatusInternalServerError, response.APIError{Message: "Failed to retrieve jobs"})
			return
		}
	}

	jobs = slf.jobMapper.ToJobResponses(entities)
	c.JSON(http.StatusOK, jobs)
}

// getByID returns a single job with its nodes
func (slf *jobHandler) getByID(c *gin.Context) {
	userID, ok := pkg.GetUserID(c)
	if !ok {
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.APIError{Message: "Invalid ID"})
		return
	}

	// Check access before returning job
	if !slf.checkAccess(c, uint(id), userID, models.Viewer) {
		return
	}

	job, accessList, err := slf.jobService.FindByIDWithAccess(uint(id))
	if err != nil {
		slf.logger.Error().Err(err).Uint64("id", id).Msg("Failed to get job")
		c.JSON(http.StatusNotFound, response.APIError{Message: "Job not found"})
		return
	}

	c.JSON(http.StatusOK, mapper.ToJobResponseWithNodes(*job, accessList))
}

// create creates a new job with optional nodes
func (slf *jobHandler) create(c *gin.Context) {
	userID, ok := pkg.GetUserID(c)
	if !ok {
		return
	}

	var req request.CreateJob
	if err := pkg.ParseAndValidate(c, &req); err != nil {
		slf.logger.Error().Err(err).Msg("Failed to parse create job request")
		c.JSON(http.StatusBadRequest, response.APIError{Message: err.Error()})
		return
	}

	job := slf.jobMapper.CreateJob(req)
	job.CreatorID = userID
	//job.Nodes = req.Nodes // Nodes directly from request

	created, err := slf.jobService.Create(job)
	if err != nil {
		slf.logger.Error().Err(err).Msg("Failed to create job")
		c.JSON(http.StatusInternalServerError, response.APIError{Message: "Failed to create job"})
		return
	}

	// Share with specified users if any
	if len(req.SharedWith) > 0 {
		if err := slf.jobService.ShareJob(created.ID, req.SharedWith, models.Viewer); err != nil {
			slf.logger.Error().Err(err).Msg("Failed to share job with users")
			// Don't fail the create, just log the error
		}
	}

	// Fetch the job with access list for response
	job2, accessList, err := slf.jobService.FindByIDWithAccess(created.ID)
	if err != nil {
		// Fallback to created job without access list
		c.JSON(http.StatusCreated, mapper.ToJobResponseWithNodes(*created, nil))
		return
	}

	c.JSON(http.StatusCreated, mapper.ToJobResponseWithNodes(*job2, accessList))
}

// update updates an existing job and optionally its nodes
func (slf *jobHandler) update(c *gin.Context) {
	userID, ok := pkg.GetUserID(c)
	if !ok {
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.APIError{Message: "Invalid ID"})
		return
	}

	// Check edit access
	if !slf.checkAccess(c, uint(id), userID, models.Editor) {
		return
	}

	var req request.UpdateJob
	if err := pkg.ParseAndValidate(c, &req); err != nil {
		slf.logger.Error().Err(err).Msg("Failed to parse update job request")
		c.JSON(http.StatusBadRequest, response.APIError{Message: err.Error()})
		return
	}

	patch := slf.jobMapper.PatchJob(req)
	nodes := mapper.JobWithNodeToModel(req)

	var updated *models.Job
	if len(nodes) > 0 {
		// Update with nodes replacement
		updated, err = slf.jobService.UpdateWithNodes(uint(id), patch, nodes)
	} else {
		// Update only job fields
		updated, err = slf.jobService.Update(uint(id), patch)
	}

	if err != nil {
		slf.logger.Error().Err(err).Uint64("id", id).Msg("Failed to update job")
		c.JSON(http.StatusInternalServerError, response.APIError{Message: "Failed to update job"})
		return
	}

	// Update sharing if specified (only owner can change sharing)
	if req.SharedWith != nil {
		canAccess, role, _ := slf.jobService.CanUserAccess(uint(id), userID)
		if canAccess && role == models.Owner {
			if err := slf.jobService.UpdateJobSharing(uint(id), req.SharedWith, models.Viewer); err != nil {
				slf.logger.Error().Err(err).Msg("Failed to update job sharing")
			}
		}
	}

	// Fetch with access list for response
	job, accessList, err := slf.jobService.FindByIDWithAccess(updated.ID)
	if err != nil {
		c.JSON(http.StatusOK, mapper.ToJobResponseWithNodes(*updated, nil))
		return
	}

	c.JSON(http.StatusOK, mapper.ToJobResponseWithNodes(*job, accessList))
}

// delete removes a job (only owner can delete)
func (slf *jobHandler) delete(c *gin.Context) {
	userID, ok := pkg.GetUserID(c)
	if !ok {
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.APIError{Message: "Invalid ID"})
		return
	}

	// Only owner can delete
	if !slf.checkAccess(c, uint(id), userID, models.Owner) {
		return
	}

	if err := slf.jobService.Delete(uint(id)); err != nil {
		slf.logger.Error().Err(err).Uint64("id", id).Msg("Failed to delete job")
		c.JSON(http.StatusInternalServerError, response.APIError{Message: "Failed to delete job"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"id": id, "deleted": true})
}

// share adds users to a job's shared access list (only owner can share)
func (slf *jobHandler) share(c *gin.Context) {
	userID, ok := pkg.GetUserID(c)
	if !ok {
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.APIError{Message: "Invalid ID"})
		return
	}

	// Only owner can share
	if !slf.checkAccess(c, uint(id), userID, models.Owner) {
		return
	}

	var req request.ShareJob
	if err := pkg.ParseAndValidate(c, &req); err != nil {
		slf.logger.Error().Err(err).Msg("Failed to parse share job request")
		c.JSON(http.StatusBadRequest, response.APIError{Message: err.Error()})
		return
	}

	if err := slf.jobService.ShareJob(uint(id), req.UserIDs, req.Role); err != nil {
		slf.logger.Error().Err(err).Uint64("id", id).Msg("Failed to share job")
		c.JSON(http.StatusInternalServerError, response.APIError{Message: "Failed to share job"})
		return
	}

	// Return updated job with access list
	job, accessList, err := slf.jobService.FindByIDWithAccess(uint(id))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "Job shared successfully"})
		return
	}

	c.JSON(http.StatusOK, mapper.ToJobResponseWithNodes(*job, accessList))
}

// unshare removes users from a job's shared access list (only owner can unshare)
func (slf *jobHandler) unshare(c *gin.Context) {
	userID, ok := pkg.GetUserID(c)
	if !ok {
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.APIError{Message: "Invalid ID"})
		return
	}

	// Only owner can unshare
	if !slf.checkAccess(c, uint(id), userID, models.Owner) {
		return
	}

	var req request.ShareJob
	if err := pkg.ParseAndValidate(c, &req); err != nil {
		slf.logger.Error().Err(err).Msg("Failed to parse unshare job request")
		c.JSON(http.StatusBadRequest, response.APIError{Message: err.Error()})
		return
	}

	if err := slf.jobService.UnshareJob(uint(id), req.UserIDs); err != nil {
		slf.logger.Error().Err(err).Uint64("id", id).Msg("Failed to unshare job")
		c.JSON(http.StatusInternalServerError, response.APIError{Message: "Failed to unshare job"})
		return
	}

	// Return updated job with access list
	job, accessList, err := slf.jobService.FindByIDWithAccess(uint(id))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "Job unshared successfully"})
		return
	}

	c.JSON(http.StatusOK, mapper.ToJobResponseWithNodes(*job, accessList))
}

func (slf *jobHandler) execute(ctx *gin.Context) {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, response.APIError{Message: "Invalid ID"})
		return
	}

	go func() {
		if err := slf.jobService.Execute(uint(id)); err != nil {
			slf.logger.Error().Err(err).Uint64("id", id).Msg("Job execution failed")
		}
	}()

	ctx.JSON(http.StatusAccepted, gin.H{"message": "Job execution started", "jobId": id})
}

func (slf *jobHandler) stop(ctx *gin.Context) {}

func (slf *jobHandler) printCode(ctx *gin.Context) {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, response.APIError{Message: "Invalid ID"})
		return
	}
	source, steps, err := slf.jobService.PrintCode(uint(id))
	if err != nil {
		slf.logger.Error().Err(err).Uint64("id", id).Msg("Failed to print job code")
		ctx.JSON(http.StatusInternalServerError, response.APIError{Message: "Failed to print job code"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		source:  source,
		"steps": steps,
	})
}
