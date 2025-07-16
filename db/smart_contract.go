package db

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"
)

type BasicResponse struct {
	Status  bool        `json:"status"`
	Message string      `json:"message"`
	Result  interface{} `json:"result"`
}

type ExecuteContractReq struct {
	Comment            string `json:"comment"`
	ExecutorAddr       string `json:"executorAddr"`
	QuorumType         int    `json:"quorumType"`
	SmartContractData  string `json:"smartContractData"`
	SmartContractToken string `json:"smartContractToken"`
}

type InferenceInfo struct {
	Timestamp string            `json:"timestamp"`
	Records   []InferenceRecord `json:"records"`
	Info      string            `json:"info"`
}

type SmartContractFuncInput struct {
	AssetID       string `json:"asset_id"`
	InferenceInfo string `json:"inference_info"`
	AssetValue    string `json:"asset_value"`
	DepinDID      string `json:"depin_did"`
}

func prepareSmartContractData(inferenceRecords []InferenceRecord, depinDID string) (string, error) {
	currTimestamp := strconv.FormatInt(time.Now().Unix(), 10)
	if len(inferenceRecords) == 0 {
		return "", fmt.Errorf("unexpected error: no inference records found while executing smart contract")
	}

	// 0th index is the asseumption that all records have the same asset_id
	info := "Inference records for asset ID: " + inferenceRecords[0].AssetID + " at " + currTimestamp

	var inferenceInfo *InferenceInfo = &InferenceInfo{
		Timestamp: currTimestamp,
		Records:   inferenceRecords,
		Info:      info,
	}

	inferenceInfoBytes, err := json.Marshal(inferenceInfo)
	if err != nil {
		return "", fmt.Errorf("failed to marshal inference info: %v", err)
	}

	contractMsg := map[string]*SmartContractFuncInput{
		"store_inference": {
			AssetID:       inferenceRecords[0].AssetID,
			InferenceInfo: string(inferenceInfoBytes),
			AssetValue:    inferenceRecords[0].AssetValue,
			DepinDID:      depinDID,
		},
	}

	smartContractData, err := json.Marshal(contractMsg)
	if err != nil {
		return "", fmt.Errorf("failed to marshal smart contract data: %v", err)
	}

	return string(smartContractData), nil
}

func ExecuteSmartContract(inferenceRecords []InferenceRecord, rubixNodeAddress string) error {
	depinDID := os.Getenv("DEPIN_DID")
	if depinDID == "" {
		return fmt.Errorf("DEPIN_DID environment variable is not set")
	}

	smartContractData, err := prepareSmartContractData(inferenceRecords, depinDID)
	if err != nil {
		return fmt.Errorf("failed to prepare smart contract data: %v", err)
	}

	inferenceStorageContractAddress := os.Getenv("INFERENCE_STORAGE_CONTRACT_ADDRESS")

	executeContractReq := &ExecuteContractReq{
		Comment:            "storing inference",
		ExecutorAddr:       depinDID,
		QuorumType:         2,
		SmartContractData:  smartContractData,
		SmartContractToken: inferenceStorageContractAddress,
	}

	executeContractReqBytes, err := json.Marshal(executeContractReq)
	if err != nil {
		return fmt.Errorf("failed to marshal execute contract request: %v", err)
	}

	resp, err := http.Post(rubixNodeAddress, "application/json", bytes.NewBuffer(executeContractReqBytes))
	if err != nil {
		return fmt.Errorf("error forwarding request to Rubix node: %v", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading response from Rubix node for execute smart contract: %v", err)
	}

	var execSmartContractResponse map[string]interface{}
	if err := json.Unmarshal(respBody, &execSmartContractResponse); err != nil {
		return fmt.Errorf("error unmarshalling response from Rubix node for execute smart contract: %v", err)
	}

	if _, ok := execSmartContractResponse["result"]; !ok {
		return fmt.Errorf("unexpected response from Rubix node for execute smart contract: %v", execSmartContractResponse)
	}

	var smartContractResult map[string]interface{}
	smartContractResult, ok := execSmartContractResponse["result"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("unexpected result format from Rubix node for execute smart contract: %v", execSmartContractResponse["result"])
	}

	responseId := smartContractResult["id"].(string)

	_, err = signatureResponse(responseId, rubixNodeAddress)
	if err != nil {
		return fmt.Errorf("failed to get signature response: %v", err)
	}

	return nil
}


func signatureResponse(requestId string, nodeAddress string) (*BasicResponse, error) {
	data := map[string]interface{}{
		"id":       requestId,
		"mode":     0,
		"password": "mypassword",
	}

	bodyJSON, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("Error marshaling JSON:", err)
	}

	url, err := url.JoinPath(nodeAddress, "/api/signature-response")
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(bodyJSON))
	if err != nil {
		return nil, fmt.Errorf("Error creating HTTP request:", err)
	}

	req.Header.Set("Content-Type", "application/json; charset=UTF-8")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Error sending HTTP request:", err)
	}
	defer resp.Body.Close()

	fmt.Println("Response Status:", resp.Status)
	
	responseBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Error reading response body: %s\n", err)
	}

	// Process the data as needed
	var response *BasicResponse
	if err := json.Unmarshal(responseBytes, &response); err != nil {
		return nil, fmt.Errorf("Error unmarshaling response body: %s\n", err)
	}

	return response, nil
}
