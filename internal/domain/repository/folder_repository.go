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

type FolderRepository struct {
	db *sqlx.DB
}

func NewFolderRepository(db *sqlx.DB) *FolderRepository {
	return &FolderRepository{db: db}
}

func (r *FolderRepository) Create(ctx context.Context, folder *entity.Folder) error {
	query := `
		INSERT INTO folders (id, tenant_id, parent_id, name, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.db.ExecContext(ctx, query,
		folder.ID, folder.TenantID, folder.ParentID, folder.Name,
		folder.CreatedBy, folder.CreatedAt, folder.UpdatedAt,
	)
	return err
}

func (r *FolderRepository) GetByID(ctx context.Context, id uuid.UUID) (*entity.Folder, error) {
	var folder entity.Folder
	query := `SELECT * FROM folders WHERE id = $1`
	err := r.db.GetContext(ctx, &folder, query, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &folder, err
}

func (r *FolderRepository) GetByTenantID(ctx context.Context, tenantID uuid.UUID, parentID *uuid.UUID, limit, offset int) ([]entity.Folder, error) {
	var folders []entity.Folder
	var query string
	var args []interface{}

	if parentID != nil {
		query = `SELECT * FROM folders WHERE tenant_id = $1 AND parent_id = $2 ORDER BY name ASC LIMIT $3 OFFSET $4`
		args = []interface{}{tenantID, *parentID, limit, offset}
	} else {
		query = `SELECT * FROM folders WHERE tenant_id = $1 AND parent_id IS NULL ORDER BY name ASC LIMIT $2 OFFSET $3`
		args = []interface{}{tenantID, limit, offset}
	}

	err := r.db.SelectContext(ctx, &folders, query, args...)
	if err != nil {
		return nil, err
	}
	return folders, nil
}

func (r *FolderRepository) CountByTenantID(ctx context.Context, tenantID uuid.UUID, parentID *uuid.UUID) (int, error) {
	var count int
	var query string
	var args []interface{}

	if parentID != nil {
		query = `SELECT COUNT(*) FROM folders WHERE tenant_id = $1 AND parent_id = $2`
		args = []interface{}{tenantID, *parentID}
	} else {
		query = `SELECT COUNT(*) FROM folders WHERE tenant_id = $1 AND parent_id IS NULL`
		args = []interface{}{tenantID}
	}

	err := r.db.GetContext(ctx, &count, query, args...)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (r *FolderRepository) Update(ctx context.Context, folder *entity.Folder) error {
	query := `
		UPDATE folders 
		SET name = $1, parent_id = $2, updated_at = $3 
		WHERE id = $4
	`
	result, err := r.db.ExecContext(ctx, query, folder.Name, folder.ParentID, time.Now(), folder.ID)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("folder not found")
	}
	return nil
}

func (r *FolderRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM folders WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("folder not found")
	}
	return nil
}

func (r *FolderRepository) GetChildFolderCount(ctx context.Context, folderID uuid.UUID) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM folders WHERE parent_id = $1`
	err := r.db.GetContext(ctx, &count, query, folderID)
	return count, err
}

func (r *FolderRepository) GetFileCount(ctx context.Context, folderID uuid.UUID) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM files WHERE folder_id = $1`
	err := r.db.GetContext(ctx, &count, query, folderID)
	return count, err
}

func (r *FolderRepository) GetBreadcrumb(ctx context.Context, folderID uuid.UUID) ([]entity.Folder, error) {
	query := `
		WITH RECURSIVE folder_path AS (
			SELECT id, tenant_id, parent_id, name, created_by, created_at, updated_at
			FROM folders
			WHERE id = $1
			UNION ALL
			SELECT f.id, f.tenant_id, f.parent_id, f.name, f.created_by, f.created_at, f.updated_at
			FROM folders f
			INNER JOIN folder_path fp ON f.id = fp.parent_id
		)
		SELECT * FROM folder_path ORDER BY created_at ASC
	`
	var folders []entity.Folder
	err := r.db.SelectContext(ctx, &folders, query, folderID)
	if err != nil {
		return nil, err
	}
	return folders, nil
}

func (r *FolderRepository) IsDescendant(ctx context.Context, parentID, childID uuid.UUID) (bool, error) {
	query := `
		WITH RECURSIVE folder_tree AS (
			SELECT id, parent_id FROM folders WHERE id = $1
			UNION ALL
			SELECT f.id, f.parent_id FROM folders f
			INNER JOIN folder_tree ft ON f.parent_id = ft.id
		)
		SELECT EXISTS(SELECT 1 FROM folder_tree WHERE id = $2)
	`
	var exists bool
	err := r.db.GetContext(ctx, &exists, query, parentID, childID)
	return exists, err
}
