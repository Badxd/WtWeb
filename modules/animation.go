package modules

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"PbxGb/client"

	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types/events"
)

var animationList = map[string][]string{
	"love":    {"❤️ I", "❤️ I Love", "❤️ I Love You", "❤️ I Love You <3"},
	"bad":     {"❤️ ᴍʏ", "💝 ᴍʏ ᴄᴜᴛᴇ", "💞 ᴍʏ ᴄᴜᴛᴇ ᴏᴡɴᴇʀ", "[⎯꯭̽🇨🇦꯭꯭ ⃪В꯭α꯭∂ ꯭м꯭υ꯭η꯭∂꯭α_꯭آآ⎯꯭ ꯭̽🌸](https://t.me/ll_BAD_MUNDA_ll)"},
	"rain":    {"🌬", "☁️", "🌩", "🌨", "🌧", "🌦", "🌨🌩🌦🌥⛅🌤"},
	"shizu":   {"😘😘ʙᴀʙʏ sᴜɴᴏ ɴᴀ😘😘", "😍ᴍᴇʀᴀ ʙᴀᴄʜᴀ 😍", "😍ʙᴀʙʏ ɪ ʟᴏᴠᴇ ʟᴏᴠᴇ ʟᴏᴠᴇ ʏᴏᴜ🥰", "😘ʙᴀʙʏ ɪ ᴍɪss ʏᴏᴜ sᴏ ᴍᴜᴄʜ🙁", "🫣ᴛᴜ ᴍᴇʀɪ ᴀ ᴊᴀᴀɴ ᴍᴇᴛɪ 🤗", "💋ʙᴀʙʏ ɪ ᴋɪss ʏᴏᴜ 🥰", "🙈sʜɪᴢᴜ ᴊᴀᴀɴ ɪ ʟᴏᴠᴇ ʏᴏᴜ ɪ ᴍɪss ʏᴏᴜ ɪ ᴋɪss ʏᴏᴜ💫🥰❤️"},
	"hmm":     {"┈┈╱▔▔▔▔▔╲┈┈┈HM┈HM\n┈╱┈┈╱▔╲╲╲▏┈┈┈HMMM\n╱┈┈╱━╱▔▔▔▔▔╲━╮┈┈\n▏┈▕┃▕╱▔╲╱▔╲▕╮┃┈┈\n▏┈▕╰━▏▊▕▕▋▕▕━╯┈┈\n╲┈┈╲╱▔╭╮▔▔┳╲╲┈┈┈\n┈╲┈┈▏╭━━━━╯▕▕┈┈┈\n┈┈╲┈╲▂▂▂▂▂▂╱╱┈┈┈"},
	"ahh":     {"ahh", "aahh", "aahhh", "aahhhh", "aahhhhh", "aahhhhhh", "aahhhhhhh", "aaahhhhhhhh"},
	"bomb":    {"▪️▪️▪️▪️\n▪️▪️▪️▪️\n▪️▪️▪️▪️\n▪️▪️▪️▪️\n▪️▪️▪️▪️", "💣💣💣💣\n▪️▪️▪️▪️\n▪️▪️▪️▪️\n▪️▪️▪️▪️\n▪️▪️▪️▪️", "▪️▪️▪️▪️\n💣💣💣💣\n▪️▪️▪️▪️\n▪️▪️▪️▪️\n▪️▪️▪️▪️", "▪️▪️▪️▪️\n▪️▪️▪️▪️\n💣💣💣💣\n▪️▪️▪️▪️\n▪️▪️▪️▪️", "▪️▪️▪️▪️\n▪️▪️▪️▪️\n▪️▪️▪️▪️\n▪️▪️▪️▪️\n💥💥💥💥", "`RIP PLOXXX......`"},
	"hack":    {"Looking for WhatsApp databases...", "User online: True\nTelegram access: True", "Hacking... 0%\n[░░░░░░░░░░░░░░░░░░░░]\nETA: 0m, 20s", "Hacking... 25%\n[█████░░░░░░░░░░░░░░░]\nETA: 0m, 15s", "Hacking... 50%\n[██████████░░░░░░░░░░]\nETA: 0m, 10s", "Hacking... 75%\n[███████████████░░░░░]\nETA: 0m, 5s", "Hacking... 100%\n[████████████████████]\nScanning file...", "Hacking complete!\n✅ File uploaded to server.\n`./DOWNLOADS/msgstore.db.crypt12`"},
	"dino":    {"`DIN DINNN.....`", "`🏃                        🦖`", "`🏃                     🦖`", "`🏃                  🦖`", "`🏃               🦖`", "`🏃            🦖`", "`🏃WORGH!   🦖`", "`🏃            🦖`", "`🏃               🦖`", "`🏃                  🦖`", "`🏃 Huh-Huh           🦖`", "`🏃         🦖`", "`HE'S GETTING CLOSER!!!`", "`🏃    🦖`", "`🏃  🦖`", "`Just give up`", "`🧎🦖`", "`-ENDED-`"},
	"theart":  {"❤️", "🧡", "💛", "💚", "💙", "💜", "🖤", "💘", "💝", "❤️", "🧡", "💛", "💝", "💜"},
	"wtf":     {"What", "What The", "What The F", "What The F Brah", "*What The F Brah* 😂"},
	"gangstar": {"EVERyBOdy", "iZ", "GangSTur", "UNtIL", "I", "ArRivE", "🔥🔥🔥", "EVERyBOdy iZ GangSTur UNtIL I ArRivE 🔥🔥🔥"},
	"charging": {"`Tesla Wireless Charging...\nDevice: Nokia 1100\nBattery: 10%`", "`Tesla Wireless Charging...\nDevice: Nokia 1100\nBattery: 30%`", "`Tesla Wireless Charging...\nDevice: Nokia 1100\nBattery: 50%`", "`Tesla Wireless Charging...\nDevice: Nokia 1100\nBattery: 70%`", "`Tesla Wireless Charging...\nDevice: Nokia 1100\nBattery: 90%`", "`Tesla Wireless Charging...\nDevice: Nokia 1100\nBattery: 100% ✅`"},
	"plain":   {"Wait for plane...", "✈-------------", "---✈----------", "------✈-------", "---------✈----", "------------✈-", "------------✈ 🛬"},
	"music":   {"starting player...", "⬤⬤⬤ 81%\n\n🎵 *Now Playing: Shape of You*\n\n`00:00` ▱▱▱▱▱▱▱▱▱▱ `00:10`\n\n`🔂` `⏮️` `⏪️` `▶️` `⏩️` `⏭️`", "⬤⬤⬤ 81%\n\n🎵 *Now Playing: Shape of You*\n\n`00:04` ▰▰▰▰▱▱▱▱▱▱ `00:10`\n\n`🔂` `⏮️` `⏪️` `⏸️` `⏩️` `⏭️`", "⬤⬤◯ 80%\n\n🎵 *Now Playing: Shape of You*\n\n`00:08` ▰▰▰▰▰▰▰▰▱▱ `00:10`\n\n`🔂` `⏮️` `⏪️` `⏸️` `⏩️` `⏭️`", "⬤⬤◯ 80%\n\n🎵 *Now Playing: Shape of You*\n\n`00:10` ▰▰▰▰▰▰▰▰▰▰ `00:10`\n\n`🔂` `⏮️` `⏪️` `⏺️` `⏩️` `⏭️`"},
	"snake":   {"◼️◼️◼️◼️◼️\n◼️◼️◼️◼️◼️\n◼️◼️◼️◼️◼️\n◼️◼️◼️◼️◼️\n◼️◼️◼️◼️◼️", "◻️◼️◼️◼️◼️\n◼️◼️◼️◼️◼️\n◼️◼️◼️◼️◼️\n◼️◼️◼️◼️◼️\n◼️◼️◼️◼️◼️", "◻️◻️◼️◼️◼️\n◼️◼️◼️◼️◼️\n◼️◼️◼️◼️◼️\n◼️◼️◼️◼️◼️\n◼️◼️◼️◼️◼️", "◻️◻️◻️◼️◼️\n◼️◼️◼️◼️◼️\n◼️◼️◼️◼️◼️\n◼️◼️◼️◼️◼️\n◼️◼️◼️◼️◼️", "◻️◻️◻️◻️◻️\n◼️◼️◼️◼️◼️\n◼️◼️◼️◼️◼️\n◼️◼️◼️◼️◼️\n◼️◼️◼️◼️◼️", "◻️◻️◻️◻️◻️\n◼️◼️◼️◼️◻️\n◼️◼️◼️◼️◻️\n◼️◼️◼️◼️◻️\n◼️◼️◼️◼️◻️", "◻️◻️◻️◻️◻️\n◻️◼️◼️◼️◻️\n◻️◼️◼️◼️◻️\n◻️◼️◼️◼️◻️\n◻️◻️◻️◻️◻️", "◻️◻️◻️◻️◻️\n◻️◻️◼️◻️◻️\n◻️◻️◻️◻️◻️\n◻️◻️◻️◻️◻️\n◻️◻️◻️◻️◻️", "◻️◻️◻️◻️◻️\n◻️◼️◻️◼️◻️\n◻️◻️◻️◻️◻️\n◻️◼️◼️◼️◻️\n◻️◻️◻️◻️◻️"},
	"gm":      {"ɢᴏᴏᴅ ᴍᴏʀɴɪɴɢ...\n｡♥｡･ﾟ♡ﾟ･｡♥｡\n╱╱╱╱╱╱╱╭╮╱╱╱╭╮╱╭╮╭╮\n╭━┳━┳━┳╯┃╭━┳╋╋━┫╰┫╰╮\n┃╋┃╋┃╋┃╋┃┃┃┃┃┃╋┃┃┃╭┫\n╰━╯╱╱╱╱╱╱╱╱╱╱╰━╯\n｡♥｡･ﾟ♡ﾟ･｡♥｡"},
	"gn":      {"ɢᴏᴏᴅ ɴɪɢʜᴛ...\n｡♥｡･ﾟ♡ﾟ･｡♥｡\n╱╱╱╱╱╱╱╭╮╱╱╱╱╱╱╱╱╱╱╭╮\n╭━┳━┳━┳╯┃╭━━┳━┳┳┳━┳╋╋━┳┳━╮\n┃╋┃╋┃╋┃╋┃┃┃┃┃╋┃╭┫┃┃┃┃┃┃╋┃\n╰━╯╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╰━╯\n｡♥｡･ﾟ♡ﾟ･｡♥｡"},
	"nikal":   {"`⠀⠀⠀⣠⣶⡾⠏⠉⠙⠳⢦⡀⠀⠀⠀⢠⠞⠉⠙⠲⡀⠀`\nNikal", "`⠀⠀⠀⣠⣶⡾⠏⠉⠙⠳⢦⡀⠀⠀⠀⢠⠞⠉⠙⠲⡀⠀`\nLavde", "`⠀⠀⠀⣠⣶⡾⠏⠉⠙⠳⢦⡀⠀⠀⠀⢠⠞⠉⠙⠲⡀⠀`\nPehli Fursat Meeee", "`⠀⠀⠀⣠⣶⡾⠏⠉⠙⠳⢦⡀⠀⠀⠀⢠⠞⠉⠙⠲⡀⠀`\nlodu Nikal 😂"},
	"loveu":   {"╲╲╲┏━━┓╭━━━╮╱╱╱\n╲╲╲┗┓┏┛┃╭━╮┃╱╱╱\n╲╲╲╲┃┃┏┫┃╭┻┻┓╱╱\n╱╱╱┏╯╰╯┃╰┫┏━╯╱╱\n╱╱┏┻━┳┳┻━┫┗┓╱╱╱\n╱╱╰━┓┃┃╲┏┫┏┛╲╲╲\n╱╱╱╱┃╰╯╲┃┃┗━╮╲╲\n╱╱╱╱╰━━━╯╰━━┛╲╲", "░I░L░O░V░E░Y░O░U░", "╔ღ═╗╔╗\n╚╗╔╝║║ღ═╦╦╦═ ღ\n╔╝╚╗ ღ╚╣║║║║╠╣\n╚═ ღ╝╚═╩═╩ ღ╩═╝"},
	"ror":     {"Rooor", "Rooor Aahh", "Rooor Aahh Aahh", "Rooor Aahh Aahh Aahh 😂"},
	"tank":    {"█۞███████]▄▄▄▄▄▄▄▄▄▄▃\n▂▄▅█████████▅▄▃▂...\n[███████████████████]\n◥⊙▲⊙▲⊙▲⊙▲⊙▲⊙▲⊙◤"},
	"cat":     {"┈┈┏━╮╭━┓┈╭━━━━╮\n┈┈┃┏┗┛┓┃╭┫cat ┃\n┈┈╰┓▋▋┏╯╯╰━━━━╯\n┈╭━┻╮╲┗━━━━╮╭╮┈\n┈┃▎▎┃╲╲╲╲╲╲┣━╯┈\n┈╰━┳┻▅╯╲╲╲╲┃┈┈┈\n┈┈┈╰━┳┓┏┳┓┏╯┈┈┈\n┈┈┈┈┈┗┻┛┗┻┛┈┈┈┈"},
	"scam":    {"`Activates Witchcraft Commands Online....`", "`Search for This Person's Name...`", "`Online Witchcraft Performed Immediately`", "0%   ▎", "30%   ███▌", "60%   ███████", "90%   ████████████", "100%   █████████████", "*Target Successfully Scammed Online 🥴*"},
	"addsudo": {"Deploying........", "*Heroku Connecting To Latest Github Build*", "*Build started by user PBX 4.0*", "*Restarting Heroku Server...*", "*State changed from up to starting*", "*Starting process with command ./start.sh*", "*State changed from starting to up*", "__INFO:PbxGb:Logged in successfully__", "__INFO:PbxGb:Successfully loaded all plugins__", "*Build Succeeded ✅*"},
}

// animEditHandler — edit karta hai ek hi message nu (Telegram style)
func animEditHandler(v *events.Message, args []string) error {
	if len(args) < 1 {
		keys := make([]string, 0, len(animationList))
		for k := range animationList {
			keys = append(keys, k)
		}
		return ReplyErr(v, fmt.Sprintf(
			"╔══════════════════╗\n║  🎬 *ANIMATIONS*   ║\n╚══════════════════╝\n\n*Available:*\n%s\n\n*Usage:* `.anim love`",
			strings.Join(keys, ", "),
		))
	}

	animType := strings.ToLower(args[0])
	frames, exists := animationList[animType]
	if !exists {
		return ReplyErr(v, "❌ Animation not found. Type `.anim` to see the list.")
	}

	wac := client.Get()

	// Pehla frame send karo — message ID milega
	firstFrame := frames[0]
	sentResp, err := wac.SendMessage(context.Background(), v.Info.Chat, &waProto.Message{
		Conversation: &firstFrame,
	})
	if err != nil {
		return err
	}

	time.Sleep(700 * time.Millisecond)

	// Baaki frames — EDIT karo same message
	for i := 1; i < len(frames); i++ {
		frame := frames[i]
		_, _ = wac.SendMessage(context.Background(), v.Info.Chat, &waProto.Message{
			ProtocolMessage: &waProto.ProtocolMessage{
				Key: &waProto.MessageKey{
					FromMe:    boolPtr(true),
					ID:        &sentResp.ID,
					RemoteJID: strPtr(v.Info.Chat.String()),
				},
				Type:          waProto.ProtocolMessage_MESSAGE_EDIT.Enum(),
				EditedMessage: &waProto.Message{Conversation: &frame},
			},
		})
		time.Sleep(700 * time.Millisecond)
	}

	return nil
}

// animListHandler — sari animations ki list
func animListHandler(v *events.Message, args []string) error {
	keys := make([]string, 0, len(animationList))
	for k := range animationList {
		keys = append(keys, k)
	}

	msg := fmt.Sprintf(
		"╔══════════════════════╗\n║  🎬 *ANIMATION LIST*   ║\n╚══════════════════════╝\n\n*Total: %d animations*\n\n%s\n\n*Usage:* `.anim <naam>`\n*Example:* `.anim love`",
		len(keys),
		strings.Join(keys, " • "),
	)
	return ReplyErr(v, msg)
}

// randomAnimHandler
func randomAnimHandler(v *events.Message, args []string) error {
	keys := make([]string, 0, len(animationList))
	for k := range animationList {
		keys = append(keys, k)
	}
	return animEditHandler(v, []string{keys[rand.Intn(len(keys))]})
}

func boolPtr(b bool) *bool { return &b }

func init() {
	Register(ModuleInfo{
		Name:        "Animation",
		Description: "Full animation pack with message edit style",
		Commands: []CommandInfo{
			{Pattern: "anim",       Handler: animEditHandler,  Sudo: true},
			{Pattern: "animlist",   Handler: animListHandler,  Sudo: true},
			{Pattern: "randomanim", Handler: randomAnimHandler, Sudo: true},
		},
	})
}
