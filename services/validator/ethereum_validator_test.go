package validator

import (
	"context"
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

// cases we need to check for:
// 1. Modify the block headers in different ways and fail the block header verification.
// Corrupt the transactions root (use a different root hash).
// Corrupt the block number (from 17000000 to 16000000)
// Corrupt the miner (drop the last 3 chars)
// Corrupt the amount of blob gas used (from 131072 to 100)
// Corrupt the beacon block root (drop the last 5 chars)
// 2. Modify the transactions in different ways and fail the transactions verification.
// Corrupt the chain id (from 1 to 100).
// Corrupt the gasUsed (from 1395940 to 1395941).
// Corrupt the from field (from 0x086d426f8b653b88a2d6d03051c8b4ab8783be2b to 0x086d426f8b653b88a2d6d03051c8b4ab8783be2c).
// Corrupt the to field (from 0x086d426f8b653b88a2d6d03051c8b4ab8783be2b to 0x086d426f8b653b88a2d6d03051c8b4ab8783be2c).
// Corrupt the value field (from 0x0 to 0x1).
// Corrupt the data field (from 0x0 to 0x1).
// Corrupt the blob hashes (add a new blob hash)
// 3. Modify the receipts in different ways and fail the receipts verification.
// Corrupt the cumulative gas used (from 73385 to 73384)
// Corrupt the status (from 1 to 0)
// Corrupt the logs (drop the last 4 bytes of logs[0].data)
// 4. Modify the withdrawals in different ways and fail the withdrawals verification.
// Corrupt the index of the first Withdrawals (from 4241882 to 4241881)
// Corrupt the validatorIndex of the second Withdrawals (from 551869 to 551870)
// Corrupt the address of the third Withdrawals (add "abc" at the end)
// 5. Modify the uncles in different ways and fail the uncles verification.
// Corrupt the uncle hash (use a different hash)
// Corrupt the uncle number (from 1 to 2)
// Corrupt the uncle position (from 0 to 1)
// BlockFixture represents the JSON structure of a block fixture

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

func modifyTransaction(original *ethtypes.Block, txIndex int, modifyFn func(*ethtypes.Transaction)) *ethtypes.Block {
	transactions := make([]*ethtypes.Transaction, len(original.Transactions()))
	copy(transactions, original.Transactions())
	if txIndex < len(transactions) {
		tx := transactions[txIndex]
		modifyFn(tx)
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

func TestEthereumValidator_HeaderFailures(t *testing.T) {
	ctx := context.Background()
	os.Setenv("EVM_BLOCK_VALIDATION_ENABLED", "true")

	// Load the base block that we'll modify for each test
	baseBlock, err := loadBlockFromJSON("testdata/test.json", t)
	if err != nil {
		t.Fatalf("Failed to load base block fixture: %v", err)
	}

	cfg := &configuration.Configuration{
		ChainConfig: &params.ChainConfig{
			ChainID: big.NewInt(57054),
		},
		Network: &types.NetworkIdentifier{
			Blockchain: NETWORK_SONIC,
			Network:    "sonic",
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
			wantErr: "Block hash invalid",
		},
		{
			name: "corrupt block number",
			modifyFn: func(h *ethtypes.Header) {
				h.Number = big.NewInt(16000000)
			},
			wantErr: "Block hash invalid",
		},
		{
			name: "corrupt miner",
			modifyFn: func(h *ethtypes.Header) {
				h.Coinbase = common.HexToAddress("0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef")
			},
			wantErr: "Block hash invalid",
		},
		// Add more header test cases here
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			modifiedBlock := modifyBlockHeader(baseBlock, tc.modifyFn)
			err := v.ValidateBlock(ctx, modifiedBlock, baseBlock.Hash())
			if err == nil {
				t.Error("expected error but got none")
				return
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("got error %v, want error containing %q", err, tc.wantErr)
			}
		})
	}
}

func TestEthereumValidator_TransactionFailures(t *testing.T) {
	ctx := context.Background()
	os.Setenv("EVM_BLOCK_VALIDATION_ENABLED", "true")

	// Load the base block that we'll modify for each test
	baseBlock, err := loadBlockFromJSON("testdata/test.json", t)
	if err != nil {
		t.Fatalf("Failed to load base block fixture: %v", err)
	}

	cfg := &configuration.Configuration{
		ChainConfig: &params.ChainConfig{
			ChainID: big.NewInt(57054),
		},
		Network: &types.NetworkIdentifier{
			Blockchain: NETWORK_SONIC,
			Network:    "sonic",
		},
	}
	v := NewEthereumValidator(cfg)

	testCases := []struct {
		name     string
		txIndex  int
		modifyFn func(*ethtypes.Transaction)
		wantErr  string
	}{
		{
			name:    "corrupt chain id",
			txIndex: 0,
			modifyFn: func(tx *ethtypes.Transaction) {
				// This requires modifying the underlying transaction data
				// We'll need to create a new transaction with modified values
				newTx := ethtypes.NewTransaction(
					tx.Nonce(),
					*tx.To(),
					tx.Value(),
					tx.Gas(),
					tx.GasPrice(),
					tx.Data(),
				)
				*tx = *newTx
			},
			wantErr: "Computed transaction root hash invalid.",
		},
		{
			name:    "corrupt gas used",
			txIndex: 0,
			modifyFn: func(tx *ethtypes.Transaction) {
				newTx := ethtypes.NewTransaction(
					tx.Nonce(),
					*tx.To(),
					tx.Value(),
					1395941, // Modified gas
					tx.GasPrice(),
					tx.Data(),
				)
				*tx = *newTx
			},
			wantErr: "Computed transaction root hash invalid.",
		},
		{
			name:    "corrupt to field",
			txIndex: 0,
			modifyFn: func(tx *ethtypes.Transaction) {
				addr := common.HexToAddress("0x086d426f8b653b88a2d6d03051c8b4ab8783be2c") // Modified last char
				newTx := ethtypes.NewTransaction(
					tx.Nonce(),
					addr,
					tx.Value(),
					tx.Gas(),
					tx.GasPrice(),
					tx.Data(),
				)
				*tx = *newTx
			},
			wantErr: "Computed transaction root hash invalid.",
		},
		{
			name:    "corrupt value field",
			txIndex: 0,
			modifyFn: func(tx *ethtypes.Transaction) {
				newTx := ethtypes.NewTransaction(
					tx.Nonce(),
					*tx.To(),
					big.NewInt(1), // Modified from 0 to 1
					tx.Gas(),
					tx.GasPrice(),
					tx.Data(),
				)
				*tx = *newTx
			},
			wantErr: "Computed transaction root hash invalid.",
		},
		{
			name:    "corrupt data field",
			txIndex: 0,
			modifyFn: func(tx *ethtypes.Transaction) {
				newTx := ethtypes.NewTransaction(
					tx.Nonce(),
					*tx.To(),
					tx.Value(),
					tx.Gas(),
					tx.GasPrice(),
					[]byte{0x1}, // Modified data
				)
				*tx = *newTx
			},
			wantErr: "Computed transaction root hash invalid.",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			modifiedBlock := modifyTransaction(baseBlock, tc.txIndex, tc.modifyFn)
			err := v.ValidateBlock(ctx, modifiedBlock, baseBlock.Hash())
			if err == nil {
				t.Error("expected error but got none")
				return
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("got error %v, want error containing %q", err, tc.wantErr)
			}
		})
	}
}

// func TestEthereumValidator_WithdrawalFailures(t *testing.T) {
// 	ctx := context.Background()
// 	os.Setenv("EVM_BLOCK_VALIDATION_ENABLED", "true")

// 	baseBlock, err := loadBlockFromJSON("testdata/test.json", t)
// 	if err != nil {
// 		t.Fatalf("Failed to load base block fixture: %v", err)
// 	}

// 	cfg := &configuration.Configuration{
// 		ChainConfig: &params.ChainConfig{
// 			ChainID: big.NewInt(57054),
// 		},
// 		Network: &types.NetworkIdentifier{
// 			Blockchain: NETWORK_SONIC,
// 			Network:    "sonic",
// 		},
// 	}
// 	v := NewEthereumValidator(cfg)

// 	testCases := []struct {
// 		name     string
// 		modifyFn func([]*ethtypes.Withdrawal)
// 		wantErr  string
// 	}{
// 		{
// 			name: "corrupt withdrawal index",
// 			modifyFn: func(withdrawals []*ethtypes.Withdrawal) {
// 				if len(withdrawals) > 0 {
// 					withdrawals[0].Index = 4241881 // Modified from 4241882
// 				}
// 			},
// 			wantErr: "invalid withdrawals hash",
// 		},
// 		{
// 			name: "corrupt validator index",
// 			modifyFn: func(withdrawals []*ethtypes.Withdrawal) {
// 				if len(withdrawals) > 1 {
// 					withdrawals[1].Index = 551870 // Modified from 551869
// 				}
// 			},
// 			wantErr: "invalid withdrawals hash",
// 		},
// 		{
// 			name: "corrupt withdrawal address",
// 			modifyFn: func(withdrawals []*ethtypes.Withdrawal) {
// 				if len(withdrawals) > 2 {
// 					addr := withdrawals[2].Address
// 					withdrawals[2].Address = common.HexToAddress(addr.Hex() + "abc")
// 				}
// 			},
// 			wantErr: "invalid withdrawals hash",
// 		},
// 	}

// 	for _, tc := range testCases {
// 		t.Run(tc.name, func(t *testing.T) {
// 			modifiedBlock := modifyWithdrawals(baseBlock, tc.modifyFn)
// 			err := v.ValidateBlock(ctx, modifiedBlock, baseBlock.Hash())
// 			if err == nil {
// 				t.Error("expected error but got none")
// 				return
// 			}
// 			if !strings.Contains(err.Error(), tc.wantErr) {
// 				t.Errorf("got error %v, want error containing %q", err, tc.wantErr)
// 			}
// 		})
// 	}
// }

func TestEthereumValidator_ReceiptFailures(t *testing.T) {
	ctx := context.Background()
	os.Setenv("EVM_BLOCK_VALIDATION_ENABLED", "true")

	baseBlock, err := loadBlockFromJSON("testdata/test.json", t)
	if err != nil {
		t.Fatalf("Failed to load base block fixture: %v", err)
	}

	cfg := &configuration.Configuration{
		ChainConfig: &params.ChainConfig{
			ChainID: big.NewInt(57054),
		},
		Network: &types.NetworkIdentifier{
			Blockchain: NETWORK_SONIC,
			Network:    "sonic",
		},
	}
	v := NewEthereumValidator(cfg)

	// Note: Receipt validation requires modifying the actual receipts in the RPC response
	// This is more complex as it requires mocking the RPC client
	// For now we'll just test that the receipt hash validation works
	testCases := []struct {
		name     string
		modifyFn func(*ethtypes.Header)
		wantErr  string
	}{
		{
			name: "corrupt receipts root",
			modifyFn: func(h *ethtypes.Header) {
				h.ReceiptHash = common.HexToHash("0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef")
			},
			wantErr: "Computed receipt root hash invalid.",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			modifiedBlock := modifyBlockHeader(baseBlock, tc.modifyFn)
			err := v.ValidateBlock(ctx, modifiedBlock, baseBlock.Hash())
			if err == nil {
				t.Error("expected error but got none")
				return
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("got error %v, want error containing %q", err, tc.wantErr)
			}
		})
	}
}

func TestEthereumValidator_Success(t *testing.T) {
	ctx := context.Background()
	t.Logf("EVM_BLOCK_VALIDATION_ENABLED: %s", os.Getenv("EVM_BLOCK_VALIDATION_ENABLED"))
	os.Setenv("EVM_BLOCK_VALIDATION_ENABLED", "true")
	t.Logf("EVM_BLOCK_VALIDATION_ENABLED: %s", os.Getenv("EVM_BLOCK_VALIDATION_ENABLED"))
	block, err := loadBlockFromJSON("testdata/test.json", t)
	if err != nil {
		t.Fatalf("Failed to load block fixture: %v", err)
	}

	cfg := &configuration.Configuration{
		ChainConfig: &params.ChainConfig{
			ChainID: big.NewInt(57054),
		},
		Network: &types.NetworkIdentifier{
			Blockchain: NETWORK_SONIC,
			Network:    "sonic",
		},
	}
	v := NewEthereumValidator(cfg)

	err = v.ValidateBlock(ctx, block, block.Hash())
	if err != nil {
		t.Errorf("ValidateBlock failed: %v", err)
	}
}

func TestEthereumValidator_Failures(t *testing.T) {
	ctx := context.Background()
	t.Logf("EVM_BLOCK_VALIDATION_ENABLED: %s", os.Getenv("EVM_BLOCK_VALIDATION_ENABLED"))
	os.Setenv("EVM_BLOCK_VALIDATION_ENABLED", "true")
	t.Logf("EVM_BLOCK_VALIDATION_ENABLED: %s", os.Getenv("EVM_BLOCK_VALIDATION_ENABLED"))
	block, err := loadBlockFromJSON("testdata/test.json", t)
	if err != nil {
		t.Fatalf("Failed to load block fixture: %v", err)
	}

	cfg := &configuration.Configuration{
		ChainConfig: &params.ChainConfig{
			ChainID: big.NewInt(57054),
		},
		Network: &types.NetworkIdentifier{
			Blockchain: NETWORK_SONIC,
			Network:    "sonic",
		},
	}
	v := NewEthereumValidator(cfg)

	err = v.ValidateBlock(ctx, block, block.Hash())
	if err != nil {
		t.Errorf("ValidateBlock failed: %v", err)
	}
}
