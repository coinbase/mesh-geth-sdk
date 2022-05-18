package construction

import (
	"context"
	"encoding/json"
	"math/big"

	evmClient "github.com/coinbase/rosetta-geth-sdk/client"
	"github.com/coinbase/rosetta-geth-sdk/configuration"
	RosettaTypes "github.com/coinbase/rosetta-sdk-go/types"
	"github.com/ethereum/go-ethereum/common"
	EthTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
)

// Client contains all the methods required to interact with go-ethereum based blockchain
type Client interface {
	// Status returns the current status of the network
	Status(
		context.Context,
	) (*RosettaTypes.BlockIdentifier,
		int64,
		*RosettaTypes.SyncStatus,
		[]*RosettaTypes.Peer,
		error,
	)

	// Balance is to fetch the state of an account based on the provided currencies
	Balance(
		context.Context,
		*RosettaTypes.AccountIdentifier,
		*RosettaTypes.PartialBlockIdentifier,
		[]*RosettaTypes.Currency,
	) (*RosettaTypes.AccountBalanceResponse, error)

	// Submit is to submit a pre-signed transaction to the blockchain
	Submit(context.Context, *EthTypes.Transaction) error

	// GetNonce returns the account nonce of the given account
	// The given account is the from address in Options input
	GetNonce(context.Context, evmClient.Options) (uint64, error)

	// GetGasPrice retrieves the currently suggested gas price
	GetGasPrice(context.Context, evmClient.Options) (*big.Int, error)

	// GetRosettaConfig returns the Rosetta config we defined for the network
	GetRosettaConfig() configuration.RosettaConfig

	// TraceBlockByHash returns all traces for each transaction in the block
	// by calling geth debug_traceBlockByHash JSON RPC.
	// The output is map which key is transaction hash, and the value is list of
	// FlatCall. Each Flatcall is populated from one single trace.
	TraceBlockByHash(
		context.Context,
		common.Hash,
		[]evmClient.RPCTransaction,
	) (map[string][]*evmClient.FlatCall, error)

	// TraceTransaction returns all traces for one transaction
	// by calling geth debug_traceTransaction JSON RPC.
	// The output is a list of FlatCall. Each Flatcall is populated from one single trace.
	TraceTransaction(ctx context.Context, hash common.Hash) (json.RawMessage, []*evmClient.FlatCall, error)

	// BlockRewardTransaction returns the block reward Rosetta transaction for the miner
	BlockRewardTransaction(
		blockIdentifier *RosettaTypes.BlockIdentifier,
		miner string,
		uncles []*EthTypes.Header,
	) *RosettaTypes.Transaction

	// GetLoadedTransaction populates the LoadedTransaction which used by Rosetta data APIs
	GetLoadedTransaction(
		ctx context.Context,
		request *RosettaTypes.BlockTransactionRequest,
	) (*evmClient.LoadedTransaction, error)

	// BlockAuthor returns the block author
	BlockAuthor(ctx context.Context, blockIndex int64) (string, error)

	// GetUncles calls eth_getUncleByBlockHashAndIndex eth RPC to load uncle blocks
	GetUncles(
		ctx context.Context,
		head *EthTypes.Header,
		body *evmClient.RPCBlock,
	) ([]*EthTypes.Header, error)

	// TraceReplayBlockTransactions returns all traces for each transaction in the block
	// by calling open ethereum trace_replayBlockTransactions JSON RPC.
	// The output is map which key is transaction hash, and the value is list of
	// FlatCall. Each Flatcall is populated from one single trace.
	TraceReplayBlockTransactions(
		ctx context.Context,
		hsh string,
	) (map[string][]*evmClient.FlatCall, error)

	// TraceTransaction returns all traces for one transaction
	// by calling open ethereum trace_replayTransaction JSON RPC.
	// The output is a list of FlatCall. Each Flatcall is populated from one single trace.
	TraceReplayTransaction(ctx context.Context, hsh string) (json.RawMessage, []*evmClient.FlatCall, error)

	// PopulateCrossChainTransactions populates all the bridge transactions for the block
	// This method is used for blockchain that supports bridging function
	PopulateCrossChainTransactions(
		*EthTypes.Block,
		[]*evmClient.LoadedTransaction,
	) ([]*RosettaTypes.Transaction, error)

	// GetContractCurrency returns the ERC20 currency into for a specific token contract address
	GetContractCurrency(addr common.Address, erc20 bool) (*evmClient.ContractCurrency, error)

	// CallContext performs a JSON-RPC call with the given arguments.
	// The method is used by the JSON RPC Client, which is the interface
	// for accessing go-ethereum's JSON RPC endpoint.
	CallContext(ctx context.Context, result interface{}, method string, args ...interface{}) error

	// BatchCall sends all given requests as a single batch and waits for the server
	// to return a response for all of them. The wait duration is bounded by the
	// context's deadline.
	// The method is used by the JSON RPC Client, which is the interface
	// for accessing go-ethereum's JSON RPC endpoint.
	BatchCallContext(ctx context.Context, b []rpc.BatchElem) error

	// GetBlockReceipts returns a list of transaction receipts, used by Rosetta block api
	GetBlockReceipts(
		ctx context.Context,
		blockHash common.Hash,
		txs []evmClient.RPCTransaction,
		baseFee *big.Int,
	) ([]*evmClient.RosettaTxReceipt, error)

	// GetTransactionReceipt returns the Rosetta transaction receipt, used by Rosetta
	// block/transaction api
	GetTransactionReceipt(
		ctx context.Context,
		tx *evmClient.LoadedTransaction,
	) (*evmClient.RosettaTxReceipt, error)

	// GetNativeTransferGasLimit returns the estimated gas limit for the native currency transfer
	// This method is used by Rosetta construction/metadata api
	GetNativeTransferGasLimit(
		ctx context.Context,
		toAddress string,
		fromAddress string,
		value *big.Int,
	) (uint64, error)

	// GetErc20TransferGasLimit returns the estimated gas limit for the ERC20 token transfer
	// This method is used by Rosetta construction/metadata api
	GetErc20TransferGasLimit(
		ctx context.Context,
		toAddress string,
		fromAddress string,
		value *big.Int,
		currency *RosettaTypes.Currency,
	) (uint64, error)

	// GetErc20TransferGasLimit returns the estimated gas limit for the ERC20 token transfer
	// This method is used by Rosetta construction/metadata api
	GetContractCallGasLimit(
		ctx context.Context,
		toAddress string,
		fromAddress string,
		data []byte,
	) (uint64, error)
}
