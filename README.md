<p align="center">
  <a href="https://www.rosetta-api.org">
    <img width="90%" alt="Rosetta" src="https://www.rosetta-api.org/img/rosetta_header.png">
  </a>
</p>
<h3 align="center">
Rosetta Generic go-ethereum-based SDK
</h3>
<!-- add shields here after publication -->
<p align="center">
Build once. 
Integrate your blockchain everywhere.
</p>

## Overview

The `rosetta-geth-sdk` repository provides a collection of packages used for interaction with the Rosetta API specification. The goal of this SDK is to help accelerate Rosetta API implementation on go-ethereum based chains. 

[Rosetta](https://www.rosetta-api.org/docs/welcome.html) is an open-source specification and set of tools that makes integrating with blockchains simpler, faster, and more reliable. The Rosetta API is specified in the [OpenAPI 3.0 format](https://www.openapis.org).

Jump to:

* [Getting Started](#Getting-Started)
* [Quick Examples](#Quick-Examples)
* [Testing](#Test-the-Implementation-with-rosetta-cli)
* [Documentation](#Documentation)
* [Related Projects](#Related-Projects)

## Quick Examples

### Complete SDK Example

This [ethereum example](examples/ethereum/) provides a reference implementation of the Rosetta API for Ethereum in Golang. This example highlights how to load a configuration, load supported operation types and errors, create a new Ethereum client, and start the Rosetta server.

```go
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
```

The [LoadConfiguration](examples/ethereum/config/config.go) file loads all the Blockchain metadata and Rosetta configuration.

The [NewEthereumClient](examples/ethereum/client/client.go) file creates a new service client, which leverages SDK functionalities, and implements a few gas related interfaces.

### SDK Packages

* [Client](client): Low-level communication with any `go-ethereum` based blockchain
* [Services](services): Rosetta RESTful services for Data and Construction APIs
* [Utils](utils): Boostrap code for starting up a Rosetta API server
* [Examples](examples): Examples of how to build your Rosetta integration with the SDK

### Configuring the SDK

See the [Configuration](configuration/configuration.go) file for more information on how to configure the SDK.

### SDK interfaces and method overriding
The SDK defines a list of [Client interfaces](services/construction/types.go), which allows the Rosetta service to interact with a go-ethereum based blockchain.

The SDK provides a default implementation of the client module for a quick start. However, these interfaces require custom integration to override:
<!-- These links will need to be updated when we publish to GitHub -->
1. [`GetBlockReceipts`](https://github.com/coinbase/rosetta-geth-sdk/blob/1a8ac6c199732ab06a40725ccc07f34b52a2b46e/services/construction/types.go#L124)
2. [`GetTransactionReceipt`](https://github.com/coinbase/rosetta-geth-sdk/blob/1a8ac6c199732ab06a40725ccc07f34b52a2b46e/services/construction/types.go#L133)
3. [`GetNativeTransferGasLimit`](https://github.com/coinbase/rosetta-geth-sdk/blob/1a8ac6c199732ab06a40725ccc07f34b52a2b46e/services/construction/types.go#L140)


## Development

Helpful commands for development:

### Install dependencies

```
make deps
```

### Run tests

```
make test
```

### Lint the source code

```
make lint
```

### Check for security concerns

```
make salus
```

### Generate a coverage report

```
make coverage-local
```

## Test the Implementation with rosetta-cli

To validate `rosetta-geth-sdk`, [install `rosetta-cli`](https://github.com/coinbase/rosetta-cli#install) and run one of the following commands:

* `rosetta-cli check:data --configuration-file rosetta-cli-conf/testnet/config.json` - This command validates that the Data API implementation is correct, using the ethereum `testnet` node. It also ensures that the implementation does not miss any balance-changing operations.
* `rosetta-cli check:construction --configuration-file rosetta-cli-conf/testnet/config.json` - This command validates the Construction API implementation. It also verifies transaction construction, signing, and submissions to the `testnet` network.
* `rosetta-cli check:data --configuration-file rosetta-cli-conf/mainnet/config.json` - This command validates that the Data API implementation is correct, using the ethereum `mainnet` node. It also ensures that the implementation does not miss any balance-changing operations.

Read the [How to Test your Rosetta Implementation](https://www.rosetta-api.org/docs/rosetta_test.html) documentation for additional details.

## Contributing

You may contribute to the `rosetta-geth-sdk` project in various ways:

* [Asking Questions](CONTRIBUTING.md/#asking-questions)
* [Providing Feedback](CONTRIBUTING.md/#providing-feedback)
* [Reporting Issues](CONTRIBUTING.md/#reporting-issues)

Read our [Contributing](CONTRIBUTING.MD) documentation for more information.

When you've finished an implementation for a blockchain, share your work in the [ecosystem category of the community site](https://community.rosetta-api.org/c/ecosystem). Platforms looking for implementations for certain blockchains will be monitoring this section of the website for high-quality implementations they can use for integration. Make sure that your implementation meets the [expectations](https://www.rosetta-api.org/docs/node_deployment.html) of any implementation.

You can also find community implementations for a variety of blockchains in the [rosetta-ecosystem](https://github.com/coinbase/rosetta-ecosystem) repository.

## Documentation

You can find the Rosetta API documentation at [rosetta-api.org](https://www.rosetta-api.org/docs/welcome.html). 

Check out the [Getting Started](https://www.rosetta-api.org/docs/getting_started.html) section to start diving into Rosetta. 

Our documentation is divided into the following sections:

* [Product Overview](https://www.rosetta-api.org/docs/welcome.html)
* [Getting Started](https://www.rosetta-api.org/docs/getting_started.html)
* [Rosetta API Spec](https://www.rosetta-api.org/docs/Reference.html)
* [Testing](https://www.rosetta-api.org/docs/rosetta_cli.html)
* [Best Practices](https://www.rosetta-api.org/docs/node_deployment.html)
* [Repositories](https://www.rosetta-api.org/docs/rosetta_specifications.html)

## Related Projects

* [rosetta-sdk-go](https://github.com/coinbase/rosetta-sdk-go) — The `rosetta-sdk-go` SDK provides a collection of packages used for interaction with the Rosetta API specification. 
* [rosetta-specifications](https://github.com/coinbase/rosetta-specifications) — Much of the SDK code is generated from this repository.
* [rosetta-cli](https://github.com/coinbase/rosetta-cli) — Use the `rosetta-cli` tool to test your Rosetta API implementation. The tool also provides the ability to look up block contents and account balances.

## License

This project is available open source under the terms of the [Apache 2.0 License](https://opensource.org/licenses/Apache-2.0).

© 2022 Coinbase
