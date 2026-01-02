package registry

import (
	"sync"

	"github.com/aidenletourneau/simulation_orchestration_server/server/internal/models"
	"github.com/gorilla/websocket"
)

// Registry manages connected simulations
type Registry struct {
	simulations map[string]*models.Simulation
	mu          sync.RWMutex
}

// NewRegistry creates a new simulation registry
func NewRegistry() *Registry {
	return &Registry{
		simulations: make(map[string]*models.Simulation),
	}
}

// Register adds a new simulation to the registry
func (r *Registry) Register(id, name string, conn *websocket.Conn) *models.Simulation {
	r.mu.Lock()
	defer r.mu.Unlock()

	sim := &models.Simulation{
		ID:         id,
		Name:       name,
		Connection: conn,
	}

	r.simulations[id] = sim
	return sim
}

// Get retrieves a simulation by ID
func (r *Registry) Get(id string) (*models.Simulation, bool) {
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
func (r *Registry) GetAll() map[string]*models.Simulation {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string]*models.Simulation)
	for k, v := range r.simulations {
		result[k] = v
	}
	return result
}
