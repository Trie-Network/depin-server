package db

import (
	"fmt"
)

func GetExistingAssets(s *InferenceStorage) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var assets []string = make([]string, 0)

	rows, err := s.db.Query("SELECT id FROM assets")
	if err != nil {
		return nil, fmt.Errorf("unable to execute query to fetch asset list from 'assets' table, err: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var assetID string = ""
		if err := rows.Scan(&assetID); err != nil {
			return nil, fmt.Errorf("failed to extract AssetID from row object, err: %v", err)
		}

		assets = append(assets, assetID)
	}

	return assets, nil
}
