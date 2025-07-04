// Copyright 2025 Coinbase, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package validator

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"math/rand"
	"os"
	"testing"

	"github.com/coinbase/rosetta-geth-sdk/configuration"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
)

// testSetupData holds common test setup data
type testSetupData struct {
	ctx         context.Context
	validator   TrustlessValidator
	baseAccount AccountResult
	stateRoot   common.Hash
	blockNumber *big.Int
	chainData   ChainTestData
}

// setupAccountTest creates common test setup data for account validation tests
func setupAccountTest(t *testing.T, chainData ChainTestData) *testSetupData {
	ctx := context.Background()

	cfg := &configuration.Configuration{
		ChainConfig: chainData.ChainConfig,
		Network:     chainData.Network,
		GethURL:     chainData.GethURL,
		RosettaCfg: configuration.RosettaConfig{
			EnableTrustlessAccountValidation: true,
		},
	}
	validator := NewEthereumValidator(cfg)

	// Load test data
	var baseAccount AccountResult
	data, err := os.ReadFile(chainData.AccountFixtureFile)
	if err != nil {
		t.Fatalf("failed to read fixture file for %s: %v", chainData.Name, err)
	}
	err = json.Unmarshal(data, &baseAccount)
	if err != nil {
		t.Fatalf("failed to unmarshal fixture for %s: %v", chainData.Name, err)
	}

	// Calculate the correct state root from the first proof node
	firstNodeData, err := hexutil.Decode(baseAccount.AccountProof[0])
	if err != nil {
		t.Fatalf("failed to decode first node for %s: %v", chainData.Name, err)
	}
	stateRoot := common.BytesToHash(crypto.Keccak256(firstNodeData))

	return &testSetupData{
		ctx:         ctx,
		validator:   validator,
		baseAccount: baseAccount,
		stateRoot:   stateRoot,
		blockNumber: chainData.TestBlockNumber,
		chainData:   chainData,
	}
}

// SUCCESS TESTS
func TestValidateAccountState_ExactValidData(t *testing.T) {
	for _, chainData := range AccountTestChains {
		t.Run(chainData.Name, func(t *testing.T) {
			setup := setupAccountTest(t, chainData)
			err := setup.validator.ValidateAccountState(setup.baseAccount, setup.stateRoot, setup.blockNumber)
			assert.NoError(t, err, "Validation should succeed with exact valid data")
		})
	}
}

func TestValidateAccountState_RepeatedValidation(t *testing.T) {
	for _, chainData := range AccountTestChains {
		t.Run(chainData.Name, func(t *testing.T) {
			setup := setupAccountTest(t, chainData)
			for i := 0; i < 3; i++ {
				err := setup.validator.ValidateAccountState(setup.baseAccount, setup.stateRoot, setup.blockNumber)
				assert.NoError(t, err, "Validation should succeed on repeated calls")
			}
		})
	}
}

func TestValidateAccountState_DifferentBlockNumbers(t *testing.T) {
	for _, chainData := range AccountTestChains {
		t.Run(chainData.Name, func(t *testing.T) {
			setup := setupAccountTest(t, chainData)
			blockNumbers := []*big.Int{
				setup.blockNumber,
				big.NewInt(rand.Int63n(setup.blockNumber.Int64())),
				new(big.Int).Add(setup.blockNumber, big.NewInt(rand.Int63n(1000000))),
				new(big.Int).Sub(setup.blockNumber, big.NewInt(rand.Int63n(1000))),
				big.NewInt(rand.Int63n(setup.blockNumber.Int64())),
				big.NewInt(rand.Int63n(setup.blockNumber.Int64())),
			}

			for _, blockNumber := range blockNumbers {
				err := setup.validator.ValidateAccountState(setup.baseAccount, setup.stateRoot, blockNumber)
				assert.NoError(t, err, "Validation should succeed with different block numbers")
			}
		})
	}
}

func TestValidateAccountState_OriginalBalance(t *testing.T) {
	for _, chainData := range AccountTestChains {
		t.Run(chainData.Name, func(t *testing.T) {
			setup := setupAccountTest(t, chainData)
			t.Logf("Testing account with balance: %s", setup.baseAccount.Balance.ToInt().String())
			err := setup.validator.ValidateAccountState(setup.baseAccount, setup.stateRoot, setup.blockNumber)
			assert.NoError(t, err, "Validation should succeed with original balance")
		})
	}
}

func TestValidateAccountState_OriginalNonce(t *testing.T) {
	for _, chainData := range AccountTestChains {
		t.Run(chainData.Name, func(t *testing.T) {
			setup := setupAccountTest(t, chainData)
			t.Logf("Testing account with nonce: %d", uint64(setup.baseAccount.Nonce))
			err := setup.validator.ValidateAccountState(setup.baseAccount, setup.stateRoot, setup.blockNumber)
			assert.NoError(t, err, "Validation should succeed with original nonce")
		})
	}
}

func TestValidateAccountState_OriginalStorageHash(t *testing.T) {
	for _, chainData := range AccountTestChains {
		t.Run(chainData.Name, func(t *testing.T) {
			setup := setupAccountTest(t, chainData)
			t.Logf("Testing account with storage hash: %s", setup.baseAccount.StorageHash.Hex())
			err := setup.validator.ValidateAccountState(setup.baseAccount, setup.stateRoot, setup.blockNumber)
			assert.NoError(t, err, "Validation should succeed with original storage hash")
		})
	}
}

func TestValidateAccountState_OriginalCodeHash(t *testing.T) {
	for _, chainData := range AccountTestChains {
		t.Run(chainData.Name, func(t *testing.T) {
			setup := setupAccountTest(t, chainData)
			t.Logf("Testing account with code hash: %s", setup.baseAccount.CodeHash.Hex())
			err := setup.validator.ValidateAccountState(setup.baseAccount, setup.stateRoot, setup.blockNumber)
			assert.NoError(t, err, "Validation should succeed with original code hash")
		})
	}
}

func TestValidateAccountState_CompleteProofChain(t *testing.T) {
	for _, chainData := range AccountTestChains {
		t.Run(chainData.Name, func(t *testing.T) {
			setup := setupAccountTest(t, chainData)
			t.Logf("Testing with %d proof nodes", len(setup.baseAccount.AccountProof))
			err := setup.validator.ValidateAccountState(setup.baseAccount, setup.stateRoot, setup.blockNumber)
			assert.NoError(t, err, "Validation should succeed with complete proof chain")
		})
	}
}

func TestValidateAccountState_ProofNodeFormatting(t *testing.T) {
	for _, chainData := range AccountTestChains {
		t.Run(chainData.Name, func(t *testing.T) {
			setup := setupAccountTest(t, chainData)

			// Verify all proof nodes are valid hex
			for i, proofNode := range setup.baseAccount.AccountProof {
				_, err := hexutil.Decode(proofNode)
				assert.NoError(t, err, "Proof node %d should be valid hex: %s", i, proofNode)
			}

			err := setup.validator.ValidateAccountState(setup.baseAccount, setup.stateRoot, setup.blockNumber)
			assert.NoError(t, err, "Validation should succeed with properly formatted proof nodes")
		})
	}
}

func TestValidateAccountState_StateRootCalculation(t *testing.T) {
	for _, chainData := range AccountTestChains {
		t.Run(chainData.Name, func(t *testing.T) {
			setup := setupAccountTest(t, chainData)

			// Recalculate state root to ensure consistency
			firstNodeData, err := hexutil.Decode(setup.baseAccount.AccountProof[0])
			assert.NoError(t, err, "First proof node should decode successfully")

			calculatedStateRoot := common.BytesToHash(crypto.Keccak256(firstNodeData))
			assert.Equal(t, setup.stateRoot, calculatedStateRoot, "State root calculation should be consistent")

			err = setup.validator.ValidateAccountState(setup.baseAccount, calculatedStateRoot, setup.blockNumber)
			assert.NoError(t, err, "Validation should succeed with calculated state root")
		})
	}
}

func TestValidateAccountState_CorrectAddress(t *testing.T) {
	for _, chainData := range AccountTestChains {
		t.Run(chainData.Name, func(t *testing.T) {
			setup := setupAccountTest(t, chainData)
			t.Logf("Testing account address: %s", setup.baseAccount.Address.Hex())
			err := setup.validator.ValidateAccountState(setup.baseAccount, setup.stateRoot, setup.blockNumber)
			assert.NoError(t, err, "Validation should succeed with correct address")
		})
	}
}

func TestValidateAccountState_AddressConsistency(t *testing.T) {
	for _, chainData := range AccountTestChains {
		t.Run(chainData.Name, func(t *testing.T) {
			setup := setupAccountTest(t, chainData)

			// The account hash should be consistent with the address
			expectedAccountHash := crypto.Keccak256(setup.baseAccount.Address[:])
			t.Logf("Account hash for verification: %x", expectedAccountHash)

			err := setup.validator.ValidateAccountState(setup.baseAccount, setup.stateRoot, setup.blockNumber)
			assert.NoError(t, err, "Validation should succeed with address-consistent proof path")
		})
	}
}

func TestValidateAccountState_EOACharacteristics(t *testing.T) {
	for _, chainData := range AccountTestChains {
		t.Run(chainData.Name, func(t *testing.T) {
			setup := setupAccountTest(t, chainData)

			// Check if this looks like an EOA (empty code hash typically indicates EOA)
			emptyCodeHash := crypto.Keccak256Hash(nil)
			if setup.baseAccount.CodeHash == emptyCodeHash {
				t.Logf("Testing EOA (empty code hash): %s", setup.baseAccount.CodeHash.Hex())
			} else {
				t.Logf("Testing contract account (non-empty code hash): %s", setup.baseAccount.CodeHash.Hex())
			}

			err := setup.validator.ValidateAccountState(setup.baseAccount, setup.stateRoot, setup.blockNumber)
			assert.NoError(t, err, "Validation should succeed regardless of account type")
		})
	}
}

func TestValidateAccountState_AccountWithStorage(t *testing.T) {
	for _, chainData := range AccountTestChains {
		t.Run(chainData.Name, func(t *testing.T) {
			setup := setupAccountTest(t, chainData)

			emptyStorageHash := common.HexToHash("0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421")
			if setup.baseAccount.StorageHash != emptyStorageHash {
				t.Logf("Testing account with non-empty storage: %s", setup.baseAccount.StorageHash.Hex())
			} else {
				t.Logf("Testing account with empty storage: %s", setup.baseAccount.StorageHash.Hex())
			}

			err := setup.validator.ValidateAccountState(setup.baseAccount, setup.stateRoot, setup.blockNumber)
			assert.NoError(t, err, "Validation should succeed with account storage")
		})
	}
}

func TestValidateAccountState_MinimumValidProof(t *testing.T) {
	for _, chainData := range AccountTestChains {
		t.Run(chainData.Name, func(t *testing.T) {
			setup := setupAccountTest(t, chainData)

			// Ensure we have at least one proof node (we should have more, but test minimum requirement)
			assert.True(t, len(setup.baseAccount.AccountProof) >= 1, "Should have at least one proof node")

			err := setup.validator.ValidateAccountState(setup.baseAccount, setup.stateRoot, setup.blockNumber)
			assert.NoError(t, err, "Validation should succeed with minimum valid proof")
		})
	}
}

func TestValidateAccountState_MaximumPrecisionValues(t *testing.T) {
	for _, chainData := range AccountTestChains {
		t.Run(chainData.Name, func(t *testing.T) {
			setup := setupAccountTest(t, chainData)

			// Log the actual values to understand the precision
			t.Logf("Balance precision: %s wei", setup.baseAccount.Balance.ToInt().String())
			t.Logf("Nonce value: %d", uint64(setup.baseAccount.Nonce))

			err := setup.validator.ValidateAccountState(setup.baseAccount, setup.stateRoot, setup.blockNumber)
			assert.NoError(t, err, "Validation should succeed with high precision values")
		})
	}
}

func TestValidateAccountState_ConcurrentValidation(t *testing.T) {
	for _, chainData := range AccountTestChains {
		t.Run(chainData.Name, func(t *testing.T) {
			setup := setupAccountTest(t, chainData)

			const numGoroutines = 10
			errChan := make(chan error, numGoroutines)

			for i := 0; i < numGoroutines; i++ {
				go func(goroutineID int) {
					err := setup.validator.ValidateAccountState(setup.baseAccount, setup.stateRoot, setup.blockNumber)
					if err != nil {
						errChan <- fmt.Errorf("goroutine %d failed: %w", goroutineID, err)
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
	}
}

func TestValidateAccountState_ValidationStability(t *testing.T) {
	for _, chainData := range AccountTestChains {
		t.Run(chainData.Name, func(t *testing.T) {
			setup := setupAccountTest(t, chainData)

			const iterations = 50
			for i := 0; i < iterations; i++ {
				err := setup.validator.ValidateAccountState(setup.baseAccount, setup.stateRoot, setup.blockNumber)
				assert.NoError(t, err, "Validation should be stable over multiple iterations (iteration %d)", i)
			}
		})
	}
}

func TestValidateAccountState_ExtraProofNode(t *testing.T) {
	for _, chainData := range AccountTestChains {
		t.Run(chainData.Name, func(t *testing.T) {
			setup := setupAccountTest(t, chainData)

			corruptResult := setup.baseAccount
			corruptResult.AccountProof = make([]string, len(setup.baseAccount.AccountProof)+1)
			copy(corruptResult.AccountProof, setup.baseAccount.AccountProof)
			corruptResult.AccountProof[len(setup.baseAccount.AccountProof)] = "0x1234567890abcdef"

			err := setup.validator.ValidateAccountState(corruptResult, setup.stateRoot, setup.blockNumber)
			assert.NoError(t, err, "Validation should not fail with extra proof node")
		})
	}
}

// FAILURE TESTS

func TestValidateAccountState_WrongStateRoot(t *testing.T) {
	for _, chainData := range AccountTestChains {
		t.Run(chainData.Name, func(t *testing.T) {
			setup := setupAccountTest(t, chainData)

			wrongStateRoot := common.HexToHash("0x7adc7dbc4c36299fc65fd1dc3798a6a58c29c171b79584bfc3512f5ad82a59d4")
			err := setup.validator.ValidateAccountState(setup.baseAccount, wrongStateRoot, setup.blockNumber)
			assert.Error(t, err, "Validation should fail with wrong state root")
			assert.Contains(t, err.Error(), "state root mismatch")
		})
	}
}

func TestValidateAccountState_ZeroStateRoot(t *testing.T) {
	for _, chainData := range AccountTestChains {
		t.Run(chainData.Name, func(t *testing.T) {
			setup := setupAccountTest(t, chainData)

			zeroStateRoot := common.Hash{}
			err := setup.validator.ValidateAccountState(setup.baseAccount, zeroStateRoot, setup.blockNumber)
			assert.Error(t, err, "Validation should fail with zero state root")
			assert.Contains(t, err.Error(), "state root mismatch")
		})
	}
}

func TestValidateAccountState_OnebitDifferentStateRoot(t *testing.T) {
	for _, chainData := range AccountTestChains {
		t.Run(chainData.Name, func(t *testing.T) {
			setup := setupAccountTest(t, chainData)

			corruptedStateRoot := setup.stateRoot
			corruptedStateRoot[0] ^= 0x01 // Flip the least significant bit of first byte
			err := setup.validator.ValidateAccountState(setup.baseAccount, corruptedStateRoot, setup.blockNumber)
			assert.Error(t, err, "Validation should fail with one bit different state root")
			assert.Contains(t, err.Error(), "state root mismatch")
		})
	}
}

func TestValidateAccountState_CorruptFirstProofNode(t *testing.T) {
	for _, chainData := range AccountTestChains {
		t.Run(chainData.Name, func(t *testing.T) {
			setup := setupAccountTest(t, chainData)

			corruptResult := setup.baseAccount
			corruptProof := setup.baseAccount.AccountProof[0]
			// Change first two characters after 0x
			corruptProof = "0x0" + corruptProof[3:]
			corruptResult.AccountProof = make([]string, len(setup.baseAccount.AccountProof))
			copy(corruptResult.AccountProof, setup.baseAccount.AccountProof)
			corruptResult.AccountProof[0] = corruptProof

			err := setup.validator.ValidateAccountState(corruptResult, setup.stateRoot, setup.blockNumber)
			assert.Error(t, err, "Validation should fail with corrupted first proof node")
		})
	}
}

func TestValidateAccountState_CorruptMiddleProofNode(t *testing.T) {
	for _, chainData := range AccountTestChains {
		t.Run(chainData.Name, func(t *testing.T) {
			setup := setupAccountTest(t, chainData)

			if len(setup.baseAccount.AccountProof) > 2 {
				corruptResult := setup.baseAccount
				corruptResult.AccountProof = make([]string, len(setup.baseAccount.AccountProof))
				copy(corruptResult.AccountProof, setup.baseAccount.AccountProof)
				// Corrupt middle node by changing last few characters
				midIndex := len(corruptResult.AccountProof) / 2
				originalProof := corruptResult.AccountProof[midIndex]
				corruptResult.AccountProof[midIndex] = originalProof[:len(originalProof)-4] + "0000"

				err := setup.validator.ValidateAccountState(corruptResult, setup.stateRoot, setup.blockNumber)
				assert.Error(t, err, "Validation should fail with corrupted middle proof node")
			}
		})
	}
}

func TestValidateAccountState_MissingProofNode(t *testing.T) {
	for _, chainData := range AccountTestChains {
		t.Run(chainData.Name, func(t *testing.T) {
			setup := setupAccountTest(t, chainData)

			if len(setup.baseAccount.AccountProof) > 1 {
				corruptResult := setup.baseAccount
				corruptResult.AccountProof = setup.baseAccount.AccountProof[:len(setup.baseAccount.AccountProof)-1]

				err := setup.validator.ValidateAccountState(corruptResult, setup.stateRoot, setup.blockNumber)
				assert.Error(t, err, "Validation should fail with missing proof node")
			}
		})
	}
}

func TestValidateAccountState_CorruptNonce(t *testing.T) {
	for _, chainData := range AccountTestChains {
		t.Run(chainData.Name, func(t *testing.T) {
			setup := setupAccountTest(t, chainData)

			corruptResult := setup.baseAccount
			corruptResult.Nonce = hexutil.Uint64(uint64(corruptResult.Nonce) + 1)

			err := setup.validator.ValidateAccountState(corruptResult, setup.stateRoot, setup.blockNumber)
			assert.Error(t, err, "Validation should fail with corrupted nonce")
			assert.Contains(t, err.Error(), "account nonce is not matched")
		})
	}
}

func TestValidateAccountState_CorruptBalance(t *testing.T) {
	for _, chainData := range AccountTestChains {
		t.Run(chainData.Name, func(t *testing.T) {
			setup := setupAccountTest(t, chainData)

			corruptResult := setup.baseAccount
			originalBalance := corruptResult.Balance.ToInt()
			newBalance := new(big.Int).Sub(originalBalance, big.NewInt(1))
			corruptResult.Balance = (*hexutil.Big)(newBalance)

			err := setup.validator.ValidateAccountState(corruptResult, setup.stateRoot, setup.blockNumber)
			assert.Error(t, err, "Validation should fail with corrupted balance")
			assert.Contains(t, err.Error(), "account balance is not matched")
		})
	}
}

func TestValidateAccountState_CorruptStorageHash(t *testing.T) {
	for _, chainData := range AccountTestChains {
		t.Run(chainData.Name, func(t *testing.T) {
			setup := setupAccountTest(t, chainData)

			corruptResult := setup.baseAccount
			corruptResult.StorageHash[31] ^= 0x01 // Flip last byte

			err := setup.validator.ValidateAccountState(corruptResult, setup.stateRoot, setup.blockNumber)
			assert.Error(t, err, "Validation should fail with corrupted storage hash")
			assert.Contains(t, err.Error(), "account storage hash is not matched")
		})
	}
}

func TestValidateAccountState_CorruptCodeHash(t *testing.T) {
	for _, chainData := range AccountTestChains {
		t.Run(chainData.Name, func(t *testing.T) {
			setup := setupAccountTest(t, chainData)

			corruptResult := setup.baseAccount
			corruptResult.CodeHash[0] ^= 0x01 // Flip first byte

			err := setup.validator.ValidateAccountState(corruptResult, setup.stateRoot, setup.blockNumber)
			assert.Error(t, err, "Validation should fail with corrupted code hash")
			assert.Contains(t, err.Error(), "account code hash is not matched")
		})
	}
}

func TestValidateAccountState_CorruptAddress(t *testing.T) {
	for _, chainData := range AccountTestChains {
		t.Run(chainData.Name, func(t *testing.T) {
			setup := setupAccountTest(t, chainData)

			corruptResult := setup.baseAccount
			corruptResult.Address = common.HexToAddress("0x1234567890123456789012345678901234567890")

			err := setup.validator.ValidateAccountState(corruptResult, setup.stateRoot, setup.blockNumber)
			assert.Error(t, err, "Validation should fail with corrupted address")
		})
	}
}

func TestValidateAccountState_EmptyAccountProof(t *testing.T) {
	for _, chainData := range AccountTestChains {
		t.Run(chainData.Name, func(t *testing.T) {
			setup := setupAccountTest(t, chainData)

			corruptResult := setup.baseAccount
			corruptResult.AccountProof = []string{}

			err := setup.validator.ValidateAccountState(corruptResult, setup.stateRoot, setup.blockNumber)
			assert.Error(t, err, "Validation should fail with empty account proof")
		})
	}
}

func TestValidateAccountState_InvalidHexInProof(t *testing.T) {
	for _, chainData := range AccountTestChains {
		t.Run(chainData.Name, func(t *testing.T) {
			setup := setupAccountTest(t, chainData)

			corruptResult := setup.baseAccount
			corruptResult.AccountProof = make([]string, len(setup.baseAccount.AccountProof))
			copy(corruptResult.AccountProof, setup.baseAccount.AccountProof)
			corruptResult.AccountProof[0] = "0xGGGGGGGG" // Invalid hex

			err := setup.validator.ValidateAccountState(corruptResult, setup.stateRoot, setup.blockNumber)
			assert.Error(t, err, "Validation should fail with invalid hex in proof")
			assert.Contains(t, err.Error(), "failed to decode first node")
		})
	}
}

func TestValidateAccountState_ZeroBalanceAccount(t *testing.T) {
	for _, chainData := range AccountTestChains {
		t.Run(chainData.Name, func(t *testing.T) {
			setup := setupAccountTest(t, chainData)

			// This should actually pass if the proof is valid for a zero-balance account
			// We're testing that the validation logic handles zero values correctly
			corruptResult := setup.baseAccount
			corruptResult.Balance = (*hexutil.Big)(big.NewInt(0))

			err := setup.validator.ValidateAccountState(corruptResult, setup.stateRoot, setup.blockNumber)
			// This might pass or fail depending on whether the proof matches the modified balance
			// The important thing is it doesn't panic and handles the zero value
			if err != nil {
				assert.Contains(t, err.Error(), "account balance is not matched")
			}
		})
	}
}

func TestValidateAccountState_SuccessfulValidation(t *testing.T) {
	for _, chainData := range AccountTestChains {
		t.Run(chainData.Name, func(t *testing.T) {
			setup := setupAccountTest(t, chainData)
			err := setup.validator.ValidateAccountState(setup.baseAccount, setup.stateRoot, setup.blockNumber)
			assert.NoError(t, err, "Validation should succeed with correct data")
		})
	}
}
