package main

import (
	"log"

	"github.com/coinbase/rosetta-geth-sdk/examples/ethereum/client"
	"github.com/coinbase/rosetta-geth-sdk/examples/ethereum/config"
	sdkTypes "github.com/coinbase/rosetta-geth-sdk/types"
	"github.com/coinbase/rosetta-geth-sdk/utils"
)

func main() {
	// Load configuration using the ENVs in the environment.
	cfg, err := config.LoadConfiguration()
	if err != nil {
		log.Fatalln("%w: unable to load configuration", err)
	}

	// Load all the supported operation types, status
	types := sdkTypes.LoadTypes()
	errors := sdkTypes.Errors

	// Create a new ethereum client by leveraging SDK functionalities
	client, err := client.NewEthereumClient()
	if err != nil {
		log.Fatalln("%w: cannot initialize client", err)
	}

	// Bootstrap to start the Rosetta API server
	err = utils.BootStrap(cfg, types, errors, client)
	if err != nil {
		log.Fatalln("%w: unable to bootstrap Rosetta server", err)
	}
}
