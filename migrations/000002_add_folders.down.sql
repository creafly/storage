DROP INDEX IF EXISTS idx_files_tenant_folder;
DROP INDEX IF EXISTS idx_files_folder_id;
ALTER TABLE files DROP COLUMN IF EXISTS folder_id;
DROP INDEX IF EXISTS idx_folders_tenant_parent;
DROP INDEX IF EXISTS idx_folders_parent_id;
DROP INDEX IF EXISTS idx_folders_tenant_id;
DROP TABLE IF EXISTS folders;
