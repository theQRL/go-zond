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

package fourbyte

import (
	"math/big"
	"testing"

	"github.com/theQRL/go-zond/common"
	"github.com/theQRL/go-zond/common/hexutil"
	"github.com/theQRL/go-zond/signer/core/apitypes"
)

func toHexBig(h string) hexutil.Big {
	b := new(big.Int).SetBytes(common.FromHex(h))
	return hexutil.Big(*b)
}
func toHexUint(h string) hexutil.Uint64 {
	b := new(big.Int).SetBytes(common.FromHex(h))
	return hexutil.Uint64(b.Uint64())
}
func dummyTxArgs(t txtestcase) *apitypes.SendTxArgs {
	to, _ := common.NewMixedcaseAddressFromString(t.to)
	from, _ := common.NewMixedcaseAddressFromString(t.from)
	n := toHexUint(t.n)
	gas := toHexUint(t.g)
	maxFeePerGas := toHexBig(t.mfpg)
	maxPriorityFeePerGas := toHexBig(t.mpfpg)
	value := toHexBig(t.value)
	var (
		data, input *hexutil.Bytes
	)
	if t.d != "" {
		a := hexutil.Bytes(common.FromHex(t.d))
		data = &a
	}
	if t.i != "" {
		a := hexutil.Bytes(common.FromHex(t.i))
		input = &a
	}
	return &apitypes.SendTxArgs{
		From:                 *from,
		To:                   to,
		Value:                value,
		Nonce:                n,
		MaxFeePerGas:         &maxFeePerGas,
		MaxPriorityFeePerGas: &maxPriorityFeePerGas,
		Gas:                  gas,
		Data:                 data,
		Input:                input,
	}
}

type txtestcase struct {
	from, to, n, g, mfpg, mpfpg, value, d, i string
	expectErr                                bool
	numMessages                              int
}

func TestTransactionValidation(t *testing.T) {
	var (
		// use empty db, there are other tests for the abi-specific stuff
		db = newEmpty()
	)
	testcases := []txtestcase{
		// Invalid to checksum
		{from: "Z000000000000000000000000000000000000dead", to: "Z000000000000000000000000000000000000dead",
			n: "0x01", g: "0x20", mfpg: "0x40", mpfpg: "0x0", value: "0x01", numMessages: 1},
		// valid Z000000000000000000000000000000000000dEaD
		{from: "Z000000000000000000000000000000000000dead", to: "Z000000000000000000000000000000000000dEaD",
			n: "0x01", g: "0x20", mfpg: "0x40", mpfpg: "0x0", value: "0x01", numMessages: 0},
		// conflicting input and data
		{from: "Z000000000000000000000000000000000000dead", to: "Z000000000000000000000000000000000000dEaD",
			n: "0x01", g: "0x20", mfpg: "0x40", mpfpg: "0x0", value: "0x01", d: "0x01", i: "0x02", expectErr: true},
		// Data can't be parsed
		{from: "Z000000000000000000000000000000000000dead", to: "Z000000000000000000000000000000000000dEaD",
			n: "0x01", g: "0x20", mfpg: "0x40", mpfpg: "0x0", value: "0x01", d: "0x0102", numMessages: 1},
		// Data (on Input) can't be parsed
		{from: "Z000000000000000000000000000000000000dead", to: "Z000000000000000000000000000000000000dEaD",
			n: "0x01", g: "0x20", mfpg: "0x40", mpfpg: "0x0", value: "0x01", i: "0x0102", numMessages: 1},
		// Send to 0
		{from: "Z000000000000000000000000000000000000dead", to: "Z0000000000000000000000000000000000000000",
			n: "0x01", g: "0x20", mfpg: "0x40", mpfpg: "0x0", value: "0x01", numMessages: 1},
		// Create empty contract (no value)
		{from: "Z000000000000000000000000000000000000dead", to: "",
			n: "0x01", g: "0x20", mfpg: "0x40", mpfpg: "0x0", value: "0x00", numMessages: 1},
		// Create empty contract (with value)
		{from: "Z000000000000000000000000000000000000dead", to: "",
			n: "0x01", g: "0x20", mfpg: "0x40", mpfpg: "0x0", value: "0x01", expectErr: true},
		// Small payload for create
		{from: "Z000000000000000000000000000000000000dead", to: "",
			n: "0x01", g: "0x20", mfpg: "0x40", mpfpg: "0x0", value: "0x01", d: "0x01", numMessages: 1},
	}
	for i, test := range testcases {
		msgs, err := db.ValidateTransaction(nil, dummyTxArgs(test))
		if err == nil && test.expectErr {
			t.Errorf("Test %d, expected error", i)
			for _, msg := range msgs.Messages {
				t.Logf("* %s: %s", msg.Typ, msg.Message)
			}
		}
		if err != nil && !test.expectErr {
			t.Errorf("Test %d, unexpected error: %v", i, err)
		}
		if err == nil {
			got := len(msgs.Messages)
			if got != test.numMessages {
				for _, msg := range msgs.Messages {
					t.Logf("* %s: %s", msg.Typ, msg.Message)
				}
				t.Errorf("Test %d, expected %d messages, got %d", i, test.numMessages, got)
			} else {
				//Debug printout, remove later
				for _, msg := range msgs.Messages {
					t.Logf("* [%d] %s: %s", i, msg.Typ, msg.Message)
				}
				t.Log()
			}
		}
	}
}
