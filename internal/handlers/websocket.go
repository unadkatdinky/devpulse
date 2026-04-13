package handlers

import (
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/unadkatdinky/devpulse/internal/hub"
	"github.com/unadkatdinky/devpulse/pkg/utils"
	"github.com/golang-jwt/jwt/v5"
)

const (
	// writeWait is how long we wait to write a message before giving up.
	writeWait = 10 * time.Second

	// pongWait is how long we wait for a pong response from the client.
	// The browser sends a pong when we send a ping — it's a heartbeat.
	pongWait = 60 * time.Second

	// pingPeriod is how often we send a ping. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// maxMessageSize is the largest message we'll accept from clients.
	maxMessageSize = 512
)

// upgrader converts a regular HTTP connection into a WebSocket connection.
// CheckOrigin controls which domains can connect — we allow all for now.
// In production you'd check the origin against your frontend's domain.
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for development
	},
}

// WSHandler handles WebSocket connections.
type WSHandler struct {
	hub       *hub.Hub
	jwtSecret string
}

// NewWSHandler creates a WSHandler.
func NewWSHandler(h *hub.Hub, jwtSecret string) *WSHandler {
	return &WSHandler{hub: h, jwtSecret: jwtSecret}
}

// ServeWS handles the WebSocket handshake and connection lifecycle.
// URL: GET /ws?token=eyJhbGci...
func (h *WSHandler) ServeWS(w http.ResponseWriter, r *http.Request) {
	// Step 1: Get token from query parameter.
	// Browsers can't set custom headers for WebSocket connections,
	// so we pass the JWT as ?token=... in the URL.
	tokenString := r.URL.Query().Get("token")
	if tokenString == "" {
		utils.JSONError(w, http.StatusUnauthorized, "missing token")
		return
	}

	// Step 2: Validate the JWT token.
	token, err := jwt.ParseWithClaims(tokenString, &jwt.MapClaims{},
		func(token *jwt.Token) (interface{}, error) {
			return []byte(h.jwtSecret), nil
		})

	if err != nil || !token.Valid {
		utils.JSONError(w, http.StatusUnauthorized, "invalid or expired token")
		return
	}

	claims, ok := token.Claims.(*jwt.MapClaims)
	if !ok {
		utils.JSONError(w, http.StatusUnauthorized, "invalid token claims")
		return
	}

	userID, _ := (*claims)["sub"].(string)
	userEmail, _ := (*claims)["email"].(string)

	// Step 3: Upgrade HTTP connection to WebSocket.
	// After this line, the HTTP connection is gone — it's now a WebSocket.
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WS: failed to upgrade connection: %v", err)
		return
	}

	// Step 4: Create a client and register with the hub.
	client := &hub.Client{
		ID:    userID,
		Email: userEmail,
		Send:  make(chan []byte, 256),
	}
	h.hub.RegisterClient(client)

	log.Printf("WS: new connection from user=%s", userEmail)

	// Step 5: Start two goroutines — one reads, one writes.
	// We need both running simultaneously — the connection is full-duplex.
	// writePump runs in a goroutine, readPump runs in the current goroutine.
	go h.writePump(conn, client)
	h.readPump(conn, client)
}

// readPump reads messages from the WebSocket connection.
// In our case we don't expect clients to send much — mainly we handle
// connection close and keep the connection alive with pong responses.
func (h *WSHandler) readPump(conn *websocket.Conn, client *hub.Client) {
	// When readPump exits (connection closed), unregister the client.
	defer func() {
		h.hub.UnregisterClient(client)
		conn.Close()
		log.Printf("WS: connection closed for user=%s", client.Email)
	}()

	conn.SetReadLimit(maxMessageSize)

	// SetReadDeadline means "if we don't hear from the client within
	// pongWait seconds, consider the connection dead".
	conn.SetReadDeadline(time.Now().Add(pongWait))

	// SetPongHandler resets the deadline every time we get a pong.
	// This is how we keep the connection alive indefinitely.
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	// Read loop — blocks here waiting for messages from the client.
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			// websocket.IsUnexpectedCloseError checks if this is a
			// normal close (user closed tab) vs unexpected error.
			if websocket.IsUnexpectedCloseError(err,
				websocket.CloseGoingAway,
				websocket.CloseAbnormalClosure) {
				log.Printf("WS: unexpected close for user=%s: %v", client.Email, err)
			}
			break
		}
	}
}

// writePump writes messages from the client's Send channel to the WebSocket.
// It also sends periodic pings to keep the connection alive.
func (h *WSHandler) writePump(conn *websocket.Conn, client *hub.Client) {
	// ticker sends a tick every pingPeriod — we use this to send pings.
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		conn.Close()
	}()

	for {
		select {
		case message, ok := <-client.Send:
			// Set a deadline for writing this message.
			conn.SetWriteDeadline(time.Now().Add(writeWait))

			if !ok {
				// Hub closed the channel — send a close message and exit.
				conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			// WriteMessage sends the message as a single WebSocket frame.
			if err := conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("WS: write error for user=%s: %v", client.Email, err)
				return
			}

		case <-ticker.C:
			// Send a ping — the browser will respond with a pong.
			conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}