package p2p

import (
	"testing"

	"github.com/33cn/chain33/queue"
	ctypes "github.com/33cn/chain33/types"
	"github.com/stretchr/testify/assert"
)

func TestManagerWait(t *testing.T) {
	mgr := &Manager{}
	mgr.Wait()
}

func TestManagerHandleP2PSub(t *testing.T) {
	mgr := &Manager{}
	mgr.handleP2PSub()
}

func TestPubBroadCastNoTypes(t *testing.T) {
	q := queue.New("test")
	mgr := &Manager{
		Client: q.Client(),
		p2pCfg: &ctypes.P2P{},
	}
	_, err := mgr.PubBroadCast("hash1", "data", ctypes.EventTx)
	assert.Nil(t, err)
}

