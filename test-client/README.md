# Simulation Test Client - React App

A simple React application for testing the Simulation Orchestration Server.

## Quick Start

### 1. Install Dependencies

```bash
npm install
```

### 2. Run Development Server

```bash
npm run dev
```

The app will be available at `http://localhost:5173`

### 3. Build for Production

```bash
npm run build
```

The built files will be in the `dist` directory.

### 4. Preview Production Build

```bash
npm run preview
```

## Usage

1. Make sure the Simulation Orchestration Server is running (default: `ws://localhost:3000/ws`)
2. Open the React app in your browser
3. Click "Connect" to establish a WebSocket connection
4. Enter a Simulation ID and Name, then click "Register"
5. Select an event type and click "Send Event" to send events to the server
6. View all messages in the log section

## Features

- WebSocket connection management
- Simulation registration
- Event sending with multiple event types
- Real-time message logging with color coding
- Clean, modern React component structure

