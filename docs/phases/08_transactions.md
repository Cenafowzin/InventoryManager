# Fase 8 — Transações (Compra e Venda)

## Objetivo
Sistema de transações com fluxo `draft → ajuste de preços → confirm`. Suporta compra de itens da loja e venda de itens do inventário, com ajuste individual e total antes de confirmar.

## Pré-requisito
Fases 6 (inventário) e 7 (loja) concluídas.

## Entregável testável
- Iniciar transação → recebe rascunho com preços calculados
- Ajustar preço individual de um item ou o total geral
- Confirmar → itens e moedas movimentados atomicamente
- Cancelar → sem efeito
- Transação confirmada/cancelada é imutável

---

## Passos

### 8.1 — Migration

**`008_create_transactions.up.sql`**

```sql
CREATE TYPE transaction_type   AS ENUM ('buy', 'sell');
CREATE TYPE transaction_status AS ENUM ('draft', 'confirmed', 'cancelled');

CREATE TABLE transactions (
    id               UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    campaign_id      UUID               NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    character_id     UUID               NOT NULL REFERENCES characters(id) ON DELETE CASCADE,
    type             transaction_type   NOT NULL,
    status           transaction_status NOT NULL DEFAULT 'draft',
    original_total   NUMERIC(18, 4)     NOT NULL,
    adjusted_total   NUMERIC(18, 4)     NOT NULL,
    total_coin_id    UUID               NOT NULL REFERENCES coin_types(id),
    notes            TEXT,
    created_by       UUID               NOT NULL REFERENCES users(id),
    created_at       TIMESTAMPTZ        NOT NULL DEFAULT NOW(),
    confirmed_at     TIMESTAMPTZ
);

CREATE INDEX idx_transactions_campaign   ON transactions(campaign_id);
CREATE INDEX idx_transactions_character  ON transactions(character_id);
CREATE INDEX idx_transactions_status     ON transactions(status);

CREATE TABLE transaction_items (
    id                    UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    transaction_id        UUID           NOT NULL REFERENCES transactions(id) ON DELETE CASCADE,
    shop_item_id          UUID,          -- compra: referência ao ShopItem
    inventory_item_id     UUID,          -- venda: referência ao Item do inventário
    name                  VARCHAR(100)   NOT NULL,  -- snapshot
    quantity              INT            NOT NULL CHECK (quantity > 0),
    unit_value            NUMERIC(18, 4) NOT NULL,  -- valor original unitário
    adjusted_unit_value   NUMERIC(18, 4) NOT NULL,  -- valor final unitário
    coin_id               UUID           NOT NULL REFERENCES coin_types(id)
);

CREATE INDEX idx_tx_items_transaction ON transaction_items(transaction_id);
```

**`008_create_transactions.down.sql`**

```sql
DROP TABLE IF EXISTS transaction_items;
DROP TABLE IF EXISTS transactions;
DROP TYPE  IF EXISTS transaction_status;
DROP TYPE  IF EXISTS transaction_type;
```

---

### 8.2 — Modelos

**`models/transaction.go`**

```go
type Transaction struct {
    ID            uuid.UUID
    CampaignID    uuid.UUID
    CharacterID   uuid.UUID
    CharacterName string    // join
    Type          string    // "buy" | "sell"
    Status        string    // "draft" | "confirmed" | "cancelled"
    OriginalTotal float64
    AdjustedTotal float64
    TotalCoinID   uuid.UUID
    TotalCoin     string    // abreviação
    Notes         string
    CreatedBy     uuid.UUID
    CreatedAt     time.Time
    ConfirmedAt   *time.Time
    Items         []TransactionItem
}

type TransactionItem struct {
    ID                 uuid.UUID
    TransactionID      uuid.UUID
    ShopItemID         *uuid.UUID
    InventoryItemID    *uuid.UUID
    Name               string
    Quantity           int
    UnitValue          float64
    AdjustedUnitValue  float64
    LineTotal          float64    // calculado: adjusted_unit_value * quantity
    CoinID             uuid.UUID
    CoinAbbreviation   string
}
```

---

### 8.3 — Repositório

**`internal/transaction/repository.go`**

```
CreateTransaction(ctx, tx *Transaction) (*Transaction, error)
GetTransactionByID(ctx, id) (*Transaction, error)
ListTransactions(ctx, campaignID, filters) ([]Transaction, error)
UpdateTransactionItems(ctx, txID, items []TransactionItem) error
UpdateTransactionTotals(ctx, txID, adjustedTotal float64) error
UpdateTransactionStatus(ctx, txID, status string, confirmedAt *time.Time) error
UpdateTransactionNotes(ctx, txID, notes string) error
```

---

### 8.4 — Serviço

**`internal/transaction/service.go`**

---

#### `CreateTransaction(ctx, campaignID, requesterID, requesterRole, input)`

`input`:
```go
type CreateTransactionInput struct {
    CharacterID uuid.UUID
    Type        string  // "buy" | "sell"
    TotalCoinID uuid.UUID  // opcional, usa padrão da campanha
    Items []struct {
        ShopItemID      *uuid.UUID  // para compra
        InventoryItemID *uuid.UUID  // para venda
        Quantity        int
    }
}
```

**Passos:**
1. Validar acesso ao personagem (ownership/gm)
2. Para cada item de **compra**: buscar `ShopItem`, validar `is_available`, calcular `unit_value = base_value`
3. Para cada item de **venda**: buscar `Item` do inventário, validar que pertence ao personagem, calcular `unit_value = item.value`
4. Calcular `original_total = SUM(unit_value * quantity)` — converter para `TotalCoinID` se moedas diferirem
5. Criar `Transaction` com `status = "draft"`, `adjusted_total = original_total`
6. Criar `TransactionItems` com `adjusted_unit_value = unit_value` (inicial)
7. Retornar transação completa com todas as linhas

---

#### `AdjustTransaction(ctx, txID, requesterID, requesterRole, patch)`

`patch`:
```go
type AdjustTransactionPatch struct {
    AdjustedTotal *float64   // altera total geral (redistribui proporcionalmente)
    Notes        *string
    Items        []struct {  // altera itens individualmente
        ItemID            uuid.UUID
        AdjustedUnitValue float64
    }
}
```

**Regras:**
- Transação deve estar em `draft`
- Apenas GM ou owner do personagem pode ajustar
- Se `AdjustedTotal` for enviado:
  - Calcular fator: `factor = adjusted_total / original_total`
  - Aplicar: `item.adjusted_unit_value = item.unit_value * factor` para todos os itens
- Se itens individuais forem enviados:
  - Atualizar `adjusted_unit_value` de cada item listado
  - Recalcular `adjusted_total = SUM(adjusted_unit_value * quantity)`
- Não podem ser enviados `AdjustedTotal` e itens individuais na mesma requisição
- `adjusted_unit_value` pode ser `0` (item de graça por roleplay)

---

#### `ConfirmTransaction(ctx, txID, requesterID, requesterRole)`

**Passos (dentro de uma única transação SQL):**

**Para compra:**
1. Verificar saldo: `SUM(adjusted_unit_value * quantity)` convertido para a moeda do personagem ≥ saldo disponível
2. Para cada `TransactionItem`:
   - Inserir em `items` o item no inventário do personagem (`shop_item_id` preenchido, `storage_space_id = null` → vai para o espaço Geral, `emoji` copiado do ShopItem)
   - Copiar categorias do `ShopItem` para o novo `Item` via `SetItemCategories`
3. Debitar `adjusted_total` da moeda do personagem (`coin_purse`)
4. Atualizar `status = "confirmed"`, `confirmed_at = NOW()`

**Para venda:**
1. Verificar que cada item ainda existe no inventário com a quantidade necessária
2. Para cada `TransactionItem`:
   - Decrementar `quantity` do item; se `quantity = 0`, deletar o item
3. Creditar `adjusted_total` na moeda do personagem
4. Atualizar `status = "confirmed"`, `confirmed_at = NOW()`

---

#### `CancelTransaction(ctx, txID, requesterID, requesterRole)`

1. Transação deve estar em `draft`
2. Atualizar `status = "cancelled"`
3. Nenhuma movimentação de itens ou moedas

---

### 8.5 — Handlers

**`internal/transaction/handler.go`**

`POST /campaigns/:cID/transactions`
- Role: `player` (owner) ou `gm`
- Body: `{ "type": "buy"|"sell", "character_id", "total_coin_id"?, "items": [{shop_item_id ou inventory_item_id, quantity}] }`
- Resposta `201`: transação em draft com totais calculados

`GET /campaigns/:cID/transactions`
- Role: `player` (vê só as suas), `gm` (vê todas)
- Query params: `?character_id=`, `?status=`, `?type=`
- Resposta `200`: lista resumida (sem itens, só totais)

`GET /campaigns/:cID/transactions/:txID`
- Role: owner ou gm
- Resposta `200`: transação completa com todas as linhas e totais

`PATCH /campaigns/:cID/transactions/:txID`
- Role: owner ou gm
- Body:
  ```json
  // Opção A: ajustar total
  { "adjusted_total": 120.00, "notes": "Desconto de roleplay" }

  // Opção B: ajustar itens individuais
  { "items": [{ "id": "...", "adjusted_unit_value": 80.0 }] }
  ```
- Resposta `200`: transação atualizada

`POST /campaigns/:cID/transactions/:txID/confirm`
- Role: owner ou gm
- Sem body
- Resposta `200`: transação confirmada + resumo da movimentação

`POST /campaigns/:cID/transactions/:txID/cancel`
- Role: owner ou gm
- Sem body
- Resposta `200`: transação cancelada

---

### 8.6 — Estrutura de pastas desta fase

```
internal/
└── transaction/
    ├── handler.go
    ├── service.go
    └── repository.go
models/
└── transaction.go
```

---

## Testes Manuais

```bash
TOKEN_GM="<token gm>"
TOKEN_PLAYER="<token player>"
BASE="http://localhost:8080/campaigns/$CAMPAIGN_ID"
CHAR_ID="<uuid personagem>"
SWORD_SHOP_ID="<uuid espada na loja>"
POTION_SHOP_ID="<uuid poção na loja>"
INVENTORY_ITEM_ID="<uuid item no inventário do personagem>"

# --- FLUXO DE COMPRA ---

# 1. Iniciar compra (draft)
curl -X POST $BASE/transactions \
  -H "Authorization: Bearer $TOKEN_PLAYER" \
  -H "Content-Type: application/json" \
  -d "{
    \"type\": \"buy\",
    \"character_id\": \"$CHAR_ID\",
    \"items\": [
      {\"shop_item_id\": \"$SWORD_SHOP_ID\", \"quantity\": 1},
      {\"shop_item_id\": \"$POTION_SHOP_ID\", \"quantity\": 3}
    ]
  }"

TX_ID="<uuid da transação>"

# 2a. GM aplica desconto no total (ex: 100 → 80)
curl -X PATCH $BASE/transactions/$TX_ID \
  -H "Authorization: Bearer $TOKEN_GM" \
  -H "Content-Type: application/json" \
  -d '{"adjusted_total": 80.0, "notes": "Desconto por missão anterior"}'

# OU 2b. GM ajusta preço individual de item
curl -X PATCH $BASE/transactions/$TX_ID \
  -H "Authorization: Bearer $TOKEN_GM" \
  -H "Content-Type: application/json" \
  -d "{\"items\": [{\"id\": \"<tx_item_id>\", \"adjusted_unit_value\": 70.0}]}"

# 3. Ver transação antes de confirmar
curl $BASE/transactions/$TX_ID \
  -H "Authorization: Bearer $TOKEN_PLAYER"

# 4. Confirmar compra
curl -X POST $BASE/transactions/$TX_ID/confirm \
  -H "Authorization: Bearer $TOKEN_PLAYER"
# Esperado: itens criados no inventário, GP debitado

# Verificar inventário
curl http://localhost:8080/campaigns/$CAMPAIGN_ID/characters/$CHAR_ID/inventory \
  -H "Authorization: Bearer $TOKEN_PLAYER"

# --- FLUXO DE VENDA ---

# 1. Iniciar venda de item do inventário
curl -X POST $BASE/transactions \
  -H "Authorization: Bearer $TOKEN_PLAYER" \
  -H "Content-Type: application/json" \
  -d "{
    \"type\": \"sell\",
    \"character_id\": \"$CHAR_ID\",
    \"items\": [
      {\"inventory_item_id\": \"$INVENTORY_ITEM_ID\", \"quantity\": 1}
    ]
  }"

TX_SELL_ID="<uuid>"

# 2. Confirmar sem ajuste
curl -X POST $BASE/transactions/$TX_SELL_ID/confirm \
  -H "Authorization: Bearer $TOKEN_PLAYER"
# Esperado: item removido do inventário, GP creditado

# --- ERROS ESPERADOS ---

# Comprar sem saldo suficiente (deve retornar 422)
# Ajustar transação já confirmada (deve retornar 409)
curl -X PATCH $BASE/transactions/$TX_ID \
  -H "Authorization: Bearer $TOKEN_GM" \
  -H "Content-Type: application/json" \
  -d '{"adjusted_total": 50}'

# Enviar total E itens individuais no mesmo PATCH (deve retornar 400)
curl -X PATCH $BASE/transactions/$TX_ID \
  -H "Authorization: Bearer $TOKEN_GM" \
  -H "Content-Type: application/json" \
  -d "{\"adjusted_total\": 50, \"items\": [{\"id\": \"...\", \"adjusted_unit_value\": 10}]}"
```

---

## Critérios de aceite

**Draft:**
- [ ] Transação criada com `status = "draft"` e totais calculados corretamente
- [ ] Item de loja indisponível (`is_available = false`) não pode ser comprado

**Ajuste:**
- [ ] Ajustar `adjusted_total` redistribui proporcionalmente entre os itens
- [ ] Ajustar itens individuais recalcula `adjusted_total`
- [ ] Enviar `adjusted_total` + itens individuais no mesmo PATCH retorna `400`
- [ ] `adjusted_unit_value = 0` é aceito (item de graça)
- [ ] Transação `confirmed` ou `cancelled` não pode ser ajustada (`409`)

**Confirmação — Compra:**
- [ ] Saldo insuficiente retorna `422`
- [ ] Itens criados no inventário com `shop_item_id` preenchido
- [ ] Moedas debitadas pelo `adjusted_total`
- [ ] Operação é atômica (falha parcial reverte tudo)

**Confirmação — Venda:**
- [ ] Item não encontrado no inventário retorna `422`
- [ ] Quantidade vendida > quantidade disponível retorna `422`
- [ ] Item com `quantity = 0` após venda é deletado do inventário
- [ ] Moedas creditadas pelo `adjusted_total`

**Cancelamento:**
- [ ] Cancelar em `draft` → sem movimentação
- [ ] Cancelar transação `confirmed` retorna `409`

**Listagem:**
- [ ] Player vê só as transações dos seus personagens
- [ ] GM vê todas as transações da campanha
