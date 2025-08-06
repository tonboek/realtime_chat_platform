package websocket

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"realtime_chat_platform/internal/database"
	"realtime_chat_platform/internal/models"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for development
	},
}

type Client struct {
	ID       string
	Username string
	Conn     *websocket.Conn
	Hub      *Hub
	Send     chan []byte
}

type Hub struct {
	clients     map[*Client]bool
	broadcast   chan []byte
	register    chan *Client
	unregister  chan *Client
	typing      chan []byte // Channel for typing events
	mutex       sync.RWMutex
	typingUsers map[string]bool // Track who's currently typing
	typingMutex sync.RWMutex
}

type Message struct {
	Username  string `json:"username"`
	Content   string `json:"content"`
	Timestamp string `json:"timestamp"`
}

type TypingEvent struct {
	Username string `json:"username"`
	IsTyping bool   `json:"is_typing"`
	Type     string `json:"type"` // "typing_start" or "typing_stop"
}

var GlobalHub = NewHub()

func NewHub() *Hub {
	return &Hub{
		clients:     make(map[*Client]bool),
		broadcast:   make(chan []byte),
		register:    make(chan *Client),
		unregister:  make(chan *Client),
		typing:      make(chan []byte),
		mutex:       sync.RWMutex{},
		typingUsers: make(map[string]bool),
		typingMutex: sync.RWMutex{},
	}
}

func (h *Hub) Run() {
	//nolint:staticcheck,SA1012 // This is the correct pattern for WebSocket hub event loop
	for {
		select {
		case client := <-h.register:
			h.mutex.Lock()
			h.clients[client] = true
			h.mutex.Unlock()
			log.Printf("Client registered: %s", client.Username)

		case client := <-h.unregister:
			h.mutex.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.Send)
			}
			h.mutex.Unlock()
			log.Printf("Client unregistered: %s", client.Username)

		case message := <-h.broadcast:
			h.mutex.RLock()
			for client := range h.clients {
				select {
				case client.Send <- message:
				default:
					close(client.Send)
					delete(h.clients, client)
				}
			}
			h.mutex.RUnlock()

		case typingEvent := <-h.typing:
			h.mutex.RLock()
			for client := range h.clients {
				select {
				case client.Send <- typingEvent:
				default:
					close(client.Send)
					delete(h.clients, client)
				}
			}
			h.mutex.RUnlock()
		}
	}
}

// GetOnlineUsers returns a list of usernames of currently connected users
func (h *Hub) GetOnlineUsers() []string {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	users := make([]string, 0, len(h.clients))
	for client := range h.clients {
		users = append(users, client.Username)
	}
	return users
}

// GetTypingUsers returns a list of usernames who are currently typing
func (h *Hub) GetTypingUsers() []string {
	h.typingMutex.RLock()
	defer h.typingMutex.RUnlock()

	users := make([]string, 0, len(h.typingUsers))
	for username, isTyping := range h.typingUsers {
		if isTyping {
			users = append(users, username)
		}
	}
	return users
}

// SetUserTyping updates the typing status of a user
func (h *Hub) SetUserTyping(username string, isTyping bool) {
	h.typingMutex.Lock()
	defer h.typingMutex.Unlock()

	if isTyping {
		h.typingUsers[username] = true
	} else {
		delete(h.typingUsers, username)
	}
}

func (c *Client) ReadPump() {
	defer func() {
		c.Hub.unregister <- c
		c.Conn.Close()
	}()

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			log.Printf("Error reading message: %v", err)
			break
		}

		// Try to parse as typing event first
		var typingEvent TypingEvent
		if err := json.Unmarshal(message, &typingEvent); err == nil && (typingEvent.Type == "typing_start" || typingEvent.Type == "typing_stop") {
			// Handle typing event
			c.Hub.SetUserTyping(typingEvent.Username, typingEvent.IsTyping)
			c.Hub.typing <- message
			continue
		}

		// Parse as regular message
		var msg Message
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Printf("Error parsing message: %v", err)
			continue
		}
		if msg.Content == "" { // Check for empty content
			log.Printf("Empty message content, skipping save.")
			continue
		}

		// Stop typing when message is sent
		c.Hub.SetUserTyping(msg.Username, false)
		stopTypingEvent := TypingEvent{
			Username: msg.Username,
			IsTyping: false,
			Type:     "typing_stop",
		}
		if stopTypingData, err := json.Marshal(stopTypingEvent); err == nil {
			c.Hub.typing <- stopTypingData
		}

		// Store message in database
		dbMessage := models.Message{
			Username: msg.Username,
			Content:  msg.Content,
		}
		if err := database.DB.Create(&dbMessage).Error; err != nil {
			log.Printf("Error saving message to database: %v", err)
		}

		// Update user's last active time without updating updated_at
		if err := database.DB.Model(&models.User{}).Where("username = ?", msg.Username).UpdateColumn("last_active", time.Now()).Error; err != nil {
			log.Printf("Error updating user last active time: %v", err)
		}

		// Broadcast to all clients
		c.Hub.broadcast <- message
	}
}

func (c *Client) WritePump() {
	defer func() {
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			if err := w.Close(); err != nil {
				return
			}
		}
	}
}

func WebSocketHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Error upgrading connection: %v", err)
		return
	}

	// For now, we'll use a simple client ID
	// In a real app, you'd extract user info from JWT token
	client := &Client{
		ID:       "client-" + conn.RemoteAddr().String(),
		Username: "Anonymous", // This should come from JWT token
		Conn:     conn,
		Hub:      GlobalHub,
		Send:     make(chan []byte, 256),
	}

	client.Hub.register <- client

	go client.WritePump()
	go client.ReadPump()
}
