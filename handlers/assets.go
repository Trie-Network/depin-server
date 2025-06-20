package handlers

import (
	"encoding/json"
	"os"

	"depin-server/utils"

	"github.com/gin-gonic/gin"
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

func HandleDownloadAsset(c *gin.Context) {
	assetID := c.Param("assetId")
	if assetID == "" {
		utils.RespondError(c, 400, "Asset ID is required", nil)
		return
	}

	assetPath := getAssetLocation(assetID)
	if _, err := os.Stat(assetPath); os.IsNotExist(err) {
		utils.LogInfo("Asset not found: %s", assetPath)
		utils.RespondError(c, 404, "Asset not found", nil)
		return
	}

	c.File(assetPath)
	utils.LogInfo("Serving asset: %s", assetPath)
}
