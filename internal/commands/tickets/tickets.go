package tickets
import (
	"bytes"
	"fmt"
	"skyvern/internal/config"
	"skyvern/internal/manager"
	"skyvern/internal/storage"
	"strings"
	"time"
	"github.com/bwmarrin/discordgo"
)
func init() {
	manager.RegisterHelp("tickets", []manager.HelpPage{
		{
			Command:     "Ticket Profiles",
			Syntax:      ".tickets profiles [add/remove] [role]",
			Description: "Configure server-wide ticket support roles.",
		},
		{
			Command:     "Ticket Panels",
			Syntax:      ".tickets panels [create/delete/set] [args...]",
			Description: "Manage ticket greeting panels.",
		},
		{
			Command:     "Ticket Options",
			Syntax:      ".tickets options [create/delete/link] [args...]",
			Description: "Manage panel options (support categories).",
		},
		{
			Command:     "Ticket Forms",
			Syntax:      ".tickets forms [add/remove] [name]",
			Description: "Configure reusable questionnaire forms.",
		},
		{
			Command:     "Ticket Blacklist",
			Syntax:      ".tickets blacklist <member/role>",
			Description: "Toggle a user or role in the ticket blacklist.",
		},
		{
			Command:     "Ticket List",
			Syntax:      ".tickets list",
			Description: "List all open ticket channels.",
		},
		{
			Command:     "Ticket Close",
			Syntax:      ".tickets close [channel] [reason]",
			Description: "Close the ticket (and lock channel from creator).",
		},
		{
			Command:     "Ticket Reopen",
			Syntax:      ".tickets reopen [channel] [reason]",
			Description: "Reopen a closed ticket.",
		},
		{
			Command:     "Ticket Claim",
			Syntax:      ".tickets claim [channel] [reason]",
			Description: "Claim a ticket as a support contact.",
		},
		{
			Command:     "Ticket Unclaim",
			Syntax:      ".tickets unclaim [channel]",
			Description: "Remove the claimer from the ticket.",
		},
		{
			Command:     "Ticket Delete",
			Syntax:      ".tickets delete [channel] [reason]",
			Description: "Permanently delete a ticket channel.",
		},
		{
			Command:     "Ticket Rename",
			Syntax:      ".tickets rename <channel> <name>",
			Description: "Rename the ticket channel.",
		},
		{
			Command:     "Ticket Move",
			Syntax:      ".tickets move <channel> <option/category>",
			Description: "Move the ticket to another option/category.",
		},
		{
			Command:     "Ticket Allow",
			Syntax:      ".tickets allow [member/role]",
			Description: "Allow a member or role to view the ticket channel.",
		},
		{
			Command:     "Ticket Deny",
			Syntax:      ".tickets deny <member/role>",
			Description: "Deny a member or role from viewing the ticket channel.",
		},
		{
			Command:     "Ticket Transcript",
			Syntax:      ".tickets transcript [channel] [reason]",
			Description: "Generate a chat log transcript.",
		},
		{
			Command:     "Ticket Resend",
			Syntax:      ".tickets resend <channel>",
			Description: "Resend a ticket panel message.",
		},
		{
			Command:     "Ticket Trainee",
			Syntax:      ".tickets trainee [grant/revoke/list] [args...]",
			Description: "Configure staff trainees.",
		},
		{
			Command:     "Ticket Staff Profile",
			Syntax:      ".tickets profile",
			Description: "Show personal claim/close stats.",
		},
		{
			Command:     "Ticket Staff Stats",
			Syntax:      ".tickets stats [member]",
			Description: "Show ticket metrics for a user.",
		},
		{
			Command:     "Ticket Action Reason",
			Syntax:      ".tickets reason <action> <target> <reason>",
			Description: "Record action notes on a ticket.",
		},
	})
}
var Tickets = &manager.Command{
	Trigger:     "tickets",
	Aliases:     []string{"ticket"},
	Name:        "tickets",
	Description: "Ticket system commands",
	Category:    "tickets",
	Execute: func(ctx *manager.CommandContext) error {
		if len(ctx.Args) == 0 {
			emb := config.Build(ctx.Cfg, config.EmbedOpt{
				Title: "🎫 Ticket System Setup Guide",
				Description: `**Step-by-Step Configuration Walkthrough:**
1️⃣ **Add Support Roles** (designate who manages tickets):
` + "`.tickets profiles add <@role_mention_or_id>`" + `
2️⃣ **Create a Ticket Panel** (where users open tickets):
` + "`.tickets panels create <panel_name> <#channel_mention_or_id>`" + `
3️⃣ **Create & Link Panel Options** (e.g. general support, billing):
` + "`.tickets options create <option_name> <emoji> <description> <category_id>`" + `
` + "`.tickets options link <panel_name> <option_name>`" + `
4️⃣ **Send/Resend the Greeting Panel**:
` + "`.tickets resend <#channel_mention_or_id>`" + `
**Common Staff Commands:**
• ` + "`.tickets claim`" + ` / ` + "`.tickets unclaim`" + ` - Claim/unclaim a ticket
• ` + "`.tickets close [channel] [reason]`" + ` - Close a ticket
• ` + "`.tickets reopen`" + ` - Reopen a closed ticket
• ` + "`.tickets delete`" + ` - Permanently delete a ticket channel
*Tips: Use '.tickets profiles' or '.tickets panels' to check current configurations.*`,
			})
			return ctx.Respond(emb)
		}
		sub := strings.ToLower(ctx.Args[0])
		db := ctx.DB
		switch sub {
		case "profiles":
			return handleProfiles(ctx, db)
		case "panels":
			return handlePanels(ctx, db)
		case "options":
			return handleOptions(ctx, db)
		case "forms":
			return handleForms(ctx, db)
		case "blacklist":
			return handleBlacklist(ctx, db)
		case "list":
			return handleList(ctx, db)
		case "close":
			return handleClose(ctx, db)
		case "reopen":
			return handleReopen(ctx, db)
		case "claim":
			return handleClaim(ctx, db)
		case "unclaim":
			return handleUnclaim(ctx, db)
		case "delete":
			return handleDelete(ctx, db)
		case "rename":
			return handleRename(ctx, db)
		case "move":
			return handleMove(ctx, db)
		case "allow":
			return handleAllow(ctx, db)
		case "deny":
			return handleDeny(ctx, db)
		case "transcript":
			return handleTranscript(ctx, db)
		case "resend":
			return handleResend(ctx, db)
		case "trainee":
			return handleTrainee(ctx, db)
		case "profile":
			return handleProfile(ctx, db)
		case "stats":
			return handleStats(ctx, db)
		case "reason":
			return handleReason(ctx, db)
		}
		return ctx.Reply("[!] Unknown ticket subcommand. Use `.tickets` to view commands.")
	},
}
func handleProfiles(ctx *manager.CommandContext, db *storage.DB) error {
	p, _ := db.GetTicketProfile(ctx.Message.GuildID)
	if len(ctx.Args) < 2 {
		var roles []string
		for _, r := range p.SupportRoles {
			roles = append(roles, fmt.Sprintf("<@&%s>", r))
		}
		desc := fmt.Sprintf("**Support Roles:** %s\n**Trainees Count:** %d\n**Blacklisted Count:** %d",
			strings.Join(roles, ", "), len(p.Trainees), len(p.Blacklist))
		emb := config.Build(ctx.Cfg, config.EmbedOpt{
			Title:       "Guild Ticket Profile Settings",
			Description: desc,
		})
		return ctx.Respond(emb)
	}
	sub := strings.ToLower(ctx.Args[1])
	if sub == "setup" || sub == "add" {
		if len(ctx.Args) < 3 {
			return ctx.Reply("Usage: `.tickets profiles add <role_mention_or_id>`")
		}
		rid := cleanID(ctx.Args[2])
		p.SupportRoles = append(p.SupportRoles, rid)
		_ = db.SaveTicketProfile(p)
		return ctx.Reply(fmt.Sprintf("[+] Added support role <@&%s>.", rid))
	} else if sub == "remove" {
		if len(ctx.Args) < 3 {
			return ctx.Reply("Usage: `.tickets profiles remove <role_mention_or_id>`")
		}
		rid := cleanID(ctx.Args[2])
		var filtered []string
		for _, r := range p.SupportRoles {
			if r != rid {
				filtered = append(filtered, r)
			}
		}
		p.SupportRoles = filtered
		_ = db.SaveTicketProfile(p)
		return ctx.Reply(fmt.Sprintf("[+] Removed support role <@&%s>.", rid))
	}
	return ctx.Reply("[!] Usage: `.tickets profiles [add/remove] [role]`")
}
func handlePanels(ctx *manager.CommandContext, db *storage.DB) error {
	if len(ctx.Args) < 2 {
		list, _ := db.ListTicketPanels(ctx.Message.GuildID)
		var sb strings.Builder
		for _, p := range list {
			sb.WriteString(fmt.Sprintf("- **%s** (Channel: <#%s>)\n", p.Name, p.ChannelID))
		}
		if len(list) == 0 {
			sb.WriteString("No panels configured.")
		}
		emb := config.Build(ctx.Cfg, config.EmbedOpt{
			Title:       "Ticket Panels",
			Description: sb.String(),
		})
		return ctx.Respond(emb)
	}
	act := strings.ToLower(ctx.Args[1])
	if act == "create" {
		if len(ctx.Args) < 4 {
			return ctx.Reply("Usage: `.tickets panels create <name> <channel_mention_or_id>`")
		}
		name := ctx.Args[2]
		cid := cleanID(ctx.Args[3])
		p := storage.TicketPanel{
			GuildID:     ctx.Message.GuildID,
			Name:        name,
			ChannelID:   cid,
			Title:       "Support Ticket",
			Description: "Click the reaction or button to open a ticket.",
		}
		_ = db.SaveTicketPanel(p)
		return ctx.Reply(fmt.Sprintf("[+] Panel **%s** created in <#%s>.", name, cid))
	} else if act == "delete" {
		if len(ctx.Args) < 3 {
			return ctx.Reply("Usage: `.tickets panels delete <name>`")
		}
		name := ctx.Args[2]
		_ = db.DeleteTicketPanel(ctx.Message.GuildID, name)
		return ctx.Reply(fmt.Sprintf("[+] Panel **%s** deleted.", name))
	} else if act == "set" {
		if len(ctx.Args) < 5 {
			return ctx.Reply("Usage: `.tickets panels set <name> <title/description> <value>`")
		}
		name := ctx.Args[2]
		field := strings.ToLower(ctx.Args[3])
		val := strings.Join(ctx.Args[4:], " ")
		p, err := db.GetTicketPanel(ctx.Message.GuildID, name)
		if err != nil {
			return ctx.Reply("[!] Panel not found.")
		}
		if field == "title" {
			p.Title = val
		} else if field == "description" {
			p.Description = val
		} else {
			return ctx.Reply("[!] Invalid field (use 'title' or 'description').")
		}
		_ = db.SaveTicketPanel(p)
		return ctx.Reply(fmt.Sprintf("[+] Panel **%s** %s updated.", name, field))
	}
	return ctx.Reply("[!] Usage: `.tickets panels [create/delete/set/list]`")
}
func handleOptions(ctx *manager.CommandContext, db *storage.DB) error {
	if len(ctx.Args) < 2 {
		list, _ := db.ListTicketOptions(ctx.Message.GuildID)
		var sb strings.Builder
		for _, o := range list {
			sb.WriteString(fmt.Sprintf("- **%s** (%s) - Category: %s\n", o.Name, o.Emoji, o.CategoryID))
		}
		if len(list) == 0 {
			sb.WriteString("No options configured.")
		}
		emb := config.Build(ctx.Cfg, config.EmbedOpt{
			Title:       "Ticket Panel Options",
			Description: sb.String(),
		})
		return ctx.Respond(emb)
	}
	act := strings.ToLower(ctx.Args[1])
	if act == "create" {
		if len(ctx.Args) < 6 {
			return ctx.Reply("Usage: `.tickets options create <name> <emoji> <description> <category_id>`")
		}
		name := ctx.Args[2]
		emoji := ctx.Args[3]
		desc := ctx.Args[4]
		catID := ctx.Args[5]
		o := storage.TicketOption{
			GuildID:     ctx.Message.GuildID,
			Name:        name,
			Emoji:       emoji,
			Description: desc,
			CategoryID:  catID,
		}
		_ = db.SaveTicketOption(o)
		return ctx.Reply(fmt.Sprintf("[+] Option **%s** created.", name))
	} else if act == "delete" {
		if len(ctx.Args) < 3 {
			return ctx.Reply("Usage: `.tickets options delete <name>`")
		}
		name := ctx.Args[2]
		_ = db.DeleteTicketOption(ctx.Message.GuildID, name)
		return ctx.Reply(fmt.Sprintf("[+] Option **%s** deleted.", name))
	} else if act == "link" {
		if len(ctx.Args) < 4 {
			return ctx.Reply("Usage: `.tickets options link <panel_name> <option_name>`")
		}
		pName := ctx.Args[2]
		oName := ctx.Args[3]
		p, err1 := db.GetTicketPanel(ctx.Message.GuildID, pName)
		_, err2 := db.GetTicketOption(ctx.Message.GuildID, oName)
		if err1 != nil || err2 != nil {
			return ctx.Reply("[!] Panel or Option not found.")
		}
		p.Options = append(p.Options, oName)
		_ = db.SaveTicketPanel(p)
		return ctx.Reply(fmt.Sprintf("[+] Option **%s** linked to Panel **%s**.", oName, pName))
	}
	return ctx.Reply("[!] Usage: `.tickets options [create/delete/link]`")
}
func handleForms(ctx *manager.CommandContext, db *storage.DB) error {
	p, _ := db.GetTicketProfile(ctx.Message.GuildID)
	if len(ctx.Args) < 2 {
		var sb strings.Builder
		for _, f := range p.Forms {
			sb.WriteString(fmt.Sprintf("- %s\n", f))
		}
		if len(p.Forms) == 0 {
			sb.WriteString("No forms configured.")
		}
		emb := config.Build(ctx.Cfg, config.EmbedOpt{
			Title:       "Reusable Ticket Forms",
			Description: sb.String(),
		})
		return ctx.Respond(emb)
	}
	act := strings.ToLower(ctx.Args[1])
	if act == "create" || act == "add" {
		if len(ctx.Args) < 3 {
			return ctx.Reply("Usage: `.tickets forms add <name>`")
		}
		name := ctx.Args[2]
		p.Forms = append(p.Forms, name)
		_ = db.SaveTicketProfile(p)
		return ctx.Reply(fmt.Sprintf("[+] Form **%s** added.", name))
	} else if act == "delete" || act == "remove" {
		if len(ctx.Args) < 3 {
			return ctx.Reply("Usage: `.tickets forms remove <name>`")
		}
		name := ctx.Args[2]
		var filtered []string
		for _, f := range p.Forms {
			if f != name {
				filtered = append(filtered, f)
			}
		}
		p.Forms = filtered
		_ = db.SaveTicketProfile(p)
		return ctx.Reply(fmt.Sprintf("[+] Form **%s** removed.", name))
	}
	return ctx.Reply("[!] Usage: `.tickets forms [add/remove] <name>`")
}
func handleBlacklist(ctx *manager.CommandContext, db *storage.DB) error {
	if len(ctx.Args) < 2 {
		return ctx.Reply("Usage: `.tickets blacklist <member_or_role_mention_or_id>`")
	}
	target := cleanID(ctx.Args[1])
	p, _ := db.GetTicketProfile(ctx.Message.GuildID)
	found := false
	var filtered []string
	for _, id := range p.Blacklist {
		if id == target {
			found = true
		} else {
			filtered = append(filtered, id)
		}
	}
	if found {
		p.Blacklist = filtered
		_ = db.SaveTicketProfile(p)
		return ctx.Reply(fmt.Sprintf("[+] Removed <@%s> / <@&%s> from the ticket blacklist.", target, target))
	}
	p.Blacklist = append(p.Blacklist, target)
	_ = db.SaveTicketProfile(p)
	return ctx.Reply(fmt.Sprintf("[+] Added <@%s> / <@&%s> to the ticket blacklist.", target, target))
}
func handleList(ctx *manager.CommandContext, db *storage.DB) error {
	list, _ := db.ListTicketChannels(ctx.Message.GuildID)
	var sb strings.Builder
	count := 0
	for _, tc := range list {
		if tc.Open {
			claimer := "Unclaimed"
			if tc.ClaimerID != "" {
				claimer = fmt.Sprintf("<@%s>", tc.ClaimerID)
			}
			sb.WriteString(fmt.Sprintf("- <#%s> (Opened by <@%s> | Claimed: %s)\n", tc.ChannelID, tc.CreatorID, claimer))
			count++
		}
	}
	if count == 0 {
		sb.WriteString("No open tickets found.")
	}
	emb := config.Build(ctx.Cfg, config.EmbedOpt{
		Title:       fmt.Sprintf("Open Tickets (%d)", count),
		Description: sb.String(),
	})
	return ctx.Respond(emb)
}
func handleClose(ctx *manager.CommandContext, db *storage.DB) error {
	cid := ctx.Message.ChannelID
	reason := "No reason given"
	if len(ctx.Args) > 1 {
		cid = cleanID(ctx.Args[1])
		if len(ctx.Args) > 2 {
			reason = strings.Join(ctx.Args[2:], " ")
		}
	}
	tc, err := db.GetTicketChannel(cid)
	if err != nil || tc.ChannelID == "" {
		return ctx.Reply("[!] Not inside a valid ticket channel.")
	}
	if !tc.Open {
		return ctx.Reply("[!] Ticket is already closed.")
	}
	tc.Open = false
	tc.Reason = reason
	_ = db.SaveTicketChannel(tc)
	stats, _ := db.GetTicketStats(ctx.Message.GuildID)
	stats.TotalClosed++
	_ = db.SaveTicketStats(stats)
	if tc.ClaimerID != "" {
		staff, _ := db.GetTicketStaffProfile(ctx.Message.GuildID, tc.ClaimerID)
		staff.CloseCount++
		_ = db.SaveTicketStaffProfile(staff)
	}
	_ = ctx.Session.ChannelPermissionSet(cid, tc.CreatorID, discordgo.PermissionOverwriteTypeMember, 0, discordgo.PermissionViewChannel|discordgo.PermissionSendMessages)
	_ = db.SaveTicketReason(storage.TicketReason{
		GuildID:   ctx.Message.GuildID,
		Action:    "close",
		TargetID:  cid,
		StaffID:   ctx.Message.Author.ID,
		Reason:    reason,
		Timestamp: time.Now().Unix(),
	})
	return ctx.Reply(fmt.Sprintf("[+] Closed ticket. Reason: **%s**", reason))
}
func handleReopen(ctx *manager.CommandContext, db *storage.DB) error {
	cid := ctx.Message.ChannelID
	reason := "No reason given"
	if len(ctx.Args) > 1 {
		cid = cleanID(ctx.Args[1])
		if len(ctx.Args) > 2 {
			reason = strings.Join(ctx.Args[2:], " ")
		}
	}
	tc, err := db.GetTicketChannel(cid)
	if err != nil || tc.ChannelID == "" {
		return ctx.Reply("[!] Ticket channel not found.")
	}
	if tc.Open {
		return ctx.Reply("[!] Ticket is already open.")
	}
	tc.Open = true
	_ = db.SaveTicketChannel(tc)
	if tc.ClaimerID != "" {
		staff, _ := db.GetTicketStaffProfile(ctx.Message.GuildID, tc.ClaimerID)
		staff.ReopenCount++
		_ = db.SaveTicketStaffProfile(staff)
	}
	_ = ctx.Session.ChannelPermissionSet(cid, tc.CreatorID, discordgo.PermissionOverwriteTypeMember, discordgo.PermissionViewChannel|discordgo.PermissionSendMessages, 0)
	_ = db.SaveTicketReason(storage.TicketReason{
		GuildID:   ctx.Message.GuildID,
		Action:    "reopen",
		TargetID:  cid,
		StaffID:   ctx.Message.Author.ID,
		Reason:    reason,
		Timestamp: time.Now().Unix(),
	})
	return ctx.Reply(fmt.Sprintf("[+] Ticket reopened. Reason: **%s**", reason))
}
func handleClaim(ctx *manager.CommandContext, db *storage.DB) error {
	cid := ctx.Message.ChannelID
	reason := "No reason given"
	if len(ctx.Args) > 1 {
		cid = cleanID(ctx.Args[1])
		if len(ctx.Args) > 2 {
			reason = strings.Join(ctx.Args[2:], " ")
		}
	}
	tc, err := db.GetTicketChannel(cid)
	if err != nil || tc.ChannelID == "" {
		return ctx.Reply("[!] Ticket channel not found.")
	}
	if tc.ClaimerID != "" {
		return ctx.Reply(fmt.Sprintf("[!] Ticket is already claimed by <@%s>.", tc.ClaimerID))
	}
	tc.ClaimerID = ctx.Message.Author.ID
	_ = db.SaveTicketChannel(tc)
	stats, _ := db.GetTicketStats(ctx.Message.GuildID)
	stats.TotalClaimed++
	_ = db.SaveTicketStats(stats)
	staff, _ := db.GetTicketStaffProfile(ctx.Message.GuildID, ctx.Message.Author.ID)
	staff.ClaimCount++
	_ = db.SaveTicketStaffProfile(staff)
	_ = db.SaveTicketReason(storage.TicketReason{
		GuildID:   ctx.Message.GuildID,
		Action:    "claim",
		TargetID:  cid,
		StaffID:   ctx.Message.Author.ID,
		Reason:    reason,
		Timestamp: time.Now().Unix(),
	})
	return ctx.Reply(fmt.Sprintf("[+] Claimed ticket. Support contact: <@%s>", ctx.Message.Author.ID))
}
func handleUnclaim(ctx *manager.CommandContext, db *storage.DB) error {
	cid := ctx.Message.ChannelID
	if len(ctx.Args) > 1 {
		cid = cleanID(ctx.Args[1])
	}
	tc, err := db.GetTicketChannel(cid)
	if err != nil || tc.ChannelID == "" {
		return ctx.Reply("[!] Ticket channel not found.")
	}
	if tc.ClaimerID == "" {
		return ctx.Reply("[!] Ticket is not currently claimed.")
	}
	tc.ClaimerID = ""
	_ = db.SaveTicketChannel(tc)
	return ctx.Reply("[+] Ticket unclaimed.")
}
func handleDelete(ctx *manager.CommandContext, db *storage.DB) error {
	cid := ctx.Message.ChannelID
	reason := "No reason given"
	if len(ctx.Args) > 1 {
		cid = cleanID(ctx.Args[1])
		if len(ctx.Args) > 2 {
			reason = strings.Join(ctx.Args[2:], " ")
		}
	}
	tc, err := db.GetTicketChannel(cid)
	if err != nil || tc.ChannelID == "" {
		return ctx.Reply("[!] Ticket channel not found.")
	}
	_ = db.DeleteTicketChannel(cid)
	_ = db.SaveTicketReason(storage.TicketReason{
		GuildID:   ctx.Message.GuildID,
		Action:    "delete",
		TargetID:  cid,
		StaffID:   ctx.Message.Author.ID,
		Reason:    reason,
		Timestamp: time.Now().Unix(),
	})
	_, _ = ctx.Session.ChannelDelete(cid)
	return nil
}
func handleRename(ctx *manager.CommandContext, db *storage.DB) error {
	if len(ctx.Args) < 3 {
		return ctx.Reply("Usage: `.tickets rename <channel_mention_or_id> <new_name>`")
	}
	cid := cleanID(ctx.Args[1])
	name := ctx.Args[2]
	tc, err := db.GetTicketChannel(cid)
	if err != nil || tc.ChannelID == "" {
		return ctx.Reply("[!] Channel is not a valid ticket channel.")
	}
	_, err = ctx.Session.ChannelEdit(cid, &discordgo.ChannelEdit{
		Name: name,
	})
	if err != nil {
		return ctx.Reply(fmt.Sprintf("[!] Failed to rename channel: %v", err))
	}
	return ctx.Reply(fmt.Sprintf("[+] Renamed channel to **%s**.", name))
}
func handleMove(ctx *manager.CommandContext, db *storage.DB) error {
	if len(ctx.Args) < 3 {
		return ctx.Reply("Usage: `.tickets move <channel_mention_or_id> <option_name_or_category_id>`")
	}
	cid := cleanID(ctx.Args[1])
	target := ctx.Args[2]
	tc, err := db.GetTicketChannel(cid)
	if err != nil || tc.ChannelID == "" {
		return ctx.Reply("[!] Not a valid ticket channel.")
	}
	opt, err := db.GetTicketOption(ctx.Message.GuildID, target)
	catID := target
	if err == nil && opt.CategoryID != "" {
		catID = opt.CategoryID
		tc.Option = opt.Name
		_ = db.SaveTicketChannel(tc)
	}
	_, err = ctx.Session.ChannelEdit(cid, &discordgo.ChannelEdit{
		ParentID: catID,
	})
	if err != nil {
		return ctx.Reply(fmt.Sprintf("[!] Failed to move channel: %v", err))
	}
	return ctx.Reply(fmt.Sprintf("[+] Moved ticket to category/option **%s**.", target))
}
func handleAllow(ctx *manager.CommandContext, db *storage.DB) error {
	cid := ctx.Message.ChannelID
	tc, err := db.GetTicketChannel(cid)
	if err != nil || tc.ChannelID == "" {
		return ctx.Reply("[!] Not inside a valid ticket channel.")
	}
	if len(ctx.Args) < 2 {
		var allowedList []string
		for _, id := range tc.Allowed {
			allowedList = append(allowedList, fmt.Sprintf("<@%s>", id))
		}
		if len(allowedList) == 0 {
			return ctx.Reply("No explicit members/roles are currently allowed.")
		}
		return ctx.Reply(fmt.Sprintf("Allowed users/roles: %s", strings.Join(allowedList, ", ")))
	}
	target := cleanID(ctx.Args[1])
	tc.Allowed = append(tc.Allowed, target)
	_ = db.SaveTicketChannel(tc)
	_ = ctx.Session.ChannelPermissionSet(cid, target, discordgo.PermissionOverwriteTypeMember, discordgo.PermissionViewChannel|discordgo.PermissionSendMessages, 0)
	_ = ctx.Session.ChannelPermissionSet(cid, target, discordgo.PermissionOverwriteTypeRole, discordgo.PermissionViewChannel|discordgo.PermissionSendMessages, 0)
	return ctx.Reply(fmt.Sprintf("[+] Allowed <@%s> / <@&%s> to view the ticket.", target, target))
}
func handleDeny(ctx *manager.CommandContext, db *storage.DB) error {
	cid := ctx.Message.ChannelID
	tc, err := db.GetTicketChannel(cid)
	if err != nil || tc.ChannelID == "" {
		return ctx.Reply("[!] Not inside a valid ticket channel.")
	}
	if len(ctx.Args) < 2 {
		return ctx.Reply("Usage: `.tickets deny <member_or_role>`")
	}
	target := cleanID(ctx.Args[1])
	tc.Denied = append(tc.Denied, target)
	var filtered []string
	for _, id := range tc.Allowed {
		if id != target {
			filtered = append(filtered, id)
		}
	}
	tc.Allowed = filtered
	_ = db.SaveTicketChannel(tc)
	_ = ctx.Session.ChannelPermissionSet(cid, target, discordgo.PermissionOverwriteTypeMember, 0, discordgo.PermissionViewChannel|discordgo.PermissionSendMessages)
	_ = ctx.Session.ChannelPermissionSet(cid, target, discordgo.PermissionOverwriteTypeRole, 0, discordgo.PermissionViewChannel|discordgo.PermissionSendMessages)
	return ctx.Reply(fmt.Sprintf("[+] Denied <@%s> / <@&%s> from viewing the ticket.", target, target))
}
func handleTranscript(ctx *manager.CommandContext, db *storage.DB) error {
	cid := ctx.Message.ChannelID
	reason := "No reason given"
	if len(ctx.Args) > 1 {
		cid = cleanID(ctx.Args[1])
		if len(ctx.Args) > 2 {
			reason = strings.Join(ctx.Args[2:], " ")
		}
	}
	tc, err := db.GetTicketChannel(cid)
	if err != nil || tc.ChannelID == "" {
		return ctx.Reply("[!] Not inside a valid ticket channel.")
	}
	msgs, err := ctx.Session.ChannelMessages(cid, 100, "", "", "")
	if err != nil {
		return ctx.Reply(fmt.Sprintf("[!] Failed to fetch messages: %v", err))
	}
	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("Transcript for ticket: %s\nGenerated at: %s\nReason: %s\n\n", cid, time.Now().Format(time.RFC1123), reason))
	for i := len(msgs) - 1; i >= 0; i-- {
		m := msgs[i]
		buf.WriteString(fmt.Sprintf("[%s] %s#%s (%s): %s\n", m.Timestamp.Format("2006-01-02 15:04:05"), m.Author.Username, m.Author.Discriminator, m.Author.ID, m.Content))
	}
	_, err = ctx.Session.ChannelMessageSendComplex(ctx.Message.ChannelID, &discordgo.MessageSend{
		Content: "[+] Transcript generated:",
		Files: []*discordgo.File{
			{
				Name:   fmt.Sprintf("transcript-%s.txt", cid),
				Reader: bytes.NewReader(buf.Bytes()),
			},
		},
	})
	if err != nil {
		return ctx.Reply(fmt.Sprintf("[!] Failed to send file: %v", err))
	}
	return nil
}
func handleResend(ctx *manager.CommandContext, db *storage.DB) error {
	if len(ctx.Args) < 2 {
		return ctx.Reply("Usage: `.tickets resend <channel_mention_or_id>`")
	}
	cid := cleanID(ctx.Args[1])
	list, _ := db.ListTicketPanels(ctx.Message.GuildID)
	var p *storage.TicketPanel
	for i := range list {
		if list[i].ChannelID == cid {
			p = &list[i]
			break
		}
	}
	if p == nil {
		return ctx.Reply("[!] No panel config found for that channel.")
	}
	emb := config.Build(ctx.Cfg, config.EmbedOpt{
		Title:       p.Title,
		Description: p.Description,
	})
	msg, err := ctx.Session.ChannelMessageSendEmbed(cid, emb)
	if err != nil {
		return ctx.Reply(fmt.Sprintf("[!] Failed to send: %v", err))
	}
	p.MessageID = msg.ID
	_ = db.SaveTicketPanel(*p)
	return ctx.Reply("[+] Panel message resent.")
}
func handleTrainee(ctx *manager.CommandContext, db *storage.DB) error {
	p, _ := db.GetTicketProfile(ctx.Message.GuildID)
	if len(ctx.Args) < 2 {
		var list []string
		for _, id := range p.Trainees {
			list = append(list, fmt.Sprintf("<@%s> / <@&%s>", id, id))
		}
		if len(list) == 0 {
			return ctx.Reply("No trainee overrides found.")
		}
		return ctx.Reply(fmt.Sprintf("Trainees: %s", strings.Join(list, ", ")))
	}
	sub := strings.ToLower(ctx.Args[1])
	if sub == "grant" {
		if len(ctx.Args) < 3 {
			return ctx.Reply("Usage: `.tickets trainee grant <member_or_role>`")
		}
		target := cleanID(ctx.Args[2])
		p.Trainees = append(p.Trainees, target)
		_ = db.SaveTicketProfile(p)
		return ctx.Reply(fmt.Sprintf("[+] Granted trainee status to <@%s> / <@&%s>.", target, target))
	} else if sub == "revoke" {
		if len(ctx.Args) < 3 {
			return ctx.Reply("Usage: `.tickets trainee revoke <member_or_role>`")
		}
		target := cleanID(ctx.Args[2])
		var filtered []string
		for _, id := range p.Trainees {
			if id != target {
				filtered = append(filtered, id)
			}
		}
		p.Trainees = filtered
		_ = db.SaveTicketProfile(p)
		return ctx.Reply(fmt.Sprintf("[+] Revoked trainee status for <@%s> / <@&%s>.", target, target))
	} else if sub == "list" {
		var list []string
		for _, id := range p.Trainees {
			list = append(list, fmt.Sprintf("<@%s>", id))
		}
		if len(list) == 0 {
			return ctx.Reply("No trainees configured.")
		}
		return ctx.Reply(fmt.Sprintf("Active Trainees: %s", strings.Join(list, ", ")))
	}
	return ctx.Reply("[!] Usage: `.tickets trainee [grant/revoke/list]`")
}
func handleProfile(ctx *manager.CommandContext, db *storage.DB) error {
	staff, _ := db.GetTicketStaffProfile(ctx.Message.GuildID, ctx.Message.Author.ID)
	desc := fmt.Sprintf("**Claims:** %d\n**Closes:** %d\n**Reopens:** %d", staff.ClaimCount, staff.CloseCount, staff.ReopenCount)
	emb := config.Build(ctx.Cfg, config.EmbedOpt{
		Title:       fmt.Sprintf("%s's Ticket Claim Profile", ctx.Message.Author.Username),
		Description: desc,
	})
	return ctx.Respond(emb)
}
func handleStats(ctx *manager.CommandContext, db *storage.DB) error {
	target := ctx.Message.Author.ID
	if len(ctx.Args) > 1 {
		target = cleanID(ctx.Args[1])
	}
	staff, _ := db.GetTicketStaffProfile(ctx.Message.GuildID, target)
	desc := fmt.Sprintf("**Total Claims:** %d\n**Total Closes:** %d\n**Total Reopens:** %d", staff.ClaimCount, staff.CloseCount, staff.ReopenCount)
	emb := config.Build(ctx.Cfg, config.EmbedOpt{
		Title:       "Ticket Stats",
		Description: desc,
	})
	return ctx.Respond(emb)
}
func handleReason(ctx *manager.CommandContext, db *storage.DB) error {
	if len(ctx.Args) < 4 {
		return ctx.Reply("Usage: `.tickets reason <action> <target_id> <reason_text>`")
	}
	action := strings.ToLower(ctx.Args[1])
	target := cleanID(ctx.Args[2])
	reason := strings.Join(ctx.Args[3:], " ")
	r := storage.TicketReason{
		GuildID:   ctx.Message.GuildID,
		Action:    action,
		TargetID:  target,
		StaffID:   ctx.Message.Author.ID,
		Reason:    reason,
		Timestamp: time.Now().Unix(),
	}
	_ = db.SaveTicketReason(r)
	return ctx.Reply(fmt.Sprintf("[+] Stored reason for action **%s** on %s.", action, target))
}
func cleanID(s string) string {
	s = strings.TrimPrefix(s, "<@!")
	s = strings.TrimPrefix(s, "<@")
	s = strings.TrimPrefix(s, "<@&")
	s = strings.TrimPrefix(s, "<#")
	s = strings.TrimSuffix(s, ">")
	return s
}