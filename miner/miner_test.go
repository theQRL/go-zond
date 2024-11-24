// Copyright 2020 The go-ethereum Authors
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

// Package miner implements Zond block creation.
package miner

import (
	"math/big"
	"sync"
	"testing"

	"github.com/theQRL/go-zond/common"
	"github.com/theQRL/go-zond/consensus/beacon"
	"github.com/theQRL/go-zond/core"
	"github.com/theQRL/go-zond/core/rawdb"
	"github.com/theQRL/go-zond/core/state"
	"github.com/theQRL/go-zond/core/txpool"
	"github.com/theQRL/go-zond/core/txpool/legacypool"
	"github.com/theQRL/go-zond/core/types"
	"github.com/theQRL/go-zond/core/vm"
	"github.com/theQRL/go-zond/crypto"
	"github.com/theQRL/go-zond/event"
	"github.com/theQRL/go-zond/params"
	"github.com/theQRL/go-zond/trie"
)

type mockBackend struct {
	bc     *core.BlockChain
	txPool *txpool.TxPool
}

func NewMockBackend(bc *core.BlockChain, txPool *txpool.TxPool) *mockBackend {
	return &mockBackend{
		bc:     bc,
		txPool: txPool,
	}
}

func (m *mockBackend) BlockChain() *core.BlockChain {
	return m.bc
}

func (m *mockBackend) TxPool() *txpool.TxPool {
	return m.txPool
}

type testBlockChain struct {
	root          common.Hash
	config        *params.ChainConfig
	statedb       *state.StateDB
	gasLimit      uint64
	chainHeadFeed *event.Feed
}

func (bc *testBlockChain) Config() *params.ChainConfig {
	return bc.config
}

func (bc *testBlockChain) CurrentBlock() *types.Header {
	return &types.Header{
		Number:   new(big.Int),
		GasLimit: bc.gasLimit,
	}
}

func (bc *testBlockChain) GetBlock(hash common.Hash, number uint64) *types.Block {
	return types.NewBlock(bc.CurrentBlock(), nil, nil, trie.NewStackTrie(nil))
}

func (bc *testBlockChain) StateAt(common.Hash) (*state.StateDB, error) {
	return bc.statedb, nil
}

func (bc *testBlockChain) HasState(root common.Hash) bool {
	return bc.root == root
}

func (bc *testBlockChain) SubscribeChainHeadEvent(ch chan<- core.ChainHeadEvent) event.Subscription {
	return bc.chainHeadFeed.Subscribe(ch)
}

func TestBuildPendingBlocks(t *testing.T) {
	miner := createMiner(t)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		block, _, _ := miner.Pending()
		if block == nil {
			t.Error("Pending failed")
		}
	}()
	wg.Wait()
}

func minerTestGenesisBlock(gasLimit uint64, faucet common.Address) *core.Genesis {
	config := *params.AllBeaconProtocolChanges

	// Assemble and return the genesis with the precompiles and faucet pre-funded
	return &core.Genesis{
		Config:    &config,
		ExtraData: append(append(make([]byte, 32), faucet[:]...), make([]byte, crypto.SignatureLength)...),
		GasLimit:  gasLimit,
		BaseFee:   big.NewInt(params.InitialBaseFee),
		Alloc: map[common.Address]core.GenesisAccount{
			common.BytesToAddress([]byte{1}): {Balance: big.NewInt(1)}, // Deposit
			common.BytesToAddress([]byte{2}): {Balance: big.NewInt(1)}, // SHA256
			common.BytesToAddress([]byte{3}): {Balance: big.NewInt(1)}, // RIPEMD
			common.BytesToAddress([]byte{4}): {Balance: big.NewInt(1)}, // Identity
			common.BytesToAddress([]byte{5}): {Balance: big.NewInt(1)}, // ModExp
			common.BytesToAddress([]byte{6}): {Balance: big.NewInt(1)}, // ECAdd
			common.BytesToAddress([]byte{7}): {Balance: big.NewInt(1)}, // ECScalarMul
			common.BytesToAddress([]byte{8}): {Balance: big.NewInt(1)}, // ECPairing
			common.BytesToAddress([]byte{9}): {Balance: big.NewInt(1)}, // BLAKE2b
			faucet:                           {Balance: new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(9))},
		},
	}
}

func createMiner(t *testing.T) *Miner {
	// Create Ethash config
	feeRecipient, _ := common.NewAddressFromString("Z0000000000000000000000000000000123456789")
	config := Config{
		PendingFeeRecipient: feeRecipient,
	}
	// Create chainConfig
	chainDB := rawdb.NewMemoryDatabase()
	triedb := trie.NewDatabase(chainDB, nil)
	faucet, _ := common.NewAddressFromString("Z0000000000000000000000000000000000012345")
	genesis := minerTestGenesisBlock(11_500_000, faucet)
	chainConfig, _, err := core.SetupGenesisBlock(chainDB, triedb, genesis)
	if err != nil {
		t.Fatalf("can't create new chain config: %v", err)
	}
	// Create consensus engine
	engine := beacon.New()
	// Create Zond backend
	bc, err := core.NewBlockChain(chainDB, nil, genesis, engine, vm.Config{}, nil)
	if err != nil {
		t.Fatalf("can't create new chain %v", err)
	}
	statedb, _ := state.New(bc.Genesis().Root(), bc.StateCache(), nil)
	blockchain := &testBlockChain{bc.Genesis().Root(), chainConfig, statedb, 10000000, new(event.Feed)}

	pool := legacypool.New(testTxPoolConfig, blockchain)
	txpool, _ := txpool.New(new(big.Int).SetUint64(testTxPoolConfig.PriceLimit), blockchain, []txpool.SubPool{pool})

	// Create Miner
	backend := NewMockBackend(bc, txpool)
	miner := New(backend, config, engine)
	return miner
}
