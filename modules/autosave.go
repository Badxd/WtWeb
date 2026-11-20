package modules

// PbxGb — Auto Save to Self Chat
// ✅ Saves any media (image/video/doc/audio) sent to you → saved messages
// ✅ Toggle on/off
// ✅ .save command to manually save replied media

import (
	"context"
	"fmt"
	"sync"

	"PbxGb/client"
	"PbxGb/config"

	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

var (
	autoSaveMu      sync.RWMutex
	autoSaveEnabled = false
)

// getSelfJID — returns owner's self-chat JID
func getSelfJID() types.JID {
	ownerJIDStr := config.C.OwnerJID
	ownerUser := ownerJIDStr
	for i, c := range ownerJIDStr {
		if c == '@' {
			ownerUser = ownerJIDStr[:i]
			break
		}
	}
	return types.NewJID(ownerUser, types.DefaultUserServer)
}

// HandleAutoSave — called from HandleMessage for every incoming message
func HandleAutoSave(v *events.Message) {

	autoSaveMu.RLock()
	enabled := autoSaveEnabled
	autoSaveMu.RUnlock()

	if !enabled {
		return
	}

	if v.Message == nil {
		return
	}

	wac := client.Get()

	// Don't save own messages
	if wac.Store.ID != nil &&
		v.Info.Sender.User == wac.Store.ID.User {
		return
	}

	selfJID := getSelfJID()

	saveMediaToSelf(v, wac, selfJID, false)
}

// saveMediaToSelf — core logic, reused by .save command too
func saveMediaToSelf(
	v *events.Message,
	wac *whatsmeow.Client,
	selfJID types.JID,
	isManual bool,
) {

	msg := v.Message

	label := "🤖 Auto Saved"
	if isManual {
		label = "💾 Manually Saved"
	}

	// IMAGE
	if img := msg.GetImageMessage(); img != nil {
		data, err := wac.Download(context.Background(), img)
		if err != nil {
			return
		}
		uploaded, err := wac.Upload(context.Background(), data, whatsmeow.MediaImage)
		if err != nil {
			return
		}
		caption := fmt.Sprintf("%s\nFrom: *%s*", label, v.Info.PushName)
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

	// VIDEO
	if vid := msg.GetVideoMessage(); vid != nil {
		data, err := wac.Download(context.Background(), vid)
		if err != nil {
			return
		}
		uploaded, err := wac.Upload(context.Background(), data, whatsmeow.MediaVideo)
		if err != nil {
			return
		}
		caption := fmt.Sprintf("%s\nFrom: *%s*", label, v.Info.PushName)
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
		return
	}

	// AUDIO
	if aud := msg.GetAudioMessage(); aud != nil {
		data, err := wac.Download(context.Background(), aud)
		if err != nil {
			return
		}
		uploaded, err := wac.Upload(context.Background(), data, whatsmeow.MediaAudio)
		if err != nil {
			return
		}
		_, _ = wac.SendMessage(context.Background(), selfJID, &waProto.Message{
			AudioMessage: &waProto.AudioMessage{
				Mimetype:      aud.Mimetype,
				URL:           &uploaded.URL,
				DirectPath:    &uploaded.DirectPath,
				MediaKey:      uploaded.MediaKey,
				FileEncSHA256: uploaded.FileEncSHA256,
				FileSHA256:    uploaded.FileSHA256,
				FileLength:    &uploaded.FileLength,
				Seconds:       aud.Seconds,
			},
		})
		return
	}

	// DOCUMENT
	if doc := msg.GetDocumentMessage(); doc != nil {
		data, err := wac.Download(context.Background(), doc)
		if err != nil {
			return
		}
		uploaded, err := wac.Upload(context.Background(), data, whatsmeow.MediaDocument)
		if err != nil {
			return
		}
		caption := fmt.Sprintf("%s\nFrom: *%s*", label, v.Info.PushName)
		_, _ = wac.SendMessage(context.Background(), selfJID, &waProto.Message{
			DocumentMessage: &waProto.DocumentMessage{
				Caption:       &caption,
				Mimetype:      doc.Mimetype,
				FileName:      doc.FileName,
				URL:           &uploaded.URL,
				DirectPath:    &uploaded.DirectPath,
				MediaKey:      uploaded.MediaKey,
				FileEncSHA256: uploaded.FileEncSHA256,
				FileSHA256:    uploaded.FileSHA256,
				FileLength:    &uploaded.FileLength,
			},
		})
	}
}

// .autosave on/off
func autoSaveToggleHandler(
	v *events.Message,
	args []string,
) error {

	if len(args) == 0 {
		autoSaveMu.RLock()
		s := autoSaveEnabled
		autoSaveMu.RUnlock()
		state := "🟢 ON"
		if !s {
			state = "🔴 OFF"
		}
		return ReplyErr(v, fmt.Sprintf("💾 Auto Save: *%s*", state))
	}

	switch args[0] {
	case "on":
		autoSaveMu.Lock()
		autoSaveEnabled = true
		autoSaveMu.Unlock()
		return ReplyErr(v, "✅ Auto Save *ON*\nAll incoming media saved to Saved Messages.")
	case "off":
		autoSaveMu.Lock()
		autoSaveEnabled = false
		autoSaveMu.Unlock()
		return ReplyErr(v, "🔴 Auto Save *OFF*")
	}

	return ReplyErr(v, "❌ Use: .autosave on / off")
}

// .save — manually save replied media
func saveHandler(
	v *events.Message,
	args []string,
) error {

	quoted := GetQuotedMessage(v)
	if quoted == nil {
		return ReplyErr(v, "❌ Reply to a media message to save it.")
	}

	wac := client.Get()
	selfJID := getSelfJID()

	// Fake events.Message with quoted content
	fakeEvt := &events.Message{
		Info:    v.Info,
		Message: quoted,
	}

	saveMediaToSelf(fakeEvt, wac, selfJID, true)

	return ReplyErr(v, "✅ Media saved to *Saved Messages*!")
}

// REGISTER
func init() {

	Register(ModuleInfo{
		Name:        "AutoSave",
		Description: "Auto save media to self chat",

		Commands: []CommandInfo{

			{
				Pattern: "autosave",
				Handler: autoSaveToggleHandler,
				Sudo:    true,
			},

			{
				Pattern: "save",
				Handler: saveHandler,
				Sudo:    true,
			},
		},
	})
}
