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
		SELECT c.id, c.campaign_id,
		       COALESCE(c.owner_user_id, '00000000-0000-0000-0000-000000000000'::uuid),
		       COALESCE(u.username, ''),
		       c.name, COALESCE(c.description, ''), c.max_carry_weight_kg,
		       c.is_reserve, c.created_at, c.updated_at
		FROM characters c
		LEFT JOIN users u ON u.id = c.owner_user_id
		WHERE c.id = $1
	`, id).Scan(
		&ch.ID, &ch.CampaignID, &ch.OwnerUserID, &ch.OwnerName,
		&ch.Name, &ch.Description, &ch.MaxCarryWeightKg,
		&ch.IsReserve, &ch.CreatedAt, &ch.UpdatedAt,
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
		SELECT c.id, c.campaign_id,
		       COALESCE(c.owner_user_id, '00000000-0000-0000-0000-000000000000'::uuid),
		       COALESCE(u.username, ''),
		       c.name, COALESCE(c.description, ''), c.max_carry_weight_kg,
		       c.is_reserve, c.created_at, c.updated_at
		FROM characters c
		LEFT JOIN users u ON u.id = c.owner_user_id
		WHERE c.campaign_id = $1 AND c.is_reserve = FALSE
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
			&ch.Name, &ch.Description, &ch.MaxCarryWeightKg,
			&ch.IsReserve, &ch.CreatedAt, &ch.UpdatedAt,
		); err != nil {
			return nil, err
		}
		chars = append(chars, ch)
	}
	return chars, nil
}

func (r *Repository) ListCharactersByOwner(ctx context.Context, campaignID, ownerUserID uuid.UUID) ([]models.Character, error) {
	rows, err := r.db.Query(ctx, `
		SELECT c.id, c.campaign_id,
		       COALESCE(c.owner_user_id, '00000000-0000-0000-0000-000000000000'::uuid),
		       COALESCE(u.username, ''),
		       c.name, COALESCE(c.description, ''), c.max_carry_weight_kg,
		       c.is_reserve, c.created_at, c.updated_at
		FROM characters c
		LEFT JOIN users u ON u.id = c.owner_user_id
		WHERE c.campaign_id = $1 AND c.owner_user_id = $2 AND c.is_reserve = FALSE
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
			&ch.Name, &ch.Description, &ch.MaxCarryWeightKg,
			&ch.IsReserve, &ch.CreatedAt, &ch.UpdatedAt,
		); err != nil {
			return nil, err
		}
		chars = append(chars, ch)
	}
	return chars, nil
}

// EnsureCampaignReserve returns the campaign's reserve character, creating it if it doesn't exist.
// Returns (character, isNew, error). Callers should create a default storage space when isNew is true.
func (r *Repository) EnsureCampaignReserve(ctx context.Context, campaignID uuid.UUID) (*models.Character, bool, error) {
	var id uuid.UUID
	err := r.db.QueryRow(ctx, `SELECT id FROM characters WHERE campaign_id = $1 AND is_reserve = TRUE LIMIT 1`, campaignID).Scan(&id)
	if err == nil {
		ch, err := r.GetCharacterByID(ctx, id)
		return ch, false, err
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, false, fmt.Errorf("get reserve: %w", err)
	}

	// Create reserve character (no owner_user_id)
	err = r.db.QueryRow(ctx, `
		INSERT INTO characters (campaign_id, name, description, is_reserve)
		VALUES ($1, 'Inventário do GM', '', TRUE)
		ON CONFLICT DO NOTHING
		RETURNING id
	`, campaignID).Scan(&id)
	if err != nil {
		// Race: another request created it
		err2 := r.db.QueryRow(ctx, `SELECT id FROM characters WHERE campaign_id = $1 AND is_reserve = TRUE LIMIT 1`, campaignID).Scan(&id)
		if err2 != nil {
			return nil, false, fmt.Errorf("ensure reserve: %w", err)
		}
		ch, err := r.GetCharacterByID(ctx, id)
		return ch, false, err
	}
	ch, err := r.GetCharacterByID(ctx, id)
	return ch, true, err
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
