package modules

import (
	"context"
	"fmt"
	"time"

	"PbxGb/client"
	"PbxGb/database"

	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

// HandleAFK is called on every incoming message.
// If AFK mode is active, it sends a one-time auto-reply to each new sender.
func HandleAFK(v *events.Message) {
	// Only handle private DMs — ignore group messages.
	if v.Info.Chat.Server != types.DefaultUserServer &&
		v.Info.Chat.Server != types.HiddenUserServer {
		return
	}

	// Never reply to our own messages.
	if v.Info.IsFromMe {
		return
	}

	wac := client.Get()
	if wac.Store.ID != nil && v.Info.Sender.User == wac.Store.ID.User {
		return
	}
	if wac.Store.LID.User != "" && v.Info.Sender.User == wac.Store.LID.User {
		return
	}

	// Load AFK state from DB.
	state := database.GetAFKState()
	if !state.Enabled {
		return
	}

	sender := v.Info.Sender.String()

	// Owner and sudo users do not get auto-replies.
	if isOwner(v) || database.IsSudo(sender) {
		return
	}

	// Each sender only gets notified once per AFK session.
	if database.IsAFKNotified(sender) {
		return
	}
	_ = database.MarkAFKNotified(sender)

	elapsed := time.Since(state.Since)
	dur := formatDuration(elapsed)

	msg := fmt.Sprintf(
		"╔══════════════════════╗\n"+
			"║    😴 *AFK MODE* 😴    ║\n"+
			"╚══════════════════════╝\n\n"+
			"👋 Hey *%s*!\n"+
			"The owner is currently *AFK* for *%s*.\n\n"+
			"📝 Reason: _%s_\n\n"+
			"⏳ Please wait — the owner will reply when back!",
		v.Info.PushName,
		dur,
		state.Reason,
	)

	senderJID := types.NewJID(v.Info.Sender.User, types.DefaultUserServer)
	_, _ = wac.SendMessage(context.Background(), senderJID, &waProto.Message{
		Conversation: &msg,
	})
}

// formatDuration converts a duration into a human-readable string.
func formatDuration(d time.Duration) string {
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	if h > 0 {
		return fmt.Sprintf("%dh %dm", h, m)
	}
	if m > 0 {
		return fmt.Sprintf("%dm %ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}

// ── COMMANDS ──────────────────────────────────────────────────────────────────

func afkHandler(v *events.Message, args []string) error {
	reason := "I am AFK right now."
	if len(args) > 0 {
		reason = GetArgs(v)
	}

	now := time.Now()
	if err := database.SetAFKState(true, reason, now); err != nil {
		return ReplyErr(v, "❌ Failed to save AFK state. Check database connection.")
	}

	// Clear old notified records so the new AFK session starts fresh.
	_ = database.ClearAFKNotified()

	return ReplyErr(v, fmt.Sprintf(
		"😴 *AFK Mode ON*\n📝 Reason: _%s_\n\nAuto-reply will be sent to new DMs.",
		reason,
	))
}

func unAfkHandler(v *events.Message, args []string) error {
	state := database.GetAFKState()

	if !state.Enabled {
		return ReplyErr(v, "ℹ️ AFK mode is not currently active.")
	}

	elapsed := time.Since(state.Since)
	dur := formatDuration(elapsed)

	if err := database.SetAFKState(false, state.Reason, state.Since); err != nil {
		return ReplyErr(v, "❌ Failed to update AFK state. Check database connection.")
	}

	// Clear notified records when AFK ends.
	_ = database.ClearAFKNotified()

	return ReplyErr(v, fmt.Sprintf(
		"✅ *AFK Mode OFF*\nYou were AFK for *%s*.",
		dur,
	))
}

func afkStatusHandler(v *events.Message, args []string) error {
	state := database.GetAFKState()

	if !state.Enabled {
		return ReplyErr(v, "💤 AFK Mode: *OFF*\nUse *.afk [reason]* to enable.")
	}

	elapsed := time.Since(state.Since)
	dur := formatDuration(elapsed)

	return ReplyErr(v, fmt.Sprintf(
		"😴 AFK Mode: *ON*\n📝 Reason: _%s_\n⏳ Duration: *%s*",
		state.Reason, dur,
	))
}

func init() {
	Register(ModuleInfo{
		Name:        "AFK",
		Description: "Away From Keyboard — persistent DB-backed auto-reply for DMs",
		Commands: []CommandInfo{
			{Pattern: "afk",       Handler: afkHandler,       Sudo: true},
			{Pattern: "unafk",     Handler: unAfkHandler,     Sudo: true},
			{Pattern: "afkstatus", Handler: afkStatusHandler, Sudo: true},
		},
	})
}
