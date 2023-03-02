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

	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/coinbase/rosetta-geth-sdk/stats"
	AssetTypes "github.com/coinbase/rosetta-geth-sdk/types"
)

// ConstructionDerive implements the /construction/derive endpoint.
func (s *APIService) ConstructionDerive(
	ctx context.Context, req *types.ConstructionDeriveRequest,
) (*types.ConstructionDeriveResponse, *types.Error) {
	if req.PublicKey == nil {
		stats.LogError(s.logger, "public key is not provided", stats.ConstructionDeriveKey, AssetTypes.ErrInvalidInput)
		return nil, AssetTypes.WrapErr(AssetTypes.ErrInvalidInput, "public key is not provided")
	}

	timer := stats.InitBlockchainClientTimer(s.statsdClient, stats.ConstructionDeriveKey)
	defer timer.Emit()

	key, err := crypto.DecompressPubkey(req.PublicKey.Bytes) // hex_bytes
	if err != nil {
		stats.LogError(s.logger, err.Error(), stats.ConstructionDeriveKey, AssetTypes.ErrConstructionDerive)
		return nil, AssetTypes.WrapErr(AssetTypes.ErrConstructionDerive, err.Error())
	}

	return &types.ConstructionDeriveResponse{
		AccountIdentifier: &types.AccountIdentifier{
			Address: crypto.PubkeyToAddress(*key).Hex(),
		},
	}, nil
}
