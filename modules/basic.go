package modules

import (
	"fmt"
	"strings"

	"PbxGb/config"

	"go.mau.fi/whatsmeow/types/events"
)

func pingHandler(v *events.Message, args []string) error {
	return ReplyErr(v, "🏓 Pong!")
}

func startHandler(v *events.Message, args []string) error {
	p := config.C.Prefix
	msg := fmt.Sprintf(
		"╔══════════════════════════╗\n║     *PbxGb 2.0 Bot* 🤖     ║\n╚══════════════════════════╝\n\n✅ Bot is online!\n📌 Type *%shelp* to see all commands.",
		p,
	)
	return ReplyErr(v, msg)
}

func helpHandler(v *events.Message, args []string) error {
	p := config.C.Prefix

	sections := []struct {
		icon  string
		title string
		cmds  []string
	}{
		{"⚡", "GENERAL", []string{
			p + "ping", p + "start", p + "help",
		}},
		{"🎬", "ANIMATION (edit style)", []string{
			p + "anim <type>    — animation chalao",
			p + "animlist       — sari animations dekho",
			p + "randomanim     — random animation",
		}},
		{"🎵", "DOWNLOAD", []string{
			p + "song <url/naam> — YouTube MP3",
			p + "yt <url/naam>   — YouTube MP4 video",
			p + "ytmp3 <url>     — YouTube MP3 (alias)",
			p + "bass            — Reply audio pe → bass boost",
			p + "img <query>     — Image search",
		}},
		{"🌐", "UTILITY", []string{
			p + "calc 2+2        — Calculator",
			p + "remind 10m text   — Set a reminder",
			p + "tinyurl <url>    — Shorten a URL",
			p + "translate hi text — Translate text",
			p + "weather Delhi   — Weather check",
		}},
		{"😂", "FUN", []string{
			p + "joke    — Random joke",
			p + "quote   — Motivational quote",
			p + "fact    — Fun fact",
			p + "toss    — Heads ya Tails",
			p + "dice    — Dice roll",
			p + "8ball   — Magic 8 ball",
			p + "rps     — Rock Paper Scissors",
		}},
		{"🎥", "MEDIA", []string{
			p + "vv              — View-once save (silent)",
			p + "autovv on/off   — Auto view-once save",
			"🫣 emoji          — Reply pe 🫣 bhejo → silent save",
			p + "sticker         — Image → sticker",
			p + "s               — Sticker alias",
		}},
		{"💾", "AUTO SAVE", []string{
			p + "autosave on/off — Auto media save",
			p + "save             — Reply to media to save it",
		}},
		{"📵", "CALL REJECT", []string{
			p + "callreject on/off — Auto call reject",
			p + "callmsg <text>    — Busy message set",
		}},
		{"🗑️", "ANTI DELETE", []string{
			p + "antidelete on/off",
		}},
		{"👁️", "ANTI VIEW ONCE", []string{
			p + "antiviewonce on/off",
		}},
		{"⚡", "AUTO REACT", []string{
			p + "autoreact on/off",
			p + "reactemoji <emoji/random>",
		}},
		{"🔐", "PM SECURITY", []string{
			p + "pmsec on/off     — PM security toggle",
			p + "approve @user    — Approve a user for PM",
			p + "disapprove @user — Disapprove",
			p + "block @user      — Block (FIXED ✅)",
			p + "unblock @user    — Unblock (FIXED ✅)",
			p + "allowlist        — Approved list",
			p + "setpmmsg <text>  — Custom PM msg",
			p + "pmwarn <n>       — Max warns set",
		}},
		{"🔗", "ANTI LINK", []string{
            p + "antilink on/off — Group mein links ban",
            p + "antilink on   — http/https/t.me/instagram/wa.me sab block",
        }},
        {"👥", "GROUP TOOLS", []string{
			p + "tagall <msg>     — Tag all group members",
			p + "kick @user    — Kick",
			p + "add 91xxx     — Add member",
			p + "promote @user  — Make user an admin",
			p + "demote @user   — Remove user from admin",
			p + "groupinfo     — Group details",
			p + "mute/unmute    — Mute or unmute the group",
		}},
		{"🧹", "CLEAN", []string{
			p + "clean <n>      — Delete last n messages",
		}},
		{"🌙", "AFK", []string{
			p + "afk <reason> — AFK mode on",
		}},
		{"👑", "SUDO", []string{
			p + "addsudo 91xxx  — Sudo add",
			p + "rmsudo 91xxx   — Sudo remove",
			p + "sudolist       — Sudo list",
		}},
	}

	var sb strings.Builder
	sb.WriteString("╔══════════════════════════════╗\n")
	sb.WriteString("║    📋 *PbxGb 2.0 COMMANDS*    ║\n")
	sb.WriteString("╚══════════════════════════════╝\n\n")

	for _, s := range sections {
		sb.WriteString(fmt.Sprintf("%s *%s*\n", s.icon, s.title))
		for _, cmd := range s.cmds {
			sb.WriteString(fmt.Sprintf("  • %s\n", cmd))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	sb.WriteString("*PbxGb 2.0* — Fully Updated ✅")

	return ReplyErr(v, sb.String())
}

func init() {
	Register(ModuleInfo{
		Name:        "Basic",
		Description: "ping, start, help",
		Commands: []CommandInfo{
			{Pattern: "ping",  Handler: pingHandler,  Sudo: true},
			{Pattern: "start", Handler: startHandler, Sudo: true},
			{Pattern: "help",  Handler: helpHandler,  Sudo: true},
		},
	})
}
