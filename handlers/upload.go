package handlers

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

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

	assetName := c.PostForm("assetName")
	assetType := c.PostForm("assetType")
	url := strings.TrimSpace(c.PostForm("url"))

	file, header, errFile := c.Request.FormFile("file")
	if errFile != nil {
		utils.LogInfo("Error reading file: %v", errFile)
		utils.RespondError(c, http.StatusBadRequest, "File read error", errFile)
		return
	}
	defer file.Close()


	if assetName == "" || assetType == "" {
		utils.LogInfo("Missing assetName or assetType in request")
		utils.RespondError(c, http.StatusBadRequest, "Both assetName and assetType fields are required", nil)
		return
	}

	switch assetType {
	case constants.ASSET_TYPE_DATASET, constants.ASSET_TYPE_MODEL:
		// valid
	default:
		utils.LogInfo("Invalid assetType: %s", assetType)
		utils.RespondError(c, http.StatusBadRequest, "Invalid assetType. Must be 'model' or 'dataset'", nil)
		return
	}

	if file == nil && url == "" {
		utils.RespondError(c, http.StatusBadRequest, "Provide either a file or a URL", nil)
		return
	}
	if file != nil && url != "" {
		utils.RespondError(c, http.StatusBadRequest, "Provide only one of file or URL, not both", nil)
		return
	}

	uploadDir := filepath.Join(uploadRoot, assetType+"s", assetName)
	if err := os.MkdirAll(uploadDir, os.ModePerm); err != nil {
		utils.LogInfo("Failed to create directory: %v", err)
		utils.RespondError(c, http.StatusInternalServerError, "Upload directory error", err)
		return
	}

	var filename string
	if file != nil {
		defer file.Close()
		filename = filepath.Base(header.Filename)
		dstPath := filepath.Join(uploadDir, filename)

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
	} else {
		if !strings.Contains(url, "huggingface.co") {
			utils.RespondError(c, http.StatusBadRequest, "URL must be from huggingface.co", nil)
			return
		}
		downloadURL := normalizeHuggingFaceURL(url)
		parts := strings.Split(downloadURL, "/")
		filename = parts[len(parts)-1]

		filename = strings.Split(filename, "?")[0]
		fullPath := filepath.Join(uploadDir, filename)
		utils.LogInfo("Downloading asset from: %s", downloadURL)
		
		stdout, stderr, err := runCommand("wget", "-O", fullPath, downloadURL)
		if err != nil {
			utils.LogInfo("wget failed: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
			utils.RespondError(c, http.StatusInternalServerError, "Failed to download asset", err)
			return
		}
	}

	defer deleteFile(uploadDir)

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

	if assetType == constants.ASSET_TYPE_MODEL {
		modelInfo := &ModelInfo{
			AssetID:       assetID,
			AssetName:     assetName,
			AssetFileName: filename,
		}

		if err := runModel(modelInfo); err != nil {
			utils.LogInfo("Failed to start Ollama model: %v", err)
			utils.RespondError(c, http.StatusInternalServerError, "Failed to launch model with Ollama", err)
			return
		}
	}

	utils.LogInfo("File uploaded: %s (Asset: %s, Type: %s)", filename, assetName, assetType)
	utils.RespondSuccess(c, "Asset uploaded successfully", gin.H{
		"fileName":  filename,
		"assetName": assetName,
		"assetType": assetType,
		"assetId":   assetID,
	})
}

func deleteFile(filePath string) error {
	err := os.Remove(filePath)
	if err != nil {
		return fmt.Errorf("failed to delete file %s: %w", filePath, err)
	}
	return nil
}

// getAssetLocation returns the full path to the asset file based on the asset ID.
func getAssetLocation(assetID string) string {
	homeDir := os.Getenv("HOME")
	if homeDir == "" {
		return ""
	}

	assetDirPath := filepath.Join(homeDir, "depin", "rubixgoplatform", "linux", "node0", "NFT", assetID)

	entries, err := os.ReadDir(assetDirPath)
	if err != nil {
		return ""
	}

	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) != ".json" {
			return filepath.Join(assetDirPath, entry.Name())
		}
	}

	return ""
}

func getAssetLocationByFilename(assetID string, filename string) string {
	homeDir := os.Getenv("HOME")

	// TODO: handle build dir for other OS
	return filepath.Join(homeDir, "depin", "rubixgoplatform", "linux", "node0", "NFT", assetID, filename)
}

func normalizeHuggingFaceURL(original string) string {
	original = strings.Replace(original, "/blob/", "/resolve/", 1)
	if !strings.Contains(original, "?download=true") {
		original += "?download=true"
	}
	return original
}
