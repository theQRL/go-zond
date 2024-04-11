// Copyright 2016 The go-ethereum Authors
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
	"errors"
	"fmt"
	"math/big"

	"github.com/theQRL/go-qrllib/dilithium"
	"github.com/theQRL/go-zond/common"
	"github.com/theQRL/go-zond/crypto/pqcrypto"
	"github.com/theQRL/go-zond/params"
)

var ErrInvalidChainId = errors.New("invalid chain id for signer")

// sigCache is used to cache the derived sender and contains
// the signer used to derive it.
type sigCache struct {
	signer Signer
	from   common.Address
}

// MakeSigner returns a Signer based on the given chain config and block number.
func MakeSigner(config *params.ChainConfig, blockNumber *big.Int, blockTime uint64) Signer {
	var signer Signer
	switch {
	case config.IsLondon(blockNumber):
		signer = NewLondonSigner(config.ChainID)
	case config.IsBerlin(blockNumber):
		signer = NewEIP2930Signer(config.ChainID)
	case config.IsEIP155(blockNumber):
		signer = NewEIP155Signer(config.ChainID)
	case config.IsHomestead(blockNumber):
		signer = HomesteadSigner{}
	default:
		signer = FrontierSigner{}
	}
	return signer
}

// LatestSigner returns the 'most permissive' Signer available for the given chain
// configuration. Specifically, this enables support of all types of transacrions
// when their respective forks are scheduled to occur at any block number (or time)
// in the chain config.
//
// Use this in transaction-handling code where the current block number is unknown. If you
// have the current block number available, use MakeSigner instead.
func LatestSigner(config *params.ChainConfig) Signer {
	if config.ChainID != nil {
		if config.LondonBlock != nil {
			return NewLondonSigner(config.ChainID)
		}
		if config.BerlinBlock != nil {
			return NewEIP2930Signer(config.ChainID)
		}
		if config.EIP155Block != nil {
			return NewEIP155Signer(config.ChainID)
		}
	}
	return HomesteadSigner{}
}

// LatestSignerForChainID returns the 'most permissive' Signer available. Specifically,
// this enables support for EIP-155 replay protection and all implemented EIP-2718
// transaction types if chainID is non-nil.
//
// Use this in transaction-handling code where the current block number and fork
// configuration are unknown. If you have a ChainConfig, use LatestSigner instead.
// If you have a ChainConfig and know the current block number, use MakeSigner instead.
func LatestSignerForChainID(chainID *big.Int) Signer {
	if chainID == nil {
		return HomesteadSigner{}
	}
	return NewLondonSigner(chainID)
}

// SignTx signs the transaction using the given dilithium signer and private key.
func SignTx(tx *Transaction, s Signer, d *dilithium.Dilithium) (*Transaction, error) {
	h := s.Hash(tx)
	sig, err := pqcrypto.Sign(h[:], d)
	if err != nil {
		return nil, err
	}
	pk := d.GetPK()
	return tx.WithSignatureAndPublicKey(s, sig[:], pk[:])
}

// SignNewTx creates a transaction and signs it.
func SignNewTx(d *dilithium.Dilithium, s Signer, txdata TxData) (*Transaction, error) {
	tx := NewTx(txdata)
	h := s.Hash(tx)
	sig, err := pqcrypto.Sign(h[:], d)
	if err != nil {
		return nil, err
	}
	pk := d.GetPK()
	return tx.WithSignatureAndPublicKey(s, sig, pk[:])
}

// MustSignNewTx creates a transaction and signs it.
// This panics if the transaction cannot be signed.
func MustSignNewTx(d *dilithium.Dilithium, s Signer, txdata TxData) *Transaction {
	tx, err := SignNewTx(d, s, txdata)
	if err != nil {
		panic(err)
	}
	return tx
}

// Sender returns the address derived from the signature (V, R, S) using secp256k1
// elliptic curve and an error if it failed deriving or upon an incorrect
// signature.
//
// Sender may cache the address, allowing it to be used regardless of
// signing method. The cache is invalidated if the cached signer does
// not match the signer used in the current call.
func Sender(signer Signer, tx *Transaction) (common.Address, error) {
	if sc := tx.from.Load(); sc != nil {
		sigCache := sc.(sigCache)
		// If the signer used to derive from in a previous
		// call is not the same as used current, invalidate
		// the cache.
		if sigCache.signer.Equal(signer) {
			return sigCache.from, nil
		}
	}

	addr, err := signer.Sender(tx)
	if err != nil {
		return common.Address{}, err
	}
	tx.from.Store(sigCache{signer: signer, from: addr})
	return addr, nil
}

// Signer encapsulates transaction signature handling. The name of this type is slightly
// misleading because Signers don't actually sign, they're just for validating and
// processing of signatures.
//
// Note that this interface is not a stable API and may change at any time to accommodate
// new protocol rules.
type Signer interface {
	// Sender returns the sender address of the transaction.
	Sender(tx *Transaction) (common.Address, error)

	// SignatureAndPublicKeyValues returns the raw signature, publicKey values corresponding to the
	// given signature.
	SignatureAndPublicKeyValues(tx *Transaction, sig, pk []byte) (signature, publicKey []byte, err error)
	ChainID() *big.Int

	// Hash returns 'signature hash', i.e. the transaction hash that is signed by the
	// private key. This hash does not uniquely identify the transaction.
	Hash(tx *Transaction) common.Hash

	// Equal returns true if the given signer is the same as the receiver.
	Equal(Signer) bool
}

type londonSigner struct{ eip2930Signer }

// NewLondonSigner returns a signer that accepts
// - EIP-1559 dynamic fee transactions
// - EIP-2930 access list transactions,
// - EIP-155 replay protected transactions, and
// - legacy Homestead transactions.
func NewLondonSigner(chainId *big.Int) Signer {
	return londonSigner{eip2930Signer{NewEIP155Signer(chainId)}}
}

func (s londonSigner) Sender(tx *Transaction) (common.Address, error) {
	if tx.Type() != DynamicFeeTxType {
		return s.eip2930Signer.Sender(tx)
	}
	if tx.ChainId().Cmp(s.chainId) != 0 {
		return common.Address{}, fmt.Errorf("%w: have %d want %d", ErrInvalidChainId, tx.ChainId(), s.chainId)
	}
	return pqcrypto.DilithiumPKToAddress(tx.RawPublicKeyValue()), nil
}

func (s londonSigner) Equal(s2 Signer) bool {
	x, ok := s2.(londonSigner)
	return ok && x.chainId.Cmp(s.chainId) == 0
}

func (s londonSigner) SignatureAndPublicKeyValues(tx *Transaction, sig, pk []byte) (Signature, PublicKey []byte, err error) {
	txdata, ok := tx.inner.(*DynamicFeeTx)
	if !ok {
		return s.eip2930Signer.SignatureAndPublicKeyValues(tx, sig, pk)
	}
	// Check that chain ID of tx matches the signer. We also accept ID zero here,
	// because it indicates that the chain ID was not specified in the tx.
	if txdata.ChainID.Sign() != 0 && txdata.ChainID.Cmp(s.chainId) != 0 {
		return nil, nil, fmt.Errorf("%w: have %d want %d", ErrInvalidChainId, txdata.ChainID, s.chainId)
	}
	Signature = decodeSignature(sig)
	PublicKey = decodePublicKey(pk)
	return Signature, PublicKey, nil
}

// Hash returns the hash to be signed by the sender.
// It does not uniquely identify the transaction.
func (s londonSigner) Hash(tx *Transaction) common.Hash {
	if tx.Type() != DynamicFeeTxType {
		return s.eip2930Signer.Hash(tx)
	}
	return prefixedRlpHash(
		tx.Type(),
		[]interface{}{
			s.chainId,
			tx.Nonce(),
			tx.GasTipCap(),
			tx.GasFeeCap(),
			tx.Gas(),
			tx.To(),
			tx.Value(),
			tx.Data(),
			tx.AccessList(),
		})
}

type eip2930Signer struct{ EIP155Signer }

// NewEIP2930Signer returns a signer that accepts EIP-2930 access list transactions,
// EIP-155 replay protected transactions, and legacy Homestead transactions.
func NewEIP2930Signer(chainId *big.Int) Signer {
	return eip2930Signer{NewEIP155Signer(chainId)}
}

func (s eip2930Signer) ChainID() *big.Int {
	return s.chainId
}

func (s eip2930Signer) Equal(s2 Signer) bool {
	x, ok := s2.(eip2930Signer)
	return ok && x.chainId.Cmp(s.chainId) == 0
}

func (s eip2930Signer) Sender(tx *Transaction) (common.Address, error) {
	if tx.ChainId().Cmp(s.chainId) != 0 {
		return common.Address{}, fmt.Errorf("%w: have %d want %d", ErrInvalidChainId, tx.ChainId(), s.chainId)
	}
	return pqcrypto.DilithiumPKToAddress(tx.RawPublicKeyValue()), nil
}

func (s eip2930Signer) SignatureAndPublicKeyValues(tx *Transaction, sig, pk []byte) (Signature, PublicKey []byte, err error) {
	switch txdata := tx.inner.(type) {
	case *LegacyTx:
		return s.EIP155Signer.SignatureAndPublicKeyValues(tx, sig, pk)
	case *AccessListTx:
		// Check that chain ID of tx matches the signer. We also accept ID zero here,
		// because it indicates that the chain ID was not specified in the tx.
		if txdata.ChainID.Sign() != 0 && txdata.ChainID.Cmp(s.chainId) != 0 {
			return nil, nil, fmt.Errorf("%w: have %d want %d", ErrInvalidChainId, txdata.ChainID, s.chainId)
		}
		Signature = decodeSignature(sig)
		PublicKey = decodePublicKey(pk)
	default:
		return nil, nil, ErrTxTypeNotSupported
	}
	return Signature, PublicKey, nil
}

// Hash returns the hash to be signed by the sender.
// It does not uniquely identify the transaction.
func (s eip2930Signer) Hash(tx *Transaction) common.Hash {
	switch tx.Type() {
	case LegacyTxType:
		return s.EIP155Signer.Hash(tx)
	case AccessListTxType:
		return prefixedRlpHash(
			tx.Type(),
			[]interface{}{
				s.chainId,
				tx.Nonce(),
				tx.GasPrice(),
				tx.Gas(),
				tx.To(),
				tx.Value(),
				tx.Data(),
				tx.AccessList(),
			})
	default:
		// This _should_ not happen, but in case someone sends in a bad
		// json struct via RPC, it's probably more prudent to return an
		// empty hash instead of killing the node with a panic
		//panic("Unsupported transaction type: %d", tx.typ)
		return common.Hash{}
	}
}

// EIP155Signer implements Signer using the EIP-155 rules. This accepts transactions which
// are replay-protected as well as unprotected homestead transactions.
type EIP155Signer struct {
	chainId, chainIdMul *big.Int
}

func NewEIP155Signer(chainId *big.Int) EIP155Signer {
	if chainId == nil {
		chainId = new(big.Int)
	}
	return EIP155Signer{
		chainId:    chainId,
		chainIdMul: new(big.Int).Mul(chainId, big.NewInt(2)),
	}
}

func (s EIP155Signer) ChainID() *big.Int {
	return s.chainId
}

func (s EIP155Signer) Equal(s2 Signer) bool {
	eip155, ok := s2.(EIP155Signer)
	return ok && eip155.chainId.Cmp(s.chainId) == 0
}

func (s EIP155Signer) Sender(tx *Transaction) (common.Address, error) {
	if tx.Type() != LegacyTxType {
		return common.Address{}, ErrTxTypeNotSupported
	}
	if tx.ChainId().Cmp(s.chainId) != 0 {
		return common.Address{}, fmt.Errorf("%w: have %d want %d", ErrInvalidChainId, tx.ChainId(), s.chainId)
	}
	return pqcrypto.DilithiumPKToAddress(tx.RawPublicKeyValue()), nil
}

// SignatureAndPublicKeyValues returns signature values. This signature
// needs to be in the [R || S || V] format where V is 0 or 1.
func (s EIP155Signer) SignatureAndPublicKeyValues(tx *Transaction, sig, pk []byte) (Signature, PublicKey []byte, err error) {
	if tx.Type() != LegacyTxType {
		return nil, nil, ErrTxTypeNotSupported
	}
	Signature = decodeSignature(sig)
	PublicKey = decodePublicKey(pk)
	return Signature, PublicKey, nil
}

// Hash returns the hash to be signed by the sender.
// It does not uniquely identify the transaction.
func (s EIP155Signer) Hash(tx *Transaction) common.Hash {
	return rlpHash([]interface{}{
		tx.Nonce(),
		tx.GasPrice(),
		tx.Gas(),
		tx.To(),
		tx.Value(),
		tx.Data(),
		s.chainId, uint(0), uint(0),
	})
}

// HomesteadSigner implements Signer interface using the
// homestead rules.
type HomesteadSigner struct{ FrontierSigner }

func (s HomesteadSigner) ChainID() *big.Int {
	return nil
}

func (s HomesteadSigner) Equal(s2 Signer) bool {
	_, ok := s2.(HomesteadSigner)
	return ok
}

// SignatureAndPublicKeyValues returns signature values. This signature
// needs to be in the [R || S || V] format where V is 0 or 1.
func (hs HomesteadSigner) SignatureAndPublicKeyValues(tx *Transaction, sig, pk []byte) (signature, publicKey []byte, err error) {
	return hs.FrontierSigner.SignatureAndPublicKeyValues(tx, sig, pk)
}

func (hs HomesteadSigner) Sender(tx *Transaction) (common.Address, error) {
	if tx.Type() != LegacyTxType {
		return common.Address{}, ErrTxTypeNotSupported
	}
	return pqcrypto.DilithiumPKToAddress(tx.RawPublicKeyValue()), nil
}

// FrontierSigner implements Signer interface using the
// frontier rules.
type FrontierSigner struct{}

func (s FrontierSigner) ChainID() *big.Int {
	return nil
}

func (s FrontierSigner) Equal(s2 Signer) bool {
	_, ok := s2.(FrontierSigner)
	return ok
}

func (fs FrontierSigner) Sender(tx *Transaction) (common.Address, error) {
	if tx.Type() != LegacyTxType {
		return common.Address{}, ErrTxTypeNotSupported
	}
	return pqcrypto.DilithiumPKToAddress(tx.RawPublicKeyValue()), nil
}

// SignatureAndPublicKeyValues returns signature values. This signature
// needs to be in the [R || S || V] format where V is 0 or 1.
func (fs FrontierSigner) SignatureAndPublicKeyValues(tx *Transaction, sig, pk []byte) (signature, publicKey []byte, err error) {
	if tx.Type() != LegacyTxType {
		return nil, nil, ErrTxTypeNotSupported
	}
	signature = decodeSignature(sig)
	publicKey = decodePublicKey(pk)
	return signature, publicKey, nil
}

// Hash returns the hash to be signed by the sender.
// It does not uniquely identify the transaction.
func (fs FrontierSigner) Hash(tx *Transaction) common.Hash {
	return rlpHash([]interface{}{
		tx.Nonce(),
		tx.GasPrice(),
		tx.Gas(),
		tx.To(),
		tx.Value(),
		tx.Data(),
	})
}

func decodeSignature(sig []byte) (signature []byte) {
	if len(sig) != pqcrypto.DilithiumSignatureLength {
		panic(fmt.Sprintf("wrong size for signature: got %d, want %d", len(sig), pqcrypto.DilithiumSignatureLength))
	}
	signature = make([]byte, pqcrypto.DilithiumSignatureLength)
	copy(signature, sig)
	return signature
}

func decodePublicKey(pk []byte) (publicKey []byte) {
	if len(pk) != pqcrypto.DilithiumPublicKeyLength {
		panic(fmt.Sprintf("wrong size for dilithium publickey: got %d, want %d", len(pk), pqcrypto.DilithiumPublicKeyLength))
	}
	publicKey = make([]byte, pqcrypto.DilithiumPublicKeyLength)
	copy(publicKey, pk)
	return publicKey
}

// deriveChainId derives the chain id from the given v parameter
func deriveChainId(v *big.Int) *big.Int {
	if v.BitLen() <= 64 {
		v := v.Uint64()
		if v == 27 || v == 28 {
			return new(big.Int)
		}
		return new(big.Int).SetUint64((v - 35) / 2)
	}
	v = new(big.Int).Sub(v, big.NewInt(35))
	return v.Div(v, big.NewInt(2))
}
