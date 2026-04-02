-- ── Extensions ────────────────────────────────────────────────────────────────

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ── Enums ─────────────────────────────────────────────────────────────────────

CREATE TYPE campaign_role      AS ENUM ('gm', 'player');
CREATE TYPE transaction_type   AS ENUM ('buy', 'sell');
CREATE TYPE transaction_status AS ENUM ('draft', 'confirmed', 'cancelled');

-- ── Users ─────────────────────────────────────────────────────────────────────

CREATE TABLE users (
    id            UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    username      VARCHAR(50)  NOT NULL UNIQUE,
    email         VARCHAR(255) NOT NULL UNIQUE,
    password_hash TEXT         NOT NULL,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

ALTER TABLE users ENABLE ROW LEVEL SECURITY;

-- ── Campaigns ─────────────────────────────────────────────────────────────────

CREATE TABLE campaigns (
    id              UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    name            VARCHAR(100) NOT NULL,
    description     TEXT,
    creator_user_id UUID         REFERENCES users(id),
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

ALTER TABLE campaigns ENABLE ROW LEVEL SECURITY;

CREATE TABLE campaign_members (
    id          UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    campaign_id UUID          NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    user_id     UUID          NOT NULL REFERENCES users(id)     ON DELETE CASCADE,
    role        campaign_role NOT NULL DEFAULT 'player',
    joined_at   TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    UNIQUE (campaign_id, user_id)
);

ALTER TABLE campaign_members ENABLE ROW LEVEL SECURITY;

CREATE INDEX idx_campaign_members_campaign ON campaign_members(campaign_id);
CREATE INDEX idx_campaign_members_user     ON campaign_members(user_id);

CREATE TABLE campaign_invites (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
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

-- ── Coins ─────────────────────────────────────────────────────────────────────

CREATE TABLE coin_types (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    campaign_id  UUID        NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    name         VARCHAR(50) NOT NULL,
    abbreviation VARCHAR(10) NOT NULL,
    emoji        VARCHAR(10),
    is_default   BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (campaign_id, abbreviation)
);

ALTER TABLE coin_types ENABLE ROW LEVEL SECURITY;

CREATE INDEX idx_coin_types_campaign ON coin_types(campaign_id);
CREATE UNIQUE INDEX idx_coin_default_per_campaign ON coin_types(campaign_id) WHERE is_default = TRUE;

CREATE TABLE coin_conversions (
    id           UUID           PRIMARY KEY DEFAULT gen_random_uuid(),
    campaign_id  UUID           NOT NULL REFERENCES campaigns(id)  ON DELETE CASCADE,
    from_coin_id UUID           NOT NULL REFERENCES coin_types(id) ON DELETE CASCADE,
    to_coin_id   UUID           NOT NULL REFERENCES coin_types(id) ON DELETE CASCADE,
    rate         NUMERIC(18, 8) NOT NULL CHECK (rate > 0),
    is_canonical BOOLEAN        NOT NULL DEFAULT TRUE,
    UNIQUE (from_coin_id, to_coin_id)
);

ALTER TABLE coin_conversions ENABLE ROW LEVEL SECURITY;

CREATE INDEX idx_conversions_campaign ON coin_conversions(campaign_id);

-- ── Categories ────────────────────────────────────────────────────────────────

CREATE TABLE categories (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    campaign_id UUID        NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    name        VARCHAR(50) NOT NULL,
    color       VARCHAR(7),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (campaign_id, name)
);

ALTER TABLE categories ENABLE ROW LEVEL SECURITY;
CREATE POLICY "allow all" ON categories FOR ALL USING (true) WITH CHECK (true);

CREATE INDEX idx_categories_campaign ON categories(campaign_id);

-- ── Characters ────────────────────────────────────────────────────────────────

CREATE TABLE characters (
    id                  UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    campaign_id         UUID         NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    owner_user_id       UUID         REFERENCES users(id) ON DELETE CASCADE, -- NULL = GM reserve
    name                VARCHAR(100) NOT NULL,
    description         TEXT,
    max_carry_weight_kg FLOAT,
    is_reserve          BOOLEAN      NOT NULL DEFAULT FALSE,
    created_at          TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

ALTER TABLE characters ENABLE ROW LEVEL SECURITY;
CREATE POLICY "allow all" ON characters FOR ALL USING (true) WITH CHECK (true);

CREATE INDEX idx_characters_campaign ON characters(campaign_id);
CREATE INDEX idx_characters_owner    ON characters(owner_user_id);
CREATE UNIQUE INDEX idx_characters_campaign_reserve ON characters(campaign_id) WHERE is_reserve = TRUE;

-- ── Storage Spaces ────────────────────────────────────────────────────────────
-- item_id added after items table to resolve circular dependency

CREATE TABLE storage_spaces (
    id                 UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    character_id       UUID          NOT NULL REFERENCES characters(id) ON DELETE CASCADE,
    name               VARCHAR(100)  NOT NULL,
    description        TEXT,
    counts_toward_load BOOLEAN       NOT NULL DEFAULT TRUE,
    capacity_kg        NUMERIC(10,4),
    is_default         BOOLEAN       NOT NULL DEFAULT FALSE,
    created_at         TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    UNIQUE (character_id, name)
);

ALTER TABLE storage_spaces ENABLE ROW LEVEL SECURITY;
CREATE POLICY "allow all" ON storage_spaces FOR ALL USING (true) WITH CHECK (true);

CREATE UNIQUE INDEX idx_storage_default_per_character ON storage_spaces(character_id) WHERE is_default = TRUE;
CREATE INDEX idx_storage_spaces_character             ON storage_spaces(character_id);

-- ── Shops ─────────────────────────────────────────────────────────────────────

CREATE TABLE shops (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    campaign_id UUID         NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    name        VARCHAR(100) NOT NULL,
    color       VARCHAR(7)   NOT NULL DEFAULT '#6366f1',
    is_active   BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

ALTER TABLE shops ENABLE ROW LEVEL SECURITY;
CREATE POLICY "allow all" ON shops FOR ALL USING (true) WITH CHECK (true);

CREATE INDEX idx_shops_campaign ON shops(campaign_id);

-- ── Shop Items ────────────────────────────────────────────────────────────────

CREATE TABLE shop_items (
    id             UUID           PRIMARY KEY DEFAULT gen_random_uuid(),
    campaign_id    UUID           NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    shop_id        UUID           REFERENCES shops(id) ON DELETE SET NULL,
    name           VARCHAR(100)   NOT NULL,
    description    TEXT,
    emoji          VARCHAR(10),
    weight_kg      NUMERIC(10, 4) NOT NULL DEFAULT 0 CHECK (weight_kg >= 0),
    base_value     NUMERIC(18, 4) NOT NULL DEFAULT 0 CHECK (base_value >= 0),
    value_coin_id  UUID           REFERENCES coin_types(id) ON DELETE SET NULL,
    is_available   BOOLEAN        NOT NULL DEFAULT TRUE,
    stock_quantity INT            CHECK (stock_quantity IS NULL OR stock_quantity >= 0),
    created_at     TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ    NOT NULL DEFAULT NOW()
);

ALTER TABLE shop_items ENABLE ROW LEVEL SECURITY;
CREATE POLICY "allow all" ON shop_items FOR ALL USING (true) WITH CHECK (true);

CREATE INDEX idx_shop_items_campaign  ON shop_items(campaign_id);
CREATE INDEX idx_shop_items_available ON shop_items(campaign_id, is_available);
CREATE INDEX idx_shop_items_shop      ON shop_items(shop_id);

CREATE TABLE shop_item_categories (
    shop_item_id UUID NOT NULL REFERENCES shop_items(id) ON DELETE CASCADE,
    category_id  UUID NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    PRIMARY KEY (shop_item_id, category_id)
);

ALTER TABLE shop_item_categories ENABLE ROW LEVEL SECURITY;
CREATE POLICY "allow all" ON shop_item_categories FOR ALL USING (true) WITH CHECK (true);

-- ── Items ─────────────────────────────────────────────────────────────────────

CREATE TABLE items (
    id               UUID           PRIMARY KEY DEFAULT gen_random_uuid(),
    character_id     UUID           NOT NULL REFERENCES characters(id)    ON DELETE CASCADE,
    storage_space_id UUID           REFERENCES storage_spaces(id)         ON DELETE SET NULL,
    shop_item_id     UUID           REFERENCES shop_items(id)             ON DELETE SET NULL,
    name             VARCHAR(100)   NOT NULL,
    description      TEXT,
    emoji            VARCHAR(10),
    weight_kg        NUMERIC(10, 4) NOT NULL DEFAULT 0 CHECK (weight_kg >= 0),
    value            NUMERIC(18, 4) NOT NULL DEFAULT 0 CHECK (value >= 0),
    value_coin_id    UUID           REFERENCES coin_types(id) ON DELETE SET NULL,
    quantity         INT            NOT NULL DEFAULT 1 CHECK (quantity > 0),
    created_at       TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ    NOT NULL DEFAULT NOW()
);

ALTER TABLE items ENABLE ROW LEVEL SECURITY;
CREATE POLICY "allow all" ON items FOR ALL USING (true) WITH CHECK (true);

CREATE INDEX idx_items_character     ON items(character_id);
CREATE INDEX idx_items_storage_space ON items(storage_space_id);

CREATE TABLE item_categories (
    item_id     UUID NOT NULL REFERENCES items(id)      ON DELETE CASCADE,
    category_id UUID NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    PRIMARY KEY (item_id, category_id)
);

ALTER TABLE item_categories ENABLE ROW LEVEL SECURITY;
CREATE POLICY "allow all" ON item_categories FOR ALL USING (true) WITH CHECK (true);

-- Resolve circular dependency: storage_spaces ↔ items
ALTER TABLE storage_spaces
    ADD COLUMN item_id UUID REFERENCES items(id) ON DELETE SET NULL;

CREATE INDEX idx_storage_spaces_item ON storage_spaces(item_id) WHERE item_id IS NOT NULL;

-- ── Coin Purse ────────────────────────────────────────────────────────────────

CREATE TABLE coin_purse (
    id           UUID           PRIMARY KEY DEFAULT gen_random_uuid(),
    character_id UUID           NOT NULL REFERENCES characters(id) ON DELETE CASCADE,
    coin_type_id UUID           NOT NULL REFERENCES coin_types(id) ON DELETE CASCADE,
    amount       NUMERIC(18, 4) NOT NULL DEFAULT 0 CHECK (amount >= 0),
    updated_at   TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    UNIQUE (character_id, coin_type_id)
);

ALTER TABLE coin_purse ENABLE ROW LEVEL SECURITY;
CREATE POLICY "allow all" ON coin_purse FOR ALL USING (true) WITH CHECK (true);

CREATE INDEX idx_coin_purse_character ON coin_purse(character_id);

-- ── Transactions ──────────────────────────────────────────────────────────────

CREATE TABLE transactions (
    id             UUID               PRIMARY KEY DEFAULT gen_random_uuid(),
    campaign_id    UUID               NOT NULL REFERENCES campaigns(id)  ON DELETE CASCADE,
    character_id   UUID               NOT NULL REFERENCES characters(id) ON DELETE CASCADE,
    type           transaction_type   NOT NULL,
    status         transaction_status NOT NULL DEFAULT 'draft',
    original_total NUMERIC(18, 4)     NOT NULL,
    adjusted_total NUMERIC(18, 4)     NOT NULL,
    total_coin_id  UUID               NOT NULL REFERENCES coin_types(id),
    notes          TEXT,
    created_by     UUID               NOT NULL REFERENCES users(id),
    created_at     TIMESTAMPTZ        NOT NULL DEFAULT NOW(),
    confirmed_at   TIMESTAMPTZ
);

CREATE INDEX idx_transactions_campaign  ON transactions(campaign_id);
CREATE INDEX idx_transactions_character ON transactions(character_id);
CREATE INDEX idx_transactions_status    ON transactions(status);

CREATE TABLE transaction_items (
    id                  UUID           PRIMARY KEY DEFAULT gen_random_uuid(),
    transaction_id      UUID           NOT NULL REFERENCES transactions(id) ON DELETE CASCADE,
    shop_item_id        UUID,
    inventory_item_id   UUID,
    name                VARCHAR(100)   NOT NULL,
    quantity            INT            NOT NULL CHECK (quantity > 0),
    unit_value          NUMERIC(18, 4) NOT NULL,
    adjusted_unit_value NUMERIC(18, 4) NOT NULL,
    coin_id             UUID           NOT NULL REFERENCES coin_types(id)
);

CREATE INDEX idx_tx_items_transaction ON transaction_items(transaction_id);
