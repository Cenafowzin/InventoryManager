CREATE TABLE shops (
    id          UUID         PRIMARY KEY DEFAULT uuid_generate_v4(),
    campaign_id UUID         NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    name        VARCHAR(100) NOT NULL,
    color       VARCHAR(7)   NOT NULL DEFAULT '#6366f1',
    is_active   BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_shops_campaign ON shops(campaign_id);

ALTER TABLE shops ENABLE ROW LEVEL SECURITY;
CREATE POLICY "allow all" ON shops FOR ALL USING (true) WITH CHECK (true);

ALTER TABLE shop_items
    ADD COLUMN shop_id        UUID REFERENCES shops(id) ON DELETE SET NULL,
    ADD COLUMN stock_quantity INT  CHECK (stock_quantity IS NULL OR stock_quantity >= 0);

CREATE INDEX idx_shop_items_shop ON shop_items(shop_id);
