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
	"errors"
	"log"

	"github.com/ethereum/go-ethereum/common"
)

// ChecksumAddress ensures an address is EIP55-compliant
func ChecksumAddress(address string) error {
	addr, err := common.NewMixedcaseAddressFromString(address)
	if err != nil {
		return err
	}

	if !addr.ValidChecksum() {
		return errors.New("checksum address is not equal to original address")
	}

	return nil
}

// MustChecksum ensures an address is EIP55-compliant
// If it does not, the program will exit.
func MustChecksum(address string) string {
	err := ChecksumAddress(address)
	if err != nil {
		log.Fatalf("invalid address %s", address)
	}

	return address
}
