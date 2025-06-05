package handlers

import (
	"io"
	"net/http"
	"os"
	"path/filepath"

	"depin-server/constants"
	"depin-server/rubix"
	"depin-server/utils"

	"github.com/gin-gonic/gin"
)

func HandleFileUpload(c *gin.Context) {
	uploadRoot := os.Getenv("UPLOAD_DIR")
	if uploadRoot == "" {
		uploadRoot = "uploads"
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		utils.LogInfo("Error reading file: %v", err)
		utils.RespondError(c, http.StatusBadRequest, "File read error", err)
		return
	}
	defer file.Close()

	assetName := c.PostForm("assetName")
	assetType := c.PostForm("assetType")

	if assetName == "" || assetType == "" {
		utils.LogInfo("Missing assetName or assetType in request")
		utils.RespondError(c, http.StatusBadRequest, "Both assetName and assetType fields are required", nil)
		return
	}

	switch assetType {
	case constants.ASSET_TYPE_DATASET, constants.ASSET_TYPE_MODEL:
	default:
		utils.LogInfo("Invalid assetType: %s", assetType)
		utils.RespondError(c, http.StatusBadRequest, "Invalid assetType. Must be 'model' or 'dataset'", nil)
		return
	}

	uploadDir := filepath.Join(uploadRoot, assetType+"s", assetName)
	if err := os.MkdirAll(uploadDir, os.ModePerm); err != nil {
		utils.LogInfo("Failed to create directory: %v", err)
		utils.RespondError(c, http.StatusInternalServerError, "Upload directory error", err)
		return
	}

	dstPath := filepath.Join(uploadDir, filepath.Base(header.Filename))
	outFile, err := os.Create(dstPath)
	if err != nil {
		utils.LogInfo("Error creating destination file: %v", err)
		utils.RespondError(c, http.StatusInternalServerError, "File creation error", err)
		return
	}
	defer outFile.Close()

	if _, err := io.Copy(outFile, file); err != nil {
		utils.LogInfo("Error saving file: %v", err)
		utils.RespondError(c, http.StatusInternalServerError, "File write error", err)
		return
	}

	assetID, err := rubix.GenerateAssetHash(assetName, assetType)
	if err != nil {
		utils.LogInfo("Error generating asset hash: %v", err)
		utils.RespondError(c, http.StatusInternalServerError, "Asset ID generation failed", err)
		return
	}

	if err := utils.AppendAssetMetadata(assetType, assetName, assetID); err != nil {
		utils.LogInfo("Error updating metadata: %v", err)
		utils.RespondError(c, http.StatusInternalServerError, "Metadata write error", err)
		return
	}

	utils.LogInfo("File uploaded: %s (Asset: %s, Type: %s)", header.Filename, assetName, assetType)
	utils.RespondSuccess(c, "File uploaded successfully", gin.H{
		"fileName": header.Filename,
		"assetName": assetName,
		"assetType": assetType,
		"assetId": assetID,
	})
}
