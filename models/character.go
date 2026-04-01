package models

import (
	"time"

	"github.com/google/uuid"
)

type Character struct {
	ID                uuid.UUID  `json:"id"`
	CampaignID        uuid.UUID  `json:"campaign_id"`
	OwnerUserID       uuid.UUID  `json:"owner_user_id"`
	OwnerName         string     `json:"owner_name"`
	Name              string     `json:"name"`
	Description       string     `json:"description"`
	MaxCarryWeightKg  *float64   `json:"max_carry_weight_kg"`
	IsReserve         bool       `json:"is_reserve"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}
