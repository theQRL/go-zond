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

// Package catalyst implements the temporary eth1/eth2 RPC integration.
package catalyst

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/theQRL/go-zond/beacon/engine"
	"github.com/theQRL/go-zond/common"
	"github.com/theQRL/go-zond/common/hexutil"
	"github.com/theQRL/go-zond/core/rawdb"
	"github.com/theQRL/go-zond/core/types"
	"github.com/theQRL/go-zond/log"
	"github.com/theQRL/go-zond/miner"
	"github.com/theQRL/go-zond/node"
	"github.com/theQRL/go-zond/rpc"
	"github.com/theQRL/go-zond/zond"
	"github.com/theQRL/go-zond/zond/downloader"
)

// Register adds the engine API to the full node.
func Register(stack *node.Node, backend *zond.Zond) error {
	log.Warn("Engine API enabled", "protocol", "zond")
	stack.RegisterAPIs([]rpc.API{
		{
			Namespace:     "engine",
			Service:       NewConsensusAPI(backend),
			Authenticated: true,
		},
	})
	return nil
}

const (
	// invalidBlockHitEviction is the number of times an invalid block can be
	// referenced in forkchoice update or new payload before it is attempted
	// to be reprocessed again.
	invalidBlockHitEviction = 128

	// invalidTipsetsCap is the max number of recent block hashes tracked that
	// have lead to some bad ancestor block. It's just an OOM protection.
	invalidTipsetsCap = 512

	// beaconUpdateStartupTimeout is the time to wait for a beacon client to get
	// attached before starting to issue warnings.
	beaconUpdateStartupTimeout = 30 * time.Second

	// beaconUpdateConsensusTimeout is the max time allowed for a beacon client
	// to send a consensus update before it's considered offline and the user is
	// warned.
	beaconUpdateConsensusTimeout = 2 * time.Minute

	// beaconUpdateWarnFrequency is the frequency at which to warn the user that
	// the beacon client is offline.
	beaconUpdateWarnFrequency = 5 * time.Minute
)

// All methods provided over the engine endpoint.
var caps = []string{
	"engine_forkchoiceUpdatedV2",
	"engine_getPayloadV2",
	"engine_newPayloadV2",
	"engine_getPayloadBodiesByHashV1",
	"engine_getPayloadBodiesByRangeV1",
}

type ConsensusAPI struct {
	zond *zond.Zond

	remoteBlocks *headerQueue  // Cache of remote payloads received
	localBlocks  *payloadQueue // Cache of local payloads generated

	// The forkchoice update and new payload method require us to return the
	// latest valid hash in an invalid chain. To support that return, we need
	// to track historical bad blocks as well as bad tipsets in case a chain
	// is constantly built on it.
	//
	// There are a few important caveats in this mechanism:
	//   - The bad block tracking is ephemeral, in-memory only. We must never
	//     persist any bad block information to disk as a bug in Gzond could end
	//     up blocking a valid chain, even if a later Gzond update would accept
	//     it.
	//   - Bad blocks will get forgotten after a certain threshold of import
	//     attempts and will be retried. The rationale is that if the network
	//     really-really-really tries to feed us a block, we should give it a
	//     new chance, perhaps us being racey instead of the block being legit
	//     bad (this happened in Gzond at a point with import vs. pending race).
	//   - Tracking all the blocks built on top of the bad one could be a bit
	//     problematic, so we will only track the head chain segment of a bad
	//     chain to allow discarding progressing bad chains and side chains,
	//     without tracking too much bad data.
	invalidBlocksHits map[common.Hash]int           // Ephemeral cache to track invalid blocks and their hit count
	invalidTipsets    map[common.Hash]*types.Header // Ephemeral cache to track invalid tipsets and their bad ancestor
	invalidLock       sync.Mutex                    // Protects the invalid maps from concurrent access

	// Gzond can appear to be stuck or do strange things if the beacon client is
	// offline or is sending us strange data. Stash some update stats away so
	// that we can warn the user and not have them open issues on our tracker.
	lastTransitionUpdate time.Time
	lastTransitionLock   sync.Mutex
	lastForkchoiceUpdate time.Time
	lastForkchoiceLock   sync.Mutex
	lastNewPayloadUpdate time.Time
	lastNewPayloadLock   sync.Mutex

	forkchoiceLock sync.Mutex // Lock for the forkChoiceUpdated method
	newPayloadLock sync.Mutex // Lock for the NewPayload method
}

// NewConsensusAPI creates a new consensus api for the given backend.
// The underlying blockchain needs to have a valid terminal total difficulty set.
func NewConsensusAPI(zond *zond.Zond) *ConsensusAPI {
	api := newConsensusAPIWithoutHeartbeat(zond)
	go api.heartbeat()
	return api
}

// newConsensusAPIWithoutHeartbeat creates a new consensus api for the SimulatedBeacon Node.
func newConsensusAPIWithoutHeartbeat(zond *zond.Zond) *ConsensusAPI {
	api := &ConsensusAPI{
		zond:              zond,
		remoteBlocks:      newHeaderQueue(),
		localBlocks:       newPayloadQueue(),
		invalidBlocksHits: make(map[common.Hash]int),
		invalidTipsets:    make(map[common.Hash]*types.Header),
	}
	zond.Downloader().SetBadBlockCallback(api.setInvalidAncestor)
	return api
}

// ForkchoiceUpdatedV2 has several responsibilities:
//
// We try to set our blockchain to the headBlock.
//
// If the method is called with an empty head block: we return success, which can be used
// to check if the engine API is enabled.
//
// If the finalizedBlockHash is set: we check if we have the finalizedBlockHash in our db,
// if not we start a sync.
//
// If there are payloadAttributes: we try to assemble a block with the payloadAttributes
// and return its payloadID.
// ForkchoiceUpdatedV2 is equivalent to V1 with the addition of withdrawals in the payload attributes.
func (api *ConsensusAPI) ForkchoiceUpdatedV2(update engine.ForkchoiceStateV1, params *engine.PayloadAttributes) (engine.ForkChoiceResponse, error) {
	if params != nil && params.Withdrawals == nil {
		return engine.STATUS_INVALID, engine.InvalidPayloadAttributes.With(errors.New("missing withdrawals"))
	}
	return api.forkchoiceUpdated(update, params, false)
}

func (api *ConsensusAPI) forkchoiceUpdated(update engine.ForkchoiceStateV1, payloadAttributes *engine.PayloadAttributes, simulatorMode bool) (engine.ForkChoiceResponse, error) {
	api.forkchoiceLock.Lock()
	defer api.forkchoiceLock.Unlock()

	log.Trace("Engine API request received", "method", "ForkchoiceUpdated", "head", update.HeadBlockHash, "finalized", update.FinalizedBlockHash, "safe", update.SafeBlockHash)
	if update.HeadBlockHash == (common.Hash{}) {
		log.Warn("Forkchoice requested update to zero hash")
		return engine.STATUS_INVALID, nil // TODO(karalabe): Why does someone send us this?
	}
	// Stash away the last update to warn the user if the beacon client goes offline
	api.lastForkchoiceLock.Lock()
	api.lastForkchoiceUpdate = time.Now()
	api.lastForkchoiceLock.Unlock()

	// Check whether we have the block yet in our database or not. If not, we'll
	// need to either trigger a sync, or to reject this forkchoice update for a
	// reason.
	block := api.zond.BlockChain().GetBlockByHash(update.HeadBlockHash)
	if block == nil {
		// If this block was previously invalidated, keep rejecting it here too
		if res := api.checkInvalidAncestor(update.HeadBlockHash, update.HeadBlockHash); res != nil {
			return engine.ForkChoiceResponse{PayloadStatus: *res, PayloadID: nil}, nil
		}
		// If the head hash is unknown (was not given to us in a newPayload request),
		// we cannot resolve the header, so not much to do. This could be extended in
		// the future to resolve from the `zond` network, but it's an unexpected case
		// that should be fixed, not papered over.
		header := api.remoteBlocks.get(update.HeadBlockHash)
		if header == nil {
			log.Warn("Forkchoice requested unknown head", "hash", update.HeadBlockHash)
			return engine.STATUS_SYNCING, nil
		}
		// If the finalized hash is known, we can direct the downloader to move
		// potentially more data to the freezer from the get go.
		finalized := api.remoteBlocks.get(update.FinalizedBlockHash)

		// Header advertised via a past newPayload request. Start syncing to it.
		context := []interface{}{"number", header.Number, "hash", header.Hash()}
		if update.FinalizedBlockHash != (common.Hash{}) {
			if finalized == nil {
				context = append(context, []interface{}{"finalized", "unknown"}...)
			} else {
				context = append(context, []interface{}{"finalized", finalized.Number}...)
			}
		}
		log.Info("Forkchoice requested sync to new head", context...)
		if err := api.zond.Downloader().BeaconSync(api.zond.SyncMode(), header, finalized); err != nil {
			return engine.STATUS_SYNCING, err
		}
		return engine.STATUS_SYNCING, nil
	}

	valid := func(id *engine.PayloadID) engine.ForkChoiceResponse {
		return engine.ForkChoiceResponse{
			PayloadStatus: engine.PayloadStatusV1{Status: engine.VALID, LatestValidHash: &update.HeadBlockHash},
			PayloadID:     id,
		}
	}
	if rawdb.ReadCanonicalHash(api.zond.ChainDb(), block.NumberU64()) != update.HeadBlockHash {
		// Block is not canonical, set head.
		if latestValid, err := api.zond.BlockChain().SetCanonical(block); err != nil {
			return engine.ForkChoiceResponse{PayloadStatus: engine.PayloadStatusV1{Status: engine.INVALID, LatestValidHash: &latestValid}}, err
		}
	} else if api.zond.BlockChain().CurrentBlock().Hash() == update.HeadBlockHash {
		// If the specified head matches with our local head, do nothing and keep
		// generating the payload. It's a special corner case that a few slots are
		// missing and we are requested to generate the payload in slot.
	} else {
		// If the head block is already in our canonical chain, the beacon client is
		// probably resyncing. Ignore the update.
		log.Info("Ignoring beacon update to old head", "number", block.NumberU64(), "hash", update.HeadBlockHash, "age", common.PrettyAge(time.Unix(int64(block.Time()), 0)), "have", api.zond.BlockChain().CurrentBlock().Number)
		return valid(nil), nil
	}
	api.zond.SetSynced()

	// If the beacon client also advertised a finalized block, mark the local
	// chain final and completely in PoS mode.
	if update.FinalizedBlockHash != (common.Hash{}) {
		// If the finalized block is not in our canonical tree, something is wrong
		finalBlock := api.zond.BlockChain().GetBlockByHash(update.FinalizedBlockHash)
		if finalBlock == nil {
			log.Warn("Final block not available in database", "hash", update.FinalizedBlockHash)
			return engine.STATUS_INVALID, engine.InvalidForkChoiceState.With(errors.New("final block not available in database"))
		} else if rawdb.ReadCanonicalHash(api.zond.ChainDb(), finalBlock.NumberU64()) != update.FinalizedBlockHash {
			log.Warn("Final block not in canonical chain", "number", block.NumberU64(), "hash", update.HeadBlockHash)
			return engine.STATUS_INVALID, engine.InvalidForkChoiceState.With(errors.New("final block not in canonical chain"))
		}
		// Set the finalized block
		api.zond.BlockChain().SetFinalized(finalBlock.Header())
	}
	// Check if the safe block hash is in our canonical tree, if not something is wrong
	if update.SafeBlockHash != (common.Hash{}) {
		safeBlock := api.zond.BlockChain().GetBlockByHash(update.SafeBlockHash)
		if safeBlock == nil {
			log.Warn("Safe block not available in database")
			return engine.STATUS_INVALID, engine.InvalidForkChoiceState.With(errors.New("safe block not available in database"))
		}
		if rawdb.ReadCanonicalHash(api.zond.ChainDb(), safeBlock.NumberU64()) != update.SafeBlockHash {
			log.Warn("Safe block not in canonical chain")
			return engine.STATUS_INVALID, engine.InvalidForkChoiceState.With(errors.New("safe block not in canonical chain"))
		}
		// Set the safe block
		api.zond.BlockChain().SetSafe(safeBlock.Header())
	}
	// If payload generation was requested, create a new block to be potentially
	// sealed by the beacon client. The payload will be requested later, and we
	// will replace it arbitrarily many times in between.
	if payloadAttributes != nil {
		args := &miner.BuildPayloadArgs{
			Parent:       update.HeadBlockHash,
			Timestamp:    payloadAttributes.Timestamp,
			FeeRecipient: payloadAttributes.SuggestedFeeRecipient,
			Random:       payloadAttributes.Random,
			Withdrawals:  payloadAttributes.Withdrawals,
		}
		id := args.Id()
		// If we already are busy generating this work, then we do not need
		// to start a second process.
		if api.localBlocks.has(id) {
			return valid(&id), nil
		}
		// If the beacon chain is ran by a simulator, then transaction insertion,
		// block insertion and block production will happen without any timing
		// delay between them. This will cause flaky simulator executions due to
		// the transaction pool running its internal reset operation on a back-
		// ground thread. To avoid the racey behavior - in simulator mode - the
		// pool will be explicitly blocked on its reset before continuing to the
		// block production below.
		if simulatorMode {
			if err := api.zond.TxPool().Sync(); err != nil {
				log.Error("Failed to sync transaction pool", "err", err)
				return valid(nil), engine.InvalidPayloadAttributes.With(err)
			}
		}
		payload, err := api.zond.Miner().BuildPayload(args)
		if err != nil {
			log.Error("Failed to build payload", "err", err)
			return valid(nil), engine.InvalidPayloadAttributes.With(err)
		}
		api.localBlocks.put(id, payload)
		return valid(&id), nil
	}
	return valid(nil), nil
}

// GetPayloadV2 returns a cached payload by id.
func (api *ConsensusAPI) GetPayloadV2(payloadID engine.PayloadID) (*engine.ExecutionPayloadEnvelope, error) {
	return api.getPayload(payloadID, false)
}

func (api *ConsensusAPI) getPayload(payloadID engine.PayloadID, full bool) (*engine.ExecutionPayloadEnvelope, error) {
	log.Trace("Engine API request received", "method", "GetPayload", "id", payloadID)
	data := api.localBlocks.get(payloadID, full)
	if data == nil {
		return nil, engine.UnknownPayload
	}
	return data, nil
}

// NewPayloadV2 creates a Zond execution block, inserts it in the chain, and returns the status of the chain.
func (api *ConsensusAPI) NewPayloadV2(params engine.ExecutableData) (engine.PayloadStatusV1, error) {
	if params.Withdrawals == nil {
		return engine.PayloadStatusV1{Status: engine.INVALID}, engine.InvalidParams.With(errors.New("nil withdrawals post-shanghai"))
	}

	return api.newPayload(params)
}

func (api *ConsensusAPI) newPayload(params engine.ExecutableData) (engine.PayloadStatusV1, error) {
	// The locking here is, strictly, not required. Without these locks, this can happen:
	//
	// 1. NewPayload( execdata-N ) is invoked from the CL. It goes all the way down to
	//      api.eth.BlockChain().InsertBlockWithoutSetHead, where it is blocked on
	//      e.g database compaction.
	// 2. The call times out on the CL layer, which issues another NewPayload (execdata-N) call.
	//    Similarly, this also get stuck on the same place. Importantly, since the
	//    first call has not gone through, the early checks for "do we already have this block"
	//    will all return false.
	// 3. When the db compaction ends, then N calls inserting the same payload are processed
	//    sequentially.
	// Hence, we use a lock here, to be sure that the previous call has finished before we
	// check whether we already have the block locally.
	api.newPayloadLock.Lock()
	defer api.newPayloadLock.Unlock()

	log.Trace("Engine API request received", "method", "NewPayload", "number", params.Number, "hash", params.BlockHash)
	block, err := engine.ExecutableDataToBlock(params)
	if err != nil {
		log.Warn("Invalid NewPayload params",
			"params.Number", params.Number,
			"params.ParentHash", params.ParentHash,
			"params.BlockHash", params.BlockHash,
			"params.StateRoot", params.StateRoot,
			"params.FeeRecipient", params.FeeRecipient,
			"params.LogsBloom", common.PrettyBytes(params.LogsBloom),
			"params.Random", params.Random,
			"params.GasLimit", params.GasLimit,
			"params.GasUsed", params.GasUsed,
			"params.Timestamp", params.Timestamp,
			"params.ExtraData", common.PrettyBytes(params.ExtraData),
			"params.BaseFeePerGas", params.BaseFeePerGas,
			"len(params.Transactions)", len(params.Transactions),
			"len(params.Withdrawals)", len(params.Withdrawals),
			"error", err)
		return api.invalid(err, nil), nil
	}
	// Stash away the last update to warn the user if the beacon client goes offline
	api.lastNewPayloadLock.Lock()
	api.lastNewPayloadUpdate = time.Now()
	api.lastNewPayloadLock.Unlock()

	// If we already have the block locally, ignore the entire execution and just
	// return a fake success.
	if block := api.zond.BlockChain().GetBlockByHash(params.BlockHash); block != nil {
		log.Warn("Ignoring already known beacon payload", "number", params.Number, "hash", params.BlockHash, "age", common.PrettyAge(time.Unix(int64(block.Time()), 0)))
		hash := block.Hash()
		return engine.PayloadStatusV1{Status: engine.VALID, LatestValidHash: &hash}, nil
	}
	// If this block was rejected previously, keep rejecting it
	if res := api.checkInvalidAncestor(block.Hash(), block.Hash()); res != nil {
		return *res, nil
	}
	// If the parent is missing, we - in theory - could trigger a sync, but that
	// would also entail a reorg. That is problematic if multiple sibling blocks
	// are being fed to us, and even more so, if some semi-distant uncle shortens
	// our live chain. As such, payload execution will not permit reorgs and thus
	// will not trigger a sync cycle. That is fine though, if we get a fork choice
	// update after legit payload executions.
	parent := api.zond.BlockChain().GetBlock(block.ParentHash(), block.NumberU64()-1)
	if parent == nil {
		return api.delayPayloadImport(block), nil
	}
	// We have an existing parent, do some sanity checks to avoid the beacon client
	// triggering too early
	if block.Time() <= parent.Time() {
		log.Warn("Invalid timestamp", "parent", block.Time(), "block", block.Time())
		return api.invalid(errors.New("invalid timestamp"), parent.Header()), nil
	}
	// Another cornercase: if the node is in snap sync mode, but the CL client
	// tries to make it import a block. That should be denied as pushing something
	// into the database directly will conflict with the assumptions of snap sync
	// that it has an empty db that it can fill itself.
	if api.zond.SyncMode() != downloader.FullSync {
		return api.delayPayloadImport(block), nil
	}
	if !api.zond.BlockChain().HasBlockAndState(block.ParentHash(), block.NumberU64()-1) {
		api.remoteBlocks.put(block.Hash(), block.Header())
		log.Warn("State not available, ignoring new payload")
		return engine.PayloadStatusV1{Status: engine.ACCEPTED}, nil
	}
	log.Trace("Inserting block without sethead", "hash", block.Hash(), "number", block.Number)
	if err := api.zond.BlockChain().InsertBlockWithoutSetHead(block); err != nil {
		log.Warn("NewPayloadV1: inserting block failed", "error", err)

		api.invalidLock.Lock()
		api.invalidBlocksHits[block.Hash()] = 1
		api.invalidTipsets[block.Hash()] = block.Header()
		api.invalidLock.Unlock()

		return api.invalid(err, parent.Header()), nil
	}
	hash := block.Hash()
	return engine.PayloadStatusV1{Status: engine.VALID, LatestValidHash: &hash}, nil
}

// delayPayloadImport stashes the given block away for import at a later time,
// either via a forkchoice update or a sync extension. This method is meant to
// be called by the newpayload command when the block seems to be ok, but some
// prerequisite prevents it from being processed (e.g. no parent, or snap sync).
func (api *ConsensusAPI) delayPayloadImport(block *types.Block) engine.PayloadStatusV1 {
	// Sanity check that this block's parent is not on a previously invalidated
	// chain. If it is, mark the block as invalid too.
	if res := api.checkInvalidAncestor(block.ParentHash(), block.Hash()); res != nil {
		return *res
	}
	// Stash the block away for a potential forced forkchoice update to it
	// at a later time.
	api.remoteBlocks.put(block.Hash(), block.Header())

	// Although we don't want to trigger a sync, if there is one already in
	// progress, try to extend it with the current payload request to relieve
	// some strain from the forkchoice update.
	err := api.zond.Downloader().BeaconExtend(api.zond.SyncMode(), block.Header())
	if err == nil {
		log.Debug("Payload accepted for sync extension", "number", block.NumberU64(), "hash", block.Hash())
		return engine.PayloadStatusV1{Status: engine.SYNCING}
	}
	// Either no beacon sync was started yet, or it rejected the delivered
	// payload as non-integratable on top of the existing sync. We'll just
	// have to rely on the beacon client to forcefully update the head with
	// a forkchoice update request.
	if api.zond.SyncMode() == downloader.FullSync {
		// In full sync mode, failure to import a well-formed block can only mean
		// that the parent state is missing and the syncer rejected extending the
		// current cycle with the new payload.
		log.Warn("Ignoring payload with missing parent", "number", block.NumberU64(), "hash", block.Hash(), "parent", block.ParentHash(), "reason", err)
	} else {
		// In non-full sync mode (i.e. snap sync) all payloads are rejected until
		// snap sync terminates as snap sync relies on direct database injections
		// and cannot afford concurrent out-if-band modifications via imports.
		log.Warn("Ignoring payload while snap syncing", "number", block.NumberU64(), "hash", block.Hash(), "reason", err)
	}
	return engine.PayloadStatusV1{Status: engine.SYNCING}
}

// setInvalidAncestor is a callback for the downloader to notify us if a bad block
// is encountered during the async sync.
func (api *ConsensusAPI) setInvalidAncestor(invalid *types.Header, origin *types.Header) {
	api.invalidLock.Lock()
	defer api.invalidLock.Unlock()

	api.invalidTipsets[origin.Hash()] = invalid
	api.invalidBlocksHits[invalid.Hash()]++
}

// checkInvalidAncestor checks whether the specified chain end links to a known
// bad ancestor. If yes, it constructs the payload failure response to return.
func (api *ConsensusAPI) checkInvalidAncestor(check common.Hash, head common.Hash) *engine.PayloadStatusV1 {
	api.invalidLock.Lock()
	defer api.invalidLock.Unlock()

	// If the hash to check is unknown, return valid
	invalid, ok := api.invalidTipsets[check]
	if !ok {
		return nil
	}
	// If the bad hash was hit too many times, evict it and try to reprocess in
	// the hopes that we have a data race that we can exit out of.
	badHash := invalid.Hash()

	api.invalidBlocksHits[badHash]++
	if api.invalidBlocksHits[badHash] >= invalidBlockHitEviction {
		log.Warn("Too many bad block import attempt, trying", "number", invalid.Number, "hash", badHash)
		delete(api.invalidBlocksHits, badHash)

		for descendant, badHeader := range api.invalidTipsets {
			if badHeader.Hash() == badHash {
				delete(api.invalidTipsets, descendant)
			}
		}
		return nil
	}
	// Not too many failures yet, mark the head of the invalid chain as invalid
	if check != head {
		log.Warn("Marked new chain head as invalid", "hash", head, "badnumber", invalid.Number, "badhash", badHash)
		for len(api.invalidTipsets) >= invalidTipsetsCap {
			for key := range api.invalidTipsets {
				delete(api.invalidTipsets, key)
				break
			}
		}
		api.invalidTipsets[head] = invalid
	}
	lastValid := &invalid.ParentHash
	failure := "links to previously rejected block"
	return &engine.PayloadStatusV1{
		Status:          engine.INVALID,
		LatestValidHash: lastValid,
		ValidationError: &failure,
	}
}

// invalid returns a response "INVALID" with the latest valid hash supplied by latest.
func (api *ConsensusAPI) invalid(err error, latestValid *types.Header) engine.PayloadStatusV1 {
	var currentHash *common.Hash
	if latestValid != nil {
		h := latestValid.Hash()
		currentHash = &h
	}
	errorMsg := err.Error()
	return engine.PayloadStatusV1{Status: engine.INVALID, LatestValidHash: currentHash, ValidationError: &errorMsg}
}

// heartbeat loops indefinitely, and checks if there have been beacon client updates
// received in the last while. If not - or if they but strange ones - it warns the
// user that something might be off with their consensus node.
//
// TODO(karalabe): Spin this goroutine down somehow
func (api *ConsensusAPI) heartbeat() {
	// Sleep a bit on startup since there's obviously no beacon client yet
	// attached, so no need to print scary warnings to the user.
	time.Sleep(beaconUpdateStartupTimeout)

	var offlineLogged time.Time

	for {
		// Sleep a bit and retrieve the last known consensus updates
		time.Sleep(5 * time.Second)

		api.lastTransitionLock.Lock()
		lastTransitionUpdate := api.lastTransitionUpdate
		api.lastTransitionLock.Unlock()

		api.lastForkchoiceLock.Lock()
		lastForkchoiceUpdate := api.lastForkchoiceUpdate
		api.lastForkchoiceLock.Unlock()

		api.lastNewPayloadLock.Lock()
		lastNewPayloadUpdate := api.lastNewPayloadUpdate
		api.lastNewPayloadLock.Unlock()

		// If there have been no updates for the past while, warn the user
		// that the beacon client is probably offline
		if time.Since(lastForkchoiceUpdate) <= beaconUpdateConsensusTimeout || time.Since(lastNewPayloadUpdate) <= beaconUpdateConsensusTimeout {
			offlineLogged = time.Time{}
			continue
		}

		if time.Since(offlineLogged) > beaconUpdateWarnFrequency {
			if lastForkchoiceUpdate.IsZero() && lastNewPayloadUpdate.IsZero() {
				if lastTransitionUpdate.IsZero() {
					log.Warn("No beacon client seen. Please launch one to follow the chain!")
				} else {
					log.Warn("Beacon client online, but never received consensus updates. Please ensure your beacon client is operational to follow the chain!")
				}
			} else {
				log.Warn("Beacon client online, but no consensus updates received in a while. Please fix your beacon client to follow the chain!")
			}
			offlineLogged = time.Now()
		}
		continue
	}
}

// ExchangeCapabilities returns the current methods provided by this node.
func (api *ConsensusAPI) ExchangeCapabilities([]string) []string {
	return caps
}

// GetPayloadBodiesByHashV1 implements engine_getPayloadBodiesByHashV1 which allows for retrieval of a list
// of block bodies by the engine api.
func (api *ConsensusAPI) GetPayloadBodiesByHashV1(hashes []common.Hash) []*engine.ExecutionPayloadBodyV1 {
	bodies := make([]*engine.ExecutionPayloadBodyV1, len(hashes))
	for i, hash := range hashes {
		block := api.zond.BlockChain().GetBlockByHash(hash)
		bodies[i] = getBody(block)
	}
	return bodies
}

// GetPayloadBodiesByRangeV1 implements engine_getPayloadBodiesByRangeV1 which allows for retrieval of a range
// of block bodies by the engine api.
func (api *ConsensusAPI) GetPayloadBodiesByRangeV1(start, count hexutil.Uint64) ([]*engine.ExecutionPayloadBodyV1, error) {
	if start == 0 || count == 0 {
		return nil, engine.InvalidParams.With(fmt.Errorf("invalid start or count, start: %v count: %v", start, count))
	}
	if count > 1024 {
		return nil, engine.TooLargeRequest.With(fmt.Errorf("requested count too large: %v", count))
	}
	// limit count up until current
	current := api.zond.BlockChain().CurrentBlock().Number.Uint64()
	last := uint64(start) + uint64(count) - 1
	if last > current {
		last = current
	}
	bodies := make([]*engine.ExecutionPayloadBodyV1, 0, uint64(count))
	for i := uint64(start); i <= last; i++ {
		block := api.zond.BlockChain().GetBlockByNumber(i)
		bodies = append(bodies, getBody(block))
	}
	return bodies, nil
}

func getBody(block *types.Block) *engine.ExecutionPayloadBodyV1 {
	if block == nil {
		return nil
	}

	var (
		body        = block.Body()
		txs         = make([]hexutil.Bytes, len(body.Transactions))
		withdrawals = body.Withdrawals
	)

	for j, tx := range body.Transactions {
		data, _ := tx.MarshalBinary()
		txs[j] = hexutil.Bytes(data)
	}

	// Post-shanghai withdrawals MUST be set to empty slice instead of nil
	if withdrawals == nil && block.Header().WithdrawalsHash != nil {
		withdrawals = make([]*types.Withdrawal, 0)
	}

	return &engine.ExecutionPayloadBodyV1{
		TransactionData: txs,
		Withdrawals:     withdrawals,
	}
}
