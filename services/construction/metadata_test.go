package construction

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/coinbase/rosetta-geth-sdk/client"
	"github.com/coinbase/rosetta-geth-sdk/configuration"
	mockedServices "github.com/coinbase/rosetta-geth-sdk/mocks/services"
	AssetTypes "github.com/coinbase/rosetta-geth-sdk/types"

	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	transferValue         = uint64(1)
	transferContractValue = uint64(0)
	transferGasPrice      = uint64(5000000000)
	transferGasLimit      = uint64(21000)
	transferGasLimitERC20 = uint64(0) // 0 or 21644?
	transferGasLimitContract = uint64(545568)
	transferNonce         = uint64(67)

	suggestedFeeMultiplier float64 = 1

	metadataGenericData   = "0x095ea7b3000000000000000000000000d10a72cf054650931365cc44d912a4fd7525" +
		"705800000000000000000000000000000000000000000000000000000000000003e8"
	tokenContractAddress = "0x2d7882beDcbfDDce29Ba99965dd3cdF7fcB10A1e"

)


func TestMetadata(t *testing.T) {
	testingClient := newTestingClient()
	// if err != nil {
	//	log.Fatalln("%w: unable to load testing configuration", err)
	// }

	tests := map[string]struct {
		options          map[string]interface{}
		mocks func(context.Context, *mockedServices.Client)
		expectedResponse *types.ConstructionMetadataResponse
		expectedError    *types.Error
	}{
		"happy path: Generic contract call metadata": {
			options: map[string]interface{}{
				"from":             testingFromAddress,
				"to":               testingToAddress,
				"value":            transferContractValue,
				"nonce":            transferNonce,
				"contract_address": tokenContractAddress,
				"suggested_fee_multiplier": suggestedFeeMultiplier,
				"data":             metadataGenericData,
				"method_signature": "approve(address,uint256)",
				"method_args":      []string{"0xD10a72Cf054650931365Cc44D912a4FD75257058", "1000"},
			},
			mocks: func(ctx context.Context, client *mockedServices.Client) {
				client.On("GetNonce", ctx, mock.Anything).
					Return(transferNonce, nil)

				client.On("GetGasPrice", ctx, mock.Anything).
					Return(big.NewInt(int64(transferGasPrice)), nil)

				contractData, _ := hexutil.Decode(metadataGenericData)
				client.On("GetContractCallGasLimit", ctx, tokenContractAddress, testingFromAddress, contractData).
					Return(transferGasLimitContract, nil)
			},
			expectedResponse: &types.ConstructionMetadataResponse{
				Metadata: map[string]interface{}{
				//	"to":               tokenContractAddress,
					"nonce":            float64(transferNonce),
					"gas_price":        float64(transferGasPrice),
					"gas_limit":        float64(transferGasLimitContract),
					"data":             metadataGenericData,
					"method_signature": "approve(address,uint256)",
					"method_args":      []interface{}{"0xD10a72Cf054650931365Cc44D912a4FD75257058", "1000"},
				},
				SuggestedFee: []*types.Amount{
					client.Amount(big.NewInt(int64(transferGasPrice) * int64(transferGasLimitContract)),
						testingClient.cfg.RosettaCfg.Currency),
				},
			},
		},
		"happy path: native currency": {
			options: map[string]interface{}{
				"from":  testingFromAddress,
				"to":    testingToAddress,
				"value": transferValue,
				"suggested_fee_multiplier": suggestedFeeMultiplier,
				"currency": map[string]interface{}{
					"decimals": float64(18),
					"symbol": "ETH",
				},
			},
			mocks: func(ctx context.Context, client *mockedServices.Client) {				
				client.On("GetNonce", ctx, mock.Anything).
					Return(transferNonce, nil)
				
				client.On("GetGasPrice", ctx, mock.Anything).	
					Return(big.NewInt(int64(transferGasPrice)), nil)

				client.On("GetNativeTransferGasLimit", ctx, testingToAddress, testingFromAddress, big.NewInt(1)).
					Return(transferGasLimit, nil)
			},
			expectedResponse: &types.ConstructionMetadataResponse{
				Metadata: map[string]interface{}{
					"nonce":     float64(transferNonce),
					"gas_price": float64(transferGasPrice), 
					"gas_limit": float64(transferGasLimit),
				},
				SuggestedFee: []*types.Amount{
						client.Amount(big.NewInt(int64(transferGasPrice) * int64(transferGasLimit)),
							testingClient.cfg.RosettaCfg.Currency),
				},
			},
		},
		"happy path: ERC20 currency": {
			options: map[string]interface{}{
				"from":  testingFromAddress,
				"to":    testingToAddress,
				"value": transferValue,
				"suggested_fee_multiplier": suggestedFeeMultiplier,
				"currency": map[string]interface{}{
					"decimals": float64(6),
					"symbol": "USDC",
					"metadata": map[string]interface{}{
						"contractAddress": "0x1E77ad77925Ac0075CF61Fb76bA35D884985019d",
					},
				},
			},
			mocks: func(ctx context.Context, client *mockedServices.Client) {				
				client.On("GetNonce", ctx, mock.Anything).
					Return(transferNonce, nil)
				
				client.On("GetGasPrice", ctx, mock.Anything).	
					Return(big.NewInt(int64(transferGasPrice)), nil)

				client.On("GetNativeTransferGasLimit", ctx, testingToAddress, testingFromAddress, big.NewInt(1)).
					Return(transferGasLimit, nil)

				client.On("GetErc20TransferGasLimit", ctx, testingToAddress, testingFromAddress, big.NewInt(1), mock.Anything).
					Return(transferGasLimitERC20, nil)
			},
			expectedResponse: &types.ConstructionMetadataResponse{
				Metadata: map[string]interface{}{
					"nonce":     float64(transferNonce),
					"gas_price": float64(transferGasPrice), 
					"gas_limit": float64(transferGasLimitERC20),
				},
				SuggestedFee: []*types.Amount{
						client.Amount(big.NewInt(int64(transferGasPrice) * int64(transferGasLimitERC20)),
							testingClient.cfg.RosettaCfg.Currency),
				},
			},
		},
		"error: missing source address": {
			options: map[string]interface{}{
				"to":    testingToAddress,
				"value": transferValue,
			},
			expectedResponse: nil,
			expectedError: templateError(
				AssetTypes.ErrInvalidInput, "from address is not provided"),
		},
		"error: missing destination address": {
			options: map[string]interface{}{
				"from":  testingFromAddress,
				"value": transferValue,
			},
			expectedResponse: nil,
			expectedError: templateError(
				AssetTypes.ErrInvalidInput, "to address is not provided"),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if test.mocks != nil {
				test.mocks(context.Background(), testingClient.mockClient)
			}

			request := &types.ConstructionMetadataRequest{
				NetworkIdentifier: ethereumNetworkIdentifier,
				Options:           test.options,
			}
			resp, err := testingClient.servicer.ConstructionMetadata(context.Background(), request)

			if err != nil {
				assert.Equal(t, test.expectedError, err)
			} else {
				assert.Equal(t, test.expectedResponse, resp)
			}
		})
	}
	t.Run("unavailable in offline mode", func(t *testing.T) {
		testingClient.cfg.Mode = configuration.ModeOffline
		resp, err := testingClient.servicer.ConstructionMetadata(
			context.Background(),
			&types.ConstructionMetadataRequest{},
		)
		assert.Nil(t, resp)
		assert.Equal(t, AssetTypes.ErrUnavailableOffline.Code, err.Code)
	})
}
