
ALTER TABLE items DROP CONSTRAINT IF EXISTS fk_items_shop_item;
ALTER TABLE shop_item_categories DROP CONSTRAINT IF EXISTS fk_shop_item_categories_shop_item;
DROP TABLE IF EXISTS shop_items;