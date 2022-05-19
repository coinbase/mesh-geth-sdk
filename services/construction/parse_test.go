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
	"testing"

	AssetTypes "github.com/coinbase/rosetta-geth-sdk/types"
	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/stretchr/testify/assert"
)

var (
	parseSignedEthereumTransfer = `{"signed_tx":"eyJ0eXBlIjoiMHgwIiwibm9uY2UiOiIweDEiLCJnYXNQcmljZSI6IjB4NzczNTk0MDkiLCJtYXhQcmlvcml0eUZlZVBlckdhcyI6bnVsbCwibWF4RmVlUGVyR2FzIjpudWxsLCJnYXMiOiIweDUyMDgiLCJ2YWx1ZSI6IjB4NjQiLCJpbnB1dCI6IjB4IiwidiI6IjB4MmEiLCJyIjoiMHg3YTg2NzAzZGNlMWM0Y2E2NTc0MjZkYmI1OTg5MTEyZTAyODg5ZTk3NzZmMWY0NjFlYmVhYzI3MTVjN2IxOGU1IiwicyI6IjB4MzBkMzVkYzY3Zjk2YzAyOTY5M2U3NGM3OWI3ZWJlN2VmMTUxYzY5OTYwMjgwYTkxOWNkZWUwNzhmODZmZWFjZiIsInRvIjoiMHhkZjdjNGZmZjMxYTE5MGU4ZDQ2ZmM5YmE4Y2RlNmFhZDhmNjlmYzc2IiwiaGFzaCI6IjB4MGM1NzBjYjdjOGY3NmQzZmU0MDQwYTRkN2U0MTM0ZjNkMDM4NjA4ZGRiYjQ4ZDZlMzIyZWU4ZjAwODlmMmFhYSJ9","currency":{"symbol":"ETH","decimals":18}}` // nolint

	parseSignedERC20Transfer = `{"signed_tx":"eyJ0eXBlIjoiMHgwIiwibm9uY2UiOiIweDIiLCJnYXNQcmljZSI6IjB4NTk2ODJmMTciLCJtYXhQcmlvcml0eUZlZVBlckdhcyI6bnVsbCwibWF4RmVlUGVyR2FzIjpudWxsLCJnYXMiOiIweDU0NDQiLCJ2YWx1ZSI6IjB4MCIsImlucHV0IjoiMHhhOTA1OWNiYjAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMGRmN2M0ZmZmMzFhMTkwZThkNDZmYzliYThjZGU2YWFkOGY2OWZjNzYwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAxIiwidiI6IjB4MmEiLCJyIjoiMHhiMzNhY2QwMTQwMTgwMTEzYzgxNmViMzBiOTEzMjQ3ZTkxNjFjZDA0NTZhNmM2YjkxYzgxMmQ1Zjc2NWQ4MzdjIiwicyI6IjB4MTQ3NTlhYWZhOGMwMWU1OTFjZmQyM2ZkMzc4YjNiMDY0NDZjYmFiMDcwMmVmNTRmMDMzYzZlNjg1NmZhYWVkNSIsInRvIjoiMHgxZTc3YWQ3NzkyNWFjMDA3NWNmNjFmYjc2YmEzNWQ4ODQ5ODUwMTlkIiwiaGFzaCI6IjB4ZTM3M2QzYzNjNzVkMjQ4NzVkYjk3YjQyZTEzMzFmZjIyNTdlMTczOTQ4NTkzYzU4ZTVmOTI4YWMyYjQ3NzY2NyJ9","currency":{"symbol":"USDC","decimals":6,"metadata":{"contractAddress":"0x1E77ad77925Ac0075CF61Fb76bA35D884985019d"}}}` // nolint

	parseTransferValue    = uint64(100)
	parseTransferGasPrice = uint64(2000000009)
	parseTransferNonce    = uint64(1)
)

func TestParse(t *testing.T) {
	testingClient := newTestingClient()
	// if err != nil {
	//	log.Fatalln("%w: unable to load testing configuration", err)
	// }

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
					"nonce":     float64(parseTransferNonce),
					"gas_price": float64(parseTransferGasPrice),
					"chain_id":  float64(ethRopstenChainID),
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
					"nonce":     float64(2),
					"gas_price": float64(1500000023),
					"chain_id":  float64(ethRopstenChainID),
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
