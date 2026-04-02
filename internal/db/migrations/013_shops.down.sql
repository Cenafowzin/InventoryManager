DROP INDEX IF EXISTS idx_shop_items_shop;
ALTER TABLE shop_items DROP COLUMN IF EXISTS shop_id;
ALTER TABLE shop_items DROP COLUMN IF EXISTS stock_quantity;
DROP TABLE IF EXISTS shops;
