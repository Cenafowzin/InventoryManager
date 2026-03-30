package coin

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

// ── CoinTypes ─────────────────────────────────────────────────────────────────

func (r *Repository) CreateCoinType(ctx context.Context, campaignID uuid.UUID, name, abbreviation, emoji string) (*models.CoinType, error) {
	var c models.CoinType
	err := r.db.QueryRow(ctx, `
		INSERT INTO coin_types (campaign_id, name, abbreviation, emoji)
		VALUES ($1, $2, $3, $4)
		RETURNING id, campaign_id, name, abbreviation, emoji, is_default, created_at
	`, campaignID, name, abbreviation, emoji).
		Scan(&c.ID, &c.CampaignID, &c.Name, &c.Abbreviation, &c.Emoji, &c.IsDefault, &c.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("create coin type: %w", err)
	}
	return &c, nil
}

func (r *Repository) GetCoinTypeByID(ctx context.Context, id uuid.UUID) (*models.CoinType, error) {
	var c models.CoinType
	err := r.db.QueryRow(ctx, `
		SELECT id, campaign_id, name, abbreviation, emoji, is_default, created_at
		FROM coin_types WHERE id = $1
	`, id).Scan(&c.ID, &c.CampaignID, &c.Name, &c.Abbreviation, &c.Emoji, &c.IsDefault, &c.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrCoinNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get coin type: %w", err)
	}
	return &c, nil
}

func (r *Repository) GetDefaultCoin(ctx context.Context, campaignID uuid.UUID) (*models.CoinType, error) {
	var c models.CoinType
	err := r.db.QueryRow(ctx, `
		SELECT id, campaign_id, name, abbreviation, emoji, is_default, created_at
		FROM coin_types WHERE campaign_id = $1 AND is_default = TRUE
	`, campaignID).Scan(&c.ID, &c.CampaignID, &c.Name, &c.Abbreviation, &c.Emoji, &c.IsDefault, &c.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNoDefaultCoin
	}
	if err != nil {
		return nil, fmt.Errorf("get default coin: %w", err)
	}
	return &c, nil
}

func (r *Repository) ListCoinTypes(ctx context.Context, campaignID uuid.UUID) ([]models.CoinType, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, campaign_id, name, abbreviation, emoji, is_default, created_at
		FROM coin_types WHERE campaign_id = $1
		ORDER BY is_default DESC, name ASC
	`, campaignID)
	if err != nil {
		return nil, fmt.Errorf("list coin types: %w", err)
	}
	defer rows.Close()

	var coins []models.CoinType
	for rows.Next() {
		var c models.CoinType
		if err := rows.Scan(&c.ID, &c.CampaignID, &c.Name, &c.Abbreviation, &c.Emoji, &c.IsDefault, &c.CreatedAt); err != nil {
			return nil, err
		}
		coins = append(coins, c)
	}
	return coins, nil
}

func (r *Repository) UpdateCoinType(ctx context.Context, id uuid.UUID, name, abbreviation, emoji string) (*models.CoinType, error) {
	var c models.CoinType
	err := r.db.QueryRow(ctx, `
		UPDATE coin_types SET name = $1, abbreviation = $2, emoji = $3
		WHERE id = $4
		RETURNING id, campaign_id, name, abbreviation, emoji, is_default, created_at
	`, name, abbreviation, emoji, id).
		Scan(&c.ID, &c.CampaignID, &c.Name, &c.Abbreviation, &c.Emoji, &c.IsDefault, &c.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrCoinNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("update coin type: %w", err)
	}
	return &c, nil
}

func (r *Repository) DeleteCoinType(ctx context.Context, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM coin_types WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete coin type: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrCoinNotFound
	}
	return nil
}

func (r *Repository) SetDefaultCoin(ctx context.Context, campaignID, coinID uuid.UUID) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `
		UPDATE coin_types SET is_default = FALSE WHERE campaign_id = $1
	`, campaignID); err != nil {
		return fmt.Errorf("unset default: %w", err)
	}

	tag, err := tx.Exec(ctx, `
		UPDATE coin_types SET is_default = TRUE WHERE id = $1 AND campaign_id = $2
	`, coinID, campaignID)
	if err != nil {
		return fmt.Errorf("set default: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrCoinNotFound
	}

	return tx.Commit(ctx)
}

// ── CoinConversions ───────────────────────────────────────────────────────────

func (r *Repository) PairExists(ctx context.Context, fromID, toID uuid.UUID) (bool, error) {
	var count int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM coin_conversions
		WHERE (from_coin_id = $1 AND to_coin_id = $2)
		   OR (from_coin_id = $2 AND to_coin_id = $1)
	`, fromID, toID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check pair exists: %w", err)
	}
	return count > 0, nil
}

func (r *Repository) CreateConversionPair(ctx context.Context, campaignID, fromID, toID uuid.UUID, rate float64) ([]models.CoinConversion, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	scanConversion := func(row pgx.Row) (models.CoinConversion, error) {
		var c models.CoinConversion
		err := row.Scan(&c.ID, &c.CampaignID, &c.FromCoinID, &c.FromCoin, &c.ToCoinID, &c.ToCoin, &c.Rate)
		return c, err
	}

	q := `
		WITH ins AS (
			INSERT INTO coin_conversions (campaign_id, from_coin_id, to_coin_id, rate, is_canonical)
			VALUES ($1, $2, $3, $4, $5)
			RETURNING id, campaign_id, from_coin_id, to_coin_id, rate
		)
		SELECT i.id, i.campaign_id, i.from_coin_id, f.abbreviation, i.to_coin_id, t.abbreviation, i.rate
		FROM ins i
		JOIN coin_types f ON f.id = i.from_coin_id
		JOIN coin_types t ON t.id = i.to_coin_id
	`

	fwd, err := scanConversion(tx.QueryRow(ctx, q, campaignID, fromID, toID, rate, true))
	if err != nil {
		return nil, fmt.Errorf("insert forward conversion: %w", err)
	}

	inv, err := scanConversion(tx.QueryRow(ctx, q, campaignID, toID, fromID, 1.0/rate, false))
	if err != nil {
		return nil, fmt.Errorf("insert inverse conversion: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	return []models.CoinConversion{fwd, inv}, nil
}

func (r *Repository) ListConversions(ctx context.Context, campaignID uuid.UUID) ([]models.CoinConversion, error) {
	rows, err := r.db.Query(ctx, `
		SELECT cc.id, cc.campaign_id, cc.from_coin_id, f.abbreviation, cc.to_coin_id, t.abbreviation, cc.rate
		FROM coin_conversions cc
		JOIN coin_types f ON f.id = cc.from_coin_id
		JOIN coin_types t ON t.id = cc.to_coin_id
		WHERE cc.campaign_id = $1 AND cc.is_canonical = TRUE
		ORDER BY f.name, t.name
	`, campaignID)
	if err != nil {
		return nil, fmt.Errorf("list conversions: %w", err)
	}
	defer rows.Close()

	var convs []models.CoinConversion
	for rows.Next() {
		var c models.CoinConversion
		if err := rows.Scan(&c.ID, &c.CampaignID, &c.FromCoinID, &c.FromCoin, &c.ToCoinID, &c.ToCoin, &c.Rate); err != nil {
			return nil, err
		}
		convs = append(convs, c)
	}
	return convs, nil
}

func (r *Repository) GetConversionByID(ctx context.Context, id uuid.UUID) (*models.CoinConversion, error) {
	var c models.CoinConversion
	err := r.db.QueryRow(ctx, `
		SELECT cc.id, cc.campaign_id, cc.from_coin_id, f.abbreviation, cc.to_coin_id, t.abbreviation, cc.rate
		FROM coin_conversions cc
		JOIN coin_types f ON f.id = cc.from_coin_id
		JOIN coin_types t ON t.id = cc.to_coin_id
		WHERE cc.id = $1
	`, id).Scan(&c.ID, &c.CampaignID, &c.FromCoinID, &c.FromCoin, &c.ToCoinID, &c.ToCoin, &c.Rate)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrConversionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get conversion: %w", err)
	}
	return &c, nil
}

func (r *Repository) DeleteConversionPair(ctx context.Context, id uuid.UUID) error {
	// Busca o par a partir de qualquer um dos dois IDs da conversão
	var fromID, toID uuid.UUID
	err := r.db.QueryRow(ctx, `
		SELECT from_coin_id, to_coin_id FROM coin_conversions WHERE id = $1
	`, id).Scan(&fromID, &toID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrConversionNotFound
	}
	if err != nil {
		return fmt.Errorf("get conversion for delete: %w", err)
	}

	_, err = r.db.Exec(ctx, `
		DELETE FROM coin_conversions
		WHERE (from_coin_id = $1 AND to_coin_id = $2)
		   OR (from_coin_id = $2 AND to_coin_id = $1)
	`, fromID, toID)
	return err
}

// IsInUse verifica se a moeda tem conversões, itens ou saldo associados.
func (r *Repository) IsInUse(ctx context.Context, coinID uuid.UUID) (bool, error) {
	var count int
	err := r.db.QueryRow(ctx, `
		SELECT (
			SELECT COUNT(*) FROM coin_conversions WHERE from_coin_id = $1 OR to_coin_id = $1
		) + (
			SELECT COUNT(*) FROM coin_purse WHERE coin_type_id = $1 AND amount > 0
		) + (
			SELECT COUNT(*) FROM items WHERE value_coin_id = $1
		)
	`, coinID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check coin usage: %w", err)
	}
	return count > 0, nil
}
