package snowman

import (
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/33cn/chain33/queue"

	"github.com/33cn/chain33/system/consensus"
	"github.com/33cn/chain33/system/consensus/snowman/utils"
	"github.com/33cn/chain33/types"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/snow/validators"
)

type vdrSet struct {
	validators.Set
	ctx     *consensus.Context
	qclient queue.Client
	self    *types.Peer
	peerIDs map[ids.NodeID]string
	lock    sync.RWMutex
	rand    *rand.Rand
}

func (s *vdrSet) init(ctx *consensus.Context, cli queue.Client) {

	s.Set = validators.NewSet()
	s.ctx = ctx
	s.qclient = cli
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
	if err != nil || len(peers) < size || size <= 0 {
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

const maxHeightDiff = 128

func (s *vdrSet) getConnectedPeers() ([]*types.Peer, error) {

	msg := s.qclient.NewMessage("p2p", types.EventPeerInfo, nil)
	err := s.qclient.Send(msg, true)
	if err != nil {
		snowLog.Error("getConnectedPeers", "client.Send err:", err)
		return nil, err
	}
	resp, err := s.qclient.WaitTimeout(msg, 5*time.Second)
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

		// 过滤未启用节点
		if p.Self || p.Blocked || len(p.GetFinalized().GetHash()) <= 0 {
			continue
		}

		headerDiff := math.Abs(float64(p.Header.GetHeight() - s.self.Header.GetHeight()))
		finalizeDiff := math.Abs(float64(p.Finalized.GetHeight() - s.self.Finalized.GetHeight()))
		// 过滤高度差较大节点
		if headerDiff > maxHeightDiff && finalizeDiff > maxHeightDiff {
			snowLog.Debug("getConnectedPeers filter", "peer", p.Name,
				"pHeight", p.Header.GetHeight(), "fHeight", p.Finalized.GetHeight(),
				"selfHeight", s.self.Header.GetHeight(), "fHeight", s.self.Finalized.GetHeight())
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

	if id == "" {
		return ids.EmptyNodeID, types.ErrInvalidParam
	}
	tempID := id
	for len(tempID) < 20 {
		tempID += tempID
	}
	nid, err := ids.ToNodeID([]byte(tempID[len(tempID)-20:]))
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
