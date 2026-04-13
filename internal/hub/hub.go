package hub

import (
	"encoding/json"
	"log"
	"sync"
)

// Client represents one connected browser tab.
// Each client has a channel — when you want to send a message to this
// specific client, you put it in their channel and their goroutine
// picks it up and writes it to the WebSocket.
type Client struct {
	// ID is the user ID from their JWT token
	ID string

	// Email is the user's email — useful for logging
	Email string

	// send is a buffered channel of messages to send to this client.
	// Buffer of 256 means up to 256 messages can queue up before blocking.
	Send chan []byte
}

// Message is the structure we send over WebSocket to the browser.
// Every message has a type so the frontend knows what to do with it.
type Message struct {
	// Type tells the frontend what kind of message this is.
	// Examples: "new_event", "ping", "error"
	Type string `json:"type"`

	// Payload is the actual data — can be anything JSON-serializable.
	// interface{} means "any type" — we'll put event data here.
	Payload interface{} `json:"payload"`
}

// Hub manages all connected clients and broadcasts messages.
type Hub struct {
	// clients is a map of all currently connected clients.
	// map[*Client]bool — the bool is just a placeholder, we only care about the keys.
	// We use a pointer (*Client) so the map holds references, not copies.
	clients map[*Client]bool

	// mu protects the clients map from concurrent access.
	// Multiple goroutines might register/unregister at the same time —
	// without a mutex this would cause a race condition and crash.
	mu sync.RWMutex

	// register receives new clients to add to the hub.
	register chan *Client

	// unregister receives clients to remove from the hub.
	unregister chan *Client

	// broadcast receives messages to send to ALL connected clients.
	broadcast chan []byte
}

// New creates a new Hub. Call this once in main().
func New() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan []byte, 256),
	}
}

// Run starts the hub's main loop. Call this in a goroutine: go hub.Run()
// This function runs forever, processing register/unregister/broadcast events.
func (h *Hub) Run() {
	log.Println("WebSocket hub started")
	for {
		// select waits for whichever channel has something ready first,
		// then runs that case. This is Go's way of handling multiple
		// channels simultaneously.
		select {
		case client := <-h.register:
			// New client connected — add to map
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			log.Printf("Hub: client connected user=%s total=%d",
				client.Email, h.clientCount())

			// Send a welcome message to just this client
			welcome := Message{
				Type:    "connected",
				Payload: map[string]string{"message": "Connected to DevPulse live feed"},
			}
			if data, err := json.Marshal(welcome); err == nil {
				client.Send <- data
			}

		case client := <-h.unregister:
			// Client disconnected — remove from map and close their channel
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.Send)
			}
			h.mu.Unlock()
			log.Printf("Hub: client disconnected user=%s total=%d",
				client.Email, h.clientCount())

		case message := <-h.broadcast:
			// Broadcast to all connected clients
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.Send <- message:
					// Message queued successfully
				default:
					// Client's send buffer is full — they're too slow or disconnected.
					// Remove them to avoid blocking the broadcast for everyone else.
					close(client.Send)
					delete(h.clients, client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

// Broadcast sends a message to all connected clients.
// Call this from the webhook handler when a new event arrives.
func (h *Hub) Broadcast(msg Message) {
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Hub: failed to marshal broadcast message: %v", err)
		return
	}
	// Non-blocking send — if the broadcast channel is full, drop the message
	// rather than blocking the webhook handler.
	select {
	case h.broadcast <- data:
	default:
		log.Println("Hub: broadcast channel full, dropping message")
	}
}

// RegisterClient adds a new client to the hub.
func (h *Hub) RegisterClient(client *Client) {
	h.register <- client
}

// UnregisterClient removes a client from the hub.
func (h *Hub) UnregisterClient(client *Client) {
	h.unregister <- client
}

// clientCount returns the number of connected clients.
func (h *Hub) clientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// ClientCount is the public version for use outside the package.
func (h *Hub) ClientCount() int {
	return h.clientCount()
}