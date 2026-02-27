package endpoints

import (
	"api"
	"api/internal/api/handler/middleware"
	"bytes"
	"net/http"
	"os/exec"
	"time"

	"github.com/gin-contrib/graceful"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

type CodeRequest struct {
	Code string `json:"code"`
}

type CodeResponse struct {
	Output string `json:"output"`
	Error  bool   `json:"error"`
}

type scriptHandler struct {
	logger zerolog.Logger
	config api.AppConfig
}

func newScriptHandler() *scriptHandler {
	return &scriptHandler{
		logger: api.Logger,
		config: api.GetConfig(),
	}
}

func ScriptHandler(router *graceful.Graceful) {
	h := newScriptHandler()

	routes := router.Group("/api/v1/script")
	routes.Use(middleware.AuthMiddleware(h.config))
	{
		routes.POST("/execute", h.executeScript)
	}
}

func (sh *scriptHandler) executeScript(c *gin.Context) {
	var req CodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// Timeout de 10 secondes
	cmd := exec.Command("python3", "-c", req.Code)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Canal pour gérer le timeout
	done := make(chan error, 1)
	go func() { done <- cmd.Run() }()

	select {
	case <-time.After(10 * time.Second):
		cmd.Process.Kill()
		c.JSON(http.StatusOK, CodeResponse{
			Output: "Erreur : timeout dépassé (10s)",
			Error:  true,
		})
	case err := <-done:
		if err != nil {
			c.JSON(http.StatusOK, CodeResponse{
				Output: stderr.String(),
				Error:  true,
			})
		} else {
			c.JSON(http.StatusOK, CodeResponse{
				Output: stdout.String(),
				Error:  false,
			})
		}
	}
}
