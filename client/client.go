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
	"strconv"

	"github.com/coinbase/rosetta-geth-sdk/configuration"
	sdkTypes "github.com/coinbase/rosetta-geth-sdk/types"

	"github.com/coinbase/rosetta-sdk-go/utils"

	RosettaTypes "github.com/coinbase/rosetta-sdk-go/types"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	EthTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	"golang.org/x/sync/semaphore"
)

type SDKClient struct {
	P  *params.ChainConfig
	tc *tracers.TraceConfig

	rosettaConfig configuration.RosettaConfig

	*RPCClient
	*EthClient
	// *ContractClient

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
		return nil, fmt.Errorf("%w: unable to load trace config", err)
	}

	return &SDKClient{
		P:              cfg.ChainConfig,
		tc:             tc,
		rosettaConfig:  cfg.RosettaCfg,
		RPCClient:      c,
		EthClient:      ec,
		traceSemaphore: semaphore.NewWeighted(maxTraceConcurrency),
		//ContractClient: NewContractClient(*ec.Client),
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

func toBlockNumArg(number *big.Int) string {
	if number == nil {
		return "latest"
	}
	pending := big.NewInt(-1)
	if number.Cmp(pending) == 0 {
		return "pending"
	}
	return hexutil.EncodeBig(number)
}

// decodeHexData accepts a fully formed hex string (including the 0x prefix) and returns a big.Int
func decodeHexData(data string) (*big.Int, error) {
	const base = 16
	decoded, ok := new(big.Int).SetString(data[2:], base)
	if !ok {
		return nil, fmt.Errorf("could not extract data from %s", data)
	}
	return decoded, nil
}

func (ec *SDKClient) getEthBlock(
	ctx context.Context,
	block *RosettaTypes.PartialBlockIdentifier,
) (json.RawMessage, error) {
	var raw json.RawMessage
	if block != nil {
		if block.Hash != nil {
			if err := ec.CallContext(ctx, &raw, "eth_getBlockByHash", block.Hash, false); err != nil {
				return nil, err
			}
		}
		if block.Hash == nil && block.Index != nil {
			if err := ec.CallContext(
				ctx,
				&raw,
				"eth_getBlockByNumber",
				hexutil.EncodeUint64(uint64(*block.Index)),
				false,
			); err != nil {
				return nil, err
			}
		}
	} else {
		err := ec.CallContext(ctx, &raw, "eth_getBlockByNumber", toBlockNumArg(nil), false)
		if err != nil {
			return nil, err
		}
	}

	if len(raw) == 0 {
		return nil, ethereum.NotFound
	}

	return raw, nil
}

func (ec *SDKClient) Balance(
	ctx context.Context,
	account *RosettaTypes.AccountIdentifier,
	block *RosettaTypes.PartialBlockIdentifier,
	currencies []*RosettaTypes.Currency,
) (*RosettaTypes.AccountBalanceResponse, error) {
	raw, err := ec.getEthBlock(ctx, block)
	if err != nil {
		return nil, err
	}

	var head *EthTypes.Header
	if err := json.Unmarshal(raw, &head); err != nil {
		return nil, err
	}

	var (
		nativeBalance hexutil.Big
		nonce         hexutil.Uint64
		code          string
	)

	blockNum := hexutil.EncodeUint64(head.Number.Uint64())
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
		{Method: "eth_getCode", Args: []interface{}{account.Address, blockNum}, Result: &code},
	}
	if err := ec.BatchCallContext(ctx, reqs); err != nil {
		return nil, err
	}
	for i := range reqs {
		if reqs[i].Error != nil {
			return nil, reqs[i].Error
		}
	}

	balances := []*RosettaTypes.Amount{}
	if len(currencies) == 0 {
		balances = append(balances, Amount(nativeBalance.ToInt(), ec.rosettaConfig.Currency))
	}

	for _, currency := range currencies {
		value, ok := currency.Metadata[ContractAddressMetadata]
		if !ok {
			if utils.Equal(currency, ec.rosettaConfig.Currency) {
				balances = append(
					balances,
					Amount(nativeBalance.ToInt(), ec.rosettaConfig.Currency),
				)
				continue
			}
			return nil, fmt.Errorf("non-native currencies must specify contractAddress in metadata")
		}

		identifierAddress := account.Address
		if has0xPrefix(identifierAddress) {
			identifierAddress = identifierAddress[2:42]
		}

		data, err := hexutil.Decode(BalanceOfMethodPrefix + identifierAddress)
		if err != nil {
			return nil, fmt.Errorf("%w: marshalling balanceOf call msg data failed", err)
		}
		encodedERC20Data := hexutil.Encode(data)

		contractAddress := common.HexToAddress(value.(string))

		callParams := map[string]string{
			"to":   value.(string),
			"data": encodedERC20Data,
		}
		var resp string
		if err := ec.CallContext(ctx, &resp, "eth_call", callParams, blockNum); err != nil {
			return nil, err
		}
		balance, err := decodeHexData(resp)
		if err != nil {
			return nil, err
		}

		amount := Erc20Amount(
			balance.Bytes(),
			contractAddress,
			currency.Symbol,
			currency.Decimals,
			false,
		)
		balances = append(balances, amount)
	}

	return &RosettaTypes.AccountBalanceResponse{
		Balances: balances,
		BlockIdentifier: &RosettaTypes.BlockIdentifier{
			Hash:  head.Hash().Hex(),
			Index: head.Number.Int64(),
		},
		Metadata: map[string]interface{}{
			"nonce": int64(nonce),
			"code":  code,
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

	var syncStatus *RosettaTypes.SyncStatus
	if ec.rosettaConfig.SupportsSyncing {
		progress, err := ec.syncProgress(ctx)
		if err != nil {
			return nil, -1, nil, nil, err
		}
		if progress != nil {
			currentIndex := int64(progress.CurrentBlock)
			targetIndex := int64(progress.HighestBlock)

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

// Header returns a block header from the current canonical chain. If number is
// nil, the latest known header is returned.
func (ec *SDKClient) blockHeader(
	ctx context.Context,
	input *RosettaTypes.PartialBlockIdentifier,
) (*EthTypes.Header, error) {
	var (
		header *EthTypes.Header
		err    error
	)

	if input == nil {
		header, err = ec.HeaderByNumber(ctx, nil)
	} else {
		if input.Hash == nil && input.Index == nil {
			return nil, ethereum.NotFound
		}

		if input.Index != nil {
			header, err = ec.HeaderByNumber(ctx, big.NewInt(*input.Index))
		} else {
			header, err = ec.HeaderByHash(ctx, common.HexToHash(*input.Hash))
		}
	}

	if err != nil {
		return nil, ethereum.NotFound
	}

	return header, nil
}

func (ec *SDKClient) BlockAuthor(ctx context.Context, blockIndex int64) (string, error) {
	return "", errors.New("BlockAuthor not implemented")
}

// syncProgress retrieves the current progress of the sync algorithm. If there's
// no sync currently running, it returns nil.
func (ec *SDKClient) syncProgress(ctx context.Context) (*ethereum.SyncProgress, error) {
	var raw json.RawMessage
	if err := ec.CallContext(ctx, &raw, "eth_syncing"); err != nil {
		return nil, err
	}

	var syncing bool
	if err := json.Unmarshal(raw, &syncing); err == nil {
		return nil, nil // Not syncing (always false)
	}

	var progress rpcProgress
	if err := json.Unmarshal(raw, &progress); err != nil {
		return nil, err
	}

	return &ethereum.SyncProgress{
		StartingBlock: uint64(progress.StartingBlock),
		CurrentBlock:  uint64(progress.CurrentBlock),
		HighestBlock:  uint64(progress.HighestBlock),
		PulledStates:  uint64(progress.PulledStates),
		KnownStates:   uint64(progress.KnownStates),
	}, nil
}

type rpcProgress struct {
	StartingBlock hexutil.Uint64
	CurrentBlock  hexutil.Uint64
	HighestBlock  hexutil.Uint64
	PulledStates  hexutil.Uint64
	KnownStates   hexutil.Uint64
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
	}
	return ethTx
}

func (ec *SDKClient) GetBlockReceipts(
	ctx context.Context,
	blockHash common.Hash,
	txs []RPCTransaction,
	baseFee *big.Int,
) ([]*RosettaTxReceipt, error) {
	return nil, errors.New("GetBlockReceipts not implemented")
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

	// Decode []*rpcCall
	if err := json.Unmarshal(raw, &calls); err != nil {
		return nil, err
	}

	m := make(map[string][]*FlatCall)
	for i, tx := range calls {
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

// TraceReplayBlockTransactions returns all transactions in a block returning the requested traces
// for each Transaction.
func (ec *SDKClient) TraceReplayBlockTransactions(ctx context.Context, hsh string) (
	map[string][]*FlatCall, error,
) {
	var raw json.RawMessage
	err := ec.CallContext(ctx, &raw, "trace_replayBlockTransactions", hsh, []string{"trace"})
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
	err := ec.CallContext(ctx, &raw, "trace_replayTransaction", hsh, []string{"trace"})
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
) int64 {
	if currentBlock.Int64() == int64(0) {
		return big.NewInt(0).Int64()
	}

	blockReward := ethash.FrontierBlockReward.Int64()
	if ec.P.IsByzantium(currentBlock) {
		blockReward = ethash.ByzantiumBlockReward.Int64()
	}

	if ec.P.IsConstantinople(currentBlock) {
		blockReward = ethash.ConstantinopleBlockReward.Int64()
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
		rewardFloat := reward.Mul(big.NewFloat(uncleReward), big.NewFloat(float64(miningReward)))
		rewardInt, _ := rewardFloat.Int64()
		minerReward += rewardInt
	}

	const base = 10
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
			Value:    strconv.FormatInt(minerReward, base),
			Currency: ec.rosettaConfig.Currency,
		},
	}
	ops = append(ops, miningRewardOp)

	// Calculate uncle rewards
	for _, b := range uncles {
		uncleMiner := b.Coinbase.String()
		uncleBlock := b.Number.Int64()
		uncleRewardBlock := new(
			big.Int,
		).Mul(
			big.NewInt(uncleBlock+sdkTypes.MaxUncleDepth-blockIdentifier.Index),
			big.NewInt(miningReward/sdkTypes.MaxUncleDepth),
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
		log.Println("Fetching gas price")
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
		log.Println("Setting existing gas price")
		gasPrice = input.GasPrice
	}
	return gasPrice, nil
}

func (ec *SDKClient) GetNativeTransferGasLimit(ctx context.Context, toAddress string,
	fromAddress string, value *big.Int) (uint64, error) {
	return 0, errors.New("GetNativeTransferGasLimit not implemented")
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

	// ToAddress for erc20 transfers is the contract address
	contractAddress := common.HexToAddress(contract.(string))
	data := GenerateErc20TransferData(toAddress, value)
	gasLimit, err := ec.EstimateGas(ctx, ethereum.CallMsg{
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
	gasLimit, err := ec.EstimateGas(ctx, ethereum.CallMsg{
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

func (ec *SDKClient) GetTransactionReceipt(
	ctx context.Context,
	tx *LoadedTransaction,
) (*RosettaTxReceipt, error) {
	return nil, errors.New("GetTransactionReceipt not implemented")
}

func (ec *SDKClient) GetLoadedTransaction(
	ctx context.Context,
	request *RosettaTypes.BlockTransactionRequest,
) (*LoadedTransaction, error) {
	header, err := ec.HeaderByHash(ctx, common.HexToHash(request.BlockIdentifier.Hash))
	if err != nil {
		return nil, fmt.Errorf("%w: failure getting header", err)
	}

	hash := common.HexToHash(request.TransactionIdentifier.Hash)
	tx, pending, err := ec.TransactionByHash(ctx, hash)
	if err != nil {
		return nil, fmt.Errorf("%w: failure getting tx", err)
	}
	if pending {
		return nil, nil
	}

	signer := EthTypes.LatestSignerForChainID(ec.P.ChainID)
	msg, err := tx.AsMessage(signer, header.BaseFee)
	if err != nil {
		return nil, err
	}
	blockNumber := header.Number.String()
	from := msg.From()
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
			return nil, fmt.Errorf("%w: could not get block author for %x", err, hash)
		}
		loadedTx.Author = MustChecksum(blockAuthor)
	} else {
		loadedTx.Miner = MustChecksum(header.Coinbase.Hex())
	}

	return loadedTx, nil
}
