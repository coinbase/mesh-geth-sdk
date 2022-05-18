package utils

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/coinbase/rosetta-geth-sdk/configuration"
	"github.com/coinbase/rosetta-geth-sdk/services"
	"github.com/coinbase/rosetta-geth-sdk/services/construction"

	AssetTypes "github.com/coinbase/rosetta-geth-sdk/types"

	"github.com/coinbase/rosetta-sdk-go/asserter"
	"github.com/coinbase/rosetta-sdk-go/server"
	RosettaTypes "github.com/coinbase/rosetta-sdk-go/types"
	"github.com/neilotoole/errgroup"
)

// BootStrap quickly starts the Rosetta server
// and begin to serve Rosetta RESTful requests
func BootStrap(
	cfg *configuration.Configuration,
	types *AssetTypes.Types,
	errors []*RosettaTypes.Error,
	client construction.Client,
) error {
	// The asserter automatically rejects incorrectly formatted
	// requests.
	asserter, err := asserter.NewServer(
		types.OperationTypes,
		AssetTypes.HistoricalBalanceSupported,
		[]*RosettaTypes.NetworkIdentifier{cfg.Network},
		types.CallMethods,
		AssetTypes.IncludeMempoolCoins,
		"",
	)
	if err != nil {
		return fmt.Errorf("%w: could not initialize server asserter", err)
	}
	router := services.NewBlockchainRouter(cfg, types, errors, client, asserter)

	loggedRouter := server.LoggerMiddleware(router)
	corsRouter := server.CorsMiddleware(loggedRouter)
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Port),
		Handler: corsRouter,
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
