package main

import (
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
)

// getEnv gets an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func main() {
	// Load .env file if it exists (ignore errors for local development)
	// In production, environment variables should be set directly
	_ = godotenv.Load()

	// Parse command line flags
	scenarioFile := flag.String("scenario", getEnv("SCENARIO_FILE", "scenarios/example.yaml"), "Path to scenario YAML file")
	port := flag.String("port", getEnv("PORT", "3000"), "Server port")
	flag.Parse()

	// Initialize components
	registry := NewRegistry()
	scenarioManager := NewScenarioManager()
	sagaManager := NewSagaManager(registry)
	logStore := NewLogStore(10000) // Store up to 10000 log entries

	// Initialize scenario store
	// Use DATABASE_URL environment variable if set, otherwise default to SQLite
	dbConnectionString := getEnv("DATABASE_URL", "scenarios.db")
	scenarioStore, err := NewScenarioStore(dbConnectionString)
	if err != nil {
		log.Fatalf("Failed to initialize scenario store: %v", err)
	}
	defer scenarioStore.Close()

	// Create event queue for ordered event processing (prevents race conditions)
	// Buffer size of 1000 should be sufficient for most use cases
	eventQueue := NewEventQueue(1000)

	// Start event queue processor (runs in background goroutine)
	eventQueue.StartProcessor(registry, scenarioManager, sagaManager, logStore)

	// Load initial scenario (optional, can be overridden via API)
	if *scenarioFile != "" {
		if err := scenarioManager.LoadScenario(*scenarioFile); err != nil {
			log.Printf("Warning: Failed to load initial scenario: %v", err)
		} else {
			logStore.LogAndStore("info", "Loaded initial scenario from: %s", *scenarioFile)
		}
	}

	logStore.LogAndStore("info", "Server starting on port %s", *port)
	logStore.LogAndStore("info", "WebSocket endpoint: ws://localhost:%s/ws", *port)

	// Setup router
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Health check endpoint
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Simulation Orchestration Server - MVP"))
	})

	// WebSocket endpoint
	r.Get("/ws", HandleWebSocket(registry, scenarioManager, sagaManager, eventQueue, logStore))

	// API endpoints
	r.Route("/api", func(r chi.Router) {
		r.Get("/simulations", HandleGetSimulations(registry))
		r.Get("/logs", HandleGetLogs(logStore))
		r.Get("/scenario", HandleGetScenario(scenarioManager))
		r.Get("/scenarios", HandleGetScenarios(scenarioStore))
		r.Get("/scenarios/{id}", HandleGetScenarioYAML(scenarioStore))
		r.Post("/scenarios/upload", HandleUploadScenario(scenarioManager, scenarioStore, logStore))
		r.Post("/scenarios/{id}/activate", HandleActivateScenario(scenarioManager, scenarioStore, logStore))
	})

	// Start server
	if err := http.ListenAndServe(":"+*port, r); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
