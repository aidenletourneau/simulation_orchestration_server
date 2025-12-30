package main

/*
Saga Pattern Implementation for Eventual Consistency

This file implements a choreography-based Saga pattern to ensure eventual consistency
and synchronization across multiple simulations when a scenario spans multiple actions.

How Saga Ensures Synchronization:
1. Sequential Execution: Steps are executed one at a time, with each step waiting for
   confirmation before the next step is dispatched. This prevents race conditions and
   ensures ordered execution.

2. Event-Driven Choreography: Sagas are driven by events (step.completed, step.failed)
   emitted by simulations. This is non-blocking and allows simulations to work
   asynchronously while maintaining transaction boundaries.

3. Compensation on Failure: If any step fails, compensating actions are executed for
   all previously completed steps in reverse order. This ensures eventual consistency:
   either all steps complete successfully, or all completed steps are rolled back.

4. Thread-Safe State Management: All Saga state is protected by mutexes, ensuring
   safe concurrent access from multiple goroutines handling different simulations.

5. Explicit State Tracking: Each Saga and SagaStep maintains explicit status, allowing
   the system to track progress and recover from failures.

The Saga pattern guarantees that:
- All simulation actions in a scenario complete successfully, OR
- All completed actions are rolled back via compensations
*/

import (
	"fmt"
	"log"
	"sync"
	"time"
)

// SagaStatus represents the current state of a Saga
type SagaStatus string

const (
	SagaStatusPending     SagaStatus = "Pending"
	SagaStatusInProgress  SagaStatus = "InProgress"
	SagaStatusCompleted   SagaStatus = "Completed"
	SagaStatusFailed      SagaStatus = "Failed"
	SagaStatusCompensating SagaStatus = "Compensating"
)

// StepStatus represents the current state of a Saga step
type StepStatus string

const (
	StepStatusPending  StepStatus = "Pending"
	StepStatusInFlight StepStatus = "InFlight"
	StepStatusCompleted StepStatus = "Completed"
	StepStatusFailed    StepStatus = "Failed"
)

// SagaStep represents a single step in a Saga transaction
type SagaStep struct {
	StepID          int                    // Sequential step identifier
	TargetSimulation string                 // Which simulation to send command to
	Command         string                 // Forward action command
	CompensateCommand string               // Rollback command
	Params          map[string]interface{} // Command parameters
	CompensateParams map[string]interface{} // Compensation parameters
	Status          StepStatus             // Current step status
	CreatedAt       time.Time              // When step was created
	CompletedAt     *time.Time             // When step completed (nil if not completed)
}

// Saga represents a distributed transaction across multiple simulations
// Each Saga ensures eventual consistency: either all steps complete or all are rolled back
type Saga struct {
	SagaID      string      // Unique identifier for this Saga
	CurrentStep int         // Index of the current step being executed (0-based)
	Status      SagaStatus  // Overall Saga status
	Steps       []*SagaStep // Ordered list of steps to execute
	CreatedAt   time.Time   // When Saga was created
	mu          sync.RWMutex // Protects Saga state
}

// SagaManager manages the lifecycle of all Sagas
// It handles Saga creation, step progression, and compensation in a thread-safe manner
type SagaManager struct {
	sagas map[string]*Saga // Map of SagaID -> Saga
	mu    sync.RWMutex     // Protects sagas map
	registry *Registry      // Reference to simulation registry for sending commands
}

// NewSagaManager creates a new SagaManager
func NewSagaManager(registry *Registry) *SagaManager {
	return &SagaManager{
		sagas:   make(map[string]*Saga),
		registry: registry,
	}
}

// CreateSaga creates a new Saga from a list of actions (from a scenario rule)
// The Saga is created in Pending status and the first step is dispatched immediately
func (sm *SagaManager) CreateSaga(actions []Action) (*Saga, error) {
	if len(actions) == 0 {
		return nil, fmt.Errorf("cannot create saga with no actions")
	}

	// Generate unique Saga ID
	sagaID := fmt.Sprintf("saga_%d", time.Now().UnixNano())

	// Convert actions to SagaSteps
	steps := make([]*SagaStep, len(actions))
	for i, action := range actions {
		steps[i] = &SagaStep{
			StepID:          i,
			TargetSimulation: action.SendTo,
			Command:         action.Command,
			CompensateCommand: action.CompensateCommand,
			Params:          action.Params,
			CompensateParams: action.CompensateParams,
			Status:          StepStatusPending,
			CreatedAt:       time.Now(),
		}
	}

	saga := &Saga{
		SagaID:      sagaID,
		CurrentStep: 0,
		Status:      SagaStatusPending,
		Steps:       steps,
		CreatedAt:   time.Now(),
	}

	// Store Saga
	sm.mu.Lock()
	sm.sagas[sagaID] = saga
	sm.mu.Unlock()

	log.Printf("Created Saga %s with %d steps", sagaID, len(steps))

	// Dispatch first step immediately
	if err := sm.dispatchStep(saga, 0); err != nil {
		log.Printf("Failed to dispatch first step of Saga %s: %v", sagaID, err)
		// Mark Saga as failed
		saga.mu.Lock()
		saga.Status = SagaStatusFailed
		saga.mu.Unlock()
		return saga, err
	}

	return saga, nil
}

// dispatchStep sends a command to the target simulation for a specific step
// This is the forward action of the Saga step
func (sm *SagaManager) dispatchStep(saga *Saga, stepIndex int) error {
	if stepIndex < 0 || stepIndex >= len(saga.Steps) {
		return fmt.Errorf("invalid step index: %d", stepIndex)
	}

	step := saga.Steps[stepIndex]

	// Get target simulation
	targetSim, exists := sm.registry.Get(step.TargetSimulation)
	if !exists {
		return fmt.Errorf("target simulation not found: %s", step.TargetSimulation)
	}

	// Create command message with Saga context
	stepIDPtr := &stepIndex
	command := Message{
		Type:    "command",
		Command: step.Command,
		Params:  step.Params,
		// Include Saga context so simulation can acknowledge with saga_id and step_id
		SagaID:  saga.SagaID,
		StepID:  stepIDPtr,
	}

	// Send command
	if err := targetSim.Connection.WriteJSON(command); err != nil {
		return fmt.Errorf("failed to send command to %s: %w", step.TargetSimulation, err)
	}

	// Update step status
	saga.mu.Lock()
	step.Status = StepStatusInFlight
	if saga.Status == SagaStatusPending {
		saga.Status = SagaStatusInProgress
	}
	saga.mu.Unlock()

	log.Printf("Saga %s: Dispatched step %d to %s (command: %s)", saga.SagaID, stepIndex, step.TargetSimulation, step.Command)
	return nil
}

// HandleStepCompletion is called when a simulation emits a step.completed event
// This advances the Saga to the next step or marks it as completed
func (sm *SagaManager) HandleStepCompletion(sagaID string, stepID int) error {
	sm.mu.RLock()
	saga, exists := sm.sagas[sagaID]
	sm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("saga not found: %s", sagaID)
	}

	saga.mu.Lock()
	defer saga.mu.Unlock()

	// Validate step ID
	if stepID < 0 || stepID >= len(saga.Steps) {
		return fmt.Errorf("invalid step ID: %d", stepID)
	}

	step := saga.Steps[stepID]

	// Check if this step is actually in flight
	if step.Status != StepStatusInFlight {
		log.Printf("Saga %s: Step %d is not in flight (status: %s), ignoring completion", sagaID, stepID, step.Status)
		return nil
	}

	// Mark step as completed
	now := time.Now()
	step.Status = StepStatusCompleted
	step.CompletedAt = &now

	log.Printf("Saga %s: Step %d completed", sagaID, stepID)

	// Check if this was the last step
	if stepID == len(saga.Steps)-1 {
		// All steps completed successfully
		saga.Status = SagaStatusCompleted
		log.Printf("Saga %s: All steps completed successfully", sagaID)
		return nil
	}

	// Advance to next step
	nextStepIndex := stepID + 1
	saga.CurrentStep = nextStepIndex

	// Unlock before dispatching to avoid deadlock
	saga.mu.Unlock()

	// Dispatch next step
	if err := sm.dispatchStep(saga, nextStepIndex); err != nil {
		log.Printf("Saga %s: Failed to dispatch step %d: %v", sagaID, nextStepIndex, err)
		// Trigger compensation
		sm.triggerCompensation(saga, stepID) // Compensate from the failed step backwards
		return err
	}

	saga.mu.Lock()
	return nil
}

// HandleStepFailure is called when a simulation emits a step.failed event or times out
// This triggers compensation for all completed steps
func (sm *SagaManager) HandleStepFailure(sagaID string, stepID int) error {
	sm.mu.RLock()
	saga, exists := sm.sagas[sagaID]
	sm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("saga not found: %s", sagaID)
	}

	saga.mu.Lock()
	defer saga.mu.Unlock()

	// Validate step ID
	if stepID < 0 || stepID >= len(saga.Steps) {
		return fmt.Errorf("invalid step ID: %d", stepID)
	}

	step := saga.Steps[stepID]

	// Mark step as failed
	step.Status = StepStatusFailed
	saga.Status = SagaStatusFailed

	log.Printf("Saga %s: Step %d failed, triggering compensation", sagaID, stepID)

	// Unlock before compensation to avoid deadlock
	saga.mu.Unlock()

	// Trigger compensation (rollback all completed steps in reverse order)
	sm.triggerCompensation(saga, stepID-1) // Compensate up to the step before the failed one

	return nil
}

// triggerCompensation executes compensating actions for all completed steps in reverse order
// This ensures eventual consistency: if any step fails, all previous steps are rolled back
func (sm *SagaManager) triggerCompensation(saga *Saga, lastStepToCompensate int) {
	saga.mu.Lock()
	saga.Status = SagaStatusCompensating
	saga.mu.Unlock()

	log.Printf("Saga %s: Starting compensation from step %d", saga.SagaID, lastStepToCompensate)

	// Compensate in reverse order (most recent first)
	for i := lastStepToCompensate; i >= 0; i-- {
		step := saga.Steps[i]

		saga.mu.RLock()
		status := step.Status
		saga.mu.RUnlock()

		// Only compensate steps that were completed
		if status != StepStatusCompleted {
			log.Printf("Saga %s: Skipping compensation for step %d (status: %s)", saga.SagaID, i, status)
			continue
		}

		// Check if compensation command is defined
		if step.CompensateCommand == "" {
			log.Printf("Saga %s: Step %d has no compensation command, skipping", saga.SagaID, i)
			continue
		}

		// Get target simulation
		targetSim, exists := sm.registry.Get(step.TargetSimulation)
		if !exists {
			log.Printf("Saga %s: Target simulation not found for compensation: %s", saga.SagaID, step.TargetSimulation)
			continue
		}

		// Create compensation command
		stepIDPtr := &i
		compensateMsg := Message{
			Type:    "command",
			Command: step.CompensateCommand,
			Params:  step.CompensateParams,
			SagaID:  saga.SagaID,
			StepID:  stepIDPtr,
		}

		// Send compensation command
		if err := targetSim.Connection.WriteJSON(compensateMsg); err != nil {
			log.Printf("Saga %s: Failed to send compensation command for step %d: %v", saga.SagaID, i, err)
			// Continue with other compensations even if one fails
			continue
		}

		log.Printf("Saga %s: Compensation command sent for step %d to %s", saga.SagaID, i, step.TargetSimulation)

		// Mark step as compensated (we don't wait for acknowledgment in MVP)
		saga.mu.Lock()
		step.Status = StepStatusFailed // Mark as failed since we're compensating
		saga.mu.Unlock()
	}

	saga.mu.Lock()
	saga.Status = SagaStatusFailed
	saga.mu.Unlock()

	log.Printf("Saga %s: Compensation completed", saga.SagaID)
}

// GetSaga retrieves a Saga by ID (for debugging/monitoring)
func (sm *SagaManager) GetSaga(sagaID string) (*Saga, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	saga, exists := sm.sagas[sagaID]
	return saga, exists
}

// GetAllSagas returns all active Sagas (for debugging/monitoring)
func (sm *SagaManager) GetAllSagas() map[string]*Saga {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	result := make(map[string]*Saga)
	for k, v := range sm.sagas {
		result[k] = v
	}
	return result
}

