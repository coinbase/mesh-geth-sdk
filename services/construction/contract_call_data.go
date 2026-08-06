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
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/big"
	"reflect"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"golang.org/x/crypto/sha3"
)

const NoMethodSig = "NO-METHOD-SIG"

// methodIDLength is the number of bytes in an ABI function selector.
const methodIDLength = 4

// ConstructContractCallDataGeneric constructs the data field of a transaction.
// The methodArgs can be already in ABI encoded format in case of a single string
// It can also be passed in as a slice of args, which requires further encoding.
func ConstructContractCallDataGeneric(methodSig string, methodArgs interface{}) ([]byte, error) {
	data, err := contractCallMethodID(methodSig)
	if err != nil {
		return nil, err
	}

	// preprocess method args for fallback pattern contract call
	args, err := preprocessArgs(methodSig, methodArgs)
	if err != nil {
		return nil, err
	}

	// switch on the type of the method args. method args can come in from json as either a string or list of strings
	switch methodArgs := args.(type) {
	// case 0: no method arguments, return the selector
	case nil:
		return data, nil

	// case 1: method args are pre-compiled ABI data. decode the hex and create the call data directly
	case string:
		methodArgs = strings.TrimPrefix(methodArgs, "0x")
		b, decErr := hex.DecodeString(methodArgs)
		if decErr != nil {
			return nil, fmt.Errorf("error decoding method args hex data: %w", decErr)
		}
		return append(data, b...), nil

	// case 2: method args are a list of interface{} which will be converted to string before encoding
	case []interface{}:
		var strList []string
		for i, genericVal := range methodArgs {
			strVal, isStrVal := genericVal.(string)
			if !isStrVal {
				return nil, fmt.Errorf("invalid method_args type at index %d: %T (must be a string)",
					i, genericVal,
				)
			}
			strList = append(strList, strVal)
		}

		return encodeMethodArgsStrings(data, methodSig, strList)

	// case 3: method args are encoded as a list of strings, which will be decoded
	case []string:
		return encodeMethodArgsStrings(data, methodSig, methodArgs)

	// case 4: there is no known way to decode the method args
	default:
		return nil, fmt.Errorf(
			"invalid method_args type, accepted values are []string and hex-encoded string."+
				" type received=%T value=%#v", methodArgs, methodArgs,
		)
	}
}

// preprocessArgs converts methodArgs to a string value if methodSig is an empty string.
// We are calling a contract written with fallback pattern, which has no method signature.
func preprocessArgs(methodSig string, methodArgs interface{}) (interface{}, error) {
	if methodSig == "" || methodSig == NoMethodSig {
		switch args := methodArgs.(type) {
		case []interface{}:
			if len(args) == 1 {
				if argStr, ok := args[0].(string); ok {
					return argStr, nil
				}
				return nil, fmt.Errorf("failed to convert method arg \"%T\" to string", args[0])
			}
		case []string:
			if len(args) == 1 {
				return args[0], nil
			}
		}
	}

	return methodArgs, nil
}

// encodeMethodArgsStrings constructs the data field of a transaction for a list of string args.
// It attempts to first convert the string arg to it's corresponding type in the method signature,
// and then performs abi encoding to the converted args list and construct the data.
func encodeMethodArgsStrings(methodID []byte, methodSig string, methodArgs []string) ([]byte, error) {
	arguments := abi.Arguments{}
	var argumentsData []interface{}

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
	argsSignature := strings.TrimSpace(splitSigByTrailingParenthesis[0])
	var splitSigByComma []string
	if argsSignature == "" {
		// Methods like pause() have zero arguments and should accept empty arg lists.
		splitSigByComma = []string{}
	} else {
		splitSigByComma = strings.Split(argsSignature, ",")
	}

	if len(splitSigByComma) != len(methodArgs) {
		return nil, errors.New("invalid method arguments")
	}

	for i, v := range splitSigByComma {
		typed, _ := abi.NewType(v, v, nil)
		argument := abi.Arguments{
			abi.Argument{
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
					log.Print(err)
					return nil, fmt.Errorf("failed to parse uint32 argument: %w", err)
				}
				argData = uint32(u64)
			}
		case v == "uint64":
			{
				u64, err := strconv.ParseUint(methodArgs[i], 10, 64)
				if err != nil {
					log.Print(err)
					return nil, fmt.Errorf("failed to parse uint64 argument: %w", err)
				}
				argData = u64
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
					log.Print(err)
					return nil, fmt.Errorf("failed to parse bytes32 argument: %w", err)
				}
				copy(value[:], bytes)
				argData = value
			}
		// Note: we must handle "bytes[]" before "bytes" because they both share the same prefix
		case v == "bytes[]":
			{
				var bytesArgs []string
				if err := json.Unmarshal([]byte(methodArgs[i]), &bytesArgs); err != nil {
					log.Print(err)
					return nil, fmt.Errorf("failed to parse bytes[] argument: %w", err)
				}

				value := make([][]byte, len(bytesArgs))
				for j, bytesArg := range bytesArgs {
					bytes, err := hexutil.Decode(bytesArg)
					if err != nil {
						log.Print(err)
						return nil, fmt.Errorf("failed to parse bytes[] argument: %w", err)
					}
					value[j] = bytes
				}
				argData = value
			}
		case strings.HasPrefix(v, "bytes"):
			{
				// No fixed size set as it would make it an "array" instead
				// of a "slice" when encoding. We want it to be a slice.
				bytes, err := hexutil.Decode(methodArgs[i])
				if err != nil {
					log.Print(err)
					return nil, fmt.Errorf("failed to parse bytes argument: %w", err)
				}
				value := make([]byte, len(bytes))
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
					log.Print(err)
					return nil, fmt.Errorf("failed to parse bool argument: %w", err)
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
	if methodSig == "" || methodSig == NoMethodSig {
		// contract call without method signature (fallback pattern)
		return []byte{}, nil
	}

	fnSignature := []byte(methodSig)
	hash := sha3.NewLegacyKeccak256()
	if _, err := hash.Write(fnSignature); err != nil {
		return nil, err
	}

	return hash.Sum(nil)[:4], nil
}

// MatchMethodSignature returns the first supported method signature whose 4-byte
// selector matches the leading bytes of the provided calldata, or an empty
// string if the calldata is too short or does not match any supported signature.
//
// This is the inverse-lookup side of contractCallMethodID: because a selector is
// a one-way hash of the signature, recovering the human-readable signature from
// calldata is only possible against a known allowlist of signatures.
func MatchMethodSignature(supportedSigs []string, callData []byte) (string, error) {
	if len(callData) < methodIDLength {
		return "", nil
	}

	selector := callData[:methodIDLength]
	for _, sig := range supportedSigs {
		id, err := contractCallMethodID(sig)
		if err != nil {
			return "", err
		}
		if len(id) == methodIDLength && bytes.Equal(id, selector) {
			return sig, nil
		}
	}

	return "", nil
}

// ParseContractCallData decodes the ABI-encoded arguments of a contract call
// into a list of string-encoded arguments, using the argument types declared in
// methodSig. It is the inverse of encodeMethodArgsStrings: the returned strings
// use the same textual representation that the encoder accepts as input, so a
// decoded (method_signature, method_args) pair round-trips back to the original
// calldata.
//
// The methodSig is expected to already match the calldata's selector (see
// MatchMethodSignature). The leading 4-byte selector is stripped before decoding.
func ParseContractCallData(methodSig string, callData []byte) ([]string, error) {
	argTypes, err := parseSigArgTypes(methodSig)
	if err != nil {
		return nil, err
	}
	if len(argTypes) == 0 {
		return []string{}, nil
	}

	arguments := abi.Arguments{}
	for _, t := range argTypes {
		typed, typeErr := abi.NewType(t, t, nil)
		if typeErr != nil {
			return nil, fmt.Errorf("failed to construct abi type for %q: %w", t, typeErr)
		}
		arguments = append(arguments, abi.Argument{Type: typed})
	}

	var payload []byte
	if len(callData) >= methodIDLength {
		payload = callData[methodIDLength:]
	}

	values, err := arguments.UnpackValues(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to decode method args for %q: %w", methodSig, err)
	}

	strArgs := make([]string, len(values))
	for i, v := range values {
		strArgs[i], err = stringifyABIValue(v)
		if err != nil {
			return nil, err
		}
	}

	return strArgs, nil
}

// parseSigArgTypes extracts the comma-separated argument types from a method
// signature (e.g. "delegate(address,uint256)" -> ["address", "uint256"]). It
// mirrors the signature-splitting logic in encodeMethodArgsStrings.
//
// Tuple/struct arguments (which contain nested parentheses, e.g.
// "attest((bytes32,uint256))") are not supported by the decoder and are rejected
// with an error rather than silently parsed into the wrong types. This keeps the
// decoded (method_signature, method_args) pair a faithful, round-trippable
// representation of the calldata for any signature this function accepts.
func parseSigArgTypes(methodSig string) ([]string, error) {
	const split = 2
	splitByLeading := strings.Split(methodSig, "(")
	if len(splitByLeading) < split {
		return nil, fmt.Errorf("invalid method signature: %q", methodSig)
	}

	// A well-formed flat signature has exactly one "(" and one ")". More than one
	// of either means the argument list contains a tuple/struct, which the naive
	// comma split below cannot decode correctly, so reject it explicitly instead
	// of returning misleading (often empty) arg types.
	if strings.Count(methodSig, "(") != 1 || strings.Count(methodSig, ")") != 1 {
		return nil, fmt.Errorf("tuple/struct arguments are not supported: %q", methodSig)
	}

	splitByTrailing := strings.Split(splitByLeading[1], ")")
	argsSignature := strings.TrimSpace(splitByTrailing[0])
	if argsSignature == "" {
		return []string{}, nil
	}

	argTypes := strings.Split(argsSignature, ",")
	for i := range argTypes {
		argTypes[i] = strings.TrimSpace(argTypes[i])
	}

	return argTypes, nil
}

// ValidateSupportedContractMethods checks that every configured method signature
// is well-formed and that its argument types are valid, decodable ABI types. It
// is intended to be called once at startup (e.g. from the chain's config load) so
// a misconfigured SupportedContractMethods entry fails loudly at boot instead of
// silently degrading /construction/parse — or, worse, matching a selector and
// then failing to decode — on live traffic.
//
// Note that decoding at request time additionally fails closed: even if a bad
// signature slips past this check, /construction/parse omits the method fields
// rather than erroring the request.
func ValidateSupportedContractMethods(supportedSigs []string) error {
	for _, sig := range supportedSigs {
		if _, err := contractCallMethodID(sig); err != nil {
			return fmt.Errorf("invalid supported contract method %q: %w", sig, err)
		}

		argTypes, err := parseSigArgTypes(sig)
		if err != nil {
			return fmt.Errorf("invalid supported contract method %q: %w", sig, err)
		}

		for _, t := range argTypes {
			if _, err := abi.NewType(t, t, nil); err != nil {
				return fmt.Errorf(
					"invalid supported contract method %q: unsupported argument type %q: %w", sig, t, err,
				)
			}
		}
	}

	return nil
}

// stringifyABIValue converts a value produced by abi.Arguments.UnpackValues into
// the same textual form that encodeMethodArgsStrings accepts, so that decoded
// arguments round-trip back to identical calldata.
func stringifyABIValue(v interface{}) (string, error) {
	switch val := v.(type) {
	case common.Address:
		return val.Hex(), nil
	case *big.Int:
		return val.String(), nil
	case bool:
		return strconv.FormatBool(val), nil
	case string:
		return val, nil
	case uint8:
		return strconv.FormatUint(uint64(val), 10), nil
	case uint16:
		return strconv.FormatUint(uint64(val), 10), nil
	case uint32:
		return strconv.FormatUint(uint64(val), 10), nil
	case uint64:
		return strconv.FormatUint(val, 10), nil
	case int8:
		return strconv.FormatInt(int64(val), 10), nil
	case int16:
		return strconv.FormatInt(int64(val), 10), nil
	case int32:
		return strconv.FormatInt(int64(val), 10), nil
	case int64:
		return strconv.FormatInt(val, 10), nil
	case []byte:
		// Dynamically-sized bytes (e.g. "bytes").
		return hexutil.Encode(val), nil
	}

	// Handle fixed-size byte arrays (e.g. bytes32 -> [32]byte) and slices of
	// nested types (e.g. bytes[] -> [][]byte) via reflection.
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Array:
		if rv.Type().Elem().Kind() == reflect.Uint8 {
			b := make([]byte, rv.Len())
			for i := 0; i < rv.Len(); i++ {
				b[i] = byte(rv.Index(i).Uint())
			}
			return hexutil.Encode(b), nil
		}
	case reflect.Slice:
		elems := make([]string, rv.Len())
		for i := 0; i < rv.Len(); i++ {
			s, err := stringifyABIValue(rv.Index(i).Interface())
			if err != nil {
				return "", err
			}
			elems[i] = s
		}
		out, err := json.Marshal(elems)
		if err != nil {
			return "", fmt.Errorf("failed to encode decoded slice argument: %w", err)
		}
		return string(out), nil
	}

	return "", fmt.Errorf("unsupported decoded argument type %T", v)
}
