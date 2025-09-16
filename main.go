package main

import (
	"fmt"
	"log"
	"os"
	"strconv"

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

	// Initialize DB
	dbConn := server.InitDB("assets.db")

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

	assetStoreInfoThreshold := os.Getenv("ASSET_STORE_INFO_THRESHOLD")
	if assetStoreInfoThreshold == "" {
		assetStoreInfoThreshold = "10"
	}

	threshold, err := strconv.Atoi(assetStoreInfoThreshold)
	if err != nil || threshold <= 0 {
		log.Fatalf("Invalid ASSET_STORE_INFO_THRESHOLD: %v", err)
	}

	storage, err := db.NewStorage(inferenceRecordDBPath, threshold)
	if err != nil {
		log.Fatalf("Failed to initialize inference storage: %v", err)
	}

	depinServerPort := os.Getenv("SERVER_PORT")
	if depinServerPort == "" {
		depinServerPort = "8080"
	}

	// Create DepinServer instance
	depinServer := server.NewDepinServer(depinServerPort, storage, rubixNodeAddress, dbConn)

	// Resubscribe assets in background
	go resubscribeAssets(storage, rubixNodeAddress)

	log.Println("Server started at :" + depinServerPort)
	if err := depinServer.Start(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func resubscribeAssets(s *db.InferenceStorage, nodeAddress string) {
	assetList, err := db.GetExistingAssets(s)
	if err != nil {
		log.Printf("failed to get asset list, err: %v\n", err)
		return
	}

	for _, assetID := range assetList {
		if err := rubix.SubscribeNFT(nodeAddress, assetID); err != nil {
			log.Printf("failed to subscribe to Asset: %v, err: %v\n", assetID, err)
		}
	}
}
