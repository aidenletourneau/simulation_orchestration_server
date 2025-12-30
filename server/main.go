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

	// Load scenario
	if err := scenarioManager.LoadScenario(*scenarioFile); err != nil {
		log.Fatalf("Failed to load scenario: %v", err)
	}

	// Setup router
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Health check endpoint
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Simulation Orchestration Server - MVP"))
	})

	// WebSocket endpoint
	r.Get("/ws", HandleWebSocket(registry, scenarioManager, sagaManager))

	// Start server
	log.Printf("Server starting on port %s", *port)
	log.Printf("WebSocket endpoint: ws://localhost:%s/ws", *port)
	if err := http.ListenAndServe(":"+*port, r); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
