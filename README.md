<h3 align="center">
Mesh Generic go-ethereum-based SDK
</h3>
<!-- add shields here after publication -->
<p align="center">
Build once. 
Integrate your blockchain everywhere.
</p>

## Overview

The `mesh-geth-sdk` repository provides a collection of packages used for interaction with the Mesh API specification. The goal of this SDK is to help accelerate Mesh API implementation on go-ethereum based chains. 

[Mesh](https://docs.cdp.coinbase.com/mesh/docs/welcome/) is an open-source specification and set of tools that makes integrating with blockchains simpler, faster, and more reliable. The Mesh API is specified in the [OpenAPI 3.0 format](https://www.openapis.org).

Jump to:

* [Getting Started](#Getting-Started)
* [Quick Examples](#Quick-Examples)
* [Testing](#Testing)
* [Documentation](#Documentation)
* [Related Projects](#Related-Projects)

## Quick Examples

### Complete SDK Example

This [ethereum example](examples/ethereum/) provides a reference implementation of the Mesh API for Ethereum in Golang. This example highlights how to load a configuration, load supported operation types and errors, create a new Ethereum client, and start the Mesh server.

```go
package main

import (
	"log"

	"github.com/coinbase/mesh-geth-sdk/examples/ethereum/client"
	"github.com/coinbase/mesh-geth-sdk/examples/ethereum/config"
	sdkTypes "github.com/coinbase/mesh-geth-sdk/types"
	"github.com/coinbase/mesh-geth-sdk/utils"
)

func main() {
	// Load configuration using the ENVs in the environment.
	cfg, err := config.LoadConfiguration()
	if err != nil {
		log.Fatalln("unable to load configuration: %w", err)
	}

	// Load all the supported operation types, status
	types := sdkTypes.LoadTypes()
	errors := sdkTypes.Errors

	// Create a new ethereum client by leveraging SDK functionalities
	client, err := client.NewEthereumClient()
	if err != nil {
		log.Fatalln("cannot initialize client: %w", err)
	}

	// Bootstrap to start the Mesh API server
	err = utils.BootStrap(cfg, types, errors, client)
	if err != nil {
		log.Fatalln("unable to bootstrap Mesh server: %w", err)
	}
}
```

The [LoadConfiguration](examples/ethereum/config/config.go) file loads all the Blockchain metadata and Mesh configuration.

The [NewEthereumClient](examples/ethereum/client/client.go) file creates a new service client, which leverages SDK functionalities, and implements a few gas related interfaces.

### Valid Tokens

See (examples/tokenList.json) for a list of supported ERC20 tokens.

### SDK Packages

* [Client](client): Low-level communication with any `go-ethereum` based blockchain
* [Services](services): Mesh RESTful services for Data and Construction APIs
* [Utils](utils): Bootstrap code for starting up a Mesh API server
* [Examples](examples): Examples of how to build your Mesh integration with the SDK

### Configuring the SDK

See the [Configuration](configuration/configuration.go) file for more information on how to configure the SDK.

### SDK interfaces and method overriding
The SDK defines a list of [Client interfaces](services/construction/types.go), which allows the Mesh service to interact with a go-ethereum based blockchain.

The SDK provides a default implementation of the client module for a quick start. However, these interfaces require custom integration to override:
<!-- These links will need to be updated when we publish to GitHub -->
1. [`GetBlockReceipts`](https://github.com/coinbase/mesh-geth-sdk/blob/1a8ac6c199732ab06a40725ccc07f34b52a2b46e/services/construction/types.go#L124)
2. [`GetTransactionReceipt`](https://github.com/coinbase/mesh-geth-sdk/blob/1a8ac6c199732ab06a40725ccc07f34b52a2b46e/services/construction/types.go#L133)
3. [`GetNativeTransferGasLimit`](https://github.com/coinbase/mesh-geth-sdk/blob/1a8ac6c199732ab06a40725ccc07f34b52a2b46e/services/construction/types.go#L140)


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

## Testing

To validate `mesh-geth-sdk`, [install `mesh-cli`](https://github.com/coinbase/mesh-cli#install) and run one of the following commands:

* `mesh-cli check:data --configuration-file mesh-cli-conf/testnet/config.json` - This command validates that the Data API implementation is correct, using the ethereum `testnet` node. It also ensures that the implementation does not miss any balance-changing operations.
* `mesh-cli check:construction --configuration-file mesh-cli-conf/testnet/config.json` - This command validates the Construction API implementation. It also verifies transaction construction, signing, and submissions to the `testnet` network.
* `mesh-cli check:data --configuration-file mesh-cli-conf/mainnet/config.json` - This command validates that the Data API implementation is correct, using the ethereum `mainnet` node. It also ensures that the implementation does not miss any balance-changing operations.

Read the [How to Test your Mesh Implementation](https://docs.cdp.coinbase.com/mesh/docs/mesh-test/) documentation for additional details.

## Contributing

You may contribute to the `mesh-geth-sdk` project in various ways:

* [Asking Questions](CONTRIBUTING.md/#asking-questions)
* [Providing Feedback](CONTRIBUTING.md/#providing-feedback)
* [Reporting Issues](CONTRIBUTING.md/#reporting-issues)

Read our [Contributing](CONTRIBUTING.MD) documentation for more information.

You can also find community implementations for a variety of blockchains in the [mesh-ecosystem](https://github.com/coinbase/mesh-ecosystem) repository.

## Documentation

You can find the Mesh API documentation [here](https://docs.cdp.coinbase.com/mesh/docs/welcome/). 

Check out the [Getting Started](https://docs.cdp.coinbase.com/mesh/docs/getting-started/) section to start diving into Mesh. 

## Related Projects

* [mesh-sdk-go](https://github.com/coinbase/mesh-sdk-go) — The `mesh-sdk-go` SDK provides a collection of packages used for interaction with the Mesh API specification. 
* [mesh-specifications](https://github.com/coinbase/mesh-specifications) — Much of the SDK code is generated from this repository.
* [mesh-cli](https://github.com/coinbase/mesh-cli) — Use the `mesh-cli` tool to test your Mesh API implementation. The tool also provides the ability to look up block contents and account balances.

## License

This project is available open source under the terms of the [Apache 2.0 License](https://opensource.org/licenses/Apache-2.0).

© 2022 Coinbase
