# Simulation Orchestration Server - MVP

A simple server that routes events between simulations based on YAML configuration.

## Quick Start

### 1. Install Dependencies

```bash
go mod tidy
```

### 2. Run the Server

```bash
go run . -scenario scenarios/example.yaml -port 3000
```

Or build first:
```bash
go build -o simulation_server.exe .
./simulation_server.exe -scenario scenarios/example.yaml -port 3000
```

### 3. Test It

Open `test_client.html` in a web browser and:
1. Click "Connect"
2. Register with ID `cyber_sim` and name "Cyber Range"
3. Send an event (e.g., `attack.detected`)
4. Open another browser window, register as `vr_sim`
5. Send an event from `cyber_sim` and watch `vr_sim` receive commands!

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

## How It Works

1. **Simulations connect** via WebSocket to `/ws`
2. **Simulations register** with an ID and name
3. **Simulations send events** like `{"type":"event","event_type":"attack.detected",...}`
4. **Server matches events** to rules in the YAML scenario file
5. **Server forwards commands** to target simulations based on rules

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

## Testing

See `TESTING.md` for detailed testing instructions.

