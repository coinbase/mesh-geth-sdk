package construction

import (
	"context"

	sdkTypes "github.com/coinbase/rosetta-geth-sdk/types"

	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/ethereum/go-ethereum/crypto"
)

// ConstructionDerive implements the /construction/derive endpoint.
func (s *APIService) ConstructionDerive(
	ctx context.Context, req *types.ConstructionDeriveRequest,
) (*types.ConstructionDeriveResponse, *types.Error) {
	if req.PublicKey == nil {
		return nil, sdkTypes.ErrInvalidInput
	}

	key, err := crypto.DecompressPubkey(req.PublicKey.Bytes) // hex_bytes
	if err != nil {
		return nil, sdkTypes.ErrInvalidInput
	}

	return &types.ConstructionDeriveResponse{
		AccountIdentifier: &types.AccountIdentifier{
			Address: crypto.PubkeyToAddress(*key).Hex(),
		},
	}, nil
}
