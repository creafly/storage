package service

import (
	"context"
	"errors"
	"time"

	"github.com/creafly/storage/internal/domain/entity"
	"github.com/creafly/storage/internal/domain/repository"
	"github.com/creafly/storage/internal/utils"
	"github.com/google/uuid"
)

var (
	ErrFolderNotFound      = errors.New("folder not found")
	ErrFolderAccessDenied  = errors.New("folder access denied")
	ErrCircularReference   = errors.New("cannot move folder into its own descendant")
	ErrFolderNameRequired  = errors.New("folder name is required")
	ErrInvalidParentFolder = errors.New("invalid parent folder")
)

type FolderService struct {
	folderRepo *repository.FolderRepository
	fileRepo   *repository.FileRepository
}

func NewFolderService(folderRepo *repository.FolderRepository, fileRepo *repository.FileRepository) *FolderService {
	return &FolderService{
		folderRepo: folderRepo,
		fileRepo:   fileRepo,
	}
}

func (s *FolderService) Create(ctx context.Context, tenantID, userID uuid.UUID, req *entity.CreateFolderRequest) (*entity.Folder, error) {
	if req.Name == "" {
		return nil, ErrFolderNameRequired
	}

	if req.ParentID != nil {
		parent, err := s.folderRepo.GetByID(ctx, *req.ParentID)
		if err != nil {
			return nil, err
		}
		if parent == nil {
			return nil, ErrInvalidParentFolder
		}
		if parent.TenantID != tenantID {
			return nil, ErrFolderAccessDenied
		}
	}

	now := time.Now()
	folder := &entity.Folder{
		ID:        utils.GenerateUUID(),
		TenantID:  tenantID,
		ParentID:  req.ParentID,
		Name:      req.Name,
		CreatedBy: userID,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.folderRepo.Create(ctx, folder); err != nil {
		return nil, err
	}

	return folder, nil
}

func (s *FolderService) GetByID(ctx context.Context, tenantID, folderID uuid.UUID) (*entity.FolderWithCounts, error) {
	folder, err := s.folderRepo.GetByID(ctx, folderID)
	if err != nil {
		return nil, err
	}
	if folder == nil {
		return nil, ErrFolderNotFound
	}
	if folder.TenantID != tenantID {
		return nil, ErrFolderAccessDenied
	}

	fileCount, err := s.folderRepo.GetFileCount(ctx, folderID)
	if err != nil {
		return nil, err
	}

	folderCount, err := s.folderRepo.GetChildFolderCount(ctx, folderID)
	if err != nil {
		return nil, err
	}

	return &entity.FolderWithCounts{
		Folder:      *folder,
		FileCount:   fileCount,
		FolderCount: folderCount,
	}, nil
}

func (s *FolderService) List(ctx context.Context, tenantID uuid.UUID, parentID *uuid.UUID, limit, offset int) (*entity.FolderList, error) {
	if parentID != nil {
		parent, err := s.folderRepo.GetByID(ctx, *parentID)
		if err != nil {
			return nil, err
		}
		if parent == nil {
			return nil, ErrFolderNotFound
		}
		if parent.TenantID != tenantID {
			return nil, ErrFolderAccessDenied
		}
	}

	folders, err := s.folderRepo.GetByTenantID(ctx, tenantID, parentID, limit, offset)
	if err != nil {
		return nil, err
	}

	total, err := s.folderRepo.CountByTenantID(ctx, tenantID, parentID)
	if err != nil {
		return nil, err
	}

	foldersWithCounts := make([]entity.FolderWithCounts, len(folders))
	for i, folder := range folders {
		fileCount, _ := s.folderRepo.GetFileCount(ctx, folder.ID)
		folderCount, _ := s.folderRepo.GetChildFolderCount(ctx, folder.ID)
		foldersWithCounts[i] = entity.FolderWithCounts{
			Folder:      folder,
			FileCount:   fileCount,
			FolderCount: folderCount,
		}
	}

	return &entity.FolderList{
		Folders: foldersWithCounts,
		Total:   total,
		Limit:   limit,
		Offset:  offset,
	}, nil
}

func (s *FolderService) Update(ctx context.Context, tenantID, folderID uuid.UUID, req *entity.UpdateFolderRequest) (*entity.Folder, error) {
	folder, err := s.folderRepo.GetByID(ctx, folderID)
	if err != nil {
		return nil, err
	}
	if folder == nil {
		return nil, ErrFolderNotFound
	}
	if folder.TenantID != tenantID {
		return nil, ErrFolderAccessDenied
	}

	if req.Name != nil {
		folder.Name = *req.Name
	}

	if req.ParentID != nil {
		if *req.ParentID == folderID {
			return nil, ErrCircularReference
		}

		isDescendant, err := s.folderRepo.IsDescendant(ctx, folderID, *req.ParentID)
		if err != nil {
			return nil, err
		}
		if isDescendant {
			return nil, ErrCircularReference
		}

		parent, err := s.folderRepo.GetByID(ctx, *req.ParentID)
		if err != nil {
			return nil, err
		}
		if parent == nil {
			return nil, ErrInvalidParentFolder
		}
		if parent.TenantID != tenantID {
			return nil, ErrFolderAccessDenied
		}
		folder.ParentID = req.ParentID
	}

	folder.UpdatedAt = time.Now()

	if err := s.folderRepo.Update(ctx, folder); err != nil {
		return nil, err
	}

	return folder, nil
}

func (s *FolderService) Move(ctx context.Context, tenantID, folderID uuid.UUID, req *entity.MoveFolderRequest) (*entity.Folder, error) {
	folder, err := s.folderRepo.GetByID(ctx, folderID)
	if err != nil {
		return nil, err
	}
	if folder == nil {
		return nil, ErrFolderNotFound
	}
	if folder.TenantID != tenantID {
		return nil, ErrFolderAccessDenied
	}

	if req.ParentID != nil {
		if *req.ParentID == folderID {
			return nil, ErrCircularReference
		}

		isDescendant, err := s.folderRepo.IsDescendant(ctx, folderID, *req.ParentID)
		if err != nil {
			return nil, err
		}
		if isDescendant {
			return nil, ErrCircularReference
		}

		parent, err := s.folderRepo.GetByID(ctx, *req.ParentID)
		if err != nil {
			return nil, err
		}
		if parent == nil {
			return nil, ErrInvalidParentFolder
		}
		if parent.TenantID != tenantID {
			return nil, ErrFolderAccessDenied
		}
	}

	folder.ParentID = req.ParentID
	folder.UpdatedAt = time.Now()

	if err := s.folderRepo.Update(ctx, folder); err != nil {
		return nil, err
	}

	return folder, nil
}

func (s *FolderService) Delete(ctx context.Context, tenantID, folderID uuid.UUID) error {
	folder, err := s.folderRepo.GetByID(ctx, folderID)
	if err != nil {
		return err
	}
	if folder == nil {
		return ErrFolderNotFound
	}
	if folder.TenantID != tenantID {
		return ErrFolderAccessDenied
	}

	return s.folderRepo.Delete(ctx, folderID)
}

func (s *FolderService) GetBreadcrumb(ctx context.Context, tenantID, folderID uuid.UUID) ([]entity.Folder, error) {
	folder, err := s.folderRepo.GetByID(ctx, folderID)
	if err != nil {
		return nil, err
	}
	if folder == nil {
		return nil, ErrFolderNotFound
	}
	if folder.TenantID != tenantID {
		return nil, ErrFolderAccessDenied
	}

	return s.folderRepo.GetBreadcrumb(ctx, folderID)
}
