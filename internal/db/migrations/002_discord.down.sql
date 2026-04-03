DROP TABLE IF EXISTS discord_link_codes;
DROP TABLE IF EXISTS discord_links;
ALTER TABLE campaigns DROP COLUMN IF EXISTS discord_channel_id;
ALTER TABLE campaigns DROP COLUMN IF EXISTS discord_guild_id;
