# Fase 7 — Loja da Campanha

## Objetivo
GM gerencia um catálogo de itens genéricos por campanha, com categorias. Players consultam a loja. A compra/venda é tratada na Fase 8 (Transações).

## Pré-requisito
Fase 6 concluída (categorias já existem).

## Entregável testável
- GM cria, edita, oculta e remove itens da loja com categorias
- Players listam apenas itens disponíveis, filtráveis por categoria
- GM vê todos (inclusive ocultos)

---

## Passos

### 7.1 — Migration

**`007_create_shop.up.sql`**

```sql
CREATE TABLE shop_items (
    id             UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    campaign_id    UUID           NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    name           VARCHAR(100)   NOT NULL,
    description    TEXT,
    emoji          VARCHAR(10),   -- ex: "⚔️", "🧪", "🛡️"
    weight_kg      NUMERIC(10, 4) NOT NULL DEFAULT 0 CHECK (weight_kg >= 0),
    base_value     NUMERIC(18, 4) NOT NULL DEFAULT 0 CHECK (base_value >= 0),
    value_coin_id  UUID           REFERENCES coin_types(id) ON DELETE SET NULL,
    is_available   BOOLEAN        NOT NULL DEFAULT TRUE,
    created_at     TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ    NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_shop_items_campaign   ON shop_items(campaign_id);
CREATE INDEX idx_shop_items_available  ON shop_items(campaign_id, is_available);

-- Adicionar FK na tabela shop_item_categories criada na Fase 6
ALTER TABLE shop_item_categories
    ADD CONSTRAINT fk_shop_item_categories_shop_item
    FOREIGN KEY (shop_item_id) REFERENCES shop_items(id) ON DELETE CASCADE;
```

**`007_create_shop.down.sql`**

```sql
DROP TABLE IF EXISTS shop_items;
```

---

### 7.2 — Modelo

**`models/shop_item.go`**

```go
type ShopItem struct {
    ID           uuid.UUID
    CampaignID   uuid.UUID
    Name         string
    Description  string
    Emoji        string      // opcional, ex: "⚔️"
    WeightKg     float64
    BaseValue    float64
    ValueCoinID  uuid.UUID
    ValueCoin    string      // abreviação (join)
    IsAvailable  bool
    Categories   []Category  // join via shop_item_categories
    CreatedAt    time.Time
    UpdatedAt    time.Time
}
```

---

### 7.3 — Repositório

**`internal/shop/repository.go`**

```
CreateShopItem(ctx, campaignID, input) (*ShopItem, error)
GetShopItemByID(ctx, id) (*ShopItem, error)
ListShopItems(ctx, campaignID, filters ShopFilters) ([]ShopItem, error)
UpdateShopItem(ctx, id, input) (*ShopItem, error)
DeleteShopItem(ctx, id) error
```

`input`: struct com `Name`, `Description`, `WeightKg`, `BaseValue`, `ValueCoinID`, `IsAvailable`, `CategoryIDs`.

```go
type ShopFilters struct {
    OnlyAvailable bool
    CategoryID    *uuid.UUID
}
```

---

### 7.4 — Serviço

**`internal/shop/service.go`**

**`CreateShopItem(ctx, campaignID, requesterRole, input)`**:
1. Apenas `gm` pode criar
2. Se `input.ValueCoinID` vazio → buscar moeda padrão da campanha
3. Validar que `value_coin_id` pertence à campanha
4. Validar que `category_ids` pertencem à campanha
5. `repo.CreateShopItem` + `category.SetShopItemCategories`

**`ListShopItems(ctx, campaignID, requesterRole)`**:
- `gm`: `onlyAvailable = false` (vê tudo)
- `player`: `onlyAvailable = true`

**`UpdateShopItem(ctx, itemID, requesterRole, input)`**:
- Apenas `gm`
- `is_available = false` oculta sem deletar

**`DeleteShopItem(ctx, itemID, requesterRole)`**:
- Apenas `gm`
- Verificar se há transações confirmadas referenciando este item (só alerta, não bloqueia — dados históricos de `TransactionItem` usam snapshot)

---

### 7.5 — Handlers

**`internal/shop/handler.go`**

`POST /campaigns/:campaignID/shop`
- Role: `gm`
- Body: `{ "name", "description", "emoji"?, "weight_kg", "weight_unit"?, "base_value", "value_coin_id"?, "is_available"?, "category_ids"? }`
- Resposta `201`: ShopItem criado com categorias

`GET /campaigns/:campaignID/shop`
- Role: `player` (GM vê todos, player vê só disponíveis — via serviço)
- Query params: `?include_unavailable=true` (só GM) e `?category_id=` (filtro por categoria)
- Resposta `200`: lista de shop items com categorias

`GET /campaigns/:campaignID/shop/:shopItemID`
- Role: `player`
- Player recebe `404` se item estiver indisponível
- GM recebe o item independente do status

`PUT /campaigns/:campaignID/shop/:shopItemID`
- Role: `gm`
- Body: qualquer subconjunto dos campos editáveis
- Resposta `200`: ShopItem atualizado

`DELETE /campaigns/:campaignID/shop/:shopItemID`
- Role: `gm`
- Resposta `204`

---

### 7.6 — Estrutura de pastas desta fase

```
internal/
└── shop/
    ├── handler.go
    ├── service.go
    └── repository.go
models/
└── shop_item.go
```

---

## Testes Manuais

```bash
TOKEN_GM="<token do gm>"
TOKEN_PLAYER="<token do player>"
CAMPAIGN_ID="<uuid>"

# GM cria itens na loja
curl -X POST http://localhost:8080/campaigns/$CAMPAIGN_ID/shop \
  -H "Authorization: Bearer $TOKEN_GM" \
  -H "Content-Type: application/json" \
  -d '{"name":"Espada Longa","description":"Aço comum","weight_kg":1.5,"base_value":100}'

curl -X POST http://localhost:8080/campaigns/$CAMPAIGN_ID/shop \
  -H "Authorization: Bearer $TOKEN_GM" \
  -H "Content-Type: application/json" \
  -d '{"name":"Poção de Cura","weight_kg":0.2,"base_value":25}'

curl -X POST http://localhost:8080/campaigns/$CAMPAIGN_ID/shop \
  -H "Authorization: Bearer $TOKEN_GM" \
  -H "Content-Type: application/json" \
  -d '{"name":"Item Secreto","weight_kg":0.1,"base_value":999,"is_available":false}'

ITEM_ID="<uuid da espada>"
SECRET_ID="<uuid do item secreto>"

# Player lista loja (não deve ver o item secreto)
curl http://localhost:8080/campaigns/$CAMPAIGN_ID/shop \
  -H "Authorization: Bearer $TOKEN_PLAYER"

# GM lista loja (deve ver todos, inclusive o secreto)
curl "http://localhost:8080/campaigns/$CAMPAIGN_ID/shop?include_unavailable=true" \
  -H "Authorization: Bearer $TOKEN_GM"

# Player tenta acessar item indisponível diretamente (deve retornar 404)
curl http://localhost:8080/campaigns/$CAMPAIGN_ID/shop/$SECRET_ID \
  -H "Authorization: Bearer $TOKEN_PLAYER"

# GM torna item disponível
curl -X PUT http://localhost:8080/campaigns/$CAMPAIGN_ID/shop/$SECRET_ID \
  -H "Authorization: Bearer $TOKEN_GM" \
  -H "Content-Type: application/json" \
  -d '{"is_available":true}'

# Player tenta criar item na loja (deve retornar 403)
curl -X POST http://localhost:8080/campaigns/$CAMPAIGN_ID/shop \
  -H "Authorization: Bearer $TOKEN_PLAYER" \
  -H "Content-Type: application/json" \
  -d '{"name":"Item do Player","base_value":1}'
```

---

## Critérios de aceite

- [ ] Item criado sem `value_coin_id` usa moeda padrão da campanha
- [ ] Player não vê itens com `is_available = false`
- [ ] Player recebe `404` ao acessar diretamente item indisponível
- [ ] GM vê todos os itens com `?include_unavailable=true`
- [ ] `is_available = false` não deleta o item (apenas oculta)
- [ ] Player não pode criar/editar/deletar itens da loja (`403`)
- [ ] `weight_unit: "lbs"` converte corretamente ao criar
- [ ] `?category_id=` filtra corretamente para player e gm
- [ ] `category_ids` com ID de categoria de outra campanha retorna erro
- [ ] Ao editar com `category_ids`, substitui todas as categorias anteriores
