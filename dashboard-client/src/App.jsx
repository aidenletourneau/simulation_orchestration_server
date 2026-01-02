import { useState, useEffect } from 'react'

function App() {
  const [simulations, setSimulations] = useState([])
  const [logs, setLogs] = useState([])
  const [serverUrl, setServerUrl] = useState('http://localhost:3000')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState(null)
  const [scenario, setScenario] = useState(null)
  const [uploadStatus, setUploadStatus] = useState(null)
  const [uploading, setUploading] = useState(false)
  const [storedScenarios, setStoredScenarios] = useState([])
  const [selectedScenarioYAML, setSelectedScenarioYAML] = useState(null)
  const [viewingScenarioId, setViewingScenarioId] = useState(null)
  const [activating, setActivating] = useState(false)

  const fetchData = async () => {
    try {
      setError(null)
      
      // Fetch simulations
      const simResponse = await fetch(`${serverUrl}/api/simulations`)
      if (!simResponse.ok) throw new Error('Failed to fetch simulations')
      const simData = await simResponse.json()
      setSimulations(simData)

      // Fetch logs
      const logResponse = await fetch(`${serverUrl}/api/logs`)
      if (!logResponse.ok) throw new Error('Failed to fetch logs')
      const logData = await logResponse.json()
      setLogs(logData)

      // Fetch scenario info
      try {
        const scenarioResponse = await fetch(`${serverUrl}/api/scenario`)
        if (scenarioResponse.ok) {
          const scenarioData = await scenarioResponse.json()
          setScenario(scenarioData)
        } else if (scenarioResponse.status !== 404) {
          throw new Error('Failed to fetch scenario')
        }
      } catch (err) {
        // Scenario fetch is optional, don't fail the whole fetch
        console.warn('Could not fetch scenario:', err)
      }

      // Fetch stored scenarios
      try {
        const storedResponse = await fetch(`${serverUrl}/api/scenarios`)
        if (storedResponse.ok) {
          const storedData = await storedResponse.json()
          setStoredScenarios(storedData)
        }
      } catch (err) {
        // Stored scenarios fetch is optional
        console.warn('Could not fetch stored scenarios:', err)
      }
    } catch (err) {
      setError(err.message)
      console.error('Error fetching data:', err)
    } finally {
      setLoading(false)
    }
  }

  const handleFileUpload = async (event) => {
    const file = event.target.files[0]
    if (!file) return

    // Validate file type
    if (!file.name.endsWith('.yaml') && !file.name.endsWith('.yml')) {
      setUploadStatus({ type: 'error', message: 'Please select a YAML file (.yaml or .yml)' })
      return
    }

    setUploading(true)
    setUploadStatus(null)

    try {
      const formData = new FormData()
      formData.append('scenario', file)

      const response = await fetch(`${serverUrl}/api/scenarios/upload`, {
        method: 'POST',
        body: formData,
      })

      if (!response.ok) {
        const errorText = await response.text()
        throw new Error(errorText || 'Failed to upload scenario')
      }

      const result = await response.json()
      // Upload now returns stored scenario info, refresh to get current scenario info
      setUploadStatus({ 
        type: 'success', 
        message: `Scenario "${result.name}" uploaded successfully! (ID: ${result.id})` 
      })
      
      // Clear the file input
      event.target.value = ''
      
      // Refresh data to show updated logs and scenarios list
      setTimeout(() => {
        fetchData()
      }, 500)
    } catch (err) {
      setUploadStatus({ type: 'error', message: err.message })
      console.error('Error uploading scenario:', err)
    } finally {
      setUploading(false)
    }
  }

  const handleViewYAML = async (scenarioId) => {
    try {
      const response = await fetch(`${serverUrl}/api/scenarios/${scenarioId}`)
      if (!response.ok) {
        throw new Error('Failed to fetch scenario YAML')
      }
      const data = await response.json()
      setSelectedScenarioYAML(data)
      setViewingScenarioId(scenarioId)
    } catch (err) {
      setError(err.message)
      console.error('Error fetching scenario YAML:', err)
    }
  }

  const handleActivateScenario = async (scenarioId) => {
    setActivating(true)
    try {
      const response = await fetch(`${serverUrl}/api/scenarios/${scenarioId}/activate`, {
        method: 'POST',
      })
      if (!response.ok) {
        const errorText = await response.text()
        throw new Error(errorText || 'Failed to activate scenario')
      }
      const result = await response.json()
      setScenario(result)
      setUploadStatus({ 
        type: 'success', 
        message: `Scenario "${result.name}" activated successfully!` 
      })
      
      // Refresh data
      setTimeout(() => {
        fetchData()
      }, 500)
    } catch (err) {
      setUploadStatus({ type: 'error', message: err.message })
      console.error('Error activating scenario:', err)
    } finally {
      setActivating(false)
    }
  }

  useEffect(() => {
    // Initial fetch
    setLoading(true)
    fetchData()

    // Set up auto-refresh every 2 seconds
    const interval = setInterval(() => {
      fetchData()
    }, 2000)

    return () => clearInterval(interval)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [serverUrl])

  const formatTimestamp = (timestamp) => {
    const date = new Date(timestamp)
    return date.toLocaleString()
  }

  const getLogLevelClass = (level) => {
    switch (level.toLowerCase()) {
      case 'error':
        return 'log-error'
      case 'warning':
        return 'log-warning'
      case 'info':
        return 'log-info'
      default:
        return 'log-default'
    }
  }

  return (
    <div>
      <h1>Simulation Orchestration Dashboard</h1>
      
      <div className="section">
        <h2>Server Configuration</h2>
        <input
          type="text"
          value={serverUrl}
          onChange={(e) => setServerUrl(e.target.value)}
          placeholder="Server URL"
          style={{ width: '300px' }}
        />
        <button onClick={() => { setLoading(true); fetchData(); }}>
          Refresh Now
        </button>
        {loading && <span style={{ marginLeft: '10px' }}>Loading...</span>}
        {error && <span style={{ marginLeft: '10px', color: '#cc0000' }}>Error: {error}</span>}
      </div>

      <div className="section">
        <h2>Scenario Management</h2>
        {scenario && (
          <div style={{ marginBottom: '15px', padding: '10px', backgroundColor: '#f0f0f0', borderRadius: '4px' }}>
            <div><strong>Current Scenario:</strong> {scenario.name}</div>
            <div><strong>Rules:</strong> {scenario.rules}</div>
          </div>
        )}
        <div style={{ marginBottom: '20px' }}>
          <label htmlFor="scenario-upload" style={{ display: 'block', marginBottom: '10px' }}>
            Upload YAML Scenario:
          </label>
          <input
            id="scenario-upload"
            type="file"
            accept=".yaml,.yml"
            onChange={handleFileUpload}
            disabled={uploading}
            style={{ marginBottom: '10px' }}
          />
          {uploading && <span style={{ marginLeft: '10px' }}>Uploading...</span>}
          {uploadStatus && (
            <div style={{ 
              marginTop: '10px', 
              padding: '8px', 
              borderRadius: '4px',
              backgroundColor: uploadStatus.type === 'success' ? '#d4edda' : '#f8d7da',
              color: uploadStatus.type === 'success' ? '#155724' : '#721c24'
            }}>
              {uploadStatus.message}
            </div>
          )}
        </div>

        <div>
          <h3>Available Scenarios ({storedScenarios.length})</h3>
          {storedScenarios.length === 0 ? (
            <p>No scenarios available. Upload a scenario to get started.</p>
          ) : (
            <div className="simulations-list">
              {storedScenarios.map((stored) => (
                <div key={stored.id} className="simulation-item">
                  <div style={{ marginBottom: '8px' }}>
                    <strong>{stored.name}</strong>
                  </div>
                  <div style={{ marginBottom: '8px', fontSize: '0.9em', color: '#666' }}>
                    Created: {new Date(stored.created_at).toLocaleString()}
                  </div>
                  <div>
                    <button
                      onClick={() => handleViewYAML(stored.id)}
                      style={{ marginRight: '5px', fontSize: '0.9em' }}
                    >
                      View YAML
                    </button>
                    <button
                      onClick={() => handleActivateScenario(stored.id)}
                      disabled={activating}
                      style={{ fontSize: '0.9em', backgroundColor: '#28a745' }}
                    >
                      {activating ? 'Activating...' : 'Activate'}
                    </button>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>

        {selectedScenarioYAML && (
          <div style={{ marginTop: '20px', padding: '15px', backgroundColor: '#f8f9fa', borderRadius: '4px', border: '1px solid #dee2e6' }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '10px' }}>
              <h3>Scenario YAML: {selectedScenarioYAML.name}</h3>
              <button onClick={() => { setSelectedScenarioYAML(null); setViewingScenarioId(null); }}>Close</button>
            </div>
            <pre style={{ 
              background: '#1e1e1e', 
              color: '#d4d4d4', 
              padding: '15px', 
              borderRadius: '4px', 
              overflow: 'auto', 
              maxHeight: '500px',
              fontFamily: "'Courier New', monospace",
              fontSize: '12px',
              whiteSpace: 'pre-wrap',
              wordWrap: 'break-word'
            }}>
              {selectedScenarioYAML.yaml_content}
            </pre>
          </div>
        )}
      </div>

      <div className="section">
        <h2>Connected Simulations ({simulations.length})</h2>
        {simulations.length === 0 ? (
          <p>No simulations connected</p>
        ) : (
          <div className="simulations-list">
            {simulations.map((sim, index) => (
              <div key={sim.id || index} className="simulation-item">
                <div className="sim-id"><strong>ID:</strong> {sim.id}</div>
                <div className="sim-name"><strong>Name:</strong> {sim.name || 'N/A'}</div>
              </div>
            ))}
          </div>
        )}
      </div>

      <div className="section">
        <h2>Server Logs ({logs.length})</h2>
        <div className="log-container">
          {logs.length === 0 ? (
            <p>No logs available</p>
          ) : (
            logs.map((logEntry, index) => (
              <div key={index} className={`log-entry ${getLogLevelClass(logEntry.level)}`}>
                <span className="log-timestamp">{formatTimestamp(logEntry.timestamp)}</span>
                <span className="log-level">[{logEntry.level}]</span>
                <span className="log-message">{logEntry.message}</span>
              </div>
            ))
          )}
        </div>
      </div>
    </div>
  )
}

export default App
