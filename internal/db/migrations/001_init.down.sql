DROP TABLE IF EXISTS transaction_items;
DROP TABLE IF EXISTS transactions;
DROP TABLE IF EXISTS coin_purse;
DROP TABLE IF EXISTS item_categories;
DROP TABLE IF EXISTS items;
DROP TABLE IF EXISTS shop_item_categories;
DROP TABLE IF EXISTS shop_items;
DROP TABLE IF EXISTS shops;
DROP TABLE IF EXISTS storage_spaces;
DROP TABLE IF EXISTS characters;
DROP TABLE IF EXISTS categories;
DROP TABLE IF EXISTS coin_conversions;
DROP TABLE IF EXISTS coin_types;
DROP TABLE IF EXISTS campaign_invites;
DROP TABLE IF EXISTS campaign_members;
DROP TABLE IF EXISTS campaigns;
DROP TABLE IF EXISTS users;

DROP TYPE IF EXISTS transaction_status;
DROP TYPE IF EXISTS transaction_type;
DROP TYPE IF EXISTS campaign_role;

DROP EXTENSION IF EXISTS "pgcrypto";
