package handlers

import (
	"encoding/json"
	"os"

	"github.com/gin-gonic/gin"
	"depin-server/utils"
)

type AssetEntry struct {
	Name    string `json:"name"`
	AssetID string `json:"assetId"`
}

type AssetMetadata struct {
	Models   []AssetEntry `json:"models"`
	Datasets []AssetEntry `json:"datasets"`
}

func HandleGetAssets(c *gin.Context) {
	const assetFile = "config/assets.json"

	data, err := os.ReadFile(assetFile)
	if err != nil {
		if os.IsNotExist(err) {
			utils.LogInfo("assets.json not found")
			utils.RespondError(c, 404, "assets.json not found", nil)
			return
		}
		utils.LogInfo("Error reading assets file: %v", err)
		utils.RespondError(c, 500, "Failed to read assets metadata", err)
		return
	}

	var metadata AssetMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		utils.LogInfo("Invalid assets.json format: %v", err)
		utils.RespondError(c, 500, "assets.json is corrupted", err)
		return
	}

	utils.RespondSuccess(c, "Assets fetched successfully", metadata)
}
