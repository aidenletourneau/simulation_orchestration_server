# Testing Guide

This guide will help you verify that everything works after the refactoring.

## 1. Build the Application

First, make sure the code compiles:

```bash
cd server
go build ./cmd/server
```

If successful, you should see a `server.exe` (Windows) or `server` (Linux/Mac) executable in the `server` directory.

## 2. Run the Server

Start the server:

```bash
# Windows
.\server.exe -port 3000

# Linux/Mac
./server -port 3000
```

Or with a specific scenario:

```bash
.\server.exe -scenario scenarios/example.yaml -port 3000
```

You should see output like:
```
Loaded scenario: Example Scenario with X rules
Server starting on port 3000
WebSocket endpoint: ws://localhost:3000/ws
```

## 3. Test HTTP Endpoints

Open a new terminal and test the API endpoints:

### Health Check
```bash
curl http://localhost:3000/
```
Should return: `Simulation Orchestration Server - MVP`

### Get Simulations
```bash
curl http://localhost:3000/api/simulations
```
Should return: `[]` (empty array if no simulations connected)

### Get Logs
```bash
curl http://localhost:3000/api/logs
```
Should return: JSON array of log entries

### Get Current Scenario
```bash
curl http://localhost:3000/api/scenario
```
Should return: JSON with scenario name and rule count

### Get All Stored Scenarios
```bash
curl http://localhost:3000/api/scenarios
```
Should return: JSON array of stored scenarios

## 4. Test WebSocket Connection

You can test the WebSocket connection using the test client or a WebSocket tool:

### Using the Test Client

1. Navigate to the `test-client` directory
2. Install dependencies: `npm install`
3. Start the client: `npm run dev`
4. Open the browser and connect to the server

### Using a WebSocket Tool

Use a tool like [WebSocket King](https://websocketking.com/) or browser DevTools:

1. Connect to: `ws://localhost:3000/ws`
2. Send a registration message:
```json
{
  "type": "register",
  "id": "test-sim-1",
  "name": "Test Simulation"
}
```
3. You should receive:
```json
{
  "type": "registered",
  "status": "ok"
}
```

## 5. Test Scenario Upload

Upload a scenario file:

```bash
curl -X POST -F "scenario=@scenarios/example.yaml" http://localhost:3000/api/scenarios/upload
```

Should return: JSON with scenario ID, name, and created_at

## 6. Test Scenario Activation

Activate a scenario by ID:

```bash
curl -X POST http://localhost:3000/api/scenarios/1/activate
```

Replace `1` with the actual scenario ID from the upload response.

## 7. Test Event Processing

1. Connect a simulation via WebSocket (see step 4)
2. Send an event:
```json
{
  "type": "event",
  "event_type": "temperature_high",
  "source": "test-sim-1",
  "payload": {
    "temperature": 85,
    "location": "room-1"
  }
}
```
3. Check the logs endpoint to see if the event was processed
4. If a scenario rule matches, you should see saga creation in the logs

## 8. Verify Database

Check that scenarios are being stored:

```bash
# If using SQLite
sqlite3 scenarios.db "SELECT * FROM scenarios;"
```

## 9. Run All Tests (if available)

```bash
go test ./...
```

## Troubleshooting

### Build Errors
- Make sure you're in the `server` directory
- Run `go mod tidy` to ensure dependencies are up to date
- Check that all imports are correct

### Runtime Errors
- Check that the `scenarios` directory exists
- Verify database connection string (if using PostgreSQL)
- Check port availability (default: 3000)

### WebSocket Errors
- Verify the server is running
- Check CORS settings if connecting from a browser
- Ensure the WebSocket endpoint is `/ws`

## Quick Test Script

Save this as `test-server.ps1` (PowerShell) or `test-server.sh` (Bash):

```powershell
# PowerShell version
Write-Host "Testing server endpoints..."

# Health check
Write-Host "`n1. Health Check:"
Invoke-WebRequest -Uri "http://localhost:3000/" -UseBasicParsing | Select-Object -ExpandProperty Content

# Get simulations
Write-Host "`n2. Get Simulations:"
Invoke-WebRequest -Uri "http://localhost:3000/api/simulations" -UseBasicParsing | Select-Object -ExpandProperty Content

# Get logs
Write-Host "`n3. Get Logs:"
$logs = Invoke-WebRequest -Uri "http://localhost:3000/api/logs" -UseBasicParsing | ConvertFrom-Json
Write-Host "Log count: $($logs.Count)"

# Get scenario
Write-Host "`n4. Get Scenario:"
Invoke-WebRequest -Uri "http://localhost:3000/api/scenario" -UseBasicParsing | Select-Object -ExpandProperty Content

Write-Host "`nAll tests completed!"
```

Run it after starting the server:
```powershell
.\test-server.ps1
```
