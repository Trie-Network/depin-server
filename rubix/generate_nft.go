package rubix

import (
	"bytes"
	"depin-server/constants"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
)

type BasicResponse struct {
	Message string `json:"message"`
	Result  string `json:"result"`
	Status  bool   `json:"status"`
}

// GenerateAssetHash calls the /api/create-nft endpoint of 
// Rubix node to generate a hash for the given asset.
func GenerateAssetHash(assetName string, assetType string) (string, error) {
	var requestBody bytes.Buffer

	writer := multipart.NewWriter(&requestBody)

	depinDid := os.Getenv("DEPIN_DID")
	if depinDid == "" {
		return "", fmt.Errorf("DEPIN_DID environment variable is not set")
	}

	uploadRoot := os.Getenv("UPLOAD_DIR")
	if uploadRoot == "" {
		uploadRoot = "uploads"
	}

	// Add form fields (simple text fields)
	writer.WriteField("did", depinDid)

	// Add the NFTFile to the form
	var artifactPath string
	switch assetType {
		case constants.ASSET_TYPE_MODEL:
			artifactPath = filepath.Join(uploadRoot, constants.ASSET_TYPE_MODEL+"s", assetName)
		case constants.ASSET_TYPE_DATASET:
			artifactPath = filepath.Join(uploadRoot, constants.ASSET_TYPE_DATASET+"s", assetName)
		default:
			return "", fmt.Errorf("Invalid asset type: %s. Must be 'model' or 'dataset'", assetType)
	}

	entries, err := os.ReadDir(artifactPath)
	if err != nil {
		return "", fmt.Errorf("Error reading directory %s: %v", artifactPath, err)
	}

	if len(entries) == 0 {
		return "", fmt.Errorf("No files found in the directory %s", artifactPath)
	}

	var nftArtifact *os.File
	var nftArtifactFilepath string = ""
	for _, entry := range entries {
		if !entry.IsDir() {
			nftArtifactFilepath = filepath.Join(artifactPath, entry.Name())
			nftArtifact, err = os.Open(nftArtifactFilepath)
			if err != nil {
				return "", fmt.Errorf("Error opening file %s: %v", nftArtifactFilepath, err)
			}
			defer nftArtifact.Close()
		}
	}

	nftArtifactFile, err := writer.CreateFormFile("artifact", nftArtifactFilepath)
	if err != nil {
		return "", fmt.Errorf("Error creating NFT Artifact file: %v", err)
	}

	_, err = io.Copy(nftArtifactFile, nftArtifact)
	if err != nil {
		return "", fmt.Errorf("Error copying NFT Artifact content: %v", err)
	}

	identMetadataJSONPath := "./rubix/ident.json"
	metadataFileInfo, err := os.Open(identMetadataJSONPath)
	if err != nil {
		return "", fmt.Errorf("Error opening NFTFileInfo file: %v", err)
	}
	defer metadataFileInfo.Close()

	// Add the NFTFileInfo part to the form
	metadataFile, err := writer.CreateFormFile("metadata", identMetadataJSONPath)
	if err != nil {
		return "", fmt.Errorf("Error creating NFTFileInfo form file: %v", err)
	}

	_, err = io.Copy(metadataFile, metadataFileInfo)
	if err != nil {
		return "", fmt.Errorf("Error copying NFTFileInfo content: %v", err)
	}

	// Close the writer to finalize the form data
	err = writer.Close()
	if err != nil {
		return "", fmt.Errorf("Error closing multipart writer: %v", err)
	}

	// Create the request URL
	nodeAddress := os.Getenv("RUBIX_NODE_URL")
	if nodeAddress == "" {
		return "", fmt.Errorf("RUBIX_NODE_URL environment variable is not set")
	}
	url, err := url.JoinPath(nodeAddress, "/api/create-nft")
	if err != nil {
		return "", fmt.Errorf("Error forming url path for Create NFT API: %v", err)
	}

	// Create a new HTTP request
	req, err := http.NewRequest("POST", url, &requestBody)
	if err != nil {
		return "", fmt.Errorf("Error creating HTTP request: %v", err)
	}

	// Set the Content-Type header to multipart/form-data with the correct boundary
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("Error sending HTTP request: %v", err)
	}
	defer resp.Body.Close()

	// Read and print the response body
	createNFTAPIResponse, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("Failed to read response body: %v", err)
	}

	var basicResponse *BasicResponse
	if err := json.Unmarshal(createNFTAPIResponse, &basicResponse); err != nil {
		return "", fmt.Errorf("Failed to parse response JSON: %v", err)
	}

	if basicResponse.Result == "" {
		return "", fmt.Errorf("unable to fetch NFT ID after CreateNFT API call")
	}

	assetID := basicResponse.Result
	if err := SubscribeNFT(nodeAddress, assetID); err != nil {
		return "", fmt.Errorf("Failed to subscribe NFT: %v", err)
	}

	return basicResponse.Result, nil
}
