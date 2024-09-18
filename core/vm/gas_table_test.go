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

package vm

import (
	"bytes"
	"math/big"
	"sort"
	"testing"

	"github.com/theQRL/go-zond/common"
	"github.com/theQRL/go-zond/common/hexutil"
	"github.com/theQRL/go-zond/core/rawdb"
	"github.com/theQRL/go-zond/core/state"
	"github.com/theQRL/go-zond/core/types"
	"github.com/theQRL/go-zond/params"
)

func TestMemoryGasCost(t *testing.T) {
	tests := []struct {
		size     uint64
		cost     uint64
		overflow bool
	}{
		{0x1fffffffe0, 36028809887088637, false},
		{0x1fffffffe1, 0, true},
	}
	for i, tt := range tests {
		v, err := memoryGasCost(&Memory{}, tt.size)
		if (err == ErrGasUintOverflow) != tt.overflow {
			t.Errorf("test %d: overflow mismatch: have %v, want %v", i, err == ErrGasUintOverflow, tt.overflow)
		}
		if v != tt.cost {
			t.Errorf("test %d: gas cost mismatch: have %v, want %v", i, v, tt.cost)
		}
	}
}

var createGasTests = []struct {
	code       string
	gasUsed    uint64
	minimumGas uint64
}{
	// legacy create(0, 0, 0xc000) _with_ 3860
	{"0x61C00060006000f0" + "600052" + "60206000F3", 44309, 44309},
	// create2(0, 0, 0xc001, 0) (too large), with 3860
	{"0x600061C00160006000f5" + "600052" + "60206000F3", 32012, 100_000},
	// create2(0, 0, 0xc000, 0)
	// This case is trying to deploy code at (within) the limit
	{"0x600061C00060006000f5" + "600052" + "60206000F3", 53528, 53528},
	// create2(0, 0, 0xc001, 0)
	// This case is trying to deploy code exceeding the limit
	{"0x600061C00160006000f5" + "600052" + "60206000F3", 32024, 100000},
}

func TestCreateGas(t *testing.T) {
	for i, tt := range createGasTests {
		var gasUsed = uint64(0)
		doCheck := func(testGas int) bool {
			address := common.BytesToAddress([]byte("contract"))
			statedb, _ := state.New(types.EmptyRootHash, state.NewDatabase(rawdb.NewMemoryDatabase()), nil)
			statedb.CreateAccount(address)
			statedb.SetCode(address, hexutil.MustDecode(tt.code))
			statedb.Finalise(true)
			vmctx := BlockContext{
				CanTransfer: func(StateDB, common.Address, *big.Int) bool { return true },
				Transfer:    func(StateDB, common.Address, common.Address, *big.Int) {},
				BlockNumber: big.NewInt(0),
			}
			config := Config{}

			vmenv := NewEVM(vmctx, TxContext{}, statedb, params.AllBeaconProtocolChanges, config)
			var startGas = uint64(testGas)
			ret, gas, err := vmenv.Call(AccountRef(common.Address{}), address, nil, startGas, new(big.Int))
			if err != nil {
				return false
			}
			gasUsed = startGas - gas
			if len(ret) != 32 {
				t.Fatalf("test %d: expected 32 bytes returned, have %d", i, len(ret))
			}
			if bytes.Equal(ret, make([]byte, 32)) {
				// Failure
				return false
			}
			return true
		}
		minGas := sort.Search(100_000, doCheck)
		if uint64(minGas) != tt.minimumGas {
			t.Fatalf("test %d: min gas error, want %d, have %d", i, tt.minimumGas, minGas)
		}
		// If the deployment succeeded, we also check the gas used
		if minGas < 100_000 {
			if gasUsed != tt.gasUsed {
				t.Errorf("test %d: gas used mismatch: have %v, want %v", i, gasUsed, tt.gasUsed)
			}
		}
	}
}
