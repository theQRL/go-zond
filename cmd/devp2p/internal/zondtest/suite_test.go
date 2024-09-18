// Copyright 2021 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

package zondtest

// TODO(now.youtrack.cloud/issue/TGZ-6)
/*
import (
	"time"

	"github.com/theQRL/go-zond/node"
	"github.com/theQRL/go-zond/p2p"
	"github.com/theQRL/go-zond/zond"
	"github.com/theQRL/go-zond/zond/zondconfig"
)

var (
	genesisFile   = "./testdata/genesis.json"
	halfchainFile = "./testdata/halfchain.rlp"
	fullchainFile = "./testdata/chain.rlp"
)

TODO(now.youtrack.cloud/issue/TGZ-6)
/*
func TestZondSuite(t *testing.T) {
	gzond, err := runGzond()
	if err != nil {
		t.Fatalf("could not run gzond: %v", err)
	}
	defer gzond.Close()

	suite, err := NewSuite(gzond.Server().Self(), fullchainFile, genesisFile)
	if err != nil {
		t.Fatalf("could not create new test suite: %v", err)
	}
	for _, test := range suite.ZondTests() {
		t.Run(test.Name, func(t *testing.T) {
			result := utesting.RunTAP([]utesting.Test{{Name: test.Name, Fn: test.Fn}}, os.Stdout)
			if result[0].Failed {
				t.Fatal()
			}
		})
	}
}

func TestSnapSuite(t *testing.T) {
	gzond, err := runGzond()
	if err != nil {
		t.Fatalf("could not run gzond: %v", err)
	}
	defer gzond.Close()

	suite, err := NewSuite(gzond.Server().Self(), fullchainFile, genesisFile)
	if err != nil {
		t.Fatalf("could not create new test suite: %v", err)
	}
	for _, test := range suite.SnapTests() {
		t.Run(test.Name, func(t *testing.T) {
			result := utesting.RunTAP([]utesting.Test{{Name: test.Name, Fn: test.Fn}}, os.Stdout)
			if result[0].Failed {
				t.Fatal()
			}
		})
	}
}

// runGzond creates and starts a gzond node
func runGzond() (*node.Node, error) {
	stack, err := node.New(&node.Config{
		P2P: p2p.Config{
			ListenAddr:  "127.0.0.1:0",
			NoDiscovery: true,
			MaxPeers:    10, // in case a test requires multiple connections, can be changed in the future
			NoDial:      true,
		},
	})
	if err != nil {
		return nil, err
	}

	err = setupGzond(stack)
	if err != nil {
		stack.Close()
		return nil, err
	}
	if err = stack.Start(); err != nil {
		stack.Close()
		return nil, err
	}
	return stack, nil
}

func setupGzond(stack *node.Node) error {
	chain, err := loadChain(halfchainFile, genesisFile)
	if err != nil {
		return err
	}

	backend, err := zond.New(stack, &zondconfig.Config{
		Genesis:        &chain.genesis,
		NetworkId:      chain.genesis.Config.ChainID.Uint64(), // 19763
		DatabaseCache:  10,
		TrieCleanCache: 10,
		TrieDirtyCache: 16,
		TrieTimeout:    60 * time.Minute,
		SnapshotCache:  10,
	})
	if err != nil {
		return err
	}

	_, err = backend.BlockChain().InsertChain(chain.blocks[1:])
	return err
}
*/
