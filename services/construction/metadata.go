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

package construction

import (
	"context"
	"fmt"
	"github.com/ethereum/go-ethereum/log"
	"math/big"

	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/coinbase/rosetta-geth-sdk/client"
	sdkTypes "github.com/coinbase/rosetta-geth-sdk/types"

	"github.com/coinbase/rosetta-sdk-go/types"
)

// ConstructionMetadata implements /construction/metadata endpoint.
//
// Get any information required to construct a transaction for a specific network.
// Metadata returned here could be a recent hash to use, an account sequence number,
// or even arbitrary chain state. The request used when calling this endpoint
// is created by calling /construction/preprocess in an offline environment.
//
func (s APIService) ConstructionMetadata( //nolint
	ctx context.Context,
	req *types.ConstructionMetadataRequest,
) (*types.ConstructionMetadataResponse, *types.Error) {
	if s.config.Mode != sdkTypes.Online {
		return nil, sdkTypes.ErrUnavailableOffline
	}

	var input client.Options
	if err := client.UnmarshalJSONMap(req.Options, &input); err != nil {
		return nil, sdkTypes.WrapErr(sdkTypes.ErrInvalidInput, err)
	}

	if len(input.From) == 0 {
		return nil, sdkTypes.WrapErr(
			sdkTypes.ErrInvalidInput,
			fmt.Errorf("from address is not provided"),
		)
	}

	if len(input.To) == 0 {
		return nil, sdkTypes.WrapErr(
			sdkTypes.ErrInvalidInput,
			fmt.Errorf("to address is not provided"),
		)
	}

	_, okFrom := client.ChecksumAddress(input.From)
	if !okFrom {
		return nil, sdkTypes.WrapErr(
			sdkTypes.ErrInvalidInput,
			fmt.Errorf("%s is not a valid address", input.From),
		)
	}
	_, okTo := client.ChecksumAddress(input.To)
	if !okTo {
		return nil, sdkTypes.WrapErr(
			sdkTypes.ErrInvalidInput,
			fmt.Errorf("%s is not a valid address", input.To),
		)
	}

	nonce, err := s.client.GetNonce(ctx, input)
	if err != nil {
		return nil, sdkTypes.WrapErr(sdkTypes.ErrNonceError, err)
	}

	gasPrice, err := s.client.GetGasPrice(ctx, input)
	if err != nil {
		return nil, sdkTypes.WrapErr(sdkTypes.ErrGasPriceError, err)
	}

	var gasLimit uint64
	if input.GasLimit == nil {
		switch {
		case len(input.ContractAddress) > 0:
			log.Info("Fetching generic contract call gas limit")
			checkContractAddress, ok := client.ChecksumAddress(input.ContractAddress)
			if !ok {
				return nil, sdkTypes.WrapErr(
					sdkTypes.ErrInvalidInput,
					fmt.Errorf("%s is not a valid address", input.To),
				)
			}
			contractData, err := hexutil.Decode(input.ContractData)
			if err != nil {
				return nil, sdkTypes.WrapErr(sdkTypes.ErrInvalidInput, err)
			}
			// Override the destination address to be the contract address
			gasLimit, err = s.client.GetContractCallGasLimit(
				ctx,
				checkContractAddress,
				input.From,
				contractData,
			)
			if err != nil {
				// client error
				return nil, sdkTypes.WrapErr(sdkTypes.ErrERC20GasLimitError, err)
			}
		case input.Currency == nil || types.Hash(input.Currency) == types.Hash(s.config.RosettaCfg.Currency):
			log.Info("Fetching native gas limit")
			gasLimit, err = s.client.GetNativeTransferGasLimit(
				ctx,
				input.To,
				input.From,
				input.Value,
			)
			if err != nil {
				// client error
				return nil, sdkTypes.WrapErr(sdkTypes.ErrNativeGasLimitError, err)
			}
		default:
			log.Info("Fetching ERC20 gas limit")
			gasLimit, err = s.client.GetErc20TransferGasLimit(
				ctx,
				input.To,
				input.From,
				input.Value,
				input.Currency,
			)
			if err != nil {
				// client error
				return nil, sdkTypes.WrapErr(sdkTypes.ErrERC20GasLimitError, err)
			}
		}
	} else {
		log.Info("Setting existing gas limit")
		gasLimit = input.GasLimit.Uint64()
	}

	metadata := &client.Metadata{
		Nonce:           nonce,
		GasPrice:        gasPrice,
		GasLimit:        gasLimit,
		ContractData:    input.ContractData,
		MethodSignature: input.MethodSignature,
		MethodArgs:      input.MethodArgs,
	}

	metadataMap, err := client.MarshalJSONMap(metadata)
	if err != nil {
		return nil, sdkTypes.WrapErr(sdkTypes.ErrInternalError, err)
	}

	suggestedFee := gasPrice.Int64() * int64(gasLimit)
	return &types.ConstructionMetadataResponse{
		Metadata: metadataMap,
		SuggestedFee: []*types.Amount{
			client.Amount(big.NewInt(suggestedFee), s.config.RosettaCfg.Currency),
		},
	}, nil
}
