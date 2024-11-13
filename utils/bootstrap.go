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

package utils

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	gethSdkClient "github.com/coinbase/rosetta-geth-sdk/client"
	"github.com/coinbase/rosetta-geth-sdk/configuration"
	"github.com/coinbase/rosetta-geth-sdk/services"
	"github.com/coinbase/rosetta-geth-sdk/services/construction"

	AssetTypes "github.com/coinbase/rosetta-geth-sdk/types"

	"github.com/coinbase/rosetta-sdk-go/asserter"
	"github.com/coinbase/rosetta-sdk-go/headerforwarder"
	"github.com/coinbase/rosetta-sdk-go/server"
	RosettaTypes "github.com/coinbase/rosetta-sdk-go/types"
	"github.com/neilotoole/errgroup"
)

const (
	ReadHeaderTimeout = time.Minute
)

// BootStrap quickly starts the Rosetta server
// and begin to serve Rosetta RESTful requests
func BootStrap(
	cfg *configuration.Configuration,
	types *AssetTypes.Types,
	errors []*RosettaTypes.Error,
	client construction.Client,
) error {
	// The asserter automatically rejects incorrectly formatted requests.
	asserter, err := asserter.NewServer(
		types.OperationTypes,
		AssetTypes.HistoricalBalanceSupported,
		[]*RosettaTypes.NetworkIdentifier{cfg.Network},
		types.CallMethods,
		AssetTypes.IncludeMempoolCoins,
		"",
	)
	if err != nil {
		return fmt.Errorf("could not initialize server asserter: %w", err)
	}

	// If header forwarding is turned on, initialize a new client
	var headerForwarder *headerforwarder.HeaderForwarder
	if cfg.RosettaCfg.SupportHeaderForwarding {
		replaceableClient, isReplaceable := client.(gethSdkClient.ReplaceableRPCClient)
		if !isReplaceable {
			return fmt.Errorf("SupportHeaderForwarding enabled, but client does not implement ReplaceableRPCClient")
		}

		headerForwarder, err = headerforwarder.NewHeaderForwarder(
			cfg.RosettaCfg.ForwardHeaders,
			gethSdkClient.NewDefaultHTTPTransport(),
		)
		if err != nil {
			return fmt.Errorf("SupportHeaderForwarding enabled, but header forwarder creation failed: %w", err)
		}

		replacedClient, err := replaceableClient.WithRPCTransport(cfg.GethURL, headerForwarder)
		if err != nil {
			return fmt.Errorf("SupportHeaderForwarding enabled, but client replacement failed: %w", err)
		}

		convertedClient, ok := replacedClient.(construction.Client)
		if !ok {
			return fmt.Errorf("SupportHeaderForwarding enabled, but converting replaced client type failed")
		}

		client = convertedClient
	}

	router := services.NewBlockchainRouter(cfg, types, errors, client, asserter)

	if cfg.RosettaCfg.SupportHeaderForwarding {
		router = headerForwarder.HeaderForwarderHandler(router)
	}

	// Add this middleware last so that it executes first
	loggedRouter := server.LoggerMiddleware(router)
	corsRouter := server.CorsMiddleware(loggedRouter)

	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.Port),
		Handler:           corsRouter,
		ReadHeaderTimeout: ReadHeaderTimeout,
	}

	// Start required services
	ctx := context.Background()

	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		log.Printf("server listening on port %d", cfg.Port)
		return server.ListenAndServe()
	})

	g.Go(func() error {
		// If we don't shutdown server in errgroup, it will
		// never stop because server.ListenAndServe doesn't
		// take any context.
		<-ctx.Done()

		return server.Shutdown(ctx)
	})

	err = g.Wait()

	return err
}
