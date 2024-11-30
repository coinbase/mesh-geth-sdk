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
	"context"
	"fmt"

	"github.com/coinbase/rosetta-geth-sdk/configuration"
	"github.com/coinbase/rosetta-geth-sdk/services"
	RosettaTypes "github.com/coinbase/rosetta-sdk-go/types"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"

	"log"
	"math/big"

	evmClient "github.com/coinbase/rosetta-geth-sdk/client"
	sdkTypes "github.com/coinbase/rosetta-geth-sdk/types"
	EthTypes "github.com/ethereum/go-ethereum/core/types"
)

type EthereumClient struct {
	// Use embedding for inheritance. So all the methods of the SDKClient
	// are instantly available on EthereumClient.
	evmClient.SDKClient
}

func (c *EthereumClient) ParseOps(
	tx *evmClient.LoadedTransaction,
) ([]*RosettaTypes.Operation, error) {
	var ops []*RosettaTypes.Operation

	// Compute fee operations
	feeOps := services.FeeOps(tx)
	ops = append(ops, feeOps...)

	traceOps := services.TraceOps(tx.Trace, len(ops))
	ops = append(ops, traceOps...)

	return ops, nil
}

func (c *EthereumClient) GetBlockReceipts(
	ctx context.Context,
	blockHash common.Hash,
	txs []evmClient.RPCTransaction,
	baseFee *big.Int,
) ([]*evmClient.RosettaTxReceipt, error) {
	if len(txs) == 0 {
		return []*evmClient.RosettaTxReceipt{}, nil
	}

	var ethReceipts []*EthTypes.Receipt
	err := c.CallContext(ctx, &ethReceipts, "eth_getBlockReceipts", blockHash.Hex())
	if err != nil {
		return nil, err
	}

	if len(ethReceipts) != len(txs) {
		return nil, fmt.Errorf("mismatch in number of transactions and receipts")
	}

	receipts := make([]*evmClient.RosettaTxReceipt, len(txs))
	for i, ethReceipt := range ethReceipts {
		gasPrice, err := evmClient.EffectiveGasPrice(txs[i].Tx, baseFee)
		if err != nil {
			return nil, err
		}
		gasUsed := new(big.Int).SetUint64(ethReceipt.GasUsed)
		feeAmount := new(big.Int).Mul(gasUsed, gasPrice)

		receipts[i] = &evmClient.RosettaTxReceipt{
			Type:           ethReceipt.Type,
			GasPrice:       gasPrice,
			GasUsed:        gasUsed,
			Logs:           ethReceipt.Logs,
			RawMessage:     nil,
			TransactionFee: feeAmount,
		}

		if ethReceipt.BlockHash != blockHash {
			return nil, fmt.Errorf(
				"expected block hash %s for Transaction but got %s: %w",
				blockHash.Hex(),
				ethReceipt.BlockHash.Hex(),
				sdkTypes.ErrClientBlockOrphaned,
			)
		}
	}

	return receipts, nil
}

func (c *EthereumClient) GetTransactionReceipt(
	ctx context.Context,
	tx *evmClient.LoadedTransaction,
) (*evmClient.RosettaTxReceipt, error) {
	var r *EthTypes.Receipt
	err := c.CallContext(ctx, &r, "eth_getTransactionReceipt", tx.TxHash)
	if err == nil {
		if r == nil {
			return nil, ethereum.NotFound
		}
	}
	gasPrice, err := evmClient.EffectiveGasPrice(tx.Transaction, tx.BaseFee)
	if err != nil {
		return nil, err
	}
	gasUsed := new(big.Int).SetUint64(r.GasUsed)
	feeAmount := new(big.Int).Mul(gasUsed, gasPrice)

	return &evmClient.RosettaTxReceipt{
		GasPrice:       gasPrice,
		GasUsed:        gasUsed,
		Logs:           r.Logs,
		RawMessage:     nil,
		TransactionFee: feeAmount,
	}, err
}

// GetNativeTransferGasLimit is Ethereum's custom implementation of estimating gas.
func (c *EthereumClient) GetNativeTransferGasLimit(ctx context.Context, toAddress string,
	fromAddress string, value *big.Int) (uint64, error) {
	if len(toAddress) == 0 || value == nil {
		// We guard against malformed inputs that may have been generated using
		// a previous version of asset's rosetta
		return 21000, nil
	}
	to := common.HexToAddress(toAddress)
	return c.EstimateGas(ctx, ethereum.CallMsg{
		From:  common.HexToAddress(fromAddress),
		To:    &to,
		Value: big.NewInt(0),
	})
}

// NewEthereumClient creates a eth client that can interact with
// Ethereum network.
func NewEthereumClient(cfg *configuration.Configuration) (*EthereumClient, error) {
	// Use SDK to quickly create a client that support JSON RPC calls
	evmClient, err := evmClient.NewClient(cfg, nil, nil)

	if err != nil {
		log.Fatalln("cannot initialize client: %w", err)
		return nil, err
	}

	// Use embedding for inheritance. So all the methods of the SDKClient
	// are instantly available on EthereumClient.
	p := &EthereumClient{
		*evmClient,
	}

	return p, err
}
