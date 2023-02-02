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
	"math/big"
	"testing"

	evmClient "github.com/coinbase/rosetta-geth-sdk/client"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
)

func TestParseTransferOps(t *testing.T) {
	a1 := common.HexToAddress("0xdd4b76b0316dcafa98862a12a92791ac9426a0e2")
	a2 := common.HexToAddress("0xdff384f754e854890e311e3280b767f80797291e")
	a3 := common.HexToAddress("0xd345e41ae2cb00311956aa7109fc801ae8c81a52")
	a4 := common.HexToAddress("0xdcceff0438b6391212ad726d42c150050d9a0226")

	m := &evmClient.FlatCall{
		Type: "call",
		BeforeEVMTransfers: []*evmClient.EVMTransfer{
			{
				From:    &a1,
				Purpose: "escrow",
				To:      &a2,
				Value:   big.NewInt(0),
			},
			{
				From:    nil,
				Purpose: "prepaid",
				To:      &a2,
				Value:   big.NewInt(20000000000000),
			},
			{
				From:    &a2,
				Purpose: "feePayment",
				To:      nil,
				Value:   big.NewInt(20000000000000),
			},
		},
		AfterEVMTransfers: []*evmClient.EVMTransfer{
			{
				From:    nil,
				Purpose: "gasRefund",
				To:      &a2,
				Value:   big.NewInt(11379900000000),
			},
			{
				From:    &a2,
				Purpose: "undoRefund",
				To:      nil,
				Value:   big.NewInt(11379900000000),
			},
			{
				From:    &a3,
				Purpose: "refund",
				To:      &a4,
				Value:   big.NewInt(41880898787024),
			},
			{
				From:    &a3,
				Purpose: "refund",
				To:      &a2,
				Value:   big.NewInt(0),
			},
			{
				From:    &a3,
				Purpose: "refund",
				To:      &a4,
				Value:   big.NewInt(11379900000000),
			},
			{
				From:    &a3,
				Purpose: "refund",
				To:      &a2,
				Value:   big.NewInt(0),
			},
			{
				From:    &a1,
				Purpose: "escrow",
				To:      &a4,
				Value:   big.NewInt(0),
			},
		},
		From:         common.HexToAddress("0x1234"),
		To:           common.HexToAddress("0x4566"),
		Value:        big.NewInt(900000),
		GasUsed:      big.NewInt(10000),
		Revert:       false,
		ErrorMessage: "",
	}

	loadedTxn := &evmClient.LoadedTransaction{
		Trace: []*evmClient.FlatCall{m},
	}
	ops := TransferOps(loadedTxn, 0)

	assert.Equal(t, len(ops), 3)
	assert.Equal(t, ops[0].Account.Address, a2.String())
	assert.Equal(t, ops[0].Amount.Value, "0")
	assert.Equal(t, ops[0].OperationIdentifier.Index, int64(0))

	assert.Equal(t, ops[1].Account.Address, a3.String())
	assert.Equal(t, ops[1].Amount.Value, "-53260798787024")
	assert.Equal(t, ops[1].OperationIdentifier.Index, int64(1))

	assert.Equal(t, ops[2].Account.Address, a4.String())
	assert.Equal(t, ops[2].Amount.Value, "53260798787024")
	assert.Equal(t, ops[2].OperationIdentifier.Index, int64(2))
	assert.Equal(t, ops[2].RelatedOperations[0].Index, int64(1))
}

func TestParseTransferOpsFirstIndexRelatedOps(t *testing.T) {
	a1 := common.HexToAddress("0xea3123e9d9911199a6711321d1277285e6d4f3ec")
	a2 := common.HexToAddress("0xd345e41ae2cb00311956aa7109fc801ae8c81a52")
	a3 := common.HexToAddress("0xa44daf8e331e59cdb2be9c26cdffa719eea0ddb7")
	a4 := common.HexToAddress("0xf7401f57a705b71a9463d8b5a2a59ea0161c93f0")
	a5 := common.HexToAddress("0x963d5de08b9beb014cad171c16b12a8d7456fb50")

	m := &evmClient.FlatCall{
		Type: "call",
		BeforeEVMTransfers: []*evmClient.EVMTransfer{
			{
				From:    &a3,
				Purpose: "escrow",
				To:      &a1,
				Value:   big.NewInt(560000000000000000),
			},
			{
				From:    nil,
				Purpose: "prepaid",
				To:      &a1,
				Value:   big.NewInt(15621500000000),
			},
			{
				From:    &a1,
				Purpose: "feePayment",
				To:      nil,
				Value:   big.NewInt(15621500000000),
			},
		},
		AfterEVMTransfers: []*evmClient.EVMTransfer{
			{
				From:    nil,
				Purpose: "gasRefund",
				To:      &a1,
				Value:   big.NewInt(7441300000000),
			},
			{
				From:    &a1,
				Purpose: "undoRefund",
				To:      nil,
				Value:   big.NewInt(7441300000000),
			},
			{
				From:    &a2,
				Purpose: "refund",
				To:      &a4,
				Value:   big.NewInt(43441788314320),
			},
			{
				From:    &a2,
				Purpose: "refund",
				To:      &a1,
				Value:   big.NewInt(0),
			},
			{
				From:    &a2,
				Purpose: "refund",
				To:      &a4,
				Value:   big.NewInt(7441300000000),
			},
			{
				From:    &a2,
				Purpose: "refund",
				To:      &a1,
				Value:   big.NewInt(0),
			},
			{
				From:    &a3,
				Purpose: "escrow",
				To:      &a5,
				Value:   big.NewInt(0),
			},
		},
		From:         common.HexToAddress("0x1234"),
		To:           common.HexToAddress("0x4566"),
		Value:        big.NewInt(900000),
		GasUsed:      big.NewInt(10000),
		Revert:       false,
		ErrorMessage: "",
	}

	loadedTxn := &evmClient.LoadedTransaction{
		Trace: []*evmClient.FlatCall{m},
	}
	ops := TransferOps(loadedTxn, 0)

	assert.Equal(t, len(ops), 5)
	assert.Equal(t, ops[0].Account.Address, a3.String())
	assert.Equal(t, ops[0].Amount.Value, "-560000000000000000")
	assert.Equal(t, ops[0].OperationIdentifier.Index, int64(0))

	assert.Equal(t, ops[1].Account.Address, a1.String())
	assert.Equal(t, ops[1].Amount.Value, "560000000000000000")
	assert.Equal(t, ops[1].OperationIdentifier.Index, int64(1))
	assert.Equal(t, ops[1].RelatedOperations[0].Index, int64(0))

	assert.Equal(t, ops[2].Account.Address, a1.String())
	assert.Equal(t, ops[2].Amount.Value, "0")
	assert.Equal(t, ops[2].OperationIdentifier.Index, int64(2))

	assert.Equal(t, ops[3].Account.Address, a2.String())
	assert.Equal(t, ops[3].Amount.Value, "-50883088314320")
	assert.Equal(t, ops[3].OperationIdentifier.Index, int64(3))

	assert.Equal(t, ops[4].Account.Address, a4.String())
	assert.Equal(t, ops[4].Amount.Value, "50883088314320")
	assert.Equal(t, ops[4].OperationIdentifier.Index, int64(4))
	assert.Equal(t, ops[4].RelatedOperations[0].Index, int64(3))
}
