package main

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "github.com/lib/pq"
	_ "modernc.org/sqlite"
)

// ScenarioStore handles database operations for scenarios
type ScenarioStore struct {
	db         *sql.DB
	dbType     string // "sqlite" or "postgres"
	driverName string
}

// StoredScenario represents a scenario stored in the database
type StoredScenario struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	YAMLContent string    `json:"yaml_content"`
	CreatedAt   time.Time `json:"created_at"`
}

// NewScenarioStore creates a new scenario store
// connectionString can be:
//   - For SQLite: a file path (e.g., "scenarios.db")
//   - For PostgreSQL: a connection string (e.g., "postgres://user:pass@host:port/dbname?sslmode=disable")
func NewScenarioStore(connectionString string) (*ScenarioStore, error) {
	var db *sql.DB
	var dbType, driverName string
	var err error

	// Detect database type from connection string
	if strings.HasPrefix(connectionString, "postgres://") || strings.HasPrefix(connectionString, "postgresql://") {
		// PostgreSQL connection
		dbType = "postgres"
		driverName = "postgres"
		db, err = sql.Open(driverName, connectionString)
	} else {
		// SQLite connection (file path)
		dbType = "sqlite"
		driverName = "sqlite"
		db, err = sql.Open(driverName, connectionString)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	store := &ScenarioStore{
		db:         db,
		dbType:     dbType,
		driverName: driverName,
	}

	// Create tables if they don't exist
	if err := store.initDB(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	return store, nil
}

// initDB creates the scenarios table if it doesn't exist
func (ss *ScenarioStore) initDB() error {
	var query string

	if ss.dbType == "postgres" {
		// PostgreSQL syntax
		query = `
		CREATE TABLE IF NOT EXISTS scenarios (
			id SERIAL PRIMARY KEY,
			name TEXT NOT NULL,
			yaml_content TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
		`
	} else {
		// SQLite syntax
		query = `
		CREATE TABLE IF NOT EXISTS scenarios (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			yaml_content TEXT NOT NULL,
			created_at TEXT DEFAULT (datetime('now'))
		);
		`
	}

	_, err := ss.db.Exec(query)
	return err
}

// SaveScenario saves a scenario to the database
func (ss *ScenarioStore) SaveScenario(name, yamlContent string) (int, error) {
	var query string
	var result sql.Result
	var err error

	if ss.dbType == "postgres" {
		// PostgreSQL uses $1, $2 for placeholders and RETURNING for last insert ID
		query = `INSERT INTO scenarios (name, yaml_content) VALUES ($1, $2) RETURNING id`
		var id int
		err = ss.db.QueryRow(query, name, yamlContent).Scan(&id)
		if err != nil {
			return 0, err
		}
		return id, nil
	} else {
		// SQLite uses ? for placeholders
		query = `INSERT INTO scenarios (name, yaml_content) VALUES (?, ?)`
		result, err = ss.db.Exec(query, name, yamlContent)
		if err != nil {
			return 0, err
		}

		id, err := result.LastInsertId()
		if err != nil {
			return 0, err
		}

		return int(id), nil
	}
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
		var err error

		if ss.dbType == "postgres" {
			// PostgreSQL returns TIMESTAMP as time.Time directly
			err = rows.Scan(&s.ID, &s.Name, &s.YAMLContent, &s.CreatedAt)
		} else {
			// SQLite returns datetime as string
			var createdAtStr string
			err = rows.Scan(&s.ID, &s.Name, &s.YAMLContent, &createdAtStr)
			if err == nil {
				// Parse SQLite datetime format: "YYYY-MM-DD HH:MM:SS"
				s.CreatedAt, err = time.Parse("2006-01-02 15:04:05", createdAtStr)
				if err != nil {
					s.CreatedAt = time.Now() // Fallback to current time
				}
			}
		}

		if err != nil {
			return nil, err
		}

		scenarios = append(scenarios, s)
	}

	return scenarios, rows.Err()
}

// GetScenarioByID returns a scenario by its ID
func (ss *ScenarioStore) GetScenarioByID(id int) (*StoredScenario, error) {
	var query string
	if ss.dbType == "postgres" {
		query = `SELECT id, name, yaml_content, created_at FROM scenarios WHERE id = $1`
	} else {
		query = `SELECT id, name, yaml_content, created_at FROM scenarios WHERE id = ?`
	}

	row := ss.db.QueryRow(query, id)

	var s StoredScenario
	var err error

	if ss.dbType == "postgres" {
		// PostgreSQL returns TIMESTAMP as time.Time directly
		err = row.Scan(&s.ID, &s.Name, &s.YAMLContent, &s.CreatedAt)
	} else {
		// SQLite returns datetime as string
		var createdAtStr string
		err = row.Scan(&s.ID, &s.Name, &s.YAMLContent, &createdAtStr)
		if err == nil {
			// Parse SQLite datetime format: "YYYY-MM-DD HH:MM:SS"
			s.CreatedAt, err = time.Parse("2006-01-02 15:04:05", createdAtStr)
			if err != nil {
				s.CreatedAt = time.Now() // Fallback to current time
			}
		}
	}

	if err != nil {
		return nil, err
	}

	return &s, nil
}

// DeleteScenario deletes a scenario by ID
func (ss *ScenarioStore) DeleteScenario(id int) error {
	var query string
	if ss.dbType == "postgres" {
		query = `DELETE FROM scenarios WHERE id = $1`
	} else {
		query = `DELETE FROM scenarios WHERE id = ?`
	}
	_, err := ss.db.Exec(query, id)
	return err
}

// Close closes the database connection
func (ss *ScenarioStore) Close() error {
	return ss.db.Close()
}
