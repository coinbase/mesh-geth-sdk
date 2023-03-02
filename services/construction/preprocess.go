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
	"errors"
	"fmt"
	"log"
	"math/big"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"golang.org/x/crypto/sha3"

	"github.com/coinbase/rosetta-geth-sdk/client"
	sdkTypes "github.com/coinbase/rosetta-geth-sdk/types"

	"github.com/coinbase/rosetta-sdk-go/parser"
	"github.com/coinbase/rosetta-sdk-go/types"

	"github.com/coinbase/rosetta-geth-sdk/stats"
)

// ConstructionPreprocess implements /construction/preprocess endpoint.
//
// Preprocess is called prior to /construction/payloads to construct a request for
// any metadata that is needed for transaction construction given (i.e. account nonce).
func (s *APIService) ConstructionPreprocess( //nolint
	ctx context.Context,
	req *types.ConstructionPreprocessRequest) (*types.ConstructionPreprocessResponse, *types.Error) {

	timer := stats.InitBlockchainClientTimer(s.statsdClient, stats.ConstructionPreprocessKey)
	defer timer.Emit()

	response, err := s.constructionPreprocess(ctx, req)
	if err != nil {
		stats.IncrementErrorCount(s.statsdClient, stats.ConstructionPreprocessKey, "ErrConstructionPreprocess")
		stats.LogError(s.logger, err.Message, stats.ConstructionPreprocessKey, err)
		return nil, err
	}

	return response, nil
}

// constructContractCallData constructs the data field of a transaction
func constructContractCallData(methodSig string, methodArgs []string) ([]byte, error) {
	arguments := abi.Arguments{}
	argumentsData := []interface{}{}

	methodID, err := contractCallMethodID(methodSig)
	if err != nil {
		return nil, err
	}

	var data []byte
	data = append(data, methodID...)

	const split = 2
	splitSigByLeadingParenthesis := strings.Split(methodSig, "(")
	if len(splitSigByLeadingParenthesis) < split {
		return data, nil
	}
	splitSigByTrailingParenthesis := strings.Split(splitSigByLeadingParenthesis[1], ")")
	if len(splitSigByTrailingParenthesis) < 1 {
		return data, nil
	}
	splitSigByComma := strings.Split(splitSigByTrailingParenthesis[0], ",")

	if len(splitSigByComma) != len(methodArgs) {
		return nil, errors.New("invalid method arguments")
	}

	for i, v := range splitSigByComma {
		typed, _ := abi.NewType(v, v, nil)
		argument := abi.Arguments{
			{
				Type: typed,
			},
		}

		arguments = append(arguments, argument...)
		var argData interface{}
		const base = 10
		switch {
		case v == "address":
			{
				argData = common.HexToAddress(methodArgs[i])
			}
		case v == "uint32":
			{
				u64, err := strconv.ParseUint(methodArgs[i], 10, 32)
				if err != nil {
					log.Fatal(err)
				}
				argData = uint32(u64)
			}
		case strings.HasPrefix(v, "uint") || strings.HasPrefix(v, "int"):
			{
				value := new(big.Int)
				value.SetString(methodArgs[i], base)
				argData = value
			}
		case v == "bytes32":
			{
				value := [32]byte{}
				bytes, err := hexutil.Decode(methodArgs[i])
				if err != nil {
					log.Fatal(err)
				}
				copy(value[:], bytes)
				argData = value
			}
		case strings.HasPrefix(v, "bytes"):
			{
				// No fixed size set as it would make it an "array" instead
				// of a "slice" when encoding. We want it to be a slice.
				value := []byte{}
				bytes, err := hexutil.Decode(methodArgs[i])
				if err != nil {
					log.Fatal(err)
				}
				copy(value[:], bytes) // nolint:gocritic
				argData = value
			}
		case strings.HasPrefix(v, "string"):
			{
				argData = methodArgs[i]
			}
		case strings.HasPrefix(v, "bool"):
			{
				value, err := strconv.ParseBool(methodArgs[i])
				if err != nil {
					log.Fatal(err)
				}
				argData = value
			}
		}
		argumentsData = append(argumentsData, argData)
	}

	abiEncodeData, err := arguments.PackValues(argumentsData)
	if err != nil {
		return nil, fmt.Errorf("failed to encode arguments: %w", err)
	}

	data = append(data, abiEncodeData...)
	return data, nil
}

// contractCallMethodID calculates the first 4 bytes of the method
// signature for function call on contract
func contractCallMethodID(methodSig string) ([]byte, error) {
	fnSignature := []byte(methodSig)
	hash := sha3.NewLegacyKeccak256()
	if _, err := hash.Write(fnSignature); err != nil {
		log.Fatal(err)
		return nil, err
	}

	return hash.Sum(nil)[:4], nil
}

func (s *APIService) constructionPreprocess(ctx context.Context, req *types.ConstructionPreprocessRequest) (*types.ConstructionPreprocessResponse, *types.Error) {
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
	toOp, amount := matches[1].First()
	toAddress := toOp.Account.Address

	fromCurrency := fromOp.Amount.Currency

	checkFrom, ok := client.ChecksumAddress(fromAddress)
	if !ok {
		return nil, sdkTypes.WrapErr(
			sdkTypes.ErrInvalidAddress,
			fmt.Errorf("%s is not a valid address", fromAddress),
		)
	}
	checkTo, ok := client.ChecksumAddress(toAddress)
	if !ok {
		return nil, sdkTypes.WrapErr(
			sdkTypes.ErrInvalidAddress,
			fmt.Errorf("%s is not a valid address", toAddress),
		)
	}

	preprocessOptions := &client.Options{
		From:                   checkFrom,
		To:                     checkTo,
		Value:                  amount.String(),
		SuggestedFeeMultiplier: req.SuggestedFeeMultiplier,
		Currency:               fromCurrency,
	}

	if v, ok := req.Metadata["gas_price"]; ok {
		stringObj, ok := v.(string)
		if !ok {
			return nil, sdkTypes.WrapErr(
				sdkTypes.ErrInvalidInput,
				fmt.Errorf("%s is not a valid gas price string", v),
			)
		}
		bigObj, ok := new(big.Int).SetString(stringObj, 10) // nolint:gomnd
		if !ok {
			return nil, sdkTypes.WrapErr(
				sdkTypes.ErrInvalidInput,
				fmt.Errorf("%s is not a valid gas price", v),
			)
		}
		preprocessOptions.GasPrice = bigObj
	}
	if v, ok := req.Metadata["gas_limit"]; ok {
		stringObj, ok := v.(string)
		if !ok {
			return nil, sdkTypes.WrapErr(
				sdkTypes.ErrInvalidInput,
				fmt.Errorf("%s is not a valid gas limit string", v),
			)
		}
		bigObj, ok := new(big.Int).SetString(stringObj, 10) // nolint:gomnd
		if !ok {
			return nil, sdkTypes.WrapErr(
				sdkTypes.ErrInvalidInput,
				fmt.Errorf("%s is not a valid gas limit", v),
			)
		}
		preprocessOptions.GasLimit = bigObj
	}
	if v, ok := req.Metadata["nonce"]; ok {
		stringObj, ok := v.(string)
		if !ok {
			return nil, sdkTypes.WrapErr(
				sdkTypes.ErrInvalidInput,
				fmt.Errorf("%s is not a valid nonce string", v),
			)
		}
		bigObj, ok := new(big.Int).SetString(stringObj, 10) // nolint:gomnd
		if !ok {
			return nil, sdkTypes.WrapErr(
				sdkTypes.ErrInvalidInput,
				fmt.Errorf("%s is not a valid nonce", v),
			)
		}
		preprocessOptions.Nonce = bigObj
	}

	if v, ok := req.Metadata["method_signature"]; ok {
		methodSigStringObj := v.(string)
		if !ok {
			return nil, sdkTypes.WrapErr(
				sdkTypes.ErrInvalidInput,
				fmt.Errorf("%s is not a valid method signature string", v),
			)
		}
		var methodArgs []string
		if v, ok := req.Metadata["method_args"]; ok {
			methodArgsBytes, _ := json.Marshal(v)
			err := json.Unmarshal(methodArgsBytes, &methodArgs)
			if err != nil {
				fmt.Println("Error in unmarshal")
			}
		}
		data, err := constructContractCallData(methodSigStringObj, methodArgs)
		if err != nil {
			return nil, sdkTypes.WrapErr(sdkTypes.ErrInvalidInput, err)
		}
		preprocessOptions.ContractAddress = checkTo
		preprocessOptions.ContractData = hexutil.Encode(data)
		preprocessOptions.MethodSignature = methodSigStringObj
		preprocessOptions.MethodArgs = methodArgs
	}

	marshaled, err := client.MarshalJSONMap(preprocessOptions)
	if err != nil {
		return nil, sdkTypes.ErrInvalidInput
	}

	return &types.ConstructionPreprocessResponse{
		Options: marshaled,
	}, nil
}
