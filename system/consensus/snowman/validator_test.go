package snowman

import (
	"strings"
	"testing"

	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/system/consensus/snowman/utils"
	"github.com/33cn/chain33/types"
	"github.com/stretchr/testify/require"
)

func TestNodeID(t *testing.T) {

	v := &vdrSet{}
	v.init(nil, nil)
	peer := "16Uiu2HAmVXKApGkrdWMvJPKx8QQzcnLafETiHefzGxjBS163LD1k"

	id, err := v.toNodeID(peer)
	require.Nil(t, err)
	ids := id.String()
	require.True(t, strings.Contains(ids, peer[len(peer)-20:]))

	ps := v.toLibp2pID(id)
	require.Equal(t, peer, ps)
	require.Equal(t, 1, v.Len())
	_, err = v.toNodeID("")
	require.Equal(t, types.ErrInvalidParam, err)
	_, err = v.toNodeID("1")
	require.Nil(t, err)
}

func Test_getConnectedPeers(t *testing.T) {

	v := &vdrSet{}

	q := queue.New("test")
	cli := q.Client()
	v.init(nil, cli)
	defer cli.Close()
	list := &types.PeerList{}
	go func() {
		cli.Sub("p2p")

		for msg := range cli.Recv() {

			if msg.Ty == types.EventPeerInfo {
				msg.Reply(cli.NewMessage("", 0, list))
			}
		}
	}()

	self := &types.Peer{Self: true, Header: &types.Header{Height: 129}}

	list.Peers = []*types.Peer{self}
	peers, err := v.getConnectedPeers()
	require.Nil(t, err)
	require.Nil(t, peers)
	peer := &types.Peer{Name: "peer", Header: &types.Header{}, Finalized: &types.SnowChoice{}, Blocked: true}

	list.Peers = []*types.Peer{peer, self}
	peers, err = v.getConnectedPeers()
	require.Nil(t, err)
	require.Equal(t, 0, len(peers))
	peer.Blocked = false
	peers, err = v.getConnectedPeers()
	require.Nil(t, err)
	require.Equal(t, 0, len(peers))
	peer.Finalized.Hash = []byte("test")
	peers, err = v.getConnectedPeers()
	require.Nil(t, err)
	require.Equal(t, 1, len(peers))
	require.Equal(t, peer, peers[0])

	// test sample
	_, err = v.Sample(2)
	require.Equal(t, utils.ErrValidatorSample, err)
	_, err = v.Sample(0)
	require.Equal(t, utils.ErrValidatorSample, err)
	peer1 := types.Clone(peer).(*types.Peer)
	peer1.Name = "peer1"
	list.Peers = []*types.Peer{peer, peer1, self}
	ids, err := v.Sample(2)
	require.Nil(t, err)
	require.Equal(t, 2, len(ids))
}

func TestVdrSetUnimplented(t *testing.T) {
	v := &vdrSet{}
	_ = v.String()
	v.RegisterCallbackListener(nil)
}
