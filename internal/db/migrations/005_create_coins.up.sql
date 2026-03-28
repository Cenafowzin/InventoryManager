CREATE TABLE coin_types (
    id            UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    campaign_id   UUID        NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    name          VARCHAR(50) NOT NULL,
    abbreviation  VARCHAR(10) NOT NULL,
    emoji         VARCHAR(10),
    is_default    BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (campaign_id, abbreviation)
);

ALTER TABLE coin_types ENABLE ROW LEVEL SECURITY;

CREATE INDEX idx_coin_types_campaign ON coin_types(campaign_id);

CREATE UNIQUE INDEX idx_coin_default_per_campaign
    ON coin_types(campaign_id)
    WHERE is_default = TRUE;

CREATE TABLE coin_conversions (
    id           UUID           PRIMARY KEY DEFAULT uuid_generate_v4(),
    campaign_id  UUID           NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    from_coin_id UUID           NOT NULL REFERENCES coin_types(id) ON DELETE CASCADE,
    to_coin_id   UUID           NOT NULL REFERENCES coin_types(id) ON DELETE CASCADE,
    rate         NUMERIC(18, 8) NOT NULL CHECK (rate > 0),
    UNIQUE (from_coin_id, to_coin_id)
);

ALTER TABLE coin_conversions ENABLE ROW LEVEL SECURITY;

CREATE INDEX idx_conversions_campaign ON coin_conversions(campaign_id);
