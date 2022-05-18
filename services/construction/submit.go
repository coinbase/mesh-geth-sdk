package construction

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/coinbase/rosetta-geth-sdk/client"
	sdkTypes "github.com/coinbase/rosetta-geth-sdk/types"

	"github.com/coinbase/rosetta-sdk-go/types"
	EthTypes "github.com/ethereum/go-ethereum/core/types"
)

// ConstructionSubmit implements /construction/submit endpoint.
//
// Submit a pre-signed Transaction to the node.
//
func (s *APIService) ConstructionSubmit(
	ctx context.Context,
	req *types.ConstructionSubmitRequest,
) (*types.TransactionIdentifierResponse, *types.Error) {
	if s.config.Mode != sdkTypes.Online {
		return nil, sdkTypes.ErrUnavailableOffline
	}

	if len(req.SignedTransaction) == 0 {
		return nil, sdkTypes.WrapErr(sdkTypes.ErrInvalidInput, fmt.Errorf("signed Transaction value is not provided"))
	}

	var wrappedTx client.SignedTransactionWrapper
	if err := json.Unmarshal([]byte(req.SignedTransaction), &wrappedTx); err != nil {
		return nil, sdkTypes.WrapErr(sdkTypes.ErrInvalidInput, err)
	}

	var signedTx EthTypes.Transaction
	if err := signedTx.UnmarshalJSON(wrappedTx.SignedTransaction); err != nil {
		return nil, sdkTypes.WrapErr(sdkTypes.ErrInvalidInput, err)
	}

	if err := s.client.Submit(ctx, &signedTx); err != nil {
		return nil, sdkTypes.WrapErr(sdkTypes.ErrInternalError, err)
	}

	return &types.TransactionIdentifierResponse{
		TransactionIdentifier: &types.TransactionIdentifier{
			Hash: signedTx.Hash().String(),
		},
	}, nil
}
