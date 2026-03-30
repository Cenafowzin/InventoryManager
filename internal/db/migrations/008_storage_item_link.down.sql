DROP INDEX IF EXISTS idx_storage_spaces_item;
ALTER TABLE storage_spaces DROP COLUMN IF EXISTS item_id;
