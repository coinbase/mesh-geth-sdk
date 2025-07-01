// Copyright 2022 Coinbase, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"math/big"

	goEthereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	EthTypes "github.com/ethereum/go-ethereum/core/types"

	client "github.com/coinbase/rosetta-geth-sdk/client"
	"github.com/coinbase/rosetta-geth-sdk/configuration"
	construction "github.com/coinbase/rosetta-geth-sdk/services/construction"
	validator "github.com/coinbase/rosetta-geth-sdk/services/validator"
	AssetTypes "github.com/coinbase/rosetta-geth-sdk/types"

	RosettaTypes "github.com/coinbase/rosetta-sdk-go/types"
	"github.com/coinbase/rosetta-sdk-go/utils"

	lru "github.com/hashicorp/golang-lru"
)

const (
	// LRUCacheSize determines how many contract currencies we cache
	LRUCacheSize = 100

	OpenEthereumTrace = iota // == 2
)

// BlockAPIService implements the server.BlockAPIServicer interface.
type BlockAPIService struct {
	config        *configuration.Configuration
	client        construction.Client
	currencyCache *lru.Cache
}

// NewBlockAPIService creates a new instance of a BlockAPIService.
func NewBlockAPIService(
	cfg *configuration.Configuration,
	client construction.Client,
) *BlockAPIService {
	currencyCache, err := lru.New(LRUCacheSize)
	if err != nil {
		log.Fatalln(err)
	}

	return &BlockAPIService{
		config:        cfg,
		client:        client,
		currencyCache: currencyCache,
	}
}

func (s *BlockAPIService) populateTransactions(
	ctx context.Context,
	blockIdentifier *RosettaTypes.BlockIdentifier,
	block *EthTypes.Block,
	loadedTransactions []*client.LoadedTransaction,
) ([]*RosettaTypes.Transaction, error) {
	rosettaCfg := s.client.GetRosettaConfig()
	transactions := make([]*RosettaTypes.Transaction, 0)

	if rosettaCfg.SupportRewardTx {
		// Compute reward transaction (block + uncle reward)
		rewardTx := s.client.BlockRewardTransaction(
			blockIdentifier,
			block.Coinbase().String(),
			block.Uncles(),
		)
		transactions = append(transactions, rewardTx)
	}

	for _, tx := range loadedTransactions {
		if tx.IsBridgedTxn {
			// Bridge tx is already handled in PopulateCrossChainTransactions flow
			continue
		}
		transaction, err := s.PopulateTransaction(ctx, tx)
		if err != nil {
			return nil, fmt.Errorf("cannot parse %s: %w", tx.TxHash, err)
		}
		transactions = append(transactions, transaction)
	}

	return transactions, nil
}

// getCurrencyFromNodeOrCache checks if the currency is in the cache and fetches it from the node if not.
func (s *BlockAPIService) getCurrencyFromNodeOrCache(address common.Address, addressStr string) (*client.ContractCurrency, error) {
	if cachedCurrency, found := s.currencyCache.Get(addressStr); found {
		return cachedCurrency.(*client.ContractCurrency), nil
	}
	currency, err := s.client.GetContractCurrency(address, true)
	if err != nil {
		return nil, err
	}
	s.currencyCache.Add(addressStr, currency)
	return currency, nil
}

func (s *BlockAPIService) PopulateTransaction(
	ctx context.Context,
	tx *client.LoadedTransaction,
) (*RosettaTypes.Transaction, error) {
	ops, err := s.client.ParseOps(tx)
	if err != nil {
		return nil, err
	}

	var receiptLogs []*EthTypes.Log
	if tx.Receipt != nil {
		receiptLogs = tx.Receipt.Logs
	}

	rosettaConfig := s.client.GetRosettaConfig()
	filterTokens := rosettaConfig.FilterTokens

	if rosettaConfig.TokenWhitelistAccessor != nil {
		whitelist, err := rosettaConfig.TokenWhitelistAccessor()
		if err != nil {
			return nil, fmt.Errorf("could not get token whitelist: %w", err)
		}

		rosettaConfig.TokenWhiteList = whitelist
	}

	tokenWhiteList := rosettaConfig.TokenWhiteList

	useTokenWhiteListMetadata := rosettaConfig.UseTokenWhiteListMetadata
	indexUnknownTokens := s.config.RosettaCfg.IndexUnknownTokens

	// Compute tx operations via tx.Receipt logs for ERC20 transfer, mint and burn
	for _, log := range receiptLogs {
		contractAddress := log.Address.String()

		if s.client.SkipTxReceiptParsing(contractAddress) {
			continue
		}

		// Only process ERC20 transfers/deposits/withdrawals
		if len(log.Topics) != TopicsInErc20DepositOrWithdrawal &&
			len(log.Topics) != TopicsInErc20Transfer {
			continue
		}

		var currency *client.ContractCurrency

		if filterTokens {
			// Check whitelist first if filtering is enabled
			tokenInfo := client.GetValidERC20Token(tokenWhiteList, contractAddress)
			if tokenInfo == nil {
				continue // Token not in whitelist
			}

			if useTokenWhiteListMetadata {
				// Use metadata from whitelist
				if tokenInfo.Decimals > math.MaxInt32 {
					return nil, fmt.Errorf("token %s has too many decimals: %d", tokenInfo.Symbol, tokenInfo.Decimals)
				}
				currency = &client.ContractCurrency{
					Symbol:   tokenInfo.Symbol,
					Decimals: int32(tokenInfo.Decimals),
				}
			} else {
				var err error
				currency, err = s.getCurrencyFromNodeOrCache(log.Address, contractAddress)
				if err != nil {
					return nil, err
				}
			}
		} else {
			var err error
			currency, err = s.getCurrencyFromNodeOrCache(log.Address, contractAddress)
			if err != nil {
				return nil, err
			}

			// Skip unknown tokens if not indexing them
			if currency.Symbol == client.UnknownERC20Symbol && !indexUnknownTokens {
				continue
			}
		}

		erc20Ops := Erc20Ops(log, currency, int64(len(ops)))
		ops = append(ops, erc20Ops...)
	}

	// Marshal receipt and trace data
	receiptMap, err := client.MarshalJSONMap(tx.Receipt)
	if err != nil {
		return nil, err
	}

	var traceList []map[string]interface{}
	for _, trace := range tx.Trace {
		traceBytes, _ := json.Marshal(trace)
		var traceMap map[string]interface{}
		if err := json.Unmarshal(traceBytes, &traceMap); err != nil {
			return nil, err
		}
		traceList = append(traceList, traceMap)
	}

	var gasLimit uint64
	if tx.Receipt != nil && tx.Receipt.GasUsed != nil {
		gasLimit = tx.Receipt.GasUsed.Uint64()
	} else {
		gasLimit = tx.Transaction.Gas()
	}

	var gasPrice *big.Int
	if tx.Receipt != nil && tx.Receipt.GasPrice != nil {
		gasPrice = tx.Receipt.GasPrice
	} else {
		gasPrice = tx.Transaction.GasPrice()
	}

	populatedTransaction := &RosettaTypes.Transaction{
		TransactionIdentifier: &RosettaTypes.TransactionIdentifier{
			Hash: tx.TxHash.String(),
		},
		Operations: ops,
		Metadata: map[string]interface{}{
			"gas_limit": hexutil.EncodeUint64(gasLimit),
			"gas_price": hexutil.EncodeBig(gasPrice),
			"receipt":   receiptMap,
			"trace":     traceList,
		},
	}

	return populatedTransaction, nil
}

// GetEthBlock returns a populated block at the *RosettaTypes.PartialBlockIdentifier.
// If neither the hash or index is populated in the *RosettaTypes.PartialBlockIdentifier,
// the current block is returned.
func (s *BlockAPIService) GetEthBlock(
	ctx context.Context,
	blockIdentifier *RosettaTypes.PartialBlockIdentifier,
) (*EthTypes.Block, []*client.LoadedTransaction, *client.RPCBlock, error) {
	if blockIdentifier != nil {
		if blockIdentifier.Hash != nil {
			return s.GetBlock(ctx, "eth_getBlockByHash", *blockIdentifier.Hash, true)
		}

		if blockIdentifier.Index != nil {
			return s.GetBlock(ctx, "eth_getBlockByNumber", client.ToBlockNumArg(big.NewInt(*blockIdentifier.Index)), true)
		}
	}

	return s.GetBlock(ctx, "eth_getBlockByNumber", client.ToBlockNumArg(nil), true)
}

func (s *BlockAPIService) GetBlock(
	ctx context.Context,
	blockMethod string,
	args ...interface{},
) (
	*EthTypes.Block,
	[]*client.LoadedTransaction,
	*client.RPCBlock,
	error,
) {
	var raw json.RawMessage
	err := s.client.CallContext(ctx, &raw, blockMethod, args...)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("block fetch failed: %w", err)
	} else if len(raw) == 0 {
		return nil, nil, nil, goEthereum.NotFound
	}

	// Decode header and transactions
	var head EthTypes.Header
	var body client.RPCBlock
	if err := json.Unmarshal(raw, &head); err != nil {
		return nil, nil, nil, err
	}
	if s.config.RosettaCfg.SupportCustomizedBlockBody {
		err = s.client.GetCustomizedBlockBody(raw, &body)
		if err != nil {
			return nil, nil, nil, err
		}
	} else {
		if err := json.Unmarshal(raw, &body); err != nil {
			return nil, nil, nil, err
		}
	}

	// Note: We need a full node to return a complete RPCBlock,
	// otherwise, only body.Hash is populated. body.Transactions is empty.
	// TODO(xiaying): log warn if len(body.Hash) > 1 && len(body.txs) == 0

	var blockAuthor string
	if s.client.GetRosettaConfig().SupportsBlockAuthor {
		blockAuthor, err = s.client.BlockAuthor(ctx, head.Number.Int64())
		if err != nil {
			return nil, nil, nil, fmt.Errorf("could not get block author for %x: %w", body.Hash[:], err)
		}
	}

	var m map[string][]*client.FlatCall
	var addTraces bool
	if head.Number.Int64() != AssetTypes.GenesisBlockIndex {
		addTraces = true
		// Use open ethereum trace API if selected.
		if s.client.GetRosettaConfig().TraceType == configuration.OpenEthereumTrace {
			m, err = s.client.TraceReplayBlockTransactions(ctx, body.Hash.String())
		} else {
			m, err = s.client.TraceBlockByHash(ctx, body.Hash, body.Transactions)
		}

		if err != nil {
			return nil, nil, nil, err
		}
	}

	// Convert all txs to loaded txs
	txs := make([]*EthTypes.Transaction, len(body.Transactions))
	loadedTxs := make([]*client.LoadedTransaction, len(body.Transactions))
	for i, tx := range body.Transactions {
		txs[i] = tx.Tx
		loadedTxs[i] = tx.LoadedTransaction()
		loadedTxs[i].Transaction = txs[i]
		loadedTxs[i].BaseFee = head.BaseFee

		if s.client.GetRosettaConfig().SupportsBlockAuthor {
			loadedTxs[i].Author = client.MustChecksum(blockAuthor)
		} else {
			loadedTxs[i].Miner = client.MustChecksum(head.Coinbase.Hex())
		}

		// Continue if calls does not exist (occurs at genesis)
		if !addTraces {
			continue
		}
		// Find traces based on Tx Hash
		hash := loadedTxs[i].TxHash.Hex()
		if flattenedCalls, ok := m[hash]; ok {
			loadedTxs[i].Trace = flattenedCalls
		}
	}

	uncles := []*EthTypes.Header{}
	if s.client.GetRosettaConfig().SupportRewardTx {
		uncles, err = s.client.GetUncles(ctx, &head, &body)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("unable to get uncles: %w", err)
		}
	}

	block := EthTypes.NewBlockWithHeader(&head).WithBody(EthTypes.Body{
		Transactions: txs,
		Uncles:       uncles,
		Withdrawals:  body.Withdrawals,
	})

	return block, loadedTxs, &body, nil
}

// convertRosettaReceiptsToEthReceipts converts already-fetched Rosetta receipts to EthTypes.Receipts for validation
func convertRosettaReceiptsToEthReceipts(rosettaReceipts []*client.RosettaTxReceipt, loadedTxns []*client.LoadedTransaction) ([]*EthTypes.Receipt, error) {
	ethReceipts := make([]*EthTypes.Receipt, len(rosettaReceipts))

	for i, rosettaReceipt := range rosettaReceipts {
		if rosettaReceipt == nil {
			return nil, fmt.Errorf("rosetta receipt %d is nil", i)
		}

		// if we have the raw message, we can unmarshal from it
		if rosettaReceipt.RawMessage != nil {
			receipt := &EthTypes.Receipt{}
			if err := receipt.UnmarshalJSON(rosettaReceipt.RawMessage); err == nil {
				ethReceipts[i] = receipt
				continue
			}
		}

		// if for some reason we can't unmarshal from RawMessage, we need to reconstruct the receipt from the available fields
		// Calculate cumulative gas used (sum of all previous + current)
		var cumulativeGasUsed uint64
		for j := 0; j <= i; j++ {
			if rosettaReceipts[j] != nil && rosettaReceipts[j].GasUsed != nil {
				cumulativeGasUsed += rosettaReceipts[j].GasUsed.Uint64()
			}
		}

		ethReceipt := &EthTypes.Receipt{
			Type:              rosettaReceipt.Type,
			Status:            rosettaReceipt.Status,
			CumulativeGasUsed: cumulativeGasUsed, // Critical for Merkle validation
			Bloom:             EthTypes.Bloom{},  // Will be reconstructed from logs
			Logs:              rosettaReceipt.Logs,
		}

		// Reconstruct bloom filter from logs (required for Merkle validation)
		if len(rosettaReceipt.Logs) > 0 {
			ethReceipt.Bloom = EthTypes.CreateBloom(ethReceipt)
		}

		ethReceipts[i] = ethReceipt
	}

	return ethReceipts, nil
}

// Block implements the /block endpoint.
func (s *BlockAPIService) Block(
	ctx context.Context,
	request *RosettaTypes.BlockRequest,
) (*RosettaTypes.BlockResponse, *RosettaTypes.Error) {
	if s.config.IsOfflineMode() {
		return nil, AssetTypes.ErrUnavailableOffline
	}

	var (
		blockIdentifier       *RosettaTypes.BlockIdentifier
		parentBlockIdentifier *RosettaTypes.BlockIdentifier
	)

	block, loadedTxns, rpcBlock, err := s.GetEthBlock(ctx, request.BlockIdentifier)
	if errors.Is(err, AssetTypes.ErrClientBlockOrphaned) {
		return nil, AssetTypes.WrapErr(AssetTypes.ErrBlockOrphaned, err)
	}

	if err != nil {
		return nil, AssetTypes.WrapErr(AssetTypes.ErrGeth, err)
	}

	var baseFee *big.Int
	// in internal is len(loadedTxns) > 1
	if len(loadedTxns) > 0 {
		baseFee = loadedTxns[0].BaseFee
	}
	receipts, err := s.client.GetBlockReceipts(ctx, rpcBlock.Hash, rpcBlock.Transactions, baseFee)
	if err != nil {
		return nil, AssetTypes.WrapErr(AssetTypes.ErrInternalError, fmt.Errorf("could not get receipts for %x: %w", rpcBlock.Hash[:], err))
	}

	for i, tx := range loadedTxns {
		if receipts != nil {
			tx.Receipt = receipts[i]
			if tx.Receipt.TransactionFee != nil {
				tx.FeeAmount = tx.Receipt.TransactionFee
			} else {
				tx.FeeAmount = big.NewInt(0)
			}
		}

		if tx.BaseFee != nil { // EIP-1559
			tx.FeeBurned = new(big.Int).Mul(tx.Receipt.GasUsed, tx.BaseFee)
		} else {
			tx.FeeBurned = baseFee
		}
	}

	blockIdentifier = &RosettaTypes.BlockIdentifier{
		Index: block.Number().Int64(),
		Hash:  block.Hash().String(),
	}
	blockIdentifier.Hash, err = s.client.GetBlockHash(ctx, *blockIdentifier)
	if err != nil {
		return nil, AssetTypes.WrapErr(AssetTypes.ErrInternalError, fmt.Errorf("could not get block hash given block identifier %v: %w", blockIdentifier, err))
	}

	parentBlockIdentifier = blockIdentifier
	if blockIdentifier.Index != AssetTypes.GenesisBlockIndex {
		parentBlockIdentifier = &RosettaTypes.BlockIdentifier{
			Hash:  block.ParentHash().Hex(),
			Index: blockIdentifier.Index - 1,
		}
	}

	crossTxns, err := s.client.PopulateCrossChainTransactions(block, loadedTxns)
	if err != nil {
		return nil, AssetTypes.WrapErr(AssetTypes.ErrGeth, err)
	}

	transactions, err := s.populateTransactions(
		ctx,
		blockIdentifier,
		block,
		loadedTxns,
	)

	if err != nil {
		return nil, AssetTypes.WrapErr(AssetTypes.ErrGeth, err)
	}

	blockResponse := &RosettaTypes.BlockResponse{
		Block: &RosettaTypes.Block{
			BlockIdentifier:       blockIdentifier,
			ParentBlockIdentifier: parentBlockIdentifier,
			Timestamp:             int64(block.Time() * utils.MillisecondsInSecond),
			Transactions:          append(transactions, crossTxns...),
			Metadata:              nil,
		},
	}

	if !s.config.IsTrustlessBlockValidationEnabled() {
		return blockResponse, nil
	}

	// run validation
	ethReceipts, err := convertRosettaReceiptsToEthReceipts(receipts, loadedTxns)
	if err != nil {
		if err != nil {
			return nil, AssetTypes.WrapErr(AssetTypes.ErrInternalError, fmt.Errorf("error fetching eth receipts: %w", err))
		}
	}

	v := validator.NewEthereumValidator(s.config)
	err = v.ValidateBlock(block, ethReceipts, rpcBlock.Hash)
	if err != nil {
		return nil, AssetTypes.WrapErr(AssetTypes.ErrInternalError, fmt.Errorf("block validation failed: %w", err))
	}

	return blockResponse, nil
}

// BlockTransaction implements the /block/transaction endpoint.
func (s *BlockAPIService) BlockTransaction(
	ctx context.Context,
	request *RosettaTypes.BlockTransactionRequest,
) (*RosettaTypes.BlockTransactionResponse, *RosettaTypes.Error) {
	if s.config.IsOfflineMode() {
		return nil, AssetTypes.ErrUnavailableOffline
	}

	if request.BlockIdentifier == nil {
		return nil, AssetTypes.WrapErr(AssetTypes.ErrInvalidInput, fmt.Errorf("block identifier is not provided"))
	}

	loadedTx, err := s.client.GetLoadedTransaction(ctx, request)
	if err != nil {
		return nil, AssetTypes.WrapErr(AssetTypes.ErrInternalError, fmt.Errorf("unable to get loaded tx: %w", err))
	}
	var (
		raw       json.RawMessage
		flattened []*client.FlatCall
		traceErr  error
	)

	if s.client.GetRosettaConfig().TraceType == configuration.OpenEthereumTrace {
		raw, flattened, traceErr = s.client.TraceReplayTransaction(ctx, loadedTx.TxHash.String())
	} else {
		raw, flattened, traceErr = s.client.TraceTransaction(ctx, *loadedTx.TxHash)
	}
	if traceErr != nil {
		return nil, AssetTypes.WrapErr(AssetTypes.ErrInternalError, fmt.Errorf("unable to get tx trace: %w", traceErr))
	}
	loadedTx.RawTrace = raw
	loadedTx.Trace = flattened

	receipt, err := s.client.GetTransactionReceipt(ctx, loadedTx)
	if err != nil {
		return nil, AssetTypes.WrapErr(AssetTypes.ErrInternalError, fmt.Errorf("unable to get tx receipt: %w", err))
	}
	loadedTx.Receipt = receipt

	loadedTx.FeeAmount = receipt.TransactionFee

	if loadedTx.BaseFee != nil { // EIP-1559
		loadedTx.FeeBurned = new(big.Int).Mul(receipt.GasUsed, loadedTx.BaseFee)
	} else {
		loadedTx.FeeBurned = nil
	}

	transaction, err := s.PopulateTransaction(ctx, loadedTx)
	if err != nil {
		return nil, AssetTypes.WrapErr(AssetTypes.ErrInternalError, fmt.Errorf("unable to populate tx: %w", err))
	}

	return &RosettaTypes.BlockTransactionResponse{
		Transaction: transaction,
	}, nil
}
