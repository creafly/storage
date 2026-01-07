package service

import (
	"context"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/creafly/outbox"
	"github.com/creafly/storage/internal/config"
	"github.com/creafly/storage/internal/domain/entity"
	"github.com/creafly/storage/internal/domain/repository"
	"github.com/creafly/storage/internal/infra/minio"
	"github.com/google/uuid"
)

type FileService struct {
	fileRepo    *repository.FileRepository
	minioClient *minio.Client
	uploadCfg   config.UploadConfig
	outboxRepo  outbox.Repository
}

func NewFileService(
	fileRepo *repository.FileRepository,
	minioClient *minio.Client,
	uploadCfg config.UploadConfig,
	outboxRepo outbox.Repository,
) *FileService {
	return &FileService{
		fileRepo:    fileRepo,
		minioClient: minioClient,
		uploadCfg:   uploadCfg,
		outboxRepo:  outboxRepo,
	}
}

type UploadError struct {
	Code    string
	Message string
}

func (e *UploadError) Error() string {
	return e.Message
}

func (s *FileService) ValidateUpload(contentType string, size int64, fileType entity.FileType) error {
	if size > s.uploadCfg.MaxFileSize {
		return &UploadError{
			Code:    "file_too_large",
			Message: fmt.Sprintf("File size exceeds maximum allowed size of %d bytes", s.uploadCfg.MaxFileSize),
		}
	}

	var allowedTypes []string
	switch fileType {
	case entity.FileTypeImage, entity.FileTypeLogo:
		allowedTypes = s.uploadCfg.AllowedImageTypes
	case entity.FileTypeDocument:
		allowedTypes = s.uploadCfg.AllowedDocumentTypes
	case entity.FileTypeVideo:
		allowedTypes = s.uploadCfg.AllowedVideoTypes
	default:
		return &UploadError{
			Code:    "invalid_file_type",
			Message: "Invalid file type",
		}
	}

	if !slices.Contains(allowedTypes, contentType) {
		return &UploadError{
			Code:    "invalid_content_type",
			Message: fmt.Sprintf("Content type %s is not allowed for %s", contentType, fileType),
		}
	}

	if fileType == entity.FileTypeLogo {
		if contentType != "image/png" && contentType != "image/svg+xml" {
			return &UploadError{
				Code:    "invalid_logo_type",
				Message: "Logo must be PNG or SVG format",
			}
		}
	}

	return nil
}

func (s *FileService) Upload(ctx context.Context, req entity.UploadFileRequest) (*entity.File, error) {
	if err := s.ValidateUpload(req.ContentType, req.Size, req.FileType); err != nil {
		return nil, err
	}

	ext := filepath.Ext(req.FileName)
	if ext == "" {
		ext = s.getExtensionFromContentType(req.ContentType)
	}
	uniqueFileName := fmt.Sprintf("%s_%d%s", uuid.New().String(), time.Now().UnixNano(), ext)

	objectPath, err := s.minioClient.Upload(ctx, req.TenantID, uniqueFileName, req.ContentType, req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to upload to storage: %w", err)
	}

	url := s.minioClient.GetPublicURL(objectPath)

	file := &entity.File{
		ID:           uuid.New(),
		TenantID:     req.TenantID,
		UploadedBy:   req.UserID,
		FolderID:     req.FolderID,
		FileName:     uniqueFileName,
		OriginalName: req.FileName,
		ContentType:  req.ContentType,
		FileType:     req.FileType,
		Size:         req.Size,
		Path:         objectPath,
		URL:          url,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := s.fileRepo.Create(ctx, file); err != nil {
		_ = s.minioClient.Delete(ctx, objectPath)
		return nil, fmt.Errorf("failed to save file record: %w", err)
	}

	return file, nil
}

func (s *FileService) GetByID(ctx context.Context, id uuid.UUID) (*entity.File, error) {
	return s.fileRepo.GetByID(ctx, id)
}

func (s *FileService) GetByTenantID(ctx context.Context, tenantID uuid.UUID, folderID *uuid.UUID, fileType *entity.FileType, limit, offset int, includeAll bool) (*entity.FileList, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	files, err := s.fileRepo.GetByTenantID(ctx, tenantID, folderID, fileType, limit, offset, includeAll)
	if err != nil {
		return nil, err
	}

	total, err := s.fileRepo.CountByTenantID(ctx, tenantID, folderID, fileType, includeAll)
	if err != nil {
		return nil, err
	}

	return &entity.FileList{
		Files:  files,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}, nil
}

func (s *FileService) GetUsage(ctx context.Context, tenantID uuid.UUID) (*entity.FileUsage, error) {
	totalSize, count, err := s.fileRepo.GetUsageByTenantID(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	return &entity.FileUsage{
		TotalSize: totalSize,
		Count:     count,
	}, nil
}

func (s *FileService) Delete(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) error {
	file, err := s.fileRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if file == nil {
		return &UploadError{Code: "not_found", Message: "File not found"}
	}

	if file.TenantID != tenantID {
		return &UploadError{Code: "forbidden", Message: "Access denied"}
	}

	isLogo := file.FileType == entity.FileTypeLogo

	if err := s.minioClient.Delete(ctx, file.Path); err != nil {
		return fmt.Errorf("failed to delete from storage: %w", err)
	}

	if err := s.fileRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete file record: %w", err)
	}

	if isLogo && s.outboxRepo != nil {
		payload, err := outbox.CreatePayload(map[string]any{
			"file_id":   id.String(),
			"tenant_id": tenantID.String(),
		})
		if err == nil {
			event := outbox.NewEvent("storage.logo_file_deleted", payload)
			_ = s.outboxRepo.Create(ctx, event)
		}
	}

	return nil
}

func (s *FileService) GetPresignedURL(ctx context.Context, id uuid.UUID, tenantID uuid.UUID, expiry time.Duration) (string, error) {
	file, err := s.fileRepo.GetByID(ctx, id)
	if err != nil {
		return "", err
	}
	if file == nil {
		return "", &UploadError{Code: "not_found", Message: "File not found"}
	}

	if file.TenantID != tenantID {
		return "", &UploadError{Code: "forbidden", Message: "Access denied"}
	}

	return s.minioClient.GetPresignedURL(ctx, file.Path, expiry)
}

func (s *FileService) getExtensionFromContentType(contentType string) string {
	extensions := map[string]string{
		"image/png":       ".png",
		"image/svg+xml":   ".svg",
		"image/jpeg":      ".jpg",
		"image/webp":      ".webp",
		"image/gif":       ".gif",
		"application/pdf": ".pdf",
		"text/plain":      ".txt",
		"text/markdown":   ".md",
	}
	if ext, ok := extensions[strings.ToLower(contentType)]; ok {
		return ext
	}
	return ""
}

type BatchDeleteResult struct {
	Deleted []uuid.UUID `json:"deleted"`
	Failed  []uuid.UUID `json:"failed"`
}

func (s *FileService) DeleteWithoutOutbox(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) error {
	file, err := s.fileRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if file == nil {
		return &UploadError{Code: "not_found", Message: "File not found"}
	}

	if file.TenantID != tenantID {
		return &UploadError{Code: "forbidden", Message: "Access denied"}
	}

	if err := s.minioClient.Delete(ctx, file.Path); err != nil {
		return fmt.Errorf("failed to delete from storage: %w", err)
	}

	if err := s.fileRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete file record: %w", err)
	}

	return nil
}

func (s *FileService) DeleteMany(ctx context.Context, ids []uuid.UUID, tenantID uuid.UUID) (*BatchDeleteResult, error) {
	if len(ids) == 0 {
		return &BatchDeleteResult{Deleted: []uuid.UUID{}, Failed: []uuid.UUID{}}, nil
	}

	files, err := s.fileRepo.GetByIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("failed to get files: %w", err)
	}

	fileMap := make(map[uuid.UUID]*entity.File)
	for i := range files {
		fileMap[files[i].ID] = &files[i]
	}

	result := &BatchDeleteResult{
		Deleted: []uuid.UUID{},
		Failed:  []uuid.UUID{},
	}

	var toDelete []uuid.UUID
	var pathsToDelete []string
	var logoFileIDs []string

	for _, id := range ids {
		file, exists := fileMap[id]
		if !exists {
			result.Failed = append(result.Failed, id)
			continue
		}

		if file.TenantID != tenantID {
			result.Failed = append(result.Failed, id)
			continue
		}

		toDelete = append(toDelete, id)
		pathsToDelete = append(pathsToDelete, file.Path)

		if file.FileType == entity.FileTypeLogo {
			logoFileIDs = append(logoFileIDs, id.String())
		}
	}

	for _, path := range pathsToDelete {
		_ = s.minioClient.Delete(ctx, path)
	}

	if len(toDelete) > 0 {
		if err := s.fileRepo.DeleteMany(ctx, toDelete); err != nil {
			return nil, fmt.Errorf("failed to delete file records: %w", err)
		}
		result.Deleted = toDelete
	}

	if len(logoFileIDs) > 0 && s.outboxRepo != nil {
		payload, err := outbox.CreatePayload(map[string]any{
			"file_ids":  logoFileIDs,
			"tenant_id": tenantID.String(),
		})
		if err == nil {
			event := outbox.NewEvent("storage.logo_files_deleted", payload)
			_ = s.outboxRepo.Create(ctx, event)
		}
	}

	return result, nil
}

func (s *FileService) DeleteManyWithoutOutbox(ctx context.Context, ids []uuid.UUID, tenantID uuid.UUID) (*BatchDeleteResult, error) {
	if len(ids) == 0 {
		return &BatchDeleteResult{Deleted: []uuid.UUID{}, Failed: []uuid.UUID{}}, nil
	}

	files, err := s.fileRepo.GetByIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("failed to get files: %w", err)
	}

	fileMap := make(map[uuid.UUID]*entity.File)
	for i := range files {
		fileMap[files[i].ID] = &files[i]
	}

	result := &BatchDeleteResult{
		Deleted: []uuid.UUID{},
		Failed:  []uuid.UUID{},
	}

	var toDelete []uuid.UUID
	var pathsToDelete []string

	for _, id := range ids {
		file, exists := fileMap[id]
		if !exists {
			result.Failed = append(result.Failed, id)
			continue
		}

		if file.TenantID != tenantID {
			result.Failed = append(result.Failed, id)
			continue
		}

		toDelete = append(toDelete, id)
		pathsToDelete = append(pathsToDelete, file.Path)
	}

	for _, path := range pathsToDelete {
		_ = s.minioClient.Delete(ctx, path)
	}

	if len(toDelete) > 0 {
		if err := s.fileRepo.DeleteMany(ctx, toDelete); err != nil {
			return nil, fmt.Errorf("failed to delete file records: %w", err)
		}
		result.Deleted = toDelete
	}

	return result, nil
}
