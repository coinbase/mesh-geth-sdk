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
	AssetTypes "github.com/coinbase/rosetta-geth-sdk/types"
	"testing"

	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/stretchr/testify/assert"
)

var (
	combineUnsignedRaw = `{"from":"0x97158A00a4D227Ec7fe3234B52f21e5608FeE3d1","to":"0xdF7C4fFf31A190E8D46FC9Ba8CdE6aaD8F69Fc76","value":100,"data":"","nonce":0,"gas_price":2000000009,"gas":21000,"chain_id":3,"currency":{"symbol":"ETH","decimals":18}}` // nolint

	combineSignedRaw = `{"signed_tx":"eyJ0eXBlIjoiMHgwIiwibm9uY2UiOiIweDAiLCJnYXNQcmljZSI6IjB4NzczNTk0MDkiLCJtYXhQcmlvcml0eUZlZVBlckdhcyI6bnVsbCwibWF4RmVlUGVyR2FzIjpudWxsLCJnYXMiOiIweDUyMDgiLCJ2YWx1ZSI6IjB4NjQiLCJpbnB1dCI6IjB4IiwidiI6IjB4MmEiLCJyIjoiMHg3YTg2NzAzZGNlMWM0Y2E2NTc0MjZkYmI1OTg5MTEyZTAyODg5ZTk3NzZmMWY0NjFlYmVhYzI3MTVjN2IxOGU1IiwicyI6IjB4MzBkMzVkYzY3Zjk2YzAyOTY5M2U3NGM3OWI3ZWJlN2VmMTUxYzY5OTYwMjgwYTkxOWNkZWUwNzhmODZmZWFjZiIsInRvIjoiMHhkZjdjNGZmZjMxYTE5MGU4ZDQ2ZmM5YmE4Y2RlNmFhZDhmNjlmYzc2IiwiaGFzaCI6IjB4OTlhYjZiYThhNDliZWRhYzkyZTRlOGE0OGU0OGUxNzY1ZmJkMGQ5ZThjODNlNzE2MTFmNDE5NzhmMzg5ZTgwYSJ9","currency":{"symbol":"ETH","decimals":18}}` // nolint

	combineSignaturesRaw = `[{"hex_bytes":"7a86703dce1c4ca657426dbb5989112e02889e9776f1f461ebeac2715c7b18e530d35dc67f96c029693e74c79b7ebe7ef151c69960280a919cdee078f86feacf01","signing_payload":{"address":"0x97158A00a4D227Ec7fe3234B52f21e5608FeE3d1","hex_bytes":"358b2c8be6153484861dac2b3668d6067759c4c427350432a595f7ffe31bfd26","signature_type":"ecdsa_recovery"},"public_key":{"hex_bytes":"28eb23ef37ff86c8ab7cebaf0a46a792bcfeac32905fd859420b24d0c18e6637c7671c7d0dce2be04fd7c71039851776207410c87baadba7ea7130646c8faab4","curve_type":"secp256k1"},"signature_type":"ecdsa_recovery"}]` // nolint
)

func TestConstructionCombine(t *testing.T) {
	testingClient := newTestingClient()

	var signatures []*types.Signature
	_ = json.Unmarshal([]byte(combineSignaturesRaw), &signatures)

	tests := map[string]struct {
		request          *types.ConstructionCombineRequest
		expectedResponse *types.ConstructionCombineResponse
		expectedError    *types.Error
	}{
		"happy path": {
			request: templateConstructCombineRequest(combineUnsignedRaw, signatures),
			expectedResponse: &types.ConstructionCombineResponse{
				SignedTransaction: combineSignedRaw,
			},
		},
		"error: missing transaction": {
			request: &types.ConstructionCombineRequest{},
			expectedError: templateError(
				AssetTypes.ErrInvalidInput, "transaction data is not provided"),
		},
		"error: missing signature": {
			request: templateConstructCombineRequest(combineUnsignedRaw, nil),
			expectedError: templateError(
				AssetTypes.ErrInvalidInput, "signature is not provided"),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			resp, err := testingClient.servicer.ConstructionCombine(
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

func templateConstructCombineRequest(
	unsignedTx string,
	signatures []*types.Signature,
) *types.ConstructionCombineRequest {
	return &types.ConstructionCombineRequest{
		NetworkIdentifier:   ethereumNetworkIdentifier,
		UnsignedTransaction: unsignedTx,
		Signatures:          signatures,
	}
}
