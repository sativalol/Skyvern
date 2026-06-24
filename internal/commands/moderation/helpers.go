package moderation

import (
	"fmt"
	"skyvern/internal/manager"
	"skyvern/internal/moderation"
	"skyvern/internal/storage"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

type ModAction struct {
	Ctx         *manager.CommandContext
	Perm        int64
	MinArgs     int
	Usage       string
	CheckHier   bool
	DMAction    string
	CaseType    string
	LogName     string
	ExtraFields []*discordgo.MessageEmbedField
	ActionFn    func(targetID string, reason string) error
	SuccessMsg  func(username string, caseID int, reason string) string
}

func runModAction(a ModAction) error {
	a.Ctx.Cfg.EmbedColor = 0x808080
	if !checkPerm(a.Ctx, a.Perm) {
		return a.Ctx.Reply("[!] You do not have permission to use this command.")
	}
	if len(a.Ctx.Args) < a.MinArgs {
		return a.Ctx.Reply("Usage: " + a.Usage)
	}

	gid := a.Ctx.GuildID()
	targetQuery := a.Ctx.Args[0]

	var targetID string
	var targetName string
	if a.CheckHier {
		m, err := moderation.ResolveMember(a.Ctx.Session, gid, targetQuery)
		if err != nil || m == nil {
			return a.Ctx.Reply("[!] Could not resolve member.")
		}
		if !checkHierarchy(a.Ctx, m.User.ID) {
			return a.Ctx.Reply("[!] You cannot moderate this member due to role hierarchy.")
		}
		targetID = m.User.ID
		targetName = m.User.Username
	} else {
		targetID = targetQuery
		targetName = targetQuery
	}

	reason := "No reason provided."
	if len(a.Ctx.Args) > a.MinArgs {
		reason = strings.Join(a.Ctx.Args[a.MinArgs:], " ")
	}

	if a.DMAction != "" && a.CheckHier {
		moderation.DMUserAction(a.Ctx.Session, gid, a.DMAction, targetID, a.Ctx.AuthorID(), reason)
	}

	if err := a.ActionFn(targetID, reason); err != nil {
		return a.Ctx.Reply(fmt.Sprintf("[!] Failed: %v", err))
	}

	var caseID int
	if a.CaseType != "" {
		c := storage.Case{
			UserID:    targetID,
			ModID:     a.Ctx.AuthorID(),
			Type:      a.CaseType,
			Reason:    reason,
			Timestamp: time.Now(),
		}
		caseID, _ = a.Ctx.DB.AddCase(gid, c)
	}

	if a.LogName != "" {
		logText := a.LogName
		if a.CaseType != "" {
			logText = fmt.Sprintf("%s (Case #%d)", a.LogName, caseID)
		}
		moderation.LogAction(a.Ctx.Session, a.Ctx.DB, gid, logText, a.Ctx.AuthorID(), targetID, reason, a.ExtraFields...)
	}

	return a.Ctx.Reply(a.SuccessMsg(targetName, caseID, reason))
}

func checkPerm(ctx *manager.CommandContext, perm int64) bool {
	uid := ctx.AuthorID()
	if uid == "" {
		return false
	}
	if isOwnerOrBypassed(ctx) {
		return true
	}
	p, err := ctx.UserChannelPermissions(uid, "")
	if err != nil {
		return false
	}
	if (p & discordgo.PermissionAdministrator) != 0 {
		return true
	}
	return (p & perm) == perm
}

func isOwner(ctx *manager.CommandContext) bool {
	uid := ctx.AuthorID()
	gid := ctx.GuildID()
	g, err := ctx.Session.State.Guild(gid)
	if err != nil {
		g, err = ctx.Session.Guild(gid)
	}
	return err == nil && g.OwnerID == uid
}

func isOwnerOrBypassed(ctx *manager.CommandContext) bool {
	if isOwner(ctx) {
		return true
	}
	return ctx.DB.HasBypass(ctx.GuildID(), ctx.AuthorID())
}

func checkHierarchy(ctx *manager.CommandContext, tid string) bool {
	return ctx.CheckHierarchy(tid)
}

func resolveMemberOrReply(ctx *manager.CommandContext, query string) (*discordgo.Member, error) {
	m, err := moderation.ResolveMember(ctx.Session, ctx.GuildID(), query)
	if err != nil || m == nil {
		_ = ctx.Reply("[!] Could not resolve member.")
		return nil, fmt.Errorf("could not resolve member")
	}
	return m, nil
}

func resolveRoleOrReply(ctx *manager.CommandContext, query string) (string, error) {
	rid := resolveRole(ctx.Session, ctx.GuildID(), query)
	if rid == "" {
		_ = ctx.Reply("[!] Could not resolve role.")
		return "", fmt.Errorf("could not resolve role")
	}
	return rid, nil
}

func resolveChannelOrReply(ctx *manager.CommandContext, query string) (string, error) {
	cid := strings.Trim(query, "<#>")
	ch, err := ctx.Session.Channel(cid)
	if err != nil || ch.GuildID != ctx.GuildID() {
		_ = ctx.Reply("[!] Invalid channel.")
		return "", fmt.Errorf("invalid channel")
	}
	return cid, nil
}
