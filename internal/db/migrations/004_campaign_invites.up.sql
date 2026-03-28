-- Adiciona criador na campanha
ALTER TABLE campaigns ADD COLUMN creator_user_id UUID REFERENCES users(id);

-- Preenche retroativamente com o primeiro GM de cada campanha
UPDATE campaigns c
SET creator_user_id = (
    SELECT user_id FROM campaign_members
    WHERE campaign_id = c.id AND role = 'gm'
    ORDER BY joined_at ASC
    LIMIT 1
);

-- Links de convite
CREATE TABLE campaign_invites (
    id          UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    campaign_id UUID        NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    code        VARCHAR(8)  NOT NULL UNIQUE,
    created_by  UUID        NOT NULL REFERENCES users(id),
    expires_at  TIMESTAMPTZ,
    used_at     TIMESTAMPTZ,
    used_by     UUID        REFERENCES users(id),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE campaign_invites ENABLE ROW LEVEL SECURITY;

CREATE INDEX idx_invites_campaign ON campaign_invites(campaign_id);
CREATE INDEX idx_invites_code     ON campaign_invites(code);
