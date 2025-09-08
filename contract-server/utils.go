package main

import (
	"fmt"

	"github.com/gin-gonic/gin"
)

type ContractInputRequest struct {
	Port              string `json:"port"`
	SmartContractHash string `json:"smart_contract_hash"`
	SmartContractData string `json:"smart_contract_data"`
	InitiatorDID      string `json:"initiator_did"`
}

func wrapError(f func(code int, obj any), msg string) {
	fmt.Println(msg)
	f(404, gin.H{"message": msg})
}
