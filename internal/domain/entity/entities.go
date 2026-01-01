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
)

type File struct {
	ID           uuid.UUID `db:"id" json:"id"`
	TenantID     uuid.UUID `db:"tenant_id" json:"tenantId"`
	UploadedBy   uuid.UUID `db:"uploaded_by" json:"uploadedBy"`
	FileName     string    `db:"file_name" json:"fileName"`
	OriginalName string    `db:"original_name" json:"originalName"`
	ContentType  string    `db:"content_type" json:"contentType"`
	FileType     FileType  `db:"file_type" json:"fileType"`
	Size         int64     `db:"size" json:"size"`
	Path         string    `db:"path" json:"path"`
	URL          string    `db:"url" json:"url"`
	CreatedAt    time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt    time.Time `db:"updated_at" json:"updatedAt"`
}

type UploadFileRequest struct {
	TenantID    uuid.UUID
	UserID      uuid.UUID
	FileType    FileType
	FileName    string
	ContentType string
	Size        int64
	Data        []byte
}

type FileFilter struct {
	TenantID *uuid.UUID
	FileType *FileType
	Limit    int
	Offset   int
}
