package bot

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"

	"github.com/rubendubeux/inventory-manager/internal/bot/format"
)

func (b *Bot) handlePersonagens(s *discordgo.Session, i *discordgo.InteractionCreate) {
	ctx := context.Background()

	campaign, err := b.campaignFromContext(ctx, i)
	if err != nil {
		ephemeral(s, i, "❌ "+err.Error())
		return
	}

	user, err := b.userFromDiscord(ctx, discordUserID(i))
	if errors.Is(err, ErrUserNotLinked) {
		ephemeral(s, i, "❌ Conta não vinculada. Use `/link <codigo>` com o código gerado no site.")
		return
	}
	if err != nil {
		ephemeral(s, i, "❌ Erro ao buscar usuário.")
		return
	}

	characters, err := b.deps.CharacterRepo.ListCharactersByOwner(ctx, campaign.ID, user.ID)
	if err != nil {
		ephemeral(s, i, "❌ Erro ao buscar personagens.")
		return
	}
	if len(characters) == 0 {
		ephemeral(s, i, fmt.Sprintf("Você não tem personagens na campanha **%s**.", campaign.Name))
		return
	}

	var lines []string
	for _, c := range characters {
		line := fmt.Sprintf("• **%s**", c.Name)
		if c.MaxCarryWeightKg != nil {
			line += fmt.Sprintf(" (carga máx: %s kg)", format.Num(*c.MaxCarryWeightKg))
		}
		lines = append(lines, line)
	}

	msg := fmt.Sprintf("**Personagens de %s em %s**\n%s", discordUsername(i), campaign.Name, strings.Join(lines, "\n"))
	respond(s, i, msg)
}
