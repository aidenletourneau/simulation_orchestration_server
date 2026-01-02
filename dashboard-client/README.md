# Simulation Dashboard

A simple React dashboard for monitoring the Simulation Orchestration Server.

## Features

- **Connected Simulations**: Displays all currently connected simulations with their ID and name
- **Server Logs**: Shows all server logs in real-time with color-coded log levels
- **Auto-refresh**: Automatically refreshes data every 2 seconds
- **Manual Refresh**: Button to manually refresh data immediately

## Getting Started

### Install Dependencies

```bash
npm install
```

### Run Development Server

```bash
npm run dev
```

The dashboard will be available at `http://localhost:5174` (different port from test-client to avoid conflicts).

### Build for Production

```bash
npm run build
```

## Configuration

The dashboard connects to the server at `http://localhost:3000` by default. You can change this in the dashboard UI using the server URL input field.

## API Endpoints Used

- `GET /api/simulations` - Fetches all connected simulations
- `GET /api/logs` - Fetches all server logs
