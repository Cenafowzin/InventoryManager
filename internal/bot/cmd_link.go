package bot

import (
	"context"
	"errors"
	"fmt"

	"github.com/bwmarrin/discordgo"

	discordpkg "github.com/rubendubeux/inventory-manager/internal/discord"
)

func (b *Bot) handleLink(s *discordgo.Session, i *discordgo.InteractionCreate) {
	ctx := context.Background()
	code := optionString(i, "codigo")
	dID := discordUserID(i)
	tag := discordUsername(i)

	if err := b.deps.DiscordSvc.ConfirmLink(ctx, code, dID, tag); err != nil {
		switch {
		case errors.Is(err, discordpkg.ErrInvalidCode):
			ephemeral(s, i, "❌ Código inválido. Gere um novo no site em **Perfil → Vincular Discord**.")
		case errors.Is(err, discordpkg.ErrCodeExpired):
			ephemeral(s, i, "❌ Código expirado (validade: 10 min). Gere um novo no site.")
		default:
			ephemeral(s, i, fmt.Sprintf("❌ Erro ao vincular: %v", err))
		}
		return
	}

	respond(s, i, fmt.Sprintf("✅ **%s** vinculou a conta ao sistema!", discordUsername(i)))
}
