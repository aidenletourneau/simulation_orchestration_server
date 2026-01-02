package api

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/aidenletourneau/simulation_orchestration_server/server/internal/logging"
	"github.com/aidenletourneau/simulation_orchestration_server/server/internal/registry"
	"github.com/aidenletourneau/simulation_orchestration_server/server/internal/scenario"
	"github.com/aidenletourneau/simulation_orchestration_server/server/internal/store"
	"github.com/go-chi/chi/v5"
)

// SimulationResponse represents a simulation in the API response
type SimulationResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// HandleGetSimulations returns all connected simulations
func HandleGetSimulations(reg *registry.Registry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		simulations := reg.GetAll()
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
func HandleGetLogs(logStore *logging.LogStore) http.HandlerFunc {
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
func HandleGetScenario(scenarioManager *scenario.ScenarioManager) http.HandlerFunc {
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

// StoredScenarioResponse represents a stored scenario in API response
type StoredScenarioResponse struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
}

// HandleUploadScenario handles YAML scenario file uploads and saves them to the database
func HandleUploadScenario(scenarioManager *scenario.ScenarioManager, scenarioStore *store.ScenarioStore, logStore *logging.LogStore) http.HandlerFunc {
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

		// Validate scenario by loading it
		if err := scenarioManager.LoadScenarioFromBytes(fileBytes); err != nil {
			logStore.LogAndStore("error", "Failed to validate uploaded scenario: %v", err)
			http.Error(w, "Failed to validate scenario: "+err.Error(), http.StatusBadRequest)
			return
		}

		scenario := scenarioManager.GetCurrentScenario()

		// Save to database
		scenarioID, err := scenarioStore.SaveScenario(scenario.Name, string(fileBytes))
		if err != nil {
			logStore.LogAndStore("error", "Failed to save scenario to database: %v", err)
			http.Error(w, "Failed to save scenario: "+err.Error(), http.StatusInternalServerError)
			return
		}

		logStore.LogAndStore("info", "Scenario uploaded and saved to database: %s (ID: %d, %d rules)", scenario.Name, scenarioID, len(scenario.Rules))

		// Return success response
		w.Header().Set("Content-Type", "application/json")
		storedScenario, err := scenarioStore.GetScenarioByID(scenarioID)
		if err != nil {
			http.Error(w, "Failed to retrieve saved scenario: "+err.Error(), http.StatusInternalServerError)
			return
		}

		response := StoredScenarioResponse{
			ID:        storedScenario.ID,
			Name:      storedScenario.Name,
			CreatedAt:  storedScenario.CreatedAt.Format("2006-01-02 15:04:05"),
		}

		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
	}
}

// HandleGetScenarios returns all stored scenarios
func HandleGetScenarios(scenarioStore *store.ScenarioStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		scenarios, err := scenarioStore.GetAllScenarios()
		if err != nil {
			http.Error(w, "Failed to retrieve scenarios: "+err.Error(), http.StatusInternalServerError)
			return
		}

		response := make([]StoredScenarioResponse, len(scenarios))
		for i, s := range scenarios {
			response[i] = StoredScenarioResponse{
				ID:        s.ID,
				Name:      s.Name,
				CreatedAt: s.CreatedAt.Format("2006-01-02 15:04:05"),
			}
		}

		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
	}
}

// ScenarioYAMLResponse represents the YAML content of a scenario
type ScenarioYAMLResponse struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	YAMLContent string `json:"yaml_content"`
	CreatedAt   string `json:"created_at"`
}

// HandleGetScenarioYAML returns the full YAML content of a scenario
func HandleGetScenarioYAML(scenarioStore *store.ScenarioStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		idParam := chi.URLParam(r, "id")
		scenarioID, err := strconv.Atoi(idParam)
		if err != nil {
			http.Error(w, "Invalid scenario ID", http.StatusBadRequest)
			return
		}

		scenario, err := scenarioStore.GetScenarioByID(scenarioID)
		if err != nil {
			http.Error(w, "Scenario not found", http.StatusNotFound)
			return
		}

		response := ScenarioYAMLResponse{
			ID:          scenario.ID,
			Name:        scenario.Name,
			YAMLContent: scenario.YAMLContent,
			CreatedAt:   scenario.CreatedAt.Format("2006-01-02 15:04:05"),
		}

		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
	}
}

// HandleActivateScenario loads and activates a scenario from the database
func HandleActivateScenario(scenarioManager *scenario.ScenarioManager, scenarioStore *store.ScenarioStore, logStore *logging.LogStore) http.HandlerFunc {
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

		idParam := chi.URLParam(r, "id")
		scenarioID, err := strconv.Atoi(idParam)
		if err != nil {
			http.Error(w, "Invalid scenario ID", http.StatusBadRequest)
			return
		}

		scenario, err := scenarioStore.GetScenarioByID(scenarioID)
		if err != nil {
			http.Error(w, "Scenario not found", http.StatusNotFound)
			return
		}

		// Load scenario from YAML content
		if err := scenarioManager.LoadScenarioFromBytes([]byte(scenario.YAMLContent)); err != nil {
			logStore.LogAndStore("error", "Failed to load scenario from database: %v", err)
			http.Error(w, "Failed to load scenario: "+err.Error(), http.StatusInternalServerError)
			return
		}

		loadedScenario := scenarioManager.GetCurrentScenario()
		logStore.LogAndStore("info", "Scenario activated: %s (ID: %d, %d rules)", loadedScenario.Name, scenarioID, len(loadedScenario.Rules))

		// Return success response
		w.Header().Set("Content-Type", "application/json")
		response := ScenarioInfoResponse{
			Name:  loadedScenario.Name,
			Rules: len(loadedScenario.Rules),
		}

		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
	}
}
