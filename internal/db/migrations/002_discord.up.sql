ALTER TABLE campaigns ADD COLUMN discord_channel_id TEXT UNIQUE;
ALTER TABLE campaigns ADD COLUMN discord_guild_id   TEXT UNIQUE;

CREATE TABLE discord_links (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID        NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    discord_id  TEXT        NOT NULL UNIQUE,
    discord_tag TEXT        NOT NULL,
    linked_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE discord_link_codes (
    code       TEXT        PRIMARY KEY,
    user_id    UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
