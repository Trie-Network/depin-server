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

	inferenceRecordDBPath := os.Getenv("INFERENCE_RECORD_DB_PATH")
	if inferenceRecordDBPath == "" {
		inferenceRecordDBPath = "inference_record.db"
	}

	inferenceStorageContractAddress := os.Getenv("INFERENCE_STORAGE_CONTRACT_ADDRESS")
	if inferenceStorageContractAddress == "" {
		log.Fatalf("INFERENCE_STORAGE_CONTRACT_ADDRESS is not set in .env")
		return
	}

	rubixNodeAddress := os.Getenv("RUBIX_NODE_ADDRESS")
	if rubixNodeAddress == "" {
		log.Fatalf("RUBIX_NODE_ADDRESS is not set in .env")
		return
	}

	assetStoreInfoThreshold := os.Getenv("ASSET_STORE_INFO_THRESHOLD")
	if assetStoreInfoThreshold == "" {
		assetStoreInfoThreshold = "10"
	}

	threshold, err := strconv.Atoi(assetStoreInfoThreshold)
	if err != nil {
		log.Fatalf("Invalid ASSET_STORE_INFO_THRESHOLD: %v", err)
		return
	}
	if threshold <= 0 {
		log.Fatalf("ASSET_STORE_INFO_THRESHOLD must be a positive integer")
		return
	}

	if os.Getenv("MODEL_METADATA_PATH") == "" { // optional, only needed if metadata download API is used
		log.Println("MODEL_METADATA_PATH is not set, metadata download API will not work")
		return
	} else {
		// Check if the directory already exists. If not, create it.
		if _, err := os.Stat(os.Getenv("MODEL_METADATA_PATH")); os.IsNotExist(err) {
			err := os.MkdirAll(os.Getenv("MODEL_METADATA_PATH"), os.ModePerm)
			if err != nil {
				log.Fatalf("Failed to create MODEL_METADATA_PATH directory: %v", err)
				return
			}
		}
	}

	rubixNFTPath := os.Getenv("RUBIX_NFT_PATH")
	if rubixNFTPath == "" {
		log.Fatal("RUBIX_NFT_PATH is not set")
		return
	}

	storage, err := db.NewStorage(inferenceRecordDBPath, threshold)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
		return
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
		return
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
