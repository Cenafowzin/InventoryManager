package models

import (
	"time"

	"github.com/google/uuid"
)

type CoinType struct {
	ID           uuid.UUID `json:"id"`
	CampaignID   uuid.UUID `json:"campaign_id"`
	Name         string    `json:"name"`
	Abbreviation string    `json:"abbreviation"`
	Emoji        string    `json:"emoji"`
	IsDefault    bool      `json:"is_default"`
	CreatedAt    time.Time `json:"created_at"`
}

type CoinConversion struct {
	ID         uuid.UUID `json:"id"`
	CampaignID uuid.UUID `json:"campaign_id"`
	FromCoinID uuid.UUID `json:"from_coin_id"`
	FromCoin   string    `json:"from_coin"`
	ToCoinID   uuid.UUID `json:"to_coin_id"`
	ToCoin     string    `json:"to_coin"`
	Rate       float64   `json:"rate"`
}
