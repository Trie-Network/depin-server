package server

import (
	"fmt"
	"io"
	//"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"database/sql"

	"depin-server/constants"
	"depin-server/rubix"
	"depin-server/utils"

	"github.com/gin-gonic/gin"
)

func (s *DepinServer) HandleFileUpload(c *gin.Context) {
	uploadRoot := os.Getenv("UPLOAD_DIR")
	if uploadRoot == "" {
		uploadRoot = "uploads"
	}

	assetName := c.PostForm("assetName")
	assetType := c.PostForm("assetType")
	url := c.PostForm("url")

	if assetName == "" || assetType == "" {
		utils.LogInfo("Missing assetName or assetType in request")
		utils.RespondError(c, http.StatusBadRequest, "Both assetName and assetType fields are required", nil)
		return
	}

	if url == "" && c.Request.MultipartForm == nil {
		if err := c.Request.ParseMultipartForm(64 << 20); err != nil { // 64MB buffer for 2 files
			utils.LogInfo("Error parsing multipart form: %v", err)
			utils.RespondError(c, http.StatusBadRequest, "Invalid form data", err)
			return
		}
	}

	uploadDir := filepath.Join(uploadRoot, assetType+"s", assetName)
	if err := os.MkdirAll(uploadDir, os.ModePerm); err != nil {
		utils.LogInfo("Failed to create directory: %v", err)
		utils.RespondError(c, http.StatusInternalServerError, "Upload directory error", err)
		return
	}

	var filenames []string
	filePresent := false

	if url == "" {
		// Handle multiple files
		form := c.Request.MultipartForm
		files := form.File["file"]

		if len(files) != 2 {
			utils.RespondError(c, http.StatusBadRequest, "Exactly 2 files must be uploaded", nil)
			return
		}

		filePresent = true
		for _, header := range files {
			file, err := header.Open()
			if err != nil {
				utils.LogInfo("Error opening file: %v", err)
				utils.RespondError(c, http.StatusInternalServerError, "Failed to open file", err)
				return
			}
			defer file.Close()

			filename := filepath.Base(header.Filename)
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
			filenames = append(filenames, filename)
		}
	}

	if filePresent && url != "" {
		utils.RespondError(c, http.StatusBadRequest, "Provide either 2 files or a URL, not both", nil)
		return
	}

	if !filePresent && url == "" {
		utils.RespondError(c, http.StatusBadRequest, "Either 2 files or a URL must be provided", nil)
		return
	}

	if url != "" {
		if !strings.Contains(url, "huggingface.co") {
			utils.RespondError(c, http.StatusBadRequest, "URL must be from huggingface.co", nil)
			return
		}

		downloadURL := normalizeHuggingFaceURL(url)
		parts := strings.Split(downloadURL, "/")
		filename := strings.Split(parts[len(parts)-1], "?")[0]
		fullPath := filepath.Join(uploadDir, filename)

		utils.LogInfo("Downloading asset from: %s", downloadURL)
		stdout, stderr, err := runCommand("wget", "-O", fullPath, downloadURL)
		if err != nil {
			utils.LogInfo("wget failed: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
			utils.RespondError(c, http.StatusInternalServerError, "Failed to download asset", err)
			return
		}
		filenames = append(filenames, filename)
	}

	// Validate assetType
	switch assetType {
	case constants.ASSET_TYPE_DATASET, constants.ASSET_TYPE_MODEL:
	default:
		utils.LogInfo("Invalid assetType: %s", assetType)
		utils.RespondError(c, http.StatusBadRequest, "Invalid assetType. Must be 'model' or 'dataset'", nil)
		return
	}

	// Generate asset ID
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

	// Run model logic for MODEL assets
	if assetType == constants.ASSET_TYPE_MODEL && len(filenames) > 0 {
		modelInfo := &ModelInfo{
			AssetID:       assetID,
			AssetName:     assetName,
			AssetFileName: filenames[0], // first file
		}

		if err := runModel(modelInfo); err != nil {
			utils.LogInfo("Failed to start Ollama model: %v", err)
			utils.RespondError(c, http.StatusInternalServerError, "Failed to launch model with Ollama", err)
			return
		}
	}

	utils.LogInfo("Assets uploaded: %v (Asset: %s, Type: %s)", filenames, assetName, assetType)
	utils.RespondSuccess(c, "Assets uploaded/imported successfully", gin.H{
		"fileNames": filenames,
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
	rubixNFTPath := os.Getenv("RUBIX_NFT_PATH")
	assetDirPath := filepath.Join(rubixNFTPath, assetID)

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
	rubixNFTPath := os.Getenv("RUBIX_NFT_PATH") 

	return filepath.Join(rubixNFTPath, assetID, filename)
}

func normalizeHuggingFaceURL(original string) string {
	original = strings.Replace(original, "/blob/", "/resolve/", 1)
	if !strings.Contains(original, "?download=true") {
		original += "?download=true"
	}
	return original
}

func (s *DepinServer) HandleGetMetadata(c *gin.Context) {
	var request struct {
		IPFSHash string `json:"ipfs_hash"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}

	metaPath := os.Getenv("M_META_PATH")
	if metaPath == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "M_META_PATH not set"})
		return
	}

	dbPath := fmt.Sprintf("%s/%s", metaPath, request.IPFSHash)

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to open DB: %v", err)})
		return
	}
	defer db.Close()

	type KeyValue struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}

	response := struct {
		Metrics []KeyValue `json:"metrics"`
		Params  []KeyValue `json:"params"`
	}{
		Metrics: []KeyValue{},
		Params:  []KeyValue{},
	}

	tables := []string{"metrics", "params"}
	for _, table := range tables {
		// Check if table exists
		var tableCount int
		err := db.QueryRow("SELECT count(name) FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&tableCount)
		if err != nil || tableCount == 0 {
			continue
		}

		// Query table rows
		rows, err := db.Query(fmt.Sprintf("SELECT key, value FROM %s", table))
		if err != nil {
			continue 
		}
		defer rows.Close()

		data := []KeyValue{}
		for rows.Next() {
			var kv KeyValue
			if err := rows.Scan(&kv.Key, &kv.Value); err != nil {
				continue
			}
			data = append(data, kv)
		}

		if table == "metrics" {
			response.Metrics = data
		} else {
			response.Params = data
		}
	}
	c.JSON(http.StatusOK, response)
}

func (s *DepinServer) HandleDownloadMetadata(c *gin.Context) {
    ipfsHash := c.Param("ipfsHash")
    if ipfsHash == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "IPFS hash is required"})
        return
    }

    metaPath := os.Getenv("M_META_PATH")
    if metaPath == "" {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "M_META_PATH not set"})
        return
    }

    filePath := fmt.Sprintf("%s/%s", metaPath, ipfsHash)

    fileInfo, err := os.Stat(filePath)
    if err != nil {
        if os.IsNotExist(err) {
            c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
        } else {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Cannot access file"})
        }
        return
    }
    if fileInfo.Size() == 0 {
        c.JSON(http.StatusNoContent, gin.H{"error": "File is empty"})
        return
    }
    c.File(filePath)
}