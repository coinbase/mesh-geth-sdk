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
	"errors"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/assert"
)

func TestConstruction_ContractCallData(t *testing.T) {
	tests := map[string]struct {
		methodSig  string
		methodArgs interface{}

		expectedResponse string
		expectedError    error
	}{
		"happy path: nil args": {
			methodSig:        "deposit()",
			methodArgs:       nil,
			expectedResponse: "0xd0e30db0",
		},
		"happy path: single string arg": {
			methodSig:        "attest((bytes32,(address,uint64,bool,bytes32,bytes,uint256)))",
			methodArgs:       "0x00000000000000000000000000000000000000000000000000000000000000201cdb5651ea836ecc9be70d044e2cf7a416e5257ec8d954deb9d09a66a8264b8e000000000000000000000000000000000000000000000000000000000000004000000000000000000000000026c58c5095c8fac99e518ee951ba8f56d3c75e8e00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000c0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000001",
			expectedResponse: "0xf17325e700000000000000000000000000000000000000000000000000000000000000201cdb5651ea836ecc9be70d044e2cf7a416e5257ec8d954deb9d09a66a8264b8e000000000000000000000000000000000000000000000000000000000000004000000000000000000000000026c58c5095c8fac99e518ee951ba8f56d3c75e8e00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000c0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000001",
		},
		"happy path: list of string args": {
			methodSig:        "register(string,address,bool)",
			methodArgs:       []string{"bool abc", "0x0000000000000000000000000000000000000000", "true"},
			expectedResponse: "0x60d7a2780000000000000000000000000000000000000000000000000000000000000060000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000008626f6f6c20616263000000000000000000000000000000000000000000000000",
		},
		"happy path: list of non string args": {
			methodSig:        "register(string,address,bool)",
			methodArgs:       []interface{}{"bool abc", "0x0000000000000000000000000000000000000000", "true"},
			expectedResponse: "0x60d7a2780000000000000000000000000000000000000000000000000000000000000060000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000008626f6f6c20616263000000000000000000000000000000000000000000000000",
		},
		"happy path: no-arg method with empty string list args": {
			methodSig:        "pause()",
			methodArgs:       []string{},
			expectedResponse: "0x8456cb59",
		},
		"happy path: no-arg method with empty interface list args": {
			methodSig:        "pause()",
			methodArgs:       []interface{}{},
			expectedResponse: "0x8456cb59",
		},
		"happy path: method sig is an empty string and args is a list of interface": {
			methodSig:        "",
			methodArgs:       []interface{}{"0xabcde12345"},
			expectedResponse: "0xabcde12345",
		},
		// https://sepolia.basescan.org/tx/0x61bca9ade0522b70524ebe3e9b3064572cd22e05cdd67d650ad8657dc0f8a1f4
		"happy path: bytes array and uint256 ": {
			methodSig:        "createAccount(bytes[],uint256)",
			methodArgs:       []interface{}{"[\"0x000000000000000000000000911a81c0a2cd632fd4c45461541bf8973d11870a\",\"0x0000000000000000000000006ecb18183838265968039955f1e8829480db5329\",\"0x0000000000000000000000000bfc799df7e440b7c88cc2454f12c58f8a29d986\"]", "0"},
			expectedResponse: "0x3ffba36f000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000003000000000000000000000000000000000000000000000000000000000000006000000000000000000000000000000000000000000000000000000000000000a000000000000000000000000000000000000000000000000000000000000000e00000000000000000000000000000000000000000000000000000000000000020000000000000000000000000911a81c0a2cd632fd4c45461541bf8973d11870a00000000000000000000000000000000000000000000000000000000000000200000000000000000000000006ecb18183838265968039955f1e8829480db532900000000000000000000000000000000000000000000000000000000000000200000000000000000000000000bfc799df7e440b7c88cc2454f12c58f8a29d986",
		},
		// https://sepolia.basescan.org/tx/0x9c6854169cccd609225bec508ca11ec0fe84620493488b54d638e54350422b79
		"happy path: address and bytes": {
			methodSig:        "createRecoverySigner(address,address,bytes)",
			methodArgs:       []string{"0x4e7E5249d2Cb9255367C716e1452752A1390e44A", "0x2149ada7A6B036c0C5215A88921856D9974D810C", "0x9ba55864842d7142d780c93c0112994ba08ce5fed82d574ec864eb0f16fb6b767b69030cec56ec2662e483ae6fbddf1f4d233b9dad4bb4190d9178fc1cdfaa951c"},
			expectedResponse: "0x25cb56700000000000000000000000004e7e5249d2cb9255367c716e1452752a1390e44a0000000000000000000000002149ada7a6b036c0c5215a88921856d9974d810c000000000000000000000000000000000000000000000000000000000000006000000000000000000000000000000000000000000000000000000000000000419ba55864842d7142d780c93c0112994ba08ce5fed82d574ec864eb0f16fb6b767b69030cec56ec2662e483ae6fbddf1f4d233b9dad4bb4190d9178fc1cdfaa951c00000000000000000000000000000000000000000000000000000000000000",
		},
		"happy path: uint64 parameter": {
			methodSig:        "testFunction(address,uint256,uint64,bool)",
			methodArgs:       []string{"0x4e7E5249d2Cb9255367C716e1452752A1390e44A", "100000000000000", "200000", "true"},
			expectedResponse: "0x3e446b180000000000000000000000004e7e5249d2cb9255367c716e1452752a1390e44a00000000000000000000000000000000000000000000000000005af3107a40000000000000000000000000000000000000000000000000000000000000030d400000000000000000000000000000000000000000000000000000000000000001",
		},
		"happy path: method sig is NO-METHOD-SIG and args is a list of interface": {
			methodSig:        NoMethodSig,
			methodArgs:       []interface{}{"0xaabbcc112233"},
			expectedResponse: "0xaabbcc112233",
		},
		"error: case string: invalid method args hex data": {
			methodSig:     "attest((bytes32,(address,uint64,bool,bytes32,bytes,uint256)))",
			methodArgs:    "!!!",
			expectedError: errors.New("error decoding method args hex data: encoding/hex: invalid byte: U+0021 '!'"),
		},
		"error: case []interface: ": {
			methodSig:     "register(string,address,bool)",
			methodArgs:    []interface{}{"bool abc", "0x0000000000000000000000000000000000000000", true},
			expectedError: errors.New("invalid method_args type at index 2: bool (must be a string)"),
		},
		"error: pass arguments to no-arg method": {
			methodSig:     "pause()",
			methodArgs:    []string{"0x12345"},
			expectedError: errors.New("invalid method arguments"),
		},
		"error: pass wrong number of arguments to the method": {
			methodSig:     "pause(uint256)",
			methodArgs:    []string{"1", "2"},
			expectedError: errors.New("invalid method arguments"),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			bytes, err := ConstructContractCallDataGeneric(test.methodSig, test.methodArgs)
			if err != nil {
				assert.EqualError(t, err, test.expectedError.Error())
			} else {
				assert.Equal(t, test.expectedResponse, hexutil.Encode(bytes))
			}
		})
	}
}

func TestConstruction_MatchMethodSignature(t *testing.T) {
	delegateData, err := ConstructContractCallDataGeneric("delegate(address)", []string{testingFromAddress})
	assert.NoError(t, err)

	supported := []string{"transfer(address,uint256)", "delegate(address)"}

	tests := map[string]struct {
		supported []string
		callData  []byte
		expected  string
	}{
		"matches supported method": {
			supported: supported,
			callData:  delegateData,
			expected:  "delegate(address)",
		},
		"no match returns empty": {
			supported: []string{"transfer(address,uint256)"},
			callData:  delegateData,
			expected:  "",
		},
		"calldata shorter than selector": {
			supported: supported,
			callData:  []byte{0x01, 0x02},
			expected:  "",
		},
		"empty supported list": {
			supported: nil,
			callData:  delegateData,
			expected:  "",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			sig, matchErr := MatchMethodSignature(test.supported, test.callData)
			assert.NoError(t, matchErr)
			assert.Equal(t, test.expected, sig)
		})
	}
}

// TestConstruction_ParseContractCallData verifies that decoding calldata is the
// exact inverse of ConstructContractCallDataGeneric: encoding a method's args
// and decoding the resulting calldata yields the original string arguments.
func TestConstruction_ParseContractCallData(t *testing.T) {
	tests := map[string]struct {
		methodSig    string
		encodeArgs   interface{}
		expectedArgs []string
	}{
		"address arg": {
			methodSig:    "delegate(address)",
			encodeArgs:   []string{testingFromAddress},
			expectedArgs: []string{testingFromAddress},
		},
		"string, address, bool args": {
			methodSig:    "register(string,address,bool)",
			encodeArgs:   []string{"bool abc", "0x0000000000000000000000000000000000000000", "true"},
			expectedArgs: []string{"bool abc", "0x0000000000000000000000000000000000000000", "true"},
		},
		"address, uint256, uint64, bool args": {
			methodSig:    "testFunction(address,uint256,uint64,bool)",
			encodeArgs:   []string{"0x4e7E5249d2Cb9255367C716e1452752A1390e44A", "100000000000000", "200000", "true"},
			expectedArgs: []string{"0x4e7E5249d2Cb9255367C716e1452752A1390e44A", "100000000000000", "200000", "true"},
		},
		"address, address, bytes args": {
			methodSig:    "createRecoverySigner(address,address,bytes)",
			encodeArgs:   []string{"0x4e7E5249d2Cb9255367C716e1452752A1390e44A", "0x2149ada7A6B036c0C5215A88921856D9974D810C", "0x9ba55864842d7142d780c93c0112994ba08ce5fed82d574ec864eb0f16fb6b767b69030cec56ec2662e483ae6fbddf1f4d233b9dad4bb4190d9178fc1cdfaa951c"},
			expectedArgs: []string{"0x4e7E5249d2Cb9255367C716e1452752A1390e44A", "0x2149ada7A6B036c0C5215A88921856D9974D810C", "0x9ba55864842d7142d780c93c0112994ba08ce5fed82d574ec864eb0f16fb6b767b69030cec56ec2662e483ae6fbddf1f4d233b9dad4bb4190d9178fc1cdfaa951c"},
		},
		"no-arg method": {
			methodSig:    "pause()",
			encodeArgs:   []string{},
			expectedArgs: []string{},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			data, err := ConstructContractCallDataGeneric(test.methodSig, test.encodeArgs)
			assert.NoError(t, err)

			decoded, err := ParseContractCallData(test.methodSig, data)
			assert.NoError(t, err)
			assert.Equal(t, test.expectedArgs, decoded)
		})
	}
}

// TestConstruction_ParseContractCallData_TupleRejected ensures tuple/struct
// signatures are rejected with an error rather than silently decoding to the
// wrong (often empty) argument list.
func TestConstruction_ParseContractCallData_TupleRejected(t *testing.T) {
	tuples := []string{
		"attest((bytes32,uint256))",   // leading tuple
		"foo((address,uint256),bool)", // tuple first arg
		"bar(uint256,(address,bool))", // tuple last arg
	}

	for _, sig := range tuples {
		t.Run(sig, func(t *testing.T) {
			_, err := ParseContractCallData(sig, []byte{0x00, 0x01, 0x02, 0x03})
			assert.Error(t, err)
		})
	}
}

func TestConstruction_ValidateSupportedContractMethods(t *testing.T) {
	tests := map[string]struct {
		sigs      []string
		expectErr bool
	}{
		"valid flat signatures": {
			sigs:      []string{"delegate(address)", "transfer(address,uint256)", "pause()"},
			expectErr: false,
		},
		"empty list is valid": {
			sigs:      nil,
			expectErr: false,
		},
		"tuple argument rejected": {
			sigs:      []string{"attest((bytes32,uint256))"},
			expectErr: true,
		},
		"malformed signature rejected": {
			sigs:      []string{"notAMethod"},
			expectErr: true,
		},
		"unknown abi type rejected": {
			sigs:      []string{"foo(notAType)"},
			expectErr: true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			err := ValidateSupportedContractMethods(test.sigs)
			if test.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConstruction_preprocessArgs(t *testing.T) {
	tests := map[string]struct {
		methodSig  string
		methodArgs interface{}

		expectedResponse interface{}
		expectedError    error
	}{
		"happy path: method sig is function name": {
			methodSig: "withdraw(address,uint256,uint32,bytes)",
			methodArgs: []interface{}{
				"0x2Ae3F1Ec7F1F5012CFEab0185bfc7aa3cf0DEc22",
				"32941055343948244352",
				"0",
				"0x",
			},
			expectedResponse: []interface{}{
				"0x2Ae3F1Ec7F1F5012CFEab0185bfc7aa3cf0DEc22",
				"32941055343948244352",
				"0",
				"0x",
			},
		},
		"happy path: method sig is empty and args is nil": {
			methodSig:        "",
			methodArgs:       nil,
			expectedResponse: nil,
		},
		"happy path: method sig is NO-METHOD-SIG and args is a single string": {
			methodSig:        NoMethodSig,
			methodArgs:       "0x12345",
			expectedResponse: "0x12345",
		},
		"happy path: method sig is empty and args is a list of interface": {
			methodSig:        "",
			methodArgs:       []interface{}{"0xabcde"},
			expectedResponse: "0xabcde",
		},
		"happy path: method sig is NO-METHOD-SIG and args is a list of strings": {
			methodSig:        NoMethodSig,
			methodArgs:       []string{"0x1a2b3c"},
			expectedResponse: "0x1a2b3c",
		},
		"unhappy path: args is a list of interface and cannot be converted to strings": {
			methodSig:     "",
			methodArgs:    []interface{}{34567},
			expectedError: errors.New("failed to convert method arg \"int\" to string"),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			argsReturned, err := preprocessArgs(test.methodSig, test.methodArgs)
			if err != nil {
				assert.EqualError(t, err, test.expectedError.Error())
			} else {
				assert.Equal(t, test.expectedResponse, argsReturned)
			}
		})
	}
}
