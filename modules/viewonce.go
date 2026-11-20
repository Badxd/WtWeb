package modules

import (
	"context"
	"fmt"
	"sync"

	"PbxGb/client"

	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

var (
	autoVVMu      sync.RWMutex
	autoVVEnabled = false
)

// HandleAutoVV — har incoming view-once message automatically save karo
func HandleAutoVV(v *events.Message) {
	autoVVMu.RLock()
	enabled := autoVVEnabled
	autoVVMu.RUnlock()
	if !enabled {
		return
	}

	// Apne messages ignore
	if v.Info.IsFromMe {
		return
	}

	wac := client.Get()
	if wac.Store.ID != nil && v.Info.Sender.User == wac.Store.ID.User {
		return
	}

	msg := v.Message
	if msg == nil {
		return
	}

	selfJID := getSelfJID()

	// View-once image
	if img := msg.GetImageMessage(); img != nil && img.GetViewOnce() {
		data, err := wac.Download(context.Background(), img)
		if err != nil {
			return
		}
		uploaded, err := wac.Upload(context.Background(), data, whatsmeow.MediaImage)
		if err != nil {
			return
		}
		caption := fmt.Sprintf("👁️ *Auto Saved View Once*\nFrom: *%s*\nChat: %s",
			v.Info.PushName, v.Info.Chat.String())
		_, _ = wac.SendMessage(context.Background(), selfJID, &waProto.Message{
			ImageMessage: &waProto.ImageMessage{
				Caption:       &caption,
				Mimetype:      img.Mimetype,
				URL:           &uploaded.URL,
				DirectPath:    &uploaded.DirectPath,
				MediaKey:      uploaded.MediaKey,
				FileEncSHA256: uploaded.FileEncSHA256,
				FileSHA256:    uploaded.FileSHA256,
				FileLength:    &uploaded.FileLength,
			},
		})
		return
	}

	// View-once video
	if vid := msg.GetVideoMessage(); vid != nil && vid.GetViewOnce() {
		data, err := wac.Download(context.Background(), vid)
		if err != nil {
			return
		}
		uploaded, err := wac.Upload(context.Background(), data, whatsmeow.MediaVideo)
		if err != nil {
			return
		}
		caption := fmt.Sprintf("👁️ *Auto Saved View Once Video*\nFrom: *%s*\nChat: %s",
			v.Info.PushName, v.Info.Chat.String())
		_, _ = wac.SendMessage(context.Background(), selfJID, &waProto.Message{
			VideoMessage: &waProto.VideoMessage{
				Caption:       &caption,
				Mimetype:      vid.Mimetype,
				URL:           &uploaded.URL,
				DirectPath:    &uploaded.DirectPath,
				MediaKey:      uploaded.MediaKey,
				FileEncSHA256: uploaded.FileEncSHA256,
				FileSHA256:    uploaded.FileSHA256,
				FileLength:    &uploaded.FileLength,
				Seconds:       vid.Seconds,
			},
		})
	}
}

// HandleEmojiVV — 🫣 emoji reply pe view-once save karo (no dot prefix needed)
func HandleEmojiVV(v *events.Message) {
	if v.Info.IsFromMe {
		return
	}

	// Sirf owner/sudo ke liye
	if !isOwner(v) {
		return
	}

	text := extractText(v)

allowedEmojis := map[string]bool{
	"🫣": true,
	"😂": true,
	"😭": true,
	"💞": true,
	"❤️": true,
	"🥹": true,
	"😍": true,
	"🥳": true,
	"😙": true,
	"🤥": true,
	"🌝": true,
	"🌚": true,
	"☠️": true,
	"😈": true,
	"👿": true,
	"🙃": true,
	"🙂": true,
}

if !allowedEmojis[text] {
	return
}

	quoted := GetQuotedMessage(v)
	if quoted == nil {
		return
	}

	// Image save
	if img := quoted.GetImageMessage(); img != nil {
		wac := client.Get()
		data, err := wac.Download(context.Background(), img)
		if err != nil {
			return
		}
		uploaded, err := wac.Upload(context.Background(), data, whatsmeow.MediaImage)
		if err != nil {
			return
		}
		selfJID := getSelfJID()
		caption := fmt.Sprintf("%s *Saved via Emoji*\nFrom: *%s*\nChat: %s",
	text, v.Info.PushName, v.Info.Chat.String())
		_, _ = wac.SendMessage(context.Background(), selfJID, &waProto.Message{
			ImageMessage: &waProto.ImageMessage{
				Caption:       &caption,
				Mimetype:      img.Mimetype,
				URL:           &uploaded.URL,
				DirectPath:    &uploaded.DirectPath,
				MediaKey:      uploaded.MediaKey,
				FileEncSHA256: uploaded.FileEncSHA256,
				FileSHA256:    uploaded.FileSHA256,
				FileLength:    &uploaded.FileLength,
			},
		})
		return
	}

	// Video save
	if vid := quoted.GetVideoMessage(); vid != nil {
		wac := client.Get()
		data, err := wac.Download(context.Background(), vid)
		if err != nil {
			return
		}
		uploaded, err := wac.Upload(context.Background(), data, whatsmeow.MediaVideo)
		if err != nil {
			return
		}
		selfJID := getSelfJID()
		caption := fmt.Sprintf("%s *Saved via Emoji*\nFrom: *%s*\nChat: %s",
	text, v.Info.PushName, v.Info.Chat.String())
		_, _ = wac.SendMessage(context.Background(), selfJID, &waProto.Message{
			VideoMessage: &waProto.VideoMessage{
				Caption:       &caption,
				Mimetype:      vid.Mimetype,
				URL:           &uploaded.URL,
				DirectPath:    &uploaded.DirectPath,
				MediaKey:      uploaded.MediaKey,
				FileEncSHA256: uploaded.FileEncSHA256,
				FileSHA256:    uploaded.FileSHA256,
				FileLength:    &uploaded.FileLength,
				Seconds:       vid.Seconds,
			},
		})
	}
}

// .autovv on/off
func autoVVToggleHandler(v *events.Message, args []string) error {
	if len(args) == 0 {
		autoVVMu.RLock()
		s := autoVVEnabled
		autoVVMu.RUnlock()
		state := "🔴 OFF"
		if s {
			state = "🟢 ON"
		}
		return ReplyErr(v, fmt.Sprintf("👁️ Auto View-Once Save: *%s*\n\n🫣 Emoji command: Reply to any image or video with 🫣 → it will be saved to your saved messages", state))
	}
	switch args[0] {
	case "on":
		autoVVMu.Lock()
		autoVVEnabled = true
		autoVVMu.Unlock()
		return ReplyErr(v, "✅ *Auto View-Once ON*\nSaari view-once media automatically saved messages vich jaayegi.")
	case "off":
		autoVVMu.Lock()
		autoVVEnabled = false
		autoVVMu.Unlock()
		return ReplyErr(v, "🔴 *Auto View-Once OFF*")
	}
	return ReplyErr(v, "❌ Use: .autovv on / off")
}

func init() {
	Register(ModuleInfo{
		Name:        "ViewOnce",
		Description: "Auto save view-once + 🫣 emoji command",
		Commands: []CommandInfo{
			{Pattern: "autovv", Handler: autoVVToggleHandler, Sudo: true},
		},
	})
}
