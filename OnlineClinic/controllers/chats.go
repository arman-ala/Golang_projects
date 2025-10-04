// ./controllers/chats.go
package controllers

import (
	"encoding/json"
	"net/http"
	"onlineClinic/config"
	"onlineClinic/models"
	"onlineClinic/utils"
	"strconv"
	"sync"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all connections (in production, restrict this to trusted origins)
	},
}

// Hub maintains the set of active Clients and broadcasts messages to them
type Hub struct {
	Clients    map[*Client]bool // Registered Clients
	broadcast  chan []byte      // Inbound messages from Clients
	register   chan *Client     // Register requests from Clients
	unregister chan *Client     // Unregister requests from Clients
	mu         sync.Mutex       // Mutex to protect the Clients map
}

// Client represents a WebSocket connection
type Client struct {
	hub  *Hub
	Conn *websocket.Conn
	send chan []byte
	ID   int // User ID of the client
}

// NewHub initializes a new Hub
func NewHub() *Hub {
	return &Hub{
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		Clients:    make(map[*Client]bool),
	}
}

// Run starts the Hub
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.Clients[client] = true
			// log.Printf("Client registered: %d", client.ID)
		case client := <-h.unregister:
			if _, ok := h.Clients[client]; ok {
				delete(h.Clients, client)
				close(client.send)
				// log.Printf("Client unregistered: %d", client.ID)
			}
		case message := <-h.broadcast:
			for client := range h.Clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.Clients, client)
				}
			}
		}
	}
}

// ServeWs handles WebSocket requests from Clients
func ServeWs(hub *Hub, w http.ResponseWriter, r *http.Request) {
	// Extract the token from the query parameters
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "Missing token", http.StatusUnauthorized)
		return
	}

	// Add "Bearer " prefix to the token
	fullToken := token

	// Validate the token
	claims, err := utils.VerifyToken(fullToken)
	if err != nil {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	// Upgrade the HTTP connection to a WebSocket connection
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		// log.Println(err)
		return
	}

	// Create a new client with the authenticated user ID
	client := &Client{
		hub:  hub,
		Conn: conn,
		send: make(chan []byte, 256),
		ID:   claims.UserID,
	}

	// Register the client
	client.hub.register <- client

	// Start goroutines to handle reading and writing messages
	go client.writePump()
	go client.readPump()
}

// readPump reads messages from the WebSocket connection
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.Conn.Close()
		// log.Printf("Client %d disconnected", c.ID)
	}()

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				// log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// Parse the incoming message
		var msg models.WSMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			// log.Printf("Error unmarshaling message: %v", err)
			continue
		}

		// Validate the message
		if msg.SenderID != c.ID {
			// log.Printf("Unauthorized message sender: expected %d, got %d", c.ID, msg.SenderID)
			continue
		}

		// Check if a chat exists between the sender and receiver
		existingChatID, err := models.ChatExists(config.DB, msg.SenderID, msg.ReceiverID)
		if err != nil {
			// log.Printf("Error checking for existing chat: %v", err)
			continue
		}

		// If no chat exists, create one
		if existingChatID == 0 {
			existingChatID, err = models.CreateChat(config.DB, msg.SenderID, msg.ReceiverID)
			if err != nil {
				// log.Printf("Error creating chat: %v", err)
				continue
			}
			// log.Printf("Created new chat with ID: %d", existingChatID)
		}

		// Update the chat_id in the message
		msg.ChatID = existingChatID

		// Save the message to the database
		var repliedMessage string
		if msg.RepliedMessage != nil {
			repliedMessage = *msg.RepliedMessage
		}

		if err := models.AddMessage(config.DB, msg.ChatID, msg.SenderID, msg.ReceiverID, msg.Message, msg.Time, repliedMessage, msg.RepliedMessageID, msg.Date, msg.AttachedFile, msg.IsRead); err != nil {
			// log.Printf("Error saving message to database: %v", err)

			// Notify the client that the message could not be sent
			c.Conn.WriteJSON(map[string]string{
				"error":   "Failed to send message",
				"details": err.Error(),
			})
			continue
		}

		// Broadcast the message to all Clients
		c.hub.broadcast <- message
	}
}

// writePump writes messages to the WebSocket connection
func (c *Client) writePump() {
	defer func() {
		c.Conn.Close()
		// log.Printf("Client %d write pump closed", c.ID)
	}()

	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				// The hub closed the channel
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			// Write the message to the WebSocket connection
			if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				// log.Printf("Error writing message: %v", err)
				return
			}
		}
	}
}

func GetChatHistory(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(utils.UserClaimsKey).(*utils.Claims)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	receiverIDStr := r.URL.Query().Get("receiver_id")
	receiverID, err := strconv.Atoi(receiverIDStr)
	if err != nil {
		http.Error(w, "Invalid receiver ID", http.StatusBadRequest)
		return
	}

	chats, err := models.GetChatHistory(config.DB, claims.UserID, receiverID)
	if err != nil {
		http.Error(w, "Failed to get chat history", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(chats)
}

func CreateChat(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(utils.UserClaimsKey).(*utils.Claims)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var request struct {
		Participants []int `json:"participants"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if len(request.Participants) != 2 {
		http.Error(w, "Exactly two participants are required", http.StatusBadRequest)
		return
	}

	// Ensure the authenticated user is one of the participants
	if claims.UserID != request.Participants[0] && claims.UserID != request.Participants[1] {
		http.Error(w, "Unauthorized: You must be one of the participants", http.StatusForbidden)
		return
	}

	// Create the chat
	chatID, err := models.CreateChat(config.DB, request.Participants[0], request.Participants[1])
	if err != nil {
		// log.Printf("Error creating chat: %v", err)
		http.Error(w, "Failed to create chat", http.StatusInternalServerError)
		return
	}

	// Return the created chat
	response := map[string]interface{}{
		"id":           chatID,
		"participants": request.Participants,
		"messages":     []models.Message{},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetAllChats retrieves all chats for a user (doctor or patient)
func GetAllChats(w http.ResponseWriter, r *http.Request) {
	// Get the user ID from the JWT token
	claims, ok := r.Context().Value(utils.UserClaimsKey).(*utils.Claims)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Call the stored procedure to get all chats for the user
	chats, err := models.GetAllChats(config.DB, claims.UserID)
	if err != nil {
		http.Error(w, "Failed to get chats", http.StatusInternalServerError)
		return
	}

	// Return the chats as JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(chats)
}

func GetUnreadChats(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(utils.UserClaimsKey).(*utils.Claims)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Fetch unread chats from the database
	unreadChats, err := models.GetUnreadChats(config.DB, claims.UserID)
	if err != nil {
		// log.Printf("Failed to fetch unread chats for user %d: %v", claims.UserID, err)
		http.Error(w, "Failed to fetch unread chats", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(unreadChats)
}
