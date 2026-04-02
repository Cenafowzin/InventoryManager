package models

import (
	"time"

	"github.com/google/uuid"
)

type ShopItem struct {
	ID            uuid.UUID  `json:"id"`
	CampaignID    uuid.UUID  `json:"campaign_id"`
	Name          string     `json:"name"`
	Description   string     `json:"description"`
	Emoji         string     `json:"emoji,omitempty"`
	WeightKg      float64    `json:"weight_kg"`
	BaseValue     float64    `json:"base_value"`
	ValueCoinID   *uuid.UUID `json:"value_coin_id"`
	ValueCoin     string     `json:"value_coin,omitempty"`
	IsAvailable   bool       `json:"is_available"`
	ShopID        *uuid.UUID `json:"shop_id"`
	ShopName      string     `json:"shop_name,omitempty"`
	ShopColor     string     `json:"shop_color,omitempty"`
	StockQuantity *int       `json:"stock_quantity"`
	Categories    []Category `json:"categories"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}