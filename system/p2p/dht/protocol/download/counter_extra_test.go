package download

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPeersCounterKey(t *testing.T) {
	key := peersCounterKey("task1", "peer1")
	assert.Equal(t, "task1-peer1", key)
}

func TestNewPeerTaskCounter(t *testing.T) {
	c := NewPeerTaskCounter("peer1", "task1")
	assert.NotNil(t, c)
	assert.Equal(t, "peer1", c.pid)
	assert.Equal(t, "task1", c.taskID)
}

func TestPeerTaskCounterAppend(t *testing.T) {
	c := NewPeerTaskCounter("p", "t")
	c.Append(100, 500)
	assert.Equal(t, 1, len(c.heightCostTimes))
	assert.Equal(t, int64(100), c.heightCostTimes[0].height)
	assert.Equal(t, int64(500), c.heightCostTimes[0].costTime)
}

func TestPeerTaskCounterAppendLatency(t *testing.T) {
	c := NewPeerTaskCounter("p", "t")
	c.AppendLatency(time.Millisecond * 100)
	assert.Equal(t, 1, len(c.latencies))
}

func TestPeerTaskCounterCounter(t *testing.T) {
	c := NewPeerTaskCounter("p", "t")
	assert.Equal(t, int64(0), c.Counter())
	c.Append(100, 500)
	assert.Equal(t, int64(1), c.Counter())
}

func TestPeerTaskCounterPretty(t *testing.T) {
	c := NewPeerTaskCounter("peer1", "task1")
	s := c.Pretty()
	assert.Contains(t, s, "peer1")
	assert.Contains(t, s, "task1")
}

func TestNewCounter(t *testing.T) {
	c := NewCounter()
	assert.NotNil(t, c)
	assert.NotNil(t, c.taskCounter)
	assert.NotNil(t, c.peerCounter)
}
