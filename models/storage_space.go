package models

import (
	"time"

	"github.com/google/uuid"
)

type StorageSpace struct {
	ID               uuid.UUID  `json:"id"`
	CharacterID      uuid.UUID  `json:"character_id"`
	ItemID           *uuid.UUID `json:"item_id,omitempty"` // item físico que representa este espaço (ex: mochila)
	Name             string     `json:"name"`
	Description      string     `json:"description"`
	CountsTowardLoad bool       `json:"counts_toward_load"`
	CapacityKg       *float64   `json:"capacity_kg"`
	IsDefault        bool       `json:"is_default"`
	CurrentWeightKg  float64    `json:"current_weight_kg"`
	CreatedAt        time.Time  `json:"created_at"`
}
