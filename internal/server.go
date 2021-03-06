package internal

import (
	"context"
	"errors"
	"log"
	"os"

	"github.com/bwmarrin/discordgo"
	"github.com/digitalungdom-se/dub/pkg"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type (
	Server struct {
		Bot              *discordgo.Member
		Channels         channels
		CommandHandler   CommandHandler
		Config           Config
		Controller       *pkg.Controller
		Database         Database
		Discord          *discordgo.Session
		Guild            *discordgo.Guild
		Mailer           Mailer
		ReactionListener pkg.ReactionListener
		Ready            bool
		Roles            roles
		Status           discordgo.UpdateStatusData
	}

	channels struct {
		General *discordgo.Channel
		Bot     *discordgo.Channel
		Music   *discordgo.Channel
		Regler  *discordgo.Channel
	}

	roles struct {
		Admin    *discordgo.Role
		Verified *discordgo.Role
	}
)

func NewServer(config Config, discord *discordgo.Session) *Server {
	server := new(Server)

	var mongoClient *mongo.Client
	mongoClient, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(os.Getenv("DB_URI")))
	if err != nil {
		log.Fatal(err)
	}

	err = mongoClient.Ping(context.TODO(), nil)
	if err != nil {
		log.Fatal(err)
	}

	mongoDatabase := mongoClient.Database(os.Getenv("DB_DATABASE"))
	server.Database = NewDatabase(mongoDatabase)

	reactionListener := pkg.NewReactionListener(discord)
	server.CommandHandler = NewCommandHandler()
	server.Config = config
	server.Discord = discord
	server.Discord.State.MaxMessageCount = 5
	server.Mailer = NewMailer(mongoDatabase.Collection("emails"))
	server.ReactionListener = reactionListener

	server.Ready = false

	return server
}

func (server *Server) Init() error {
	guild, err := server.Discord.Guild(server.Config.GuildID)
	if err != nil {
		return err
	}

	server.Discord.State.GuildAdd(guild)
	server.Guild, err = server.Discord.State.Guild(server.Config.GuildID)
	if err != nil {
		return err
	}

	discordStatus := discordgo.UpdateStatusData{}
	discordStatus.Game = new(discordgo.Game)

	server.Discord.UpdateStatusComplex(discordStatus)
	server.Status = discordStatus

	server.Channels = channels{}
	channels, err := server.Discord.GuildChannels(server.Config.GuildID)
	if err != nil {
		return err
	}

	for _, channel := range channels {
		switch channel.Name {
		case "music":
			server.Discord.State.ChannelAdd(channel)
			server.Channels.Music, err = server.Discord.State.Channel(channel.ID)
		case "bot":
			server.Discord.State.ChannelAdd(channel)
			server.Channels.Bot, err = server.Discord.State.Channel(channel.ID)
		case "general":
			server.Discord.State.ChannelAdd(channel)
			server.Channels.General, err = server.Discord.State.Channel(channel.ID)
		case "regler":
			server.Discord.State.ChannelAdd(channel)
			server.Channels.Regler, err = server.Discord.State.Channel(channel.ID)
		}

		if err != nil {
			return err
		}
	}

	for _, role := range server.Guild.Roles {
		switch role.Name {
		case "admin":
			server.Discord.State.RoleAdd(server.Guild.ID, role)
			server.Roles.Admin, err = server.Discord.State.Role(server.Guild.ID, role.ID)
		case "verified":
			server.Discord.State.RoleAdd(server.Guild.ID, role)
			server.Roles.Verified, err = server.Discord.State.Role(server.Guild.ID, role.ID)
		}

		if err != nil {
			return err
		}
	}

	if server.Channels.General == nil ||
		server.Channels.Music == nil ||
		server.Channels.Bot == nil ||
		server.Channels.Regler == nil ||
		server.Roles.Admin == nil ||
		server.Roles.Verified == nil {
		return errors.New("could not find channels or roles")
	}

	bot, err := server.Discord.User("@me")
	if err != nil {
		return err
	}
	for _, member := range server.Guild.Members {
		if member.User.ID == bot.ID {
			err = server.Discord.State.MemberAdd(member)
			if err != nil {
				return err
			}
		}
	}

	server.Bot, err = server.Discord.State.Member(server.Config.GuildID, bot.ID)
	if err != nil {
		return err
	}

	if server.Bot == nil {
		return errors.New("could not find bot")
	}

	server.Ready = true

	return err
}

func (server *Server) InitController() error {
	var err error

	server.Controller, err = pkg.NewController(server.Discord, server.Channels.Music, &server.Status, &server.ReactionListener)

	return err
}

func (server *Server) InitRules() error {
	var err error

	for {
		messages, err := server.Discord.ChannelMessages(server.Channels.Regler.ID, 100, "", "", "")
		if err != nil {
			return err
		}

		var messagesID []string

		for _, message := range messages {
			messagesID = append(messagesID, message.ID)
		}

		err = server.Discord.ChannelMessagesBulkDelete(server.Channels.Regler.ID, messagesID)
		if err != nil {
			return err
		}

		if len(messages) <= 100 {
			break
		}
	}

	rulesEmbed := pkg.NewEmbed()
	rulesEmbed.SetTitle("Regler för Discord kanalen.")
	rulesEmbed.SetColor(16711680)
	rulesEmbed.AddField("__**Du måste vara medlem i Digital Ungdom**__", "Bli medlem här: https://digitalungdom.se/bli-medlem.\n")
	rulesEmbed.AddField("__**Använd ditt förnamn**__", "Vi vill gärna skapa en personlig miljö. Använd `/nick`-kommandot. Du kan behålla ditt alias genom ex. `/nick Nikolaus (Jultomten)`.\n")
	rulesEmbed.AddField("__**Gyllene regeln**__", "Gör mot andra som du själv vill bli behandlad.\n")
	rulesEmbed.AddField("__**Inget stötande beteende.**__", "- Inga rasistiska, sexistiska, eller aggressiva kommentarer.\n- Inget spam.\n- Undvik gärna svordomar.")

	thinkAbout := pkg.NewEmbed()
	thinkAbout.SetTitle("Tänk på...")
	thinkAbout.SetColor(589568)
	thinkAbout.AddField("__**Undvik elitism**__", " Var upplyftande gentemot nybörjare! Alla vill lära sig mer och det blir roligare om vi bidrar till varandras utveckling och framsteg. Försök att inte överväldiga nybörjare med en onödigt teknisk jargong.\n")
	thinkAbout.AddField("__**Använd rätt kanal**__", " Vi har försökt strukturera kanalerna efter ämne, men om du inte känner att något passar kan du antingen skriva till oss eller skriva i #general.\n")
	thinkAbout.AddField("__**Var gärna tydlig med din nivå**__", "Som nybörjare kan en få mycket hjälp genom att förtydliga att en är ny till vissa områden. Som erfaren kan en känna sig osäker om ens korrespondent egentligen kan mycket mer än en tror!\n")

	reactionator := pkg.NewReactionator(server.Channels.Regler.ID, server.Discord, &server.ReactionListener, false, false, pkg.ReactionatorTypeController, nil)
	err = reactionator.AddDefaultPage("", "__**TRYCK PÅ DEN GRÖNA KNAPPEN UNDER MEDDELANDET FÖR ATT GODKÄNNA REGLERNA OCH DÄRMED KUNNA SKRIVA OCH DELTA I SAMTAL**__")
	if err != nil {
		return err
	}

	err = reactionator.Add("✅", func(message *discordgo.MessageReaction) {
		var medlemRoleID string

		for _, role := range server.Guild.Roles {
			if role.Name == "accepterat_reglerna" {
				medlemRoleID = role.ID
			}
		}

		server.Discord.GuildMemberRoleAdd(server.Guild.ID, message.UserID, medlemRoleID)
	})

	if err != nil {
		return err
	}

	server.Discord.ChannelMessageSendEmbed(server.Channels.Regler.ID, rulesEmbed.MessageEmbed)
	server.Discord.ChannelMessageSendEmbed(server.Channels.Regler.ID, thinkAbout.MessageEmbed)
	err = reactionator.Initiate()

	return err
}
