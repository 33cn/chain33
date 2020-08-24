package download

import (
	"testing"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/stretchr/testify/assert"
)

func testJobs(t *testing.T) {
	pid1, _ := peer.Decode("16Uiu2HAmGpMpYDDidb27555ALTx7a1aZbqYDa7B2EUwwCiBcL67M")
	pid2, _ := peer.Decode("16Uiu2HAmTdgKpRmE6sXj512HodxBPMZmjh6vHG1m4ftnXY3wLSpg")
	pid3, _ := peer.Decode("16Uiu2HAm45QtjUVYxnc3eqfHoE4eSFovSh99SgsoF6Qm1eRXTd5W")

	t1 := &taskInfo{
		ID:      "123456",
		TaskNum: 10,
		Pid:     pid1,
		Index:   0,
		Latency: time.Second * 10,
	}
	t2 := &taskInfo{
		ID:      "123456",
		TaskNum: 11,
		Pid:     pid2,
		Index:   1,
		Latency: time.Second * 5,
	}
	t3 := &taskInfo{
		ID:      "123456",
		TaskNum: 14,
		Pid:     pid3,
		Index:   2,
		Latency: time.Second * 20,
	}

	var myjobs tasks
	assert.Equal(t, myjobs.Len(), 0)
	myjobs = append(myjobs, t1, t2, t3)
	assert.Equal(t, myjobs.Len(), 3)

	assert.Equal(t, myjobs[0].Pid, pid1)

	assert.Equal(t, myjobs[1].Pid, pid2)

	assert.Equal(t, myjobs[2].Pid, pid3)
	//test sort
	myjobs.Sort()

	assert.Equal(t, myjobs[0].Pid, pid2)
	assert.Equal(t, myjobs[1].Pid, pid1)
	assert.Equal(t, myjobs[2].Pid, pid3)

	//test delete
	myjobs = myjobs.Remove(&taskInfo{Index: 4})
	assert.Equal(t, 3, myjobs.Len())
	myjobs = myjobs.Remove(&taskInfo{Index: 0})
	assert.Equal(t, 2, myjobs.Len())

}

func TestJob(t *testing.T) {
	testJobs(t)
}
