package download

import (
	"sync"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
)

// jobs datastruct
type jobs []*JobInfo

type JobInfo struct {
	Id      string
	Limit   int
	Pid     peer.ID
	Index   int
	Latency time.Duration
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
