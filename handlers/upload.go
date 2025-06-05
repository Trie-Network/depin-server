package handlers

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"depin-server/utils"
)

func HandleFileUpload(c *gin.Context) {
	uploadDir := os.Getenv("UPLOAD_DIR")
	if uploadDir == "" {
		uploadDir = "uploads"
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		utils.LogInfo("Error reading file: %v", err)
		utils.RespondError(c, http.StatusBadRequest, "File read error", err)
		return
	}
	defer file.Close()

	assetName := c.PostForm("assetName")
	if assetName == "" {
		utils.LogInfo("Missing assetName in upload request")
		utils.RespondError(c, http.StatusBadRequest, "Missing assetName field", fmt.Errorf("assetName is required"))
		return
	}

	// Use assetName directly (no sanitization)
	uploadDir = filepath.Join(uploadDir, assetName)
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

	utils.LogInfo("File uploaded: %s (Asset: %s)", header.Filename, assetName)
	utils.AppendModelMetadata(assetName)

	utils.RespondSuccess(c, "File uploaded successfully", gin.H{
		"fileName":  header.Filename,
		"assetName": assetName,
	})
}
