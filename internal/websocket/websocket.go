package websocket

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"realtime_chat_platform/internal/config"
	"realtime_chat_platform/internal/database"
	"realtime_chat_platform/internal/models"

	"github.com/golang-jwt/jwt/v4"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
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
	typing      chan []byte
	mutex       sync.RWMutex
	typingUsers map[string]bool
	typingMutex sync.RWMutex
}

type Message struct {
	Username  string `json:"username"`
	Content   string `json:"content"`
	Timestamp string `json:"timestamp"`
	Avatar    string `json:"avatar"`
}

type TypingEvent struct {
	Username string `json:"username"`
	IsTyping bool   `json:"is_typing"`
	Type     string `json:"type"`
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

type OnlineUser struct {
	Username string `json:"username"`
	Avatar   string `json:"avatar"`
}

// возвращает список онлайн пользователей с их отображаемой информацией
func (h *Hub) GetOnlineUsers() []OnlineUser {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	users := make([]OnlineUser, 0, len(h.clients))
	for client := range h.clients {
		var user models.User
		if err := database.DB.Where("username = ?", client.Username).First(&user).Error; err == nil {
			displayName := user.Username
			if user.Nickname != "" {
				displayName = user.Nickname
			}
			users = append(users, OnlineUser{
				Username: displayName,
				Avatar:   user.Avatar,
			})
		} else {
			users = append(users, OnlineUser{
				Username: client.Username,
				Avatar:   "",
			})
		}
	}
	return users
}

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

// обновляет статус печати пользователя
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

		var typingEvent TypingEvent
		if err := json.Unmarshal(message, &typingEvent); err == nil && (typingEvent.Type == "typing_start" || typingEvent.Type == "typing_stop") {
			c.Hub.SetUserTyping(typingEvent.Username, typingEvent.IsTyping)
			c.Hub.typing <- message
			continue
		}

		var msg Message
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Printf("Error parsing message: %v", err)
			continue
		}
		if msg.Content == "" {
			log.Printf("Empty message content, skipping save.")
			continue
		}

		var user models.User
		if err := database.DB.Where("username = ?", msg.Username).First(&user).Error; err == nil {
			displayName := user.Username
			if user.Nickname != "" {
				displayName = user.Nickname
			}
			msg.Username = displayName
			msg.Avatar = user.Avatar
		}

		msg.Timestamp = time.Now().Format("2006-01-02 15:04:05")

		c.Hub.SetUserTyping(msg.Username, false)
		stopTypingEvent := TypingEvent{
			Username: msg.Username,
			IsTyping: false,
			Type:     "typing_stop",
		}
		if stopTypingData, err := json.Marshal(stopTypingEvent); err == nil {
			c.Hub.typing <- stopTypingData
		}

		dbMessage := models.Message{
			Username: c.Username,
			Content:  msg.Content,
		}
		if err := database.DB.Create(&dbMessage).Error; err != nil {
			log.Printf("Error saving message to database: %v", err)
		}

		if err := database.DB.Model(&models.User{}).Where("username = ?", c.Username).UpdateColumn("last_active", time.Now()).Error; err != nil {
			log.Printf("Error updating user last active time: %v", err)
		}

		if broadcastData, err := json.Marshal(msg); err == nil {
			c.Hub.broadcast <- broadcastData
		}
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
	// извлечение JWT токена из параметра запроса или заголовка
	tokenString := r.URL.Query().Get("token")
	if tokenString == "" {
		authHeader := r.Header.Get("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			tokenString = strings.TrimPrefix(authHeader, "Bearer ")
		}
	}

	var username string

	if tokenString != "" {
		// парсинг и проверка JWT токена
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return []byte(config.JWTSecret), nil
		})

		if err == nil && token.Valid {
			if claims, ok := token.Claims.(jwt.MapClaims); ok {
				if usernameClaim, ok := claims["username"].(string); ok {
					username = usernameClaim
				}
			}
		}
	}

	// если нет валидного токена, используется анонимный пользователь
	if username == "" {
		username = "Anonymous"
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Error upgrading connection: %v", err)
		return
	}

	client := &Client{
		ID:       "client-" + conn.RemoteAddr().String(),
		Username: username,
		Conn:     conn,
		Hub:      GlobalHub,
		Send:     make(chan []byte, 256),
	}

	client.Hub.register <- client

	go client.WritePump()
	go client.ReadPump()
}
