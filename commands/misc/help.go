package misc

import (
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/digitalungdom-se/dub/pkg"
)

var groupReactions = map[string]string{
	"info":          "ℹ",
	"digitalungdom": "🖥",
	"music":         "🎵",
	"misc":          "🛠",
	"admin":         "🚨",
	"close":         "🔥",
}

var groupReactionOrder = []string{"digitalungdom", "music", "misc"}

var Help = pkg.Command{
	Name:        "help",
	Description: "Listar alla tillgängliga kommandon",
	Aliases:     []string{"commands", "command", "hjälp", "kommando", "kommandon"},
	Group:       "misc",
	Usage:       "help <command>",
	Example:     "help",
	ServerOnly:  false,
	AdminOnly:   false,

	Execute: func(ctx *pkg.Context) error {
		if len(ctx.Args) != 0 {
			command, found := ctx.Server.CommandHandler.GetCommand(ctx.Args[0])

			if !found {
				_, err := ctx.Discord.ChannelMessageSend(ctx.Message.ChannelID,
					"Kunde inte hitta information om kommandot då den inte finns")
				return err
			}

			embed := pkg.NewEmbed().
				SetTitle(fmt.Sprintf("**%v**", command.Name)).
				SetDescription(fmt.Sprintf("*%v*", command.Description)).
				AddField("ANVÄNDNING", fmt.Sprintf(">`%v`", command.Usage)).
				AddField("EXEMPEL", fmt.Sprintf(">`%v`", command.Example)).
				SetColor(4086462)

			if len(command.Aliases) > 0 {
				embed.AddField("ALIAS", fmt.Sprintf("`%v`", strings.Join(command.Aliases[:], ", ")))
			}

			_, err := ctx.ReplyEmbed(embed.MessageEmbed)
			if err != nil {
				return err
			}

			return nil
		}

		if !ctx.IsDM() {
			_, err := ctx.Reply("Ett direkt meddelande har skickats till dig med alla kommandon. Du finner dem längst upp till vänster.")

			if err != nil {
				return err
			}
		}

		commands := ctx.Server.CommandHandler.GetCommands("")
		groups := make(map[string][]pkg.Command)

		for _, command := range commands {
			groups[command.Group] = append(groups[command.Group], command)
		}

		embeds := make(map[string]*discordgo.MessageEmbed)

		description := "__Tryck knapparna längst ned för att byta sida__.\n" +
			"Du kan få mer information om ett kommando genom att köra `>help <command>`.\n\n" +
			":information_source: **--** Denna sida\n" +
			":desktop: **--** Digital Ungdom kommandon\n" +
			":musical_note:  **--** Musik kommandon\n" +
			":tools: **--** Misc kommandon\n"

		admin := false

		for _, member := range ctx.Server.Guild.Members {
			if member.User.ID == ctx.Message.Author.ID {
				if pkg.StringInSlice(ctx.Server.Roles.Admin.ID, member.Roles) {
					description += "🚨 **--** Admin kommandon\n"
					admin = true
				}
			}
		}

		description += ":fire:  **--** Stäng hjälp sida\n"

		embeds["info"] = pkg.NewEmbed().
			SetTitle("**HJÄLP SIDA**").
			SetDescription(description).
			SetColor(4086462).
			MessageEmbed

		for group, groupCommands := range groups {
			embed := pkg.NewEmbed().
				SetTitle(fmt.Sprintf("**%v**", group)).
				SetDescription(fmt.Sprintf("Hjälp sida för kommandon i *%v* gruppen", group))

			for _, command := range groupCommands {
				embed.AddField(fmt.Sprintf("__**%v**__", command.Name),
					fmt.Sprintf("%v\n>`%v`", command.Description, command.Usage))
			}

			embed = embed.SetColor(4086462)

			embeds[group] = embed.MessageEmbed
		}

		privateDM, err := ctx.Discord.UserChannelCreate(ctx.Message.Author.ID)
		if err != nil {
			return err
		}

		reactionator := pkg.NewReactionator(privateDM.ID, ctx.Discord, ctx.Server.ReactionListener,
			true, true, pkg.ReactionatorTypeHelp, ctx.Message.Author)

		err = reactionator.AddDefaultPage(groupReactions["info"], embeds["info"])
		if err != nil {
			return err
		}

		for _, group := range groupReactionOrder {
			err = reactionator.Add(groupReactions[group], embeds[group])
			if err != nil {
				return err
			}
		}

		if admin {
			err = reactionator.Add(groupReactions["admin"], embeds["admin"])
			if err != nil {
				return err
			}
		}

		err = reactionator.CloseButton()
		if err != nil {
			return err
		}

		reactionator.CloseAfter(3 * time.Minute)

		if activeReactionators, ok := ctx.Server.ReactionListener.Users[ctx.Message.Author.ID]; ok {
			if activeReactionators.Help != nil {
				activeReactionators.Help.Close()
			}
		}

		err = reactionator.Initiate()
		if err != nil {
			return err
		}

		return nil
	},
}
