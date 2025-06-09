package handlers

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"depin-server/utils"
)

type ModelInfo struct {
	AssetID   string `json:"assetID"`
	AssetName string `json:"assetName"`
	AssetFileName string `json:"assetFilename"`
}

// runModel checks file type and launches appropriate runtime if supported.
// Currently only .gguf files are handled via runModelWithOllama.
func runModel(modelInfo *ModelInfo) error {
	ext := strings.ToLower(filepath.Ext(modelInfo.AssetFileName))

	if ext == ".gguf" {
		utils.LogInfo("Launching Ollama runtime for .gguf model: %s", modelInfo.AssetName)
		return runModelWithOllama(
			modelInfo.AssetID, 
			modelInfo.AssetName, 
			modelInfo.AssetFileName)
	}

	utils.LogInfo("No runtime associated with file type: %s (skipping execution)", ext)
	return nil
}

func runModelWithOllama(assetID, assetName, filename string) error {
	createScriptPath := os.Getenv("CREATE_OLLAMA_MODEL")
	if createScriptPath == "" {
		return fmt.Errorf("CREATE_OLLAMA_MODEL is not set")
	}

	ggufPath := getAssetLocation(assetID, filename)

	// Step 1: Run create.sh
	stdout, stderr, err := runCommand(createScriptPath, ggufPath, assetID)
	if err != nil {
		utils.LogInfo("create.sh failed: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
		return fmt.Errorf("create.sh failed: %w", err)
	}

	// Step 2: Start tmux session to run Ollama
	session := "ollama-" + assetID
	stdout, stderr, err = runCommand("tmux", "new", "-s", session, "-d", "ollama", "run", assetID+":latest")
	if err != nil {
		utils.LogInfo("tmux run failed: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
		return fmt.Errorf("tmux run failed: %w", err)
	}

	utils.LogInfo("Successfully launched Ollama session for model: %v (%v)", assetName, assetID)
	return nil
}

// RunCommand runs a shell command with arguments and returns stdout, stderr, and error.
func runCommand(name string, args ...string) (string, string, error) {
	cmd := exec.Command(name, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	return stdout.String(), stderr.String(), err
}
