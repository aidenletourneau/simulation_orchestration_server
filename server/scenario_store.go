package main

import (
	"database/sql"
	"time"

	_ "modernc.org/sqlite"
)

// ScenarioStore handles database operations for scenarios
type ScenarioStore struct {
	db *sql.DB
}

// StoredScenario represents a scenario stored in the database
type StoredScenario struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	YAMLContent string    `json:"yaml_content"`
	CreatedAt   time.Time `json:"created_at"`
}

// NewScenarioStore creates a new scenario store with SQLite database
func NewScenarioStore(dbPath string) (*ScenarioStore, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	store := &ScenarioStore{db: db}

	// Create tables if they don't exist
	if err := store.initDB(); err != nil {
		db.Close()
		return nil, err
	}

	return store, nil
}

// initDB creates the scenarios table if it doesn't exist
func (ss *ScenarioStore) initDB() error {
	query := `
	CREATE TABLE IF NOT EXISTS scenarios (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		yaml_content TEXT NOT NULL,
		created_at TEXT DEFAULT (datetime('now'))
	);
	`
	_, err := ss.db.Exec(query)
	return err
}

// SaveScenario saves a scenario to the database
func (ss *ScenarioStore) SaveScenario(name, yamlContent string) (int, error) {
	query := `INSERT INTO scenarios (name, yaml_content) VALUES (?, ?)`
	result, err := ss.db.Exec(query, name, yamlContent)
	if err != nil {
		return 0, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	return int(id), nil
}

// GetAllScenarios returns all scenarios from the database
func (ss *ScenarioStore) GetAllScenarios() ([]StoredScenario, error) {
	query := `SELECT id, name, yaml_content, created_at FROM scenarios ORDER BY created_at DESC`
	rows, err := ss.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var scenarios []StoredScenario
	for rows.Next() {
		var s StoredScenario
		var createdAtStr string
		err := rows.Scan(&s.ID, &s.Name, &s.YAMLContent, &createdAtStr)
		if err != nil {
			return nil, err
		}

		// Parse SQLite datetime format: "YYYY-MM-DD HH:MM:SS"
		s.CreatedAt, err = time.Parse("2006-01-02 15:04:05", createdAtStr)
		if err != nil {
			s.CreatedAt = time.Now() // Fallback to current time
		}

		scenarios = append(scenarios, s)
	}

	return scenarios, rows.Err()
}

// GetScenarioByID returns a scenario by its ID
func (ss *ScenarioStore) GetScenarioByID(id int) (*StoredScenario, error) {
	query := `SELECT id, name, yaml_content, created_at FROM scenarios WHERE id = ?`
	row := ss.db.QueryRow(query, id)

	var s StoredScenario
	var createdAtStr string
	err := row.Scan(&s.ID, &s.Name, &s.YAMLContent, &createdAtStr)
	if err != nil {
		return nil, err
	}

	// Parse SQLite datetime format: "YYYY-MM-DD HH:MM:SS"
	s.CreatedAt, err = time.Parse("2006-01-02 15:04:05", createdAtStr)
	if err != nil {
		s.CreatedAt = time.Now() // Fallback to current time
	}

	return &s, nil
}

// DeleteScenario deletes a scenario by ID
func (ss *ScenarioStore) DeleteScenario(id int) error {
	query := `DELETE FROM scenarios WHERE id = ?`
	_, err := ss.db.Exec(query, id)
	return err
}

// Close closes the database connection
func (ss *ScenarioStore) Close() error {
	return ss.db.Close()
}
