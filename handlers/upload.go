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
		c.String(http.StatusBadRequest, "File read error")
		return
	}
	defer file.Close()

	assetName := c.PostForm("assetName")
	if assetName == "" {
		utils.LogInfo("Missing assetName in upload request")
		c.String(http.StatusBadRequest, "Missing assetName field")
		return
	}

	if err := os.MkdirAll(uploadDir, os.ModePerm); err != nil {
		utils.LogInfo("Failed to create directory: %v", err)
		c.String(http.StatusInternalServerError, "Upload directory error")
		return
	}

	dstPath := filepath.Join(uploadDir, filepath.Base(header.Filename))
	outFile, err := os.Create(dstPath)
	if err != nil {
		utils.LogInfo("Error creating destination file: %v", err)
		c.String(http.StatusInternalServerError, "File creation error")
		return
	}
	defer outFile.Close()

	if _, err := io.Copy(outFile, file); err != nil {
		utils.LogInfo("Error saving file: %v", err)
		c.String(http.StatusInternalServerError, "File write error")
		return
	}

	utils.LogInfo("File uploaded: %s (Asset: %s)", header.Filename, assetName)
	utils.AppendModelMetadata(assetName)

	c.String(http.StatusOK, fmt.Sprintf("Uploaded %s", header.Filename))
}
