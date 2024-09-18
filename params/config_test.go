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

package params

import (
	"reflect"
	"testing"
	"time"

	"github.com/theQRL/go-zond/common"
)

func TestCheckCompatible(t *testing.T) {
	type test struct {
		stored, new   *ChainConfig
		headBlock     uint64
		headTimestamp uint64
		wantErr       *ConfigCompatError
	}
	tests := []test{
		{stored: AllBeaconProtocolChanges, new: AllBeaconProtocolChanges, headBlock: 0, headTimestamp: 0, wantErr: nil},
		{stored: AllBeaconProtocolChanges, new: AllBeaconProtocolChanges, headBlock: 0, headTimestamp: uint64(time.Now().Unix()), wantErr: nil},
		{stored: AllBeaconProtocolChanges, new: AllBeaconProtocolChanges, headBlock: 100, wantErr: nil},
		{
			stored: &ChainConfig{},
			new:    &ChainConfig{},
			// headBlock: 9,
			wantErr: nil,
		},
		{
			stored: &ChainConfig{ChainID: common.Big1},
			new:    &ChainConfig{ChainID: common.Big32},
			wantErr: &ConfigCompatError{
				What:        "chain ID",
				StoredBlock: common.Big1,
				NewBlock:    common.Big32,
			},
		},
		// NOTE(rgeraldes24): not valid at the moment
		/*
			{
				stored:    AllBeaconProtocolChanges,
				new:       &ChainConfig{},
				headBlock: 3,
				wantErr: &ConfigCompatError{
					What:          "Homestead fork block",
					StoredBlock:   big.NewInt(0),
					NewBlock:      nil,
					RewindToBlock: 0,
				},
			},
			{
				stored:    AllBeaconProtocolChanges,
				new:       &ChainConfig{},
				headBlock: 3,
				wantErr: &ConfigCompatError{
					What:          "Homestead fork block",
					StoredBlock:   big.NewInt(0),
					NewBlock:      big.NewInt(1),
					RewindToBlock: 0,
				},
			},
			{
				stored:    &ChainConfig{},
				new:       &ChainConfig{},
				headBlock: 25,
				wantErr: &ConfigCompatError{
					What:          "EIP150 fork block",
					StoredBlock:   big.NewInt(10),
					NewBlock:      big.NewInt(20),
					RewindToBlock: 9,
				},
			},
			{
				stored:    &ChainConfig{},
				new:       &ChainConfig{},
				headBlock: 40,
				wantErr:   nil,
			},
			{
				stored:    &ChainConfig{},
				new:       &ChainConfig{},
				headBlock: 40,
				wantErr: &ConfigCompatError{
					What:          "Petersburg fork block",
					StoredBlock:   nil,
					NewBlock:      big.NewInt(31),
					RewindToBlock: 30,
				},
			},
			{
				stored:        &ChainConfig{},
				new:           &ChainConfig{},
				headTimestamp: 9,
				wantErr:       nil,
			},
			{
				stored:        &ChainConfig{},
				new:           &ChainConfig{},
				headTimestamp: 25,
				wantErr: &ConfigCompatError{
					What:         "Shanghai fork timestamp",
					StoredTime:   newUint64(10),
					NewTime:      newUint64(20),
					RewindToTime: 9,
				},
			},
		*/
	}

	for _, test := range tests {
		err := test.stored.CheckCompatible(test.new, test.headBlock, test.headTimestamp)
		if !reflect.DeepEqual(err, test.wantErr) {
			t.Errorf("error mismatch:\nstored: %v\nnew: %v\nheadBlock: %v\nheadTimestamp: %v\nerr: %v\nwant: %v", test.stored, test.new, test.headBlock, test.headTimestamp, err, test.wantErr)
		}
	}
}

// NOTE(rgeraldes24): not valid at the moment
/*
func TestConfigRules(t *testing.T) {
	c := &ChainConfig{
		ShanghaiTime: newUint64(500),
	}
	var stamp uint64
	if r := c.Rules(big.NewInt(0), true, stamp); r.IsShanghai {
		t.Errorf("expected %v to not be shanghai", stamp)
	}
	stamp = 500
	if r := c.Rules(big.NewInt(0), true, stamp); !r.IsShanghai {
		t.Errorf("expected %v to be shanghai", stamp)
	}
	stamp = math.MaxInt64
	if r := c.Rules(big.NewInt(0), true, stamp); !r.IsShanghai {
		t.Errorf("expected %v to be shanghai", stamp)
	}
}
*/
