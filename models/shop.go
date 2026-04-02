package models

import (
	"time"

	"github.com/google/uuid"
)

type Shop struct {
	ID         uuid.UUID `json:"id"`
	CampaignID uuid.UUID `json:"campaign_id"`
	Name       string    `json:"name"`
	Color      string    `json:"color"`
	IsActive   bool      `json:"is_active"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}
