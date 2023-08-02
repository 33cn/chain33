package download

import (
	"fmt"
	"sync"
	"time"
)

func peersCounterKey(taskID, pid string) string {
	return fmt.Sprintf("%s-%s", taskID, pid)
}

type heightCostTime struct {
	height   int64
	costTime int64
}

// PeerTaskCounter ...
type PeerTaskCounter struct {
	pid             string
	latencies       []time.Duration
	taskID          string
	heightCostTimes []heightCostTime
}

// NewPeerTaskCounter ...
func NewPeerTaskCounter(pid, taskID string) *PeerTaskCounter {
	return &PeerTaskCounter{
		pid:             pid,
		taskID:          taskID,
		latencies:       []time.Duration{},
		heightCostTimes: []heightCostTime{},
	}
}

// Pretty print information
func (p *PeerTaskCounter) Pretty() string {
	return fmt.Sprintf("pid = %s, taskID = %s, latencys = %v, counter = %d, heightCostTimes = %v", p.pid, p.taskID, p.latencies, len(p.heightCostTimes), p.heightCostTimes)
}

// Append add counter info
func (p *PeerTaskCounter) Append(height, costTime int64) {
	p.heightCostTimes = append(p.heightCostTimes, heightCostTime{
		height:   height,
		costTime: costTime,
	})
}

// AppendLatency ...
func (p *PeerTaskCounter) AppendLatency(latency time.Duration) {
	p.latencies = append(p.latencies, latency)
}

// Counter ..
func (p *PeerTaskCounter) Counter() int64 {
	return int64(len(p.heightCostTimes))
}

// Counter ...
type Counter struct {
	taskCounter map[string][]string         //taskID:pid
	peerCounter map[string]*PeerTaskCounter //taskID-pid:PeerTaskCounter
	rw          sync.Mutex
}

// NewCounter ...
func NewCounter() *Counter {
	return &Counter{
		taskCounter: map[string][]string{},
		peerCounter: map[string]*PeerTaskCounter{},
		rw:          sync.Mutex{},
	}
}

// UpdateTaskInfo ...
func (c *Counter) UpdateTaskInfo(taskID, pid string, height, costTime int64) {
	c.rw.Lock()
	if counter, ok := c.peerCounter[peersCounterKey(taskID, pid)]; ok {
		counter.Append(height, costTime)
	}
	c.rw.Unlock()
}

// AddTaskInfo ...
func (c *Counter) AddTaskInfo(taskID, pid string, latency time.Duration) {
	c.rw.Lock()
	if ps, ok := c.taskCounter[taskID]; ok {
		c.taskCounter[taskID] = append(ps, pid)
	} else {
		c.taskCounter[taskID] = []string{pid}
	}
	key := peersCounterKey(taskID, pid)
	if counter, ok := c.peerCounter[key]; ok {
		counter.AppendLatency(latency)
	} else {
		counter = NewPeerTaskCounter(pid, taskID)
		counter.AppendLatency(latency)
		c.peerCounter[key] = counter
	}
	c.rw.Unlock()
}

// Release task by id
func (c *Counter) Release(tasksID string) {
	c.rw.Lock()
	defer c.rw.Unlock()
	if pids, ok := c.taskCounter[tasksID]; ok {
		for _, pid := range pids {
			key := peersCounterKey(tasksID, pid)
			if counter, ok := c.peerCounter[key]; ok {
				log.Info("Release", "Counter ", counter.Pretty())
				delete(c.peerCounter, key)
			}
		}
		delete(c.taskCounter, tasksID)
	}
}
