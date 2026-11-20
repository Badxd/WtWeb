package modules

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"PbxGb/client"

	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

var (
	activeSpamChats   = make(map[string]bool)
	activeSpamChatsMu sync.Mutex
)

func setSpamActive(chatID string) {
	activeSpamChatsMu.Lock()
	activeSpamChats[chatID] = true
	activeSpamChatsMu.Unlock()
}

func setSpamStopped(chatID string) {
	activeSpamChatsMu.Lock()
	delete(activeSpamChats, chatID)
	activeSpamChatsMu.Unlock()
}

func isSpamActive(chatID string) bool {
	activeSpamChatsMu.Lock()
	defer activeSpamChatsMu.Unlock()
	return activeSpamChats[chatID]
}

// ─────────────────────────────────────────────
// .spam <count> <text>
// ─────────────────────────────────────────────
func spamHandler(v *events.Message, args []string) error {
	if len(args) < 2 {
		return ReplyErr(v, "⚠️ *Usage:*\n`.spam 10 Hello World`")
	}

	var count int
	fmt.Sscanf(args[0], "%d", &count)
	if count < 1 {
		count = 10
	}
	if count > 50 {
		count = 50
	}

	text := GetArgsAfter(v, 1)

	revokeCmd(v)

	chatID := v.Info.Chat.String()
	setSpamActive(chatID)
	defer setSpamStopped(chatID)

	for i := 0; i < count; i++ {
		if !isSpamActive(chatID) {
			break
		}

		_, _ = client.Get().SendMessage(context.Background(), v.Info.Chat, &waProto.Message{
			Conversation: &text,
		})

		time.Sleep(700 * time.Millisecond)
	}
	return nil
}

// ─────────────────────────────────────────────
// .ds / .delayspam <delay> <count> <text>
// ─────────────────────────────────────────────
func delaySpamHandler(v *events.Message, args []string) error {
	if len(args) < 3 {
		return ReplyErr(v, "⚠️ *Usage:*\n`.ds 1.5 10 Hello World`")
	}

	var delaySec float64
	var count int
	fmt.Sscanf(args[0], "%f", &delaySec)
	fmt.Sscanf(args[1], "%d", &count)

	if count < 1 {
		count = 10
	}
	if count > 50 {
		count = 50
	}

	text := GetArgsAfter(v, 2)
	delay := time.Duration(delaySec * float64(time.Second))

	revokeCmd(v)

	chatID := v.Info.Chat.String()
	setSpamActive(chatID)
	defer setSpamStopped(chatID)

	for i := 0; i < count; i++ {
		if !isSpamActive(chatID) {
			break
		}

		_, _ = client.Get().SendMessage(context.Background(), v.Info.Chat, &waProto.Message{
			Conversation: &text,
		})

		time.Sleep(delay)
	}
	return nil
}

// ─────────────────────────────────────────────
// .sspam <count>
// ─────────────────────────────────────────────
func sspamHandler(v *events.Message, args []string) error {
	if !IsReply(v) {
		return ReplyErr(v, "↩️ Reply to media with:\n`.sspam 10`")
	}

	var count int
	fmt.Sscanf(GetArgs(v), "%d", &count)
	if count < 1 {
		count = 5
	}
	if count > 30 {
		count = 30
	}

	quoted := GetQuotedMessage(v)
	if quoted == nil {
		return ReplyErr(v, "❌ Failed to get quoted message.")
	}

	revokeCmd(v)

	chatID := v.Info.Chat.String()
	setSpamActive(chatID)
	defer setSpamStopped(chatID)

	for i := 0; i < count; i++ {
		if !isSpamActive(chatID) {
			break
		}

		_, _ = client.Get().SendMessage(context.Background(), v.Info.Chat, quoted)
		time.Sleep(1500 * time.Millisecond)
	}
	return nil
}

// ─────────────────────────────────────────────
// .hang <count>
// ─────────────────────────────────────────────
func hangHandler(v *events.Message, args []string) error {
	var count int
	fmt.Sscanf(GetArgs(v), "%d", &count)
	if count < 1 {
		count = 10
	}
	if count > 30 {
		count = 30
	}

	hero := "⃟꙰⃟꙰꙰⃟꙰⃟꙰⃟꙰꙰⃟꙰⃟꙰⃟꙰"

	revokeCmd(v)

	chatID := v.Info.Chat.String()
	setSpamActive(chatID)
	defer setSpamStopped(chatID)

	for i := 0; i < count; i++ {
		if !isSpamActive(chatID) {
			break
		}

		_, _ = client.Get().SendMessage(context.Background(), v.Info.Chat, &waProto.Message{
			Conversation: &hero,
		})
		time.Sleep(700 * time.Millisecond)
	}
	return nil
}

// ─────────────────────────────────────────────
// .stopspam
// ─────────────────────────────────────────────
func stopSpamHandler(v *events.Message, args []string) error {
	chatID := v.Info.Chat.String()
	setSpamStopped(chatID)
	return ReplyErr(v, "🛑 *Spam Stopped!*")
}

// ── Helpers ─────────────────────────────────────
func GetArgsAfter(v *events.Message, start int) string {
	text := extractText(v)
	parts := strings.Fields(text)
	if len(parts) <= start {
		return ""
	}
	return strings.Join(parts[start:], " ")
}

func revokeCmd(v *events.Message) {
	wac := client.Get()
	_, _ = wac.SendMessage(context.Background(), v.Info.Chat, wac.BuildRevoke(v.Info.Chat, types.EmptyJID, v.Info.ID))
}

// REGISTER
func init() {
	Register(ModuleInfo{
		Name:        "Spam",
		Description: "Spam Commands",

		Commands: []CommandInfo{
			{Pattern: "spam", Handler: spamHandler, Sudo: true},
			{Pattern: "ds", Handler: delaySpamHandler, Sudo: true},
			{Pattern: "delayspam", Handler: delaySpamHandler, Sudo: true},
			{Pattern: "sspam", Handler: sspamHandler, Sudo: true},
			{Pattern: "hang", Handler: hangHandler, Sudo: true},
			{Pattern: "stopspam", Handler: stopSpamHandler, Sudo: true},
		},
	})
}
