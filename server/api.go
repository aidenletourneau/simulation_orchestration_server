package main

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
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

// ScenarioInfoResponse represents scenario information in API response
type ScenarioInfoResponse struct {
	Name  string `json:"name"`
	Rules int    `json:"rules"`
}

// HandleGetScenario returns information about the current scenario
func HandleGetScenario(scenarioManager *ScenarioManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		scenario := scenarioManager.GetCurrentScenario()
		if scenario == nil {
			http.Error(w, "No scenario loaded", http.StatusNotFound)
			return
		}

		response := ScenarioInfoResponse{
			Name:  scenario.Name,
			Rules: len(scenario.Rules),
		}

		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
	}
}

// HandleUploadScenario handles YAML scenario file uploads
func HandleUploadScenario(scenarioManager *ScenarioManager, logStore *LogStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		// Handle preflight OPTIONS request
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Parse multipart form (max 10MB)
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			http.Error(w, "Failed to parse form: "+err.Error(), http.StatusBadRequest)
			return
		}

		// Get the file from form
		file, header, err := r.FormFile("scenario")
		if err != nil {
			http.Error(w, "No file uploaded or invalid form field: "+err.Error(), http.StatusBadRequest)
			return
		}
		defer file.Close()

		// Check file extension
		filename := strings.ToLower(header.Filename)
		if !strings.HasSuffix(filename, ".yaml") && !strings.HasSuffix(filename, ".yml") {
			http.Error(w, "File must be a YAML file (.yaml or .yml)", http.StatusBadRequest)
			return
		}

		// Read file content
		fileBytes, err := io.ReadAll(file)
		if err != nil {
			http.Error(w, "Failed to read file: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Load scenario from bytes
		if err := scenarioManager.LoadScenarioFromBytes(fileBytes); err != nil {
			logStore.LogAndStore("error", "Failed to load uploaded scenario: %v", err)
			http.Error(w, "Failed to load scenario: "+err.Error(), http.StatusBadRequest)
			return
		}

		scenario := scenarioManager.GetCurrentScenario()
		logStore.LogAndStore("info", "Scenario uploaded successfully: %s (%d rules)", scenario.Name, len(scenario.Rules))

		// Return success response
		w.Header().Set("Content-Type", "application/json")
		response := ScenarioInfoResponse{
			Name:  scenario.Name,
			Rules: len(scenario.Rules),
		}

		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
	}
}
