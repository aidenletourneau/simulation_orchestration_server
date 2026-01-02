package websocket

import (
	"github.com/aidenletourneau/simulation_orchestration_server/server/internal/logging"
	"github.com/aidenletourneau/simulation_orchestration_server/server/internal/models"
	"github.com/aidenletourneau/simulation_orchestration_server/server/internal/saga"
	"github.com/aidenletourneau/simulation_orchestration_server/server/internal/scenario"
)

// CreateEventHandler creates an event handler function that processes events and creates Sagas
func CreateEventHandler(
	scenarioManager *scenario.ScenarioManager,
	sagaManager *saga.SagaManager,
	logStore *logging.LogStore,
) func(sourceID string, msg models.Message) {
	return func(sourceID string, msg models.Message) {
		// Create event
		event := models.Event{
			Type:      msg.Type,
			EventType: msg.EventType,
			Source:    sourceID,
			Payload:   msg.Payload,
		}

		logStore.LogAndStore("info", "Event received from %s: %s", sourceID, msg.EventType)

		// Process event through scenario manager to get matching actions
		actions := scenarioManager.ProcessEvent(event)

		if len(actions) == 0 {
			logStore.LogAndStore("info", "No matching rules for event: %s", msg.EventType)
			return
		}

		// Create a Saga from the actions
		// The Saga ensures eventual consistency: either all steps complete or all are rolled back
		saga, err := sagaManager.CreateSaga(actions)
		if err != nil {
			logStore.LogAndStore("error", "Failed to create Saga: %v", err)
			return
		}

		logStore.LogAndStore("info", "Saga %s created from event %s with %d steps", saga.SagaID, msg.EventType, len(actions))
		// Note: The first step is dispatched automatically by CreateSaga
		// Subsequent steps will be dispatched when step.completed events are received
	}
}
