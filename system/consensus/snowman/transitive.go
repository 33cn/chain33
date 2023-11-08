// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package snowman

import (
	"context"
	"github.com/33cn/chain33/types"

	"go.uber.org/zap"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/snow"
	"github.com/ava-labs/avalanchego/snow/choices"
	"github.com/ava-labs/avalanchego/snow/consensus/snowman"
	"github.com/ava-labs/avalanchego/snow/consensus/snowman/poll"
	"github.com/ava-labs/avalanchego/snow/engine/common"
	"github.com/ava-labs/avalanchego/snow/events"
	"github.com/ava-labs/avalanchego/utils/wrappers"
	//"github.com/ava-labs/avalanchego/version"
)

// New new snow engine
func New(config Config) (*Transitive, error) {
	return newTransitive(config)
}

// Transitive implements the Engine interface by attempting to fetch all
// Transitive dependencies.
type Transitive struct {
	metrics
	getter
	sender
	Config

	// list of NoOpsHandler for messages dropped by engine
	//common.StateSummaryFrontierHandler
	//common.AcceptedStateSummaryHandler
	//common.AcceptedFrontierHandler
	//common.AcceptedHandler
	//common.AncestorsHandler

	RequestID uint32

	// track outstanding preference requests
	polls poll.Set

	// blocks that have we have sent get requests for but haven't yet received
	blkReqs common.Requests

	// blocks that are queued to be issued to consensus once missing dependencies are fetched
	// Block ID --> Block
	pending map[ids.ID]snowman.Block

	// Block ID --> Parent ID
	nonVerifieds AncestorTree

	// operations that are blocked on a block being issued. This could be
	// issuing another block, responding to a query, or applying votes to consensus
	blocked events.Blocker

	// number of times build block needs to be called once the number of
	// processing blocks has gone below the optimal number.
	pendingBuildBlocks int

	// errs tracks if an error has occurred in a callback
	errs wrappers.Errs
}

func newTransitive(config Config) (*Transitive, error) {
	log.Info("initializing consensus engine")

	factory := poll.NewEarlyTermNoTraversalFactory(config.params.Alpha)
	t := &Transitive{
		Config: config,
		//StateSummaryFrontierHandler: common.NewNoOpStateSummaryFrontierHandler(config.log),
		//AcceptedStateSummaryHandler: common.NewNoOpAcceptedStateSummaryHandler(config.log),
		//AcceptedFrontierHandler:     common.NewNoOpAcceptedFrontierHandler(config.ctx.Log),
		//AcceptedHandler:             common.NewNoOpAcceptedHandler(config.ctx.Log),
		//AncestorsHandler:            common.NewNoOpAncestorsHandler(config.ctx.Log),
		pending:      make(map[ids.ID]snowman.Block),
		nonVerifieds: NewAncestorTree(),
		polls: poll.NewSet(factory,
			config.log,
			"",
			config.registerer,
		),
	}

	return t, t.metrics.Initialize("", config.registerer)
}

// Put put block
func (t *Transitive) Put(nodeID ids.NodeID, block *types.Block) error {

	sb := newSnowBlock(block, t.chainCfg)

	if t.wasIssued(sb) {
		t.metrics.numUselessPutBytes.Add(float64(block.Size()))
	}

	// issue the block into consensus. If the block has already been issued,
	// this will be a noop. If this block has missing dependencies, vdr will
	// receive requests to fill the ancestry. dependencies that have already
	// been fetched, but with missing dependencies themselves won't be requested
	// from the vdr.
	if _, err := t.issueFrom(nodeID, sb); err != nil {
		return err
	}
	return t.buildBlocks()
}

// GetFailed get failed
func (t *Transitive) GetFailed(ctx context.Context, nodeID ids.NodeID, requestID uint32) error {
	// We don't assume that this function is called after a failed Get message.
	// Check to see if we have an outstanding request and also get what the request was for if it exists.
	blkID, ok := t.blkReqs.Remove(nodeID, requestID)
	if !ok {
		t.log.Debug("unexpected GetFailed",
			zap.Stringer("nodeID", nodeID),
			zap.Uint32("requestID", requestID),
		)
		return nil
	}

	// Because the get request was dropped, we no longer expect blkID to be issued.
	t.blocked.Abandon(ctx, blkID)
	t.metrics.numBlockers.Set(float64(t.blocked.Len()))
	return t.buildBlocks()
}

// handlePullQuery handle pull query from nodeID
func (t *Transitive) handlePullQuery(nodeID ids.NodeID, requestID uint32, blkID ids.ID) error {
	// type here.
	t.sendChits(nodeID, requestID, []ids.ID{t.consensus.Preference()})

	// Try to issue [blkID] to consensus.
	// If we're missing an ancestor, request it from [vdr]
	if _, err := t.issueFromByID(nodeID, blkID); err != nil {
		return err
	}

	return t.buildBlocks()
}

// handlePushQuery handle push query from other node
func (t *Transitive) handlePushQuery(nodeID ids.NodeID, requestID uint32, block *types.Block) error {
	// TODO: once everyone supports ChitsV2 - we should be sending that message
	// type here.
	t.sendChits(nodeID, requestID, []ids.ID{t.consensus.Preference()})

	sb := newSnowBlock(block, t.chainCfg)

	if t.wasIssued(sb) {
		t.metrics.numUselessPushQueryBytes.Add(float64(block.Size()))
	}

	// issue the block into consensus. If the block has already been issued,
	// this will be a noop. If this block has missing dependencies, nodeID will
	// receive requests to fill the ancestry. dependencies that have already
	// been fetched, but with missing dependencies themselves won't be requested
	// from the vdr.
	if _, err := t.issueFrom(nodeID, sb); err != nil {
		return err
	}

	return t.buildBlocks()
}

// handleChits  handle chits
func (t *Transitive) handleChits(nodeID ids.NodeID, requestID uint32, votes []ids.ID) error {
	// Since this is a linear chain, there should only be one ID in the vote set
	if len(votes) != 1 {
		t.log.Debug("failing Chits",
			zap.String("reason", "expected only 1 vote"),
			zap.Int("numVotes", len(votes)),
			zap.Stringer("nodeID", nodeID),
			zap.Uint32("requestID", requestID),
		)
		// because QueryFailed doesn't utilize the assumption that we actually
		// sent a Query message, we can safely call QueryFailed here to
		// potentially abandon the request.
		return t.QueryFailed(nodeID, requestID)
	}
	blkID := votes[0]

	t.log.Verbo("called Chits for the block",
		zap.Stringer("blkID", blkID),
		zap.Stringer("nodeID", nodeID),
		zap.Uint32("requestID", requestID))

	// Will record chits once [blkID] has been issued into consensus
	v := &voter{
		t:         t,
		vdr:       nodeID,
		requestID: requestID,
		response:  blkID,
	}

	added, err := t.issueFromByID(nodeID, blkID)
	if err != nil {
		return err
	}
	// Wait until [blkID] has been issued to consensus before applying this chit.
	if !added {
		v.deps.Add(blkID)
	}

	t.blocked.Register(v)
	t.metrics.numBlockers.Set(float64(t.blocked.Len()))
	return t.buildBlocks()
}

func (t *Transitive) ChitsV2(vdr ids.NodeID, requestID uint32, _ []ids.ID, vote ids.ID) error {
	return t.Chits(vdr, requestID, []ids.ID{vote})
}

func (t *Transitive) QueryFailed(vdr ids.NodeID, requestID uint32) error {
	t.blocked.Register(&voter{
		t:         t,
		vdr:       vdr,
		requestID: requestID,
	})
	t.metrics.numBlockers.Set(float64(t.blocked.Len()))
	return t.buildBlocks()
}

func (t *Transitive) Notify(msg common.Message) error {
	if msg != common.PendingTxs {
		t.log.Warn("received an unexpected message from the VM",
			zap.Stringer("message", msg),
		)
		return nil
	}

	// the pending txs message means we should attempt to build a block.
	t.pendingBuildBlocks++
	return t.buildBlocks()
}

func (t *Transitive) Context() *snow.ConsensusContext {
	return t.ctx
}

func (t *Transitive) Start(startReqID uint32) error {
	t.RequestID = startReqID

	lastAccepted := newSnowBlock(t.getter.getLastAcceptBlock(), t.chainCfg)
	lastAcceptedID := lastAccepted.ID()
	// initialize consensus to the last accepted blockID
	if err := t.consensus.Initialize(t.ctx, t.params, lastAcceptedID, lastAccepted.Height()); err != nil {
		return err
	}

	// to maintain the invariant that oracle blocks are issued in the correct
	// preferences, we need to handle the case that we are bootstrapping into an oracle block
	//if oracleBlk, ok := lastAccepted.(snowman.OracleBlock); ok {
	//	options, err := oracleBlk.Options()
	//	switch {
	//	case err == snowman.ErrNotOracle:
	//		// if there aren't blocks we need to deliver on startup, we need to set
	//		// the preference to the last accepted block
	//		if err := t.VM.SetPreference(lastAcceptedID); err != nil {
	//			return err
	//		}
	//	case err != nil:
	//		return err
	//	default:
	//		for _, blk := range options {
	//			// note that deliver will set the VM's preference
	//			if err := t.deliver(blk); err != nil {
	//				return err
	//			}
	//		}
	//	}
	//} else if err := t.VM.SetPreference(lastAcceptedID); err != nil {
	//	return err
	//}

	t.ctx.Log.Info("consensus starting",
		zap.Stringer("lastAcceptedBlock", lastAcceptedID),
	)
	t.metrics.bootstrapFinished.Set(1)

	t.ctx.SetState(snow.NormalOp)
	//if err := t.VM.SetState(snow.NormalOp); err != nil {
	//	return fmt.Errorf("failed to notify VM that consensus is starting: %w",
	//		err)
	//}
	return nil
}

func (t *Transitive) HealthCheck() (interface{}, error) {
	consensusIntf, consensusErr := t.consensus.HealthCheck()
	return consensusIntf, consensusErr
}

func (t *Transitive) GetBlock(blkID ids.ID) (snowman.Block, error) {
	if blk, ok := t.pending[blkID]; ok {
		return blk, nil
	}
	blk, err := t.getBlock(blkID[:])
	if err != nil {
		return nil, err
	}
	return newSnowBlock(blk, t.chainCfg), nil
}

// Build blocks if they have been requested and the number of processing blocks
// is less than optimal.
func (t *Transitive) buildBlocks() error {
	if err := t.errs.Err; err != nil {
		return err
	}

	return nil
}

// Issue another poll to the network, asking what it prefers given the block we prefer.
// Helps move consensus along.
func (t *Transitive) repoll() {
	// if we are issuing a repoll, we should gossip our current preferences to
	// propagate the most likely branch as quickly as possible
	prefID := t.consensus.Preference()

	for i := t.polls.Len(); i < t.params.ConcurrentRepolls; i++ {
		t.pullQuery(prefID)
	}
}

// issueFromByID attempts to issue the branch ending with a block [blkID] into consensus.
// If we do not have [blkID], request it.
// Returns true if the block is processing in consensus or is decided.
func (t *Transitive) issueFromByID(nodeID ids.NodeID, blkID ids.ID) (bool, error) {
	blk, err := t.GetBlock(blkID)
	if err != nil {
		t.sendRequest(nodeID, blkID)
		return false, nil
	}
	return t.issueFrom(nodeID, blk)
}

// issueFrom attempts to issue the branch ending with block [blkID] to consensus.
// Returns true if the block is processing in consensus or is decided.
// If a dependency is missing, request it from [vdr].
func (t *Transitive) issueFrom(nodeID ids.NodeID, blk snowman.Block) (bool, error) {
	blkID := blk.ID()
	// issue [blk] and its ancestors to consensus.
	for !t.wasIssued(blk) {
		if err := t.issue(blk); err != nil {
			return false, err
		}

		blkID = blk.Parent()
		var err error
		blk, err = t.GetBlock(blkID)

		// If we don't have this ancestor, request it from [vdr]
		if err != nil || !blk.Status().Fetched() {
			t.sendRequest(nodeID, blkID)
			return false, nil
		}
	}

	// Remove any outstanding requests for this block
	t.blkReqs.RemoveAny(blkID)

	issued := t.consensus.Decided(blk) || t.consensus.Processing(blkID)
	if issued {
		// A dependency should never be waiting on a decided or processing
		// block. However, if the block was marked as rejected by the VM, the
		// dependencies may still be waiting. Therefore, they should abandoned.
		t.blocked.Abandon(blkID)
	}

	// Tracks performance statistics
	t.metrics.numRequests.Set(float64(t.blkReqs.Len()))
	t.metrics.numBlockers.Set(float64(t.blocked.Len()))
	return issued, t.errs.Err
}

// issueWithAncestors attempts to issue the branch ending with [blk] to consensus.
// Returns true if the block is processing in consensus or is decided.
// If a dependency is missing and the dependency hasn't been requested, the issuance will be abandoned.
func (t *Transitive) issueWithAncestors(blk snowman.Block) (bool, error) {
	blkID := blk.ID()
	// issue [blk] and its ancestors into consensus
	status := blk.Status()
	for status.Fetched() && !t.wasIssued(blk) {
		if err := t.issue(blk); err != nil {
			return false, err
		}
		blkID = blk.Parent()
		var err error
		if blk, err = t.GetBlock(blkID); err != nil {
			status = choices.Unknown
			break
		}
		status = blk.Status()
	}

	// The block was issued into consensus. This is the happy path.
	if status != choices.Unknown && (t.consensus.Decided(blk) || t.consensus.Processing(blkID)) {
		return true, nil
	}

	// There's an outstanding request for this block.
	// We can just wait for that request to succeed or fail.
	if t.blkReqs.Contains(blkID) {
		return false, nil
	}

	// We don't have this block and have no reason to expect that we will get it.
	// Abandon the block to avoid a memory leak.
	t.blocked.Abandon(blkID)
	t.metrics.numBlockers.Set(float64(t.blocked.Len()))
	return false, t.errs.Err
}

// If the block has been decided, then it is marked as having been issued.
// If the block is processing, then it was issued.
// If the block is queued to be added to consensus, then it was issued.
func (t *Transitive) wasIssued(blk snowman.Block) bool {
	blkID := blk.ID()
	return t.consensus.Decided(blk) || t.consensus.Processing(blkID) || t.pendingContains(blkID)
}

// Issue [blk] to consensus once its ancestors have been issued.
func (t *Transitive) issue(blk snowman.Block) error {
	blkID := blk.ID()

	// mark that the block is queued to be added to consensus once its ancestors have been
	t.pending[blkID] = blk

	// Remove any outstanding requests for this block
	t.blkReqs.RemoveAny(blkID)

	// Will add [blk] to consensus once its ancestors have been
	i := &issuer{
		t:   t,
		blk: blk,
	}

	// block on the parent if needed
	parentID := blk.Parent()
	if parent, err := t.GetBlock(parentID); err != nil || !(t.consensus.Decided(parent) || t.consensus.Processing(parentID)) {
		t.ctx.Log.Verbo("block waiting for parent to be issued",
			zap.Stringer("blkID", blkID),
			zap.Stringer("parentID", parentID),
		)
		i.deps.Add(parentID)
	}

	t.blocked.Register(i)

	// Tracks performance statistics
	t.metrics.numRequests.Set(float64(t.blkReqs.Len()))
	t.metrics.numBlocked.Set(float64(len(t.pending)))
	t.metrics.numBlockers.Set(float64(t.blocked.Len()))
	return t.errs.Err
}

// Request that [vdr] send us block [blkID]
func (t *Transitive) sendRequest(nodeID ids.NodeID, blkID ids.ID) {
	// There is already an outstanding request for this block
	if t.blkReqs.Contains(blkID) {
		return
	}

	t.RequestID++
	t.blkReqs.Add(nodeID, t.RequestID, blkID)
	t.ctx.Log.Verbo("sending Get request",
		zap.Stringer("nodeID", nodeID),
		zap.Uint32("requestID", t.RequestID),
		zap.Stringer("blkID", blkID),
	)
	t.sender.sendGet(nodeID, t.RequestID, blkID)

	// Tracks performance statistics
	t.metrics.numRequests.Set(float64(t.blkReqs.Len()))
}

// send a pull query for this block ID
func (t *Transitive) pullQuery(blkID ids.ID) {
	t.ctx.Log.Verbo("sampling from validators",
		zap.Stringer("validators", t.validators),
	)
	// The validators we will query
	vdrs, err := t.validators.Sample(t.params.K)
	if err != nil {
		t.ctx.Log.Error("dropped query for block",
			zap.String("reason", "insufficient number of validators"),
			zap.Stringer("blkID", blkID),
		)
		return
	}

	vdrBag := ids.NodeIDBag{}
	for _, vdr := range vdrs {
		vdrBag.Add(vdr.ID())
	}

	t.RequestID++
	if t.polls.Add(t.RequestID, vdrBag) {
		vdrList := vdrBag.List()
		vdrSet := ids.NewNodeIDSet(len(vdrList))
		vdrSet.Add(vdrList...)
		t.sender.sendPullQuery(vdrSet, t.RequestID, blkID)
	}
}

// Send a query for this block. Some validators will be sent
// a Push Query and some will be sent a Pull Query.
func (t *Transitive) sendMixedQuery(blk snowman.Block) {
	t.ctx.Log.Verbo("sampling from validators",
		zap.Stringer("validators", t.validators),
	)
	vdrs, err := t.validators.Sample(t.params.K)
	if err != nil {
		t.ctx.Log.Error("dropped query for block",
			zap.String("reason", "insufficient number of validators"),
			zap.Stringer("blkID", blk.ID()),
		)
		return
	}

	vdrBag := ids.NodeIDBag{}
	for _, vdr := range vdrs {
		vdrBag.Add(vdr.ID())
	}

	t.RequestID++
	if t.polls.Add(t.RequestID, vdrBag) {
		// Send a push query to some of the validators, and a pull query to the rest.
		numPushTo := t.params.MixedQueryNumPushVdr
		if !t.validators.Contains(t.ctx.NodeID) {
			numPushTo = t.params.MixedQueryNumPushNonVdr
		}
		common.SendMixedQuery(
			t.sender,
			vdrBag.List(), // Note that this doesn't contain duplicates; length may be < k
			numPushTo,
			t.RequestID,
			blk.ID(),
			blk.Bytes(),
		)
	}
}

// issue [blk] to consensus
func (t *Transitive) deliver(blk snowman.Block) error {
	blkID := blk.ID()
	if t.consensus.Decided(blk) || t.consensus.Processing(blkID) {
		return nil
	}

	// we are no longer waiting on adding the block to consensus, so it is no
	// longer pending
	t.removeFromPending(blk)
	parentID := blk.Parent()
	parent, err := t.GetBlock(parentID)
	// Because the dependency must have been fulfilled by the time this function
	// is called - we don't expect [err] to be non-nil. But it is handled for
	// completness and future proofing.
	if err != nil || !(parent.Status() == choices.Accepted || t.consensus.Processing(parentID)) {
		// if the parent isn't processing or the last accepted block, then this
		// block is effectively rejected
		t.blocked.Abandon(blkID)
		t.metrics.numBlocked.Set(float64(len(t.pending))) // Tracks performance statistics
		t.metrics.numBlockers.Set(float64(t.blocked.Len()))
		return t.errs.Err
	}

	// By ensuring that the parent is either processing or accepted, it is
	// guaranteed that the parent was successfully verified. This means that
	// calling Verify on this block is allowed.

	// make sure this block is valid
	if err := blk.Verify(); err != nil {
		t.ctx.Log.Debug("block verification failed",
			zap.Error(err),
		)

		// if verify fails, then all descendants are also invalid
		t.addToNonVerifieds(blk)
		t.blocked.Abandon(blkID)
		t.metrics.numBlocked.Set(float64(len(t.pending))) // Tracks performance statistics
		t.metrics.numBlockers.Set(float64(t.blocked.Len()))
		return t.errs.Err
	}
	t.nonVerifieds.Remove(blkID)
	t.metrics.numNonVerifieds.Set(float64(t.nonVerifieds.Len()))
	t.ctx.Log.Verbo("adding block to consensus",
		zap.Stringer("blkID", blkID),
	)
	wrappedBlk := &memoryBlock{
		Block:   blk,
		metrics: &t.metrics,
		tree:    t.nonVerifieds,
	}
	if err := t.consensus.Add(wrappedBlk); err != nil {
		return err
	}

	// Add all the oracle blocks if they exist. We call verify on all the blocks
	// and add them to consensus before marking anything as fulfilled to avoid
	// any potential reentrant bugs.
	var added []snowman.Block
	var dropped []snowman.Block
	if blk, ok := blk.(snowman.OracleBlock); ok {
		options, err := blk.Options()
		if err != snowman.ErrNotOracle {
			if err != nil {
				return err
			}

			for _, blk := range options {
				if err := blk.Verify(); err != nil {
					t.ctx.Log.Debug("block verification failed",
						zap.Error(err),
					)
					dropped = append(dropped, blk)
					// block fails verification, hold this in memory for bubbling
					t.addToNonVerifieds(blk)
				} else {
					// correctly verified will be passed to consensus as processing block
					// no need to keep it anymore
					t.nonVerifieds.Remove(blk.ID())
					t.metrics.numNonVerifieds.Set(float64(t.nonVerifieds.Len()))
					wrappedBlk := &memoryBlock{
						Block:   blk,
						metrics: &t.metrics,
						tree:    t.nonVerifieds,
					}
					if err := t.consensus.Add(wrappedBlk); err != nil {
						return err
					}
					added = append(added, blk)
				}
			}
		}
	}

	if err := t.VM.SetPreference(t.consensus.Preference()); err != nil {
		return err
	}

	// If the block is now preferred, query the network for its preferences
	// with this new block.
	if t.consensus.IsPreferred(blk) {
		t.sendMixedQuery(blk)
	}

	t.blocked.Fulfill(blkID)
	for _, blk := range added {
		if t.consensus.IsPreferred(blk) {
			t.sendMixedQuery(blk)
		}

		blkID := blk.ID()
		t.removeFromPending(blk)
		t.blocked.Fulfill(blkID)
		t.blkReqs.RemoveAny(blkID)
	}
	for _, blk := range dropped {
		blkID := blk.ID()
		t.removeFromPending(blk)
		t.blocked.Abandon(blkID)
		t.blkReqs.RemoveAny(blkID)
	}

	// If we should issue multiple queries at the same time, we need to repoll
	t.repoll()

	// Tracks performance statistics
	t.metrics.numRequests.Set(float64(t.blkReqs.Len()))
	t.metrics.numBlocked.Set(float64(len(t.pending)))
	t.metrics.numBlockers.Set(float64(t.blocked.Len()))
	return t.errs.Err
}

// Returns true if the block whose ID is [blkID] is waiting to be issued to consensus
func (t *Transitive) pendingContains(blkID ids.ID) bool {
	_, ok := t.pending[blkID]
	return ok
}

func (t *Transitive) removeFromPending(blk snowman.Block) {
	delete(t.pending, blk.ID())
}

func (t *Transitive) addToNonVerifieds(blk snowman.Block) {
	// don't add this blk if it's decided or processing.
	blkID := blk.ID()
	if t.consensus.Decided(blk) || t.consensus.Processing(blkID) {
		return
	}
	parentID := blk.Parent()
	// we might still need this block so we can bubble votes to the parent
	// only add blocks with parent already in the tree or processing.
	// decided parents should not be in this map.
	if t.nonVerifieds.Has(parentID) || t.consensus.Processing(parentID) {
		t.nonVerifieds.Add(blkID, parentID)
		t.metrics.numNonVerifieds.Set(float64(t.nonVerifieds.Len()))
	}
}
