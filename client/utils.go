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
	"encoding/json"
	"golang.org/x/crypto/sha3"
	"math/big"

	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

const (
	BalanceOfMethodPrefix      = "0x70a08231000000000000000000000000"
	GenericTransferBytesLength = 68
	requiredPaddingBytes       = 32
	TransferFnSignature        = "transfer(address,uint256)" // do not include spaces in the string
	Milliseconds               = 1000
)

// has0xPrefix validates str begins with '0x' or '0X'.
// Copied from the go-ethereum hextuil.go library
func has0xPrefix(str string) bool {
	return len(str) >= 2 && str[0] == '0' && (str[1] == 'x' || str[1] == 'X')
}

func Amount(value *big.Int, currency *types.Currency) *types.Amount {
	if value == nil {
		return nil
	}
	return &types.Amount{
		Value:    value.String(),
		Currency: currency,
	}
}

func ToBlockNumArg(number *big.Int) string {
	if number == nil {
		return "latest"
	}
	pending := big.NewInt(-1)
	if number.Cmp(pending) == 0 {
		return "pending"
	}
	return hexutil.EncodeBig(number)
}

func Erc20Amount(
	bytes []byte,
	addr common.Address,
	symbol string,
	decimals int32,
	sender bool) *types.Amount {
	value := common.BytesToHash(bytes).Big()

	if sender {
		value = new(big.Int).Neg(value)
	}

	currency := Erc20Currency(symbol, decimals, addr.String())
	return &types.Amount{
		Value:    value.String(),
		Currency: currency,
	}
}

func Erc20Currency(symbol string, decimals int32, contractAddress string) *types.Currency {
	return &types.Currency{
		Symbol:   symbol,
		Decimals: decimals,
		Metadata: map[string]interface{}{
			"contractAddress": contractAddress,
		},
	}
}

func convertTime(time uint64) int64 {
	return int64(time) * Milliseconds
}

// Custom UnmarshalJSON for RPCTransaction to populate tx, and extra info.
func (tx *RPCTransaction) UnmarshalJSON(msg []byte) error {
	if err := json.Unmarshal(msg, &tx.Tx); err != nil {
		return err
	}
	return json.Unmarshal(msg, &tx.TxExtraInfo)
}

// UnmarshalJSONMap converts map[string]interface{} into a interface{}.
func UnmarshalJSONMap(m map[string]interface{}, i interface{}) error {
	b, err := json.Marshal(m)
	if err != nil {
		return err
	}

	return json.Unmarshal(b, i)
}

// marshalJSONMap converts an interface into a map[string]interface{}.
func MarshalJSONMap(i interface{}) (map[string]interface{}, error) {
	b, err := json.Marshal(i)
	if err != nil {
		return nil, err
	}

	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}

	return m, nil
}

func getTransferMethodID() []byte {
	transferSignature := []byte(TransferFnSignature) // do not include spaces in the string
	hash := sha3.NewLegacyKeccak256()
	hash.Write(transferSignature)
	methodID := hash.Sum(nil)[:4]
	return methodID
}

func Account(address *common.Address) *types.AccountIdentifier {
	if address == nil {
		return nil
	}
	return &types.AccountIdentifier{
		Address: address.String(),
	}
}

// ConvertEVMTopicHashToAddress uses the last 20 bytes of a common.Hash to create a common.Address
func ConvertEVMTopicHashToAddress(hash *common.Hash) *common.Address {
	if hash == nil {
		return nil
	}
	address := common.BytesToAddress(hash[12:32])
	return &address
}

// EqualFoldContains checks if the array contains the string regardless of casing
// func IsValidERC20Token(whiteList []configuration.Token, address string) bool {
//	for _, token := range whiteList {
//		if strings.EqualFold(token.Address, address) {
//			return true
//		}
//	}
//	return false
// }

func GenerateErc20TransferData(toAddress string, value *big.Int) []byte {
	to := common.HexToAddress(toAddress)
	methodID := getTransferMethodID()

	paddedAddress := common.LeftPadBytes(to.Bytes(), requiredPaddingBytes)
	paddedAmount := common.LeftPadBytes(value.Bytes(), requiredPaddingBytes)

	var data []byte
	data = append(data, methodID...)
	data = append(data, paddedAddress...)
	data = append(data, paddedAmount...)
	return data
}
