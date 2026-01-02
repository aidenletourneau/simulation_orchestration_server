package main

import (
	"fmt"
	"log"
	"sync"
	"time"
)

// LogEntry represents a single log entry
type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Message   string    `json:"message"`
	Level     string    `json:"level"`
}

// LogStore stores logs in memory
type LogStore struct {
	entries []LogEntry
	mu      sync.RWMutex
	maxSize int // Maximum number of logs to keep (0 = unlimited)
}

// NewLogStore creates a new log store
func NewLogStore(maxSize int) *LogStore {
	return &LogStore{
		entries: make([]LogEntry, 0),
		maxSize: maxSize,
	}
}

// Add adds a log entry to the store
func (ls *LogStore) Add(level, message string) {
	ls.mu.Lock()
	defer ls.mu.Unlock()

	entry := LogEntry{
		Timestamp: time.Now(),
		Message:   message,
		Level:     level,
	}

	ls.entries = append(ls.entries, entry)

	// Trim if we exceed max size
	if ls.maxSize > 0 && len(ls.entries) > ls.maxSize {
		ls.entries = ls.entries[len(ls.entries)-ls.maxSize:]
	}
}

// GetAll returns all log entries
func (ls *LogStore) GetAll() []LogEntry {
	ls.mu.RLock()
	defer ls.mu.RUnlock()

	// Return a copy to prevent race conditions
	result := make([]LogEntry, len(ls.entries))
	copy(result, ls.entries)
	return result
}

// Clear clears all log entries
func (ls *LogStore) Clear() {
	ls.mu.Lock()
	defer ls.mu.Unlock()
	ls.entries = make([]LogEntry, 0)
}

// LogAndStore logs a message using the standard log package and stores it in the log store
func (ls *LogStore) LogAndStore(level, format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	log.Printf(format, args...)
	ls.Add(level, message)
}
