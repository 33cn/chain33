package snowman

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/33cn/chain33/system/consensus"
	"github.com/33cn/chain33/system/consensus/snowman/utils"
	"github.com/33cn/chain33/types"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/snow/validators"

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

func (s *vdrSet) init(ctx *consensus.Context) {

	s.Set = validators.NewSet()
	s.ctx = ctx
	s.rand = rand.New(rand.NewSource(types.Now().Unix()))
	s.peerIDs = make(map[ids.NodeID]string)
}

func (s *vdrSet) Len() int {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return len(s.peerIDs)
}

func (s *vdrSet) String() string {
	return fmt.Sprintf("Validator Set: Size = %d", s.Len())
}

// Sample returns a collection of validatorIDs, potentially with duplicates.
// If sampling the requested size isn't possible, an error will be returned.
func (s *vdrSet) Sample(size int) ([]ids.NodeID, error) {

	snowLog.Debug("vdrSet Sample", "require", size)
	peers, err := s.getConnectedPeers()
	if err != nil || len(peers) < size {
		snowLog.Error("vdrSet Sample", "connected", len(peers), "require", size, "err", err)
		return nil, utils.ErrValidatorSample
	}
	indices := s.rand.Perm(len(peers))
	nodeIDS := make([]ids.NodeID, 0, size)

	for _, idx := range indices {

		nid, err := s.toNodeID(peers[idx].Name)
		if err != nil {
			snowLog.Error("vdrSet Sample", "pid", peers[idx].Name, "to nodeID err", err)
			continue
		}

		nodeIDS = append(nodeIDS, nid)

		if len(nodeIDS) >= size {
			break
		}
	}

	if len(nodeIDS) < size {
		snowLog.Error("vdrSet Sample not enough", "len", len(peers), "size", size, "err", err)
		return nil, utils.ErrValidatorSample
	}

	return nodeIDS, nil
}

func (s *vdrSet) getConnectedPeers() ([]*types.Peer, error) {
	snowLog.Debug("vdrSet getConnectedPeers")
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

	peerlist := resp.GetData().(*types.PeerList)
	count := len(peerlist.GetPeers())

	if count < 2 {
		return nil, nil
	}
	s.self = peerlist.GetPeers()[count-1]
	peers := make([]*types.Peer, 0, count)
	for _, p := range peerlist.GetPeers() {

		if p.Self || p.Blocked ||
			p.Header.GetHeight() < s.self.Finalized.GetHeight() ||
			p.GetFinalized().GetHeight() < s.self.Finalized.GetHeight()-128 {
			continue
		}
		peers = append(peers, p)

	}
	return peers, nil
}

func (s *vdrSet) toLibp2pID(id ids.NodeID) string {
	s.lock.RLock()
	defer s.lock.RUnlock()
	name, ok := s.peerIDs[id]
	if !ok {
		panic("peer id not exist")
	}
	return name
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

	s.lock.Lock()
	defer s.lock.Unlock()
	s.peerIDs[nid] = id

	return nid, nil
}

func (s *vdrSet) RegisterCallbackListener(callbackListener validators.SetCallbackListener) {
	snowLog.Debug("RegisterCallbackListener Not Implemented Function")
}
