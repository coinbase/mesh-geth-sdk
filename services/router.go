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
	"net/http"

	"github.com/coinbase/rosetta-geth-sdk/configuration"
	"github.com/coinbase/rosetta-geth-sdk/headers"
	AssetTypes "github.com/coinbase/rosetta-geth-sdk/types"
	"github.com/coinbase/rosetta-sdk-go/types"

	construction "github.com/coinbase/rosetta-geth-sdk/services/construction"
	"github.com/coinbase/rosetta-sdk-go/asserter"
	"github.com/coinbase/rosetta-sdk-go/server"
)

// NewBlockchainRouter creates a Mux http.Handler from a collection
// of server controllers.
func NewBlockchainRouter(
	config *configuration.Configuration,
	types *AssetTypes.Types,
	errors []*types.Error,
	client construction.Client,
	asserter *asserter.Asserter,
) http.Handler {
	var contextFromRequest func(r *http.Request) context.Context = nil
	if config.RosettaCfg.SupportHeaderForwarding {
		contextFromRequest = headers.ContextWithHeaders
	}

	networkAPIService := NewNetworkAPIService(config, types, errors, client)
	networkAPIController := server.NewNetworkAPIController(
		networkAPIService,
		asserter,
		contextFromRequest,
	)

	accountAPIService := NewAccountAPIService(config, types, errors, client)
	accountAPIController := server.NewAccountAPIController(
		accountAPIService,
		asserter,
		contextFromRequest,
	)

	blockAPIService := NewBlockAPIService(config, client)
	blockAPIController := server.NewBlockAPIController(
		blockAPIService,
		asserter,
		contextFromRequest,
	)

	constructionAPIService := construction.NewAPIService(config, types, errors, client)
	constructionAPIController := server.NewConstructionAPIController(
		constructionAPIService,
		asserter,
		contextFromRequest,
	)

	// mempoolAPIService := NewMempoolAPIService()
	// mempoolAPIController := server.NewMempoolAPIController(
	// 	mempoolAPIService,
	// 	asserter,
	//  contextFromRequest,
	// )

	// callAPIService := NewCallAPIService(config, client)
	// callAPIController := server.NewCallAPIController(
	// 	callAPIService,
	// 	asserter,
	//  contextFromRequest,
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
