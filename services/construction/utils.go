package construction

import (
	"github.com/coinbase/rosetta-geth-sdk/client"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func EthTransaction(tx *client.Transaction) *types.Transaction {
	var to *common.Address
	if tx.To != "" {
		x := common.HexToAddress(tx.To)
		to = &x
	}
	if eip1559Tx := tx.GasTipCap != nil && tx.GasFeeCap != nil; eip1559Tx {
		return types.NewTx(&types.DynamicFeeTx{
			Nonce:     tx.Nonce,
			GasTipCap: tx.GasTipCap,
			GasFeeCap: tx.GasFeeCap,
			Gas:       tx.GasLimit,
			To:        to,
			Value:     tx.Value,
			Data:      tx.Data,
		})
	} else {
		return types.NewTx(&types.LegacyTx{
			Nonce:    tx.Nonce,
			GasPrice: tx.GasPrice,
			Gas:      tx.GasLimit,
			To:       to,
			Value:    tx.Value,
			Data:     tx.Data,
		})
	}
}
