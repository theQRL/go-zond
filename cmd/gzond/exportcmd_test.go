// Copyright 2022 The go-ethereum Authors
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

package main

import (
	"bytes"
	"fmt"
	"os"
	"testing"

	"github.com/theQRL/go-zond/common"
)

func initGzond(t *testing.T) string {
	args := []string{"--networkid=42", "init", "./testdata/genesis.json"}
	t.Logf("Initializing gzond: %v ", args)
	g := runGzond(t, args...)
	datadir := g.Datadir
	g.WaitExit()
	return datadir
}

// TestExport does a basic test of "gzond export", exporting the test-genesis.
func TestExport(t *testing.T) {
	outfile := fmt.Sprintf("%v/testExport.out", os.TempDir())
	defer os.Remove(outfile)
	gzond := runGzond(t, "--datadir", initGzond(t), "export", outfile)
	gzond.WaitExit()
	if have, want := gzond.ExitStatus(), 0; have != want {
		t.Errorf("exit error, have %d want %d", have, want)
	}
	have, err := os.ReadFile(outfile)
	if err != nil {
		t.Fatal(err)
	}
	want := common.FromHex("0xf90266f90261a00000000000000000000000000000000000000000000000000000000000000000940000000000000000000000000000000000000000a08758259b018f7bce3d2be2ddb62f325eaeea0a0c188cf96623eab468a4413e03a056e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421a056e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421b901000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000080837a12008080b875000000000000000000000000000000000000000000000000000000000000000002f0d131f1f97aef08aec6e3291b957d9efe71050000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000a00000000000000000000000000000000000000000000000000000000000000000843b9aca00a056e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421c0c0")
	if !bytes.Equal(have, want) {
		t.Fatalf("wrong content exported")
	}
}
