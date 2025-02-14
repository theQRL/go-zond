// Copyright 2017 The go-ethereum Authors
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

package core

import (
	"encoding/json"
	"math/big"
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/theQRL/go-zond/common"
	"github.com/theQRL/go-zond/core/rawdb"
	"github.com/theQRL/go-zond/params"
	"github.com/theQRL/go-zond/trie"
	"github.com/theQRL/go-zond/trie/triedb/pathdb"
	"github.com/theQRL/go-zond/zonddb"
)

func TestSetupGenesis(t *testing.T) {
	testSetupGenesis(t, rawdb.HashScheme)
	testSetupGenesis(t, rawdb.PathScheme)
}

func testSetupGenesis(t *testing.T, scheme string) {
	var (
		customghash = common.HexToHash("0x512a0d99941f1551db550852bdec6c9e213595356ede9dd23d1572199a8d66ba")
		customg     = Genesis{
			Config: &params.ChainConfig{},
			Alloc: GenesisAlloc{
				{1}: {Balance: big.NewInt(1), Storage: map[common.Hash]common.Hash{{1}: {1}}},
			},
		}
		oldcustomg = customg
	)
	oldcustomg.Config = &params.ChainConfig{}

	tests := []struct {
		name       string
		fn         func(zonddb.Database) (*params.ChainConfig, common.Hash, error)
		wantConfig *params.ChainConfig
		wantHash   common.Hash
		wantErr    error
	}{
		{
			name: "genesis without ChainConfig",
			fn: func(db zonddb.Database) (*params.ChainConfig, common.Hash, error) {
				return SetupGenesisBlock(db, trie.NewDatabase(db, newDbConfig(scheme)), new(Genesis))
			},
			wantErr:    errGenesisNoConfig,
			wantConfig: params.AllBeaconProtocolChanges,
		},
		// TODO(now.youtrack.cloud/issue/TGZ-16)
		/*
			{
				name: "no block in DB, genesis == nil",
				fn: func(db zonddb.Database) (*params.ChainConfig, common.Hash, error) {
					return SetupGenesisBlock(db, trie.NewDatabase(db, newDbConfig(scheme)), nil)
				},
				// wantHash:   params.MainnetGenesisHash,
				wantHash:   common.HexToHash("b3de630542cf9acf842e24f428c7c21b7824b38a7718a632e424b58ba0f562c6"),
				wantConfig: params.MainnetChainConfig,
			},
			{
				name: "mainnet block in DB, genesis == nil",
				fn: func(db zonddb.Database) (*params.ChainConfig, common.Hash, error) {
					DefaultGenesisBlock().MustCommit(db, trie.NewDatabase(db, newDbConfig(scheme)))
					return SetupGenesisBlock(db, trie.NewDatabase(db, newDbConfig(scheme)), nil)
				},
				wantHash:   params.MainnetGenesisHash,
				wantConfig: params.MainnetChainConfig,
			},
		*/
		{
			name: "custom block in DB, genesis == nil",
			fn: func(db zonddb.Database) (*params.ChainConfig, common.Hash, error) {
				tdb := trie.NewDatabase(db, newDbConfig(scheme))
				customg.Commit(db, tdb)
				return SetupGenesisBlock(db, tdb, nil)
			},
			wantHash:   customghash,
			wantConfig: customg.Config,
		},
		// TODO(now.youtrack.cloud/issue/TGZ-16)
		/*
			{
				name: "custom block in DB, genesis == goerli",
				fn: func(db zonddb.Database) (*params.ChainConfig, common.Hash, error) {
					tdb := trie.NewDatabase(db, newDbConfig(scheme))
					customg.Commit(db, tdb)
					return SetupGenesisBlock(db, tdb, DefaultGoerliGenesisBlock())
				},
				wantErr:    &GenesisMismatchError{Stored: customghash, New: params.GoerliGenesisHash},
				wantHash:   params.GoerliGenesisHash,
				wantConfig: params.GoerliChainConfig,
			},
		*/
		{
			name: "compatible config in DB",
			fn: func(db zonddb.Database) (*params.ChainConfig, common.Hash, error) {
				tdb := trie.NewDatabase(db, newDbConfig(scheme))
				oldcustomg.Commit(db, tdb)
				return SetupGenesisBlock(db, tdb, &customg)
			},
			wantHash:   customghash,
			wantConfig: customg.Config,
		},
		// NOTE(rgeraldes24): not valid for now
		/*
			{
				name: "incompatible config in DB",
				fn: func(db zonddb.Database) (*params.ChainConfig, common.Hash, error) {
					// Commit the 'old' genesis block with Homestead transition at #2.
					// Advance to block #4, past the homestead transition block of customg.
					tdb := trie.NewDatabase(db, newDbConfig(scheme))
					oldcustomg.Commit(db, tdb)

					bc, _ := NewBlockChain(db, DefaultCacheConfigWithScheme(scheme), &oldcustomg, beacon.NewFullFaker(), vm.Config{}, nil, nil)
					defer bc.Stop()

					_, blocks, _ := GenerateChainWithGenesis(&oldcustomg, beacon.NewFaker(), 4, nil)
					bc.InsertChain(blocks)

					// This should return a compatibility error.
					return SetupGenesisBlock(db, tdb, &customg)
				},
				wantHash:   customghash,
				wantConfig: customg.Config,
				wantErr: &params.ConfigCompatError{
					What:          "Homestead fork block",
					StoredBlock:   big.NewInt(2),
					NewBlock:      big.NewInt(3),
					RewindToBlock: 1,
				},
			},
		*/
	}

	for _, test := range tests {
		db := rawdb.NewMemoryDatabase()
		config, hash, err := test.fn(db)
		// Check the return values.
		if !reflect.DeepEqual(err, test.wantErr) {
			spew := spew.ConfigState{DisablePointerAddresses: true, DisableCapacities: true}
			t.Errorf("%s: returned error %#v, want %#v", test.name, spew.NewFormatter(err), spew.NewFormatter(test.wantErr))
		}
		if !reflect.DeepEqual(config, test.wantConfig) {
			t.Errorf("%s:\nreturned %v\nwant     %v", test.name, config, test.wantConfig)
		}
		if hash != test.wantHash {
			t.Errorf("%s: returned hash %s, want %s", test.name, hash.Hex(), test.wantHash.Hex())
		} else if err == nil {
			// Check database content.
			stored := rawdb.ReadBlock(db, test.wantHash, 0)
			if stored.Hash() != test.wantHash {
				t.Errorf("%s: block in DB has hash %s, want %s", test.name, stored.Hash(), test.wantHash)
			}
		}
	}
}

// TestGenesisHashes checks the congruity of default genesis data to
// corresponding hardcoded genesis hash values.
func TestGenesisHashes(t *testing.T) {
	for i, c := range []struct {
		genesis *Genesis
		want    common.Hash
	}{
		// TODO(now.youtrack.cloud/issue/TGZ-16)
		{DefaultGenesisBlock(), params.MainnetGenesisHash},
		{DefaultBetaNetGenesisBlock(), params.BetaNetGenesisHash},
	} {
		// Test via MustCommit
		db := rawdb.NewMemoryDatabase()
		if have := c.genesis.MustCommit(db, trie.NewDatabase(db, trie.HashDefaults)).Hash(); have != c.want {
			t.Errorf("case: %d a), want: %s, got: %s", i, c.want.Hex(), have.Hex())
		}
		// Test via ToBlock
		if have := c.genesis.ToBlock().Hash(); have != c.want {
			t.Errorf("case: %d a), want: %s, got: %s", i, c.want.Hex(), have.Hex())
		}
	}
}

func TestGenesis_Commit(t *testing.T) {
	genesis := &Genesis{
		Config: params.TestChainConfig,
		// basefee is nil
	}

	db := rawdb.NewMemoryDatabase()
	genesisBlock := genesis.MustCommit(db, trie.NewDatabase(db, trie.HashDefaults))

	if genesis.BaseFee != nil {
		t.Fatalf("assumption wrong")
	}

	// This value should have been set as default in the ToBlock method.
	if genesisBlock.BaseFee().Cmp(new(big.Int).SetUint64(params.InitialBaseFee)) != 0 {
		t.Errorf("assumption wrong: want: %d, got: %v", params.InitialBaseFee, genesisBlock.BaseFee())
	}

	// Expect the stored basefee to be the basefee of the genesis block.
	blk := rawdb.ReadBlock(db, genesisBlock.Hash(), 0)
	if blk == nil {
		t.Errorf("unable to retrieve block %d for canonical hash: %s", blk.NumberU64(), blk.Hash())
		return
	}

	if blk.BaseFee().Cmp(genesisBlock.BaseFee()) != 0 {
		t.Errorf("inequal difficulty; stored: %v, genesisBlock: %v", blk.BaseFee(), genesisBlock.BaseFee())
	}
}

func TestReadWriteGenesisAlloc(t *testing.T) {
	var (
		db    = rawdb.NewMemoryDatabase()
		alloc = &GenesisAlloc{
			{1}: {Balance: big.NewInt(1), Storage: map[common.Hash]common.Hash{{1}: {1}}},
			{2}: {Balance: big.NewInt(2), Storage: map[common.Hash]common.Hash{{2}: {2}}},
		}
		hash, _ = alloc.deriveHash()
	)
	blob, _ := json.Marshal(alloc)
	rawdb.WriteGenesisStateSpec(db, hash, blob)

	var reload GenesisAlloc
	err := reload.UnmarshalJSON(rawdb.ReadGenesisStateSpec(db, hash))
	if err != nil {
		t.Fatalf("Failed to load genesis state %v", err)
	}
	if len(reload) != len(*alloc) {
		t.Fatal("Unexpected genesis allocation")
	}
	for addr, account := range reload {
		want, ok := (*alloc)[addr]
		if !ok {
			t.Fatal("Account is not found")
		}
		if !reflect.DeepEqual(want, account) {
			t.Fatal("Unexpected account")
		}
	}
}

func newDbConfig(scheme string) *trie.Config {
	if scheme == rawdb.HashScheme {
		return trie.HashDefaults
	}
	return &trie.Config{PathDB: pathdb.Defaults}
}
