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
	"math/big"

	"errors"

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

	// Address validation
	if len(input.From) == 0 {
		return nil, sdkTypes.WrapErr(sdkTypes.ErrInvalidInput, errors.New("from address is not provided"))
	}
	if len(input.To) == 0 {
		return nil, sdkTypes.WrapErr(sdkTypes.ErrInvalidInput, errors.New("to address is not provided"))
	}
	if _, err := client.ChecksumAddress(input.From); err != nil {
		return nil, sdkTypes.WrapErr(sdkTypes.ErrInvalidAddress, fmt.Errorf("%s is not a valid address: %w", input.From, err))
	}
	if _, err := client.ChecksumAddress(input.To); err != nil {
		return nil, sdkTypes.WrapErr(sdkTypes.ErrInvalidAddress, fmt.Errorf("%s is not a valid address: %w", input.To, err))
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
	if input.GasLimit == nil || input.GasLimit.Uint64() == 0 {
		switch {
		case len(input.ContractAddress) > 0:
			contractAddress, err := client.ChecksumAddress(input.ContractAddress)
			if err != nil {
				return nil, sdkTypes.WrapErr(sdkTypes.ErrInvalidAddress, fmt.Errorf("%s is not a valid address: %w", input.To, err))
			}

			contractData, err := hexutil.Decode(input.ContractData)
			if err != nil {
				return nil, sdkTypes.WrapErr(sdkTypes.ErrInvalidInput, err)
			}

			// Override the destination address to be the contract address
			gasLimit, err = s.client.GetContractCallGasLimit(ctx, contractAddress, input.From, contractData)
			if err != nil {
				return nil, sdkTypes.WrapErr(sdkTypes.ErrERC20GasLimitError, err)
			}
		case input.Currency == nil || types.Hash(input.Currency) == types.Hash(s.config.RosettaCfg.Currency):
			value := new(big.Int)
			value.SetString(input.Value, 10) // nolint:gomnd

			gasLimit, err = s.client.GetNativeTransferGasLimit(ctx, input.To, input.From, value)
			if err != nil {
				return nil, sdkTypes.WrapErr(sdkTypes.ErrNativeGasLimitError, err)
			}
		default:
			value := new(big.Int)
			value.SetString(input.Value, 10) // nolint:gomnd

			gasLimit, err = s.client.GetErc20TransferGasLimit(ctx, input.To, input.From, value, input.Currency)
			if err != nil {
				return nil, sdkTypes.WrapErr(sdkTypes.ErrERC20GasLimitError, err)
			}
		}
	} else {
		gasLimit = input.GasLimit.Uint64()
	}

	var gasTipCap *big.Int
	var gasFeeCap *big.Int
	if s.client.GetRosettaConfig().SupportsEIP1559 {
		gasTipCap, err = s.client.GetGasTipCap(ctx, input)
		if err != nil {
			return nil, sdkTypes.WrapErr(sdkTypes.ErrGasTipCapError, err)
		}

		gasFeeCap, err = s.client.GetGasFeeCap(ctx, input, gasTipCap)
		if err != nil {
			return nil, sdkTypes.WrapErr(sdkTypes.ErrGasFeeCapError, err)
		}
	}

	metadata := &client.Metadata{
		Nonce:           nonce,
		GasPrice:        gasPrice,
		GasLimit:        gasLimit,
		GasTipCap:       gasTipCap,
		GasFeeCap:       gasFeeCap,
		ContractData:    input.ContractData,
		MethodSignature: input.MethodSignature,
		MethodArgs:      input.MethodArgs,
	}

	metadataMap, err := client.MarshalJSONMap(metadata)
	if err != nil {
		return nil, sdkTypes.WrapErr(sdkTypes.ErrInternalError, err)
	}

	suggestedFee := gasPrice.Int64() * int64(gasLimit)
	if gasFeeCap != nil {
		suggestedFee = gasFeeCap.Int64() * int64(gasLimit)
	}

	return &types.ConstructionMetadataResponse{
		Metadata: metadataMap,
		SuggestedFee: []*types.Amount{
			client.Amount(big.NewInt(suggestedFee), s.config.RosettaCfg.Currency),
		},
	}, nil
}
