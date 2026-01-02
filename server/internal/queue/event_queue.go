package queue

import (
	"log"
	"sync"
	"time"

	"github.com/aidenletourneau/simulation_orchestration_server/server/internal/models"
)

/*
Event Queue System for Synchronization

This file implements an event queue to ensure ordered processing of events from
multiple simulations. This prevents race conditions when concurrent events arrive
and need to be processed sequentially.

The queue ensures:
1. Events are processed in order (FIFO)
2. Only one event is processed at a time
3. Predictable ordering when multiple simulations send events concurrently
*/

// QueuedEvent represents an event waiting to be processed
type QueuedEvent struct {
	SourceID  string
	Message   models.Message
	Timestamp time.Time
}

// EventQueue manages a queue of events to be processed sequentially
type EventQueue struct {
	events chan QueuedEvent
	mu     sync.RWMutex
	closed bool
}

// NewEventQueue creates a new event queue with the specified buffer size
func NewEventQueue(bufferSize int) *EventQueue {
	return &EventQueue{
		events: make(chan QueuedEvent, bufferSize),
		closed: false,
	}
}

// Enqueue adds an event to the queue for processing
// Returns false if the queue is closed
func (eq *EventQueue) Enqueue(sourceID string, msg models.Message) bool {
	eq.mu.RLock()
	defer eq.mu.RUnlock()

	if eq.closed {
		log.Printf("Event queue is closed, dropping event from %s", sourceID)
		return false
	}

	queuedEvent := QueuedEvent{
		SourceID:  sourceID,
		Message:   msg,
		Timestamp: time.Now(),
	}

	select {
	case eq.events <- queuedEvent:
		log.Printf("Event queued from %s: %s (queue length: %d)", sourceID, msg.EventType, len(eq.events))
		return true
	default:
		log.Printf("Event queue is full, dropping event from %s", sourceID)
		return false
	}
}

// ProcessorFunc is a function type for processing events
type ProcessorFunc func(sourceID string, msg models.Message)

// StartProcessor starts a goroutine that processes events from the queue sequentially
// This ensures only one event is processed at a time, preventing race conditions
func (eq *EventQueue) StartProcessor(processor ProcessorFunc) {
	go func() {
		for queuedEvent := range eq.events {
			processor(queuedEvent.SourceID, queuedEvent.Message)
		}
	}()
}

// Close closes the event queue and stops accepting new events
func (eq *EventQueue) Close() {
	eq.mu.Lock()
	defer eq.mu.Unlock()

	if !eq.closed {
		eq.closed = true
		close(eq.events)
		log.Println("Event queue closed")
	}
}

// GetQueueLength returns the current number of events in the queue
func (eq *EventQueue) GetQueueLength() int {
	return len(eq.events)
}
