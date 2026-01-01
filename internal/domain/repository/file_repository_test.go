package repository

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/creafly/storage/internal/domain/entity"
	"github.com/creafly/storage/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileRepository_Create(t *testing.T) {
	tdb := testutil.SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewFileRepository(tdb.DB)
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		file := testutil.NewTestFile()

		err := repo.Create(ctx, file)
		require.NoError(t, err)

		created, err := repo.GetByID(ctx, file.ID)
		require.NoError(t, err)
		require.NotNil(t, created)
		assert.Equal(t, file.ID, created.ID)
		assert.Equal(t, file.TenantID, created.TenantID)
		assert.Equal(t, file.FileName, created.FileName)
		assert.Equal(t, file.ContentType, created.ContentType)
		assert.Equal(t, file.FileType, created.FileType)
		assert.Equal(t, file.Size, created.Size)
	})

	t.Run("duplicate id fails", func(t *testing.T) {
		file := testutil.NewTestFile()

		err := repo.Create(ctx, file)
		require.NoError(t, err)

		err = repo.Create(ctx, file)
		require.Error(t, err)
	})
}

func TestFileRepository_GetByID(t *testing.T) {
	tdb := testutil.SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewFileRepository(tdb.DB)
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		file := testutil.NewTestFile()
		err := repo.Create(ctx, file)
		require.NoError(t, err)

		found, err := repo.GetByID(ctx, file.ID)
		require.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, file.ID, found.ID)
		assert.Equal(t, file.OriginalName, found.OriginalName)
	})

	t.Run("not found returns nil", func(t *testing.T) {
		found, err := repo.GetByID(ctx, uuid.New())
		require.NoError(t, err)
		assert.Nil(t, found)
	})
}

func TestFileRepository_GetByTenantID(t *testing.T) {
	tdb := testutil.SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewFileRepository(tdb.DB)
	ctx := context.Background()

	t.Run("returns files for tenant", func(t *testing.T) {
		tenantID := uuid.New()

		for i := 0; i < 3; i++ {
			file := testutil.NewTestFileWithTenant(tenantID)
			err := repo.Create(ctx, file)
			require.NoError(t, err)
		}

		otherFile := testutil.NewTestFile()
		err := repo.Create(ctx, otherFile)
		require.NoError(t, err)

		files, err := repo.GetByTenantID(ctx, tenantID, nil, 10, 0)
		require.NoError(t, err)
		assert.Len(t, files, 3)

		for _, f := range files {
			assert.Equal(t, tenantID, f.TenantID)
		}
	})

	t.Run("filters by file type", func(t *testing.T) {
		tenantID := uuid.New()

		imageFile := testutil.NewTestFileWithTenant(tenantID)
		imageFile.FileType = entity.FileTypeImage
		err := repo.Create(ctx, imageFile)
		require.NoError(t, err)

		logoFile := testutil.NewTestFileWithTenant(tenantID)
		logoFile.FileType = entity.FileTypeLogo
		err = repo.Create(ctx, logoFile)
		require.NoError(t, err)

		docFile := testutil.NewTestFileWithTenant(tenantID)
		docFile.FileType = entity.FileTypeDocument
		err = repo.Create(ctx, docFile)
		require.NoError(t, err)

		imageType := entity.FileTypeImage
		files, err := repo.GetByTenantID(ctx, tenantID, &imageType, 10, 0)
		require.NoError(t, err)
		assert.Len(t, files, 1)
		assert.Equal(t, entity.FileTypeImage, files[0].FileType)
	})

	t.Run("respects limit and offset", func(t *testing.T) {
		tenantID := uuid.New()

		for i := 0; i < 5; i++ {
			file := testutil.NewTestFileWithTenant(tenantID)
			err := repo.Create(ctx, file)
			require.NoError(t, err)
		}

		files, err := repo.GetByTenantID(ctx, tenantID, nil, 2, 0)
		require.NoError(t, err)
		assert.Len(t, files, 2)

		files, err = repo.GetByTenantID(ctx, tenantID, nil, 2, 2)
		require.NoError(t, err)
		assert.Len(t, files, 2)

		files, err = repo.GetByTenantID(ctx, tenantID, nil, 2, 4)
		require.NoError(t, err)
		assert.Len(t, files, 1)
	})

	t.Run("empty result for unknown tenant", func(t *testing.T) {
		files, err := repo.GetByTenantID(ctx, uuid.New(), nil, 10, 0)
		require.NoError(t, err)
		assert.Empty(t, files)
	})
}

func TestFileRepository_Delete(t *testing.T) {
	tdb := testutil.SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewFileRepository(tdb.DB)
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		file := testutil.NewTestFile()
		err := repo.Create(ctx, file)
		require.NoError(t, err)

		err = repo.Delete(ctx, file.ID)
		require.NoError(t, err)

		found, err := repo.GetByID(ctx, file.ID)
		require.NoError(t, err)
		assert.Nil(t, found)
	})

	t.Run("not found returns error", func(t *testing.T) {
		err := repo.Delete(ctx, uuid.New())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "file not found")
	})
}

func TestFileRepository_UpdateURL(t *testing.T) {
	tdb := testutil.SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewFileRepository(tdb.DB)
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		file := testutil.NewTestFile()
		err := repo.Create(ctx, file)
		require.NoError(t, err)

		newURL := "https://new-storage.example.com/updated-path.png"
		err = repo.UpdateURL(ctx, file.ID, newURL)
		require.NoError(t, err)

		updated, err := repo.GetByID(ctx, file.ID)
		require.NoError(t, err)
		require.NotNil(t, updated)
		assert.Equal(t, newURL, updated.URL)
		assert.True(t, updated.UpdatedAt.After(file.UpdatedAt) || updated.UpdatedAt.Equal(file.UpdatedAt))
	})

	t.Run("update non-existent file does not error", func(t *testing.T) {
		err := repo.UpdateURL(ctx, uuid.New(), "https://example.com/test.png")
		require.NoError(t, err)
	})
}
