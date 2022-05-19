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
	"testing"

	mockedServices "github.com/coinbase/rosetta-geth-sdk/mocks/services"

	AssetTypes "github.com/coinbase/rosetta-geth-sdk/types"
	types "github.com/coinbase/rosetta-sdk-go/types"

	"github.com/coinbase/rosetta-geth-sdk/configuration"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
)

var (
	defaultNetworkOptions = &types.NetworkOptionsResponse{
		Version: &types.Version{
			RosettaVersion: types.RosettaAPIVersion,
			NodeVersion:    "1.9.24",
		},
		Allow: &types.Allow{
			OperationStatuses:       AssetTypes.OperationStatuses,
			OperationTypes:          AssetTypes.OperationTypes,
			Errors:                  AssetTypes.Errors,
			HistoricalBalanceLookup: AssetTypes.HistoricalBalanceSupported,
			CallMethods:             AssetTypes.CallMethods,
		},
	}

	networkIdentifier = &types.NetworkIdentifier{
		Network:    "testnet",
		Blockchain: "rosetta",
	}

	TestnetGenesisHash = common.HexToHash(
		"0x7ee576b35482195fc49205cec9af72ce14f003b9ae69f6ba0faef4514be8b442",
	)
	TestnetGenesisBlockIdentifier = &types.BlockIdentifier{
		Hash:  TestnetGenesisHash.Hex(),
		Index: 0,
	}
	loadedTypes = AssetTypes.LoadTypes()
)

func TestNetworkEndpoints_Offline(t *testing.T) {
	cfg := &configuration.Configuration{
		Mode:    configuration.ModeOffline,
		Network: networkIdentifier,
	}
	mockClient := &mockedServices.Client{}
	servicer := NewNetworkAPIService(
		cfg,
		loadedTypes,
		AssetTypes.Errors,
		mockClient,
	)
	ctx := context.Background()

	networkList, err := servicer.NetworkList(ctx, nil)
	assert.Nil(t, err)
	assert.Equal(t, []*types.NetworkIdentifier{
		networkIdentifier,
	}, networkList.NetworkIdentifiers)

	networkStatus, err := servicer.NetworkStatus(ctx, nil)
	assert.Nil(t, networkStatus)
	assert.Equal(t, AssetTypes.ErrUnavailableOffline.Code, err.Code)
	assert.Equal(t, AssetTypes.ErrUnavailableOffline.Message, err.Message)

	networkOptions, err := servicer.NetworkOptions(ctx, nil)
	assert.Nil(t, err)
	assert.Equal(t, defaultNetworkOptions, networkOptions)

	mockClient.AssertExpectations(t)
}

func TestNetworkEndpoints_Online(t *testing.T) {
	cfg := &configuration.Configuration{
		Mode:                   configuration.ModeOnline,
		Network:                networkIdentifier,
		GenesisBlockIdentifier: TestnetGenesisBlockIdentifier,
	}
	mockClient := &mockedServices.Client{}
	servicer := NewNetworkAPIService(
		cfg,
		loadedTypes,
		AssetTypes.Errors,
		mockClient,
	)
	ctx := context.Background()

	networkList, err := servicer.NetworkList(ctx, nil)
	assert.Nil(t, err)
	assert.Equal(t, []*types.NetworkIdentifier{
		networkIdentifier,
	}, networkList.NetworkIdentifiers)

	currentBlock := &types.BlockIdentifier{
		Index: 10,
		Hash:  "block 10",
	}

	currentTime := int64(1000000000000)

	syncStatus := &types.SyncStatus{
		CurrentIndex: types.Int64(100),
	}

	peers := []*types.Peer{
		{
			PeerID: "77.93.223.9:8333",
		},
	}

	mockClient.On(
		"Status",
		ctx,
	).Return(
		currentBlock,
		currentTime,
		syncStatus,
		peers,
		nil,
	)
	networkStatus, err := servicer.NetworkStatus(ctx, nil)
	assert.Nil(t, err)
	assert.Equal(t, &types.NetworkStatusResponse{
		GenesisBlockIdentifier: TestnetGenesisBlockIdentifier,
		CurrentBlockIdentifier: currentBlock,
		CurrentBlockTimestamp:  currentTime,
		Peers:                  peers,
		SyncStatus:             syncStatus,
	}, networkStatus)

	networkOptions, err := servicer.NetworkOptions(ctx, nil)
	assert.Nil(t, err)
	assert.Equal(t, defaultNetworkOptions, networkOptions)

	mockClient.AssertExpectations(t)
}
