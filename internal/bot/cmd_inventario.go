package bot

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"

	"github.com/rubendubeux/inventory-manager/internal/bot/format"
	"github.com/rubendubeux/inventory-manager/internal/inventory"
)

func (b *Bot) handleInventario(s *discordgo.Session, i *discordgo.InteractionCreate) {
	ctx := context.Background()

	campaign, err := b.campaignFromContext(ctx, i)
	if err != nil {
		ephemeral(s, i, "❌ "+err.Error())
		return
	}

	user, err := b.userFromDiscord(ctx, discordUserID(i))
	if errors.Is(err, ErrUserNotLinked) {
		ephemeral(s, i, "❌ Conta não vinculada. Use `/link <codigo>`.")
		return
	}
	if err != nil {
		ephemeral(s, i, "❌ Erro ao buscar usuário.")
		return
	}

	char, err := b.resolveCharacter(ctx, i, campaign.ID, user.ID)
	if err != nil {
		ephemeral(s, i, "❌ "+err.Error())
		return
	}

	items, err := b.deps.ItemRepo.ListItemsByCharacter(ctx, char.ID, inventory.ItemFilters{})
	if err != nil {
		ephemeral(s, i, "❌ Erro ao buscar inventário.")
		return
	}

	if len(items) == 0 {
		ephemeral(s, i, fmt.Sprintf("**%s** não tem itens no inventário.", char.Name))
		return
	}

	// Agrupa por espaço de armazenamento
	grouped := map[string][]string{}
	order := []string{}
	seen := map[string]bool{}

	for _, item := range items {
		space := item.StorageSpace
		if space == "" {
			space = "Sem local"
		}
		if !seen[space] {
			order = append(order, space)
			seen[space] = true
		}
		emoji := item.Emoji
		if emoji == "" {
			emoji = "•"
		}
		line := fmt.Sprintf("%s **%s** ×%d", emoji, item.Name, item.Quantity)
		if item.WeightKg > 0 {
			line += fmt.Sprintf(" (%s kg)", format.Num(item.WeightKg*float64(item.Quantity)))
		}
		grouped[space] = append(grouped[space], line)
	}

	var sections []string
	for _, space := range order {
		lines := grouped[space]
		sections = append(sections, format.Block(space, lines))
	}

	// Peso total
	var totalWeight float64
	for _, item := range items {
		totalWeight += item.WeightKg * float64(item.Quantity)
	}
	footer := fmt.Sprintf("\n%s\n⚖️ Peso total: **%s kg**", format.Divider(), format.Num(totalWeight))
	if char.MaxCarryWeightKg != nil {
		footer += fmt.Sprintf(" / %s kg", format.Num(*char.MaxCarryWeightKg))
	}

	msg := fmt.Sprintf("🎒 **Inventário de %s**\n\n%s%s", char.Name, strings.Join(sections, "\n\n"), footer)

	// Discord limita mensagens a 2000 chars
	if len(msg) > 1900 {
		msg = msg[:1900] + "\n*(lista truncada)*"
	}

	ephemeral(s, i, msg)
}
