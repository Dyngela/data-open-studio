package service

import (
	"api"
	"api/internal/api/models"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupMetadataTestDB initializes the database connection for metadata tests
func setupMetadataTestDB(t *testing.T) {
	api.InitConfig("../../../.env.test")

	err := api.DB.AutoMigrate(&models.MetadataDatabase{}, &models.MetadataSftp{})
	require.NoError(t, err, "Failed to migrate metadata tables")
}

// cleanupDbMetadata removes test db metadata from the database
func cleanupDbMetadata(t *testing.T, id uint) {
	if id > 0 {
		api.DB.Unscoped().Delete(&models.MetadataDatabase{}, id)
	}
}

// cleanupSftpMetadata removes test sftp metadata from the database
func cleanupSftpMetadata(t *testing.T, id uint) {
	if id > 0 {
		api.DB.Unscoped().Delete(&models.MetadataSftp{}, id)
	}
}

// ============ DB Metadata Tests ============

func TestDbMetadata_Create(t *testing.T) {
	setupMetadataTestDB(t)

	service := NewMetadataService()

	metadata := models.MetadataDatabase{
		Host:         "localhost",
		Port:         "5433",
		User:         "testuser",
		Password:     "testpass",
		DatabaseName: "testdb",
		SSLMode:      "disable",
		Extra:        "",
	}

	created, err := service.Create(metadata)
	require.NoError(t, err, "Failed to create db metadata")
	require.NotNil(t, created, "Created metadata should not be nil")
	require.NotZero(t, created.ID, "Created metadata ID should not be zero")

	defer cleanupDbMetadata(t, created.ID)

	assert.Equal(t, "localhost", created.Host)
	assert.Equal(t, "5433", created.Port)
	assert.Equal(t, "testuser", created.User)
	assert.Equal(t, "testpass", created.Password)
	assert.Equal(t, "testdb", created.DatabaseName)
	assert.Equal(t, "disable", created.SSLMode)
}

func TestDbMetadata_FindByID(t *testing.T) {
	setupMetadataTestDB(t)

	service := NewMetadataService()

	// Create test metadata
	metadata := models.MetadataDatabase{
		Host:         "db.example.com",
		Port:         "5432",
		User:         "admin",
		Password:     "secret",
		DatabaseName: "production",
		SSLMode:      "require",
	}

	created, err := service.Create(metadata)
	require.NoError(t, err)
	defer cleanupDbMetadata(t, created.ID)

	// Find by ID
	found, err := service.FindByID(created.ID)
	require.NoError(t, err, "Failed to find db metadata by ID")
	require.NotNil(t, found)

	assert.Equal(t, created.ID, found.ID)
	assert.Equal(t, "db.example.com", found.Host)
	assert.Equal(t, "5432", found.Port)
	assert.Equal(t, "admin", found.User)
	assert.Equal(t, "production", found.DatabaseName)
}

func TestDbMetadata_FindByID_NotFound(t *testing.T) {
	setupMetadataTestDB(t)

	service := NewMetadataService()

	_, err := service.FindByID(99999)
	require.Error(t, err, "Should return error for non-existent ID")
}

func TestDbMetadata_FindAll(t *testing.T) {
	setupMetadataTestDB(t)

	service := NewMetadataService()

	// Create multiple test metadata
	metadata1 := models.MetadataDatabase{
		Host:         "host1.example.com",
		Port:         "5432",
		User:         "user1",
		Password:     "pass1",
		DatabaseName: "db1",
		SSLMode:      "disable",
	}
	metadata2 := models.MetadataDatabase{
		Host:         "host2.example.com",
		Port:         "5433",
		User:         "user2",
		Password:     "pass2",
		DatabaseName: "db2",
		SSLMode:      "require",
	}

	created1, err := service.Create(metadata1)
	require.NoError(t, err)
	defer cleanupDbMetadata(t, created1.ID)

	created2, err := service.Create(metadata2)
	require.NoError(t, err)
	defer cleanupDbMetadata(t, created2.ID)

	// Find all
	all, err := service.FindAll()
	require.NoError(t, err, "Failed to find all db metadata")
	assert.GreaterOrEqual(t, len(all), 2, "Should have at least 2 metadata entries")
}

func TestDbMetadata_Update(t *testing.T) {
	setupMetadataTestDB(t)

	service := NewMetadataService()

	// Create test metadata
	metadata := models.MetadataDatabase{
		Host:         "old-host.example.com",
		Port:         "5432",
		User:         "olduser",
		Password:     "oldpass",
		DatabaseName: "olddb",
		SSLMode:      "disable",
	}

	created, err := service.Create(metadata)
	require.NoError(t, err)
	defer cleanupDbMetadata(t, created.ID)

	// Update
	patch := map[string]any{
		"host":          "new-host.example.com",
		"user":          "newuser",
		"database_name": "newdb",
	}

	updated, err := service.Update(created.ID, patch)
	require.NoError(t, err, "Failed to update db metadata")
	require.NotNil(t, updated)

	assert.Equal(t, "new-host.example.com", updated.Host)
	assert.Equal(t, "newuser", updated.User)
	assert.Equal(t, "newdb", updated.DatabaseName)
	// Unchanged fields
	assert.Equal(t, "5432", updated.Port)
	assert.Equal(t, "oldpass", updated.Password)
}

func TestDbMetadata_Delete(t *testing.T) {
	setupMetadataTestDB(t)

	service := NewMetadataService()

	// Create test metadata
	metadata := models.MetadataDatabase{
		Host:         "delete-me.example.com",
		Port:         "5432",
		User:         "user",
		Password:     "pass",
		DatabaseName: "db",
		SSLMode:      "disable",
	}

	created, err := service.Create(metadata)
	require.NoError(t, err)

	// Delete
	err = service.Delete(created.ID)
	require.NoError(t, err, "Failed to delete db metadata")

	// Verify deleted
	_, err = service.FindByID(created.ID)
	require.Error(t, err, "Should not find deleted metadata")
}

// ============ SFTP Metadata Tests ============

func TestSftpMetadata_Create(t *testing.T) {
	setupMetadataTestDB(t)

	service := NewSftpMetadataService()

	metadata := models.MetadataSftp{
		Host:       "sftp.example.com",
		Port:       "22",
		User:       "sftpuser",
		Password:   "sftppass",
		PrivateKey: "",
		BasePath:   "/data/uploads",
		Extra:      "",
	}

	created, err := service.Create(metadata)
	require.NoError(t, err, "Failed to create sftp metadata")
	require.NotNil(t, created, "Created metadata should not be nil")
	require.NotZero(t, created.ID, "Created metadata ID should not be zero")

	defer cleanupSftpMetadata(t, created.ID)

	assert.Equal(t, "sftp.example.com", created.Host)
	assert.Equal(t, "22", created.Port)
	assert.Equal(t, "sftpuser", created.User)
	assert.Equal(t, "sftppass", created.Password)
	assert.Equal(t, "/data/uploads", created.BasePath)
}

func TestSftpMetadata_CreateWithPrivateKey(t *testing.T) {
	setupMetadataTestDB(t)

	service := NewSftpMetadataService()

	privateKey := `-----BEGIN OPENSSH PRIVATE KEY-----
b3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQAAAAAAAAABAAAAMwAAAAtzc2gtZW
QyNTUxOQAAACBbeWvHh1hVy4k9P/l0JsY3TEstVZDQ7CfNMgAAAA==
-----END OPENSSH PRIVATE KEY-----`

	metadata := models.MetadataSftp{
		Host:       "secure-sftp.example.com",
		Port:       "22",
		User:       "keyuser",
		Password:   "",
		PrivateKey: privateKey,
		BasePath:   "/secure/data",
	}

	created, err := service.Create(metadata)
	require.NoError(t, err, "Failed to create sftp metadata with private key")
	defer cleanupSftpMetadata(t, created.ID)

	assert.Equal(t, "secure-sftp.example.com", created.Host)
	assert.Equal(t, "keyuser", created.User)
	assert.NotEmpty(t, created.PrivateKey)
	assert.Empty(t, created.Password)
}

func TestSftpMetadata_FindByID(t *testing.T) {
	setupMetadataTestDB(t)

	service := NewSftpMetadataService()

	metadata := models.MetadataSftp{
		Host:     "find-me.example.com",
		Port:     "2222",
		User:     "finduser",
		Password: "findpass",
		BasePath: "/find/path",
	}

	created, err := service.Create(metadata)
	require.NoError(t, err)
	defer cleanupSftpMetadata(t, created.ID)

	found, err := service.FindByID(created.ID)
	require.NoError(t, err, "Failed to find sftp metadata by ID")
	require.NotNil(t, found)

	assert.Equal(t, created.ID, found.ID)
	assert.Equal(t, "find-me.example.com", found.Host)
	assert.Equal(t, "2222", found.Port)
	assert.Equal(t, "finduser", found.User)
}

func TestSftpMetadata_FindByID_NotFound(t *testing.T) {
	setupMetadataTestDB(t)

	service := NewSftpMetadataService()

	_, err := service.FindByID(99999)
	require.Error(t, err, "Should return error for non-existent ID")
}

func TestSftpMetadata_FindAll(t *testing.T) {
	setupMetadataTestDB(t)

	service := NewSftpMetadataService()

	metadata1 := models.MetadataSftp{
		Host:     "sftp1.example.com",
		Port:     "22",
		User:     "user1",
		Password: "pass1",
		BasePath: "/path1",
	}
	metadata2 := models.MetadataSftp{
		Host:     "sftp2.example.com",
		Port:     "2222",
		User:     "user2",
		Password: "pass2",
		BasePath: "/path2",
	}

	created1, err := service.Create(metadata1)
	require.NoError(t, err)
	defer cleanupSftpMetadata(t, created1.ID)

	created2, err := service.Create(metadata2)
	require.NoError(t, err)
	defer cleanupSftpMetadata(t, created2.ID)

	all, err := service.FindAll()
	require.NoError(t, err, "Failed to find all sftp metadata")
	assert.GreaterOrEqual(t, len(all), 2, "Should have at least 2 metadata entries")
}

func TestSftpMetadata_Update(t *testing.T) {
	setupMetadataTestDB(t)

	service := NewSftpMetadataService()

	metadata := models.MetadataSftp{
		Host:     "old-sftp.example.com",
		Port:     "22",
		User:     "olduser",
		Password: "oldpass",
		BasePath: "/old/path",
	}

	created, err := service.Create(metadata)
	require.NoError(t, err)
	defer cleanupSftpMetadata(t, created.ID)

	patch := map[string]any{
		"host":      "new-sftp.example.com",
		"user":      "newuser",
		"base_path": "/new/path",
	}

	updated, err := service.Update(created.ID, patch)
	require.NoError(t, err, "Failed to update sftp metadata")
	require.NotNil(t, updated)

	assert.Equal(t, "new-sftp.example.com", updated.Host)
	assert.Equal(t, "newuser", updated.User)
	assert.Equal(t, "/new/path", updated.BasePath)
	// Unchanged fields
	assert.Equal(t, "22", updated.Port)
	assert.Equal(t, "oldpass", updated.Password)
}

func TestSftpMetadata_Delete(t *testing.T) {
	setupMetadataTestDB(t)

	service := NewSftpMetadataService()

	metadata := models.MetadataSftp{
		Host:     "delete-sftp.example.com",
		Port:     "22",
		User:     "user",
		Password: "pass",
		BasePath: "/delete",
	}

	created, err := service.Create(metadata)
	require.NoError(t, err)

	err = service.Delete(created.ID)
	require.NoError(t, err, "Failed to delete sftp metadata")

	_, err = service.FindByID(created.ID)
	require.Error(t, err, "Should not find deleted metadata")
}
