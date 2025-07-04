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
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"strconv"
	"strings"
	"testing"

	client "github.com/coinbase/rosetta-geth-sdk/client"
	"github.com/coinbase/rosetta-geth-sdk/configuration"
	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

const (
	NETWORK_SONIC     = "sonic"
	NETWORK_BERACHAIN = "berachain"
)

const (
	BLOCKCHAIN_SONIC     = "sonic"
	BLOCKCHAIN_BERACHAIN = "berachain"
)

type BlockFixture struct {
	ParentHash       string            `json:"parentHash"`
	Sha3Uncles       string            `json:"sha3Uncles"`
	Miner            string            `json:"miner"`
	StateRoot        string            `json:"stateRoot"`
	TransactionsRoot string            `json:"transactionsRoot"`
	ReceiptsRoot     string            `json:"receiptsRoot"`
	Number           string            `json:"number"`
	GasLimit         string            `json:"gasLimit"`
	GasUsed          string            `json:"gasUsed"`
	Timestamp        string            `json:"timestamp"`
	ExtraData        string            `json:"extraData"`
	MixHash          string            `json:"mixHash"`
	Nonce            string            `json:"nonce"`
	Transactions     []json.RawMessage `json:"transactions"`
}

// Chain configurations for all supported chains
var (
	// SonicChainConfig is the chain parameters for Sonic network
	SonicChainConfig = &params.ChainConfig{
		ChainID:             big.NewInt(57054),
		HomesteadBlock:      big.NewInt(0),
		DAOForkSupport:      false,
		EIP150Block:         big.NewInt(0),
		EIP155Block:         big.NewInt(0),
		EIP158Block:         big.NewInt(0),
		ByzantiumBlock:      big.NewInt(0),
		ConstantinopleBlock: big.NewInt(0),
		PetersburgBlock:     big.NewInt(0),
		IstanbulBlock:       big.NewInt(0),
		BerlinBlock:         big.NewInt(0),
		LondonBlock:         big.NewInt(0),
	}

	BerachainChainConfig = &params.ChainConfig{
		ChainID:             big.NewInt(80094),
		HomesteadBlock:      big.NewInt(0),
		DAOForkSupport:      false,
		EIP150Block:         big.NewInt(0),
		EIP155Block:         big.NewInt(0),
		EIP158Block:         big.NewInt(0),
		ByzantiumBlock:      big.NewInt(0),
		ConstantinopleBlock: big.NewInt(0),
		PetersburgBlock:     big.NewInt(0),
		IstanbulBlock:       big.NewInt(0),
		BerlinBlock:         big.NewInt(0),
		LondonBlock:         big.NewInt(0),
	}
)

var (
	// testnet
	SonicNetwork = &types.NetworkIdentifier{
		Blockchain: BLOCKCHAIN_SONIC,
		Network:    NETWORK_SONIC,
	}

	// mainnet
	BerachainNetwork = &types.NetworkIdentifier{
		Blockchain: BLOCKCHAIN_BERACHAIN,
		Network:    NETWORK_BERACHAIN,
	}
)

// ChainTestData holds configuration for each chain's tests
type ChainTestData struct {
	Name               string
	ChainConfig        *params.ChainConfig
	Network            *types.NetworkIdentifier
	BlockFixtureFile   string
	AccountFixtureFile string
	ReceiptFixtureFile string
	TestBlockNumber    *big.Int
	GethURL            string
}

// All supported chains for testing
var BlockTestChains = []ChainTestData{
	{
		Name:               "Sonic",
		ChainConfig:        SonicChainConfig,
		Network:            SonicNetwork,
		BlockFixtureFile:   "testdata/sonic_test.json",
		ReceiptFixtureFile: "testdata/sonic_receipts.json",
		TestBlockNumber:    big.NewInt(5219647),
	},
	{
		Name:               "Berachain",
		ChainConfig:        BerachainChainConfig,
		Network:            BerachainNetwork,
		BlockFixtureFile:   "testdata/berachain_test.json",
		ReceiptFixtureFile: "testdata/berachain_receipts.json",
		TestBlockNumber:    big.NewInt(6925006),
	},
}

// All supported chains for testing
var AccountTestChains = []ChainTestData{
	{
		Name:               "Sonic",
		ChainConfig:        SonicChainConfig,
		Network:            SonicNetwork,
		AccountFixtureFile: "testdata/sonic_account_proof.json",
		TestBlockNumber:    big.NewInt(5219647),
	},
}

func loadReceiptsFromJSON(filepath string) (ethtypes.Receipts, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read fixture file: %w", err)
	}
	var receipts ethtypes.Receipts
	if err := json.Unmarshal(data, &receipts); err != nil {
		return nil, fmt.Errorf("failed to unmarshal receipts: %w", err)
	}
	return receipts, nil
}

// loadBlockFromJSON loads a block from a JSON fixture file
func loadBlockFromJSON(filepath string, t *testing.T) (*ethtypes.Block, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read fixture file: %w", err)
	}

	var fixture BlockFixture
	if err := json.Unmarshal(data, &fixture); err != nil {
		return nil, fmt.Errorf("failed to unmarshal fixture: %w", err)
	}

	// Helper function to decode hex values
	decodeHex := func(hex string) uint64 {
		// Remove "0x" prefix if present
		hex = strings.TrimPrefix(hex, "0x")
		// Remove leading zeros
		hex = strings.TrimLeft(hex, "0")
		if hex == "" {
			return 0
		}
		val, err := strconv.ParseUint(hex, 16, 64)
		if err != nil {
			t.Logf("Error decoding hex %s: %v", hex, err)
			return 0
		}
		return val
	}

	// Create header from fixture data
	header := &ethtypes.Header{
		ParentHash:  common.HexToHash(fixture.ParentHash),
		UncleHash:   common.HexToHash(fixture.Sha3Uncles),
		Coinbase:    common.HexToAddress(fixture.Miner),
		Root:        common.HexToHash(fixture.StateRoot),
		TxHash:      common.HexToHash(fixture.TransactionsRoot),
		ReceiptHash: common.HexToHash(fixture.ReceiptsRoot),
		Number:      new(big.Int).SetUint64(decodeHex(fixture.Number)),
		GasLimit:    decodeHex(fixture.GasLimit),
		GasUsed:     decodeHex(fixture.GasUsed),
		Time:        decodeHex(fixture.Timestamp),
		Extra:       common.FromHex(fixture.ExtraData),
		MixDigest:   common.HexToHash(fixture.MixHash),
		Nonce:       ethtypes.EncodeNonce(decodeHex(fixture.Nonce)),
	}

	// Create transactions from fixture data
	transactions := make([]*ethtypes.Transaction, len(fixture.Transactions))
	for i, txData := range fixture.Transactions {
		var tx client.RPCTransaction
		if err := json.Unmarshal(txData, &tx); err != nil {
			t.Logf("Error unmarshaling transaction: %v", err)
			continue
		}
		transactions[i] = tx.Tx
	}

	return ethtypes.NewBlockWithHeader(header).WithBody(ethtypes.Body{
		Transactions: transactions,
		Uncles:       []*ethtypes.Header{},
	}), nil
}

// Helper functions to modify block values
func modifyBlockHeader(original *ethtypes.Block, modifyFn func(*ethtypes.Header)) *ethtypes.Block {
	header := *original.Header() // Create a copy
	modifyFn(&header)
	return ethtypes.NewBlockWithHeader(&header).WithBody(ethtypes.Body{
		Transactions: original.Transactions(),
		Uncles:       original.Uncles(),
		Withdrawals:  original.Withdrawals(),
	})
}

func modifyTransaction(original *ethtypes.Block, txIndex int, modifyFn func(*ethtypes.Transaction) *ethtypes.Transaction) *ethtypes.Block {
	transactions := make([]*ethtypes.Transaction, len(original.Transactions()))
	copy(transactions, original.Transactions())
	if txIndex < len(transactions) {
		tx := transactions[txIndex]
		modifiedTx := modifyFn(tx)
		transactions[txIndex] = modifiedTx
	}
	return ethtypes.NewBlockWithHeader(original.Header()).WithBody(ethtypes.Body{
		Transactions: transactions,
		Uncles:       original.Uncles(),
		Withdrawals:  original.Withdrawals(),
	})
}

func modifyWithdrawals(original *ethtypes.Block, modifyFn func([]*ethtypes.Withdrawal)) *ethtypes.Block {
	withdrawals := make([]*ethtypes.Withdrawal, len(original.Withdrawals()))
	copy(withdrawals, original.Withdrawals())
	modifyFn(withdrawals)
	return ethtypes.NewBlockWithHeader(original.Header()).WithBody(ethtypes.Body{
		Transactions: original.Transactions(),
		Uncles:       original.Uncles(),
		Withdrawals:  withdrawals,
	})
}

func TestBlockValidator_HeaderFailures(t *testing.T) {
	for _, chainData := range BlockTestChains {
		t.Run(chainData.Name, func(t *testing.T) {
			// Load the base block that we'll modify for each test
			baseBlock, err := loadBlockFromJSON(chainData.BlockFixtureFile, t)
			if err != nil {
				t.Fatalf("Failed to load base block fixture for %s: %v", chainData.Name, err)
			}

			cfg := &configuration.Configuration{
				ChainConfig: chainData.ChainConfig,
				Network:     chainData.Network,
				RosettaCfg: configuration.RosettaConfig{
					EnableTrustlessBlockValidation: true,
				},
			}
			v := NewEthereumValidator(cfg)

			testCases := []struct {
				name     string
				modifyFn func(*ethtypes.Header)
				wantErr  string
			}{
				{
					name: "corrupt transactions root",
					modifyFn: func(h *ethtypes.Header) {
						h.TxHash = common.HexToHash("0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef")
					},
					wantErr: "invalid block header",
				},
				{
					name: "corrupt block number",
					modifyFn: func(h *ethtypes.Header) {
						h.Number = big.NewInt(16000000)
					},
					wantErr: "invalid block header",
				},
				{
					name: "corrupt miner",
					modifyFn: func(h *ethtypes.Header) {
						h.Coinbase = common.HexToAddress("0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef")
					},
					wantErr: "invalid block header",
				},
				// Add more header test cases here
			}

			for _, tc := range testCases {
				t.Run(tc.name, func(t *testing.T) {
					modifiedBlock := modifyBlockHeader(baseBlock, tc.modifyFn)
					// Provide empty receipts since we're testing header validation
					emptyReceipts := make(ethtypes.Receipts, 0)
					err := v.ValidateBlock(modifiedBlock, emptyReceipts, baseBlock.Hash())
					if err == nil {
						t.Error("expected error but got none")
						return
					}
					if !strings.Contains(err.Error(), tc.wantErr) {
						t.Errorf("got error %v, want error containing %q", err, tc.wantErr)
					}
				})
			}
		})
	}
}

func TestBlockValidator_TransactionFailures(t *testing.T) {
	for _, chainData := range BlockTestChains {
		t.Run(chainData.Name, func(t *testing.T) {
			// Load the base block that we'll modify for each test
			baseBlock, err := loadBlockFromJSON(chainData.BlockFixtureFile, t)
			if err != nil {
				t.Fatalf("Failed to load base block fixture for %s: %v", chainData.Name, err)
			}

			cfg := &configuration.Configuration{
				ChainConfig: chainData.ChainConfig,
				Network:     chainData.Network,
				RosettaCfg: configuration.RosettaConfig{
					EnableTrustlessBlockValidation: true,
				},
			}
			v := NewEthereumValidator(cfg)

			testCases := []struct {
				name     string
				txIndex  int
				modifyFn func(*ethtypes.Transaction) *ethtypes.Transaction
				wantErr  string
			}{
				{
					name:    "corrupt chain id",
					txIndex: 0,
					modifyFn: func(tx *ethtypes.Transaction) *ethtypes.Transaction {
						// Create a new transaction with modified values
						return ethtypes.NewTransaction(
							tx.Nonce(),
							*tx.To(),
							tx.Value(),
							tx.Gas(),
							tx.GasPrice(),
							tx.Data(),
						)
					},
					wantErr: "invalid transactions hash",
				},
				{
					name:    "corrupt gas used",
					txIndex: 0,
					modifyFn: func(tx *ethtypes.Transaction) *ethtypes.Transaction {
						return ethtypes.NewTransaction(
							tx.Nonce(),
							*tx.To(),
							tx.Value(),
							1395941, // Modified gas
							tx.GasPrice(),
							tx.Data(),
						)
					},
					wantErr: "invalid transactions hash",
				},
				{
					name:    "corrupt to field",
					txIndex: 0,
					modifyFn: func(tx *ethtypes.Transaction) *ethtypes.Transaction {
						addr := common.HexToAddress("0x086d426f8b653b88a2d6d03051c8b4ab8783be2c") // Modified last char
						return ethtypes.NewTransaction(
							tx.Nonce(),
							addr,
							tx.Value(),
							tx.Gas(),
							tx.GasPrice(),
							tx.Data(),
						)
					},
					wantErr: "invalid transactions hash",
				},
				{
					name:    "corrupt value field",
					txIndex: 0,
					modifyFn: func(tx *ethtypes.Transaction) *ethtypes.Transaction {
						return ethtypes.NewTransaction(
							tx.Nonce(),
							*tx.To(),
							big.NewInt(1), // Modified from 0 to 1
							tx.Gas(),
							tx.GasPrice(),
							tx.Data(),
						)
					},
					wantErr: "invalid transactions hash",
				},
				{
					name:    "corrupt data field",
					txIndex: 0,
					modifyFn: func(tx *ethtypes.Transaction) *ethtypes.Transaction {
						return ethtypes.NewTransaction(
							tx.Nonce(),
							*tx.To(),
							tx.Value(),
							tx.Gas(),
							tx.GasPrice(),
							[]byte{0x1}, // Modified data
						)
					},
					wantErr: "invalid transactions hash",
				},
			}

			for _, tc := range testCases {
				t.Run(tc.name, func(t *testing.T) {
					modifiedBlock := modifyTransaction(baseBlock, tc.txIndex, tc.modifyFn)
					// Provide empty receipts since we're testing header validation
					emptyReceipts := make(ethtypes.Receipts, 0)
					err := v.ValidateBlock(modifiedBlock, emptyReceipts, baseBlock.Hash())
					if err == nil {
						t.Error("expected error but got none")
						return
					}
					if !strings.Contains(err.Error(), tc.wantErr) {
						t.Errorf("got error %v, want error containing %q", err, tc.wantErr)
					}
				})
			}
		})
	}
}

func TestBlockValidator_WithdrawalFailures(t *testing.T) {
	for _, chainData := range BlockTestChains {
		t.Run(chainData.Name, func(t *testing.T) {
			// Load the base block that we'll modify for each test
			baseBlock, err := loadBlockFromJSON(chainData.BlockFixtureFile, t)
			if err != nil {
				t.Fatalf("Failed to load base block fixture for %s: %v", chainData.Name, err)
			}

			// Skip test if block has no withdrawals
			if len(baseBlock.Withdrawals()) == 0 {
				t.Skipf("Skipping withdrawal test for %s: block has no withdrawals", chainData.Name)
				return
			}

			cfg := &configuration.Configuration{
				ChainConfig: chainData.ChainConfig,
				Network:     chainData.Network,
				RosettaCfg: configuration.RosettaConfig{
					EnableTrustlessBlockValidation: true,
				},
			}
			v := NewEthereumValidator(cfg)

			testCases := []struct {
				name     string
				modifyFn func([]*ethtypes.Withdrawal)
				wantErr  string
			}{
				{
					name: "corrupt withdrawal index",
					modifyFn: func(withdrawals []*ethtypes.Withdrawal) {
						if len(withdrawals) > 0 {
							withdrawals[0].Index++ // Modify index
						}
					},
					wantErr: "invalid withdrawals hash",
				},
				{
					name: "corrupt validator index",
					modifyFn: func(withdrawals []*ethtypes.Withdrawal) {
						if len(withdrawals) > 0 {
							withdrawals[0].Validator++ // Modify validator index
						}
					},
					wantErr: "invalid withdrawals hash",
				},
				{
					name: "corrupt withdrawal address",
					modifyFn: func(withdrawals []*ethtypes.Withdrawal) {
						if len(withdrawals) > 0 {
							addr := withdrawals[0].Address
							// Modify the last byte of the address
							modifiedAddr := common.BytesToAddress(append(addr.Bytes()[:19], 0xFF))
							withdrawals[0].Address = modifiedAddr
						}
					},
					wantErr: "invalid withdrawals hash",
				},
				{
					name: "corrupt withdrawal amount",
					modifyFn: func(withdrawals []*ethtypes.Withdrawal) {
						if len(withdrawals) > 0 {
							withdrawals[0].Amount++ // Modify amount
						}
					},
					wantErr: "invalid withdrawals hash",
				},
			}

			for _, tc := range testCases {
				t.Run(tc.name, func(t *testing.T) {
					modifiedBlock := modifyWithdrawals(baseBlock, tc.modifyFn)
					// Provide empty receipts since we're testing withdrawal validation
					emptyReceipts := make(ethtypes.Receipts, 0)
					err := v.ValidateBlock(modifiedBlock, emptyReceipts, baseBlock.Hash())
					if err == nil {
						t.Error("expected error but got none")
						return
					}
					if !strings.Contains(err.Error(), tc.wantErr) {
						t.Errorf("got error %v, want error containing %q", err, tc.wantErr)
					}
				})
			}
		})
	}
}

func TestBlockValidator_ReceiptFailures(t *testing.T) {
	for _, chainData := range BlockTestChains {
		t.Run(chainData.Name, func(t *testing.T) {
			baseBlock, err := loadBlockFromJSON(chainData.BlockFixtureFile, t)
			if err != nil {
				t.Fatalf("Failed to load base block fixture for %s: %v", chainData.Name, err)
			}

			// Load the actual receipts
			baseReceipts, err := loadReceiptsFromJSON(chainData.ReceiptFixtureFile)
			if err != nil {
				t.Fatalf("Failed to load receipts fixture for %s: %v", chainData.Name, err)
			}

			cfg := &configuration.Configuration{
				ChainConfig: chainData.ChainConfig,
				Network:     chainData.Network,
				RosettaCfg: configuration.RosettaConfig{
					EnableTrustlessBlockValidation: true,
				},
			}
			v := NewEthereumValidator(cfg)

			testCases := []struct {
				name     string
				modifyFn func(ethtypes.Receipts) ethtypes.Receipts
				wantErr  string
			}{
				{
					name: "corrupt receipt status",
					modifyFn: func(receipts ethtypes.Receipts) ethtypes.Receipts {
						if len(receipts) > 0 {
							modifiedReceipts := make(ethtypes.Receipts, len(receipts))
							copy(modifiedReceipts, receipts)
							// Create a new receipt with modified status
							modifiedReceipt := &ethtypes.Receipt{
								Type:              receipts[0].Type,
								PostState:         receipts[0].PostState,
								Status:            1 - receipts[0].Status, // Flip status (0->1 or 1->0)
								CumulativeGasUsed: receipts[0].CumulativeGasUsed,
								Bloom:             receipts[0].Bloom,
								Logs:              receipts[0].Logs,
								TxHash:            receipts[0].TxHash,
								ContractAddress:   receipts[0].ContractAddress,
								GasUsed:           receipts[0].GasUsed,
								EffectiveGasPrice: receipts[0].EffectiveGasPrice,
								BlockHash:         receipts[0].BlockHash,
								BlockNumber:       receipts[0].BlockNumber,
								TransactionIndex:  receipts[0].TransactionIndex,
							}
							modifiedReceipts[0] = modifiedReceipt
							return modifiedReceipts
						}
						return receipts
					},
					wantErr: "invalid receipts hash",
				},
				{
					name: "corrupt cumulative gas used",
					modifyFn: func(receipts ethtypes.Receipts) ethtypes.Receipts {
						if len(receipts) > 0 {
							modifiedReceipts := make(ethtypes.Receipts, len(receipts))
							copy(modifiedReceipts, receipts)
							modifiedReceipt := &ethtypes.Receipt{
								Type:              receipts[0].Type,
								PostState:         receipts[0].PostState,
								Status:            receipts[0].Status,
								CumulativeGasUsed: receipts[0].CumulativeGasUsed + 1000, // Add 1000 to gas
								Bloom:             receipts[0].Bloom,
								Logs:              receipts[0].Logs,
								TxHash:            receipts[0].TxHash,
								ContractAddress:   receipts[0].ContractAddress,
								GasUsed:           receipts[0].GasUsed,
								EffectiveGasPrice: receipts[0].EffectiveGasPrice,
								BlockHash:         receipts[0].BlockHash,
								BlockNumber:       receipts[0].BlockNumber,
								TransactionIndex:  receipts[0].TransactionIndex,
							}
							modifiedReceipts[0] = modifiedReceipt
							return modifiedReceipts
						}
						return receipts
					},
					wantErr: "invalid receipts hash",
				},
				{
					name: "corrupt bloom filter",
					modifyFn: func(receipts ethtypes.Receipts) ethtypes.Receipts {
						if len(receipts) > 0 {
							modifiedReceipts := make(ethtypes.Receipts, len(receipts))
							copy(modifiedReceipts, receipts)
							// Create a modified bloom filter
							modifiedBloom := receipts[0].Bloom
							modifiedBloom[0] ^= 0xFF // Flip first byte
							modifiedReceipt := &ethtypes.Receipt{
								Type:              receipts[0].Type,
								PostState:         receipts[0].PostState,
								Status:            receipts[0].Status,
								CumulativeGasUsed: receipts[0].CumulativeGasUsed,
								Bloom:             modifiedBloom,
								Logs:              receipts[0].Logs,
								TxHash:            receipts[0].TxHash,
								ContractAddress:   receipts[0].ContractAddress,
								GasUsed:           receipts[0].GasUsed,
								EffectiveGasPrice: receipts[0].EffectiveGasPrice,
								BlockHash:         receipts[0].BlockHash,
								BlockNumber:       receipts[0].BlockNumber,
								TransactionIndex:  receipts[0].TransactionIndex,
							}
							modifiedReceipts[0] = modifiedReceipt
							return modifiedReceipts
						}
						return receipts
					},
					wantErr: "invalid receipts hash",
				},
				// Note: We don't test TxHash, BlockHash, BlockNumber, TransactionIndex, ContractAddress, or GasUsed
				// because these are "implementation fields" that are NOT part of the receipt hash calculation.
				// Only "consensus fields" (Type, Status, CumulativeGasUsed, Bloom, Logs) are used in the hash.
				{
					name: "add extra receipt",
					modifyFn: func(receipts ethtypes.Receipts) ethtypes.Receipts {
						if len(receipts) > 0 {
							modifiedReceipts := make(ethtypes.Receipts, len(receipts)+1)
							copy(modifiedReceipts, receipts)
							// Add a duplicate of the first receipt
							extraReceipt := &ethtypes.Receipt{
								Type:              receipts[0].Type,
								PostState:         receipts[0].PostState,
								Status:            receipts[0].Status,
								CumulativeGasUsed: receipts[0].CumulativeGasUsed,
								Bloom:             receipts[0].Bloom,
								Logs:              receipts[0].Logs,
								TxHash:            receipts[0].TxHash,
								ContractAddress:   receipts[0].ContractAddress,
								GasUsed:           receipts[0].GasUsed,
								EffectiveGasPrice: receipts[0].EffectiveGasPrice,
								BlockHash:         receipts[0].BlockHash,
								BlockNumber:       receipts[0].BlockNumber,
								TransactionIndex:  receipts[0].TransactionIndex,
							}
							modifiedReceipts[len(receipts)] = extraReceipt
							return modifiedReceipts
						}
						return receipts
					},
					wantErr: "invalid receipts hash",
				},
				{
					name: "remove receipt",
					modifyFn: func(receipts ethtypes.Receipts) ethtypes.Receipts {
						if len(receipts) > 1 {
							// Remove the last receipt
							return receipts[:len(receipts)-1]
						}
						return receipts
					},
					wantErr: "invalid receipts hash",
				},
			}

			// Also test the header corruption case
			headerTestCases := []struct {
				name     string
				modifyFn func(*ethtypes.Header)
				wantErr  string
			}{
				{
					name: "corrupt receipts root in header",
					modifyFn: func(h *ethtypes.Header) {
						h.ReceiptHash = common.HexToHash("0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef")
					},
					wantErr: "invalid block hash",
				},
			}

			// Test receipt modifications
			for _, tc := range testCases {
				t.Run(tc.name, func(t *testing.T) {
					modifiedReceipts := tc.modifyFn(baseReceipts)
					err := v.ValidateBlock(baseBlock, modifiedReceipts, baseBlock.Hash())
					if err == nil {
						t.Error("expected error but got none")
						return
					}
					if !strings.Contains(err.Error(), tc.wantErr) {
						t.Errorf("got error %v, want error containing %q", err, tc.wantErr)
					}
				})
			}

			// Test header modifications
			for _, tc := range headerTestCases {
				t.Run(tc.name, func(t *testing.T) {
					modifiedBlock := modifyBlockHeader(baseBlock, tc.modifyFn)
					err := v.ValidateBlock(modifiedBlock, baseReceipts, baseBlock.Hash())
					if err == nil {
						t.Error("expected error but got none")
						return
					}
					if !strings.Contains(err.Error(), tc.wantErr) {
						t.Errorf("got error %v, want error containing %q", err, tc.wantErr)
					}
				})
			}
		})
	}
}

func TestBlockValidator_Success(t *testing.T) {
	for _, chainData := range BlockTestChains {
		t.Run(chainData.Name, func(t *testing.T) {
			block, err := loadBlockFromJSON(chainData.BlockFixtureFile, t)
			if err != nil {
				t.Fatalf("Failed to load block fixture for %s: %v", chainData.Name, err)
			}

			cfg := &configuration.Configuration{
				ChainConfig: chainData.ChainConfig,
				Network:     chainData.Network,
			}
			v := NewEthereumValidator(cfg)

			// Load receipts for validation test
			receipts, err := loadReceiptsFromJSON(chainData.ReceiptFixtureFile)
			if err != nil {
				t.Fatalf("Failed to load receipts fixture for %s: %v", chainData.Name, err)
			}

			err = v.ValidateBlock(block, receipts, block.Hash())
			if err != nil {
				t.Errorf("ValidateBlock failed for %s: %v", chainData.Name, err)
			}
		})
	}
}
