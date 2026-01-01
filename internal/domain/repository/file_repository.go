package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/creafly/storage/internal/domain/entity"
	"github.com/google/uuid"
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
		INSERT INTO files (id, tenant_id, uploaded_by, folder_id, file_name, original_name, content_type, file_type, size, path, url, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`
	_, err := r.db.ExecContext(ctx, query,
		file.ID, file.TenantID, file.UploadedBy, file.FolderID, file.FileName, file.OriginalName,
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

func (r *FileRepository) GetByTenantID(ctx context.Context, tenantID uuid.UUID, folderID *uuid.UUID, fileType *entity.FileType, limit, offset int, includeAll bool) ([]entity.File, error) {
	var files []entity.File
	var query string
	var args []interface{}

	if includeAll {
		if fileType != nil {
			query = `SELECT * FROM files WHERE tenant_id = $1 AND file_type = $2 ORDER BY created_at DESC LIMIT $3 OFFSET $4`
			args = []interface{}{tenantID, *fileType, limit, offset}
		} else {
			query = `SELECT * FROM files WHERE tenant_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`
			args = []interface{}{tenantID, limit, offset}
		}
	} else if folderID != nil && fileType != nil {
		query = `SELECT * FROM files WHERE tenant_id = $1 AND folder_id = $2 AND file_type = $3 ORDER BY created_at DESC LIMIT $4 OFFSET $5`
		args = []interface{}{tenantID, *folderID, *fileType, limit, offset}
	} else if folderID != nil {
		query = `SELECT * FROM files WHERE tenant_id = $1 AND folder_id = $2 ORDER BY created_at DESC LIMIT $3 OFFSET $4`
		args = []interface{}{tenantID, *folderID, limit, offset}
	} else if fileType != nil {
		query = `SELECT * FROM files WHERE tenant_id = $1 AND folder_id IS NULL AND file_type = $2 ORDER BY created_at DESC LIMIT $3 OFFSET $4`
		args = []interface{}{tenantID, *fileType, limit, offset}
	} else {
		query = `SELECT * FROM files WHERE tenant_id = $1 AND folder_id IS NULL ORDER BY created_at DESC LIMIT $2 OFFSET $3`
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

func (r *FileRepository) GetByIDs(ctx context.Context, ids []uuid.UUID) ([]entity.File, error) {
	if len(ids) == 0 {
		return []entity.File{}, nil
	}

	query, args, err := sqlx.In(`SELECT * FROM files WHERE id IN (?)`, ids)
	if err != nil {
		return nil, err
	}
	query = r.db.Rebind(query)

	var files []entity.File
	err = r.db.SelectContext(ctx, &files, query, args...)
	if err != nil {
		return nil, err
	}
	return files, nil
}

func (r *FileRepository) DeleteMany(ctx context.Context, ids []uuid.UUID) error {
	if len(ids) == 0 {
		return nil
	}

	query, args, err := sqlx.In(`DELETE FROM files WHERE id IN (?)`, ids)
	if err != nil {
		return err
	}
	query = r.db.Rebind(query)

	_, err = r.db.ExecContext(ctx, query, args...)
	return err
}

func (r *FileRepository) CountByTenantID(ctx context.Context, tenantID uuid.UUID, folderID *uuid.UUID, fileType *entity.FileType, includeAll bool) (int, error) {
	var count int
	var query string
	var args []interface{}

	if includeAll {
		if fileType != nil {
			query = `SELECT COUNT(*) FROM files WHERE tenant_id = $1 AND file_type = $2`
			args = []interface{}{tenantID, *fileType}
		} else {
			query = `SELECT COUNT(*) FROM files WHERE tenant_id = $1`
			args = []interface{}{tenantID}
		}
	} else if folderID != nil && fileType != nil {
		query = `SELECT COUNT(*) FROM files WHERE tenant_id = $1 AND folder_id = $2 AND file_type = $3`
		args = []interface{}{tenantID, *folderID, *fileType}
	} else if folderID != nil {
		query = `SELECT COUNT(*) FROM files WHERE tenant_id = $1 AND folder_id = $2`
		args = []interface{}{tenantID, *folderID}
	} else if fileType != nil {
		query = `SELECT COUNT(*) FROM files WHERE tenant_id = $1 AND folder_id IS NULL AND file_type = $2`
		args = []interface{}{tenantID, *fileType}
	} else {
		query = `SELECT COUNT(*) FROM files WHERE tenant_id = $1 AND folder_id IS NULL`
		args = []interface{}{tenantID}
	}

	err := r.db.GetContext(ctx, &count, query, args...)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (r *FileRepository) GetUsageByTenantID(ctx context.Context, tenantID uuid.UUID) (int64, int, error) {
	var result struct {
		TotalSize int64 `db:"total_size"`
		Count     int   `db:"count"`
	}
	query := `SELECT COALESCE(SUM(size), 0) as total_size, COUNT(*) as count FROM files WHERE tenant_id = $1`
	err := r.db.GetContext(ctx, &result, query, tenantID)
	if err != nil {
		return 0, 0, err
	}
	return result.TotalSize, result.Count, nil
}

func (r *FileRepository) UpdateFolderID(ctx context.Context, id uuid.UUID, folderID *uuid.UUID) error {
	query := `UPDATE files SET folder_id = $1, updated_at = $2 WHERE id = $3`
	result, err := r.db.ExecContext(ctx, query, folderID, time.Now(), id)
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

func (r *FileRepository) MoveFilesToFolder(ctx context.Context, fileIDs []uuid.UUID, folderID *uuid.UUID) error {
	if len(fileIDs) == 0 {
		return nil
	}

	query := `UPDATE files SET folder_id = $1, updated_at = $2 WHERE id = ANY($3)`
	_, err := r.db.ExecContext(ctx, query, folderID, time.Now(), fileIDs)
	return err
}
