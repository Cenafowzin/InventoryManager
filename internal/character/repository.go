package character

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

func (r *Repository) IsMember(ctx context.Context, campaignID, userID uuid.UUID) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM campaign_members
			WHERE campaign_id = $1 AND user_id = $2
		)
	`, campaignID, userID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check membership: %w", err)
	}
	return exists, nil
}

func (r *Repository) CreateCharacter(ctx context.Context, campaignID, ownerUserID uuid.UUID, name, description string, maxCarryWeightKg *float64) (*models.Character, error) {
	var id uuid.UUID
	err := r.db.QueryRow(ctx, `
		INSERT INTO characters (campaign_id, owner_user_id, name, description, max_carry_weight_kg)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`, campaignID, ownerUserID, name, description, maxCarryWeightKg).Scan(&id)
	if err != nil {
		return nil, fmt.Errorf("create character: %w", err)
	}
	return r.GetCharacterByID(ctx, id)
}

func (r *Repository) GetCharacterByID(ctx context.Context, id uuid.UUID) (*models.Character, error) {
	var ch models.Character
	err := r.db.QueryRow(ctx, `
		SELECT c.id, c.campaign_id, c.owner_user_id, u.username,
		       c.name, c.description, c.max_carry_weight_kg, c.created_at, c.updated_at
		FROM characters c
		JOIN users u ON u.id = c.owner_user_id
		WHERE c.id = $1
	`, id).Scan(
		&ch.ID, &ch.CampaignID, &ch.OwnerUserID, &ch.OwnerName,
		&ch.Name, &ch.Description, &ch.MaxCarryWeightKg, &ch.CreatedAt, &ch.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrCharacterNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get character: %w", err)
	}
	return &ch, nil
}

func (r *Repository) ListCharactersByCampaign(ctx context.Context, campaignID uuid.UUID) ([]models.Character, error) {
	rows, err := r.db.Query(ctx, `
		SELECT c.id, c.campaign_id, c.owner_user_id, u.username,
		       c.name, c.description, c.max_carry_weight_kg, c.created_at, c.updated_at
		FROM characters c
		JOIN users u ON u.id = c.owner_user_id
		WHERE c.campaign_id = $1
		ORDER BY c.created_at ASC
	`, campaignID)
	if err != nil {
		return nil, fmt.Errorf("list characters: %w", err)
	}
	defer rows.Close()

	var chars []models.Character
	for rows.Next() {
		var ch models.Character
		if err := rows.Scan(
			&ch.ID, &ch.CampaignID, &ch.OwnerUserID, &ch.OwnerName,
			&ch.Name, &ch.Description, &ch.MaxCarryWeightKg, &ch.CreatedAt, &ch.UpdatedAt,
		); err != nil {
			return nil, err
		}
		chars = append(chars, ch)
	}
	return chars, nil
}

func (r *Repository) ListCharactersByOwner(ctx context.Context, campaignID, ownerUserID uuid.UUID) ([]models.Character, error) {
	rows, err := r.db.Query(ctx, `
		SELECT c.id, c.campaign_id, c.owner_user_id, u.username,
		       c.name, c.description, c.max_carry_weight_kg, c.created_at, c.updated_at
		FROM characters c
		JOIN users u ON u.id = c.owner_user_id
		WHERE c.campaign_id = $1 AND c.owner_user_id = $2
		ORDER BY c.created_at ASC
	`, campaignID, ownerUserID)
	if err != nil {
		return nil, fmt.Errorf("list characters by owner: %w", err)
	}
	defer rows.Close()

	var chars []models.Character
	for rows.Next() {
		var ch models.Character
		if err := rows.Scan(
			&ch.ID, &ch.CampaignID, &ch.OwnerUserID, &ch.OwnerName,
			&ch.Name, &ch.Description, &ch.MaxCarryWeightKg, &ch.CreatedAt, &ch.UpdatedAt,
		); err != nil {
			return nil, err
		}
		chars = append(chars, ch)
	}
	return chars, nil
}

func (r *Repository) UpdateCharacter(ctx context.Context, id uuid.UUID, name, description string, maxCarryWeightKg *float64) (*models.Character, error) {
	tag, err := r.db.Exec(ctx, `
		UPDATE characters
		SET name = $1, description = $2, max_carry_weight_kg = $3, updated_at = NOW()
		WHERE id = $4
	`, name, description, maxCarryWeightKg, id)
	if err != nil {
		return nil, fmt.Errorf("update character: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrCharacterNotFound
	}
	return r.GetCharacterByID(ctx, id)
}

func (r *Repository) DeleteCharacter(ctx context.Context, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM characters WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete character: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrCharacterNotFound
	}
	return nil
}
