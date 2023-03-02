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

	"github.com/coinbase/rosetta-geth-sdk/configuration"
	"github.com/coinbase/rosetta-geth-sdk/services/construction"
	AssetTypes "github.com/coinbase/rosetta-geth-sdk/types"
	"github.com/coinbase/rosetta-sdk-go/asserter"

	"github.com/coinbase/rosetta-sdk-go/types"

	"github.com/DataDog/datadog-go/statsd"
	"github.com/coinbase/rosetta-geth-sdk/stats"
	"go.uber.org/zap"
)

// NetworkAPIService implements the server.NetworkAPIServicer interface.
type NetworkAPIService struct {
	config       *configuration.Configuration
	types        *AssetTypes.Types
	errors       []*types.Error
	client       construction.Client
	logger       *zap.Logger
	statsdClient *statsd.Client
}

// NewNetworkAPIService creates a new instance of a NetworkAPIService.
func NewNetworkAPIService(
	cfg *configuration.Configuration,
	types *AssetTypes.Types,
	errors []*types.Error,
	client construction.Client,
	logger *zap.Logger,
	statsdClient *statsd.Client,
) *NetworkAPIService {
	return &NetworkAPIService{
		config:       cfg,
		types:        types,
		errors:       errors,
		client:       client,
		logger:       logger,
		statsdClient: statsdClient,
	}
}

// NetworkList implements the /network/list endpoint
func (s *NetworkAPIService) NetworkList(
	ctx context.Context,
	request *types.MetadataRequest,
) (*types.NetworkListResponse, *types.Error) {
	time := stats.InitBlockchainClientTimer(s.statsdClient, stats.NetworkListKey)
	defer time.Emit()

	return &types.NetworkListResponse{
		NetworkIdentifiers: []*types.NetworkIdentifier{s.config.Network},
	}, nil
}

// NetworkOptions implements the /network/options endpoint.
func (s *NetworkAPIService) NetworkOptions(
	ctx context.Context,
	request *types.NetworkRequest,
) (*types.NetworkOptionsResponse, *types.Error) {
	timer := stats.InitBlockchainClientTimer(s.statsdClient, stats.NetworkOptionsKey)
	defer timer.Emit()

	return &types.NetworkOptionsResponse{
		Version: &types.Version{
			NodeVersion:    s.types.NodeVersion,
			RosettaVersion: types.RosettaAPIVersion,
		},
		Allow: &types.Allow{
			Errors:                  s.errors,
			OperationTypes:          s.types.OperationTypes,
			OperationStatuses:       s.types.OperationStatuses,
			HistoricalBalanceLookup: s.types.HistoricalBalanceSupported,
			CallMethods:             s.types.CallMethods,
		},
	}, nil
}

// NetworkStatus implements the /network/status endpoint.
func (s *NetworkAPIService) NetworkStatus(
	ctx context.Context,
	request *types.NetworkRequest,
) (*types.NetworkStatusResponse, *types.Error) {
	if s.config.IsOfflineMode() {
		return nil, AssetTypes.ErrUnavailableOffline
	}

	timer := stats.InitBlockchainClientTimer(s.statsdClient, stats.NetworkStatusKey)
	defer timer.Emit()

	response, err := s.networkStatus(ctx, request)
	if err != nil {
		stats.IncrementErrorCount(s.statsdClient, stats.NetworkStatusKey, "ErrGetNetworkStatus")
		stats.LogError(s.logger, err.Message, stats.NetworkStatusKey, AssetTypes.ErrGetNetworkStatus)
		return nil, AssetTypes.WrapErr(AssetTypes.ErrGetNetworkStatus, err)
	}

	return response, nil
}

func (s *NetworkAPIService) networkStatus(ctx context.Context, request *types.NetworkRequest) (*types.NetworkStatusResponse, *types.Error) {
	currentBlock, currentTime, syncStatus, peers, err := s.client.Status(ctx)
	if err != nil {
		return nil, AssetTypes.WrapErr(AssetTypes.ErrGeth, err)
	}

	if currentTime < asserter.MinUnixEpoch {
		return nil, AssetTypes.ErrGethNotReady
	}

	return &types.NetworkStatusResponse{
		CurrentBlockIdentifier: currentBlock,
		CurrentBlockTimestamp:  currentTime,
		GenesisBlockIdentifier: s.config.GenesisBlockIdentifier,
		SyncStatus:             syncStatus,
		Peers:                  peers,
	}, nil
}
