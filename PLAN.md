# RPG Inventory Manager — Plano do Sistema

## Visão Geral

API REST em Go para gerenciar inventários de personagens de RPG, com sistema de campanhas, roles por campanha, loja e transações negociáveis. Consumida primariamente por um bot do Discord.

---

## Domínio Completo

```
User
  └── CampaignMember (role: GM | Player)
        └── Campaign
              ├── Characters[]  (personagem pertence a um player da campanha)
              │     ├── StorageSpaces[]  (mochila, baú, etc. — com flag de carga)
              │     └── Inventory
              │           ├── Items[]  (cada item → StorageSpace + Categories[])
              │           └── CoinPurse[]
              ├── ShopItems[]   (itens genéricos criados pelo GM, com Categories[])
              ├── Categories[]  (tags reutilizáveis por campanha: "arma", "poção", etc.)
              └── CoinTypes[]   (moedas configuradas na campanha)
                    └── CoinConversions[]
```

---

## Modelos de Dados

### User

| Campo        | Tipo   | Descrição              |
|--------------|--------|------------------------|
| `id`         | UUID   |                        |
| `username`   | string | Único                  |
| `email`      | string | Único                  |
| `password_hash` | string |                     |
| `created_at` | time   |                        |

### Campaign

| Campo        | Tipo   | Descrição               |
|--------------|--------|-------------------------|
| `id`         | UUID   |                         |
| `name`       | string |                         |
| `description`| string |                         |
| `created_at` | time   |                         |

### CampaignMember (N:N User ↔ Campaign)

| Campo          | Tipo   | Descrição                         |
|----------------|--------|-----------------------------------|
| `id`           | UUID   |                                   |
| `campaign_id`  | UUID   |                                   |
| `user_id`      | UUID   |                                   |
| `role`         | enum   | `gm` \| `player`                  |
| `joined_at`    | time   |                                   |

> Uma campanha pode ter múltiplos GMs.

### CoinType (por campanha)

| Campo          | Tipo    | Descrição                                       |
|----------------|---------|-------------------------------------------------|
| `id`           | UUID    |                                                 |
| `campaign_id`  | UUID    |                                                 |
| `name`         | string  | Ex: "Ouro"                                      |
| `abbreviation` | string  | Ex: "GP"                                        |
| `emoji`        | string  | Ex: "🪙", "⚪", "🟤" (opcional)                 |
| `is_default`   | bool    | Moeda padrão da campanha (apenas uma por vez)   |
| `created_at`   | time    |                                                 |

### CoinConversion

Sempre bidirecional: ao cadastrar `GP → SP = 10`, o sistema persiste automaticamente `SP → GP = 0.1`.

| Campo          | Tipo    | Descrição                                            |
|----------------|---------|------------------------------------------------------|
| `id`           | UUID    |                                                      |
| `campaign_id`  | UUID    |                                                      |
| `from_coin_id` | UUID    |                                                      |
| `to_coin_id`   | UUID    |                                                      |
| `rate`         | float64 | Quantas unidades de destino 1 unidade de origem vale |

### Character

| Campo                  | Tipo    | Descrição                                           |
|------------------------|---------|-----------------------------------------------------|
| `id`                   | UUID    |                                                     |
| `campaign_id`          | UUID    |                                                     |
| `owner_user_id`        | UUID    | Player dono do personagem                           |
| `name`                 | string  |                                                     |
| `description`          | string  |                                                     |
| `max_carry_weight_kg`  | float64 | Capacidade máxima de carga (nullable — sem limite se nulo) |
| `created_at`           | time    |                                                     |
| `updated_at`           | time    |                                                     |

> GMs podem ler/editar qualquer personagem da campanha. Players só acessam os seus.

### StorageSpace (espaços de armazenamento do personagem)

| Campo                    | Tipo    | Descrição                                                   |
|--------------------------|---------|-------------------------------------------------------------|
| `id`                     | UUID    |                                                             |
| `character_id`           | UUID    |                                                             |
| `name`                   | string  | Ex: "Mochila", "Baú do Acampamento", "Mão Direita"          |
| `description`            | string  |                                                             |
| `counts_toward_load`     | bool    | Se true, o peso dos itens aqui entra no cálculo de carga    |
| `capacity_kg`            | float64 | Capacidade máxima do espaço (nullable — sem limite se nulo) |
| `created_at`             | time    |                                                             |

> Itens sem `storage_space_id` explicitamente definido ficam em um espaço default criado automaticamente ("Geral") que conta para carga.

### Category (tags por campanha)

| Campo          | Tipo    | Descrição                                      |
|----------------|---------|------------------------------------------------|
| `id`           | UUID    |                                                |
| `campaign_id`  | UUID    |                                                |
| `name`         | string  | Ex: "Arma", "Armadura", "Poção", "Quest Item"  |
| `color`        | string  | Hex color opcional para UI/Discord (`#FF5733`) |
| `created_at`   | time    |                                                |

### ItemCategory (N:N Item ↔ Category)

| Campo         | Tipo | Descrição |
|---------------|------|-----------|
| `item_id`     | UUID |           |
| `category_id` | UUID |           |

### ShopItemCategory (N:N ShopItem ↔ Category)

| Campo           | Tipo | Descrição |
|-----------------|------|-----------|
| `shop_item_id`  | UUID |           |
| `category_id`   | UUID |           |

### Item (inventário do personagem)

| Campo               | Tipo    | Descrição                                                  |
|---------------------|---------|------------------------------------------------------------|
| `id`                | UUID    |                                                            |
| `character_id`      | UUID    |                                                            |
| `name`              | string  |                                                            |
| `description`       | string  |                                                            |
| `emoji`             | string  | Ex: "⚔️", "🧪", "🛡️" (opcional)                          |
| `weight_kg`         | float64 | Sempre armazenado em kg                                    |
| `value`             | float64 | Valor unitário                                             |
| `value_coin_id`     | UUID    | Moeda do valor (padrão da campanha se omitido)             |
| `storage_space_id`  | UUID    | Espaço de armazenamento (FK → StorageSpace; nullable = Geral) |
| `quantity`          | int     | Quantidade                                                 |
| `shop_item_id`      | UUID?   | Referência ao ShopItem de origem (nullable)                |
| `created_at`        | time    |                                                            |
| `updated_at`        | time    |                                                            |

> Categorias do item ficam em `ItemCategory` (relação N:N).

### CoinPurse (saldo de moedas do personagem)

| Campo          | Tipo    | Descrição               |
|----------------|---------|-------------------------|
| `id`           | UUID    |                         |
| `character_id` | UUID    |                         |
| `coin_type_id` | UUID    |                         |
| `amount`       | float64 |                         |
| `updated_at`   | time    |                         |

### ShopItem (loja da campanha, gerenciado pelo GM)

| Campo           | Tipo    | Descrição                                            |
|-----------------|---------|------------------------------------------------------|
| `id`            | UUID    |                                                      |
| `campaign_id`   | UUID    |                                                      |
| `name`          | string  |                                                      |
| `description`   | string  |                                                      |
| `emoji`         | string  | Ex: "⚔️", "🧪", "🛡️" (opcional)                    |
| `weight_kg`     | float64 |                                                      |
| `base_value`    | float64 | Preço base (pode ser alterado na transação)          |
| `value_coin_id` | UUID    | Moeda do preço (padrão da campanha se omitido)       |
| `is_available`  | bool    | GM pode ocultar itens sem deletar                    |
| `created_at`    | time    |                                                      |
| `updated_at`    | time    |                                                      |

> Categorias do shop item ficam em `ShopItemCategory` (relação N:N).

---

## Sistema de Transações (Compra/Venda)

### Fluxo de uma transação

```
1. Iniciar transação (POST /transactions)
   → retorna rascunho com preços originais calculados

2. Ajustar preços (PATCH /transactions/:id)
   → alterar valor individual de item OU total geral
   → sistema recalcula a distribuição proporcional se total for alterado

3. Confirmar (POST /transactions/:id/confirm)
   → efetiva movimentação de itens e moedas

4. Cancelar (POST /transactions/:id/cancel)
   → descarta sem efeito
```

### Transaction

| Campo              | Tipo    | Descrição                                              |
|--------------------|---------|--------------------------------------------------------|
| `id`               | UUID    |                                                        |
| `campaign_id`      | UUID    |                                                        |
| `character_id`     | UUID    | Quem compra/vende                                      |
| `type`             | enum    | `buy` \| `sell`                                        |
| `status`           | enum    | `draft` \| `confirmed` \| `cancelled`                  |
| `original_total`   | float64 | Total calculado pelos preços base                      |
| `adjusted_total`   | float64 | Total final (pode ser alterado antes de confirmar)     |
| `total_coin_id`    | UUID    | Moeda do total                                         |
| `notes`            | string  | Anotações do GM (ex: "desconto por roleplay")          |
| `created_by`       | UUID    | Quem iniciou a transação                               |
| `created_at`       | time    |                                                        |
| `confirmed_at`     | time?   |                                                        |

### TransactionItem (linhas da transação)

| Campo              | Tipo    | Descrição                                              |
|--------------------|---------|--------------------------------------------------------|
| `id`               | UUID    |                                                        |
| `transaction_id`   | UUID    |                                                        |
| `shop_item_id`     | UUID?   | Para compras (referência ao ShopItem)                  |
| `inventory_item_id`| UUID?   | Para vendas (referência ao Item do personagem)         |
| `name`             | string  | Snapshot do nome no momento da transação               |
| `quantity`         | int     |                                                        |
| `unit_value`       | float64 | Valor unitário original                                |
| `adjusted_unit_value` | float64 | Valor unitário ajustado (pode ser igual ao original) |
| `coin_id`          | UUID    |                                                        |

> Ao confirmar uma compra: itens são criados no inventário do personagem e o saldo de moedas é debitado.
> Ao confirmar uma venda: itens são removidos do inventário e o saldo de moedas é creditado.

---

## Conversão de Peso

- Unidade base: **kg** (sempre persistida assim).
- A API aceita peso em `lbs` via parâmetro opcional `weight_unit: "lbs"` — converte automaticamente na entrada.
- Taxa fixa: `1 lb = 0.453592 kg` (constante no código, sem tabela no banco).
- Respostas sempre retornam `weight_kg`. O cliente Discord pode exibir como preferir.

---

## Endpoints da API

### Auth

| Método | Rota              | Descrição                    |
|--------|-------------------|------------------------------|
| POST   | `/auth/register`  | Criar conta                  |
| POST   | `/auth/login`     | Login → retorna JWT          |
| POST   | `/auth/refresh`   | Renovar token                |

### Campanhas

| Método | Rota                                    | Descrição                          | Role mínima |
|--------|-----------------------------------------|------------------------------------|-------------|
| POST   | `/campaigns`                            | Criar campanha (vira GM)           | autenticado |
| GET    | `/campaigns`                            | Listar campanhas do usuário        | autenticado |
| GET    | `/campaigns/:id`                        | Detalhar campanha                  | member      |
| PUT    | `/campaigns/:id`                        | Editar campanha                    | gm          |
| DELETE | `/campaigns/:id`                        | Deletar campanha                   | gm          |
| POST   | `/campaigns/:id/members`               | Convidar membro                    | gm          |
| GET    | `/campaigns/:id/members`               | Listar membros                     | member      |
| PUT    | `/campaigns/:id/members/:user_id`      | Alterar role de membro             | gm          |
| DELETE | `/campaigns/:id/members/:user_id`      | Remover membro                     | gm          |

### Moedas (por campanha)

| Método | Rota                                          | Role mínima |
|--------|-----------------------------------------------|-------------|
| POST   | `/campaigns/:id/coins`                        | gm          |
| GET    | `/campaigns/:id/coins`                        | member      |
| PUT    | `/campaigns/:id/coins/:coin_id`               | gm          |
| DELETE | `/campaigns/:id/coins/:coin_id`               | gm          |
| PUT    | `/campaigns/:id/coins/:coin_id/set-default`   | gm          |
| POST   | `/campaigns/:id/coins/conversions`            | gm          |
| GET    | `/campaigns/:id/coins/conversions`            | member      |
| DELETE | `/campaigns/:id/coins/conversions/:conv_id`   | gm          |

### Personagens

| Método | Rota                                          | Descrição                          | Role mínima          |
|--------|-----------------------------------------------|------------------------------------|----------------------|
| POST   | `/campaigns/:id/characters`                   | Criar personagem                   | member               |
| GET    | `/campaigns/:id/characters`                   | Listar personagens                 | member               |
| GET    | `/campaigns/:id/characters/:char_id`          | Detalhar personagem                | owner ou gm          |
| PUT    | `/campaigns/:id/characters/:char_id`          | Editar personagem                  | owner ou gm          |
| DELETE | `/campaigns/:id/characters/:char_id`          | Deletar personagem                 | owner ou gm          |

### Categorias (por campanha)

| Método | Rota                                          | Descrição                        | Role mínima |
|--------|-----------------------------------------------|----------------------------------|-------------|
| POST   | `/campaigns/:id/categories`                   | Criar categoria                  | gm          |
| GET    | `/campaigns/:id/categories`                   | Listar categorias                | member      |
| PUT    | `/campaigns/:id/categories/:cat_id`           | Editar categoria                 | gm          |
| DELETE | `/campaigns/:id/categories/:cat_id`           | Remover categoria                | gm          |

### Espaços de Armazenamento (por personagem)

| Método | Rota                                                             | Descrição                                         | Role mínima |
|--------|------------------------------------------------------------------|---------------------------------------------------|-------------|
| POST   | `/campaigns/:id/characters/:char_id/storages`                    | Criar espaço de armazenamento                     | owner ou gm |
| GET    | `/campaigns/:id/characters/:char_id/storages`                    | Listar espaços (com peso atual e capacidade)      | owner ou gm |
| PUT    | `/campaigns/:id/characters/:char_id/storages/:storage_id`        | Editar espaço                                     | owner ou gm |
| DELETE | `/campaigns/:id/characters/:char_id/storages/:storage_id`        | Remover espaço (itens voltam para Geral)          | owner ou gm |

### Inventário — Itens

| Método | Rota                                                     | Role mínima |
|--------|----------------------------------------------------------|-------------|
| POST   | `/campaigns/:id/characters/:char_id/items`               | owner ou gm |
| GET    | `/campaigns/:id/characters/:char_id/items`               | owner ou gm |
| GET    | `/campaigns/:id/characters/:char_id/items/:item_id`      | owner ou gm |
| PUT    | `/campaigns/:id/characters/:char_id/items/:item_id`      | owner ou gm |
| DELETE | `/campaigns/:id/characters/:char_id/items/:item_id`      | owner ou gm |

> Filtros disponíveis em `GET /items`: `?category_id=`, `?storage_id=`

### Inventário — Moedas do Personagem

| Método | Rota                                                         | Descrição                         | Role mínima |
|--------|--------------------------------------------------------------|-----------------------------------|-------------|
| GET    | `/campaigns/:id/characters/:char_id/coins`                   | Ver saldo de moedas               | owner ou gm |
| PUT    | `/campaigns/:id/characters/:char_id/coins/:coin_id`          | Definir saldo de uma moeda        | owner ou gm |
| POST   | `/campaigns/:id/characters/:char_id/coins/convert`           | Converter entre moedas            | owner ou gm |

### Inventário — Resumo e Carga

| Método | Rota                                                | Descrição                                                   |
|--------|-----------------------------------------------------|-------------------------------------------------------------|
| GET    | `/campaigns/:id/characters/:char_id/inventory`      | Itens + moedas + carga atual + carga máx + valor total      |
| GET    | `/campaigns/:id/characters/:char_id/load`           | Apenas resumo de carga (por espaço + total)                 |

### Loja da Campanha

| Método | Rota                                           | Descrição                          | Role mínima |
|--------|------------------------------------------------|------------------------------------|-------------|
| POST   | `/campaigns/:id/shop`                          | Criar item na loja                 | gm          |
| GET    | `/campaigns/:id/shop`                          | Listar itens disponíveis           | member      |
| GET    | `/campaigns/:id/shop/:shop_item_id`            | Detalhar item da loja              | member      |
| PUT    | `/campaigns/:id/shop/:shop_item_id`            | Editar item da loja                | gm          |
| DELETE | `/campaigns/:id/shop/:shop_item_id`            | Remover item da loja               | gm          |

### Transações

| Método | Rota                                     | Descrição                                               | Role mínima |
|--------|------------------------------------------|---------------------------------------------------------|-------------|
| POST   | `/campaigns/:id/transactions`            | Iniciar transação (compra ou venda)                     | member      |
| GET    | `/campaigns/:id/transactions`            | Listar transações da campanha                           | member      |
| GET    | `/campaigns/:id/transactions/:tx_id`     | Detalhar transação (com linhas e totais)                | member      |
| PATCH  | `/campaigns/:id/transactions/:tx_id`     | Ajustar preços (itens individuais ou total)             | gm ou owner |
| POST   | `/campaigns/:id/transactions/:tx_id/confirm` | Confirmar e executar transação                      | gm ou owner |
| POST   | `/campaigns/:id/transactions/:tx_id/cancel` | Cancelar transação                                  | gm ou owner |

---

## Regras de Negócio

### Autenticação & Autorização
- JWT com tempo de expiração curto + refresh token.
- Middleware verifica role na campanha para cada rota protegida.
- GMs veem e editam qualquer personagem da campanha.
- Players só acessam personagens onde são `owner_user_id`.

### Moedas
- `is_default = true` é único por campanha; trocar o padrão desativa o anterior.
- Conversão bidirecional: ao cadastrar `A → B = rate`, o sistema persiste automaticamente `B → A = 1/rate`.
- Converter moedas ajusta saldo em tempo real (não é transação de loja).

### Transações
- Transação nasce com `status = draft`.
- Em `draft`, preços individuais e o total podem ser alterados livremente.
- Alterar o `adjusted_total` distribui a diferença proporcionalmente entre os itens (ou mantém individuais e soma).
- `confirm` valida se o personagem tem moedas suficientes (compra) ou tem os itens (venda).
- Após `confirmed` ou `cancelled`, nenhum campo pode ser alterado.
- O snapshot do nome/preço do item é salvo em `TransactionItem` para histórico.

### Peso e Carga
- Sempre persistido em `weight_kg`.
- Entrada aceita `weight_unit: "lbs"` opcionalmente; conversão: `× 0.453592`.
- **Carga atual** (`current_load_kg`): `SUM(item.weight_kg * item.quantity)` apenas para itens em espaços com `counts_toward_load = true`.
- **Carga máxima** (`max_carry_weight_kg`): definida no personagem; nullable (sem limite).
- **Capacidade do espaço** (`capacity_kg`): opcional por espaço; o sistema alerta se excedida, mas não bloqueia.
- Ao deletar um `StorageSpace`, seus itens são realocados para o espaço "Geral" (criado automaticamente).
- O espaço "Geral" não pode ser deletado.

### Categorias
- Pertencem à campanha e são reutilizáveis por qualquer item ou shop item da mesma campanha.
- Um item pode ter zero ou mais categorias.
- Deletar uma categoria remove as associações, mas não os itens.
- Filtros de listagem: `GET /items?category_id=` e `GET /shop?category_id=`.

### Loja
- ShopItems são templates; ao comprar, um `Item` real é criado no inventário.
- `is_available = false` oculta o item da loja sem perder histórico.
- Categorias do ShopItem são copiadas para o Item criado na compra.

---

## Estrutura de Pastas

```
InventoryManager/
├── cmd/
│   └── api/
│       └── main.go
├── internal/
│   ├── auth/
│   │   ├── handler.go
│   │   ├── service.go
│   │   └── repository.go
│   ├── campaign/
│   │   ├── handler.go
│   │   ├── service.go
│   │   └── repository.go
│   ├── character/
│   │   ├── handler.go
│   │   ├── service.go
│   │   └── repository.go
│   ├── inventory/
│   │   ├── handler.go
│   │   ├── item_service.go
│   │   ├── coin_service.go
│   │   ├── storage_service.go
│   │   ├── summary_service.go
│   │   └── repository.go
│   ├── category/
│   │   ├── handler.go
│   │   ├── service.go
│   │   └── repository.go
│   ├── coin/
│   │   ├── handler.go
│   │   ├── service.go
│   │   └── repository.go
│   ├── shop/
│   │   ├── handler.go
│   │   ├── service.go
│   │   └── repository.go
│   ├── transaction/
│   │   ├── handler.go
│   │   ├── service.go
│   │   └── repository.go
│   └── db/
│       ├── db.go
│       └── migrations/
│           ├── 001_create_users.sql
│           ├── 002_create_campaigns.sql
│           ├── 003_create_coins.sql
│           ├── 004_create_characters.sql
│           ├── 005_create_inventory.sql
│           ├── 006_create_shop.sql
│           └── 007_create_transactions.sql
├── pkg/
│   ├── middleware/
│   │   ├── auth.go         # valida JWT
│   │   └── campaign_role.go # verifica role na campanha
│   └── weight/
│       └── convert.go      # lbs → kg
├── PLAN.md
└── go.mod
```

---

## Stack Tecnológica

| Componente   | Escolha            | Motivo                                          |
|--------------|--------------------|-------------------------------------------------|
| Router       | `chi`              | Leve, bom suporte a middleware encadeado        |
| Banco        | PostgreSQL          | Relações complexas, FK, transações ACID         |
| Queries      | `sqlc`             | Queries tipadas geradas a partir de SQL puro    |
| Migrations   | `golang-migrate`   | Simples, baseado em arquivos SQL versionados    |
| Auth         | `golang-jwt/jwt`   | JWT padrão                                      |
| Config       | `godotenv`         | `.env` local, env vars em produção              |
| Senhas       | `bcrypt`           | Hash seguro                                     |

---

## Exemplo de Fluxo: Compra com Desconto de Roleplay

```
POST /campaigns/abc/transactions
{
  "type": "buy",
  "character_id": "char-123",
  "items": [
    { "shop_item_id": "sword-01", "quantity": 1 },
    { "shop_item_id": "potion-02", "quantity": 3 }
  ]
}

→ Resposta (draft):
{
  "id": "tx-999",
  "status": "draft",
  "original_total": 150.00,
  "adjusted_total": 150.00,
  "items": [
    { "name": "Espada Longa", "quantity": 1, "unit_value": 100, "adjusted_unit_value": 100 },
    { "name": "Poção de Cura", "quantity": 3, "unit_value": 16.67, "adjusted_unit_value": 16.67 }
  ]
}

PATCH /campaigns/abc/transactions/tx-999
{
  "adjusted_total": 120.00,  // GM dá 20% de desconto
  "notes": "Desconto por ajudar o ferreiro no último capítulo"
}

→ Sistema redistribui proporcionalmente:
  Espada: 100 → 80
  Poção × 3: 50 → 40

POST /campaigns/abc/transactions/tx-999/confirm
→ Cria itens no inventário, debita 120 GP do personagem
```
