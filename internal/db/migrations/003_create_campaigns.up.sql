CREATE TABLE campaigns (
    id          UUID         PRIMARY KEY DEFAULT uuid_generate_v4(),
    name        VARCHAR(100) NOT NULL,
    description TEXT,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

ALTER TABLE campaigns ENABLE ROW LEVEL SECURITY;

CREATE TYPE campaign_role AS ENUM ('gm', 'player');

CREATE TABLE campaign_members (
    id          UUID          PRIMARY KEY DEFAULT uuid_generate_v4(),
    campaign_id UUID          NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    user_id     UUID          NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role        campaign_role NOT NULL DEFAULT 'player',
    joined_at   TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    UNIQUE (campaign_id, user_id)
);

ALTER TABLE campaign_members ENABLE ROW LEVEL SECURITY;

CREATE INDEX idx_campaign_members_campaign ON campaign_members(campaign_id);
CREATE INDEX idx_campaign_members_user     ON campaign_members(user_id);
