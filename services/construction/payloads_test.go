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
	"testing"

	AssetTypes "github.com/coinbase/rosetta-geth-sdk/types"
	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/stretchr/testify/assert"
)

var (
	payloadsTransferValue    = uint64(1)
	payloadsTransferGasPrice = uint64(5000000000)
	payloadsTransferGasLimit = uint64(21000)
	payloadsTransferNonce    = uint64(67)
	payloadsGenericData      = "0x095ea7b3000000000000000000000000d10a72cf054650931365cc" +
		"44d912a4fd7525705800000000000000000000000000000000000000000000000000000000000003e8"

	payloadsUnsignedRaw = `{"from":"0x97158A00a4D227Ec7fe3234B52f21e5608FeE3d1","to":"0xdF7C4fFf31A190E8D46FC9Ba8CdE6aaD8F69Fc76","value":1,"data":"","nonce":67,"gas_price":5000000000,"gas":21000,"chain_id":3,"currency":{"symbol":"ETH","decimals":18}}` //nolint

	payloadsRaw = `[{"address":"0x97158A00a4D227Ec7fe3234B52f21e5608FeE3d1","hex_bytes":"809c6fed4cd9352aebdbb7b67fad5a60d1f69fb425869c9e1a35586d1a97bb4e","account_identifier":{"address":"0x97158A00a4D227Ec7fe3234B52f21e5608FeE3d1"},"signature_type":"ecdsa_recovery"}]` // nolint

	payloads []*types.SigningPayload

	payloadsUnsignedRawContract = `{"from":"0x97158A00a4D227Ec7fe3234B52f21e5608FeE3d1","to":"0xdF7C4fFf31A190E8D46FC9Ba8CdE6aaD8F69Fc76","value":1,"data":"CV6nswAAAAAAAAAAAAAAANEKcs8FRlCTE2XMRNkSpP11JXBYAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA+g=","nonce":67,"gas_price":5000000000,"gas":21000,"chain_id":3,"currency":{"symbol":"ETH","decimals":18}}` // nolint

	payloadsRawContract = `[{"address":"0x97158A00a4D227Ec7fe3234B52f21e5608FeE3d1","hex_bytes":"df6f14704460f6e722fc4084138e468f90e4c4c760713b554b1f13a2e11ee1b7","account_identifier":{"address":"0x97158A00a4D227Ec7fe3234B52f21e5608FeE3d1"},"signature_type":"ecdsa_recovery"}]` // nolint

	payloadsContract []*types.SigningPayload

	payloadsUnsignedRawERC20 = `{"from":"0x97158A00a4D227Ec7fe3234B52f21e5608FeE3d1","to":"0x1E77ad77925Ac0075CF61Fb76bA35D884985019d","value":0,"data":"qQWcuwAAAAAAAAAAAAAAAN98T/8xoZDo1G/Juozeaq2Pafx2AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAE=","nonce":67,"gas_price":5000000000,"gas":21000,"chain_id":3,"currency":{"symbol":"USDC","decimals":6,"metadata":{"contractAddress":"0x1E77ad77925Ac0075CF61Fb76bA35D884985019d"}}}` // nolint

	payloadsRawERC20 = `[{"address":"0x97158A00a4D227Ec7fe3234B52f21e5608FeE3d1","hex_bytes":"607bd01f8ce114dad8e20b5268be15b345b083bb0de5fed5c8dfb213f14541d5","account_identifier":{"address":"0x97158A00a4D227Ec7fe3234B52f21e5608FeE3d1"},"signature_type":"ecdsa_recovery"}]` // nolint

	payloadsERC20 []*types.SigningPayload
)

func TestPayloads(t *testing.T) {
	testingClient := newTestingClient()

	assert.NoError(t, json.Unmarshal([]byte(payloadsRaw), &payloads))
	assert.NoError(t, json.Unmarshal([]byte(payloadsRawERC20), &payloadsERC20))
	assert.NoError(t, json.Unmarshal([]byte(payloadsRawContract), &payloadsContract))

	tests := map[string]struct {
		request          *types.ConstructionPayloadsRequest
		expectedResponse *types.ConstructionPayloadsResponse
		expectedError    *types.Error
	}{
		"happy path: Generic contract call": {
			request: &types.ConstructionPayloadsRequest{
				NetworkIdentifier: ethereumNetworkIdentifier,
				Operations: templateOperations(
					payloadsTransferValue,
					ethereumCurrencyConfig,
					"CALL",
				),
				Metadata: map[string]interface{}{
					"nonce":            float64(payloadsTransferNonce),
					"gas_price":        float64(payloadsTransferGasPrice),
					"gas_limit":        float64(payloadsTransferGasLimit),
					"data":             payloadsGenericData,
					"method_signature": "approve(address,uint256)",
					"method_args": []interface{}{
						"0xD10a72Cf054650931365Cc44D912a4FD75257058",
						"1000",
					},
				},
			},
			expectedResponse: &types.ConstructionPayloadsResponse{
				UnsignedTransaction: payloadsUnsignedRawContract,
				Payloads:            payloadsContract,
			},
		},
		"happy path: native currency": {
			request: &types.ConstructionPayloadsRequest{
				NetworkIdentifier: ethereumNetworkIdentifier,
				Operations: templateOperations(
					payloadsTransferValue,
					ethereumCurrencyConfig,
					"CALL",
				),
				Metadata: map[string]interface{}{
					"nonce":     float64(payloadsTransferNonce),
					"gas_price": float64(payloadsTransferGasPrice),
					"gas_limit": float64(payloadsTransferGasLimit),
				},
			},
			expectedResponse: &types.ConstructionPayloadsResponse{
				UnsignedTransaction: payloadsUnsignedRaw,
				Payloads:            payloads,
			},
		},
		"happy path: ERC20 currency": {
			request: &types.ConstructionPayloadsRequest{
				NetworkIdentifier: ethereumNetworkIdentifier,
				Operations: templateOperations(payloadsTransferValue, &types.Currency{
					Symbol:   "USDC",
					Decimals: 6,
					Metadata: map[string]interface{}{
						"contractAddress": "0x1E77ad77925Ac0075CF61Fb76bA35D884985019d",
					},
				}, "ERC20_TRANSFER"),
				Metadata: map[string]interface{}{
					"nonce":     float64(payloadsTransferNonce),
					"gas_price": float64(payloadsTransferGasPrice),
					"gas_limit": float64(payloadsTransferGasLimit),
				},
			},
			expectedResponse: &types.ConstructionPayloadsResponse{
				UnsignedTransaction: payloadsUnsignedRawERC20,
				Payloads:            payloadsERC20,
			},
		},
		"error: ErrInvalidInput: currency info doesn't match between the operations": {
			request: &types.ConstructionPayloadsRequest{
				NetworkIdentifier: ethereumNetworkIdentifier,
				Operations: func() []*types.Operation {
					operations := templateOperations(
						payloadsTransferValue,
						ethereumCurrencyConfig,
						"CALL",
					)
					operations[1].Amount.Currency = &types.Currency{
						Symbol:   "BTC",
						Decimals: 18,
					}
					return operations
				}(),
				Metadata: map[string]interface{}{
					"nonce":     float64(payloadsTransferNonce),
					"gas_price": float64(payloadsTransferGasPrice),
					"gas_limit": float64(payloadsTransferGasLimit),
				},
			},
			expectedError: templateError(
				AssetTypes.ErrInvalidInput,
				"currency info doesn't match between the operations",
			),
		},
		"error: ErrInvalidInput: non-native currency must have contractAddress in Metadata": {
			request: &types.ConstructionPayloadsRequest{
				NetworkIdentifier: ethereumNetworkIdentifier,
				Operations: templateOperations(payloadsTransferValue, &types.Currency{
					Symbol:   "USDC",
					Decimals: 6,
					Metadata: map[string]interface{}{},
				}, "ERC20_TRANSFER"),
				Metadata: map[string]interface{}{
					"nonce":     float64(payloadsTransferNonce),
					"gas_price": float64(payloadsTransferGasPrice),
					"gas_limit": float64(payloadsTransferGasLimit),
				},
			},
			expectedError: templateError(
				AssetTypes.ErrInvalidInput,
				"non-native currency must have contractAddress in Metadata",
			),
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			resp, err := testingClient.servicer.ConstructionPayloads(
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
