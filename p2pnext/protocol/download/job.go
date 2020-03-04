package download

import (
	"sort"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
)

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

//Len size of the Invs data
func (i jobs) Len() int {
	return len(i)
}

//Less Sort from low to high
func (i jobs) Less(a, b int) bool {
	return i[a].Latency < i[b].Latency
}

//Swap  the param
func (i jobs) Swap(a, b int) {
	i[a], i[b] = i[b], i[a]
}

func (j jobs) Remove(index int) jobs {
	j = append(j[:index], j[index+1:]...)
	return j

}
func (i jobs) Sort() jobs {
	sort.Sort(i)
	return i
}

func (i jobs) Size() int {
	return len(i)
}

func (d *DownloadProtol) initJob(pids []string, jobId string) jobs {
	var JobPeerIds jobs
	var pIDs []peer.ID
	for _, pid := range pids {
		pID, err := peer.IDB58Decode(pid)
		if err != nil {
			continue
		}
		pIDs = append(pIDs, pID)
	}
	if len(pIDs) == 0 {
		pIDs = d.ConnManager.Fetch()

	}
	latency := d.GetConnsManager().GetLatencyByPeer(pIDs)
	for _, pID := range pIDs {
		if pID.Pretty() == d.GetHost().ID().Pretty() {
			continue
		}
		var job JobInfo
		job.Pid = pID
		job.Id = jobId
		var ok bool
		latency, ok := latency[job.Pid.Pretty()]
		if ok {
			if latency == 0 {
				continue
			}
			job.Latency = latency
		}
		job.Limit = 0
		JobPeerIds = append(JobPeerIds, &job)
	}

	return JobPeerIds
}

func (d *DownloadProtol) CheckJob(jobID string, pids []string, faildJobs sync.Map) {
	defer faildJobs.Delete(jobID)
	v, ok := faildJobs.Load(jobID)
	if !ok {
		return
	}
	faildJob := v.(sync.Map)

	faildJob.Range(func(k, v interface{}) bool {
		blockheight := k.(int64)
		jobS := d.initJob(pids, jobID)
		log.Warn("CheckJob<<<<<<<<<<", "jobID", jobID, "faildJob", blockheight)

		d.syncDownloadBlock(blockheight, jobS)

		return true

	})

}

func (d *DownloadProtol) availbJob(js jobs, blockheight int64) *JobInfo {

	var limit int
	if len(js) > 10 {
		limit = 20 //节点数大于10，每个节点限制最大下载任务数为20个
	} else {
		limit = 50 //节点数较少，每个节点节点最大下载任务数位50个
	}
	for i, job := range js {
		//check blockheight
		peerInfo := d.GetPeerInfoManager().Get(job.Pid.Pretty())
		if peerInfo != nil {
			if peerInfo.GetHeader().GetHeight() < blockheight {
				continue
			}
		} else {
			log.Error("CheckAvailbJob", "PeerInfoManager No ths Peer info...", job.Pid.Pretty())
			continue
		}

		if job.Limit < limit {
			job.mtx.Lock()
			job.Limit++
			job.mtx.Unlock()
			job.Index = i
			log.Debug("getFreeJob", " limit", job.Limit, "latency", job.Latency, "peerid", job.Pid)
			return job
		}
	}

	return nil

}

func (d *DownloadProtol) releaseJob(js *JobInfo) {
	js.mtx.Lock()
	defer js.mtx.Unlock()
	js.Limit--
	if js.Limit < 0 {
		js.Limit = 0
	}
}
