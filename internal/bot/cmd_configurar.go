package bot

import (
	"context"
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
)

func (b *Bot) handleConfigurar(s *discordgo.Session, i *discordgo.InteractionCreate) {
	ctx := context.Background()

	user, err := b.userFromDiscord(ctx, discordUserID(i))
	if err != nil {
		ephemeral(s, i, "❌ Conta não vinculada. Use `/link` primeiro.")
		return
	}

	campaignIDStr := optionString(i, "campanha")
	campaignID, err := uuid.Parse(campaignIDStr)
	if err != nil {
		ephemeral(s, i, "❌ Campanha inválida.")
		return
	}

	role, err := b.memberRole(ctx, campaignID, user.ID)
	if err != nil || role != "gm" {
		ephemeral(s, i, "❌ Apenas o GM pode configurar o canal.")
		return
	}

	modo := optionString(i, "modo")

	// Busca nome da campanha para a mensagem de confirmação
	campaigns, _ := b.deps.CampaignRepo.ListCampaignsAsGM(ctx, user.ID)
	campaignName := campaignIDStr
	for _, c := range campaigns {
		if c.ID == campaignID {
			campaignName = c.Name
			break
		}
	}

	switch modo {
	case "canal":
		if err := b.deps.CampaignRepo.SetDiscordChannel(ctx, campaignID, i.ChannelID); err != nil {
			ephemeral(s, i, "❌ Erro ao salvar configuração.")
			return
		}
		respond(s, i, fmt.Sprintf("✅ Canal vinculado à campanha **%s**.", campaignName))
	case "servidor":
		if err := b.deps.CampaignRepo.SetDiscordGuild(ctx, campaignID, i.GuildID); err != nil {
			ephemeral(s, i, "❌ Erro ao salvar configuração.")
			return
		}
		respond(s, i, fmt.Sprintf("✅ Servidor vinculado à campanha **%s**.", campaignName))
	default:
		ephemeral(s, i, "❌ Modo inválido.")
	}
}

func (b *Bot) autocompleteConfigurar(s *discordgo.Session, i *discordgo.InteractionCreate) {
	ctx := context.Background()

	typed := ""
	for _, o := range i.ApplicationCommandData().Options {
		if o.Name == "campanha" && o.Focused {
			typed = o.StringValue()
		}
	}

	user, err := b.userFromDiscord(ctx, discordUserID(i))
	if err != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionApplicationCommandAutocompleteResult,
			Data: &discordgo.InteractionResponseData{Choices: []*discordgo.ApplicationCommandOptionChoice{}},
		})
		return
	}

	campaigns, _ := b.deps.CampaignRepo.ListCampaignsAsGM(ctx, user.ID)

	var choices []*discordgo.ApplicationCommandOptionChoice
	for _, c := range campaigns {
		if typed == "" || containsFold(c.Name, typed) {
			choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
				Name:  c.Name,
				Value: c.ID.String(),
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
