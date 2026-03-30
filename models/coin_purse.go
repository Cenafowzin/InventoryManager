package models

import "github.com/google/uuid"

type CoinPurse struct {
	CoinTypeID   uuid.UUID `json:"coin_type_id"`
	CoinName     string    `json:"coin_name"`
	Abbreviation string    `json:"abbreviation"`
	Emoji        string    `json:"emoji,omitempty"`
	Amount       float64   `json:"amount"`
}
