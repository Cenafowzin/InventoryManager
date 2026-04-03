package discord

import "errors"

var (
	ErrInvalidCode   = errors.New("código inválido")
	ErrCodeExpired   = errors.New("código expirado")
	ErrNotLinked     = errors.New("conta não vinculada ao Discord")
	ErrAlreadyLinked = errors.New("conta Discord já vinculada a outro usuário")
)
