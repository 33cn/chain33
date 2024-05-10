package snowman

import (
	"testing"

	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/utils/bag"
	"github.com/ava-labs/avalanchego/utils/set"
	"github.com/stretchr/testify/require"
)

func TestMsgSender(t *testing.T) {

	q := queue.New("test")
	cli := q.Client()
	v := &vdrSet{}
	v.init(nil, cli)
	s := newMsgSender(v, cli, newSnowContext(types.NewChain33Config(types.GetDefaultCfgstring())))
	defer cli.Close()
	peer := "testpeer"
	nodeID, err := v.toNodeID(peer)
	require.Nil(t, err)

	go func() {
		cli.Sub("p2p")
		var reqID, checkID uint32
		var checkName string
		for msg := range cli.Recv() {
			reqID++
			if msg.Ty == types.EventSnowmanChits {
				req := msg.Data.(*types.SnowChits)
				checkID = req.GetRequestID()
				checkName = req.GetPeerName()
			} else if msg.Ty == types.EventSnowmanPutBlock {
				req := msg.Data.(*types.SnowPutBlock)
				checkID = req.GetRequestID()
				checkName = req.GetPeerName()
			} else if msg.Ty == types.EventSnowmanGetBlock {
				req := msg.Data.(*types.SnowGetBlock)
				checkID = req.GetRequestID()
				checkName = req.GetPeerName()
			} else if msg.Ty == types.EventSnowmanPullQuery {
				req := msg.Data.(*types.SnowPullQuery)
				checkID = req.GetRequestID()
				checkName = req.GetPeerName()

			} else if msg.Ty == types.EventSnowmanPushQuery {
				req := msg.Data.(*types.SnowPushQuery)
				checkID = req.GetRequestID()
				checkName = req.GetPeerName()
			}

			require.Equal(t, peer, checkName)
			require.Equal(t, reqID, checkID)
		}
	}()

	s.SendChits(nil, nodeID, 1, ids.Empty, ids.Empty)
	s.SendGet(nil, nodeID, 2, ids.Empty)
	s.SendPut(nil, nodeID, 3, nil)
	vdrBag := bag.Bag[ids.NodeID]{}
	vdrBag.Add(nodeID)
	nodeIDs := set.Of(vdrBag.List()...)
	s.SendPullQuery(nil, nodeIDs, 4, ids.Empty)
	s.SendPushQuery(nil, nodeIDs, 5, nil)

	// test unimplented method
	s.SendGossip(nil, nil)
	s.SendGetAncestors(nil, ids.EmptyNodeID, 0, ids.Empty)
	s.SendAncestors(nil, ids.EmptyNodeID, 0, nil)
}
