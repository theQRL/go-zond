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

package zond

import (
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/theQRL/go-zond/common"
	"github.com/theQRL/go-zond/consensus/beacon"
	"github.com/theQRL/go-zond/core"
	"github.com/theQRL/go-zond/core/forkid"
	"github.com/theQRL/go-zond/core/rawdb"
	"github.com/theQRL/go-zond/core/types"
	"github.com/theQRL/go-zond/core/vm"
	"github.com/theQRL/go-zond/event"
	"github.com/theQRL/go-zond/p2p"
	"github.com/theQRL/go-zond/p2p/enode"
	"github.com/theQRL/go-zond/params"
	"github.com/theQRL/go-zond/zond/downloader"
	"github.com/theQRL/go-zond/zond/protocols/zond"
)

// testZondHandler is a mock event handler to listen for inbound network requests
// on the `zond` protocol and convert them into a more easily testable form.
type testZondHandler struct {
	txAnnounces  event.Feed
	txBroadcasts event.Feed
}

func (h *testZondHandler) Chain() *core.BlockChain                { panic("no backing chain") }
func (h *testZondHandler) TxPool() zond.TxPool                    { panic("no backing tx pool") }
func (h *testZondHandler) AcceptTxs() bool                        { return true }
func (h *testZondHandler) RunPeer(*zond.Peer, zond.Handler) error { panic("not used in tests") }
func (h *testZondHandler) PeerInfo(enode.ID) interface{}          { panic("not used in tests") }

func (h *testZondHandler) Handle(peer *zond.Peer, packet zond.Packet) error {
	switch packet := packet.(type) {
	case *zond.NewPooledTransactionHashesPacket66:
		h.txAnnounces.Send(([]common.Hash)(*packet))
		return nil

	case *zond.NewPooledTransactionHashesPacket68:
		h.txAnnounces.Send(packet.Hashes)
		return nil

	case *zond.TransactionsPacket:
		h.txBroadcasts.Send(([]*types.Transaction)(*packet))
		return nil

	case *zond.PooledTransactionsPacket:
		h.txBroadcasts.Send(([]*types.Transaction)(*packet))
		return nil

	default:
		panic(fmt.Sprintf("unexpected zond packet type in tests: %T", packet))
	}
}

// Tests that peers are correctly accepted (or rejected) based on the advertised
// fork IDs in the protocol handshake.
func TestForkIDSplit68(t *testing.T) { testForkIDSplit(t, zond.ETH68) }

func testForkIDSplit(t *testing.T, protocol uint) {
	t.Parallel()

	var (
		engine = beacon.NewFaker()

		configNoFork  = &params.ChainConfig{}
		configProFork = &params.ChainConfig{}
		dbNoFork      = rawdb.NewMemoryDatabase()
		dbProFork     = rawdb.NewMemoryDatabase()

		gspecNoFork  = &core.Genesis{Config: configNoFork}
		gspecProFork = &core.Genesis{Config: configProFork}

		chainNoFork, _  = core.NewBlockChain(dbNoFork, nil, gspecNoFork, engine, vm.Config{}, nil, nil)
		chainProFork, _ = core.NewBlockChain(dbProFork, nil, gspecProFork, engine, vm.Config{}, nil, nil)

		_, blocksNoFork, _  = core.GenerateChainWithGenesis(gspecNoFork, engine, 2, nil)
		_, blocksProFork, _ = core.GenerateChainWithGenesis(gspecProFork, engine, 2, nil)

		zondNoFork, _ = newHandler(&handlerConfig{
			Database:   dbNoFork,
			Chain:      chainNoFork,
			TxPool:     newTestTxPool(),
			Network:    1,
			Sync:       downloader.FullSync,
			BloomCache: 1,
		})
		zondProFork, _ = newHandler(&handlerConfig{
			Database:   dbProFork,
			Chain:      chainProFork,
			TxPool:     newTestTxPool(),
			Network:    1,
			Sync:       downloader.FullSync,
			BloomCache: 1,
		})
	)
	zondNoFork.Start(1000)
	zondProFork.Start(1000)

	// Clean up everything after ourselves
	defer chainNoFork.Stop()
	defer chainProFork.Stop()

	defer zondNoFork.Stop()
	defer zondProFork.Stop()

	// Both nodes should allow the other to connect (same genesis, next fork is the same)
	p2pNoFork, p2pProFork := p2p.MsgPipe()
	defer p2pNoFork.Close()
	defer p2pProFork.Close()

	peerNoFork := zond.NewPeer(protocol, p2p.NewPeerPipe(enode.ID{1}, "", nil, p2pNoFork), p2pNoFork, nil)
	peerProFork := zond.NewPeer(protocol, p2p.NewPeerPipe(enode.ID{2}, "", nil, p2pProFork), p2pProFork, nil)
	defer peerNoFork.Close()
	defer peerProFork.Close()

	errc := make(chan error, 2)
	go func(errc chan error) {
		errc <- zondNoFork.runZondPeer(peerProFork, func(peer *zond.Peer) error { return nil })
	}(errc)
	go func(errc chan error) {
		errc <- zondProFork.runZondPeer(peerNoFork, func(peer *zond.Peer) error { return nil })
	}(errc)

	for i := 0; i < 2; i++ {
		select {
		case err := <-errc:
			if err != nil {
				t.Fatalf("frontier nofork <-> profork failed: %v", err)
			}
		case <-time.After(250 * time.Millisecond):
			t.Fatalf("frontier nofork <-> profork handler timeout")
		}
	}
	// Progress into Homestead. Fork's match, so we don't care what the future holds
	chainNoFork.InsertChain(blocksNoFork[:1])
	chainProFork.InsertChain(blocksProFork[:1])

	p2pNoFork, p2pProFork = p2p.MsgPipe()
	defer p2pNoFork.Close()
	defer p2pProFork.Close()

	peerNoFork = zond.NewPeer(protocol, p2p.NewPeer(enode.ID{1}, "", nil), p2pNoFork, nil)
	peerProFork = zond.NewPeer(protocol, p2p.NewPeer(enode.ID{2}, "", nil), p2pProFork, nil)
	defer peerNoFork.Close()
	defer peerProFork.Close()

	errc = make(chan error, 2)
	go func(errc chan error) {
		errc <- zondNoFork.runZondPeer(peerProFork, func(peer *zond.Peer) error { return nil })
	}(errc)
	go func(errc chan error) {
		errc <- zondProFork.runZondPeer(peerNoFork, func(peer *zond.Peer) error { return nil })
	}(errc)

	for i := 0; i < 2; i++ {
		select {
		case err := <-errc:
			if err != nil {
				t.Fatalf("homestead nofork <-> profork failed: %v", err)
			}
		case <-time.After(250 * time.Millisecond):
			t.Fatalf("homestead nofork <-> profork handler timeout")
		}
	}
	// Progress into Spurious. Forks mismatch, signalling differing chains, reject
	chainNoFork.InsertChain(blocksNoFork[1:2])
	chainProFork.InsertChain(blocksProFork[1:2])

	p2pNoFork, p2pProFork = p2p.MsgPipe()
	defer p2pNoFork.Close()
	defer p2pProFork.Close()

	peerNoFork = zond.NewPeer(protocol, p2p.NewPeerPipe(enode.ID{1}, "", nil, p2pNoFork), p2pNoFork, nil)
	peerProFork = zond.NewPeer(protocol, p2p.NewPeerPipe(enode.ID{2}, "", nil, p2pProFork), p2pProFork, nil)
	defer peerNoFork.Close()
	defer peerProFork.Close()

	errc = make(chan error, 2)
	go func(errc chan error) {
		errc <- zondNoFork.runZondPeer(peerProFork, func(peer *zond.Peer) error { return nil })
	}(errc)
	go func(errc chan error) {
		errc <- zondProFork.runZondPeer(peerNoFork, func(peer *zond.Peer) error { return nil })
	}(errc)

	var successes int
	for i := 0; i < 2; i++ {
		select {
		case err := <-errc:
			if err == nil {
				successes++
				if successes == 2 { // Only one side disconnects
					t.Fatalf("fork ID rejection didn't happen")
				}
			}
		case <-time.After(250 * time.Millisecond):
			t.Fatalf("split peers not rejected")
		}
	}
}

// Tests that received transactions are added to the local pool.
func TestRecvTransactions68(t *testing.T) { testRecvTransactions(t, zond.ETH68) }

func testRecvTransactions(t *testing.T, protocol uint) {
	t.Parallel()

	// Create a message handler, configure it to accept transactions and watch them
	handler := newTestHandler()
	defer handler.close()

	handler.handler.acceptTxs.Store(true) // mark synced to accept transactions

	txs := make(chan core.NewTxsEvent)
	sub := handler.txpool.SubscribeNewTxsEvent(txs)
	defer sub.Unsubscribe()

	// Create a source peer to send messages through and a sink handler to receive them
	p2pSrc, p2pSink := p2p.MsgPipe()
	defer p2pSrc.Close()
	defer p2pSink.Close()

	src := zond.NewPeer(protocol, p2p.NewPeerPipe(enode.ID{1}, "", nil, p2pSrc), p2pSrc, handler.txpool)
	sink := zond.NewPeer(protocol, p2p.NewPeerPipe(enode.ID{2}, "", nil, p2pSink), p2pSink, handler.txpool)
	defer src.Close()
	defer sink.Close()

	go handler.handler.runZondPeer(sink, func(peer *zond.Peer) error {
		return zond.Handle((*zondHandler)(handler.handler), peer)
	})
	// Run the handshake locally to avoid spinning up a source handler
	var (
		genesis = handler.chain.Genesis()
		head    = handler.chain.CurrentBlock()
	)
	if err := src.Handshake(1, head.Hash(), genesis.Hash(), forkid.NewIDWithChain(handler.chain), forkid.NewFilter(handler.chain)); err != nil {
		t.Fatalf("failed to run protocol handshake")
	}
	// Send the transaction to the sink and verify that it's added to the tx pool
	tx := types.NewTransaction(0, common.Address{}, big.NewInt(0), 100000, big.NewInt(0), nil)
	tx, _ = types.SignTx(tx, types.ShanghaiSigner{ChainId: big.NewInt(0)}, testKey)

	if err := src.SendTransactions([]*types.Transaction{tx}); err != nil {
		t.Fatalf("failed to send transaction: %v", err)
	}
	select {
	case event := <-txs:
		if len(event.Txs) != 1 {
			t.Errorf("wrong number of added transactions: got %d, want 1", len(event.Txs))
		} else if event.Txs[0].Hash() != tx.Hash() {
			t.Errorf("added wrong tx hash: got %v, want %v", event.Txs[0].Hash(), tx.Hash())
		}
	case <-time.After(2 * time.Second):
		t.Errorf("no NewTxsEvent received within 2 seconds")
	}
}

// This test checks that pending transactions are sent.
func TestSendTransactions68(t *testing.T) { testSendTransactions(t, zond.ETH68) }

func testSendTransactions(t *testing.T, protocol uint) {
	t.Parallel()

	// Create a message handler and fill the pool with big transactions
	handler := newTestHandler()
	defer handler.close()

	insert := make([]*types.Transaction, 100)
	for nonce := range insert {
		tx := types.NewTransaction(uint64(nonce), common.Address{}, big.NewInt(0), 100000, big.NewInt(0), make([]byte, 10240))
		tx, _ = types.SignTx(tx, types.ShanghaiSigner{ChainId: big.NewInt(0)}, testKey)

		insert[nonce] = tx
	}
	// TODO(rgeraldes24)
	// go handler.txpool.AddRemotes(insert) // Need goroutine to not block on feed
	time.Sleep(250 * time.Millisecond) // Wait until tx events get out of the system (can't use events, tx broadcaster races with peer join)

	// Create a source handler to send messages through and a sink peer to receive them
	p2pSrc, p2pSink := p2p.MsgPipe()
	defer p2pSrc.Close()
	defer p2pSink.Close()

	src := zond.NewPeer(protocol, p2p.NewPeerPipe(enode.ID{1}, "", nil, p2pSrc), p2pSrc, handler.txpool)
	sink := zond.NewPeer(protocol, p2p.NewPeerPipe(enode.ID{2}, "", nil, p2pSink), p2pSink, handler.txpool)
	defer src.Close()
	defer sink.Close()

	go handler.handler.runZondPeer(src, func(peer *zond.Peer) error {
		return zond.Handle((*zondHandler)(handler.handler), peer)
	})
	// Run the handshake locally to avoid spinning up a source handler
	var (
		genesis = handler.chain.Genesis()
		head    = handler.chain.CurrentBlock()
	)
	if err := sink.Handshake(1, head.Hash(), genesis.Hash(), forkid.NewIDWithChain(handler.chain), forkid.NewFilter(handler.chain)); err != nil {
		t.Fatalf("failed to run protocol handshake")
	}
	// After the handshake completes, the source handler should stream the sink
	// the transactions, subscribe to all inbound network events
	backend := new(testZondHandler)

	anns := make(chan []common.Hash)
	annSub := backend.txAnnounces.Subscribe(anns)
	defer annSub.Unsubscribe()

	bcasts := make(chan []*types.Transaction)
	bcastSub := backend.txBroadcasts.Subscribe(bcasts)
	defer bcastSub.Unsubscribe()

	go zond.Handle(backend, sink)

	// Make sure we get all the transactions on the correct channels
	seen := make(map[common.Hash]struct{})
	for len(seen) < len(insert) {
		switch protocol {
		case 66, 67, 68:
			select {
			case hashes := <-anns:
				for _, hash := range hashes {
					if _, ok := seen[hash]; ok {
						t.Errorf("duplicate transaction announced: %x", hash)
					}
					seen[hash] = struct{}{}
				}
			case <-bcasts:
				t.Errorf("initial tx broadcast received on post zond/66")
			}

		default:
			panic("unsupported protocol, please extend test")
		}
	}
	for _, tx := range insert {
		if _, ok := seen[tx.Hash()]; !ok {
			t.Errorf("missing transaction: %x", tx.Hash())
		}
	}
}

// Tests that transactions get propagated to all attached peers, either via direct
// broadcasts or via announcements/retrievals.
func TestTransactionPropagation68(t *testing.T) { testTransactionPropagation(t, zond.ETH68) }

func testTransactionPropagation(t *testing.T, protocol uint) {
	t.Parallel()

	// Create a source handler to send transactions from and a number of sinks
	// to receive them. We need multiple sinks since a one-to-one peering would
	// broadcast all transactions without announcement.
	source := newTestHandler()
	source.handler.snapSync.Store(false) // Avoid requiring snap, otherwise some will be dropped below
	defer source.close()

	sinks := make([]*testHandler, 10)
	for i := 0; i < len(sinks); i++ {
		sinks[i] = newTestHandler()
		defer sinks[i].close()

		sinks[i].handler.acceptTxs.Store(true) // mark synced to accept transactions
	}
	// Interconnect all the sink handlers with the source handler
	for i, sink := range sinks {
		sink := sink // Closure for gorotuine below

		sourcePipe, sinkPipe := p2p.MsgPipe()
		defer sourcePipe.Close()
		defer sinkPipe.Close()

		sourcePeer := zond.NewPeer(protocol, p2p.NewPeerPipe(enode.ID{byte(i + 1)}, "", nil, sourcePipe), sourcePipe, source.txpool)
		sinkPeer := zond.NewPeer(protocol, p2p.NewPeerPipe(enode.ID{0}, "", nil, sinkPipe), sinkPipe, sink.txpool)
		defer sourcePeer.Close()
		defer sinkPeer.Close()

		go source.handler.runZondPeer(sourcePeer, func(peer *zond.Peer) error {
			return zond.Handle((*zondHandler)(source.handler), peer)
		})
		go sink.handler.runZondPeer(sinkPeer, func(peer *zond.Peer) error {
			return zond.Handle((*zondHandler)(sink.handler), peer)
		})
	}
	// Subscribe to all the transaction pools
	txChs := make([]chan core.NewTxsEvent, len(sinks))
	for i := 0; i < len(sinks); i++ {
		txChs[i] = make(chan core.NewTxsEvent, 1024)

		sub := sinks[i].txpool.SubscribeNewTxsEvent(txChs[i])
		defer sub.Unsubscribe()
	}
	// Fill the source pool with transactions and wait for them at the sinks
	txs := make([]*types.Transaction, 1024)
	for nonce := range txs {
		tx := types.NewTransaction(uint64(nonce), common.Address{}, big.NewInt(0), 100000, big.NewInt(0), nil)
		tx, _ = types.SignTx(tx, types.ShanghaiSigner{ChainId: big.NewInt(0)}, testKey)

		txs[nonce] = tx
	}
	// TODO(rgeraldes24)
	// source.txpool.AddRemotes(txs)

	// Iterate through all the sinks and ensure they all got the transactions
	for i := range sinks {
		for arrived, timeout := 0, false; arrived < len(txs) && !timeout; {
			select {
			case event := <-txChs[i]:
				arrived += len(event.Txs)
			case <-time.After(2 * time.Second):
				t.Errorf("sink %d: transaction propagation timed out: have %d, want %d", i, arrived, len(txs))
				timeout = true
			}
		}
	}
}
