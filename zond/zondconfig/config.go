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

// Package ethconfig contains the configuration of the ZOND protocol.
package zondconfig

import (
	"errors"
	"time"

	"github.com/theQRL/go-zond/common"
	"github.com/theQRL/go-zond/consensus"
	"github.com/theQRL/go-zond/consensus/beacon"
	"github.com/theQRL/go-zond/consensus/clique"
	"github.com/theQRL/go-zond/consensus/ethash"
	"github.com/theQRL/go-zond/core"
	"github.com/theQRL/go-zond/core/rawdb"
	"github.com/theQRL/go-zond/core/txpool/legacypool"
	"github.com/theQRL/go-zond/miner"
	"github.com/theQRL/go-zond/params"
	"github.com/theQRL/go-zond/zond/downloader"
	"github.com/theQRL/go-zond/zond/gasprice"
	"github.com/theQRL/go-zond/zonddb"
)

// FullNodeGPO contains default gasprice oracle settings for full node.
var FullNodeGPO = gasprice.Config{
	Blocks:           20,
	Percentile:       60,
	MaxHeaderHistory: 1024,
	MaxBlockHistory:  1024,
	MaxPrice:         gasprice.DefaultMaxPrice,
	IgnorePrice:      gasprice.DefaultIgnorePrice,
}

// Defaults contains default settings for use on the Ethereum main net.
var Defaults = Config{
	SyncMode:           downloader.SnapSync,
	NetworkId:          1,
	TransactionHistory: 2350000,
	StateHistory:       params.FullImmutabilityThreshold,
	StateScheme:        rawdb.HashScheme,
	DatabaseCache:      512,
	TrieCleanCache:     154,
	TrieDirtyCache:     256,
	TrieTimeout:        60 * time.Minute,
	SnapshotCache:      102,
	FilterLogCacheSize: 32,
	Miner:              miner.DefaultConfig,
	TxPool:             legacypool.DefaultConfig,
	RPCGasCap:          50000000,
	RPCEVMTimeout:      5 * time.Second,
	GPO:                FullNodeGPO,
	RPCTxFeeCap:        1, // 1 ether
}

//go:generate go run github.com/fjl/gencodec -type Config -formats toml -out gen_config.go

// Config contains configuration options for of the Zond protocol.
type Config struct {
	// The genesis block, which is inserted if the database is empty.
	// If nil, the Zond main net block is used.
	Genesis *core.Genesis `toml:",omitempty"`

	// Protocol options
	NetworkId uint64 // Network ID to use for selecting peers to connect to
	SyncMode  downloader.SyncMode

	// This can be set to list of enrtree:// URLs which will be queried for
	// for nodes to connect to.
	ZondDiscoveryURLs []string
	SnapDiscoveryURLs []string

	NoPruning  bool // Whether to disable pruning and flush everything to disk
	NoPrefetch bool // Whether to disable prefetching and only load state on demand

	TransactionHistory uint64 `toml:",omitempty"` // The maximum number of blocks from head whose tx indices are reserved.
	StateHistory       uint64 `toml:",omitempty"` // The maximum number of blocks from head whose state histories are reserved.
	StateScheme        string `toml:",omitempty"` // State scheme used to store zond state and merkle trie nodes on top

	// RequiredBlocks is a set of block number -> hash mappings which must be in the
	// canonical chain of all remote peers. Setting the option makes gzond verify the
	// presence of these blocks for every new peer connection.
	RequiredBlocks map[uint64]common.Hash `toml:"-"`

	// Database options
	SkipBcVersionCheck bool `toml:"-"`
	DatabaseHandles    int  `toml:"-"`
	DatabaseCache      int
	DatabaseFreezer    string

	TrieCleanCache int
	TrieDirtyCache int
	TrieTimeout    time.Duration
	SnapshotCache  int
	Preimages      bool

	// This is the number of blocks for which logs will be cached in the filter system.
	FilterLogCacheSize int

	// Mining options
	Miner miner.Config

	// Transaction pool options
	TxPool legacypool.Config

	// Gas Price Oracle options
	GPO gasprice.Config

	// Enables tracking of SHA3 preimages in the VM
	EnablePreimageRecording bool

	// Miscellaneous options
	DocRoot string `toml:"-"`

	// RPCGasCap is the global gas cap for eth-call variants.
	RPCGasCap uint64

	// RPCEVMTimeout is the global timeout for eth-call.
	RPCEVMTimeout time.Duration

	// RPCTxFeeCap is the global transaction fee(price * gaslimit) cap for
	// send-transaction variants. The unit is ether.
	RPCTxFeeCap float64
}

// CreateConsensusEngine creates a consensus engine for the given chain config.
// Clique is allowed for now to live standalone, but ethash is forbidden and can
// only exist on already merged networks.
func CreateConsensusEngine(config *params.ChainConfig, db zonddb.Database) (consensus.Engine, error) {
	// If proof-of-authority is requested, set it up
	if config.Clique != nil {
		return beacon.New(clique.New(config.Clique, db)), nil
	}
	// If defaulting to proof-of-work, enforce an already merged network since
	// we cannot run PoW algorithms and more, so we cannot even follow a chain
	// not coordinated by a beacon node.
	if !config.TerminalTotalDifficultyPassed {
		return nil, errors.New("ethash is only supported as a historical component of already merged networks")
	}
	return beacon.New(ethash.NewFaker()), nil
}
