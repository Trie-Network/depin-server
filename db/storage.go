package db

import (
	"database/sql"
	"fmt"
	"sync"

	_ "github.com/mattn/go-sqlite3"
)

type InferenceStorage struct {
	mu        sync.Mutex
	db        *sql.DB
	threshold int
}

type InferenceRecord struct {
	ID        string `json:"id"`
	Did       string `json:"did"`
	Timestamp string `json:"timestamp"`
	Signature string `json:"signature"`
	Query     string `json:"query"`
	AssetID   string `json:"asset_id"`
	AssetValue string `json:"asset_value"`	
}

func applyDBConfig(db *sql.DB) error {
	// Set the busy timeout to 5 seconds
	if _, err := db.Exec("PRAGMA busy_timeout = 5000"); err != nil {
		return fmt.Errorf("failed to set busy timeout: %v", err)
	}
	return nil
}

func NewStorage(dbPath string, threshold int) (*InferenceStorage, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
	}

	if err := applyDBConfig(db); err != nil {
		return nil, err
	}

	// Create table if not exists
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS inference_record_queue (
			id TEXT PRIMARY KEY,
			did TEXT NOT NULL,
			timestamp TEXT NOT NULL,
			signature TEXT NOT NULL,
			asset_id TEXT NOT NULL,
			asset_value TEXT NOT NULL
		)
	`)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create inference_record_queue table: %v", err)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS assets (
			id TEXT PRIMARY KEY,
		)
	`)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create inference_record_queue table: %v", err)
	}

	storage := &InferenceStorage{
		db:        db,
		threshold: threshold,
	}

	return storage, nil
}
