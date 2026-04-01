ALTER TABLE characters ALTER COLUMN owner_user_id DROP NOT NULL;
ALTER TABLE characters ADD COLUMN is_reserve BOOLEAN NOT NULL DEFAULT FALSE;
CREATE UNIQUE INDEX idx_characters_campaign_reserve ON characters(campaign_id) WHERE is_reserve = TRUE;
