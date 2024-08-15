// Copyright 2020 Coinbase, Inc.
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
	"fmt"

	"github.com/coinbase/rosetta-geth-sdk/configuration"
	AssetTypes "github.com/coinbase/rosetta-geth-sdk/types"

	construction "github.com/coinbase/rosetta-geth-sdk/services/construction"
	"github.com/coinbase/rosetta-sdk-go/types"
)

// AccountAPIService implements the server.AccountAPIServicer interface.
type AccountAPIService struct {
	config *configuration.Configuration
	types  *AssetTypes.Types
	errors []*types.Error
	client construction.Client
}

// NewAccountAPIService returns a new *AccountAPIService.
func NewAccountAPIService(
	cfg *configuration.Configuration,
	types *AssetTypes.Types,
	errors []*types.Error,
	client construction.Client,
) *AccountAPIService {
	return &AccountAPIService{
		config: cfg,
		types:  types,
		errors: errors,
		client: client,
	}
}

// AccountBalance implements /account/balance.
func (s *AccountAPIService) AccountBalance(
	ctx context.Context,
	request *types.AccountBalanceRequest,
) (*types.AccountBalanceResponse, *types.Error) {
	// if s.config.Mode != AssetTypes.Online {
	// 	return nil, AssetTypes.ErrUnavailableOffline
	// }

	if request.AccountIdentifier == nil {
		return nil, AssetTypes.ErrInvalidInput
	}

	balanceResponse, err := s.client.Balance(
		ctx,
		request.AccountIdentifier,
		request.BlockIdentifier,
		request.Currencies,
	)
	if err != nil {
		return nil, AssetTypes.WrapErr(AssetTypes.ErrGeth, err)
	}

	// get block hash if the block hash can't be calculated from keccak256 hash of its RLP encoding
	balanceResponse.BlockIdentifier.Hash, err = s.client.GetBlockHash(ctx, *balanceResponse.BlockIdentifier)
	if err != nil {
		return nil, AssetTypes.WrapErr(AssetTypes.ErrInternalError, fmt.Errorf("could not get block hash given block identifier %v: %w", request.BlockIdentifier, err))
	}

	return balanceResponse, nil
}

// AccountCoins implements /account/coins.
func (s *AccountAPIService) AccountCoins(
	ctx context.Context,
	request *types.AccountCoinsRequest,
) (*types.AccountCoinsResponse, *types.Error) {
	return nil, AssetTypes.WrapErr(AssetTypes.ErrUnimplemented, nil)
}
