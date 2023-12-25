package snowman

import (
	"github.com/33cn/chain33/system/consensus"
	"github.com/33cn/chain33/system/consensus/snowman/utils"
	"github.com/33cn/chain33/types"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/snow/validators"
	"math/rand"
	"sync"
	"time"

	libpeer "github.com/libp2p/go-libp2p/core/peer"
)

type vdrSet struct {
	validators.Set
	ctx     *consensus.Context
	self    *types.Peer
	peerIDs map[ids.NodeID]string
	lock    sync.RWMutex
	rand    *rand.Rand
}

func newVdrSet() *vdrSet {

	s := &vdrSet{}
	s.Set = validators.NewSet()
	return s
}

func (s *vdrSet) init(ctx *consensus.Context) {

	s.Set = validators.NewSet()
	s.ctx = ctx
	s.rand = rand.New(rand.NewSource(types.Now().Unix()))
}

// Sample returns a collection of validatorIDs, potentially with duplicates.
// If sampling the requested size isn't possible, an error will be returned.
func (s *vdrSet) Sample(size int) ([]ids.NodeID, error) {

	peers, err := s.getConnectedPeers()
	if err != nil || len(peers) < size {
		snowLog.Error("Sample", "len", len(peers), "size", size, "err", err)
		return nil, utils.ErrValidatorSample
	}
	indices := s.rand.Perm(len(peers))
	ids := make([]ids.NodeID, 0, size)

	s.lock.Lock()
	defer s.lock.Unlock()
	for _, idx := range indices {

		nid, err := s.toNodeID(peers[idx].Name)
		if err != nil {
			snowLog.Error("Sample", "pid", peers[idx].Name, "to nodeID err", err)
			continue
		}
		s.peerIDs[nid] = peers[idx].Name
		ids = append(ids, nid)

		if len(ids) >= size {
			break
		}
	}

	if len(ids) < size {
		snowLog.Error("Sample not enough", "len", len(peers), "size", size, "err", err)
		return nil, utils.ErrValidatorSample
	}

	return ids, nil
}

func (s *vdrSet) getConnectedPeers() ([]*types.Peer, error) {

	msg := s.ctx.Base.GetQueueClient().NewMessage("p2p", types.EventPeerInfo, nil)
	err := s.ctx.Base.GetQueueClient().Send(msg, true)
	if err != nil {
		snowLog.Error("getConnectedPeers", "client.Send err:", err)
		return nil, err
	}
	resp, err := s.ctx.Base.GetQueueClient().WaitTimeout(msg, 5*time.Second)
	if err != nil {
		snowLog.Error("getConnectedPeers", "client.Wait err:", err)
		return nil, err
	}

	peerlist, ok := resp.GetData().(*types.PeerList)
	count := len(peerlist.GetPeers())
	if !ok || count < 2 {
		snowLog.Error("getConnectedPeers", "len", len(peerlist.GetPeers()), "ok", ok)
		return nil, types.ErrTypeAsset
	}

	s.self = peerlist.GetPeers()[count-1]
	peers := make([]*types.Peer, 0, count)
	for _, p := range peerlist.GetPeers() {

		if p.Self || p.Blocked || p.Header.GetHeight() < s.self.Header.GetHeight()-128 {
			continue
		}
		peers = append(peers, p)

	}
	return peers, nil
}

func (s *vdrSet) toLibp2pID(id ids.NodeID) string {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.peerIDs[id]
}

func (s *vdrSet) toNodeID(id string) (ids.NodeID, error) {

	pid, err := libpeer.Decode(id)
	if err != nil {
		return ids.EmptyNodeID, err
	}

	nid, err := ids.ToNodeID([]byte(pid)[:20])
	if err != nil {
		return ids.EmptyNodeID, err
	}

	return nid, nil
}
