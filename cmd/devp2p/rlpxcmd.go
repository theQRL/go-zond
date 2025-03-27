// Copyright 2020 The go-ethereum Authors
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
	"errors"
	"fmt"
	"net"

	"github.com/theQRL/go-zond/crypto"
	"github.com/theQRL/go-zond/p2p"
	"github.com/theQRL/go-zond/p2p/rlpx"
	"github.com/theQRL/go-zond/rlp"
	"github.com/urfave/cli/v2"
)

var (
	rlpxCommand = &cli.Command{
		Name:  "rlpx",
		Usage: "RLPx Commands",
		Subcommands: []*cli.Command{
			rlpxPingCommand,
			rlpxZondTestCommand,
			rlpxSnapTestCommand,
		},
	}
	rlpxPingCommand = &cli.Command{
		Name:   "ping",
		Usage:  "ping <node>",
		Action: rlpxPing,
	}
	rlpxZondTestCommand = &cli.Command{
		Name:      "zond-test",
		Usage:     "Runs tests against a node",
		ArgsUsage: "<node> <chain.rlp> <genesis.json>",
		Action:    rlpxZondTest,
		Flags: []cli.Flag{
			testPatternFlag,
			testTAPFlag,
		},
	}
	rlpxSnapTestCommand = &cli.Command{
		Name:      "snap-test",
		Usage:     "Runs tests against a node",
		ArgsUsage: "<node> <chain.rlp> <genesis.json>",
		Action:    rlpxSnapTest,
		Flags: []cli.Flag{
			testPatternFlag,
			testTAPFlag,
		},
	}
)

func rlpxPing(ctx *cli.Context) error {
	n := getNodeArg(ctx)
	fd, err := net.Dial("tcp", fmt.Sprintf("%v:%d", n.IP(), n.TCP()))
	if err != nil {
		return err
	}
	conn := rlpx.NewConn(fd, n.Pubkey())
	ourKey, _ := crypto.GenerateKey()
	_, err = conn.Handshake(ourKey)
	if err != nil {
		return err
	}
	code, data, _, err := conn.Read()
	if err != nil {
		return err
	}
	switch code {
	case 0:
		// TODO(now.youtrack.cloud/issue/TGZ-6)
		// var h zondtest.Hello
		// if err := rlp.DecodeBytes(data, &h); err != nil {
		// 	return fmt.Errorf("invalid handshake: %v", err)
		// }
		// fmt.Printf("%+v\n", h)
	case 1:
		var msg []p2p.DiscReason
		if rlp.DecodeBytes(data, &msg); len(msg) == 0 {
			return errors.New("invalid disconnect message")
		}
		return fmt.Errorf("received disconnect message: %v", msg[0])
	default:
		return fmt.Errorf("invalid message code %d, expected handshake (code zero)", code)
	}
	return nil
}

// rlpxZondTest runs the zond protocol test suite.
func rlpxZondTest(ctx *cli.Context) error {
	if ctx.NArg() < 3 {
		exit("missing path to chain.rlp as command-line argument")
	}
	// TODO(now.youtrack.cloud/issue/TGZ-6)
	// suite, err := zondtest.NewSuite(getNodeArg(ctx), ctx.Args().Get(1), ctx.Args().Get(2))
	// if err != nil {
	// 	exit(err)
	// }
	// return runTests(ctx, suite.ZondTests())
	return nil
}

// rlpxSnapTest runs the snap protocol test suite.
func rlpxSnapTest(ctx *cli.Context) error {
	if ctx.NArg() < 3 {
		exit("missing path to chain.rlp as command-line argument")
	}
	// TODO(now.youtrack.cloud/issue/TGZ-6)
	// suite, err := zondtest.NewSuite(getNodeArg(ctx), ctx.Args().Get(1), ctx.Args().Get(2))
	// if err != nil {
	// 	exit(err)
	// }
	// return runTests(ctx, suite.SnapTests())
	return nil
}
