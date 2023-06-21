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
	"encoding/hex"
	"testing"

	AssetTypes "github.com/coinbase/rosetta-geth-sdk/types"

	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/stretchr/testify/assert"
)

func TestConstructionDerive(t *testing.T) {
	testingClient := newTestingClient()

	tests := map[string]struct {
		request          *types.ConstructionDeriveRequest
		expectedResponse *types.ConstructionDeriveResponse
		expectedError    *types.Error
	}{
		"happy path": {
			request: templateDeriveRequest(
				"03d3d3358e7f69cbe45bde38d7d6f24660c7eeeaee5c5590cfab985c8839b21fd5",
			),
			expectedResponse: &types.ConstructionDeriveResponse{
				AccountIdentifier: &types.AccountIdentifier{
					Address: "0xe3a5B4d7f79d64088C8d4ef153A7DDe2B2d47309",
				},
			},
		},
		"error: missing public key": {
			request:       &types.ConstructionDeriveRequest{},
			expectedError: AssetTypes.ErrInvalidInput,
		},
		"error: empty public key": {
			request:       templateDeriveRequest(""),
			expectedError: AssetTypes.ErrInvalidInput,
		},
		"error: invalid public key": {
			request:       templateDeriveRequest("invalid input"),
			expectedError: AssetTypes.ErrInvalidInput,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			resp, err := testingClient.servicer.ConstructionDerive(
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

func templateDeriveRequest(pubKey string) *types.ConstructionDeriveRequest {
	var bytes []byte
	if len(pubKey) != 0 {
		bytes, _ = hex.DecodeString(pubKey)
	}
	return &types.ConstructionDeriveRequest{
		NetworkIdentifier: &types.NetworkIdentifier{},
		PublicKey: &types.PublicKey{
			Bytes:     bytes,
			CurveType: types.Secp256k1,
		},
	}
}
