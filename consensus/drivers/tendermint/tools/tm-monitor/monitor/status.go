package monitor

import (
	"sync"
	"time"
)

type Health int

const (
	// FullHealth means all nodes online, synced, validators making blocks
	FullHealth = Health(0)
	// ModerateHealth means we're making blocks
	ModerateHealth = Health(1)
	// Dead means we're not making blocks due to all validators freezing or crashing
	Dead = Health(2)
)

type Status struct {
	Height int64 `json:"height"`

	AvgBlockTime    float64 `json:"avg_block_time" amino:"unsafe"`    // ms (avg over last minute)
	AvgTxThroughput float64 `json:"avg_tx_throughput" amino:"unsafe"` // tx/s (avg over last minute)
	AvgBlockLatency float64 `json:"avg_block_latency" amino:"unsafe"` // ms (avg over last minute)

	NumValidators           int `json:"num_validators"`
	NumNodesMonitored       int `json:"num_nodes_monitored"`
	NumNodesMonitoredOnline int `json:"num_nodes_monitored_online"`

	Health Health `json:"health"`

	UptimeData *UptimeData `json:"uptime_data"`

	nodeStatusMap map[string]bool

	mu sync.Mutex
}

func (n *Status) NewNode(name string) {
	n.NumNodesMonitored++
	n.NumNodesMonitoredOnline++
}

func NewStatus() *Status {
	return &Status{
		Health: FullHealth,
		UptimeData: &UptimeData{
			StartTime: time.Now(),
			Uptime:    100.0,
		},
		nodeStatusMap: make(map[string]bool),
	}
}

func (s *Status) StartTime() time.Time {
	return s.UptimeData.StartTime
}

func (s *Status) Uptime() float64 {
	return s.UptimeData.Uptime
}

type UptimeData struct {
	StartTime time.Time `json:"start_time"`
	Uptime    float64   `json:"uptime" amino:"unsafe"` // percentage of time we've been healthy, ever

	totalDownTime time.Duration // total downtime (only updated when we come back online)
	wentDown      time.Time
}

func (s *Status) NodeDeleted(name string) {
	s.NumNodesMonitored--
	s.NumNodesMonitoredOnline--
}

func (s *Status) NodeIsOnline(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, ok := s.nodeStatusMap[name]
	if !ok {
		s.nodeStatusMap[name] = true
	}

	online, ok := s.nodeStatusMap[name]
	if ok && !online {
		s.nodeStatusMap[name] = true
		s.NumNodesMonitoredOnline++
		s.UptimeData.totalDownTime += time.Since(s.UptimeData.wentDown)
		s.updateHealth()
	}
}

func (s *Status) NodeIsDown(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if online, ok := s.nodeStatusMap[name]; !ok || online {
		s.nodeStatusMap[name] = false
		s.NumNodesMonitoredOnline--
		s.UptimeData.wentDown = time.Now()
		s.updateHealth()
	}
}

func (s *Status) updateHealth() {
	// if we are connected to all validators, we're at full health
	// TODO: make sure they're all at the same height (within a block)
	// and all proposing (and possibly validating ) Alternatively, just
	// check there hasn't been a new round in numValidators rounds
	// if s.NumValidators != 0 && s.NumNodesMonitoredOnline == s.NumValidators {
	if s.NumNodesMonitoredOnline == s.NumNodesMonitored {
		s.Health = FullHealth
	} else if s.NumNodesMonitoredOnline > 0 && s.NumNodesMonitoredOnline <= s.NumNodesMonitored {
		s.Health = ModerateHealth
	} else {
		s.Health = Dead
	}
}

func (s *Status) GetHealthString() string {
	switch s.Health {
	case FullHealth:
		return "full"
	case ModerateHealth:
		return "moderate"
	case Dead:
		return "dead"
	default:
		return "undefined"
	}
}
