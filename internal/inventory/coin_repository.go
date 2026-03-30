package inventory

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rubendubeux/inventory-manager/models"
)

type CoinRepository struct {
	db *pgxpool.Pool
}

func NewCoinRepository(db *pgxpool.Pool) *CoinRepository {
	return &CoinRepository{db: db}
}

func (r *CoinRepository) GetCoinPurse(ctx context.Context, characterID uuid.UUID) ([]models.CoinPurse, error) {
	rows, err := r.db.Query(ctx, `
		SELECT ct.id, ct.name, ct.abbreviation, COALESCE(ct.emoji, ''), COALESCE(cp.amount, 0)
		FROM coin_types ct
		JOIN characters ch ON ch.campaign_id = ct.campaign_id
		LEFT JOIN coin_purse cp ON cp.coin_type_id = ct.id AND cp.character_id = $1
		WHERE ch.id = $1
		ORDER BY ct.is_default DESC, ct.name ASC
	`, characterID)
	if err != nil {
		return nil, fmt.Errorf("get coin purse: %w", err)
	}
	defer rows.Close()

	var purse []models.CoinPurse
	for rows.Next() {
		var cp models.CoinPurse
		if err := rows.Scan(&cp.CoinTypeID, &cp.CoinName, &cp.Abbreviation, &cp.Emoji, &cp.Amount); err != nil {
			return nil, err
		}
		purse = append(purse, cp)
	}
	return purse, nil
}

func (r *CoinRepository) GetCoinBalance(ctx context.Context, characterID, coinTypeID uuid.UUID) (float64, error) {
	var amount float64
	err := r.db.QueryRow(ctx, `
		SELECT COALESCE(amount, 0) FROM coin_purse
		WHERE character_id = $1 AND coin_type_id = $2
	`, characterID, coinTypeID).Scan(&amount)
	if err != nil {
		// sem registro = saldo zero
		return 0, nil
	}
	return amount, nil
}

func (r *CoinRepository) SetCoinBalance(ctx context.Context, characterID, coinTypeID uuid.UUID, amount float64) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO coin_purse (character_id, coin_type_id, amount, updated_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (character_id, coin_type_id)
		DO UPDATE SET amount = $3, updated_at = NOW()
	`, characterID, coinTypeID, amount)
	if err != nil {
		return fmt.Errorf("set coin balance: %w", err)
	}
	return nil
}

func (r *CoinRepository) AddToBalance(ctx context.Context, characterID, coinTypeID uuid.UUID, delta float64) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO coin_purse (character_id, coin_type_id, amount, updated_at)
		VALUES ($1, $2, GREATEST(0, $3), NOW())
		ON CONFLICT (character_id, coin_type_id)
		DO UPDATE SET amount = GREATEST(0, coin_purse.amount + $3), updated_at = NOW()
	`, characterID, coinTypeID, delta)
	if err != nil {
		return fmt.Errorf("add to balance: %w", err)
	}
	return nil
}

func (r *CoinRepository) GetConversionRate(ctx context.Context, fromID, toID uuid.UUID) (float64, error) {
	var rate float64
	err := r.db.QueryRow(ctx, `
		SELECT rate FROM coin_conversions
		WHERE from_coin_id = $1 AND to_coin_id = $2
	`, fromID, toID).Scan(&rate)
	if err != nil {
		return 0, ErrNoConversion
	}
	return rate, nil
}

type ConversionEdge struct {
	FromID uuid.UUID
	ToID   uuid.UUID
	Rate   float64
}

// ListAllConversionsForCharacter retorna todas as conversões (ambas as direções) da campanha do personagem,
// usadas para encontrar caminhos transitivos via BFS.
func (r *CoinRepository) ListAllConversionsForCharacter(ctx context.Context, characterID uuid.UUID) ([]ConversionEdge, error) {
	rows, err := r.db.Query(ctx, `
		SELECT cc.from_coin_id, cc.to_coin_id, cc.rate
		FROM coin_conversions cc
		JOIN coin_types ct ON ct.id = cc.from_coin_id
		JOIN characters ch ON ch.campaign_id = ct.campaign_id
		WHERE ch.id = $1
	`, characterID)
	if err != nil {
		return nil, fmt.Errorf("list conversions for character: %w", err)
	}
	defer rows.Close()

	var edges []ConversionEdge
	for rows.Next() {
		var e ConversionEdge
		if err := rows.Scan(&e.FromID, &e.ToID, &e.Rate); err != nil {
			return nil, err
		}
		edges = append(edges, e)
	}
	return edges, nil
}
