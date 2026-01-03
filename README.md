# Multi-Simulation Orchestration Server

## 1. Overview

Department of Energy (DoE) training exercises frequently rely on multiple independent simulation systems, such as cyber ranges, VR/AR emergency response environments, and physical facility or sensor simulations. While standards like HLA, DIS, and TENA enable data-level interoperability, they do not provide a simple, reusable mechanism for scenario-level orchestration, defining how events in one simulation should trigger coordinated actions in others.

This project proposes a cloud-hosted orchestration server written in Go that acts as a central “scenario brain.” The server ingests real-time events from connected simulations, evaluates them against a declarative YAML scenario file, and dispatches commands to other simulations accordingly. The system is designed to be cloud-native, extensible, and security-aware, making it suitable as a prototype for future DoE multi-domain training environments.

---

## 2. Goals

### Goals
- Enable **scenario-level orchestration** across heterogeneous simulations.
- Provide **real-time event-driven coordination** using a lightweight communication layer.
- Support **pluggable simulation clients** (cyber, VR, physical/sensor).
- Be **cloud deployable** using Docker and AWS.


## 3. High-Level Architecture

### Core Components
1. **Client Gateway**
   - Accepts real-time connections from simulation clients.
   - Supports WebSockets (recommended for demo) or gRPC streaming.

2. **Simulation Registry**
   - Tracks connected simulations, their identities, capabilities, and current status.

3. **Event Bus (Internal)**
   - Normalizes incoming events and routes them to the scenario engine.
   - Initially implemented with in-memory channels; extensible to NATS or Redis Streams.

4. **Scenario Engine**
   - Loads and validates YAML scenario files.
   - Maintains scenario state and phases.
   - Evaluates trigger rules and determines which actions to execute.

5. **Dispatcher**
   - Sends commands to targeted simulations based on rule evaluation.

6. **Audit Log**
   - Records all events, rule firings, and dispatched actions for after-action review.

---

## 4. Technology Choices and Rationale

### Language: Go
- Strong concurrency primitives (goroutines, channels).
- High performance and small memory footprint.
- Widely used for cloud-native and distributed systems.

### HTTP Routing: `chi`
- Lightweight, idiomatic Go router.
- Well-suited for REST APIs and middleware-based services.

### Configuration Management: `joho/godotenv`
- Supports environment variable loading from `.env` files.
- Useful for switching between local/dev/cloud configurations.

### YAML Parsing: `gopkg.in/yaml.v3`
- Canonical YAML parsing library for Go.
- Supports strict decoding and schema validation.

### Database: `modernc.org/sqlite` and `lib/pq`
- SQLite for local development and lightweight deployments.
- PostgreSQL support via `lib/pq` for production environments.

### Real-Time Communication
- **WebSockets** (primary choice): easy integration with Unity and Godot clients.
- **gRPC streaming** (future option): strong typing and service-to-service integration.

### Deployment
- Docker for packaging.
- AWS EC2 or ECS for hosting.

---

## 5. Simplified Design Document

### 5.1 Core Concept

The server acts as a **message router** between simulations:
1. Simulations connect via WebSocket
2. When Simulation A sends an event, the server checks the YAML scenario file
3. If the event matches a rule, the server forwards commands to other simulations (B, C, etc.)

### 5.2 Simple Architecture

```
Simulation A  ──WebSocket──>  Server  ──WebSocket──>  Simulation B
   (sends event)              (reads YAML,              (receives command)
                               matches rules)
```

### 5.3 Components

#### 5.3.1 WebSocket Handler
**What it does**: Accepts connections from simulations and handles message passing.

**Simple flow**:
1. Simulation connects to `/ws`
2. Simulation sends registration: `{"type": "register", "id": "sim1", "name": "Cyber Sim"}`
3. Server stores the connection
4. Simulation can send events: `{"type": "event", "event_type": "attack.detected", "payload": {...}}`
5. Server can send commands: `{"type": "command", "command": "trigger_alert", "params": {...}}`

#### 5.3.2 Simulation Registry
**What it does**: Keeps track of which simulations are connected.

**Simple structure**:
```go
type Simulation struct {
    ID         string
    Name       string
    Connection *websocket.Conn
}

// Just a map: map[string]*Simulation
```

#### 5.3.3 Scenario Engine
**What it does**: Reads YAML file and matches incoming events to rules.

**YAML Format** (simplified):
```yaml
scenario:
  rules:
    - when:
        event_type: "attack.detected"
        from: "cyber_sim"  # optional: which simulation sent it
      then:
        - send_to: "vr_sim"
          command: "show_alert"
          params:
            message: "Cyber attack detected!"
        
        - send_to: "sensor_sim"
          command: "activate"
          params:
            duration: 60
```

**How it works**:
1. Load YAML file at startup
2. When event arrives, check each rule
3. If `event_type` matches and `from` matches (if specified), execute `then` actions
4. For each action, send command to target simulation via WebSocket

### 5.4 Data Structures

```go
// Event from simulation
type Event struct {
    Type      string                 // "attack.detected", "sensor.triggered", etc.
    Source    string                 // Which simulation sent it
    Payload   map[string]interface{} // Any data
}

// Command to simulation
type Command struct {
    Command   string                 // "show_alert", "activate", etc.
    Params    map[string]interface{} // Command parameters
}

// Simulation connection
type Simulation struct {
    ID         string
    Name       string
    Connection *websocket.Conn
}

// YAML Rule
type Rule struct {
    When  WhenCondition
    Then  []Action
}

type WhenCondition struct {
    EventType string  // "attack.detected"
    From      string  // "cyber_sim" (optional)
}

type Action struct {
    SendTo  string                 // Target simulation ID
    Command string                 // Command name
    Params  map[string]interface{} // Command parameters
}
```

### 5.5 Message Flow Example

1. **Cyber simulation connects**:
   ```
   Client → Server: {"type": "register", "id": "cyber_sim", "name": "Cyber Range"}
   Server → Client: {"type": "registered", "status": "ok"}
   ```

2. **VR simulation connects**:
   ```
   Client → Server: {"type": "register", "id": "vr_sim", "name": "VR Environment"}
   Server → Client: {"type": "registered", "status": "ok"}
   ```

3. **Cyber simulation sends event**:
   ```
   Cyber → Server: {"type": "event", "event_type": "attack.detected", "payload": {"severity": 8}}
   ```

4. **Server processes event**:
   - Checks YAML rules
   - Finds matching rule: `when: event_type: "attack.detected"`
   - Executes action: `send_to: "vr_sim", command: "show_alert"`

5. **Server sends command to VR simulation**:
   ```
   Server → VR: {"type": "command", "command": "show_alert", "params": {"message": "Cyber attack detected!"}}
   ```

---

## 6. Simplified Implementation Plan

### 6.1 Project Structure

```
server/
├── cmd/
│   └── server/
│       └── main.go            # Entry point
├── internal/
│   ├── api/
│   │   └── handlers.go        # HTTP API handlers
│   ├── logging/
│   │   └── log_store.go       # Audit log storage
│   ├── models/
│   │   └── models.go          # Data structures
│   ├── queue/
│   │   └── event_queue.go     # Event bus implementation
│   ├── registry/
│   │   └── registry.go        # Simulation registry
│   ├── saga/
│   │   └── saga.go           # Saga pattern for multi-step actions
│   ├── scenario/
│   │   └── scenario.go       # YAML loader and rule matcher
│   ├── store/
│   │   └── scenario_store.go # Scenario persistence
│   └── websocket/
│       ├── websocket.go       # WebSocket handler
│       └── event_handler.go   # Event processing
├── scenarios/
│   ├── disaster_response.yaml
│   ├── example.yaml
│   ├── industrial_automation.yaml
│   ├── security_monitoring.yaml
│   ├── sensor_network.yaml
│   ├── smart_building.yaml
│   └── training_exercise.yaml
├── docs/
│   ├── README.md
│   └── YAML_SCENARIO_LANGUAGE.md
├── docker-compose.yml
├── docker-compose.local.yml
├── Dockerfile
├── go.mod
└── go.sum
```

### 6.2 Implementation Steps

#### Step 1: Basic WebSocket Server
**Goal**: Accept WebSocket connections

**Tasks**:
1. Set up HTTP server with chi router
2. Add WebSocket upgrade handler at `/ws`
3. Accept connections and store them
4. Handle basic JSON message sending/receiving

**Dependencies**:
- `github.com/go-chi/chi/v5`
- `github.com/gorilla/websocket`
- `github.com/joho/godotenv`

#### Step 2: Simulation Registry
**Goal**: Track connected simulations

**Tasks**:
1. Create `Simulation` struct
2. Create map to store connections: `map[string]*Simulation`
3. Add mutex for thread safety
4. Implement `Register(id, name, conn)` and `Get(id)` functions
5. Handle disconnections (remove from map)

#### Step 3: YAML Scenario Loader
**Goal**: Load and parse YAML scenario file

**Tasks**:
1. Define Go structs matching YAML format
2. Use `gopkg.in/yaml.v3` to load YAML file
3. Parse rules into memory
4. Load scenario at server startup

**Dependencies**:
- `gopkg.in/yaml.v3`
- `modernc.org/sqlite` (for scenario persistence)
- `github.com/lib/pq` (for PostgreSQL support)

#### Step 4: Event Processing
**Goal**: Match events to rules and send commands

**Tasks**:
1. When event arrives from simulation:
   - Extract `event_type` and `source`
   - Loop through all rules in scenario
   - Check if `when.event_type` matches
   - Check if `when.from` matches (if specified)
   - If match: execute all actions in `then`
2. For each action:
   - Look up target simulation in registry
   - Send command via WebSocket connection

#### Step 5: Testing & Polish
**Goal**: Make it work end-to-end

**Tasks**:
1. Create example scenario YAML file
2. Test with 2-3 mock simulations
3. Add basic error handling (missing simulation, invalid messages)
4. Add simple logging (fmt.Println or basic logger)


### 6.3 Example Scenario YAML

```yaml
scenario:
  name: "Simple Test Scenario"
  rules:
    # When cyber sim detects attack, alert VR sim
    - when:
        event_type: "attack.detected"
        from: "cyber_sim"
      then:
        - send_to: "vr_sim"
          command: "show_alert"
          params:
            message: "Cyber attack detected!"
            severity: 8
    
    # When sensor triggers, notify both cyber and VR
    - when:
        event_type: "sensor.triggered"
        from: "sensor_sim"
      then:
        - send_to: "cyber_sim"
          command: "log_event"
          params:
            event: "sensor_triggered"
        - send_to: "vr_sim"
          command: "update_status"
          params:
            status: "sensor_active"
```
