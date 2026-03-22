package main

import (
	"fmt"
	"log"
	"net"
	"os"

	"github.com/pion/turn/v4"
)

func main() {
	publicIP := os.Getenv("PUBLIC_IP")
	if publicIP == "" {
		publicIP = "192.168.1.78"
	}

	// TURN server credentials
	turnUsername := os.Getenv("TURN_USERNAME")
	if turnUsername == "" {
		turnUsername = "meeting"
	}
	turnPassword := os.Getenv("TURN_PASSWORD")
	if turnPassword == "" {
		turnPassword = "meeting123"
	}

	// Create a UDP listener to accept TURN connections
	udpListener, err := net.ListenPacket("udp4", ":3478")
	if err != nil {
		log.Fatalf("Failed to create UDP listener: %v", err)
	}

	// Create a TCP listener for TURN connections
	tcpListener, err := net.Listen("tcp4", ":3478")
	if err != nil {
		log.Fatalf("Failed to create TCP listener: %v", err)
	}

	// Create user map
	usersMap := map[string][]byte{
		turnUsername: turn.GenerateAuthKey(turnUsername, "pion-turn", turnPassword),
	}

	// Create TURN server
	server, err := turn.NewServer(turn.ServerConfig{
		Realm: "pion-turn",
		// Set AuthHandler callback
		AuthHandler: func(username string, realm string, srcAddr net.Addr) ([]byte, bool) {
			if key, ok := usersMap[username]; ok {
				return key, true
			}
			return nil, false
		},
		// ListenerConfigs is a collection of TURN listeners
		ListenerConfigs: []turn.ListenerConfig{
			{
				Listener: tcpListener,
				RelayAddressGenerator: &turn.RelayAddressGeneratorStatic{
					RelayAddress: net.ParseIP(publicIP),
					Address:      "0.0.0.0",
				},
			},
		},
		PacketConnConfigs: []turn.PacketConnConfig{
			{
				PacketConn: udpListener,
				RelayAddressGenerator: &turn.RelayAddressGeneratorStatic{
					RelayAddress: net.ParseIP(publicIP),
					Address:      "0.0.0.0",
				},
			},
		},
	})
	if err != nil {
		log.Fatalf("Failed to create TURN server: %v", err)
	}

	fmt.Printf("TURN/STUN server starting on %s:3478\n", publicIP)
	fmt.Printf("Username: %s, Password: %s\n", turnUsername, turnPassword)
	log.Fatal(server.Close())
}
