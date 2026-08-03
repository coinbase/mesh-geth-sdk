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
	"context"
	"encoding/json"
	"math/big"
	"testing"

	"github.com/coinbase/rosetta-geth-sdk/client"
	"github.com/coinbase/rosetta-geth-sdk/configuration"
	mockedServices "github.com/coinbase/rosetta-geth-sdk/mocks/services"
	AssetTypes "github.com/coinbase/rosetta-geth-sdk/types"
	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/assert"
)

var (
	parseSignedEthereumTransfer = `{"signed_tx":"eyJ0eXBlIjoiMHgwIiwibm9uY2UiOiIweDEiLCJnYXNQcmljZSI6IjB4NzczNTk0MDkiLCJtYXhQcmlvcml0eUZlZVBlckdhcyI6bnVsbCwibWF4RmVlUGVyR2FzIjpudWxsLCJnYXMiOiIweDUyMDgiLCJ2YWx1ZSI6IjB4NjQiLCJpbnB1dCI6IjB4IiwidiI6IjB4MmEiLCJyIjoiMHg3YTg2NzAzZGNlMWM0Y2E2NTc0MjZkYmI1OTg5MTEyZTAyODg5ZTk3NzZmMWY0NjFlYmVhYzI3MTVjN2IxOGU1IiwicyI6IjB4MzBkMzVkYzY3Zjk2YzAyOTY5M2U3NGM3OWI3ZWJlN2VmMTUxYzY5OTYwMjgwYTkxOWNkZWUwNzhmODZmZWFjZiIsInRvIjoiMHhkZjdjNGZmZjMxYTE5MGU4ZDQ2ZmM5YmE4Y2RlNmFhZDhmNjlmYzc2IiwiaGFzaCI6IjB4MGM1NzBjYjdjOGY3NmQzZmU0MDQwYTRkN2U0MTM0ZjNkMDM4NjA4ZGRiYjQ4ZDZlMzIyZWU4ZjAwODlmMmFhYSJ9","currency":{"symbol":"ETH","decimals":18}}` // nolint

	parseSignedERC20Transfer = `{"signed_tx":"eyJ0eXBlIjoiMHgwIiwibm9uY2UiOiIweDIiLCJnYXNQcmljZSI6IjB4NTk2ODJmMTciLCJtYXhQcmlvcml0eUZlZVBlckdhcyI6bnVsbCwibWF4RmVlUGVyR2FzIjpudWxsLCJnYXMiOiIweDU0NDQiLCJ2YWx1ZSI6IjB4MCIsImlucHV0IjoiMHhhOTA1OWNiYjAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMGRmN2M0ZmZmMzFhMTkwZThkNDZmYzliYThjZGU2YWFkOGY2OWZjNzYwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAxIiwidiI6IjB4MmEiLCJyIjoiMHhiMzNhY2QwMTQwMTgwMTEzYzgxNmViMzBiOTEzMjQ3ZTkxNjFjZDA0NTZhNmM2YjkxYzgxMmQ1Zjc2NWQ4MzdjIiwicyI6IjB4MTQ3NTlhYWZhOGMwMWU1OTFjZmQyM2ZkMzc4YjNiMDY0NDZjYmFiMDcwMmVmNTRmMDMzYzZlNjg1NmZhYWVkNSIsInRvIjoiMHgxZTc3YWQ3NzkyNWFjMDA3NWNmNjFmYjc2YmEzNWQ4ODQ5ODUwMTlkIiwiaGFzaCI6IjB4ZTM3M2QzYzNjNzVkMjQ4NzVkYjk3YjQyZTEzMzFmZjIyNTdlMTczOTQ4NTkzYzU4ZTVmOTI4YWMyYjQ3NzY2NyJ9","currency":{"symbol":"USDC","decimals":6,"metadata":{"contractAddress":"0x1E77ad77925Ac0075CF61Fb76bA35D884985019d"}}}` // nolint

	parseTransferValue          = uint64(100)
	parseTransferGasPrice       = uint64(2000000009)
	parseTransferNonce          = uint64(1)
	parseEthTransferGasTipCap   = uint64(2000000009)
	parseEthTransferGasFeeCap   = uint64(2000000009)
	parseEthTransferGasLimit    = uint64(21000)
	parseErc20TransferGasTipCap = uint64(1500000023)
	parseErc20TransferGasFeeCap = uint64(1500000023)
	parseErc20TransferGasLimit  = uint64(21572)
)

func TestParse(t *testing.T) {
	testingClient := newTestingClient()

	tests := map[string]struct {
		request          *types.ConstructionParseRequest
		expectedResponse *types.ConstructionParseResponse
		expectedError    *types.Error
	}{
		"happy path: signed Ethereum transfer": {
			request: &types.ConstructionParseRequest{
				NetworkIdentifier: ethereumNetworkIdentifier,
				Signed:            true,
				Transaction:       parseSignedEthereumTransfer,
			},
			expectedResponse: &types.ConstructionParseResponse{
				Operations: templateOperations(parseTransferValue, ethereumCurrencyConfig, "CALL"),
				AccountIdentifierSigners: []*types.AccountIdentifier{
					{
						Address: testingFromAddress,
					},
				},
				Metadata: map[string]interface{}{
					"nonce":       float64(parseTransferNonce),
					"gas_price":   float64(parseTransferGasPrice),
					"chain_id":    float64(ethRopstenChainID),
					"gas_tip_cap": float64(parseEthTransferGasTipCap),
					"gas_fee_cap": float64(parseEthTransferGasFeeCap),
					"gas_limit":   float64(parseEthTransferGasLimit),
				},
			},
		},
		"happy path: signed ERC20 transfer tx": {
			request: &types.ConstructionParseRequest{
				NetworkIdentifier: ethereumNetworkIdentifier,
				Signed:            true,
				Transaction:       parseSignedERC20Transfer,
			},
			expectedResponse: &types.ConstructionParseResponse{
				Operations: templateOperations(payloadsTransferValue, &types.Currency{
					Symbol:   "USDC",
					Decimals: 6,
					Metadata: map[string]interface{}{
						"contractAddress": "0x1E77ad77925Ac0075CF61Fb76bA35D884985019d",
					},
				}, "ERC20_TRANSFER"),
				AccountIdentifierSigners: []*types.AccountIdentifier{
					{
						Address: testingFromAddress,
					},
				},
				Metadata: map[string]interface{}{
					"nonce":       float64(2),
					"gas_price":   float64(1500000023),
					"chain_id":    float64(ethRopstenChainID),
					"gas_tip_cap": float64(parseErc20TransferGasTipCap),
					"gas_fee_cap": float64(parseErc20TransferGasFeeCap),
					"gas_limit":   float64(parseErc20TransferGasLimit),
				},
			},
		},
		"error: empty transaction": {
			request: &types.ConstructionParseRequest{
				NetworkIdentifier: ethereumNetworkIdentifier,
				Signed:            true,
				Transaction:       "",
			},
			expectedError: templateError(
				AssetTypes.ErrUnableToParseIntermediateResult, "unexpected end of JSON input"),
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			resp, err := testingClient.servicer.ConstructionParse(
				context.Background(),
				test.request,
			)

			if err != nil {
				assert.Equal(t, test.expectedError, err)
			} else {
				assert.Equal(t, test.expectedResponse, resp)
			}
		})
	}
}

// TestParseContractCallMethodDecoding verifies that /construction/parse decodes
// method_signature and method_args from the transaction calldata for supported
// contract methods, and emits neither field for unsupported methods.
func TestParseContractCallMethodDecoding(t *testing.T) {
	delegatee := testingFromAddress
	calldata, err := ConstructContractCallDataGeneric("delegate(address)", []string{delegatee})
	assert.NoError(t, err)

	newServicer := func(supported []string) *APIService {
		cfg := &configuration.Configuration{
			Mode:    configuration.ModeOnline,
			Network: ethereumNetworkIdentifier,
			RosettaCfg: configuration.RosettaConfig{
				Currency:                 ethereumCurrencyConfig,
				SupportedContractMethods: supported,
			},
			ChainConfig: &params.ChainConfig{ChainID: big.NewInt(int64(ethRopstenChainID))},
		}
		return NewAPIService(cfg, AssetTypes.LoadTypes(), AssetTypes.Errors, &mockedServices.Client{})
	}

	tx := &client.Transaction{
		From:     testingFromAddress,
		To:       "0x1E77ad77925Ac0075CF61Fb76bA35D884985019d",
		Value:    big.NewInt(0),
		Data:     calldata,
		Nonce:    1,
		GasPrice: big.NewInt(2000000009),
		GasLimit: 21000,
		ChainID:  big.NewInt(int64(ethRopstenChainID)),
		Currency: ethereumCurrencyConfig,
	}
	txJSON, err := json.Marshal(tx)
	assert.NoError(t, err)

	request := &types.ConstructionParseRequest{
		NetworkIdentifier: ethereumNetworkIdentifier,
		Signed:            false,
		Transaction:       string(txJSON),
	}

	t.Run("supported method is decoded from calldata", func(t *testing.T) {
		resp, parseErr := newServicer([]string{"delegate(address)"}).ConstructionParse(
			context.Background(),
			request,
		)
		assert.Nil(t, parseErr)
		assert.Equal(t, "delegate(address)", resp.Metadata["method_signature"])
		// []string is marshaled through JSON into []interface{} in the metadata map.
		assert.Equal(t, []interface{}{delegatee}, resp.Metadata["method_args"])
	})

	t.Run("unsupported method emits no method fields", func(t *testing.T) {
		resp, parseErr := newServicer(nil).ConstructionParse(
			context.Background(),
			request,
		)
		assert.Nil(t, parseErr)
		_, hasSig := resp.Metadata["method_signature"]
		_, hasArgs := resp.Metadata["method_args"]
		assert.False(t, hasSig)
		assert.False(t, hasArgs)
	})
}
