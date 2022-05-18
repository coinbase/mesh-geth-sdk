// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package client

import (
	"errors"
	"math/big"
	"strings"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

// Reference imports to suppress errors if they are not otherwise used.
var (
	_ = errors.New
	_ = big.NewInt
	_ = strings.NewReader
	_ = ethereum.NotFound
	_ = bind.Bind
	_ = common.Big1
	_ = types.BloomLookup
	_ = event.NewSubscription
)

// ContractInfoTokenMetaData contains all meta data concerning the ContractInfoToken contract.
var ContractInfoTokenMetaData = &bind.MetaData{
	ABI: "[{\"type\":\"function\",\"stateMutability\":\"view\",\"outputs\":[{\"type\":\"uint8\",\"name\":\"\",\"internalType\":\"uint8\"}],\"name\":\"decimals\",\"inputs\":[]},{\"type\":\"function\",\"stateMutability\":\"view\",\"outputs\":[{\"type\":\"string\",\"name\":\"\",\"internalType\":\"string\"}],\"name\":\"symbol\",\"inputs\":[]}]",
}

// ContractInfoTokenABI is the input ABI used to generate the binding from.
// Deprecated: Use ContractInfoTokenMetaData.ABI instead.
var ContractInfoTokenABI = ContractInfoTokenMetaData.ABI

// ContractInfoToken is an auto generated Go binding around an Ethereum contract.
type ContractInfoToken struct {
	ContractInfoTokenCaller     // Read-only binding to the contract
	ContractInfoTokenTransactor // Write-only binding to the contract
	ContractInfoTokenFilterer   // Log filterer for contract events
}

// ContractInfoTokenCaller is an auto generated read-only Go binding around an Ethereum contract.
type ContractInfoTokenCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ContractInfoTokenTransactor is an auto generated write-only Go binding around an Ethereum contract.
type ContractInfoTokenTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ContractInfoTokenFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type ContractInfoTokenFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ContractInfoTokenSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type ContractInfoTokenSession struct {
	Contract     *ContractInfoToken // Generic contract binding to set the session for
	CallOpts     bind.CallOpts      // Call options to use throughout this session
	TransactOpts bind.TransactOpts  // Transaction auth options to use throughout this session
}

// ContractInfoTokenCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type ContractInfoTokenCallerSession struct {
	Contract *ContractInfoTokenCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts            // Call options to use throughout this session
}

// ContractInfoTokenTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type ContractInfoTokenTransactorSession struct {
	Contract     *ContractInfoTokenTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts            // Transaction auth options to use throughout this session
}

// ContractInfoTokenRaw is an auto generated low-level Go binding around an Ethereum contract.
type ContractInfoTokenRaw struct {
	Contract *ContractInfoToken // Generic contract binding to access the raw methods on
}

// ContractInfoTokenCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type ContractInfoTokenCallerRaw struct {
	Contract *ContractInfoTokenCaller // Generic read-only contract binding to access the raw methods on
}

// ContractInfoTokenTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type ContractInfoTokenTransactorRaw struct {
	Contract *ContractInfoTokenTransactor // Generic write-only contract binding to access the raw methods on
}

// NewContractInfoToken creates a new instance of ContractInfoToken, bound to a specific deployed contract.
func NewContractInfoToken(address common.Address, backend bind.ContractBackend) (*ContractInfoToken, error) {
	contract, err := bindContractInfoToken(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &ContractInfoToken{ContractInfoTokenCaller: ContractInfoTokenCaller{contract: contract}, ContractInfoTokenTransactor: ContractInfoTokenTransactor{contract: contract}, ContractInfoTokenFilterer: ContractInfoTokenFilterer{contract: contract}}, nil
}

// NewContractInfoTokenCaller creates a new read-only instance of ContractInfoToken, bound to a specific deployed contract.
func NewContractInfoTokenCaller(address common.Address, caller bind.ContractCaller) (*ContractInfoTokenCaller, error) {
	contract, err := bindContractInfoToken(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &ContractInfoTokenCaller{contract: contract}, nil
}

// NewContractInfoTokenTransactor creates a new write-only instance of ContractInfoToken, bound to a specific deployed contract.
func NewContractInfoTokenTransactor(address common.Address, transactor bind.ContractTransactor) (*ContractInfoTokenTransactor, error) {
	contract, err := bindContractInfoToken(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &ContractInfoTokenTransactor{contract: contract}, nil
}

// NewContractInfoTokenFilterer creates a new log filterer instance of ContractInfoToken, bound to a specific deployed contract.
func NewContractInfoTokenFilterer(address common.Address, filterer bind.ContractFilterer) (*ContractInfoTokenFilterer, error) {
	contract, err := bindContractInfoToken(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &ContractInfoTokenFilterer{contract: contract}, nil
}

// bindContractInfoToken binds a generic wrapper to an already deployed contract.
func bindContractInfoToken(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(ContractInfoTokenABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ContractInfoToken *ContractInfoTokenRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ContractInfoToken.Contract.ContractInfoTokenCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ContractInfoToken *ContractInfoTokenRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ContractInfoToken.Contract.ContractInfoTokenTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ContractInfoToken *ContractInfoTokenRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ContractInfoToken.Contract.ContractInfoTokenTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ContractInfoToken *ContractInfoTokenCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ContractInfoToken.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ContractInfoToken *ContractInfoTokenTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ContractInfoToken.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ContractInfoToken *ContractInfoTokenTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ContractInfoToken.Contract.contract.Transact(opts, method, params...)
}

// Decimals is a free data retrieval call binding the contract method 0x313ce567.
//
// Solidity: function decimals() view returns(uint8)
func (_ContractInfoToken *ContractInfoTokenCaller) Decimals(opts *bind.CallOpts) (uint8, error) {
	var out []interface{}
	err := _ContractInfoToken.contract.Call(opts, &out, "decimals")

	if err != nil {
		return *new(uint8), err
	}

	out0 := *abi.ConvertType(out[0], new(uint8)).(*uint8)

	return out0, err

}

// Decimals is a free data retrieval call binding the contract method 0x313ce567.
//
// Solidity: function decimals() view returns(uint8)
func (_ContractInfoToken *ContractInfoTokenSession) Decimals() (uint8, error) {
	return _ContractInfoToken.Contract.Decimals(&_ContractInfoToken.CallOpts)
}

// Decimals is a free data retrieval call binding the contract method 0x313ce567.
//
// Solidity: function decimals() view returns(uint8)
func (_ContractInfoToken *ContractInfoTokenCallerSession) Decimals() (uint8, error) {
	return _ContractInfoToken.Contract.Decimals(&_ContractInfoToken.CallOpts)
}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() view returns(string)
func (_ContractInfoToken *ContractInfoTokenCaller) Symbol(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _ContractInfoToken.contract.Call(opts, &out, "symbol")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() view returns(string)
func (_ContractInfoToken *ContractInfoTokenSession) Symbol() (string, error) {
	return _ContractInfoToken.Contract.Symbol(&_ContractInfoToken.CallOpts)
}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() view returns(string)
func (_ContractInfoToken *ContractInfoTokenCallerSession) Symbol() (string, error) {
	return _ContractInfoToken.Contract.Symbol(&_ContractInfoToken.CallOpts)
}
