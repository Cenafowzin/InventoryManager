# Discord Bot — Planejamento

## Visão Geral

Bot Discord integrado ao Inventory Manager. Roda como goroutine dentro do mesmo binário Go do servidor HTTP, compartilhando conexão com o banco e os serviços existentes.

```
Discord Gateway (WebSocket)
        │
        ▼
  internal/bot/          ← novo pacote
        │
        ├── usa diretamente os services existentes
        │       (character, inventory, shop, coin...)
        └── acessa banco via pool compartilhado

  cmd/api/main.go
        ├── go startHTTPServer()   ← API REST (atual)
        └── go startDiscordBot()   ← bot (novo)
```

---

## Variáveis de Ambiente

```env
DISCORD_TOKEN=Bot <token>    # token do bot no Developer Portal
DISCORD_GUILD_ID=            # ID do servidor (opcional — registra comandos instantaneamente em dev)
```

---

## Banco de Dados — Nova Migração

### `002_discord.up.sql`

```sql
-- Vinculação Discord ↔ campanha (canal tem prioridade sobre servidor)
ALTER TABLE campaigns ADD COLUMN discord_channel_id TEXT UNIQUE;
ALTER TABLE campaigns ADD COLUMN discord_guild_id   TEXT UNIQUE;

-- Vinculação conta Discord ↔ conta do site
CREATE TABLE discord_links (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID        NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    discord_id  TEXT        NOT NULL UNIQUE,
    discord_tag TEXT        NOT NULL,   -- ex: "rodrigo" (username novo do Discord)
    linked_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Códigos temporários para o fluxo de vinculação
CREATE TABLE discord_link_codes (
    code       TEXT        PRIMARY KEY,
    user_id    UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

---

## Vinculação Discord ↔ Campanha

Dois modos de configuração, escolhidos pelo GM no `/configurar`:

| Modo | Comportamento |
|---|---|
| `canal` | Só aquele canal responde por essa campanha. Ideal para servidores com várias campanhas. |
| `servidor` | Qualquer canal do servidor responde por essa campanha. Ideal para servidores dedicados a uma campanha. |

```
Exemplo com os dois modos no mesmo servidor:

#rpg-strahd    →  vinculado por canal  →  "A Maldição de Strahd"
#rpg-waterdeep →  vinculado por canal  →  "Waterdeep Dragon Heist"
#geral         →  sem canal, mas servidor vinculado → fallback para campanha do servidor
#off-topic     →  sem canal, mas servidor vinculado → fallback para campanha do servidor
```

### Comando `/configurar`
- Apenas GM pode executar
- Parâmetros: `campanha` (nome ou ID) + `modo` (`canal` | `servidor`)
- Bot captura `channel_id` e `guild_id` automaticamente do contexto
- Avisa se já existir vínculo antes de sobrescrever

### Resolução de campanha nos outros comandos
```
1. SELECT ... WHERE discord_channel_id = <channel_id>   → encontrou? usa
2. SELECT ... WHERE discord_guild_id   = <guild_id>     → encontrou? usa
3. Nenhum → ephemeral: "Canal não vinculado. GM use /configurar."
```

---

## Fluxo de Vinculação de Conta

```
1. Usuário digita /link no canal vinculado
2. Bot gera código aleatório (ex: "A3F7X2"), salva em discord_link_codes
   com expires_at = agora + 10 minutos
3. Bot responde ephemeral: "Acesse o site → Perfil → Vincular Discord
   e insira o código: A3F7X2  (expira em 10 min)"
4. No site, usuário logado insere o código
5. Site valida o código, salva em discord_links {user_id, discord_id, discord_tag}
   e deleta o código usado
6. A partir daí, todos os comandos resolvem user_id via discord_id
```

---

## Estrutura de Arquivos

```
internal/bot/
├── bot.go              — inicialização, registro de comandos, loop
├── resolver.go         — channel_id → campaignID / discord_id → userID
├── commands/
│   ├── configurar.go   — /configurar (GM vincula canal à campanha)
│   ├── link.go         — /link (vincula conta Discord ao site)
│   ├── personagens.go  — /personagens
│   ├── inventario.go   — /inventario
│   ├── moedas.go       — /moedas
│   └── loja.go         — /loja
└── format/
    └── format.go       — helpers de formatação das respostas embed

internal/discord/       — repositório para discord_links, link_codes e channel lookup
├── repository.go
└── service.go

internal/db/migrations/
└── 002_discord.up.sql
```

---

## Comandos — Fase 1 (Somente Leitura)

### `/configurar`
Vincula o Discord a uma campanha. Apenas GM.
- Parâmetros: `campanha` (nome ou ID) + `modo` (`canal` | `servidor`)
- `canal` → só aquele canal responde pela campanha
- `servidor` → qualquer canal do servidor responde (canal tem prioridade se configurado)

### `/link`
Inicia vinculação da conta Discord com a conta do site.
- Gera código de 6 chars aleatório
- Resposta ephemeral (só o usuário que chamou vê)

### `/personagens`
Lista os personagens do usuário na campanha do canal.
- Requer conta vinculada
- Mostra: nome, peso atual/máximo

### `/inventario [personagem]`
Mostra o inventário de um personagem.
- Se o usuário tiver só um personagem na campanha → mostra automaticamente
- Se tiver vários → exibe seletor ou aceita nome como parâmetro
- Exibe: itens agrupados por espaço de armazenamento, peso total, moedas

### `/moedas [personagem]`
Mostra a carteira de moedas de um personagem.
- Exibe todas as moedas com emoji e quantidade

### `/loja`
Lista os itens disponíveis na loja da campanha.
- Filtra `is_available = true` e loja ativa
- Agrupa por loja (se houver)
- Exibe: emoji, nome, preço

---

## Comandos — Fase 2 (Ações — futuro)

| Comando | Descrição |
|---|---|
| `/comprar <item> [personagem]` | Inicia transação de compra (draft) |
| `/confirmar` | Confirma o draft em aberto |
| `/vender <item> [personagem]` | Inicia transação de venda |
| `/transferir <item> <destino> <qtd>` | Transfere item entre personagens |
| `/desvincular` | Remove a vinculação Discord ↔ conta |

---

## Implementação — Ordem de Execução

1. **Migração** `002_discord.up.sql`
2. **`internal/discord/repository.go`** — CRUD para `discord_links`, `discord_link_codes` e lookup de `discord_channel_id`
3. **`internal/discord/service.go`** — `GenerateLinkCode`, `ResolveUser(discordID)`, `ResolveCampaign(channelID, guildID)`, `ConfirmLink(code, discordID, tag)`
4. **Endpoint no site** `POST /auth/discord/link` — recebe `{ code }` do usuário logado, confirma vinculação
5. **`internal/bot/bot.go`** — conecta ao gateway, registra slash commands, rota para handlers
6. **`internal/bot/resolver.go`** — helpers `CampaignFromContext(channelID, guildID)` (cascata canal→servidor) e `UserFromDiscord(discordID)`
7. **Comandos** na ordem: `/configurar` → `/link` → `/personagens` → `/moedas` → `/inventario` → `/loja`
8. **`cmd/api/main.go`** — adiciona `go bot.Start(pool, services...)` antes do `ListenAndServe`
9. **`.env` / Render** — adiciona `DISCORD_TOKEN`

---

## Dependência

```bash
go get github.com/bwmarrin/discordgo
```

---

## Notas

- Todos os comandos respondem **ephemeral** por padrão (só o usuário que chamou vê) para não poluir o chat
- Se `DISCORD_TOKEN` não estiver no `.env`, o bot simplesmente não sobe — não quebra o servidor HTTP
- `DISCORD_GUILD_ID` acelera o registro de slash commands durante desenvolvimento (instantâneo vs ~1h global)
- O bot nunca armazena dados sensíveis — apenas `discord_id` (string pública) e `discord_tag`
