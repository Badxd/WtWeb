package modules

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
	pornWatchers   = make(map[string]pornWatcherSession)
	pornWatchersMu sync.Mutex
)

type pornWatcherSession struct {
	targetJID string
}

// .pspam
func pspamHandler(v *events.Message, args []string) error {
	var count int
	fmt.Sscanf(GetArgs(v), "%d", &count)
	if count < 1 {
		count = 5
	}
	if count > 30 {
		count = 30
	}

	revokeCmd(v)
	chatID := v.Info.Chat.String()
	setSpamActive(chatID)
	defer setSpamStopped(chatID)

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	for i := 0; i < count; i++ {
		if !isSpamActive(chatID) {
			break
		}
		videoURL := config.PORNVIDEOS[rng.Intn(len(config.PORNVIDEOS))]
		_, _ = client.Get().SendMessage(context.Background(), v.Info.Chat, &waProto.Message{
			VideoMessage: &waProto.VideoMessage{
				URL:      &videoURL,
				Mimetype: stringPtr("video/mp4"),
			},
		})
		time.Sleep(900 * time.Millisecond)
	}
	return nil
}

// .praid
func praidHandler(v *events.Message, args []string) error {
	var count int
	fmt.Sscanf(GetArgs(v), "%d", &count)
	if count < 1 {
		count = 5
	}
	if count > 25 {
		count = 25
	}

	revokeCmd(v)
	chatID := v.Info.Chat.String()
	setSpamActive(chatID)
	defer setSpamStopped(chatID)

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	for i := 0; i < count; i++ {
		if !isSpamActive(chatID) {
			break
		}
		text := config.PORNTEXT[rng.Intn(len(config.PORNTEXT))]
		videoURL := config.PORNVIDEOS[rng.Intn(len(config.PORNVIDEOS))]

		_, _ = client.Get().SendMessage(context.Background(), v.Info.Chat, &waProto.Message{
			VideoMessage: &waProto.VideoMessage{
				URL:      &videoURL,
				Mimetype: stringPtr("video/mp4"),
				Caption:  &text,
			},
		})
		time.Sleep(1300 * time.Millisecond)
	}
	return nil
}

// .preplyraid
func ppreplyRaidHandler(v *events.Message, args []string) error {
	if !IsReply(v) {
		return ReplyErr(v, "↩️ Reply to target first")
	}

	targetJID := ""
	quoted := GetQuotedMessage(v)
	if quoted != nil && quoted.GetExtendedTextMessage() != nil {
		if ctx := quoted.GetExtendedTextMessage().GetContextInfo(); ctx != nil {
			if p := ctx.GetParticipant(); p != "" {
				targetJID = p
			}
		}
	}
	if targetJID == "" {
		targetJID = v.Info.Sender.String()
	}

	chatID := v.Info.Chat.String()
	pornWatchersMu.Lock()
	pornWatchers[chatID] = pornWatcherSession{targetJID: targetJID}
	pornWatchersMu.Unlock()

	revokeCmd(v)

	return ReplyErr(v, fmt.Sprintf("👁️ *PReply Raid ON*\nTarget: %s", targetJID))
}

// Trigger
func TriggerPReplyRaidIfActive(v *events.Message) {
	chatID := v.Info.Chat.String()

	pornWatchersMu.Lock()
	session, active := pornWatchers[chatID]
	pornWatchersMu.Unlock()

	if !active || v.Info.Sender.String() != session.targetJID {
		return
	}

	pornWatchersMu.Lock()
	delete(pornWatchers, chatID)
	pornWatchersMu.Unlock()

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	text := config.PORNTEXT[rng.Intn(len(config.PORNTEXT))]
	videoURL := config.PORNVIDEOS[rng.Intn(len(config.PORNVIDEOS))]

	_, _ = client.Get().SendMessage(context.Background(), v.Info.Chat, &waProto.Message{
		VideoMessage: &waProto.VideoMessage{
			URL:      &videoURL,
			Mimetype: stringPtr("video/mp4"),
			Caption:  &text,
		},
	})
}

// REGISTER
func init() {
	Register(ModuleInfo{
		Name:        "PornRaid",
		Description: "Porn Spam + Reply Raid",

		Commands: []CommandInfo{
			{Pattern: "pspam", Handler: pspamHandler, Sudo: true},
			{Pattern: "praid", Handler: praidHandler, Sudo: true},
			{Pattern: "preplyraid", Handler: preplyRaidHandler, Sudo: true},
		},
	})
}
