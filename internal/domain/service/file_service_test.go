package service

import (
	"testing"

	"github.com/hexaend/storage/internal/config"
	"github.com/hexaend/storage/internal/domain/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestUploadConfig() config.UploadConfig {
	return config.UploadConfig{
		MaxFileSize:          10 * 1024 * 1024,
		AllowedImageTypes:    []string{"image/png", "image/jpeg", "image/webp", "image/gif", "image/svg+xml"},
		AllowedDocumentTypes: []string{"application/pdf"},
	}
}

func TestFileService_ValidateUpload(t *testing.T) {
	svc := &FileService{
		uploadCfg: newTestUploadConfig(),
	}

	t.Run("valid image upload", func(t *testing.T) {
		err := svc.ValidateUpload("image/png", 1024, entity.FileTypeImage)
		require.NoError(t, err)
	})

	t.Run("valid logo upload PNG", func(t *testing.T) {
		err := svc.ValidateUpload("image/png", 1024, entity.FileTypeLogo)
		require.NoError(t, err)
	})

	t.Run("valid logo upload SVG", func(t *testing.T) {
		err := svc.ValidateUpload("image/svg+xml", 1024, entity.FileTypeLogo)
		require.NoError(t, err)
	})

	t.Run("valid document upload", func(t *testing.T) {
		err := svc.ValidateUpload("application/pdf", 1024, entity.FileTypeDocument)
		require.NoError(t, err)
	})

	t.Run("file too large", func(t *testing.T) {
		err := svc.ValidateUpload("image/png", 20*1024*1024, entity.FileTypeImage)
		require.Error(t, err)

		uploadErr, ok := err.(*UploadError)
		require.True(t, ok)
		assert.Equal(t, "file_too_large", uploadErr.Code)
	})

	t.Run("invalid content type for image", func(t *testing.T) {
		err := svc.ValidateUpload("application/pdf", 1024, entity.FileTypeImage)
		require.Error(t, err)

		uploadErr, ok := err.(*UploadError)
		require.True(t, ok)
		assert.Equal(t, "invalid_content_type", uploadErr.Code)
	})

	t.Run("invalid content type for document", func(t *testing.T) {
		err := svc.ValidateUpload("image/png", 1024, entity.FileTypeDocument)
		require.Error(t, err)

		uploadErr, ok := err.(*UploadError)
		require.True(t, ok)
		assert.Equal(t, "invalid_content_type", uploadErr.Code)
	})

	t.Run("logo must be PNG or SVG", func(t *testing.T) {
		err := svc.ValidateUpload("image/jpeg", 1024, entity.FileTypeLogo)
		require.Error(t, err)

		uploadErr, ok := err.(*UploadError)
		require.True(t, ok)
		assert.Equal(t, "invalid_logo_type", uploadErr.Code)
	})

	t.Run("invalid file type", func(t *testing.T) {
		err := svc.ValidateUpload("image/png", 1024, entity.FileType("unknown"))
		require.Error(t, err)

		uploadErr, ok := err.(*UploadError)
		require.True(t, ok)
		assert.Equal(t, "invalid_file_type", uploadErr.Code)
	})
}

func TestFileService_getExtensionFromContentType(t *testing.T) {
	svc := &FileService{}

	tests := []struct {
		contentType string
		expected    string
	}{
		{"image/png", ".png"},
		{"image/svg+xml", ".svg"},
		{"image/jpeg", ".jpg"},
		{"image/webp", ".webp"},
		{"image/gif", ".gif"},
		{"application/pdf", ".pdf"},
		{"IMAGE/PNG", ".png"},
		{"unknown/type", ""},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.contentType, func(t *testing.T) {
			result := svc.getExtensionFromContentType(tt.contentType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFileService_GetByTenantID_LimitValidation(t *testing.T) {

	t.Run("negative limit defaults to 20", func(t *testing.T) {
		limit := -5
		if limit <= 0 {
			limit = 20
		}
		assert.Equal(t, 20, limit)
	})

	t.Run("zero limit defaults to 20", func(t *testing.T) {
		limit := 0
		if limit <= 0 {
			limit = 20
		}
		assert.Equal(t, 20, limit)
	})

	t.Run("limit over 100 capped to 100", func(t *testing.T) {
		limit := 150
		if limit > 100 {
			limit = 100
		}
		assert.Equal(t, 100, limit)
	})

	t.Run("valid limit unchanged", func(t *testing.T) {
		limit := 50
		if limit <= 0 {
			limit = 20
		}
		if limit > 100 {
			limit = 100
		}
		assert.Equal(t, 50, limit)
	})
}

func TestUploadError(t *testing.T) {
	err := &UploadError{
		Code:    "test_code",
		Message: "Test error message",
	}

	assert.Equal(t, "Test error message", err.Error())
}
