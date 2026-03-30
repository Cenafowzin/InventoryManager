package models

import (
	"time"

	"github.com/google/uuid"
)

type Category struct {
	ID         uuid.UUID `json:"id"`
	CampaignID uuid.UUID `json:"campaign_id"`
	Name       string    `json:"name"`
	Color      string    `json:"color,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}
