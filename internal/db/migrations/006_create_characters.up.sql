CREATE TABLE characters (
    id                  UUID         PRIMARY KEY DEFAULT uuid_generate_v4(),
    campaign_id         UUID         NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    owner_user_id       UUID         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name                VARCHAR(100) NOT NULL,
    description         TEXT,
    max_carry_weight_kg FLOAT,
    created_at          TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_characters_campaign ON characters(campaign_id);
CREATE INDEX idx_characters_owner    ON characters(owner_user_id);

ALTER TABLE characters ENABLE ROW LEVEL SECURITY;
CREATE POLICY "allow all" ON characters FOR ALL USING (true) WITH CHECK (true);
