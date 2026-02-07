package api

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"orderbook-backend/internal/yellow"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for development
	},
}

// Message is a WebSocket message
type Message struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

// Client represents a WebSocket client
type Client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan []byte

	// Yellow Network session info
	yellowToken      string
	yellowSessionKey string
	yellowAddress    string
}

// Hub manages all WebSocket clients
type Hub struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
}

// NewHub creates a new WebSocket hub
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

// Run starts the hub
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()

		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

// Broadcast sends a message to all clients
func (h *Hub) Broadcast(msg Message) {
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Failed to marshal message: %v", err)
		return
	}

	select {
	case h.broadcast <- data:
	default:
		log.Printf("Broadcast channel full, dropping message")
	}
}

// ClientCount returns the number of connected clients
func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// handleWebSocket handles WebSocket connections
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	client := &Client{
		hub:  s.wsHub,
		conn: conn,
		send: make(chan []byte, 256),
	}

	s.wsHub.register <- client

	// Start write pump
	go client.writePump()

	// Start read pump (handles disconnection)
	go client.readPump()

	// Send welcome message - client should request specific market orderbook
	msg := Message{
		Type: "connected",
		Data: map[string]string{"status": "connected"},
	}
	data, _ := json.Marshal(msg)
	client.send <- data
}

// writePump sends messages to the WebSocket connection
func (c *Client) writePump() {
	defer func() {
		c.conn.Close()
	}()

	for message := range c.send {
		if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
			return
		}
	}
}

// readPump reads messages from the WebSocket connection
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// Try to parse as Yellow auth message
		if authMsg, err := yellow.ParseYellowAuth(message); err == nil {
			c.handleYellowAuth(authMsg)
			continue
		}

		// Handle other message types here if needed
		log.Printf("Received unhandled message: %s", string(message))
	}
}

// handleYellowAuth handles Yellow Network authentication
func (c *Client) handleYellowAuth(msg *yellow.YellowAuthMessage) {
	log.Printf("Received Yellow auth: session_key=%s", msg.SessionKey)

	// Validate the JWT token
	session, err := yellow.ValidateToken(msg.JWTToken)
	if err != nil {
		log.Printf("Yellow auth failed: %v", err)
		errorMsg := Message{
			Type: "error",
			Data: map[string]string{
				"error": "Invalid Yellow authentication",
			},
		}
		data, _ := json.Marshal(errorMsg)
		c.send <- data
		return
	}

	// Store Yellow session info
	c.yellowToken = msg.JWTToken
	c.yellowSessionKey = msg.SessionKey
	c.yellowAddress = session.Address

	log.Printf("✓ Yellow auth successful for address: %s", c.yellowAddress)

	// Send success response
	successMsg := Message{
		Type: "yellow_auth_success",
		Data: map[string]interface{}{
			"address":     session.Address,
			"session_key": session.SessionKey,
			"expires_at":  session.ExpiresAt.Unix(),
		},
	}
	data, _ := json.Marshal(successMsg)
	c.send <- data
}
