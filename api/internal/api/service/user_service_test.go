package service

import (
	"api"
	"api/internal/api/handler/request"
	"api/internal/api/models"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupUserTestDB(t *testing.T) {
	api.InitConfig("../../../.env.test")

	err := api.DB.AutoMigrate(&models.User{})
	require.NoError(t, err, "Failed to migrate user table")
}

func cleanupUser(t *testing.T, id uint) {
	if id > 0 {
		api.DB.Unscoped().Delete(&models.User{}, id)
	}
}

func uniqueEmail() string {
	return fmt.Sprintf("test-%d@example.com", time.Now().UnixNano())
}

func TestUser_Register(t *testing.T) {
	setupUserTestDB(t)

	service := NewUserService()
	email := uniqueEmail()

	dto := request.RegisterDTO{
		Email:    email,
		Password: "testpassword123",
		Prenom:   "Jean",
		Nom:      "Dupont",
	}

	result, err := service.Register(dto)
	require.NoError(t, err, "Failed to register user")
	require.NotNil(t, result)
	defer cleanupUser(t, result.User.ID)

	assert.NotEmpty(t, result.Token)
	assert.NotEmpty(t, result.RefreshToken)
	assert.Equal(t, email, result.User.Email)
	assert.Equal(t, "Jean", result.User.Prenom)
	assert.Equal(t, "Dupont", result.User.Nom)
	assert.True(t, result.User.Actif)
}

func TestUser_Register_DuplicateEmail(t *testing.T) {
	setupUserTestDB(t)

	service := NewUserService()
	email := uniqueEmail()

	dto := request.RegisterDTO{
		Email:    email,
		Password: "testpassword123",
		Prenom:   "Jean",
		Nom:      "Dupont",
	}

	result, err := service.Register(dto)
	require.NoError(t, err)
	defer cleanupUser(t, result.User.ID)

	// Try to register again with the same email
	_, err = service.Register(dto)
	require.Error(t, err, "Should fail on duplicate email")
	assert.Contains(t, err.Error(), "already exists")
}

func TestUser_Login(t *testing.T) {
	setupUserTestDB(t)

	service := NewUserService()
	email := uniqueEmail()

	// Register first
	regDTO := request.RegisterDTO{
		Email:    email,
		Password: "loginpassword",
		Prenom:   "Marie",
		Nom:      "Martin",
	}
	regResult, err := service.Register(regDTO)
	require.NoError(t, err)
	defer cleanupUser(t, regResult.User.ID)

	// Login
	loginDTO := request.LoginDTO{
		Email:    email,
		Password: "loginpassword",
	}

	loginResult, err := service.Login(loginDTO)
	require.NoError(t, err, "Failed to login")
	require.NotNil(t, loginResult)

	assert.NotEmpty(t, loginResult.Token)
	assert.NotEmpty(t, loginResult.RefreshToken)
	assert.Equal(t, email, loginResult.User.Email)
}

func TestUser_Login_WrongPassword(t *testing.T) {
	setupUserTestDB(t)

	service := NewUserService()
	email := uniqueEmail()

	regDTO := request.RegisterDTO{
		Email:    email,
		Password: "correctpassword",
		Prenom:   "Pierre",
		Nom:      "Durand",
	}
	regResult, err := service.Register(regDTO)
	require.NoError(t, err)
	defer cleanupUser(t, regResult.User.ID)

	loginDTO := request.LoginDTO{
		Email:    email,
		Password: "wrongpassword",
	}

	_, err = service.Login(loginDTO)
	require.Error(t, err, "Should fail on wrong password")
	assert.Equal(t, "invalid email or password", err.Error())
}

func TestUser_Login_WrongEmail(t *testing.T) {
	setupUserTestDB(t)

	service := NewUserService()

	loginDTO := request.LoginDTO{
		Email:    "nonexistent@example.com",
		Password: "anything",
	}

	_, err := service.Login(loginDTO)
	require.Error(t, err, "Should fail on wrong email")
	assert.Equal(t, "invalid email or password", err.Error())
}

func TestUser_Login_InactiveAccount(t *testing.T) {
	setupUserTestDB(t)

	service := NewUserService()
	email := uniqueEmail()

	// Register
	regDTO := request.RegisterDTO{
		Email:    email,
		Password: "testpassword",
		Prenom:   "Inactive",
		Nom:      "User",
	}
	regResult, err := service.Register(regDTO)
	require.NoError(t, err)
	defer cleanupUser(t, regResult.User.ID)

	// Deactivate the user directly in DB
	api.DB.Model(&models.User{}).Where("id = ?", regResult.User.ID).Update("actif", false)

	loginDTO := request.LoginDTO{
		Email:    email,
		Password: "testpassword",
	}

	_, err = service.Login(loginDTO)
	require.Error(t, err, "Should fail on inactive account")
	assert.Equal(t, "account is inactive", err.Error())
}

func TestUser_GetByID(t *testing.T) {
	setupUserTestDB(t)

	service := NewUserService()
	email := uniqueEmail()

	regDTO := request.RegisterDTO{
		Email:    email,
		Password: "testpassword",
		Prenom:   "GetBy",
		Nom:      "ID",
	}
	regResult, err := service.Register(regDTO)
	require.NoError(t, err)
	defer cleanupUser(t, regResult.User.ID)

	user, err := service.GetByID(regResult.User.ID)
	require.NoError(t, err, "Failed to get user by ID")

	assert.Equal(t, regResult.User.ID, user.ID)
	assert.Equal(t, email, user.Email)
	assert.Equal(t, "GetBy", user.Prenom)
	assert.Equal(t, "ID", user.Nom)
}

func TestUser_GetByID_NotFound(t *testing.T) {
	setupUserTestDB(t)

	service := NewUserService()

	_, err := service.GetByID(99999)
	require.Error(t, err, "Should return error for non-existent user")
	assert.Equal(t, "user not found", err.Error())
}

func TestUser_RefreshToken(t *testing.T) {
	setupUserTestDB(t)

	service := NewUserService()
	email := uniqueEmail()

	regDTO := request.RegisterDTO{
		Email:    email,
		Password: "testpassword",
		Prenom:   "Refresh",
		Nom:      "Token",
	}
	regResult, err := service.Register(regDTO)
	require.NoError(t, err)
	defer cleanupUser(t, regResult.User.ID)

	// Use the refresh token to get new tokens
	refreshResult, err := service.RefreshToken(regResult.RefreshToken)
	require.NoError(t, err, "Failed to refresh token")

	assert.NotEmpty(t, refreshResult.Token)
	assert.NotEmpty(t, refreshResult.RefreshToken)
	assert.Equal(t, email, refreshResult.User.Email)
	// New refresh token should differ from the old one
	assert.NotEqual(t, regResult.RefreshToken, refreshResult.RefreshToken)
}

func TestUser_RefreshToken_Invalid(t *testing.T) {
	setupUserTestDB(t)

	service := NewUserService()

	_, err := service.RefreshToken("not-a-real-token")
	require.Error(t, err, "Should fail on invalid refresh token")
	assert.Contains(t, err.Error(), "invalid or expired refresh token")
}

func TestUser_RefreshToken_Mismatch(t *testing.T) {
	setupUserTestDB(t)

	service := NewUserService()
	email := uniqueEmail()

	regDTO := request.RegisterDTO{
		Email:    email,
		Password: "testpassword",
		Prenom:   "Mismatch",
		Nom:      "Test",
	}
	regResult, err := service.Register(regDTO)
	require.NoError(t, err)
	defer cleanupUser(t, regResult.User.ID)

	oldRefreshToken := regResult.RefreshToken

	// Change the stored refresh token by logging in again (which generates new tokens)
	loginDTO := request.LoginDTO{
		Email:    email,
		Password: "testpassword",
	}
	_, err = service.Login(loginDTO)
	require.NoError(t, err)

	// Try to use the old refresh token — it should be mismatched
	_, err = service.RefreshToken(oldRefreshToken)
	require.Error(t, err, "Should fail on mismatched refresh token")
	assert.Contains(t, err.Error(), "invalid refresh token")
}
