package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	// Parse command line flags
	scenarioFile := flag.String("scenario", "scenarios/example.yaml", "Path to scenario YAML file")
	port := flag.String("port", "3000", "Server port")
	flag.Parse()

	// Initialize components
	registry := NewRegistry()
	scenarioManager := NewScenarioManager()
	sagaManager := NewSagaManager(registry)
	logStore := NewLogStore(10000) // Store up to 10000 log entries

	// Create event queue for ordered event processing (prevents race conditions)
	// Buffer size of 1000 should be sufficient for most use cases
	eventQueue := NewEventQueue(1000)

	// Start event queue processor (runs in background goroutine)
	eventQueue.StartProcessor(registry, scenarioManager, sagaManager, logStore)

	// Load scenario
	if err := scenarioManager.LoadScenario(*scenarioFile); err != nil {
		log.Fatalf("Failed to load scenario: %v", err)
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
	})

	// Start server
	if err := http.ListenAndServe(":"+*port, r); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
