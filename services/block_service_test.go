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

package services

import (
	"context"
	"encoding/json"
	"io/ioutil"

	EthTypes "github.com/ethereum/go-ethereum/core/types"

	"math/big"
	"testing"

	"github.com/coinbase/rosetta-geth-sdk/client"
	mockedServices "github.com/coinbase/rosetta-geth-sdk/mocks/services"

	AssetTypes "github.com/coinbase/rosetta-geth-sdk/types"

	RosettaTypes "github.com/coinbase/rosetta-sdk-go/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/coinbase/rosetta-geth-sdk/configuration"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const (
	hsh = "0xd83b1dcf7d47c4115d78ce0361587604e8157591b118bd64ada02e86c9d5ca7e"
)

// func loadTokenWhiteList() []configuration.Token {
//	content, err := ioutil.ReadFile("testdata/tokenList.json")
//	if err != nil {
//		log.Fatal("Error when opening file: ", err)
//	}
//
//	var payload []configuration.Token
//	err = json.Unmarshal(content, &payload)
//	if err != nil {
//		log.Fatal("Error during Unmarshal(): ", err)
//	}
//	return payload
// }

func TestBlockService_Offline(t *testing.T) {
	cfg := &configuration.Configuration{
		Mode: configuration.ModeOffline,
	}
	mockClient := &mockedServices.Client{}
	servicer := NewBlockAPIService(cfg, mockClient)
	ctx := context.Background()

	block, err := servicer.Block(ctx, &RosettaTypes.BlockRequest{})
	assert.Nil(t, block)
	assert.Equal(t, AssetTypes.ErrUnavailableOffline.Code, err.Code)
	assert.Equal(t, AssetTypes.ErrUnavailableOffline.Message, err.Message)

	blockTransaction, err := servicer.BlockTransaction(ctx, &RosettaTypes.BlockTransactionRequest{})
	assert.Nil(t, blockTransaction)
	assert.Equal(t, AssetTypes.ErrUnavailableOffline.Code, err.Code)
	assert.Equal(t, AssetTypes.ErrUnavailableOffline.Message, err.Message)

	mockClient.AssertExpectations(t)
}

func TestBlockService_Online(t *testing.T) {
	cfg := &configuration.Configuration{
		Mode: configuration.ModeOnline,
	}
	mockClient := &mockedServices.Client{}
	servicer := NewBlockAPIService(cfg, mockClient)
	ctx := context.Background()

	block := &RosettaTypes.Block{
		BlockIdentifier: &RosettaTypes.BlockIdentifier{
			Index: 10992,
			Hash:  "0xba9ded5ca1ec9adb9451bf062c9de309d9552fa0f0254a7b982d3daf7ae436ae",
		},
	}

	blockResponse := &RosettaTypes.BlockResponse{
		Block: block,
	}

	t.Run("populated identifier", func(t *testing.T) {
		mockClient.On(
			"CallContext",
			ctx,
			mock.Anything,
			"eth_getBlockByNumber",
			"latest",
			true,
		).Return(
			nil,
		).Run(
			func(args mock.Arguments) {
				r := args.Get(1).(*json.RawMessage)

				file, err := ioutil.ReadFile("testdata/block_10992.json")
				assert.NoError(t, err)

				*r = json.RawMessage(file)
			},
		).Once()

		txs := make([]client.RPCTransaction, 0)
		mockClient.On(
			"TraceBlockByHash",
			ctx,
			mock.Anything,
			txs,
		).Return(
			nil,
			nil,
		).Once()

		var baseFee *big.Int

		mockClient.On(
			"GetBlockReceipts",
			ctx,
			mock.Anything,
			txs,
			baseFee,
		).Return(
			nil,
			nil,
		).Once()

		loadedTxn := make([]*client.LoadedTransaction, 0)
		rosettaTxs := make([]*RosettaTypes.Transaction, 0)
		mockClient.On(
			"PopulateCrossChainTransactions",
			mock.Anything,
			loadedTxn,
		).Return(
			rosettaTxs,
			nil,
		).Once()

		mockClient.On(
			"GetRosettaConfig",
		).Return(
			configuration.RosettaConfig{},
		).Times(4)

		b, err := servicer.Block(ctx, &RosettaTypes.BlockRequest{})
		assert.Nil(t, err)
		assert.Equal(t, blockResponse.Block.BlockIdentifier, b.Block.BlockIdentifier)
	})

	t.Run("orphaned block", func(t *testing.T) {
		pbIdentifier := RosettaTypes.ConstructPartialBlockIdentifier(block.BlockIdentifier)

		mockClient.On(
			"CallContext",
			ctx,
			mock.Anything,
			"eth_getBlockByHash",
			"0xba9ded5ca1ec9adb9451bf062c9de309d9552fa0f0254a7b982d3daf7ae436ae",
			true,
		).Return(
			AssetTypes.ErrClientBlockOrphaned,
		).Run(
			func(args mock.Arguments) {
				r := args.Get(1).(*json.RawMessage)

				file, err := ioutil.ReadFile("testdata/block_10992.json")
				assert.NoError(t, err)

				*r = json.RawMessage(file)
			},
		).Once()

		b, err := servicer.Block(ctx, &RosettaTypes.BlockRequest{
			BlockIdentifier: pbIdentifier,
		})

		assert.Nil(t, b)
		assert.Equal(t, AssetTypes.ErrBlockOrphaned.Code, err.Code)
		assert.Equal(t, AssetTypes.ErrBlockOrphaned.Message, err.Message)
		assert.Equal(t, AssetTypes.ErrBlockOrphaned.Retriable, err.Retriable)
	})

	t.Run("populated transactions", func(t *testing.T) {
		blockWithTxns := &RosettaTypes.Block{
			BlockIdentifier: &RosettaTypes.BlockIdentifier{
				Index: 10994,
				Hash:  "0xb6a2558c2e54bfb11247d0764311143af48d122f29fc408d9519f47d70aa2d50",
			},
		}
		blockResp := &RosettaTypes.BlockResponse{
			Block: blockWithTxns,
		}

		mockClient.On(
			"CallContext",
			ctx,
			mock.Anything,
			"eth_getBlockByNumber",
			"latest",
			true,
		).Return(
			nil,
		).Run(
			func(args mock.Arguments) {
				r := args.Get(1).(*json.RawMessage)

				file, err := ioutil.ReadFile("testdata/block_10994.json")
				assert.NoError(t, err)

				*r = json.RawMessage(file)
			},
		).Once()
		m := make(map[string][]*client.FlatCall)
		m[hsh] = append(m[hsh], &client.FlatCall{
			Type:         "call",
			From:         common.HexToAddress("0x1234"),
			To:           common.HexToAddress("0x4566"),
			Value:        big.NewInt(900000),
			GasUsed:      big.NewInt(10000),
			Revert:       false,
			ErrorMessage: "",
		})

		// TraceBlockByHash returns valid traces map
		mockClient.On(
			"TraceBlockByHash",
			ctx,
			mock.Anything,
			mock.Anything,
		).Return(
			m,
			nil,
		).Once()

		var baseFee *big.Int
		mockClient.On(
			"GetBlockReceipts",
			ctx,
			mock.Anything,
			mock.Anything,
			baseFee,
		).Return(
			nil,
			nil,
		).Once()

		rosettaTxs := make([]*RosettaTypes.Transaction, 0)
		mockClient.On(
			"PopulateCrossChainTransactions",
			mock.Anything,
			mock.Anything,
		).Return(
			rosettaTxs,
			nil,
		).Once()

		mockClient.On(
			"GetRosettaConfig",
		).Return(
			configuration.RosettaConfig{},
		)

		b, err := servicer.Block(ctx, &RosettaTypes.BlockRequest{})
		assert.Nil(t, err)
		assert.Equal(t, blockResp.Block.BlockIdentifier, b.Block.BlockIdentifier)
		assert.Equal(t, 1, len(b.Block.Transactions))
		assert.Equal(t, 2, len(b.Block.Transactions[0].Operations))
		assert.Equal(t, "CALL", b.Block.Transactions[0].Operations[0].Type)
	})

	t.Run("populated transactions and receipts", func(t *testing.T) {
		blockWithTxns := &RosettaTypes.Block{
			BlockIdentifier: &RosettaTypes.BlockIdentifier{
				Index: 10994,
				Hash:  "0xb6a2558c2e54bfb11247d0764311143af48d122f29fc408d9519f47d70aa2d50",
			},
		}
		blockResp := &RosettaTypes.BlockResponse{
			Block: blockWithTxns,
		}

		mockClient.On(
			"CallContext",
			ctx,
			mock.Anything,
			"eth_getBlockByNumber",
			"latest",
			true,
		).Return(
			nil,
		).Run(
			func(args mock.Arguments) {
				r := args.Get(1).(*json.RawMessage)

				file, err := ioutil.ReadFile("testdata/block_10994.json")
				assert.NoError(t, err)

				*r = json.RawMessage(file)
			},
		).Once()

		m := make(map[string][]*client.FlatCall)
		m[hsh] = append(m[hsh], &client.FlatCall{
			Type:         "call",
			From:         common.HexToAddress("0x1234"),
			To:           common.HexToAddress("0x4566"),
			Value:        big.NewInt(900000),
			GasUsed:      big.NewInt(10000),
			Revert:       false,
			ErrorMessage: "",
		})

		// TraceBlockByHash returns valid traces map
		mockClient.On(
			"TraceBlockByHash",
			ctx,
			mock.Anything,
			mock.Anything,
		).Return(
			m,
			nil,
		).Once()

		tokenAddress := common.HexToAddress("0x4DBCdF9B62e891a7cec5A2568C3F4FAF9E8Abe2b")
		erc20TransferEvent := common.HexToHash("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef")
		fromAddress := common.HexToHash("0x0000000000000000000000004dc8f417d4eb731d179a0f08b1feaf25216cefd0")
		toAddress := common.HexToHash("0x0000000000000000000000000d2b2fb39b10cd50cab7aa8e834879069ab1a8d4")

		log := EthTypes.Log{
			Address: tokenAddress,
			Topics:  []common.Hash{erc20TransferEvent, fromAddress, toAddress},
			Data:    []byte{},
			TxHash:  common.HexToHash(hsh),
		}

		mockClient.On(
			"GetContractCurrency",
			mock.Anything,
			mock.Anything,
		).Return(
			&client.ContractCurrency{
				Symbol:   "USDC",
				Decimals: 6,
			},
			nil,
		).Once()

		receipt := client.RosettaTxReceipt{
			TransactionFee: big.NewInt(10000),
			Logs:           []*EthTypes.Log{&log},
		}
		rosettaReceipts := make([]*client.RosettaTxReceipt, 0)
		rosettaReceipts = append(rosettaReceipts, &receipt)

		var baseFee *big.Int
		mockClient.On(
			"GetBlockReceipts",
			ctx,
			mock.Anything,
			mock.Anything,
			baseFee,
		).Return(
			rosettaReceipts,
			nil,
		).Once()

		rosettaTxs := make([]*RosettaTypes.Transaction, 0)
		mockClient.On(
			"PopulateCrossChainTransactions",
			mock.Anything,
			mock.Anything,
		).Return(
			rosettaTxs,
			nil,
		).Once()

		// mockClient.On(
		//	"GetRosettaConfig",
		// ).Return(
		//	configuration.RosettaConfig{
		//		TokenWhiteList: loadTokenWhiteList(),
		//	},
		//)

		b, err := servicer.Block(ctx, &RosettaTypes.BlockRequest{})
		assert.Nil(t, err)
		assert.Equal(t, blockResp.Block.BlockIdentifier, b.Block.BlockIdentifier)
		assert.Equal(t, 1, len(b.Block.Transactions))
		// assert.Equal(t, 4, len(b.Block.Transactions[0].Operations))
		// FEE operation
		assert.Equal(t, "FEE", b.Block.Transactions[0].Operations[0].Type)
		assert.Equal(t, "-10000", b.Block.Transactions[0].Operations[0].Amount.Value)
		// Debit operation
		assert.Equal(t, "-900000", b.Block.Transactions[0].Operations[2].Amount.Value)
		assert.Equal(t, "0x0000000000000000000000000000000000001234",
			b.Block.Transactions[0].Operations[2].Account.Address)
		// ERC20 operations
		//		assert.Equal(t, AssetTypes.OpErc20Transfer, b.Block.Transactions[0].Operations[4].Type)
		//	assert.Equal(t, AssetTypes.OpErc20Transfer, b.Block.Transactions[0].Operations[5].Type)
	})

	// t.Run("populated transactions with unknown token", func(t *testing.T) {
	//	blockWithTxns := &RosettaTypes.Block{
	//		BlockIdentifier: &RosettaTypes.BlockIdentifier{
	//			Index: 10994,
	//			Hash:  "0xb6a2558c2e54bfb11247d0764311143af48d122f29fc408d9519f47d70aa2d50",
	//		},
	//	}
	//	blockResp := &RosettaTypes.BlockResponse{
	//		Block: blockWithTxns,
	//	}
	//
	//	mockClient.On(
	//		"CallContext",
	//		ctx,
	//		mock.Anything,
	//		"eth_getBlockByNumber",
	//		"latest",
	//		true,
	//	).Return(
	//		nil,
	//	).Run(
	//		func(args mock.Arguments) {
	//			r := args.Get(1).(*json.RawMessage)
	//
	//			file, err := ioutil.ReadFile("testdata/block_10994.json")
	//			assert.NoError(t, err)
	//
	//			*r = json.RawMessage(file)
	//		},
	//	).Once()
	//
	//	m := make(map[string][]*client.FlatCall)
	//	m[hsh] = append(m[hsh], &client.FlatCall{
	//		Type:         "call",
	//		From:         common.HexToAddress("0x1234"),
	//		To:           common.HexToAddress("0x4566"),
	//		Value:        big.NewInt(900000),
	//		GasUsed:      big.NewInt(10000),
	//		Revert:       false,
	//		ErrorMessage: "",
	//	})
	//
	//	// TraceBlockByHash returns valid traces map
	//	mockClient.On(
	//		"TraceBlockByHash",
	//		ctx,
	//		mock.Anything,
	//		mock.Anything,
	//	).Return(
	//		m,
	//		nil,
	//	).Once()
	//
	//	// mockClient.On(
	//	//	"GetContractCurrency",
	//	//	mock.Anything,
	//	//	mock.Anything,
	//	// ).Return(
	//	//	&client.ContractCurrency{
	//	//		Symbol:   client.UnknownERC20Symbol,
	//	//		Decimals: client.UnknownERC20Decimals,
	//	//	},
	//	//	nil,
	//	// ).Once()
	//
	//	fakeTokenAddress := common.HexToAddress("0x4DBCdF9B62e891a7cec5A2568C3F4FAF9E8Abe2c")
	//	erc20TransferEvent := common.HexToHash("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef")
	//	fromAddress := common.HexToHash("0x0000000000000000000000004dc8f417d4eb731d179a0f08b1feaf25216cefd0")
	//	toAddress := common.HexToHash("0x0000000000000000000000000d2b2fb39b10cd50cab7aa8e834879069ab1a8d4")
	//
	//	log := EthTypes.Log{
	//		Address: fakeTokenAddress,
	//		Topics:  []common.Hash{erc20TransferEvent, fromAddress, toAddress},
	//		Data:    []byte{},
	//		TxHash:  common.HexToHash(hsh),
	//	}
	//
	//	receipt := client.RosettaTxReceipt{
	//		TransactionFee: big.NewInt(10000),
	//		Logs:           []*EthTypes.Log{&log},
	//	}
	//	rosettaReceipts := make([]*client.RosettaTxReceipt, 0)
	//	rosettaReceipts = append(rosettaReceipts, &receipt)
	//
	//	var baseFee *big.Int
	//	mockClient.On(
	//		"GetBlockReceipts",
	//		ctx,
	//		mock.Anything,
	//		mock.Anything,
	//		baseFee,
	//	).Return(
	//		rosettaReceipts,
	//		nil,
	//	).Once()
	//
	//	rosettaTxs := make([]*RosettaTypes.Transaction, 0)
	//	mockClient.On(
	//		"PopulateCrossChainTransactions",
	//		mock.Anything,
	//		mock.Anything,
	//	).Return(
	//		rosettaTxs,
	//		nil,
	//	).Once()
	//
	//	mockClient.On(
	//		"GetRosettaConfig",
	//	).Return(
	//		configuration.RosettaConfig{
	//			TokenWhiteList: loadTokenWhiteList(),
	//		},
	//	)
	//
	//	b, err := servicer.Block(ctx, &RosettaTypes.BlockRequest{})
	//	assert.Nil(t, err)
	//	// No ERC20 ops because token address is invalid
	//	for _, tx := range b.Block.Transactions {
	//		for _, op := range tx.Operations {
	//			assert.NotEqual(t, AssetTypes.OpErc20Transfer, op.Type)
	//		}
	//	}
	//	assert.Equal(t, blockResp.Block.BlockIdentifier, b.Block.BlockIdentifier)
	//	assert.Equal(t, 1, len(b.Block.Transactions))
	//	assert.Equal(t, 4, len(b.Block.Transactions[0].Operations))
	//	// FEE operation
	//	assert.Equal(t, "FEE", b.Block.Transactions[0].Operations[0].Type)
	//	assert.Equal(t, "-10000", b.Block.Transactions[0].Operations[0].Amount.Value)
	//	// Debit operation
	//	assert.Equal(t, "-900000", b.Block.Transactions[0].Operations[2].Amount.Value)
	//	assert.Equal(t, "0x0000000000000000000000000000000000001234",
	//		b.Block.Transactions[0].Operations[2].Account.Address)
	// })

	mockClient.AssertExpectations(t)
}
