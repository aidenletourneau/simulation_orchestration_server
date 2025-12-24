package main

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins for MVP
		return true
	},
}

// HandleWebSocket handles WebSocket connections
func HandleWebSocket(registry *Registry, scenarioManager *ScenarioManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("WebSocket upgrade failed: %v", err)
			return
		}
		defer conn.Close()

		log.Println("New WebSocket connection established")

		// Wait for registration message
		var msg Message
		if err := conn.ReadJSON(&msg); err != nil {
			log.Printf("Failed to read registration: %v", err)
			return
		}

		if msg.Type != "register" {
			log.Printf("Expected registration message, got: %s", msg.Type)
			return
		}

		// Register simulation
		simID := msg.ID
		if simID == "" {
			log.Println("Registration missing ID")
			return
		}

		registry.Register(simID, msg.Name, conn)
		log.Printf("Simulation registered: %s (%s)", simID, msg.Name)

		// Send registration confirmation
		response := Message{
			Type:   "registered",
			Status: "ok",
		}
		if err := conn.WriteJSON(response); err != nil {
			log.Printf("Failed to send registration confirmation: %v", err)
			return
		}

		// Handle messages
		for {
			var msg Message
			if err := conn.ReadJSON(&msg); err != nil {
				log.Printf("Error reading message from %s: %v", simID, err)
				break
			}

			// Handle different message types
			switch msg.Type {
			case "event":
				handleEvent(simID, msg, registry, scenarioManager)
			default:
				log.Printf("Unknown message type: %s", msg.Type)
			}
		}

		// Cleanup on disconnect
		registry.Unregister(simID)
		log.Printf("Simulation disconnected: %s", simID)
	}
}

func handleEvent(sourceID string, msg Message, registry *Registry, scenarioManager *ScenarioManager) {
	// Create event
	event := Event{
		Type:      msg.Type,
		EventType: msg.EventType,
		Source:    sourceID,
		Payload:   msg.Payload,
	}

	log.Printf("Event received from %s: %s", sourceID, msg.EventType)

	// Process event through scenario manager
	actions := scenarioManager.ProcessEvent(event)

	// Execute actions
	for _, action := range actions {
		// Get target simulation
		targetSim, exists := registry.Get(action.SendTo)
		if !exists {
			log.Printf("Target simulation not found: %s", action.SendTo)
			continue
		}

		// Create command message
		command := Message{
			Type:    "command",
			Command: action.Command,
			Params:  action.Params,
		}

		// Send command to target
		if err := targetSim.Connection.WriteJSON(command); err != nil {
			log.Printf("Failed to send command to %s: %v", action.SendTo, err)
			continue
		}

		log.Printf("Command sent to %s: %s", action.SendTo, action.Command)
	}
}

