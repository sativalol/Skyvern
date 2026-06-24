package manager
import (
	"fmt"
	"strings"
	"time"
	"github.com/bwmarrin/discordgo"
	"skyvern/internal/config"
	"skyvern/internal/storage"
)
type CommandContext struct {
	Session  *discordgo.Session
	Message  *discordgo.Message
	Interact *discordgo.Interaction
	Args     []string
	Cfg      config.ResCfg
	DB       *storage.DB
	ClientID string
	Mgr      *Manager
}
type Command struct {
	Trigger     string
	Aliases     []string
	Name        string
	Description string
	Category    string
	Execute     func(ctx *CommandContext) error
}
func (ctx *CommandContext) Respond(embed *discordgo.MessageEmbed) error {
	if ctx.Interact != nil {
		return ctx.Session.InteractionRespond(ctx.Interact, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{embed}},
		})
	}
	_, err := ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
	return err
}
func (ctx *CommandContext) RespondAndGet(embed *discordgo.MessageEmbed) (*discordgo.Message, error) {
	if ctx.Interact != nil {
		err := ctx.Session.InteractionRespond(ctx.Interact, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{embed}},
		})
		if err != nil {
			return nil, err
		}
		return ctx.Session.InteractionResponse(ctx.Interact)
	}
	return ctx.Session.ChannelMessageSendEmbed(ctx.ChanID(), embed)
}
func (ctx *CommandContext) GuildID() string {
	if ctx.Interact != nil {
		return ctx.Interact.GuildID
	}
	if ctx.Message != nil {
		return ctx.Message.GuildID
	}
	return ""
}
func (ctx *CommandContext) ChanID() string {
	if ctx.Interact != nil {
		return ctx.Interact.ChannelID
	}
	if ctx.Message != nil {
		return ctx.Message.ChannelID
	}
	return ""
}
func (ctx *CommandContext) AuthorID() string {
	if ctx.Interact != nil && ctx.Interact.Member != nil && ctx.Interact.Member.User != nil {
		return ctx.Interact.Member.User.ID
	}
	if ctx.Message != nil && ctx.Message.Author != nil {
		return ctx.Message.Author.ID
	}
	return ""
}
func (ctx *CommandContext) AuthorTag() string {
	if ctx.Interact != nil && ctx.Interact.Member != nil && ctx.Interact.Member.User != nil {
		return ctx.Interact.Member.User.Username
	}
	if ctx.Message != nil && ctx.Message.Author != nil {
		return ctx.Message.Author.Username
	}
	return "Unknown"
}
func (ctx *CommandContext) Reply(text string) error {
	return ctx.Respond(config.Wrap(ctx.Cfg, text))
}
func (ctx *CommandContext) SendText(text string) error {
	if ctx.Interact != nil {
		return ctx.Session.InteractionRespond(ctx.Interact, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: text},
		})
	}
	_, err := ctx.Session.ChannelMessageSend(ctx.ChanID(), text)
	return err
}
func (ctx *CommandContext) ReplyAndGet(text string) (*discordgo.Message, error) {
	emb := config.Wrap(ctx.Cfg, text)
	if ctx.Interact != nil {
		err := ctx.Session.InteractionRespond(ctx.Interact, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{emb}},
		})
		if err != nil {
			return nil, err
		}
		return ctx.Session.InteractionResponse(ctx.Interact)
	}
	return ctx.Session.ChannelMessageSendEmbed(ctx.ChanID(), emb)
}
func (ctx *CommandContext) EditReply(msg *discordgo.Message, text string) error {
	emb := config.Wrap(ctx.Cfg, text)
	if ctx.Interact != nil {
		_, err := ctx.Session.InteractionResponseEdit(ctx.Interact, &discordgo.WebhookEdit{
			Embeds: &[]*discordgo.MessageEmbed{emb},
		})
		return err
	}
	if msg != nil {
		_, err := ctx.Session.ChannelMessageEditEmbed(ctx.ChanID(), msg.ID, emb)
		return err
	}
	return nil
}
func (ctx *CommandContext) EditOrReplyLarge(msg *discordgo.Message, text string, filename ...string) error {
	fname := "output.txt"
	if len(filename) > 0 && filename[0] != "" {
		fname = filename[0]
	}
	if len(text) <= 1900 {
		return ctx.EditReply(msg, text)
	}
	_ = ctx.EditReply(msg, "[*] Response too large, uploading as file...")
	sr := strings.NewReader(text)
	ms := &discordgo.MessageSend{
		Content: "[+] Here is the full AI response:",
		Files: []*discordgo.File{
			{
				Name:   fname,
				Reader: sr,
			},
		},
	}
	if ctx.Interact != nil {
		_, err := ctx.Session.InteractionResponseEdit(ctx.Interact, &discordgo.WebhookEdit{
			Content: &ms.Content,
			Files:   ms.Files,
		})
		return err
	}
	_, err := ctx.Session.ChannelMessageSendComplex(ctx.ChanID(), ms)
	return err
}
func (ctx *CommandContext) ReplyLarge(text string, filename ...string) error {
	fname := "output.txt"
	if len(filename) > 0 && filename[0] != "" {
		fname = filename[0]
	}
	if len(text) <= 1900 {
		return ctx.Reply(text)
	}
	if len(text) > 8000 {
		sr := strings.NewReader(text)
		ms := &discordgo.MessageSend{
			Content: "[*] Output too large, sent as file:",
			Files: []*discordgo.File{
				{
					Name:   fname,
					Reader: sr,
				},
			},
		}
		if ctx.Interact != nil {
			return ctx.Session.InteractionRespond(ctx.Interact, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: ms.Content,
					Files:   ms.Files,
				},
			})
		}
		_, err := ctx.Session.ChannelMessageSendComplex(ctx.ChanID(), ms)
		return err
	}
	lines := strings.Split(text, "\n")
	var chunk strings.Builder
	for _, line := range lines {
		if chunk.Len()+len(line)+1 > 1900 {
			if chunk.Len() > 0 {
				err := ctx.Reply(chunk.String())
				if err != nil {
					return err
				}
				chunk.Reset()
			}
			for len(line) > 1900 {
				err := ctx.Reply(line[:1900])
				if err != nil {
					return err
				}
				line = line[1900:]
			}
		}
		if chunk.Len() > 0 {
			chunk.WriteByte('\n')
		}
		chunk.WriteString(line)
	}
	if chunk.Len() > 0 {
		return ctx.Reply(chunk.String())
	}
	return nil
}
func (ctx *CommandContext) Ban(uid, reason string, days int) error {
	auditReason := fmt.Sprintf("Forced by %s (%s) | Reason: %s", ctx.AuthorTag(), ctx.AuthorID(), reason)
	return ctx.Session.GuildBanCreateWithReason(ctx.GuildID(), uid, auditReason, days)
}
func (ctx *CommandContext) Unban(uid string, reason ...string) error {
	r := "Manual unban"
	if len(reason) > 0 && reason[0] != "" {
		r = reason[0]
	}
	auditReason := fmt.Sprintf("Forced by %s (%s) | Reason: %s", ctx.AuthorTag(), ctx.AuthorID(), r)
	return ctx.Session.GuildBanDelete(ctx.GuildID(), uid, discordgo.WithAuditLogReason(auditReason))
}
func (ctx *CommandContext) Kick(uid string, reason ...string) error {
	r := "Manual kick"
	if len(reason) > 0 && reason[0] != "" {
		r = reason[0]
	}
	auditReason := fmt.Sprintf("Forced by %s (%s) | Reason: %s", ctx.AuthorTag(), ctx.AuthorID(), r)
	return ctx.Session.GuildMemberDelete(ctx.GuildID(), uid, discordgo.WithAuditLogReason(auditReason))
}
func (ctx *CommandContext) Timeout(uid string, until *time.Time, reason ...string) error {
	r := "Manual timeout"
	if len(reason) > 0 && reason[0] != "" {
		r = reason[0]
	}
	auditReason := fmt.Sprintf("Forced by %s (%s) | Reason: %s", ctx.AuthorTag(), ctx.AuthorID(), r)
	return ctx.Session.GuildMemberTimeout(ctx.GuildID(), uid, until, discordgo.WithAuditLogReason(auditReason))
}
func (ctx *CommandContext) Nick(uid, nick string, reason ...string) error {
	r := "Manual nickname update"
	if len(reason) > 0 && reason[0] != "" {
		r = reason[0]
	}
	auditReason := fmt.Sprintf("Forced by %s (%s) | Reason: %s", ctx.AuthorTag(), ctx.AuthorID(), r)
	return ctx.Session.GuildMemberNickname(ctx.GuildID(), uid, nick, discordgo.WithAuditLogReason(auditReason))
}
func (ctx *CommandContext) ChannelPermissionSet(chID, targetID string, targetType discordgo.PermissionOverwriteType, allowVal, denyVal int64, reason ...string) error {
	r := "Update channel permissions"
	if len(reason) > 0 && reason[0] != "" {
		r = reason[0]
	}
	auditReason := fmt.Sprintf("Forced by %s (%s) | Reason: %s", ctx.AuthorTag(), ctx.AuthorID(), r)
	return ctx.Session.ChannelPermissionSet(chID, targetID, targetType, allowVal, denyVal, discordgo.WithAuditLogReason(auditReason))
}
func (ctx *CommandContext) ChannelPermissionDelete(chID, targetID string, reason ...string) error {
	r := "Delete channel permissions override"
	if len(reason) > 0 && reason[0] != "" {
		r = reason[0]
	}
	auditReason := fmt.Sprintf("Forced by %s (%s) | Reason: %s", ctx.AuthorTag(), ctx.AuthorID(), r)
	return ctx.Session.ChannelPermissionDelete(chID, targetID, discordgo.WithAuditLogReason(auditReason))
}
func (ctx *CommandContext) Delete(msgID string) error {
	return ctx.Session.ChannelMessageDelete(ctx.ChanID(), msgID)
}
func (ctx *CommandContext) BulkDelete(ids []string) error {
	return ctx.Session.ChannelMessagesBulkDelete(ctx.ChanID(), ids)
}
func (ctx *CommandContext) UserChannelPermissions(userID, channelID string) (int64, error) {
	return UserChannelPermissions(ctx.Session, userID, ctx.GuildID(), channelID)
}
func UserChannelPermissions(s *discordgo.Session, userID, guildID, channelID string) (int64, error) {
	if guildID == "" {
		return 0, fmt.Errorf("guild ID required")
	}
	if p, err := s.State.UserChannelPermissions(userID, channelID); err == nil {
		return p, nil
	}
	g, err := s.State.Guild(guildID)
	if err != nil {
		g, err = s.Guild(guildID)
		if err != nil {
			return 0, err
		}
	}
	if g.OwnerID == userID {
		return discordgo.PermissionAdministrator, nil
	}
	mem, err := s.State.Member(guildID, userID)
	if err != nil {
		mem, err = s.GuildMember(guildID, userID)
		if err != nil {
			return 0, err
		}
	}
	var permissions int64
	var everyoneRole *discordgo.Role
	for _, r := range g.Roles {
		if r.ID == guildID {
			everyoneRole = r
			break
		}
	}
	if everyoneRole != nil {
		permissions = everyoneRole.Permissions
	}
	roleMap := make(map[string]*discordgo.Role)
	for _, r := range g.Roles {
		roleMap[r.ID] = r
	}
	for _, roleID := range mem.Roles {
		if r, ok := roleMap[roleID]; ok {
			permissions |= r.Permissions
		}
	}
	if (permissions & discordgo.PermissionAdministrator) == discordgo.PermissionAdministrator {
		return discordgo.PermissionAdministrator, nil
	}
	if channelID == "" {
		return permissions, nil
	}
	ch, err := s.State.Channel(channelID)
	if err != nil {
		ch, err = s.Channel(channelID)
		if err != nil {
			return permissions, nil
		}
	}
	var allow, deny int64
	for _, o := range ch.PermissionOverwrites {
		if o.Type == discordgo.PermissionOverwriteTypeRole {
			if o.ID == guildID {
				allow |= o.Allow
				deny |= o.Deny
			} else {
				for _, roleID := range mem.Roles {
					if o.ID == roleID {
						allow |= o.Allow
						deny |= o.Deny
						break
					}
				}
			}
		}
	}
	permissions &= ^deny
	permissions |= allow
	for _, o := range ch.PermissionOverwrites {
		if o.Type == discordgo.PermissionOverwriteTypeMember && o.ID == userID {
			permissions &= ^o.Deny
			permissions |= o.Allow
			break
		}
	}
	return permissions, nil
}
func (ctx *CommandContext) SuccessEmoji() string {
	return ctx.Mgr.ResolveEmoji(ctx.Session, ctx.GuildID(), "sys_checkmark")
}
func (ctx *CommandContext) ErrorEmoji() string {
	return ctx.Mgr.ResolveEmoji(ctx.Session, ctx.GuildID(), "sys_x")
}
func (ctx *CommandContext) WarningEmoji() string {
	return ctx.Mgr.ResolveEmoji(ctx.Session, ctx.GuildID(), "sys_warning")
}
func (ctx *CommandContext) LockEmoji() string {
	return ctx.Mgr.ResolveEmoji(ctx.Session, ctx.GuildID(), "sys_lock")
}
func (ctx *CommandContext) UnlockEmoji() string {
	return ctx.Mgr.ResolveEmoji(ctx.Session, ctx.GuildID(), "sys_unlock")
}

func (ctx *CommandContext) CheckHierarchy(tID string) bool {
	gid := ctx.GuildID()
	mid := ctx.AuthorID()
	bid := ctx.Session.State.User.ID

	g, err := ctx.Session.State.Guild(gid)
	if err == nil && g.OwnerID == mid {
		return true
	}

	roles, err := ctx.Session.GuildRoles(gid)
	if err != nil {
		return false
	}
	pos := make(map[string]int)
	for _, r := range roles {
		pos[r.ID] = r.Position
	}

	maxRole := func(uid string) int {
		mem, err := ctx.Session.State.Member(gid, uid)
		if err != nil {
			mem, err = ctx.Session.GuildMember(gid, uid)
			if err != nil {
				return -1
			}
		}
		mMax := -1
		for _, r := range mem.Roles {
			if v, ok := pos[r]; ok && v > mMax {
				mMax = v
			}
		}
		return mMax
	}

	mMax := maxRole(mid)
	tMax := maxRole(tID)
	bMax := maxRole(bid)

	return mMax > tMax && bMax > tMax
}

func (ctx *CommandContext) CanManageRole(rID string) bool {
	gid := ctx.GuildID()
	mid := ctx.AuthorID()

	g, err := ctx.Session.State.Guild(gid)
	if err == nil && g.OwnerID == mid {
		return true
	}

	roles, err := ctx.Session.GuildRoles(gid)
	if err != nil {
		return false
	}
	var targetRole *discordgo.Role
	for _, r := range roles {
		if r.ID == rID {
			targetRole = r
			break
		}
	}
	if targetRole == nil {
		return false
	}

	mem, err := ctx.Session.State.Member(gid, mid)
	if err != nil {
		mem, err = ctx.Session.GuildMember(gid, mid)
		if err != nil {
			return false
		}
	}
	pos := make(map[string]int)
	for _, r := range roles {
		pos[r.ID] = r.Position
	}

	mMax := -1
	for _, r := range mem.Roles {
		if v, ok := pos[r]; ok && v > mMax {
			mMax = v
		}
	}

	return mMax > targetRole.Position
}