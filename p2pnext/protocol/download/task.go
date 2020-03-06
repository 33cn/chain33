package download

import (
	"sort"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
)

// task datastruct
type Tasks []*TaskInfo

type TaskInfo struct {
	Id      string        //一次下载任务的任务ID
	TaskNum int           //节点同时最大处理任务数量
	Pid     peer.ID       //节点ID
	Index   int           // 节点在任务列表中索引，方便下载失败后，把该节点从下载列表中删除
	Latency time.Duration // 任务所在节点的时延
	mtx     sync.Mutex
}

//Len size of the Invs data
func (i Tasks) Len() int {
	return len(i)
}

//Less Sort from low to high
func (i Tasks) Less(a, b int) bool {
	return i[a].Latency < i[b].Latency
}

//Swap  the param
func (i Tasks) Swap(a, b int) {
	i[a], i[b] = i[b], i[a]
}

func (j Tasks) Remove(index int) Tasks {
	if index+1 > j.Size() {
		return j
	}

	j = append(j[:index], j[index+1:]...)
	return j

}
func (i Tasks) Sort() Tasks {
	sort.Sort(i)
	return i
}

func (i Tasks) Size() int {
	return len(i)
}

func (d *DownloadProtol) initJob(pids []string, jobId string) Tasks {
	var JobPeerIds Tasks
	var pIDs []peer.ID
	for _, pid := range pids {
		pID, err := peer.IDB58Decode(pid)
		if err != nil {
			continue
		}
		pIDs = append(pIDs, pID)
	}
	if len(pIDs) == 0 {
		pIDs = d.ConnManager.FindNearestPeers()

	}
	latency := d.GetConnsManager().GetLatencyByPeer(pIDs)
	for _, pID := range pIDs {
		if pID.Pretty() == d.GetHost().ID().Pretty() {
			continue
		}
		var job TaskInfo
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
		job.TaskNum = 0
		JobPeerIds = append(JobPeerIds, &job)
	}

	return JobPeerIds
}

func (d *DownloadProtol) CheckTask(taskID string, pids []string, faildJobs sync.Map) {
	defer faildJobs.Delete(taskID)
	v, ok := faildJobs.Load(taskID)
	if !ok {
		return
	}
	faildJob := v.(sync.Map)

	faildJob.Range(func(k, v interface{}) bool {
		blockheight := k.(int64)
		jobS := d.initJob(pids, taskID)
		log.Warn("CheckTask<<<<<<<<<<", "taskID", taskID, "faildJob", blockheight)
		d.downloadBlock(blockheight, jobS)
		return true

	})

}

func (d *DownloadProtol) availbTask(ts Tasks, blockheight int64) *TaskInfo {

	var limit int
	if len(ts) > 10 {
		limit = 20 //节点数大于10，每个节点限制最大下载任务数为20个
	} else {
		limit = 50 //节点数较少，每个节点节点最大下载任务数位50个
	}
	for i, task := range ts {
		//check blockheight
		peerInfo := d.GetPeerInfoManager().GetPeerInfoInMin(task.Pid.Pretty())
		if peerInfo != nil {
			if peerInfo.GetHeader().GetHeight() < blockheight {
				continue
			}
		} else {
			log.Error("CheckAvailbJob", "PeerInfoManager No ths Peer info...", task.Pid.Pretty())
			continue
		}

		if task.TaskNum < limit {
			task.mtx.Lock()
			task.TaskNum++
			task.mtx.Unlock()
			task.Index = i
			log.Debug("getFreeJob", " taskNum", task.TaskNum, "latency", task.Latency, "peerid", task.Pid)
			return task
		}
	}

	return nil

}

func (d *DownloadProtol) releaseJob(js *TaskInfo) {
	js.mtx.Lock()
	defer js.mtx.Unlock()
	js.TaskNum--
	if js.TaskNum < 0 {
		js.TaskNum = 0
	}
}
