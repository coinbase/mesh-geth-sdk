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
	evmClient "github.com/coinbase/rosetta-geth-sdk/client"
	RosettaTypes "github.com/coinbase/rosetta-sdk-go/types"
	EthTypes "github.com/ethereum/go-ethereum/core/types"

	"log"
	"math/big"
	"strings"

	sdkTypes "github.com/coinbase/rosetta-geth-sdk/types"
)

const (
	TopicsInErc20DepositOrWithdrawal = 2
	TopicsInErc20Transfer            = 3
	zeroAddress                      = "0x0000000000000000000000000000000000000000000000000000000000000000"
)

func parseTransferOps(startIndex int, transfers []*evmClient.EVMTransfer, addrs map[string]*RosettaTypes.Operation) (
	[]*RosettaTypes.Operation, map[string]*RosettaTypes.Operation) {
	count := startIndex
	var ops []*RosettaTypes.Operation
	for _, transfer := range transfers {
		var address string
		var key string
		amount := transfer.Value
		if amount.Uint64() == 0 {
			continue
		}
		shouldAdd := true
		if transfer.From == nil {
			address = transfer.To.String()
			shouldAdd = false
			key = address
		} else if transfer.To == nil {
			address = transfer.From.String()
			amount = new(big.Int).Neg(transfer.Value)
			shouldAdd = false
			key = address
		}

		if shouldAdd {
			address = transfer.From.String()
			amount = new(big.Int).Neg(transfer.Value)
			key = transfer.From.String() + transfer.To.String()
		}

		val, exists := addrs[key]
		if exists {
			amt := new(big.Int)
			amt, ok := amt.SetString(val.Amount.Value, 10) // nolint:gomnd
			if !ok {
				log.Println("error consolidating transfer data")
				return nil, nil
			}
			newAmt := amt.Add(amt, amount)
			addrs[key].Amount.Value = newAmt.String()
			if shouldAdd {
				addrs[transfer.To.String()+transfer.From.String()].Amount.Value = new(big.Int).Neg(newAmt).String()
			}
			continue
		}

		singleOp := &RosettaTypes.Operation{
			OperationIdentifier: &RosettaTypes.OperationIdentifier{
				Index: int64(count),
			},
			Type:   sdkTypes.FeeOpType,
			Status: RosettaTypes.String(sdkTypes.SuccessStatus),
			Account: &RosettaTypes.AccountIdentifier{
				Address: address,
			},
			Amount: evmClient.Amount(amount, sdkTypes.Currency),
		}
		addrs[key] = singleOp
		ops = append(ops, singleOp)

		if shouldAdd {
			doubleOp := &RosettaTypes.Operation{
				OperationIdentifier: &RosettaTypes.OperationIdentifier{
					Index: int64(count + 1),
				},
				RelatedOperations: []*RosettaTypes.OperationIdentifier{
					{
						Index: int64(count),
					},
				},
				Type:   sdkTypes.FeeOpType,
				Status: RosettaTypes.String(sdkTypes.SuccessStatus),
				Account: &RosettaTypes.AccountIdentifier{
					Address: transfer.To.String(),
				},
				Amount: evmClient.Amount(transfer.Value, sdkTypes.Currency),
			}
			doubleKey := transfer.To.String() + transfer.From.String()
			addrs[doubleKey] = doubleOp
			ops = append(ops, doubleOp)
			count++
		}
		count++
	}
	return ops, addrs
}

func TransferOps(tx *evmClient.LoadedTransaction, startIndex int) []*RosettaTypes.Operation {
	var ops []*RosettaTypes.Operation
	addrMap := make(map[string]*RosettaTypes.Operation)
	for _, trace := range tx.Trace {
		beforeOps, addrMap := parseTransferOps(startIndex+len(ops), trace.BeforeEVMTransfers, addrMap)
		ops = append(ops, beforeOps...)
		afterOps, _ := parseTransferOps(startIndex+len(ops), trace.AfterEVMTransfers, addrMap)
		ops = append(ops, afterOps...)
	}
	return ops
}

func FeeOps(tx *evmClient.LoadedTransaction, args ...interface{}) []*RosettaTypes.Operation {
	var currency = getCurrency(args)

	var minerEarnedAmount *big.Int
	if tx.FeeBurned == nil {
		minerEarnedAmount = tx.FeeAmount
	} else {
		minerEarnedAmount = new(big.Int).Sub(tx.FeeAmount, tx.FeeBurned)
	}

	if minerEarnedAmount == nil {
		return nil
	}

	feeRewarder := tx.Miner
	if len(tx.Author) > 0 {
		feeRewarder = tx.Author
	}

	ops := []*RosettaTypes.Operation{
		{
			OperationIdentifier: &RosettaTypes.OperationIdentifier{
				Index: 0,
			},
			Type:   sdkTypes.FeeOpType,
			Status: RosettaTypes.String(sdkTypes.SuccessStatus),
			Account: &RosettaTypes.AccountIdentifier{
				Address: evmClient.MustChecksum(tx.From.String()),
			},
			Amount: evmClient.Amount(new(big.Int).Neg(minerEarnedAmount), currency),
		},

		{
			OperationIdentifier: &RosettaTypes.OperationIdentifier{
				Index: 1,
			},
			RelatedOperations: []*RosettaTypes.OperationIdentifier{
				{
					Index: 0,
				},
			},
			Type:   sdkTypes.FeeOpType,
			Status: RosettaTypes.String(sdkTypes.SuccessStatus),
			Account: &RosettaTypes.AccountIdentifier{
				Address: evmClient.MustChecksum(feeRewarder),
			},
			Amount: evmClient.Amount(minerEarnedAmount, currency),
		},
	}

	if tx.FeeBurned == nil {
		return ops
	}

	burntOp := &RosettaTypes.Operation{
		OperationIdentifier: &RosettaTypes.OperationIdentifier{
			Index: 2, // nolint:gomnd
		},
		Type:    sdkTypes.FeeOpType,
		Status:  RosettaTypes.String(sdkTypes.SuccessStatus),
		Account: evmClient.Account(tx.From),
		Amount:  evmClient.Amount(new(big.Int).Neg(tx.FeeBurned), sdkTypes.Currency),
	}

	ops = append(ops, burntOp)

	return ops
}

func getCurrency(params []interface{}) *RosettaTypes.Currency {
	var nativeCurrency *RosettaTypes.Currency
	if len(params) > 0 {
		if currencyParam, ok := params[0].(*RosettaTypes.Currency); ok {
			nativeCurrency = currencyParam
		}
	}
	if nativeCurrency == nil {
		nativeCurrency = sdkTypes.Currency
	}
	return nativeCurrency
}

// TraceOps returns all *RosettaTypes.Operation for a given
// array of flattened traces.
// nolint:gocognit
func TraceOps(
	calls []*evmClient.FlatCall,
	startIndex int,
	args ...interface{},
) []*RosettaTypes.Operation { // nolint: gocognit
	var currency = getCurrency(args)
	var ops []*RosettaTypes.Operation
	if len(calls) == 0 {
		return ops
	}

	destroyedAccounts := map[string]*big.Int{}
	for _, trace := range calls {
		// Handle partial transaction success
		metadata := map[string]interface{}{}
		opStatus := sdkTypes.SuccessStatus
		if trace.Revert {
			opStatus = sdkTypes.FailureStatus
			metadata["error"] = trace.ErrorMessage
		}

		var zeroValue bool
		if trace.Value.Sign() == 0 {
			zeroValue = true
		}

		// Skip all 0 value CallType operations (TODO: make optional to include)
		//
		// We can't continue here because we may need to adjust our destroyed
		// accounts map if a CallTYpe operation resurrects an account.
		shouldAdd := true
		traceType := strings.ToUpper(trace.Type)
		if zeroValue && sdkTypes.CallType(trace.Type) {
			shouldAdd = false
		}

		// Checksum addresses
		from := evmClient.MustChecksum(trace.From.String())
		to := evmClient.MustChecksum(trace.To.String())

		if shouldAdd {
			fromOp := &RosettaTypes.Operation{
				OperationIdentifier: &RosettaTypes.OperationIdentifier{
					Index: int64(len(ops) + startIndex),
				},
				Type:   traceType,
				Status: RosettaTypes.String(opStatus),
				Account: &RosettaTypes.AccountIdentifier{
					Address: from,
				},
				Amount: &RosettaTypes.Amount{
					Value:    new(big.Int).Neg(trace.Value).String(),
					Currency: currency,
				},
				Metadata: metadata,
			}
			if zeroValue {
				fromOp.Amount = nil
			} else {
				_, destroyed := destroyedAccounts[from]
				if destroyed && opStatus == sdkTypes.SuccessStatus {
					destroyedAccounts[from] = new(big.Int).Sub(destroyedAccounts[from], trace.Value)
				}
			}

			ops = append(ops, fromOp)
		}

		// Add to destroyed accounts if SELFDESTRUCT
		// and overwrite existing balance.
		if traceType == sdkTypes.SelfDestructOpType {
			destroyedAccounts[from] = new(big.Int)

			// If destination of of SELFDESTRUCT is self,
			// we should skip. In the EVM, the balance is reset
			// after the balance is increased on the destination
			// so this is a no-op.
			if from == to {
				continue
			}
		}

		// Skip empty to addresses (this may not
		// actually occur but leaving it as a
		// sanity check)
		if len(trace.To.String()) == 0 {
			continue
		}

		// If the account is resurrected, we remove it from
		// the destroyed accounts map.
		if sdkTypes.CreateType(traceType) {
			delete(destroyedAccounts, to)
		}

		if shouldAdd {
			lastOpIndex := ops[len(ops)-1].OperationIdentifier.Index
			toOp := &RosettaTypes.Operation{
				OperationIdentifier: &RosettaTypes.OperationIdentifier{
					Index: lastOpIndex + 1,
				},
				RelatedOperations: []*RosettaTypes.OperationIdentifier{
					{
						Index: lastOpIndex,
					},
				},
				Type:   traceType,
				Status: RosettaTypes.String(opStatus),
				Account: &RosettaTypes.AccountIdentifier{
					Address: to,
				},
				Amount: &RosettaTypes.Amount{
					Value:    trace.Value.String(),
					Currency: currency,
				},
				Metadata: metadata,
			}

			if zeroValue {
				toOp.Amount = nil
			} else {
				_, destroyed := destroyedAccounts[to]
				if destroyed && opStatus == sdkTypes.SuccessStatus {
					destroyedAccounts[to] = new(big.Int).Add(destroyedAccounts[to], trace.Value)
				}
			}

			ops = append(ops, toOp)
		}
	}

	// Zero-out all destroyed accounts that are removed
	// during transaction finalization.
	for acct, val := range destroyedAccounts {
		if _, err := evmClient.ChecksumAddress(acct); err != nil {
			continue
		}

		if val.Sign() == 0 {
			continue
		}

		if val.Sign() < 0 {
			log.Fatalf("negative balance for suicided account %s: %s\n", acct, val.String())
		}

		ops = append(ops, &RosettaTypes.Operation{
			OperationIdentifier: &RosettaTypes.OperationIdentifier{
				Index: ops[len(ops)-1].OperationIdentifier.Index + 1,
			},
			Type:   sdkTypes.DestructOpType,
			Status: RosettaTypes.String(sdkTypes.SuccessStatus),
			Account: &RosettaTypes.AccountIdentifier{
				Address: acct,
			},
			Amount: &RosettaTypes.Amount{
				Value:    new(big.Int).Neg(val).String(),
				Currency: currency,
			},
		})
	}

	return ops
}

// Erc20Ops returns a list of erc20 operations parsed from the log from a transaction receipt
func Erc20Ops(
	transferLog *EthTypes.Log,
	currency *evmClient.ContractCurrency,
	opsLen int64,
) []*RosettaTypes.Operation {
	ops := []*RosettaTypes.Operation{}
	contractAddress := transferLog.Address
	event := transferLog.Topics[0]

	if len(transferLog.Topics) == TopicsInErc20DepositOrWithdrawal {
		address := transferLog.Topics[1]

		if event.Hex() == evmClient.Erc20LogTopicMap[evmClient.Erc20DepositLogTopic] {
			mintOp := RosettaTypes.Operation{
				OperationIdentifier: &RosettaTypes.OperationIdentifier{
					Index: opsLen,
				},
				Status:  RosettaTypes.String(sdkTypes.SuccessStatus),
				Type:    sdkTypes.OpErc20Mint,
				Amount:  evmClient.Erc20Amount(transferLog.Data, contractAddress, currency.Symbol, currency.Decimals, false),
				Account: evmClient.Account(evmClient.ConvertEVMTopicHashToAddress(&address)),
			}
			ops = append(ops, &mintOp)
			return ops
		}

		if event.Hex() == evmClient.Erc20LogTopicMap[evmClient.Erc20WithdrawalLogTopic] {
			burnOp := RosettaTypes.Operation{
				OperationIdentifier: &RosettaTypes.OperationIdentifier{
					Index: opsLen,
				},
				Status:  RosettaTypes.String(sdkTypes.SuccessStatus),
				Type:    sdkTypes.OpErc20Burn,
				Amount:  evmClient.Erc20Amount(transferLog.Data, contractAddress, currency.Symbol, currency.Decimals, true),
				Account: evmClient.Account(evmClient.ConvertEVMTopicHashToAddress(&address)),
			}
			ops = append(ops, &burnOp)
			return ops
		}
	}

	if len(transferLog.Topics) == TopicsInErc20Transfer {
		addressFrom := transferLog.Topics[1]
		addressTo := transferLog.Topics[2]

		if addressFrom.Hex() == zeroAddress {
			mintOp := RosettaTypes.Operation{
				OperationIdentifier: &RosettaTypes.OperationIdentifier{
					Index: opsLen,
				},
				Status:  RosettaTypes.String(sdkTypes.SuccessStatus),
				Type:    sdkTypes.OpErc20Mint,
				Amount:  evmClient.Erc20Amount(transferLog.Data, contractAddress, currency.Symbol, currency.Decimals, false),
				Account: evmClient.Account(evmClient.ConvertEVMTopicHashToAddress(&addressTo)),
			}
			ops = append(ops, &mintOp)
			return ops
		}

		if addressTo.Hex() == zeroAddress {
			burnOp := RosettaTypes.Operation{
				OperationIdentifier: &RosettaTypes.OperationIdentifier{
					Index: opsLen,
				},
				Status:  RosettaTypes.String(sdkTypes.SuccessStatus),
				Type:    sdkTypes.OpErc20Burn,
				Amount:  evmClient.Erc20Amount(transferLog.Data, contractAddress, currency.Symbol, currency.Decimals, true),
				Account: evmClient.Account(evmClient.ConvertEVMTopicHashToAddress(&addressFrom)),
			}
			ops = append(ops, &burnOp)
			return ops
		}

		if event.Hex() == evmClient.Erc20LogTopicMap[evmClient.Erc20TransferLogTopic] {
			sendingOp := RosettaTypes.Operation{
				OperationIdentifier: &RosettaTypes.OperationIdentifier{
					Index: opsLen,
				},
				Status:  RosettaTypes.String(sdkTypes.SuccessStatus),
				Type:    sdkTypes.OpErc20Transfer,
				Amount:  evmClient.Erc20Amount(transferLog.Data, contractAddress, currency.Symbol, currency.Decimals, true),
				Account: evmClient.Account(evmClient.ConvertEVMTopicHashToAddress(&addressFrom)),
			}
			receiptOp := RosettaTypes.Operation{
				OperationIdentifier: &RosettaTypes.OperationIdentifier{
					Index: opsLen + 1,
				},
				Status:  RosettaTypes.String(sdkTypes.SuccessStatus),
				Type:    sdkTypes.OpErc20Transfer,
				Amount:  evmClient.Erc20Amount(transferLog.Data, contractAddress, currency.Symbol, currency.Decimals, false),
				Account: evmClient.Account(evmClient.ConvertEVMTopicHashToAddress(&addressTo)),
				RelatedOperations: []*RosettaTypes.OperationIdentifier{
					{
						Index: opsLen,
					},
				},
			}
			ops = append(ops, &sendingOp)
			ops = append(ops, &receiptOp)
			return ops
		}
	}

	return ops
}
