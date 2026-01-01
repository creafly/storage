package entity

import (
	"time"

	"github.com/google/uuid"
)

type FileType string

const (
	FileTypeImage    FileType = "image"
	FileTypeLogo     FileType = "logo"
	FileTypeDocument FileType = "document"
	FileTypeVideo    FileType = "video"
)

type Folder struct {
	ID        uuid.UUID  `db:"id" json:"id"`
	TenantID  uuid.UUID  `db:"tenant_id" json:"tenantId"`
	ParentID  *uuid.UUID `db:"parent_id" json:"parentId,omitempty"`
	Name      string     `db:"name" json:"name"`
	CreatedBy uuid.UUID  `db:"created_by" json:"createdBy"`
	CreatedAt time.Time  `db:"created_at" json:"createdAt"`
	UpdatedAt time.Time  `db:"updated_at" json:"updatedAt"`
}

type FolderWithCounts struct {
	Folder
	FileCount   int `json:"fileCount"`
	FolderCount int `json:"folderCount"`
}

type CreateFolderRequest struct {
	Name     string     `json:"name" binding:"required,min=1,max=255"`
	ParentID *uuid.UUID `json:"parentId,omitempty"`
}

type UpdateFolderRequest struct {
	Name     *string    `json:"name,omitempty" binding:"omitempty,min=1,max=255"`
	ParentID *uuid.UUID `json:"parentId,omitempty"`
}

type MoveFolderRequest struct {
	ParentID *uuid.UUID `json:"parentId"`
}

type FolderList struct {
	Folders []FolderWithCounts `json:"folders"`
	Total   int                `json:"total"`
	Limit   int                `json:"limit"`
	Offset  int                `json:"offset"`
}

type File struct {
	ID           uuid.UUID  `db:"id" json:"id"`
	TenantID     uuid.UUID  `db:"tenant_id" json:"tenantId"`
	UploadedBy   uuid.UUID  `db:"uploaded_by" json:"uploadedBy"`
	FolderID     *uuid.UUID `db:"folder_id" json:"folderId,omitempty"`
	FileName     string     `db:"file_name" json:"fileName"`
	OriginalName string     `db:"original_name" json:"originalName"`
	ContentType  string     `db:"content_type" json:"contentType"`
	FileType     FileType   `db:"file_type" json:"fileType"`
	Size         int64      `db:"size" json:"size"`
	Path         string     `db:"path" json:"path"`
	URL          string     `db:"url" json:"url"`
	CreatedAt    time.Time  `db:"created_at" json:"createdAt"`
	UpdatedAt    time.Time  `db:"updated_at" json:"updatedAt"`
}

type UploadFileRequest struct {
	TenantID    uuid.UUID
	UserID      uuid.UUID
	FolderID    *uuid.UUID
	FileType    FileType
	FileName    string
	ContentType string
	Size        int64
	Data        []byte
}

type MoveFileRequest struct {
	FolderID *uuid.UUID `json:"folderId"`
}

type FileFilter struct {
	TenantID *uuid.UUID
	FolderID *uuid.UUID
	FileType *FileType
	Limit    int
	Offset   int
}

type FileList struct {
	Files  []File `json:"files"`
	Total  int    `json:"total"`
	Limit  int    `json:"limit"`
	Offset int    `json:"offset"`
}

type FileUsage struct {
	TotalSize int64 `json:"totalSize"`
	Count     int   `json:"count"`
}
