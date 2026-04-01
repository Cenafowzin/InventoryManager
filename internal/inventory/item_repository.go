package inventory

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rubendubeux/inventory-manager/models"
)

type ItemRepository struct {
	db *pgxpool.Pool
}

func NewItemRepository(db *pgxpool.Pool) *ItemRepository {
	return &ItemRepository{db: db}
}

type ItemFilters struct {
	CategoryID     *uuid.UUID
	StorageSpaceID *uuid.UUID
}

func (r *ItemRepository) CreateItem(ctx context.Context, characterID, storageSpaceID uuid.UUID, shopItemID, valueCoinID *uuid.UUID, name, description, emoji string, weightKg, value float64, quantity int) (*models.Item, error) {
	var id uuid.UUID
	err := r.db.QueryRow(ctx, `
		INSERT INTO items (character_id, storage_space_id, shop_item_id, name, description, emoji, weight_kg, value, value_coin_id, quantity)
		VALUES ($1, $2, $3, $4, NULLIF($5, ''), NULLIF($6, ''), $7, $8, $9, $10)
		RETURNING id
	`, characterID, storageSpaceID, shopItemID, name, description, emoji, weightKg, value, valueCoinID, quantity).Scan(&id)
	if err != nil {
		return nil, fmt.Errorf("create item: %w", err)
	}
	return r.GetItemByID(ctx, id)
}

func (r *ItemRepository) loadCategoriesForItems(ctx context.Context, items []models.Item) error {
	if len(items) == 0 {
		return nil
	}
	ids := make([]uuid.UUID, len(items))
	for i, it := range items {
		ids[i] = it.ID
	}
	rows, err := r.db.Query(ctx, `
		SELECT ic.item_id, c.id, c.campaign_id, c.name, c.color, c.created_at
		FROM item_categories ic
		JOIN categories c ON c.id = ic.category_id
		WHERE ic.item_id = ANY($1)
	`, ids)
	if err != nil {
		return fmt.Errorf("load item categories: %w", err)
	}
	defer rows.Close()

	catMap := make(map[uuid.UUID][]models.Category)
	for rows.Next() {
		var itemID uuid.UUID
		var cat models.Category
		if err := rows.Scan(&itemID, &cat.ID, &cat.CampaignID, &cat.Name, &cat.Color, &cat.CreatedAt); err != nil {
			return err
		}
		catMap[itemID] = append(catMap[itemID], cat)
	}
	for i := range items {
		if cats, ok := catMap[items[i].ID]; ok {
			items[i].Categories = cats
		} else {
			items[i].Categories = []models.Category{}
		}
	}
	return nil
}

func (r *ItemRepository) GetItemByID(ctx context.Context, id uuid.UUID) (*models.Item, error) {
	var item models.Item
	err := r.db.QueryRow(ctx, `
		SELECT i.id, i.character_id, i.storage_space_id, COALESCE(s.name, ''),
		       i.name, COALESCE(i.description, ''), COALESCE(i.emoji, ''),
		       i.weight_kg, i.value, i.value_coin_id, COALESCE(ct.abbreviation, ''),
		       i.quantity, i.shop_item_id, i.created_at, i.updated_at
		FROM items i
		LEFT JOIN storage_spaces s ON s.id = i.storage_space_id
		LEFT JOIN coin_types ct ON ct.id = i.value_coin_id
		WHERE i.id = $1
	`, id).Scan(
		&item.ID, &item.CharacterID, &item.StorageSpaceID, &item.StorageSpace,
		&item.Name, &item.Description, &item.Emoji,
		&item.WeightKg, &item.Value, &item.ValueCoinID, &item.ValueCoin,
		&item.Quantity, &item.ShopItemID, &item.CreatedAt, &item.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrItemNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get item: %w", err)
	}
	items := []models.Item{item}
	if err := r.loadCategoriesForItems(ctx, items); err != nil {
		return nil, err
	}
	return &items[0], nil
}

func (r *ItemRepository) ListItemsByCharacter(ctx context.Context, characterID uuid.UUID, filters ItemFilters) ([]models.Item, error) {
	query := `
		SELECT i.id, i.character_id, i.storage_space_id, COALESCE(s.name, ''),
		       i.name, COALESCE(i.description, ''), COALESCE(i.emoji, ''),
		       i.weight_kg, i.value, i.value_coin_id, COALESCE(ct.abbreviation, ''),
		       i.quantity, i.shop_item_id, i.created_at, i.updated_at
		FROM items i
		LEFT JOIN storage_spaces s ON s.id = i.storage_space_id
		LEFT JOIN coin_types ct ON ct.id = i.value_coin_id
		WHERE i.character_id = $1
	`
	args := []any{characterID}
	idx := 2

	if filters.CategoryID != nil {
		query += fmt.Sprintf(` AND EXISTS (SELECT 1 FROM item_categories ic WHERE ic.item_id = i.id AND ic.category_id = $%d)`, idx)
		args = append(args, *filters.CategoryID)
		idx++
	}
	if filters.StorageSpaceID != nil {
		query += fmt.Sprintf(` AND i.storage_space_id = $%d`, idx)
		args = append(args, *filters.StorageSpaceID)
	}
	query += ` ORDER BY i.created_at ASC`

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list items: %w", err)
	}
	defer rows.Close()

	var items []models.Item
	for rows.Next() {
		var item models.Item
		if err := rows.Scan(
			&item.ID, &item.CharacterID, &item.StorageSpaceID, &item.StorageSpace,
			&item.Name, &item.Description, &item.Emoji,
			&item.WeightKg, &item.Value, &item.ValueCoinID, &item.ValueCoin,
			&item.Quantity, &item.ShopItemID, &item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := r.loadCategoriesForItems(ctx, items); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *ItemRepository) UpdateItem(ctx context.Context, id, storageSpaceID uuid.UUID, valueCoinID *uuid.UUID, name, description, emoji string, weightKg, value float64, quantity int) (*models.Item, error) {
	tag, err := r.db.Exec(ctx, `
		UPDATE items
		SET storage_space_id = $1, name = $2, description = NULLIF($3, ''), emoji = NULLIF($4, ''),
		    weight_kg = $5, value = $6, value_coin_id = $7, quantity = $8, updated_at = NOW()
		WHERE id = $9
	`, storageSpaceID, name, description, emoji, weightKg, value, valueCoinID, quantity, id)
	if err != nil {
		return nil, fmt.Errorf("update item: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrItemNotFound
	}
	return r.GetItemByID(ctx, id)
}

func (r *ItemRepository) DeleteItem(ctx context.Context, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM items WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete item: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrItemNotFound
	}
	return nil
}

func (r *ItemRepository) DecrementQuantity(ctx context.Context, id uuid.UUID, qty int) error {
	_, err := r.db.Exec(ctx, `
		UPDATE items SET quantity = quantity - $1, updated_at = NOW() WHERE id = $2 AND quantity >= $1
	`, qty, id)
	return err
}

func (r *ItemRepository) TransferItem(ctx context.Context, sourceItemID, targetCharID uuid.UUID, quantity int) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// Lock and read source item
	var srcCharID uuid.UUID
	var shopItemID *uuid.UUID
	var name, description, emoji string
	var weightKg, value float64
	var valueCoinID *uuid.UUID
	var srcQty int
	err = tx.QueryRow(ctx, `
		SELECT character_id, shop_item_id, name,
		       COALESCE(description, ''), COALESCE(emoji, ''),
		       weight_kg, value, value_coin_id, quantity
		FROM items WHERE id = $1 FOR UPDATE
	`, sourceItemID).Scan(
		&srcCharID, &shopItemID, &name, &description, &emoji,
		&weightKg, &value, &valueCoinID, &srcQty,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrItemNotFound
	}
	if err != nil {
		return fmt.Errorf("lock source item: %w", err)
	}

	if srcCharID == targetCharID {
		return ErrSameCharacter
	}
	if srcQty < quantity {
		return ErrInsufficientQuantity
	}

	// Read source categories before any modification
	catRows, err := tx.Query(ctx, `SELECT category_id FROM item_categories WHERE item_id = $1`, sourceItemID)
	if err != nil {
		return fmt.Errorf("query source categories: %w", err)
	}
	var catIDs []uuid.UUID
	for catRows.Next() {
		var catID uuid.UUID
		if err := catRows.Scan(&catID); err != nil {
			catRows.Close()
			return err
		}
		catIDs = append(catIDs, catID)
	}
	catRows.Close()

	// Decrement or delete source
	if srcQty == quantity {
		if _, err = tx.Exec(ctx, `DELETE FROM items WHERE id = $1`, sourceItemID); err != nil {
			return fmt.Errorf("delete source item: %w", err)
		}
	} else {
		if _, err = tx.Exec(ctx, `UPDATE items SET quantity = quantity - $1, updated_at = NOW() WHERE id = $2`, quantity, sourceItemID); err != nil {
			return fmt.Errorf("decrement source item: %w", err)
		}
	}

	// Get target's default storage space
	var targetStorageID uuid.UUID
	err = tx.QueryRow(ctx, `SELECT id FROM storage_spaces WHERE character_id = $1 AND is_default = true LIMIT 1`, targetCharID).Scan(&targetStorageID)
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("target character has no default storage space")
	}
	if err != nil {
		return fmt.Errorf("get target storage: %w", err)
	}

	// Try to merge by shop_item_id
	merged := false
	if shopItemID != nil {
		var existingID uuid.UUID
		mergeErr := tx.QueryRow(ctx, `
			SELECT id FROM items WHERE character_id = $1 AND shop_item_id = $2 LIMIT 1 FOR UPDATE
		`, targetCharID, shopItemID).Scan(&existingID)
		if mergeErr == nil {
			if _, err = tx.Exec(ctx, `UPDATE items SET quantity = quantity + $1, updated_at = NOW() WHERE id = $2`, quantity, existingID); err != nil {
				return fmt.Errorf("merge item quantity: %w", err)
			}
			merged = true
		} else if !errors.Is(mergeErr, pgx.ErrNoRows) {
			return fmt.Errorf("check merge: %w", mergeErr)
		}
	}

	if !merged {
		var newItemID uuid.UUID
		err = tx.QueryRow(ctx, `
			INSERT INTO items (character_id, storage_space_id, shop_item_id, name, description, emoji, weight_kg, value, value_coin_id, quantity)
			VALUES ($1, $2, $3, $4, NULLIF($5, ''), NULLIF($6, ''), $7, $8, $9, $10)
			RETURNING id
		`, targetCharID, targetStorageID, shopItemID, name, description, emoji, weightKg, value, valueCoinID, quantity).Scan(&newItemID)
		if err != nil {
			return fmt.Errorf("insert transferred item: %w", err)
		}
		for _, catID := range catIDs {
			if _, err = tx.Exec(ctx, `INSERT INTO item_categories (item_id, category_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, newItemID, catID); err != nil {
				return fmt.Errorf("copy category: %w", err)
			}
		}
	}

	return tx.Commit(ctx)
}
