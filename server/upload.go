package server

import (
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

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

	metaPath := os.Getenv("MODEL_METADATA_PATH")
	if metaPath == "" {
		metaPath = "metadata" // default fallback, if needed
	}

    // Extract params
    assetName := c.PostForm("assetName")
    assetType := c.PostForm("assetType")
    url := c.PostForm("url")

    if assetName == "" || assetType == "" {
        utils.LogInfo("Missing assetName or assetType in request")
        utils.RespondError(c, http.StatusBadRequest, "Both assetName and assetType fields are required", nil)
        return
    }

    // Attempt to fetch files
    var assetFile multipart.File
    var assetHeader *multipart.FileHeader
    var metaFile multipart.File
    var metaHeader *multipart.FileHeader
    var errAsset, errMeta error

    assetFile, assetHeader, errAsset = c.Request.FormFile("assetFile")
    metaFile, metaHeader, errMeta = c.Request.FormFile("modelMetadata") // optional

    // Exclusivity check
    filesProvided := (errAsset == nil)
    if url != "" && filesProvided {
        utils.RespondError(c, http.StatusBadRequest, "Provide either files OR url, not both", nil)
        return
    }
    if url == "" && !filesProvided {
        utils.RespondError(c, http.StatusBadRequest, "assetFile is required if url is not provided", nil)
        return
    }
    if url != "" && (errMeta == nil) {
        utils.RespondError(c, http.StatusBadRequest, "Provide either files OR url, not both", nil)
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

    var filenames []string

    if filesProvided {
        // Save asset file
        defer assetFile.Close()
        filename := filepath.Base(assetHeader.Filename)
        dstPath := filepath.Join(uploadDir, filename)
        

        assetFileObj, err := os.Create(dstPath)
        if err != nil {
            utils.LogInfo("Error creating file: %v", err)
            utils.RespondError(c, http.StatusInternalServerError, "File creation error", err)
            return
        }

        defer func() {
            assetFileObj.Close()
            deleteTempAssetDir(uploadDir) // Clean up file after processing
        }()
        
        if _, err := io.Copy(assetFileObj, assetFile); err != nil {
            utils.LogInfo("Error saving file: %v", err)
            utils.RespondError(c, http.StatusInternalServerError, "File write error", err)
            return
        }

        filenames = append(filenames, filename)

    } else {
        if !strings.Contains(url, "huggingface.co") {
            utils.RespondError(c, http.StatusBadRequest, "URL must be from huggingface.co", nil)
            return
        }
        downloadURL := normalizeHuggingFaceURL(url)
        parts := strings.Split(downloadURL, "/")
        filename := strings.Split(parts[len(parts)-1], "?")[0]
        fullPath := filepath.Join(uploadDir, filename)

        stdout, stderr, err := runCommand("wget", "-O", fullPath, downloadURL)
        if err != nil {
            utils.LogInfo("wget failed: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
            utils.RespondError(c, http.StatusInternalServerError, "Failed to download asset", err)
            return
        }
        filenames = append(filenames, filename)
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

    if errMeta == nil {
        defer metaFile.Close()
        
        metaDir := filepath.Join(metaPath, assetID)
        if err := os.MkdirAll(metaDir, os.ModePerm); err != nil {
            utils.LogInfo("Failed to create model metadata directory: %v", err)
            utils.RespondError(c, http.StatusInternalServerError, "Upload directory error", err)
            return
        }

        metaName := filepath.Base(metaHeader.Filename)
        metaDst := filepath.Join(metaDir, metaName)

        metaOut, err := os.Create(metaDst)
        if err != nil {
            utils.LogInfo("Error creating metadata file: %v", err)
            utils.RespondError(c, http.StatusInternalServerError, "Metadata file creation error", err)
            return
        }
        defer metaOut.Close()
        if _, err := io.Copy(metaOut, metaFile); err != nil {
            utils.LogInfo("Error saving metadata file: %v", err)
            utils.RespondError(c, http.StatusInternalServerError, "Metadata write error", err)
            return
        }
        filenames = append(filenames, metaName)
    }

    if assetType == constants.ASSET_TYPE_MODEL && len(filenames) > 0 {
        modelInfo := &ModelInfo{
            AssetID:       assetID,
            AssetName:     assetName,
            AssetFileName: filenames[0],
        }
        if err := runModel(modelInfo); err != nil {
            utils.LogInfo("Failed to start Ollama model: %v", err)
            utils.RespondError(c, http.StatusInternalServerError, "Failed to launch model with Ollama", err)
            return
        }
    }

    utils.LogInfo("Assets uploaded: %v (Asset: %s, Type: %s)", filenames, assetName, assetType)
    utils.RespondSuccess(c, "Asset uploaded/imported successfully", gin.H{
        "fileNames": filenames,
        "assetName": assetName,
        "assetType": assetType,
        "assetId":   assetID,
    })
}


func deleteTempAssetDir(dirPath string) error {
	err := os.RemoveAll(dirPath)
	if err != nil {
		err := fmt.Errorf("failed to delete directory %s: %w", dirPath, err)
        utils.LogInfo("%v", err)
        return err
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
	rubixNFTPath := os.Getenv("RUBIX_NFT_PATH") // Path til NFT directory of rubix config dir

	// TODO: handle build dir for other OS
	return filepath.Join(rubixNFTPath, assetID, filename)
}

func normalizeHuggingFaceURL(original string) string {
	original = strings.Replace(original, "/blob/", "/resolve/", 1)
	if !strings.Contains(original, "?download=true") {
		original += "?download=true"
	}
	return original
}