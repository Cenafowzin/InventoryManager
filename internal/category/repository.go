package category

import (
	"context"
	"errors"
	"fmt"
	"strings"

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

func (r *Repository) CreateCategory(ctx context.Context, campaignID uuid.UUID, name, color string) (*models.Category, error) {
	var c models.Category
	err := r.db.QueryRow(ctx, `
		INSERT INTO categories (campaign_id, name, color)
		VALUES ($1, $2, NULLIF($3, ''))
		RETURNING id, campaign_id, name, COALESCE(color, ''), created_at
	`, campaignID, name, color).Scan(&c.ID, &c.CampaignID, &c.Name, &c.Color, &c.CreatedAt)
	if err != nil {
		if strings.Contains(err.Error(), "unique") {
			return nil, ErrDuplicateName
		}
		return nil, fmt.Errorf("create category: %w", err)
	}
	return &c, nil
}

func (r *Repository) GetCategoryByID(ctx context.Context, id uuid.UUID) (*models.Category, error) {
	var c models.Category
	err := r.db.QueryRow(ctx, `
		SELECT id, campaign_id, name, COALESCE(color, ''), created_at
		FROM categories WHERE id = $1
	`, id).Scan(&c.ID, &c.CampaignID, &c.Name, &c.Color, &c.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrCategoryNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get category: %w", err)
	}
	return &c, nil
}

func (r *Repository) ListCategories(ctx context.Context, campaignID uuid.UUID) ([]models.Category, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, campaign_id, name, COALESCE(color, ''), created_at
		FROM categories WHERE campaign_id = $1
		ORDER BY name ASC
	`, campaignID)
	if err != nil {
		return nil, fmt.Errorf("list categories: %w", err)
	}
	defer rows.Close()

	var cats []models.Category
	for rows.Next() {
		var c models.Category
		if err := rows.Scan(&c.ID, &c.CampaignID, &c.Name, &c.Color, &c.CreatedAt); err != nil {
			return nil, err
		}
		cats = append(cats, c)
	}
	return cats, nil
}

func (r *Repository) UpdateCategory(ctx context.Context, id uuid.UUID, name, color string) (*models.Category, error) {
	var c models.Category
	err := r.db.QueryRow(ctx, `
		UPDATE categories SET name = $1, color = NULLIF($2, '')
		WHERE id = $3
		RETURNING id, campaign_id, name, COALESCE(color, ''), created_at
	`, name, color, id).Scan(&c.ID, &c.CampaignID, &c.Name, &c.Color, &c.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrCategoryNotFound
	}
	if err != nil {
		if strings.Contains(err.Error(), "unique") {
			return nil, ErrDuplicateName
		}
		return nil, fmt.Errorf("update category: %w", err)
	}
	return &c, nil
}

func (r *Repository) DeleteCategory(ctx context.Context, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM categories WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete category: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrCategoryNotFound
	}
	return nil
}

// SetItemCategories substitui todas as categorias de um item.
func (r *Repository) SetItemCategories(ctx context.Context, itemID uuid.UUID, categoryIDs []uuid.UUID) error {
	return r.setEntityCategories(ctx, "item_categories", "item_id", itemID, categoryIDs)
}

// SetShopItemCategories substitui todas as categorias de um shop item.
func (r *Repository) SetShopItemCategories(ctx context.Context, shopItemID uuid.UUID, categoryIDs []uuid.UUID) error {
	return r.setEntityCategories(ctx, "shop_item_categories", "shop_item_id", shopItemID, categoryIDs)
}

func (r *Repository) setEntityCategories(ctx context.Context, table, idCol string, entityID uuid.UUID, categoryIDs []uuid.UUID) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, fmt.Sprintf(`DELETE FROM %s WHERE %s = $1`, table, idCol), entityID); err != nil {
		return fmt.Errorf("clear categories: %w", err)
	}

	for _, catID := range categoryIDs {
		if _, err := tx.Exec(ctx,
			fmt.Sprintf(`INSERT INTO %s (%s, category_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, table, idCol),
			entityID, catID,
		); err != nil {
			return fmt.Errorf("insert category: %w", err)
		}
	}

	return tx.Commit(ctx)
}

// GetItemCategories retorna as categorias de um item.
func (r *Repository) GetItemCategories(ctx context.Context, itemID uuid.UUID) ([]models.Category, error) {
	return r.getEntityCategories(ctx, "item_categories", "item_id", itemID)
}

// GetShopItemCategories retorna as categorias de um shop item.
func (r *Repository) GetShopItemCategories(ctx context.Context, shopItemID uuid.UUID) ([]models.Category, error) {
	return r.getEntityCategories(ctx, "shop_item_categories", "shop_item_id", shopItemID)
}

func (r *Repository) getEntityCategories(ctx context.Context, joinTable, idCol string, entityID uuid.UUID) ([]models.Category, error) {
	rows, err := r.db.Query(ctx, fmt.Sprintf(`
		SELECT c.id, c.campaign_id, c.name, COALESCE(c.color, ''), c.created_at
		FROM categories c
		JOIN %s j ON j.category_id = c.id
		WHERE j.%s = $1
		ORDER BY c.name ASC
	`, joinTable, idCol), entityID)
	if err != nil {
		return nil, fmt.Errorf("get categories: %w", err)
	}
	defer rows.Close()

	var cats []models.Category
	for rows.Next() {
		var c models.Category
		if err := rows.Scan(&c.ID, &c.CampaignID, &c.Name, &c.Color, &c.CreatedAt); err != nil {
			return nil, err
		}
		cats = append(cats, c)
	}
	return cats, nil
}

// ValidateCategoryIDs verifica se todos os IDs pertencem à campanha.
func (r *Repository) ValidateCategoryIDs(ctx context.Context, campaignID uuid.UUID, ids []uuid.UUID) error {
	if len(ids) == 0 {
		return nil
	}
	var count int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM categories
		WHERE campaign_id = $1 AND id = ANY($2)
	`, campaignID, ids).Scan(&count)
	if err != nil {
		return fmt.Errorf("validate categories: %w", err)
	}
	if count != len(ids) {
		return fmt.Errorf("one or more category_ids do not belong to this campaign")
	}
	return nil
}
