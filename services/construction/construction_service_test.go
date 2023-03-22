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
	"math/big"

	"github.com/coinbase/rosetta-geth-sdk/configuration"
	"github.com/coinbase/rosetta-geth-sdk/mocks/services"
	AssetTypes "github.com/coinbase/rosetta-geth-sdk/types"
	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/ethereum/go-ethereum/params"

	"github.com/coinbase/rosetta-geth-sdk/stats"
)

var (
	ethereumCurrencyConfig = &types.Currency{
		Symbol:   "ETH",
		Decimals: 18,
	}

	ethereumNetworkIdentifier = &types.NetworkIdentifier{
		Network:    "Ropsten",
		Blockchain: "Ethereum",
	}

	ethRopstenChainID = uint64(3)

	testingFromAddress = "0x97158A00a4D227Ec7fe3234B52f21e5608FeE3d1"
	testingToAddress   = "0xdF7C4fFf31A190E8D46FC9Ba8CdE6aaD8F69Fc76"
)

type testingClient struct {
	cfg        *configuration.Configuration
	mockClient *services.Client
	servicer   *APIService
}

func newTestingClient() *testingClient {
	cfg := &configuration.Configuration{
		Mode:    configuration.ModeOnline,
		Network: ethereumNetworkIdentifier,
		RosettaCfg: configuration.RosettaConfig{
			Currency: ethereumCurrencyConfig,
		},
		ChainConfig: &params.ChainConfig{ChainID: big.NewInt(int64(ethRopstenChainID))},
		ServiceName: configuration.DefaultServiceName,
	}
	mockClient := &services.Client{}
	mockLogger, _, _ := stats.InitLogger(cfg)
	mockStats, _, _ := stats.InitStatsd(mockLogger, cfg)
	servicer := NewAPIService(
		cfg,
		AssetTypes.LoadTypes(),
		AssetTypes.Errors,
		mockClient,
		mockLogger,
		mockStats,
	)

	return &testingClient{
		cfg:        cfg,
		mockClient: mockClient,
		servicer:   servicer,
	}
}

func templateOperations(
	amount uint64,
	currency *types.Currency,
	operationType string,
) []*types.Operation {
	return rosettaOperations(
		testingFromAddress,
		testingToAddress,
		big.NewInt(int64(amount)),
		currency,
		operationType,
	)
}

func bigAmountTemplateOperations(
	amount *big.Int,
	currency *types.Currency,
	operationType string,
) []*types.Operation {
	return rosettaOperations(
		testingFromAddress,
		testingToAddress,
		amount,
		currency,
		operationType,
	)
}

func rosettaOperations(
	fromAddress string,
	toAddress string,
	amount *big.Int,
	currency *types.Currency,
	operationType string,
) []*types.Operation {
	return []*types.Operation{
		{
			OperationIdentifier: &types.OperationIdentifier{
				Index: 0,
			},
			Type: operationType,
			Account: &types.AccountIdentifier{
				Address: fromAddress,
			},
			Amount: &types.Amount{
				Value:    new(big.Int).Neg(amount).String(),
				Currency: currency,
			},
		},
		{
			OperationIdentifier: &types.OperationIdentifier{
				Index: 1,
			},
			Type: operationType,
			Account: &types.AccountIdentifier{
				Address: toAddress,
			},
			Amount: &types.Amount{
				Value:    amount.String(),
				Currency: currency,
			},
		},
	}
}

func templateError(error *types.Error, context string) *types.Error {
	return &types.Error{
		Code:      error.Code,
		Message:   error.Message,
		Retriable: false,
		Details: map[string]interface{}{
			"context": context,
		},
	}
}
