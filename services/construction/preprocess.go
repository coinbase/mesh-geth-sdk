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
)

// ConstructionPreprocess implements /construction/preprocess endpoint.
//
// Preprocess is called prior to /construction/payloads to construct a request for
// any metadata that is needed for transaction construction given (i.e. account nonce).
func (s *APIService) ConstructionPreprocess( //nolint
	ctx context.Context,
	req *types.ConstructionPreprocessRequest, ) (*types.ConstructionPreprocessResponse, *types.Error) {
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
		return nil, sdkTypes.WrapErr(sdkTypes.ErrInvalidAddress, fmt.Errorf("%s is not a valid address", fromAddress))
	}
	checkTo, ok := client.ChecksumAddress(toAddress)
	if !ok {
		return nil, sdkTypes.WrapErr(sdkTypes.ErrInvalidAddress, fmt.Errorf("%s is not a valid address", toAddress))
	}

	preprocessOptions := &client.Options{
		From:                   checkFrom,
		To:                     checkTo,
		Value:                  amount,
		SuggestedFeeMultiplier: req.SuggestedFeeMultiplier,
		Currency:               fromCurrency,
	}

	if v, ok := req.Metadata["gas_price"]; ok {
		stringObj, ok := v.(string)
		if !ok {
			return nil, sdkTypes.WrapErr(sdkTypes.ErrInvalidInput, fmt.Errorf("%s is not a valid gas price string", v))
		}
		bigObj, ok := new(big.Int).SetString(stringObj, 10) // nolint:gomnd
		if !ok {
			return nil, sdkTypes.WrapErr(sdkTypes.ErrInvalidInput, fmt.Errorf("%s is not a valid gas price", v))
		}
		preprocessOptions.GasPrice = bigObj
	}
	if v, ok := req.Metadata["gas_limit"]; ok {
		stringObj, ok := v.(string)
		if !ok {
			return nil, sdkTypes.WrapErr(sdkTypes.ErrInvalidInput, fmt.Errorf("%s is not a valid gas limit string", v))
		}
		bigObj, ok := new(big.Int).SetString(stringObj, 10) // nolint:gomnd
		if !ok {
			return nil, sdkTypes.WrapErr(sdkTypes.ErrInvalidInput, fmt.Errorf("%s is not a valid gas limit", v))
		}
		preprocessOptions.GasLimit = bigObj
	}
	if v, ok := req.Metadata["nonce"]; ok {
		stringObj, ok := v.(string)
		if !ok {
			return nil, sdkTypes.WrapErr(sdkTypes.ErrInvalidInput, fmt.Errorf("%s is not a valid nonce string", v))
		}
		bigObj, ok := new(big.Int).SetString(stringObj, 10) // nolint:gomnd
		if !ok {
			return nil, sdkTypes.WrapErr(sdkTypes.ErrInvalidInput, fmt.Errorf("%s is not a valid nonce", v))
		}
		preprocessOptions.Nonce = bigObj
	}

	if v, ok := req.Metadata["method_signature"]; ok {
		methodSigStringObj := v.(string)
		if !ok {
			return nil, sdkTypes.WrapErr(sdkTypes.ErrInvalidInput, fmt.Errorf("%s is not a valid method signature string", v))
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
		return nil, errors.New("Invalid method arguments")
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
		case strings.HasPrefix(v, "uint") || strings.HasPrefix(v, "int"):
			{
				value := new(big.Int)
				value.SetString(methodArgs[i], base)
				argData = value
			}
		case strings.HasPrefix(v, "bytes"):
			{
				value := [32]byte{}
				copy(value[:], methodArgs[i])
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
	abiEncodeData, _ := arguments.PackValues(argumentsData)
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
