package inventory

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

type StorageRepository struct {
	db *pgxpool.Pool
}

func NewStorageRepository(db *pgxpool.Pool) *StorageRepository {
	return &StorageRepository{db: db}
}

// storageCols inclui item_id e current_weight_kg calculado via SUM.
// Requer GROUP BY s.id na query.
const storageCols = `
	s.id, s.character_id, s.item_id, s.name, COALESCE(s.description, ''),
	s.counts_toward_load, s.capacity_kg, s.is_default,
	COALESCE(SUM(itm.weight_kg * itm.quantity), 0) AS current_weight_kg,
	s.created_at
`

func scanStorage(row pgx.Row, ss *models.StorageSpace) error {
	return row.Scan(
		&ss.ID, &ss.CharacterID, &ss.ItemID, &ss.Name, &ss.Description,
		&ss.CountsTowardLoad, &ss.CapacityKg, &ss.IsDefault,
		&ss.CurrentWeightKg, &ss.CreatedAt,
	)
}

func (r *StorageRepository) CreateDefaultSpace(ctx context.Context, characterID uuid.UUID) (*models.StorageSpace, error) {
	var id uuid.UUID
	err := r.db.QueryRow(ctx, `
		INSERT INTO storage_spaces (character_id, name, counts_toward_load, is_default)
		VALUES ($1, 'Corpo', true, true)
		RETURNING id
	`, characterID).Scan(&id)
	if err != nil {
		return nil, fmt.Errorf("create default space: %w", err)
	}
	return r.GetStorageSpaceByID(ctx, id)
}

func (r *StorageRepository) CreateStorageSpace(ctx context.Context, characterID uuid.UUID, name, description string, countsTowardLoad bool, capacityKg *float64, itemID *uuid.UUID) (*models.StorageSpace, error) {
	var id uuid.UUID
	err := r.db.QueryRow(ctx, `
		INSERT INTO storage_spaces (character_id, name, description, counts_toward_load, capacity_kg, item_id)
		VALUES ($1, $2, NULLIF($3, ''), $4, $5, $6)
		RETURNING id
	`, characterID, name, description, countsTowardLoad, capacityKg, itemID).Scan(&id)
	if err != nil {
		if strings.Contains(err.Error(), "unique") {
			return nil, ErrDuplicateStorageName
		}
		return nil, fmt.Errorf("create storage space: %w", err)
	}
	return r.GetStorageSpaceByID(ctx, id)
}

func (r *StorageRepository) GetStorageSpaceByID(ctx context.Context, id uuid.UUID) (*models.StorageSpace, error) {
	var ss models.StorageSpace
	err := scanStorage(r.db.QueryRow(ctx, `
		SELECT `+storageCols+`
		FROM storage_spaces s
		LEFT JOIN items itm ON itm.storage_space_id = s.id
		WHERE s.id = $1
		GROUP BY s.id
	`, id), &ss)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrStorageNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get storage space: %w", err)
	}
	return &ss, nil
}

func (r *StorageRepository) GetStorageSpaceByItemID(ctx context.Context, itemID uuid.UUID) (*models.StorageSpace, error) {
	var ss models.StorageSpace
	err := scanStorage(r.db.QueryRow(ctx, `
		SELECT `+storageCols+`
		FROM storage_spaces s
		LEFT JOIN items itm ON itm.storage_space_id = s.id
		WHERE s.item_id = $1
		GROUP BY s.id
	`, itemID), &ss)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil // não é um container — ok
	}
	if err != nil {
		return nil, fmt.Errorf("get storage by item: %w", err)
	}
	return &ss, nil
}

func (r *StorageRepository) GetDefaultStorageSpace(ctx context.Context, characterID uuid.UUID) (*models.StorageSpace, error) {
	var ss models.StorageSpace
	err := scanStorage(r.db.QueryRow(ctx, `
		SELECT `+storageCols+`
		FROM storage_spaces s
		LEFT JOIN items itm ON itm.storage_space_id = s.id
		WHERE s.character_id = $1 AND s.is_default = TRUE
		GROUP BY s.id
	`, characterID), &ss)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrStorageNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get default storage: %w", err)
	}
	return &ss, nil
}

func (r *StorageRepository) ListStorageSpaces(ctx context.Context, characterID uuid.UUID) ([]models.StorageSpace, error) {
	rows, err := r.db.Query(ctx, `
		SELECT `+storageCols+`
		FROM storage_spaces s
		LEFT JOIN items itm ON itm.storage_space_id = s.id
		WHERE s.character_id = $1
		GROUP BY s.id
		ORDER BY s.is_default DESC, s.created_at ASC
	`, characterID)
	if err != nil {
		return nil, fmt.Errorf("list storage spaces: %w", err)
	}
	defer rows.Close()

	var spaces []models.StorageSpace
	for rows.Next() {
		var ss models.StorageSpace
		if err := rows.Scan(
			&ss.ID, &ss.CharacterID, &ss.ItemID, &ss.Name, &ss.Description,
			&ss.CountsTowardLoad, &ss.CapacityKg, &ss.IsDefault,
			&ss.CurrentWeightKg, &ss.CreatedAt,
		); err != nil {
			return nil, err
		}
		spaces = append(spaces, ss)
	}
	return spaces, nil
}

func (r *StorageRepository) UpdateStorageSpace(ctx context.Context, id uuid.UUID, name, description string, countsTowardLoad bool, capacityKg *float64) (*models.StorageSpace, error) {
	tag, err := r.db.Exec(ctx, `
		UPDATE storage_spaces
		SET name = $1, description = NULLIF($2, ''), counts_toward_load = $3, capacity_kg = $4
		WHERE id = $5
	`, name, description, countsTowardLoad, capacityKg, id)
	if err != nil {
		if strings.Contains(err.Error(), "unique") {
			return nil, ErrDuplicateStorageName
		}
		return nil, fmt.Errorf("update storage space: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrStorageNotFound
	}
	return r.GetStorageSpaceByID(ctx, id)
}

func (r *StorageRepository) ReassignItemsToDefault(ctx context.Context, fromSpaceID, toSpaceID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `
		UPDATE items SET storage_space_id = $1 WHERE storage_space_id = $2
	`, toSpaceID, fromSpaceID)
	return err
}

func (r *StorageRepository) DeleteStorageSpace(ctx context.Context, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM storage_spaces WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete storage space: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrStorageNotFound
	}
	return nil
}
