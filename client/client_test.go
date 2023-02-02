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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"strconv"
	"testing"

	"github.com/coinbase/rosetta-geth-sdk/configuration"
	mocks "github.com/coinbase/rosetta-geth-sdk/mocks/client"

	RosettaTypes "github.com/coinbase/rosetta-sdk-go/types"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/sync/semaphore"
)

func TestOpenEthTraceAPI_EmptyTrace(t *testing.T) {
	ctx := context.Background()

	mockJSONRPC := &mocks.JSONRPC{}
	txHash := "0xb358c6958b1cab722752939cbb92e3fec6b6023de360305910ce80c56c3dad9d"
	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"_replayBlockTransactions",
		txHash,
		[]string{"trace"},
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*json.RawMessage)

			file, err := ioutil.ReadFile(
				"testdata/trace_block_empty.json",
			)
			assert.NoError(t, err)

			*r = json.RawMessage(file)
		},
	).Once()

	rpcClient := &RPCClient{
		JSONRPC: mockJSONRPC,
	}
	sdkClient := &SDKClient{
		RPCClient: rpcClient,
	}

	m, err := sdkClient.TraceReplayBlockTransactions(ctx, txHash)
	assert.NoError(t, err)
	assert.Equal(t, len(m), 0)

	mockJSONRPC.AssertExpectations(t)
}

func TestTraceBlockByHash(t *testing.T) {
	ctx := context.Background()

	mockJSONRPC := &mocks.JSONRPC{}
	blkHsh := common.HexToHash("0xc08307ec6df58a995dcd2b5f83ddc6a0c08d437b4a97437e35d0f9854321ea35")
	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"debug_traceBlockByHash",
		blkHsh,
		mock.Anything,
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*json.RawMessage)

			file, err := ioutil.ReadFile(
				"testdata/block_trace_0xd88e8376ec3eef899d9fbc6349e8330ebfc102b245fef784a999ac854091cb64.json",
			)
			assert.NoError(t, err)

			*r = json.RawMessage(file)
		},
	).Twice()

	rpcClient := &RPCClient{
		JSONRPC: mockJSONRPC,
	}
	sdkClient := &SDKClient{
		RPCClient:      rpcClient,
		traceSemaphore: semaphore.NewWeighted(100),
		rosettaConfig: configuration.RosettaConfig{
			TraceType: configuration.OpenEthereumTrace,
		},
	}

	// Test TraceBlockByHash flow
	tx1Hsh := common.HexToHash("0x123456")
	tx1 := RPCTransaction{
		TxExtraInfo: TxExtraInfo{
			TxHash: &tx1Hsh,
		},
	}
	txHash := "0x992a9c832adda14e2a56b3c932fd38a8c738ac38cecaaaed3da59002a75fe98c"
	tx2Hsh := common.HexToHash(txHash)
	tx2 := RPCTransaction{
		TxExtraInfo: TxExtraInfo{
			TxHash: &tx2Hsh,
		},
	}
	txs := []RPCTransaction{tx1, tx2}
	m, err := sdkClient.TraceBlockByHash(ctx, blkHsh, txs)
	assert.NoError(t, err)
	assert.Equal(t, len(m), 2)
	assert.NotNil(t, m[txHash])
}

func TestOpenEthTraceAPI_1Txn(t *testing.T) {
	ctx := context.Background()

	mockJSONRPC := &mocks.JSONRPC{}
	blkHash := "0xc08307ec6df58a995dcd2b5f83ddc6a0c08d437b4a97437e35d0f9854321ea35"
	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"_replayBlockTransactions",
		blkHash,
		[]string{"trace"},
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*json.RawMessage)

			file, err := ioutil.ReadFile(
				"testdata/trace_block_1_tx.json",
			)
			assert.NoError(t, err)

			*r = json.RawMessage(file)
		},
	).Once()

	rpcClient := &RPCClient{
		JSONRPC: mockJSONRPC,
	}
	sdkClient := &SDKClient{
		RPCClient:      rpcClient,
		traceSemaphore: semaphore.NewWeighted(100),
		rosettaConfig: configuration.RosettaConfig{
			TraceType: configuration.OpenEthereumTrace,
		},
	}

	m, err := sdkClient.TraceReplayBlockTransactions(ctx, blkHash)
	assert.NoError(t, err)
	assert.Equal(t, len(m), 1)

	txHash := "0x992a9c832adda14e2a56b3c932fd38a8c738ac38cecaaaed3da59002a75fe98c"
	assert.NotNil(t, m[txHash])
	assert.Equal(t, "0x0043d60e87c5dd08C86C3123340705a1556C4719", m[txHash][0].From.Hex())
	assert.Equal(t, "0xeba709BF0aFDa796b3ad133CdF7DcCAA66eeEa33", m[txHash][0].To.Hex())
	assert.Equal(t, "99800000000000009", m[txHash][0].Value.String())
	assert.Equal(t, "572199", m[txHash][0].GasUsed.String())

	mockJSONRPC.AssertExpectations(t)
}

func TestOpenEthTraceAPI_MultiTxns(t *testing.T) {
	ctx := context.Background()

	mockJSONRPC := &mocks.JSONRPC{}
	txHash := "0xe66577512915b8fa57c1d390ccd5daec02aed38cd6c0c382d4b68995b2eff084"
	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"_replayBlockTransactions",
		txHash,
		[]string{"trace"},
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*json.RawMessage)

			file, err := ioutil.ReadFile(
				"testdata/trace_block_many_traces.json",
			)
			assert.NoError(t, err)

			*r = json.RawMessage(file)
		},
	).Once()

	rpcClient := &RPCClient{
		JSONRPC: mockJSONRPC,
	}
	sdkClient := &SDKClient{
		RPCClient: rpcClient,
	}

	m, err := sdkClient.TraceReplayBlockTransactions(ctx, txHash)
	assert.NoError(t, err)
	// block 10324614 has 1 tx, and 26 traces.
	assert.Equal(t, len(m), 1)
	assert.Equal(t, len(m[txHash]), 26)

	mockJSONRPC.AssertExpectations(t)
}

func TestBalance(t *testing.T) {
	ctx := context.Background()

	mockJSONRPC := &mocks.JSONRPC{}
	rpcClient := &RPCClient{
		JSONRPC: mockJSONRPC,
	}
	sdkClient := &SDKClient{
		RPCClient: rpcClient,
	}

	blockNum := fmt.Sprintf("0x%s", strconv.FormatInt(10992, 16))

	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"eth_getBlockByNumber",
		blockNum,
		false,
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

	account := "0x97158A00a4D227Ec7fe3234B52f21e5608FeE3d1"
	mockJSONRPC.On(
		"BatchCallContext",
		ctx,
		mock.MatchedBy(func(rpcs []rpc.BatchElem) bool {
			return len(rpcs) == 3 && rpcs[0].Method == "eth_getBalance" &&
				rpcs[1].Method == "eth_getTransactionCount" && rpcs[2].Method == "eth_getCode"
		}),
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).([]rpc.BatchElem)

			assert.Len(t, r, 3)
			for i := range r {
				assert.Len(t, r[i].Args, 2)
				assert.Equal(t, r[i].Args[0], account)
				assert.Equal(t, r[i].Args[1], blockNum)
			}

			balance := hexutil.MustDecodeBig("0x2324c0d180077fe7000")
			*(r[0].Result.(*hexutil.Big)) = (hexutil.Big)(*balance)
			*(r[1].Result.(*hexutil.Uint64)) = hexutil.Uint64(0)
			*(r[2].Result.(*string)) = "0x"
		},
	).Once()

	callData, err := hexutil.Decode(BalanceOfMethodPrefix + account[2:42])
	encodedERC20Data := hexutil.Encode(callData)
	tokenAddress := common.HexToAddress("0x1E77ad77925Ac0075CF61Fb76bA35D884985019d")
	assert.NoError(t, err)
	mockJSONRPC.On(
		"CallContext",
		ctx,
		mock.Anything,
		"eth_call",
		map[string]string{
			"data": encodedERC20Data,
			"to":   tokenAddress.String(),
		},
		blockNum,
	).Return(
		nil,
	).Run(
		func(args mock.Arguments) {
			r := args.Get(1).(*string)
			var expected map[string]interface{}
			file, err := ioutil.ReadFile("testdata/call_balance_token_10992.json")
			assert.NoError(t, err)

			err = json.Unmarshal(file, &expected)
			assert.NoError(t, err)

			*r = expected["data"].(string)
		},
	).Once()

	currencies := []*RosettaTypes.Currency{
		{
			Symbol:   "USDC",
			Decimals: 6,
			Metadata: map[string]interface{}{ContractAddressMetadata: "0x1E77ad77925Ac0075CF61Fb76bA35D884985019d"},
		},
	}

	resp, err := sdkClient.Balance(
		ctx,
		&RosettaTypes.AccountIdentifier{
			Address: account,
		},
		&RosettaTypes.PartialBlockIdentifier{
			Index: RosettaTypes.Int64(10992),
		},
		currencies,
	)

	assert.Equal(t, &RosettaTypes.AccountBalanceResponse{
		BlockIdentifier: &RosettaTypes.BlockIdentifier{
			Hash:  "0xba9ded5ca1ec9adb9451bf062c9de309d9552fa0f0254a7b982d3daf7ae436ae",
			Index: 10992,
		},
		Balances: []*RosettaTypes.Amount{
			{
				Value: "8098",
				Currency: &RosettaTypes.Currency{
					Symbol:   "USDC",
					Decimals: 6,
					Metadata: map[string]interface{}{
						"contractAddress": "0x1E77ad77925Ac0075CF61Fb76bA35D884985019d",
					},
				},
			},
		},
		Metadata: map[string]interface{}{
			"code":  "0x",
			"nonce": int64(0),
		},
	}, resp)
	assert.NoError(t, err)

	mockJSONRPC.AssertExpectations(t)
}

// TestBatchCallContext tests the BatchCallContext method for the SDKClient override.
func TestBatchCallContext(t *testing.T) {
	ctx := context.Background()

	// Construct the mock clients
	mockJSONRPC := &mocks.JSONRPC{}
	rpcClient := &RPCClient{
		JSONRPC: mockJSONRPC,
	}
	sdkClient := &SDKClient{
		RPCClient:    rpcClient,
		maxBatchSize: 1,
	}

	account := "0x97158A00a4D227Ec7fe3234B52f21e5608FeE3d1"
	blockNum := fmt.Sprintf("0x%s", strconv.FormatInt(10992, 16))

	elements := []rpc.BatchElem{
		{
			Method: "eth_getBalance",
			Args: []interface{}{
				account,
				blockNum,
			},
			Result: new(big.Int),
		},
		{
			Method: "eth_getBalance",
			Args: []interface{}{
				account,
				blockNum,
			},
			Result: new(big.Int),
		},
	}

	balanceA := hexutil.MustDecodeBig("0x2324c0d180077fe7000")
	balanceB := hexutil.MustDecodeBig("0x2324c0d180077fe7001")
	mockJSONRPC.On("BatchCallContext", ctx, []rpc.BatchElem{elements[0]}).Return(nil).Run(
		func(args mock.Arguments) {
			r := args.Get(1).([]rpc.BatchElem)
			*(r[0].Result.(*big.Int)) = *balanceA
		},
	)
	mockJSONRPC.On("BatchCallContext", ctx, []rpc.BatchElem{elements[1]}).Return(nil).Run(
		func(args mock.Arguments) {
			r := args.Get(1).([]rpc.BatchElem)
			*(r[0].Result.(*big.Int)) = *balanceB
		},
	)

	// Execute two getBalance calls in a batch
	err := sdkClient.BatchCallContext(
		ctx,
		elements,
	)
	assert.Equal(t, balanceA, elements[0].Result)
	assert.Equal(t, balanceB, elements[1].Result)
	assert.NoError(t, err)

	mockJSONRPC.AssertExpectations(t)
}

// TestBatchCallContextNoLimit tests the BatchCallContext method for the SDKClient override.
// Tests without setting a maxBatchSize.
func TestBatchCallContextNoLimit(t *testing.T) {
	ctx := context.Background()

	// Construct the mock clients
	mockJSONRPC := &mocks.JSONRPC{}
	rpcClient := &RPCClient{
		JSONRPC: mockJSONRPC,
	}
	sdkClient := &SDKClient{
		RPCClient: rpcClient,
	}

	account := "0x97158A00a4D227Ec7fe3234B52f21e5608FeE3d1"
	blockNum := fmt.Sprintf("0x%s", strconv.FormatInt(10992, 16))

	elements := []rpc.BatchElem{
		{
			Method: "eth_getBalance",
			Args: []interface{}{
				account,
				blockNum,
			},
			Result: new(big.Int),
		},
		{
			Method: "eth_getBalance",
			Args: []interface{}{
				account,
				blockNum,
			},
			Result: new(big.Int),
		},
	}

	balanceA := hexutil.MustDecodeBig("0x2324c0d180077fe7000")
	balanceB := hexutil.MustDecodeBig("0x2324c0d180077fe7001")
	mockJSONRPC.On("BatchCallContext", ctx, elements).Return(nil).Run(
		func(args mock.Arguments) {
			r := args.Get(1).([]rpc.BatchElem)
			*(r[0].Result.(*big.Int)) = *balanceA
			*(r[1].Result.(*big.Int)) = *balanceB
		},
	)

	// Execute two getBalance calls in a batch
	err := sdkClient.BatchCallContext(
		ctx,
		elements,
	)
	assert.Equal(t, balanceA, elements[0].Result)
	assert.Equal(t, balanceB, elements[1].Result)
	assert.NoError(t, err)

	mockJSONRPC.AssertExpectations(t)
}
