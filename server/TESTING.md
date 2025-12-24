# Testing the MVP

## Quick Start

### 1. Start the Server

```bash
cd server
go run . -scenario scenarios/example.yaml -port 3000
```

Or build and run:
```bash
go build -o simulation_server.exe .
./simulation_server.exe -scenario scenarios/example.yaml -port 3000
```

The server will start on port 3000 and load the scenario file.

### 2. Test with Web Browser

1. Open `test_client.html` in a web browser (or serve it with a simple HTTP server)
2. Click "Connect" to connect to the server
3. Enter a simulation ID (e.g., `cyber_sim`, `vr_sim`, `sensor_sim`)
4. Enter a simulation name
5. Click "Register"
6. Select an event type and click "Send Event"
7. Watch the log to see commands being sent to other simulations

### 3. Test with Multiple Clients

To test event routing between simulations:

1. Open `test_client.html` in **two different browser windows/tabs**
2. In Window 1:
   - Connect
   - Register as `cyber_sim` with name "Cyber Range"
   - Send event: `attack.detected`
3. In Window 2:
   - Connect
   - Register as `vr_sim` with name "VR Environment"
   - Wait for commands to arrive (you should see a command when cyber_sim sends an event)

### 4. Test Scenario Rules

The example scenario (`scenarios/example.yaml`) has these rules:

- **Rule 1**: When `cyber_sim` sends `attack.detected` → sends `show_alert` command to `vr_sim`
- **Rule 2**: When `sensor_sim` sends `sensor.triggered` → sends commands to both `cyber_sim` and `vr_sim`
- **Rule 3**: When any simulation sends `emergency.activated` → sends `emergency_mode` to both `vr_sim` and `cyber_sim`

### 5. Test with Command Line (using wscat or similar)

If you have `wscat` installed:

```bash
# Connect and register
wscat -c ws://localhost:3000/ws
> {"type":"register","id":"cyber_sim","name":"Cyber Sim"}

# Send an event
> {"type":"event","event_type":"attack.detected","source":"cyber_sim","payload":{"severity":8}}
```

## Expected Behavior

1. **Registration**: Server responds with `{"type":"registered","status":"ok"}`
2. **Event Processing**: When an event matches a rule, the server logs it and sends commands to target simulations
3. **Command Delivery**: Target simulations receive `{"type":"command","command":"...","params":{...}}`

## Troubleshooting

- **Connection refused**: Make sure the server is running on port 3000
- **No commands received**: Check that:
  - The event type matches a rule in the scenario file
  - The `from` field in the rule matches your simulation ID (if specified)
  - The target simulation is registered
- **Scenario not loading**: Check that `scenarios/example.yaml` exists and is valid YAML

