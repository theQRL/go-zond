// Copyright 2015 The go-ethereum Authors
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

package zond

import (
	"github.com/theQRL/go-zond/common"
	"github.com/theQRL/go-zond/common/hexutil"
)

// ZondAPI provides an API to access Zond full node-related information.
type ZondAPI struct {
	z *Zond
}

// NewZondAPI creates a new Zond protocol API for full nodes.
func NewZondAPI(z *Zond) *ZondAPI {
	return &ZondAPI{z}
}

// Etherbase is the address that mining rewards will be sent to.
func (api *ZondAPI) Etherbase() (common.Address, error) {
	return api.z.Etherbase()
}

// Coinbase is the address that mining rewards will be sent to (alias for Etherbase).
func (api *ZondAPI) Coinbase() (common.Address, error) {
	return api.Etherbase()
}

// Hashrate returns the POW hashrate.
func (api *ZondAPI) Hashrate() hexutil.Uint64 {
	return hexutil.Uint64(api.z.Miner().Hashrate())
}

// Mining returns an indication if this node is currently mining.
func (api *ZondAPI) Mining() bool {
	return api.z.IsMining()
}
