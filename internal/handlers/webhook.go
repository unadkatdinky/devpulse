// package handlers

// import (
// 	"crypto/hmac"
// 	"crypto/sha256"
// 	"encoding/hex"
// 	"encoding/json"
// 	"io"
// 	"log"
// 	"net/http"
// 	"strings"

// 	"github.com/unadkatdinky/devpulse/internal/models"
// 	"github.com/unadkatdinky/devpulse/internal/worker"
// 	"gorm.io/gorm"
// )

// // WebhookHandler holds the dependencies our webhook handler needs.
// // This is the same pattern you used for AuthHandler in Day 2.
// type WebhookHandler struct {
// 	db            *gorm.DB
// 	workerPool    *worker.Pool
// 	webhookSecret string
// }

// // NewWebhookHandler creates a WebhookHandler with its dependencies.
// func NewWebhookHandler(db *gorm.DB, pool *worker.Pool, secret string) *WebhookHandler {
// 	return &WebhookHandler{
// 		db:            db,
// 		workerPool:    pool,
// 		webhookSecret: secret,
// 	}
// }

// // HandleGitHubWebhook is called every time GitHub POSTs to /webhooks/github.
// func (h *WebhookHandler) HandleGitHubWebhook(w http.ResponseWriter, r *http.Request) {
// 	// Step 1: Read the entire request body into memory.
// 	// io.ReadAll reads all the bytes from the HTTP request body.
// 	// We need the raw bytes both for signature verification AND for storing
// 	// in the database — so we read it once and reuse it.
// 	body, err := io.ReadAll(r.Body)
// 	if err != nil {
// 		log.Printf("Webhook: failed to read body: %v", err)
// 		http.Error(w, "could not read request body", http.StatusBadRequest)
// 		return
// 	}
// 	defer r.Body.Close()

// 	// Step 2: Verify the HMAC signature.
// 	// GitHub sends the signature in the X-Hub-Signature-256 header.
// 	// If verification fails, we return 401 Unauthorized and stop.
// 	signature := r.Header.Get("X-Hub-Signature-256")
// 	if !h.verifySignature(body, signature) {
// 		log.Printf("Webhook: invalid signature, rejecting request")
// 		http.Error(w, "invalid signature", http.StatusUnauthorized)
// 		return
// 	}

// 	// Step 3: Read GitHub's special headers.
// 	// X-GitHub-Event tells us what kind of event this is (push, pull_request, etc.)
// 	// X-GitHub-Delivery is GitHub's unique ID for this specific delivery.
// 	eventType := r.Header.Get("X-GitHub-Event")
// 	deliveryID := r.Header.Get("X-GitHub-Delivery")

// 	if eventType == "" || deliveryID == "" {
// 		http.Error(w, "missing required GitHub headers", http.StatusBadRequest)
// 		return
// 	}

// 	// Step 4: Parse just enough of the JSON to get the repo name and sender.
// 	// We use an anonymous struct here — a struct defined right where we need it,
// 	// just for this one parsing job. We don't need a named type because we
// 	// store the full raw payload anyway.
// 	var partial struct {
// 		Repository struct {
// 			FullName string `json:"full_name"`
// 		} `json:"repository"`
// 		Sender struct {
// 			Login string `json:"login"`
// 		} `json:"sender"`
// 	}
// 	// json.Unmarshal tries to decode the JSON bytes into our struct.
// 	// If fields are missing it just leaves them as empty strings — no crash.
// 	if err := json.Unmarshal(body, &partial); err != nil {
// 		log.Printf("Webhook: failed to parse JSON body: %v", err)
// 		http.Error(w, "invalid JSON body", http.StatusBadRequest)
// 		return
// 	}

// 	// Step 5: Build the GitHubEvent model and save it to the database.
// 	event := &models.GitHubEvent{
// 		DeliveryID:   deliveryID,
// 		EventType:    eventType,
// 		RepoFullName: partial.Repository.FullName,
// 		Sender:       partial.Sender.Login,
// 		Payload:      string(body), // store the full raw JSON
// 		Processed:    false,
// 	}

// 	// db.Create inserts a new row. The BeforeCreate hook on the model
// 	// will auto-generate the UUID before the INSERT runs.
// 	if err := h.db.Create(event).Error; err != nil {
// 		// Check if this is a duplicate delivery (same delivery ID sent twice).
// 		// GitHub sometimes retries — we don't want to error, just silently skip.
// 		if strings.Contains(err.Error(), "duplicate key") ||
// 			strings.Contains(err.Error(), "unique constraint") {
// 			log.Printf("Webhook: duplicate delivery %s, ignoring", deliveryID)
// 			w.WriteHeader(http.StatusOK)
// 			json.NewEncoder(w).Encode(map[string]string{"status": "duplicate, ignored"})
// 			return
// 		}
// 		log.Printf("Webhook: failed to save event: %v", err)
// 		http.Error(w, "failed to save event", http.StatusInternalServerError)
// 		return
// 	}

// 	// Step 6: Hand the event to the worker pool and return immediately.
// 	// This is the key step — we do NOT wait for processing to finish.
// 	// The worker will handle it in the background.
// 	h.workerPool.Submit(worker.EventJob{Event: event})

// 	log.Printf("Webhook: received and queued event type=%s repo=%s delivery=%s",
// 		eventType, partial.Repository.FullName, deliveryID)

// 	// Step 7: Respond to GitHub with 200 OK.
// 	// GitHub considers a webhook successful only if it gets a 2xx response
// 	// within 10 seconds. We always get here fast because the real work
// 	// is happening in the background goroutine.
// 	w.WriteHeader(http.StatusOK)
// 	json.NewEncoder(w).Encode(map[string]string{
// 		"status":      "received",
// 		"delivery_id": deliveryID,
// 		"event_type":  eventType,
// 	})
// }

// // verifySignature checks that the request genuinely came from GitHub.
// // This is the HMAC-SHA256 calculation explained in the concepts section.
// func (h *WebhookHandler) verifySignature(body []byte, signature string) bool {
// 	// If no secret is configured, skip verification.
// 	// This lets you test locally without setting up a secret.
// 	if h.webhookSecret == "" {
// 		log.Println("WARNING: No webhook secret configured — skipping signature check")
// 		return true
// 	}

// 	// GitHub sends the signature as "sha256=<hex string>".
// 	// We strip the "sha256=" prefix to get just the hex value.
// 	if !strings.HasPrefix(signature, "sha256=") {
// 		return false
// 	}
// 	receivedHash := strings.TrimPrefix(signature, "sha256=")

// 	// Compute our own HMAC-SHA256 of the body using our secret.
// 	// hmac.New creates the hasher, h.Write feeds it the body bytes,
// 	// h.Sum(nil) produces the final hash as bytes,
// 	// hex.EncodeToString turns those bytes into a hex string.
// 	mac := hmac.New(sha256.New, []byte(h.webhookSecret))
// 	mac.Write(body)
// 	expectedHash := hex.EncodeToString(mac.Sum(nil))

// 	// hmac.Equal does a constant-time comparison.
// 	// We use this instead of == to prevent "timing attacks" —
// 	// a security technique where an attacker can guess your secret
// 	// by measuring how long string comparisons take.
// 	return hmac.Equal([]byte(receivedHash), []byte(expectedHash))
// }

package handlers

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/unadkatdinky/devpulse/internal/hub"
	"github.com/unadkatdinky/devpulse/internal/models"
	"github.com/unadkatdinky/devpulse/internal/queue"
	"gorm.io/gorm"
)

// WebhookHandler now also holds a reference to the hub for broadcasting.
type WebhookHandler struct {
	db            *gorm.DB
	queue         *queue.Queue
	webhookSecret string
	hub           *hub.Hub
}

// NewWebhookHandler creates a WebhookHandler with Redis queue and hub.
func NewWebhookHandler(db *gorm.DB, q *queue.Queue, secret string, h *hub.Hub) *WebhookHandler {
	return &WebhookHandler{
		db:            db,
		queue:         q,
		webhookSecret: secret,
		hub:           h,
	}
}

// HandleGitHubWebhook receives, verifies, saves, queues, and broadcasts events.
func (h *WebhookHandler) HandleGitHubWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Webhook: failed to read body: %v", err)
		http.Error(w, "could not read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	signature := r.Header.Get("X-Hub-Signature-256")
	if !h.verifySignature(body, signature) {
		log.Printf("Webhook: invalid signature, rejecting request")
		http.Error(w, "invalid signature", http.StatusUnauthorized)
		return
	}

	eventType := r.Header.Get("X-GitHub-Event")
	deliveryID := r.Header.Get("X-GitHub-Delivery")

	if eventType == "" || deliveryID == "" {
		http.Error(w, "missing required GitHub headers", http.StatusBadRequest)
		return
	}

	var partial struct {
		Repository struct {
			FullName string `json:"full_name"`
		} `json:"repository"`
		Sender struct {
			Login string `json:"login"`
		} `json:"sender"`
	}
	if err := json.Unmarshal(body, &partial); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	event := &models.GitHubEvent{
		DeliveryID:   deliveryID,
		EventType:    eventType,
		RepoFullName: partial.Repository.FullName,
		Sender:       partial.Sender.Login,
		Payload:      string(body),
		Processed:    false,
	}

	if err := h.db.Create(event).Error; err != nil {
		if strings.Contains(err.Error(), "duplicate key") ||
			strings.Contains(err.Error(), "unique constraint") {
			log.Printf("Webhook: duplicate delivery %s, ignoring", deliveryID)
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"status": "duplicate, ignored"})
			return
		}
		log.Printf("Webhook: failed to save event: %v", err)
		http.Error(w, "failed to save event", http.StatusInternalServerError)
		return
	}

	// Push to Redis queue for background processing.
	job := queue.EventJob{
		EventID:      event.ID,
		EventType:    event.EventType,
		RepoFullName: event.RepoFullName,
		DeliveryID:   event.DeliveryID,
	}
	if err := h.queue.Push(context.Background(), job); err != nil {
		log.Printf("Webhook: failed to push job to Redis: %v", err)
	}

	// Broadcast to all connected WebSocket clients immediately.
	// This is what makes the frontend update in real time.
	// We send a summary — not the full payload — to keep messages small.
	h.hub.Broadcast(hub.Message{
		Type: "new_event",
		Payload: map[string]interface{}{
			"id":             event.ID,
			"event_type":     event.EventType,
			"repo_full_name": event.RepoFullName,
			"sender":         event.Sender,
			"delivery_id":    event.DeliveryID,
			"created_at":     event.CreatedAt,
		},
	})

	log.Printf("Webhook: received, queued, and broadcast event type=%s repo=%s",
		eventType, partial.Repository.FullName)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":      "received",
		"delivery_id": deliveryID,
		"event_type":  eventType,
	})
}

func (h *WebhookHandler) verifySignature(body []byte, signature string) bool {
	if h.webhookSecret == "" {
		log.Println("WARNING: No webhook secret configured — skipping signature check")
		return true
	}
	if !strings.HasPrefix(signature, "sha256=") {
		return false
	}
	receivedHash := strings.TrimPrefix(signature, "sha256=")
	mac := hmac.New(sha256.New, []byte(h.webhookSecret))
	mac.Write(body)
	expectedHash := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(receivedHash), []byte(expectedHash))
}