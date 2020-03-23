package admin

import (
	"github.com/bwmarrin/discordgo"
	"github.com/digitalungdom-se/dub/events"
	"github.com/digitalungdom-se/dub/pkg"
)

var Join = pkg.Command{
	Name:        "join",
	Description: "Simulerar en användare joinar",
	Aliases:     []string{},
	Group:       "admin",
	Usage:       "join @<user>",
	Example:     "join @kelszo#6200",
	ServerOnly:  true,
	AdminOnly:   true,

	Execute: func(ctx *pkg.Context) error {
		var member *discordgo.Member

		for _, guildMember := range ctx.Server.Guild.Members {
			if guildMember.User.ID == ctx.Message.Mentions[0].ID {
				member = guildMember
			}
		}

		guildMemberAdd := &discordgo.GuildMemberAdd{Member: member}

		events.GuildMemberAddHandler(ctx.Server)(ctx.Discord, guildMemberAdd)
		return nil
	},
}
