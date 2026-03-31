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

// listar itens da loja
type ShopFilters struct {
	OnlyAvailable bool
	CategoryID    *uuid.UUID
	Limit         int
	Offset        int
}

// Cria um novo item na loja e retorna o item completo
func (r *Repository) CreateShopItem(
	ctx context.Context,
	campaignID uuid.UUID,
	valueCoinID *uuid.UUID,
	name, description, emoji string,
	weightKg, baseValue float64,
	isAvailable bool,
) (*models.ShopItem, error) {
	var id uuid.UUID

	err := r.db.QueryRow(ctx, `
		INSERT INTO shop_items (campaign_id, name, description, emoji, weight_kg, base_value, value_coin_id, is_available)
		VALUES ($1, $2, NULLIF($3, ''), NULLIF($4, ''), $5, $6, $7, $8)
		RETURNING id
	`, campaignID, name, description, emoji, weightKg, baseValue, valueCoinID, isAvailable).Scan(&id)
	if err != nil {
		return nil, fmt.Errorf("create shop item: %w", err)
	}

	return r.GetShopItemByID(ctx, id)
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

	// Atribuir categoria
	for i := range items {
		if cats, ok := catMap[items[i].ID]; ok {
			items[i].Categories = cats
		} else {
			items[i].Categories = []models.Category{} // Item sem categorias
		}
	}

	return nil
}

// Busca um item específico pelo ID
func (r *Repository) GetShopItemByID(ctx context.Context, id uuid.UUID) (*models.ShopItem, error) {
	var item models.ShopItem

	err := r.db.QueryRow(ctx, `
		SELECT si.id, si.campaign_id, si.name, COALESCE(si.description, ''), COALESCE(si.emoji, ''),
			   si.weight_kg, si.base_value, si.value_coin_id, COALESCE(ct.abbreviation, ''),
			   si.is_available, si.created_at, si.updated_at
		FROM shop_items si
		LEFT JOIN coin_types ct ON ct.id = si.value_coin_id
		WHERE si.id = $1
	`, id).Scan(
		&item.ID, &item.CampaignID, &item.Name, &item.Description, &item.Emoji,
		&item.WeightKg, &item.BaseValue, &item.ValueCoinID, &item.ValueCoin,
		&item.IsAvailable, &item.CreatedAt, &item.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrShopItemNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get shop item: %w", err)
	}

	// Carrega as categorias do item
	items := []models.ShopItem{item}
	if err := r.loadCategoriesForItems(ctx, items); err != nil {
		return nil, err
	}

	return &items[0], nil
}

func (r *Repository) ListShopItems(ctx context.Context, campaignID uuid.UUID, filters ShopFilters) ([]models.ShopItem, error) {
	query := `
		SELECT si.id, si.campaign_id, si.name, COALESCE(si.description, ''), COALESCE(si.emoji, ''),
			   si.weight_kg, si.base_value, si.value_coin_id, COALESCE(ct.abbreviation, ''),
			   si.is_available, si.created_at, si.updated_at
		FROM shop_items si
		LEFT JOIN coin_types ct ON ct.id = si.value_coin_id
		WHERE si.campaign_id = $1
	`
	args := []any{campaignID}
	idx := 2

	if filters.OnlyAvailable {
		query += ` AND si.is_available = true`
	}
	if filters.CategoryID != nil {

		query += fmt.Sprintf(` AND EXISTS (
			SELECT 1
			FROM shop_item_categories sic
			WHERE sic.shop_item_id = si.id AND sic.category_id = $%d
		)`, idx)
		args = append(args, *filters.CategoryID)
		idx++
	}

	query += ` ORDER BY si.name ASC`

	if filters.Limit > 0 {
		query += fmt.Sprintf(` LIMIT $%d`, idx)
		args = append(args, filters.Limit)
		idx++
	}
	if filters.Offset > 0 {
		query += fmt.Sprintf(` OFFSET $%d`, idx)
		args = append(args, filters.Offset)
		idx++
	}

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
			&item.IsAvailable, &item.CreatedAt, &item.UpdatedAt,
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

// Retorna o item atualizado completo
func (r *Repository) UpdateShopItem(
	ctx context.Context,
	id uuid.UUID,
	valueCoinID *uuid.UUID,
	name, description, emoji string,
	weightKg, baseValue float64,
	isAvailable *bool,
) (*models.ShopItem, error) {
	query := `UPDATE shop_items si SET `
	args := []any{}
	idx := 1

	if name != "" {
		query += fmt.Sprintf(`name = $%d, `, idx)
		args = append(args, name)
		idx++
	}
	if description != "" {
		query += fmt.Sprintf(`description = NULLIF($%d, ''), `, idx)
		args = append(args, description)
		idx++
	}
	if emoji != "" {
		query += fmt.Sprintf(`emoji = NULLIF($%d, ''), `, idx)
		args = append(args, emoji)
		idx++
	}
	if weightKg >= 0 {
		query += fmt.Sprintf(`weight_kg = $%d, `, idx)
		args = append(args, weightKg)
		idx++
	}
	if baseValue >= 0 {
		query += fmt.Sprintf(`base_value = $%d, `, idx)
		args = append(args, baseValue)
		idx++
	}
	if valueCoinID != nil {
		query += fmt.Sprintf(`value_coin_id = $%d, `, idx)
		args = append(args, *valueCoinID)
		idx++
	}
	if isAvailable != nil {
		query += fmt.Sprintf(`is_available = $%d, `, idx)
		args = append(args, *isAvailable)
		idx++
	}

	if idx == 1 {
		return r.GetShopItemByID(ctx, id)
	}

	query += fmt.Sprintf(`
		updated_at = NOW()
		WHERE si.id = $%d
		RETURNING si.id, si.campaign_id, si.name, COALESCE(si.description, ''), COALESCE(si.emoji, ''),
				  si.weight_kg, si.base_value, si.value_coin_id,
				  COALESCE((SELECT ct.abbreviation FROM coin_types ct WHERE ct.id = si.value_coin_id), ''),
				  si.is_available, si.created_at, si.updated_at
	`, idx)
	args = append(args, id)

	var item models.ShopItem
	err := r.db.QueryRow(ctx, query, args...).Scan(
		&item.ID, &item.CampaignID, &item.Name, &item.Description, &item.Emoji,
		&item.WeightKg, &item.BaseValue, &item.ValueCoinID, &item.ValueCoin,
		&item.IsAvailable, &item.CreatedAt, &item.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrShopItemNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("update shop item: %w", err)
	}

	// Carrega as categorias do item atualizado
	items := []models.ShopItem{item}
	if err := r.loadCategoriesForItems(ctx, items); err != nil {
		return nil, err
	}

	return &items[0], nil
}

// Remove um item da loja pelo ID
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
