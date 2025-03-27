// Copyright 2022 The go-ethereum Authors
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
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>

package miner

import (
	"math/big"
	"reflect"
	"testing"
	"time"

	"github.com/theQRL/go-zond/beacon/engine"
	"github.com/theQRL/go-zond/common"
	"github.com/theQRL/go-zond/consensus"
	"github.com/theQRL/go-zond/consensus/beacon"
	"github.com/theQRL/go-zond/core"
	"github.com/theQRL/go-zond/core/rawdb"
	"github.com/theQRL/go-zond/core/txpool"
	"github.com/theQRL/go-zond/core/txpool/legacypool"
	"github.com/theQRL/go-zond/core/types"
	"github.com/theQRL/go-zond/core/vm"
	"github.com/theQRL/go-zond/crypto"
	"github.com/theQRL/go-zond/params"
	"github.com/theQRL/go-zond/zonddb"
)

var (
	// Test chain configurations
	testTxPoolConfig  legacypool.Config
	beaconChainConfig *params.ChainConfig

	// Test accounts
	testBankKey, _  = crypto.GenerateDilithiumKey()
	testBankAddress = testBankKey.GetAddress()
	testBankFunds   = big.NewInt(1000000000000000000)

	testUserKey, _  = crypto.GenerateKey()
	testUserAddress = crypto.PubkeyToAddress(testUserKey.PublicKey)

	// Test transactions
	pendingTxs []*types.Transaction
	newTxs     []*types.Transaction

	testConfig = Config{
		PendingFeeRecipient: testBankAddress,
		Recommit:            time.Second,
		GasCeil:             params.GenesisGasLimit,
	}
)

func init() {
	testTxPoolConfig = legacypool.DefaultConfig
	testTxPoolConfig.Journal = ""
	beaconChainConfig = new(params.ChainConfig)
	*beaconChainConfig = *params.TestChainConfig

	signer := types.LatestSigner(params.TestChainConfig)
	tx1 := types.MustSignNewTx(testBankKey, signer, &types.DynamicFeeTx{
		ChainID:   params.TestChainConfig.ChainID,
		Nonce:     0,
		To:        &testUserAddress,
		Value:     big.NewInt(1000),
		Gas:       params.TxGas,
		GasFeeCap: big.NewInt(params.InitialBaseFee),
	})
	pendingTxs = append(pendingTxs, tx1)

	tx2 := types.MustSignNewTx(testBankKey, signer, &types.DynamicFeeTx{
		Nonce:     1,
		To:        &testUserAddress,
		Value:     big.NewInt(1000),
		Gas:       params.TxGas,
		GasFeeCap: big.NewInt(params.InitialBaseFee),
	})
	newTxs = append(newTxs, tx2)
}

// testWorkerBackend implements worker.Backend interfaces and wraps all information needed during the testing.
type testWorkerBackend struct {
	db      zonddb.Database
	txPool  *txpool.TxPool
	chain   *core.BlockChain
	genesis *core.Genesis
}

func newTestWorkerBackend(t *testing.T, chainConfig *params.ChainConfig, engine consensus.Engine, db zonddb.Database, n int) *testWorkerBackend {
	var gspec = &core.Genesis{
		Config: chainConfig,
		Alloc:  core.GenesisAlloc{testBankAddress: {Balance: testBankFunds}},
	}
	switch engine.(type) {
	case *beacon.Beacon:
	default:
		t.Fatalf("unexpected consensus engine type: %T", engine)
	}
	chain, err := core.NewBlockChain(db, &core.CacheConfig{TrieDirtyDisabled: true}, gspec, engine, vm.Config{}, nil)
	if err != nil {
		t.Fatalf("core.NewBlockChain failed: %v", err)
	}
	pool := legacypool.New(testTxPoolConfig, chain)
	txpool, _ := txpool.New(new(big.Int).SetUint64(testTxPoolConfig.PriceLimit), chain, []txpool.SubPool{pool})

	return &testWorkerBackend{
		db:      db,
		chain:   chain,
		txPool:  txpool,
		genesis: gspec,
	}
}

func (b *testWorkerBackend) BlockChain() *core.BlockChain { return b.chain }
func (b *testWorkerBackend) TxPool() *txpool.TxPool       { return b.txPool }

func newTestWorker(t *testing.T, chainConfig *params.ChainConfig, engine consensus.Engine, db zonddb.Database, blocks int) (*Miner, *testWorkerBackend) {
	backend := newTestWorkerBackend(t, chainConfig, engine, db, blocks)
	backend.txPool.Add(pendingTxs, true, true)
	w := New(backend, testConfig, engine)
	return w, backend
}

func TestBuildPayload(t *testing.T) {
	var (
		db           = rawdb.NewMemoryDatabase()
		recipient, _ = common.NewAddressFromString("Z00000000000000000000000000000000deadbeef")
	)
	w, b := newTestWorker(t, params.TestChainConfig, beacon.NewFaker(), db, 0)

	timestamp := uint64(time.Now().Unix())
	args := &BuildPayloadArgs{
		Parent:       b.chain.CurrentBlock().Hash(),
		Timestamp:    timestamp,
		Random:       common.Hash{},
		FeeRecipient: recipient,
	}
	payload, err := w.buildPayload(args)
	if err != nil {
		t.Fatalf("Failed to build payload %v", err)
	}
	verify := func(outer *engine.ExecutionPayloadEnvelope, txs int) {
		payload := outer.ExecutionPayload
		if payload.ParentHash != b.chain.CurrentBlock().Hash() {
			t.Fatal("Unexpected parent hash")
		}
		if payload.Random != (common.Hash{}) {
			t.Fatal("Unexpected random value")
		}
		if payload.Timestamp != timestamp {
			t.Fatal("Unexpected timestamp")
		}
		if payload.FeeRecipient != recipient {
			t.Fatal("Unexpected fee recipient")
		}
		if len(payload.Transactions) != txs {
			t.Fatal("Unexpected transaction set")
		}
	}
	empty := payload.ResolveEmpty()
	verify(empty, 0)

	full := payload.ResolveFull()
	verify(full, len(pendingTxs))

	// Ensure resolve can be called multiple times and the
	// result should be unchanged
	dataOne := payload.Resolve()
	dataTwo := payload.Resolve()
	if !reflect.DeepEqual(dataOne, dataTwo) {
		t.Fatal("Unexpected payload data")
	}
}

func TestPayloadId(t *testing.T) {
	t.Parallel()
	ids := make(map[string]int)
	for i, tt := range []*BuildPayloadArgs{
		{
			Parent:       common.Hash{1},
			Timestamp:    1,
			Random:       common.Hash{0x1},
			FeeRecipient: common.Address{0x1},
		},
		// Different parent
		{
			Parent:       common.Hash{2},
			Timestamp:    1,
			Random:       common.Hash{0x1},
			FeeRecipient: common.Address{0x1},
		},
		// Different timestamp
		{
			Parent:       common.Hash{2},
			Timestamp:    2,
			Random:       common.Hash{0x1},
			FeeRecipient: common.Address{0x1},
		},
		// Different Random
		{
			Parent:       common.Hash{2},
			Timestamp:    2,
			Random:       common.Hash{0x2},
			FeeRecipient: common.Address{0x1},
		},
		// Different fee-recipient
		{
			Parent:       common.Hash{2},
			Timestamp:    2,
			Random:       common.Hash{0x2},
			FeeRecipient: common.Address{0x2},
		},
		// Different withdrawals (non-empty)
		{
			Parent:       common.Hash{2},
			Timestamp:    2,
			Random:       common.Hash{0x2},
			FeeRecipient: common.Address{0x2},
			Withdrawals: []*types.Withdrawal{
				{
					Index:     0,
					Validator: 0,
					Address:   common.Address{},
					Amount:    0,
				},
			},
		},
		// Different withdrawals (non-empty)
		{
			Parent:       common.Hash{2},
			Timestamp:    2,
			Random:       common.Hash{0x2},
			FeeRecipient: common.Address{0x2},
			Withdrawals: []*types.Withdrawal{
				{
					Index:     2,
					Validator: 0,
					Address:   common.Address{},
					Amount:    0,
				},
			},
		},
	} {
		id := tt.Id().String()
		if prev, exists := ids[id]; exists {
			t.Errorf("ID collision, case %d and case %d: id %v", prev, i, id)
		}
		ids[id] = i
	}
}
