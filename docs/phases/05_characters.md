# Fase 5 — Personagens

## Objetivo
CRUD de personagens dentro de campanhas, com controle de propriedade (owner) e permissões por role.

## Pré-requisito
Fase 3 concluída (campanhas + middleware de role).

## Entregável testável
- GM cria e gerencia qualquer personagem
- Player cria e gerencia apenas seus próprios personagens
- Listagem com visibilidade correta por role

---

## Passos

### 5.1 — Migration

**`005_create_characters.up.sql`**

```sql
CREATE TABLE characters (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    campaign_id   UUID         NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    owner_user_id UUID         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name          VARCHAR(100) NOT NULL,
    description   TEXT,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_characters_campaign ON characters(campaign_id);
CREATE INDEX idx_characters_owner    ON characters(owner_user_id);
```

**`005_create_characters.down.sql`**

```sql
DROP TABLE IF EXISTS characters;
```

---

### 5.2 — Modelo

**`models/character.go`**

```go
type Character struct {
    ID          uuid.UUID
    CampaignID  uuid.UUID
    OwnerUserID uuid.UUID
    OwnerName   string    // join com users (exibição)
    Name        string
    Description string
    CreatedAt   time.Time
    UpdatedAt   time.Time
}
```

---

### 5.3 — Repositório

**`internal/character/repository.go`**

```
CreateCharacter(ctx, campaignID, ownerUserID, name, description) (*Character, error)
GetCharacterByID(ctx, id) (*Character, error)
ListCharactersByCampaign(ctx, campaignID) ([]Character, error)
ListCharactersByOwner(ctx, campaignID, ownerUserID) ([]Character, error)
UpdateCharacter(ctx, id, name, description) (*Character, error)
DeleteCharacter(ctx, id) error
```

---

### 5.4 — Serviço

**`internal/character/service.go`**

**`CreateCharacter(ctx, campaignID, requesterID, requesterRole, ownerUserID, name, description)`**:
- Se `requesterRole == "player"`: forçar `ownerUserID = requesterID` (player só cria para si)
- Se `requesterRole == "gm"`: pode criar para qualquer membro da campanha
- Verificar que `ownerUserID` é membro da campanha

**`GetCharacter(ctx, characterID, requesterID, requesterRole)`**:
- Se `requesterRole == "gm"`: acesso irrestrito
- Se `requesterRole == "player"`: validar `character.OwnerUserID == requesterID` → `403`

**`ListCharacters(ctx, campaignID, requesterID, requesterRole)`**:
- Se `gm`: retorna todos os personagens da campanha
- Se `player`: retorna só os personagens do requester

**`UpdateCharacter` / `DeleteCharacter`**:
- Mesma lógica de propriedade do `GetCharacter`

---

### 5.5 — Handlers

**`internal/character/handler.go`**

`POST /campaigns/:campaignID/characters`
- Body: `{ "name": "Arathor", "description": "...", "owner_user_id": "..." }`
  - `owner_user_id` opcional: se omitido, usa o requester; se GM, pode especificar outro membro
- Resposta `201`: personagem criado

`GET /campaigns/:campaignID/characters`
- Resposta `200`: lista (filtrada por role conforme serviço)
- Cada item inclui: `id`, `name`, `description`, `owner` (nome do player)

`GET /campaigns/:campaignID/characters/:charID`
- Resposta `200`: personagem completo
- `403` se player tentando ver personagem de outro

`PUT /campaigns/:campaignID/characters/:charID`
- Body: `{ "name": "...", "description": "..." }`
- `403` se player tentando editar personagem de outro

`DELETE /campaigns/:campaignID/characters/:charID`
- `204` em sucesso
- `403` se player tentando deletar personagem de outro

---

### 5.6 — Estrutura de pastas desta fase

```
internal/
└── character/
    ├── handler.go
    ├── service.go
    └── repository.go
models/
└── character.go
```

---

## Testes Manuais

```bash
TOKEN_GM="<token do gm>"
TOKEN_PLAYER="<token do player>"
CAMPAIGN_ID="<uuid>"
PLAYER_USER_ID="<uuid do player>"

# GM cria personagem para o player
curl -X POST http://localhost:8080/campaigns/$CAMPAIGN_ID/characters \
  -H "Authorization: Bearer $TOKEN_GM" \
  -H "Content-Type: application/json" \
  -d "{\"name\":\"Arathor, o Guerreiro\",\"description\":\"Elfo ranger das montanhas\",\"owner_user_id\":\"$PLAYER_USER_ID\"}"

CHAR_GM_ID="<uuid do personagem criado pelo gm>"

# Player cria o próprio personagem (owner_user_id ignorado/forçado)
curl -X POST http://localhost:8080/campaigns/$CAMPAIGN_ID/characters \
  -H "Authorization: Bearer $TOKEN_PLAYER" \
  -H "Content-Type: application/json" \
  -d '{"name":"Sylvara","description":"Maga das sombras"}'

CHAR_PLAYER_ID="<uuid do personagem do player>"

# GM lista todos os personagens
curl http://localhost:8080/campaigns/$CAMPAIGN_ID/characters \
  -H "Authorization: Bearer $TOKEN_GM"
# Deve retornar AMBOS os personagens

# Player lista personagens
curl http://localhost:8080/campaigns/$CAMPAIGN_ID/characters \
  -H "Authorization: Bearer $TOKEN_PLAYER"
# Deve retornar APENAS o personagem do player

# Player tenta editar personagem de outro (deve retornar 403)
curl -X PUT http://localhost:8080/campaigns/$CAMPAIGN_ID/characters/$CHAR_GM_ID \
  -H "Authorization: Bearer $TOKEN_PLAYER" \
  -H "Content-Type: application/json" \
  -d '{"name":"Hackeado"}'

# GM edita qualquer personagem
curl -X PUT http://localhost:8080/campaigns/$CAMPAIGN_ID/characters/$CHAR_PLAYER_ID \
  -H "Authorization: Bearer $TOKEN_GM" \
  -H "Content-Type: application/json" \
  -d '{"name":"Sylvara, a Temida","description":"Maga das sombras profundas"}'
```

---

## Critérios de aceite

- [ ] Player ao criar personagem → `owner_user_id` é sempre o próprio (ignorado se enviado)
- [ ] GM pode criar personagem para qualquer membro
- [ ] GM ver lista → retorna todos; Player → retorna só os seus
- [ ] Player acessar personagem de outro → `403`
- [ ] GM acessa e edita qualquer personagem
- [ ] `owner_user_id` deve ser membro da campanha (não pode criar personagem para não-membro)
