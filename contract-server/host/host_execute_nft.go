package host

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/bytecodealliance/wasmtime-go"
	"github.com/rubixchain/rubix-wasm/go-wasm-bridge/context"
	"github.com/rubixchain/rubix-wasm/go-wasm-bridge/host"
	"github.com/rubixchain/rubix-wasm/go-wasm-bridge/utils"

	wasmContext "github.com/rubixchain/rubix-wasm/go-wasm-bridge/context"
)

type ExecuteNFTReq struct {
	NFT        string  `json:"nft"`
	Executor   string  `json:"executor"`
	Receiver   string  `json:"receiver"`
	Comment    string  `json:"comment"`
	NFTValue   float64 `json:"nft_value"`
	NFTData    string  `json:"nft_data"`
	QuorumType int32   `json:"quorum_type"`
}

type DoExecuteNFT struct {
	allocFunc   *wasmtime.Func
	memory      *wasmtime.Memory
	nodeAddress string
	quorumType  int
	wasmContext *context.WasmContext
}

func NewDoExecuteNFT() *DoExecuteNFT {
	return &DoExecuteNFT{}
}
func (h *DoExecuteNFT) Name() string {
	return "do_execute_nft"
}
func (h *DoExecuteNFT) FuncType() *wasmtime.FuncType {
	return wasmtime.NewFuncType(
		[]*wasmtime.ValType{
			wasmtime.NewValType(wasmtime.KindI32), // input_ptr
			wasmtime.NewValType(wasmtime.KindI32), // input_len
			wasmtime.NewValType(wasmtime.KindI32), // resp_ptr_ptr
			wasmtime.NewValType(wasmtime.KindI32), // resp_len_ptr
		},
		[]*wasmtime.ValType{wasmtime.NewValType(wasmtime.KindI32)}, // return i32
	)
}

func (h *DoExecuteNFT) Initialize(allocFunc, deallocFunc *wasmtime.Func, memory *wasmtime.Memory, nodeAddress string, quorumType int, wasmCtx *wasmContext.WasmContext) {
	h.allocFunc = allocFunc
	h.memory = memory
	h.nodeAddress = nodeAddress
	h.quorumType = quorumType
	h.wasmContext = wasmCtx
}

func (h *DoExecuteNFT) Callback() host.HostFunctionCallBack {
	return h.callback
}

func callExecuteNFTAPI(nodeAddress string, quorumType int, executeNFTdata ExecuteNFTReq) error {
	executeNFTdata.QuorumType = int32(quorumType)
	fmt.Println("printing the data in callExecuteNFTAPI function is:", executeNFTdata)

	executeNFTdataBytes, _ := json.Marshal(executeNFTdata)

	executeNFTUrl, err := url.JoinPath(nodeAddress, "/api/execute-nft")
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", executeNFTUrl, bytes.NewBuffer(executeNFTdataBytes))
	if err != nil {
		fmt.Println("Error creating HTTP request:", err)
		return err
	}

	req.Header.Set("Content-Type", "application/json; charset=UTF-8")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending HTTP request:", err)
		return err
	}
	

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response body: %s\n", err)
		return err
	}

	var executeNFTResponse map[string]interface{}
	err3 := json.Unmarshal(responseBody, &executeNFTResponse)
	if err3 != nil {
		fmt.Println("Error unmarshaling response:", err3)
		return err3
	}

	result, ok := executeNFTResponse["result"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("unexpected response format: %v", executeNFTResponse)
	}

	id := result["id"].(string)
	
	defer resp.Body.Close()
	
	_, err = signatureResponse(id, nodeAddress)

	return err
}

func (h *DoExecuteNFT) callback(
	caller *wasmtime.Caller,
	args []wasmtime.Val,
) ([]wasmtime.Val, *wasmtime.Trap) {
	inputArgs, outputArgs := utils.HostFunctionParamExtraction(args, true, true)

	// Extract input bytes and convert to string
	inputBytes, memory, err := utils.ExtractDataFromWASM(caller, inputArgs)
	if err != nil {
		fmt.Println("Failed to extract data from WASM", err)
		return utils.HandleError(err.Error())
	}
	h.memory = memory // Assign memory to Host struct for future use
	var executeNFTData ExecuteNFTReq

	//Unmarshaling the data which has been read from the wasm memory
	err3 := json.Unmarshal(inputBytes, &executeNFTData)
	if err3 != nil {
		fmt.Println("Error unmarshaling response in callback function:", err3)
		errMsg := "Error unmashalling response in callback function" + err3.Error()
		return utils.HandleError(errMsg)
	}
	callExecuteNFTAPIRespErr := callExecuteNFTAPI(h.nodeAddress, h.quorumType, executeNFTData)
	if callExecuteNFTAPIRespErr != nil {
		fmt.Println("failed to execute NFT", callExecuteNFTAPIRespErr)
		errMsg := "failed to execute NFT" + callExecuteNFTAPIRespErr.Error()
		return utils.HandleError(errMsg)
	}

	responseStr := "success"
	err = utils.UpdateDataToWASM(caller, h.allocFunc, responseStr, outputArgs)
	if err != nil {
		fmt.Println("Failed to update data to WASM", err)
		return utils.HandleError(err.Error())
	}

	return utils.HandleOk() // Success

}
