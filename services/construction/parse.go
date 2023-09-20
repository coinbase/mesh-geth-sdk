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
	"context"
	"encoding/json"
	"fmt"
	"math/big"

	"errors"
	"github.com/coinbase/rosetta-geth-sdk/client"
	sdkTypes "github.com/coinbase/rosetta-geth-sdk/types"
	goEthTypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	EthTypes "github.com/ethereum/go-ethereum/core/types"
	"golang.org/x/crypto/sha3"
)

// ConstructionParse implements the /construction/parse endpoint.
func (s *APIService) ConstructionParse(
	ctx context.Context,
	request *types.ConstructionParseRequest,
) (*types.ConstructionParseResponse, *types.Error) {
	var tx client.Transaction

	if !request.Signed {
		err := json.Unmarshal([]byte(request.Transaction), &tx)
		if err != nil {
			return nil, sdkTypes.WrapErr(sdkTypes.ErrUnableToParseIntermediateResult, err)
		}
	} else {
		var wrappedTx client.SignedTransactionWrapper
		if err := json.Unmarshal([]byte(request.Transaction), &wrappedTx); err != nil {
			return nil, sdkTypes.WrapErr(sdkTypes.ErrUnableToParseIntermediateResult, err)
		}

		var t EthTypes.Transaction
		if err := t.UnmarshalJSON(wrappedTx.SignedTransaction); err != nil {
			return nil, sdkTypes.WrapErr(sdkTypes.ErrInvalidInput, err)
		}

		tx.To = t.To().String()
		tx.Value = t.Value()
		tx.Data = t.Data()
		tx.Nonce = t.Nonce()
		tx.GasPrice = t.GasPrice()
		tx.GasLimit = t.Gas()
		tx.ChainID = t.ChainId()
		tx.Currency = wrappedTx.Currency

		from, err := goEthTypes.Sender(EthTypes.NewEIP155Signer(t.ChainId()), &t)
		if err != nil {
			return nil, sdkTypes.WrapErr(sdkTypes.ErrUnableToParseIntermediateResult, err)
		}
		tx.From = from.Hex()
	}

	//TODO: add logic for contract call parsing

	value := tx.Value
	opMethod := sdkTypes.CallOpType
	fromAddress := tx.From
	toAddress := tx.To

	// ERC20 transfer
	if len(tx.Data) != 0 && hasERC20TransferData(tx.Data) {
		address, amountSent, err := parseErc20TransferData(tx.Data)
		if err != nil {
			return nil, sdkTypes.WrapErr(sdkTypes.ErrInvalidInput, err)
		}

		value = amountSent
		opMethod = sdkTypes.OpErc20Transfer
		toAddress = address.Hex()
	}

	// Address validation
	from, err := client.ChecksumAddress(fromAddress)
	if err != nil {
		return nil, sdkTypes.WrapErr(sdkTypes.ErrInvalidAddress, fmt.Errorf("%s is not a valid address: %w", tx.From, err))
	}
	to, err := client.ChecksumAddress(toAddress)
	if err != nil {
		return nil, sdkTypes.WrapErr(sdkTypes.ErrInvalidAddress, fmt.Errorf("%s is not a valid address: %w", tx.To, err))
	}

	ops := []*types.Operation{
		{
			Type: opMethod,
			OperationIdentifier: &types.OperationIdentifier{
				Index: 0,
			},
			Account: &types.AccountIdentifier{
				Address: from,
			},
			Amount: &types.Amount{
				Value:    new(big.Int).Neg(value).String(),
				Currency: tx.Currency,
			},
		},
		{
			Type: opMethod,
			OperationIdentifier: &types.OperationIdentifier{
				Index: 1,
			},
			Account: &types.AccountIdentifier{
				Address: to,
			},
			Amount: &types.Amount{
				Value:    value.String(),
				Currency: tx.Currency,
			},
		},
	}

	metadata := &client.ParseMetadata{
		Nonce:    tx.Nonce,
		GasPrice: tx.GasPrice,
		ChainID:  tx.ChainID,
	}
	metaMap, err := client.MarshalJSONMap(metadata)
	if err != nil {
		return nil, sdkTypes.WrapErr(sdkTypes.ErrUnableToParseIntermediateResult, err)
	}

	var resp *types.ConstructionParseResponse
	if request.Signed {
		resp = &types.ConstructionParseResponse{
			Operations: ops,
			AccountIdentifierSigners: []*types.AccountIdentifier{
				{
					Address: fromAddress,
				},
			},
			Metadata: metaMap,
		}
	} else {
		resp = &types.ConstructionParseResponse{
			Operations:               ops,
			AccountIdentifierSigners: []*types.AccountIdentifier{},
			Metadata:                 metaMap,
		}
	}
	return resp, nil
}

// erc20TransferMethodID calculates the first 4 bytes of the method
// signature for transfer on an ERC20 contract
func erc20TransferMethodID() ([]byte, error) {
	transferFnSignature := []byte(client.TransferFnSignature)
	hash := sha3.NewLegacyKeccak256()
	if _, err := hash.Write(transferFnSignature); err != nil {
		return nil, err
	}

	return hash.Sum(nil)[:4], nil
}

func parseErc20TransferData(data []byte) (*common.Address, *big.Int, error) {
	if len(data) != client.GenericTransferBytesLength {
		return nil, nil, errors.New("incorrect length for data array")
	}

	methodID, _ := erc20TransferMethodID()
	if hexutil.Encode(data[:4]) != hexutil.Encode(methodID) {
		return nil, nil, errors.New("incorrect methodID signature")
	}

	address := common.BytesToAddress(data[5:36])
	amount := new(big.Int).SetBytes(data[37:])

	return &address, amount, nil
}

func hasERC20TransferData(data []byte) bool {
	methodID := data[:4]
	expectedMethodID, _ := erc20TransferMethodID()

	return bytes.Equal(methodID, expectedMethodID)
}
