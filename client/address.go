package client

import (
	"log"

	"github.com/ethereum/go-ethereum/common"
)

// ChecksumAddress ensures an Ethereum hex address
// is in Checksum Format. If the address cannot be converted,
// it returns !ok.
func ChecksumAddress(address string) (string, bool) {
	addr, err := common.NewMixedcaseAddressFromString(address)
	if err != nil {
		return "", false
	}

	return addr.Address().Hex(), true
}

// MustChecksum ensures an address can be converted
// into a valid checksum. If it does not, the program
// will exit.
func MustChecksum(address string) string {
	addr, ok := ChecksumAddress(address)
	if !ok {
		log.Fatalf("invalid address %s", address)
	}

	return addr
}
