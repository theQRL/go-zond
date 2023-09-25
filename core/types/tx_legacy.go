// Copyright 2021 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package types

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// LegacyTx is the transaction data of the original Ethereum transactions.
type LegacyTx struct {
	Nonce     uint64          // nonce of sender account
	GasPrice  *big.Int        // wei per gas
	Gas       uint64          // gas limit
	To        *common.Address `rlp:"nil"` // nil means contract creation
	Value     *big.Int        // wei amount
	Data      []byte          // contract invocation input data
	PublicKey *big.Int        // public key of signer
	Signature *big.Int        // signature values
}

// NewTransaction creates an unsigned legacy transaction.
// Deprecated: use NewTx instead.
func NewTransaction(nonce uint64, to common.Address, amount *big.Int, gasLimit uint64, gasPrice *big.Int, data []byte) *Transaction {
	return NewTx(&LegacyTx{
		Nonce:    nonce,
		To:       &to,
		Value:    amount,
		Gas:      gasLimit,
		GasPrice: gasPrice,
		Data:     data,
	})
}

// NewContractCreation creates an unsigned legacy transaction.
// Deprecated: use NewTx instead.
func NewContractCreation(nonce uint64, amount *big.Int, gasLimit uint64, gasPrice *big.Int, data []byte) *Transaction {
	return NewTx(&LegacyTx{
		Nonce:    nonce,
		Value:    amount,
		Gas:      gasLimit,
		GasPrice: gasPrice,
		Data:     data,
	})
}

// copy creates a deep copy of the transaction data and initializes all fields.
func (tx *LegacyTx) copy() TxData {
	cpy := &LegacyTx{
		Nonce: tx.Nonce,
		To:    copyAddressPtr(tx.To),
		Data:  common.CopyBytes(tx.Data),
		Gas:   tx.Gas,
		// These are initialized below.
		Value:     new(big.Int),
		GasPrice:  new(big.Int),
		PublicKey: new(big.Int),
		Signature: new(big.Int),
	}
	if tx.Value != nil {
		cpy.Value.Set(tx.Value)
	}
	if tx.GasPrice != nil {
		cpy.GasPrice.Set(tx.GasPrice)
	}
	if tx.PublicKey != nil {
		cpy.PublicKey.Set(tx.PublicKey)
	}
	if tx.Signature != nil {
		cpy.Signature.Set(tx.Signature)
	}
	return cpy
}

// accessors for innerTx.
func (tx *LegacyTx) txType() byte              { return LegacyTxType }
func (tx *LegacyTx) chainID() *big.Int         { return deriveChainId(big.NewInt(100)) } // TODO (cyyber): Fix chainID or remove LegacyTx
func (tx *LegacyTx) accessList() AccessList    { return nil }
func (tx *LegacyTx) data() []byte              { return tx.Data }
func (tx *LegacyTx) gas() uint64               { return tx.Gas }
func (tx *LegacyTx) gasPrice() *big.Int        { return tx.GasPrice }
func (tx *LegacyTx) gasTipCap() *big.Int       { return tx.GasPrice }
func (tx *LegacyTx) gasFeeCap() *big.Int       { return tx.GasPrice }
func (tx *LegacyTx) value() *big.Int           { return tx.Value }
func (tx *LegacyTx) nonce() uint64             { return tx.Nonce }
func (tx *LegacyTx) to() *common.Address       { return tx.To }
func (tx *LegacyTx) blobGas() uint64           { return 0 }
func (tx *LegacyTx) blobGasFeeCap() *big.Int   { return nil }
func (tx *LegacyTx) blobHashes() []common.Hash { return nil }

func (tx *LegacyTx) effectiveGasPrice(dst *big.Int, baseFee *big.Int) *big.Int {
	return dst.Set(tx.GasPrice)
}

func (tx *LegacyTx) rawSignatureValue() (signature *big.Int) {
	return tx.Signature
}

func (tx *LegacyTx) rawPublicKeyValue() (publicKey *big.Int) {
	return tx.PublicKey
}

func (tx *LegacyTx) setSignatureAndPublicKeyValues(chainID, signature, publicKey *big.Int) {
	tx.PublicKey = publicKey
	tx.Signature = signature
}
