package service

import (
	"api"
	"api/internal/api/models"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupJobTestDB(t *testing.T) {
	api.InitConfig("../../../.env.test")

	err := api.DB.AutoMigrate(
		&models.User{},
		&models.Job{},
		&models.Node{},
		&models.Port{},
		&models.JobUserAccess{},
	)
	require.NoError(t, err, "Failed to migrate job-related tables")
}

func cleanupJob(t *testing.T, id uint) {
	if id > 0 {
		api.DB.Where("job_id = ?", id).Unscoped().Delete(&models.JobUserAccess{})
		api.DB.Unscoped().Delete(&models.Job{}, id)
	}
}

func createTestUser(t *testing.T, email string) models.User {
	user := models.User{
		Email:    email,
		Password: "hashed",
		Prenom:   "Test",
		Nom:      "User",
		Role:     models.RoleUser,
		Actif:    true,
	}
	err := api.DB.Create(&user).Error
	require.NoError(t, err)
	return user
}

func cleanupTestUser(t *testing.T, id uint) {
	if id > 0 {
		api.DB.Unscoped().Delete(&models.User{}, id)
	}
}

// ============ Job CRUD Tests ============

func TestJob_Create(t *testing.T) {
	setupJobTestDB(t)

	service := NewJobService()

	user := createTestUser(t, uniqueEmail())
	defer cleanupTestUser(t, user.ID)

	job := models.Job{
		Name:        "Test Job",
		Description: "A test job",
		FilePath:    "/projects/test/",
		CreatorID:   user.ID,
		Active:      true,
		Visibility:  models.JobVisibilityPrivate,
	}

	created, err := service.Create(job)
	require.NoError(t, err, "Failed to create job")
	require.NotNil(t, created)
	require.NotZero(t, created.ID)
	defer cleanupJob(t, created.ID)

	assert.Equal(t, "Test Job", created.Name)
	assert.Equal(t, "A test job", created.Description)
	assert.Equal(t, "/projects/test/", created.FilePath)
	assert.Equal(t, user.ID, created.CreatorID)
	assert.True(t, created.Active)
}

func TestJob_FindByID(t *testing.T) {
	setupJobTestDB(t)

	service := NewJobService()

	user := createTestUser(t, uniqueEmail())
	defer cleanupTestUser(t, user.ID)

	job := models.Job{
		Name:        "Find Me Job",
		Description: "Should be findable",
		CreatorID:   user.ID,
		Visibility:  models.JobVisibilityPrivate,
	}

	created, err := service.Create(job)
	require.NoError(t, err)
	defer cleanupJob(t, created.ID)

	found, err := service.FindByID(created.ID)
	require.NoError(t, err, "Failed to find job by ID")
	require.NotNil(t, found)

	assert.Equal(t, created.ID, found.ID)
	assert.Equal(t, "Find Me Job", found.Name)
}

func TestJob_FindByID_NotFound(t *testing.T) {
	setupJobTestDB(t)

	service := NewJobService()

	_, err := service.FindByID(99999)
	require.Error(t, err, "Should return error for non-existent job")
	assert.Equal(t, "job not found", err.Error())
}

func TestJob_Update(t *testing.T) {
	setupJobTestDB(t)

	service := NewJobService()

	user := createTestUser(t, uniqueEmail())
	defer cleanupTestUser(t, user.ID)

	job := models.Job{
		Name:        "Old Name",
		Description: "Old description",
		CreatorID:   user.ID,
		Visibility:  models.JobVisibilityPrivate,
	}

	created, err := service.Create(job)
	require.NoError(t, err)
	defer cleanupJob(t, created.ID)

	patch := map[string]any{
		"name":        "New Name",
		"description": "New description",
	}

	updated, err := service.Update(created.ID, patch)
	require.NoError(t, err, "Failed to update job")
	require.NotNil(t, updated)

	assert.Equal(t, "New Name", updated.Name)
	assert.Equal(t, "New description", updated.Description)
	// CreatorID should remain unchanged
	assert.Equal(t, user.ID, updated.CreatorID)
}

func TestJob_Delete(t *testing.T) {
	setupJobTestDB(t)

	service := NewJobService()

	user := createTestUser(t, uniqueEmail())
	defer cleanupTestUser(t, user.ID)

	job := models.Job{
		Name:       "Delete Me Job",
		CreatorID:  user.ID,
		Visibility: models.JobVisibilityPrivate,
	}

	created, err := service.Create(job)
	require.NoError(t, err)

	err = service.Delete(created.ID)
	require.NoError(t, err, "Failed to delete job")

	_, err = service.FindByID(created.ID)
	require.Error(t, err, "Should not find deleted job")
}

func TestJob_FindAllForUser(t *testing.T) {
	setupJobTestDB(t)

	service := NewJobService()

	owner := createTestUser(t, uniqueEmail())
	defer cleanupTestUser(t, owner.ID)

	otherUser := createTestUser(t, uniqueEmail())
	defer cleanupTestUser(t, otherUser.ID)

	// Public job by other user
	publicJob := models.Job{
		Name:       "Public Job",
		CreatorID:  otherUser.ID,
		Visibility: models.JobVisibilityPublic,
	}
	createdPublic, err := service.Create(publicJob)
	require.NoError(t, err)
	defer cleanupJob(t, createdPublic.ID)

	// Private job by owner
	privateJob := models.Job{
		Name:       "Owner Private Job",
		CreatorID:  owner.ID,
		Visibility: models.JobVisibilityPrivate,
	}
	createdPrivate, err := service.Create(privateJob)
	require.NoError(t, err)
	defer cleanupJob(t, createdPrivate.ID)

	// Private job by other user (should NOT be visible to owner)
	otherPrivateJob := models.Job{
		Name:       "Other Private Job",
		CreatorID:  otherUser.ID,
		Visibility: models.JobVisibilityPrivate,
	}
	createdOtherPrivate, err := service.Create(otherPrivateJob)
	require.NoError(t, err)
	defer cleanupJob(t, createdOtherPrivate.ID)

	// Get all jobs for owner
	jobs, err := service.FindAllForUser(owner.ID)
	require.NoError(t, err)

	// Should include public job and owner's private job, but NOT other's private job
	jobIDs := make(map[uint]bool)
	for _, j := range jobs {
		jobIDs[j.ID] = true
	}

	assert.True(t, jobIDs[createdPublic.ID], "Should see public job")
	assert.True(t, jobIDs[createdPrivate.ID], "Should see own private job")
	assert.False(t, jobIDs[createdOtherPrivate.ID], "Should NOT see other's private job")
}

func TestJob_FindByFilePathForUser(t *testing.T) {
	setupJobTestDB(t)

	service := NewJobService()

	user := createTestUser(t, uniqueEmail())
	defer cleanupTestUser(t, user.ID)

	job1 := models.Job{
		Name:       "ETL Job 1",
		FilePath:   "/projects/etl/",
		CreatorID:  user.ID,
		Visibility: models.JobVisibilityPrivate,
	}
	created1, err := service.Create(job1)
	require.NoError(t, err)
	defer cleanupJob(t, created1.ID)

	job2 := models.Job{
		Name:       "Other Job",
		FilePath:   "/projects/other/",
		CreatorID:  user.ID,
		Visibility: models.JobVisibilityPrivate,
	}
	created2, err := service.Create(job2)
	require.NoError(t, err)
	defer cleanupJob(t, created2.ID)

	// Filter by file path
	jobs, err := service.FindByFilePathForUser("/projects/etl/", user.ID)
	require.NoError(t, err)

	found := false
	for _, j := range jobs {
		if j.ID == created1.ID {
			found = true
		}
		assert.Equal(t, "/projects/etl/", j.FilePath)
	}
	assert.True(t, found, "Should find job with matching file path")
}

// ============ Access Control Tests ============

func TestJob_CanUserAccess_Owner(t *testing.T) {
	setupJobTestDB(t)

	service := NewJobService()

	user := createTestUser(t, uniqueEmail())
	defer cleanupTestUser(t, user.ID)

	job := models.Job{
		Name:       "Owner Access Job",
		CreatorID:  user.ID,
		Visibility: models.JobVisibilityPrivate,
	}
	created, err := service.Create(job)
	require.NoError(t, err)
	defer cleanupJob(t, created.ID)

	canAccess, role, err := service.CanUserAccess(created.ID, user.ID)
	require.NoError(t, err)
	assert.True(t, canAccess)
	assert.Equal(t, models.Owner, role)
}

func TestJob_CanUserAccess_PublicJob(t *testing.T) {
	setupJobTestDB(t)

	service := NewJobService()

	owner := createTestUser(t, uniqueEmail())
	defer cleanupTestUser(t, owner.ID)

	other := createTestUser(t, uniqueEmail())
	defer cleanupTestUser(t, other.ID)

	job := models.Job{
		Name:       "Public Access Job",
		CreatorID:  owner.ID,
		Visibility: models.JobVisibilityPublic,
	}
	created, err := service.Create(job)
	require.NoError(t, err)
	defer cleanupJob(t, created.ID)

	canAccess, role, err := service.CanUserAccess(created.ID, other.ID)
	require.NoError(t, err)
	assert.True(t, canAccess)
	assert.Equal(t, models.Viewer, role)
}

func TestJob_CanUserAccess_SharedUser(t *testing.T) {
	setupJobTestDB(t)

	service := NewJobService()

	owner := createTestUser(t, uniqueEmail())
	defer cleanupTestUser(t, owner.ID)

	sharedUser := createTestUser(t, uniqueEmail())
	defer cleanupTestUser(t, sharedUser.ID)

	job := models.Job{
		Name:       "Shared Access Job",
		CreatorID:  owner.ID,
		Visibility: models.JobVisibilityPrivate,
	}
	created, err := service.Create(job)
	require.NoError(t, err)
	defer cleanupJob(t, created.ID)

	// Share with the other user as editor
	err = service.ShareJob(created.ID, []uint{sharedUser.ID}, models.Editor)
	require.NoError(t, err)

	canAccess, role, err := service.CanUserAccess(created.ID, sharedUser.ID)
	require.NoError(t, err)
	assert.True(t, canAccess)
	assert.Equal(t, models.Editor, role)
}

func TestJob_CanUserAccess_NoAccess(t *testing.T) {
	setupJobTestDB(t)

	service := NewJobService()

	owner := createTestUser(t, uniqueEmail())
	defer cleanupTestUser(t, owner.ID)

	stranger := createTestUser(t, uniqueEmail())
	defer cleanupTestUser(t, stranger.ID)

	job := models.Job{
		Name:       "No Access Job",
		CreatorID:  owner.ID,
		Visibility: models.JobVisibilityPrivate,
	}
	created, err := service.Create(job)
	require.NoError(t, err)
	defer cleanupJob(t, created.ID)

	canAccess, _, err := service.CanUserAccess(created.ID, stranger.ID)
	require.NoError(t, err)
	assert.False(t, canAccess)
}

// ============ Sharing Tests ============

func TestJob_ShareJob(t *testing.T) {
	setupJobTestDB(t)

	service := NewJobService()

	owner := createTestUser(t, uniqueEmail())
	defer cleanupTestUser(t, owner.ID)

	u1 := createTestUser(t, uniqueEmail())
	defer cleanupTestUser(t, u1.ID)

	u2 := createTestUser(t, uniqueEmail())
	defer cleanupTestUser(t, u2.ID)

	job := models.Job{
		Name:       "Share Test Job",
		CreatorID:  owner.ID,
		Visibility: models.JobVisibilityPrivate,
	}
	created, err := service.Create(job)
	require.NoError(t, err)
	defer cleanupJob(t, created.ID)

	err = service.ShareJob(created.ID, []uint{u1.ID, u2.ID}, models.Viewer)
	require.NoError(t, err)

	accessList, err := service.GetJobAccess(created.ID)
	require.NoError(t, err)
	assert.Len(t, accessList, 2)
}

func TestJob_UnshareJob(t *testing.T) {
	setupJobTestDB(t)

	service := NewJobService()

	owner := createTestUser(t, uniqueEmail())
	defer cleanupTestUser(t, owner.ID)

	u1 := createTestUser(t, uniqueEmail())
	defer cleanupTestUser(t, u1.ID)

	job := models.Job{
		Name:       "Unshare Test Job",
		CreatorID:  owner.ID,
		Visibility: models.JobVisibilityPrivate,
	}
	created, err := service.Create(job)
	require.NoError(t, err)
	defer cleanupJob(t, created.ID)

	// Share then unshare
	err = service.ShareJob(created.ID, []uint{u1.ID}, models.Viewer)
	require.NoError(t, err)

	err = service.UnshareJob(created.ID, []uint{u1.ID})
	require.NoError(t, err)

	accessList, err := service.GetJobAccess(created.ID)
	require.NoError(t, err)
	assert.Len(t, accessList, 0)
}

func TestJob_UpdateJobSharing(t *testing.T) {
	setupJobTestDB(t)

	service := NewJobService()

	owner := createTestUser(t, uniqueEmail())
	defer cleanupTestUser(t, owner.ID)

	u1 := createTestUser(t, uniqueEmail())
	defer cleanupTestUser(t, u1.ID)

	u2 := createTestUser(t, uniqueEmail())
	defer cleanupTestUser(t, u2.ID)

	u3 := createTestUser(t, uniqueEmail())
	defer cleanupTestUser(t, u3.ID)

	job := models.Job{
		Name:       "Update Sharing Job",
		CreatorID:  owner.ID,
		Visibility: models.JobVisibilityPrivate,
	}
	created, err := service.Create(job)
	require.NoError(t, err)
	defer cleanupJob(t, created.ID)

	// Share with u1, u2
	err = service.ShareJob(created.ID, []uint{u1.ID, u2.ID}, models.Viewer)
	require.NoError(t, err)

	// Replace sharing list with u2, u3
	err = service.UpdateJobSharing(created.ID, []uint{u2.ID, u3.ID}, models.Editor)
	require.NoError(t, err)

	accessList, err := service.GetJobAccess(created.ID)
	require.NoError(t, err)
	assert.Len(t, accessList, 2)

	userIDs := make(map[uint]bool)
	for _, a := range accessList {
		userIDs[a.UserID] = true
		assert.Equal(t, models.Editor, a.Role)
	}
	assert.False(t, userIDs[u1.ID], "u1 should no longer have access")
	assert.True(t, userIDs[u2.ID], "u2 should still have access")
	assert.True(t, userIDs[u3.ID], "u3 should now have access")
}

func TestJob_GetJobAccess(t *testing.T) {
	setupJobTestDB(t)

	service := NewJobService()

	owner := createTestUser(t, uniqueEmail())
	defer cleanupTestUser(t, owner.ID)

	u1 := createTestUser(t, uniqueEmail())
	defer cleanupTestUser(t, u1.ID)

	job := models.Job{
		Name:       "Get Access Job",
		CreatorID:  owner.ID,
		Visibility: models.JobVisibilityPrivate,
	}
	created, err := service.Create(job)
	require.NoError(t, err)
	defer cleanupJob(t, created.ID)

	// Initially empty
	accessList, err := service.GetJobAccess(created.ID)
	require.NoError(t, err)
	assert.Len(t, accessList, 0)

	// Share and verify
	err = service.ShareJob(created.ID, []uint{u1.ID}, models.Editor)
	require.NoError(t, err)

	accessList, err = service.GetJobAccess(created.ID)
	require.NoError(t, err)
	assert.Len(t, accessList, 1)
	assert.Equal(t, u1.ID, accessList[0].UserID)
	assert.Equal(t, models.Editor, accessList[0].Role)
}
