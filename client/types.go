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

package client

import (
	"encoding/json"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"

	RosettaTypes "github.com/coinbase/rosetta-sdk-go/types"

	EthTypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/ethereum/go-ethereum/common"
)

const (
	gethHTTPTimeout = 120 * time.Second

	maxTraceConcurrency  = int64(16) // nolint:gomnd
	semaphoreTraceWeight = int64(1)  // nolint:gomnd

	// ERC20 Standard Definition for the Transfer Event Logs Topics
	Erc20TransferEventLogTopics = "Transfer(address,address,uint256)"

	// While parsing ERC20 ops, we will ignore any event logs that we think are an ERC20 transfer
	// that do not contain 3 topics and who's `data` field is not a single 32 byte hex string
	// representing the amount of the transfer
	NumTopicsERC20Transfer = 3
	UnknownERC20Symbol     = "ERC20_UNKNOWN"
	UnknownERC20Decimals   = 0

	UnknownERC721Symbol   = "ERC721_UNKNOWN"
	UnknownERC721Decimals = 0

	// eip1559TxType is the EthTypes.Transaction.Type() value that indicates this Transaction
	// follows EIP-1559.
	eip1559TxType = 2

	ContractAddressMetadata = "contractAddress"
)

type ContractCurrency struct {
	Symbol   string `json:"symbol"`
	Decimals int32  `json:"decimals"`
}

type RPCBlock struct {
	Hash         common.Hash      `json:"hash"`
	Transactions []RPCTransaction `json:"transactions"`
	UncleHashes  []common.Hash    `json:"uncles"`
}

type TxExtraInfo struct {
	BlockNumber *string         `json:"blockNumber,omitempty"`
	BlockHash   *common.Hash    `json:"blockHash,omitempty"`
	From        *common.Address `json:"from,omitempty"`
	TxHash      *common.Hash    `json:"hash,omitempty"`
}

type Metadata struct {
	Nonce           uint64   `json:"nonce"`
	GasPrice        *big.Int `json:"gas_price"`
	GasLimit        uint64   `json:"gas_limit"`
	ContractData    string   `json:"data,omitempty"`
	MethodSignature string   `json:"method_signature,omitempty"`
	MethodArgs      []string `json:"method_args,omitempty"`
}

type ParseMetadata struct {
	Nonce    uint64   `json:"nonce"`
	GasPrice *big.Int `json:"gas_price"`
	ChainID  *big.Int `json:"chain_id"`
}

type Transaction struct {
	From  string   `json:"from"`
	To    string   `json:"to"`
	Value *big.Int `json:"value"`
	Data  []byte   `json:"data"`
	// ContractData     string          `json:"contractData"`
	Nonce    uint64                 `json:"nonce"`
	GasPrice *big.Int               `json:"gas_price"`
	GasLimit uint64                 `json:"gas"`
	ChainID  *big.Int               `json:"chain_id"`
	Currency *RosettaTypes.Currency `json:"currency,omitempty"`
}

type LoadedTransaction struct {
	Transaction *EthTypes.Transaction
	From        *common.Address
	BlockNumber *string
	BlockHash   *common.Hash
	TxHash      *common.Hash // may not equal Transaction.Hash() due to state sync indicator
	FeeAmount   *big.Int
	FeeBurned   *big.Int // nil if no fees were burned
	Miner       string
	Author      string
	Status      bool

	Trace    []*FlatCall
	RawTrace json.RawMessage
	Receipt  *RosettaTxReceipt

	BaseFee      *big.Int
	IsBridgedTxn bool
}

type SignedTransactionWrapper struct {
	SignedTransaction []byte                 `json:"signed_tx"`
	Currency          *RosettaTypes.Currency `json:"currency,omitempty"`
}

// EthTypes.Transaction contains TxData, which is DynamicFeeTx:
// https://github.com/ethereum/go-ethereum/blob/980b7682b474db61ecbd78171e7cacfec8214048
// /core/types/dynamic_fee_tx.go#L25
type RPCTransaction struct {
	Tx *EthTypes.Transaction
	TxExtraInfo
}

type RosettaTxReceipt struct {
	Type           uint8 `json:"type,omitempty"`
	GasPrice       *big.Int
	GasUsed        *big.Int
	TransactionFee *big.Int
	Logs           []*EthTypes.Log
	RawMessage     json.RawMessage
	Status         uint64 `json:"status"`
}

type FeeSetResult struct {
	L1Transaction *hexutil.Big `json:"l1Transaction"`
	L1Calldata    *hexutil.Big `json:"l1Calldata"`
	L2Storage     *hexutil.Big `json:"l2Storage"`
	L2Computation *hexutil.Big `json:"l2Computation"`
}

type FeeStatsResult struct {
	Prices    *FeeSetResult `json:"prices"`
	UnitsUsed *FeeSetResult `json:"unitsUsed"`
	Paid      *FeeSetResult `json:"paid"`
}

type L1InboxBatchInfo struct {
	Confirmations *hexutil.Big   `json:"confirmations"`
	BlockNumber   *hexutil.Big   `json:"blockNumber"`
	LogAddress    common.Address `json:"logAddress"`
	LogTopics     []common.Hash  `json:"logTopics"`
	LogData       hexutil.Bytes  `json:"logData"`
}

type PayloadsResponse struct {
	TransferData []byte
	Address      common.Address
	Amount       *big.Int
}

type Options struct {
	From                   string                 `json:"from"`
	To                     string                 `json:"to"`
	TokenAddress           string                 `json:"token_address,omitempty"`
	ContractAddress        string                 `json:"contract_address,omitempty"`
	Value                  string                 `json:"value"`
	SuggestedFeeMultiplier *float64               `json:"suggested_fee_multiplier,omitempty"`
	GasPrice               *big.Int               `json:"gas_price,omitempty"`
	GasLimit               *big.Int               `json:"gas_limit,omitempty"`
	Nonce                  *big.Int               `json:"nonce,omitempty"`
	Currency               *RosettaTypes.Currency `json:"currency,omitempty"`
	MethodSignature        string                 `json:"method_signature,omitempty"`
	MethodArgs             []string               `json:"method_args,omitempty"`
	ContractData           string                 `json:"data,omitempty"`
}

// Receipt represents the results of a transaction.
type GetTransactionReceiptResult struct {
	TransactionHash   common.Hash     `json:"transactionHash"`
	TransactionIndex  hexutil.Uint64  `json:"transactionIndex"`
	BlockHash         common.Hash     `json:"blockHash"`
	BlockNumber       *hexutil.Big    `json:"blockNumber"`
	From              common.Address  `json:"from"`
	To                *common.Address `json:"to"`
	CumulativeGasUsed hexutil.Uint64  `json:"cumulativeGasUsed"`
	GasUsed           hexutil.Uint64  `json:"gasUsed"`
	EffectiveGasPrice hexutil.Uint64  `json:"effectiveGasPrice"`
	ContractAddress   *common.Address `json:"contractAddress"`
	Logs              []*EthTypes.Log `json:"logs"`
	LogsBloom         hexutil.Bytes   `json:"logsBloom"`
	Status            hexutil.Uint64  `json:"status"`

	// L2 Specific Fields
	ReturnCode       hexutil.Uint64    `json:"returnCode"`
	ReturnData       hexutil.Bytes     `json:"returnData"`
	FeeStats         *FeeStatsResult   `json:"feeStats"`
	L1BlockNumber    *hexutil.Big      `json:"l1BlockNumber"`
	L1InboxBatchInfo *L1InboxBatchInfo `json:"l1InboxBatchInfo"`
	Type             string            `json:"type,omitempty"`
	PostState        []byte            `json:"root"`
}
