#!/bin/bash

# Rubix Node URL 
RUBIX_MAINNET_NODE_API="http://localhost:20003/api/subscribe-smart-contract"
RUBIX_TESTNET_NODE_API="http://localhost:20000/api/subscribe-smart-contract"

MAINNET_SMART_CONTRACT_TOKENS=()

for TOKEN in "${MAINNET_SMART_CONTRACT_TOKENS[@]}"
do
  echo "Subscribing smart contract: $TOKEN"

  curl --location --request POST "$RUBIX_MAINNET_NODE_API" \
    --header 'Content-Type: application/json' \
    --data "{
      \"smartContractToken\": \"$TOKEN\"
    }"
done

TESTNET_SMART_CONTRACT_TOKENS=(
  # Inference Exec Contract: Writes inference details to the NFT Asset chain
  "QmYXmJNx4DpnBGEMRKj11WWa6iAtBQaUQWht362HP8WLaP"
)

for TOKEN in "${TESTNET_SMART_CONTRACT_TOKENS[@]}"
do
  echo "Subscribing smart contract: $TOKEN"

  curl --location --request POST "$RUBIX_TESTNET_NODE_API" \
    --header 'Content-Type: application/json' \
    --data "{
      \"smartContractToken\": \"$TOKEN\"
    }"
done