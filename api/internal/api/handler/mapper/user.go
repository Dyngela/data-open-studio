package mapper

import (
	"api/internal/api/handler/request"
	"api/internal/api/handler/response"
	"api/internal/api/models"
)

//go:generate go run ../../../../tools/dtomapper -type=UserMapper
type UserMapper interface {
	// --update
	DtoToUpdate(req request.UpdateUser, vehicle *models.User)

	EntityToUserResponse(user models.User) response.UserResponseDTO
}
