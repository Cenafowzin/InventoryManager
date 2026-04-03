package bot

import (
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"

	campaignpkg "github.com/rubendubeux/inventory-manager/internal/campaign"
	characterpkg "github.com/rubendubeux/inventory-manager/internal/character"
	discordpkg "github.com/rubendubeux/inventory-manager/internal/discord"
	inventorypkg "github.com/rubendubeux/inventory-manager/internal/inventory"
	shopkg "github.com/rubendubeux/inventory-manager/internal/shop"
	coinpkg "github.com/rubendubeux/inventory-manager/internal/coin"
)

// Deps agrupa todas as dependências que o bot precisa.
type Deps struct {
	DiscordSvc    *discordpkg.Service
	CampaignRepo  *campaignpkg.Repository
	CharacterRepo *characterpkg.Repository
	ItemRepo      *inventorypkg.ItemRepository
	StorageRepo   *inventorypkg.StorageRepository
	CoinRepo      *inventorypkg.CoinRepository
	ShopRepo      *shopkg.Repository
	CoinTypeRepo  *coinpkg.Repository
}

// Bot é o cliente Discord do sistema.
type Bot struct {
	session *discordgo.Session
	deps    Deps
	guildID string // vazio em produção; preenchido em dev para registro instantâneo
}

// New cria e inicia o bot. Retorna erro se o token for inválido.
func New(token, guildID string, deps Deps) (*Bot, error) {
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, err
	}
	b := &Bot{session: dg, deps: deps, guildID: guildID}
	dg.AddHandler(b.onInteraction)
	return b, nil
}

// Start conecta ao gateway e registra os slash commands.
func (b *Bot) Start() error {
	if err := b.session.Open(); err != nil {
		return err
	}
	return b.registerCommands()
}

// Stop fecha a conexão com o gateway.
func (b *Bot) Stop() {
	b.session.Close()
}

// ── Command definitions ───────────────────────────────────────────────────────

var commandDefs = []*discordgo.ApplicationCommand{
	{
		Name:        "configurar",
		Description: "Vincula este canal ou servidor a uma campanha (apenas GM)",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:         "campanha",
				Description:  "Nome da campanha",
				Type:         discordgo.ApplicationCommandOptionString,
				Required:     true,
				Autocomplete: true,
			},
			{
				Name:        "modo",
				Description: "Vincular só este canal ou o servidor inteiro",
				Type:        discordgo.ApplicationCommandOptionString,
				Required:    true,
				Choices: []*discordgo.ApplicationCommandOptionChoice{
					{Name: "Canal (só este chat)", Value: "canal"},
					{Name: "Servidor (todos os canais)", Value: "servidor"},
				},
			},
		},
	},
	{
		Name:        "link",
		Description: "Vincula sua conta Discord à conta do site",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:        "codigo",
				Description: "Código gerado no site (Perfil → Vincular Discord)",
				Type:        discordgo.ApplicationCommandOptionString,
				Required:    true,
			},
		},
	},
	{
		Name:        "personagens",
		Description: "Lista seus personagens na campanha deste canal",
	},
	{
		Name:        "moedas",
		Description: "Mostra a carteira de moedas de um personagem",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:         "personagem",
				Description:  "Nome do personagem (opcional se tiver só um)",
				Type:         discordgo.ApplicationCommandOptionString,
				Required:     false,
				Autocomplete: true,
			},
		},
	},
	{
		Name:        "inventario",
		Description: "Mostra o inventário de um personagem",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:         "personagem",
				Description:  "Nome do personagem (opcional se tiver só um)",
				Type:         discordgo.ApplicationCommandOptionString,
				Required:     false,
				Autocomplete: true,
			},
		},
	},
	{
		Name:        "loja",
		Description: "Lista os itens disponíveis na loja da campanha",
	},
}

func (b *Bot) registerCommands() error {
	appID := b.session.State.User.ID
	for _, cmd := range commandDefs {
		if _, err := b.session.ApplicationCommandCreate(appID, b.guildID, cmd); err != nil {
			log.Printf("bot: erro ao registrar comando %s: %v", cmd.Name, err)
		}
	}
	log.Println("bot: slash commands registrados")
	return nil
}

// ── Interaction router ────────────────────────────────────────────────────────

func (b *Bot) onInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	switch i.Type {
	case discordgo.InteractionApplicationCommand:
		switch i.ApplicationCommandData().Name {
		case "configurar":
			b.handleConfigurar(s, i)
		case "link":
			b.handleLink(s, i)
		case "personagens":
			b.handlePersonagens(s, i)
		case "moedas":
			b.handleMoedas(s, i)
		case "inventario":
			b.handleInventario(s, i)
		case "loja":
			b.handleLoja(s, i)
		}

	case discordgo.InteractionApplicationCommandAutocomplete:
		switch i.ApplicationCommandData().Name {
		case "configurar":
			b.autocompleteConfigurar(s, i)
		case "moedas", "inventario":
			b.autocompletePersonagem(s, i)
		}
	}
}

// ── Helpers ───────────────────────────────────────────────────────────────────

// ephemeral envia uma resposta visível só para quem chamou o comando.
func ephemeral(s *discordgo.Session, i *discordgo.InteractionCreate, msg string) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: msg,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

// optionString retorna o valor de uma opção string por nome.
func optionString(i *discordgo.InteractionCreate, name string) string {
	for _, o := range i.ApplicationCommandData().Options {
		if o.Name == name {
			return o.StringValue()
		}
	}
	return ""
}

// containsFold verifica se s contém sub (case-insensitive).
func containsFold(s, sub string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(sub))
}
