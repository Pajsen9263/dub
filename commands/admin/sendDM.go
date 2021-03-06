package admin

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/cbroglie/mustache"
	"github.com/digitalungdom-se/dub/internal"
	"github.com/digitalungdom-se/dub/pkg"
)

var SendDM = internal.Command{
	Name:        "senddm",
	Description: "Skickar meddelande till medlemmarna via direct message",
	Aliases:     []string{},
	Group:       "admin",
	Usage:       "senddm <[!]groupID|userID|\"EVERYONE\"> <message|[{{{nick|username|mention}}}]>",
	Example:     "senddm !568110630809370624 Hej **{{{alias}}}**, vi ser att du inte är verifierad, bli det gärna!",
	ServerOnly:  true,
	AdminOnly:   true,

	Execute: func(ctx *pkg.Context, server *internal.Server) error {
		if len(ctx.Args) < 2 {
			ctx.Reply("Du måste ange groupID|userID|\"EVERYONE\" och ett meddelande.")
			return nil
		}

		if strings.ToUpper(ctx.Args[0]) != "EVERYONE" && !pkg.IsSnowflake(ctx.Args[0]) && !pkg.IsSnowflake(string(ctx.Args[0][1:])) {
			ctx.Reply("Första argumentet måste vara en snowflake eller EVERYONE.")
			return nil
		}

		messageTemplate := strings.Join(ctx.Args[1:], " ")

		var membersToSend []*discordgo.Member

		if strings.ToUpper(ctx.Args[0]) == "EVERYONE" {
			membersToSend = server.Guild.Members
		} else {
			snowflake := ctx.Args[0]
			isOpposite := false
			isRole := false

			if string(snowflake[0]) == "!" {
				snowflake = string(snowflake[1:])
				isOpposite = true
			}

			for _, role := range server.Guild.Roles {
				if role.ID == snowflake {
					isRole = true
				}
			}

			if isOpposite {
				if !isRole {
					ctx.Reply("Ingen roll med det IDt finns.")
					return nil
				}

				for _, member := range server.Guild.Members {
					if !pkg.StringInSlice(snowflake, member.Roles) {
						membersToSend = append(membersToSend, member)
					}
				}
			} else {
				if isRole {
					for _, member := range server.Guild.Members {
						if pkg.StringInSlice(snowflake, member.Roles) {
							membersToSend = append(membersToSend, member)
						}
					}
				} else {
					for _, member := range server.Guild.Members {
						if member.User.ID == snowflake {
							membersToSend = append(membersToSend, member)
						}
					}
				}
			}
		}

		for _, member := range membersToSend {
			if member.User.Bot {
				continue
			}

			message, err := mustache.Render(messageTemplate, map[string]string{"nick": member.Nick, "username": member.User.Username, "mention": member.Mention()})
			if err != nil {
				ctx.Reply(fmt.Sprintf("Error med att kompilera meddelandet till %v.", member.Nick))
				return err
			}

			privateDM, err := ctx.Discord.UserChannelCreate(member.User.ID)
			if err != nil {
				ctx.Reply(fmt.Sprintf("Error med att skicka meddelandet till %v.", member.Nick))
				return err
			}

			ctx.Discord.ChannelMessageSend(privateDM.ID, message)
			if err != nil {
				ctx.Reply(fmt.Sprintf("Error med att skicka meddelandet till %v.", member.Nick))
				return err
			}
		}

		return nil
	},
}
