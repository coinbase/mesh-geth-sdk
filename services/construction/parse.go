package construction

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/coinbase/rosetta-geth-sdk/client"
	sdkTypes "github.com/coinbase/rosetta-geth-sdk/types"

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
	// var sender common.Address

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

		msg, err := t.AsMessage(EthTypes.NewEIP155Signer(t.ChainId()), nil)
		if err != nil {
			return nil, sdkTypes.WrapErr(sdkTypes.ErrUnableToParseIntermediateResult, err)
		}
		// sender = msg.From()
		tx.From = msg.From().Hex()
	}

	//TODO: add logic for contract call parsing
	var opMethod string
	var value *big.Int
	var toAddressHex string
	// Erc20 transfer
	if len(tx.Data) != 0 && hasERC20TransferData(tx.Data) {
		toAddress, amountSent, err := parseErc20TransferData(tx.Data)
		if err != nil {
			return nil, sdkTypes.WrapErr(sdkTypes.ErrInvalidInput, err)
		}

		value = amountSent
		opMethod = sdkTypes.OpErc20Transfer
		toAddressHex = toAddress.Hex()
	} else {
		value = tx.Value
		opMethod = sdkTypes.CallOpType
		toAddressHex = tx.To
	}

	// Ensure valid from address
	checkFrom, ok := client.ChecksumAddress(tx.From)
	if !ok {
		return nil, sdkTypes.WrapErr(sdkTypes.ErrInvalidAddress, fmt.Errorf("%s is not a valid address", tx.From))
	}

	// Ensure valid to address
	checkTo, ok := client.ChecksumAddress(toAddressHex)
	if !ok {
		return nil, sdkTypes.WrapErr(sdkTypes.ErrInvalidAddress, fmt.Errorf("%s is not a valid address", tx.To))
	}

	ops := []*types.Operation{
		{
			Type: opMethod,
			OperationIdentifier: &types.OperationIdentifier{
				Index: 0,
			},
			Account: &types.AccountIdentifier{
				Address: checkFrom,
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
				Address: checkTo,
			},
			Amount: &types.Amount{
				Value:    value.String(),
				Currency: tx.Currency,
			},
		},
	}

	// var gasPrice, _ = new(big.Int).SetString(tx.GasPrice.String(), 10) // nolint:gomnd
	// gasUsed := tx.GasLimit * gasPrice.Uint64()

	// txFee := new(big.Int).SetUint64(gasUsed)
	// txFee = txFee.Mul(txFee, tx.GasPrice)

	// feeOps := []*types.Operation{
	// 	{
	// 		OperationIdentifier: &types.OperationIdentifier{
	// 			Index: 2, // nolint:gomnd
	// 		},
	// 		Type: sdkTypes.FeeOpType,
	// 		// Status:  types.String(sdkTypes.SuccessStatus),
	// 		Account: client.Account(&sender),
	// 		Amount:  client.Amount(new(big.Int).Neg(txFee), tx.Currency),
	// 	},
	// }

	// ops = append(ops, feeOps...)

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
					Address: checkFrom,
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
	transferFnSignature := []byte("transfer(address,uint256)")
	hash := sha3.NewLegacyKeccak256()
	if _, err := hash.Write(transferFnSignature); err != nil {
		return nil, err
	}

	return hash.Sum(nil)[:4], nil
}

func parseErc20TransferData(data []byte) (*common.Address, *big.Int, error) {
	if len(data) != client.GenericTransferBytesLength {
		return nil, nil, fmt.Errorf("incorrect length for data array")
	}
	methodID := getTransferMethodID()
	if hexutil.Encode(data[:4]) != hexutil.Encode(methodID) {
		return nil, nil, fmt.Errorf("incorrect methodID signature")
	}

	address := common.BytesToAddress(data[5:36])
	amount := new(big.Int).SetBytes(data[37:])
	return &address, amount, nil
}

func getTransferMethodID() []byte {
	transferSignature := []byte(client.TransferFnSignature) // do not include spaces in the string
	hash := sha3.NewLegacyKeccak256()
	hash.Write(transferSignature)
	methodID := hash.Sum(nil)[:4]
	return methodID
}

func hasERC20TransferData(data []byte) bool {
	methodID := data[:4]
	expectedMethodID, _ := erc20TransferMethodID()
	res := bytes.Compare(methodID, expectedMethodID)
	return res == 0
}
