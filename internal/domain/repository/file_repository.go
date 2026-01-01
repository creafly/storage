package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/hexaend/storage/internal/domain/entity"
	"github.com/jmoiron/sqlx"
)

type FileRepository struct {
	db *sqlx.DB
}

func NewFileRepository(db *sqlx.DB) *FileRepository {
	return &FileRepository{db: db}
}

func (r *FileRepository) Create(ctx context.Context, file *entity.File) error {
	query := `
		INSERT INTO files (id, tenant_id, uploaded_by, file_name, original_name, content_type, file_type, size, path, url, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`
	_, err := r.db.ExecContext(ctx, query,
		file.ID, file.TenantID, file.UploadedBy, file.FileName, file.OriginalName,
		file.ContentType, file.FileType, file.Size, file.Path, file.URL,
		file.CreatedAt, file.UpdatedAt,
	)
	return err
}

func (r *FileRepository) GetByID(ctx context.Context, id uuid.UUID) (*entity.File, error) {
	var file entity.File
	query := `SELECT * FROM files WHERE id = $1`
	err := r.db.GetContext(ctx, &file, query, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &file, err
}

func (r *FileRepository) GetByTenantID(ctx context.Context, tenantID uuid.UUID, fileType *entity.FileType, limit, offset int) ([]entity.File, error) {
	var files []entity.File
	var query string
	var args []interface{}

	if fileType != nil {
		query = `SELECT * FROM files WHERE tenant_id = $1 AND file_type = $2 ORDER BY created_at DESC LIMIT $3 OFFSET $4`
		args = []interface{}{tenantID, *fileType, limit, offset}
	} else {
		query = `SELECT * FROM files WHERE tenant_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`
		args = []interface{}{tenantID, limit, offset}
	}

	err := r.db.SelectContext(ctx, &files, query, args...)
	if err != nil {
		return nil, err
	}
	return files, nil
}

func (r *FileRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM files WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("file not found")
	}
	return nil
}

func (r *FileRepository) UpdateURL(ctx context.Context, id uuid.UUID, url string) error {
	query := `UPDATE files SET url = $1, updated_at = $2 WHERE id = $3`
	_, err := r.db.ExecContext(ctx, query, url, time.Now(), id)
	return err
}
