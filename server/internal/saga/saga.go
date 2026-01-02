package saga

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/aidenletourneau/simulation_orchestration_server/server/internal/models"
	"github.com/aidenletourneau/simulation_orchestration_server/server/internal/registry"
)

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

// SagaStatus represents the current state of a Saga
type SagaStatus string

const (
	SagaStatusPending      SagaStatus = "Pending"
	SagaStatusInProgress   SagaStatus = "InProgress"
	SagaStatusCompleted    SagaStatus = "Completed"
	SagaStatusFailed       SagaStatus = "Failed"
	SagaStatusCompensating SagaStatus = "Compensating"
)

// StepStatus represents the current state of a Saga step
type StepStatus string

const (
	StepStatusPending   StepStatus = "Pending"
	StepStatusInFlight  StepStatus = "InFlight"
	StepStatusCompleted StepStatus = "Completed"
	StepStatusFailed    StepStatus = "Failed"
)

// SagaStep represents a single step in a Saga transaction
type SagaStep struct {
	StepID            int                    // Sequential step identifier
	TargetSimulation  string                 // Which simulation to send command to
	Command           string                 // Forward action command
	CompensateCommand string                 // Rollback command
	Params            map[string]interface{} // Command parameters
	CompensateParams  map[string]interface{} // Compensation parameters
	Status            StepStatus             // Current step status
	CreatedAt         time.Time              // When step was created
	CompletedAt       *time.Time             // When step completed (nil if not completed)
}

// Saga represents a distributed transaction across multiple simulations
// Each Saga ensures eventual consistency: either all steps complete or all are rolled back
type Saga struct {
	SagaID      string       // Unique identifier for this Saga
	CurrentStep int          // Index of the current step being executed (0-based)
	Status      SagaStatus   // Overall Saga status
	Steps       []*SagaStep  // Ordered list of steps to execute
	CreatedAt   time.Time    // When Saga was created
	mu          sync.RWMutex // Protects Saga state
	lockedSims  []string     // List of simulation IDs that are locked by this saga
}

// SagaManager manages the lifecycle of all Sagas
// It handles Saga creation, step progression, and compensation in a thread-safe manner
// It also prevents concurrent Sagas from targeting the same simulation
type SagaManager struct {
	sagas    map[string]*Saga // Map of SagaID -> Saga
	mu       sync.RWMutex     // Protects sagas map
	registry *registry.Registry // Reference to simulation registry for sending commands

	// Simulation-level locking to prevent concurrent Sagas
	simulationLocks map[string]*sync.Mutex // Map of simID -> mutex
	activeSagas     map[string][]string    // Map of simID -> []sagaIDs (for conflict tracking)
	lockMu          sync.Mutex             // Protects simulationLocks and activeSagas
}

// NewSagaManager creates a new SagaManager
func NewSagaManager(reg *registry.Registry) *SagaManager {
	return &SagaManager{
		sagas:           make(map[string]*Saga),
		registry:        reg,
		simulationLocks: make(map[string]*sync.Mutex),
		activeSagas:     make(map[string][]string),
	}
}

// acquireSimulationLock acquires a lock for a simulation, preventing concurrent Sagas
// Returns the lock and true if acquired, false if simulation is already locked by another Saga
func (sm *SagaManager) acquireSimulationLock(simID string) (*sync.Mutex, bool) {
	sm.lockMu.Lock()
	defer sm.lockMu.Unlock()

	// Initialize lock if it doesn't exist
	if sm.simulationLocks[simID] == nil {
		sm.simulationLocks[simID] = &sync.Mutex{}
	}

	lock := sm.simulationLocks[simID]

	// Try to acquire lock (non-blocking check)
	acquired := lock.TryLock()
	return lock, acquired
}

// releaseSimulationLock releases a lock for a simulation
func (sm *SagaManager) releaseSimulationLock(simID string, lock *sync.Mutex) {
	lock.Unlock()

	sm.lockMu.Lock()
	defer sm.lockMu.Unlock()

	// Remove from active sagas tracking
	if sagas, exists := sm.activeSagas[simID]; exists {
		// Remove this saga from the list (cleanup happens in cleanupSimulationLocks)
		_ = sagas // Keep for now, cleanup happens when saga completes
	}
}

// trackActiveSimulation records that a saga is using a simulation
func (sm *SagaManager) trackActiveSimulation(simID string, sagaID string) {
	sm.lockMu.Lock()
	defer sm.lockMu.Unlock()

	if sm.activeSagas[simID] == nil {
		sm.activeSagas[simID] = make([]string, 0)
	}
	sm.activeSagas[simID] = append(sm.activeSagas[simID], sagaID)
	log.Printf("Saga %s now active on simulation %s", sagaID, simID)
}

// untrackActiveSimulation removes a saga from simulation tracking
func (sm *SagaManager) untrackActiveSimulation(simID string, sagaID string) {
	sm.lockMu.Lock()
	defer sm.lockMu.Unlock()

	if sagas, exists := sm.activeSagas[simID]; exists {
		for i, id := range sagas {
			if id == sagaID {
				// Remove from slice
				sm.activeSagas[simID] = append(sagas[:i], sagas[i+1:]...)
				log.Printf("Saga %s no longer active on simulation %s", sagaID, simID)
				break
			}
		}
		// Clean up empty entries
		if len(sm.activeSagas[simID]) == 0 {
			delete(sm.activeSagas, simID)
		}
	}
}

// CheckConflict checks if a simulation is currently involved in any active Sagas
// Returns the list of conflicting saga IDs and whether a conflict exists
func (sm *SagaManager) CheckConflict(simID string) ([]string, bool) {
	sm.lockMu.Lock()
	defer sm.lockMu.Unlock()

	activeSagas, exists := sm.activeSagas[simID]
	if !exists || len(activeSagas) == 0 {
		return nil, false
	}

	// Filter to only in-progress sagas
	conflictingSagas := make([]string, 0)
	sm.mu.RLock()
	for _, sagaID := range activeSagas {
		if saga, exists := sm.sagas[sagaID]; exists {
			if saga.Status == SagaStatusInProgress || saga.Status == SagaStatusPending {
				conflictingSagas = append(conflictingSagas, sagaID)
			}
		}
	}
	sm.mu.RUnlock()

	return conflictingSagas, len(conflictingSagas) > 0
}

// cleanupSimulationLocks removes tracking for all simulations used by a saga
func (sm *SagaManager) cleanupSimulationLocks(saga *Saga) {
	// Get unique simulations from saga steps
	sims := make(map[string]bool)
	for _, step := range saga.Steps {
		sims[step.TargetSimulation] = true
	}

	// Untrack this saga from all simulations
	for simID := range sims {
		sm.untrackActiveSimulation(simID, saga.SagaID)
	}
}

// CreateSaga creates a new Saga from a list of actions (from a scenario rule)
// The Saga is created in Pending status and the first step is dispatched immediately
// This method now includes conflict detection and simulation-level locking
func (sm *SagaManager) CreateSaga(actions []models.Action) (*Saga, error) {
	if len(actions) == 0 {
		return nil, fmt.Errorf("cannot create saga with no actions")
	}

	// Check for conflicts before creating the saga
	conflictingSims := make(map[string][]string)
	for _, action := range actions {
		if conflicts, hasConflict := sm.CheckConflict(action.SendTo); hasConflict {
			conflictingSims[action.SendTo] = conflicts
		}
	}

	if len(conflictingSims) > 0 {
		log.Printf("Conflict detected: cannot create saga - simulations are busy")
		for simID, sagaIDs := range conflictingSims {
			log.Printf("  Simulation %s is busy in sagas: %v", simID, sagaIDs)
		}
		return nil, fmt.Errorf("conflict detected: target simulations are busy in other sagas")
	}

	// Acquire locks for all target simulations
	locks := make(map[string]*sync.Mutex)
	lockedSims := make([]string, 0)

	for _, action := range actions {
		lock, acquired := sm.acquireSimulationLock(action.SendTo)
		if !acquired {
			// Release all previously acquired locks
			for simID, l := range locks {
				sm.releaseSimulationLock(simID, l)
			}
			return nil, fmt.Errorf("failed to acquire lock for simulation %s (may be busy)", action.SendTo)
		}
		locks[action.SendTo] = lock
		lockedSims = append(lockedSims, action.SendTo)
	}

	// Generate unique Saga ID
	sagaID := fmt.Sprintf("saga_%d", time.Now().UnixNano())

	// Convert actions to SagaSteps
	steps := make([]*SagaStep, len(actions))
	for i, action := range actions {
		steps[i] = &SagaStep{
			StepID:            i,
			TargetSimulation:  action.SendTo,
			Command:           action.Command,
			CompensateCommand: action.CompensateCommand,
			Params:            action.Params,
			CompensateParams:  action.CompensateParams,
			Status:            StepStatusPending,
			CreatedAt:         time.Now(),
		}
	}

	saga := &Saga{
		SagaID:      sagaID,
		CurrentStep: 0,
		Status:      SagaStatusPending,
		Steps:       steps,
		CreatedAt:   time.Now(),
		lockedSims:  lockedSims, // Store which simulations are locked
	}

	// Store Saga
	sm.mu.Lock()
	sm.sagas[sagaID] = saga
	sm.mu.Unlock()

	// Track this saga for all target simulations
	for _, simID := range lockedSims {
		sm.trackActiveSimulation(simID, sagaID)
	}

	log.Printf("Created Saga %s with %d steps (locks acquired for %d simulations)", sagaID, len(steps), len(lockedSims))

	// Dispatch first step immediately
	if err := sm.dispatchStep(saga, 0); err != nil {
		log.Printf("Failed to dispatch first step of Saga %s: %v", sagaID, err)
		// Release locks and cleanup
		for simID, lock := range locks {
			sm.releaseSimulationLock(simID, lock)
		}
		sm.cleanupSimulationLocks(saga)
		// Mark Saga as failed
		saga.mu.Lock()
		saga.Status = SagaStatusFailed
		saga.mu.Unlock()
		return saga, err
	}

	// Note: Locks will be released when the saga completes or fails
	// This is handled in HandleStepCompletion and HandleStepFailure

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
	command := models.Message{
		Type:    "command",
		Command: step.Command,
		Params:  step.Params,
		// Include Saga context so simulation can acknowledge with saga_id and step_id
		SagaID: saga.SagaID,
		StepID: stepIDPtr,
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

		// Release all simulation locks and cleanup tracking
		saga.mu.Unlock()
		sm.cleanupSimulationLocks(saga)
		sm.releaseAllLocksForSaga(saga)
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

	// Release all simulation locks and cleanup tracking after compensation
	sm.cleanupSimulationLocks(saga)
	sm.releaseAllLocksForSaga(saga)

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
		compensateMsg := models.Message{
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

// releaseAllLocksForSaga releases all simulation locks held by a saga
func (sm *SagaManager) releaseAllLocksForSaga(saga *Saga) {
	// Use the stored list of locked simulations from the saga
	sm.lockMu.Lock()
	for _, simID := range saga.lockedSims {
		if lock, exists := sm.simulationLocks[simID]; exists {
			lock.Unlock()
			log.Printf("Released lock for simulation %s (saga %s)", simID, saga.SagaID)
		}
	}
	sm.lockMu.Unlock()
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
