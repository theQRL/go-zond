// Copyright 2016 The go-ethereum Authors
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
	"runtime"
	"strings"
	"testing"

	"github.com/cespare/cp"
)

// These tests are 'smoke tests' for the account related
// subcommands and flags.
//
// For most tests, the test files from package accounts
// are copied into a temporary keystore directory.

func tmpDatadirWithKeystore(t *testing.T) string {
	datadir := t.TempDir()
	keystore := filepath.Join(datadir, "keystore")
	source := filepath.Join("..", "..", "accounts", "keystore", "testdata", "keystore")
	if err := cp.CopyAll(keystore, source); err != nil {
		t.Fatal(err)
	}
	return datadir
}

func TestAccountListEmpty(t *testing.T) {
	gzond := runGzond(t, "account", "list")
	gzond.ExpectExit()
}

func TestAccountList(t *testing.T) {
	datadir := tmpDatadirWithKeystore(t)
	var want = `
Account #0: {2099d76d9a34cdd2694c4dc703930a6fbbc1d402} keystore://{{.Datadir}}/keystore/UTC--2024-05-27T07-48-33.872599000Z--2099d76d9a34cdd2694c4dc703930a6fbbc1d402
Account #1: {205547ba6232eec096770f7161d57dea54fd13d0} keystore://{{.Datadir}}/keystore/aaa
Account #2: {206f5f53d348954856a6d2cde75ad6381945fb46} keystore://{{.Datadir}}/keystore/zzz
`
	if runtime.GOOS == "windows" {
		want = `
Account #0: {2099d76d9a34cdd2694c4dc703930a6fbbc1d402} keystore://{{.Datadir}}\keystore\UTC--2024-05-27T07-48-33.872599000Z--2099d76d9a34cdd2694c4dc703930a6fbbc1d402
Account #1: {205547ba6232eec096770f7161d57dea54fd13d0} keystore://{{.Datadir}}\keystore\aaa
Account #2: {206f5f53d348954856a6d2cde75ad6381945fb46} keystore://{{.Datadir}}\keystore\zzz
`
	}
	{
		gzond := runGzond(t, "account", "list", "--datadir", datadir)
		gzond.Expect(want)
		gzond.ExpectExit()
	}
	{
		gzond := runGzond(t, "--datadir", datadir, "account", "list")
		gzond.Expect(want)
		gzond.ExpectExit()
	}
}

func TestAccountNew(t *testing.T) {
	gzond := runGzond(t, "account", "new", "--lightkdf")
	defer gzond.ExpectExit()
	gzond.Expect(`
Your new account is locked with a password. Please give a password. Do not forget this password.
!! Unsupported terminal, password will be echoed.
Password: {{.InputLine "foobar"}}
Repeat password: {{.InputLine "foobar"}}

Your new key was generated
`)
	gzond.ExpectRegexp(`
Public address of the key:   0x[0-9a-fA-F]{40}
Path of the secret key file: .*UTC--.+--[0-9a-f]{40}

- You can share your public address with anyone. Others need it to interact with you.
- You must NEVER share the secret key with anyone! The key controls access to your funds!
- You must BACKUP your key file! Without the key, it's impossible to access account funds!
- You must REMEMBER your password! Without the password, it's impossible to decrypt the key!
`)
}

func TestAccountImport(t *testing.T) {
	tests := []struct{ name, seed, output string }{
		{
			name:   "correct account",
			seed:   "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdeffcad0b19bb29d4674531d6f115237e16",
			output: "Address: {20b0ebf635349c8167daac7d7246b8e0d892926f}\n",
		},
		{
			name:   "invalid character",
			seed:   "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdeffcad0b19bb29d4674531d6f115237e161",
			output: "Fatal: Failed to load the private key: invalid character '1' at end of key file\n",
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			importAccountWithExpect(t, test.seed, test.output)
		})
	}
}

func TestAccountHelp(t *testing.T) {
	gzond := runGzond(t, "account", "-h")
	gzond.WaitExit()
	if have, want := gzond.ExitStatus(), 0; have != want {
		t.Errorf("exit error, have %d want %d", have, want)
	}

	gzond = runGzond(t, "account", "import", "-h")
	gzond.WaitExit()
	if have, want := gzond.ExitStatus(), 0; have != want {
		t.Errorf("exit error, have %d want %d", have, want)
	}
}

func importAccountWithExpect(t *testing.T, seed string, expected string) {
	dir := t.TempDir()
	seedfile := filepath.Join(dir, "seed.txt")
	if err := os.WriteFile(seedfile, []byte(seed), 0600); err != nil {
		t.Error(err)
	}
	passwordFile := filepath.Join(dir, "password.txt")
	if err := os.WriteFile(passwordFile, []byte("foobar"), 0600); err != nil {
		t.Error(err)
	}
	gzond := runGzond(t, "--lightkdf", "account", "import", "-password", passwordFile, seedfile)
	defer gzond.ExpectExit()
	gzond.Expect(expected)
}

func TestAccountNewBadRepeat(t *testing.T) {
	gzond := runGzond(t, "account", "new", "--lightkdf")
	defer gzond.ExpectExit()
	gzond.Expect(`
Your new account is locked with a password. Please give a password. Do not forget this password.
!! Unsupported terminal, password will be echoed.
Password: {{.InputLine "something"}}
Repeat password: {{.InputLine "something else"}}
Fatal: Passwords do not match
`)
}

func TestAccountUpdate(t *testing.T) {
	datadir := tmpDatadirWithKeystore(t)
	gzond := runGzond(t, "account", "update",
		"--datadir", datadir, "--lightkdf",
		"206f5f53d348954856a6d2cde75ad6381945fb46")
	defer gzond.ExpectExit()
	gzond.Expect(`
Unlocking account 206f5f53d348954856a6d2cde75ad6381945fb46 | Attempt 1/3
!! Unsupported terminal, password will be echoed.
Password: {{.InputLine "1234567890"}}
Please give a new password. Do not forget this password.
Password: {{.InputLine "foobar2"}}
Repeat password: {{.InputLine "foobar2"}}
`)
}

func TestUnlockFlag(t *testing.T) {
	gzond := runMinimalGzond(t, "--port", "0", "--ipcdisable", "--datadir", tmpDatadirWithKeystore(t),
		"--unlock", "206f5f53d348954856a6d2cde75ad6381945fb46", "console", "--exec", "loadScript('testdata/empty.js')")
	gzond.Expect(`
Unlocking account 206f5f53d348954856a6d2cde75ad6381945fb46 | Attempt 1/3
!! Unsupported terminal, password will be echoed.
Password: {{.InputLine "1234567890"}}
undefined
`)
	gzond.ExpectExit()

	wantMessages := []string{
		"Unlocked account",
		"=0x206f5f53D348954856a6D2cDe75Ad6381945fB46",
	}
	for _, m := range wantMessages {
		if !strings.Contains(gzond.StderrText(), m) {
			t.Errorf("stderr text does not contain %q", m)
		}
	}
}

func TestUnlockFlagWrongPassword(t *testing.T) {
	gzond := runMinimalGzond(t, "--port", "0", "--ipcdisable", "--datadir", tmpDatadirWithKeystore(t),
		"--unlock", "205547ba6232eec096770f7161d57dea54fd13d0", "console", "--exec", "loadScript('testdata/empty.js')")

	defer gzond.ExpectExit()
	gzond.Expect(`
Unlocking account 205547ba6232eec096770f7161d57dea54fd13d0 | Attempt 1/3
!! Unsupported terminal, password will be echoed.
Password: {{.InputLine "wrong1"}}
Unlocking account 205547ba6232eec096770f7161d57dea54fd13d0 | Attempt 2/3
Password: {{.InputLine "wrong2"}}
Unlocking account 205547ba6232eec096770f7161d57dea54fd13d0 | Attempt 3/3
Password: {{.InputLine "wrong3"}}
Fatal: Failed to unlock account 205547ba6232eec096770f7161d57dea54fd13d0 (could not decrypt key with given password)
`)
}

func TestUnlockFlagMultiIndex(t *testing.T) {
	gzond := runMinimalGzond(t, "--port", "0", "--ipcdisable", "--datadir", tmpDatadirWithKeystore(t),
		"--unlock", "205547ba6232eec096770f7161d57dea54fd13d0", "--unlock", "0,2", "console", "--exec", "loadScript('testdata/empty.js')")

	gzond.Expect(`
Unlocking account 0 | Attempt 1/3
!! Unsupported terminal, password will be echoed.
Password: {{.InputLine "1234567890"}}
Unlocking account 2 | Attempt 1/3
Password: {{.InputLine "1234567890"}}
undefined
`)
	gzond.ExpectExit()

	wantMessages := []string{
		"Unlocked account",
		"=0x2099d76D9a34cDd2694c4DC703930A6fBbc1d402",
		"=0x206f5f53D348954856a6D2cDe75Ad6381945fB46",
	}
	for _, m := range wantMessages {
		if !strings.Contains(gzond.StderrText(), m) {
			t.Errorf("stderr text does not contain %q", m)
		}
	}
}

func TestUnlockFlagPasswordFile(t *testing.T) {
	gzond := runMinimalGzond(t, "--port", "0", "--ipcdisable", "--datadir", tmpDatadirWithKeystore(t),
		"--unlock", "205547ba6232eec096770f7161d57dea54fd13d0", "--password", "testdata/passwords.txt", "--unlock", "0,2", "console", "--exec", "loadScript('testdata/empty.js')")

	gzond.Expect(`
undefined
`)
	gzond.ExpectExit()

	wantMessages := []string{
		"Unlocked account",
		"=0x2099d76D9a34cDd2694c4DC703930A6fBbc1d402",
		"=0x206f5f53D348954856a6D2cDe75Ad6381945fB46",
	}
	for _, m := range wantMessages {
		if !strings.Contains(gzond.StderrText(), m) {
			t.Errorf("stderr text does not contain %q", m)
		}
	}
}

func TestUnlockFlagPasswordFileWrongPassword(t *testing.T) {
	gzond := runMinimalGzond(t, "--port", "0", "--ipcdisable", "--datadir", tmpDatadirWithKeystore(t),
		"--unlock", "205547ba6232eec096770f7161d57dea54fd13d0", "--password",
		"testdata/wrong-passwords.txt", "--unlock", "0,2")
	defer gzond.ExpectExit()
	gzond.Expect(`
Fatal: Failed to unlock account 0 (could not decrypt key with given password)
`)
}

func TestUnlockFlagAmbiguous(t *testing.T) {
	store := filepath.Join("..", "..", "accounts", "keystore", "testdata", "dupes")
	gzond := runMinimalGzond(t, "--port", "0", "--ipcdisable", "--datadir", tmpDatadirWithKeystore(t),
		"--unlock", "205547ba6232eec096770f7161d57dea54fd13d0", "--keystore",
		store, "--unlock", "205547ba6232eec096770f7161d57dea54fd13d0",
		"console", "--exec", "loadScript('testdata/empty.js')")
	defer gzond.ExpectExit()

	// Helper for the expect template, returns absolute keystore path.
	gzond.SetTemplateFunc("keypath", func(file string) string {
		abs, _ := filepath.Abs(filepath.Join(store, file))
		return abs
	})
	gzond.Expect(`
Unlocking account 205547ba6232eec096770f7161d57dea54fd13d0 | Attempt 1/3
!! Unsupported terminal, password will be echoed.
Password: {{.InputLine ""}}
Multiple key files exist for address 205547ba6232eec096770f7161d57dea54fd13d0:
   keystore://{{keypath "1"}}
   keystore://{{keypath "2"}}
Testing your password against all of them...
Your password unlocked keystore://{{keypath "1"}}
In order to avoid this warning, you need to remove the following duplicate key files:
   keystore://{{keypath "2"}}
undefined
`)
	gzond.ExpectExit()

	wantMessages := []string{
		"Unlocked account",
		"=0x205547bA6232eEc096770f7161d57dEA54FD13D0",
	}
	for _, m := range wantMessages {
		if !strings.Contains(gzond.StderrText(), m) {
			t.Errorf("stderr text does not contain %q", m)
		}
	}
}

func TestUnlockFlagAmbiguousWrongPassword(t *testing.T) {
	store := filepath.Join("..", "..", "accounts", "keystore", "testdata", "dupes")
	gzond := runMinimalGzond(t, "--port", "0", "--ipcdisable", "--datadir", tmpDatadirWithKeystore(t),
		"--unlock", "205547ba6232eec096770f7161d57dea54fd13d0", "--keystore",
		store, "--unlock", "205547ba6232eec096770f7161d57dea54fd13d0")

	defer gzond.ExpectExit()

	// Helper for the expect template, returns absolute keystore path.
	gzond.SetTemplateFunc("keypath", func(file string) string {
		abs, _ := filepath.Abs(filepath.Join(store, file))
		return abs
	})
	gzond.Expect(`
Unlocking account 205547ba6232eec096770f7161d57dea54fd13d0 | Attempt 1/3
!! Unsupported terminal, password will be echoed.
Password: {{.InputLine "wrong"}}
Multiple key files exist for address 205547ba6232eec096770f7161d57dea54fd13d0:
   keystore://{{keypath "1"}}
   keystore://{{keypath "2"}}
Testing your password against all of them...
Fatal: None of the listed files could be unlocked.
`)
	gzond.ExpectExit()
}
