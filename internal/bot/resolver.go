package bot

import (
	"context"
	"errors"

	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"

	discordpkg "github.com/rubendubeux/inventory-manager/internal/discord"
	"github.com/rubendubeux/inventory-manager/models"
)

var (
	ErrChannelNotConfigured = errors.New("canal não vinculado a nenhuma campanha — GM use /configurar")
	ErrUserNotLinked        = discordpkg.ErrNotLinked
	ErrNotMember            = errors.New("você não é membro desta campanha")
)

// campaignFromContext resolve a campanha pelo canal (prioridade) ou servidor.
func (b *Bot) campaignFromContext(ctx context.Context, i *discordgo.InteractionCreate) (*models.Campaign, error) {
	c, err := b.deps.CampaignRepo.GetByDiscordChannel(ctx, i.ChannelID)
	if err != nil {
		return nil, err
	}
	if c != nil {
		return c, nil
	}

	if i.GuildID != "" {
		c, err = b.deps.CampaignRepo.GetByDiscordGuild(ctx, i.GuildID)
		if err != nil {
			return nil, err
		}
		if c != nil {
			return c, nil
		}
	}

	return nil, ErrChannelNotConfigured
}

// userFromDiscord resolve o usuário do site pelo discord_id.
func (b *Bot) userFromDiscord(ctx context.Context, discordID string) (*models.User, error) {
	return b.deps.DiscordSvc.GetUserByDiscordID(ctx, discordID)
}

// memberRole retorna o role do usuário na campanha ou ErrNotMember.
func (b *Bot) memberRole(ctx context.Context, campaignID, userID uuid.UUID) (string, error) {
	member, err := b.deps.CampaignRepo.GetMember(ctx, campaignID, userID)
	if err != nil {
		return "", ErrNotMember
	}
	return member.Role, nil
}

// discordUserID retorna o ID do usuário Discord que disparou a interação.
func discordUserID(i *discordgo.InteractionCreate) string {
	if i.Member != nil {
		return i.Member.User.ID
	}
	return i.User.ID
}

func discordUsername(i *discordgo.InteractionCreate) string {
	if i.Member != nil {
		return i.Member.User.Username
	}
	return i.User.Username
}
