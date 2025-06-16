package validator

import (
	"bytes"
	"context"
	"fmt"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
	"golang.org/x/xerrors"

	config "github.com/coinbase/rosetta-geth-sdk/configuration"
	geth "github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	gethparams "github.com/ethereum/go-ethereum/params"
)

// Although we use a pinned version of the geth respository, these configs are the up to date configs from the geth
// repository.
var (
	MainnetTerminalTotalDifficulty, _ = new(big.Int).SetString("58_750_000_000_000_000_000_000", 0)

	// MainnetChainConfig is the chain parameters to run a node on the main network.
	MainnetChainConfig = &gethparams.ChainConfig{
		ChainID:                 big.NewInt(1),
		HomesteadBlock:          big.NewInt(1_150_000),
		DAOForkBlock:            big.NewInt(1_920_000),
		DAOForkSupport:          true,
		EIP150Block:             big.NewInt(2_463_000),
		EIP155Block:             big.NewInt(2_675_000),
		EIP158Block:             big.NewInt(2_675_000),
		ByzantiumBlock:          big.NewInt(4_370_000),
		ConstantinopleBlock:     big.NewInt(7_280_000),
		PetersburgBlock:         big.NewInt(7_280_000),
		IstanbulBlock:           big.NewInt(9_069_000),
		MuirGlacierBlock:        big.NewInt(9_200_000),
		BerlinBlock:             big.NewInt(12_244_000),
		LondonBlock:             big.NewInt(12_965_000),
		ArrowGlacierBlock:       big.NewInt(13_773_000),
		GrayGlacierBlock:        big.NewInt(15_050_000),
		TerminalTotalDifficulty: MainnetTerminalTotalDifficulty, // 58_750_000_000_000_000_000_000
		ShanghaiTime:            newUint64(1681338455),
		CancunTime:              newUint64(1710338135),
		DepositContractAddress:  geth.HexToAddress("0x00000000219ab540356cbb839cbe05303d7705fa"),
		Ethash:                  new(gethparams.EthashConfig),
	}
	// HoleskyChainConfig contains the chain parameters to run a node on the Holesky test network.
	HoleskyChainConfig = &gethparams.ChainConfig{
		ChainID:                 big.NewInt(17000),
		HomesteadBlock:          big.NewInt(0),
		DAOForkBlock:            nil,
		DAOForkSupport:          true,
		EIP150Block:             big.NewInt(0),
		EIP155Block:             big.NewInt(0),
		EIP158Block:             big.NewInt(0),
		ByzantiumBlock:          big.NewInt(0),
		ConstantinopleBlock:     big.NewInt(0),
		PetersburgBlock:         big.NewInt(0),
		IstanbulBlock:           big.NewInt(0),
		MuirGlacierBlock:        nil,
		BerlinBlock:             big.NewInt(0),
		LondonBlock:             big.NewInt(0),
		ArrowGlacierBlock:       nil,
		GrayGlacierBlock:        nil,
		TerminalTotalDifficulty: big.NewInt(0),
		MergeNetsplitBlock:      nil,
		ShanghaiTime:            newUint64(1696000704),
		CancunTime:              newUint64(1707305664),
		Ethash:                  new(gethparams.EthashConfig),
	}
	// SepoliaChainConfig contains the chain parameters to run a node on the Sepolia test network.
	SepoliaChainConfig = &gethparams.ChainConfig{
		ChainID:                 big.NewInt(11155111),
		HomesteadBlock:          big.NewInt(0),
		DAOForkBlock:            nil,
		DAOForkSupport:          true,
		EIP150Block:             big.NewInt(0),
		EIP155Block:             big.NewInt(0),
		EIP158Block:             big.NewInt(0),
		ByzantiumBlock:          big.NewInt(0),
		ConstantinopleBlock:     big.NewInt(0),
		PetersburgBlock:         big.NewInt(0),
		IstanbulBlock:           big.NewInt(0),
		MuirGlacierBlock:        big.NewInt(0),
		BerlinBlock:             big.NewInt(0),
		LondonBlock:             big.NewInt(0),
		ArrowGlacierBlock:       nil,
		GrayGlacierBlock:        nil,
		TerminalTotalDifficulty: big.NewInt(17_000_000_000_000_000),
		MergeNetsplitBlock:      big.NewInt(1735371),
		ShanghaiTime:            newUint64(1677557088),
		CancunTime:              newUint64(1706655072),
		Ethash:                  new(gethparams.EthashConfig),
	}
	// GoerliChainConfig contains the chain parameters to run a node on the Görli test network.
	GoerliChainConfig = &gethparams.ChainConfig{
		ChainID:                 big.NewInt(5),
		HomesteadBlock:          big.NewInt(0),
		DAOForkBlock:            nil,
		DAOForkSupport:          true,
		EIP150Block:             big.NewInt(0),
		EIP155Block:             big.NewInt(0),
		EIP158Block:             big.NewInt(0),
		ByzantiumBlock:          big.NewInt(0),
		ConstantinopleBlock:     big.NewInt(0),
		PetersburgBlock:         big.NewInt(0),
		IstanbulBlock:           big.NewInt(1_561_651),
		MuirGlacierBlock:        nil,
		BerlinBlock:             big.NewInt(4_460_644),
		LondonBlock:             big.NewInt(5_062_605),
		ArrowGlacierBlock:       nil,
		TerminalTotalDifficulty: big.NewInt(10_790_000),
		ShanghaiTime:            newUint64(1678832736),
		CancunTime:              newUint64(1705473120),
		Clique: &gethparams.CliqueConfig{
			Period: 15,
			Epoch:  30000,
		},
	}
)

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
	NETWORK_ETHEREUM_MAINNET = "ethereum"
	NETWORK_ETHEREUM_GOERLI  = "goerli"
	NETWORK_ETHEREUM_SEPOLIA = "sepolia"
	NETWORK_ETHEREUM_HOLESKY = "holesky"
	NETWORK_SONIC            = "sonic"
)

func newUint64(val uint64) *uint64 { return &val }

const (
	maxFromValidationRoutines = 10
)

type TrustlessValidator interface {
	ValidateBlock(ctx context.Context, block *ethtypes.Block, hash geth.Hash) error
	ValidateAccountState(ctx context.Context, result AccountResult, stateRoot geth.Hash, blockNumber *big.Int) error
	GetAccountProof(ctx context.Context, account geth.Address, blockNumber *big.Int) (AccountResult, error)
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

func (v *trustlessValidator) ValidateBlock(ctx context.Context, block *ethtypes.Block, hash geth.Hash) error {
	// if block.Skipped {
	// 	// By definition skipped blocks do not need to be validated.
	// 	return nil
	// }

	// ethereumBlock := block.GetEthereum()
	// if ethereumBlock == nil {
	// 	return xerrors.New("not an ethereum block")
	// }

	// Verify the block header.
	err := v.validateBlockHeader(ctx, block.Header(), hash)
	if err != nil {
		return xerrors.Errorf("block header validation error: %w", err)
	}

	// Verify the Withdrawals in the block.
	err = v.validateWithdrawals(ctx, block.Withdrawals(), block.Header().WithdrawalsHash, block.Time())
	if err != nil {
		return xerrors.Errorf("withdrawals validation error: %w", err)
	}

	// Verify the transactions in the block.
	err = v.validateTransactions(ctx, block, block.Header().TxHash)
	if err != nil {
		return xerrors.Errorf("transactions validation error: %w", err)
	}

	// Verify the receipts in the block.
	err = v.validateReceipts(ctx, block.Transactions(), block.Header().ReceiptHash)
	if err != nil {
		return xerrors.Errorf("receipts validation error: %w", err)
	}
	fmt.Println("valid!")

	return nil
}

func (v *trustlessValidator) GetAccountProof(ctx context.Context, account geth.Address, blockNumber *big.Int) (AccountResult, error) {
	// Connect to the configured blockchain node
	if v.config.GethURL == "" {
		return AccountResult{}, xerrors.Errorf("GethURL not configured")
	}

	client, err := ethclient.Dial(v.config.GethURL)
	if err != nil {
		return AccountResult{}, xerrors.Errorf("failed to connect to blockchain node at %s: %w", v.config.GethURL, err)
	}
	defer client.Close()

	// Convert block number to hex string
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

	return result, nil
}

func (v *trustlessValidator) ValidateAccountState(ctx context.Context, result AccountResult, stateRoot geth.Hash, blockNumber *big.Int) error {

	// Create the in-memory DB state of the state trie proof
	proofDB := rawdb.NewMemoryDatabase()

	if len(result.AccountProof) == 0 {
		return xerrors.Errorf("no account proof provided: %w", ErrAccountVerifyProofFailure)
	}

	// First, decode and validate the state root node
	firstNodeData, err := hexutil.Decode(result.AccountProof[0])
	if err != nil {
		return xerrors.Errorf("failed to decode first node: %w", err)
	}

	// The provided state root should match the hash of the first proof node
	expectedStateRoot := crypto.Keccak256Hash(firstNodeData)
	if expectedStateRoot != stateRoot {
		return xerrors.Errorf("state root mismatch: provided=%s, expected=%s: %w", stateRoot.Hex(), expectedStateRoot.Hex(), ErrAccountVerifyProofFailure)
	}

	// Store the state root node
	err = proofDB.Put(stateRoot.Bytes(), firstNodeData)
	if err != nil {
		return xerrors.Errorf("failed to store state root node: %w", err)
	}

	// Then store the rest of the proof nodes
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

	// Calculate the account hash - this is the key in the state trie
	accountHash := crypto.Keccak256(result.Address[:])

	// Use state_root_hash to walk through the returned proof to verify the state
	validAccountState, err := trie.VerifyProof(stateRoot, accountHash, proofDB)
	if err != nil {
		return xerrors.Errorf("VerifyProof fails with %v for the account %s: %w", err, result.Address, ErrAccountVerifyProofFailure)
	}

	// If the err is nil and returned account state is nil, then this means that the state trie doesn't include the target account, and the verification fails.
	if validAccountState == nil {
		return xerrors.Errorf("VerifyProof fails, the account %s is not included in the state trie: %w", result.Address, ErrAccountVerifyProofFailure)
	}

	// If successful, decode the stored account state, and return it.
	var verifiedAccountState ethtypes.StateAccount
	if err := rlp.DecodeBytes(validAccountState, &verifiedAccountState); err != nil {
		return xerrors.Errorf("failed to rlp decode the verified account state: %w", err)
	}

	// After the verification is successful, we further check the input account proof is the same as the returned verified account state.
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

	return nil
}

func (v *trustlessValidator) validateBlockHeader(ctx context.Context, header *ethtypes.Header, actualHash geth.Hash) error {
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
func (v *trustlessValidator) validateWithdrawals(ctx context.Context, withdrawals ethtypes.Withdrawals, withdrawalsRoot *geth.Hash, blockTimestamp uint64) error {
	if withdrawalsRoot != nil {
		// https://github.com/ethereum-optimism/op-geth/blame/36501a7023fd85f3492a1af6f1474a0113bb83fe//core/block_validator.go#L76-L79
		// if isOptimismIsthmus(v.config.Network(), blockTimestamp) {
		// 	if len(withdrawals) > 0 {
		// 		return xerrors.Errorf("no withdrawal block-operations allowed, withdrawalsRoot is set to storage root")
		// 	}
		// 	return nil
		// }

		// This is how geth calculates the withdrawals trie hash. We just leverage this function of geth to recompute it.
		if actualHash := ethtypes.DeriveSha(withdrawals, trie.NewStackTrie(nil)); actualHash != *withdrawalsRoot {
			return xerrors.Errorf("Withdrawals root hash mismatch (expected=%x, actual=%x): %w", withdrawalsRoot, actualHash, ErrInvalidWithdrawalsHash)
		}
	} else if len(withdrawals) != 0 {
		return xerrors.Errorf("unexpected withdrawals in block body")
	}

	return nil
}

// Verify all the transactions in the block with the transaction trie root hash.
func (v *trustlessValidator) validateTransactions(ctx context.Context, block *ethtypes.Block, transactionsRoot geth.Hash) error {
	transactions := block.Transactions()
	numTxs := len(transactions)

	fmt.Printf("validateTransactions: numTxs=%d, chainID=%v\n", numTxs, v.config.ChainConfig.ChainID)
	// case common.Blockchain_BLOCKCHAIN_POLYGON:
	// 	// For Polygon, it is possible that there is a state-sync transaction at the end of transaction array.
	// 	// It is an internal transaction used to read data from Ethereum in Polygon. It is an internal transaction, and
	// 	// it is not used to calculate the transaction trie root hash. Once we identify such transaction, we need to
	// 	// exclude it from the transaction and receipt verification.
	// 	if hasStateSyncTx(transactions) {
	// 		numTxs = numTxs - 1
	// 	}
	// }

	// This is how geth calculates the transaction trie hash. We just leverage this function of geth to recompute it.
	if actualHash := ethtypes.DeriveSha(transactions, trie.NewStackTrie(nil)); actualHash != transactionsRoot {
		return xerrors.Errorf("Computed transaction root hash invalid. One or more transactions are tampered (expected=%x, actual=%x): %w", transactionsRoot, actualHash, ErrInvalidTransactionsHash)
	}

	signer := v.GetSigner(block)
	fmt.Printf("signer type: %T, chainID: %v\n", signer, signer.ChainID())
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

				fmt.Printf("Transaction %d: Type=%d\n", idx, tx.Type())

				// Skip validation for unsupported transaction types
				if tx.Type() == ethtypes.SetCodeTxType {
					fmt.Printf("Transaction %d: Skipping SetCode transaction\n", idx)
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
			fmt.Printf("Validation error: %v\n", err)
		}

		// Return combined errors if any
		if len(errs) > 0 {
			return xerrors.Errorf("validation errors: %v", errs)
		}
	}

	fmt.Println("All transactions validated successfully!")
	return nil
}

// TODO: uncomment and implement this
// func hasStateSyncTx(transactions []*api.EthereumTransaction) bool {
// 	num := len(transactions)
// 	if num == 0 {
// 		return false
// 	}

// 	// This is an state-sync transaction in Polygon. It only appears as the last transaction.
// 	return transactions[num-1].From == ethNullAddress && transactions[num-1].To == ethNullAddress
// }

// Recalculate the from field from the signer and r,s,v, and compare recalculated from field with the actual from field
// to ensure it was not tampered with.
func (v *trustlessValidator) isValidFromField(actualFrom geth.Address, gethTransaction *ethtypes.Transaction, signer ethtypes.Signer) error {
	//if v.config.RosettaCfg == config.EnvProduction {
	//	return nil
	//}

	// Signer doesn't support this transaction type yet.
	if gethTransaction.Type() == ethtypes.SetCodeTxType {
		return nil
	}

	expectedFrom, err := ethtypes.Sender(signer, gethTransaction)
	if err != nil {
		return xerrors.Errorf("failed to recalculate sender: %w)", err)
	}
	if actualFrom != expectedFrom {
		return xerrors.Errorf("from mismatch (expected=%x, actual=%x): %w", expectedFrom.String(), actualFrom, ErrInvalidFromField)
	}
	return nil
}

func (v *trustlessValidator) GetSigner(block *ethtypes.Block) ethtypes.Signer {
	if v.config == nil {
		fmt.Printf("Config is nil!\n")
		return nil
	}

	bn := block.Header().Number
	bts := uint64(block.Time())

	fmt.Printf("Block number: %v, Block time: %v\n", bn, bts)

	// Use the chain config from the validator's configuration
	cfg := v.config.ChainConfig
	if cfg == nil {
		fmt.Printf("Chain config is nil, falling back to network-based config\n")
		// Fallback to determining chain config based on network
		fmt.Printf("Network: %v\n", v.config.Network.Network)
		switch v.config.Network.Blockchain {
		case NETWORK_ETHEREUM_MAINNET:
			cfg = MainnetChainConfig
		case NETWORK_ETHEREUM_GOERLI:
			cfg = GoerliChainConfig
		case NETWORK_ETHEREUM_SEPOLIA:
			cfg = SepoliaChainConfig
		case NETWORK_ETHEREUM_HOLESKY:
			cfg = HoleskyChainConfig
		default:
			fmt.Printf("Unknown network: %v, using generic EVM configuration\n", v.config.Network.Network)
			// For unknown networks (including Sonic and other EVM chains),
			// create a generic EVM chain config that supports modern features
			if v.config.ChainConfig != nil && v.config.ChainConfig.ChainID != nil {
				fmt.Printf("Using provided chain ID: %v\n", v.config.ChainConfig.ChainID)
				cfg = &gethparams.ChainConfig{
					ChainID:             v.config.ChainConfig.ChainID,
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
			} else {
				fmt.Printf("No chain configuration available\n")
				return nil
			}
		}
	} else {
		fmt.Printf("Using provided chain config with chain ID: %v\n", cfg.ChainID)
	}

	signer := ethtypes.MakeSigner(cfg, bn, bts)
	fmt.Printf("Created signer of type %T with chain ID %v\n", signer, signer.ChainID())
	return signer
}

// Verify all the receipts in the block with the receipt trie root hash.
func (v *trustlessValidator) validateReceipts(ctx context.Context, transactions []*ethtypes.Transaction, receiptsRoot geth.Hash) error {
	// Similar to validateTransactions(), we need to handle the receipts in state-sync transactions of Polygon.
	numTxs := len(transactions)
	// if v.config.Network.Blockchain == common.Blockchain_BLOCKCHAIN_POLYGON && hasStateSyncTx(transactions) {
	// 	numTxs = numTxs - 1
	// }

	// Connect to the configured blockchain node
	if v.config.GethURL == "" {
		return xerrors.Errorf("GethURL not configured for receipt validation")
	}

	client, err := ethclient.Dial(v.config.GethURL)
	if err != nil {
		return xerrors.Errorf("failed to connect to blockchain node at %s: %w", v.config.GethURL, err)
	}
	defer client.Close()

	// make receipts list
	gethReceipts := make(ethtypes.Receipts, numTxs)
	for i := 0; i < numTxs; i++ {
		receipt, err := client.TransactionReceipt(ctx, transactions[i].Hash())
		if err != nil {
			fmt.Printf("Failed to fetch receipt for tx %s: %v\n", transactions[i].Hash().Hex(), err)
			return xerrors.Errorf("failed to fetch receipt for transaction %s: %w", transactions[i].Hash().Hex(), err)
		}
		gethReceipts[i] = receipt
		fmt.Printf("Tx %s used %d gas\n", transactions[i].Hash().Hex(), receipt.GasUsed)
	}

	// This is how geth calculates the receipt trie hash. We just leverage this function of geth to recompute it.
	if actualHash := ethtypes.DeriveSha(gethReceipts, trie.NewStackTrie(nil)); actualHash != receiptsRoot {
		return xerrors.Errorf("Computed receipt root hash invalid. One or more receipts are tampered (expected=%x, actual=%x): %w", receiptsRoot, actualHash, ErrInvalidReceiptsHash)
	}

	return nil
}
