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
	"bytes"
	"math/big"

	"github.com/theQRL/go-zond/common"
	"github.com/theQRL/go-zond/crypto/pqcrypto"
	"github.com/theQRL/go-zond/rlp"
)

// DynamicFeeTx represents an EIP-1559 transaction.
type DynamicFeeTx struct {
	ChainID    *big.Int
	Nonce      uint64
	GasTipCap  *big.Int // a.k.a. maxPriorityFeePerGas
	GasFeeCap  *big.Int // a.k.a. maxFeePerGas
	Gas        uint64
	To         *common.Address `rlp:"nil"` // nil means contract creation
	Value      *big.Int
	Data       []byte
	AccessList AccessList

	// Public Key & Signature values
	PublicKey []byte
	Signature []byte
}

// copy creates a deep copy of the transaction data and initializes all fields.
func (tx *DynamicFeeTx) copy() TxData {
	cpy := &DynamicFeeTx{
		Nonce: tx.Nonce,
		To:    copyAddressPtr(tx.To),
		Data:  common.CopyBytes(tx.Data),
		Gas:   tx.Gas,
		// These are copied below.
		AccessList: make(AccessList, len(tx.AccessList)),
		Value:      new(big.Int),
		ChainID:    new(big.Int),
		GasTipCap:  new(big.Int),
		GasFeeCap:  new(big.Int),
		PublicKey:  make([]byte, pqcrypto.DilithiumPublicKeyLength),
		Signature:  make([]byte, pqcrypto.DilithiumSignatureLength),
	}
	copy(cpy.AccessList, tx.AccessList)
	if tx.Value != nil {
		cpy.Value.Set(tx.Value)
	}
	if tx.ChainID != nil {
		cpy.ChainID.Set(tx.ChainID)
	}
	if tx.GasTipCap != nil {
		cpy.GasTipCap.Set(tx.GasTipCap)
	}
	if tx.GasFeeCap != nil {
		cpy.GasFeeCap.Set(tx.GasFeeCap)
	}
	if tx.PublicKey != nil {
		copy(cpy.PublicKey[:pqcrypto.DilithiumPublicKeyLength], tx.PublicKey)
	}
	if tx.Signature != nil {
		copy(cpy.Signature[:pqcrypto.DilithiumSignatureLength], tx.Signature)
	}
	return cpy
}

// accessors for innerTx.
func (tx *DynamicFeeTx) txType() byte           { return DynamicFeeTxType }
func (tx *DynamicFeeTx) chainID() *big.Int      { return tx.ChainID }
func (tx *DynamicFeeTx) accessList() AccessList { return tx.AccessList }
func (tx *DynamicFeeTx) data() []byte           { return tx.Data }
func (tx *DynamicFeeTx) gas() uint64            { return tx.Gas }
func (tx *DynamicFeeTx) gasFeeCap() *big.Int    { return tx.GasFeeCap }
func (tx *DynamicFeeTx) gasTipCap() *big.Int    { return tx.GasTipCap }
func (tx *DynamicFeeTx) gasPrice() *big.Int     { return tx.GasFeeCap }
func (tx *DynamicFeeTx) value() *big.Int        { return tx.Value }
func (tx *DynamicFeeTx) nonce() uint64          { return tx.Nonce }
func (tx *DynamicFeeTx) to() *common.Address    { return tx.To }

func (tx *DynamicFeeTx) effectiveGasPrice(dst *big.Int, baseFee *big.Int) *big.Int {
	if baseFee == nil {
		return dst.Set(tx.GasFeeCap)
	}
	tip := dst.Sub(tx.GasFeeCap, baseFee)
	if tip.Cmp(tx.GasTipCap) > 0 {
		tip.Set(tx.GasTipCap)
	}
	return tip.Add(tip, baseFee)
}

func (tx *DynamicFeeTx) rawSignatureValue() (signature []byte) {
	return tx.Signature
}

func (tx *DynamicFeeTx) rawPublicKeyValue() (publicKey []byte) {
	return tx.PublicKey
}

func (tx *DynamicFeeTx) setSignatureAndPublicKeyValues(chainID *big.Int, signature, publicKey []byte) {
	tx.ChainID, tx.PublicKey, tx.Signature = chainID, publicKey, signature
}

func (tx *DynamicFeeTx) encode(b *bytes.Buffer) error {
	return rlp.Encode(b, tx)
}

func (tx *DynamicFeeTx) decode(input []byte) error {
	return rlp.DecodeBytes(input, tx)
}
