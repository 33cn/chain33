package tendermint

import (
	"encoding/hex"
	"fmt"
	"math"
	"net"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/golang/protobuf/proto"
	"gitlab.33.cn/chain33/chain33/common/crypto"
	ttypes "gitlab.33.cn/chain33/chain33/plugin/consensus/tendermint/types"
	tmtypes "gitlab.33.cn/chain33/chain33/plugin/dapp/valnode/types"
)

const (
	numBufferedConnections             = 10
	MaxNumPeers                        = 50
	tryListenSeconds                   = 5
	HandshakeTimeout                   = 20 // * time.Second,
	maxSendQueueSize                   = 1024
	defaultSendTimeout                 = 60 * time.Second
	MaxMsgPacketPayloadSize            = 10 * 1024 * 1024
	DefaultDialTimeout                 = 3 * time.Second
	dialRandomizerIntervalMilliseconds = 3000
	// repeatedly try to reconnect for a few minutes
	// ie. 5 * 20 = 100s
	reconnectAttempts = 20
	reconnectInterval = 5 * time.Second

	// then move into exponential backoff mode for ~1day
	// ie. 3**10 = 16hrs
	reconnectBackOffAttempts    = 10
	reconnectBackOffBaseSeconds = 3

	minReadBufferSize  = 1024
	minWriteBufferSize = 65536

	broadcastEvidenceIntervalS = 60 // broadcast uncommitted evidence this often
)

func Parallel(tasks ...func()) {
	var wg sync.WaitGroup
	wg.Add(len(tasks))
	for _, task := range tasks {
		go func(task func()) {
			task()
			wg.Done()
		}(task)
	}
	wg.Wait()
}

func GenAddressByPubKey(pubkey crypto.PubKey) []byte {
	//must add 3 bytes ahead to make compatibly
	typeAddr := append([]byte{byte(0x01), byte(0x01), byte(0x20)}, pubkey.Bytes()...)
	return crypto.Ripemd160(typeAddr)
}

type IP2IPPort struct {
	mutex   sync.RWMutex
	mapList map[string]string
}

func NewMutexMap() *IP2IPPort {
	return &IP2IPPort{
		mapList: make(map[string]string),
	}
}

func (ipp *IP2IPPort) Has(ip string) bool {
	ipp.mutex.RLock()
	defer ipp.mutex.RUnlock()
	_, ok := ipp.mapList[ip]
	return ok
}

func (ipp *IP2IPPort) Set(ip string, ipport string) {
	ipp.mutex.Lock()
	defer ipp.mutex.Unlock()
	ipp.mapList[ip] = ipport
}

func (ipp *IP2IPPort) Delete(ip string) {
	ipp.mutex.Lock()
	defer ipp.mutex.Unlock()
	delete(ipp.mapList, ip)
}

type NodeInfo struct {
	ID      ID     `json:"id"`
	Network string `json:"network"`
	Version string `json:"version"`
	IP      string `json:"ip,omitempty"`
}

type Node struct {
	listener    net.Listener
	connections chan net.Conn
	privKey     crypto.PrivKey
	Network     string
	Version     string
	ID          ID
	IP          string //get ip from connect to ourself

	localIPs map[string]net.IP
	peerSet  *PeerSet

	dialing      *IP2IPPort
	reconnecting *IP2IPPort

	seeds    []string
	protocol string
	lAddr    string

	state            *ConsensusState
	evpool           *EvidencePool
	broadcastChannel chan MsgInfo
	started          uint32 // atomic
	stopped          uint32 // atomic
	quit             chan struct{}
}

func NewNode(seeds []string, protocol string, lAddr string, privKey crypto.PrivKey, network string, version string, state *ConsensusState, evpool *EvidencePool) *Node {
	address := GenAddressByPubKey(privKey.PubKey())

	node := &Node{
		peerSet:     NewPeerSet(),
		seeds:       seeds,
		protocol:    protocol,
		lAddr:       lAddr,
		connections: make(chan net.Conn, numBufferedConnections),

		privKey:          privKey,
		Network:          network,
		Version:          version,
		ID:               ID(hex.EncodeToString(address)),
		dialing:          NewMutexMap(),
		reconnecting:     NewMutexMap(),
		broadcastChannel: make(chan MsgInfo, maxSendQueueSize),
		state:            state,
		evpool:           evpool,
		localIPs:         make(map[string]net.IP),
	}

	state.SetOurID(node.ID)
	state.SetBroadcastChannel(node.broadcastChannel)

	localIPs := getNaiveExternalAddress(true)
	if len(localIPs) > 0 {
		for _, item := range localIPs {
			node.localIPs[item.String()] = item
		}
	}
	return node
}

func (node *Node) Start() {
	if atomic.CompareAndSwapUint32(&node.started, 0, 1) {
		// Create listener
		var listener net.Listener
		var err error
		for i := 0; i < tryListenSeconds; i++ {
			listener, err = net.Listen(node.protocol, node.lAddr)
			if err == nil {
				break
			} else if i < tryListenSeconds-1 {
				time.Sleep(time.Second * 1)
			}
		}
		if err != nil {
			panic(err)
		}
		node.listener = listener
		// Actual listener local IP & port
		listenerIP, listenerPort := splitHostPort(listener.Addr().String())
		tendermintlog.Info("Local listener", "ip", listenerIP, "port", listenerPort)

		go node.listenRoutine()

		for i := 0; i < len(node.seeds); i++ {
			go func(i int) {
				addr := node.seeds[i]
				ip, _ := splitHostPort(addr)
				_, ok := node.localIPs[ip]
				if ok {
					tendermintlog.Info("find our ip ", "ourip", ip)
					node.IP = ip
					return
				}

				randomSleep(0)
				err := node.DialPeerWithAddress(addr)
				if err != nil {
					tendermintlog.Debug("Error dialing peer", "err", err)
				}
			}(i)
		}

		go node.StartConsensusRoutine()
		go node.BroadcastRoutine()
		go node.evidenceBroadcastRoutine()
	}
}

func (node *Node) DialPeerWithAddress(addr string) error {
	ip, _ := splitHostPort(addr)
	node.dialing.Set(ip, addr)
	defer node.dialing.Delete(ip)
	return node.addOutboundPeerWithConfig(addr)
}

func (node *Node) addOutboundPeerWithConfig(addr string) error {
	tendermintlog.Info("Dialing peer", "address", addr)

	peerConn, err := newOutboundPeerConn(addr, node.privKey, node.StopPeerForError, node.state, node.evpool)
	if err != nil {
		go node.reconnectToPeer(addr)
		return err
	}

	if err := node.addPeer(peerConn); err != nil {
		peerConn.CloseConn()
		return err
	}
	return nil
}

func (node *Node) Stop() {
	atomic.CompareAndSwapUint32(&node.stopped, 0, 1)
	node.listener.Close()
	if node.quit != nil {
		close(node.quit)
	}
	// Stop peers
	for _, peer := range node.peerSet.List() {
		peer.Stop()
		node.peerSet.Remove(peer)
	}
	// Stop reactors
	tendermintlog.Debug("Switch: Stopping reactors")
}

func (node *Node) IsRunning() bool {
	return atomic.LoadUint32(&node.started) == 1 && atomic.LoadUint32(&node.stopped) == 0
}

func (node *Node) listenRoutine() {
	for {
		conn, err := node.listener.Accept()

		if !node.IsRunning() {
			break // Go to cleanup
		}

		// listener wasn't stopped,
		// yet we encountered an error.
		if err != nil {
			panic(err)
		}

		go node.connectComming(conn)
	}

	// Cleanup
	close(node.connections)
	for range node.connections {
		// Drain
	}
}

func (node *Node) StartConsensusRoutine() {
	for {
		if node.peerSet.Size() >= 0 {
			node.state.Start()
			break
		}
		time.Sleep(1 * time.Second)
	}
}

func (node *Node) evidenceBroadcastRoutine() {
	ticker := time.NewTicker(time.Second * broadcastEvidenceIntervalS)
	for {
		select {
		case evidence := <-node.evpool.EvidenceChan():
			// broadcast some new evidence
			data, err := proto.Marshal(evidence.Child())
			if err != nil {
				msg := MsgInfo{TypeID: ttypes.EvidenceListID,
					Msg: &tmtypes.EvidenceData{
						Evidence: []*tmtypes.EvidenceEnvelope{
							{
								TypeName: evidence.TypeName(),
								Data:     data,
							},
						},
					},
					PeerID: node.ID, PeerIP: node.IP,
				}
				node.Broadcast(msg)

				// TODO: Broadcast runs asynchronously, so this should wait on the successChan
				// in another routine before marking to be proper.
				node.evpool.evidenceStore.MarkEvidenceAsBroadcasted(evidence)
			}

		case <-ticker.C:
			// broadcast all pending evidence
			var eData tmtypes.EvidenceData
			evidence := node.evpool.PendingEvidence()
			for _, item := range evidence {
				ev := item.Child()
				if ev != nil {
					data, err := proto.Marshal(ev)
					if err != nil {
						panic("AddEvidence marshal failed")
					}
					env := &tmtypes.EvidenceEnvelope{
						TypeName: item.TypeName(),
						Data:     data,
					}
					eData.Evidence = append(eData.Evidence, env)
				}
			}
			msg := MsgInfo{TypeID: ttypes.EvidenceListID,
				Msg:    &eData,
				PeerID: node.ID,
				PeerIP: node.IP,
			}
			node.Broadcast(msg)
		case _, ok := <-node.quit:
			if !ok {
				node.quit = nil
				tendermintlog.Info("evidenceBroadcastRoutine quit")
				return
			}
		}
	}
}

func (node *Node) BroadcastRoutine() {
	for {
		msg, ok := <-node.broadcastChannel
		if !ok {
			tendermintlog.Debug("broadcastChannel closed")
			return
		}
		node.Broadcast(msg)
	}
}

func (node *Node) connectComming(inConn net.Conn) {
	maxPeers := MaxNumPeers
	if maxPeers <= node.peerSet.Size() {
		tendermintlog.Debug("Ignoring inbound connection: already have enough peers", "address", inConn.RemoteAddr().String(), "numPeers", node.peerSet.Size(), "max", maxPeers)
		return
	}

	// New inbound connection!
	err := node.addInboundPeer(inConn)
	if err != nil {
		tendermintlog.Info("Ignoring inbound connection: error while adding peer", "address", inConn.RemoteAddr().String(), "err", err)
		return
	}
}

func (node *Node) stopAndRemovePeer(peer Peer, reason interface{}) {
	node.peerSet.Remove(peer)
	peer.Stop()
}

func (node *Node) StopPeerForError(peer Peer, reason interface{}) {
	tendermintlog.Error("Stopping peer for error", "peer", peer, "err", reason)
	addr, err := peer.RemoteAddr()
	node.stopAndRemovePeer(peer, reason)

	if peer.IsPersistent() {
		if err == nil && addr != nil {
			go node.reconnectToPeer(addr.String())
		}
	}
}

func (node *Node) addInboundPeer(conn net.Conn) error {
	peerConn, err := newInboundPeerConn(conn, node.privKey, node.StopPeerForError, node.state, node.evpool)
	if err != nil {
		conn.Close()
		return err
	}
	if err = node.addPeer(peerConn); err != nil {
		peerConn.CloseConn()
		return err
	}

	return nil
}

// addPeer checks the given peer's validity, performs a handshake, and adds the
// peer to the switch and to all registered reactors.
// NOTE: This performs a blocking handshake before the peer is added.
// NOTE: If error is returned, caller is responsible for calling peer.CloseConn()
func (node *Node) addPeer(pc peerConn) error {
	addr := pc.conn.RemoteAddr()
	if err := node.FilterConnByAddr(addr); err != nil {
		return err
	}

	remoteIP, rErr := pc.RemoteIP()

	nodeinfo := NodeInfo{
		ID:      node.ID,
		Network: node.Network,
		Version: node.Version,
	}
	// Exchange NodeInfo on the conn
	peerNodeInfo, err := pc.HandshakeTimeout(nodeinfo, HandshakeTimeout*time.Second)
	if err != nil {
		return err
	}

	peerID := peerNodeInfo.ID

	// ensure connection key matches self reported key
	connID := pc.ID()

	if peerID != connID {
		return fmt.Errorf(
			"nodeInfo.ID() (%v) doesn't match conn.ID() (%v)",
			peerID,
			connID,
		)
	}

	// Avoid self
	if node.ID == peerID {
		return fmt.Errorf("Connect to self: %v", addr)
	}

	// Avoid duplicate
	if node.peerSet.Has(peerID) {
		return fmt.Errorf("Duplicate peer ID %v", peerID)
	}

	// Check for duplicate connection or peer info IP.
	if rErr == nil && node.peerSet.HasIP(remoteIP) {
		return fmt.Errorf("Duplicate peer IP %v", remoteIP)
	} else if rErr != nil {
		return fmt.Errorf("get remote ip failed:%v", rErr)
	}

	// Filter peer against ID white list
	//if err := node.FilterConnByID(peerID); err != nil {
	//return err
	//}

	// Check version, chain id
	if err := node.CompatibleWith(peerNodeInfo); err != nil {
		return err
	}

	tendermintlog.Info("Successful handshake with peer", "peerNodeInfo", peerNodeInfo)

	// All good. Start peer
	if node.IsRunning() {
		pc.SetTransferChannel(node.state.peerMsgQueue)
		if err = node.startInitPeer(&pc); err != nil {
			return err
		}
	}

	// Add the peer to .peers.
	// We start it first so that a peer in the list is safe to Stop.
	// It should not err since we already checked peers.Has().
	if err := node.peerSet.Add(&pc); err != nil {
		return err
	}
	//node.metrics.Peers.Add(float64(1))

	tendermintlog.Info("Added peer", "peer", pc.ip)
	return nil
}

func (node *Node) Broadcast(msg MsgInfo) chan bool {
	successChan := make(chan bool, len(node.peerSet.List()))
	tendermintlog.Debug("Broadcast", "msgtype", msg.TypeID)
	var wg sync.WaitGroup
	for _, peer := range node.peerSet.List() {
		wg.Add(1)
		go func(peer Peer) {
			defer wg.Done()
			success := peer.Send(msg)
			successChan <- success
		}(peer)
	}
	go func() {
		wg.Wait()
		close(successChan)
	}()
	return successChan
}

func (node *Node) startInitPeer(peer *peerConn) error {
	err := peer.Start() // spawn send/recv routines
	if err != nil {
		// Should never happen
		tendermintlog.Error("Error starting peer", "peer", peer, "err", err)
		return err
	}

	return nil
}

func (node *Node) FilterConnByAddr(addr net.Addr) error {
	//if node.filterConnByAddr != nil {
	//	return node.filterConnByAddr(addr)
	//}
	return nil
}

func (node *Node) CompatibleWith(other NodeInfo) error {
	iMajor, iMinor, _, iErr := splitVersion(node.Version)
	oMajor, oMinor, _, oErr := splitVersion(other.Version)

	// if our own version number is not formatted right, we messed up
	if iErr != nil {
		return iErr
	}

	// version number must be formatted correctly ("x.x.x")
	if oErr != nil {
		return oErr
	}

	// major version must match
	if iMajor != oMajor {
		return fmt.Errorf("Peer is on a different major version. Got %v, expected %v", oMajor, iMajor)
	}

	// minor version can differ
	if iMinor != oMinor {
		// ok
	}

	// nodes must be on the same network
	if node.Network != other.Network {
		return fmt.Errorf("Peer is on a different network. Got %v, expected %v", other.Network, node.Network)
	}

	return nil
}

func (node *Node) reconnectToPeer(addr string) {
	host, _ := splitHostPort(addr)
	if node.reconnecting.Has(host) {
		return
	}
	node.reconnecting.Set(host, addr)
	defer node.reconnecting.Delete(host)

	start := time.Now()
	tendermintlog.Info("Reconnecting to peer", "addr", addr)
	for i := 0; i < reconnectAttempts; i++ {
		if !node.IsRunning() {
			return
		}

		ips, err := net.LookupIP(host)
		if err != nil {
			tendermintlog.Info("LookupIP failed", "host", host)
			continue
		}
		if node.peerSet.HasIP(ips[0]) {
			tendermintlog.Info("Reconnecting to peer exit, already connect to the peer", "peer", host)
			return
		}

		err = node.DialPeerWithAddress(addr)
		if err == nil {
			return // success
		}

		tendermintlog.Info("Error reconnecting to peer. Trying again", "tries", i, "err", err, "addr", addr)
		// sleep a set amount
		randomSleep(reconnectInterval)
		continue
	}

	tendermintlog.Error("Failed to reconnect to peer. Beginning exponential backoff",
		"addr", addr, "elapsed", time.Since(start))
	for i := 0; i < reconnectBackOffAttempts; i++ {
		if !node.IsRunning() {
			return
		}

		// sleep an exponentially increasing amount
		sleepIntervalSeconds := math.Pow(reconnectBackOffBaseSeconds, float64(i))
		time.Sleep(time.Duration(sleepIntervalSeconds) * time.Second)
		err := node.DialPeerWithAddress(addr)
		if err == nil {
			return // success
		}
		tendermintlog.Info("Error reconnecting to peer. Trying again", "tries", i, "err", err, "addr", addr)
	}
	tendermintlog.Error("Failed to reconnect to peer. Giving up", "addr", addr, "elapsed", time.Since(start))
}

//---------------------------------------------------------------------
func randomSleep(interval time.Duration) {
	r := time.Duration(ttypes.RandInt63n(dialRandomizerIntervalMilliseconds)) * time.Millisecond
	time.Sleep(r + interval)
}

func isIpv6(ip net.IP) bool {
	v4 := ip.To4()
	if v4 != nil {
		return false
	}

	ipString := ip.String()

	// Extra check just to be sure it's IPv6
	return (strings.Contains(ipString, ":") && !strings.Contains(ipString, "."))
}

func getNaiveExternalAddress(defaultToIPv4 bool) []net.IP {
	var ips []net.IP
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		panic(fmt.Sprintf("Could not fetch interface addresses: %v", err))
	}

	for _, a := range addrs {
		ipnet, ok := a.(*net.IPNet)
		if !ok {
			continue
		}
		if defaultToIPv4 || !isIpv6(ipnet.IP) {
			v4 := ipnet.IP.To4()
			if v4 == nil || v4[0] == 127 {
				// loopback
				continue
			}
		} else if ipnet.IP.IsLoopback() {
			// IPv6, check for loopback
			continue
		}
		ips = append(ips, ipnet.IP)
	}
	return ips
}

func splitVersion(version string) (string, string, string, error) {
	spl := strings.Split(version, ".")
	if len(spl) != 3 {
		return "", "", "", fmt.Errorf("Invalid version format %v", version)
	}
	return spl[0], spl[1], spl[2], nil
}

func splitHostPort(addr string) (host string, port int) {
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		panic(err)
	}
	port, err = strconv.Atoi(portStr)
	if err != nil {
		panic(err)
	}
	return host, port
}

func dial(addr string) (net.Conn, error) {
	conn, err := net.DialTimeout("tcp", addr, DefaultDialTimeout)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func newOutboundPeerConn(addr string, ourNodePrivKey crypto.PrivKey, onPeerError func(Peer, interface{}), state *ConsensusState, evpool *EvidencePool) (peerConn, error) {
	conn, err := dial(addr)
	if err != nil {
		return peerConn{}, fmt.Errorf("Error creating peer:%v", err)
	}

	pc, err := newPeerConn(conn, true, true, ourNodePrivKey, onPeerError, state, evpool)
	if err != nil {
		if cerr := conn.Close(); cerr != nil {
			return peerConn{}, fmt.Errorf("newPeerConn failed:%v, connection close failed:%v", err, cerr)
		}
		return peerConn{}, err
	}

	return pc, nil
}

func newInboundPeerConn(
	conn net.Conn,
	ourNodePrivKey crypto.PrivKey,
	onPeerError func(Peer, interface{}),
	state *ConsensusState,
	evpool *EvidencePool,
) (peerConn, error) {

	// TODO: issue PoW challenge

	return newPeerConn(conn, false, false, ourNodePrivKey, onPeerError, state, evpool)
}

func newPeerConn(
	rawConn net.Conn,
	outbound, persistent bool,
	ourNodePrivKey crypto.PrivKey,
	onPeerError func(Peer, interface{}),
	state *ConsensusState,
	evpool *EvidencePool,
) (pc peerConn, err error) {
	conn := rawConn

	// Set deadline for secret handshake
	dl := time.Now().Add(HandshakeTimeout * time.Second)
	if err := conn.SetDeadline(dl); err != nil {
		return pc, fmt.Errorf("Error setting deadline while encrypting connection:%v", err)
	}

	// Encrypt connection
	conn, err = MakeSecretConnection(conn, ourNodePrivKey)
	if err != nil {
		return pc, fmt.Errorf("Error creating peer:%v", err)
	}

	// Only the information we already have
	return peerConn{
		outbound:    outbound,
		persistent:  persistent,
		conn:        conn,
		onPeerError: onPeerError,
		myState:     state,
		myevpool:    evpool,
	}, nil
}
