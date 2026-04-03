package bot

import (
	"context"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"

	"github.com/rubendubeux/inventory-manager/internal/bot/format"
	shopkg "github.com/rubendubeux/inventory-manager/internal/shop"
)

func (b *Bot) handleLoja(s *discordgo.Session, i *discordgo.InteractionCreate) {
	ctx := context.Background()

	campaign, err := b.campaignFromContext(ctx, i)
	if err != nil {
		ephemeral(s, i, "❌ "+err.Error())
		return
	}

	items, err := b.deps.ShopRepo.ListShopItems(ctx, campaign.ID, shopkg.ShopFilters{})
	if err != nil {
		ephemeral(s, i, "❌ Erro ao buscar loja.")
		return
	}
	if len(items) == 0 {
		ephemeral(s, i, "A loja está vazia no momento.")
		return
	}

	// Agrupa por loja
	grouped := map[string][]string{}
	order := []string{}
	seen := map[string]bool{}

	for _, item := range items {
		shopName := item.ShopName
		if shopName == "" {
			shopName = "Geral"
		}
		if !seen[shopName] {
			order = append(order, shopName)
			seen[shopName] = true
		}
		emoji := item.Emoji
		if emoji == "" {
			emoji = "•"
		}
		coin := item.ValueCoin
		line := fmt.Sprintf("%s **%s** — %s %s", emoji, item.Name, format.Num(item.BaseValue), coin)
		if item.StockQuantity != nil {
			line += fmt.Sprintf(" *(estoque: %d)*", *item.StockQuantity)
		}
		grouped[shopName] = append(grouped[shopName], line)
	}

	var sections []string
	for _, shop := range order {
		sections = append(sections, format.Block("🏪 "+shop, grouped[shop]))
	}

	msg := fmt.Sprintf("**Loja — %s**\n\n%s", campaign.Name, strings.Join(sections, "\n\n"))

	if len(msg) > 1900 {
		msg = msg[:1900] + "\n*(lista truncada)*"
	}

	respond(s, i, msg)
}
