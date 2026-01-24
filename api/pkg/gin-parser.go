package pkg

import (
	"api/internal/api/handler/response"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

var validate = validator.New()

func ParseAndValidate(c *gin.Context, dto interface{}) error {
	if err := c.ShouldBindJSON(dto); err != nil {
		return err
	}
	return validate.Struct(dto)
}

func GetUserID(c *gin.Context) (uint, bool) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, response.APIError{Message: "User not authenticated"})
		return 0, false
	}
	return userID.(uint), true
}
