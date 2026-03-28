# InventoryManager — Memória do Projeto

## Stack
- Go + chi router, PostgreSQL, pgx/v5, golang-migrate, golang-jwt, bcrypt, godotenv
- Módulo: `github.com/rubendubeux/inventory-manager`

## Estrutura de Pacotes
- `cmd/api/main.go` — wiring de tudo
- `internal/{auth,campaign,coin,character}/` — handler, service, repository, errors
- `models/` — structs compartilhadas (user, campaign, coin, character)
- `pkg/middleware/` — auth.go (JWT), campaign_role.go (role checker)
- `internal/db/migrations/` — arquivos SQL versionados

## Padrões do Código
- Repositório: queries raw SQL com pgx; sem ORM
- `errors.go` por pacote com sentinel errors (`ErrXxx = errors.New(...)`)
- Handler: decodifica body → chama service → mapeia erros para HTTP status
- Context: `middleware.UserIDFromContext(ctx)` e `middleware.CampaignRoleFromContext(ctx)`
- Listas vazias retornam `[]any{}` em vez de null
- Scan duplo para INSERT+JOIN: insert retorna `id`, depois `GetByID` faz o JOIN

## Supabase — RLS obrigatório
Toda migration que cria tabela DEVE incluir ao final:
```sql
ALTER TABLE <tabela> ENABLE ROW LEVEL SECURITY;
CREATE POLICY "allow all" ON <tabela> FOR ALL USING (true) WITH CHECK (true);
```
O controle de acesso real é feito pela aplicação (JWT + middleware), não pelo banco.

## Migrações (ordem atual)
- 001_init — uuid-ossp extension
- 002_create_users
- 003_create_campaigns + campaign_members
- 004_campaign_invites
- 005_create_coins (coin_types, coin_conversions, coin_purse)
- 006_create_characters

## Fases Implementadas
- [x] Fase 1 — Foundation
- [x] Fase 2 — Auth (register, login, refresh JWT)
- [x] Fase 3 — Campaigns (CRUD + members + invite por código)
- [x] Fase 4 — Coins (coin_types, conversions bidirecionais, set-default)
- [x] Fase 5 — Characters (CRUD com controle owner/gm, max_carry_weight_kg)
- [ ] Fase 6 — Inventory (storages, items, coin purse, load summary)
- [ ] Fase 7 — Shop
- [ ] Fase 8 — Transactions

## Regras de Role
- `player`: só vê/edita personagens onde é `owner_user_id`
- `gm`: acesso irrestrito a qualquer personagem da campanha
- Middleware injeta role no context; service consulta via `CampaignRoleFromContext`
