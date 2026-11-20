package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"PbxGb/client"
	"PbxGb/config"
	"PbxGb/database"
	"PbxGb/modules"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types/events"

	_ "github.com/mattn/go-sqlite3"
)

const sessionDB = "session.db"

func main() {
	config.Load()

	if config.C.Session != "" {
		if err := restoreSession(config.C.Session); err != nil {
			log.Fatalf("❌ Session restore failed: %v", err)
		}
		fmt.Println("✅ Session restored")
	}

	if err := database.Connect(config.C.MongoURI); err != nil {
		log.Fatalf("❌ MongoDB error: %v", err)
	}
	defer database.Disconnect()

	// Load PM security settings into memory now that DB is connected.
	modules.InitPMCache()

	container, err := sqlstore.New(
		context.Background(),
		"sqlite3",
		"file:"+sessionDB+"?_foreign_keys=on",
		nil,
	)
	if err != nil {
		log.Fatalf("❌ SQLite error: %v", err)
	}

	deviceStore, err := container.GetFirstDevice(context.Background())
	if err != nil {
		log.Fatalf("❌ Device error: %v", err)
	}

	wac := whatsmeow.NewClient(deviceStore, nil)
	client.Set(wac)

	wac.AddEventHandler(func(evt interface{}) {
		switch v := evt.(type) {

		case *events.Message:
			// Cache every message for anti-delete
			modules.CacheMessage(v)

			// Anti Delete
			modules.HandleAntiDelete(v)

			// Anti View Once — save all view-once incoming
			modules.HandleAntiViewOnce(v)

			// Auto VV — automatically save view-once to saved messages
			modules.HandleAutoVV(v)
			modules.HandleEmojiVV(v)
			modules.HandleAntiLink(v)

			// Auto Save media to self chat
			modules.HandleAutoSave(v)

			// Auto React to DMs
			modules.HandleAutoReact(v)

			// AFK auto-reply
			modules.HandleAFK(v)

			// PM Security
			modules.HandleGbanRejoin(v)
			modules.HandlePM(v)

			// Command handler (all .commands)
			modules.HandleMessage(v)

		case *events.CallOffer:
			modules.HandleCall(v)
		}
	})

	if wac.Store.ID == nil {
		if err := pairCodeLogin(wac); err != nil {
			log.Fatalf("❌ Login failed: %v", err)
		}
	} else {
		if err := wac.Connect(); err != nil {
			log.Fatalf("❌ Connect error: %v", err)
		}
		fmt.Println("✅ PbxGb Connected")
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	fmt.Println("")
	fmt.Println("🔌 Disconnecting...")
	wac.Disconnect()
}

func restoreSession(sessionStr string) error {
	data, err := base64.StdEncoding.DecodeString(sessionStr)
	if err != nil {
		return fmt.Errorf("base64 decode: %w", err)
	}
	return os.WriteFile(sessionDB, data, 0600)
}

func saveSessionToEnv() error {
	data, err := os.ReadFile(sessionDB)
	if err != nil {
		return err
	}
	session := strings.TrimSpace(base64.StdEncoding.EncodeToString(data))
	envData, err := os.ReadFile(".env")
	if err != nil {
		return err
	}
	var newLines []string
	for _, line := range strings.Split(string(envData), "\n") {
		if !strings.HasPrefix(line, "SESSION=") {
			newLines = append(newLines, line)
		}
	}
	newLines = append(newLines, "SESSION="+session)
	return os.WriteFile(".env", []byte(strings.Join(newLines, "\n")), 0644)
}

func pairCodeLogin(wac *whatsmeow.Client) error {
	banner("PAIR CODE LOGIN")

	if err := wac.Connect(); err != nil {
		return err
	}
	time.Sleep(10 * time.Second)

	var number string
	fmt.Print("Enter WhatsApp Number (91xxxxxxxxxx): ")
	fmt.Scanln(&number)

	number = strings.TrimSpace(number)
	number = strings.ReplaceAll(number, "+", "")
	number = strings.ReplaceAll(number, " ", "")

	code, err := wac.PairPhone(
		context.Background(),
		number,
		true,
		whatsmeow.PairClientChrome,
		"Chrome (Linux)",
	)
	if err != nil {
		fmt.Println("⚠️ Retry after 5 seconds...")
		time.Sleep(5 * time.Second)
		code, err = wac.PairPhone(
			context.Background(),
			number,
			true,
			whatsmeow.PairClientChrome,
			"Chrome (Linux)",
		)
		if err != nil {
			return err
		}
	}

	banner("YOUR PAIR CODE")
	fmt.Println(code)
	fmt.Println("")
	fmt.Println("WhatsApp > Linked Devices > Link with phone number")
	fmt.Println("Enter the code above")
	fmt.Println("Waiting for login...")

	loggedIn := make(chan struct{})
	wac.AddEventHandler(func(evt interface{}) {
		switch evt.(type) {
		case *events.Connected:
			select {
			case <-loggedIn:
			default:
				close(loggedIn)
			}
		}
	})

	select {
	case <-loggedIn:
		fmt.Println("✅ Logged in successfully")
		time.Sleep(5 * time.Second)
		if err := saveSessionToEnv(); err != nil {
			fmt.Println("❌ Failed to save SESSION in .env")
		} else {
			fmt.Println("✅ SESSION saved in .env")
		}
		return nil
	case <-time.After(3 * time.Minute):
		return fmt.Errorf("login timeout")
	}
}

func banner(title string) {
	line := "==================================="
	fmt.Println(line)
	fmt.Printf(" %s\n", title)
	fmt.Println(line)
}
