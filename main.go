package main

import (
	"context"
	"crypto/rand"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"

	_ "github.com/mattn/go-sqlite3"
)

//go:embed index.html
var indexHTML []byte

// ── Force IPv4 to fix Termux VPN IPv6 hangs ───────────────────────────

func init() {
	websocket.DefaultDialer.NetDial = func(network, addr string) (net.Conn, error) {
		return net.DialTimeout("tcp4", addr, 10*time.Second)
	}
	net.DefaultResolver.PreferGo = true
	net.DefaultResolver.Dial = func(ctx context.Context, network, address string) (net.Conn, error) {
		d := net.Dialer{Timeout: 10 * time.Second}
		return d.DialContext(ctx, "udp4", address)
	}
}

// ── Job Tracking ──────────────────────────────────────────────────────

type job struct {
	mu         sync.Mutex
	status     string // "waiting" -> "code" -> "done" | "error"
	code       string
	sessionOut string
	errMsg     string
	createdAt  time.Time
}

var (
	jobsMu sync.Mutex
	jobs   = map[string]*job{}
)

func newJobID() string {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func janitor() {
	for {
		time.Sleep(2 * time.Minute)
		cutoff := time.Now().Add(-10 * time.Minute)
		jobsMu.Lock()
		for id, j := range jobs {
			j.mu.Lock()
			old := j.createdAt.Before(cutoff)
			j.mu.Unlock()
			if old {
				delete(jobs, id)
			}
		}
		jobsMu.Unlock()
	}
}

// ── HTTP Handlers ─────────────────────────────────────────────────────

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}

	go janitor()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(indexHTML)
	})
	http.HandleFunc("/pair", handlePair)
	http.HandleFunc("/status", handleStatus)

	log.Printf("🌐 Website running at: http://localhost:%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func handlePair(w http.ResponseWriter, r *http.Request) {
	number := strings.TrimSpace(r.URL.Query().Get("number"))
	number = strings.NewReplacer("+", "", " ", "", "-", "").Replace(number)
	if number == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Enter a WhatsApp number."})
		return
	}

	id := newJobID()
	j := &job{status: "waiting", createdAt: time.Now()}
	jobsMu.Lock()
	jobs[id] = j
	jobsMu.Unlock()

	go runPairingJob(id, number, j)

	// Wait max 25 seconds for pair code
	for i := 0; i < 25; i++ {
		time.Sleep(1 * time.Second)
		j.mu.Lock()
		status := j.status
		code := j.code
		errMsg := j.errMsg
		j.mu.Unlock()

		if status == "code" {
			writeJSON(w, http.StatusOK, map[string]string{"id": id, "code": code})
			return
		}
		if status == "error" {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": errMsg})
			return
		}
	}

	writeJSON(w, http.StatusOK, map[string]string{"id": id, "status": "waiting"})
}

func runPairingJob(id string, number string, j *job) {
	dbPath := filepath.Join(os.TempDir(), "session-web-"+id+".db")
	cleanupFile := func() {
		_ = os.Remove(dbPath)
		_ = os.Remove(dbPath + "-shm")
		_ = os.Remove(dbPath + "-wal")
	}

	container, err := sqlstore.New(
		context.Background(),
		"sqlite3",
		"file:"+dbPath+"?_foreign_keys=on",
		nil,
	)
	if err != nil {
		j.mu.Lock()
		j.status = "error"
		j.errMsg = "DB error: " + err.Error()
		j.mu.Unlock()
		cleanupFile()
		return
	}

	device := container.NewDevice()
	wac := whatsmeow.NewClient(device, waLog.Noop)

	wac.AddEventHandler(func(evt interface{}) {
		switch v := evt.(type) {
		case *events.PairSuccess:
			time.Sleep(2 * time.Second) // Sync keys
			if wac.Store.ID != nil {
				sess, err := encodeShort(wac.Store)
				j.mu.Lock()
				defer j.mu.Unlock()
				if err == nil && sess != "" {
					j.status = "done"
					j.sessionOut = sess
				} else {
					j.status = "error"
					j.errMsg = "Failed to extract session: " + err.Error()
				}
				go func() {
					time.Sleep(2 * time.Second)
					wac.Disconnect()
					_ = container.Close()
					cleanupFile()
				}()
			}
		case *events.PairError:
			log.Printf("[pair %s] Pair Error: %v", id, v)
		}
	})

	if err := wac.Connect(); err != nil {
		j.mu.Lock()
		j.status = "error"
		j.errMsg = "Connect error: " + err.Error()
		j.mu.Unlock()
		_ = container.Close()
		cleanupFile()
		return
	}

	// Wait for connection
	connected := false
	for i := 0; i < 30; i++ {
		if wac.IsConnected() {
			connected = true
			break
		}
		time.Sleep(1 * time.Second)
	}
	if !connected {
		j.mu.Lock()
		j.status = "error"
		j.errMsg = "Timeout connecting to WhatsApp."
		j.mu.Unlock()
		wac.Disconnect()
		_ = container.Close()
		cleanupFile()
		return
	}

	// Use context.Background() so HTTP timeout doesn't cancel pairing
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	code, err := wac.PairPhone(ctx, number, true, whatsmeow.PairClientChrome, "Chrome (Linux)")
	if err != nil {
		j.mu.Lock()
		j.status = "error"
		j.errMsg = "PairPhone failed: " + err.Error()
		j.mu.Unlock()
		wac.Disconnect()
		_ = container.Close()
		cleanupFile()
		return
	}

	j.mu.Lock()
	j.status = "code"
	j.code = code
	j.mu.Unlock()
}

func handleStatus(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	jobsMu.Lock()
	j, ok := jobs[id]
	jobsMu.Unlock()
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}

	j.mu.Lock()
	status := j.status
	code := j.code
	sess := j.sessionOut
	errMsg := j.errMsg
	j.mu.Unlock()

	resp := map[string]string{"status": status}
	if status == "code" {
		resp["code"] = code
	} else if status == "done" {
		resp["session"] = sess
	} else if status == "error" {
		resp["error"] = errMsg
	}
	writeJSON(w, http.StatusOK, resp)

	if status == "done" || status == "error" {
		jobsMu.Lock()
		delete(jobs, id)
		jobsMu.Unlock()
	}
}

// ── Session Encoding ──────────────────────────────────────────────────

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
	if device.ID == nil || device.Account == nil {
		return "", fmt.Errorf("not logged in")
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
