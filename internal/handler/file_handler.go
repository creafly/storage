package handler

import (
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/creafly/storage/internal/domain/entity"
	"github.com/creafly/storage/internal/domain/service"
	"github.com/creafly/storage/internal/i18n"
	"github.com/creafly/storage/internal/middleware"
)

type FileHandler struct {
	fileService *service.FileService
}

func NewFileHandler(fileService *service.FileService) *FileHandler {
	return &FileHandler{fileService: fileService}
}

func (h *FileHandler) Upload(c *gin.Context) {
	locale := middleware.GetLocale(c)
	messages := i18n.GetMessages(locale)

	tenantID, exists := c.Get("tenantID")
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": messages.Errors.TenantRequired})
		return
	}

	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": messages.Errors.Unauthorized})
		return
	}

	fileType := c.PostForm("type")
	if fileType == "" {
		fileType = "image"
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": messages.Errors.FileRequired})
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": messages.Errors.ReadFailed})
		return
	}

	req := entity.UploadFileRequest{
		TenantID:    tenantID.(uuid.UUID),
		UserID:      userID.(uuid.UUID),
		FileType:    entity.FileType(fileType),
		FileName:    header.Filename,
		ContentType: header.Header.Get("Content-Type"),
		Size:        header.Size,
		Data:        data,
	}

	uploadedFile, err := h.fileService.Upload(c.Request.Context(), req)
	if err != nil {
		if uploadErr, ok := err.(*service.UploadError); ok {
			status := http.StatusBadRequest
			if uploadErr.Code == "file_too_large" {
				status = http.StatusRequestEntityTooLarge
			}
			c.JSON(status, gin.H{"error": uploadErr.Message, "code": uploadErr.Code})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": messages.Errors.UploadFailed})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"file": uploadedFile})
}

func (h *FileHandler) List(c *gin.Context) {
	locale := middleware.GetLocale(c)
	messages := i18n.GetMessages(locale)

	tenantID, exists := c.Get("tenantID")
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": messages.Errors.TenantRequired})
		return
	}

	var fileType *entity.FileType
	if ft := c.Query("type"); ft != "" {
		t := entity.FileType(ft)
		fileType = &t
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	files, err := h.fileService.GetByTenantID(c.Request.Context(), tenantID.(uuid.UUID), fileType, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": messages.Errors.FetchFailed})
		return
	}

	c.JSON(http.StatusOK, gin.H{"files": files})
}

func (h *FileHandler) GetByID(c *gin.Context) {
	locale := middleware.GetLocale(c)
	messages := i18n.GetMessages(locale)

	tenantID, exists := c.Get("tenantID")
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": messages.Errors.TenantRequired})
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": messages.Errors.InvalidID})
		return
	}

	file, err := h.fileService.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": messages.Errors.FetchFailed})
		return
	}
	if file == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": messages.Errors.NotFound})
		return
	}

	if file.TenantID != tenantID.(uuid.UUID) {
		c.JSON(http.StatusForbidden, gin.H{"error": messages.Errors.Forbidden})
		return
	}

	c.JSON(http.StatusOK, gin.H{"file": file})
}

func (h *FileHandler) Delete(c *gin.Context) {
	locale := middleware.GetLocale(c)
	messages := i18n.GetMessages(locale)

	tenantID, exists := c.Get("tenantID")
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": messages.Errors.TenantRequired})
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": messages.Errors.InvalidID})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": messages.Errors.DeleteFailed})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": messages.Success.FileDeleted})
}

func (h *FileHandler) GetPresignedURL(c *gin.Context) {
	locale := middleware.GetLocale(c)
	messages := i18n.GetMessages(locale)

	tenantID, exists := c.Get("tenantID")
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": messages.Errors.TenantRequired})
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": messages.Errors.InvalidID})
		return
	}

	expiryMinutes, _ := strconv.Atoi(c.DefaultQuery("expiry", "60"))
	expiry := time.Duration(expiryMinutes) * time.Minute

	url, err := h.fileService.GetPresignedURL(c.Request.Context(), id, tenantID.(uuid.UUID), expiry)
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": messages.Errors.FetchFailed})
		return
	}

	c.JSON(http.StatusOK, gin.H{"url": url, "expiresIn": expiryMinutes * 60})
}
