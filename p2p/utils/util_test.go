package utils

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_spaceLimitCache(t *testing.T) {

	c := NewSpaceLimitCache(3, 10)
	assert.True(t, c.Add(1, 1, 1))
	assert.True(t, c.Add(1, 1, 1))
	assert.False(t, c.Add(2, 2, 20))
	assert.Nil(t, c.Get(2))
	assert.True(t, c.Add(2, 1, 10))
	c.Add(3, 2, 2)
	c.Add(4, 2, 2)
	c.Add(5, 2, 2)
	c.Add(6, 2, 2)
	assert.False(t, c.Contains(2))
	assert.Equal(t, 3, c.data.Len())
	assert.True(t, c.Add(7, 7, 10))
	assert.True(t, c.Contains(7))
	assert.Equal(t, 1, c.data.Len())
	_, exist := c.Del(7)
	assert.True(t, exist)
	_, exist = c.Del(6)
	assert.False(t, exist)
}

func testChannelVersion(t *testing.T, channel, version int32) {

	chanVer := CalcChannelVersion(channel, version)
	chann, ver := DecodeChannelVersion(chanVer)

	assert.True(t, chann == channel)
	assert.True(t, ver == version)
}

func Test_ChannelVersion(t *testing.T) {

	testChannelVersion(t, 0, 100)
	testChannelVersion(t, 128, 100)
}

func TestFilter(t *testing.T) {
	filter := NewFilter(10)
	go filter.ManageRecvFilter(time.Millisecond)
	defer filter.Close()
	filter.GetAtomicLock()

	now := time.Now().Unix()
	assert.Equal(t, false, filter.Add("key", now))
	assert.Equal(t, true, filter.Contains("key"))
	val, ok := filter.Get("key")
	assert.Equal(t, true, ok)
	assert.Equal(t, now, val.(int64))
	filter.Remove("key")
	assert.Equal(t, false, filter.Contains("key"))
	filter.ReleaseAtomicLock()
	assert.Equal(t, false, filter.AddWithCheckAtomic("key", time.Now().Unix()))
	assert.Equal(t, true, filter.AddWithCheckAtomic("key", time.Now().Unix()))
	assert.False(t, filter.isClose())
	time.Sleep(time.Millisecond * 10)
}
