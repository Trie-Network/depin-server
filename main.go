package main

import (
	"fmt"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	_ "github.com/joho/godotenv/autoload"

	"depin-server/handlers"
	"depin-server/utils"
)

func main() {
	if _, err := os.Stat(".env"); os.IsNotExist(err) {
		fmt.Fprintln(os.Stderr, "⚠️  .env file not found. Please copy from .env.sample")
		os.Exit(1)
	}

	logFilePath := os.Getenv("LOG_FILE")
	depinServerPort := os.Getenv("SERVER_PORT")
	if depinServerPort == "" {
		depinServerPort = "8080"
	}

	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		panic(err)
	}
	log.SetOutput(logFile)
	os.Stdout = logFile
	os.Stderr = logFile

	r := gin.Default()

	apiV1 := r.Group("/depin-server/v1")
	{
		apiV1.GET("/healthz", handlers.HandleHealthCheck)

		if os.Getenv("ENABLE_ASSET_UPLOAD") == "true" {
			apiV1.POST("/upload", handlers.HandleFileUpload)
			apiV1.POST("/inference", handlers.HandleInference)
			apiV1.GET("/assets", handlers.HandleGetAssets)
			apiV1.GET("/assets/download/:assetId", handlers.HandleDownloadAsset)
			apiV1.POST("/import", handlers.HandleImportModel)
		} else {
			utils.LogInfo("Depin Server is not accepting new assets, set ENABLE_ASSET_UPLOAD to true to allow uploads")
		}
	}

	utils.LogInfo("Depin server started on port %s", depinServerPort)
	r.Run(":" + depinServerPort)
}
