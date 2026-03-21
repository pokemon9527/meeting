package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/pion/webrtc/v3"
)

type SFUServer struct {
	rooms map[string]*SFURoom
	mu    sync.RWMutex
}

type SFURoom struct {
	MeetingID string
	Peers     map[string]*SFUPeer
	mu        sync.RWMutex
}

type SFUPeer struct {
	ID             string
	PeerConnection *webrtc.PeerConnection
	Producers      map[string]*webrtc.TrackLocalStaticRTP
	Consumers      map[string]*webrtc.TrackRemote
	mu             sync.RWMutex
}

func NewSFUServer() *SFUServer {
	return &SFUServer{
		rooms: make(map[string]*SFURoom),
	}
}

func (s *SFUServer) GetOrCreateRoom(meetingID string) *SFURoom {
	s.mu.Lock()
	defer s.mu.Unlock()

	if room, exists := s.rooms[meetingID]; exists {
		return room
	}

	room := &SFURoom{
		MeetingID: meetingID,
		Peers:     make(map[string]*SFUPeer),
	}
	s.rooms[meetingID] = room
	return room
}

func (s *SFUServer) RemoveRoom(meetingID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if room, exists := s.rooms[meetingID]; exists {
		room.mu.Lock()
		for _, peer := range room.Peers {
			if peer.PeerConnection != nil {
				peer.PeerConnection.Close()
			}
		}
		room.mu.Unlock()
		delete(s.rooms, meetingID)
	}
}

type CreateRoomRequest struct {
	MeetingID string `json:"meeting_id"`
	PeerID    string `json:"peer_id"`
}

func createMediaEngine() *webrtc.MediaEngine {
	m := &webrtc.MediaEngine{}
	m.RegisterCodec(webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{
			MimeType: webrtc.MimeTypeOpus, ClockRate: 48000, Channels: 2,
		},
		PayloadType: 111,
	}, webrtc.RTPCodecTypeAudio)
	m.RegisterCodec(webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{
			MimeType: webrtc.MimeTypeVP8, ClockRate: 90000,
		},
		PayloadType: 96,
	}, webrtc.RTPCodecTypeVideo)
	m.RegisterCodec(webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{
			MimeType: webrtc.MimeTypeVP9, ClockRate: 90000,
		},
		PayloadType: 98,
	}, webrtc.RTPCodecTypeVideo)
	return m
}

func main() {
	sfu := NewSFUServer()

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		sfu.mu.RLock()
		roomCount := len(sfu.rooms)
		sfu.mu.RUnlock()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "ok",
			"rooms":  roomCount,
		})
	})

	http.HandleFunc("/api/rooms/create", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req CreateRoomRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		room := sfu.GetOrCreateRoom(req.MeetingID)

		m := createMediaEngine()
		api := webrtc.NewAPI(webrtc.WithMediaEngine(m))

		config := webrtc.Configuration{
			ICEServers: []webrtc.ICEServer{
				{URLs: []string{"stun:stun.l.google.com:19302"}},
			},
		}

		pc, err := api.NewPeerConnection(config)
		if err != nil {
			http.Error(w, "Failed to create peer connection", http.StatusInternalServerError)
			return
		}

		peer := &SFUPeer{
			ID:             req.PeerID,
			PeerConnection: pc,
			Producers:      make(map[string]*webrtc.TrackLocalStaticRTP),
			Consumers:      make(map[string]*webrtc.TrackRemote),
		}

		room.mu.Lock()
		room.Peers[req.PeerID] = peer
		room.mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"room_id": room.MeetingID,
		})
	})

	http.HandleFunc("/api/rooms/join", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req CreateRoomRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		room := sfu.GetOrCreateRoom(req.MeetingID)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"room_id": room.MeetingID,
		})
	})

	http.HandleFunc("/api/rooms/leave", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req CreateRoomRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		sfu.mu.RLock()
		room, exists := sfu.rooms[req.MeetingID]
		sfu.mu.RUnlock()

		if exists {
			room.mu.Lock()
			if peer, ok := room.Peers[req.PeerID]; ok {
				if peer.PeerConnection != nil {
					peer.PeerConnection.Close()
				}
				delete(room.Peers, req.PeerID)
			}
			room.mu.Unlock()

			if len(room.Peers) == 0 {
				sfu.RemoveRoom(req.MeetingID)
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
		})
	})

	http.HandleFunc("/api/rooms/producers", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			MeetingID string `json:"meeting_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		sfu.mu.RLock()
		room, exists := sfu.rooms[req.MeetingID]
		sfu.mu.RUnlock()

		producers := []map[string]interface{}{}
		if exists {
			room.mu.RLock()
			for _, peer := range room.Peers {
				peer.mu.RLock()
				for id := range peer.Producers {
					producers = append(producers, map[string]interface{}{
						"id":      id,
						"peer_id": peer.ID,
					})
				}
				peer.mu.RUnlock()
			}
			room.mu.RUnlock()
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"producers": producers,
		})
	})

	port := "8082"
	fmt.Printf("SFU server starting on port %s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
