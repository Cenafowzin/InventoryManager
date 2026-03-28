# Roadmap de Implementação

Cada fase tem seu próprio documento com migration, modelos, repositório, serviço, handlers, testes manuais e critérios de aceite.

| Fase | Documento | Entregável |
|------|-----------|------------|
| 1 | [01_foundation.md](phases/01_foundation.md) | Projeto Go rodando, banco conectado, `GET /health` |
| 2 | [02_auth.md](phases/02_auth.md) | Registro, login, JWT, middleware de autenticação |
| 3 | [03_campaigns.md](phases/03_campaigns.md) | CRUD de campanhas, membros, roles (gm/player), middleware de role |
| 4 | [04_coins.md](phases/04_coins.md) | Tipos de moeda, moeda padrão, conversão bidirecional automática |
| 5 | [05_characters.md](phases/05_characters.md) | CRUD de personagens com controle de ownership |
| 6 | [06_inventory.md](phases/06_inventory.md) | Categorias (tags), espaços de armazenamento, itens, carga do personagem, moedas |
| 7 | [07_shop.md](phases/07_shop.md) | Loja da campanha com categorias (GM gerencia, player consulta) |
| 8 | [08_transactions.md](phases/08_transactions.md) | Compra/venda com draft → ajuste → confirm |

## Dependências entre fases

```
1 (fundação)
└── 2 (auth)
    └── 3 (campanhas)
        ├── 4 (moedas)
        │   └── 6 (inventário) ←─┐
        ├── 5 (personagens)  ────┘
        │   └── 6 (inventário)
        │       └── 7 (loja)
        │           └── 8 (transações)
        └── 7 (loja)
```

## Stack

| Componente   | Lib                  |
|--------------|----------------------|
| Router       | `chi`                |
| Banco        | PostgreSQL 16        |
| Driver       | `pgx/v5`             |
| Migrations   | `golang-migrate`     |
| Auth         | `golang-jwt/jwt/v5`  |
| Senha        | `bcrypt`             |
| Config       | `godotenv`           |
| UUIDs        | `google/uuid`        |
