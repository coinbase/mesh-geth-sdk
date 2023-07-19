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

package client

import (
	"log"

	"github.com/ethereum/go-ethereum/common"
)

// ChecksumAddress ensures an address can be EIP55-compliant format
func ChecksumAddress(address string) (string, error) {
	addr, err := common.NewMixedcaseAddressFromString(address)
	if err != nil {
		return "", err
	}

	return addr.Address().Hex(), nil
}

// MustChecksum ensures an address can be EIP55-compliant format
func MustChecksum(address string) string {
	addr, err := ChecksumAddress(address)
	if err != nil {
		log.Fatalf("invalid address %s: %v", address, err)
	}

	return addr
}
