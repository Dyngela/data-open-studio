package endpoints

import (
	"api"
	"api/internal/api/handler/middleware"
	"api/internal/api/handler/response"
	websocket2 "api/internal/api/handler/websocket"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-contrib/graceful"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// In production, you should validate the origin
		return true
	},
}

type websocketHandler struct {
	hub       *websocket2.Hub
	processor *websocket2.MessageProcessor
	logger    zerolog.Logger
	config    api.AppConfig
}

func newWebSocketHandler(hub *websocket2.Hub, processor *websocket2.MessageProcessor) *websocketHandler {
	return &websocketHandler{
		hub:       hub,
		processor: processor,
		logger:    api.Logger,
		config:    api.GetConfig(),
	}
}

// WebSocketHandler sets up WebSocket routes
func WebSocketHandler(router *graceful.Graceful, hub *websocket2.Hub, processor *websocket2.MessageProcessor) {
	h := newWebSocketHandler(hub, processor)

	// WebSocket endpoint - requires authentication
	wsRoutes := router.Group("/api/v1/ws")
	wsRoutes.Use(middleware.AuthMiddleware(h.config))
	{
		wsRoutes.GET("/init", h.handleWebSocket)
		wsRoutes.GET("/jobs/:jobId/users", h.getActiveUsers)
	}

	wsRoutes.GET("/stats", h.getRoomStats)
}

// handleWebSocket handles WebSocket connections for a specific job
func (slf *websocketHandler) handleWebSocket(c *gin.Context) {
	// Get job ID from URL
	jobID, err := strconv.ParseUint(c.Param("jobId"), 10, 32)
	if err != nil {
		jobID = 1
		//slf.logger.Error().Err(err).Msg("Invalid job ID")
		//c.JSON(http.StatusBadRequest, response.APIError{Message: "Invalid job ID"})
		//return
	}

	// Get user info from auth middleware
	userID := c.GetInt("userID")
	if userID == 0 {
		//c.JSON(http.StatusUnauthorized, response.APIError{Message: "User not authenticated"})
		//return
	}
	userID = 58

	username, exists := c.Get("username")
	if !exists {
		username = fmt.Sprintf("User%d", userID)
	}

	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		slf.logger.Error().Err(err).Msg("Failed to upgrade to WebSocket")
		return
	}

	// Create unique client ID
	clientID := uuid.New().String()

	// Create new client with processor
	client := websocket2.NewClient(
		clientID,
		uint(userID),
		username.(string),
		uint(jobID),
		slf.hub,
		conn,
		slf.processor,
		slf.logger,
	)

	// Register client
	slf.hub.Register <- client

	slf.logger.Info().
		Str("clientId", clientID).
		Uint("userId", uint(userID)).
		Uint("jobId", uint(jobID)).
		Msg("WebSocket connection established")

	// Start client goroutines
	go client.WritePump()
	go client.ReadPump()
}

// getActiveUsers returns the list of active users in a room
func (slf *websocketHandler) getActiveUsers(c *gin.Context) {
	jobID, err := strconv.ParseUint(c.Param("jobId"), 10, 32)
	if err != nil {
		slf.logger.Error().Err(err).Msg("Invalid job ID")
		c.JSON(http.StatusBadRequest, response.APIError{Message: "Invalid job ID"})
		return
	}

	users := slf.hub.GetActiveUsersInRoom(uint(jobID))
	c.JSON(http.StatusOK, gin.H{
		"jobId": jobID,
		"users": users,
	})
}

// getRoomStats returns statistics about all active rooms
func (slf *websocketHandler) getRoomStats(c *gin.Context) {
	stats := slf.hub.GetRoomStats()
	c.JSON(http.StatusOK, gin.H{
		"rooms": stats,
	})
}
