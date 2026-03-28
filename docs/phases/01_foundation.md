# Fase 1 — Fundação do Projeto

## Objetivo
Montar a estrutura base do projeto: módulo Go, dependências, banco de dados, migrations e servidor HTTP rodando.

## Entregável testável
`GET /health` retorna `200 OK` com o banco conectado.

---

## Passos

### 1.1 — Inicializar módulo Go

```bash
go mod init github.com/<seu-user>/inventory-manager
```

Instalar dependências:

```bash
go get github.com/go-chi/chi/v5
go get github.com/jackc/pgx/v5
go get github.com/jackc/pgx/v5/pgxpool
go get github.com/golang-migrate/migrate/v4
go get github.com/golang-migrate/migrate/v4/database/postgres
go get github.com/golang-migrate/migrate/v4/source/file
go get github.com/joho/godotenv
go get github.com/google/uuid
```

---

### 1.2 — Variáveis de ambiente

Criar `.env`:

```env
DATABASE_URL=postgres://user:password@localhost:5432/inventory_manager?sslmode=disable
PORT=8080
```

Criar `.env.example` com as mesmas chaves sem valores (commitar este).
Adicionar `.env` ao `.gitignore`.

---

### 1.3 — Conexão com o banco

**`internal/db/db.go`**

- Função `Connect(databaseURL string) (*pgxpool.Pool, error)`
- Usa `pgxpool.New` com context
- Chama `pool.Ping` para validar a conexão

---

### 1.4 — Migrations

Criar pasta `internal/db/migrations/`.

**`001_create_users.up.sql`** — apenas placeholder por enquanto:
```sql
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
```

**`001_create_users.down.sql`**:
```sql
-- vazio por enquanto
```

**`internal/db/migrate.go`**

- Função `RunMigrations(databaseURL string)`
- Usa `golang-migrate` com source `file://internal/db/migrations`
- Loga a versão atual após rodar

---

### 1.5 — Servidor HTTP base

**`cmd/api/main.go`**

```
1. Carregar .env (godotenv.Load)
2. Conectar ao banco (db.Connect)
3. Rodar migrations (db.RunMigrations)
4. Criar router chi
5. Registrar GET /health
6. Iniciar http.ListenAndServe na PORT
```

**Handler `/health`**:
- Faz `pool.Ping` no banco
- Retorna `{"status": "ok", "db": "connected"}`

---

### 1.6 — Estrutura de pastas final desta fase

```
InventoryManager/
├── cmd/api/main.go
├── internal/db/
│   ├── db.go
│   ├── migrate.go
│   └── migrations/
│       ├── 001_init.up.sql
│       └── 001_init.down.sql
├── .env
├── .env.example
├── .gitignore
└── go.mod
```

---

## Teste Manual

```bash
# Subir postgres local (Docker)
docker run --name rpg-db -e POSTGRES_PASSWORD=password -e POSTGRES_DB=inventory_manager -p 5432:5432 -d postgres:16

# Rodar servidor
go run ./cmd/api

# Verificar health
curl http://localhost:8080/health
# Esperado: {"status":"ok","db":"connected"}
```

---

## Critérios de aceite

- [ ] `go build ./...` sem erros
- [ ] Banco conecta e migrations rodam sem erro
- [ ] `GET /health` retorna `200` com banco conectado
- [ ] `GET /health` retorna `503` se banco estiver fora
