package bot

import (
	"github.com/bwmarrin/discordgo"
)

// /site — manda o link do site publicamente (não ephemeral, para todo mundo ver)
func (b *Bot) handleSite(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "🌐 **Acesse o sistema aqui:**\n" + b.deps.SiteURL,
		},
	})
}

// /help — envia a lista de comandos no privado do usuário
func (b *Bot) handleHelp(s *discordgo.Session, i *discordgo.InteractionCreate) {
	msg := `📖 **Comandos disponíveis**

**Configuração**
` + "`/configurar`" + ` — Vincula este canal ou servidor a uma campanha *(apenas GM)*
` + "`/site`" + ` — Mostra o link do sistema

**Conta**
` + "`/comecar`" + ` — Passo a passo para criar conta e vincular ao Discord
` + "`/link <codigo>`" + ` — Vincula sua conta do site ao Discord

**Personagens**
` + "`/personagens`" + ` — Lista seus personagens na campanha
` + "`/inventario [personagem]`" + ` — Mostra o inventário de um personagem
` + "`/moedas [personagem]`" + ` — Mostra a carteira de moedas

**Loja**
` + "`/loja`" + ` — Lista os itens disponíveis na loja

` + "`/help`" + ` — Mostra esta mensagem`

	// Tenta enviar no privado
	ch, err := s.UserChannelCreate(discordUserID(i))
	if err == nil {
		s.ChannelMessageSend(ch.ID, msg)
		ephemeral(s, i, "📬 Enviei a lista de comandos no seu privado!")
		return
	}

	// Se não conseguir abrir DM, responde ephemeral no canal
	ephemeral(s, i, msg)
}

// /comecar — passo a passo de como criar conta, vincular e começar a usar
func (b *Bot) handleComecar(s *discordgo.Session, i *discordgo.InteractionCreate) {
	msg := "👋 **Bem-vindo ao sistema de inventário!**\n\n" +
		"Siga os passos abaixo para começar:\n\n" +
		"**1️⃣ Criar sua conta**\n" +
		"Acesse o site e clique em **Criar conta**:\n" +
		b.deps.SiteURL + "/register\n\n" +
		"**2️⃣ Entrar no site**\n" +
		"Faça login com seu e-mail e senha:\n" +
		b.deps.SiteURL + "/login\n\n" +
		"**3️⃣ Vincular sua conta ao Discord**\n" +
		"No site, vá em **👤 Perfil → Gerar código**.\n" +
		"Um código aparecerá — copie o comando gerado.\n\n" +
		"**4️⃣ Usar o código aqui**\n" +
		"Cole o comando gerado neste canal:\n" +
		"> `/link <codigo>`\n\n" +
		"**5️⃣ Pronto!** 🎉\n" +
		"Use `/personagens`, `/inventario`, `/moedas` e `/loja` para jogar.\n\n" +
		"💡 Dica: use `/help` para ver todos os comandos."

	// Tenta enviar no privado
	ch, err := s.UserChannelCreate(discordUserID(i))
	if err == nil {
		s.ChannelMessageSend(ch.ID, msg)
		ephemeral(s, i, "📬 Enviei o guia de início no seu privado!")
		return
	}

	ephemeral(s, i, msg)
}
