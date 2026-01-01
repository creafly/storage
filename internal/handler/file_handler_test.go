package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/creafly/storage/internal/domain/entity"
	"github.com/creafly/storage/internal/domain/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type FileServiceInterface interface {
	ValidateUpload(contentType string, size int64, fileType entity.FileType) error
	Upload(ctx context.Context, req entity.UploadFileRequest) (*entity.File, error)
	GetByID(ctx context.Context, id uuid.UUID) (*entity.File, error)
	GetByTenantID(ctx context.Context, tenantID uuid.UUID, fileType *entity.FileType, limit, offset int) ([]entity.File, error)
	Delete(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) error
	GetPresignedURL(ctx context.Context, id uuid.UUID, tenantID uuid.UUID, expiry time.Duration) (string, error)
}

type MockFileService struct {
	mock.Mock
}

func (m *MockFileService) ValidateUpload(contentType string, size int64, fileType entity.FileType) error {
	args := m.Called(contentType, size, fileType)
	return args.Error(0)
}

func (m *MockFileService) Upload(ctx context.Context, req entity.UploadFileRequest) (*entity.File, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.File), args.Error(1)
}

func (m *MockFileService) GetByID(ctx context.Context, id uuid.UUID) (*entity.File, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.File), args.Error(1)
}

func (m *MockFileService) GetByTenantID(ctx context.Context, tenantID uuid.UUID, fileType *entity.FileType, limit, offset int) ([]entity.File, error) {
	args := m.Called(ctx, tenantID, fileType, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]entity.File), args.Error(1)
}

func (m *MockFileService) Delete(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) error {
	args := m.Called(ctx, id, tenantID)
	return args.Error(0)
}

func (m *MockFileService) GetPresignedURL(ctx context.Context, id uuid.UUID, tenantID uuid.UUID, expiry time.Duration) (string, error) {
	args := m.Called(ctx, id, tenantID, expiry)
	return args.String(0), args.Error(1)
}

type TestableFileHandler struct {
	fileService FileServiceInterface
}

func NewTestableFileHandler(fileService FileServiceInterface) *TestableFileHandler {
	return &TestableFileHandler{fileService: fileService}
}

func (h *TestableFileHandler) List(c *gin.Context) {
	tenantID, exists := c.Get("tenantID")
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tenant required"})
		return
	}

	var fileType *entity.FileType
	if ft := c.Query("type"); ft != "" {
		t := entity.FileType(ft)
		fileType = &t
	}

	limit := 20
	offset := 0

	files, err := h.fileService.GetByTenantID(c.Request.Context(), tenantID.(uuid.UUID), fileType, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch files"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"files": files})
}

func (h *TestableFileHandler) GetByID(c *gin.Context) {
	tenantID, exists := c.Get("tenantID")
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tenant required"})
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	file, err := h.fileService.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch file"})
		return
	}
	if file == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
		return
	}

	if file.TenantID != tenantID.(uuid.UUID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"file": file})
}

func (h *TestableFileHandler) Delete(c *gin.Context) {
	tenantID, exists := c.Get("tenantID")
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tenant required"})
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	err = h.fileService.Delete(c.Request.Context(), id, tenantID.(uuid.UUID))
	if err != nil {
		if uploadErr, ok := err.(*service.UploadError); ok {
			status := http.StatusBadRequest
			if uploadErr.Code == "not_found" {
				status = http.StatusNotFound
			} else if uploadErr.Code == "forbidden" {
				status = http.StatusForbidden
			}
			c.JSON(status, gin.H{"error": uploadErr.Message, "code": uploadErr.Code})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete file"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "File deleted successfully"})
}

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func TestTestableFileHandler_List(t *testing.T) {
	tenantID := uuid.New()

	t.Run("success", func(t *testing.T) {
		mockSvc := new(MockFileService)
		handler := NewTestableFileHandler(mockSvc)

		files := []entity.File{
			{ID: uuid.New(), TenantID: tenantID, FileName: "test1.png"},
			{ID: uuid.New(), TenantID: tenantID, FileName: "test2.png"},
		}

		mockSvc.On("GetByTenantID", mock.Anything, tenantID, (*entity.FileType)(nil), 20, 0).Return(files, nil)

		router := setupTestRouter()
		router.GET("/files", func(c *gin.Context) {
			c.Set("tenantID", tenantID)
			handler.List(c)
		})

		req := httptest.NewRequest(http.MethodGet, "/files", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Contains(t, response, "files")

		mockSvc.AssertExpectations(t)
	})

	t.Run("no tenant", func(t *testing.T) {
		mockSvc := new(MockFileService)
		handler := NewTestableFileHandler(mockSvc)

		router := setupTestRouter()
		router.GET("/files", handler.List)

		req := httptest.NewRequest(http.MethodGet, "/files", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestTestableFileHandler_GetByID(t *testing.T) {
	tenantID := uuid.New()
	fileID := uuid.New()

	t.Run("success", func(t *testing.T) {
		mockSvc := new(MockFileService)
		handler := NewTestableFileHandler(mockSvc)

		file := &entity.File{
			ID:       fileID,
			TenantID: tenantID,
			FileName: "test.png",
		}

		mockSvc.On("GetByID", mock.Anything, fileID).Return(file, nil)

		router := setupTestRouter()
		router.GET("/files/:id", func(c *gin.Context) {
			c.Set("tenantID", tenantID)
			handler.GetByID(c)
		})

		req := httptest.NewRequest(http.MethodGet, "/files/"+fileID.String(), nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		mockSvc.AssertExpectations(t)
	})

	t.Run("not found", func(t *testing.T) {
		mockSvc := new(MockFileService)
		handler := NewTestableFileHandler(mockSvc)

		mockSvc.On("GetByID", mock.Anything, fileID).Return(nil, nil)

		router := setupTestRouter()
		router.GET("/files/:id", func(c *gin.Context) {
			c.Set("tenantID", tenantID)
			handler.GetByID(c)
		})

		req := httptest.NewRequest(http.MethodGet, "/files/"+fileID.String(), nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
		mockSvc.AssertExpectations(t)
	})

	t.Run("forbidden - different tenant", func(t *testing.T) {
		mockSvc := new(MockFileService)
		handler := NewTestableFileHandler(mockSvc)

		differentTenantID := uuid.New()
		file := &entity.File{
			ID:       fileID,
			TenantID: differentTenantID,
			FileName: "test.png",
		}

		mockSvc.On("GetByID", mock.Anything, fileID).Return(file, nil)

		router := setupTestRouter()
		router.GET("/files/:id", func(c *gin.Context) {
			c.Set("tenantID", tenantID)
			handler.GetByID(c)
		})

		req := httptest.NewRequest(http.MethodGet, "/files/"+fileID.String(), nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
		mockSvc.AssertExpectations(t)
	})

	t.Run("invalid id", func(t *testing.T) {
		mockSvc := new(MockFileService)
		handler := NewTestableFileHandler(mockSvc)

		router := setupTestRouter()
		router.GET("/files/:id", func(c *gin.Context) {
			c.Set("tenantID", tenantID)
			handler.GetByID(c)
		})

		req := httptest.NewRequest(http.MethodGet, "/files/invalid-uuid", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestTestableFileHandler_Delete(t *testing.T) {
	tenantID := uuid.New()
	fileID := uuid.New()

	t.Run("success", func(t *testing.T) {
		mockSvc := new(MockFileService)
		handler := NewTestableFileHandler(mockSvc)

		mockSvc.On("Delete", mock.Anything, fileID, tenantID).Return(nil)

		router := setupTestRouter()
		router.DELETE("/files/:id", func(c *gin.Context) {
			c.Set("tenantID", tenantID)
			handler.Delete(c)
		})

		req := httptest.NewRequest(http.MethodDelete, "/files/"+fileID.String(), nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		mockSvc.AssertExpectations(t)
	})

	t.Run("not found", func(t *testing.T) {
		mockSvc := new(MockFileService)
		handler := NewTestableFileHandler(mockSvc)

		mockSvc.On("Delete", mock.Anything, fileID, tenantID).Return(&service.UploadError{
			Code:    "not_found",
			Message: "File not found",
		})

		router := setupTestRouter()
		router.DELETE("/files/:id", func(c *gin.Context) {
			c.Set("tenantID", tenantID)
			handler.Delete(c)
		})

		req := httptest.NewRequest(http.MethodDelete, "/files/"+fileID.String(), nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
		mockSvc.AssertExpectations(t)
	})

	t.Run("forbidden", func(t *testing.T) {
		mockSvc := new(MockFileService)
		handler := NewTestableFileHandler(mockSvc)

		mockSvc.On("Delete", mock.Anything, fileID, tenantID).Return(&service.UploadError{
			Code:    "forbidden",
			Message: "Access denied",
		})

		router := setupTestRouter()
		router.DELETE("/files/:id", func(c *gin.Context) {
			c.Set("tenantID", tenantID)
			handler.Delete(c)
		})

		req := httptest.NewRequest(http.MethodDelete, "/files/"+fileID.String(), nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
		mockSvc.AssertExpectations(t)
	})
}

func createMultipartForm(fileName string, fileContent []byte, contentType string) (*bytes.Buffer, string) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, _ := writer.CreateFormFile("file", fileName)
	io.Copy(part, bytes.NewReader(fileContent))

	writer.WriteField("type", "image")
	writer.Close()

	return body, writer.FormDataContentType()
}
