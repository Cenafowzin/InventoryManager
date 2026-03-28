# Fase 3 — Campanhas e Membros

## Objetivo
CRUD de campanhas e sistema de membros com roles (`gm` / `player`). Middleware que verifica role na campanha para proteger rotas.

## Pré-requisito
Fase 2 concluída.

## Entregável testável
- Criar e gerenciar campanhas
- Convidar membros e alterar roles
- Middleware de role bloqueando acessos indevidos

---

## Passos

### 3.1 — Migration

**`003_create_campaigns.up.sql`**

```sql
CREATE TABLE campaigns (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name        VARCHAR(100) NOT NULL,
    description TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TYPE campaign_role AS ENUM ('gm', 'player');

CREATE TABLE campaign_members (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    campaign_id UUID NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role        campaign_role NOT NULL DEFAULT 'player',
    joined_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (campaign_id, user_id)
);

CREATE INDEX idx_campaign_members_campaign ON campaign_members(campaign_id);
CREATE INDEX idx_campaign_members_user ON campaign_members(user_id);
```

**`003_create_campaigns.down.sql`**

```sql
DROP TABLE IF EXISTS campaign_members;
DROP TYPE  IF EXISTS campaign_role;
DROP TABLE IF EXISTS campaigns;
```

---

### 3.2 — Modelos

**`models/campaign.go`**

```go
type Campaign struct {
    ID          uuid.UUID
    Name        string
    Description string
    CreatedAt   time.Time
    UpdatedAt   time.Time
}

type CampaignMember struct {
    ID         uuid.UUID
    CampaignID uuid.UUID
    UserID     uuid.UUID
    Username   string    // join com users (para exibição)
    Role       string    // "gm" | "player"
    JoinedAt   time.Time
}
```

---

### 3.3 — Repositório

**`internal/campaign/repository.go`**

```
CreateCampaign(ctx, name, description) (*Campaign, error)
GetCampaignByID(ctx, campaignID) (*Campaign, error)
ListCampaignsByUser(ctx, userID) ([]Campaign, error)
UpdateCampaign(ctx, campaignID, name, description) (*Campaign, error)
DeleteCampaign(ctx, campaignID) error

AddMember(ctx, campaignID, userID, role) (*CampaignMember, error)
GetMember(ctx, campaignID, userID) (*CampaignMember, error)
ListMembers(ctx, campaignID) ([]CampaignMember, error)
UpdateMemberRole(ctx, campaignID, userID, role) error
RemoveMember(ctx, campaignID, userID) error
```

---

### 3.4 — Serviço

**`internal/campaign/service.go`**

**`CreateCampaign(ctx, creatorUserID, name, description)`**:
1. `repo.CreateCampaign`
2. `repo.AddMember(campaignID, creatorUserID, "gm")` — criador vira GM automaticamente
3. Retornar campanha

**`InviteMember(ctx, campaignID, invitedUserID, role)`**:
1. Verificar se usuário já é membro → `ErrAlreadyMember`
2. Verificar se usuário convidado existe (via user repo)
3. `repo.AddMember`

**`UpdateMemberRole(ctx, campaignID, targetUserID)`**:
1. Não permitir que o último GM seja rebaixado → `ErrLastGM`
2. `repo.UpdateMemberRole`

**`RemoveMember(ctx, campaignID, targetUserID)`**:
1. Não permitir remover o último GM → `ErrLastGM`
2. `repo.RemoveMember`

---

### 3.5 — Handlers

**`internal/campaign/handler.go`**

| Rota | Body / Params | Resposta |
|------|---------------|----------|
| `POST /campaigns` | `{name, description}` | `201` campanha criada |
| `GET /campaigns` | — | `200` lista de campanhas do usuário logado |
| `GET /campaigns/:campaignID` | — | `200` campanha |
| `PUT /campaigns/:campaignID` | `{name, description}` | `200` campanha atualizada |
| `DELETE /campaigns/:campaignID` | — | `204` |
| `POST /campaigns/:campaignID/members` | `{user_id, role}` | `201` membro criado |
| `GET /campaigns/:campaignID/members` | — | `200` lista de membros |
| `PUT /campaigns/:campaignID/members/:userID` | `{role}` | `200` membro atualizado |
| `DELETE /campaigns/:campaignID/members/:userID` | — | `204` |

---

### 3.6 — Middleware de role na campanha

**`pkg/middleware/campaign_role.go`**

Função `RequireCampaignRole(minRole string)` retorna um `http.Handler` middleware que:

1. Extrai `campaignID` do URL param (`:campaignID`)
2. Extrai `userID` do context (injetado pelo middleware de auth)
3. Consulta `campaign_members` para obter a role do usuário
4. Compara com `minRole` (`"gm"` ou `"player"`)
5. Injeta a role no context para uso nos handlers
6. Retorna `403` se insuficiente, `404` se não for membro

Helper `CampaignRoleFromContext(ctx) string` — retorna `"gm"` ou `"player"`.

**Hierarquia de roles:**
```
gm > player
```

---

### 3.7 — Registrar rotas

```go
r.Group(func(r chi.Router) {
    r.Use(middleware.Authenticate)

    r.Post("/campaigns", campaignHandler.Create)
    r.Get("/campaigns",  campaignHandler.List)

    r.Route("/campaigns/{campaignID}", func(r chi.Router) {
        r.Use(middleware.RequireCampaignRole("player")) // mínimo: ser membro

        r.Get("/",    campaignHandler.Get)
        r.Put("/",    campaignHandler.Update)     // serviço valida gm internamente
        r.Delete("/", campaignHandler.Delete)     // serviço valida gm internamente

        r.Get("/members",              campaignHandler.ListMembers)
        r.Post("/members",             campaignHandler.AddMember)        // gm
        r.Put("/members/{userID}",     campaignHandler.UpdateMember)     // gm
        r.Delete("/members/{userID}",  campaignHandler.RemoveMember)     // gm
    })
})
```

> Rotas de GM: o handler checa `CampaignRoleFromContext` e retorna `403` se for `player`.

---

### 3.8 — Estrutura de pastas desta fase

```
internal/
└── campaign/
    ├── handler.go
    ├── service.go
    └── repository.go
pkg/middleware/
└── campaign_role.go
models/
└── campaign.go
```

---

## Testes Manuais

```bash
TOKEN="<access_token do login>"
CAMPAIGN_ID="<uuid retornado>"
USER2_ID="<uuid de outro usuário>"

# Criar campanha
curl -X POST http://localhost:8080/campaigns \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"A Maldição do Liche","description":"Campanha de terror gótico"}'

# Listar minhas campanhas
curl http://localhost:8080/campaigns \
  -H "Authorization: Bearer $TOKEN"

# Convidar membro como player
curl -X POST http://localhost:8080/campaigns/$CAMPAIGN_ID/members \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"user_id\":\"$USER2_ID\",\"role\":\"player\"}"

# Tentar deletar campanha com token de player (deve retornar 403)
TOKEN_PLAYER="<access_token do user2>"
curl -X DELETE http://localhost:8080/campaigns/$CAMPAIGN_ID \
  -H "Authorization: Bearer $TOKEN_PLAYER"

# Tentar acessar campanha sem ser membro (deve retornar 403/404)
TOKEN_STRANGER="<token de user não membro>"
curl http://localhost:8080/campaigns/$CAMPAIGN_ID \
  -H "Authorization: Bearer $TOKEN_STRANGER"
```

---

## Critérios de aceite

- [ ] Criar campanha → criador vira GM automaticamente
- [ ] `GET /campaigns` lista só as campanhas do usuário logado
- [ ] Player não consegue editar/deletar campanha (`403`)
- [ ] Não-membro não acessa rotas da campanha (`403`)
- [ ] Não é possível rebaixar o último GM
- [ ] Não é possível remover o último GM
- [ ] Convidar usuário já membro retorna erro apropriado
