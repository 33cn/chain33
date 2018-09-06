package monitor

import (
	"sync"
	"time"

	"net"
	"strings"
	"syscall"

	"github.com/tendermint/tendermint/libs/log"
	"gitlab.33.cn/chain33/chain33/rpc"
)

// waiting more than this many seconds for a block means we're unhealthy

type Monitor struct {
	mtx   sync.Mutex
	Nodes []*Node

	Status *Status

	monitorQuit chan struct{}            // monitor exitting
	nodeQuit    map[string]chan struct{} // node is being stopped and removed from under the monitor

	recalculateNetworkUptimeEvery time.Duration
	numValidatorsUpdateInterval   time.Duration

	logger log.Logger
}

func NewMonitor(options ...func(*Monitor)) *Monitor {
	m := &Monitor{
		Nodes:                         make([]*Node, 0),
		Status:                        NewStatus(),
		monitorQuit:                   make(chan struct{}),
		nodeQuit:                      make(map[string]chan struct{}),
		recalculateNetworkUptimeEvery: 10 * time.Second,
		numValidatorsUpdateInterval:   5 * time.Second,
		logger: log.NewNopLogger(),
	}

	for _, option := range options {
		option(m)
	}

	return m
}

func (m *Monitor) Monitor(n *Node) error {
	m.mtx.Lock()
	m.Nodes = append(m.Nodes, n)
	m.mtx.Unlock()

	m.Status.NewNode(n.Name)
	m.nodeQuit[n.Name] = make(chan struct{})

	go m.getInfo(n)

	return nil
}

func (m *Monitor) Stop() {
	close(m.monitorQuit)

	for _, n := range m.Nodes {
		m.Unmonitor(n)
	}
}

func (m *Monitor) Unmonitor(n *Node) {
	m.Status.NodeDeleted(n.Name)

	n.Stop()
	if m.nodeQuit[n.Name] != nil {
		close(m.nodeQuit[n.Name])
		delete(m.nodeQuit, n.Name)
	}

	i, _ := m.NodeByName(n.Name)

	m.mtx.Lock()
	m.Nodes[i] = m.Nodes[len(m.Nodes)-1]
	m.Nodes = m.Nodes[:len(m.Nodes)-1]
	m.mtx.Unlock()
}

func (m *Monitor) NodeByName(name string) (index int, node *Node) {
	m.mtx.Lock()
	defer m.mtx.Unlock()

	for i, n := range m.Nodes {
		if name == n.Name {
			return i, n
		}
	}
	return -1, nil
}

func (m *Monitor) getInfo(n *Node) {
	for {
		err := m.GetMemInfo(n)
		if err != nil {
			m.logger.Error("Get member info failed...", "err", err)
		}
		header, err := m.GetLastHeader(n)
		if err != nil {
			m.Status.NodeIsDown(n.Name)
			n.Online = false
			continue
		}

		// Update the status info if receive the new block
		m.Status.NodeIsOnline(n.Name)
		m.NodeIsOnline(n.Name)
		if header.Height != n.Height {
			m.Status.Height = header.Height
			n.UpdateBlockTimeArr(header.BlockTime)
			n.UpdateStatus(header)
			n.BlockLatency = n.GetAverageBlockTime()
		}
	}
}

func (m *Monitor) NodeIsOnline(name string) {
	_, node := m.NodeByName(name)
	if nil != node {

		online, ok := m.Status.nodeStatusMap[name]
		if ok && online {
			m.mtx.Lock()
			node.Online = online
			m.mtx.Unlock()
		}
	}
}

//func (m *Monitor) listen(nodeName string, blockCh <-chan tmtypes.Header, blockLatencyCh <-chan float64, disconnectCh <-chan bool, quit <-chan struct{}) {
//	for {
//		select {
//		case <-quit:
//			return
//		case b := <-blockCh:
//			m.status.NewBlock(b)
//			m.status.NodeIsOnline(nodeName)
//			m.NodeIsOnline(nodeName)
//		case l := <-blockLatencyCh:
//			m.status.NewBlockLatency(l)
//			m.status.NodeIsOnline(nodeName)
//			m.NodeIsOnline(nodeName)
//		case disconnected := <-disconnectCh:
//			if disconnected {
//				m.status.NodeIsDown(nodeName)
//			} else {
//				m.status.NodeIsOnline(nodeName)
//				m.NodeIsOnline(nodeName)
//			}
//		case <-time.After(nodeLivenessTimeout):
//			mlog.Info("event", fmt.Sprintf("node was not responding for %v", nodeLivenessTimeout))
//			m.Network.NodeIsDown(nodeName)
//		}
//	}
//}

func (n *Node) UpdateStatus(header rpc.Header) {
	n.Name = n.rpcAddr
	n.Height = header.Height
	n.BlockTime = header.BlockTime - n.LastBlockTime
	n.LastBlockTime = header.BlockTime
	n.StateHash = header.StateHash
	n.TxCount = header.TxCount
	n.Online = true
}

func GetJsonClient(n *Node) (*rpc.JSONClient, error) {
	addr := GetIPAndPort(n.Name)
	// In go 1.10, if the destination is unreachable, the response will not return until timeout.
	// Add a check in this place , it will return immediately if the connection is wrong.
	err := CheckConnection(addr)
	if err != nil {
		n.logger.Error("Check connection failed...", "err", err)
		return nil, err
	}

	client, err := rpc.NewJSONClient(n.rpcAddr)
	if err != nil {
		n.logger.Error("New json client failed...", "err", err)
		return nil, err
	}

	return client, nil
}

func (m *Monitor) GetPeerInfo(n *Node) (rpc.PeerList, error) {
	err := CheckConnection(n.rpcAddr)
	if err != nil {
		return rpc.PeerList{}, err
	} else {
		var res rpc.PeerList
		err = JsonClientCallBack(n.rpcAddr, "Chain33.GetPeerInfo", nil, &res)
		if err != nil {
			m.logger.Error("Get peer info failed...", "err", err)
			return rpc.PeerList{}, err
		}
		return res, nil
	}

}

func (m *Monitor) GetLastHeader(n *Node) (rpc.Header, error) {
	_, err := GetJsonClient(n)
	if err != nil {
		m.logger.Error("Get json client failed...", "err", err)
		return rpc.Header{}, err
	} else {
		var header rpc.Header
		err = JsonClientCallBack(n.rpcAddr, "Chain33.GetLastHeader", nil, &header)
		if err != nil {
			m.logger.Error("Get last header", "err", err)
			return rpc.Header{}, err
		}
		return header, nil
	}
}

func GetIPAndPort(input string) string {
	left := strings.LastIndex(input, "/")
	if left > 0 {
		return input[left+1 : len(input)]
	}
	return ""
}

func GetIP(input string) string {
	left := strings.LastIndex(input, "/")
	right := strings.LastIndex(input, ":")
	if left > 0 && right > 0 {
		return input[left+1 : right]
	}
	return ""
}

func GetPort(input string) string {
	left := strings.LastIndex(input, ":")
	if left > 0 {
		return input[left+1 : len(input)]
	}
	return ""
}

func CheckConnection(addr string) error {
	conn, err := net.DialTimeout("tcp", addr, time.Millisecond*500)
	if err != nil {
		return err
	}
	if conn != nil {
		conn.Close()
	}
	return nil
}

func (m *Monitor) GetMemInfo(n *Node) error {
	sysInfo := new(syscall.Sysinfo_t)
	err := syscall.Sysinfo(sysInfo)
	if err == nil {
		n.MemInfo.All = sysInfo.Totalram / 1024 / 1024
		n.MemInfo.Free = sysInfo.Freeram / 1024 / 1024 // * uint64(syscall.Getpagesize())
		n.MemInfo.Used = n.MemInfo.All - n.MemInfo.Free
		return nil
	}
	return err
}

func (m *Monitor) SetLogger(l log.Logger) {
	m.logger = l
}
