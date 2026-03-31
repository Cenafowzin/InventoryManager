CREATE TABLE shop_items (
    id             UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    campaign_id    UUID           NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    name           VARCHAR(100)   NOT NULL,
    description    TEXT,
    emoji          VARCHAR(10),
    weight_kg      NUMERIC(10, 4) NOT NULL DEFAULT 0 CHECK (weight_kg >= 0),
    base_value     NUMERIC(18, 4) NOT NULL DEFAULT 0 CHECK (base_value >= 0),
    value_coin_id  UUID           REFERENCES coin_types(id) ON DELETE SET NULL,
    is_available   BOOLEAN        NOT NULL DEFAULT TRUE,
    created_at     TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ    NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_shop_items_campaign   ON shop_items(campaign_id);
CREATE INDEX idx_shop_items_available  ON shop_items(campaign_id, is_available);

ALTER TABLE shop_items ENABLE ROW LEVEL SECURITY;
CREATE POLICY "allow all" ON shop_items FOR ALL USING (true) WITH CHECK (true);

--Adicionar FK 
ALTER TABLE shop_item_categories
    ADD CONSTRAINT fk_shop_item_categories_shop_item
    FOREIGN KEY (shop_item_id) REFERENCES shop_items(id) ON DELETE CASCADE;

ALTER TABLE items
    ADD CONSTRAINT fk_items_shop_item
    FOREIGN KEY (shop_item_id) REFERENCES shop_items(id) ON DELETE SET NULL;