package pkg

import (
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
