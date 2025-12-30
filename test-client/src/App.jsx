import { useState, useRef, useEffect } from 'react'

function App() {
  const [ws, setWs] = useState(null)
  const [connected, setConnected] = useState(false)
  const [registered, setRegistered] = useState(false)
  const [serverUrl, setServerUrl] = useState('ws://localhost:3000/ws')
  const [simId, setSimId] = useState('cyber_sim')
  const [simName, setSimName] = useState('Cyber Range Simulator')
  const [eventType, setEventType] = useState('attack.detected')
  const [logs, setLogs] = useState([])
  const wsRef = useRef(null)

  const log = (message, type = 'info') => {
    const timestamp = new Date().toLocaleTimeString()
    setLogs(prev => [...prev, { timestamp, message, type }])
  }

  const clearLog = () => {
    setLogs([])
  }

  useEffect(() => {
    // Scroll log to bottom when new entry is added
    const logContainer = document.querySelector('.log-container')
    if (logContainer) {
      logContainer.scrollTop = logContainer.scrollHeight
    }
  }, [logs])

  const connect = () => {
    log(`Connecting to ${serverUrl}...`)
    
    const websocket = new WebSocket(serverUrl)
    wsRef.current = websocket
    
    websocket.onopen = () => {
      log('Connected!', 'received')
      setConnected(true)
      setWs(websocket)
    }
    
    websocket.onmessage = (event) => {
      try {
        const msg = JSON.parse(event.data)
        log('Received: ' + JSON.stringify(msg, null, 2), 'received')
        
        if (msg.type === 'registered' && msg.status === 'ok') {
          setRegistered(true)
          log('Registration successful!', 'received')
        } else if (msg.type === 'command') {
          log('COMMAND RECEIVED: ' + msg.command + ' with params: ' + JSON.stringify(msg.params), 'received')
        }
      } catch (error) {
        log('Error parsing message: ' + error.message, 'error')
      }
    }
    
    websocket.onerror = (error) => {
      log('WebSocket error: ' + error, 'error')
    }
    
    websocket.onclose = () => {
      log('Disconnected', 'error')
      setConnected(false)
      setRegistered(false)
      setWs(null)
      wsRef.current = null
    }
  }

  const disconnect = () => {
    if (wsRef.current) {
      wsRef.current.close()
      wsRef.current = null
      setWs(null)
    }
  }

  const register = () => {
    if (!wsRef.current || wsRef.current.readyState !== WebSocket.OPEN) {
      log('Not connected!', 'error')
      return
    }

    const msg = {
      type: 'register',
      id: simId,
      name: simName
    }

    wsRef.current.send(JSON.stringify(msg))
    log('Sent: ' + JSON.stringify(msg), 'sent')
  }

  const sendEvent = () => {
    if (!wsRef.current || wsRef.current.readyState !== WebSocket.OPEN) {
      log('Not connected!', 'error')
      return
    }

    if (!registered) {
      log('Not registered! Please register first.', 'error')
      return
    }

    const msg = {
      type: 'event',
      event_type: eventType,
      source: simId,
      payload: {
        timestamp: new Date().toISOString(),
        severity: 8
      }
    }

    wsRef.current.send(JSON.stringify(msg))
    log('Sent event: ' + JSON.stringify(msg), 'sent')
  }

  return (
    <div>
      <h1>Simulation Orchestration Server - Test Client</h1>
      
      <div className="section">
        <h2>Connection</h2>
        <input
          type="text"
          value={serverUrl}
          onChange={(e) => setServerUrl(e.target.value)}
          placeholder="WebSocket URL"
        />
        <button onClick={connect} disabled={connected}>
          Connect
        </button>
        <button onClick={disconnect} disabled={!connected}>
          Disconnect
        </button>
        <div>Status: {connected ? 'Connected' : 'Not connected'}</div>
      </div>

      <div className="section">
        <h2>Registration</h2>
        <input
          type="text"
          value={simId}
          onChange={(e) => setSimId(e.target.value)}
          placeholder="Simulation ID"
        />
        <input
          type="text"
          value={simName}
          onChange={(e) => setSimName(e.target.value)}
          placeholder="Simulation Name"
        />
        <button onClick={register} disabled={!connected}>
          Register
        </button>
      </div>

      <div className="section">
        <h2>Send Event</h2>
        <select value={eventType} onChange={(e) => setEventType(e.target.value)}>
          <option value="attack.detected">attack.detected</option>
          <option value="sensor.triggered">sensor.triggered</option>
          <option value="emergency.activated">emergency.activated</option>
        </select>
        <button onClick={sendEvent} disabled={!registered}>
          Send Event
        </button>
      </div>

      <div className="section">
        <h2>Log</h2>
        <button onClick={clearLog}>Clear Log</button>
        <div className="log-container">
          {logs.map((logEntry, index) => (
            <div key={index} className={`log-entry log-${logEntry.type}`}>
              {logEntry.timestamp} - {logEntry.message}
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}

export default App

