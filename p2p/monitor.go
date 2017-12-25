package p2p

import (
	"sync"
	"time"
)

type Monitor struct {
	mtx       sync.Mutex
	count     uint
	done      chan bool
	isrunning bool
	lastok    time.Duration
	lastop    time.Duration
}

func (m *Monitor) Update(op bool) {

	m.mtx.Lock()
	defer m.mtx.Unlock()
	if op {
		m.lastok = time.Duration(time.Now().Unix())
		if m.count >= 1 {
			m.count--
		} else {
			m.count = 0
		}
	}

	m.lastop = time.Duration(time.Now().Unix())
	if !op {
		m.count++
	}

}

func NewMonitor() *Monitor {

	var m = &Monitor{
		done:      make(chan bool),
		lastok:    time.Duration(time.Now().Unix()),
		lastop:    time.Duration(time.Now().Unix()),
		count:     0,
		isrunning: true,
	}
	m.Start()
	return m
}

func (m *Monitor) Start() {
	go func(m *Monitor) {
		for {
			tick := time.NewTicker(time.Second * 5)
			select {
			case <-tick.C:
				m.ChangeRunning()
			case <-m.done:
				break
			}
		}

	}(m)

}

func (m *Monitor) GetCount() uint {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	return m.count
}
func (m *Monitor) GetLastOp() time.Duration {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	return m.lastop
}
func (m *Monitor) GetLastOk() time.Duration {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	return m.lastok
}
func (m *Monitor) MonitorInfo() *Monitor {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	return m
}
func (m *Monitor) ChangeRunning() {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	log.Debug("Monitor", "count", m.count)
	if m.lastop-m.lastok > 600 || m.count > 30 {
		m.isrunning = false
	}
}
func (m *Monitor) IsRunning() bool {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	return m.isrunning
}
func (m *Monitor) Stop() {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	m.isrunning = false
	m.done <- false
}
