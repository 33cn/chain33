package manage

import (
	"context"
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
	"github.com/libp2p/go-libp2p"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestPeerInfoManager(t *testing.T) {
	h1, err := libp2p.New(context.Background())
	assert.Nil(t, err)
	q := queue.New("test")
	go sub(q.Client())
	mgr := NewPeerInfoManager(context.Background(), h1, q.Client())
	assert.Nil(t, mgr.Fetch(h1.ID()))
	assert.Nil(t, mgr.FetchAll())
	mgr.Refresh(&types.Peer{
		Name: h1.ID().Pretty(),
		Header: &types.Header{
			Height: 888,
		},
	})
	assert.NotNil(t, mgr.Fetch(h1.ID()))
	assert.Equal(t, 1, len(mgr.FetchAll()))
	assert.Equal(t, int64(888), mgr.PeerHeight(h1.ID()))
	mgr.prune(10000)
	assert.Equal(t, int64(888), mgr.PeerHeight(h1.ID()))
}

func sub(cli queue.Client) {
	cli.Sub("blockchain")
	for range cli.Recv() {
		cli.Reply(cli.NewMessage("", 0, &types.Header{Height: 10000}))
	}
}
