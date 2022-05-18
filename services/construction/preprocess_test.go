package construction

import (
	"context"
	"testing"

	AssetTypes "github.com/coinbase/rosetta-geth-sdk/types"
	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/stretchr/testify/assert"
)

var (
	// preprocessNetworkIdentifier = &types.NetworkIdentifier{
	// 	Network:    "Ropsten",
	// 	Blockchain: "Ethereum",
	// }

	preprocessTransferValue     = uint64(1)
	methodSignature             = "approve(address,uint256)"
	methodArgs                  = []string{"0xD10a72Cf054650931365Cc44D912a4FD75257058", "1000"}
	expectedMethodArgs          = []interface{}{"0xD10a72Cf054650931365Cc44D912a4FD75257058", "1000"}
	preprocessZeroTransferValue = uint64(0)
	// preprocessTransferValueHex     = hexutil.EncodeUint64(preprocessTransferValue)
	// preprocessGasPrice             = uint64(100000000000)
	// preprocessGasPriceHex          = hexutil.EncodeUint64(preprocessGasPrice)
	preprocessGenericData = "0x095ea7b3000000000000000000000000d10a72cf054650931365cc44d912a4fd7525705" +
		"800000000000000000000000000000000000000000000000000000000000003e8"
)

func TestConstructionPreprocess(t *testing.T) {
	testingClient := newTestingClient()
	// if err != nil {
	//	log.Fatalln("%w: unable to load testing configuration", err)
	// }

	tests := map[string]struct {
		operations []*types.Operation
		metadata   map[string]interface{}

		expectedResponse *types.ConstructionPreprocessResponse
		expectedError    *types.Error
	}{
		"happy path: Generic Contract call": {
			operations: templateOperations(preprocessTransferValue, ethereumCurrencyConfig, "CALL"),
			metadata: map[string]interface{}{
				"method_signature": methodSignature,
				"method_args":      methodArgs,
			},
			expectedResponse: &types.ConstructionPreprocessResponse{
				Options: map[string]interface{}{
					"from":             testingFromAddress,
					"to":               testingToAddress, // it will be contract address user need to pass in operation
					"value":            float64(preprocessTransferValue),
					"contract_address": testingToAddress,
					"data":             preprocessGenericData,
					"method_signature": methodSignature,
					"method_args":      expectedMethodArgs,
					"currency": map[string]interface{}{
						"decimals": float64(18),
						"symbol":   "ETH",
					},
				},
			},
		},
		"happy path: Generic Contract call with zero transfer value": {
			operations: templateOperations(preprocessZeroTransferValue, ethereumCurrencyConfig, "CALL"),
			metadata: map[string]interface{}{
				"method_signature": "approve(address,uint256)",
				"method_args":      []string{"0xD10a72Cf054650931365Cc44D912a4FD75257058", "1000"},
			},
			expectedResponse: &types.ConstructionPreprocessResponse{
				Options: map[string]interface{}{
					"from":             testingFromAddress,
					"to":               testingToAddress, // it will be contract address user need to pass in operation
					"value":            float64(preprocessZeroTransferValue),
					"contract_address": testingToAddress,
					"data":             preprocessGenericData,
					"method_signature": methodSignature,
					"method_args":      expectedMethodArgs,
					"currency": map[string]interface{}{
						"decimals": float64(18),
						"symbol":   "ETH",
					},
				},
			},
		},
		"happy path: native currency": {
			operations: templateOperations(preprocessTransferValue, ethereumCurrencyConfig, "CALL"),
			expectedResponse: &types.ConstructionPreprocessResponse{
				Options: map[string]interface{}{
					"from":  testingFromAddress,
					"to":    testingToAddress,
					"value": float64(preprocessTransferValue),
					"currency": map[string]interface{}{
						"decimals": float64(18),
						"symbol":   "ETH",
					},
				},
			},
		},
		"happy path: ERC20 currency": {
			operations: templateOperations(preprocessTransferValue, &types.Currency{
				Symbol:   "USDC",
				Decimals: 6,
				Metadata: map[string]interface{}{
					"contractAddress": "0x1E77ad77925Ac0075CF61Fb76bA35D884985019d",
				},
			}, "ERC20_TRANSFER"),
			expectedResponse: &types.ConstructionPreprocessResponse{
				Options: map[string]interface{}{
					"from":  testingFromAddress,
					"to":    testingToAddress,
					"value": float64(preprocessTransferValue),
					"currency": map[string]interface{}{
						"decimals": float64(6),
						"symbol":   "USDC",
						"metadata": map[string]interface{}{
							"contractAddress": "0x1E77ad77925Ac0075CF61Fb76bA35D884985019d",
						},
					},
				},
			},
		},
		"error: both positive amount": {
			operations: func() []*types.Operation {
				operations := templateOperations(preprocessTransferValue, ethereumCurrencyConfig, "CALL")
				operations[0].Amount.Value = "1"
				return operations
			}(),
			expectedResponse: nil,
			expectedError:    templateError(AssetTypes.ErrInvalidInput, "unable to find match for operation: at index 1"),
		},
		"error: missing currency": {
			operations: func() []*types.Operation {
				operations := templateOperations(preprocessTransferValue, ethereumCurrencyConfig, "CALL")
				operations[0].Amount.Currency = nil
				return operations
			}(),
			expectedResponse: nil,
			expectedError: templateError(
				AssetTypes.ErrInvalidInput, "invalid currency on operation"),
		},
		"error: unequal currency": {
			operations: func() []*types.Operation {
				operations := templateOperations(preprocessTransferValue, ethereumCurrencyConfig, "CALL")
				operations[0].Amount.Currency = &types.Currency{
					Symbol:   "USDC",
					Decimals: 18,
				}
				return operations
			}(),
			expectedResponse: nil,
			expectedError: templateError(
				AssetTypes.ErrInvalidInput, "currency info doesn't match between the operations"),
		},
		"error: invalid from address": {
			operations: func() []*types.Operation {
				operations := templateOperations(preprocessTransferValue, ethereumCurrencyConfig, "CALL")
				operations[0].Account.Address = "invalid"
				return operations
			}(),
			expectedResponse: nil,
			expectedError: templateError(
				AssetTypes.ErrInvalidAddress, "invalid is not a valid address"),
		},
		"error: invalid destination address": {
			operations: func() []*types.Operation {
				operations := templateOperations(preprocessTransferValue, ethereumCurrencyConfig, "CALL")
				operations[1].Account.Address = "invalid"
				return operations
			}(),
			expectedResponse: nil,
			expectedError: templateError(
				AssetTypes.ErrInvalidAddress, "invalid is not a valid address"),
		},
		"error: missing token address": {
			operations: templateOperations(preprocessTransferValue, &types.Currency{
				Symbol:   "USDC",
				Decimals: 6,
			}, "ERC20_TRANSFER"),
			expectedResponse: nil,
			expectedError: templateError(
				AssetTypes.ErrInvalidInput, "non-native currency must have contractAddress in Metadata"),
		},
		"error: token address not a string": {
			operations: templateOperations(preprocessTransferValue, &types.Currency{
				Symbol:   "USDC",
				Decimals: 6,
				Metadata: map[string]interface{}{
					"contractAddress": map[string]string{},
				},
			}, "ERC20_TRANSFER"),
			expectedResponse: nil,
			expectedError: templateError(
				AssetTypes.ErrInvalidInput, "non-native currency must have contractAddress in Metadata"),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			request := &types.ConstructionPreprocessRequest{
				NetworkIdentifier: ethereumNetworkIdentifier,
				Operations:        test.operations,
				Metadata:          test.metadata,
			}
			resp, err := testingClient.servicer.ConstructionPreprocess(context.Background(), request)

			if err != nil {
				assert.Equal(t, test.expectedError, err)
			} else {
				assert.Equal(t, test.expectedResponse, resp)
			}
		})
	}
}
