package modules

// PbxGb — Raid Module (WhatsApp)
// ✅ .raid, .hraid, .eraid, .punraid
// ✅ .replyraid, .hreplyraid, .ereplyraid, .preplyraid
// ✅ .stopraid

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"PbxGb/client"
	"PbxGb/config"

	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types/events"
)

var (
	activeRaidChats   = make(map[string]bool)
	activeRaidChatsMu sync.Mutex

	replyRaidWatchers   = make(map[string]bool)
	replyRaidWatchersMu sync.Mutex
)

func setRaidActive(chatID string) {
	activeRaidChatsMu.Lock()
	activeRaidChats[chatID] = true
	activeRaidChatsMu.Unlock()
}

func setRaidStopped(chatID string) {
	activeRaidChatsMu.Lock()
	delete(activeRaidChats, chatID)
	activeRaidChatsMu.Unlock()
}

func isRaidActive(chatID string) bool {
	activeRaidChatsMu.Lock()
	defer activeRaidChatsMu.Unlock()
	return activeRaidChats[chatID]
}

// ─────────────────────────────────────────────
// Normal Raid Handlers
// ─────────────────────────────────────────────

func raidHandler(v *events.Message, args []string) error {
	return genericRaid(v, config.RAID, "raid", 700*time.Millisecond)
}

func hraidHandler(v *events.Message, args []string) error {
	return genericRaid(v, config.HRAID, "hraid", 700*time.Millisecond)
}

func eraidHandler(v *events.Message, args []string) error {
	return genericRaid(v, config.ERAID, "eraid", 700*time.Millisecond)
}

func punraidHandler(v *events.Message, args []string) error {
	return genericRaid(v, config.PUNRAID, "punraid", 700*time.Millisecond)
}

// ─────────────────────────────────────────────
// Reply Raid Handlers
// ─────────────────────────────────────────────

func replyRaidHandler(v *events.Message, args []string) error {
	return startReplyRaidWatcher(v, config.RAID, "raid")
}

func hreplyRaidHandler(v *events.Message, args []string) error {
	return startReplyRaidWatcher(v, config.HRAID, "hraid")
}

func ereplyRaidHandler(v *events.Message, args []string) error {
	return startReplyRaidWatcher(v, config.ERAID, "eraid")
}

func preplyRaidHandler(v *events.Message, args []string) error {
	return startReplyRaidWatcher(v, config.PUNRAID, "punraid")
}

// ─────────────────────────────────────────────
// .stopraid
// ─────────────────────────────────────────────
func stopRaidHandler(v *events.Message, args []string) error {
	chatID := v.Info.Chat.String()

	setRaidStopped(chatID)

	replyRaidWatchersMu.Lock()
	delete(replyRaidWatchers, chatID)
	replyRaidWatchersMu.Unlock()

	// Clear from DB if using
	// database.DeleteRaid(chatID)  // Uncomment if you add DB support later

	return ReplyErr(v, "🛑 *All Raids Stopped!*")
}

// ─────────────────────────────────────────────
// Generic Normal Raid
// ─────────────────────────────────────────────
func genericRaid(v *events.Message, raidList []string, raidType string, delay time.Duration) error {
	var count int
	fmt.Sscanf(GetArgs(v), "%d", &count)
	if count < 1 {
		return ReplyErr(v, "❌ Count must be at least 1.")
	}

	chatID := v.Info.Chat.String()
	setRaidActive(chatID)
	defer setRaidStopped(chatID)

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	for i := 0; i < count; i++ {
		if !isRaidActive(chatID) {
			break
		}

		text := raidList[rng.Intn(len(raidList))]

		_, _ = client.Get().SendMessage(context.Background(), v.Info.Chat, &waProto.Message{
			Conversation: &text,
		})

		time.Sleep(delay)
	}
	return nil
}

// ─────────────────────────────────────────────
// Reply Raid Watcher
// ─────────────────────────────────────────────
func startReplyRaidWatcher(v *events.Message, raidList []string, raidType string) error {
	if !IsReply(v) {
		return ReplyErr(v, "↩️ Reply to the target's message first.")
	}

	// Get target from replied message
	targetJID := "" 
	quoted := GetQuotedMessage(v)
	if quoted != nil && quoted.GetExtendedTextMessage() != nil {
		ctx := quoted.GetExtendedTextMessage().GetContextInfo()
		if ctx != nil && ctx.Participant != nil {
			targetJID = *ctx.Participant
		}
	}
	if targetJID == "" {
		targetJID = v.Info.Sender.String() // fallback
	}

	chatID := v.Info.Chat.String()

	replyRaidWatchersMu.Lock()
	replyRaidWatchers[chatID] = true
	replyRaidWatchersMu.Unlock()

	_, _ = client.Get().SendMessage(context.Background(), v.Info.Chat, &waProto.Message{
		Conversation: stringPtr(fmt.Sprintf(
			"👁 *ReplyRaid Activated*\nTarget: %s\nWaiting for their next message...",
			targetJID,
		)),
	})

	return nil
}

// ─────────────────────────────────────────────
// Trigger Reply Raid (Call from module.go)
// ─────────────────────────────────────────────
func TriggerReplyRaidIfActive(v *events.Message) {
	chatID := v.Info.Chat.String()

	replyRaidWatchersMu.Lock()
	_, active := replyRaidWatchers[chatID]
	replyRaidWatchersMu.Unlock()

	if !active {
		return
	}

	// Get target (you can enhance this with DB later)
	// For now simple: if any replyraid is active in chat, reply to anyone (you can improve)
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Use RAID as default, or make dynamic
	text := config.RAID[rng.Intn(len(config.RAID))]

	_, _ = client.Get().SendMessage(context.Background(), v.Info.Chat, &waProto.Message{
		Conversation: &text,
	})

	// Auto stop after one reply
	replyRaidWatchersMu.Lock()
	delete(replyRaidWatchers, chatID)
	replyRaidWatchersMu.Unlock()
}

// ─────────────────────────────────────────────
// Helper
// ─────────────────────────────────────────────
func stringPtr(s string) *string { return &s }

// ─────────────────────────────────────────────
// REGISTER
// ─────────────────────────────────────────────
func init() {
	Register(ModuleInfo{
		Name:        "Raid",
		Description: "Multi Language Raid + Reply Raid",

		Commands: []CommandInfo{
			{Pattern: "raid", Handler: raidHandler, Sudo: true},
			{Pattern: "hraid", Handler: hraidHandler, Sudo: true},
			{Pattern: "eraid", Handler: eraidHandler, Sudo: true},
			{Pattern: "punraid", Handler: punraidHandler, Sudo: true},

			{Pattern: "replyraid", Handler: replyRaidHandler, Sudo: true},
			{Pattern: "hreplyraid", Handler: hreplyRaidHandler, Sudo: true},
			{Pattern: "ereplyraid", Handler: ereplyRaidHandler, Sudo: true},
			{Pattern: "preplyraid", Handler: preplyRaidHandler, Sudo: true},

			{Pattern: "stopraid", Handler: stopRaidHandler, Sudo: true},
		},
	})
}
