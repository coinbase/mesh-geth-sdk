package client

import (
	"fmt"

	"github.com/ethereum/go-ethereum/ethclient"
)

type EthClient struct {
	*ethclient.Client
}

// NewEthClient connects a SDKClient to the given URL.
func NewEthClient(endpoint string) (*EthClient, error) {
	client, err := ethclient.Dial(endpoint)

	if err != nil {
		return nil, fmt.Errorf("%w: unable to dial node", err)
	}

	return &EthClient{client}, nil
}

// Close shuts down the RPC SDKClient connection.
func (ec *EthClient) Close() {}
