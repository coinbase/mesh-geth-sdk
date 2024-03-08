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

package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/big"

	"github.com/coinbase/rosetta-geth-sdk/configuration"
	sdkTypes "github.com/coinbase/rosetta-geth-sdk/types"

	"github.com/coinbase/rosetta-sdk-go/utils"

	RosettaTypes "github.com/coinbase/rosetta-sdk-go/types"

	goEthereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	EthTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/holiman/uint256"
	"golang.org/x/sync/semaphore"
)

type SDKClient struct {
	P  *params.ChainConfig
	tc *tracers.TraceConfig

	rosettaConfig configuration.RosettaConfig

	*RPCClient
	*EthClient

	traceSemaphore *semaphore.Weighted

	skipAdminCalls bool
}

// NewClient creates a client that connects to the network.
func NewClient(cfg *configuration.Configuration, rpcClient *RPCClient) (*SDKClient, error) {
	c, err := NewRPCClient(cfg.GethURL)
	if err != nil {
		return nil, err
	}

	if rpcClient != nil {
		c = rpcClient
	}

	ec, err := NewEthClient(cfg.GethURL)
	if err != nil {
		return nil, err
	}

	enableNativeTracer := cfg.RosettaCfg.TraceType == configuration.GethNativeTrace
	tc, err := GetTraceConfig(enableNativeTracer)
	if err != nil {
		return nil, fmt.Errorf("unable to load trace config: %w", err)
	}

	return &SDKClient{
		P:              cfg.ChainConfig,
		tc:             tc,
		rosettaConfig:  cfg.RosettaCfg,
		RPCClient:      c,
		EthClient:      ec,
		traceSemaphore: semaphore.NewWeighted(maxTraceConcurrency),
	}, nil
}

func (ec *SDKClient) PopulateCrossChainTransactions(
	*EthTypes.Block,
	[]*LoadedTransaction,
) ([]*RosettaTypes.Transaction, error) {
	return nil, nil
}

func (ec *SDKClient) GetRosettaConfig() configuration.RosettaConfig {
	return ec.rosettaConfig
}

func (ec *SDKClient) GetClient() *SDKClient {
	return ec
}

// decodeHexData accepts a fully formed hex string (including the 0x prefix) and returns a big.Int
func decodeHexData(data string) (*big.Int, error) {
	rawData := data[2:]
	if rawData == "" {
		return big.NewInt(0), nil
	}
	const base = 16
	decoded, ok := new(big.Int).SetString(rawData, base)
	if !ok {
		return nil, fmt.Errorf("could not extract data from %s", data)
	}
	return decoded, nil
}

func (ec *SDKClient) Balance(
	ctx context.Context,
	account *RosettaTypes.AccountIdentifier,
	blockIdentifier *RosettaTypes.PartialBlockIdentifier,
	currencies []*RosettaTypes.Currency,
) (*RosettaTypes.AccountBalanceResponse, error) {
	header, err := ec.blockHeader(ctx, blockIdentifier)
	if err != nil {
		return nil, fmt.Errorf("failed to get block header: %w", err)
	}

	var (
		nativeBalance hexutil.Big
		nonce         hexutil.Uint64
	)

	// Get ETH balance, nonce, and bytecode of smart contract
	blockNum := hexutil.EncodeUint64(header.Number.Uint64())
	reqs := []rpc.BatchElem{
		{
			Method: "eth_getBalance",
			Args:   []interface{}{account.Address, blockNum},
			Result: &nativeBalance,
		},
		{
			Method: "eth_getTransactionCount",
			Args:   []interface{}{account.Address, blockNum},
			Result: &nonce,
		},
	}
	if err := ec.BatchCallContext(ctx, reqs); err != nil {
		return nil, err
	}
	for i := range reqs {
		if reqs[i].Error != nil {
			return nil, reqs[i].Error
		}
	}

	// No currencies are specified, return ETH balance
	balances := []*RosettaTypes.Amount{}
	if len(currencies) == 0 {
		balances = append(balances, Amount(nativeBalance.ToInt(), ec.rosettaConfig.Currency))
	}

	for _, currency := range currencies {
		address, ok := currency.Metadata[ContractAddressMetadata]
		if !ok {
			if utils.Equal(currency, ec.rosettaConfig.Currency) {
				// ETH is specified in the currencies
				balances = append(balances, Amount(nativeBalance.ToInt(), ec.rosettaConfig.Currency))
				continue
			}
			return nil, fmt.Errorf("non-native currencies must specify contractAddress in metadata")
		}

		// ERC20 is specified in the currencies
		identifierAddress := account.Address
		if has0xPrefix(identifierAddress) {
			identifierAddress = identifierAddress[2:42]
		}

		contractAddress := address.(string)
		data := BalanceOfMethodPrefix + identifierAddress
		callParams := map[string]string{
			"to":   contractAddress,
			"data": data,
		}
		var resp string
		if err := ec.CallContext(ctx, &resp, "eth_call", callParams, blockNum); err != nil {
			return nil, err
		}
		balance, err := decodeHexData(resp)
		if err != nil {
			return nil, fmt.Errorf("failed to decode balanceOf call response: %w", err)
		}
		balances = append(balances, Amount(balance, Erc20Currency(currency.Symbol, currency.Decimals, contractAddress)))
	}

	return &RosettaTypes.AccountBalanceResponse{
		Balances: balances,
		BlockIdentifier: &RosettaTypes.BlockIdentifier{
			Hash:  header.Hash().Hex(),
			Index: header.Number.Int64(),
		},
		Metadata: map[string]interface{}{
			"nonce": int64(nonce),
		},
	}, nil
}

// Status returns geth status information
// for determining node healthiness.
func (ec *SDKClient) Status(ctx context.Context) (
	*RosettaTypes.BlockIdentifier,
	int64,
	*RosettaTypes.SyncStatus,
	[]*RosettaTypes.Peer,
	error,
) {
	header, err := ec.blockHeader(ctx, nil)
	if err != nil {
		return nil, -1, nil, nil, err
	}

	// Get sync status
	var syncStatus *RosettaTypes.SyncStatus
	if ec.rosettaConfig.SupportsSyncing {
		syncProgress, err := ec.SyncProgress(ctx)
		if err != nil {
			return nil, -1, nil, nil, err
		}
		if syncProgress != nil {
			currentIndex := int64(syncProgress.CurrentBlock)
			targetIndex := int64(syncProgress.HighestBlock)

			syncStatus = &RosettaTypes.SyncStatus{
				CurrentIndex: &currentIndex,
				TargetIndex:  &targetIndex,
			}
		}
	} else {
		syncStatus = &RosettaTypes.SyncStatus{
			Synced: RosettaTypes.Bool(true),
			Stage:  RosettaTypes.String("SYNCED"),
		}
	}

	// Get peers information
	var peers []*RosettaTypes.Peer
	if ec.rosettaConfig.SupportsPeering {
		peers, err = ec.peers(ctx)
		if err != nil {
			return nil, -1, nil, nil, err
		}
	} else {
		peers = []*RosettaTypes.Peer{}
	}

	return &RosettaTypes.BlockIdentifier{
			Hash:  header.Hash().Hex(),
			Index: header.Number.Int64(),
		},
		convertTime(header.Time),
		syncStatus,
		peers,
		nil
}

// blockHeader returns a block header from the current canonical chain.
// If number is nil, the latest known header is returned.
func (ec *SDKClient) blockHeader(
	ctx context.Context,
	blockIdentifier *RosettaTypes.PartialBlockIdentifier,
) (*EthTypes.Header, error) {
	var (
		header *EthTypes.Header
		err    error
	)

	if blockIdentifier == nil || (blockIdentifier.Hash == nil && blockIdentifier.Index == nil) {
		defaultBlockNumber := ec.rosettaConfig.DefaultBlockNumber
		if len(defaultBlockNumber) != 0 {
			// Handle reorg issues of Optimism and Base
			err = ec.CallContext(ctx, &header, "eth_getBlockByNumber", defaultBlockNumber, false)
		} else {
			err = ec.CallContext(ctx, &header, "eth_getBlockByNumber", ToBlockNumArg(nil), false)
		}
	} else {
		if blockIdentifier.Index != nil {
			err = ec.CallContext(ctx, &header, "eth_getBlockByNumber", ToBlockNumArg(big.NewInt(*blockIdentifier.Index)), false)
		} else {
			err = ec.CallContext(ctx, &header, "eth_getBlockByHash", common.HexToHash(*blockIdentifier.Hash), false)
		}
	}

	if err == nil && header == nil {
		return nil, goEthereum.NotFound
	}
	return header, err
}

// Peers retrieves all peers of the node.
func (ec *SDKClient) peers(ctx context.Context) ([]*RosettaTypes.Peer, error) {
	var info []*p2p.PeerInfo

	if ec.skipAdminCalls {
		return []*RosettaTypes.Peer{}, nil
	}

	if err := ec.CallContext(ctx, &info, "admin_peers"); err != nil {
		return nil, err
	}

	peers := make([]*RosettaTypes.Peer, len(info))
	for i, peerInfo := range info {
		peers[i] = &RosettaTypes.Peer{
			PeerID: peerInfo.ID,
			Metadata: map[string]interface{}{
				"name":      peerInfo.Name,
				"enode":     peerInfo.Enode,
				"caps":      peerInfo.Caps,
				"enr":       peerInfo.ENR,
				"protocols": peerInfo.Protocols,
			},
		}
	}

	return peers, nil
}

func (ec *SDKClient) GetUncles(
	ctx context.Context,
	head *EthTypes.Header,
	body *RPCBlock,
) ([]*EthTypes.Header, error) {
	// Quick-verify Transaction and uncle lists. This mostly helps with debugging the server.
	if head.UncleHash == EthTypes.EmptyUncleHash && len(body.UncleHashes) > 0 {
		return nil, fmt.Errorf(
			"server returned non-empty uncle list but block header indicates no uncles",
		)
	}
	if head.UncleHash != EthTypes.EmptyUncleHash && len(body.UncleHashes) == 0 {
		return nil, fmt.Errorf(
			"server returned empty uncle list but block header indicates uncles",
		)
	}
	if head.TxHash == EthTypes.EmptyRootHash && len(body.Transactions) > 0 {
		return nil, fmt.Errorf(
			"server returned non-empty Transaction list but block header indicates no transactions",
		)
	}
	if head.TxHash != EthTypes.EmptyRootHash && len(body.Transactions) == 0 {
		return nil, fmt.Errorf(
			"server returned empty Transaction list but block header indicates transactions",
		)
	}
	// Load uncles because they are not included in the block response.
	var uncles []*EthTypes.Header
	if len(body.UncleHashes) > 0 {
		uncles = make([]*EthTypes.Header, len(body.UncleHashes))
		reqs := make([]rpc.BatchElem, len(body.UncleHashes))
		for i := range reqs {
			reqs[i] = rpc.BatchElem{
				Method: "eth_getUncleByBlockHashAndIndex",
				Args:   []interface{}{body.Hash, hexutil.EncodeUint64(uint64(i))},
				Result: &uncles[i],
			}
		}
		if err := ec.BatchCallContext(ctx, reqs); err != nil {
			return nil, err
		}
		for i := range reqs {
			if reqs[i].Error != nil {
				return nil, reqs[i].Error
			}
			if uncles[i] == nil {
				return nil, fmt.Errorf(
					"got null header for uncle %d of block %x",
					i,
					body.Hash[:],
				)
			}
		}
	}
	return uncles, nil
}

func (tx *RPCTransaction) LoadedTransaction() *LoadedTransaction {
	ethTx := &LoadedTransaction{
		Transaction: tx.Tx,
		From:        tx.TxExtraInfo.From,
		BlockNumber: tx.TxExtraInfo.BlockNumber,
		BlockHash:   tx.TxExtraInfo.BlockHash,
		TxHash:      tx.TxExtraInfo.TxHash,
		Mint:        tx.TxExtraInfo.Mint,
	}
	return ethTx
}

// TraceBlockByHash returns the Transaction traces of all transactions in the block
func (ec *SDKClient) TraceBlockByHash(
	ctx context.Context,
	blockHash common.Hash,
	txs []RPCTransaction,
) (map[string][]*FlatCall, error) {
	if err := ec.traceSemaphore.Acquire(ctx, semaphoreTraceWeight); err != nil {
		return nil, err
	}
	defer ec.traceSemaphore.Release(semaphoreTraceWeight)

	var calls []*rpcCall
	var raw json.RawMessage
	err := ec.CallContext(ctx, &raw, "debug_traceBlockByHash", blockHash, ec.tc)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(raw, &calls); err != nil {
		return nil, err
	}
	m := make(map[string][]*FlatCall)
	for i, tx := range calls {
		if tx.Result.Type == "" {
			// ignore calls with an empty type
			continue
		}
		flatCalls := FlattenTraces(tx.Result, []*FlatCall{})
		// Ethereum native traces are guaranteed to return all transactions
		txHash := txs[i].TxExtraInfo.TxHash.Hex()
		if txHash == "" {
			return nil, fmt.Errorf("could not get %dth tx hash for block %s", i, blockHash.Hex())
		}
		m[txHash] = flatCalls
	}
	return m, nil
}

// TraceTransaction returns a Transaction trace
func (ec *SDKClient) TraceTransaction(
	ctx context.Context,
	hash common.Hash,
) (json.RawMessage, []*FlatCall, error) {
	result := &Call{}
	var raw json.RawMessage
	err := ec.CallContext(ctx, &raw, "debug_traceTransaction", hash, ec.tc)
	if err != nil {
		return nil, nil, err
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, nil, err
	}
	flattened := FlattenTraces(result, []*FlatCall{})
	return raw, flattened, nil
}

// TraceReplayBlockTransactions returns all transactions in a block returning the requested traces for each Transaction.
func (ec *SDKClient) TraceReplayBlockTransactions(ctx context.Context, hsh string) (
	map[string][]*FlatCall, error,
) {
	var raw json.RawMessage
	err := ec.CallContext(ctx, &raw, ec.rosettaConfig.TracePrefix+"_replayBlockTransactions", hsh, []string{"trace"})
	if err != nil {
		return nil, err
	}
	var results []*OpenEthTraceCall
	if err := json.Unmarshal(raw, &results); err != nil {
		return nil, err
	}
	if len(results) == 0 {
		log.Printf("Block %s does not have traces", hsh)
	}

	m := make(map[string][]*FlatCall)
	for _, result := range results {
		if len(result.Trace) == 0 {
			continue
		}

		for _, child := range result.Trace {
			if child.TransactionHash == "" {
				continue
			}
			action := child.Action
			traceType := action.Type
			if traceType == "" {
				traceType = child.Type
			}
			flattenCall := &FlatCall{
				Type:    traceType,
				From:    action.From,
				To:      action.To,
				Value:   action.Value,
				GasUsed: action.GasUsed,
				// Revert:       t.Revert,
				// ErrorMessage: t.ErrorMessage,
			}
			m[child.TransactionHash] = append(m[child.TransactionHash], flattenCall)
		}
	}
	return m, nil
}

// TraceReplayTransaction returns a Transaction trace
func (ec *SDKClient) TraceReplayTransaction(
	ctx context.Context,
	hsh string,
) (json.RawMessage, []*FlatCall, error) {
	var raw json.RawMessage
	err := ec.CallContext(ctx, &raw, ec.rosettaConfig.TracePrefix+"_replayTransaction", hsh, []string{"trace"})
	if err != nil {
		log.Fatalln(err)
	}

	if err != nil {
		return nil, nil, err
	}

	var result *OpenEthTraceCall
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, nil, err
	}

	if len(result.Trace) == 0 {
		return nil, nil, fmt.Errorf("Transaction(%s) does not have traces", hsh)
	}
	flattened := FlattenOpenEthTraces(result, []*FlatCall{})
	return raw, flattened, nil
}

// EffectiveGasPrice returns the price of gas charged to this Transaction to be included in the
// block.
func EffectiveGasPrice(tx *EthTypes.Transaction, baseFee *big.Int) (*big.Int, error) {
	if tx.Type() != eip1559TxType {
		return tx.GasPrice(), nil
	}
	// For EIP-1559 the gas price is determined by the base fee & miner tip sinstead
	// of the tx-specified gas price.
	tip, err := tx.EffectiveGasTip(baseFee)
	if err != nil {
		return nil, err
	}
	return new(big.Int).Add(tip, baseFee), nil
}

// flattenTraces recursively flattens all traces.
func FlattenTraces(data *Call, flattened []*FlatCall) []*FlatCall {
	if data == nil {
		return flattened
	}
	results := append(flattened, data.flatten()) //nolint
	for _, child := range data.Calls {
		// Ensure all children of a reverted call
		// are also reverted!
		if data.Revert {
			child.Revert = true

			// Copy error message from parent
			// if child does not have one
			if len(child.ErrorMessage) == 0 {
				child.ErrorMessage = data.ErrorMessage
			}
		}

		children := FlattenTraces(child, flattened)
		results = append(results, children...)
	}
	return results
}

// miningReward returns the mining reward
// for a given block height.
//
// Source:
// https://github.com/ethereum/go-ethereum/blob/master/consensus/ethash/consensus.go#L646-L653
func (ec *SDKClient) miningReward(
	currentBlock *big.Int,
) *uint256.Int {
	if currentBlock.Int64() == int64(0) {
		return uint256.NewInt(0)
	}

	blockReward := ethash.FrontierBlockReward

	if ec.P.IsByzantium(currentBlock) {
		blockReward = ethash.ByzantiumBlockReward
	}
	if ec.P.IsConstantinople(currentBlock) {
		blockReward = ethash.ConstantinopleBlockReward
	}

	return blockReward
}

func (ec *SDKClient) BlockRewardTransaction(
	blockIdentifier *RosettaTypes.BlockIdentifier,
	miner string,
	uncles []*EthTypes.Header,
) *RosettaTypes.Transaction {
	var ops []*RosettaTypes.Operation
	miningReward := ec.miningReward(big.NewInt(blockIdentifier.Index))

	// https://github.com/ethereum/go-ethereum/blob/
	// aaca58a7a1d9acbd24bbc74c49933efa2f1af183/consensus/ethash/consensus.go#L645
	// Calculate miner rewards:
	// mining_reward * (1 / 32) * num_of_uncles + mining_reward = final_mining_reward
	// Calculate uncle miner rewards:
	// (uncle_block_index + 8 - current_block_index) * final_mining_reward / 8
	minerReward := miningReward
	numUncles := len(uncles)
	if len(uncles) > 0 {
		reward := new(big.Float)
		uncleReward := float64(numUncles) / sdkTypes.UnclesRewardMultiplier
		rewardFloat := reward.Mul(big.NewFloat(uncleReward), big.NewFloat(miningReward.Float64()))
		rewardInt := new(big.Int)
		rewardFloat.Int(rewardInt)
		minerReward.Add(minerReward, uint256.MustFromBig(rewardInt))
	}

	miningRewardOp := &RosettaTypes.Operation{
		OperationIdentifier: &RosettaTypes.OperationIdentifier{
			Index: 0,
		},
		Type:   sdkTypes.MinerRewardOpType,
		Status: RosettaTypes.String(sdkTypes.SuccessStatus),
		Account: &RosettaTypes.AccountIdentifier{
			Address: MustChecksum(miner),
		},
		Amount: &RosettaTypes.Amount{
			Value:    minerReward.Dec(),
			Currency: ec.rosettaConfig.Currency,
		},
	}
	ops = append(ops, miningRewardOp)

	// Calculate uncle rewards
	for _, b := range uncles {
		uncleMiner := b.Coinbase.String()
		uncleBlock := b.Number.Int64()
		miningRewardPerUncle := minerReward.Clone()
		miningRewardPerUncle.Div(miningRewardPerUncle, uint256.NewInt(sdkTypes.MaxUncleDepth))
		uncleRewardBlock := new(
			big.Int,
		).Mul(
			big.NewInt(uncleBlock+sdkTypes.MaxUncleDepth-blockIdentifier.Index),
			miningRewardPerUncle.ToBig(),
		)

		uncleRewardOp := &RosettaTypes.Operation{
			OperationIdentifier: &RosettaTypes.OperationIdentifier{
				Index: int64(len(ops)),
			},
			Type:   sdkTypes.UncleRewardOpType,
			Status: RosettaTypes.String(sdkTypes.SuccessStatus),
			Account: &RosettaTypes.AccountIdentifier{
				Address: MustChecksum(uncleMiner),
			},
			Amount: &RosettaTypes.Amount{
				Value:    uncleRewardBlock.String(),
				Currency: ec.rosettaConfig.Currency,
			},
		}
		ops = append(ops, uncleRewardOp)
	}

	return &RosettaTypes.Transaction{
		TransactionIdentifier: &RosettaTypes.TransactionIdentifier{
			Hash: blockIdentifier.Hash,
		},
		Operations: ops,
	}
}

func (ec *SDKClient) Submit(
	ctx context.Context,
	signedTx *EthTypes.Transaction,
) error {
	return ec.SendTransaction(ctx, signedTx)
}

func (ec *SDKClient) GetNonce(
	ctx context.Context,
	input Options,
) (uint64, error) {
	var nonce uint64
	var err error
	if input.Nonce == nil {
		nonce, err = ec.NonceAt(ctx, common.HexToAddress(input.From), nil)
		if err != nil {
			return 0, err
		}
	} else {
		nonce = input.Nonce.Uint64()
	}
	return nonce, nil
}

func (ec *SDKClient) GetGasPrice(
	ctx context.Context,
	input Options,
) (*big.Int, error) {
	var gasPrice *big.Int
	var err error
	if input.GasPrice == nil || input.GasPrice.Uint64() == 0 {
		gasPrice, err = ec.SuggestGasPrice(ctx)
		if err != nil {
			return nil, err
		}
		if input.SuggestedFeeMultiplier != nil {
			newGasPrice := new(big.Float).Mul(
				big.NewFloat(*input.SuggestedFeeMultiplier),
				new(big.Float).SetInt(gasPrice),
			)
			newGasPrice.Int(gasPrice)
		}
	} else {
		gasPrice = input.GasPrice
	}
	return gasPrice, nil
}

func (ec *SDKClient) GetGasTipCap(ctx context.Context, input Options) (*big.Int, error) {
	if input.GasTipCap == nil {
		var hex hexutil.Big
		if err := ec.CallContext(ctx, &hex, "eth_maxPriorityFeePerGas"); err != nil {
			return nil, err
		}

		gasTipCap := hex.ToInt()
		priorityFeeDivisor := getPriorityFeeDivisor(ec.rosettaConfig)
		adjustedPriorityFee := new(big.Int).Div(gasTipCap, priorityFeeDivisor)

		return adjustedPriorityFee, nil
	}

	return input.GasTipCap, nil
}

func (ec *SDKClient) GetGasFeeCap(ctx context.Context, input Options, gasTipCap *big.Int) (*big.Int, error) {
	if input.GasFeeCap == nil {
		baseFee, err := ec.GetBaseFee(ctx)
		if err != nil {
			return nil, err
		}

		if baseFee != nil {
			// Calculate max fee per gas (i.e. gas fee cap)
			// Formula: GasFeeCap = max(BaseFeeMultiplier * BaseFee, BaseFeeFloor) + GasTipCap
			// BaseFeeFloor: when base fee is decreasing dramatically, we can leverage BaseFeeFloor to speed up the tx onchain landing process
			// BaseFeeMultiplier: when base fee is increasing dramatically, we can leverage BaseFeeMultiplier to ensure the tx can be landed onchain with enough fee
			// BaseFeeFloor and BaseFeeMultiplier are chain specific, if the downstream service doesn't specify them in Rosetta config,
			// the default formula in Rosetta layer is EIP-1559 neutral, which is GasFeeCap = BaseFee + GasTipCap
			baseFeeFloor := getBaseFeeFloor(ec.rosettaConfig)
			baseFeeMultiplier := getBaseFeeMultiplier(ec.rosettaConfig)
			adjustedBaseFee := new(big.Int).Mul(baseFee, baseFeeMultiplier)
			gasFeeCap := new(big.Int).Set(bigIntMax(adjustedBaseFee, baseFeeFloor))
			gasFeeCap.Add(gasFeeCap, gasTipCap)

			return gasFeeCap, nil
		}
	}

	return input.GasFeeCap, nil
}

func getBaseFeeFloor(rosettaConfig configuration.RosettaConfig) *big.Int {
	baseFeeFloor := big.NewInt(configuration.DefaultBaseFeeFloor)
	if rosettaConfig.BaseFeeFloor != nil {
		baseFeeFloor = rosettaConfig.BaseFeeFloor
	}

	return baseFeeFloor
}

func getBaseFeeMultiplier(rosettaConfig configuration.RosettaConfig) *big.Int {
	baseFeeMultiplier := big.NewInt(configuration.DefaultBaseFeeMultiplier)
	if rosettaConfig.BaseFeeMultiplier != nil {
		baseFeeMultiplier = rosettaConfig.BaseFeeMultiplier
	}

	return baseFeeMultiplier
}

func getPriorityFeeDivisor(rosettaConfig configuration.RosettaConfig) *big.Int {
	priorityFeeDivisor := big.NewInt(configuration.DefaultPriorityFeeDivisor)
	if rosettaConfig.PriorityFeeDivisor != nil {
		priorityFeeDivisor = rosettaConfig.PriorityFeeDivisor
	}

	return priorityFeeDivisor
}

func bigIntMax(a *big.Int, b *big.Int) *big.Int {
	if a.Cmp(b) == -1 {
		return b
	}

	return a
}

func (ec *SDKClient) GetBaseFee(ctx context.Context) (*big.Int, error) {
	var head *Header
	if err := ec.CallContext(ctx, &head, "eth_getBlockByNumber", "latest", false); err != nil {
		return nil, err
	}
	if head == nil {
		return nil, goEthereum.NotFound
	}
	return head.BaseFee.ToInt(), nil
}

func (ec *SDKClient) GetErc20TransferGasLimit(
	ctx context.Context,
	toAddress string,
	fromAddress string,
	value *big.Int,
	currency *RosettaTypes.Currency,
) (uint64, error) {
	contract, ok := currency.Metadata[ContractAddressMetadata]
	if !ok {
		return 0, fmt.Errorf("unable to find contract info for currency %s", currency.Symbol)
	}

	// toAddress in function param is the address to receive token
	// the To address in EstimateGas is the contract address
	contractAddress := common.HexToAddress(contract.(string))
	data := GenerateErc20TransferData(toAddress, value)
	gasLimit, err := ec.EstimateGas(ctx, goEthereum.CallMsg{
		From: common.HexToAddress(fromAddress),
		To:   &contractAddress,
		Data: data,
	})
	if err != nil {
		return 0, err
	}
	return gasLimit, nil
}

func (ec *SDKClient) GetContractCallGasLimit(
	ctx context.Context,
	toAddress string,
	fromAddress string,
	data []byte,
) (uint64, error) {
	// ToAddress for contract address is the contract address
	contractAddress := common.HexToAddress(toAddress)
	gasLimit, err := ec.EstimateGas(ctx, goEthereum.CallMsg{
		From: common.HexToAddress(fromAddress),
		To:   &contractAddress,
		Data: data,
	})
	if err != nil {
		return 0, err
	}
	return gasLimit, nil
}

// GetContractCurrency returns the currency for a specific address
func (ec *SDKClient) GetContractCurrency(
	addr common.Address,
	erc20 bool,
) (*ContractCurrency, error) {
	token, err := NewContractInfoToken(addr, ec.EthClient)
	if err != nil {
		return nil, err
	}

	symbol, symbolErr := token.Symbol(nil)
	decimals, decimalErr := token.Decimals(nil)

	// Any of these indicate a failure to get complete information from contract
	if symbolErr != nil || decimalErr != nil || symbol == "" || decimals == 0 {
		if erc20 {
			symbol = UnknownERC20Symbol
			decimals = UnknownERC20Decimals
		} else {
			symbol = UnknownERC721Symbol
			decimals = UnknownERC721Decimals
		}
	}

	currency := &ContractCurrency{
		Symbol:   symbol,
		Decimals: int32(decimals),
	}

	return currency, nil
}

// nolint:staticcheck
func (ec *SDKClient) GetLoadedTransaction(
	ctx context.Context,
	request *RosettaTypes.BlockTransactionRequest,
) (*LoadedTransaction, error) {
	header, err := ec.HeaderByHash(ctx, common.HexToHash(request.BlockIdentifier.Hash))
	if err != nil {
		return nil, fmt.Errorf("failure getting header: %w", err)
	}

	hash := common.HexToHash(request.TransactionIdentifier.Hash)
	tx, pending, err := ec.TransactionByHash(ctx, hash)
	if err != nil {
		return nil, fmt.Errorf("failure getting tx: %w", err)
	}
	if pending {
		return nil, nil
	}

	signer := EthTypes.LatestSignerForChainID(ec.P.ChainID)
	msg, err := core.TransactionToMessage(tx, signer, header.BaseFee)
	if err != nil {
		return nil, err
	}
	blockNumber := header.Number.String()
	from := msg.From
	txHash := tx.Hash()

	txInfo := TxExtraInfo{
		BlockNumber: &blockNumber,
		BlockHash:   &hash,
		From:        &from,
		TxHash:      &txHash,
	}
	rpcTx := RPCTransaction{tx, txInfo}
	loadedTx := rpcTx.LoadedTransaction()

	loadedTx.BaseFee = header.BaseFee

	if ec.rosettaConfig.SupportsBlockAuthor {
		blockAuthor, err := ec.BlockAuthor(ctx, header.Number.Int64())
		if err != nil {
			return nil, fmt.Errorf("could not get block author for %x: %w", hash, err)
		}
		loadedTx.Author = MustChecksum(blockAuthor)
	} else {
		miner := header.Coinbase.Hex()
		loadedTx.Miner = MustChecksum(miner)
	}
	return loadedTx, nil
}

///////////////////////////////////////////////////////////////////////////
// Below are functions that should be implemented by chain specific Rosetta
///////////////////////////////////////////////////////////////////////////

func (ec *SDKClient) ParseOps(
	tx *LoadedTransaction,
) ([]*RosettaTypes.Operation, error) {
	return nil, errors.New("ParseOps not implemented")
}

// nolint:staticcheck
func (ec *SDKClient) BlockAuthor(ctx context.Context, blockIndex int64) (string, error) {
	return "", errors.New("BlockAuthor not implemented")
}

func (ec *SDKClient) GetTransactionReceipt(
	ctx context.Context,
	tx *LoadedTransaction,
) (*RosettaTxReceipt, error) {
	return nil, errors.New("GetTransactionReceipt not implemented")
}

func (ec *SDKClient) GetBlockReceipts(
	ctx context.Context,
	blockHash common.Hash,
	txs []RPCTransaction,
	baseFee *big.Int,
) ([]*RosettaTxReceipt, error) {
	return nil, errors.New("GetBlockReceipts not implemented")
}

func (ec *SDKClient) GetNativeTransferGasLimit(ctx context.Context, toAddress string,
	fromAddress string, value *big.Int) (uint64, error) {
	return 0, errors.New("GetNativeTransferGasLimit not implemented")
}

func (ec *SDKClient) GetL1DataFee(ctx context.Context, ethTxBytes []byte) (*big.Int, error) {
	return nil, errors.New("GetL1DataFee not implemented")
}
