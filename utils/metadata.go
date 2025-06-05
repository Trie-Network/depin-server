package utils

import (
	"depin-server/constants"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type AssetEntry struct {
	Name    string `json:"name"`
	AssetID string `json:"assetId"`
}

type AssetMetadata struct {
	Models   []AssetEntry `json:"models"`
	Datasets []AssetEntry `json:"datasets"`
}

func AppendAssetMetadata(assetType, name, assetId string) error {
	metaPath := filepath.Join("config", "assets.json")

	var metadata AssetMetadata
	if _, err := os.Stat(metaPath); os.IsNotExist(err) {
		metadata = AssetMetadata{}
	} else {
		f, err := os.Open(metaPath)
		if err != nil {
			return err
		}
		defer f.Close()
		if err := json.NewDecoder(f).Decode(&metadata); err != nil {
			return err
		}
	}

	newEntry := AssetEntry{Name: name, AssetID: assetId}

	switch assetType {
	case constants.ASSET_TYPE_MODEL:
		metadata.Models = append(metadata.Models, newEntry)
	case constants.ASSET_TYPE_DATASET:
		metadata.Datasets = append(metadata.Datasets, newEntry)
	default:
		return fmt.Errorf("invalid assetType: %s", assetType)
	}

	f, err := os.Create(metaPath)
	if err != nil {
		return err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "    ")
	return enc.Encode(metadata)
}
