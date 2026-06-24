package utility

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"skyvern/internal/config"
	"skyvern/internal/manager"
	"skyvern/internal/storage"
)

func init() {
	manager.RegisterHelp("backup", []manager.HelpPage{
		{
			Command:     "Backup Create",
			Syntax:      ".backup create",
			Description: "Create a backup of the current server.",
		},
		{
			Command:     "Backup Load",
			Syntax:      ".backup load <backup_id>",
			Description: "Load a backup to restore the server structure.",
		},
		{
			Command:     "Backup List",
			Syntax:      ".backup list",
			Description: "List all backups created for this server.",
		},
		{
			Command:     "Backup Delete",
			Syntax:      ".backup delete <backup_id>",
			Description: "Delete a backup.",
		},
		{
			Command:     "Backup Info",
			Syntax:      ".backup info <backup_id>",
			Description: "Display information about a backup.",
		},
		{
			Command:     "Backup Export",
			Syntax:      ".backup export <backup_id> [password]",
			Description: "Export a backup to an AES encrypted file.",
		},
		{
			Command:     "Backup Import",
			Syntax:      ".backup import [password]",
			Description: "Import an encrypted backup file (attach or reply to the file).",
		},
	})
}

func genBackupID() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return fmt.Sprintf("bk-%s", hex.EncodeToString(b))
}

var Backup = &manager.Command{
	Trigger:     "backup",
	Aliases:     []string{"backups"},
	Name:        "backup",
	Description: "Server backup and restore system",
	Category:    "utility",
	Execute: func(ctx *manager.CommandContext) error {
		gid := ctx.GuildID()
		if gid == "" {
			return ctx.Reply("[!] This command must be used in a server.")
		}
		isOwner := isBotOwner(ctx.AuthorID())
		if !isOwner {
			p, err := ctx.Session.UserChannelPermissions(ctx.AuthorID(), ctx.ChanID())
			if err != nil || (p&discordgo.PermissionAdministrator) == 0 {
				return ctx.Reply("[!] Only Server Administrators or Bot Owners can use backup commands.")
			}
		}
		if len(ctx.Args) == 0 {
			return ctx.SendHelp("backup")
		}
		sub := strings.ToLower(ctx.Args[0])
		switch sub {
		case "create":
			_ = ctx.Reply("[*] Scanning server roles and channels...")
			g, err := ctx.Session.Guild(gid)
			if err != nil {
				return ctx.Reply(fmt.Sprintf("[!] Failed to fetch guild: %v", err))
			}
			chans, err := ctx.Session.GuildChannels(gid)
			if err != nil {
				return ctx.Reply(fmt.Sprintf("[!] Failed to fetch channels: %v", err))
			}
			roles, err := ctx.Session.GuildRoles(gid)
			if err != nil {
				return ctx.Reply(fmt.Sprintf("[!] Failed to fetch roles: %v", err))
			}
			roleMap := make(map[string]*discordgo.Role)
			for _, r := range roles {
				roleMap[r.ID] = r
			}
			var rBackups []storage.RoleBackup
			for _, r := range roles {
				if r.ID != gid && r.Managed {
					continue
				}
				rBackups = append(rBackups, storage.RoleBackup{
					Name:        r.Name,
					Color:       r.Color,
					Hoist:       r.Hoist,
					Mentionable: r.Mentionable,
					Permissions: r.Permissions,
					Position:    r.Position,
				})
			}
			var cBackups []storage.ChannelBackup
			for _, c := range chans {
				var ows []storage.OverwriteBackup
				for _, o := range c.PermissionOverwrites {
					name := o.ID
					if o.Type == discordgo.PermissionOverwriteTypeRole {
						if r, ok := roleMap[o.ID]; ok {
							name = r.Name
						}
					}
					ows = append(ows, storage.OverwriteBackup{
						ID:    o.ID,
						Name:  name,
						Type:  int(o.Type),
						Allow: o.Allow,
						Deny:  o.Deny,
					})
				}
				cBackups = append(cBackups, storage.ChannelBackup{
					ID:                   c.ID,
					Name:                 c.Name,
					Type:                 int(c.Type),
					Topic:                c.Topic,
					Bitrate:              c.Bitrate,
					UserLimit:            c.UserLimit,
					ParentID:             c.ParentID,
					Position:             c.Position,
					NSFW:                 c.NSFW,
					RateLimitPerUser:     c.RateLimitPerUser,
					PermissionOverwrites: ows,
				})
			}
			bk := storage.ServerBackup{
				GuildID:   gid,
				Name:      g.Name,
				CreatedAt: time.Now(),
				Roles:     rBackups,
				Channels:  cBackups,
			}

			// Rotation: keep only 3 backups max per guild
			list, _ := ctx.DB.ListGuildBackups(gid)
			if len(list) >= 3 {
				type bkItem struct {
					id string
					ts time.Time
				}
				var items []bkItem
				for k, v := range list {
					items = append(items, bkItem{id: k, ts: v.CreatedAt})
				}
				sort.Slice(items, func(i, j int) bool {
					return items[i].ts.Before(items[j].ts)
				})
				for i := 0; i <= len(items)-3; i++ {
					_ = ctx.DB.DeleteBackup(items[i].id)
				}
			}

			bid := genBackupID()
			if err := ctx.DB.SaveBackup(bid, bk); err != nil {
				return ctx.Reply(fmt.Sprintf("[!] Failed to save backup: %v", err))
			}
			emb := config.Build(ctx.Cfg, config.EmbedOpt{
				Title:       "Server Backup Created",
				Description: fmt.Sprintf("[+] Successfully backed up server **%s**.\n\n**Backup ID:** `%s`\n**Roles:** %d\n**Channels:** %d", g.Name, bid, len(rBackups), len(cBackups)),
			})
			emb.Color = 0x2b2d31
			return ctx.Respond(emb)
		case "load":
			if len(ctx.Args) < 2 {
				return ctx.Reply("[!] Usage: `.backup load <backup_id>`")
			}
			bid := ctx.Args[1]
			bk, err := ctx.DB.GetBackup(bid)
			if err != nil {
				return ctx.Reply("[!] Backup not found or invalid.")
			}
			botPerms, err := ctx.Session.UserChannelPermissions(ctx.Session.State.User.ID, ctx.ChanID())
			if err != nil || (botPerms&discordgo.PermissionAdministrator) == 0 {
				return ctx.Reply("[!] The bot must have **Administrator** permission to restore backups.")
			}
			emb := config.Build(ctx.Cfg, config.EmbedOpt{
				Title: "CRITICAL WARNING",
				Description: fmt.Sprintf("You are about to load backup **%s** (Created <t:%d:R>).\n\n"+
					"**This will delete all existing channels and roles** (except bot roles and @everyone).\n\n"+
					"Are you absolutely sure you want to proceed?", bk.Name, bk.CreatedAt.Unix()),
			})
			emb.Color = 0xff0000
			components := []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.Button{
							Label:    "Confirm Restore",
							Style:    discordgo.DangerButton,
							CustomID: "backup_confirm:" + bid,
						},
						discordgo.Button{
							Label:    "Cancel",
							Style:    discordgo.SecondaryButton,
							CustomID: "backup_cancel:" + bid,
						},
					},
				},
			}
			if ctx.Interact != nil {
				return ctx.Session.InteractionRespond(ctx.Interact, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Embeds:     []*discordgo.MessageEmbed{emb},
						Components: components,
					},
				})
			}
			_, err = ctx.Session.ChannelMessageSendComplex(ctx.ChanID(), &discordgo.MessageSend{
				Embeds:     []*discordgo.MessageEmbed{emb},
				Components: components,
			})
			return err
		case "list":
			list, err := ctx.DB.ListGuildBackups(gid)
			if err != nil || len(list) == 0 {
				return ctx.Reply("[!] No backups found for this server.")
			}
			var sb strings.Builder
			for id, bk := range list {
				sb.WriteString(fmt.Sprintf("• `%s` | **%s** (<t:%d:R>)\n  └ %d Roles | %d Channels\n", 
					id, bk.Name, bk.CreatedAt.Unix(), len(bk.Roles), len(bk.Channels)))
			}
			emb := config.Build(ctx.Cfg, config.EmbedOpt{
				Title:       "Server Backups",
				Description: sb.String(),
			})
			emb.Color = 0x2b2d31
			return ctx.Respond(emb)
		case "delete":
			if len(ctx.Args) < 2 {
				return ctx.Reply("[!] Usage: `.backup delete <backup_id>`")
			}
			bid := ctx.Args[1]
			bk, err := ctx.DB.GetBackup(bid)
			if err != nil {
				return ctx.Reply("[!] Backup not found.")
			}
			if bk.GuildID != gid {
				return ctx.Reply("[!] You can only delete backups created for this server.")
			}
			_ = ctx.DB.DeleteBackup(bid)
			return ctx.Reply("[+] Backup deleted successfully.")
		case "info":
			if len(ctx.Args) < 2 {
				return ctx.Reply("[!] Usage: `.backup info <backup_id>`")
			}
			bid := ctx.Args[1]
			bk, err := ctx.DB.GetBackup(bid)
			if err != nil {
				return ctx.Reply("[!] Backup not found.")
			}
			cats, text, voice := 0, 0, 0
			for _, ch := range bk.Channels {
				switch ch.Type {
				case 4:
					cats++
				case 0, 5:
					text++
				case 2, 13:
					voice++
				default:
					text++
				}
			}
			emb := config.Build(ctx.Cfg, config.EmbedOpt{
				Title: "Backup Details - " + bid,
				Description: fmt.Sprintf("• **Server Name:** %s\n"+
					"• **Created At:** <t:%d:F> (<t:%d:R>)\n"+
					"• **Roles:** %d\n"+
					"• **Channels:** %d total\n"+
					"  └ %d Categories\n"+
					"  └ %d Text Channels\n"+
					"  └ %d Voice/Stage Channels",
					bk.Name, bk.CreatedAt.Unix(), bk.CreatedAt.Unix(), len(bk.Roles), len(bk.Channels), cats, text, voice),
			})
			emb.Color = 0x2b2d31
			return ctx.Respond(emb)
		case "export":
			if len(ctx.Args) < 2 {
				return ctx.Reply("[!] Usage: `.backup export <backup_id> [password]`")
			}
			bid := ctx.Args[1]
			bk, err := ctx.DB.GetBackup(bid)
			if err != nil {
				return ctx.Reply("[!] Backup not found.")
			}
			if bk.GuildID != gid && !isBotOwner(ctx.AuthorID()) {
				return ctx.Reply("[!] You can only export backups created for this server.")
			}
			pass := ""
			if len(ctx.Args) > 2 {
				pass = ctx.Args[2]
			}
			secret := getSecret(ctx, pass)
			raw, err := json.Marshal(bk)
			if err != nil {
				return ctx.Reply(fmt.Sprintf("[!] Failed to serialize backup: %v", err))
			}
			encrypted, err := encryptBackup(raw, secret)
			if err != nil {
				return ctx.Reply(fmt.Sprintf("[!] Failed to encrypt backup: %v", err))
			}
			sr := bytes.NewReader(encrypted)
			fname := fmt.Sprintf("%s.skyvern", bid)
			ms := &discordgo.MessageSend{
				Files: []*discordgo.File{
					{
						Name:   fname,
						Reader: sr,
					},
				},
			}
			desc := fmt.Sprintf("[+] **Backup `%s` successfully exported!**\n\nDownload the attached file to share or import it.", bid)
			if pass != "" {
				desc += "\n\n[!] **This file is password-protected.** Make sure to share the password along with the file!"
			} else {
				desc += "\n\n🔑 Encrypted using bot-specific credentials. It can be imported back without a password on this bot instance."
			}
			emb := config.Build(ctx.Cfg, config.EmbedOpt{
				Title:       "Backup Exported",
				Description: desc,
			})
			emb.Color = 0x2b2d31
			ms.Embeds = []*discordgo.MessageEmbed{emb}
			if ctx.Interact != nil {
				return ctx.Session.InteractionRespond(ctx.Interact, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Embeds:     ms.Embeds,
						Files:      ms.Files,
					},
				})
			}
			_, err = ctx.Session.ChannelMessageSendComplex(ctx.ChanID(), ms)
			return err
		case "import":
			pass := ""
			if len(ctx.Args) > 1 {
				pass = ctx.Args[1]
			}
			var attach *discordgo.MessageAttachment
			if ctx.Message != nil && len(ctx.Message.Attachments) > 0 {
				attach = ctx.Message.Attachments[0]
			} else if ctx.Message != nil && ctx.Message.ReferencedMessage != nil && len(ctx.Message.ReferencedMessage.Attachments) > 0 {
				attach = ctx.Message.ReferencedMessage.Attachments[0]
			}
			if attach == nil {
				return ctx.Reply("[!] Please attach or reply to a `.skyvern` backup file to import.")
			}
			res, err := http.Get(attach.URL)
			if err != nil {
				return ctx.Reply(fmt.Sprintf("[!] Failed to download attachment: %v", err))
			}
			defer res.Body.Close()
			encrypted, err := io.ReadAll(res.Body)
			if err != nil {
				return ctx.Reply(fmt.Sprintf("[!] Failed to read attachment: %v", err))
			}
			secret := getSecret(ctx, pass)
			decrypted, err := decryptBackup(encrypted, secret)
			if err != nil {
				return ctx.Reply("[!] **Decryption failed.** If this backup is password-protected, please provide the password: `.backup import [password]`")
			}
			var bk storage.ServerBackup
			if err := json.Unmarshal(decrypted, &bk); err != nil {
				return ctx.Reply("[!] **Import failed.** The decrypted file is corrupted or not a valid backup.")
			}
			if bk.Name == "" || len(bk.Channels) == 0 {
				return ctx.Reply("[!] **Import failed.** Invalid backup structure (no channels/name found).")
			}
			newBid := genBackupID()
			bk.GuildID = gid
			bk.CreatedAt = time.Now()
			if err := ctx.DB.SaveBackup(newBid, bk); err != nil {
				return ctx.Reply(fmt.Sprintf("[!] Failed to save imported backup: %v", err))
			}
			emb := config.Build(ctx.Cfg, config.EmbedOpt{
				Title:       "Backup Imported Successfully",
				Description: fmt.Sprintf("[+] **Successfully imported backup!**\n\n**New Backup ID:** `%s`\n**Original Server Name:** **%s**\n**Roles:** %d\n**Channels:** %d\n\nYou can now load this backup using `.backup load %s`.", newBid, bk.Name, len(bk.Roles), len(bk.Channels), newBid),
			})
			emb.Color = 0x2b2d31
			return ctx.Respond(emb)
		default:
			return ctx.Reply("Unknown subcommand. Options: `create`, `load`, `list`, `delete`, `info`, `export`, `import`")
		}
	},
}

func HandleBackupConfirm(s *discordgo.Session, i *discordgo.InteractionCreate, m *manager.Manager) {
	customID := i.MessageComponentData().CustomID
	bid := strings.TrimPrefix(customID, "backup_confirm:")
	db := m.DB()
	bk, err := db.GetBackup(bid)
	if err != nil {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "[!] Backup not found.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}
	gid := i.GuildID
	uid := i.Member.User.ID
	p, err := s.UserChannelPermissions(uid, i.ChannelID)
	if err != nil || (p&discordgo.PermissionAdministrator) == 0 {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "[!] You must be a Server Administrator to restore backups.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    "**Restoration started, please wait...**",
			Embeds:     nil,
			Components: nil,
		},
	})
	go func() {
		chans, err := s.GuildChannels(gid)
		if err == nil {
			for _, ch := range chans {
				if ch.ID == i.ChannelID {
					continue
				}
				_, _ = s.ChannelDelete(ch.ID)
			}
		}
		roles, err := s.GuildRoles(gid)
		if err == nil {
			for _, r := range roles {
				if r.ID == gid || r.Managed {
					continue
				}
				_ = s.GuildRoleDelete(gid, r.ID)
			}
		}
		roleMap := make(map[string]string)
		for _, r := range bk.Roles {
			if r.Name == "@everyone" {
				_, _ = s.GuildRoleEdit(gid, gid, &discordgo.RoleParams{
					Permissions: &r.Permissions,
				})
				roleMap["@everyone"] = gid
				continue
			}
			newRole, err := s.GuildRoleCreate(gid, &discordgo.RoleParams{
				Name:        r.Name,
				Color:       &r.Color,
				Hoist:       &r.Hoist,
				Mentionable: &r.Mentionable,
				Permissions: &r.Permissions,
			})
			if err == nil {
				roleMap[r.Name] = newRole.ID
			}
		}
		categoryMap := make(map[string]string)
		for _, ch := range bk.Channels {
			if ch.Type != 4 {            
				continue
			}
			var ows []*discordgo.PermissionOverwrite
			for _, o := range ch.PermissionOverwrites {
				targetID := o.ID
				if o.Type == 0 {        
					targetID = roleMap[o.Name]
				}
				if targetID != "" {
					ows = append(ows, &discordgo.PermissionOverwrite{
						ID:    targetID,
						Type:  discordgo.PermissionOverwriteType(o.Type),
						Allow: o.Allow,
						Deny:  o.Deny,
					})
				}
			}
			newCh, err := s.GuildChannelCreateComplex(gid, discordgo.GuildChannelCreateData{
				Name:                 ch.Name,
				Type:                 discordgo.ChannelType(ch.Type),
				Position:             ch.Position,
				NSFW:                 ch.NSFW,
				RateLimitPerUser:     ch.RateLimitPerUser,
				PermissionOverwrites: ows,
			})
			if err == nil {
				categoryMap[ch.ID] = newCh.ID
			}
		}
		var textChanID string
		for _, ch := range bk.Channels {
			if ch.Type == 4 {
				continue
			}
			var ows []*discordgo.PermissionOverwrite
			for _, o := range ch.PermissionOverwrites {
				targetID := o.ID
				if o.Type == 0 {        
					targetID = roleMap[o.Name]
				}
				if targetID != "" {
					ows = append(ows, &discordgo.PermissionOverwrite{
						ID:    targetID,
						Type:  discordgo.PermissionOverwriteType(o.Type),
						Allow: o.Allow,
						Deny:  o.Deny,
					})
				}
			}
			newParentID := categoryMap[ch.ParentID]
			newCh, err := s.GuildChannelCreateComplex(gid, discordgo.GuildChannelCreateData{
				Name:                 ch.Name,
				Type:                 discordgo.ChannelType(ch.Type),
				Topic:                ch.Topic,
				Bitrate:              ch.Bitrate,
				UserLimit:            ch.UserLimit,
				ParentID:             newParentID,
				Position:             ch.Position,
				NSFW:                 ch.NSFW,
				RateLimitPerUser:     ch.RateLimitPerUser,
				PermissionOverwrites: ows,
			})
			if err == nil && textChanID == "" && ch.Type == 0 {
				textChanID = newCh.ID
			}
		}
		if textChanID != "" {
			resCfg, ok := m.ResolvedCfgFor(s.State.User.ID)
			if !ok {
				resCfg = config.Resolve(config.GetGlobal(), config.BotInst{})
			}
			emb := config.Build(resCfg, config.EmbedOpt{
				Title:       "Server Restored Successfully",
				Description: fmt.Sprintf("[+] **The server layout has been successfully rebuilt from backup: `%s`**\n\n"+
					"**Restored Statistics:**\n"+
					"• **Roles Created:** %d\n"+
					"• **Channels & Categories Created:** %d\n\n"+
					"*All permission overrides and hierarchy positions have been reapplied.*",
					bk.Name, len(bk.Roles), len(bk.Channels)),
			})
			emb.Color = 0x2b2d31
			_, _ = s.ChannelMessageSendEmbed(textChanID, emb)
		}
		_, _ = s.ChannelDelete(i.ChannelID)
	}()
}

func HandleBackupCancel(s *discordgo.Session, i *discordgo.InteractionCreate, m *manager.Manager) {
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    "[!] **Restoration cancelled.**",
			Embeds:     nil,
			Components: nil,
		},
	})
}