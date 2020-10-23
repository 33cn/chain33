package manage

import (
	"context"
	"testing"

	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
	"github.com/libp2p/go-libp2p"
	"github.com/stretchr/testify/require"
)

func TestPeerInfoManager(t *testing.T) {
	h1, err := libp2p.New(context.Background())
	require.Nil(t, err)
	q := queue.New("test")
	go sub(q.Client())
	mgr := NewPeerInfoManager(context.Background(), h1, q.Client())
	require.Nil(t, mgr.Fetch(h1.ID()))
	require.Nil(t, mgr.FetchAll())
	mgr.Refresh(&types.Peer{
		Name: h1.ID().Pretty(),
		Header: &types.Header{
			Height: 888,
		},
	})
	require.NotNil(t, mgr.Fetch(h1.ID()))
	require.Equal(t, 1, len(mgr.FetchAll()))
	require.Equal(t, int64(888), mgr.PeerHeight(h1.ID()))
	mgr.prune()
	require.Equal(t, int64(888), mgr.PeerHeight(h1.ID()))
}

func sub(cli queue.Client) {
	cli.Sub("blockchain")
	for range cli.Recv() {
		cli.Reply(cli.NewMessage("", 0, &types.Header{Height: 10000}))
	}
}
