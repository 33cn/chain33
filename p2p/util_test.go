package p2p

import (
	"sort"
	"testing"

	"github.com/33cn/chain33/types"
	"github.com/stretchr/testify/assert"
)

func Test_AddDelStream(t *testing.T) {

	s := NewP2pServer()
	peerName := "testpeer"
	delChan := s.addStreamHandler(peerName)
	//replace
	dataChan := s.addStreamHandler(peerName)

	_, ok := <-delChan
	assert.False(t, ok)

	//del old
	s.deleteStream(peerName, delChan)
	_, ok = s.streams[peerName]
	assert.True(t, ok)
	//del nil
	s.deleteStream("", delChan)
	//del exist
	s.deleteStream(peerName, dataChan)

	_, ok = s.streams[peerName]
	assert.False(t, ok)
}

func Test_spaceLimitCache(t *testing.T) {

	c := newSpaceLimitCache(3, 10)
	assert.True(t, c.add(1, 1, 1))
	assert.True(t, c.add(1, 1, 1))
	assert.False(t, c.add(2, 2, 20))
	assert.Nil(t, c.get(2))
	assert.True(t, c.add(2, 1, 10))
	c.add(3, 2, 2)
	c.add(4, 2, 2)
	c.add(5, 2, 2)
	c.add(6, 2, 2)
	assert.False(t, c.contains(2))
	assert.Equal(t, 3, c.data.Len())
	assert.True(t, c.add(7, 7, 10))
	assert.True(t, c.contains(7))
	assert.Equal(t, 1, c.data.Len())
	_, exist := c.del(7)
	assert.True(t, exist)
	_, exist = c.del(6)
	assert.False(t, exist)
}

func testChannelVersion(t *testing.T, channel int32) {
	chanVer := calcChannelVersion(channel)
	chann, ver := decodeChannelVersion(chanVer)

	assert.True(t, chann == channel)
	assert.True(t, ver == VERSION)
}

func Test_ChannelVersion(t *testing.T) {

	testChannelVersion(t, 0)
	testChannelVersion(t, 128)
}

func TestRandStr(t *testing.T) {
	t.Log(P2pComm.RandStr(5))
}

func TestBytesToInt32(t *testing.T) {

	t.Log(P2pComm.BytesToInt32([]byte{0xff}))
	t.Log(P2pComm.Int32ToBytes(255))
}

func TestSortArr(t *testing.T) {
	var Inventorys = make(Invs, 0)
	for i := 100; i >= 0; i-- {
		var inv types.Inventory
		inv.Ty = 111
		inv.Height = int64(i)
		Inventorys = append(Inventorys, &inv)
	}
	sort.Sort(Inventorys)
}

func TestFilter(t *testing.T) {
	filter := NewFilter(10)
	go filter.ManageRecvFilter()
	defer filter.Close()
	filter.GetLock()

	assert.Equal(t, true, filter.RegRecvData("key"))
	assert.Equal(t, true, filter.QueryRecvData("key"))
	filter.RemoveRecvData("key")
	assert.Equal(t, false, filter.QueryRecvData("key"))
	filter.ReleaseLock()

}
