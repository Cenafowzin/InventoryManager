# Fase 6 — Inventário (Itens, Moedas, Categorias e Armazenamento)

## Objetivo
Gerenciar o inventário completo do personagem: itens com categorias, espaços de armazenamento com controle de carga, saldo de moedas com conversão e endpoint de resumo com cálculo de carga.

## Pré-requisito
Fases 4 (moedas) e 5 (personagens) concluídas.

## Entregável testável
- CRUD de categorias (tags) por campanha
- CRUD de espaços de armazenamento por personagem
- CRUD de itens com categorias e espaço de armazenamento
- Filtro de itens por categoria e por espaço
- Consultar e ajustar saldo de moedas, converter entre moedas
- Endpoint `/inventory` com resumo completo incluindo carga
- Endpoint `/load` com detalhamento de carga por espaço

---

## Passos

### 6.1 — Migration

**`006_create_inventory.up.sql`**

```sql
-- Categorias da campanha
CREATE TABLE categories (
    id           UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    campaign_id  UUID         NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    name         VARCHAR(50)  NOT NULL,
    color        VARCHAR(7),  -- hex: "#FF5733" (opcional)
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    UNIQUE (campaign_id, name)
);

CREATE INDEX idx_categories_campaign ON categories(campaign_id);

-- Espaços de armazenamento do personagem
CREATE TABLE storage_spaces (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    character_id        UUID           NOT NULL REFERENCES characters(id) ON DELETE CASCADE,
    name                VARCHAR(100)   NOT NULL,
    description         TEXT,
    counts_toward_load  BOOLEAN        NOT NULL DEFAULT TRUE,
    capacity_kg         NUMERIC(10, 4),  -- NULL = sem limite
    is_default          BOOLEAN        NOT NULL DEFAULT FALSE,  -- espaço "Geral"
    created_at          TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    UNIQUE (character_id, name)
);

-- Garante que só um espaço por personagem seja o default
CREATE UNIQUE INDEX idx_storage_default_per_character
    ON storage_spaces(character_id)
    WHERE is_default = TRUE;

CREATE INDEX idx_storage_spaces_character ON storage_spaces(character_id);

-- Itens do inventário
CREATE TABLE items (
    id                UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    character_id      UUID           NOT NULL REFERENCES characters(id) ON DELETE CASCADE,
    storage_space_id  UUID           REFERENCES storage_spaces(id) ON DELETE SET NULL,
    name              VARCHAR(100)   NOT NULL,
    description       TEXT,
    emoji             VARCHAR(10),   -- ex: "⚔️", "🧪", "🛡️"
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

-- N:N itens ↔ categorias
CREATE TABLE item_categories (
    item_id      UUID NOT NULL REFERENCES items(id) ON DELETE CASCADE,
    category_id  UUID NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    PRIMARY KEY (item_id, category_id)
);

-- N:N shop_items ↔ categorias (usado na Fase 7, criado aqui para evitar migration extra)
CREATE TABLE shop_item_categories (
    shop_item_id  UUID NOT NULL,  -- FK adicionada na Fase 7
    category_id   UUID NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    PRIMARY KEY (shop_item_id, category_id)
);

-- Moedas do personagem
CREATE TABLE coin_purse (
    id             UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    character_id   UUID           NOT NULL REFERENCES characters(id) ON DELETE CASCADE,
    coin_type_id   UUID           NOT NULL REFERENCES coin_types(id) ON DELETE CASCADE,
    amount         NUMERIC(18, 4) NOT NULL DEFAULT 0 CHECK (amount >= 0),
    updated_at     TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    UNIQUE (character_id, coin_type_id)
);

CREATE INDEX idx_coin_purse_character ON coin_purse(character_id);
```

**`006_create_inventory.down.sql`**

```sql
DROP TABLE IF EXISTS shop_item_categories;
DROP TABLE IF EXISTS item_categories;
DROP TABLE IF EXISTS coin_purse;
DROP TABLE IF EXISTS items;
DROP TABLE IF EXISTS storage_spaces;
DROP TABLE IF EXISTS categories;
```

---

### 6.2 — Atualizar migration de characters

> Adicionar `max_carry_weight_kg` à tabela `characters` (pode ser uma migration separada `006b` ou incluída na `005`).

```sql
-- 006b_add_carry_weight_to_characters.up.sql
ALTER TABLE characters
    ADD COLUMN max_carry_weight_kg NUMERIC(10, 4);  -- NULL = sem limite
```

```sql
-- 006b_add_carry_weight_to_characters.down.sql
ALTER TABLE characters DROP COLUMN IF EXISTS max_carry_weight_kg;
```

---

### 6.3 — Modelos

**`models/category.go`**

```go
type Category struct {
    ID         uuid.UUID
    CampaignID uuid.UUID
    Name       string
    Color      string    // "#FF5733" ou ""
    CreatedAt  time.Time
}
```

**`models/storage_space.go`**

```go
type StorageSpace struct {
    ID                uuid.UUID
    CharacterID       uuid.UUID
    Name              string
    Description       string
    CountsTowardLoad  bool
    CapacityKg        *float64  // nil = sem limite
    IsDefault         bool
    CurrentWeightKg   float64   // calculado na query
    CreatedAt         time.Time
}
```

**`models/item.go`**

```go
type Item struct {
    ID              uuid.UUID
    CharacterID     uuid.UUID
    StorageSpaceID  *uuid.UUID
    StorageSpace    string      // nome do espaço (join)
    Name            string
    Description     string
    Emoji           string      // opcional, ex: "⚔️"
    WeightKg        float64
    Value           float64
    ValueCoinID     uuid.UUID
    ValueCoin       string
    Quantity        int
    ShopItemID      *uuid.UUID
    Categories      []Category  // carregado via join
    CreatedAt       time.Time
    UpdatedAt       time.Time
}
```

**`models/coin_purse.go`**

```go
type CoinPurse struct {
    CoinTypeID   uuid.UUID
    CoinName     string
    Abbreviation string
    Amount       float64
}
```

---

### 6.4 — Repositório de Categorias

**`internal/category/repository.go`**

```
CreateCategory(ctx, campaignID, name, color) (*Category, error)
GetCategoryByID(ctx, id) (*Category, error)
ListCategories(ctx, campaignID) ([]Category, error)
UpdateCategory(ctx, id, name, color) (*Category, error)
DeleteCategory(ctx, id) error

-- Associações
SetItemCategories(ctx, itemID, categoryIDs []uuid.UUID) error     -- substitui todas
SetShopItemCategories(ctx, shopItemID, categoryIDs []uuid.UUID) error
GetItemCategories(ctx, itemID) ([]Category, error)
```

---

### 6.5 — Serviço de Categorias

**`internal/category/service.go`**

**`CreateCategory(ctx, campaignID, requesterRole, name, color)`**:
- Apenas `gm`
- Validar nome único na campanha

**`DeleteCategory(ctx, id, requesterRole)`**:
- Apenas `gm`
- Remove as associações (`item_categories`, `shop_item_categories`) automaticamente via cascade
- Não remove os itens

---

### 6.6 — Repositório de Espaços de Armazenamento

**`internal/inventory/storage_repository.go`**

```
CreateStorageSpace(ctx, characterID, input) (*StorageSpace, error)
GetStorageSpaceByID(ctx, id) (*StorageSpace, error)
GetDefaultStorageSpace(ctx, characterID) (*StorageSpace, error)
ListStorageSpaces(ctx, characterID) ([]StorageSpace, error)   -- inclui peso atual via SUM
UpdateStorageSpace(ctx, id, input) (*StorageSpace, error)
DeleteStorageSpace(ctx, id) error
ReassignItemsToDefault(ctx, fromSpaceID, toSpaceID) error
```

A query de listagem usa um `LEFT JOIN` com `SUM(items.weight_kg * items.quantity)` para retornar `current_weight_kg` de cada espaço já calculado.

---

### 6.7 — Serviço de Espaços de Armazenamento

**`internal/inventory/storage_service.go`**

**`EnsureDefaultSpace(ctx, characterID)`**:
- Chamado ao criar um personagem
- Cria o espaço "Geral" com `is_default = true`, `counts_toward_load = true`

**`CreateStorageSpace(ctx, characterID, requesterID, requesterRole, input)`**:
1. Validar acesso ao personagem
2. Validar nome único para o personagem
3. `repo.CreateStorageSpace`

**`DeleteStorageSpace(ctx, spaceID, requesterID, requesterRole)`**:
1. Não permitir deletar o espaço default (`is_default = true`) → `ErrCannotDeleteDefault`
2. Buscar espaço default do personagem
3. `repo.ReassignItemsToDefault(spaceID, defaultSpaceID)`
4. `repo.DeleteStorageSpace(spaceID)`

---

### 6.8 — Repositório de Itens

**`internal/inventory/item_repository.go`**

```
CreateItem(ctx, input) (*Item, error)
GetItemByID(ctx, id) (*Item, error)
ListItemsByCharacter(ctx, characterID, filters) ([]Item, error)
UpdateItem(ctx, id, input) (*Item, error)
DeleteItem(ctx, id) error
DecrementQuantity(ctx, id, qty) error   -- para venda; deleta se qty = 0
```

`filters`:
```go
type ItemFilters struct {
    CategoryID     *uuid.UUID
    StorageSpaceID *uuid.UUID
}
```

`ListItemsByCharacter` retorna itens com categorias via `LEFT JOIN item_categories JOIN categories`.

---

### 6.9 — Serviço de Itens

**`internal/inventory/item_service.go`**

**`CreateItem(ctx, characterID, requesterID, requesterRole, input)`**:
1. Validar acesso ao personagem
2. Se `storage_space_id` omitido → buscar espaço default do personagem
3. Validar que `storage_space_id` pertence ao personagem
4. Se `value_coin_id` omitido → buscar moeda padrão da campanha
5. Validar que `value_coin_id` pertence à campanha
6. Validar que `category_ids` pertencem à campanha
7. `repo.CreateItem` + `repo.SetItemCategories`

**`UpdateItem(ctx, id, requesterID, requesterRole, input)`**:
- Mesma validação de acesso
- Se `category_ids` fornecido → `repo.SetItemCategories` (substitui todas)
- Se `storage_space_id` fornecido → validar que pertence ao personagem

---

### 6.10 — Repositório de Moedas do Personagem

**`internal/inventory/coin_repository.go`**

```
GetCoinPurse(ctx, characterID) ([]CoinPurse, error)
GetCoinBalance(ctx, characterID, coinTypeID) (float64, error)
SetCoinBalance(ctx, characterID, coinTypeID, amount) error   -- UPSERT
AddToBalance(ctx, characterID, coinTypeID, delta) error
```

---

### 6.11 — Serviço de Moedas do Personagem

**`internal/inventory/coin_service.go`**

**`ConvertCoins(ctx, characterID, fromCoinID, toCoinID, amount)`**:
1. Buscar conversão `from → to` em `coin_conversions`
2. Verificar `GetCoinBalance >= amount` → `ErrInsufficientFunds`
3. Calcular `received = amount * rate`
4. Transação SQL:
   - `AddToBalance(characterID, fromCoinID, -amount)`
   - `AddToBalance(characterID, toCoinID, +received)`
5. Retornar novo saldo das duas moedas

---

### 6.12 — Serviço de Resumo e Carga

**`internal/inventory/summary_service.go`**

**`GetLoad(ctx, characterID)`** → retorna:
```json
{
  "max_carry_weight_kg": 50.0,
  "current_load_kg": 18.4,
  "load_percentage": 36.8,
  "is_overloaded": false,
  "spaces": [
    {
      "id": "...",
      "name": "Mochila",
      "counts_toward_load": true,
      "capacity_kg": 20.0,
      "current_weight_kg": 15.2,
      "is_over_capacity": false
    },
    {
      "id": "...",
      "name": "Baú do Acampamento",
      "counts_toward_load": false,
      "capacity_kg": null,
      "current_weight_kg": 42.0,
      "is_over_capacity": false
    }
  ]
}
```

Regras:
- `current_load_kg` = soma de peso apenas dos espaços com `counts_toward_load = true`
- `load_percentage` = `null` se `max_carry_weight_kg` for nulo
- `is_overloaded` = `current_load_kg > max_carry_weight_kg` (false se max for nulo)
- `is_over_capacity` = `current_weight_kg > capacity_kg` (false se capacity for nula)

**`GetInventorySummary(ctx, characterID)`** → retorna load + itens + moedas + `total_value` estimado na moeda padrão.

---

### 6.13 — Handlers

**`internal/category/handler.go`**

| Rota | Método | Body | Resposta | Role |
|------|--------|------|----------|------|
| `/campaigns/:cID/categories` | POST | `{name, color?}` | `201` | gm |
| `/campaigns/:cID/categories` | GET | — | `200` lista | member |
| `/campaigns/:cID/categories/:catID` | PUT | `{name?, color?}` | `200` | gm |
| `/campaigns/:cID/categories/:catID` | DELETE | — | `204` | gm |

**`internal/inventory/handler.go`**

**Espaços de armazenamento:**

| Rota | Método | Body | Resposta | Role |
|------|--------|------|----------|------|
| `/campaigns/:cID/characters/:charID/storages` | POST | `{name, description?, counts_toward_load?, capacity_kg?}` | `201` | owner/gm |
| `/campaigns/:cID/characters/:charID/storages` | GET | — | `200` lista com peso atual | owner/gm |
| `/campaigns/:cID/characters/:charID/storages/:sID` | PUT | campos editáveis | `200` | owner/gm |
| `/campaigns/:cID/characters/:charID/storages/:sID` | DELETE | — | `204` (itens → Geral) | owner/gm |

**Itens:**

| Rota | Método | Body / Query | Resposta | Role |
|------|--------|--------------|----------|------|
| `/campaigns/:cID/characters/:charID/items` | POST | `{name, description?, emoji?, weight_kg, weight_unit?, value, value_coin_id?, storage_space_id?, category_ids?, quantity?, shop_item_id?}` | `201` | owner/gm |
| `/campaigns/:cID/characters/:charID/items` | GET | `?category_id=&storage_id=` | `200` lista com categorias | owner/gm |
| `/campaigns/:cID/characters/:charID/items/:itemID` | GET | — | `200` | owner/gm |
| `/campaigns/:cID/characters/:charID/items/:itemID` | PUT | campos editáveis + `category_ids?` | `200` | owner/gm |
| `/campaigns/:cID/characters/:charID/items/:itemID` | DELETE | — | `204` | owner/gm |

**Moedas:**

| Rota | Método | Body | Resposta | Role |
|------|--------|------|----------|------|
| `/campaigns/:cID/characters/:charID/coins` | GET | — | `200` lista de saldos | owner/gm |
| `/campaigns/:cID/characters/:charID/coins/:coinID` | PUT | `{amount}` | `200` | owner/gm |
| `/campaigns/:cID/characters/:charID/coins/convert` | POST | `{from_coin_id, to_coin_id, amount}` | `200` novos saldos | owner/gm |

**Resumo e Carga:**

| Rota | Método | Resposta | Role |
|------|--------|----------|------|
| `/campaigns/:cID/characters/:charID/inventory` | GET | Resumo completo (itens + moedas + carga + valor total) | owner/gm |
| `/campaigns/:cID/characters/:charID/load` | GET | Carga detalhada por espaço | owner/gm |

---

### 6.14 — Estrutura de pastas desta fase

```
internal/
├── category/
│   ├── handler.go
│   ├── service.go
│   └── repository.go
└── inventory/
    ├── handler.go
    ├── item_service.go
    ├── coin_service.go
    ├── storage_service.go
    ├── summary_service.go
    ├── item_repository.go
    ├── coin_repository.go
    └── storage_repository.go
models/
├── category.go
├── storage_space.go
├── item.go
└── coin_purse.go
```

---

## Testes Manuais

```bash
TOKEN="<token do owner ou gm>"
BASE="http://localhost:8080/campaigns/$CAMPAIGN_ID"
CHAR="$BASE/characters/$CHAR_ID"

# --- CATEGORIAS ---

# GM cria categorias
curl -X POST $BASE/categories \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Arma","color":"#E74C3C"}'

curl -X POST $BASE/categories \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Consumível","color":"#2ECC71"}'

WEAPON_CAT_ID="<uuid>"
POTION_CAT_ID="<uuid>"

# --- ESPAÇOS DE ARMAZENAMENTO ---

# Criar mochila (conta para carga, limite de 15kg)
curl -X POST $CHAR/storages \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Mochila","counts_toward_load":true,"capacity_kg":15}'

# Criar baú (NÃO conta para carga)
curl -X POST $CHAR/storages \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Baú do Acampamento","counts_toward_load":false}'

BACKPACK_ID="<uuid>"
CHEST_ID="<uuid>"

# Listar espaços (deve mostrar peso atual e o espaço "Geral" criado automaticamente)
curl $CHAR/storages -H "Authorization: Bearer $TOKEN"

# --- ITENS ---

# Adicionar espada na mochila com categoria "Arma"
curl -X POST $CHAR/items \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"name\": \"Espada Longa\",
    \"weight_kg\": 1.5,
    \"value\": 100,
    \"storage_space_id\": \"$BACKPACK_ID\",
    \"category_ids\": [\"$WEAPON_CAT_ID\"],
    \"quantity\": 1
  }"

# Adicionar poções (sem storage → vai para Geral)
curl -X POST $CHAR/items \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"name\": \"Poção de Cura\",
    \"weight_kg\": 0.2,
    \"value\": 25,
    \"category_ids\": [\"$POTION_CAT_ID\"],
    \"quantity\": 5
  }"

# Adicionar item em lbs (deve converter para kg)
curl -X POST $CHAR/items \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"name\": \"Armadura de Couro\",
    \"weight_kg\": 15,
    \"weight_unit\": \"lbs\",
    \"value\": 50,
    \"storage_space_id\": \"$BACKPACK_ID\",
    \"category_ids\": [\"$WEAPON_CAT_ID\"],
    \"quantity\": 1
  }"
# 15 lbs = 6.8038 kg

# Filtrar só armas
curl "$CHAR/items?category_id=$WEAPON_CAT_ID" \
  -H "Authorization: Bearer $TOKEN"

# Filtrar itens na mochila
curl "$CHAR/items?storage_id=$BACKPACK_ID" \
  -H "Authorization: Bearer $TOKEN"

# --- CARGA ---

# Ver carga detalhada por espaço
curl $CHAR/load -H "Authorization: Bearer $TOKEN"
# Esperado:
# - Mochila: 1.5 + 6.8 = 8.3 kg (conta para carga, limite 15)
# - Geral: 1.0 kg de poções (conta para carga)
# - Baú: 0 kg (NÃO conta para carga)
# - current_load_kg = 9.3 (apenas espaços que contam)

# Ver inventário completo
curl $CHAR/inventory -H "Authorization: Bearer $TOKEN"

# --- MOEDAS ---

# Definir saldo
curl -X PUT $CHAR/coins/$GP_ID \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"amount":500}'

# Converter 100 GP → SP
curl -X POST $CHAR/coins/convert \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"from_coin_id\":\"$GP_ID\",\"to_coin_id\":\"$SP_ID\",\"amount\":100}"

# --- DELEÇÃO DE ESPAÇO ---

# Deletar mochila (itens devem ir para "Geral")
curl -X DELETE $CHAR/storages/$BACKPACK_ID \
  -H "Authorization: Bearer $TOKEN"

# Verificar que os itens da mochila estão agora no Geral
curl "$CHAR/items?storage_id=<geral_id>" \
  -H "Authorization: Bearer $TOKEN"

# Tentar deletar espaço Geral (deve retornar erro)
curl -X DELETE $CHAR/storages/<geral_id> \
  -H "Authorization: Bearer $TOKEN"
```

---

## Critérios de aceite

**Categorias:**
- [ ] Categoria com nome duplicado na campanha retorna erro
- [ ] Deletar categoria remove associações mas não os itens
- [ ] Player não pode criar/editar/deletar categorias (`403`)
- [ ] `GET /items?category_id=` filtra corretamente
- [ ] Item pode ter zero ou mais categorias

**Espaços de Armazenamento:**
- [ ] Espaço "Geral" criado automaticamente ao criar personagem (`is_default = true`)
- [ ] Espaço "Geral" não pode ser deletado
- [ ] Deletar espaço realoca itens para o Geral
- [ ] Nome de espaço duplicado no personagem retorna erro
- [ ] `GET /storages` retorna `current_weight_kg` calculado para cada espaço
- [ ] `capacity_kg` pode ser nulo (sem limite)

**Itens:**
- [ ] Item sem `storage_space_id` vai para o espaço Geral
- [ ] Item sem `value_coin_id` usa moeda padrão da campanha
- [ ] `weight_unit: "lbs"` converte corretamente ao salvar
- [ ] `PUT /items/:id` com `category_ids` substitui todas as categorias anteriores

**Carga (`GET /load`):**
- [ ] `current_load_kg` soma apenas espaços com `counts_toward_load = true`
- [ ] `load_percentage` é `null` se `max_carry_weight_kg` do personagem for nulo
- [ ] `is_overloaded` é `false` quando não há limite definido
- [ ] `is_over_capacity` é `false` quando `capacity_kg` do espaço é nulo

**Moedas:**
- [ ] Saldo não fica negativo
- [ ] Conversão sem taxa cadastrada retorna erro descritivo
- [ ] Conversão com saldo insuficiente retorna `422`
- [ ] Conversão é atômica
