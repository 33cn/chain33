package net

import (
	"testing"

	clientMocks "github.com/33cn/chain33/client/mocks"
	"github.com/33cn/chain33/queue"
	ctypes "github.com/33cn/chain33/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	netOjb *netHandler
	qapi   *clientMocks.QueueProtocolAPI
	q      = queue.New("test")
)

func init() {
	qapi = &clientMocks.QueueProtocolAPI{}
	cfg := ctypes.NewChain33Config(ctypes.GetDefaultCfgstring())
	q.SetConfig(cfg)
	netOjb = &netHandler{}
	netOjb.cfg = cfg
	netOjb.cli.Init(q.Client(), qapi)

}

func TestNewNetApi_Version(t *testing.T) {
	version, err := netOjb.Version()
	assert.Nil(t, err)
	assert.Equal(t, version, "33")
}

func TestNewNetApi_Listening(t *testing.T) {
	listen, err := netOjb.Listening()
	assert.Nil(t, err)
	assert.Equal(t, listen, true)
}

func TestNewNetApi_PeerCount(t *testing.T) {
	var plist ctypes.PeerList
	var peer = &ctypes.Peer{
		Addr:        "192.168.0.19",
		Port:        13803,
		Name:        "16Uiu2HAmBdwm5i6Ao6hBedNXHSM44ZUhM4243s5yJGAKPyHRjESw",
		MempoolSize: 0,
		Self:        false,
		Header: &ctypes.Header{
			Version:    0,
			ParentHash: common.Hex2Bytes("0x56a00b884045e7087a6df93b58da8ba06d8127e3f9bf822b3c57271bf493aa62"),
			TxHash:     common.Hex2Bytes("0x91b60784d75497bc7c2ed681e51a985d933ec63ee6adf032ba6024a6adcb3dc9"),
			StateHash:  common.Hex2Bytes("0x593cfe7c38f21d87f2377983a911601140b679763e7e0439b830eab45dc26187"),
			Height:     2,
			BlockTime:  1647411454,
			TxCount:    1,
			Hash:       common.Hex2Bytes("0x560f19205f79a41bd89ccfc288dc89fe9902c33afafce9a43d00ed5c6c8e082e"),
			Difficulty: 0,
		},
		Version:     "6.0.0-3877-g1405d606a-1405d606@1.0.0",
		RunningTime: "0.044 minutes",
	}

	plist.Peers = append(plist.Peers, peer)

	testNewNetAPIGetPeers(t, &plist)
}

func testNewNetAPIGetPeers(t *testing.T, plist *ctypes.PeerList) {
	qapi.On("PeerInfo", mock.Anything).Return(plist, nil)

	count, err := netOjb.PeerCount()
	assert.Nil(t, err)
	t.Log("peers", count)
}
