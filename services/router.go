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
	"net/http"

	"github.com/coinbase/rosetta-geth-sdk/configuration"
	AssetTypes "github.com/coinbase/rosetta-geth-sdk/types"
	"github.com/coinbase/rosetta-sdk-go/types"

	construction "github.com/coinbase/rosetta-geth-sdk/services/construction"
	"github.com/coinbase/rosetta-sdk-go/asserter"
	"github.com/coinbase/rosetta-sdk-go/server"

	"github.com/DataDog/datadog-go/statsd"
	"go.uber.org/zap"
)

// NewBlockchainRouter creates a Mux http.Handler from a collection
// of server controllers.
func NewBlockchainRouter(
	config *configuration.Configuration,
	types *AssetTypes.Types,
	errors []*types.Error,
	client construction.Client,
	asserter *asserter.Asserter,
	logger *zap.Logger,
	statsdClient *statsd.Client,
) http.Handler {
	networkAPIService := NewNetworkAPIService(config, types, errors, client, logger, statsdClient)
	networkAPIController := server.NewNetworkAPIController(
		networkAPIService,
		asserter,
	)

	accountAPIService := NewAccountAPIService(config, types, errors, client, logger, statsdClient)
	accountAPIController := server.NewAccountAPIController(
		accountAPIService,
		asserter,
	)

	blockAPIService := NewBlockAPIService(config, client, logger, statsdClient)
	blockAPIController := server.NewBlockAPIController(
		blockAPIService,
		asserter,
	)

	constructionAPIService := construction.NewAPIService(config, types, errors, client, logger, statsdClient)
	constructionAPIController := server.NewConstructionAPIController(
		constructionAPIService,
		asserter,
	)

	// mempoolAPIService := NewMempoolAPIService()
	// mempoolAPIController := server.NewMempoolAPIController(
	// 	mempoolAPIService,
	// 	asserter,
	// )

	// callAPIService := NewCallAPIService(config, client)
	// callAPIController := server.NewCallAPIController(
	// 	callAPIService,
	// 	asserter,
	// )

	return server.NewRouter(
		networkAPIController,
		accountAPIController,
		blockAPIController,
		constructionAPIController,
		// mempoolAPIController,
		// callAPIController,
	)
}
