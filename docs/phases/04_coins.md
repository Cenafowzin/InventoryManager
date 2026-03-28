# Fase 4 — Tipos de Moeda e Conversões

## Objetivo
CRUD de tipos de moeda por campanha, com conversão bidirecional automática e definição de moeda padrão.

## Pré-requisito
Fase 3 concluída.

## Entregável testável
- GM cria moedas por campanha
- Definir moeda padrão
- Cadastrar conversão em um sentido → o inverso é persistido automaticamente
- Players podem listar moedas e conversões

---

## Passos

### 4.1 — Migration

**`004_create_coins.up.sql`**

```sql
CREATE TABLE coin_types (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    campaign_id   UUID        NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    name          VARCHAR(50) NOT NULL,
    abbreviation  VARCHAR(10) NOT NULL,
    emoji         VARCHAR(10),  -- ex: "🪙", "⚪", "🟤"
    is_default    BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (campaign_id, abbreviation)
);

CREATE INDEX idx_coin_types_campaign ON coin_types(campaign_id);

-- Garante no banco que só uma moeda é padrão por campanha
CREATE UNIQUE INDEX idx_coin_default_per_campaign
    ON coin_types(campaign_id)
    WHERE is_default = TRUE;

CREATE TABLE coin_conversions (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    campaign_id   UUID    NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    from_coin_id  UUID    NOT NULL REFERENCES coin_types(id) ON DELETE CASCADE,
    to_coin_id    UUID    NOT NULL REFERENCES coin_types(id) ON DELETE CASCADE,
    rate          NUMERIC(18, 8) NOT NULL CHECK (rate > 0),
    UNIQUE (from_coin_id, to_coin_id)
);

CREATE INDEX idx_conversions_campaign ON coin_conversions(campaign_id);
```

**`004_create_coins.down.sql`**

```sql
DROP TABLE IF EXISTS coin_conversions;
DROP TABLE IF EXISTS coin_types;
```

---

### 4.2 — Modelos

**`models/coin.go`**

```go
type CoinType struct {
    ID           uuid.UUID
    CampaignID   uuid.UUID
    Name         string
    Abbreviation string
    Emoji        string    // opcional, ex: "🪙"
    IsDefault    bool
    CreatedAt    time.Time
}

type CoinConversion struct {
    ID           uuid.UUID
    CampaignID   uuid.UUID
    FromCoinID   uuid.UUID
    FromCoin     string    // nome/abreviação para exibição
    ToCoinID     uuid.UUID
    ToCoin       string
    Rate         float64
}
```

---

### 4.3 — Repositório

**`internal/coin/repository.go`**

```
CreateCoinType(ctx, campaignID, name, abbreviation) (*CoinType, error)
GetCoinTypeByID(ctx, id) (*CoinType, error)
GetDefaultCoin(ctx, campaignID) (*CoinType, error)
ListCoinTypes(ctx, campaignID) ([]CoinType, error)
UpdateCoinType(ctx, id, name, abbreviation) (*CoinType, error)
DeleteCoinType(ctx, id) error
SetDefaultCoin(ctx, campaignID, coinID) error  -- dentro de transação SQL

CreateConversion(ctx, campaignID, fromID, toID, rate) (*CoinConversion, error)
ListConversions(ctx, campaignID) ([]CoinConversion, error)
GetConversionByPair(ctx, fromID, toID) (*CoinConversion, error)
DeleteConversion(ctx, id) error
DeleteConversionPair(ctx, fromID, toID) error  -- deleta ambos os sentidos
```

---

### 4.4 — Serviço

**`internal/coin/service.go`**

**`SetDefaultCoin(ctx, campaignID, coinID)`**:
1. Verificar se a moeda pertence à campanha
2. Dentro de uma transação SQL:
   - `UPDATE coin_types SET is_default = FALSE WHERE campaign_id = $1`
   - `UPDATE coin_types SET is_default = TRUE WHERE id = $2`

**`CreateConversion(ctx, campaignID, fromID, toID, rate)`**:
1. Verificar se `fromID` e `toID` pertencem à campanha
2. Verificar se from ≠ to
3. Verificar se a conversão `from → to` já existe
4. Dentro de uma transação SQL:
   - Inserir `from → to` com `rate`
   - Inserir `to → from` com `1 / rate` (bidirecional automático)
5. Retornar o par criado

**`DeleteConversion(ctx, conversionID)`**:
1. Buscar a conversão pelo ID
2. Deletar o par inverso também (via `DeleteConversionPair`)

**`DeleteCoinType(ctx, coinID)`**:
1. Verificar se a moeda está em uso (itens, coin_purse, conversões)
2. Se sim → `ErrCoinInUse`
3. Se a moeda for a padrão → não permitir exclusão direta (deve definir outra como padrão antes)

---

### 4.5 — Handlers

**`internal/coin/handler.go`**

`POST /campaigns/:campaignID/coins`
- Body: `{ "name": "Ouro", "abbreviation": "GP", "emoji": "🪙" }`
- Role: `gm`
- Resposta `201`: CoinType criado

`GET /campaigns/:campaignID/coins`
- Role: `player`
- Resposta `200`: lista de moedas, destacando qual é a padrão

`PUT /campaigns/:campaignID/coins/:coinID`
- Body: `{ "name": "...", "abbreviation": "..." }`
- Role: `gm`

`DELETE /campaigns/:campaignID/coins/:coinID`
- Role: `gm`
- Erro `409` se em uso

`PUT /campaigns/:campaignID/coins/:coinID/set-default`
- Role: `gm`
- Sem body
- Resposta `200`: moeda atualizada com `is_default: true`

`POST /campaigns/:campaignID/coins/conversions`
- Body: `{ "from_coin_id": "...", "to_coin_id": "...", "rate": 10 }`
- Role: `gm`
- Resposta `201`: `{ "created": [ {from→to}, {to→from} ] }`

`GET /campaigns/:campaignID/coins/conversions`
- Role: `player`
- Resposta `200`: lista de conversões (pode mostrar só um sentido visualmente)

`DELETE /campaigns/:campaignID/coins/conversions/:conversionID`
- Role: `gm`
- Deleta os dois sentidos do par

---

### 4.6 — Estrutura de pastas desta fase

```
internal/
└── coin/
    ├── handler.go
    ├── service.go
    └── repository.go
models/
└── coin.go
```

---

## Testes Manuais

```bash
TOKEN_GM="<token do gm>"
CAMPAIGN_ID="<uuid>"

# Criar moedas
curl -X POST http://localhost:8080/campaigns/$CAMPAIGN_ID/coins \
  -H "Authorization: Bearer $TOKEN_GM" \
  -H "Content-Type: application/json" \
  -d '{"name":"Ouro","abbreviation":"GP"}'

curl -X POST http://localhost:8080/campaigns/$CAMPAIGN_ID/coins \
  -H "Authorization: Bearer $TOKEN_GM" \
  -H "Content-Type: application/json" \
  -d '{"name":"Prata","abbreviation":"SP"}'

GP_ID="<uuid do GP>"
SP_ID="<uuid do SP>"

# Definir GP como padrão
curl -X PUT http://localhost:8080/campaigns/$CAMPAIGN_ID/coins/$GP_ID/set-default \
  -H "Authorization: Bearer $TOKEN_GM"

# Criar conversão GP → SP (deve criar SP → GP automaticamente)
curl -X POST http://localhost:8080/campaigns/$CAMPAIGN_ID/coins/conversions \
  -H "Authorization: Bearer $TOKEN_GM" \
  -H "Content-Type: application/json" \
  -d "{\"from_coin_id\":\"$GP_ID\",\"to_coin_id\":\"$SP_ID\",\"rate\":10}"

# Listar conversões (deve mostrar ambos os sentidos)
curl http://localhost:8080/campaigns/$CAMPAIGN_ID/coins/conversions \
  -H "Authorization: Bearer $TOKEN_GM"

# Player tenta criar moeda (deve retornar 403)
TOKEN_PLAYER="<token de player>"
curl -X POST http://localhost:8080/campaigns/$CAMPAIGN_ID/coins \
  -H "Authorization: Bearer $TOKEN_PLAYER" \
  -H "Content-Type: application/json" \
  -d '{"name":"Cobre","abbreviation":"CP"}'
```

---

## Critérios de aceite

- [ ] Criar moeda com abbreviation duplicada na campanha retorna erro
- [ ] Definir moeda padrão desativa a anterior automaticamente (índice único garante isso)
- [ ] Criar conversão `GP → SP = 10` persiste também `SP → GP = 0.1`
- [ ] Deletar conversão remove os dois sentidos do par
- [ ] Deletar moeda padrão é bloqueado
- [ ] Deletar moeda em uso (com itens ou saldo) retorna `409`
- [ ] Player pode listar moedas/conversões mas não criar/editar/deletar
