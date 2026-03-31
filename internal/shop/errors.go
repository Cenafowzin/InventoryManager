package shop

import "errors"

var (
	ErrShopItemNotFound = errors.New("shop item not found")
	ErrForbidden        = errors.New("forbidden: only GM can manage shop items")
	ErrCoinNotInCampaign = errors.New("coin does not belong to this campaign")
)
