package validator

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
	"golang.org/x/xerrors"

	"strings"

	config "github.com/coinbase/rosetta-geth-sdk/configuration"
	"github.com/coinbase/rosetta-sdk-go/types"
	geth "github.com/ethereum/go-ethereum/common"
	EthTypes "github.com/ethereum/go-ethereum/core/types"
)

// Although we use a pinned version of the geth respository, these configs are the up to date configs from the geth
// repository.

// Result structs for eth_getProof
type (
	AccountResult struct {
		Address      geth.Address    `json:"address"`
		AccountProof []string        `json:"accountProof"`
		Balance      *hexutil.Big    `json:"balance"`
		CodeHash     geth.Hash       `json:"codeHash"`
		Nonce        hexutil.Uint64  `json:"nonce"`
		StorageHash  geth.Hash       `json:"storageHash"`
		StorageProof []StorageResult `json:"storageProof"`
	}
	StorageResult struct {
		Key   string       `json:"key"`
		Value *hexutil.Big `json:"value"`
		Proof []string     `json:"proof"`
	}
)

const (
	maxFromValidationRoutines = 10
	StorageHash               = "0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421"
)

type TrustlessValidator interface {
	ValidateBlock(ctx context.Context, block *EthTypes.Block, receipts EthTypes.Receipts, hash geth.Hash) error
	ValidateAccount(ctx context.Context, balanceResponse *types.AccountBalanceResponse, address string) error
	ValidateAccountState(ctx context.Context, result AccountResult, stateRoot geth.Hash, blockNumber *big.Int) error
	GetAccountProof(ctx context.Context, account geth.Address, blockNumber *big.Int) (AccountResult, error)
	GetBlockStateRoot(ctx context.Context, blockNumber *big.Int) (geth.Hash, error)
}
type (
	trustlessValidator struct {
		config *config.Configuration
	}
)

var (
	_                               TrustlessValidator = (*trustlessValidator)(nil)
	ErrInvalidBlockHash                                = xerrors.New("invalid block hash")
	ErrInvalidWithdrawalsHash                          = xerrors.New("invalid withdrawals hash")
	ErrInvalidTransactionsHash                         = xerrors.New("invalid transactions hash")
	ErrInvalidReceiptsHash                             = xerrors.New("invalid receipts hash")
	ErrAccountVerifyProofFailure                       = xerrors.New("account verification fails")
	ErrAccountNonceNotMatched                          = xerrors.New("mismatched account nonce")
	ErrAccountBalanceNotMatched                        = xerrors.New("mismatched account balance")
	ErrAccountStorageHashNotMatched                    = xerrors.New("mismatched account storage hash")
	ErrAccountCodeHashNotMatched                       = xerrors.New("mismatched account code hash")
	ErrInvalidFromField                                = xerrors.New("invalid from field")
)

func NewEthereumValidator(cfg *config.Configuration) TrustlessValidator {
	return &trustlessValidator{
		config: cfg,
	}
}

func (v *trustlessValidator) ValidateBlock(ctx context.Context, block *EthTypes.Block, receipts EthTypes.Receipts, hash geth.Hash) error {
	err := v.validateBlockHeader(ctx, block.Header(), hash)
	if err != nil {
		return xerrors.Errorf("block header validation error: %w", err)
	}

	err = v.validateWithdrawals(ctx, block.Withdrawals(), block.Header().WithdrawalsHash, block.Time())
	if err != nil {
		return xerrors.Errorf("withdrawals validation error: %w", err)
	}

	err = v.validateTransactions(ctx, block, block.Header().TxHash)
	if err != nil {
		return xerrors.Errorf("transactions validation error: %w", err)
	}

	err = v.validateReceipts(ctx, receipts, block.Header().ReceiptHash)
	if err != nil {
		return xerrors.Errorf("receipts validation error: %w", err)
	}

	log.Printf("Block %s successfully validated", block.Hash().String())

	return nil
}

func (v *trustlessValidator) ValidateAccount(ctx context.Context, balanceResponse *types.AccountBalanceResponse, address string) error {
	log.Printf("Running account validation for block %d and account %s", balanceResponse.BlockIdentifier.Index, address)
	addr := common.HexToAddress(address)

	result, err := v.GetAccountProof(ctx, addr, big.NewInt(balanceResponse.BlockIdentifier.Index))
	if err != nil {
		// Check if this is a proof window error - if so, skip validation gracefully
		// This is caused if we are using a node with a low proof window, will not be a case
		// On production and mainnet environments
		if isProofWindowError(err) {
			fmt.Printf("Skipping account validation: block %d is outside proof window\n", balanceResponse.BlockIdentifier.Index)
		} else {
			return xerrors.Errorf("%w", err)
		}
	} else {
		// Get the state root from the block
		stateRoot, err := v.GetBlockStateRoot(ctx, big.NewInt(balanceResponse.BlockIdentifier.Index))
		if err != nil {
			return xerrors.Errorf("failed to get block state root: %w", err)
		}

		// Run a Merkle Tree Proof on the state root and account
		err = v.ValidateAccountState(ctx, result, stateRoot, big.NewInt(balanceResponse.BlockIdentifier.Index))
		if err != nil {
			return err
		}
	}
	return nil
}

// isProofWindowError checks if the error is related to proof window limitations
func isProofWindowError(err error) bool {
	if err == nil {
		return false
	}

	errMsg := strings.ToLower(err.Error())
	return strings.Contains(errMsg, "distance to target block exceeds maximum proof window") ||
		strings.Contains(errMsg, "proof window") ||
		strings.Contains(errMsg, "too far from head") ||
		strings.Contains(errMsg, "block too old")
}

func (v *trustlessValidator) GetAccountProof(ctx context.Context, account geth.Address, blockNumber *big.Int) (AccountResult, error) {
	if v.config.GethURL == "" {
		return AccountResult{}, xerrors.Errorf("GethURL not configured")
	}

	client, err := ethclient.Dial(v.config.GethURL)
	if err != nil {
		return AccountResult{}, xerrors.Errorf("failed to connect to blockchain node at %s: %w", v.config.GethURL, err)
	}
	defer client.Close()

	blockNumberHex := hexutil.EncodeBig(blockNumber)

	// Get the account proof using eth_getProof
	var result AccountResult
	err = client.Client().CallContext(ctx, &result, "eth_getProof", account, []string{}, blockNumberHex)
	if err != nil {
		return AccountResult{}, xerrors.Errorf("failed to get account proof: %w", err)
	}

	// Verify that this proofResult is for the target account
	if result.Address != account {
		return AccountResult{}, xerrors.Errorf("the input proofResult has different account address, address in proof: %s, expected: %s", result.Address.Hex(), account)
	}
	log.Printf("Account proof retrieved for account %s", account.String())

	return result, nil
}

func (v *trustlessValidator) ValidateAccountState(ctx context.Context, result AccountResult, stateRoot geth.Hash, blockNumber *big.Int) error {

	// Create the in-memory DB state of the state trie proof
	proofDB := rawdb.NewMemoryDatabase()

	if len(result.AccountProof) == 0 {
		return xerrors.Errorf("no account proof provided: %w", ErrAccountVerifyProofFailure)
	}

	// decode the state root node
	stateRootNode, err := hexutil.Decode(result.AccountProof[0])
	if err != nil {
		return xerrors.Errorf("failed to decode first node: %w", err)
	}

	// state root should match the hash of the first proof node
	expectedStateRoot := crypto.Keccak256Hash(stateRootNode)
	if expectedStateRoot != stateRoot {
		return xerrors.Errorf("state root mismatch: provided=%s, expected=%s: %w", stateRoot.Hex(), expectedStateRoot.Hex(), ErrAccountVerifyProofFailure)
	}

	err = proofDB.Put(stateRoot.Bytes(), stateRootNode)
	if err != nil {
		return xerrors.Errorf("failed to store state root node: %w", err)
	}

	// store the rest of the proof nodes
	for i, node := range result.AccountProof[1:] {
		nodeData, err := hexutil.Decode(node)
		if err != nil {
			return xerrors.Errorf("failed to decode node %d: %w", i+1, err)
		}
		nodeHash := crypto.Keccak256(nodeData)
		err = proofDB.Put(nodeHash, nodeData)
		if err != nil {
			return xerrors.Errorf("failed to store proof node %d: %w", i+1, err)
		}
	}

	// Calculate the account hash
	accountHash := crypto.Keccak256(result.Address[:])

	// Use state_root_hash to walk through the returned proof to verify the state
	validAccountState, err := trie.VerifyProof(stateRoot, accountHash, proofDB)
	if err != nil {
		return xerrors.Errorf("VerifyProof fails with %v for the account %s: %w", err, result.Address, ErrAccountVerifyProofFailure)
	}

	if validAccountState == nil {
		return xerrors.Errorf("VerifyProof fails, the account %s is not included in the state trie: %w", result.Address, ErrAccountVerifyProofFailure)
	}

	var verifiedAccountState EthTypes.StateAccount
	if err := rlp.DecodeBytes(validAccountState, &verifiedAccountState); err != nil {
		return xerrors.Errorf("failed to rlp decode the verified account state: %w", err)
	}

	// check the input account proof is the same as the returned verified account state.
	if result.Nonce != hexutil.Uint64(verifiedAccountState.Nonce) {
		return xerrors.Errorf("account nonce is not matched, (nonce in proof=%v, nonce in verified result=%v): %w", result.Nonce, hexutil.Uint64(verifiedAccountState.Nonce), ErrAccountNonceNotMatched)
	}
	if verifiedAccountState.Balance.CmpBig(result.Balance.ToInt()) != 0 {
		return xerrors.Errorf("account balance is not matched, (balance in proof=%v, balance in verified result=%v): %w", result.Balance.ToInt(), verifiedAccountState.Balance, ErrAccountBalanceNotMatched)
	}
	if result.StorageHash != verifiedAccountState.Root {
		return xerrors.Errorf("account storage hash is not matched, (storage hash in proof=%v, storage hash in verified result=%v): %w", result.StorageHash, verifiedAccountState.Root, ErrAccountStorageHashNotMatched)
	}
	if !bytes.Equal(result.CodeHash.Bytes(), verifiedAccountState.CodeHash) {
		return xerrors.Errorf("account code hash is not matched, (code hash in proof=%v, code hash in verified result=%v): %w", result.CodeHash.Bytes(), verifiedAccountState.CodeHash, ErrAccountCodeHashNotMatched)
	}

	log.Printf("Account state validated for account %s", result.Address.String())

	return nil
}

func (v *trustlessValidator) validateBlockHeader(ctx context.Context, header *EthTypes.Header, actualHash geth.Hash) error {
	if header == nil {
		return xerrors.New("block header is nil")
	}

	// Note that Hash returns the block hash of the header, which is simply the keccak256 hash of its RLP encoding.
	// We expect that the block hash recomputed following the protocol should match the one from the payload itself.
	expectedHash := header.Hash()
	if expectedHash != actualHash {
		return xerrors.Errorf("Block hash invalid. One or more components of the block header are tampered (expected=%v, actual=%v): %w", expectedHash, actualHash, ErrInvalidBlockHash)
	}

	return nil
}

// Verify the withdrawals in the block with the withdrawals trie root hash.
func (v *trustlessValidator) validateWithdrawals(ctx context.Context, withdrawals EthTypes.Withdrawals, withdrawalsRoot *geth.Hash, blockTimestamp uint64) error {
	if withdrawalsRoot != nil {
		// https://github.com/ethereum-optimism/op-geth/blame/36501a7023fd85f3492a1af6f1474a0113bb83fe//core/block_validator.go#L76-L79
		// Optimism based chains use the storage root as the withdrawals root and do not have withdrawals.
		if isOptimismWithdrawal(withdrawals, withdrawalsRoot) {

			// For Optimism, we trust the withdrawalsRoot from the block header
			// as they use a different withdrawal format that doesn't follow standard Ethereum
			fmt.Printf("Detected Optimism-style withdrawal, skipping standard validation\n")
			return nil
		}

		// This is how geth calculates the withdrawals trie hash. We just leverage this function of geth to recompute it.
		if actualHash := EthTypes.DeriveSha(withdrawals, trie.NewStackTrie(nil)); actualHash != *withdrawalsRoot {
			return xerrors.Errorf("Withdrawals root hash mismatch (expected=%x, actual=%x): %w", withdrawalsRoot, actualHash, ErrInvalidWithdrawalsHash)
		}
	} else if len(withdrawals) != 0 {
		return xerrors.Errorf("unexpected withdrawals in block body")
	}

	return nil
}

// isOptimismWithdrawal checks if the withdrawals follow Optimism's non-standard format
// Optimism uses 0xffffffffffffffff (max uint64) for both index and validatorIndex
// Berachain is an example of an Optimism based chain.
func isOptimismWithdrawal(withdrawals EthTypes.Withdrawals, withdrawalsRoot *geth.Hash) bool {
	const OptimismWithdrawalsRoot = "0xdaaca84d096cebe9194d542b918abce37bc8e2d9cc85d5be4d6a9c947552a6ef"
	if len(withdrawals) == 0 && (withdrawalsRoot.String() == OptimismWithdrawalsRoot) {
		return true
	}

	return false
}

// Verify all the transactions in the block with the transaction trie root hash.
func (v *trustlessValidator) validateTransactions(ctx context.Context, block *EthTypes.Block, transactionsRoot geth.Hash) error {
	transactions := block.Transactions()
	numTxs := len(transactions)

	// This is how geth calculates the transaction trie hash. We just leverage this function of geth to recompute it.
	if actualHash := EthTypes.DeriveSha(transactions, trie.NewStackTrie(nil)); actualHash != transactionsRoot {
		return xerrors.Errorf("Computed transaction root hash invalid. One or more transactions are tampered (expected=%x, actual=%x): %w", transactionsRoot, actualHash, ErrInvalidTransactionsHash)
	}

	signer := v.GetSigner(block)
	if signer != nil {
		// Create channels for error handling and throttling
		errCh := make(chan error, numTxs)
		sem := make(chan struct{}, maxFromValidationRoutines)
		var wg sync.WaitGroup

		// Process each transaction
		for i := 0; i < numTxs; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()

				// Acquire semaphore
				sem <- struct{}{}
				defer func() { <-sem }()

				// Get transaction
				tx := transactions[idx]

				// Skip validation for unsupported transaction types
				if tx.Type() == EthTypes.SetCodeTxType {
					return
				}

				// Get the cached sender from the transaction
				sender, err := signer.Sender(tx)
				if err != nil {
					errCh <- xerrors.Errorf("transaction %d: failed to get sender: %w", idx, err)
					return
				}

				// Compare with the actual sender
				if err := v.isValidFromField(sender, tx, signer); err != nil {
					errCh <- xerrors.Errorf("transaction %d: %w", idx, err)
				}
			}(i)
		}

		// Wait for all goroutines to finish
		wg.Wait()
		close(errCh)

		// Collect any errors
		var errs []error
		for err := range errCh {
			errs = append(errs, err)
		}

		// Return combined errors if any
		if len(errs) > 0 {
			return xerrors.Errorf("validation errors: %v", errs)
		}
	}

	return nil
}

// Recalculate the from field from the signer and r,s,v, and compare recalculated from field with the actual from field
// to ensure it was not tampered with.
func (v *trustlessValidator) isValidFromField(actualFrom geth.Address, gethTransaction *EthTypes.Transaction, signer EthTypes.Signer) error {
	// if v.config.RosettaCfg.Env == config.EnvProduction {
	// 	return nil
	// }

	// Signer doesn't support this transaction type yet.
	if gethTransaction.Type() == EthTypes.SetCodeTxType {
		return nil
	}

	expectedFrom, err := EthTypes.Sender(signer, gethTransaction)
	if err != nil {
		return xerrors.Errorf("failed to recalculate sender: %w)", err)
	}
	if actualFrom != expectedFrom {
		return xerrors.Errorf("from mismatch (expected=%x, actual=%x): %w", expectedFrom.String(), actualFrom, ErrInvalidFromField)
	}
	return nil
}

func (v *trustlessValidator) GetSigner(block *EthTypes.Block) EthTypes.Signer {
	if v.config == nil {
		return nil
	}

	// since we don't have a complete chain config for all chains we use
	// we can't use MakeSigner, which is the best way of getting the correct
	// signer, but LatestSignerForChainID is the next best thing.
	signer := EthTypes.LatestSignerForChainID(v.config.ChainConfig.ChainID)
	return signer
}

// Verify all the receipts in the block with the receipt trie root hash.
func (v *trustlessValidator) validateReceipts(ctx context.Context, receipts EthTypes.Receipts, receiptsRoot geth.Hash) error {
	numTxs := len(receipts)

	gethReceipts := receipts[:numTxs]

	// This is how geth calculates the receipt trie hash. We just leverage this function of geth to recompute it.
	if actualHash := EthTypes.DeriveSha(gethReceipts, trie.NewStackTrie(nil)); actualHash != receiptsRoot {
		return xerrors.Errorf("Computed receipt root hash invalid. One or more receipts are tampered (expected=%x, actual=%x): %w", receiptsRoot, actualHash, ErrInvalidReceiptsHash)
	}

	return nil
}

func (v *trustlessValidator) GetBlockStateRoot(ctx context.Context, blockNumber *big.Int) (geth.Hash, error) {
	// Connect to the configured blockchain node
	if v.config.GethURL == "" {
		return geth.Hash{}, xerrors.Errorf("GethURL not configured")
	}

	client, err := ethclient.Dial(v.config.GethURL)
	if err != nil {
		return geth.Hash{}, xerrors.Errorf("failed to connect to blockchain node at %s: %w", v.config.GethURL, err)
	}
	defer client.Close()

	// Get the block by number
	block, err := client.BlockByNumber(ctx, blockNumber)
	if err != nil {
		return geth.Hash{}, xerrors.Errorf("failed to get block: %w", err)
	}

	// Extract state root from block header
	stateRoot := block.Header().Root

	log.Printf("Block state root retrieved for block %d: %s", blockNumber, stateRoot.Hex())

	return stateRoot, nil
}
