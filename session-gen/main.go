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

	// LOAD CONFIG
	config.Load()

	// RESTORE SESSION
	if config.C.Session != "" {

		if err := restoreSession(config.C.Session); err != nil {

			log.Fatalf(
				"❌ Session restore failed: %v",
				err,
			)
		}

		fmt.Println("✅ Session restored")
	}

	// CONNECT MONGODB
	if err := database.Connect(
		config.C.MongoURI,
	); err != nil {

		log.Fatalf(
			"❌ MongoDB error: %v",
			err,
		)
	}

	defer database.Disconnect()

	// SQLITE STORE
	container, err := sqlstore.New(
		context.Background(),
		"sqlite3",
		"file:"+sessionDB+"?_foreign_keys=on",
		nil,
	)

	if err != nil {

		log.Fatalf(
			"❌ SQLite error: %v",
			err,
		)
	}

	// DEVICE STORE
	deviceStore, err := container.GetFirstDevice(
		context.Background(),
	)

	if err != nil {

		log.Fatalf(
			"❌ Device error: %v",
			err,
		)
	}

	// WHATSAPP CLIENT
	wac := whatsmeow.NewClient(
		deviceStore,
		nil,
	)

	client.Set(wac)

	// EVENTS
	wac.AddEventHandler(func(evt interface{}) {

		switch v := evt.(type) {

		case *events.Message:

			modules.HandleMessage(v)
		}
	})

	// LOGIN
	if wac.Store.ID == nil {

		if err := pairCodeLogin(wac); err != nil {

			log.Fatalf(
				"❌ Login failed: %v",
				err,
			)
		}

	} else {

		if err := wac.Connect(); err != nil {

			log.Fatalf(
				"❌ Connect error: %v",
				err,
			)
		}

		fmt.Println("✅ PbxGb Connected")
	}

	// WAIT
	quit := make(chan os.Signal, 1)

	signal.Notify(
		quit,
		os.Interrupt,
		syscall.SIGTERM,
	)

	<-quit

	fmt.Println("")
	fmt.Println("🔌 Disconnecting...")

	wac.Disconnect()
}

// RESTORE SESSION
func restoreSession(
	sessionStr string,
) error {

	data, err := base64.StdEncoding.DecodeString(
		sessionStr,
	)

	if err != nil {

		return fmt.Errorf(
			"base64 decode: %w",
			err,
		)
	}

	return os.WriteFile(
		sessionDB,
		data,
		0600,
	)
}

// SAVE SESSION TO .ENV
func saveSessionToEnv() error {

	data, err := os.ReadFile(
		sessionDB,
	)

	if err != nil {
		return err
	}

	session := base64.StdEncoding.EncodeToString(
		data,
	)

	envData, err := os.ReadFile(
		".env",
	)

	if err != nil {
		return err
	}

	content := string(envData)

	lines := strings.Split(
		content,
		"\n",
	)

	var newLines []string

	for _, line := range lines {

		if !strings.HasPrefix(
			line,
			"SESSION=",
		) {

			newLines = append(
				newLines,
				line,
			)
		}
	}

	newLines = append(
		newLines,
		"SESSION="+session,
	)

	finalEnv := strings.Join(
		newLines,
		"\n",
	)

	return os.WriteFile(
		".env",
		[]byte(finalEnv),
		0644,
	)
}

// PAIR LOGIN
func pairCodeLogin(
	wac *whatsmeow.Client,
) error {

	banner("PAIR CODE LOGIN")

	// CONNECT
	if err := wac.Connect(); err != nil {

		return err
	}

	// WAIT WS READY
	time.Sleep(10 * time.Second)

	// INPUT NUMBER
	var number string

	fmt.Print(
		"Enter WhatsApp Number (91xxxxxxxxxx): ",
	)

	fmt.Scanln(&number)

	number = strings.TrimSpace(
		number,
	)

	number = strings.ReplaceAll(
		number,
		"+",
		"",
	)

	number = strings.ReplaceAll(
		number,
		" ",
		"",
	)

	// PAIR CODE
	code, err := wac.PairPhone(
		context.Background(),
		number,
		true,
		whatsmeow.PairClientChrome,
		"Chrome (Linux)",
	)

	// RETRY
	if err != nil {

		fmt.Println(
			"⚠️ Retry after 5 seconds...",
		)

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

	// SHOW CODE
	banner("YOUR PAIR CODE")

	fmt.Println(code)

	fmt.Println("")
	fmt.Println("WhatsApp > Linked Devices")
	fmt.Println("Link with phone number")
	fmt.Println("Enter the code")
	fmt.Println("")
	fmt.Println("Waiting for login...")

	// LOGIN WAIT
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

		fmt.Println(
			"✅ Logged in successfully",
		)

		// WAIT SESSION SAVE
		time.Sleep(5 * time.Second)

		// SAVE SESSION
		err := saveSessionToEnv()

		if err != nil {

			fmt.Println(
				"❌ Failed to save session in .env",
			)

		} else {

			fmt.Println(
				"✅ Session saved in .env",
			)
		}

		return nil

	case <-time.After(
		3 * time.Minute,
	):

		return fmt.Errorf(
			"login timeout",
		)
	}
}

// BANNER
func banner(
	title string,
) {

	line := "==================================="

	fmt.Println(line)

	fmt.Printf(
		" %s\n",
		title,
	)

	fmt.Println(line)
}
