# Simulation Orchestration Server - MVP

A simple server that routes events between simulations based on YAML configuration.

## Quick Start

### 1. Install Dependencies

```bash
go mod tidy
```

### 2. Configure Environment Variables (Optional)

Copy the example environment file and customize it:

```bash
cp .env.example .env
```

Edit `.env` with your settings. See [Environment Variables](#environment-variables) section below for details.

### 3. Run the Server

**Using environment variables:**
```bash
go run .
```

**Using command line flags:**
```bash
go run . -scenario scenarios/example.yaml -port 3000
```

**Or build first:**
```bash
go build -o simulation_server.exe .
./simulation_server.exe
```

The server will automatically load environment variables from `.env` if present, or use defaults.

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

**For production (PostgreSQL):**
```env
PORT=3000
DATABASE_URL=postgres://username:password@hostname:5432/dbname?sslmode=require
```

See `DATABASE_SETUP.md` for detailed database configuration instructions.

## Testing

See `TESTING.md` for detailed testing instructions.

