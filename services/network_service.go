package services

import (
	"context"

	"github.com/coinbase/rosetta-geth-sdk/configuration"
	"github.com/coinbase/rosetta-geth-sdk/services/construction"
	AssetTypes "github.com/coinbase/rosetta-geth-sdk/types"
	"github.com/coinbase/rosetta-sdk-go/asserter"

	"github.com/coinbase/rosetta-sdk-go/types"
)

// NetworkAPIService implements the server.NetworkAPIServicer interface.
type NetworkAPIService struct {
	config *configuration.Configuration
	types  *AssetTypes.Types
	errors []*types.Error
	client construction.Client
}

// NewNetworkAPIService creates a new instance of a NetworkAPIService.
func NewNetworkAPIService(
	cfg *configuration.Configuration,
	types *AssetTypes.Types,
	errors []*types.Error,
	client construction.Client,
) *NetworkAPIService {
	return &NetworkAPIService{
		config: cfg,
		types:  types,
		errors: errors,
		client: client,
	}
}

// NetworkList implements the /network/list endpoint
func (s *NetworkAPIService) NetworkList(
	ctx context.Context,
	request *types.MetadataRequest,
) (*types.NetworkListResponse, *types.Error) {
	return &types.NetworkListResponse{
		NetworkIdentifiers: []*types.NetworkIdentifier{s.config.Network},
	}, nil
}

// NetworkOptions implements the /network/options endpoint.
func (s *NetworkAPIService) NetworkOptions(
	ctx context.Context,
	request *types.NetworkRequest,
) (*types.NetworkOptionsResponse, *types.Error) {
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
