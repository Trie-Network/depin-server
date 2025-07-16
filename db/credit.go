package db

import (
	"fmt"
	"log"
	"strings"
)

func AddInferenceRecord(s *InferenceStorage, r *InferenceRecord, rubixNodeAddress string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %v", err)
	}

	_, err = tx.Exec(
		"INSERT INTO inference_record_queue (id, did, timestamp, signature, asset_id, asset_value) VALUES (?, ?, ?, ?, ?)",
		r.ID, r.Did, r.Timestamp, r.Signature, r.AssetID, r.AssetValue,
	)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to insert record: %v", err)
	}

	// Check record count
	var count int
	err = tx.QueryRow("SELECT COUNT(*) FROM inference_record_queue WHERE asset_id = ?", r.AssetID).Scan(&count)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to count records: %v", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %v", err)
	}

	if count >= s.threshold {
		go ProcessBatchInferenceRecords(s, r.AssetID, rubixNodeAddress)
	}
	return nil
}

func ProcessBatchInferenceRecords(s *InferenceStorage, assetID string, rubixNodeAddress string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Fetch records ordered by timestamp (oldest first)
	rows, err := s.db.Query("SELECT id, did, timestamp, signature, asset_id FROM inference_record_queue WHERE asset_id = ? ORDER BY timestamp ASC LIMIT ?", assetID, s.threshold)
	if err != nil {
		log.Printf("Error querying records: %v", err)
		return
	}
	defer rows.Close()

	var records []InferenceRecord
	var ids []string
	for rows.Next() {
		var r InferenceRecord
		if err := rows.Scan(&r.ID, &r.Did, &r.Timestamp, &r.Signature, &r.AssetID, &r.AssetValue); err != nil {
			log.Printf("Error scanning record: %v", err)
			return
		}
		records = append(records, r)
		ids = append(ids, r.ID)
	}

	if len(records) == 0 || len(ids) == 0 {
		log.Printf("No records to process")
		return
	}

	tx, err := s.db.Begin()
	if err != nil {
		log.Printf("Error starting delete transaction: %v", err)
		return
	}

	if err := ExecuteSmartContract(records, rubixNodeAddress); err != nil {
		log.Printf("Error executing smart contract: %v", err)
		return
	}

	// Delete processed records
	recordsDeleteQuery := fmt.Sprintf("DELETE FROM inference_record_queue WHERE id IN (%s)", strings.Repeat("?,", len(ids)-1)+"?")
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		args[i] = id
	}

	_, err = tx.Exec(recordsDeleteQuery, args...)
	if err != nil {
		tx.Rollback()
		log.Printf("Error deleting records: %v", err)
		return
	}

	if err := tx.Commit(); err != nil {
		log.Printf("Error committing delete transaction: %v", err)
	}
}
