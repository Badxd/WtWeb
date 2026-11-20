// session-web is a standalone, disposable website: it pairs a WhatsApp
// number via whatsmeow, shows the resulting SHORT session string once in
// the browser, and never writes it to disk, a database, or a log line.
// Nothing here talks to Mongo, nothing here writes files — it's meant to
// be deployed on its own (its own Heroku app / its own dyno) purely as a
// "get my session string" tool.
package main

import (
	"context"
	"crypto/rand"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
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

//go:embed index.html
var indexHTML []byte

// ── in-memory-only job tracking (never touches disk) ───────────────────

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

// janitor drops jobs nobody ever polled, so memory doesn't grow forever.
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

	log.Printf("session-web listening on :%s", port)
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

	// A real (but temporary) SQLite file. Heroku's filesystem is ephemeral
	// anyway, and we delete this file ourselves within seconds of pairing —
	// it never gets backed up or persisted, it's just more reliable than
	// SQLite's shared in-memory URI mode across driver versions.
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
		log.Printf("[pair %s] sqlstore.New failed: %v", id, err)
		cleanupFile()
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error (store): " + err.Error()})
		return
	}

	device := container.NewDevice()
	wac := whatsmeow.NewClient(device, waLog.Noop)

	wac.AddEventHandler(func(evt interface{}) {
		switch evt.(type) {
		case *events.Disconnected:
			log.Printf("[pair %s] WhatsApp websocket disconnected", id)
			return
		case *events.Connected:
			log.Printf("[pair %s] WhatsApp connected event received", id)
		default:
			return
		}
		j.mu.Lock()
		defer j.mu.Unlock()
		if j.status == "done" {
			return
		}
		sess, err := encodeShort(wac.Store)
		if err != nil {
			j.status = "error"
			j.errMsg = "Paired, but could not read back the device identity: " + err.Error()
			return
		}
		j.status = "done"
		j.sessionOut = sess

		// Nothing is persisted: disconnect and delete the temp file
		// shortly after so no state lingers on disk either.
		go func() {
			time.Sleep(3 * time.Second)
			wac.Disconnect()
			_ = container.Close()
			cleanupFile()
		}()
	})

	log.Printf("[pair %s] connecting to WhatsApp", id)
	if err := wac.Connect(); err != nil {
		log.Printf("[pair %s] Connect failed: %v", id, err)
		_ = container.Close()
		cleanupFile()
		jobsMu.Lock()
		delete(jobs, id)
		jobsMu.Unlock()
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Could not connect to WhatsApp, try again."})
		return
	}

	// PairPhone needs an active websocket. A fixed sleep doesn't guarantee
	// that the connection is still alive, especially on hosted dynos.
	var code string
	var pairErr error
	for attempt := 1; attempt <= 3; attempt++ {
		if !wac.IsConnected() {
			log.Printf("[pair %s] websocket disconnected before attempt %d; reconnecting", id, attempt)
			if err := wac.Connect(); err != nil {
				pairErr = fmt.Errorf("reconnect: %w", err)
				log.Printf("[pair %s] %v", id, pairErr)
				time.Sleep(2 * time.Second)
				continue
			}
		}
		// Allow the connection to settle, but abort early if it drops.
		ready := false
		for wait := 0; wait < 20; wait++ {
			if wac.IsConnected() {
				ready = true
				break
			}
			time.Sleep(250 * time.Millisecond)
		}
		if !ready {
			pairErr = fmt.Errorf("websocket did not become connected")
			continue
		}
		log.Printf("[pair %s] requesting pair code (attempt %d)", id, attempt)
		ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
		code, pairErr = wac.PairPhone(ctx, number, true, whatsmeow.PairClientChrome, "Chrome (Linux)")
		cancel()
		if pairErr == nil {
			break
		}
		log.Printf("[pair %s] PairPhone attempt %d failed (connected=%t): %v", id, attempt, wac.IsConnected(), pairErr)
		if wac.IsConnected() {
			// Non-network errors (invalid number, rate limits, etc.) shouldn't
			// trigger repeated pairing requests.
			break
		}
		time.Sleep(2 * time.Second)
	}
	if pairErr != nil {
		log.Printf("[pair %s] PairPhone failed: %v", id, pairErr)
		wac.Disconnect()
		_ = container.Close()
		cleanupFile()
		jobsMu.Lock()
		delete(jobs, id)
		jobsMu.Unlock()
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "WhatsApp connection dropped during pairing. Check Heroku logs for the disconnect reason."})
		return
	}

	j.mu.Lock()
	j.status = "code"
	j.code = code
	j.mu.Unlock()

	writeJSON(w, http.StatusOK, map[string]string{"id": id, "code": code})
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
	sess := j.sessionOut
	errMsg := j.errMsg
	j.mu.Unlock()

	resp := map[string]string{"status": status}
	if status == "done" {
		resp["session"] = sess
	} else if status == "error" {
		resp["error"] = errMsg
	}
	writeJSON(w, http.StatusOK, resp)

	// One-time reveal: once the browser has picked up the final state,
	// forget it server-side so the session string can't be re-fetched
	// from this process again.
	if status == "done" || status == "error" {
		jobsMu.Lock()
		delete(jobs, id)
		jobsMu.Unlock()
	}
}

// ── short session encoding (same format as PbxGb/session.Encode) ──────
//
// Duplicated here on purpose: this tool is meant to be deployed
// completely standalone from the bot repo, with no shared module.

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
