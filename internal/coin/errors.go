package coin

import "errors"

var (
	ErrCoinNotFound        = errors.New("coin type not found")
	ErrNoDefaultCoin       = errors.New("no default coin set for this campaign")
	ErrConversionNotFound  = errors.New("conversion not found")
	ErrConversionExists    = errors.New("a conversion between these coins already exists")
	ErrCoinInUse           = errors.New("coin is in use and cannot be deleted")
	ErrSameCoin            = errors.New("from_coin_id and to_coin_id must be different")
)
