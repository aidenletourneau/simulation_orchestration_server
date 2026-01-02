package websocket

import (
	"net/http"

	"github.com/aidenletourneau/simulation_orchestration_server/server/internal/logging"
	"github.com/aidenletourneau/simulation_orchestration_server/server/internal/models"
	"github.com/aidenletourneau/simulation_orchestration_server/server/internal/queue"
	"github.com/aidenletourneau/simulation_orchestration_server/server/internal/registry"
	"github.com/aidenletourneau/simulation_orchestration_server/server/internal/saga"
	"github.com/aidenletourneau/simulation_orchestration_server/server/internal/scenario"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins for MVP
		return true
	},
}

// EventHandler is a function type for handling events
type EventHandler func(sourceID string, msg models.Message)

// HandleWebSocket handles WebSocket connections
func HandleWebSocket(
	reg *registry.Registry,
	scenarioManager *scenario.ScenarioManager,
	sagaManager *saga.SagaManager,
	eventQueue *queue.EventQueue,
	logStore *logging.LogStore,
	eventHandler EventHandler,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			logStore.LogAndStore("error", "WebSocket upgrade failed: %v", err)
			return
		}
		defer conn.Close()

		logStore.LogAndStore("info", "New WebSocket connection established")

		// Wait for registration message
		var msg models.Message
		if err := conn.ReadJSON(&msg); err != nil {
			logStore.LogAndStore("error", "Failed to read registration: %v", err)
			return
		}

		if msg.Type != "register" {
			logStore.LogAndStore("error", "Expected registration message, got: %s", msg.Type)
			return
		}

		// Register simulation
		simID := msg.ID
		if simID == "" {
			logStore.LogAndStore("error", "Registration missing ID")
			return
		}

		reg.Register(simID, msg.Name, conn)
		logStore.LogAndStore("info", "Simulation registered: %s (%s)", simID, msg.Name)

		// Send registration confirmation
		response := models.Message{
			Type:   "registered",
			Status: "ok",
		}
		if err := conn.WriteJSON(response); err != nil {
			logStore.LogAndStore("error", "Failed to send registration confirmation: %v", err)
			return
		}

		// Handle messages
		for {
			var msg models.Message
			if err := conn.ReadJSON(&msg); err != nil {
				logStore.LogAndStore("error", "Error reading message from %s: %v", simID, err)
				break
			}

			// Handle different message types
			switch msg.Type {
			case "event":
				// Enqueue event for sequential processing to prevent race conditions
				if !eventQueue.Enqueue(simID, msg) {
					logStore.LogAndStore("error", "Failed to enqueue event from %s: %s", simID, msg.EventType)
					// Optionally send error response to simulation
					errorResponse := models.Message{
						Type:   "error",
						Status: "queue_full",
					}
					conn.WriteJSON(errorResponse)
				}
			case "step.completed":
				// Step completion events don't need queuing - they're part of existing sagas
				handleStepCompleted(simID, msg, sagaManager, logStore)
			case "step.failed":
				// Step failure events don't need queuing - they're part of existing sagas
				handleStepFailed(simID, msg, sagaManager, logStore)
			default:
				logStore.LogAndStore("warning", "Unknown message type: %s", msg.Type)
			}
		}

		// Cleanup on disconnect
		reg.Unregister(simID)
		logStore.LogAndStore("info", "Simulation disconnected: %s", simID)
	}
}

// handleStepCompleted processes step.completed events from simulations
// This advances the Saga to the next step or marks it as completed
func handleStepCompleted(simID string, msg models.Message, sagaManager *saga.SagaManager, logStore *logging.LogStore) {
	if msg.SagaID == "" {
		logStore.LogAndStore("error", "step.completed event missing saga_id from %s", simID)
		return
	}

	if msg.StepID == nil {
		logStore.LogAndStore("error", "step.completed event missing step_id from %s", simID)
		return
	}

	stepID := *msg.StepID
	logStore.LogAndStore("info", "Step completion received from %s: Saga %s, Step %d", simID, msg.SagaID, stepID)

	if err := sagaManager.HandleStepCompletion(msg.SagaID, stepID); err != nil {
		logStore.LogAndStore("error", "Failed to handle step completion: %v", err)
	}
}

// handleStepFailed processes step.failed events from simulations
// This triggers compensation for all previously completed steps
func handleStepFailed(simID string, msg models.Message, sagaManager *saga.SagaManager, logStore *logging.LogStore) {
	if msg.SagaID == "" {
		logStore.LogAndStore("error", "step.failed event missing saga_id from %s", simID)
		return
	}

	if msg.StepID == nil {
		logStore.LogAndStore("error", "step.failed event missing step_id from %s", simID)
		return
	}

	stepID := *msg.StepID
	logStore.LogAndStore("info", "Step failure received from %s: Saga %s, Step %d", simID, msg.SagaID, stepID)

	if err := sagaManager.HandleStepFailure(msg.SagaID, stepID); err != nil {
		logStore.LogAndStore("error", "Failed to handle step failure: %v", err)
	}
}
