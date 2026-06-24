package captcha
import (
	"bytes"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"
	"github.com/bwmarrin/discordgo"
	"github.com/steambap/captcha"
	bolt "go.etcd.io/bbolt"
	"skyvern/internal/manager"
	"skyvern/internal/plugins"
	"skyvern/internal/storage"
)
func init() {
	plugins.Register(&CaptchaPlugin{})
}
type CaptchaPlugin struct {
	db  *storage.DB
	mgr *manager.Manager
}
func (p *CaptchaPlugin) Name() string {
	return "captcha"
}
func (p *CaptchaPlugin) Init(db *storage.DB, mgr *manager.Manager) error {
	p.db = db
	p.mgr = mgr
	if err := db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(bktCaptchaCfg)
		return err
	}); err != nil {
		return err
	}
	mgr.RegisterComponentHandler("captcha_start:*", func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		p.HandleCaptchaStart(s, i)
	})
	mgr.RegisterComponentHandler("captcha_select:*", func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		p.HandleCaptchaSelect(s, i)
	})
	mgr.RegisterJoinHandler(p.HandleMemberJoin)
	return nil
}
func (p *CaptchaPlugin) Commands() []*manager.Command {
	return captchaCommands(p)
}
func (p *CaptchaPlugin) HandleMemberJoin(s *discordgo.Session, e *discordgo.GuildMemberAdd) {
	cfg := getCaptchaCfg(p.db, e.GuildID)
	if !cfg.Enabled {
		return
	}
	if cfg.UnverifiedRoleID != "" {
		_ = s.GuildMemberRoleAdd(e.GuildID, e.Member.User.ID, cfg.UnverifiedRoleID)
	}
	_ = p.SendChallenge(s, e.Member.User.ID, e.GuildID, nil)
}
func (p *CaptchaPlugin) HandleCaptchaStart(s *discordgo.Session, i *discordgo.InteractionCreate) {
	customID := i.MessageComponentData().CustomID
	parts := strings.Split(customID, ":")
	if len(parts) < 2 {
		return
	}
	guildID := parts[1]
	var userID string
	if i.User != nil {
		userID = i.User.ID
	} else if i.Member != nil && i.Member.User != nil {
		userID = i.Member.User.ID
	}
	err := p.SendChallenge(s, userID, guildID, i.Interaction)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[CaptchaPlugin] SendChallenge error: %v\n", err)
	}
}
func (p *CaptchaPlugin) SendChallenge(s *discordgo.Session, userID, guildID string, editInteract *discordgo.Interaction) error {
	cfg := getCaptchaCfg(p.db, guildID)
	if !cfg.Enabled {
		return nil
	}
	if editInteract != nil {
		_ = s.InteractionRespond(editInteract, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Flags: discordgo.MessageFlagsEphemeral,
			},
		})
	}
	img, err := captcha.New(150, 64)
	if err != nil {
		if editInteract != nil {
			errMsg := "❌ Failed to generate captcha challenge."
			_, _ = s.InteractionResponseEdit(editInteract, &discordgo.WebhookEdit{
				Content: &errMsg,
			})
		}
		return err
	}
	setSession(userID, guildID, img.Text, cfg.TimeoutMinutes)
	var pngBuf bytes.Buffer
	if err := img.WriteImage(&pngBuf); err != nil {
		if editInteract != nil {
			errMsg := "❌ Failed to render captcha image."
			_, _ = s.InteractionResponseEdit(editInteract, &discordgo.WebhookEdit{
				Content: &errMsg,
			})
		}
		return err
	}
	decoys := []string{
		genDecoy(img.Text),
		genDecoy(img.Text),
		genDecoy(img.Text),
		genDecoy(img.Text),
	}
	options := append(decoys, img.Text)
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(options), func(i, j int) {
		options[i], options[j] = options[j], options[i]
	})
	var menuOptions []discordgo.SelectMenuOption
	for _, opt := range options {
		menuOptions = append(menuOptions, discordgo.SelectMenuOption{
			Label: opt,
			Value: opt,
		})
	}
	var footer string
	var footerIcon string
	if rc, ok := p.mgr.ResolvedCfgFor(s.State.User.ID); ok {
		footer = rc.Footer
		footerIcon = rc.FooterIcon
	}
	if footer == "" {
		footer = "esoteric.win"
	}
	embed := &discordgo.MessageEmbed{
		Title:       "Server Verification Required",
		Description: fmt.Sprintf("Please select the correct 4-character code shown in the image below from the dropdown menu.\n\n⌛ **Expires:** <t:%d:R>", time.Now().Add(time.Duration(cfg.TimeoutMinutes)*time.Minute).Unix()),
		Color:       0x2b2d31,
		Image: &discordgo.MessageEmbedImage{
			URL: "attachment://captcha.png",
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text:    footer,
			IconURL: footerIcon,
		},
	}
	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.SelectMenu{
					CustomID:    "captcha_select:" + guildID,
					Placeholder: "Select the correct code...",
					Options:     menuOptions,
					MinValues:   intPtr(1),
					MaxValues:   1,
				},
			},
		},
	}
	ch, err := s.UserChannelCreate(userID)
	if err == nil {
		_, err = s.ChannelMessageSendComplex(ch.ID, &discordgo.MessageSend{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: components,
			Files: []*discordgo.File{
				{
					Name:        "captcha.png",
					ContentType: "image/png",
					Reader:      &pngBuf,
				},
			},
		})
	}
	if editInteract != nil {
		var content string
		if err != nil {
			content = "❌ Could not send captcha to your DMs. Please check that your Direct Messages are enabled for this server and try again."
		} else {
			content = "📬 Captcha challenge has been sent to your DMs!"
		}
		_, _ = s.InteractionResponseEdit(editInteract, &discordgo.WebhookEdit{
			Content: &content,
		})
	}
	return err
}
func (p *CaptchaPlugin) HandleCaptchaSelect(s *discordgo.Session, i *discordgo.InteractionCreate) {
	customID := i.MessageComponentData().CustomID
	parts := strings.Split(customID, ":")
	if len(parts) < 2 {
		return
	}
	guildID := parts[1]
	userID := i.User.ID
	if userID == "" && i.Member != nil && i.Member.User != nil {
		userID = i.Member.User.ID
	}
	cfg := getCaptchaCfg(p.db, guildID)
	session, ok := getSession(userID, guildID)
	if !ok {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Content:    "❌ Verification session expired. Please click the button in the server again to get a new captcha.",
				Embeds:     nil,
				Components: nil,
			},
		})
		return
	}
	values := i.MessageComponentData().Values
	if len(values) == 0 {
		return
	}
	selected := values[0]
	if selected == session.Answer {
		deleteSession(userID, guildID)
		if cfg.UnverifiedRoleID != "" {
			_ = s.GuildMemberRoleRemove(guildID, userID, cfg.UnverifiedRoleID)
		}
		if cfg.VerifiedRoleID != "" {
			_ = s.GuildMemberRoleAdd(guildID, userID, cfg.VerifiedRoleID)
		}
		successMsg := "✅ **Verification Successful**\n\nYou have successfully completed the captcha challenge."
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Content:    successMsg,
				Embeds:     nil,
				Components: nil,
			},
		})
	} else {
		newAttempts := session.Attempts + 1
		if newAttempts >= cfg.MaxAttempts {
			deleteSession(userID, guildID)
			_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseUpdateMessage,
				Data: &discordgo.InteractionResponseData{
					Content:    fmt.Sprintf("❌ **Verification Failed**\n\nYou failed the captcha challenge after %d attempts.", cfg.MaxAttempts),
					Embeds:     nil,
					Components: nil,
				},
			})
			if cfg.FailureAction == "ban" {
				_ = s.GuildBanCreateWithReason(guildID, userID, "Failed captcha verification", 0)
			} else if cfg.FailureAction == "kick" {
				_ = s.GuildMemberDeleteWithReason(guildID, userID, "Failed captcha verification")
			}
		} else {
			updateAttempts(userID, guildID, newAttempts)
			img, err := captcha.New(150, 64)
			if err != nil {
				return
			}
			updateAnswer(userID, guildID, img.Text)
			var pngBuf bytes.Buffer
			_ = img.WriteImage(&pngBuf)
			decoys := []string{
				genDecoy(img.Text),
				genDecoy(img.Text),
				genDecoy(img.Text),
				genDecoy(img.Text),
			}
			options := append(decoys, img.Text)
			rand.Seed(time.Now().UnixNano())
			rand.Shuffle(len(options), func(i, j int) {
				options[i], options[j] = options[j], options[i]
			})
			var menuOptions []discordgo.SelectMenuOption
			for _, opt := range options {
				menuOptions = append(menuOptions, discordgo.SelectMenuOption{
					Label: opt,
					Value: opt,
				})
			}
			var footer string
			var footerIcon string
			if rc, ok := p.mgr.ResolvedCfgFor(s.State.User.ID); ok {
				footer = rc.Footer
				footerIcon = rc.FooterIcon
			}
			if footer == "" {
				footer = "esoteric.win"
			}
			embed := &discordgo.MessageEmbed{
				Title:       "Incorrect Code",
				Description: fmt.Sprintf("❌ That was not the correct code. You have %d attempts remaining.\n\nTry again with the new image below:", cfg.MaxAttempts-newAttempts),
				Color:       0xf1c40f,
				Image: &discordgo.MessageEmbedImage{
					URL: "attachment://captcha.png",
				},
				Footer: &discordgo.MessageEmbedFooter{
					Text:    footer,
					IconURL: footerIcon,
				},
			}
			components := []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.SelectMenu{
							CustomID:    "captcha_select:" + guildID,
							Placeholder: "Select the correct code...",
							Options:     menuOptions,
							MinValues:   intPtr(1),
							MaxValues:   1,
						},
					},
				},
			}
			_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseUpdateMessage,
				Data: &discordgo.InteractionResponseData{
					Embeds:     []*discordgo.MessageEmbed{embed},
					Components: components,
					Files: []*discordgo.File{
						{
							Name:        "captcha.png",
							ContentType: "image/png",
							Reader:      &pngBuf,
						},
					},
				},
			})
		}
	}
}
func intPtr(v int) *int {
	return &v
}