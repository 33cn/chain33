package web3

import (
	"testing"

	clientMocks "github.com/33cn/chain33/client/mocks"
	"github.com/33cn/chain33/queue"
	ctypes "github.com/33cn/chain33/types"
	"github.com/stretchr/testify/assert"
)

var (
	webObj *web3Handler
	qapi   *clientMocks.QueueProtocolAPI
	q      = queue.New("test")
)

func init() {
	qapi = &clientMocks.QueueProtocolAPI{}
	cfg := ctypes.NewChain33Config(ctypes.GetDefaultCfgstring())
	q.SetConfig(cfg)
	webObj = &web3Handler{}
	webObj.cfg = cfg
	webObj.cli.Init(q.Client(), qapi)

}

func TestWeb3Handler_Sha3(t *testing.T) {
	hash, err := webObj.Sha3("0x68656c6c6f20776f726c64")
	if err != nil {
		t.Log("err:", err)
		return
	}
	t.Log("hash:", hash)
	assert.Equal(t, hash, "0x47173285a8d7341e5e972fc677286384f802f8ef42a5ec5f03bbfa254cb01fad")

}

func TestWeb3Handler_ClientVersion(t *testing.T) {
	verStr, err := webObj.ClientVersion()
	assert.Nil(t, err)
	t.Log(verStr)
	assert.Equal(t, verStr, "Geth/v1.8.15-omnibus-255989da/linux-amd64/go1.10.1")
}
