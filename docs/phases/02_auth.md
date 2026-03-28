# Fase 2 — Autenticação

## Objetivo
Implementar registro, login e proteção de rotas via JWT.

## Pré-requisito
Fase 1 concluída.

## Entregável testável
- `POST /auth/register` cria usuário
- `POST /auth/login` retorna JWT + refresh token
- `POST /auth/refresh` renova o access token
- Rotas protegidas rejeitam requisições sem token válido

---

## Passos

### 2.1 — Migration: tabela `users`

**`002_create_users.up.sql`**

```sql
CREATE TABLE users (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    username    VARCHAR(50)  NOT NULL UNIQUE,
    email       VARCHAR(255) NOT NULL UNIQUE,
    password_hash TEXT       NOT NULL,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
```

**`002_create_users.down.sql`**

```sql
DROP TABLE IF EXISTS users;
```

---

### 2.2 — Dependências

```bash
go get github.com/golang-jwt/jwt/v5
go get golang.org/x/crypto
```

---

### 2.3 — Variáveis de ambiente adicionais

```env
JWT_SECRET=sua-chave-secreta-longa-aqui
JWT_ACCESS_TTL=15m
JWT_REFRESH_TTL=168h
```

---

### 2.4 — Modelo

**`models/user.go`**

```go
type User struct {
    ID           uuid.UUID
    Username     string
    Email        string
    PasswordHash string
    CreatedAt    time.Time
}
```

---

### 2.5 — Repositório

**`internal/auth/repository.go`**

Métodos:
- `CreateUser(ctx, username, email, passwordHash) (*User, error)`
- `GetUserByEmail(ctx, email) (*User, error)`
- `GetUserByID(ctx, id) (*User, error)`

---

### 2.6 — Serviço

**`internal/auth/service.go`**

**`Register(username, email, password string)`**:
1. Validar campos (email válido, senha mínimo 8 chars)
2. Checar se email/username já existem → `ErrEmailTaken`, `ErrUsernameTaken`
3. Hash da senha com `bcrypt` (cost 12)
4. Chamar `repo.CreateUser`
5. Retornar usuário criado (sem o hash)

**`Login(email, password string)`**:
1. `repo.GetUserByEmail`
2. `bcrypt.CompareHashAndPassword`
3. Gerar access token (JWT, TTL curto, claims: `user_id`, `username`, `exp`)
4. Gerar refresh token (JWT, TTL longo, claims: `user_id`, `exp`)
5. Retornar ambos os tokens

**`Refresh(refreshToken string)`**:
1. Validar e parsear refresh token
2. Checar expiração
3. `repo.GetUserByID` para confirmar usuário ainda existe
4. Gerar novo access token
5. Retornar novo access token

---

### 2.7 — Handlers

**`internal/auth/handler.go`**

`POST /auth/register`
- Body: `{ "username": "...", "email": "...", "password": "..." }`
- Resposta `201`: `{ "id": "...", "username": "...", "email": "..." }`
- Erros: `400` (validação), `409` (email/username duplicado)

`POST /auth/login`
- Body: `{ "email": "...", "password": "..." }`
- Resposta `200`: `{ "access_token": "...", "refresh_token": "...", "expires_in": 900 }`
- Erros: `401` (credenciais inválidas)

`POST /auth/refresh`
- Body: `{ "refresh_token": "..." }`
- Resposta `200`: `{ "access_token": "...", "expires_in": 900 }`
- Erros: `401` (token inválido ou expirado)

---

### 2.8 — Middleware de autenticação

**`pkg/middleware/auth.go`**

- Extrai `Authorization: Bearer <token>` do header
- Valida e parseia o JWT
- Injeta `user_id` no context da requisição via `context.WithValue`
- Retorna `401` se ausente ou inválido

Helper `UserIDFromContext(ctx) (uuid.UUID, bool)` — usado pelos handlers nas próximas fases.

---

### 2.9 — Registrar rotas no main.go

```go
r.Post("/auth/register", authHandler.Register)
r.Post("/auth/login",    authHandler.Login)
r.Post("/auth/refresh",  authHandler.Refresh)

// Grupo protegido (próximas fases usarão este grupo)
r.Group(func(r chi.Router) {
    r.Use(middleware.Authenticate)
    // rotas protegidas aqui
})
```

---

### 2.10 — Estrutura de pastas desta fase

```
internal/
└── auth/
    ├── handler.go
    ├── service.go
    └── repository.go
pkg/
└── middleware/
    └── auth.go
models/
└── user.go
internal/db/migrations/
└── 002_create_users.up.sql
└── 002_create_users.down.sql
```

---

## Testes Manuais

```bash
# Registrar
curl -X POST http://localhost:8080/auth/register \
  -H "Content-Type: application/json" \
  -d '{"username":"rodrigo","email":"r@test.com","password":"senha1234"}'

# Login
curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"r@test.com","password":"senha1234"}'
# Salvar o access_token retornado

# Refresh
curl -X POST http://localhost:8080/auth/refresh \
  -H "Content-Type: application/json" \
  -d '{"refresh_token":"<refresh_token>"}'

# Rota protegida sem token (deve retornar 401)
curl http://localhost:8080/campaigns
```

---

## Critérios de aceite

- [ ] Registro com email duplicado retorna `409`
- [ ] Login com senha errada retorna `401`
- [ ] Login correto retorna access + refresh token
- [ ] Refresh com token válido retorna novo access token
- [ ] Refresh com token expirado/inválido retorna `401`
- [ ] Rota protegida sem token retorna `401`
- [ ] Rota protegida com token válido passa pelo middleware
