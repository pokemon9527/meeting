package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
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

type Room struct {
	MeetingID string
	Clients   map[string]*Client
	mu        sync.RWMutex
}

type Hub struct {
	Rooms      map[string]*Room
	Register   chan *Client
	Unregister chan *Client
	Redis      *redis.Client
	mu         sync.RWMutex
}

func NewHub(redisClient *redis.Client) *Hub {
	return &Hub{
		Rooms:      make(map[string]*Room),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		Redis:      redisClient,
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.handleRegister(client)
		case client := <-h.Unregister:
			h.handleUnregister(client)
		}
	}
}

func (h *Hub) handleRegister(client *Client) {
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

	log.Printf("Client %s (%s) joined room %s", client.Username, client.ID, client.MeetingID)

	// 发送加入响应
	peers := h.getRoomPeers(client.MeetingID)
	response, _ := json.Marshal(map[string]interface{}{
		"event": "join-room-response",
		"data": map[string]interface{}{
			"success": true,
			"peer_id": client.ID,
			"peers":   peers,
		},
	})
	client.Send <- response

	// 广播新参与者加入
	broadcast, _ := json.Marshal(map[string]interface{}{
		"event": "peer-joined",
		"data": map[string]interface{}{
			"peer_id":  client.ID,
			"user_id":  client.UserID,
			"username": client.Username,
			"avatar":   client.Avatar,
			"role":     client.Role,
		},
	})
	h.Broadcast(client.MeetingID, broadcast, client.ID)
}

func (h *Hub) handleUnregister(client *Client) {
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

	// 广播参与者离开
	broadcast, _ := json.Marshal(map[string]interface{}{
		"event": "peer-left",
		"data": map[string]interface{}{
			"peer_id": client.ID,
		},
	})
	h.Broadcast(client.MeetingID, broadcast, client.ID)
	log.Printf("Client %s left room %s", client.Username, client.MeetingID)
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

func (h *Hub) getRoomPeers(meetingID string) []map[string]interface{} {
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

func (h *Hub) GetClientCount(meetingID string) int {
	h.mu.RLock()
	room, exists := h.Rooms[meetingID]
	h.mu.RUnlock()

	if !exists {
		return 0
	}

	room.mu.RLock()
	defer room.mu.RUnlock()
	return len(room.Clients)
}

// 信令处理方法
func (c *Client) handleMessage(hub *Hub, msg WebSocketMessage) {
	log.Printf("Received message: %s from client %s", msg.Event, c.ID)
	switch msg.Event {
	case "join-room":
		c.handleJoinRoom(hub, msg.Data)
	case "leave-room":
		c.handleLeaveRoom(hub)
	case "toggle-audio":
		c.handleToggleAudio(hub, msg.Data)
	case "toggle-video":
		c.handleToggleVideo(hub, msg.Data)
	case "raise-hand":
		c.handleRaiseHand(hub, msg.Data)
	case "send-message":
		c.handleSendMessage(hub, msg.Data)
	case "mute-participant":
		c.handleMuteParticipant(hub, msg.Data)
	case "remove-participant":
		c.handleRemoveParticipant(hub, msg.Data)
	// WebRTC 信令
	case "create-transport":
		c.handleCreateTransport(hub, msg.Data)
	case "connect-transport":
		c.handleConnectTransport(hub, msg.Data)
	case "produce":
		c.handleProduce(hub, msg.Data)
	case "consume":
		c.handleConsume(hub, msg.Data)
	case "consumer-resume":
		c.handleConsumerResume(hub, msg.Data)
	case "close-producer":
		c.handleCloseProducer(hub, msg.Data)
	// P2P WebRTC
	case "offer":
		c.handleOffer(hub, msg.Data)
	case "answer":
		c.handleAnswer(hub, msg.Data)
	case "ice-candidate":
		c.handleIceCandidate(hub, msg.Data)
	}
}

func (c *Client) handleJoinRoom(hub *Hub, data json.RawMessage) {
	// 已在注册时处理
}

func (c *Client) handleLeaveRoom(hub *Hub) {
	broadcast, _ := json.Marshal(map[string]interface{}{
		"event": "peer-left",
		"data": map[string]interface{}{
			"peer_id": c.ID,
		},
	})
	hub.Broadcast(c.MeetingID, broadcast, c.ID)
}

func (c *Client) handleToggleAudio(hub *Hub, data json.RawMessage) {
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
	hub.Broadcast(c.MeetingID, broadcast, c.ID)
}

func (c *Client) handleToggleVideo(hub *Hub, data json.RawMessage) {
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
	hub.Broadcast(c.MeetingID, broadcast, c.ID)
}

func (c *Client) handleRaiseHand(hub *Hub, data json.RawMessage) {
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
	hub.Broadcast(c.MeetingID, broadcast, "")
}

func (c *Client) handleSendMessage(hub *Hub, data json.RawMessage) {
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
			"timestamp":     time.Now().Format(time.RFC3339),
			"is_private":    payload.ReceiverID != "",
		},
	})

	if payload.ReceiverID != "" {
		hub.SendToClient(c.MeetingID, payload.ReceiverID, message)
		c.Send <- message
	} else {
		hub.Broadcast(c.MeetingID, message, "")
	}
}

func (c *Client) handleMuteParticipant(hub *Hub, data json.RawMessage) {
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
		hub.SendToClient(c.MeetingID, payload.TargetPeerID, msg)
	}
}

func (c *Client) handleRemoveParticipant(hub *Hub, data json.RawMessage) {
	var payload struct {
		TargetPeerID string `json:"target_peer_id"`
	}
	json.Unmarshal(data, &payload)

	if c.Role == "host" {
		msg, _ := json.Marshal(map[string]interface{}{
			"event": "removed-from-meeting",
			"data":  map[string]interface{}{},
		})
		hub.SendToClient(c.MeetingID, payload.TargetPeerID, msg)
	}
}

// WebRTC 信令处理
func (c *Client) handleCreateTransport(hub *Hub, data json.RawMessage) {
	// 转发到 SFU 服务
	response, _ := json.Marshal(map[string]interface{}{
		"event": "create-transport-response",
		"data": map[string]interface{}{
			"success": true,
			"id":      fmt.Sprintf("transport_%d", time.Now().UnixNano()),
		},
	})
	c.Send <- response
}

func (c *Client) handleConnectTransport(hub *Hub, data json.RawMessage) {
	response, _ := json.Marshal(map[string]interface{}{
		"event": "connect-transport-response",
		"data": map[string]interface{}{
			"success": true,
		},
	})
	c.Send <- response
}

func (c *Client) handleProduce(hub *Hub, data json.RawMessage) {
	var payload struct {
		TransportID string `json:"transport_id"`
		Kind        string `json:"kind"`
	}
	json.Unmarshal(data, &payload)

	producerID := fmt.Sprintf("producer_%d", time.Now().UnixNano())

	// 通知其他参与者有新的生产者
	broadcast, _ := json.Marshal(map[string]interface{}{
		"event": "new-producer",
		"data": map[string]interface{}{
			"producer_id": producerID,
			"peer_id":     c.ID,
			"kind":        payload.Kind,
		},
	})
	hub.Broadcast(c.MeetingID, broadcast, c.ID)

	response, _ := json.Marshal(map[string]interface{}{
		"event": "produce-response",
		"data": map[string]interface{}{
			"success": true,
			"id":      producerID,
		},
	})
	c.Send <- response
}

func (c *Client) handleConsume(hub *Hub, data json.RawMessage) {
	response, _ := json.Marshal(map[string]interface{}{
		"event": "consume-response",
		"data": map[string]interface{}{
			"success":        true,
			"id":             fmt.Sprintf("consumer_%d", time.Now().UnixNano()),
			"producer_id":    "",
			"kind":           "video",
			"rtp_parameters": map[string]interface{}{},
		},
	})
	c.Send <- response
}

func (c *Client) handleConsumerResume(hub *Hub, data json.RawMessage) {
	response, _ := json.Marshal(map[string]interface{}{
		"event": "consumer-resume-response",
		"data": map[string]interface{}{
			"success": true,
		},
	})
	c.Send <- response
}

func (c *Client) handleCloseProducer(hub *Hub, data json.RawMessage) {
	var payload struct {
		ProducerID string `json:"producer_id"`
	}
	json.Unmarshal(data, &payload)

	broadcast, _ := json.Marshal(map[string]interface{}{
		"event": "producer-closed",
		"data": map[string]interface{}{
			"producer_id": payload.ProducerID,
			"peer_id":     c.ID,
		},
	})
	hub.Broadcast(c.MeetingID, broadcast, c.ID)
}

// P2P WebRTC 信令处理
func (c *Client) handleOffer(hub *Hub, data json.RawMessage) {
	var payload struct {
		TargetPeerID string          `json:"target_peer_id"`
		SDP          json.RawMessage `json:"sdp"`
		Type         string          `json:"type"`
	}
	json.Unmarshal(data, &payload)
	log.Printf("handleOffer: from=%s, to=%s", c.ID, payload.TargetPeerID)

	msg, _ := json.Marshal(map[string]interface{}{
		"event": "offer",
		"data": map[string]interface{}{
			"from_peer_id": c.ID,
			"sdp":          payload.SDP,
			"type":         payload.Type,
		},
	})
	hub.SendToClient(c.MeetingID, payload.TargetPeerID, msg)
}

func (c *Client) handleAnswer(hub *Hub, data json.RawMessage) {
	var payload struct {
		TargetPeerID string          `json:"target_peer_id"`
		SDP          json.RawMessage `json:"sdp"`
		Type         string          `json:"type"`
	}
	json.Unmarshal(data, &payload)
	log.Printf("handleAnswer: from=%s, to=%s", c.ID, payload.TargetPeerID)

	msg, _ := json.Marshal(map[string]interface{}{
		"event": "answer",
		"data": map[string]interface{}{
			"from_peer_id": c.ID,
			"sdp":          payload.SDP,
			"type":         payload.Type,
		},
	})
	hub.SendToClient(c.MeetingID, payload.TargetPeerID, msg)
}

func (c *Client) handleIceCandidate(hub *Hub, data json.RawMessage) {
	var payload struct {
		TargetPeerID string      `json:"target_peer_id"`
		Candidate    interface{} `json:"candidate"`
	}
	json.Unmarshal(data, &payload)

	msg, _ := json.Marshal(map[string]interface{}{
		"event": "ice-candidate",
		"data": map[string]interface{}{
			"from_peer_id": c.ID,
			"candidate":    payload.Candidate,
		},
	})
	hub.SendToClient(c.MeetingID, payload.TargetPeerID, msg)
}

func (c *Client) readPump(hub *Hub) {
	defer func() {
		hub.Unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			break
		}

		var msg WebSocketMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			continue
		}

		c.handleMessage(hub, msg)
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func main() {
	// 连接 Redis
	redisAddr := getEnv("REDIS_ADDR", "localhost:6379")
	rdb := redis.NewClient(&redis.Options{
		Addr: redisAddr,
		DB:   0,
	})

	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Printf("Warning: Redis connection failed: %v", err)
	} else {
		fmt.Println("Redis connected")
	}

	hub := NewHub(rdb)
	go hub.Run()

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "ok",
			"rooms":  len(hub.Rooms),
		})
	})

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
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

		hub.Register <- client

		go client.writePump()
		go client.readPump(hub)
	})

	port := getEnv("SIGNALING_PORT", "8081")
	fmt.Printf("Signaling server starting on port %s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
