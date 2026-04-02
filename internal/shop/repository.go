package shop

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rubendubeux/inventory-manager/models"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// ── Shops ─────────────────────────────────────────────────────────────────────

func (r *Repository) CreateShop(ctx context.Context, campaignID uuid.UUID, name, color string) (*models.Shop, error) {
	var s models.Shop
	err := r.db.QueryRow(ctx, `
		INSERT INTO shops (campaign_id, name, color)
		VALUES ($1, $2, $3)
		RETURNING id, campaign_id, name, color, is_active, created_at, updated_at
	`, campaignID, name, color).Scan(
		&s.ID, &s.CampaignID, &s.Name, &s.Color, &s.IsActive, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create shop: %w", err)
	}
	return &s, nil
}

func (r *Repository) ListShops(ctx context.Context, campaignID uuid.UUID) ([]models.Shop, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, campaign_id, name, color, is_active, created_at, updated_at
		FROM shops WHERE campaign_id = $1 ORDER BY name ASC
	`, campaignID)
	if err != nil {
		return nil, fmt.Errorf("list shops: %w", err)
	}
	defer rows.Close()
	var shops []models.Shop
	for rows.Next() {
		var s models.Shop
		if err := rows.Scan(&s.ID, &s.CampaignID, &s.Name, &s.Color, &s.IsActive, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		shops = append(shops, s)
	}
	return shops, nil
}

func (r *Repository) UpdateShop(ctx context.Context, id uuid.UUID, name, color string, isActive bool) (*models.Shop, error) {
	var s models.Shop
	err := r.db.QueryRow(ctx, `
		UPDATE shops SET name = $1, color = $2, is_active = $3, updated_at = NOW()
		WHERE id = $4
		RETURNING id, campaign_id, name, color, is_active, created_at, updated_at
	`, name, color, isActive, id).Scan(
		&s.ID, &s.CampaignID, &s.Name, &s.Color, &s.IsActive, &s.CreatedAt, &s.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrShopNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("update shop: %w", err)
	}
	return &s, nil
}

func (r *Repository) DeleteShop(ctx context.Context, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM shops WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete shop: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrShopNotFound
	}
	return nil
}

// ── Shop items ────────────────────────────────────────────────────────────────

const shopItemCols = `
	si.id, si.campaign_id, si.name, COALESCE(si.description, ''), COALESCE(si.emoji, ''),
	si.weight_kg, si.base_value, si.value_coin_id, COALESCE(ct.abbreviation, ''),
	si.is_available, si.shop_id, COALESCE(s.name, ''), COALESCE(s.color, ''),
	si.stock_quantity, si.created_at, si.updated_at
`

func scanShopItem(row pgx.Row, item *models.ShopItem) error {
	return row.Scan(
		&item.ID, &item.CampaignID, &item.Name, &item.Description, &item.Emoji,
		&item.WeightKg, &item.BaseValue, &item.ValueCoinID, &item.ValueCoin,
		&item.IsAvailable, &item.ShopID, &item.ShopName, &item.ShopColor,
		&item.StockQuantity, &item.CreatedAt, &item.UpdatedAt,
	)
}

type ShopFilters struct {
	OnlyAvailable bool
	CategoryID    *uuid.UUID
	ShopID        *uuid.UUID
}

func (r *Repository) CreateShopItem(
	ctx context.Context,
	campaignID uuid.UUID,
	valueCoinID *uuid.UUID,
	shopID *uuid.UUID,
	name, description, emoji string,
	weightKg, baseValue float64,
	stockQuantity *int,
	isAvailable bool,
) (*models.ShopItem, error) {
	var id uuid.UUID
	err := r.db.QueryRow(ctx, `
		INSERT INTO shop_items (campaign_id, name, description, emoji, weight_kg, base_value, value_coin_id, is_available, shop_id, stock_quantity)
		VALUES ($1, $2, NULLIF($3, ''), NULLIF($4, ''), $5, $6, $7, $8, $9, $10)
		RETURNING id
	`, campaignID, name, description, emoji, weightKg, baseValue, valueCoinID, isAvailable, shopID, stockQuantity).Scan(&id)
	if err != nil {
		return nil, fmt.Errorf("create shop item: %w", err)
	}
	return r.GetShopItemByID(ctx, id)
}

func (r *Repository) GetShopItemByID(ctx context.Context, id uuid.UUID) (*models.ShopItem, error) {
	var item models.ShopItem
	err := scanShopItem(r.db.QueryRow(ctx, `
		SELECT `+shopItemCols+`
		FROM shop_items si
		LEFT JOIN coin_types ct ON ct.id = si.value_coin_id
		LEFT JOIN shops s ON s.id = si.shop_id
		WHERE si.id = $1
	`, id), &item)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrShopItemNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get shop item: %w", err)
	}
	items := []models.ShopItem{item}
	if err := r.loadCategoriesForItems(ctx, items); err != nil {
		return nil, err
	}
	return &items[0], nil
}

func (r *Repository) ListShopItems(ctx context.Context, campaignID uuid.UUID, filters ShopFilters) ([]models.ShopItem, error) {
	query := `
		SELECT ` + shopItemCols + `
		FROM shop_items si
		LEFT JOIN coin_types ct ON ct.id = si.value_coin_id
		LEFT JOIN shops s ON s.id = si.shop_id
		WHERE si.campaign_id = $1
	`
	args := []any{campaignID}
	idx := 2

	if filters.OnlyAvailable {
		query += ` AND si.is_available = true AND (si.shop_id IS NULL OR s.is_active = true)`
	}
	if filters.CategoryID != nil {
		query += fmt.Sprintf(` AND EXISTS (SELECT 1 FROM shop_item_categories sic WHERE sic.shop_item_id = si.id AND sic.category_id = $%d)`, idx)
		args = append(args, *filters.CategoryID)
		idx++
	}
	if filters.ShopID != nil {
		query += fmt.Sprintf(` AND si.shop_id = $%d`, idx)
		args = append(args, *filters.ShopID)
		idx++
	}
	query += ` ORDER BY si.name ASC`
	_ = idx

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list shop items: %w", err)
	}
	defer rows.Close()

	var items []models.ShopItem
	for rows.Next() {
		var item models.ShopItem
		if err := rows.Scan(
			&item.ID, &item.CampaignID, &item.Name, &item.Description, &item.Emoji,
			&item.WeightKg, &item.BaseValue, &item.ValueCoinID, &item.ValueCoin,
			&item.IsAvailable, &item.ShopID, &item.ShopName, &item.ShopColor,
			&item.StockQuantity, &item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list shop items: %w", err)
	}
	if err := r.loadCategoriesForItems(ctx, items); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *Repository) UpdateShopItem(
	ctx context.Context,
	id uuid.UUID,
	valueCoinID *uuid.UUID,
	shopID *uuid.UUID,
	name, description, emoji string,
	weightKg, baseValue float64,
	stockQuantity *int,
	isAvailable bool,
) (*models.ShopItem, error) {
	tag, err := r.db.Exec(ctx, `
		UPDATE shop_items
		SET name = $1, description = NULLIF($2, ''), emoji = NULLIF($3, ''),
		    weight_kg = $4, base_value = $5, value_coin_id = $6,
		    shop_id = $7, stock_quantity = $8, is_available = $9,
		    updated_at = NOW()
		WHERE id = $10
	`, name, description, emoji, weightKg, baseValue, valueCoinID, shopID, stockQuantity, isAvailable, id)
	if err != nil {
		return nil, fmt.Errorf("update shop item: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrShopItemNotFound
	}
	return r.GetShopItemByID(ctx, id)
}

func (r *Repository) DeleteShopItem(ctx context.Context, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM shop_items WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete shop item: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrShopItemNotFound
	}
	return nil
}

func (r *Repository) loadCategoriesForItems(ctx context.Context, items []models.ShopItem) error {
	if len(items) == 0 {
		return nil
	}
	ids := make([]uuid.UUID, len(items))
	for i, it := range items {
		ids[i] = it.ID
	}
	rows, err := r.db.Query(ctx, `
		SELECT sic.shop_item_id, c.id, c.campaign_id, c.name, COALESCE(c.color, ''), c.created_at
		FROM shop_item_categories sic
		JOIN categories c ON c.id = sic.category_id
		WHERE sic.shop_item_id = ANY($1)
	`, ids)
	if err != nil {
		return fmt.Errorf("load shop item categories: %w", err)
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
	if err := rows.Err(); err != nil {
		return fmt.Errorf("load shop item categories: %w", err)
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
