// Copyright 2026 Coinbase, Inc.
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
	sdkTypes "github.com/coinbase/rosetta-geth-sdk/types"
	RosettaTypes "github.com/coinbase/rosetta-sdk-go/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
)

var (
	traceOuter    = common.HexToAddress("0x1000000000000000000000000000000000000001")
	traceWrapper  = common.HexToAddress("0x2000000000000000000000000000000000000002")
	traceContract = common.HexToAddress("0x3000000000000000000000000000000000000003")
	traceBenef    = common.HexToAddress("0x4000000000000000000000000000000000000004")
	traceSink     = common.HexToAddress("0x5000000000000000000000000000000000000005")
)

func destructOps(ops []*RosettaTypes.Operation) []*RosettaTypes.Operation {
	var out []*RosettaTypes.Operation
	for _, op := range ops {
		if op.Type == sdkTypes.DestructOpType {
			out = append(out, op)
		}
	}
	return out
}

// A SELFDESTRUCT inside a reverted frame is rolled back by the EVM, so the
// account keeps its balance and can still send value afterwards. It should
// not be treated as a destroyed account.
func TestTraceOpsRevertedSelfDestructThenDebit(t *testing.T) {
	oneEth := big.NewInt(1e18)
	calls := []*evmClient.FlatCall{
		{Type: "CALL", From: traceOuter, To: traceWrapper, Value: big.NewInt(0), Revert: true},
		{Type: "CALL", From: traceWrapper, To: traceContract, Value: big.NewInt(0), Revert: true},
		{Type: "SELFDESTRUCT", From: traceContract, To: traceBenef, Value: oneEth, Revert: true},
		{Type: "CALL", From: traceOuter, To: traceContract, Value: big.NewInt(0)},
		{Type: "CALL", From: traceContract, To: traceSink, Value: oneEth},
	}

	ops := TraceOps(calls, 0)
	assert.Empty(t, destructOps(ops))

	// The reverted SELFDESTRUCT is still reported, with failure status.
	assert.Equal(t, "SELFDESTRUCT", ops[0].Type)
	assert.Equal(t, sdkTypes.FailureStatus, *ops[0].Status)
}

// Under EIP-6780, SELFDESTRUCT of a pre-existing contract to itself does not
// remove its balance, so the contract can still send value afterwards. No
// DESTRUCT operation should be emitted.
func TestTraceOpsSelfDestructToSelfThenDebit(t *testing.T) {
	oneEth := big.NewInt(1e18)
	calls := []*evmClient.FlatCall{
		{Type: "SELFDESTRUCT", From: traceContract, To: traceContract, Value: oneEth},
		{Type: "CALL", From: traceContract, To: traceSink, Value: oneEth},
	}

	ops := TraceOps(calls, 0)
	assert.Empty(t, destructOps(ops))
}

// Existing behavior: value received by an account after a successful
// SELFDESTRUCT is burned at transaction finalization via a DESTRUCT op.
func TestTraceOpsSuccessfulSelfDestructThenCredit(t *testing.T) {
	oneEth := big.NewInt(1e18)
	half := big.NewInt(5e17)
	calls := []*evmClient.FlatCall{
		{Type: "SELFDESTRUCT", From: traceContract, To: traceBenef, Value: oneEth},
		{Type: "CALL", From: traceOuter, To: traceContract, Value: half},
	}

	ops := TraceOps(calls, 0)
	destructs := destructOps(ops)
	if assert.Len(t, destructs, 1) {
		assert.Equal(t, traceContract.Hex(), destructs[0].Account.Address)
		assert.Equal(t, "-500000000000000000", destructs[0].Amount.Value)
		assert.Equal(t, int64(len(ops)-1), destructs[0].OperationIdentifier.Index)
	}
}

// A credit to a destroyed account that is later reverted must not be burned.
func TestTraceOpsSuccessfulSelfDestructThenRevertedCredit(t *testing.T) {
	oneEth := big.NewInt(1e18)
	calls := []*evmClient.FlatCall{
		{Type: "SELFDESTRUCT", From: traceContract, To: traceBenef, Value: oneEth},
		{Type: "CALL", From: traceOuter, To: traceContract, Value: oneEth, Revert: true},
	}

	ops := TraceOps(calls, 0)
	assert.Empty(t, destructOps(ops))
}
