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

	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/coinbase/rosetta-geth-sdk/client"
	sdkTypes "github.com/coinbase/rosetta-geth-sdk/types"

	"github.com/coinbase/rosetta-sdk-go/parser"
	"github.com/coinbase/rosetta-sdk-go/types"
)

// ConstructionPreprocess implements /construction/preprocess endpoint.
//
// Preprocess is called prior to /construction/payloads to construct a request for
// any metadata that is needed for transaction construction given (i.e. account nonce).
func (s *APIService) ConstructionPreprocess( //nolint
	ctx context.Context,
	req *types.ConstructionPreprocessRequest) (*types.ConstructionPreprocessResponse, *types.Error) {
	isContractCall := false
	if _, ok := req.Metadata["method_signature"]; ok {
		isContractCall = true
	}

	operationDescriptions, err := s.CreateOperationDescription(req.Operations, isContractCall)
	if err != nil {
		return nil, sdkTypes.WrapErr(sdkTypes.ErrInvalidInput, err)
	}

	descriptions := &parser.Descriptions{
		OperationDescriptions: operationDescriptions,
		ErrUnmatched:          true,
	}

	matches, err := parser.MatchOperations(descriptions, req.Operations)
	if err != nil {
		return nil, sdkTypes.WrapErr(sdkTypes.ErrInvalidInput, err)
	}

	fromOp, _ := matches[0].First()
	fromAddress := fromOp.Account.Address
	currency := fromOp.Amount.Currency
	toOp, amount := matches[1].First()
	toAddress := toOp.Account.Address

	// Address validation
	from, err := client.ChecksumAddress(fromAddress)
	if err != nil {
		return nil, sdkTypes.WrapErr(sdkTypes.ErrInvalidAddress, fmt.Errorf("%s is not a valid address: %w", fromAddress, err))
	}
	to, err := client.ChecksumAddress(toAddress)
	if err != nil {
		return nil, sdkTypes.WrapErr(sdkTypes.ErrInvalidAddress, fmt.Errorf("%s is not a valid address: %w", toAddress, err))
	}

	preprocessOptions := &client.Options{
		From:                   from,
		To:                     to,
		Value:                  amount.String(),
		SuggestedFeeMultiplier: req.SuggestedFeeMultiplier,
		Currency:               currency,
	}

	// Load tx construction data from metadata
	if err := loadMetadata(req, preprocessOptions); err != nil {
		return nil, sdkTypes.WrapErr(sdkTypes.ErrInvalidInput, err)
	}

	options, err := client.MarshalJSONMap(preprocessOptions)
	if err != nil {
		return nil, sdkTypes.ErrInvalidInput
	}

	return &types.ConstructionPreprocessResponse{
		Options: options,
	}, nil
}

func loadNumericMetadata(req *types.ConstructionPreprocessRequest, metadata string, options *client.Options) error {
	if v, ok := req.Metadata[metadata]; ok {
		stringObj, ok := v.(string)
		if !ok {
			return fmt.Errorf("%s is not a valid %s string", v, metadata)
		}

		bigObj, ok := new(big.Int).SetString(stringObj, 10) // nolint:gomnd
		if !ok {
			return fmt.Errorf("%s is not a valid %s", v, metadata)
		}

		switch metadata {
		case "gas_price":
			options.GasPrice = bigObj
		case "gas_limit":
			options.GasLimit = bigObj
		case "nonce":
			options.Nonce = bigObj
		case "gas_tip_cap":
			options.GasTipCap = bigObj
		case "gas_fee_cap":
			options.GasFeeCap = bigObj
		}
	}

	return nil
}

func loadMetadata(req *types.ConstructionPreprocessRequest, options *client.Options) error {
	if err := loadNumericMetadata(req, "gas_price", options); err != nil {
		return err
	}
	if err := loadNumericMetadata(req, "gas_limit", options); err != nil {
		return err
	}
	if err := loadNumericMetadata(req, "nonce", options); err != nil {
		return err
	}
	if err := loadNumericMetadata(req, "gas_tip_cap", options); err != nil {
		return err
	}
	if err := loadNumericMetadata(req, "gas_fee_cap", options); err != nil {
		return err
	}

	if v, ok := req.Metadata["method_signature"]; ok {
		methodSigStringObj, ok := v.(string)
		if !ok {
			return fmt.Errorf("%s is not a valid method signature string", v)
		}

		data, err := ConstructContractCallDataGeneric(methodSigStringObj, req.Metadata["method_args"])
		if err != nil {
			return err
		}

		options.ContractAddress = options.To
		options.ContractData = hexutil.Encode(data)
		options.MethodSignature = methodSigStringObj
		options.MethodArgs = req.Metadata["method_args"]
	}

	return nil
}
