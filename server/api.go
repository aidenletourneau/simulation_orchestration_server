package main

import (
	"encoding/json"
	"net/http"
)

// SimulationResponse represents a simulation in the API response
type SimulationResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// HandleGetSimulations returns all connected simulations
func HandleGetSimulations(registry *Registry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		simulations := registry.GetAll()
		response := make([]SimulationResponse, 0, len(simulations))
		for id, sim := range simulations {
			response = append(response, SimulationResponse{
				ID:   id,
				Name: sim.Name,
			})
		}

		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
	}
}

// HandleGetLogs returns all log entries
func HandleGetLogs(logStore *LogStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		logs := logStore.GetAll()
		if err := json.NewEncoder(w).Encode(logs); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
	}
}
