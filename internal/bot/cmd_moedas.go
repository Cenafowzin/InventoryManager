package bot

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"

	"github.com/rubendubeux/inventory-manager/internal/bot/format"
)

func (b *Bot) handleMoedas(s *discordgo.Session, i *discordgo.InteractionCreate) {
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

	purse, err := b.deps.CoinRepo.GetCoinPurse(ctx, char.ID)
	if err != nil {
		ephemeral(s, i, "❌ Erro ao buscar moedas.")
		return
	}

	var lines []string
	for _, p := range purse {
		if p.Amount > 0 {
			lines = append(lines, fmt.Sprintf("%s **%s** %s", p.Emoji, format.Num(p.Amount), p.Abbreviation))
		}
	}
	if len(lines) == 0 {
		ephemeral(s, i, fmt.Sprintf("**%s** não tem moedas.", char.Name))
		return
	}

	respond(s, i, fmt.Sprintf("💰 **Carteira de %s**\n%s", char.Name, strings.Join(lines, "\n")))
}

func (b *Bot) autocompletePersonagem(s *discordgo.Session, i *discordgo.InteractionCreate) {
	ctx := context.Background()

	campaign, err := b.campaignFromContext(ctx, i)
	if err != nil {
		respondEmptyAutocomplete(s, i)
		return
	}

	user, err := b.userFromDiscord(ctx, discordUserID(i))
	if err != nil {
		respondEmptyAutocomplete(s, i)
		return
	}

	typed := ""
	for _, o := range i.ApplicationCommandData().Options {
		if o.Name == "personagem" && o.Focused {
			typed = o.StringValue()
		}
	}

	chars, err := b.deps.CharacterRepo.ListCharactersByOwner(ctx, campaign.ID, user.ID)
	if err != nil {
		respondEmptyAutocomplete(s, i)
		return
	}

	var choices []*discordgo.ApplicationCommandOptionChoice
	for _, c := range chars {
		if typed == "" || containsFold(c.Name, typed) {
			choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
				Name:  c.Name,
				Value: c.Name,
			})
		}
		if len(choices) >= 25 {
			break
		}
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionApplicationCommandAutocompleteResult,
		Data: &discordgo.InteractionResponseData{Choices: choices},
	})
}
