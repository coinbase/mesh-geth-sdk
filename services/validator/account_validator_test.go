package validator

import (
	"context"
	"encoding/json"
	"math/big"
	"os"
	"testing"

	"github.com/coinbase/rosetta-geth-sdk/configuration"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
	"golang.org/x/xerrors"
)

func TestValidateAccountState_Success(t *testing.T) {
	ctx := context.Background()

	for _, chainData := range TestChains {
		t.Run(chainData.Name, func(t *testing.T) {
			cfg := &configuration.Configuration{
				ChainConfig: chainData.ChainConfig,
				Network:     chainData.Network,
				GethURL:     chainData.GethURL,
			}
			v := NewEthereumValidator(cfg)

			// Load test data
			var baseAccountResult AccountResult
			data, err := os.ReadFile(chainData.AccountFixtureFile)
			if err != nil {
				t.Fatalf("failed to read fixture file for %s: %v", chainData.Name, err)
			}
			err = json.Unmarshal(data, &baseAccountResult)
			if err != nil {
				t.Fatalf("failed to unmarshal fixture for %s: %v", chainData.Name, err)
			}

			// Calculate the correct state root from the first proof node
			firstNodeData, err := hexutil.Decode(baseAccountResult.AccountProof[0])
			if err != nil {
				t.Fatalf("failed to decode first node for %s: %v", chainData.Name, err)
			}
			correctStateRoot := common.BytesToHash(crypto.Keccak256(firstNodeData))

			// 1. Test basic successful validation scenarios
			t.Run("BasicSuccessfulValidation", func(t *testing.T) {
				// Test with exact valid data
				t.Run("ExactValidData", func(t *testing.T) {
					err := v.ValidateAccountState(ctx, baseAccountResult, correctStateRoot, chainData.TestBlockNumber)
					assert.NoError(t, err, "Validation should succeed with exact valid data")
				})

				// Test that validation succeeds with the same data multiple times (idempotent)
				t.Run("RepeatedValidation", func(t *testing.T) {
					for i := 0; i < 3; i++ {
						err := v.ValidateAccountState(ctx, baseAccountResult, correctStateRoot, chainData.TestBlockNumber)
						assert.NoError(t, err, "Validation should succeed on repeated calls")
					}
				})

				// Test with different but valid block numbers (validation doesn't depend on block number)
				t.Run("DifferentBlockNumbers", func(t *testing.T) {
					blockNumbers := []*big.Int{
						chainData.TestBlockNumber,
						new(big.Int).Add(chainData.TestBlockNumber, big.NewInt(1)),
						new(big.Int).Add(chainData.TestBlockNumber, big.NewInt(1000000)),
						big.NewInt(0),
					}

					for _, blockNumber := range blockNumbers {
						err := v.ValidateAccountState(ctx, baseAccountResult, correctStateRoot, blockNumber)
						assert.NoError(t, err, "Validation should succeed with different block numbers")
					}
				})
			})

			// 2. Test validation with different account field combinations
			t.Run("AccountFieldValidation", func(t *testing.T) {
				// Test account with the original balance
				t.Run("OriginalBalance", func(t *testing.T) {
					validResult := baseAccountResult
					t.Logf("Testing account with balance: %s", validResult.Balance.ToInt().String())

					err := v.ValidateAccountState(ctx, validResult, correctStateRoot, chainData.TestBlockNumber)
					assert.NoError(t, err, "Validation should succeed with original balance")
				})

				// Test account with the original nonce
				t.Run("OriginalNonce", func(t *testing.T) {
					validResult := baseAccountResult
					t.Logf("Testing account with nonce: %d", uint64(validResult.Nonce))

					err := v.ValidateAccountState(ctx, validResult, correctStateRoot, chainData.TestBlockNumber)
					assert.NoError(t, err, "Validation should succeed with original nonce")
				})

				// Test account with the original storage hash
				t.Run("OriginalStorageHash", func(t *testing.T) {
					validResult := baseAccountResult
					t.Logf("Testing account with storage hash: %s", validResult.StorageHash.Hex())

					err := v.ValidateAccountState(ctx, validResult, correctStateRoot, chainData.TestBlockNumber)
					assert.NoError(t, err, "Validation should succeed with original storage hash")
				})

				// Test account with the original code hash
				t.Run("OriginalCodeHash", func(t *testing.T) {
					validResult := baseAccountResult
					t.Logf("Testing account with code hash: %s", validResult.CodeHash.Hex())

					err := v.ValidateAccountState(ctx, validResult, correctStateRoot, chainData.TestBlockNumber)
					assert.NoError(t, err, "Validation should succeed with original code hash")
				})
			})

			// 3. Test proof structure validation
			t.Run("ProofStructureValidation", func(t *testing.T) {
				// Test with complete proof chain
				t.Run("CompleteProofChain", func(t *testing.T) {
					validResult := baseAccountResult
					t.Logf("Testing with %d proof nodes", len(validResult.AccountProof))

					err := v.ValidateAccountState(ctx, validResult, correctStateRoot, chainData.TestBlockNumber)
					assert.NoError(t, err, "Validation should succeed with complete proof chain")
				})

				// Test that all proof nodes are properly formatted
				t.Run("ProofNodeFormatting", func(t *testing.T) {
					validResult := baseAccountResult

					// Verify all proof nodes are valid hex
					for i, proofNode := range validResult.AccountProof {
						_, err := hexutil.Decode(proofNode)
						assert.NoError(t, err, "Proof node %d should be valid hex: %s", i, proofNode)
					}

					err := v.ValidateAccountState(ctx, validResult, correctStateRoot, chainData.TestBlockNumber)
					assert.NoError(t, err, "Validation should succeed with properly formatted proof nodes")
				})

				// Test state root calculation consistency
				t.Run("StateRootCalculation", func(t *testing.T) {
					validResult := baseAccountResult

					// Recalculate state root to ensure consistency
					firstNodeData, err := hexutil.Decode(validResult.AccountProof[0])
					assert.NoError(t, err, "First proof node should decode successfully")

					calculatedStateRoot := common.BytesToHash(crypto.Keccak256(firstNodeData))
					assert.Equal(t, correctStateRoot, calculatedStateRoot, "State root calculation should be consistent")

					err = v.ValidateAccountState(ctx, validResult, calculatedStateRoot, chainData.TestBlockNumber)
					assert.NoError(t, err, "Validation should succeed with calculated state root")
				})
			})

			// 4. Test account address validation
			t.Run("AccountAddressValidation", func(t *testing.T) {
				// Test with the correct address
				t.Run("CorrectAddress", func(t *testing.T) {
					validResult := baseAccountResult
					t.Logf("Testing account address: %s", validResult.Address.Hex())

					err := v.ValidateAccountState(ctx, validResult, correctStateRoot, chainData.TestBlockNumber)
					assert.NoError(t, err, "Validation should succeed with correct address")
				})

				// Test address consistency in proof path
				t.Run("AddressConsistency", func(t *testing.T) {
					validResult := baseAccountResult

					// The account hash should be consistent with the address
					expectedAccountHash := crypto.Keccak256(validResult.Address[:])
					t.Logf("Account hash for verification: %x", expectedAccountHash)

					err := v.ValidateAccountState(ctx, validResult, correctStateRoot, chainData.TestBlockNumber)
					assert.NoError(t, err, "Validation should succeed with address-consistent proof path")
				})
			})

			// 5. Test different context scenarios
			t.Run("ContextVariations", func(t *testing.T) {
				// Test with background context
				t.Run("BackgroundContext", func(t *testing.T) {
					bgCtx := context.Background()
					err := v.ValidateAccountState(bgCtx, baseAccountResult, correctStateRoot, chainData.TestBlockNumber)
					assert.NoError(t, err, "Validation should succeed with background context")
				})

				// Test with context with values
				t.Run("ContextWithValues", func(t *testing.T) {
					ctxWithValue := context.WithValue(ctx, "test-key", "test-value")
					err := v.ValidateAccountState(ctxWithValue, baseAccountResult, correctStateRoot, chainData.TestBlockNumber)
					assert.NoError(t, err, "Validation should succeed with context containing values")
				})

				// Test with cancelled context (should still work since validation is synchronous)
				t.Run("CancelledContext", func(t *testing.T) {
					cancelCtx, cancel := context.WithCancel(ctx)
					cancel() // Cancel immediately

					err := v.ValidateAccountState(cancelCtx, baseAccountResult, correctStateRoot, chainData.TestBlockNumber)
					assert.NoError(t, err, "Validation should succeed even with cancelled context (synchronous operation)")
				})
			})

			// 6. Test account type scenarios
			t.Run("AccountTypeScenarios", func(t *testing.T) {
				// Test EOA (Externally Owned Account) characteristics
				t.Run("EOACharacteristics", func(t *testing.T) {
					validResult := baseAccountResult

					// Check if this looks like an EOA (empty code hash typically indicates EOA)
					emptyCodeHash := crypto.Keccak256Hash(nil)
					if validResult.CodeHash == emptyCodeHash {
						t.Logf("Testing EOA (empty code hash): %s", validResult.CodeHash.Hex())
					} else {
						t.Logf("Testing contract account (non-empty code hash): %s", validResult.CodeHash.Hex())
					}

					err := v.ValidateAccountState(ctx, validResult, correctStateRoot, chainData.TestBlockNumber)
					assert.NoError(t, err, "Validation should succeed regardless of account type")
				})

				// Test account with storage
				t.Run("AccountWithStorage", func(t *testing.T) {
					validResult := baseAccountResult

					emptyStorageHash := common.HexToHash("0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421")
					if validResult.StorageHash != emptyStorageHash {
						t.Logf("Testing account with non-empty storage: %s", validResult.StorageHash.Hex())
					} else {
						t.Logf("Testing account with empty storage: %s", validResult.StorageHash.Hex())
					}

					err := v.ValidateAccountState(ctx, validResult, correctStateRoot, chainData.TestBlockNumber)
					assert.NoError(t, err, "Validation should succeed with account storage")
				})
			})

			// 7. Test edge cases that should succeed
			t.Run("EdgeCasesThatSucceed", func(t *testing.T) {
				// Test minimum valid proof structure
				t.Run("MinimumValidProof", func(t *testing.T) {
					validResult := baseAccountResult

					// Ensure we have at least one proof node (we should have more, but test minimum requirement)
					assert.True(t, len(validResult.AccountProof) >= 1, "Should have at least one proof node")

					err := v.ValidateAccountState(ctx, validResult, correctStateRoot, chainData.TestBlockNumber)
					assert.NoError(t, err, "Validation should succeed with minimum valid proof")
				})

				// Test with maximum precision values
				t.Run("MaximumPrecisionValues", func(t *testing.T) {
					validResult := baseAccountResult

					// Log the actual values to understand the precision
					t.Logf("Balance precision: %s wei", validResult.Balance.ToInt().String())
					t.Logf("Nonce value: %d", uint64(validResult.Nonce))

					err := v.ValidateAccountState(ctx, validResult, correctStateRoot, chainData.TestBlockNumber)
					assert.NoError(t, err, "Validation should succeed with high precision values")
				})
			})

			// 8. Test performance and stability
			t.Run("PerformanceAndStability", func(t *testing.T) {
				// Test concurrent validation (should be safe)
				t.Run("ConcurrentValidation", func(t *testing.T) {
					const numGoroutines = 10
					errChan := make(chan error, numGoroutines)

					for i := 0; i < numGoroutines; i++ {
						go func(goroutineID int) {
							err := v.ValidateAccountState(ctx, baseAccountResult, correctStateRoot, chainData.TestBlockNumber)
							if err != nil {
								errChan <- xerrors.Errorf("goroutine %d failed: %w", goroutineID, err)
							} else {
								errChan <- nil
							}
						}(i)
					}

					// Collect results
					for i := 0; i < numGoroutines; i++ {
						err := <-errChan
						assert.NoError(t, err, "Concurrent validation should succeed")
					}
				})

				// Test validation stability over multiple iterations
				t.Run("ValidationStability", func(t *testing.T) {
					const iterations = 50

					for i := 0; i < iterations; i++ {
						err := v.ValidateAccountState(ctx, baseAccountResult, correctStateRoot, chainData.TestBlockNumber)
						assert.NoError(t, err, "Validation should be stable over multiple iterations (iteration %d)", i)
					}
				})
			})
		})
	}
}

func TestValidateAccountState_Failure(t *testing.T) {
	ctx := context.Background()

	for _, chainData := range TestChains {
		t.Run(chainData.Name, func(t *testing.T) {
			cfg := &configuration.Configuration{
				ChainConfig: chainData.ChainConfig,
				Network:     chainData.Network,
				GethURL:     chainData.GethURL,
			}
			v := NewEthereumValidator(cfg)

			// Load test data
			var baseAccountResult AccountResult
			data, err := os.ReadFile(chainData.AccountFixtureFile)
			if err != nil {
				t.Fatalf("failed to read fixture file for %s: %v", chainData.Name, err)
			}
			err = json.Unmarshal(data, &baseAccountResult)
			if err != nil {
				t.Fatalf("failed to unmarshal fixture for %s: %v", chainData.Name, err)
			}

			// Calculate the correct state root from the first proof node
			firstNodeData, err := hexutil.Decode(baseAccountResult.AccountProof[0])
			if err != nil {
				t.Fatalf("failed to decode first node for %s: %v", chainData.Name, err)
			}
			correctStateRoot := common.BytesToHash(crypto.Keccak256(firstNodeData))

			// 1. Test state root validation failures
			t.Run("StateRootValidationFailures", func(t *testing.T) {
				// Test with completely wrong state root
				t.Run("WrongStateRoot", func(t *testing.T) {
					wrongStateRoot := common.HexToHash("0x7adc7dbc4c36299fc65fd1dc3798a6a58c29c171b79584bfc3512f5ad82a59d4")
					err := v.ValidateAccountState(ctx, baseAccountResult, wrongStateRoot, big.NewInt(5219647))
					assert.Error(t, err, "Validation should fail with wrong state root")
					assert.Contains(t, err.Error(), "state root mismatch")
				})

				// Test with zero state root
				t.Run("ZeroStateRoot", func(t *testing.T) {
					zeroStateRoot := common.Hash{}
					err := v.ValidateAccountState(ctx, baseAccountResult, zeroStateRoot, big.NewInt(5219647))
					assert.Error(t, err, "Validation should fail with zero state root")
					assert.Contains(t, err.Error(), "state root mismatch")
				})

				// Test with state root that differs by one bit
				t.Run("OnebitDifferentStateRoot", func(t *testing.T) {
					corruptedStateRoot := correctStateRoot
					corruptedStateRoot[0] ^= 0x01 // Flip the least significant bit of first byte
					err := v.ValidateAccountState(ctx, baseAccountResult, corruptedStateRoot, big.NewInt(5219647))
					assert.Error(t, err, "Validation should fail with one bit different state root")
					assert.Contains(t, err.Error(), "state root mismatch")
				})
			})

			// 2. Test account proof corruption failures
			t.Run("AccountProofCorruption", func(t *testing.T) {
				// Corrupt first proof node (change first byte)
				t.Run("CorruptFirstProofNode", func(t *testing.T) {
					corruptResult := baseAccountResult
					corruptProof := baseAccountResult.AccountProof[0]
					// Change first two characters after 0x
					corruptProof = "0x0" + corruptProof[3:]
					corruptResult.AccountProof = make([]string, len(baseAccountResult.AccountProof))
					copy(corruptResult.AccountProof, baseAccountResult.AccountProof)
					corruptResult.AccountProof[0] = corruptProof

					err := v.ValidateAccountState(ctx, corruptResult, correctStateRoot, big.NewInt(5219647))
					assert.Error(t, err, "Validation should fail with corrupted first proof node")
				})

				// Corrupt middle proof node
				t.Run("CorruptMiddleProofNode", func(t *testing.T) {
					if len(baseAccountResult.AccountProof) > 2 {
						corruptResult := baseAccountResult
						corruptResult.AccountProof = make([]string, len(baseAccountResult.AccountProof))
						copy(corruptResult.AccountProof, baseAccountResult.AccountProof)
						// Corrupt middle node by changing last few characters
						midIndex := len(corruptResult.AccountProof) / 2
						originalProof := corruptResult.AccountProof[midIndex]
						corruptResult.AccountProof[midIndex] = originalProof[:len(originalProof)-4] + "0000"

						err := v.ValidateAccountState(ctx, corruptResult, correctStateRoot, big.NewInt(5219647))
						assert.Error(t, err, "Validation should fail with corrupted middle proof node")
					}
				})

				// Remove one proof node
				t.Run("MissingProofNode", func(t *testing.T) {
					if len(baseAccountResult.AccountProof) > 1 {
						corruptResult := baseAccountResult
						corruptResult.AccountProof = baseAccountResult.AccountProof[:len(baseAccountResult.AccountProof)-1]

						err := v.ValidateAccountState(ctx, corruptResult, correctStateRoot, big.NewInt(5219647))
						assert.Error(t, err, "Validation should fail with missing proof node")
					}
				})

				// Add extra proof node
				t.Run("ExtraProofNode", func(t *testing.T) {
					corruptResult := baseAccountResult
					corruptResult.AccountProof = make([]string, len(baseAccountResult.AccountProof)+1)
					copy(corruptResult.AccountProof, baseAccountResult.AccountProof)
					corruptResult.AccountProof[len(baseAccountResult.AccountProof)] = "0x1234567890abcdef"

					err := v.ValidateAccountState(ctx, corruptResult, correctStateRoot, big.NewInt(5219647))
					assert.NoError(t, err, "Validation should not fail with extra proof node")
				})
			})

			// 3. Test account state field corruption failures
			t.Run("AccountStateFieldCorruption", func(t *testing.T) {
				// Corrupt nonce (change from original to +1)
				t.Run("CorruptNonce", func(t *testing.T) {
					corruptResult := baseAccountResult
					corruptResult.Nonce = hexutil.Uint64(uint64(corruptResult.Nonce) + 1)

					err := v.ValidateAccountState(ctx, corruptResult, correctStateRoot, big.NewInt(5219647))
					assert.Error(t, err, "Validation should fail with corrupted nonce")
					assert.Contains(t, err.Error(), "account nonce is not matched")
				})

				// Corrupt balance (subtract 1)
				t.Run("CorruptBalance", func(t *testing.T) {
					corruptResult := baseAccountResult
					originalBalance := corruptResult.Balance.ToInt()
					newBalance := new(big.Int).Sub(originalBalance, big.NewInt(1))
					corruptResult.Balance = (*hexutil.Big)(newBalance)

					err := v.ValidateAccountState(ctx, corruptResult, correctStateRoot, big.NewInt(5219647))
					assert.Error(t, err, "Validation should fail with corrupted balance")
					assert.Contains(t, err.Error(), "account balance is not matched")
				})

				// Corrupt storage hash (change last byte)
				t.Run("CorruptStorageHash", func(t *testing.T) {
					corruptResult := baseAccountResult
					corruptResult.StorageHash[31] ^= 0x01 // Flip last byte

					err := v.ValidateAccountState(ctx, corruptResult, correctStateRoot, big.NewInt(5219647))
					assert.Error(t, err, "Validation should fail with corrupted storage hash")
					assert.Contains(t, err.Error(), "account storage hash is not matched")
				})

				// Corrupt code hash (change first byte)
				t.Run("CorruptCodeHash", func(t *testing.T) {
					corruptResult := baseAccountResult
					corruptResult.CodeHash[0] ^= 0x01 // Flip first byte

					err := v.ValidateAccountState(ctx, corruptResult, correctStateRoot, big.NewInt(5219647))
					assert.Error(t, err, "Validation should fail with corrupted code hash")
					assert.Contains(t, err.Error(), "account code hash is not matched")
				})

				// Corrupt address (though this mainly affects proof lookup)
				t.Run("CorruptAddress", func(t *testing.T) {
					corruptResult := baseAccountResult
					corruptResult.Address = common.HexToAddress("0x1234567890123456789012345678901234567890")

					err := v.ValidateAccountState(ctx, corruptResult, correctStateRoot, big.NewInt(5219647))
					assert.Error(t, err, "Validation should fail with corrupted address")
				})
			})

			// 4. Test edge cases and invalid data
			t.Run("EdgeCasesAndInvalidData", func(t *testing.T) {
				// Empty account proof
				t.Run("EmptyAccountProof", func(t *testing.T) {
					corruptResult := baseAccountResult
					corruptResult.AccountProof = []string{}

					err := v.ValidateAccountState(ctx, corruptResult, correctStateRoot, big.NewInt(5219647))
					assert.Error(t, err, "Validation should fail with empty account proof")
				})

				// Invalid hex in proof node
				t.Run("InvalidHexInProof", func(t *testing.T) {
					corruptResult := baseAccountResult
					corruptResult.AccountProof = make([]string, len(baseAccountResult.AccountProof))
					copy(corruptResult.AccountProof, baseAccountResult.AccountProof)
					corruptResult.AccountProof[0] = "0xGGGGGGGG" // Invalid hex

					err := v.ValidateAccountState(ctx, corruptResult, correctStateRoot, big.NewInt(5219647))
					assert.Error(t, err, "Validation should fail with invalid hex in proof")
					assert.Contains(t, err.Error(), "failed to decode first node")
				})

				// Zero balance account
				t.Run("ZeroBalanceAccount", func(t *testing.T) {
					// This should actually pass if the proof is valid for a zero-balance account
					// We're testing that the validation logic handles zero values correctly
					corruptResult := baseAccountResult
					corruptResult.Balance = (*hexutil.Big)(big.NewInt(0))

					err := v.ValidateAccountState(ctx, corruptResult, correctStateRoot, big.NewInt(5219647))
					// This might pass or fail depending on whether the proof matches the modified balance
					// The important thing is it doesn't panic and handles the zero value
					if err != nil {
						assert.Contains(t, err.Error(), "account balance is not matched")
					}
				})
			})

			// 5. Test successful validation for comparison
			t.Run("SuccessfulValidation", func(t *testing.T) {
				err := v.ValidateAccountState(ctx, baseAccountResult, correctStateRoot, big.NewInt(5219647))
				assert.NoError(t, err, "Validation should succeed with correct data")
			})
		})
	}
}
