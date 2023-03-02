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
	"encoding/json"
	"fmt"

	"github.com/coinbase/rosetta-geth-sdk/client"
	sdkTypes "github.com/coinbase/rosetta-geth-sdk/types"

	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/ethereum/go-ethereum/common"
	EthTypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/coinbase/rosetta-geth-sdk/stats"
)

// ConstructionCombine implements /construction/combine endpoint.
//
// Combine creates a network-specific Transaction from an unsigned Transaction
// and an array of provided signatures. The signed Transaction returned from
// this method will be sent to the /construction/submit endpoint by the caller.
//
func (s *APIService) ConstructionCombine(
	ctx context.Context,
	req *types.ConstructionCombineRequest,
) (*types.ConstructionCombineResponse, *types.Error) {
	timer := stats.InitBlockchainClientTimer(s.statsdClient, stats.ConstructionCombineKey)
	defer timer.Emit()

	response, err := s.constructionCombine(ctx, req)
	if err != nil {
		stats.IncrementErrorCount(s.statsdClient, stats.ConstructionCombineKey, "ErrConstructionCombine")
		stats.LogError(s.logger, err.Message, stats.ConstructionCombineKey, err)
		return nil, err
	}

	return response, nil
}

func (s *APIService) constructionCombine(ctx context.Context, req *types.ConstructionCombineRequest) (*types.ConstructionCombineResponse, *types.Error) {
	if len(req.UnsignedTransaction) == 0 {
		return nil, sdkTypes.WrapErr(
			sdkTypes.ErrInvalidInput,
			fmt.Errorf("transaction data is not provided"),
		)
	}
	if len(req.Signatures) == 0 {
		return nil, sdkTypes.WrapErr(
			sdkTypes.ErrInvalidInput,
			fmt.Errorf("signature is not provided"),
		)
	}

	var unsignedTx client.Transaction
	if err := json.Unmarshal([]byte(req.UnsignedTransaction), &unsignedTx); err != nil {
		return nil, sdkTypes.WrapErr(sdkTypes.ErrInvalidInput, err)
	}

	ethTransaction := EthTypes.NewTransaction(
		unsignedTx.Nonce,
		common.HexToAddress(unsignedTx.To),
		unsignedTx.Value,
		unsignedTx.GasLimit,
		unsignedTx.GasPrice,
		unsignedTx.Data,
	)

	signer := EthTypes.LatestSignerForChainID(unsignedTx.ChainID)
	signedTx, err := ethTransaction.WithSignature(signer, req.Signatures[0].Bytes)
	if err != nil {
		return nil, sdkTypes.WrapErr(sdkTypes.ErrInvalidInput, err)
	}

	signedTxJSON, err := signedTx.MarshalJSON()
	if err != nil {
		return nil, sdkTypes.WrapErr(sdkTypes.ErrInternalError, err)
	}

	wrappedSignedTx := client.SignedTransactionWrapper{
		SignedTransaction: signedTxJSON,
		Currency:          unsignedTx.Currency,
	}

	wrappedSignedTxJSON, err := json.Marshal(wrappedSignedTx)
	if err != nil {
		return nil, sdkTypes.WrapErr(sdkTypes.ErrInternalError, err)
	}

	return &types.ConstructionCombineResponse{
		SignedTransaction: string(wrappedSignedTxJSON),
	}, nil
}
