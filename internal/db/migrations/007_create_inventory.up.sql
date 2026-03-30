-- Categorias da campanha
CREATE TABLE categories (
    id           UUID         PRIMARY KEY DEFAULT uuid_generate_v4(),
    campaign_id  UUID         NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    name         VARCHAR(50)  NOT NULL,
    color        VARCHAR(7),
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    UNIQUE (campaign_id, name)
);

CREATE INDEX idx_categories_campaign ON categories(campaign_id);

ALTER TABLE categories ENABLE ROW LEVEL SECURITY;
CREATE POLICY "allow all" ON categories FOR ALL USING (true) WITH CHECK (true);

-- Espaços de armazenamento do personagem
CREATE TABLE storage_spaces (
    id                  UUID         PRIMARY KEY DEFAULT uuid_generate_v4(),
    character_id        UUID         NOT NULL REFERENCES characters(id) ON DELETE CASCADE,
    name                VARCHAR(100) NOT NULL,
    description         TEXT,
    counts_toward_load  BOOLEAN      NOT NULL DEFAULT TRUE,
    capacity_kg         NUMERIC(10, 4),
    is_default          BOOLEAN      NOT NULL DEFAULT FALSE,
    created_at          TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    UNIQUE (character_id, name)
);

CREATE UNIQUE INDEX idx_storage_default_per_character ON storage_spaces(character_id) WHERE is_default = TRUE;
CREATE INDEX idx_storage_spaces_character ON storage_spaces(character_id);

ALTER TABLE storage_spaces ENABLE ROW LEVEL SECURITY;
CREATE POLICY "allow all" ON storage_spaces FOR ALL USING (true) WITH CHECK (true);

-- Itens do inventário
CREATE TABLE items (
    id                UUID           PRIMARY KEY DEFAULT uuid_generate_v4(),
    character_id      UUID           NOT NULL REFERENCES characters(id) ON DELETE CASCADE,
    storage_space_id  UUID           REFERENCES storage_spaces(id) ON DELETE SET NULL,
    name              VARCHAR(100)   NOT NULL,
    description       TEXT,
    emoji             VARCHAR(10),
    weight_kg         NUMERIC(10, 4) NOT NULL DEFAULT 0 CHECK (weight_kg >= 0),
    value             NUMERIC(18, 4) NOT NULL DEFAULT 0 CHECK (value >= 0),
    value_coin_id     UUID           REFERENCES coin_types(id) ON DELETE SET NULL,
    quantity          INT            NOT NULL DEFAULT 1 CHECK (quantity > 0),
    shop_item_id      UUID,
    created_at        TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ    NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_items_character     ON items(character_id);
CREATE INDEX idx_items_storage_space ON items(storage_space_id);

ALTER TABLE items ENABLE ROW LEVEL SECURITY;
CREATE POLICY "allow all" ON items FOR ALL USING (true) WITH CHECK (true);

-- N:N itens <-> categorias
CREATE TABLE item_categories (
    item_id      UUID NOT NULL REFERENCES items(id) ON DELETE CASCADE,
    category_id  UUID NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    PRIMARY KEY (item_id, category_id)
);

ALTER TABLE item_categories ENABLE ROW LEVEL SECURITY;
CREATE POLICY "allow all" ON item_categories FOR ALL USING (true) WITH CHECK (true);

-- N:N shop_items <-> categorias (FK para shop_items adicionada na Fase 7)
CREATE TABLE shop_item_categories (
    shop_item_id  UUID NOT NULL,
    category_id   UUID NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    PRIMARY KEY (shop_item_id, category_id)
);

ALTER TABLE shop_item_categories ENABLE ROW LEVEL SECURITY;
CREATE POLICY "allow all" ON shop_item_categories FOR ALL USING (true) WITH CHECK (true);

-- Moedas do personagem
CREATE TABLE coin_purse (
    id             UUID           PRIMARY KEY DEFAULT uuid_generate_v4(),
    character_id   UUID           NOT NULL REFERENCES characters(id) ON DELETE CASCADE,
    coin_type_id   UUID           NOT NULL REFERENCES coin_types(id) ON DELETE CASCADE,
    amount         NUMERIC(18, 4) NOT NULL DEFAULT 0 CHECK (amount >= 0),
    updated_at     TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    UNIQUE (character_id, coin_type_id)
);

CREATE INDEX idx_coin_purse_character ON coin_purse(character_id);

ALTER TABLE coin_purse ENABLE ROW LEVEL SECURITY;
CREATE POLICY "allow all" ON coin_purse FOR ALL USING (true) WITH CHECK (true);
