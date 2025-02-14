// Copyright 2018 The go-ethereum Authors
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
	"os"
	"path/filepath"
	"testing"

	"github.com/theQRL/go-zond/accounts/keystore"
	"github.com/theQRL/go-zond/cmd/utils"
	"github.com/theQRL/go-zond/common"
)

func TestMessageSignVerify(t *testing.T) {
	tmpdir := t.TempDir()

	keyfile := filepath.Join(tmpdir, "the-keyfile")
	message := "test message"

	// Create the key.
	generate := runZondkey(t, "generate", "--lightkdf", keyfile)
	generate.Expect(`
!! Unsupported terminal, password will be echoed.
Password: {{.InputLine "foobar"}}
Repeat password: {{.InputLine "foobar"}}
`)
	generate.ExpectRegexp(`Address: (Z[0-9a-fA-F]{40})\n`)
	generate.ExpectExit()

	// Sign a message.
	sign := runZondkey(t, "signmessage", keyfile, message)
	sign.Expect(`
!! Unsupported terminal, password will be echoed.
Password: {{.InputLine "foobar"}}
`)
	_, matches := sign.ExpectRegexp(`Signature: ([0-9a-f]+)\n`)
	signature := matches[1]
	sign.ExpectExit()

	// Read key from file.
	keyjson, err := os.ReadFile(keyfile)
	if err != nil {
		utils.Fatalf("Failed to read the keyfile at '%s': %v", keyfile, err)
	}

	// Decrypt key with passphrase.
	key, err := keystore.DecryptKey(keyjson, "foobar")
	if err != nil {
		utils.Fatalf("Error decrypting key: %v", err)
	}

	// Verify the message.
	publicKey := key.Dilithium.GetPK()
	verify := runZondkey(t, "verifymessage", signature, common.Bytes2Hex(publicKey[:]), message)
	verify.ExpectRegexp(`
Signature verification successful!
`)
	verify.ExpectExit()
}
