package main

import "github.com/gorilla/websocket"

// Event represents an incoming event from a simulation
type Event struct {
	Type      string                 `json:"type"`
	EventType string                 `json:"event_type"`
	Source    string                 `json:"source"`
	Payload   map[string]interface{} `json:"payload"`
}

// Command represents an outgoing command to a simulation
type Command struct {
	Type    string                 `json:"type"`
	Command string                 `json:"command"`
	Params  map[string]interface{} `json:"params"`
}

// Simulation represents a connected simulation client
type Simulation struct {
	ID         string
	Name       string
	Connection *websocket.Conn
}

// Message represents a WebSocket message
type Message struct {
	Type      string                 `json:"type"`
	ID        string                 `json:"id,omitempty"`
	Name      string                 `json:"name,omitempty"`
	EventType string                 `json:"event_type,omitempty"`
	Source    string                 `json:"source,omitempty"`
	Payload   map[string]interface{} `json:"payload,omitempty"`
	Command   string                 `json:"command,omitempty"`
	Params    map[string]interface{} `json:"params,omitempty"`
	Status    string                 `json:"status,omitempty"`
}

// Scenario represents the loaded YAML scenario
type Scenario struct {
	Name  string `yaml:"name"`
	Rules []Rule `yaml:"rules"`
}

// Rule represents a trigger-action rule
type Rule struct {
	When WhenCondition `yaml:"when"`
	Then []Action      `yaml:"then"`
}

// WhenCondition defines when a rule should fire
type WhenCondition struct {
	EventType string `yaml:"event_type"`
	From      string `yaml:"from,omitempty"`
}

// Action defines what to do when rule fires
type Action struct {
	SendTo  string                 `yaml:"send_to"`
	Command string                 `yaml:"command"`
	Params  map[string]interface{} `yaml:"params"`
}

