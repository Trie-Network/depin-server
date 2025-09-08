package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"

	"depin-server/db"
	"depin-server/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

/*
{
  "model": "QmPGmLyLxQmiqThYajLgtnLW8rvyRJ5ySCPf8fTDNeiz7C:latest",
  "messages": [
    {
      "role": "user",
      "content": "hey how are you doing"
    },
    {
      "role": "assistant",
      "content": "im just a large language model and how can I help you today"
    },
    {
      "role": "user",
      "content": "Hey"
    }
  ],
  "stream": false
}
*/

type InferenceInput struct {
	Model    string     `json:"model"`
	Messages []*Message `json:"messages"`
	Stream   bool       `json:"stream"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type HandleInferenceReq struct {
	OllamaInferenceInput *InferenceInput `json:"ollama_inference_input"`
	Did                  string          `json:"did"`
	Timestamp            string          `json:"timestamp"`
	Signature            string          `json:"signature"`
	AssetID              string          `json:"asset_id"`
	AssetValue           string          `json:"asset_value"`
}

func (s *DepinServer) HandleInference(c *gin.Context) {
	ollamaAPI := os.Getenv("OLLAMA_API")
	if ollamaAPI == "" {
		utils.LogInfo("OLLAMA_API environment variable is not set")
		utils.RespondError(c, http.StatusInternalServerError, "OLLAMA_API is not configured", nil)
		return
	}

	ollamaServer := ollamaAPI + "/api/chat"

	// Read the incoming request body
	inputReqBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		utils.LogInfo("Error reading request body: %v", err)
		utils.RespondError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// Unmarshal the request body into our custom struct
	var inferenceReq HandleInferenceReq
	err = json.Unmarshal(inputReqBytes, &inferenceReq)
	if err != nil {
		utils.LogInfo("Error unmarshalling request body: %v", err)
		utils.RespondError(c, http.StatusBadRequest, "Invalid request format", err)
		return
	}

	ollamaInferenceInputBytes, err := json.Marshal(inferenceReq.OllamaInferenceInput)
	if err != nil {
		utils.LogInfo("Error marshalling ollama_inference_input: %v", err)
		utils.RespondError(c, http.StatusBadRequest, "Invalid ollama_inference_input format", err)
		return
	}

	// Forward the request to the target API
	resp, err := http.Post(ollamaServer, "application/json", bytes.NewBuffer(ollamaInferenceInputBytes))
	if err != nil {
		utils.LogInfo("Error forwarding request to OLLAMA_API: %v", err)
		utils.RespondError(c, http.StatusBadGateway, "Error contacting inference backend", err)
		return
	}
	defer resp.Body.Close()

	// Read the response from the target API
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		utils.LogInfo("Error reading response from OLLAMA_API: %v", err)
		utils.RespondError(c, http.StatusBadGateway, "Error reading inference response", err)
		return
	}

	userInferenceInput, err := getUserInferenceInput(inferenceReq.OllamaInferenceInput)
	if err != nil {
		utils.LogInfo("Error getting user inference input: %v", err)
		utils.RespondError(c, http.StatusBadRequest, "Invalid inference input", err)
		return
	}

	// Add inference record to DB
	userInferenceRecord := &db.InferenceRecord{
		ID:        uuid.New().String(),
		Did:       inferenceReq.Did,
		Timestamp: inferenceReq.Timestamp,
		Signature: inferenceReq.Signature,
		AssetID:   inferenceReq.AssetID,
		AssetValue: inferenceReq.AssetValue,
		Query:     userInferenceInput,
	}

	if err := db.AddInferenceRecord(s.Storage, userInferenceRecord, s.RubixNodeAddress); err != nil {
		utils.LogInfo("Error adding inference record to DB: %v", err)
		utils.RespondError(c, http.StatusInternalServerError, "Failed to record inference", err)
		return
	}

	// Set the same content-type as received
	c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), respBody)
}

func getUserInferenceInput(inferenceInput *InferenceInput) (string, error) {
	if inferenceInput == nil {
		return "", errors.New("inferenceInput is required")
	}

	if len(inferenceInput.Messages) < 1 {
		return "", fmt.Errorf("atleast one message is expected, got %d", len(inferenceInput.Messages))
	}

	// User query will be the content of the last message always
	idx := len(inferenceInput.Messages) - 1

	return inferenceInput.Messages[idx].Content, nil
}
