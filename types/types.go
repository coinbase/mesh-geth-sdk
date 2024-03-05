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

package types

import (
	RosettaTypes "github.com/coinbase/rosetta-sdk-go/types"
	"github.com/ethereum/go-ethereum/common"
	EthTypes "github.com/ethereum/go-ethereum/core/types"
)

const (
	// NodeVersion is the version of geth we are using.
	NodeVersion = "1.9.24"

	// Symbol is the symbol value
	// used in Currency.
	Symbol = "ETH"

	// Decimals is the decimals value
	// used in Currency.
	Decimals = 18

	// MinerRewardOpType is used to describe
	// a miner block reward.
	MinerRewardOpType = "MINER_REWARD"

	// UncleRewardOpType is used to describe
	// an uncle block reward.
	UncleRewardOpType = "UNCLE_REWARD"

	// FeeOpType is used to represent fee operations.
	FeeOpType = "FEE"

	// CallOpType is used to represent CALL trace operations.
	CallOpType = "CALL"

	// CreateOpType is used to represent CREATE trace operations.
	CreateOpType = "CREATE"

	// Create2OpType is used to represent CREATE2 trace operations.
	Create2OpType = "CREATE2"

	// InvalidOpType is used to represent operations outside EVM.
	InvalidOpType = "INVALID"

	// SelfDestructOpType is used to represent SELFDESTRUCT trace operations.
	SelfDestructOpType = "SELFDESTRUCT"

	// CallCodeOpType is used to represent CALLCODE trace operations.
	CallCodeOpType = "CALLCODE"

	// DelegateCallOpType is used to represent DELEGATECALL trace operations.
	DelegateCallOpType = "DELEGATECALL"

	// StaticCallOpType is used to represent STATICCALL trace operations.
	StaticCallOpType = "STATICCALL"

	// DestructOpType is a synthetic operation used to represent the
	// deletion of suicided accounts that still have funds at the end
	// of a transaction.
	DestructOpType = "DESTRUCT"

	OpErc20Transfer = "ERC20_TRANSFER"

	OpErc20Mint = "ERC20_MINT"

	OpErc20Burn = "ERC20_BURN"

	// SuccessStatus is the status of any
	// Ethereum operation considered successful.
	SuccessStatus = "SUCCESS"

	// FailureStatus is the status of any
	// Ethereum operation considered unsuccessful.
	FailureStatus = "FAILURE"

	// HistoricalBalanceSupported is whether
	// historical balance is supported.
	HistoricalBalanceSupported = true

	// UnclesRewardMultiplier is the uncle reward
	// multiplier.
	UnclesRewardMultiplier = 32

	// MaxUncleDepth is the maximum depth for
	// an uncle to be rewarded.
	MaxUncleDepth = 8

	// GenesisBlockIndex is the index of the
	// genesis block.
	GenesisBlockIndex = int64(0)

	// TransferGasLimit is the gas limit
	// of a transfer.
	TransferGasLimit = int64(21000) //nolint:gomnd

	// MainnetGethArguments are the arguments to start a mainnet geth instance.
	MainnetGethArguments = `--config=/app/ethereum/geth.toml --gcmode=archive --graphql`

	// IncludeMempoolCoins does not apply to rosetta-ethereum as it is not UTXO-based.
	IncludeMempoolCoins = false

	Online = "ONLINE"

	Offline = "OFFLINE"
)

var (
	// OperationTypes are all suppoorted operation types.
	OperationTypes = []string{
		MinerRewardOpType,
		UncleRewardOpType,
		FeeOpType,
		CallOpType,
		CreateOpType,
		Create2OpType,
		OpErc20Transfer,
		SelfDestructOpType,
		CallCodeOpType,
		DelegateCallOpType,
		StaticCallOpType,
		DestructOpType,
		OpErc20Mint,
		OpErc20Burn,
	}

	// OperationStatuses are all supported operation statuses.
	OperationStatuses = []*RosettaTypes.OperationStatus{
		{
			Status:     SuccessStatus,
			Successful: true,
		},
		{
			Status:     FailureStatus,
			Successful: false,
		},
	}

	// CallMethods are all supported call methods.
	CallMethods = []string{
		"eth_getBlockByNumber",
		"eth_getTransactionReceipt",
		"eth_call",
		"eth_estimateGas",
	}

	Currency = &RosettaTypes.Currency{
		Symbol:   Symbol,
		Decimals: Decimals,
	}
)

type Types struct {
	// OperationTypes are all supported operation types.
	OperationTypes []string

	// OperationStatuses are all supported operation statuses.
	OperationStatuses []*RosettaTypes.OperationStatus

	// CallMethods are all supported call methods.
	CallMethods []string

	Currency *RosettaTypes.Currency

	HistoricalBalanceSupported bool
	NodeVersion                string
}

// CallType returns a boolean indicating
// if the provided trace type is a call type.
func CallType(t string) bool {
	callTypes := []string{
		CallOpType,
		CallCodeOpType,
		DelegateCallOpType,
		StaticCallOpType,
	}

	for _, callType := range callTypes {
		if callType == t {
			return true
		}
	}

	return false
}

// CreateType returns a boolean indicating
// if the provided trace type is a create type.
func CreateType(t string) bool {
	createTypes := []string{
		CreateOpType,
		Create2OpType,
	}

	for _, createType := range createTypes {
		if createType == t {
			return true
		}
	}

	return false
}

func LoadTypes() *Types {
	types := &Types{}

	types.OperationStatuses = OperationStatuses
	types.OperationTypes = OperationTypes
	types.CallMethods = CallMethods
	types.Currency = Currency
	types.HistoricalBalanceSupported = HistoricalBalanceSupported
	types.NodeVersion = NodeVersion

	return types
}

// ExtendedHeader Temporarily fix for Sepolia block hash issue
// We are using an old version of go-ethereum. As a result, block hash could not computed correctly on the fly.
// As a work around, we use block hash returned from node to unblock processing
// TODO: revert once we upgrade go-ethereum to latest version
type ExtendedHeader struct {
	*EthTypes.Header
	*BlockHash
}

// Hash returns the block hash from node
func (eh *ExtendedHeader) Hash() common.Hash {
	return eh.BlockHash.Hash
}

type BlockHash struct {
	Hash common.Hash `json:"hash"`
}
