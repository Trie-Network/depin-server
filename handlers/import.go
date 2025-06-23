package handlers

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"depin-server/rubix"
	"depin-server/utils"

	"github.com/gin-gonic/gin"
)

type ImportRequest struct {
	URL       string `json:"url" binding:"required"`
	AssetName string `json:"assetName" binding:"required"`
	AssetType string `json:"assetType" binding:"required"` // "model" or "dataset"
}

func HandleImportModel(c *gin.Context) {
	var req ImportRequest

	// Bind input
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "Missing or invalid fields", err)
		return
	}

	// Validate domain
	if !strings.Contains(req.URL, "huggingface.co") {
		utils.RespondError(c, http.StatusBadRequest, "URL must be from huggingface.co", nil)
		return
	}

	// Validate assetType
	if req.AssetType != "model" && req.AssetType != "dataset" {
		utils.RespondError(c, http.StatusBadRequest, "Invalid assetType. Must be 'model' or 'dataset'", nil)
		return
	}

	// Normalize URL
	downloadURL := normalizeHuggingFaceURL(req.URL)

	// Extract filename
	parts := strings.Split(downloadURL, "/")
	filename := parts[len(parts)-1]
	filename = strings.Split(filename, "?")[0] // clean query param

	// Destination path
	uploadRoot := os.Getenv("UPLOAD_DIR")
	if uploadRoot == "" {
		uploadRoot = "uploads"
	}
	saveDir := filepath.Join(uploadRoot, req.AssetType+"s", req.AssetName)
	if err := os.MkdirAll(saveDir, os.ModePerm); err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "Failed to create asset folder", err)
		return
	}
	fullPath := filepath.Join(saveDir, filename)

	// Download via wget
	utils.LogInfo("Downloading asset from: %s", downloadURL)
	stdout, stderr, err := runCommand("wget", "-O", fullPath, downloadURL)
	if err != nil {
		utils.LogInfo("wget failed: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
		utils.RespondError(c, http.StatusInternalServerError, "Failed to download asset", err)
		return
	}

	// Generate asset ID
	assetID, err := rubix.GenerateAssetHash(req.AssetName, req.AssetType)
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "Asset ID generation failed", err)
		return
	}

	// Append metadata
	if err := utils.AppendAssetMetadata(req.AssetType, req.AssetName, assetID); err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "Metadata write failed", err)
		return
	}

	// Attempt to run model if supported
	model := &ModelInfo{
		AssetID:       assetID,
		AssetName:     req.AssetName,
		AssetFileName: filename,
	}

	if err := runModel(model); err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "Runtime launch failed", err)
		return
	}

	utils.RespondSuccess(c, "Asset imported and launched successfully", gin.H{
		"assetName": req.AssetName,
		"assetType": req.AssetType,
		"assetId":   assetID,
		"fileName":  filename,
	})
}

func normalizeHuggingFaceURL(original string) string {
	original = strings.Replace(original, "/blob/", "/resolve/", 1)

	if !strings.Contains(original, "?download=true") {
		original += "?download=true"
	}
	return original
}
