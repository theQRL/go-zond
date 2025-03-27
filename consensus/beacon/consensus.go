// Copyright 2021 The go-ethereum Authors
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

package beacon

import (
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/theQRL/go-zond/common"
	"github.com/theQRL/go-zond/consensus"
	"github.com/theQRL/go-zond/consensus/misc/eip1559"
	"github.com/theQRL/go-zond/core/state"
	"github.com/theQRL/go-zond/core/types"
	"github.com/theQRL/go-zond/params"
	"github.com/theQRL/go-zond/rpc"
	"github.com/theQRL/go-zond/trie"
)

// Various error messages to mark blocks invalid. These should be private to
// prevent engine specific errors from being referenced in the remainder of the
// codebase, inherently breaking if the engine is swapped out. Please put common
// error types into the consensus package.
var (
	errInvalidTimestamp = errors.New("invalid timestamp")
)

// Beacon is a consensus engine that uses the proof-of-stake algorithm.
type Beacon struct {
	fakeFail  *uint64        // Block number which fails PoW check even in fake mode
	fakeDelay *time.Duration // Time delay to sleep for before returning from verify
	fakeFull  bool           // Accepts everything as valid
}

// New creates a consensus engine.
func New() *Beacon {
	return &Beacon{}
}

func NewFaker() *Beacon {
	return new(Beacon)
}

func NewFakeFailer(fail uint64) *Beacon {
	return &Beacon{
		fakeFail: &fail,
	}
}

func NewFakeDelayer(delay time.Duration) *Beacon {
	return &Beacon{
		fakeDelay: &delay,
	}
}

func NewFullFaker() *Beacon {
	return &Beacon{
		fakeFull: true,
	}
}

// Author implements consensus.Engine, returning the verified author of the block.
func (beacon *Beacon) Author(header *types.Header) (common.Address, error) {
	return header.Coinbase, nil
}

// VerifyHeader checks whether a header conforms to the consensus rules of the
// stock Zond consensus engine.
func (beacon *Beacon) VerifyHeader(chain consensus.ChainHeaderReader, header *types.Header) error {
	// Short circuit if the parent is not known
	parent := chain.GetHeader(header.ParentHash, header.Number.Uint64()-1)
	if parent == nil {
		return consensus.ErrUnknownAncestor
	}

	// Sanity checks passed, do a proper verification
	return beacon.verifyHeader(chain, header, parent)
}

// VerifyHeaders is similar to VerifyHeader, but verifies a batch of headers
// concurrently. The method returns a quit channel to abort the operations and
// a results channel to retrieve the async verifications.
// VerifyHeaders expect the headers to be ordered and continuous.
func (beacon *Beacon) VerifyHeaders(chain consensus.ChainHeaderReader, headers []*types.Header) (chan<- struct{}, <-chan error) {
	return beacon.verifyHeaders(chain, headers, nil)
}

// verifyHeader checks whether a header conforms to the consensus rules of the
// stock Zond consensus engine. The difference between the beacon and classic is
// (a) The following fields are expected to be constants:
//
//	to be the desired constants
//
// (b) we don't verify if a block is in the future anymore
// (c) the extradata is limited to 32 bytes
func (beacon *Beacon) verifyHeader(chain consensus.ChainHeaderReader, header, parent *types.Header) error {
	// Ensure that the header's extra-data section is of a reasonable size
	if len(header.Extra) > 32 {
		return fmt.Errorf("extra-data longer than 32 bytes (%d)", len(header.Extra))
	}
	// Verify the timestamp
	if header.Time <= parent.Time {
		return errInvalidTimestamp
	}
	// Verify that the gas limit is <= 2^63-1
	if header.GasLimit > params.MaxGasLimit {
		return fmt.Errorf("invalid gasLimit: have %v, max %v", header.GasLimit, params.MaxGasLimit)
	}
	// Verify that the gasUsed is <= gasLimit
	if header.GasUsed > header.GasLimit {
		return fmt.Errorf("invalid gasUsed: have %d, gasLimit %d", header.GasUsed, header.GasLimit)
	}
	// Verify that the block number is parent's +1
	if diff := new(big.Int).Sub(header.Number, parent.Number); diff.Cmp(common.Big1) != 0 {
		return consensus.ErrInvalidNumber
	}
	// Verify the header's EIP-1559 attributes.
	if err := eip1559.VerifyEIP1559Header(chain.Config(), parent, header); err != nil {
		return err
	}
	// Verify existence / non-existence of withdrawalsHash.
	if header.WithdrawalsHash == nil {
		return errors.New("missing withdrawalsHash")
	}

	if beacon.fakeDelay != nil {
		time.Sleep(*beacon.fakeDelay)
	}
	if beacon.fakeFail != nil && *beacon.fakeFail == header.Number.Uint64() {
		return errors.New("invalid tester pos")
	}

	return nil
}

// verifyHeaders is similar to verifyHeader, but verifies a batch of headers
// concurrently. The method returns a quit channel to abort the operations and
// a results channel to retrieve the async verifications. An additional parent
// header will be passed if the relevant header is not in the database yet.
func (beacon *Beacon) verifyHeaders(chain consensus.ChainHeaderReader, headers []*types.Header, ancestor *types.Header) (chan<- struct{}, <-chan error) {
	var (
		abort   = make(chan struct{})
		results = make(chan error, len(headers))
	)
	if beacon.fakeFull || len(headers) == 0 {
		for i := 0; i < len(headers); i++ {
			results <- nil
		}
		return abort, results
	}
	go func() {
		for i, header := range headers {
			var parent *types.Header
			if i == 0 {
				if ancestor != nil {
					parent = ancestor
				} else {
					parent = chain.GetHeader(headers[0].ParentHash, headers[0].Number.Uint64()-1)
				}
			} else if headers[i-1].Hash() == headers[i].ParentHash {
				parent = headers[i-1]
			}
			if parent == nil {
				select {
				case <-abort:
					return
				case results <- consensus.ErrUnknownAncestor:
				}
				continue
			}
			err := beacon.verifyHeader(chain, header, parent)
			select {
			case <-abort:
				return
			case results <- err:
			}
		}
	}()
	return abort, results
}

// Finalize implements consensus.Engine and processes withdrawals on top.
func (beacon *Beacon) Finalize(chain consensus.ChainHeaderReader, header *types.Header, state *state.StateDB, body *types.Body) {
	// Withdrawals processing.
	for _, w := range body.Withdrawals {
		// Convert amount from gwei to wei.
		amount := new(big.Int).SetUint64(w.Amount)
		amount = amount.Mul(amount, big.NewInt(params.GWei))
		state.AddBalance(w.Address, amount)
	}
	// No block reward which is issued by consensus layer instead.
}

// FinalizeAndAssemble implements consensus.Engine, setting the final state and
// assembling the block.
func (beacon *Beacon) FinalizeAndAssemble(chain consensus.ChainHeaderReader, header *types.Header, state *state.StateDB, body *types.Body, receipts []*types.Receipt) (*types.Block, error) {
	// All blocks after Shanghai must include a withdrawals root.
	if body.Withdrawals == nil {
		body.Withdrawals = make([]*types.Withdrawal, 0)
	}
	// Finalize and assemble the block.
	beacon.Finalize(chain, header, state, body)

	// Assign the final state root to header.
	header.Root = state.IntermediateRoot(true)

	// Assemble and return the final block.
	return types.NewBlock(header, body, receipts, trie.NewStackTrie(nil)), nil
}

// APIs implements consensus.Engine, returning the user facing RPC APIs.
func (beacon *Beacon) APIs(chain consensus.ChainHeaderReader) []rpc.API {
	return []rpc.API{}
}

// Close shutdowns the consensus engine
func (beacon *Beacon) Close() error {
	return nil
}
