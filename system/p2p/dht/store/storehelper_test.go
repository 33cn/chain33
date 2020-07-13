package store

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	p2pty "github.com/33cn/chain33/system/p2p/dht/types"
	"github.com/33cn/chain33/types"
	"github.com/libp2p/go-libp2p"
	host "github.com/libp2p/go-libp2p-core/host"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/assert"
)

func createNode(t *testing.T, port int32, path string) (*dht.IpfsDHT, host.Host) {
	m, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", port))
	assert.NoError(t, err)

	host, err := libp2p.New(context.Background(),
		libp2p.ListenAddrs(m),
	)
	assert.NoError(t, err)

	subCfg := &p2pty.P2PSubConfig{DHTDataPath: path}
	dhtRouting, err := dht.New(context.Background(), host, dataStoreOption(subCfg), validatorOption())
	assert.NoError(t, err)

	return dhtRouting, host
}

func TestStoreHelper(t *testing.T) {
	dir1, err := ioutil.TempDir("", "p2pstore1")
	assert.NoError(t, err)
	defer os.RemoveAll(dir1)
	os.RemoveAll(dir1)

	dir2, err := ioutil.TempDir("", "p2pstore2")
	assert.NoError(t, err)
	defer os.RemoveAll(dir2)
	os.RemoveAll(dir2)

	dhtRouting1, host1 := createNode(t, 8888, dir1)
	dhtRouting2, host2 := createNode(t, 8889, dir2)

	host1.Peerstore().AddAddr(host2.ID(), host2.Addrs()[0], 24*time.Hour)
	dhtRouting1.Update(context.Background(), host2.ID())

	host2.Peerstore().AddAddr(host1.ID(), host1.Addrs()[0], 24*time.Hour)
	dhtRouting2.Update(context.Background(), host1.ID())

	block1 := &types.Block{}
	block1.MainHash = block1.HashNew()
	block1.Height = 99999

	storeHelper := &Helper{}
	err1 := storeHelper.putBlock(dhtRouting1, block1, block1.MainHash)
	assert.NoError(t, err1)

	block2, err := storeHelper.getBlockByHash(dhtRouting2, block1.MainHash)
	assert.NoError(t, err)

	assert.Equal(t, block1.Height, block2.Height)
	assert.Equal(t, block1.MainHash, block2.MainHash)
}
