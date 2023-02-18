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
	"fmt"
	"math/big"

	sdkTypes "github.com/coinbase/rosetta-geth-sdk/types"

	"github.com/coinbase/rosetta-sdk-go/parser"
	EthTypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/coinbase/rosetta-geth-sdk/client"
	"github.com/coinbase/rosetta-geth-sdk/configuration"

	"github.com/coinbase/rosetta-sdk-go/types"
)

const (
	numOfValidOpsForDescription = 2
)

// APIService implements /construction/* endpoints
type APIService struct {
	config *configuration.Configuration
	types  *sdkTypes.Types
	errors []*types.Error
	client Client
}

// NewAPIService creates a new instance of a APIService.
func NewAPIService(
	cfg *configuration.Configuration,
	types *sdkTypes.Types,
	errors []*types.Error,
	client Client,
) *APIService {
	return &APIService{
		config: cfg,
		types:  types,
		errors: errors,
		client: client,
	}
}

func (s *APIService) CreateOperationDescription(
	operations []*types.Operation,
	isContractCall bool,
) ([]*parser.OperationDescription, error) {
	if len(operations) != numOfValidOpsForDescription {
		return nil, fmt.Errorf("invalid number of operations")
	}

	firstCurrency := operations[0].Amount.Currency
	secondCurrency := operations[1].Amount.Currency

	if firstCurrency == nil || secondCurrency == nil {
		return nil, fmt.Errorf("invalid currency on operation")
	}

	if types.Hash(firstCurrency) != types.Hash(secondCurrency) {
		return nil, fmt.Errorf("currency info doesn't match between the operations")
	}

	if isContractCall {
		const base = 10
		i := new(big.Int)
		i.SetString(operations[0].Amount.Value, base)
		j := new(big.Int)
		j.SetString(operations[1].Amount.Value, base)

		if i.Cmp(big.NewInt(0)) == 0 {
			if j.Cmp(big.NewInt(0)) != 0 {
				return nil, fmt.Errorf("for generic call both values should be zero")
			}
		}

		return s.CreateOperationDescriptionContractCall(), nil
	}

	if types.Hash(firstCurrency) == types.Hash(s.config.RosettaCfg.Currency) {
		return s.CreateOperationDescriptionNative(), nil
	}
	firstContract, firstOk := firstCurrency.Metadata[client.ContractAddressMetadata].(string)
	_, secondOk := secondCurrency.Metadata[client.ContractAddressMetadata].(string)

	// Not Native curr
	if !firstOk || !secondOk {
		return nil, fmt.Errorf("non-native currency must have contractAddress in Metadata")
	}

	// ERC20 logic
	return s.CreateOperationDescriptionERC20(firstContract, firstCurrency), nil
}

func (s *APIService) CreateOperationDescriptionContractCall() []*parser.OperationDescription {
	var descriptions []*parser.OperationDescription

	nativeSend := parser.OperationDescription{
		Type: sdkTypes.CallOpType,
		Account: &parser.AccountDescription{
			Exists: true,
		},
		Amount: &parser.AmountDescription{
			Exists:   true,
			Sign:     parser.AnyAmountSign,
			Currency: s.config.RosettaCfg.Currency,
		},
	}
	nativeReceive := parser.OperationDescription{
		Type: sdkTypes.CallOpType,
		Account: &parser.AccountDescription{
			Exists: true,
		},
		Amount: &parser.AmountDescription{
			Exists:   true,
			Sign:     parser.AnyAmountSign,
			Currency: s.config.RosettaCfg.Currency,
		},
	}

	descriptions = append(descriptions, &nativeSend)
	descriptions = append(descriptions, &nativeReceive)
	return descriptions
}

func (s *APIService) CreateOperationDescriptionNative() []*parser.OperationDescription {
	var descriptions []*parser.OperationDescription

	nativeSend := parser.OperationDescription{
		Type: sdkTypes.CallOpType,
		Account: &parser.AccountDescription{
			Exists: true,
		},
		Amount: &parser.AmountDescription{
			Exists:   true,
			Sign:     parser.NegativeAmountSign,
			Currency: s.config.RosettaCfg.Currency,
		},
	}
	nativeReceive := parser.OperationDescription{
		Type: sdkTypes.CallOpType,
		Account: &parser.AccountDescription{
			Exists: true,
		},
		Amount: &parser.AmountDescription{
			Exists:   true,
			Sign:     parser.PositiveAmountSign,
			Currency: s.config.RosettaCfg.Currency,
		},
	}

	descriptions = append(descriptions, &nativeSend)
	descriptions = append(descriptions, &nativeReceive)
	return descriptions
}

func (s *APIService) CreateOperationDescriptionERC20(
	contractAddress string, currencyInfo *types.Currency,
) []*parser.OperationDescription {
	var descriptions []*parser.OperationDescription
	currency := client.Erc20Currency(currencyInfo.Symbol, currencyInfo.Decimals, contractAddress)

	send := parser.OperationDescription{
		Type: sdkTypes.OpErc20Transfer,
		Account: &parser.AccountDescription{
			Exists: true,
		},
		Amount: &parser.AmountDescription{
			Exists:   true,
			Sign:     parser.NegativeAmountSign,
			Currency: currency,
		},
	}
	receive := parser.OperationDescription{
		Type: sdkTypes.OpErc20Transfer,
		Account: &parser.AccountDescription{
			Exists: true,
		},
		Amount: &parser.AmountDescription{
			Exists:   true,
			Sign:     parser.PositiveAmountSign,
			Currency: currency,
		},
	}

	descriptions = append(descriptions, &send)
	descriptions = append(descriptions, &receive)
	return descriptions
}

// ConstructionHash implements /construction/hash endpoint.
//
// TransactionHash returns the network-specific Transaction hash for a signed Transaction.
//
func (s *APIService) ConstructionHash(
	ctx context.Context,
	req *types.ConstructionHashRequest,
) (*types.TransactionIdentifierResponse, *types.Error) {
	if len(req.SignedTransaction) == 0 {
		return nil, sdkTypes.WrapErr(
			sdkTypes.ErrInvalidInput,
			fmt.Errorf("signed Transaction value is not provided"),
		)
	}

	var wrappedTx client.SignedTransactionWrapper
	if err := json.Unmarshal([]byte(req.SignedTransaction), &wrappedTx); err != nil {
		return nil, sdkTypes.WrapErr(sdkTypes.ErrInvalidInput, err)
	}

	var signedTx EthTypes.Transaction
	if err := signedTx.UnmarshalJSON(wrappedTx.SignedTransaction); err != nil {
		return nil, sdkTypes.WrapErr(sdkTypes.ErrInvalidInput, err)
	}

	return &types.TransactionIdentifierResponse{
		TransactionIdentifier: &types.TransactionIdentifier{
			Hash: signedTx.Hash().Hex(),
		},
	}, nil
}
