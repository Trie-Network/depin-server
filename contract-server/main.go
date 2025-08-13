package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path"

	"github.com/gin-gonic/gin"
	_ "github.com/joho/godotenv/autoload"
	"contract-server/host"

	wasmbridge "github.com/rubixchain/rubix-wasm/go-wasm-bridge"
)

func main() {
	logFilePath := os.Getenv("CONTRACT_SERVER_LOG") // TODO: will be integrated into with log file shared by DePIN Server
	if logFilePath == "" {
		logFilePath = "contract_server.log"
	}

	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		panic(err)
	}
	log.SetOutput(logFile)
	os.Stdout = logFile
	os.Stderr = logFile

	// Check if .env file exists
	if _, err := os.Stat(".env"); os.IsNotExist(err) {
		fmt.Fprintln(os.Stderr, "⚠️  .env file not found. Please copy from .env.sample")
		os.Exit(1)
	}

	serverPort := os.Getenv("CONTRACT_SERVER_PORT")
	if serverPort == "" {
		serverPort = "8082"
	}

	rubixNodeAddress := os.Getenv("RUBIX_NODE_ADDRESS")
	if rubixNodeAddress == "" {
		log.Fatalf("RUBIX_NODE_ADDRESS is not set in .env")
		os.Exit(1)
	}

	router := gin.Default()

	router.POST("/store_inference", handleStoreInference)

	router.Run(":" + serverPort)
}


func  handleStoreInference(c *gin.Context) {
	nodeAddress := os.Getenv("RUBIX_NODE_ADDRESS")
	quorumType := 2

	selfContractHashPath := path.Join("./wasm_artifacts/inference_execution_contract.wasm")

	var contractInputRequest ContractInputRequest

	err := json.NewDecoder(c.Request.Body).Decode(&contractInputRequest)
	if err != nil {
		wrapError(c.JSON, "err: Invalid request body")
		return
	}

	// Create Import function registry
	hostFnRegistry := wasmbridge.NewHostFunctionRegistry()
	hostFnRegistry.Register(host.NewDoExecuteNFT())

	// Initialize the WASM module
	wasmModule, err := wasmbridge.NewWasmModule(
		selfContractHashPath,
		hostFnRegistry,
		wasmbridge.WithRubixNodeAddress(nodeAddress),
		wasmbridge.WithQuorumType(quorumType),
	)
	if err != nil {
		wrapError(c.JSON, fmt.Sprintf("unable to initialize wasmModule: %v", err))
		return
	}

	if contractInputRequest.SmartContractData == "" {
		wrapError(c.JSON, fmt.Sprintf("unable to fetch Smart Contract from callback"))
		return
	}

	_, err = wasmModule.CallFunction(contractInputRequest.SmartContractData)
	if err != nil {
		wrapError(c.JSON, fmt.Sprintf("unable to execute function, err: %v", err))
		return
	}
}

