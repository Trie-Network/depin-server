package handlers

import (
	"bytes"
	"io"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"depin-server/utils"
)

func HandleInference(c *gin.Context) {
	ollamaAPI := os.Getenv("OLLAMA_API")
	if ollamaAPI == "" {
		utils.LogInfo("OLLAMA_API environment variable is not set")
		utils.RespondError(c, http.StatusInternalServerError, "OLLAMA_API is not configured", nil)
		return
	}

	targetURL := ollamaAPI + "/api/chat"

	// Read the incoming request body
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		utils.LogInfo("Error reading request body: %v", err)
		utils.RespondError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// Forward the request to the target API
	resp, err := http.Post(targetURL, "application/json", bytes.NewBuffer(bodyBytes))
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

	// Set the same content-type as received
	c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), respBody)
}
