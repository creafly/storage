package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/creafly/storage/internal/domain/entity"
	"github.com/creafly/storage/internal/domain/service"
	"github.com/creafly/storage/internal/i18n"
	"github.com/creafly/storage/internal/middleware"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type FolderHandler struct {
	folderService *service.FolderService
}

func NewFolderHandler(folderService *service.FolderService) *FolderHandler {
	return &FolderHandler{folderService: folderService}
}

func (h *FolderHandler) Create(c *gin.Context) {
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

	var req entity.CreateFolderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": messages.Errors.InvalidRequest})
		return
	}

	folder, err := h.folderService.Create(c.Request.Context(), tenantID.(uuid.UUID), userID.(uuid.UUID), &req)
	if err != nil {
		if errors.Is(err, service.ErrFolderNameRequired) {
			c.JSON(http.StatusBadRequest, gin.H{"error": messages.Errors.InvalidRequest})
			return
		}
		if errors.Is(err, service.ErrInvalidParentFolder) {
			c.JSON(http.StatusBadRequest, gin.H{"error": messages.Errors.NotFound})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": messages.Errors.CreateFailed})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"folder": folder})
}

func (h *FolderHandler) List(c *gin.Context) {
	locale := middleware.GetLocale(c)
	messages := i18n.GetMessages(locale)

	tenantID, exists := c.Get("tenantID")
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": messages.Errors.TenantRequired})
		return
	}

	var parentID *uuid.UUID
	if parentIDStr := c.Query("parentId"); parentIDStr != "" {
		if parsedID, err := uuid.Parse(parentIDStr); err == nil {
			parentID = &parsedID
		}
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	folderList, err := h.folderService.List(c.Request.Context(), tenantID.(uuid.UUID), parentID, limit, offset)
	if err != nil {
		if errors.Is(err, service.ErrFolderNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": messages.Errors.NotFound})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": messages.Errors.FetchFailed})
		return
	}

	c.JSON(http.StatusOK, folderList)
}

func (h *FolderHandler) GetByID(c *gin.Context) {
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

	folder, err := h.folderService.GetByID(c.Request.Context(), tenantID.(uuid.UUID), id)
	if err != nil {
		if errors.Is(err, service.ErrFolderNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": messages.Errors.NotFound})
			return
		}
		if errors.Is(err, service.ErrFolderAccessDenied) {
			c.JSON(http.StatusForbidden, gin.H{"error": messages.Errors.Forbidden})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": messages.Errors.FetchFailed})
		return
	}

	c.JSON(http.StatusOK, gin.H{"folder": folder})
}

func (h *FolderHandler) Update(c *gin.Context) {
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

	var req entity.UpdateFolderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": messages.Errors.InvalidRequest})
		return
	}

	folder, err := h.folderService.Update(c.Request.Context(), tenantID.(uuid.UUID), id, &req)
	if err != nil {
		if errors.Is(err, service.ErrFolderNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": messages.Errors.NotFound})
			return
		}
		if errors.Is(err, service.ErrFolderAccessDenied) {
			c.JSON(http.StatusForbidden, gin.H{"error": messages.Errors.Forbidden})
			return
		}
		if errors.Is(err, service.ErrCircularReference) {
			c.JSON(http.StatusBadRequest, gin.H{"error": messages.Errors.CircularReference})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": messages.Errors.UpdateFailed})
		return
	}

	c.JSON(http.StatusOK, gin.H{"folder": folder})
}

func (h *FolderHandler) Move(c *gin.Context) {
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

	var req entity.MoveFolderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": messages.Errors.InvalidRequest})
		return
	}

	folder, err := h.folderService.Move(c.Request.Context(), tenantID.(uuid.UUID), id, &req)
	if err != nil {
		if errors.Is(err, service.ErrFolderNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": messages.Errors.NotFound})
			return
		}
		if errors.Is(err, service.ErrFolderAccessDenied) {
			c.JSON(http.StatusForbidden, gin.H{"error": messages.Errors.Forbidden})
			return
		}
		if errors.Is(err, service.ErrCircularReference) {
			c.JSON(http.StatusBadRequest, gin.H{"error": messages.Errors.CircularReference})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": messages.Errors.UpdateFailed})
		return
	}

	c.JSON(http.StatusOK, gin.H{"folder": folder})
}

func (h *FolderHandler) Delete(c *gin.Context) {
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

	err = h.folderService.Delete(c.Request.Context(), tenantID.(uuid.UUID), id)
	if err != nil {
		if errors.Is(err, service.ErrFolderNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": messages.Errors.NotFound})
			return
		}
		if errors.Is(err, service.ErrFolderAccessDenied) {
			c.JSON(http.StatusForbidden, gin.H{"error": messages.Errors.Forbidden})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": messages.Errors.DeleteFailed})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": messages.Success.FolderDeleted})
}

func (h *FolderHandler) GetBreadcrumb(c *gin.Context) {
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

	breadcrumb, err := h.folderService.GetBreadcrumb(c.Request.Context(), tenantID.(uuid.UUID), id)
	if err != nil {
		if errors.Is(err, service.ErrFolderNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": messages.Errors.NotFound})
			return
		}
		if errors.Is(err, service.ErrFolderAccessDenied) {
			c.JSON(http.StatusForbidden, gin.H{"error": messages.Errors.Forbidden})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": messages.Errors.FetchFailed})
		return
	}

	c.JSON(http.StatusOK, gin.H{"breadcrumb": breadcrumb})
}
