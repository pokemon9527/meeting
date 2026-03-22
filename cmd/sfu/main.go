package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gorilla/websocket"
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
	ID                string
	Username          string
	Avatar            string
	PC                *webrtc.PeerConnection
	Producers         map[string]*webrtc.TrackRemote
	ScreenTrackID     string
	IsScreenSharing   bool
	Conn              *websocket.Conn
	PendingCandidates []webrtc.ICECandidateInit
	NegotiationQueued bool
	mu                sync.RWMutex
}

type WebSocketMessage struct {
	Event string          `json:"event"`
	Data  json.RawMessage `json:"data"`
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
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
			if peer.PC != nil {
				peer.PC.Close()
			}
		}
		room.mu.Unlock()
		delete(s.rooms, meetingID)
	}
}

func (r *SFURoom) Broadcast(exceptID string, message []byte) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, peer := range r.Peers {
		if peer.ID != exceptID && peer.Conn != nil {
			peer.Conn.WriteMessage(websocket.TextMessage, message)
		}
	}
}

func (r *SFURoom) SendToPeer(peerID string, message []byte) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if peer, ok := r.Peers[peerID]; ok && peer.Conn != nil {
		peer.Conn.WriteMessage(websocket.TextMessage, message)
	}
}

func (r *SFURoom) AddTrackToAllPeers(track *webrtc.TrackRemote, senderID string, isScreenShare ...bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	screenShare := false
	if len(isScreenShare) > 0 {
		screenShare = isScreenShare[0]
	}

	trackKind := track.Kind().String()
	if screenShare {
		trackKind = "screen"
	}

	for _, peer := range r.Peers {
		if peer.ID != senderID && peer.PC != nil {
			localTrack, err := webrtc.NewTrackLocalStaticRTP(
				track.Codec().RTPCodecCapability,
				track.ID(),
				track.StreamID(),
			)
			if err != nil {
				log.Printf("Failed to create local track: %v", err)
				continue
			}

			_, err = peer.PC.AddTrack(localTrack)
			if err != nil {
				log.Printf("Failed to add track to peer %s: %v", peer.ID, err)
				continue
			}

			log.Printf("Added track %s (kind: %s) from peer %s to peer %s", track.ID(), trackKind, senderID, peer.ID)

			// Forward RTP packets in background
			go func(p *SFUPeer, lt *webrtc.TrackLocalStaticRTP) {
				rtpBuf := make([]byte, 1500)
				for {
					i, _, readErr := track.Read(rtpBuf)
					if readErr != nil {
						return
					}
					_, writeErr := lt.Write(rtpBuf[:i])
					if writeErr != nil {
						return
					}
				}
			}(peer, localTrack)

			// Notify peer about new producer
			notify, _ := json.Marshal(map[string]interface{}{
				"event": "new-producer",
				"data": map[string]interface{}{
					"producer_id": track.ID(),
					"peer_id":     senderID,
					"kind":        trackKind,
					"is_screen":   screenShare,
				},
			})
			if peer.Conn != nil {
				peer.Conn.WriteMessage(websocket.TextMessage, notify)
			}
		}
	}
}

func (r *SFURoom) GetExistingTracks(exceptPeerID string) []map[string]string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tracks := []map[string]string{}
	for peerID, peer := range r.Peers {
		if peerID != exceptPeerID {
			peer.mu.RLock()
			for trackID, track := range peer.Producers {
				tracks = append(tracks, map[string]string{
					"track_id": trackID,
					"peer_id":  peerID,
					"kind":     track.Kind().String(),
				})
			}
			peer.mu.RUnlock()
		}
	}
	return tracks
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

func handleWebSocket(s *SFUServer, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	var peerID string
	var room *SFURoom

	m := createMediaEngine()

	// Configure ICE with public IP if specified
	settingEngine := webrtc.SettingEngine{}
	if publicIP := os.Getenv("PUBLIC_IP"); publicIP != "" && publicIP != "127.0.0.1" {
		log.Printf("Using public IP: %s", publicIP)
		settingEngine.SetNAT1To1IPs([]string{publicIP}, webrtc.ICECandidateTypeHost)
	}
	settingEngine.SetNetworkTypes([]webrtc.NetworkType{webrtc.NetworkTypeUDP4})

	api := webrtc.NewAPI(webrtc.WithMediaEngine(m), webrtc.WithSettingEngine(settingEngine))

	// Use STUN for NAT traversal
	publicIP := os.Getenv("PUBLIC_IP")
	turnUsername := os.Getenv("TURN_USERNAME")
	turnPassword := os.Getenv("TURN_PASSWORD")

	iceServers := []webrtc.ICEServer{
		{URLs: []string{"stun:stun.l.google.com:19302"}},
	}

	// Add local TURN server if configured
	if turnUsername != "" && turnPassword != "" && publicIP != "" && publicIP != "127.0.0.1" {
		iceServers = append(iceServers, webrtc.ICEServer{
			URLs:       []string{fmt.Sprintf("turn:%s:3478", publicIP)},
			Username:   turnUsername,
			Credential: turnPassword,
		})
		log.Printf("Added TURN server: turn:%s:3478", publicIP)
	}

	config := webrtc.Configuration{
		ICEServers: iceServers,
	}

	pc, err := api.NewPeerConnection(config)
	if err != nil {
		log.Printf("Failed to create peer connection: %v", err)
		return
	}
	defer pc.Close()

	// Function to perform renegotiation
	var doRenegotiation func(senderID string)
	doRenegotiation = func(senderID string) {
		if pc == nil {
			return
		}

		if pc.SignalingState() != webrtc.SignalingStateStable {
			log.Printf("Still not stable for renegotiation - signaling state is %s, re-queuing", pc.SignalingState().String())
			// Re-queue the negotiation request
			if currentPeer, exists := room.Peers[peerID]; exists {
				currentPeer.mu.Lock()
				currentPeer.NegotiationQueued = true
				currentPeer.mu.Unlock()
			}
			return
		}

		log.Printf("Performing renegotiation for peer %s", peerID)

		offer, err := pc.CreateOffer(nil)
		if err != nil {
			log.Printf("CreateOffer failed during renegotiation: %v", err)
			return
		}

		err = pc.SetLocalDescription(offer)
		if err != nil {
			log.Printf("SetLocalDescription failed: %v", err)
			return
		}

		// Wait for ICE gathering
		<-webrtc.GatheringCompletePromise(pc)

		resp, _ := json.Marshal(map[string]interface{}{
			"event": "offer",
			"data": map[string]interface{}{
				"sdp":     pc.LocalDescription().SDP,
				"type":    pc.LocalDescription().Type.String(),
				"peer_id": senderID,
			},
		})
		conn.WriteMessage(websocket.TextMessage, resp)
		log.Printf("Sent renegotiation offer to peer %s", peerID)
	}

	// Handle negotiation needed (when tracks are added)
	pc.OnNegotiationNeeded(func() {
		if pc.SignalingState() != webrtc.SignalingStateStable {
			log.Printf("Queuing negotiation for peer %s - signaling state is %s", peerID, pc.SignalingState().String())
			if currentPeer, exists := room.Peers[peerID]; exists {
				currentPeer.mu.Lock()
				currentPeer.NegotiationQueued = true
				currentPeer.mu.Unlock()
			}
			return
		}

		log.Printf("Negotiation needed for peer %s, performing immediately", peerID)
		doRenegotiation("")
	})

	// Handle ICE candidates - send to client
	pc.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate != nil && conn != nil {
			resp, _ := json.Marshal(map[string]interface{}{
				"event": "ice-candidate",
				"data": map[string]interface{}{
					"candidate": candidate.ToJSON(),
				},
			})
			conn.WriteMessage(websocket.TextMessage, resp)
			log.Printf("Sent ICE candidate to peer %s", peerID)
		}
	})

	// Handle incoming tracks
	pc.OnTrack(func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		log.Printf("Received track %s from peer %s (kind: %s, streamID: %s)", track.ID(), peerID, track.Kind(), track.StreamID())

		if room == nil {
			return
		}

		isScreenShare := false
		if track.Kind() == webrtc.RTPCodecTypeVideo && room != nil {
			room.mu.RLock()
			if p, exists := room.Peers[peerID]; exists && p.IsScreenSharing {
				isScreenShare = true
			}
			room.mu.RUnlock()
		}

		// Store producer
		room.mu.Lock()
		if peer, exists := room.Peers[peerID]; exists {
			peer.mu.Lock()
			peer.Producers[track.ID()] = track
			if isScreenShare {
				peer.ScreenTrackID = track.ID()
				peer.IsScreenSharing = true
			}
			peer.mu.Unlock()
		}
		room.mu.Unlock()

		// Add track to all other peers
		room.AddTrackToAllPeers(track, peerID, isScreenShare)

		// Read and forward RTP packets
		rtpBuf := make([]byte, 1500)
		for {
			i, _, readErr := track.Read(rtpBuf)
			if readErr != nil {
				log.Printf("Track %s read error: %v", track.ID(), readErr)
				break
			}
			_ = i
		}

		// Clean up when track ends
		room.mu.Lock()
		if peer, exists := room.Peers[peerID]; exists {
			peer.mu.Lock()
			delete(peer.Producers, track.ID())
			if peer.ScreenTrackID == track.ID() {
				peer.ScreenTrackID = ""
				peer.IsScreenSharing = false
			}
			peer.mu.Unlock()
		}
		room.mu.Unlock()

		// Notify about track removal
		notify, _ := json.Marshal(map[string]interface{}{
			"event": "track-removed",
			"data": map[string]interface{}{
				"track_id":  track.ID(),
				"peer_id":   peerID,
				"is_screen": isScreenShare,
			},
		})
		room.Broadcast("", notify)
	})

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Printf("WebSocket read error: %v", err)
			break
		}

		var msg WebSocketMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Printf("JSON parse error: %v", err)
			continue
		}

		log.Printf("Received message: %s", msg.Event)

		switch msg.Event {
		case "join":
			var data struct {
				MeetingID string `json:"meeting_id"`
				PeerID    string `json:"peer_id"`
				Username  string `json:"username"`
				Avatar    string `json:"avatar"`
			}
			json.Unmarshal(msg.Data, &data)

			peerID = data.PeerID
			room = s.GetOrCreateRoom(data.MeetingID)

			peer := &SFUPeer{
				ID:        peerID,
				Username:  data.Username,
				Avatar:    data.Avatar,
				PC:        pc,
				Producers: make(map[string]*webrtc.TrackRemote),
				Conn:      conn,
			}

			room.mu.Lock()
			room.Peers[peerID] = peer
			room.mu.Unlock()

			log.Printf("Peer %s joined room %s", peerID, data.MeetingID)

			// Get existing producers
			producers := []map[string]string{}
			room.mu.RLock()
			for id, p := range room.Peers {
				if id != peerID {
					p.mu.RLock()
					for trackID, track := range p.Producers {
						producers = append(producers, map[string]string{
							"producer_id": trackID,
							"peer_id":     id,
							"kind":        track.Kind().String(),
						})
					}
					p.mu.RUnlock()
				}
			}
			room.mu.RUnlock()

			resp, _ := json.Marshal(map[string]interface{}{
				"event": "join-response",
				"data": map[string]interface{}{
					"success":   true,
					"peer_id":   peerID,
					"producers": producers,
				},
			})
			conn.WriteMessage(websocket.TextMessage, resp)

			// Add existing tracks to the new peer FIRST
			room.mu.RLock()
			for _, existingPeer := range room.Peers {
				if existingPeer.ID == peerID {
					continue
				}
				existingPeer.mu.RLock()
				for _, track := range existingPeer.Producers {
					localTrack, err := webrtc.NewTrackLocalStaticRTP(
						track.Codec().RTPCodecCapability,
						track.ID(),
						track.StreamID(),
					)
					if err != nil {
						existingPeer.mu.RUnlock()
						continue
					}

					_, err = pc.AddTrack(localTrack)
					if err != nil {
						existingPeer.mu.RUnlock()
						continue
					}

					log.Printf("Added existing track %s from peer %s to new peer %s", track.ID(), existingPeer.ID, peerID)

					go func(t *webrtc.TrackRemote, lt *webrtc.TrackLocalStaticRTP) {
						rtpBuf := make([]byte, 1500)
						for {
							i, _, readErr := t.Read(rtpBuf)
							if readErr != nil {
								return
							}
							_, writeErr := lt.Write(rtpBuf[:i])
							if writeErr != nil {
								return
							}
						}
					}(track, localTrack)
				}
				existingPeer.mu.RUnlock()
			}
			room.mu.RUnlock()

			// Notify other peers about this new peer AFTER adding existing tracks
			notify, _ := json.Marshal(map[string]interface{}{
				"event": "peer-joined",
				"data": map[string]interface{}{
					"peer_id": peerID,
				},
			})
			room.Broadcast(peerID, notify)

		case "offer":
			var data struct {
				SDP  string `json:"sdp"`
				Type string `json:"type"`
			}
			json.Unmarshal(msg.Data, &data)

			log.Printf("Processing offer from peer %s, SDP length: %d", peerID, len(data.SDP))

			err := pc.SetRemoteDescription(webrtc.SessionDescription{
				Type: webrtc.SDPTypeOffer,
				SDP:  data.SDP,
			})
			if err != nil {
				log.Printf("SetRemoteDescription failed: %v", err)
				continue
			}

			// Add pending ICE candidates
			if currentPeer, exists := room.Peers[peerID]; exists {
				currentPeer.mu.Lock()
				for _, candidate := range currentPeer.PendingCandidates {
					err := pc.AddICECandidate(candidate)
					if err != nil {
						log.Printf("AddICECandidate failed: %v", err)
					}
				}
				currentPeer.PendingCandidates = nil
				currentPeer.mu.Unlock()
			}

			// Create answer
			answer, err := pc.CreateAnswer(nil)
			if err != nil {
				log.Printf("CreateAnswer failed: %v", err)
				continue
			}

			gatherComplete := webrtc.GatheringCompletePromise(pc)

			err = pc.SetLocalDescription(answer)
			if err != nil {
				log.Printf("SetLocalDescription failed: %v", err)
				continue
			}

			<-gatherComplete
			log.Printf("ICE gathering complete for peer %s", peerID)

			resp, _ := json.Marshal(map[string]interface{}{
				"event": "answer",
				"data": map[string]interface{}{
					"sdp":  pc.LocalDescription().SDP,
					"type": pc.LocalDescription().Type.String(),
				},
			})
			conn.WriteMessage(websocket.TextMessage, resp)
			log.Printf("Answer sent to peer %s, signaling state: %s", peerID, pc.SignalingState().String())

			// Check if there's a queued negotiation request
			if currentPeer, exists := room.Peers[peerID]; exists {
				currentPeer.mu.Lock()
				if currentPeer.NegotiationQueued {
					currentPeer.NegotiationQueued = false
					currentPeer.mu.Unlock()
					log.Printf("Processing queued negotiation for peer %s", peerID)
					go doRenegotiation("")
				} else {
					currentPeer.mu.Unlock()
				}
			}

		case "ice-candidate":
			var data struct {
				Candidate webrtc.ICECandidateInit `json:"candidate"`
			}
			json.Unmarshal(msg.Data, &data)

			if pc.RemoteDescription() == nil {
				// Queue candidate for later
				if currentPeer, exists := room.Peers[peerID]; exists {
					currentPeer.mu.Lock()
					currentPeer.PendingCandidates = append(currentPeer.PendingCandidates, data.Candidate)
					log.Printf("Queued ICE candidate for peer %s", peerID)
					currentPeer.mu.Unlock()
				}
			} else {
				// Add candidate immediately
				err := pc.AddICECandidate(data.Candidate)
				if err != nil {
					log.Printf("AddICECandidate failed: %v", err)
				}
			}

		case "send-message":
			var data struct {
				Content string `json:"content"`
				Type    string `json:"type"`
			}
			json.Unmarshal(msg.Data, &data)

			room.mu.RLock()
			senderPeer := room.Peers[peerID]
			room.mu.RUnlock()

			username := ""
			avatar := ""
			if senderPeer != nil {
				username = senderPeer.Username
				avatar = senderPeer.Avatar
			}

			notify, _ := json.Marshal(map[string]interface{}{
				"event": "new-message",
				"data": map[string]interface{}{
					"id":        fmt.Sprintf("%d", time.Now().UnixNano()),
					"peer_id":   peerID,
					"username":  username,
					"avatar":    avatar,
					"content":   data.Content,
					"type":      data.Type,
					"timestamp": time.Now().Format(time.RFC3339),
				},
			})
			room.Broadcast(peerID, notify)

		case "toggle-screen-share":
			var data struct {
				Enabled        bool   `json:"enabled"`
				ScreenStreamID string `json:"screen_stream_id"`
			}
			json.Unmarshal(msg.Data, &data)

			if room != nil {
				room.mu.Lock()
				if peer, exists := room.Peers[peerID]; exists {
					peer.IsScreenSharing = data.Enabled
					if !data.Enabled {
						peer.ScreenTrackID = ""
					}
					log.Printf("Peer %s screen sharing: %v", peerID, data.Enabled)
				}
				room.mu.Unlock()

				room.mu.RLock()
				peer := room.Peers[peerID]
				room.mu.RUnlock()

				username := ""
				if peer != nil {
					username = peer.Username
				}

				broadcast, _ := json.Marshal(map[string]interface{}{
					"event": "peer-screen-share",
					"data": map[string]interface{}{
						"peer_id":           peerID,
						"username":          username,
						"is_screen_sharing": data.Enabled,
						"screen_stream_id":  data.ScreenStreamID,
					},
				})
				room.Broadcast(peerID, broadcast)
			}

		case "leave":
			if room != nil {
				room.mu.Lock()
				if peer, exists := room.Peers[peerID]; exists {
					if peer.PC != nil {
						peer.PC.Close()
					}
					delete(room.Peers, peerID)
				}
				room.mu.Unlock()

				if len(room.Peers) == 0 {
					s.RemoveRoom(room.MeetingID)
				}

				notify, _ := json.Marshal(map[string]interface{}{
					"event": "peer-left",
					"data": map[string]interface{}{
						"peer_id": peerID,
					},
				})
				room.Broadcast("", notify)
			}
		}
	}

	// Clean up on disconnect
	if room != nil && peerID != "" {
		log.Printf("Peer %s disconnected", peerID)

		room.mu.Lock()
		if peer, exists := room.Peers[peerID]; exists {
			if peer.PC != nil {
				peer.PC.Close()
			}
			delete(room.Peers, peerID)
		}
		room.mu.Unlock()

		if len(room.Peers) == 0 {
			s.RemoveRoom(room.MeetingID)
		}

		notify, _ := json.Marshal(map[string]interface{}{
			"event": "peer-left",
			"data": map[string]interface{}{
				"peer_id": peerID,
			},
		})
		room.Broadcast("", notify)
	}
}

var sfu *SFUServer

func main() {
	sfu = NewSFUServer()

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

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		handleWebSocket(sfu, w, r)
	})

	port := "8082"
	fmt.Printf("SFU server starting on port %s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
