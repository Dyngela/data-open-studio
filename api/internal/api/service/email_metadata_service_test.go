package service

import (
	"api"
	"api/internal/api/models"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupEmailMetadataTestDB(t *testing.T) {
	api.InitConfig("../../../.env.test")

	err := api.DB.AutoMigrate(&models.MetadataEmail{})
	require.NoError(t, err, "Failed to migrate metadata_email table")
}

func cleanupEmailMetadata(t *testing.T, id uint) {
	if id > 0 {
		api.DB.Unscoped().Delete(&models.MetadataEmail{}, id)
	}
}

func TestEmailMetadata_Create(t *testing.T) {
	setupEmailMetadataTestDB(t)

	service := NewEmailMetadataService()

	metadata := models.MetadataEmail{
		Name:     "Test Email",
		ImapHost: "imap.example.com",
		ImapPort: 993,
		SmtpHost: "smtp.example.com",
		SmtpPort: 587,
		Username: "testuser@example.com",
		Password: "testpass",
		UseTLS:   true,
	}

	created, err := service.Create(metadata)
	require.NoError(t, err, "Failed to create email metadata")
	require.NotNil(t, created)
	require.NotZero(t, created.ID)

	defer cleanupEmailMetadata(t, created.ID)

	assert.Equal(t, "Test Email", created.Name)
	assert.Equal(t, "imap.example.com", created.ImapHost)
	assert.Equal(t, 993, created.ImapPort)
	assert.Equal(t, "smtp.example.com", created.SmtpHost)
	assert.Equal(t, 587, created.SmtpPort)
	assert.Equal(t, "testuser@example.com", created.Username)
	assert.Equal(t, "testpass", created.Password)
	assert.True(t, created.UseTLS)
}

func TestEmailMetadata_FindByID(t *testing.T) {
	setupEmailMetadataTestDB(t)

	service := NewEmailMetadataService()

	metadata := models.MetadataEmail{
		Name:     "Find Me Email",
		ImapHost: "imap.find.com",
		ImapPort: 993,
		SmtpHost: "smtp.find.com",
		SmtpPort: 465,
		Username: "find@example.com",
		Password: "findpass",
		UseTLS:   true,
	}

	created, err := service.Create(metadata)
	require.NoError(t, err)
	defer cleanupEmailMetadata(t, created.ID)

	found, err := service.FindByID(created.ID)
	require.NoError(t, err, "Failed to find email metadata by ID")
	require.NotNil(t, found)

	assert.Equal(t, created.ID, found.ID)
	assert.Equal(t, "imap.find.com", found.ImapHost)
	assert.Equal(t, "find@example.com", found.Username)
}

func TestEmailMetadata_FindByID_NotFound(t *testing.T) {
	setupEmailMetadataTestDB(t)

	service := NewEmailMetadataService()

	_, err := service.FindByID(99999)
	require.Error(t, err, "Should return error for non-existent ID")
}

func TestEmailMetadata_FindAll(t *testing.T) {
	setupEmailMetadataTestDB(t)

	service := NewEmailMetadataService()

	m1 := models.MetadataEmail{
		Name:     "Email 1",
		ImapHost: "imap1.example.com",
		ImapPort: 993,
		Username: "user1@example.com",
		Password: "pass1",
	}
	m2 := models.MetadataEmail{
		Name:     "Email 2",
		ImapHost: "imap2.example.com",
		ImapPort: 993,
		Username: "user2@example.com",
		Password: "pass2",
	}

	created1, err := service.Create(m1)
	require.NoError(t, err)
	defer cleanupEmailMetadata(t, created1.ID)

	created2, err := service.Create(m2)
	require.NoError(t, err)
	defer cleanupEmailMetadata(t, created2.ID)

	all, err := service.FindAll()
	require.NoError(t, err, "Failed to find all email metadata")
	assert.GreaterOrEqual(t, len(all), 2, "Should have at least 2 email metadata entries")
}

func TestEmailMetadata_Update(t *testing.T) {
	setupEmailMetadataTestDB(t)

	service := NewEmailMetadataService()

	metadata := models.MetadataEmail{
		Name:     "Old Email",
		ImapHost: "old-imap.example.com",
		ImapPort: 993,
		SmtpHost: "old-smtp.example.com",
		SmtpPort: 587,
		Username: "old@example.com",
		Password: "oldpass",
		UseTLS:   true,
	}

	created, err := service.Create(metadata)
	require.NoError(t, err)
	defer cleanupEmailMetadata(t, created.ID)

	patch := map[string]any{
		"name":      "Updated Email",
		"imap_host": "new-imap.example.com",
		"username":  "new@example.com",
	}

	updated, err := service.Update(created.ID, patch)
	require.NoError(t, err, "Failed to update email metadata")
	require.NotNil(t, updated)

	assert.Equal(t, "Updated Email", updated.Name)
	assert.Equal(t, "new-imap.example.com", updated.ImapHost)
	assert.Equal(t, "new@example.com", updated.Username)
	// Unchanged fields
	assert.Equal(t, "old-smtp.example.com", updated.SmtpHost)
	assert.Equal(t, "oldpass", updated.Password)
}

func TestEmailMetadata_Delete(t *testing.T) {
	setupEmailMetadataTestDB(t)

	service := NewEmailMetadataService()

	metadata := models.MetadataEmail{
		Name:     "Delete Me Email",
		ImapHost: "delete-imap.example.com",
		ImapPort: 993,
		Username: "delete@example.com",
		Password: "deletepass",
	}

	created, err := service.Create(metadata)
	require.NoError(t, err)

	err = service.Delete(created.ID)
	require.NoError(t, err, "Failed to delete email metadata")

	_, err = service.FindByID(created.ID)
	require.Error(t, err, "Should not find deleted email metadata")
}
