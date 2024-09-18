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
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package zondapi

import (
	"context"
	"errors"
	"math/big"
	"reflect"
	"testing"
	"time"

	"github.com/theQRL/go-zond"
	"github.com/theQRL/go-zond/accounts"
	"github.com/theQRL/go-zond/common"
	"github.com/theQRL/go-zond/common/hexutil"
	"github.com/theQRL/go-zond/consensus"
	"github.com/theQRL/go-zond/core"
	"github.com/theQRL/go-zond/core/bloombits"
	"github.com/theQRL/go-zond/core/state"
	"github.com/theQRL/go-zond/core/types"
	"github.com/theQRL/go-zond/core/vm"
	"github.com/theQRL/go-zond/event"
	"github.com/theQRL/go-zond/params"
	"github.com/theQRL/go-zond/rpc"
	"github.com/theQRL/go-zond/zonddb"
)

// TestSetFeeDefaults tests the logic for filling in default fee values works as expected.
func TestSetFeeDefaults(t *testing.T) {
	type test struct {
		name string
		in   *TransactionArgs
		want *TransactionArgs
		err  error
	}

	var (
		b        = newBackendMock()
		fortytwo = (*hexutil.Big)(big.NewInt(42))
		maxFee   = (*hexutil.Big)(new(big.Int).Add(new(big.Int).Mul(b.current.BaseFee, big.NewInt(2)), fortytwo.ToInt()))
		al       = &types.AccessList{types.AccessTuple{Address: common.Address{0xaa}, StorageKeys: []common.Hash{{0x01}}}}
	)

	tests := []test{
		{
			"dynamic tx",
			&TransactionArgs{AccessList: al},
			&TransactionArgs{AccessList: al, MaxFeePerGas: maxFee, MaxPriorityFeePerGas: fortytwo},
			nil,
		},
		{
			"dynamic tx, only max fee",
			&TransactionArgs{MaxFeePerGas: maxFee},
			&TransactionArgs{MaxFeePerGas: maxFee, MaxPriorityFeePerGas: fortytwo},
			nil,
		},
		{
			"dynamic tx, only priority fee",
			&TransactionArgs{MaxFeePerGas: maxFee},
			&TransactionArgs{MaxFeePerGas: maxFee, MaxPriorityFeePerGas: fortytwo},
			nil,
		},
		{
			"dynamic fee tx, maxFee < priorityFee",
			&TransactionArgs{MaxFeePerGas: maxFee, MaxPriorityFeePerGas: (*hexutil.Big)(big.NewInt(1000))},
			nil,
			errors.New("maxFeePerGas (0x3e) < maxPriorityFeePerGas (0x3e8)"),
		},
		{
			"dynamic fee tx, maxFee < priorityFee while setting default",
			&TransactionArgs{MaxFeePerGas: (*hexutil.Big)(big.NewInt(7))},
			nil,
			errors.New("maxFeePerGas (0x7) < maxPriorityFeePerGas (0x2a)"),
		},
	}

	ctx := context.Background()
	for i, test := range tests {
		got := test.in
		err := got.setFeeDefaults(ctx, b)
		if err != nil && err.Error() == test.err.Error() {
			// Test threw expected error.
			continue
		} else if err != nil {
			t.Fatalf("test %d (%s): unexpected error: %s", i, test.name, err)
		}
		if !reflect.DeepEqual(got, test.want) {
			t.Fatalf("test %d (%s): did not fill defaults as expected: (got: %v, want: %v)", i, test.name, got, test.want)
		}
	}
}

type backendMock struct {
	current *types.Header
	config  *params.ChainConfig
}

func newBackendMock() *backendMock {
	config := &params.ChainConfig{
		ChainID: big.NewInt(42),
	}
	return &backendMock{
		current: &types.Header{
			Number:   big.NewInt(1100),
			GasLimit: 8_000_000,
			GasUsed:  8_000_000,
			Time:     555,
			Extra:    make([]byte, 32),
			BaseFee:  big.NewInt(10),
		},
		config: config,
	}
}

func (b *backendMock) SuggestGasTipCap(ctx context.Context) (*big.Int, error) {
	return big.NewInt(42), nil
}
func (b *backendMock) CurrentHeader() *types.Header     { return b.current }
func (b *backendMock) ChainConfig() *params.ChainConfig { return b.config }

// Other methods needed to implement Backend interface.
func (b *backendMock) SyncProgress() zond.SyncProgress { return zond.SyncProgress{} }
func (b *backendMock) FeeHistory(ctx context.Context, blockCount uint64, lastBlock rpc.BlockNumber, rewardPercentiles []float64) (*big.Int, [][]*big.Int, []*big.Int, []float64, error) {
	return nil, nil, nil, nil, nil
}
func (b *backendMock) ChainDb() zonddb.Database          { return nil }
func (b *backendMock) AccountManager() *accounts.Manager { return nil }
func (b *backendMock) ExtRPCEnabled() bool               { return false }
func (b *backendMock) RPCGasCap() uint64                 { return 0 }
func (b *backendMock) RPCEVMTimeout() time.Duration      { return time.Second }
func (b *backendMock) RPCTxFeeCap() float64              { return 0 }
func (b *backendMock) SetHead(number uint64)             {}
func (b *backendMock) HeaderByNumber(ctx context.Context, number rpc.BlockNumber) (*types.Header, error) {
	return nil, nil
}
func (b *backendMock) HeaderByHash(ctx context.Context, hash common.Hash) (*types.Header, error) {
	return nil, nil
}
func (b *backendMock) HeaderByNumberOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (*types.Header, error) {
	return nil, nil
}
func (b *backendMock) CurrentBlock() *types.Header { return nil }
func (b *backendMock) BlockByNumber(ctx context.Context, number rpc.BlockNumber) (*types.Block, error) {
	return nil, nil
}
func (b *backendMock) BlockByHash(ctx context.Context, hash common.Hash) (*types.Block, error) {
	return nil, nil
}
func (b *backendMock) BlockByNumberOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (*types.Block, error) {
	return nil, nil
}
func (b *backendMock) GetBody(ctx context.Context, hash common.Hash, number rpc.BlockNumber) (*types.Body, error) {
	return nil, nil
}
func (b *backendMock) StateAndHeaderByNumber(ctx context.Context, number rpc.BlockNumber) (*state.StateDB, *types.Header, error) {
	return nil, nil, nil
}
func (b *backendMock) StateAndHeaderByNumberOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (*state.StateDB, *types.Header, error) {
	return nil, nil, nil
}
func (b *backendMock) Pending() (*types.Block, types.Receipts, *state.StateDB) { return nil, nil, nil }
func (b *backendMock) GetReceipts(ctx context.Context, hash common.Hash) (types.Receipts, error) {
	return nil, nil
}
func (b *backendMock) GetLogs(ctx context.Context, blockHash common.Hash, number uint64) ([][]*types.Log, error) {
	return nil, nil
}
func (b *backendMock) GetEVM(ctx context.Context, msg *core.Message, state *state.StateDB, header *types.Header, vmConfig *vm.Config, blockCtx *vm.BlockContext) *vm.EVM {
	return nil
}
func (b *backendMock) SubscribeChainEvent(ch chan<- core.ChainEvent) event.Subscription { return nil }
func (b *backendMock) SubscribeChainHeadEvent(ch chan<- core.ChainHeadEvent) event.Subscription {
	return nil
}
func (b *backendMock) SubscribeChainSideEvent(ch chan<- core.ChainSideEvent) event.Subscription {
	return nil
}
func (b *backendMock) SendTx(ctx context.Context, signedTx *types.Transaction) error { return nil }
func (b *backendMock) GetTransaction(ctx context.Context, txHash common.Hash) (*types.Transaction, common.Hash, uint64, uint64, error) {
	return nil, [32]byte{}, 0, 0, nil
}
func (b *backendMock) GetPoolTransactions() (types.Transactions, error)         { return nil, nil }
func (b *backendMock) GetPoolTransaction(txHash common.Hash) *types.Transaction { return nil }
func (b *backendMock) GetPoolNonce(ctx context.Context, addr common.Address) (uint64, error) {
	return 0, nil
}
func (b *backendMock) Stats() (pending int, queued int) { return 0, 0 }
func (b *backendMock) TxPoolContent() (map[common.Address][]*types.Transaction, map[common.Address][]*types.Transaction) {
	return nil, nil
}
func (b *backendMock) TxPoolContentFrom(addr common.Address) ([]*types.Transaction, []*types.Transaction) {
	return nil, nil
}
func (b *backendMock) SubscribeNewTxsEvent(chan<- core.NewTxsEvent) event.Subscription      { return nil }
func (b *backendMock) BloomStatus() (uint64, uint64)                                        { return 0, 0 }
func (b *backendMock) ServiceFilter(ctx context.Context, session *bloombits.MatcherSession) {}
func (b *backendMock) SubscribeLogsEvent(ch chan<- []*types.Log) event.Subscription         { return nil }
func (b *backendMock) SubscribeRemovedLogsEvent(ch chan<- core.RemovedLogsEvent) event.Subscription {
	return nil
}

func (b *backendMock) Engine() consensus.Engine { return nil }
