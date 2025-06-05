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
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		panic(err)
	}
	log.SetOutput(logFile)
	os.Stdout = logFile
	os.Stderr = logFile

	r := gin.Default()

	api := r.Group("/depin-server/v1")
	{
		api.GET("/healthz", handlers.HandleHealthCheck)

		if os.Getenv("ENABLE_ASSET_UPLOAD") == "true" {
			api.POST("/upload", handlers.HandleFileUpload)
			utils.LogInfo("Upload endpoint enabled")
		} else {
			utils.LogInfo("Upload endpoint disabled (ENABLE_ASSET_UPLOAD is not true)")
		}
	}

	utils.LogInfo("Depin server started on port %s", port)
	r.Run(":" + port)
}
