package commands

import (
	"skyvern/internal/commands/fun"
	"skyvern/internal/commands/general"
	"skyvern/internal/commands/moderation"
	"skyvern/internal/commands/music"
	"skyvern/internal/commands/tickets"
	"skyvern/internal/commands/utility"
	"skyvern/internal/manager"

	"github.com/bwmarrin/discordgo"
)

var Registry = []*manager.Command{
	general.Ping,
	general.Yo,
	general.Help,
	general.Autorole,
	general.DailyQuestion,
	general.DailyQuote,
	general.ServerInfo,
	general.RoleInfo,
	general.Whois,
	general.Pfp,
	general.Banner,
	general.FirstMessage,
	general.InRole,
	general.Math,
	general.Messages,
	general.WhoisWeb,
	general.BoostConfig,
	general.BoosterRole,
	general.Hall,
	general.Timezone,
	general.Birthday,
	general.BumpReminder,
	general.ButtonRole,
	general.ReactionRole,
	general.React,
	general.Reaction,
	general.PreviousReact,
	general.NoSelfReact,
	utility.Uptime,
	utility.Backup,

	moderation.Ban,
	moderation.Unban,
	moderation.Hardban,
	moderation.Softban,
	moderation.Tempban,
	moderation.Kick,
	moderation.Timeout,
	moderation.Untimeout,
	moderation.Nickname,
	moderation.ForceNick,
	moderation.UnforceNick,
	moderation.Modlog,
	moderation.Purge,

	moderation.Warn,
	moderation.Unwarn,
	moderation.Jail,
	moderation.Unjail,
	moderation.Lockdown,
	moderation.Unlock,
	moderation.StripStaff,
	moderation.History,
	moderation.ModStats,
	moderation.ModSearch,
	moderation.Perms,
	moderation.Reason,
	moderation.RMute,
	moderation.Log,
	moderation.Antispam,
	moderation.Filter,
	moderation.Antilink,
	moderation.Roles,
	moderation.Role,
	moderation.Antinuke,
	moderation.Antiraid,
	moderation.Quarantine,
	moderation.Release,
	moderation.Quarantined,
	moderation.Nuke,
	moderation.Prefix,
	moderation.Slowmode,
	moderation.Temprole,
	moderation.Stickyrole,
	moderation.Settings,
	moderation.Thread,
	moderation.Notes,
	moderation.UnbanAll,
	moderation.ClearInvites,
	moderation.Drag,
	moderation.NewMembers,
	moderation.RecentBan,
	moderation.Talk,
	moderation.RevokeFiles,
	moderation.Restrict,
	moderation.Topic,
	moderation.Naughty,
	moderation.Levels,
	moderation.SetXP,
	moderation.RemoveXP,
	moderation.SetLevel,
	moderation.Giveaways,

	utility.Invoke,
	utility.Snipe,
	utility.EditSnipe,
	utility.ReactionSnipe,
	utility.ClearSnipe,
	utility.Hide,
	utility.Unhide,
	utility.ChannelCmd,
	utility.MoveAll,
	utility.AFK,
	utility.Autoreact,
	utility.Autoresponder,
	utility.Dig,
	utility.Convert,
	utility.IP,
	utility.MCServer,
	utility.Remind,
	utility.Reminders,
	utility.Schedule,
	utility.Screenshot,
	utility.Starboard,
	utility.Clownboard,
	utility.Tag,
	utility.VoiceMaster,
	utility.Ticker,
	general.Vanity,
	general.Vouch,
	general.Vouches,
	general.Welcome,
	general.Goodbye,
	utility.ImgOnly,
	utility.Alias,
	utility.StickyMessage,
	utility.ScrapeCmd,
	utility.CrawlCmd,
	utility.AskCmd,
	utility.RedeemCmd,
	utility.TokensCmd,
	utility.AIHistoryCmd,
	utility.OwnerCmd,
	utility.Obfuscate,

	general.Sticker,
	general.Emoji,
	fun.Osu,
	fun.Telegram,
	utility.Highlight,
	utility.InviteInfo,
	general.Boosters,
	general.Bots,
	general.MemberCount,
	general.ChannelInfo,
	general.ServerAvatar,
	general.ServerBanner,
	general.GuildIcon,
	general.GuildBanner,
	general.Splash,

	fun.Define,
	fun.Urban,
	fun.Anime,
	fun.Character,
	fun.Book,
	fun.TVShow,
	fun.Twitch,
	fun.Youtube,
	fun.Game,
	fun.Chess,
	fun.Github,
	fun.Cashapp,
	fun.Tiktok,
	fun.Twitter,
	fun.Spotify,
	fun.Activity,
	fun.Streaming,
	fun.Lyrics,
	fun.FindSong,
	fun.FindID,
	fun.Kanye,
	fun.Compliment,
	fun.Fact,
	fun.Cat,
	fun.Dog,
	fun.ASCII,
	fun.Owoify,
	fun.Piglatin,
	fun.Translate,
	fun.TTS,
	fun.QR,
	fun.Shorten,
	fun.RandomIP,
	fun.Weather,
	fun.DuckDuckGo,
	fun.OCR,
	fun.OCRTR,
	fun.Palette,
	fun.Steal,
	fun.Blunt,
	fun.Juul,
	fun.Yart,
	fun.Weed,
	fun.Rate,
	fun.Ship,

	utility.EmbedExpansion,
	utility.CreateEmbedCmd,
	utility.EditEmbedCmd,
	utility.EmbedCodeCmd,
	utility.NamesCmd,
	utility.ClearNamesCmd,
	utility.GNamesCmd,
	utility.InvitesCmd,
	utility.TopCommandsCmd,

	fun.Uwoify,
	fun.Freaky,
	fun.Quickpoll,
	fun.Poll,
	fun.Timediff,
	fun.Fyp,
	fun.Randomhex,
	fun.Charinfo,
	fun.ColorCmd,
	fun.RPSCmd,
	fun.ChooseCmd,
	fun.JumboCmd,
	fun.WouldyouratherCmd,
	fun.Makemp3Cmd,
	fun.Quote,
	fun.FakeMessage,
	fun.Impersonate,

	music.Play,
	music.Stop,
	music.Pause,
	music.Resume,
	music.Skip,
	music.Queue,
	music.NP,
	music.Seek,
	music.Volume,
	music.Loop,
	music.Shuffle,
	music.Clear,
	music.Fastforward,
	music.Rewind,
	music.Preset,
	tickets.Tickets,
}

func Init(mgr *manager.Manager) {
	mgr.RegisterComponentHandler("help_*", func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		general.HandleHelpComponent(s, i, mgr)
	})
	mgr.RegisterComponentHandler("snipe_*", func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		utility.HandleSnipeComponent(s, i, mgr)
	})
	mgr.RegisterComponentHandler("esnipe_*", func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		utility.HandleSnipeComponent(s, i, mgr)
	})
	mgr.RegisterComponentHandler("rsnipe_*", func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		utility.HandleSnipeComponent(s, i, mgr)
	})
	mgr.RegisterComponentHandler("cmdhelp_*", func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		manager.HandleGlobalHelpComponent(s, i, mgr)
	})
	mgr.RegisterComponentHandler("inrole_*", func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		general.HandleInRoleComponent(s, i, mgr)
	})
	mgr.RegisterComponentHandler("vouch:*", func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		general.HandleVouchComponent(s, i, mgr)
	})
	mgr.RegisterComponentHandler("history_*", func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		moderation.HandleHistoryComponent(s, i, mgr)
	})
	mgr.RegisterComponentHandler("vm_*", func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		utility.HandleVoiceMasterComponent(s, i, mgr)
	})
	mgr.RegisterComponentHandler("steal_*", func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		fun.HandleStealComponent(s, i, mgr)
	})
	mgr.RegisterComponentHandler("weed_page:*", func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		fun.HandleWeedComponent(s, i, mgr)
	})
	mgr.RegisterComponentHandler("giveaway_join_*", func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		moderation.HandleGiveawayJoin(s, i, mgr)
	})
	mgr.RegisterComponentHandler("backup_confirm:*", func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		utility.HandleBackupConfirm(s, i, mgr)
	})
	mgr.RegisterComponentHandler("backup_cancel:*", func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		utility.HandleBackupCancel(s, i, mgr)
	})
	mgr.RegisterComponentHandler("obf_lyrics:*", func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		utility.HandleObfuscateLyricsComponent(s, i, mgr)
	})
}

func init() {
	Registry = append(Registry, fun.RoleplayCommands...)
}
