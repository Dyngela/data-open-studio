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
	"github.com/go-playground/validator/v10"
	"github.com/rs/zerolog"
)

type authHandler struct {
	userService *service.UserService
	validator   *validator.Validate
	logger      zerolog.Logger
	config      api.AppConfig
}

func newAuthHandler() *authHandler {
	return &authHandler{
		userService: service.NewUserService(),
		validator:   validator.New(),
		logger:      api.Logger,
		config:      api.GetConfig(),
	}
}

func AuthHandler(router *graceful.Graceful) {
	h := newAuthHandler()

	auth := router.Group("/api/v1/auth")
	{
		auth.POST("/register", h.register)
		auth.POST("/login", h.login)
		auth.POST("/refresh", h.refreshToken)
	}

	protected := router.Group("/api/v1")
	protected.Use(middleware.AuthMiddleware(h.config))
	{
		protected.GET("/me", h.getMe)
		protected.GET("/users/search", h.searchUsers)
	}

	admin := router.Group("/api/v1/admin")
	admin.Use(middleware.AuthMiddleware(h.config))
	admin.Use(middleware.RequireRole("admin"))
	{
		// Add admin-only routes here
	}
}

func (slf *authHandler) register(c *gin.Context) {
	var registerDTO request.RegisterDTO

	err := pkg.ParseAndValidate(c, &registerDTO)
	if err != nil {
		slf.logger.Error().Err(err).Msg("Error parsing and validating register DTO")
		c.JSON(http.StatusBadRequest, response.APIError{Message: err.Error()})
		return
	}

	// Call service
	authResponse, err := slf.userService.Register(registerDTO)
	if err != nil {
		slf.logger.Error().Err(err).Msg("Error registering user")
		c.JSON(http.StatusBadRequest, response.APIError{Message: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, authResponse)
}

func (slf *authHandler) login(c *gin.Context) {
	var loginDTO request.LoginDTO
	err := pkg.ParseAndValidate(c, &loginDTO)
	if err != nil {
		slf.logger.Error().Err(err).Msg("Error parsing and validating login DTO")
		c.JSON(http.StatusBadRequest, response.APIError{Message: err.Error()})
		return
	}

	// Call service
	authResponse, err := slf.userService.Login(loginDTO)
	if err != nil {
		slf.logger.Error().Err(err).Msg("Error logging in user")
		c.JSON(http.StatusUnauthorized, response.APIError{Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, authResponse)
}

func (slf *authHandler) getMe(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, response.APIError{Message: "User not authenticated"})
		return
	}

	// Call service
	user, err := slf.userService.GetByID(userID.(uint))
	if err != nil {
		slf.logger.Error().Err(err).Uint("userId", userID.(uint)).Msg("Error getting user")
		c.JSON(http.StatusNotFound, response.APIError{Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, user)
}

func (slf *authHandler) searchUsers(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, response.APIError{Message: "Query parameter 'q' is required"})
		return
	}

	users, err := slf.userService.SearchUsers(query)
	if err != nil {
		slf.logger.Error().Err(err).Str("query", query).Msg("Error searching users")
		c.JSON(http.StatusInternalServerError, response.APIError{Message: "Failed to search users"})
		return
	}

	c.JSON(http.StatusOK, users)
}

func (slf *authHandler) refreshToken(c *gin.Context) {
	var refreshDTO request.RefreshTokenDTO
	err := pkg.ParseAndValidate(c, &refreshDTO)
	if err != nil {
		slf.logger.Error().Err(err).Msg("Error parsing and validating refresh token DTO")
		c.JSON(http.StatusBadRequest, response.APIError{Message: err.Error()})
		return
	}

	// Call service
	authResponse, err := slf.userService.RefreshToken(refreshDTO.RefreshToken)
	if err != nil {
		slf.logger.Error().Err(err).Msg("Error refreshing token")
		c.JSON(http.StatusUnauthorized, response.APIError{Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, authResponse)
}
