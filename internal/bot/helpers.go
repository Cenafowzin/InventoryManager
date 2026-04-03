package bot

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"

	"github.com/rubendubeux/inventory-manager/models"
)

// resolveCharacter retorna o personagem a usar para o comando.
// Se o parâmetro "personagem" foi passado, busca por nome.
// Se não foi passado e o usuário tem só um, usa ele.
// Caso contrário, pede para especificar.
func (b *Bot) resolveCharacter(ctx context.Context, i *discordgo.InteractionCreate, campaignID, userID uuid.UUID) (*models.Character, error) {
	chars, err := b.deps.CharacterRepo.ListCharactersByOwner(ctx, campaignID, userID)
	if err != nil {
		return nil, errors.New("erro ao buscar personagens")
	}
	if len(chars) == 0 {
		return nil, errors.New("você não tem personagens nesta campanha")
	}

	name := optionString(i, "personagem")
	if name == "" {
		if len(chars) == 1 {
			return &chars[0], nil
		}
		names := make([]string, len(chars))
		for i, c := range chars {
			names[i] = c.Name
		}
		return nil, fmt.Errorf("você tem vários personagens. Especifique: %s", joinNames(names))
	}

	for _, c := range chars {
		if c.Name == name {
			return &c, nil
		}
	}
	return nil, fmt.Errorf("personagem \"%s\" não encontrado", name)
}

func joinNames(names []string) string {
	return strings.Join(names, ", ")
}

func respondEmptyAutocomplete(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionApplicationCommandAutocompleteResult,
		Data: &discordgo.InteractionResponseData{Choices: []*discordgo.ApplicationCommandOptionChoice{}},
	})
}
