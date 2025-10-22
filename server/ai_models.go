package server

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/gin-gonic/gin"
)

type MLFlowMetrics struct {
	Metrics map[string]interface{} `json:"metrics"`
	Params  map[string]interface{} `json:"params"`
}

func getMLFlowMetadataDBPath(assetId string) (string, error) {
	metaPath := os.Getenv("MODEL_METADATA_PATH")
	if metaPath == "" {
		return "", fmt.Errorf("MODEL_METADATA_PATH environment variable is not set")
	}

	dbDir := filepath.Join(metaPath, assetId)
	if _, err := os.Stat(dbDir); os.IsNotExist(err) {
		return "", fmt.Errorf("metadata directory does not exist for asset ID: %s", assetId)
	}

	files, err := os.ReadDir(dbDir)
	if err != nil {
		return "", fmt.Errorf("failed to read metadata directory: %v", err)
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}
		if filepath.Ext(file.Name()) == ".db" {
			return filepath.Join(dbDir, file.Name()), nil
		}
	}

	return "", fmt.Errorf("no MLFlow Metadata DB file found for asset ID: %s", assetId)
}

func (s *DepinServer) HandleGetMLFlowMetadata(c *gin.Context) {
	assetId := c.Param("assetId")
	if assetId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Asset ID is required"})
		return
	}

	dbPath, err := getMLFlowMetadataDBPath(assetId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to get metadata DB path: %v", err)})
		return
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to open DB: %v", err)})
		return
	}
	defer db.Close()

	var metrics *MLFlowMetrics = &MLFlowMetrics{
		Metrics: make(map[string]interface{}),
		Params:  make(map[string]interface{}),
	}

	// metrics
	metricsRows, err := db.Query("SELECT key, value FROM metrics")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to query metrics: %v", err)})
		return
	}
	defer metricsRows.Close()

	for metricsRows.Next() {
		var metricName string
		var metricValue interface{}
		if err := metricsRows.Scan(&metricName, &metricValue); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to scan metrics row: %v", err)})
			return
		}
		metrics.Metrics[metricName] = metricValue
	}

	// params
	paramsRows, err := db.Query("SELECT key, value FROM params")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to query metrics: %v", err)})
		return
	}
	defer paramsRows.Close()

	for paramsRows.Next() {
		var paramName string
		var paramValue interface{}
		if err := paramsRows.Scan(&paramName, &paramValue); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to scan params row: %v", err)})
			return
		}
		metrics.Params[paramName] = paramValue
	}

	c.JSON(http.StatusOK, metrics)
}

func (s *DepinServer) HandleDownloadMLFlowMetadata(c *gin.Context) {
	assetId := c.Param("assetId")
	if assetId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Asset ID is required"})
		return
	}

	mlflowDBFilePath, err := getMLFlowMetadataDBPath(assetId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to get metadata DB path: %v", err)})
		return
	}

	mlflowDBFileInfo, err := os.Stat(mlflowDBFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Cannot access file"})
		}
		return
	}

	if mlflowDBFileInfo.Size() == 0 {
		c.JSON(http.StatusNoContent, gin.H{"error": "File is empty"})
		return
	}

	mlflowDBFileName := filepath.Base(mlflowDBFilePath)

	c.Header("Content-Disposition", "attachment; filename="+strconv.Quote(mlflowDBFileName))
	c.File(mlflowDBFilePath)
}
