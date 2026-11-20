// session-cli is a standalone, disposable CLI tool: it pairs a WhatsApp
// number via whatsmeow, prints the pairing code to the terminal, and
// outputs the resulting SHORT session string once paired.
package main

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"

	_ "github.com/mattn/go-sqlite3"
)

// ── short session encoding (same format as PbxGb/session.Encode) ──────

func b64(b []byte) string { return base64.StdEncoding.EncodeToString(b) }

type shortSessionBlob struct {
	ID             string `json:"id"`
	LID            string `json:"lid,omitempty"`
	RegistrationID uint32 `json:"reg"`

	NoiseKeyPriv    string `json:"nk"`
	IdentityKeyPriv string `json:"ik"`

	SignedPreKeyPriv string `json:"spk"`
	SignedPreKeyID   uint32 `json:"spkid"`
	SignedPreKeySig  string `json:"spksig"`

	AdvSecretKey string `json:"adv"`

	AccountDetails string `json:"acd"`
	AccountSigKey  string `json:"acsk"`
	AccountSig     string `json:"acs"`
	DeviceSig      string `json:"ds"`

	Platform     string `json:"pl,omitempty"`
	BusinessName string `json:"bn,omitempty"`
	PushName     string `json:"pn,omitempty"`
}

func encodeShort(device *store.Device) (string, error) {
	if device.ID == nil {
		return "", fmt.Errorf("device is not logged in yet")
	}
	if device.Account == nil {
		return "", fmt.Errorf("device has no account identity yet")
	}

	b := shortSessionBlob{
		ID:             device.ID.String(),
		RegistrationID: device.RegistrationID,

		NoiseKeyPriv:    b64(device.NoiseKey.Priv[:]),
		IdentityKeyPriv: b64(device.IdentityKey.Priv[:]),

		SignedPreKeyPriv: b64(device.SignedPreKey.Priv[:]),
		SignedPreKeyID:   device.SignedPreKey.KeyID,
		SignedPreKeySig:  b64(device.SignedPreKey.Signature[:]),

		AdvSecretKey: b64(device.AdvSecretKey),

		AccountDetails: b64(device.Account.Details),
		AccountSigKey:  b64(device.Account.AccountSignatureKey),
		AccountSig:     b64(device.Account.AccountSignature),
		DeviceSig:      b64(device.Account.DeviceSignature),

		Platform:     device.Platform,
		BusinessName: device.BusinessName,
		PushName:     device.PushName,
	}
	if !device.LID.IsEmpty() {
		b.LID = device.LID.String()
	}

	raw, err := json.Marshal(b)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(raw), nil
}

// ── Main Logic ─────────────────────────────────────────────────────────

func main() {
	// 1. Ask for Number
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("📱 Enter WhatsApp Number (e.g. 919465031516): ")
	numberRaw, _ := reader.ReadString('\n')
	number := strings.NewReplacer("+", "", " ", "", "-", "", "\n", "", "\r", "").Replace(numberRaw)

	if number == "" {
		log.Fatal("❌ Number cannot be empty.")
	}

	// 2. Setup temporary SQLite DB
	dbPath := filepath.Join(os.TempDir(), "session-cli-temp.db")
	defer func() {
		os.Remove(dbPath)
		os.Remove(dbPath + "-shm")
		os.Remove(dbPath + "-wal")
	}()

	container, err := sqlstore.New(
		context.Background(),
		"sqlite3",
		"file:"+dbPath+"?_foreign_keys=on",
		nil,
	)
	if err != nil {
		log.Fatalf("❌ DB Error: %v", err)
	}

	device := container.NewDevice()
	wac := whatsmeow.NewClient(device, waLog.Noop)

	// 3. Channels to track state
	connectedCh := make(chan struct{})
	sessionCh := make(chan string)
	var connectedOnce sync.Once
	var pairSuccess bool

	wac.AddEventHandler(func(evt interface{}) {
		switch evt.(type) {
		case *events.Disconnected:
			log.Println("⚠️ WhatsApp websocket disconnected")

		case *events.Connected:
			log.Println("✅ Connected to WhatsApp")
			connectedOnce.Do(func() { close(connectedCh) })

		case *events.PairSuccess:
			pairSuccess = true
			log.Println("✅ Pair Success! Logging in...")

		case *events.PairError:
			log.Println("❌ Pair Error. Code may be invalid or expired.")

		default:
			// If we are paired and have an ID, we can extract the session
			if pairSuccess && wac.Store.ID != nil {
				sess, err := encodeShort(wac.Store)
				if err == nil && sess != "" {
					select {
					case sessionCh <- sess:
						// Sent to main thread
					default:
						// Avoid blocking
					}
				}
			}
		}
	})

	// 4. Connect
	log.Println("🔌 Connecting to WhatsApp...")
	if err := wac.Connect(); err != nil {
		log.Fatalf("❌ Could not connect: %v", err)
	}

	// Wait for connection (max 30s)
	select {
	case <-connectedCh:
		// Ready
	case <-time.After(30 * time.Second):
		log.Fatal("❌ Timeout waiting for connection.")
	}

	// 5. Request Pair Code
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	log.Printf("🔑 Requesting Pair Code for %s...", number)
	code, err := wac.PairPhone(ctx, number, true, whatsmeow.PairClientChrome, "Chrome (Linux)")
	if err != nil {
		log.Fatalf("❌ PairPhone failed: %v", err)
	}

	fmt.Println("\n==============================================")
	fmt.Printf("🔑 YOUR PAIR CODE: %s\n", code)
	fmt.Println("==============================================")
	fmt.Println("👉 1. Open WhatsApp on your phone.")
	fmt.Println("👉 2. Go to Linked Devices -> Link with phone number.")
	fmt.Println("👉 3. Enter the code above.")
	fmt.Println("⏳ Waiting for you to enter the code (max 2 minutes)...\n")

	// 6. Wait for Session
	select {
	case session := <-sessionCh:
		fmt.Println("\n==============================================")
		fmt.Println("✅ SUCCESS! HERE IS YOUR SESSION STRING:")
		fmt.Println("==============================================")
		fmt.Println(session)
		fmt.Println("==============================================")
	case <-time.After(120 * time.Second):
		log.Fatal("❌ Timeout: You did not enter the code on your phone in time.")
	}

	// Cleanup
	wac.Disconnect()
}
