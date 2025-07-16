package main

import (
	"fmt"
	"log"
	"os"

	_ "github.com/joho/godotenv/autoload"

	"depin-server/db"
	"depin-server/rubix"
	"depin-server/server"
)

func main() {
	if _, err := os.Stat(".env"); os.IsNotExist(err) {
		fmt.Fprintln(os.Stderr, "⚠️  .env file not found. Please copy from .env.sample")
		os.Exit(1)
	}

	inferenceRecordDBPath := os.Getenv("INFERENCE_RECORD_DB_PATH")
	if inferenceRecordDBPath == "" {
		inferenceRecordDBPath = "inference_record.db"
	}

	inferenceStorageContractAddress := os.Getenv("INFERENCE_STORAGE_CONTRACT_ADDRESS")
	if inferenceStorageContractAddress == "" {
		log.Fatalf("INFERENCE_STORAGE_CONTRACT_ADDRESS is not set in .env")
	}

	rubixNodeAddress := os.Getenv("RUBIX_NODE_ADDRESS")
	if rubixNodeAddress == "" {
		log.Fatalf("RUBIX_NODE_ADDRESS is not set in .env")
	}

	storage, err := db.NewStorage(inferenceRecordDBPath, 10)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
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

	go resubscribeAssets(storage, rubixNodeAddress)

	depinServer := server.NewDepinServer(depinServerPort, storage, rubixNodeAddress)
	if err := depinServer.Start(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}


// resubscribeAssets is meant for subscribing back the Assets in case of
// DePIN server or Rubix Node restart
func resubscribeAssets(s *db.InferenceStorage, nodeAddress string) {
	assetList, err := db.GetExistingAssets(s)
	if err != nil {
		log.Printf("failed to get asset list, err: %v\n", err)
		return
	}

	for _, assetID := range assetList {
		err := rubix.SubscribeNFT(nodeAddress, assetID)
		if err != nil {
			log.Printf("failed to subscribe to Asset: %v, err: %v\n", assetID, err)
		}
	}
}