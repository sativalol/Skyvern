package general
import (
	"encoding/json"
	"fmt"
	"regexp"
	"skyvern/internal/manager"
	"strconv"
	"strings"
	"time"
	"github.com/bwmarrin/discordgo"
)
var rxBtnRole = regexp.MustCompile(`^<@&(\d+)>$`)
var rxBtnMsgLink = regexp.MustCompile(`channels/(?:\d+|@me)/(\d+)/(\d+)`)
func resolveButtonRole(s *discordgo.Session, gid, query string) string {
	q := strings.TrimSpace(query)
	if q == "" {
		return ""
	}
	if m := rxBtnRole.FindStringSubmatch(q); len(m) > 1 {
		return m[1]
	}
	roles, err := s.GuildRoles(gid)
	if err != nil {
		return ""
	}
	for _, r := range roles {
		if r.ID == q {
			return r.ID
		}
	}
	ql := strings.ToLower(q)
	for _, r := range roles {
		if strings.ToLower(r.Name) == ql {
			return r.ID
		}
	}
	return ""
}
func init() {
	manager.RegisterHelp("buttonrole", []manager.HelpPage{
		{
			Command:     "Button Role Help",
			Syntax:      ".buttonrole",
			Description: "Set up and manage button roles.",
		},
		{
			Command:     "Button Role List",
			Syntax:      ".buttonrole list",
			Description: "View a list of all button roles in the server.",
		},
		{
			Command:     "Button Role Add",
			Syntax:      ".buttonrole add <message link> <role> <style> <emoji> <label>",
			Description: "Add a button role to a message.",
		},
		{
			Command:     "Button Role Reset",
			Syntax:      ".buttonrole reset",
			Description: "Clears all button roles from the server.",
		},
		{
			Command:     "Button Role Remove All",
			Syntax:      ".buttonrole removeall <message link>",
			Description: "Removes all button roles from a message.",
		},
		{
			Command:     "Button Role Remove Index",
			Syntax:      ".buttonrole remove <message link> <index>",
			Description: "Removes a specific button role from a message by index.",
		},
	})
}
var ButtonRole = &manager.Command{
	Trigger:     "buttonrole",
	Aliases:     []string{"btnrole"},
	Name:        "buttonrole",
	Description: "Manage self-assignable roles using button panels",
	Category:    "general",
	Execute: func(ctx *manager.CommandContext) error {
		if !checkPerm(ctx, discordgo.PermissionManageRoles) {
			return ctx.Reply("[!] You need Manage Roles permission to use this command.")
		}
		if len(ctx.Args) == 0 {
			return ctx.SendHelp("buttonrole")
		}
		sub := strings.ToLower(ctx.Args[0])
		gid := ctx.GuildID()
		switch sub {
		case "list":
			list, err := ctx.DB.ListButtonRoles(gid)
			if err != nil || len(list) == 0 {
				return ctx.Reply("[*] No button roles configured for this server.")
			}
			var sb strings.Builder
			sb.WriteString("**Configured Button Roles:**\n\n")
			for k, roleID := range list {
				parts := strings.Split(k, ":")
				if len(parts) >= 2 {
					msgID := parts[0]
					customID := parts[1]
					sb.WriteString(fmt.Sprintf("- Message ID: `%s` | Button ID: `%s` -> <@&%s>\n", msgID, customID, roleID))
				}
			}
			return ctx.Reply(sb.String())
		case "add":
			if len(ctx.Args) < 6 {
				return ctx.Reply("Usage: `.buttonrole add <message link> <role> <style> <emoji> <label>`")
			}
			link := ctx.Args[1]
			roleArg := ctx.Args[2]
			styleStr := ctx.Args[3]
			emojiStr := ctx.Args[4]
			label := strings.Join(ctx.Args[5:], " ")
			parts := rxBtnMsgLink.FindStringSubmatch(link)
			if len(parts) < 3 {
				return ctx.Reply("[!] Invalid message link.")
			}
			chanID := parts[1]
			msgID := parts[2]
			rid := resolveButtonRole(ctx.Session, gid, roleArg)
			if rid == "" {
				return ctx.Reply("[!] Could not resolve role.")
			}
			botMember, err := ctx.Session.GuildMember(gid, ctx.ClientID)
			if err != nil {
				return ctx.Reply("[!] Failed to verify bot status.")
			}
			roles, err := ctx.Session.GuildRoles(gid)
			if err != nil {
				return ctx.Reply("[!] Failed to fetch guild roles.")
			}
			botMaxPos := -1
			var targetRole *discordgo.Role
			for _, r := range roles {
				if r.ID == rid {
					targetRole = r
				}
				for _, botRoleID := range botMember.Roles {
					if r.ID == botRoleID && r.Position > botMaxPos {
						botMaxPos = r.Position
					}
				}
			}
			if targetRole != nil && targetRole.Position >= botMaxPos {
				return ctx.Reply(fmt.Sprintf("[!] Security Alert: Role <@&%s> is higher than or equal to the bot's own role. Action blocked.", rid))
			}
			style := discordgo.PrimaryButton
			switch strings.ToLower(styleStr) {
			case "secondary", "grey", "gray":
				style = discordgo.SecondaryButton
			case "success", "green":
				style = discordgo.SuccessButton
			case "danger", "red":
				style = discordgo.DangerButton
			case "primary", "blue":
				style = discordgo.PrimaryButton
			}
			var compEmoji *discordgo.ComponentEmoji
			if emojiStr != "" && strings.ToLower(emojiStr) != "none" {
				compEmoji = &discordgo.ComponentEmoji{Name: emojiStr}
				if strings.HasPrefix(emojiStr, "<") {
					rx := regexp.MustCompile(`<a?:([a-zA-Z0-9_]+):(\d+)>`)
					if m := rx.FindStringSubmatch(emojiStr); len(m) > 2 {
						compEmoji = &discordgo.ComponentEmoji{
							Name: m[1],
							ID:   m[2],
						}
					}
				}
			}
			msg, err := ctx.Session.ChannelMessage(chanID, msgID)
			if err != nil {
				return ctx.Reply("[!] Message not found or inaccessible.")
			}
			if msg.Author.ID != ctx.ClientID {
				return ctx.Reply("[!] I can only add button roles to messages sent by me.")
			}
			customID := fmt.Sprintf("btnrole_%s_%d", rid, time.Now().UnixNano())
			newBtn := discordgo.Button{
				Label:    label,
				Style:    style,
				CustomID: customID,
				Emoji:    compEmoji,
			}
			var actRows []discordgo.ActionsRow
			rawBytes, err := json.Marshal(msg.Components)
			if err == nil {
				_ = json.Unmarshal(rawBytes, &actRows)
			}
			if len(actRows) == 0 {
				actRows = append(actRows, discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{newBtn},
				})
			} else {
				lastIdx := len(actRows) - 1
				if len(actRows[lastIdx].Components) < 5 {
					actRows[lastIdx].Components = append(actRows[lastIdx].Components, newBtn)
				} else if len(actRows) < 5 {
					actRows = append(actRows, discordgo.ActionsRow{
						Components: []discordgo.MessageComponent{newBtn},
					})
				} else {
					return ctx.Reply("[!] Message has reached maximum number of buttons.")
				}
			}
			var nextComps []discordgo.MessageComponent
			for _, r := range actRows {
				nextComps = append(nextComps, r)
			}
			_, err = ctx.Session.ChannelMessageEditComplex(&discordgo.MessageEdit{
				Channel:    chanID,
				ID:         msgID,
				Components: &nextComps,
			})
			if err != nil {
				return ctx.Reply(fmt.Sprintf("[!] Failed to edit message: %v", err))
			}
			_ = ctx.DB.SaveButtonRole(gid, msgID, customID, rid)
			return ctx.Reply("[+] Button role added successfully.")
		case "removeall":
			if len(ctx.Args) < 2 {
				return ctx.Reply("Usage: `.buttonrole removeall <message link>`")
			}
			link := ctx.Args[1]
			parts := rxBtnMsgLink.FindStringSubmatch(link)
			if len(parts) < 3 {
				return ctx.Reply("[!] Invalid message link.")
			}
			chanID := parts[1]
			msgID := parts[2]
			_, err := ctx.Session.ChannelMessage(chanID, msgID)
			if err != nil {
				return ctx.Reply("[!] Message not found.")
			}
			_, err = ctx.Session.ChannelMessageEditComplex(&discordgo.MessageEdit{
				Channel:    chanID,
				ID:         msgID,
				Components: &[]discordgo.MessageComponent{},
			})
			if err != nil {
				return ctx.Reply(fmt.Sprintf("[!] Failed to remove buttons from message: %v", err))
			}
			_ = ctx.DB.DeleteAllButtonRolesForMsg(gid, msgID)
			return ctx.Reply("[+] Removed all button roles from message.")
		case "remove":
			if len(ctx.Args) < 3 {
				return ctx.Reply("Usage: `.buttonrole remove <message link> <index>`")
			}
			link := ctx.Args[1]
			idxStr := ctx.Args[2]
			index, err := strconv.Atoi(idxStr)
			if err != nil {
				return ctx.Reply("[!] Index must be an integer.")
			}
			parts := rxBtnMsgLink.FindStringSubmatch(link)
			if len(parts) < 3 {
				return ctx.Reply("[!] Invalid message link.")
			}
			chanID := parts[1]
			msgID := parts[2]
			msg, err := ctx.Session.ChannelMessage(chanID, msgID)
			if err != nil {
				return ctx.Reply("[!] Message not found.")
			}
			var actRows []discordgo.ActionsRow
			rawBytes, err := json.Marshal(msg.Components)
			if err == nil {
				_ = json.Unmarshal(rawBytes, &actRows)
			}
			type btnPos struct {
				row      int
				col      int
				customID string
			}
			var posList []btnPos
			for rIdx, row := range actRows {
				for cIdx, comp := range row.Components {
					rawCompBytes, _ := json.Marshal(comp)
					var btn discordgo.Button
					if json.Unmarshal(rawCompBytes, &btn) == nil && btn.CustomID != "" {
						posList = append(posList, btnPos{row: rIdx, col: cIdx, customID: btn.CustomID})
					}
				}
			}
			targetIdx := index - 1
			if targetIdx < 0 || targetIdx >= len(posList) {
				targetIdx = index                       
				if targetIdx < 0 || targetIdx >= len(posList) {
					return ctx.Reply("[!] Button index out of range.")
				}
			}
			pos := posList[targetIdx]
			rowComps := actRows[pos.row].Components
			actRows[pos.row].Components = append(rowComps[:pos.col], rowComps[pos.col+1:]...)
			var nextComps []discordgo.MessageComponent
			for _, r := range actRows {
				if len(r.Components) > 0 {
					nextComps = append(nextComps, r)
				}
			}
			_, err = ctx.Session.ChannelMessageEditComplex(&discordgo.MessageEdit{
				Channel:    chanID,
				ID:         msgID,
				Components: &nextComps,
			})
			if err != nil {
				return ctx.Reply(fmt.Sprintf("[!] Failed to update message: %v", err))
			}
			_ = ctx.DB.DeleteButtonRole(gid, msgID, pos.customID)
			return ctx.Reply("[+] Button role removed successfully.")
		case "reset":
			_ = ctx.DB.ClearButtonRoles(gid)
			return ctx.Reply("[+] Cleared all button role database registrations.")
		default:
			return ctx.SendHelp("buttonrole")
		}
	},
}