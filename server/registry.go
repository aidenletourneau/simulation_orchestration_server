package main

import (
	"sync"

	"github.com/gorilla/websocket"
)

// Registry manages connected simulations
type Registry struct {
	simulations map[string]*Simulation
	mu          sync.RWMutex
}

// NewRegistry creates a new simulation registry
func NewRegistry() *Registry {
	return &Registry{
		simulations: make(map[string]*Simulation),
	}
}

// Register adds a new simulation to the registry
func (r *Registry) Register(id, name string, conn *websocket.Conn) *Simulation {
	r.mu.Lock()
	defer r.mu.Unlock()

	sim := &Simulation{
		ID:         id,
		Name:       name,
		Connection: conn,
	}

	r.simulations[id] = sim
	return sim
}

// Get retrieves a simulation by ID
func (r *Registry) Get(id string) (*Simulation, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	sim, exists := r.simulations[id]
	return sim, exists
}

// Unregister removes a simulation from the registry
func (r *Registry) Unregister(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.simulations, id)
}

// GetAll returns all registered simulations
func (r *Registry) GetAll() map[string]*Simulation {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string]*Simulation)
	for k, v := range r.simulations {
		result[k] = v
	}
	return result
}

