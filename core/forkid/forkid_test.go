// Copyright 2019 The go-ethereum Authors
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

package forkid

import (
	"bytes"
	"hash/crc32"
	"math/big"

	// "hash/crc32"
	"math"
	// "math/big"
	"testing"

	"github.com/theQRL/go-zond/common"
	"github.com/theQRL/go-zond/core"
	"github.com/theQRL/go-zond/core/types"
	"github.com/theQRL/go-zond/params"

	// "github.com/theQRL/go-zond/core"
	// "github.com/theQRL/go-zond/core/types"
	// "github.com/theQRL/go-zond/params"
	"github.com/theQRL/go-zond/rlp"
)

// TestCreation tests that different genesis and fork rule combinations result in
// the correct fork ID.
func TestCreation(t *testing.T) {
	type testcase struct {
		head uint64
		time uint64
		want ID
	}
	tests := []struct {
		config  *params.ChainConfig
		genesis *types.Block
		cases   []testcase
	}{
		// Mainnet test cases
		{
			params.MainnetChainConfig,
			core.DefaultGenesisBlock().ToBlock(),
			[]testcase{
				{0, 0, ID{Hash: checksumToBytes(0x6170f487), Next: 0}},
				// NOTE(rgeraldes24): revisit upon new fork
				/*
					{0, 0, ID{Hash: checksumToBytes(0xfc64ec04), Next: 1150000}},                    // Unsynced
					{1149999, 0, ID{Hash: checksumToBytes(0xfc64ec04), Next: 1150000}},              // Last Frontier block
					{1150000, 0, ID{Hash: checksumToBytes(0x97c2c34c), Next: 1920000}},              // First Homestead block
					{1919999, 0, ID{Hash: checksumToBytes(0x97c2c34c), Next: 1920000}},              // Last Homestead block
					{1920000, 0, ID{Hash: checksumToBytes(0x91d1f948), Next: 2463000}},              // First DAO block
					{2462999, 0, ID{Hash: checksumToBytes(0x91d1f948), Next: 2463000}},              // Last DAO block
					{2463000, 0, ID{Hash: checksumToBytes(0x7a64da13), Next: 2675000}},              // First Tangerine block
					{2674999, 0, ID{Hash: checksumToBytes(0x7a64da13), Next: 2675000}},              // Last Tangerine block
					{2675000, 0, ID{Hash: checksumToBytes(0x3edd5b10), Next: 4370000}},              // First Spurious block
					{4369999, 0, ID{Hash: checksumToBytes(0x3edd5b10), Next: 4370000}},              // Last Spurious block
					{4370000, 0, ID{Hash: checksumToBytes(0xa00bc324), Next: 7280000}},              // First Byzantium block
					{7279999, 0, ID{Hash: checksumToBytes(0xa00bc324), Next: 7280000}},              // Last Byzantium block
					{7280000, 0, ID{Hash: checksumToBytes(0x668db0af), Next: 9069000}},              // First and last Constantinople, first Petersburg block
					{9068999, 0, ID{Hash: checksumToBytes(0x668db0af), Next: 9069000}},              // Last Petersburg block
					{9069000, 0, ID{Hash: checksumToBytes(0x879d6e30), Next: 9200000}},              // First Istanbul and first Muir Glacier block
					{9199999, 0, ID{Hash: checksumToBytes(0x879d6e30), Next: 9200000}},              // Last  and first Muir Glacier block
					{9200000, 0, ID{Hash: checksumToBytes(0xe029e991), Next: 12244000}},             // First Muir Glacier block
					{12243999, 0, ID{Hash: checksumToBytes(0xe029e991), Next: 12244000}},            // Last Muir Glacier block
					{12244000, 0, ID{Hash: checksumToBytes(0x0eb440f6), Next: 12965000}},            // First Berlin block
					{12964999, 0, ID{Hash: checksumToBytes(0x0eb440f6), Next: 12965000}},            // Last Berlin block
					{12965000, 0, ID{Hash: checksumToBytes(0xb715077d), Next: 13773000}},            // First London block
					{13772999, 0, ID{Hash: checksumToBytes(0xb715077d), Next: 13773000}},            // Last London block
					{13773000, 0, ID{Hash: checksumToBytes(0x20c327fc), Next: 15050000}},            // First Arrow Glacier block
					{15049999, 0, ID{Hash: checksumToBytes(0x20c327fc), Next: 15050000}},            // Last Arrow Glacier block
					{15050000, 0, ID{Hash: checksumToBytes(0xf0afd0e3), Next: 1681338455}},          // First Gray Glacier block
					{20000000, 1681338454, ID{Hash: checksumToBytes(0xf0afd0e3), Next: 1681338455}}, // Last Gray Glacier block
					{20000000, 1681338455, ID{Hash: checksumToBytes(0xdce96c2d), Next: 0}},          // First Shanghai block
					{30000000, 2000000000, ID{Hash: checksumToBytes(0xdce96c2d), Next: 0}},          // Future Shanghai block
				*/
			},
		},
	}
	for i, tt := range tests {
		for j, ttt := range tt.cases {
			if have := NewID(tt.config, tt.genesis, ttt.head, ttt.time); have != ttt.want {
				t.Errorf("test %d, case %d: fork ID mismatch: have %x, want %x", i, j, have, ttt.want)
			}
		}
	}
}

// NOTE(rgeraldes24): revisit upon new fork
// TestValidation tests that a local peer correctly validates and accepts a remote
// fork ID.
func TestValidation(t *testing.T) {
	tests := []struct {
		config *params.ChainConfig
		head   uint64
		time   uint64
		id     ID
		err    error
	}{
		//----------------------
		// Timestamp based tests
		//----------------------

		// Local is mainnet Shanghai, remote announces the same. No future fork is announced.
		{params.MainnetChainConfig, 15050000, 20000000, ID{Hash: checksumToBytes(0x6170f487), Next: 0}, nil},

		// Local is mainnet Shanghai, remote announces the same. Remote also announces a next fork
		// at time 0xffffffff, but that is uncertain.
		{params.MainnetChainConfig, 15050000, 20000000, ID{Hash: checksumToBytes(0x6170f487), Next: math.MaxUint64}, nil},

		// Local is mainnet Shanghai, and isn't aware of more forks. Remote announces Shanghai +
		// 0xffffffff. Local needs software update, reject.
		{params.MainnetChainConfig, 20000000, 1681338455, ID{Hash: checksumToBytes(checksumUpdate(0xdce96c2d, math.MaxUint64)), Next: 0}, ErrLocalIncompatibleOrStale},

		// Local is mainnet Shanghai, remote is random Shanghai.
		{params.MainnetChainConfig, 20000000, 1681338455, ID{Hash: checksumToBytes(0x12345678), Next: 0}, ErrLocalIncompatibleOrStale},

		// Local is mainnet Shanghai, far in the future. Remote announces Gopherium (non existing fork)
		// at some future timestamp 8888888888, for itself, but past block for local. Local is incompatible.
		//
		// This case detects non-upgraded nodes with majority hash power (typical Ropsten mess).
		{params.MainnetChainConfig, 88888888, 8888888888, ID{Hash: checksumToBytes(0xdce96c2d), Next: 8888888888}, ErrLocalIncompatibleOrStale},
	}
	for i, tt := range tests {
		filter := newFilter(tt.config, core.DefaultGenesisBlock().ToBlock(), func() (uint64, uint64) { return tt.head, tt.time })
		if err := filter(tt.id); err != tt.err {
			t.Errorf("test %d: validation error mismatch: have %v, want %v", i, err, tt.err)
		}
	}
}

// Tests that IDs are properly RLP encoded (specifically important because we
// use uint32 to store the hash, but we need to encode it as [4]byte).
func TestEncoding(t *testing.T) {
	tests := []struct {
		id   ID
		want []byte
	}{
		{ID{Hash: checksumToBytes(0), Next: 0}, common.Hex2Bytes("c6840000000080")},
		{ID{Hash: checksumToBytes(0xdeadbeef), Next: 0xBADDCAFE}, common.Hex2Bytes("ca84deadbeef84baddcafe,")},
		{ID{Hash: checksumToBytes(math.MaxUint32), Next: math.MaxUint64}, common.Hex2Bytes("ce84ffffffff88ffffffffffffffff")},
	}
	for i, tt := range tests {
		have, err := rlp.EncodeToBytes(tt.id)
		if err != nil {
			t.Errorf("test %d: failed to encode forkid: %v", i, err)
			continue
		}
		if !bytes.Equal(have, tt.want) {
			t.Errorf("test %d: RLP mismatch: have %x, want %x", i, have, tt.want)
		}
	}
}

// Tests that time-based forks which are active at genesis are not included in
// forkid hash.
func TestTimeBasedForkInGenesis(t *testing.T) {
	var (
		time       = uint64(1690475657)
		genesis    = types.NewBlockWithHeader(&types.Header{Time: time})
		forkidHash = checksumToBytes(crc32.ChecksumIEEE(genesis.Hash().Bytes()))
		config     = func() *params.ChainConfig {
			return &params.ChainConfig{
				ChainID: big.NewInt(1337),
			}
		}
	)
	tests := []struct {
		config *params.ChainConfig
		want   ID
	}{
		// NOTE(rgeraldes24): revisit upon new fork
		/*
			// Shanghai active before genesis, skip
			{config(), ID{Hash: forkidHash, Next: time + 1}},

			// Shanghai active at genesis, skip
			{config(), ID{Hash: forkidHash, Next: time + 1}},

			// Shanghai not active, skip
			{config(), ID{Hash: forkidHash, Next: time + 1}},
		*/
		// No forks
		{config(), ID{Hash: forkidHash, Next: 0}},
	}
	for _, tt := range tests {
		if have := NewID(tt.config, genesis, 0, time); have != tt.want {
			t.Fatalf("incorrect forkid hash: have %x, want %x", have, tt.want)
		}
	}
}
