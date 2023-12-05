// Copyright 2022 The go-ethereum Authors
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

package rawdb

import (
	"github.com/theQRL/go-zond/core/types"
	"github.com/theQRL/go-zond/log"
	"github.com/theQRL/go-zond/rlp"
	"github.com/theQRL/go-zond/zonddb"
)

// ReadSkeletonSyncStatus retrieves the serialized sync status saved at shutdown.
func ReadSkeletonSyncStatus(db zonddb.KeyValueReader) []byte {
	data, _ := db.Get(skeletonSyncStatusKey)
	return data
}

// WriteSkeletonSyncStatus stores the serialized sync status to save at shutdown.
func WriteSkeletonSyncStatus(db zonddb.KeyValueWriter, status []byte) {
	if err := db.Put(skeletonSyncStatusKey, status); err != nil {
		log.Crit("Failed to store skeleton sync status", "err", err)
	}
}

// DeleteSkeletonSyncStatus deletes the serialized sync status saved at the last
// shutdown
func DeleteSkeletonSyncStatus(db zonddb.KeyValueWriter) {
	if err := db.Delete(skeletonSyncStatusKey); err != nil {
		log.Crit("Failed to remove skeleton sync status", "err", err)
	}
}

// ReadSkeletonHeader retrieves a block header from the skeleton sync store,
func ReadSkeletonHeader(db zonddb.KeyValueReader, number uint64) *types.Header {
	data, _ := db.Get(skeletonHeaderKey(number))
	if len(data) == 0 {
		return nil
	}
	header := new(types.Header)
	if err := rlp.DecodeBytes(data, header); err != nil {
		log.Error("Invalid skeleton header RLP", "number", number, "err", err)
		return nil
	}
	return header
}

// WriteSkeletonHeader stores a block header into the skeleton sync store.
func WriteSkeletonHeader(db zonddb.KeyValueWriter, header *types.Header) {
	data, err := rlp.EncodeToBytes(header)
	if err != nil {
		log.Crit("Failed to RLP encode header", "err", err)
	}
	key := skeletonHeaderKey(header.Number.Uint64())
	if err := db.Put(key, data); err != nil {
		log.Crit("Failed to store skeleton header", "err", err)
	}
}

// DeleteSkeletonHeader removes all block header data associated with a hash.
func DeleteSkeletonHeader(db zonddb.KeyValueWriter, number uint64) {
	if err := db.Delete(skeletonHeaderKey(number)); err != nil {
		log.Crit("Failed to delete skeleton header", "err", err)
	}
}
