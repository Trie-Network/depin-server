package rubix

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

)

func SubscribeNFT(rubixNodeAddress string, nftId string) error {
	subscribeSmartContractReq := map[string]interface{}{
		"nft": nftId,
	}

	subscribeSmartContractReqBytes, err := json.Marshal(subscribeSmartContractReq)
	if err != nil {
		return fmt.Errorf("failed to marshal subscribe smart contract request: %v", err)
	}

	subscribeContractURL, err := url.JoinPath(rubixNodeAddress, "/api/subscribe-nft")
	if err != nil {
		return fmt.Errorf("error joining URL path: %v", err)
	}

	resp, err := http.Post(subscribeContractURL, "application/json", bytes.NewBuffer(subscribeSmartContractReqBytes))
	if err != nil {
		return fmt.Errorf("error forwarding request to Rubix node: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("error reading response body: %v", err)
		}
		return fmt.Errorf("unexpected response from Rubix node: %s", respBody)
	}

	return nil
}

