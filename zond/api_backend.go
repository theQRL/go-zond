// Copyright 2015 The go-ethereum Authors
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

package zond

import (
	"context"
	"errors"
	"math/big"
	"time"

	"github.com/theQRL/go-zond"
	"github.com/theQRL/go-zond/accounts"
	"github.com/theQRL/go-zond/common"
	"github.com/theQRL/go-zond/consensus"
	"github.com/theQRL/go-zond/core"
	"github.com/theQRL/go-zond/core/bloombits"
	"github.com/theQRL/go-zond/core/rawdb"
	"github.com/theQRL/go-zond/core/state"
	"github.com/theQRL/go-zond/core/txpool"
	"github.com/theQRL/go-zond/core/types"
	"github.com/theQRL/go-zond/core/vm"
	"github.com/theQRL/go-zond/event"
	"github.com/theQRL/go-zond/params"
	"github.com/theQRL/go-zond/rpc"
	"github.com/theQRL/go-zond/zond/gasprice"
	"github.com/theQRL/go-zond/zond/tracers"
	"github.com/theQRL/go-zond/zonddb"
)

// ZondAPIBackend implements zondapi.Backend for full nodes
type ZondAPIBackend struct {
	extRPCEnabled bool
	zond          *Zond
	gpo           *gasprice.Oracle
}

// ChainConfig returns the active chain configuration.
func (b *ZondAPIBackend) ChainConfig() *params.ChainConfig {
	return b.zond.blockchain.Config()
}

func (b *ZondAPIBackend) CurrentBlock() *types.Header {
	return b.zond.blockchain.CurrentBlock()
}

func (b *ZondAPIBackend) SetHead(number uint64) {
	b.zond.handler.downloader.Cancel()
	b.zond.blockchain.SetHead(number)
}

func (b *ZondAPIBackend) HeaderByNumber(ctx context.Context, number rpc.BlockNumber) (*types.Header, error) {
	// Pending block is only known by the miner
	if number == rpc.PendingBlockNumber {
		block, _, _ := b.zond.miner.Pending()
		if block == nil {
			return nil, errors.New("pending block is not available")
		}
		return block.Header(), nil
	}
	// Otherwise resolve and return the block
	if number == rpc.LatestBlockNumber {
		return b.zond.blockchain.CurrentBlock(), nil
	}
	if number == rpc.FinalizedBlockNumber {
		block := b.zond.blockchain.CurrentFinalBlock()
		if block == nil {
			return nil, errors.New("finalized block not found")
		}
		return block, nil
	}
	if number == rpc.SafeBlockNumber {
		block := b.zond.blockchain.CurrentSafeBlock()
		if block == nil {
			return nil, errors.New("safe block not found")
		}
		return block, nil
	}
	return b.zond.blockchain.GetHeaderByNumber(uint64(number)), nil
}

func (b *ZondAPIBackend) HeaderByNumberOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (*types.Header, error) {
	if blockNr, ok := blockNrOrHash.Number(); ok {
		return b.HeaderByNumber(ctx, blockNr)
	}
	if hash, ok := blockNrOrHash.Hash(); ok {
		header := b.zond.blockchain.GetHeaderByHash(hash)
		if header == nil {
			return nil, errors.New("header for hash not found")
		}
		if blockNrOrHash.RequireCanonical && b.zond.blockchain.GetCanonicalHash(header.Number.Uint64()) != hash {
			return nil, errors.New("hash is not currently canonical")
		}
		return header, nil
	}
	return nil, errors.New("invalid arguments; neither block nor hash specified")
}

func (b *ZondAPIBackend) HeaderByHash(ctx context.Context, hash common.Hash) (*types.Header, error) {
	return b.zond.blockchain.GetHeaderByHash(hash), nil
}

func (b *ZondAPIBackend) BlockByNumber(ctx context.Context, number rpc.BlockNumber) (*types.Block, error) {
	// Pending block is only known by the miner
	if number == rpc.PendingBlockNumber {
		block, _, _ := b.zond.miner.Pending()
		if block == nil {
			return nil, errors.New("pending block is not available")
		}
		return block, nil
	}
	// Otherwise resolve and return the block
	if number == rpc.LatestBlockNumber {
		header := b.zond.blockchain.CurrentBlock()
		return b.zond.blockchain.GetBlock(header.Hash(), header.Number.Uint64()), nil
	}
	if number == rpc.FinalizedBlockNumber {
		header := b.zond.blockchain.CurrentFinalBlock()
		if header == nil {
			return nil, errors.New("finalized block not found")
		}
		return b.zond.blockchain.GetBlock(header.Hash(), header.Number.Uint64()), nil
	}
	if number == rpc.SafeBlockNumber {
		header := b.zond.blockchain.CurrentSafeBlock()
		if header == nil {
			return nil, errors.New("safe block not found")
		}
		return b.zond.blockchain.GetBlock(header.Hash(), header.Number.Uint64()), nil
	}
	return b.zond.blockchain.GetBlockByNumber(uint64(number)), nil
}

func (b *ZondAPIBackend) BlockByHash(ctx context.Context, hash common.Hash) (*types.Block, error) {
	return b.zond.blockchain.GetBlockByHash(hash), nil
}

// GetBody returns body of a block. It does not resolve special block numbers.
func (b *ZondAPIBackend) GetBody(ctx context.Context, hash common.Hash, number rpc.BlockNumber) (*types.Body, error) {
	if number < 0 || hash == (common.Hash{}) {
		return nil, errors.New("invalid arguments; expect hash and no special block numbers")
	}
	if body := b.zond.blockchain.GetBody(hash); body != nil {
		return body, nil
	}
	return nil, errors.New("block body not found")
}

func (b *ZondAPIBackend) BlockByNumberOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (*types.Block, error) {
	if blockNr, ok := blockNrOrHash.Number(); ok {
		return b.BlockByNumber(ctx, blockNr)
	}
	if hash, ok := blockNrOrHash.Hash(); ok {
		header := b.zond.blockchain.GetHeaderByHash(hash)
		if header == nil {
			return nil, errors.New("header for hash not found")
		}
		if blockNrOrHash.RequireCanonical && b.zond.blockchain.GetCanonicalHash(header.Number.Uint64()) != hash {
			return nil, errors.New("hash is not currently canonical")
		}
		block := b.zond.blockchain.GetBlock(hash, header.Number.Uint64())
		if block == nil {
			return nil, errors.New("header found, but block body is missing")
		}
		return block, nil
	}
	return nil, errors.New("invalid arguments; neither block nor hash specified")
}

func (b *ZondAPIBackend) Pending() (*types.Block, types.Receipts, *state.StateDB) {
	return b.zond.miner.Pending()
}

func (b *ZondAPIBackend) StateAndHeaderByNumber(ctx context.Context, number rpc.BlockNumber) (*state.StateDB, *types.Header, error) {
	// Pending state is only known by the miner
	if number == rpc.PendingBlockNumber {
		block, _, state := b.zond.miner.Pending()
		if block == nil || state == nil {
			return nil, nil, errors.New("pending state is not available")
		}
		return state, block.Header(), nil
	}
	// Otherwise resolve the block number and return its state
	header, err := b.HeaderByNumber(ctx, number)
	if err != nil {
		return nil, nil, err
	}
	if header == nil {
		return nil, nil, errors.New("header not found")
	}
	stateDb, err := b.zond.BlockChain().StateAt(header.Root)
	if err != nil {
		return nil, nil, err
	}
	return stateDb, header, nil
}

func (b *ZondAPIBackend) StateAndHeaderByNumberOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (*state.StateDB, *types.Header, error) {
	if blockNr, ok := blockNrOrHash.Number(); ok {
		return b.StateAndHeaderByNumber(ctx, blockNr)
	}
	if hash, ok := blockNrOrHash.Hash(); ok {
		header, err := b.HeaderByHash(ctx, hash)
		if err != nil {
			return nil, nil, err
		}
		if header == nil {
			return nil, nil, errors.New("header for hash not found")
		}
		if blockNrOrHash.RequireCanonical && b.zond.blockchain.GetCanonicalHash(header.Number.Uint64()) != hash {
			return nil, nil, errors.New("hash is not currently canonical")
		}
		stateDb, err := b.zond.BlockChain().StateAt(header.Root)
		if err != nil {
			return nil, nil, err
		}
		return stateDb, header, nil
	}
	return nil, nil, errors.New("invalid arguments; neither block nor hash specified")
}

func (b *ZondAPIBackend) GetReceipts(ctx context.Context, hash common.Hash) (types.Receipts, error) {
	return b.zond.blockchain.GetReceiptsByHash(hash), nil
}

func (b *ZondAPIBackend) GetLogs(ctx context.Context, hash common.Hash, number uint64) ([][]*types.Log, error) {
	return rawdb.ReadLogs(b.zond.chainDb, hash, number), nil
}

func (b *ZondAPIBackend) GetZVM(ctx context.Context, msg *core.Message, state *state.StateDB, header *types.Header, vmConfig *vm.Config, blockCtx *vm.BlockContext) *vm.ZVM {
	if vmConfig == nil {
		vmConfig = b.zond.blockchain.GetVMConfig()
	}
	txContext := core.NewZVMTxContext(msg)
	var context vm.BlockContext
	if blockCtx != nil {
		context = *blockCtx
	} else {
		context = core.NewZVMBlockContext(header, b.zond.BlockChain(), nil)
	}
	return vm.NewZVM(context, txContext, state, b.ChainConfig(), *vmConfig)
}

func (b *ZondAPIBackend) SubscribeRemovedLogsEvent(ch chan<- core.RemovedLogsEvent) event.Subscription {
	return b.zond.BlockChain().SubscribeRemovedLogsEvent(ch)
}

func (b *ZondAPIBackend) SubscribeChainEvent(ch chan<- core.ChainEvent) event.Subscription {
	return b.zond.BlockChain().SubscribeChainEvent(ch)
}

func (b *ZondAPIBackend) SubscribeChainHeadEvent(ch chan<- core.ChainHeadEvent) event.Subscription {
	return b.zond.BlockChain().SubscribeChainHeadEvent(ch)
}

func (b *ZondAPIBackend) SubscribeChainSideEvent(ch chan<- core.ChainSideEvent) event.Subscription {
	return b.zond.BlockChain().SubscribeChainSideEvent(ch)
}

func (b *ZondAPIBackend) SubscribeLogsEvent(ch chan<- []*types.Log) event.Subscription {
	return b.zond.BlockChain().SubscribeLogsEvent(ch)
}

func (b *ZondAPIBackend) SendTx(ctx context.Context, signedTx *types.Transaction) error {
	return b.zond.txPool.Add([]*types.Transaction{signedTx}, true, false)[0]
}

func (b *ZondAPIBackend) GetPoolTransactions() (types.Transactions, error) {
	pending := b.zond.txPool.Pending(txpool.PendingFilter{})
	var txs types.Transactions
	for _, batch := range pending {
		for _, lazy := range batch {
			if tx := lazy.Resolve(); tx != nil {
				txs = append(txs, tx)
			}
		}
	}
	return txs, nil
}

func (b *ZondAPIBackend) GetPoolTransaction(hash common.Hash) *types.Transaction {
	return b.zond.txPool.Get(hash)
}

func (b *ZondAPIBackend) GetTransaction(ctx context.Context, txHash common.Hash) (*types.Transaction, common.Hash, uint64, uint64, error) {
	tx, blockHash, blockNumber, index := rawdb.ReadTransaction(b.zond.ChainDb(), txHash)
	return tx, blockHash, blockNumber, index, nil
}

func (b *ZondAPIBackend) GetPoolNonce(ctx context.Context, addr common.Address) (uint64, error) {
	return b.zond.txPool.Nonce(addr), nil
}

func (b *ZondAPIBackend) Stats() (runnable int, blocked int) {
	return b.zond.txPool.Stats()
}

func (b *ZondAPIBackend) TxPoolContent() (map[common.Address][]*types.Transaction, map[common.Address][]*types.Transaction) {
	return b.zond.txPool.Content()
}

func (b *ZondAPIBackend) TxPoolContentFrom(addr common.Address) ([]*types.Transaction, []*types.Transaction) {
	return b.zond.txPool.ContentFrom(addr)
}

func (b *ZondAPIBackend) TxPool() *txpool.TxPool {
	return b.zond.txPool
}

func (b *ZondAPIBackend) SubscribeNewTxsEvent(ch chan<- core.NewTxsEvent) event.Subscription {
	return b.zond.txPool.SubscribeTransactions(ch)
}

func (b *ZondAPIBackend) SyncProgress() zond.SyncProgress {
	return b.zond.Downloader().Progress()
}

func (b *ZondAPIBackend) SuggestGasTipCap(ctx context.Context) (*big.Int, error) {
	return b.gpo.SuggestTipCap(ctx)
}

func (b *ZondAPIBackend) FeeHistory(ctx context.Context, blockCount uint64, lastBlock rpc.BlockNumber, rewardPercentiles []float64) (firstBlock *big.Int, reward [][]*big.Int, baseFee []*big.Int, gasUsedRatio []float64, err error) {
	return b.gpo.FeeHistory(ctx, blockCount, lastBlock, rewardPercentiles)
}

func (b *ZondAPIBackend) ChainDb() zonddb.Database {
	return b.zond.ChainDb()
}

func (b *ZondAPIBackend) EventMux() *event.TypeMux {
	return b.zond.EventMux()
}

func (b *ZondAPIBackend) AccountManager() *accounts.Manager {
	return b.zond.AccountManager()
}

func (b *ZondAPIBackend) ExtRPCEnabled() bool {
	return b.extRPCEnabled
}

func (b *ZondAPIBackend) RPCGasCap() uint64 {
	return b.zond.config.RPCGasCap
}

func (b *ZondAPIBackend) RPCZVMTimeout() time.Duration {
	return b.zond.config.RPCZVMTimeout
}

func (b *ZondAPIBackend) RPCTxFeeCap() float64 {
	return b.zond.config.RPCTxFeeCap
}

func (b *ZondAPIBackend) BloomStatus() (uint64, uint64) {
	sections, _, _ := b.zond.bloomIndexer.Sections()
	return params.BloomBitsBlocks, sections
}

func (b *ZondAPIBackend) ServiceFilter(ctx context.Context, session *bloombits.MatcherSession) {
	for i := 0; i < bloomFilterThreads; i++ {
		go session.Multiplex(bloomRetrievalBatch, bloomRetrievalWait, b.zond.bloomRequests)
	}
}

func (b *ZondAPIBackend) Engine() consensus.Engine {
	return b.zond.engine
}

func (b *ZondAPIBackend) CurrentHeader() *types.Header {
	return b.zond.blockchain.CurrentHeader()
}

func (b *ZondAPIBackend) StateAtBlock(ctx context.Context, block *types.Block, reexec uint64, base *state.StateDB, readOnly bool, preferDisk bool) (*state.StateDB, tracers.StateReleaseFunc, error) {
	return b.zond.stateAtBlock(ctx, block, reexec, base, readOnly, preferDisk)
}

func (b *ZondAPIBackend) StateAtTransaction(ctx context.Context, block *types.Block, txIndex int, reexec uint64) (*core.Message, vm.BlockContext, *state.StateDB, tracers.StateReleaseFunc, error) {
	return b.zond.stateAtTransaction(ctx, block, txIndex, reexec)
}
