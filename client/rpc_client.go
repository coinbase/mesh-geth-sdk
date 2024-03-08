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

package client

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/ethereum/go-ethereum/rpc"
)

// JSONRPC is the interface for accessing go-ethereum's JSON RPC endpoint.
// https://github.com/ethereum/go-ethereum/blob/0169d579d0eed4f6366697985a7b0f0b99402783/rpc/client.go#L308
type JSONRPC interface {
	CallContext(ctx context.Context, result interface{}, method string, args ...interface{}) error
	BatchCallContext(ctx context.Context, b []rpc.BatchElem) error
	Close()
}

type RPCClient struct {
	JSONRPC
}

// GraphQL is the interface for accessing go-ethereum's GraphQL endpoint.
type GraphQL interface {
	Query(ctx context.Context, input string) (string, error)
}

const (
	// DefaultElapsedTime is the default limit on time
	// spent retrying a fetch.
	DefaultElapsedTime = 1 * time.Minute

	// DefaultRetries is the default number of times to
	// attempt a retry on a failed request.
	DefaultRetries = 10

	// DefaultHTTPTimeout is the default timeout for
	// HTTP requests.
	DefaultHTTPTimeout = 10 * time.Second

	// DefaultIdleConnTimeout is the maximum amount of time an idle
	// (keep-alive) connection will remain idle before closing
	// itself.
	DefaultIdleConnTimeout = 30 * time.Second

	// DefaultMaxConnections limits the number of concurrent
	// connections we will attempt to make. Most OS's have a
	// default connection limit of 128, so we set the default
	// below that.
	DefaultMaxConnections = 120
)

// NewRPCClient connects a SDKClient to the given URL.
func NewRPCClient(endpoint string) (*RPCClient, error) {
	// Override transport idle connection settings
	//
	// See this conversation around why `.Clone()` is used here:
	// https://github.com/golang/go/issues/26013
	defaultTransport := http.DefaultTransport.(*http.Transport).Clone()
	defaultTransport.IdleConnTimeout = DefaultIdleConnTimeout
	defaultTransport.MaxIdleConns = DefaultMaxConnections
	defaultTransport.MaxIdleConnsPerHost = DefaultMaxConnections

	clientOptions := rpc.WithHTTPClient(&http.Client{
		Timeout:   gethHTTPTimeout,
		Transport: defaultTransport,
	})
	ctx := context.Background()
	client, err := rpc.DialOptions(ctx, endpoint, clientOptions)
	/*client, err := rpc.DialHTTPWithClient(endpoint, &http.Client{
		Timeout:   gethHTTPTimeout,
		Transport: defaultTransport,
	})*/
	if err != nil {
		return nil, fmt.Errorf("unable to dial node: %w", err)
	}
	return &RPCClient{client}, nil
}

// Close shuts down the RPC SDKClient connection.
func (ec *RPCClient) Close() {}
