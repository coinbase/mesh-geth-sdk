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
	"fmt"
	"math/big"
	"testing"

	AssetTypes "github.com/coinbase/rosetta-geth-sdk/types"
	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/stretchr/testify/assert"
)

var (
	// preprocessNetworkIdentifier = &types.NetworkIdentifier{
	// 	Network:    "Ropsten",
	// 	Blockchain: "Ethereum",
	// }

	preprocessTransferValue = uint64(1)
	methodSignature         = "approve(address,uint256)"
	methodArgs              = []string{"0xD10a72Cf054650931365Cc44D912a4FD75257058", "1000"}
	expectedMethodArgs      = []interface{}{
		"0xD10a72Cf054650931365Cc44D912a4FD75257058",
		"1000",
	}
	preprocessZeroTransferValue = uint64(0)
	// preprocessTransferValueHex     = hexutil.EncodeUint64(preprocessTransferValue)
	// preprocessGasPrice             = uint64(100000000000)
	// preprocessGasPriceHex          = hexutil.EncodeUint64(preprocessGasPrice)
	preprocessGenericData = "0x095ea7b3000000000000000000000000d10a72cf054650931365cc44d912a4fd7525705" +
		"800000000000000000000000000000000000000000000000000000000000003e8"
	preprocessRedeemData     = "0xeda1122c2b65269ff5a2a05ba8d589d6fb068d095c50d296c0abb17bd3e98430d8d89a36"
	redeemMethodSignature    = "redeem(bytes32)"
	redeemMethodArgs         = []string{"0x2b65269ff5a2a05ba8d589d6fb068d095c50d296c0abb17bd3e98430d8d89a36"}
	expectedRedeemMethodArgs = []interface{}{"0x2b65269ff5a2a05ba8d589d6fb068d095c50d296c0abb17bd3e98430d8d89a36"}
	// bridge withdraw params
	bridgeWithdrawMethodSig  = "withdraw(address,uint256,uint32,bytes)"
	bridgeWithdrawMethodArgs = []string{
		"0xDeadDeAddeAddEAddeadDEaDDEAdDeaDDeAD0000",
		"23535",
		"0",
		"0x",
	}
	bridgeWithdrawRedeemData         = "0x32b7006d000000000000000000000000deaddeaddeaddeaddeaddeaddeaddeaddead00000000000000000000000000000000000000000000000000000000000000005bef000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000000"
	expectedBridgeWithdrawMethodArgs = []interface{}{
		"0xDeadDeAddeAddEAddeadDEaDDEAdDeaDDeAD0000",
		"23535",
		"0",
		"0x",
	}
)

func TestConstructionPreprocess(t *testing.T) {
	testingClient := newTestingClient()

	preprocessNoZeroTransferValue, _ := big.NewInt(0).SetString("-23946292673190280600", 10)

	tests := map[string]struct {
		operations []*types.Operation
		metadata   map[string]interface{}

		expectedResponse *types.ConstructionPreprocessResponse
		expectedError    *types.Error
	}{
		"happy path: Generic Contract call": {
			operations: templateOperations(preprocessTransferValue, ethereumCurrencyConfig, "CALL"),
			metadata: map[string]interface{}{
				"method_signature": methodSignature,
				"method_args":      methodArgs,
			},
			expectedResponse: &types.ConstructionPreprocessResponse{
				Options: map[string]interface{}{
					"from":             testingFromAddress,
					"to":               testingToAddress, // it will be contract address user need to pass in operation
					"value":            fmt.Sprint(preprocessTransferValue),
					"contract_address": testingToAddress,
					"data":             preprocessGenericData,
					"method_signature": methodSignature,
					"method_args":      expectedMethodArgs,
					"currency": map[string]interface{}{
						"decimals": float64(18),
						"symbol":   "ETH",
					},
				},
			},
		},
		"happy path: Approve call with zero transfer value": {
			operations: templateOperations(preprocessZeroTransferValue, ethereumCurrencyConfig, "CALL"),
			metadata: map[string]interface{}{
				"method_signature": "approve(address,uint256)",
				"method_args":      []string{"0xD10a72Cf054650931365Cc44D912a4FD75257058", "1000"},
			},
			expectedResponse: &types.ConstructionPreprocessResponse{
				Options: map[string]interface{}{
					"from":             testingFromAddress,
					"to":               testingToAddress, // it will be contract address user need to pass in operation
					"value":            fmt.Sprint(preprocessZeroTransferValue),
					"contract_address": testingToAddress,
					"data":             preprocessGenericData,
					"method_signature": methodSignature,
					"method_args":      expectedMethodArgs,
					"currency": map[string]interface{}{
						"decimals": float64(18),
						"symbol":   "ETH",
					},
				},
			},
		},
		"happy path: Redeem call with zero transfer value": {
			operations: templateOperations(preprocessZeroTransferValue, ethereumCurrencyConfig, "CALL"),
			metadata: map[string]interface{}{
				"method_signature": redeemMethodSignature,
				"method_args":      redeemMethodArgs,
			},
			expectedResponse: &types.ConstructionPreprocessResponse{
				Options: map[string]interface{}{
					"from":             testingFromAddress,
					"to":               testingToAddress, // it will be contract address user need to pass in operation
					"value":            fmt.Sprint(preprocessZeroTransferValue),
					"contract_address": testingToAddress,
					"data":             preprocessRedeemData,
					"method_signature": redeemMethodSignature,
					"method_args":      expectedRedeemMethodArgs,
					"currency": map[string]interface{}{
						"decimals": float64(18),
						"symbol":   "ETH",
					},
				},
			},
		},
		"happy path: Bridge Withdraw": {
			operations: templateOperations(preprocessZeroTransferValue, ethereumCurrencyConfig, "CALL"),
			metadata: map[string]interface{}{
				"method_signature": bridgeWithdrawMethodSig,
				"method_args":      bridgeWithdrawMethodArgs,
			},
			expectedResponse: &types.ConstructionPreprocessResponse{
				Options: map[string]interface{}{
					"from":             testingFromAddress,
					"to":               testingToAddress, // it will be contract address user need to pass in operation
					"value":            fmt.Sprint(preprocessZeroTransferValue),
					"contract_address": testingToAddress,
					"data":             bridgeWithdrawRedeemData,
					"method_signature": bridgeWithdrawMethodSig,
					"method_args":      expectedBridgeWithdrawMethodArgs,
					"currency": map[string]interface{}{
						"decimals": float64(18),
						"symbol":   "ETH",
					},
				},
			},
		},
		"happy path: Outbound transfer call with zero transfer value": {
			operations: templateOperations(preprocessZeroTransferValue, ethereumCurrencyConfig, "CALL"),
			metadata: map[string]interface{}{
				"method_signature": "outboundTransfer(address,address,uint256,bytes)",
				"method_args": []string{
					"0x07865c6E87B9F70255377e024ace6630C1Eaa37F", // L1 token
					"0xb53c4cda2de7becd6ad0fe3f0ded29fc6b0aa6f6", // To address
					"1000000", // Amount
					"0x",      // Data
				},
			},
			expectedResponse: &types.ConstructionPreprocessResponse{
				Options: map[string]interface{}{
					"from":             testingFromAddress,
					"to":               testingToAddress, // it will be contract address user need to pass in operation
					"value":            fmt.Sprint(preprocessZeroTransferValue),
					"contract_address": testingToAddress,
					"data":             "0x7b3a3c8b00000000000000000000000007865c6e87b9f70255377e024ace6630c1eaa37f000000000000000000000000b53c4cda2de7becd6ad0fe3f0ded29fc6b0aa6f600000000000000000000000000000000000000000000000000000000000f424000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000000", // nolint:lll
					"method_signature": "outboundTransfer(address,address,uint256,bytes)",
					"method_args": []interface{}{
						"0x07865c6E87B9F70255377e024ace6630C1Eaa37F",
						"0xb53c4cda2de7becd6ad0fe3f0ded29fc6b0aa6f6",
						"1000000",
						"0x",
					},
					"currency": map[string]interface{}{
						"decimals": float64(18),
						"symbol":   "ETH",
					},
				},
			},
		},
		"happy path: native currency": {
			operations: templateOperations(preprocessTransferValue, ethereumCurrencyConfig, "CALL"),
			expectedResponse: &types.ConstructionPreprocessResponse{
				Options: map[string]interface{}{
					"from":  testingFromAddress,
					"to":    testingToAddress,
					"value": fmt.Sprint(preprocessTransferValue),
					"currency": map[string]interface{}{
						"decimals": float64(18),
						"symbol":   "ETH",
					},
				},
			},
		},
		"happy path: ERC20 currency": {
			operations: templateOperations(preprocessTransferValue, &types.Currency{
				Symbol:   "USDC",
				Decimals: 6,
				Metadata: map[string]interface{}{
					"contractAddress": "0x1E77ad77925Ac0075CF61Fb76bA35D884985019d",
				},
			}, "ERC20_TRANSFER"),
			expectedResponse: &types.ConstructionPreprocessResponse{
				Options: map[string]interface{}{
					"from":  testingFromAddress,
					"to":    testingToAddress,
					"value": fmt.Sprint(preprocessTransferValue),
					"currency": map[string]interface{}{
						"decimals": float64(6),
						"symbol":   "USDC",
						"metadata": map[string]interface{}{
							"contractAddress": "0x1E77ad77925Ac0075CF61Fb76bA35D884985019d",
						},
					},
				},
			},
		},
		"error: both positive amount": {
			operations: func() []*types.Operation {
				operations := templateOperations(
					preprocessTransferValue,
					ethereumCurrencyConfig,
					"CALL",
				)
				operations[0].Amount.Value = "1"
				return operations
			}(),
			expectedResponse: nil,
			expectedError: templateError(
				AssetTypes.ErrInvalidInput,
				"at index 1: unable to find match for operation",
			),
		},
		"error: missing currency": {
			operations: func() []*types.Operation {
				operations := templateOperations(
					preprocessTransferValue,
					ethereumCurrencyConfig,
					"CALL",
				)
				operations[0].Amount.Currency = nil
				return operations
			}(),
			expectedResponse: nil,
			expectedError: templateError(
				AssetTypes.ErrInvalidInput, "invalid currency on operation"),
		},
		"error: unequal currency": {
			operations: func() []*types.Operation {
				operations := templateOperations(
					preprocessTransferValue,
					ethereumCurrencyConfig,
					"CALL",
				)
				operations[0].Amount.Currency = &types.Currency{
					Symbol:   "USDC",
					Decimals: 18,
				}
				return operations
			}(),
			expectedResponse: nil,
			expectedError: templateError(
				AssetTypes.ErrInvalidInput, "currency info doesn't match between the operations"),
		},
		"error: invalid from address": {
			operations: func() []*types.Operation {
				operations := templateOperations(
					preprocessTransferValue,
					ethereumCurrencyConfig,
					"CALL",
				)
				operations[0].Account.Address = "invalid"
				return operations
			}(),
			expectedResponse: nil,
			expectedError: templateError(
				AssetTypes.ErrInvalidAddress, "invalid is not a valid address: invalid address"),
		},
		"error: invalid destination address": {
			operations: func() []*types.Operation {
				operations := templateOperations(
					preprocessTransferValue,
					ethereumCurrencyConfig,
					"CALL",
				)
				operations[1].Account.Address = "invalid"
				return operations
			}(),
			expectedResponse: nil,
			expectedError: templateError(
				AssetTypes.ErrInvalidAddress, "invalid is not a valid address: invalid address"),
		},
		"error: missing token address": {
			operations: templateOperations(preprocessTransferValue, &types.Currency{
				Symbol:   "USDC",
				Decimals: 6,
			}, "ERC20_TRANSFER"),
			expectedResponse: nil,
			expectedError: templateError(
				AssetTypes.ErrInvalidInput,
				"non-native currency must have contractAddress in Metadata",
			),
		},
		"error: token address not a string": {
			operations: templateOperations(preprocessTransferValue, &types.Currency{
				Symbol:   "USDC",
				Decimals: 6,
				Metadata: map[string]interface{}{
					"contractAddress": map[string]string{},
				},
			}, "ERC20_TRANSFER"),
			expectedResponse: nil,
			expectedError: templateError(
				AssetTypes.ErrInvalidInput,
				"non-native currency must have contractAddress in Metadata",
			),
		},
		"error: reject call with non-zero transfer value": {
			operations: bigAmountTemplateOperations(preprocessNoZeroTransferValue, ethereumCurrencyConfig, "CALL"),
			metadata: map[string]interface{}{
				"method_signature": "approve(address,uint256)",
				"method_args":      []string{"0xD10a72Cf054650931365Cc44D912a4FD75257058", "1000"},
			},
			expectedResponse: &types.ConstructionPreprocessResponse{
				Options: map[string]interface{}{
					"from":             testingFromAddress,
					"to":               testingToAddress, // it will be contract address user need to pass in operation
					"value":            fmt.Sprint(preprocessNoZeroTransferValue),
					"contract_address": testingToAddress,
					"data":             preprocessGenericData,
					"method_signature": methodSignature,
					"method_args":      expectedMethodArgs,
					"currency": map[string]interface{}{
						"decimals": float64(18),
						"symbol":   "ETH",
					},
				},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			request := &types.ConstructionPreprocessRequest{
				NetworkIdentifier: ethereumNetworkIdentifier,
				Operations:        test.operations,
				Metadata:          test.metadata,
			}
			resp, err := testingClient.servicer.ConstructionPreprocess(
				context.Background(),
				request,
			)

			if err != nil {
				assert.Equal(t, test.expectedError, err)
			} else {
				assert.Equal(t, test.expectedResponse, resp)
			}
		})
	}
}
