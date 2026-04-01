DROP INDEX IF EXISTS idx_characters_campaign_reserve;
ALTER TABLE characters DROP COLUMN IF EXISTS is_reserve;
