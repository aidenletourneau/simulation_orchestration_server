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
func HandleWebSocket(registry *Registry, scenarioManager *ScenarioManager, sagaManager *SagaManager) http.HandlerFunc {
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
				handleEvent(simID, msg, registry, scenarioManager, sagaManager)
			case "step.completed":
				handleStepCompleted(simID, msg, sagaManager)
			case "step.failed":
				handleStepFailed(simID, msg, sagaManager)
			default:
				log.Printf("Unknown message type: %s", msg.Type)
			}
		}

		// Cleanup on disconnect
		registry.Unregister(simID)
		log.Printf("Simulation disconnected: %s", simID)
	}
}

// handleEvent processes incoming events and creates Sagas when rules match
// This is the entry point for Saga-based transaction coordination
func handleEvent(sourceID string, msg Message, registry *Registry, scenarioManager *ScenarioManager, sagaManager *SagaManager) {
	// Create event
	event := Event{
		Type:      msg.Type,
		EventType: msg.EventType,
		Source:    sourceID,
		Payload:   msg.Payload,
	}

	log.Printf("Event received from %s: %s", sourceID, msg.EventType)

	// Process event through scenario manager to get matching actions
	actions := scenarioManager.ProcessEvent(event)

	if len(actions) == 0 {
		log.Printf("No matching rules for event: %s", msg.EventType)
		return
	}

	// Create a Saga from the actions
	// The Saga ensures eventual consistency: either all steps complete or all are rolled back
	saga, err := sagaManager.CreateSaga(actions)
	if err != nil {
		log.Printf("Failed to create Saga: %v", err)
		return
	}

	log.Printf("Saga %s created from event %s with %d steps", saga.SagaID, msg.EventType, len(actions))
	// Note: The first step is dispatched automatically by CreateSaga
	// Subsequent steps will be dispatched when step.completed events are received
}

// handleStepCompleted processes step.completed events from simulations
// This advances the Saga to the next step or marks it as completed
func handleStepCompleted(simID string, msg Message, sagaManager *SagaManager) {
	if msg.SagaID == "" {
		log.Printf("step.completed event missing saga_id from %s", simID)
		return
	}

	if msg.StepID == nil {
		log.Printf("step.completed event missing step_id from %s", simID)
		return
	}

	stepID := *msg.StepID
	log.Printf("Step completion received from %s: Saga %s, Step %d", simID, msg.SagaID, stepID)

	if err := sagaManager.HandleStepCompletion(msg.SagaID, stepID); err != nil {
		log.Printf("Failed to handle step completion: %v", err)
	}
}

// handleStepFailed processes step.failed events from simulations
// This triggers compensation for all previously completed steps
func handleStepFailed(simID string, msg Message, sagaManager *SagaManager) {
	if msg.SagaID == "" {
		log.Printf("step.failed event missing saga_id from %s", simID)
		return
	}

	if msg.StepID == nil {
		log.Printf("step.failed event missing step_id from %s", simID)
		return
	}

	stepID := *msg.StepID
	log.Printf("Step failure received from %s: Saga %s, Step %d", simID, msg.SagaID, stepID)

	if err := sagaManager.HandleStepFailure(msg.SagaID, stepID); err != nil {
		log.Printf("Failed to handle step failure: %v", err)
	}
}

