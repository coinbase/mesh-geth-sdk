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

	eip1559Tx := tx.GasTipCap != nil && tx.GasFeeCap != nil
	if eip1559Tx {
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
