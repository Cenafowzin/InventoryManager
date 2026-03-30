ALTER TABLE storage_spaces
    ADD COLUMN item_id UUID REFERENCES items(id) ON DELETE SET NULL;

CREATE INDEX idx_storage_spaces_item ON storage_spaces(item_id) WHERE item_id IS NOT NULL;
