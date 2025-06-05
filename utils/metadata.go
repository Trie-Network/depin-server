package utils

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type ModelEntry struct {
	Name    string `json:"name"`
	AssetID string `json:"assetId"`
}

type ModelsFile struct {
	Models []ModelEntry `json:"models"`
}

const modelsPath = "config/models.json"

func AppendModelMetadata(assetName string) {
	if err := os.MkdirAll(filepath.Dir(modelsPath), os.ModePerm); err != nil {
		LogInfo("Failed to create config directory: %v", err)
		return
	}

	var models ModelsFile

	// Read existing models.json
	if data, err := os.ReadFile(modelsPath); err == nil {
		json.Unmarshal(data, &models)
	}

	// Add new entry
	models.Models = append(models.Models, ModelEntry{
		Name:    assetName,
		AssetID: "",
	})

	// Write back
	file, err := os.Create(modelsPath)
	if err != nil {
		LogInfo("Failed to write models.json: %v", err)
		return
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "    ")
	if err := encoder.Encode(models); err != nil {
		LogInfo("Failed to encode models.json: %v", err)
	}
	LogInfo("Added assetName '%s' to models.json", assetName)
}
