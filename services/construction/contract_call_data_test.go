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
	"fmt"
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
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			bytes, err := ConstructContractCallDataGeneric(test.methodSig, test.methodArgs)
			if err != nil {
				fmt.Println(err)
				assert.EqualError(t, err, test.expectedError.Error())
			} else {
				assert.Equal(t, test.expectedResponse, hexutil.Encode(bytes))
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
				"0x"},
			expectedResponse: []interface{}{
				"0x2Ae3F1Ec7F1F5012CFEab0185bfc7aa3cf0DEc22",
				"32941055343948244352",
				"0",
				"0x"},
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
