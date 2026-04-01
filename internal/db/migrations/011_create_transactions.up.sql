CREATE TYPE transaction_type   AS ENUM ('buy', 'sell');
CREATE TYPE transaction_status AS ENUM ('draft', 'confirmed', 'cancelled');

CREATE TABLE transactions (
    id             UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    campaign_id    UUID               NOT NULL REFERENCES campaigns(id)   ON DELETE CASCADE,
    character_id   UUID               NOT NULL REFERENCES characters(id)  ON DELETE CASCADE,
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
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
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
