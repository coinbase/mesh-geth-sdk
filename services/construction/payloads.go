// Copyright 2022 Coinbase, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package construction

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/coinbase/rosetta-geth-sdk/client"
	sdkTypes "github.com/coinbase/rosetta-geth-sdk/types"

	"github.com/coinbase/rosetta-sdk-go/parser"
	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/ethereum/go-ethereum/common"
	EthTypes "github.com/ethereum/go-ethereum/core/types"
)

// ConstructionPayloads implements /construction/payloads endpoint
//
// Payloads is called with an array of operations and the response from /construction/Metadata.
// It returns an unsigned Transaction blob and a collection of payloads that must
// be signed by particular AccountIdentifiers using a certain SignatureType.
// The array of operations provided in Transaction construction often times can
// not specify all "effects" of a Transaction (consider invoked transactions in Ethereum).
// However, they can deterministically specify the "intent" of the Transaction,
// which is sufficient for construction. For this reason, parsing the corresponding
// Transaction in the Data API (when it lands on chain) will contain a superset of
// whatever operations were provided during construction.
//
func (s *APIService) ConstructionPayloads(
	ctx context.Context,
	req *types.ConstructionPayloadsRequest) (*types.ConstructionPayloadsResponse, *types.Error) {
	isContractCall := false
	if _, ok := req.Metadata["method_signature"]; ok {
		isContractCall = true
	}
	operationDescriptions, err := s.CreateOperationDescription(req.Operations, isContractCall)
	if err != nil {
		return nil, sdkTypes.WrapErr(sdkTypes.ErrInvalidInput, err)
	}

	descriptions := &parser.Descriptions{
		OperationDescriptions: operationDescriptions,
		ErrUnmatched:          true,
	}

	matches, err := parser.MatchOperations(descriptions, req.Operations)
	if err != nil {
		return nil, sdkTypes.WrapErr(sdkTypes.ErrInternalError, err)
	}

	var metadata client.Metadata
	if err := client.UnmarshalJSONMap(req.Metadata, &metadata); err != nil {
		return nil, sdkTypes.WrapErr(sdkTypes.ErrInternalError, err)
	}

	toOp, amount := matches[1].First()
	toAddress := toOp.Account.Address
	nonce := metadata.Nonce
	gasPrice := metadata.GasPrice
	gasLimit := metadata.GasLimit

	chainID := s.config.ChainConfig.ChainID

	fromOp, _ := matches[0].First()
	fromAddress := fromOp.Account.Address
	fromCurrency := fromOp.Amount.Currency

	checkFrom, ok := client.ChecksumAddress(fromAddress)
	if !ok {
		return nil, sdkTypes.WrapErr(
			sdkTypes.ErrInvalidInput,
			fmt.Errorf("%s is not a valid address", fromAddress),
		)
	}

	checkTo, ok := client.ChecksumAddress(toAddress)
	if !ok {
		return nil, sdkTypes.WrapErr(
			sdkTypes.ErrInvalidInput,
			fmt.Errorf("%s is not a valid address", toAddress),
		)
	}
	var transferData []byte
	var sendToAddress common.Address

	switch {
	case isContractCall:
		contractData, err := hexutil.Decode(metadata.ContractData)
		if err != nil {
			return nil, sdkTypes.WrapErr(sdkTypes.ErrInvalidInput, err)
		}

		data, err := constructContractCallData(metadata.MethodSignature, metadata.MethodArgs)
		if err != nil {
			return nil, sdkTypes.WrapErr(sdkTypes.ErrInvalidInput, err)
		}
		res := bytes.Compare(data, contractData)
		if res != 0 {
			return nil, sdkTypes.WrapErr(
				sdkTypes.ErrInvalidInput,
				fmt.Errorf("invalid conctract data"),
			)
		}
		transferData = contractData
		sendToAddress = common.HexToAddress(checkTo)
	case types.Hash(fromCurrency) == types.Hash(s.config.RosettaCfg.Currency):
		transferData = []byte{}
		sendToAddress = common.HexToAddress(checkTo)
	default:
		// ERC20 logic
		contract, ok := fromCurrency.Metadata[client.ContractAddressMetadata].(string)
		if !ok {
			return nil, sdkTypes.WrapErr(
				sdkTypes.ErrInvalidInput,
				fmt.Errorf(
					"%s currency doesn't have a contract address in Metadata",
					fromCurrency.Symbol,
				),
			)
		}

		transferData = client.GenerateErc20TransferData(toAddress, amount)
		sendToAddress = common.HexToAddress(contract)
		amount = big.NewInt(0)
	}

	tx := EthTypes.NewTransaction(
		nonce,
		sendToAddress,
		amount,
		gasLimit,
		gasPrice,
		transferData,
	)

	unsignedTx := &client.Transaction{
		From:     checkFrom,
		To:       sendToAddress.Hex(),
		Value:    amount,
		Data:     tx.Data(),
		Nonce:    tx.Nonce(),
		GasPrice: gasPrice,
		GasLimit: tx.Gas(),
		ChainID:  chainID,
		Currency: fromCurrency,
	}

	// Construct SigningPayload
	signer := EthTypes.LatestSignerForChainID(chainID)

	payload := &types.SigningPayload{
		AccountIdentifier: &types.AccountIdentifier{Address: checkFrom},
		Bytes:             signer.Hash(tx).Bytes(),
		SignatureType:     types.EcdsaRecovery,
	}

	unsignedTxJSON, err := json.Marshal(unsignedTx)
	if err != nil {
		return nil, sdkTypes.WrapErr(sdkTypes.ErrInvalidInput, err)
	}

	return &types.ConstructionPayloadsResponse{
		UnsignedTransaction: string(unsignedTxJSON),
		Payloads:            []*types.SigningPayload{payload},
	}, nil
}
