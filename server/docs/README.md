## Project Structure

```
server/
├── main.go              # Server entry point
├── models.go            # Data structures
├── registry.go          # Simulation connection registry
├── scenario.go          # YAML scenario loader and rule matcher
├── websocket.go         # WebSocket handler
├── scenarios/
│   └── example.yaml     # Example scenario configuration
├── test_client.html     # Browser-based test client
└── go.mod               # Go dependencies
```

## Architecture Overview

The Simulation Orchestration Server coordinates multiple simulations through event-driven orchestration. It ensures consistency through two key mechanisms:

1. **Event Queue**: Processes events sequentially to prevent race conditions
2. **Saga Pattern**: Manages distributed transactions across simulations with automatic rollback on failure

### System Architecture Diagram

```
┌─────────────┐         ┌──────────────────────────────────────┐         ┌─────────────┐
│ Simulation  │         │      Orchestration Server            │         │ Simulation  │
│     A       │◄────────┤                                      ├────────►│     B       │
└─────────────┘         │  ┌────────────┐  ┌──────────────┐  │         └─────────────┘
                        │  │   Event    │  │     Saga     │  │
                        │  │   Queue    │  │   Manager    │  │
                        │  │ (FIFO)     │  │              │  │
                        │  └────────────┘  └──────────────┘  │
                        │                                      │
                        │  ┌────────────┐  ┌──────────────┐  │
                        │  │  Scenario  │  │  Registry   │  │
                        │  │  Manager   │  │              │  │
                        │  └────────────┘  └──────────────┘  │
                        └──────────────────────────────────────┘
```

### Event Flow Diagram

```
Simulation A                    Server                          Simulation B
     │                            │                                 │
     │─── WebSocket Connect ─────►│                                 │
     │                            │                                 │
     │─── Register ──────────────►│                                 │
     │  {type: "register",        │                                 │
     │   id: "sim_a",             │                                 │
     │   name: "Simulation A"}    │                                 │
     │                            │                                 │
     │◄── Registered ─────────────│                                 │
     │  {type: "registered",     │                                 │
     │   status: "ok"}            │                                 │
     │                            │                                 │
     │─── Send Event ────────────►│                                 │
     │  {type: "event",           │                                 │
     │   event_type: "attack...", │                                 │
     │   payload: {...}}          │                                 │
     │                            │                                 │
     │                            │─── Enqueue Event ──────────────►│
     │                            │    (Event Queue)                │
     │                            │                                 │
     │                            │─── Process Event ───────────────│
     │                            │    (Match Rules)                │
     │                            │                                 │
     │                            │─── Create Saga ─────────────────│
     │                            │    (Multi-step Transaction)     │
     │                            │                                 │
     │                            │─── Send Command ──────────────►│
     │                            │  {type: "command",              │
     │                            │   command: "show_alert",        │
     │                            │   params: {...},                │
     │                            │   saga_id: "...",               │
     │                            │   step_id: 0}                    │
     │                            │                                 │
     │                            │◄── Step Completed ─────────────│
     │                            │  {type: "step.completed",       │
     │                            │   saga_id: "...",                │
     │                            │   step_id: 0}                    │
     │                            │                                 │
     │                            │─── Next Step ──────────────────►│
     │                            │    (if more steps)               │
```

## How It Works

1. **Simulations connect** via WebSocket to `/ws`
2. **Simulations register** with an ID and name
3. **Simulations send events** like `{"type":"event","event_type":"attack.detected",...}`
4. **Server enqueues events** in the Event Queue for sequential processing
5. **Server matches events** to rules in the YAML scenario file
6. **Server creates a Saga** to manage multi-step transactions
7. **Server forwards commands** to target simulations based on rules
8. **Simulations acknowledge** step completion/failure
9. **Server advances Saga** or triggers compensation on failure

## Scenario YAML Format

```yaml
scenario:
  name: "My Scenario"
  rules:
    - when:
        event_type: "attack.detected"
        from: "cyber_sim"  # optional: specific simulation
      then:
        - send_to: "vr_sim"
          command: "show_alert"
          params:
            message: "Attack detected!"
```

See `scenarios/example.yaml` for more examples.

## Environment Variables

The server supports configuration via environment variables or a `.env` file. Create a `.env` file in the `server/` directory based on `.env.example`.

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | Server port | `3000` |
| `DATABASE_URL` | Database connection string. For SQLite: file path (e.g., `scenarios.db`). For PostgreSQL: connection string (e.g., `postgres://user:pass@host:5432/dbname?sslmode=require`) | `scenarios.db` |
| `SCENARIO_FILE` | Path to initial scenario YAML file to load on startup | `scenarios/example.yaml` |

**Example `.env` file:**
```env
PORT=3000
DATABASE_URL=scenarios.db
SCENARIO_FILE=scenarios/example.yaml
```

## Connecting Simulations

### WebSocket Connection

Simulations connect to the server via WebSocket at:
```
ws://localhost:3000/ws
```

### Connection Protocol

#### 1. Establish WebSocket Connection

First, establish a WebSocket connection to the server endpoint.

**Example (Python):**
```python
import websocket
import json

ws = websocket.WebSocket()
ws.connect("ws://localhost:3000/ws")
```

#### 2. Register Simulation

Immediately after connecting, send a registration message. The server expects registration as the first message.

**Message Format:**
```json
{
  "type": "register",
  "id": "simulation_id",
  "name": "Simulation Name"
}
```

**Example (Python):**
```python
register_msg = {
    "type": "register",
    "id": "cyber_sim",
    "name": "Cyber Range Simulation"
}
ws.send(json.dumps(register_msg))
```

**Server Response:**
```json
{
  "type": "registered",
  "status": "ok"
}
```

#### 3. Send Events

After registration, simulations can send events that trigger scenario rules.

**Message Format:**
```json
{
  "type": "event",
  "event_type": "attack.detected",
  "payload": {
    "severity": 8,
    "source_ip": "192.168.1.100",
    "timestamp": "2024-01-01T12:00:00Z"
  }
}
```

**Example (Python):**
```python
event_msg = {
    "type": "event",
    "event_type": "attack.detected",
    "payload": {
        "severity": 8,
        "source_ip": "192.168.1.100"
    }
}
ws.send(json.dumps(event_msg))
```

#### 4. Receive Commands

Simulations receive commands from the server when events match scenario rules.

**Command Message Format:**
```json
{
  "type": "command",
  "command": "show_alert",
  "params": {
    "message": "Cyber attack detected!",
    "severity": 8
  },
  "saga_id": "saga_1234567890",
  "step_id": 0
}
```

**Example Handler (Python):**
```python
def handle_message(ws, message):
    msg = json.loads(message)
    
    if msg['type'] == 'command':
        command = msg['command']
        params = msg['params']
        saga_id = msg['saga_id']
        step_id = msg['step_id']
        
        # Execute the command
        execute_command(command, params)
        
        # Acknowledge step completion
        ack = {
            "type": "step.completed",
            "saga_id": saga_id,
            "step_id": step_id
        }
        ws.send(json.dumps(ack))
```

#### 5. Acknowledge Step Completion

When a simulation successfully completes a command, it must send a `step.completed` message.

**Message Format:**
```json
{
  "type": "step.completed",
  "saga_id": "saga_1234567890",
  "step_id": 0
}
```

#### 6. Report Step Failure

If a simulation cannot complete a command, it should send a `step.failed` message. This triggers compensation (rollback) of all previous steps in the saga.

**Message Format:**
```json
{
  "type": "step.failed",
  "saga_id": "saga_1234567890",
  "step_id": 0
}
```

## Consistency Mechanisms

### Event Queue

The Event Queue ensures **ordered, sequential processing** of events from all simulations, preventing race conditions when multiple events arrive concurrently.

**How it works:**
1. Events from all simulations are enqueued in a FIFO (First-In-First-Out) queue
2. A single background processor dequeues and processes events one at a time
3. This guarantees predictable ordering and prevents concurrent rule evaluation conflicts

**Event Queue Flow:**
```
Simulation A ──┐
                │
Simulation B ──┼──► [Event Queue] ──► [Processor] ──► [Scenario Matching] ──► [Saga Creation]
                │      (FIFO)          (Sequential)
Simulation C ──┘
```

**Benefits:**
- **Prevents race conditions**: Only one event is processed at a time
- **Deterministic ordering**: Events are processed in the order they arrive
- **Thread-safe**: All event processing happens in a single goroutine

### Saga Pattern

The Saga Pattern ensures **eventual consistency** across distributed transactions involving multiple simulations. It guarantees that either all steps complete successfully, or all completed steps are rolled back.

**How it works:**
1. When an event matches a rule with multiple actions, a Saga is created
2. Steps execute sequentially, waiting for completion before proceeding
3. Each step locks the target simulation to prevent concurrent sagas
4. If any step fails, compensation commands are sent in reverse order
5. Simulations acknowledge completion/failure via `step.completed` or `step.failed`

**Saga Lifecycle:**
```
Event Received
    │
    ▼
Create Saga (with all steps)
    │
    ▼
Acquire Locks (for all target simulations)
    │
    ▼
Dispatch Step 0 ──────────────► Simulation A
    │                                │
    │                                │ Execute Command
    │                                │
    │◄────── step.completed ──────────┘
    │
    ▼
Dispatch Step 1 ──────────────► Simulation B
    │                                │
    │                                │ Execute Command
    │                                │
    │◄────── step.completed ──────────┘
    │
    ▼
Dispatch Step 2 ──────────────► Simulation C
    │                                │
    │                                │ Execute Command
    │                                │
    │◄────── step.failed ─────────────┘
    │
    ▼
Trigger Compensation
    │
    ├──► Compensate Step 1 ────────► Simulation B
    │
    └──► Compensate Step 0 ────────► Simulation A
```

**Saga States:**
- **Pending**: Saga created, first step about to be dispatched
- **InProgress**: One or more steps are executing
- **Completed**: All steps completed successfully
- **Failed**: A step failed, compensation triggered
- **Compensating**: Compensation commands being sent

**Simulation Locking:**
- Each simulation can only be involved in one active saga at a time
- Locks are acquired when a saga is created
- Locks are released when the saga completes or fails
- This prevents conflicting concurrent operations on the same simulation

**Compensation:**
- If a step fails, all previously completed steps are compensated
- Compensation commands are sent in reverse order (most recent first)
- Compensation commands are defined in the scenario YAML:
  ```yaml
  - send_to: "simulation_id"
    command: "forward_action"
    params: {...}
    compensate_command: "rollback_action"  # Optional
    compensate_params: {...}               # Optional
  ```

## Message Reference

### Outgoing Messages (Simulation → Server)

#### Registration
```json
{
  "type": "register",
  "id": "simulation_id",
  "name": "Simulation Name"
}
```

#### Event
```json
{
  "type": "event",
  "event_type": "event.type.name",
  "payload": {
    "key": "value"
  }
}
```

#### Step Completed
```json
{
  "type": "step.completed",
  "saga_id": "saga_1234567890",
  "step_id": 0
}
```

#### Step Failed
```json
{
  "type": "step.failed",
  "saga_id": "saga_1234567890",
  "step_id": 0
}
```

### Incoming Messages (Server → Simulation)

#### Registration Confirmation
```json
{
  "type": "registered",
  "status": "ok"
}
```

#### Command
```json
{
  "type": "command",
  "command": "command_name",
  "params": {
    "key": "value"
  },
  "saga_id": "saga_1234567890",
  "step_id": 0
}
```

#### Error
```json
{
  "type": "error",
  "status": "error_message"
}
```