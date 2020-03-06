package download

import (
	"testing"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/stretchr/testify/assert"
)

/*
// jobs datastruct
	type jobs []*JobInfo

	type JobInfo struct {
		Id      string        //一次下载任务的任务ID
		Limit   int           //节点同时最大处理任务数量
		Pid     peer.ID       //节点ID
		Index   int           // 任务在任务列表中索引
		Latency time.Duration // 任务所在节点的时延
		mtx     sync.Mutex
	}

*/
func testJobs(t *testing.T) {
	pid1, _ := peer.IDB58Decode("16Uiu2HAmGpMpYDDidb27555ALTx7a1aZbqYDa7B2EUwwCiBcL67M")
	pid2, _ := peer.IDB58Decode("16Uiu2HAmTdgKpRmE6sXj512HodxBPMZmjh6vHG1m4ftnXY3wLSpg")
	pid3, _ := peer.IDB58Decode("16Uiu2HAm45QtjUVYxnc3eqfHoE4eSFovSh99SgsoF6Qm1eRXTd5W")

	t1 := &JobInfo{
		Id:      "123456",
		Limit:   10,
		Pid:     pid1,
		Index:   0,
		Latency: time.Duration(time.Second * 10),
	}
	t2 := &JobInfo{
		Id:      "123456",
		Limit:   11,
		Pid:     pid2,
		Index:   1,
		Latency: time.Duration(time.Second * 5),
	}
	t3 := &JobInfo{
		Id:      "123456",
		Limit:   14,
		Pid:     pid3,
		Index:   2,
		Latency: time.Duration(time.Second * 20),
	}

	var myjobs jobs
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
	myjobs = myjobs.Remove(4)
	assert.Equal(t, 3, myjobs.Len())
	myjobs = myjobs.Remove(0)
	assert.Equal(t, 2, myjobs.Len())

}

func TestJob(t *testing.T) {
	testJobs(t)
}
