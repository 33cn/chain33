package tendermint

import (
	"bufio"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"reflect"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	ttypes "gitlab.33.cn/chain33/chain33/plugin/consensus/tendermint/types"
	tmtypes "gitlab.33.cn/chain33/chain33/plugin/dapp/valnode/types"
)

// ID is a hex-encoded crypto.Address
type ID string

// Messages in channels are chopped into smaller msgPackets for multiplexing.
type msgPacket struct {
	TypeID byte
	Bytes  []byte
}

type MsgInfo struct {
	TypeID byte
	Msg    proto.Message
	PeerID ID
	PeerIP string
}

type Peer interface {
	ID() ID
	RemoteIP() (net.IP, error) // remote IP of the connection
	RemoteAddr() (net.Addr, error)
	IsOutbound() bool
	IsPersistent() bool

	Send(msg MsgInfo) bool
	TrySend(msg MsgInfo) bool

	Stop()

	SetTransferChannel(chan MsgInfo)
	//Set(string, interface{})
	//Get(string) interface{}
}

type PeerConnState struct {
	mtx sync.Mutex
	ip  net.IP
	ttypes.PeerRoundState
}

type peerConn struct {
	outbound bool

	conn      net.Conn // source connection
	bufReader *bufio.Reader
	bufWriter *bufio.Writer

	persistent bool
	ip         net.IP
	id         ID

	sendQueue     chan MsgInfo
	sendQueueSize int32
	pongChannel   chan struct{}

	started uint32 //atomic
	stopped uint32 // atomic

	quit     chan struct{}
	waitQuit sync.WaitGroup

	transferChannel chan MsgInfo

	sendBuffer []byte

	onPeerError func(Peer, interface{})

	myState  *ConsensusState
	myevpool *EvidencePool

	state            *PeerConnState
	updateStateQueue chan MsgInfo
	heartbeatQueue   chan proto.Message
}

type PeerSet struct {
	mtx    sync.Mutex
	lookup map[ID]*peerSetItem
	list   []Peer
}

type peerSetItem struct {
	peer  Peer
	index int
}

func NewPeerSet() *PeerSet {
	return &PeerSet{
		lookup: make(map[ID]*peerSetItem),
		list:   make([]Peer, 0, 256),
	}
}

// Add adds the peer to the PeerSet.
// It returns an error carrying the reason, if the peer is already present.
func (ps *PeerSet) Add(peer Peer) error {
	ps.mtx.Lock()
	defer ps.mtx.Unlock()

	if ps.lookup[peer.ID()] != nil {
		return fmt.Errorf("Duplicate peer ID %v", peer.ID())
	}

	index := len(ps.list)
	// Appending is safe even with other goroutines
	// iterating over the ps.list slice.
	ps.list = append(ps.list, peer)
	ps.lookup[peer.ID()] = &peerSetItem{peer, index}
	return nil
}

// Has returns true if the set contains the peer referred to by this
// peerKey, otherwise false.
func (ps *PeerSet) Has(peerKey ID) bool {
	ps.mtx.Lock()
	_, ok := ps.lookup[peerKey]
	ps.mtx.Unlock()
	return ok
}

// HasIP returns true if the set contains the peer referred to by this IP
// address, otherwise false.
func (ps *PeerSet) HasIP(peerIP net.IP) bool {
	ps.mtx.Lock()
	defer ps.mtx.Unlock()

	return ps.hasIP(peerIP)
}

// hasIP does not acquire a lock so it can be used in public methods which
// already lock.
func (ps *PeerSet) hasIP(peerIP net.IP) bool {
	for _, item := range ps.lookup {
		if ip, err := item.peer.RemoteIP(); err == nil && ip.Equal(peerIP) {
			return true
		}
	}

	return false
}

func (ps *PeerSet) Size() int {
	ps.mtx.Lock()
	defer ps.mtx.Unlock()
	return len(ps.list)
}

// List returns the threadsafe list of peers.
func (ps *PeerSet) List() []Peer {
	ps.mtx.Lock()
	defer ps.mtx.Unlock()
	return ps.list
}

// Remove discards peer by its Key, if the peer was previously memoized.
func (ps *PeerSet) Remove(peer Peer) {
	ps.mtx.Lock()
	defer ps.mtx.Unlock()

	item := ps.lookup[peer.ID()]
	if item == nil {
		return
	}

	index := item.index
	// Create a new copy of the list but with one less item.
	// (we must copy because we'll be mutating the list).
	newList := make([]Peer, len(ps.list)-1)
	copy(newList, ps.list)
	// If it's the last peer, that's an easy special case.
	if index == len(ps.list)-1 {
		ps.list = newList
		delete(ps.lookup, peer.ID())
		return
	}

	// Replace the popped item with the last item in the old list.
	lastPeer := ps.list[len(ps.list)-1]
	lastPeerKey := lastPeer.ID()
	lastPeerItem := ps.lookup[lastPeerKey]
	newList[index] = lastPeer
	lastPeerItem.index = index
	ps.list = newList
	delete(ps.lookup, peer.ID())
}

//-------------------------peer connection--------------------------------
func (pc *peerConn) ID() ID {
	if len(pc.id) != 0 {
		return pc.id
	}
	address := GenAddressByPubKey(pc.conn.(*SecretConnection).RemotePubKey())
	pc.id = ID(hex.EncodeToString(address))
	return pc.id
}

func (pc *peerConn) RemoteIP() (net.IP, error) {
	if pc.ip != nil && len(pc.ip) > 0 {
		return pc.ip, nil
	}

	// In test cases a conn could not be present at all or be an in-memory
	// implementation where we want to return a fake ip.
	if pc.conn == nil || pc.conn.RemoteAddr().String() == "pipe" {
		return nil, errors.New("connect is nil or just pipe")
	}

	host, _, err := net.SplitHostPort(pc.conn.RemoteAddr().String())
	if err != nil {
		panic(err)
	}

	ips, err := net.LookupIP(host)
	if err != nil {
		panic(err)
	}

	pc.ip = ips[0]

	return pc.ip, nil
}

func (pc *peerConn) RemoteAddr() (net.Addr, error) {
	if pc.conn == nil || pc.conn.RemoteAddr().String() == "pipe" {
		return nil, errors.New("connect is nil or just pipe")
	}

	return pc.conn.RemoteAddr(), nil
}

func (pc *peerConn) SetTransferChannel(transferChannel chan MsgInfo) {
	pc.transferChannel = transferChannel
}

func (pc *peerConn) CloseConn() {
	pc.conn.Close() // nolint: errcheck
}

func (pc *peerConn) HandshakeTimeout(
	ourNodeInfo NodeInfo,
	timeout time.Duration,
) (peerNodeInfo NodeInfo, err error) {
	peerNodeInfo = NodeInfo{}
	// Set deadline for handshake so we don't block forever on conn.ReadFull
	if err := pc.conn.SetDeadline(time.Now().Add(timeout)); err != nil {
		return peerNodeInfo, errors.Wrap(err, "Error setting deadline")
	}

	var err1 error
	var err2 error
	Parallel(
		func() {
			info, err1 := json.Marshal(ourNodeInfo)
			if err1 != nil {
				tendermintlog.Error("Peer handshake peerNodeInfo failed", "err", err1)
				return
			} else {
				frame := make([]byte, 4)
				binary.BigEndian.PutUint32(frame, uint32(len(info)))
				_, err1 = pc.conn.Write(frame)
				_, err1 = pc.conn.Write(info[:])
			}
			//var n int
			//wire.WriteBinary(ourNodeInfo, p.conn, &n, &err1)
		},
		func() {
			readBuffer := make([]byte, 4)
			_, err2 = io.ReadFull(pc.conn, readBuffer[:])
			if err2 != nil {
				return
			}
			len := binary.BigEndian.Uint32(readBuffer)
			readBuffer = make([]byte, len)
			_, err2 = io.ReadFull(pc.conn, readBuffer[:])
			if err2 != nil {
				return
			}
			err2 = json.Unmarshal(readBuffer, &peerNodeInfo)
			if err2 != nil {
				return
			}

			//var n int
			//wire.ReadBinary(peerNodeInfo, p.conn, maxNodeInfoSize, &n, &err2)
			tendermintlog.Info("Peer handshake", "peerNodeInfo", peerNodeInfo)
		},
	)
	if err1 != nil {
		return peerNodeInfo, errors.Wrap(err1, "Error during handshake/write")
	}
	if err2 != nil {
		return peerNodeInfo, errors.Wrap(err2, "Error during handshake/read")
	}

	// Remove deadline
	if err := pc.conn.SetDeadline(time.Time{}); err != nil {
		return peerNodeInfo, errors.Wrap(err, "Error removing deadline")
	}

	return peerNodeInfo, nil
}

func (pc *peerConn) IsOutbound() bool {
	return pc.outbound
}

func (pc *peerConn) IsPersistent() bool {
	return pc.persistent
}

func (pc *peerConn) Send(msg MsgInfo) bool {
	if !pc.IsRunning() {
		return false
	}
	select {
	case pc.sendQueue <- msg:
		atomic.AddInt32(&pc.sendQueueSize, 1)
		return true
	case <-time.After(defaultSendTimeout):
		tendermintlog.Error("send msg timeout", "peerip", msg.PeerIP, "msg", msg.Msg)
		return false
	}
}

func (pc *peerConn) TrySend(msg MsgInfo) bool {
	if !pc.IsRunning() {
		return false
	}
	select {
	case pc.sendQueue <- msg:
		atomic.AddInt32(&pc.sendQueueSize, 1)
		return true
	default:
		return false
	}
}

func (pc *peerConn) IsRunning() bool {
	return atomic.LoadUint32(&pc.started) == 1 && atomic.LoadUint32(&pc.stopped) == 0
}

func (pc *peerConn) Start() error {
	if atomic.CompareAndSwapUint32(&pc.started, 0, 1) {
		if atomic.LoadUint32(&pc.stopped) == 1 {
			tendermintlog.Error("peerConn already stoped can not start", "peerIP", pc.ip.String())
			return nil
		}
		pc.bufReader = bufio.NewReaderSize(pc.conn, minReadBufferSize)
		pc.bufWriter = bufio.NewWriterSize(pc.conn, minWriteBufferSize)
		pc.pongChannel = make(chan struct{})
		pc.sendQueue = make(chan MsgInfo, maxSendQueueSize)
		pc.sendBuffer = make([]byte, 0, MaxMsgPacketPayloadSize)
		pc.quit = make(chan struct{})
		pc.state = &PeerConnState{ip: pc.ip, PeerRoundState: ttypes.PeerRoundState{
			Round:              -1,
			ProposalPOLRound:   -1,
			LastCommitRound:    -1,
			CatchupCommitRound: -1,
		}}
		pc.updateStateQueue = make(chan MsgInfo, maxSendQueueSize)
		pc.heartbeatQueue = make(chan proto.Message, 100)
		pc.waitQuit.Add(5) //sendRoutine, updateStateRoutine,gossipDataRoutine,gossipVotesRoutine,queryMaj23Routine

		go pc.sendRoutine()
		go pc.recvRoutine()
		go pc.updateStateRoutine()
		go pc.heartbeatRoutine()

		go pc.gossipDataRoutine()
		go pc.gossipVotesRoutine()
		go pc.queryMaj23Routine()

	}
	return nil
}

func (pc *peerConn) Stop() {
	if atomic.CompareAndSwapUint32(&pc.stopped, 0, 1) {
		if pc.heartbeatQueue != nil {
			close(pc.heartbeatQueue)
			pc.heartbeatQueue = nil
		}
		if pc.quit != nil {
			close(pc.quit)
			tendermintlog.Info("peerConn stop quit wait", "peerIP", pc.ip.String())
			pc.waitQuit.Wait()
			tendermintlog.Info("peerConn stop quit wait finish", "peerIP", pc.ip.String())
			pc.quit = nil
		}
		close(pc.sendQueue)
		pc.transferChannel = nil
		pc.CloseConn()
	}
}

// Catch panics, usually caused by remote disconnects.
func (pc *peerConn) _recover() {
	if r := recover(); r != nil {
		stack := debug.Stack()
		err := StackError{r, stack}
		pc.stopForError(err)
	}
}

func (pc *peerConn) stopForError(r interface{}) {
	tendermintlog.Error("peerConn recovered panic", "error", r, "peer", pc.ip.String())
	if pc.onPeerError != nil {
		pc.onPeerError(pc, r)
	}
	pc.Stop()
}

func (pc *peerConn) sendRoutine() {
	defer pc._recover()
FOR_LOOP:
	for {
		select {
		case <-pc.quit:
			pc.waitQuit.Done()
			break FOR_LOOP
		case msg := <-pc.sendQueue:
			bytes, err := proto.Marshal(msg.Msg)
			if err != nil {
				tendermintlog.Error("peerConn sendroutine marshal data failed", "error", err)
				pc.stopForError(err)
				break FOR_LOOP
			}

			len := len(bytes)
			bytelen := make([]byte, 4)
			binary.BigEndian.PutUint32(bytelen, uint32(len))
			pc.sendBuffer = pc.sendBuffer[:0]
			pc.sendBuffer = append(pc.sendBuffer, msg.TypeID)
			pc.sendBuffer = append(pc.sendBuffer, bytelen...)

			pc.sendBuffer = append(pc.sendBuffer, bytes...)
			if len+5 > MaxMsgPacketPayloadSize {
				pc.sendBuffer = append(pc.sendBuffer, bytes[MaxMsgPacketPayloadSize-5:]...)
			}
			_, err = pc.bufWriter.Write(pc.sendBuffer[:len+5])
			if err != nil {
				tendermintlog.Error("peerConn sendroutine write data failed", "error", err)
				pc.stopForError(err)
				break FOR_LOOP
			}
			pc.bufWriter.Flush()
		case _, ok := <-pc.pongChannel:
			if ok {
				tendermintlog.Debug("Send Pong")
				var pong [5]byte
				pong[0] = ttypes.PacketTypePong
				_, err := pc.bufWriter.Write(pong[:])
				if err != nil {
					tendermintlog.Error("peerConn sendroutine write pong failed", "error", err)
					pc.stopForError(err)
					break FOR_LOOP
				}
			} else {
				pc.pongChannel = nil
			}
		}
	}
}

func (pc *peerConn) recvRoutine() {
	defer pc._recover()
FOR_LOOP:
	for {
		//typeID+msgLen+msg
		var buf [5]byte
		_, err := io.ReadFull(pc.bufReader, buf[:])
		if err != nil {
			tendermintlog.Error("Connection failed @ recvRoutine (reading byte)", "conn", pc, "err", err)
			pc.stopForError(err)
			break FOR_LOOP
		}
		pkt := msgPacket{}
		pkt.TypeID = buf[0]
		len := binary.BigEndian.Uint32(buf[1:])
		if len > 0 {
			buf2 := make([]byte, len)
			_, err = io.ReadFull(pc.bufReader, buf2)
			if err != nil {
				tendermintlog.Error("Connection failed @ recvRoutine", "conn", pc, "err", err)
				pc.stopForError(err)
				panic(fmt.Sprintf("peerConn recvRoutine packetTypeMsg failed :%v", err))
			}
			pkt.Bytes = buf2
		}

		if pkt.TypeID == ttypes.PacketTypePong {
			tendermintlog.Debug("Receive Pong")
		} else if pkt.TypeID == ttypes.PacketTypePing {
			tendermintlog.Debug("Receive Ping")
			pc.pongChannel <- struct{}{}
		} else {
			if v, ok := ttypes.MsgMap[pkt.TypeID]; ok {
				realMsg := reflect.New(v).Interface()
				err := proto.Unmarshal(pkt.Bytes, realMsg.(proto.Message))
				if err != nil {
					tendermintlog.Error("peerConn recvRoutine Unmarshal data failed", "err", err)
					continue
				}
				if pc.transferChannel != nil && (pkt.TypeID == ttypes.ProposalID || pkt.TypeID == ttypes.VoteID || pkt.TypeID == ttypes.ProposalBlockID) {
					pc.transferChannel <- MsgInfo{pkt.TypeID, realMsg.(proto.Message), pc.ID(), pc.ip.String()}
					if pkt.TypeID == ttypes.ProposalID {
						proposal := realMsg.(*tmtypes.Proposal)
						tendermintlog.Debug("Receiving proposal", "proposal-height", proposal.Height, "peerip", pc.ip.String())
						pc.state.SetHasProposal(proposal)
					} else if pkt.TypeID == ttypes.VoteID {
						vote := &ttypes.Vote{Vote: realMsg.(*tmtypes.Vote)}
						tendermintlog.Debug("Receiving vote", "vote-height", vote.Height, "peerip", pc.ip.String())
						pc.state.SetHasVote(vote)
					} else if pkt.TypeID == ttypes.ProposalBlockID {
						block := &ttypes.TendermintBlock{realMsg.(*tmtypes.TendermintBlock)}
						tendermintlog.Debug("Receiving proposal block", "block-height", block.Header.Height, "peerip", pc.ip.String())
						pc.state.SetHasProposalBlock(block)
					}
				} else if pkt.TypeID == ttypes.ProposalHeartbeatID {
					pc.heartbeatQueue <- realMsg.(*tmtypes.Heartbeat)
				} else if pkt.TypeID == ttypes.EvidenceListID {
					go func() {
						for _, ev := range realMsg.(*tmtypes.EvidenceData).Evidence {
							evidence := ttypes.EvidenceEnvelope2Evidence(ev)
							if evidence != nil {
								err := pc.myevpool.AddEvidence(evidence.(ttypes.Evidence))
								if err != nil {
									tendermintlog.Error("Evidence is not valid", "evidence", ev, "err", err)
									// TODO: punish peer
								}
							}
						}
					}()
				} else {
					pc.updateStateQueue <- MsgInfo{pkt.TypeID, realMsg.(proto.Message), pc.ID(), pc.ip.String()}
				}
			} else {
				err := fmt.Errorf("Unknown message type %v", pkt.TypeID)
				tendermintlog.Error("Connection failed @ recvRoutine", "conn", pc, "err", err)
				pc.stopForError(err)
				break FOR_LOOP
			}
		}
	}

	close(pc.pongChannel)
	for range pc.pongChannel {
		// Drain
	}
}

func (pc *peerConn) updateStateRoutine() {
FOR_LOOP:
	for {
		select {
		case msg := <-pc.updateStateQueue:
			typeID := msg.TypeID
			if typeID == ttypes.NewRoundStepID {
				pc.state.ApplyNewRoundStepMessage(msg.Msg.(*tmtypes.NewRoundStepMsg))
			} else if typeID == ttypes.CommitStepID {
				pc.state.ApplyCommitStepMessage(msg.Msg.(*tmtypes.CommitStepMsg))
			} else if typeID == ttypes.HasVoteID {
				pc.state.ApplyHasVoteMessage(msg.Msg.(*tmtypes.HasVoteMsg))
			} else if typeID == ttypes.VoteSetMaj23ID {
				tmp := msg.Msg.(*tmtypes.VoteSetMaj23Msg)
				pc.myState.SetPeerMaj23(tmp.Height, int(tmp.Round), byte(tmp.Type), pc.id, tmp.BlockID)
				myVotes := pc.myState.GetPrevotesState(tmp.Height, int(tmp.Round), tmp.BlockID)
				if myVotes != nil && myVotes.TendermintBitArray != nil {
					voteSetBitMsg := &tmtypes.VoteSetBitsMsg{
						Height:  tmp.Height,
						Round:   tmp.Round,
						Type:    tmp.Type,
						BlockID: tmp.BlockID,
						Votes:   myVotes.TendermintBitArray,
					}
					pc.sendQueue <- MsgInfo{TypeID: ttypes.VoteSetBitsID, Msg: voteSetBitMsg, PeerID: pc.id, PeerIP: pc.ip.String()}
				}

			} else if typeID == ttypes.ProposalPOLID {
				pc.state.ApplyProposalPOLMessage(msg.Msg.(*tmtypes.ProposalPOLMsg))
			} else if typeID == ttypes.VoteSetBitsID {
				tmp := msg.Msg.(*tmtypes.VoteSetBitsMsg)
				if pc.myState.Height == tmp.Height {
					myVotes := pc.myState.GetPrevotesState(tmp.Height, int(tmp.Round), tmp.BlockID)
					pc.state.ApplyVoteSetBitsMessage(tmp, myVotes)
				} else {
					pc.state.ApplyVoteSetBitsMessage(tmp, nil)
				}
			} else {
				tendermintlog.Error("msg not deal in updateStateRoutine", "msgTypeName", msg.Msg.String())
			}
		case <-pc.quit:
			pc.waitQuit.Done()
			break FOR_LOOP
		}
	}
	close(pc.updateStateQueue)
	for range pc.updateStateQueue {
		// Drain
	}
}

func (pc *peerConn) heartbeatRoutine() {
	for {
		heartbeat, ok := <-pc.heartbeatQueue
		if !ok {
			tendermintlog.Debug("heartbeatQueue closed")
			return
		}
		msg := heartbeat.(*tmtypes.Heartbeat)
		tendermintlog.Debug("Received proposal heartbeat message",
			"height", msg.Height, "round", msg.Round, "sequence", msg.Sequence,
			"valIdx", msg.ValidatorIndex, "valAddr", msg.ValidatorAddress)
	}
}

func (pc *peerConn) gossipDataRoutine() {
OUTER_LOOP:
	for {
		// Manage disconnects from self or peer.

		if !pc.IsRunning() {
			tendermintlog.Error("Stopping gossipDataRoutine for peer")
			pc.waitQuit.Done()
			return
		}

		rs := pc.myState.GetRoundState()
		prs := pc.state

		// If the peer is on a previous height, help catch up.
		if (0 < prs.Height) && (prs.Height < rs.Height) {
			if prs.Height+1 == rs.Height && prs.Round == rs.LastCommit.Round() && prs.Step == ttypes.RoundStepCommit && prs.ProposalBlock {
				tendermintlog.Debug("Peer is waiting for finalizeCommit finish", "peerip", pc.ip.String(),
					"state", fmt.Sprintf("%v/%v/%v", prs.Height, prs.Round, prs.Step))
				time.Sleep(10 * pc.myState.PeerGossipSleep())
				continue OUTER_LOOP
			}
			tendermintlog.Info("help catch up", "peerip", pc.ip.String(), "selfHeight", rs.Height, "peerHeight", prs.Height)
			proposalBlock := pc.myState.client.LoadProposalBlock(prs.Height)
			if proposalBlock == nil {
				tendermintlog.Error("Failed to load propsal block", "selfHeight", rs.Height, "blockstoreHeight", pc.myState.client.GetCurrentHeight())
				time.Sleep(pc.myState.PeerGossipSleep())
				continue OUTER_LOOP
			}
			msg := MsgInfo{TypeID: ttypes.ProposalBlockID, Msg: proposalBlock, PeerID: pc.id, PeerIP: pc.ip.String()}
			tendermintlog.Info("Sending block for catchup", "peerip", pc.ip.String(), "block(H/R)",
				fmt.Sprintf("%v/%v", proposalBlock.Header.Height, proposalBlock.Header.Round))
			if pc.Send(msg) {
				//prs.SetHasProposalBlock(rs.ProposalBlock)
			}
			time.Sleep(10 * pc.myState.PeerGossipSleep())
			continue OUTER_LOOP
		}

		// If height and round don't match, sleep.
		if (rs.Height != prs.Height) || (rs.Round != prs.Round) {
			time.Sleep(pc.myState.PeerGossipSleep())
			continue OUTER_LOOP
		}

		// By here, height and round match.
		// Proposal block parts were already matched and sent if any were wanted.
		// (These can match on hash so the round doesn't matter)
		// Now consider sending other things, like the Proposal itself.

		// Send Proposal && ProposalPOL BitArray?
		if rs.Proposal != nil && !prs.Proposal {
			// Proposal: share the proposal metadata with peer.
			{
				msg := MsgInfo{TypeID: ttypes.ProposalID, Msg: rs.Proposal, PeerID: pc.id, PeerIP: pc.ip.String()}
				tendermintlog.Debug(fmt.Sprintf("Sending proposal. Self state: %v/%v/%v", rs.Height, rs.Round, rs.Step),
					"peerip", pc.ip.String(), "proposal-height", rs.Proposal.Height, "proposal-round", rs.Proposal.Round)
				if pc.Send(msg) {
					prs.SetHasProposal(rs.Proposal)
				}
			}
			// ProposalPOL: lets peer know which POL votes we have so far.
			// Peer must receive ttypes.ProposalMessage first.
			// rs.Proposal was validated, so rs.Proposal.POLRound <= rs.Round,
			// so we definitely have rs.Votes.Prevotes(rs.Proposal.POLRound).
			if 0 <= rs.Proposal.POLRound {
				msg := MsgInfo{TypeID: ttypes.ProposalPOLID, Msg: &tmtypes.ProposalPOLMsg{
					Height:           rs.Height,
					ProposalPOLRound: rs.Proposal.POLRound,
					ProposalPOL:      rs.Votes.Prevotes(int(rs.Proposal.POLRound)).BitArray().TendermintBitArray,
				}, PeerID: pc.id, PeerIP: pc.ip.String()}
				tendermintlog.Debug("Sending POL", "height", prs.Height, "round", prs.Round)
				pc.Send(msg)
			}
			continue OUTER_LOOP
		}

		// Send proposal block
		if rs.ProposalBlock != nil && !prs.ProposalBlock {
			msg := MsgInfo{TypeID: ttypes.ProposalBlockID, Msg: rs.ProposalBlock.TendermintBlock, PeerID: pc.id, PeerIP: pc.ip.String()}
			tendermintlog.Debug(fmt.Sprintf("Sending proposal block. Self state: %v/%v/%v", rs.Height, rs.Round, rs.Step),
				"peerip", pc.ip.String(), "block-height", rs.ProposalBlock.Header.Height, "block-round", rs.ProposalBlock.Header.Round)
			if pc.Send(msg) {
				prs.SetHasProposalBlock(rs.ProposalBlock)
			}
			continue OUTER_LOOP
		}

		// Nothing to do. Sleep.
		time.Sleep(pc.myState.PeerGossipSleep())
		continue OUTER_LOOP
	}
}

func (pc *peerConn) gossipVotesRoutine() {
	// Simple hack to throttle logs upon sleep.
	var sleeping = 0

OUTER_LOOP:
	for {
		// Manage disconnects from self or peer.
		if !pc.IsRunning() {
			tendermintlog.Info("Stopping gossipVotesRoutine for peer")
			pc.waitQuit.Done()
			return
		}

		rs := pc.myState.GetRoundState()
		prs := pc.state
		//tendermintlog.Debug("gossipVotesRoutine", "rs(H/R/S)", fmt.Sprintf("%v/%v/%v", rs.Height, rs.Round, rs.Step.String()),
		//	"prs(H/R/S)", fmt.Sprintf("%v/%v/%v", prs.Height, prs.Round, prs.Step.String()),
		//	"precommits", rs.Votes.Precommits(prs.Round).BitArray().String(), "peerip", pc.ip.String())

		switch sleeping {
		case 1: // First sleep
			sleeping = 2
		case 2: // No more sleep
			sleeping = 0
		}

		// If height matches, then send LastCommit, Prevotes, Precommits.
		if rs.Height == prs.Height {
			if pc.gossipVotesForHeight(rs, &prs.PeerRoundState) {
				continue OUTER_LOOP
			}
		}

		// Special catchup logic.
		// If peer is lagging by height 1, send LastCommit.
		if prs.Height != 0 && rs.Height == prs.Height+1 {
			if vote, ok := pc.state.PickVoteToSend(rs.LastCommit); ok {
				msg := MsgInfo{TypeID: ttypes.VoteID, Msg: vote.Vote, PeerID: pc.id, PeerIP: pc.ip.String()}
				pc.Send(msg)
				tendermintlog.Debug("Picked rs.LastCommit to send", "peerip", pc.ip.String(), "height", prs.Height, "vote.Height", vote.Height)
			} else {
				continue OUTER_LOOP
			}
		}

		// Catchup logic
		// If peer is lagging by more than 1, send Commit.
		if prs.Height != 0 && rs.Height >= prs.Height+2 {
			// Load the block commit for prs.Height,
			// which contains precommit signatures for prs.Height.
			commit := pc.myState.client.LoadSeenCommit(prs.Height)
			commitObj := &ttypes.Commit{TendermintCommit: commit}
			if vote, ok := pc.state.PickVoteToSend(commitObj); ok {
				msg := MsgInfo{TypeID: ttypes.VoteID, Msg: vote.Vote, PeerID: pc.id, PeerIP: pc.ip.String()}
				pc.Send(msg)
				tendermintlog.Info("Picked Catchup commit to send", "BitArray", commitObj.BitArray().String(), "valIndex", vote.ValidatorIndex,
					"peerip", pc.ip.String(), "height", prs.Height, "vote.Height", vote.Height)
			} else {
				continue OUTER_LOOP
			}
		}

		if sleeping == 0 {
			// We sent nothing. Sleep...
			sleeping = 1
			tendermintlog.Debug("No votes to send, sleeping", "peerip", pc.ip.String(), "rs.Height", rs.Height, "prs.Height", prs.Height,
				"localPV", rs.Votes.Prevotes(rs.Round).BitArray(), "peerPV", prs.Prevotes,
				"localPC", rs.Votes.Precommits(rs.Round).BitArray(), "peerPC", prs.Precommits)
		} else if sleeping == 2 {
			// Continued sleep...
			sleeping = 1
		}

		time.Sleep(pc.myState.PeerGossipSleep())
		continue OUTER_LOOP
	}
}

func (pc *peerConn) gossipVotesForHeight(rs *ttypes.RoundState, prs *ttypes.PeerRoundState) bool {
	// If there are lastCommits to send...
	if prs.Step == ttypes.RoundStepNewHeight {
		if vote, ok := pc.state.PickVoteToSend(rs.LastCommit); ok {
			msg := MsgInfo{TypeID: ttypes.VoteID, Msg: vote.Vote, PeerID: pc.id, PeerIP: pc.ip.String()}
			pc.Send(msg)
			tendermintlog.Debug("Picked rs.LastCommit to send", "peerip", pc.ip.String(), "peer(H/R)", fmt.Sprintf("%v/%v", prs.Height, prs.Round))
			return true
		}
	}
	// If there are prevotes to send...
	if prs.Step <= ttypes.RoundStepPrevoteWait && prs.Round != -1 && prs.Round <= rs.Round {
		if vote, ok := pc.state.PickVoteToSend(rs.Votes.Prevotes(prs.Round)); ok {
			msg := MsgInfo{TypeID: ttypes.VoteID, Msg: vote.Vote, PeerID: pc.id, PeerIP: pc.ip.String()}
			pc.Send(msg)
			tendermintlog.Debug("Picked rs.Prevotes(prs.Round) to send", "peerip", pc.ip.String(), "peer(H/R)", fmt.Sprintf("%v/%v", prs.Height, prs.Round))
			return true
		}
	}
	// If there are precommits to send...
	if prs.Step <= ttypes.RoundStepPrecommitWait && prs.Round != -1 && prs.Round <= rs.Round {
		if vote, ok := pc.state.PickVoteToSend(rs.Votes.Precommits(prs.Round)); ok {
			msg := MsgInfo{TypeID: ttypes.VoteID, Msg: vote.Vote, PeerID: pc.id, PeerIP: pc.ip.String()}
			pc.Send(msg)
			tendermintlog.Debug("Picked rs.Precommits(prs.Round) to send", "peerip", pc.ip.String(), "peer(H/R)", fmt.Sprintf("%v/%v", prs.Height, prs.Round))
			return true
		}
	}
	// If there are prevotes to send...Needed because of validBlock mechanism
	if prs.Round != -1 && prs.Round <= rs.Round {
		if vote, ok := pc.state.PickVoteToSend(rs.Votes.Prevotes(prs.Round)); ok {
			msg := MsgInfo{TypeID: ttypes.VoteID, Msg: vote.Vote, PeerID: pc.id, PeerIP: pc.ip.String()}
			pc.Send(msg)
			tendermintlog.Debug("Picked rs.Prevotes(prs.Round) to send", "peerip", pc.ip.String(), "peer(H/R)", fmt.Sprintf("%v/%v", prs.Height, prs.Round))
			return true
		}
	}
	// If there are POLPrevotes to send...
	if prs.ProposalPOLRound != -1 {
		if polPrevotes := rs.Votes.Prevotes(prs.ProposalPOLRound); polPrevotes != nil {
			if vote, ok := pc.state.PickVoteToSend(polPrevotes); ok {
				msg := MsgInfo{TypeID: ttypes.VoteID, Msg: vote.Vote, PeerID: pc.id, PeerIP: pc.ip.String()}
				pc.Send(msg)
				tendermintlog.Debug("Picked rs.Prevotes(prs.ProposalPOLRound) to send", "peerip", pc.ip.String(), "round", prs.ProposalPOLRound)
				return true
			}
		}
	}
	return false
}

func (pc *peerConn) queryMaj23Routine() {
OUTER_LOOP:
	for {
		// Manage disconnects from self or peer.
		if !pc.IsRunning() {
			tendermintlog.Info("Stopping queryMaj23Routine for peer")
			pc.waitQuit.Done()
			return
		}

		// Maybe send Height/Round/Prevotes
		{
			rs := pc.myState.GetRoundState()
			prs := pc.state
			if rs.Height == prs.Height {
				if maj23, ok := rs.Votes.Prevotes(prs.Round).TwoThirdsMajority(); ok {
					msg := MsgInfo{TypeID: ttypes.VoteSetMaj23ID, Msg: &tmtypes.VoteSetMaj23Msg{
						Height:  prs.Height,
						Round:   int32(prs.Round),
						Type:    int32(ttypes.VoteTypePrevote),
						BlockID: &maj23,
					}, PeerID: pc.id, PeerIP: pc.ip.String(),
					}
					pc.TrySend(msg)
					time.Sleep(pc.myState.PeerQueryMaj23Sleep())
				}
			}
		}

		// Maybe send Height/Round/Precommits
		{
			rs := pc.myState.GetRoundState()
			prs := pc.state.GetRoundState()
			if rs.Height == prs.Height {
				if maj23, ok := rs.Votes.Precommits(prs.Round).TwoThirdsMajority(); ok {
					msg := MsgInfo{TypeID: ttypes.VoteSetMaj23ID, Msg: &tmtypes.VoteSetMaj23Msg{
						Height:  prs.Height,
						Round:   int32(prs.Round),
						Type:    int32(ttypes.VoteTypePrecommit),
						BlockID: &maj23,
					}, PeerID: pc.id, PeerIP: pc.ip.String(),
					}
					pc.TrySend(msg)
					time.Sleep(pc.myState.PeerQueryMaj23Sleep())
				}
			}
		}

		// Maybe send Height/Round/ProposalPOL
		{
			rs := pc.myState.GetRoundState()
			prs := pc.state.GetRoundState()
			if rs.Height == prs.Height && prs.ProposalPOLRound >= 0 {
				if maj23, ok := rs.Votes.Prevotes(prs.ProposalPOLRound).TwoThirdsMajority(); ok {
					msg := MsgInfo{TypeID: ttypes.VoteSetMaj23ID, Msg: &tmtypes.VoteSetMaj23Msg{
						Height:  prs.Height,
						Round:   int32(prs.ProposalPOLRound),
						Type:    int32(ttypes.VoteTypePrevote),
						BlockID: &maj23,
					}, PeerID: pc.id, PeerIP: pc.ip.String(),
					}
					pc.TrySend(msg)
					time.Sleep(pc.myState.PeerQueryMaj23Sleep())
				}
			}
		}

		// Little point sending LastCommitRound/LastCommit,
		// These are fleeting and non-blocking.

		// Maybe send Height/CatchupCommitRound/CatchupCommit.
		{
			prs := pc.state.GetRoundState()
			if prs.CatchupCommitRound != -1 && 0 < prs.Height && prs.Height <= pc.myState.client.GetCurrentHeight() {
				commit := pc.myState.LoadCommit(prs.Height)
				commitTmp := ttypes.Commit{TendermintCommit: commit}
				msg := MsgInfo{TypeID: ttypes.VoteSetMaj23ID, Msg: &tmtypes.VoteSetMaj23Msg{
					Height:  prs.Height,
					Round:   int32(commitTmp.Round()),
					Type:    int32(ttypes.VoteTypePrecommit),
					BlockID: commit.BlockID,
				}, PeerID: pc.id, PeerIP: pc.ip.String(),
				}
				pc.TrySend(msg)
				time.Sleep(pc.myState.PeerQueryMaj23Sleep())
			}
		}

		time.Sleep(pc.myState.PeerQueryMaj23Sleep())

		continue OUTER_LOOP
	}
}

type StackError struct {
	Err   interface{}
	Stack []byte
}

func (se StackError) String() string {
	return fmt.Sprintf("Error: %v\nStack: %s", se.Err, se.Stack)
}

func (se StackError) Error() string {
	return se.String()
}

//-----------------------------------------------------------------
// GetRoundState returns an atomic snapshot of the PeerRoundState.
// There's no point in mutating it since it won't change PeerState.
func (ps *PeerConnState) GetRoundState() *ttypes.PeerRoundState {
	ps.mtx.Lock()
	defer ps.mtx.Unlock()

	prs := ps.PeerRoundState // copy
	return &prs
}

// GetHeight returns an atomic snapshot of the PeerRoundState's height
// used by the mempool to ensure peers are caught up before broadcasting new txs
func (ps *PeerConnState) GetHeight() int64 {
	ps.mtx.Lock()
	defer ps.mtx.Unlock()
	return ps.PeerRoundState.Height
}

// SetHasProposal sets the given proposal as known for the peer.
func (ps *PeerConnState) SetHasProposal(proposal *tmtypes.Proposal) {
	ps.mtx.Lock()
	defer ps.mtx.Unlock()

	if ps.Height != proposal.Height || ps.Round != int(proposal.Round) {
		return
	}
	if ps.Proposal {
		return
	}
	tendermintlog.Debug("Peer set proposal", "peerip", ps.ip.String(), "peer-state", fmt.Sprintf("%v/%v/%v", ps.Height, ps.Round, ps.Step),
		"proposal(H/R)", fmt.Sprintf("%v/%v", proposal.Height, proposal.Round))
	ps.Proposal = true

	ps.ProposalPOLRound = int(proposal.POLRound)
	ps.ProposalPOL = nil // Nil until ttypes.ProposalPOLMessage received.
}

// SetHasProposalBlock sets the given proposal block as known for the peer.
func (ps *PeerConnState) SetHasProposalBlock(block *ttypes.TendermintBlock) {
	ps.mtx.Lock()
	defer ps.mtx.Unlock()

	if ps.Height != block.Header.Height || ps.Round != int(block.Header.Round) {
		return
	}
	if ps.ProposalBlock {
		return
	}
	tendermintlog.Debug("Peer set proposal block", "peerip", ps.ip.String(), "peer-state", fmt.Sprintf("%v/%v/%v", ps.Height, ps.Round, ps.Step),
		"block(H/R)", fmt.Sprintf("%v/%v", block.Header.Height, block.Header.Round))
	ps.ProposalBlock = true
}

// PickVoteToSend picks a vote to send to the peer.
// Returns true if a vote was picked.
// NOTE: `votes` must be the correct Size() for the Height().
func (ps *PeerConnState) PickVoteToSend(votes ttypes.VoteSetReader) (vote *ttypes.Vote, ok bool) {
	ps.mtx.Lock()
	defer ps.mtx.Unlock()

	if votes.Size() == 0 {
		return nil, false
	}

	height, round, type_, size := votes.Height(), votes.Round(), votes.Type(), votes.Size()

	// Lazily set data using 'votes'.
	if votes.IsCommit() {
		ps.ensureCatchupCommitRound(height, round, size)
	}
	ps.ensureVoteBitArrays(height, size)

	psVotes := ps.getVoteBitArray(height, round, type_)
	if psVotes == nil {
		return nil, false // Not something worth sending
	}

	if index, ok := votes.BitArray().Sub(psVotes).PickRandom(); ok {
		tendermintlog.Debug("PickVoteToSend", "height", height, "index", index, "type", type_, "selfVotes", votes.BitArray().String(),
			"peerVotes", psVotes.String(), "peerip", ps.ip.String())
		ps.setHasVote(height, round, type_, index)
		return votes.GetByIndex(index), true
	}
	return nil, false
}

func (ps *PeerConnState) getVoteBitArray(height int64, round int, type_ byte) *ttypes.BitArray {
	if !ttypes.IsVoteTypeValid(type_) {
		return nil
	}

	if ps.Height == height {
		if ps.Round == round {
			switch type_ {
			case ttypes.VoteTypePrevote:
				return ps.Prevotes
			case ttypes.VoteTypePrecommit:
				return ps.Precommits
			}
		}
		if ps.CatchupCommitRound == round {
			switch type_ {
			case ttypes.VoteTypePrevote:
				return nil
			case ttypes.VoteTypePrecommit:
				return ps.CatchupCommit
			}
		}
		if ps.ProposalPOLRound == round {
			switch type_ {
			case ttypes.VoteTypePrevote:
				return ps.ProposalPOL
			case ttypes.VoteTypePrecommit:
				return nil
			}
		}
		return nil
	}
	if ps.Height == height+1 {
		if ps.LastCommitRound == round {
			switch type_ {
			case ttypes.VoteTypePrevote:
				return nil
			case ttypes.VoteTypePrecommit:
				return ps.LastCommit
			}
		}
		return nil
	}
	return nil
}

// 'round': A round for which we have a +2/3 commit.
func (ps *PeerConnState) ensureCatchupCommitRound(height int64, round int, numValidators int) {
	if ps.Height != height {
		return
	}
	/*
		NOTE: This is wrong, 'round' could change.
		e.g. if orig round is not the same as block LastCommit round.
		if ps.CatchupCommitRound != -1 && ps.CatchupCommitRound != round {
			ttypes.PanicSanity(ttypes.Fmt("Conflicting CatchupCommitRound. Height: %v, Orig: %v, New: %v", height, ps.CatchupCommitRound, round))
		}
	*/
	if ps.CatchupCommitRound == round {
		return // Nothing to do!
	}
	tendermintlog.Debug("ensureCatchupCommitRound", "height", height, "round", round, "ps.CatchupCommitRound", ps.CatchupCommitRound,
		"ps.Round", ps.Round, "peerip", ps.ip.String())
	ps.CatchupCommitRound = round
	if round == ps.Round {
		ps.CatchupCommit = ps.Precommits
	} else {
		ps.CatchupCommit = ttypes.NewBitArray(numValidators)
	}
}

// EnsureVoteVitArrays ensures the bit-arrays have been allocated for tracking
// what votes this peer has received.
// NOTE: It's important to make sure that numValidators actually matches
// what the node sees as the number of validators for height.
func (ps *PeerConnState) EnsureVoteBitArrays(height int64, numValidators int) {
	ps.mtx.Lock()
	defer ps.mtx.Unlock()
	ps.ensureVoteBitArrays(height, numValidators)
}

func (ps *PeerConnState) ensureVoteBitArrays(height int64, numValidators int) {
	if ps.Height == height {
		if ps.Prevotes == nil {
			ps.Prevotes = ttypes.NewBitArray(numValidators)
		}
		if ps.Precommits == nil {
			ps.Precommits = ttypes.NewBitArray(numValidators)
		}
		if ps.CatchupCommit == nil {
			ps.CatchupCommit = ttypes.NewBitArray(numValidators)
		}
		if ps.ProposalPOL == nil {
			ps.ProposalPOL = ttypes.NewBitArray(numValidators)
		}
	} else if ps.Height == height+1 {
		if ps.LastCommit == nil {
			ps.LastCommit = ttypes.NewBitArray(numValidators)
		}
	}
}

// SetHasVote sets the given vote as known by the peer
func (ps *PeerConnState) SetHasVote(vote *ttypes.Vote) {
	ps.mtx.Lock()
	defer ps.mtx.Unlock()

	ps.setHasVote(vote.Height, int(vote.Round), byte(vote.Type), int(vote.ValidatorIndex))
}

func (ps *PeerConnState) setHasVote(height int64, round int, type_ byte, index int) {
	// NOTE: some may be nil BitArrays -> no side effects.
	psVotes := ps.getVoteBitArray(height, round, type_)
	tendermintlog.Debug("setHasVote before", "height", height, "psVotes", psVotes.String(), "peerip", ps.ip.String())
	if psVotes != nil {
		psVotes.SetIndex(index, true)
	}
	tendermintlog.Debug("setHasVote after", "height", height, "index", index, "type", type_, "peerVotes", psVotes.String(), "peerip", ps.ip.String())
}

// ApplyNewRoundStepMessage updates the peer state for the new round.
func (ps *PeerConnState) ApplyNewRoundStepMessage(msg *tmtypes.NewRoundStepMsg) {
	ps.mtx.Lock()
	defer ps.mtx.Unlock()

	// Ignore duplicates or decreases
	if CompareHRS(msg.Height, int(msg.Round), ttypes.RoundStepType(msg.Step), ps.Height, ps.Round, ps.Step) <= 0 {
		return
	}

	// Just remember these values.
	psHeight := ps.Height
	psRound := ps.Round
	//psStep := ps.Step
	psCatchupCommitRound := ps.CatchupCommitRound
	psCatchupCommit := ps.CatchupCommit
	psPrecommits := ps.Precommits

	startTime := time.Now().Add(-1 * time.Duration(msg.SecondsSinceStartTime) * time.Second)
	ps.Height = msg.Height
	ps.Round = int(msg.Round)
	ps.Step = ttypes.RoundStepType(msg.Step)
	ps.StartTime = startTime

	if psHeight != msg.Height || psRound != int(msg.Round) {
		tendermintlog.Debug("Reset Proposal, Prevotes, Precommits", "peerip", ps.ip.String(), "peer(H/R)", fmt.Sprintf("%v/%v", psHeight, psRound),
			"msg(H/R)", fmt.Sprintf("%v/%v", msg.Height, msg.Round))
		ps.Proposal = false
		ps.ProposalBlock = false
		ps.ProposalPOLRound = -1
		ps.ProposalPOL = nil
		// We'll update the BitArray capacity later.
		ps.Prevotes = nil
		ps.Precommits = nil
	}
	if psHeight == msg.Height && psRound != int(msg.Round) && int(msg.Round) == psCatchupCommitRound {
		// Peer caught up to CatchupCommitRound.
		// Preserve psCatchupCommit!
		// NOTE: We prefer to use prs.Precommits if
		// pr.Round matches pr.CatchupCommitRound.
		ps.Precommits = psCatchupCommit
	}
	if psHeight != msg.Height {
		tendermintlog.Debug("Reset LastCommit, CatchupCommit", "peerip", ps.ip.String())
		// Shift Precommits to LastCommit.
		if psHeight+1 == msg.Height && psRound == int(msg.LastCommitRound) {
			ps.LastCommitRound = int(msg.LastCommitRound)
			ps.LastCommit = psPrecommits
		} else {
			ps.LastCommitRound = int(msg.LastCommitRound)
			ps.LastCommit = nil
		}
		// We'll update the BitArray capacity later.
		ps.CatchupCommitRound = -1
		ps.CatchupCommit = nil
	}
}

// ApplyCommitStepMessage updates the peer state for the new commit.
func (ps *PeerConnState) ApplyCommitStepMessage(msg *tmtypes.CommitStepMsg) {
	ps.mtx.Lock()
	defer ps.mtx.Unlock()

	if ps.Height != msg.Height {
		return
	}

}

// ApplyProposalPOLMessage updates the peer state for the new proposal POL.
func (ps *PeerConnState) ApplyProposalPOLMessage(msg *tmtypes.ProposalPOLMsg) {
	ps.mtx.Lock()
	defer ps.mtx.Unlock()

	if ps.Height != msg.Height {
		return
	}
	if ps.ProposalPOLRound != int(msg.ProposalPOLRound) {
		return
	}

	// TODO: Merge onto existing ps.ProposalPOL?
	// We might have sent some prevotes in the meantime.
	ps.ProposalPOL = &ttypes.BitArray{TendermintBitArray: msg.ProposalPOL}
}

// ApplyHasVoteMessage updates the peer state for the new vote.
func (ps *PeerConnState) ApplyHasVoteMessage(msg *tmtypes.HasVoteMsg) {
	ps.mtx.Lock()
	defer ps.mtx.Unlock()

	if ps.Height != msg.Height {
		return
	}

	tendermintlog.Debug("ApplyHasVoteMessage", "msg(H/R)", fmt.Sprintf("%v/%v", msg.Height, msg.Round), "peerip", ps.ip.String())
	ps.setHasVote(msg.Height, int(msg.Round), byte(msg.Type), int(msg.Index))
}

// ApplyVoteSetBitsMessage updates the peer state for the bit-array of votes
// it claims to have for the corresponding BlockID.
// `ourVotes` is a BitArray of votes we have for msg.BlockID
// NOTE: if ourVotes is nil (e.g. msg.Height < rs.Height),
// we conservatively overwrite ps's votes w/ msg.Votes.
func (ps *PeerConnState) ApplyVoteSetBitsMessage(msg *tmtypes.VoteSetBitsMsg, ourVotes *ttypes.BitArray) {
	ps.mtx.Lock()
	defer ps.mtx.Unlock()

	votes := ps.getVoteBitArray(msg.Height, int(msg.Round), byte(msg.Type))
	if votes != nil {
		if ourVotes == nil {
			bitarray := &ttypes.BitArray{TendermintBitArray: msg.Votes}
			votes.Update(bitarray)
		} else {
			otherVotes := votes.Sub(ourVotes)
			bitarray := &ttypes.BitArray{TendermintBitArray: msg.Votes}
			hasVotes := otherVotes.Or(bitarray)
			votes.Update(hasVotes)
		}
	}
}
