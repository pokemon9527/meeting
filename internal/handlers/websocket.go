package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type WebSocketMessage struct {
	Event string          `json:"event"`
	Data  json.RawMessage `json:"data"`
}

type Client struct {
	ID        string
	UserID    string
	Username  string
	Avatar    string
	MeetingID string
	Role      string
	Conn      *websocket.Conn
	Send      chan []byte
}

type Hub struct {
	Rooms      map[string]*Room
	Register   chan *Client
	Unregister chan *Client
	mu         sync.RWMutex
}

type Room struct {
	MeetingID string
	Clients   map[string]*Client
	mu        sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{
		Rooms:      make(map[string]*Room),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.mu.Lock()
			room, exists := h.Rooms[client.MeetingID]
			if !exists {
				room = &Room{
					MeetingID: client.MeetingID,
					Clients:   make(map[string]*Client),
				}
				h.Rooms[client.MeetingID] = room
			}
			room.mu.Lock()
			room.Clients[client.ID] = client
			room.mu.Unlock()
			h.mu.Unlock()
			log.Printf("Client %s joined room %s", client.Username, client.MeetingID)

		case client := <-h.Unregister:
			h.mu.Lock()
			if room, exists := h.Rooms[client.MeetingID]; exists {
				room.mu.Lock()
				if _, ok := room.Clients[client.ID]; ok {
					delete(room.Clients, client.ID)
					close(client.Send)
					if len(room.Clients) == 0 {
						delete(h.Rooms, client.MeetingID)
					}
				}
				room.mu.Unlock()
			}
			h.mu.Unlock()
			log.Printf("Client %s left room %s", client.Username, client.MeetingID)
		}
	}
}

func (h *Hub) Broadcast(meetingID string, message []byte, excludeID string) {
	h.mu.RLock()
	room, exists := h.Rooms[meetingID]
	h.mu.RUnlock()

	if !exists {
		return
	}

	room.mu.RLock()
	defer room.mu.RUnlock()

	for id, client := range room.Clients {
		if id != excludeID {
			select {
			case client.Send <- message:
			default:
				close(client.Send)
				delete(room.Clients, id)
			}
		}
	}
}

func (h *Hub) SendToClient(meetingID, clientID string, message []byte) {
	h.mu.RLock()
	room, exists := h.Rooms[meetingID]
	h.mu.RUnlock()

	if !exists {
		return
	}

	room.mu.RLock()
	defer room.mu.RUnlock()

	if client, ok := room.Clients[clientID]; ok {
		select {
		case client.Send <- message:
		default:
			close(client.Send)
			delete(room.Clients, clientID)
		}
	}
}

func (h *Hub) GetRoomPeers(meetingID string) []map[string]interface{} {
	h.mu.RLock()
	room, exists := h.Rooms[meetingID]
	h.mu.RUnlock()

	if !exists {
		return []map[string]interface{}{}
	}

	room.mu.RLock()
	defer room.mu.RUnlock()

	peers := make([]map[string]interface{}, 0, len(room.Clients))
	for _, client := range room.Clients {
		peers = append(peers, map[string]interface{}{
			"id":       client.ID,
			"user_id":  client.UserID,
			"username": client.Username,
			"avatar":   client.Avatar,
			"role":     client.Role,
		})
	}
	return peers
}

var GlobalHub *Hub

func init() {
	GlobalHub = NewHub()
	go GlobalHub.Run()
}

func HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket upgrade error:", err)
		return
	}

	client := &Client{
		ID:        fmt.Sprintf("%s_%d", r.URL.Query().Get("user_id"), time.Now().UnixNano()),
		UserID:    r.URL.Query().Get("user_id"),
		Username:  r.URL.Query().Get("username"),
		Avatar:    r.URL.Query().Get("avatar"),
		MeetingID: r.URL.Query().Get("meeting_id"),
		Role:      "participant",
		Conn:      conn,
		Send:      make(chan []byte, 256),
	}

	GlobalHub.Register <- client

	go client.writePump()
	go client.readPump()
}

func (c *Client) readPump() {
	defer func() {
		GlobalHub.Unregister <- c
		c.Conn.Close()
	}()

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			break
		}

		var msg WebSocketMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			continue
		}

		c.handleMessage(msg)
	}
}

func (c *Client) writePump() {
	defer c.Conn.Close()

	for message := range c.Send {
		if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
			break
		}
	}
}

func (c *Client) handleMessage(msg WebSocketMessage) {
	switch msg.Event {
	case "join-room":
		c.handleJoinRoom(msg.Data)
	case "leave-room":
		c.handleLeaveRoom()
	case "toggle-audio":
		c.handleToggleAudio(msg.Data)
	case "toggle-video":
		c.handleToggleVideo(msg.Data)
	case "raise-hand":
		c.handleRaiseHand(msg.Data)
	case "send-message":
		c.handleSendMessage(msg.Data)
	case "mute-participant":
		c.handleMuteParticipant(msg.Data)
	case "remove-participant":
		c.handleRemoveParticipant(msg.Data)
	}
}

func (c *Client) handleJoinRoom(data json.RawMessage) {
	if c.Role == "" {
		c.Role = "participant"
	}

	peers := GlobalHub.GetRoomPeers(c.MeetingID)
	response, _ := json.Marshal(map[string]interface{}{
		"event": "join-room-response",
		"data": map[string]interface{}{
			"success": true,
			"peer_id": c.ID,
			"peers":   peers,
		},
	})
	c.Send <- response

	broadcast, _ := json.Marshal(map[string]interface{}{
		"event": "peer-joined",
		"data": map[string]interface{}{
			"peer_id":  c.ID,
			"user_id":  c.UserID,
			"username": c.Username,
			"avatar":   c.Avatar,
			"role":     c.Role,
		},
	})
	GlobalHub.Broadcast(c.MeetingID, broadcast, c.ID)
}

func (c *Client) handleLeaveRoom() {
	broadcast, _ := json.Marshal(map[string]interface{}{
		"event": "peer-left",
		"data": map[string]interface{}{
			"peer_id": c.ID,
		},
	})
	GlobalHub.Broadcast(c.MeetingID, broadcast, c.ID)
}

func (c *Client) handleToggleAudio(data json.RawMessage) {
	var payload struct {
		Enabled bool `json:"enabled"`
	}
	json.Unmarshal(data, &payload)

	broadcast, _ := json.Marshal(map[string]interface{}{
		"event": "peer-updated",
		"data": map[string]interface{}{
			"peer_id":       c.ID,
			"audio_enabled": payload.Enabled,
		},
	})
	GlobalHub.Broadcast(c.MeetingID, broadcast, c.ID)
}

func (c *Client) handleToggleVideo(data json.RawMessage) {
	var payload struct {
		Enabled bool `json:"enabled"`
	}
	json.Unmarshal(data, &payload)

	broadcast, _ := json.Marshal(map[string]interface{}{
		"event": "peer-updated",
		"data": map[string]interface{}{
			"peer_id":       c.ID,
			"video_enabled": payload.Enabled,
		},
	})
	GlobalHub.Broadcast(c.MeetingID, broadcast, c.ID)
}

func (c *Client) handleRaiseHand(data json.RawMessage) {
	var payload struct {
		Raised bool `json:"raised"`
	}
	json.Unmarshal(data, &payload)

	broadcast, _ := json.Marshal(map[string]interface{}{
		"event": "hand-raised",
		"data": map[string]interface{}{
			"peer_id": c.ID,
			"raised":  payload.Raised,
		},
	})
	GlobalHub.Broadcast(c.MeetingID, broadcast, "")
}

func (c *Client) handleSendMessage(data json.RawMessage) {
	var payload struct {
		Content    string `json:"content"`
		Type       string `json:"type"`
		ReceiverID string `json:"receiver_id"`
	}
	json.Unmarshal(data, &payload)

	msgType := payload.Type
	if msgType == "" {
		msgType = "text"
	}

	message, _ := json.Marshal(map[string]interface{}{
		"event": "new-message",
		"data": map[string]interface{}{
			"id":            fmt.Sprintf("%d", time.Now().UnixNano()),
			"sender_id":     c.UserID,
			"sender_name":   c.Username,
			"sender_avatar": c.Avatar,
			"content":       payload.Content,
			"type":          msgType,
			"timestamp":     time.Now(),
			"is_private":    payload.ReceiverID != "",
		},
	})

	if payload.ReceiverID != "" {
		GlobalHub.SendToClient(c.MeetingID, payload.ReceiverID, message)
		c.Send <- message
	} else {
		GlobalHub.Broadcast(c.MeetingID, message, "")
	}
}

func (c *Client) handleMuteParticipant(data json.RawMessage) {
	var payload struct {
		TargetPeerID string `json:"target_peer_id"`
	}
	json.Unmarshal(data, &payload)

	if c.Role == "host" || c.Role == "cohost" {
		msg, _ := json.Marshal(map[string]interface{}{
			"event": "peer-muted",
			"data": map[string]interface{}{
				"peer_id": payload.TargetPeerID,
			},
		})
		GlobalHub.SendToClient(c.MeetingID, payload.TargetPeerID, msg)
	}
}

func (c *Client) handleRemoveParticipant(data json.RawMessage) {
	var payload struct {
		TargetPeerID string `json:"target_peer_id"`
	}
	json.Unmarshal(data, &payload)

	if c.Role == "host" {
		msg, _ := json.Marshal(map[string]interface{}{
			"event": "removed-from-meeting",
			"data":  map[string]interface{}{},
		})
		GlobalHub.SendToClient(c.MeetingID, payload.TargetPeerID, msg)
	}
}
