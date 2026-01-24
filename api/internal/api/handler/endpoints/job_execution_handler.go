package endpoints

import (
	"api"

	"github.com/gin-contrib/graceful"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

// ProgressRequest represents a progress update from a running job executable
type ProgressRequest struct {
	NodeID        int    `json:"nodeId"`
	NodeName      string `json:"nodeName"`
	Status        string `json:"status"` // running, completed, error
	RowsProcessed int64  `json:"rowsProcessed"`
	Message       string `json:"message,omitempty"`
}

type jobExecutionHandler struct {
	logger zerolog.Logger
	config api.AppConfig
}

func newJobExecutionHandler() *jobExecutionHandler {
	return &jobExecutionHandler{
		logger: api.Logger,
		config: api.GetConfig(),
	}
}

// JobExecutionHandler sets up job execution routes
func JobExecutionHandler(router *graceful.Graceful) {
	h := newJobExecutionHandler()

	// Internal API for job executables - uses API key auth instead of JWT
	internalRoutes := router.Group("/api/internal/jobs")
	{
		internalRoutes.POST("/:jobId/progress", h.handleProgress)
	}
}

// handleProgress receives progress updates from job executables and broadcasts to websocket
func (h *jobExecutionHandler) handleProgress(c *gin.Context) {
	//// Get job ID from URL
	//jobID, err := strconv.ParseUint(c.Param("jobId"), 10, 32)
	//if err != nil {
	//	h.logger.Error().Err(err).Msg("Invalid job ID")
	//	c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid job ID"})
	//	return
	//}
	//
	//// Parse progress request
	//var req ProgressRequest
	//if err := c.ShouldBindJSON(&req); err != nil {
	//	h.logger.Error().Err(err).Msg("Failed to parse progress request")
	//	c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
	//	return
	//}
	//
	//// Create websocket message
	//progress := websocket2.JobProgress{
	//	NodeID:        req.NodeID,
	//	NodeName:      req.NodeName,
	//	Status:        req.Status,
	//	RowsProcessed: req.RowsProcessed,
	//	Message:       req.Message,
	//}
	//
	//msg := websocket2.NewJobProgressMessage(uint(jobID), progress)
	//
	//// Broadcast to websocket hub
	//h.hub.Broadcast <- msg
	//
	//h.logger.Debug().
	//	Uint64("jobId", jobID).
	//	Int("nodeId", req.NodeID).
	//	Str("status", req.Status).
	//	Int64("rows", req.RowsProcessed).
	//	Msg("Progress update received and broadcasted")
	//
	//c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
