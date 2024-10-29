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

package configuration

import (
	"math/big"

	RosettaTypes "github.com/coinbase/rosetta-sdk-go/types"
	"github.com/ethereum/go-ethereum/params"
)

// Configuration determines how the we setup the blockchain and Rosetta server
type Configuration struct {
	// Mode is the setting that determines if
	// the implementation is "online" or "offline".
	Mode Mode

	// NetworkIdentifier specifies which network our Rosetta server uses
	Network *RosettaTypes.NetworkIdentifier

	// GenesisBlockIdentifier is the genesis block
	GenesisBlockIdentifier *RosettaTypes.BlockIdentifier

	// GethURL is the blockchain node we are connecting to
	GethURL string

	// RemoteGeth indicates whether we are using a local or remote blockchain node
	RemoteGeth bool

	// Port is the Rosetta server's listening port
	Port int

	// SkipGethAdminEnv is an optional environment variable
	// to skip geth `admin` calls which are typically not supported
	// by hosted node services. When not set, defaults to false.
	SkipGethAdmin bool

	// GethArguments are the arguments to start a blockchain instance.
	GethArguments string

	// ChainConfig is the core config which determines the blockchain settings.
	ChainConfig *params.ChainConfig

	// RosettaCfg defines the config that used to implement Rosetta APIs
	RosettaCfg RosettaConfig
}

type RosettaConfig struct {
	// SupportRewardTx indicates whether the blockchain supports block reward
	SupportRewardTx bool

	// TraceType sets which type of tracing the blockchain supports
	// The options are: GethNativeTrace, GethJsTrace, and OpenEthereumTrace
	TraceType int

	// SupportsSyncing indicates if the blockchain support eth_syncing RPC or not.
	// Status syncing is used in Rosetta /network/status api
	SupportsSyncing bool

	// SupportsPeering indicates if the blockchain support admin_peers RPC or not.
	// Peers retrieving is used in Rosetta /network/status api
	SupportsPeering bool

	// SupportsBlockAuthor indicates if blockchain supports author
	SupportsBlockAuthor bool

	// SupportsEIP1559 indicates if the blockchain supports EIP-1559
	SupportsEIP1559 bool

	// SupportsOpStack indicates if the blockchain supports OP stack
	SupportsOpStack bool

	// Currency is the native currency blockchain supports
	Currency *RosettaTypes.Currency

	// TracePrefix is the prefix appended to trace RPC calls
	TracePrefix string

	// IngestionMode indicates if blockchain ingestion mode
	IngestionMode string

	// IndexUnknownTokens determines whether we parse unknown ERC20 tokens
	IndexUnknownTokens bool

	// FilterToken determines whether we using our token whitelist
	FilterTokens bool

	// TokenWhiteList is a list of ERC20 tokens we only support
	TokenWhiteList []Token

	// UseTokenWhiteListMetadata indicates whether we use token metadata from token white list or fetch from nodes
	UseTokenWhiteListMetadata bool

	// DefaultBlockNumber is the default block number if block identifier is not specified
	// This is mainly used for Optimism and Base, it can be "safe" or "finalized" to avoid reorg issues
	DefaultBlockNumber string

	// BaseFeeFloor is the floor base fee for EIP-1559
	BaseFeeFloor *big.Int

	// BaseFeeMultiplier is the multiplier of base fee for EIP-1559
	BaseFeeMultiplier *big.Int

	// PriorityFeeDivisor is the divisor of priority fee for EIP-1559
	PriorityFeeDivisor *big.Int

	// SupportCustomizedTraceConfig indicates if the blockchain supports customized trace config
	SupportCustomizedTraceConfig bool

	// CustomizedTraceConfig is the blockchain customized trace config
	CustomizedTraceConfig interface{}

	// SupportCustomizedBlockBody indicates if the blockchain supports customized block body
	SupportCustomizedBlockBody bool

	// SupportHeaderForwarding indicates if rosetta should forward rosetta request headers to the
	// native node, and forward native node response headers to the rosetta caller
	SupportHeaderForwarding bool

	// ForwardHeaders is the list of headers to forward to and from the native node
	ForwardHeaders []string
}

type Token struct {
	ChainID  uint64 `json:"chainId"`
	Address  string `json:"address"`
	Name     string `json:"name"`
	Symbol   string `json:"symbol"`
	Decimals uint64 `json:"decimals"`
}

// Mode is the setting that determines if
// the implementation is "online" or "offline".
type Mode string

// HTTPHeader is key, value pair to be set on the HTTP and GraphQL SDKClient.
type HTTPHeader struct {
	Key   string
	Value string
}

const (
	GethNativeTrace   = iota // == 0
	GethJsTrace       = iota // == 1
	OpenEthereumTrace = iota // == 2

	ModeOffline        = "OFFLINE"
	ModeOnline         = "ONLINE"
	StandardIngestion  = "standard"
	AnalyticsIngestion = "analytics"

	DefaultBaseFeeFloor       = 0
	DefaultBaseFeeMultiplier  = 1
	DefaultPriorityFeeDivisor = 1
)

// IsOfflineMode returns true if running in offline mode
func (c Configuration) IsOfflineMode() bool {
	return c.Mode == ModeOffline
}

// IsOnlineMode returns true if running in online mode
func (c Configuration) IsOnlineMode() bool {
	return c.Mode == ModeOnline
}

// IsStandardMode returns true if running in standard
func (c Configuration) IsStandardMode() bool {
	return c.RosettaCfg.IngestionMode == StandardIngestion
}

// IsAnalyticsMode returns true if running in standard
func (c Configuration) IsAnalyticsMode() bool {
	return c.RosettaCfg.IngestionMode == AnalyticsIngestion
}

// IsTokenListEmpty returns true if the token addresses list is empty
func (c Configuration) IsTokenListEmpty() bool {
	return len(c.RosettaCfg.TokenWhiteList) == 0
}
