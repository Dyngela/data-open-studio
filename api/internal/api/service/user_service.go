package service

import (
	"api"
	"api/internal/api/handler/mapper"
	"api/internal/api/handler/request"
	"api/internal/api/handler/response"
	"api/internal/api/models"
	"api/internal/api/repo"
	"api/pkg"
	"errors"

	"github.com/rs/zerolog"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type UserService struct {
	userRepo   *repo.UserRepository
	config     api.AppConfig
	logger     zerolog.Logger
	userMapper mapper.UserMapper
}

func NewUserService() *UserService {
	return &UserService{
		userRepo: repo.NewUserRepository(),
		config:   api.GetConfig(),
		logger:   api.Logger,
	}
}

func (slf *UserService) Register(registerDTO request.RegisterDTO) (*response.AuthResponseDTO, error) {
	exists, err := slf.userRepo.ExistsByEmail(registerDTO.Email)
	if err != nil {
		slf.logger.Error().Err(err).Msg("Error checking if user exists")
		return nil, err
	}
	if exists {
		return nil, errors.New("user with this email already exists")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(registerDTO.Password), bcrypt.DefaultCost)
	if err != nil {
		slf.logger.Error().Err(err).Msg("Error hashing password")
		return nil, err
	}

	user := models.User{
		Email:    registerDTO.Email,
		Password: string(hashedPassword),
		Prenom:   registerDTO.Prenom,
		Nom:      registerDTO.Nom,
		Role:     "user",
		Actif:    true,
	}

	if err = slf.userRepo.Create(user); err != nil {
		slf.logger.Error().Err(err).Msg("Error creating user")
		return nil, err
	}

	token, err := pkg.GenerateToken(user.ID, user.Email, string(user.Role), slf.config.JWTConfig.Secret, slf.config.JWTConfig.Expiration)
	if err != nil {
		slf.logger.Error().Err(err).Msg("Error generating token")
		return nil, err
	}

	refreshToken, err := pkg.GenerateRefreshToken(user.ID, slf.config.JWTConfig.Secret, 30) // 30 days
	if err != nil {
		slf.logger.Error().Err(err).Msg("Error generating refresh token")
		return nil, err
	}

	user.RefreshToken = refreshToken
	if err = slf.userRepo.Update(user); err != nil {
		slf.logger.Error().Err(err).Msg("Error updating user with refresh token")
		return nil, err
	}

	slf.logger.Info().Uint("userId", user.ID).Msg("User registered successfully")
	return &response.AuthResponseDTO{
		Token:        token,
		RefreshToken: refreshToken,
		User:         slf.userMapper.EntityToUserResponse(user),
	}, nil
}

func (slf *UserService) Login(loginDTO request.LoginDTO) (*response.AuthResponseDTO, error) {
	user, err := slf.userRepo.FindByEmail(loginDTO.Email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("invalid email or password")
		}
		slf.logger.Error().Err(err).Msg("Error finding user by email")
		return nil, err
	}

	if !user.Actif {
		return nil, errors.New("account is inactive")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(loginDTO.Password)); err != nil {
		return nil, errors.New("invalid email or password")
	}

	token, err := pkg.GenerateToken(user.ID, user.Email, string(user.Role), slf.config.JWTConfig.Secret, slf.config.JWTConfig.Expiration)
	if err != nil {
		slf.logger.Error().Err(err).Msg("Error generating token")
		return nil, err
	}

	refreshToken, err := pkg.GenerateRefreshToken(user.ID, slf.config.JWTConfig.Secret, slf.config.JWTConfig.RefreshExpiration)
	if err != nil {
		slf.logger.Error().Err(err).Msg("Error generating refresh token")
		return nil, err
	}

	user.RefreshToken = refreshToken
	if err = slf.userRepo.Update(user); err != nil {
		slf.logger.Error().Err(err).Msg("Error updating user with refresh token")
		return nil, err
	}

	slf.logger.Info().Uint("userId", user.ID).Msg("User logged in successfully")

	return &response.AuthResponseDTO{
		Token:        token,
		RefreshToken: refreshToken,
		User:         slf.userMapper.EntityToUserResponse(user),
	}, nil
}

func (slf *UserService) GetByID(id uint) (response.UserResponseDTO, error) {
	user, err := slf.userRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.UserResponseDTO{}, errors.New("user not found")
		}
		slf.logger.Error().Err(err).Uint("userId", id).Msg("Error finding user by ID")
		return response.UserResponseDTO{}, err
	}

	return slf.userMapper.EntityToUserResponse(user), nil
}

func (slf *UserService) RefreshToken(refreshToken string) (response.AuthResponseDTO, error) {
	claims, err := pkg.ValidateRefreshToken(refreshToken, slf.config.JWTConfig.Secret)
	if err != nil {
		slf.logger.Error().Err(err).Msg("Invalid refresh token")
		return response.AuthResponseDTO{}, errors.New("invalid or expired refresh token")
	}

	user, err := slf.userRepo.FindByID(claims.UserID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.AuthResponseDTO{}, errors.New("user not found")
		}
		slf.logger.Error().Err(err).Uint("userId", claims.UserID).Msg("Error finding user by ID")
		return response.AuthResponseDTO{}, err
	}

	if !user.Actif {
		return response.AuthResponseDTO{}, errors.New("account is inactive")
	}

	if user.RefreshToken != refreshToken {
		slf.logger.Warn().Uint("userId", user.ID).Msg("Refresh token mismatch")
		return response.AuthResponseDTO{}, errors.New("invalid refresh token")
	}

	token, err := pkg.GenerateToken(user.ID, user.Email, string(user.Role), slf.config.JWTConfig.Secret, slf.config.JWTConfig.Expiration)
	if err != nil {
		slf.logger.Error().Err(err).Msg("Error generating token")
		return response.AuthResponseDTO{}, err
	}

	newRefreshToken, err := pkg.GenerateRefreshToken(user.ID, slf.config.JWTConfig.Secret, slf.config.JWTConfig.RefreshExpiration)
	if err != nil {
		slf.logger.Error().Err(err).Msg("Error generating refresh token")
		return response.AuthResponseDTO{}, err
	}

	user.RefreshToken = newRefreshToken
	if err = slf.userRepo.Update(user); err != nil {
		slf.logger.Error().Err(err).Msg("Error updating user with refresh token")
		return response.AuthResponseDTO{}, err
	}

	slf.logger.Info().Uint("userId", user.ID).Msg("Token refreshed successfully")
	return response.AuthResponseDTO{
		Token:        token,
		RefreshToken: newRefreshToken,
		User:         slf.userMapper.EntityToUserResponse(user),
	}, nil
}
