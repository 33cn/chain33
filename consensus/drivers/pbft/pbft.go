package pbft

import (
	"errors"
	pb "gitlab.33.cn/chain33/chain33/types"
	"net"
	"strings"
)

const (
	CHECKPOINT_PERIOD uint32 = 128
	CONSTANT_FACTOR   uint32 = 2
)

//var plog = log.New("module", "Pbft")

type Replica struct {
	ln net.Listener

	ID       uint32
	replicas map[uint32]string

	activeView bool
	view       uint32
	sequence   uint32

	requestChan chan *pb.Request
	replyChan   chan *pb.ClientReply
	errChan     chan error
	doneChan    chan string
	requests    map[string][]*pb.Request
	replies     map[string][]*pb.ClientReply
	lastReply   *pb.ClientReply

	// prep    []*pb.Entry
	// prePrep []*pb.Entry

	pendingVC []*pb.Request

	executed    []uint32
	checkpoints []*pb.Checkpoint
}

func NewReplica(id uint32, PeersURL string, addr string) (chan *pb.ClientReply, chan *pb.Request, bool) {
	replyChan := make(chan *pb.ClientReply)
	requestChan := make(chan *pb.Request)
	pn := &Replica{
		ID:          id,
		replicas:    make(map[uint32]string),
		activeView:  true,
		view:        1,
		sequence:    0,
		requestChan: requestChan,
		replyChan:   replyChan,
		errChan:     make(chan error),
		requests:    make(map[string][]*pb.Request),
		replies:     make(map[string][]*pb.ClientReply),
		doneChan:    make(chan string),
		lastReply:   nil,
		pendingVC:   make([]*pb.Request, 10),
		executed:    make([]uint32, 10),
		//checkpoints: make([]*pb.Checkpoint,10),

	}
	peers := strings.Split(PeersURL, ",")
	for num, peer := range peers {
		pn.replicas[uint32(num)] = peer
	}
	pn.checkpoints = []*pb.Checkpoint{pb.ToCheckpoint(0, []byte(""))}

	//go func(){pn.requestChan<-req}()
	pn.Startnode(addr)
	return replyChan, requestChan, pn.isPrimary(pn.ID)

}

func (rep *Replica) Startnode(addr string) {
	//log.Println("hello")
	rep.acceptConnections(addr)
	/*var operation uint32
	go func(){
		for   {
			fmt.Scanf("%d",&operation)
			//ep.doneChan<-"enter the req done"
			op:=&pb.Operation{operation}
			req:=pb.ToRequestClient(op,time.Now().String(),addr)
			rep.multicast(req)
		}
	}()*/
	/*go func() {
		for {
			log.Println(<-rep.doneChan)
		}
	}()*/
}

// Basic operations

func (rep *Replica) primary() uint32 {
	return rep.view % uint32(len(rep.replicas)+1)
}

func (rep *Replica) newPrimary(view uint32) uint32 {
	return view % uint32(len(rep.replicas)+1)
}

func (rep *Replica) isPrimary(ID uint32) bool {
	return ID == rep.primary()
}

func (rep *Replica) oneThird(count int) bool {
	return count >= (len(rep.replicas)+1)/3
}

func (rep *Replica) overOneThird(count int) bool {
	return count > (len(rep.replicas)+1)/3
}

func (rep *Replica) twoThirds(count int) bool {
	return count >= 2*(len(rep.replicas)-1)/3
}

func (rep *Replica) overTwoThirds(count int) bool {
	return count > 2*(len(rep.replicas)-1)/3
}

func (rep *Replica) lowWaterMark() uint32 {
	return rep.lastStable().Sequence
}

func (rep *Replica) highWaterMark() uint32 {
	return rep.lowWaterMark() + CHECKPOINT_PERIOD*CONSTANT_FACTOR
}

func (rep *Replica) sequenceInRange(sequence uint32) bool {
	return sequence > rep.lowWaterMark() && sequence <= rep.highWaterMark()
}

func (rep *Replica) lastExecuted() uint32 {
	return rep.executed[len(rep.executed)-1]
}

func (rep *Replica) lastStable() *pb.Checkpoint {
	//log.Println(len(rep.checkpoints)-1)
	return rep.checkpoints[len(rep.checkpoints)-1]
}

func (rep *Replica) theLastReply() *pb.ClientReply {
	lastReply := rep.lastReply
	for _, replies := range rep.replies {
		reply := replies[len(replies)-1]
		if reply.Timestamp > lastReply.Timestamp {
			lastReply = reply
		}
	}
	return lastReply
}

func (rep *Replica) lastReplyToClient(client string) *pb.ClientReply {
	if v, ok := rep.replies[client]; ok {
		return v[len(rep.replies[client])-1]
	} else {
		return nil
	}
}

func (rep *Replica) stateDigest() []byte {
	return rep.theLastReply().Digest()
}

func (rep *Replica) isCheckpoint(sequence uint32) bool {
	return sequence%CHECKPOINT_PERIOD == 0
}

func (rep *Replica) addCheckpoint(checkpoint *pb.Checkpoint) {
	rep.checkpoints = append(rep.checkpoints, checkpoint)
}

func (rep *Replica) acceptConnections(addr string) {
	//log.Println("out msg")
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		plog.Error("tcp connect error")
	}

	go rep.sendRoutine()
	go func() {

		for {
			// log.Println("start to listen")
			conn, err := ln.Accept()

			if err != nil {
				plog.Error("Accept error")
			}
			req := &pb.Request{}
			err = pb.ReadMessage(conn, req)
			if err != nil {
				plog.Error("readmessage error")
			}
			rep.handleRequest(req)
			//log.Println("handle mes done")
			//rep.doneChan<-"handle mes done"
		}
	}()
}

// Sends

func (rep *Replica) multicast(REQ *pb.Request) error {
	for _, replica := range rep.replicas {
		err := pb.WriteMessage(replica, REQ)
		if err != nil {
			return err
		}
	}
	return nil
}

func (rep *Replica) sendRoutine() {
	//log.Println("start select from chan ")
	//timeout:=time.After(time.Second)
	for {
		select {
		//case <-timeout:
		//log.Println("timeout")
		case REQ := <-rep.requestChan:
			switch REQ.Value.(type) {
			case *pb.Request_Ack:
				view := REQ.GetAck().View
				primaryID := rep.newPrimary(view)
				primary := rep.replicas[primaryID]
				if primary == "" {
					plog.Error("primary not exeist")
					continue
				}
				err := pb.WriteMessage(primary, REQ)
				if err != nil {
					go func() {
						rep.errChan <- err
					}()
				}
			default:
				err := rep.multicast(REQ)
				//log.Println("take from chan and multicast")
				if err != nil {
					go func() {
						rep.errChan <- err
					}()
				}
			}
			//case reply := <-rep.replyChan:
			//client := reply.Client
			/*err := pb.WriteMessage(client, reply)
			if err != nil {
				go func() {
					rep.errChan <- err
				}()
			}*/
			//log.Println("client: "+client+"req has done")
		}
		//rep.doneChan<-"take from chan and multicast"
	}
}

func (rep *Replica) allRequests() []*pb.Request {
	var requests []*pb.Request
	for _, reqs := range rep.requests {
		requests = append(requests, reqs...)
	}
	return requests
}

// Log
// !hasRequest before append?

func (rep *Replica) logRequest(REQ *pb.Request) {
	switch REQ.Value.(type) {
	case *pb.Request_Client:
		rep.requests["client"] = append(rep.requests["client"], REQ)
	case *pb.Request_Preprepare:
		rep.requests["pre-prepare"] = append(rep.requests["pre-prepare"], REQ)
	case *pb.Request_Prepare:
		rep.requests["prepare"] = append(rep.requests["prepare"], REQ)
	case *pb.Request_Commit:
		rep.requests["commit"] = append(rep.requests["commit"], REQ)
	case *pb.Request_Checkpoint:
		rep.requests["checkpoint"] = append(rep.requests["checkpoint"], REQ)
	case *pb.Request_Viewchange:
		rep.requests["view-change"] = append(rep.requests["view-change"], REQ)
	case *pb.Request_Ack:
		rep.requests["ack"] = append(rep.requests["ack"], REQ)
	case *pb.Request_Newview:
		rep.requests["new-view"] = append(rep.requests["new-view"], REQ)
	default:
		plog.Info("Replica %d tried logging unrecognized request type\n", rep.ID)
	}
}

func (rep *Replica) logPendingVC(REQ *pb.Request) error {
	switch REQ.Value.(type) {
	case *pb.Request_Viewchange:
		rep.pendingVC = append(rep.pendingVC, REQ)
		return nil
	default:
		return errors.New("Request is wrong type")
	}
}

func (rep *Replica) logReply(client string, reply *pb.ClientReply) {
	lastReplyToClient := rep.lastReplyToClient(client)
	if lastReplyToClient == nil || lastReplyToClient.Timestamp < reply.Timestamp {
		rep.replies[client] = append(rep.replies[client], reply)
	}
}

/*
func (rep *Replica) logPrep(prep *pb.RequestPrepare)
	rep.prep = append(rep.prep, prep)
}

func (rep *Replica) logPrePrep(prePrep *pb.RequestPrePrepare) {
	rep.prePrep = append(rep.prePrep, prePrep)
}
*/

// Has requests

func (rep *Replica) hasRequest(REQ *pb.Request) bool {

	switch REQ.Value.(type) {
	case *pb.Request_Preprepare:
		return rep.hasRequestPreprepare(REQ)
	case *pb.Request_Prepare:
		return rep.hasRequestPrepare(REQ)
	case *pb.Request_Commit:
		return rep.hasRequestCommit(REQ)
	case *pb.Request_Viewchange:
		return rep.hasRequestViewChange(REQ)
	case *pb.Request_Ack:
		return rep.hasRequestAck(REQ)
	case *pb.Request_Newview:
		return rep.hasRequestNewView(REQ)
	default:
		return false
	}
}

func (rep *Replica) hasRequestPreprepare(REQ *pb.Request) bool {
	view := REQ.GetPreprepare().View
	sequence := REQ.GetPreprepare().Sequence
	digest := REQ.GetPreprepare().Digest
	for _, req := range rep.requests["pre-prepare"] {
		v := req.GetPreprepare().View
		s := req.GetPreprepare().Sequence
		d := req.GetPreprepare().Digest
		if v == view && s == sequence && pb.EQ(d, digest) {
			return true
		}
	}
	return false
}

func (rep *Replica) hasRequestPrepare(REQ *pb.Request) bool {
	view := REQ.GetPrepare().View
	sequence := REQ.GetPrepare().Sequence
	digest := REQ.GetPrepare().Digest
	replica := REQ.GetPrepare().Replica
	for _, req := range rep.requests["prepare"] {
		v := req.GetPrepare().View
		s := req.GetPrepare().Sequence
		d := req.GetPrepare().Digest
		r := req.GetPrepare().Replica
		if v == view && s == sequence && pb.EQ(d, digest) && r == replica {
			return true
		}
	}
	return false
}

func (rep *Replica) hasRequestCommit(REQ *pb.Request) bool {
	view := REQ.GetCommit().View
	sequence := REQ.GetCommit().Sequence
	replica := REQ.GetCommit().Replica
	for _, req := range rep.requests["commit"] {
		v := req.GetCommit().View
		s := req.GetCommit().Sequence
		r := req.GetCommit().Replica
		if v == view && s == sequence && r == replica {
			return true
		}
	}
	return false
}

func (rep *Replica) hasRequestViewChange(REQ *pb.Request) bool {
	view := REQ.GetViewchange().View
	replica := REQ.GetViewchange().Replica
	for _, req := range rep.requests["view-change"] {
		v := req.GetViewchange().View
		r := req.GetViewchange().Replica
		if v == view && r == replica {
			return true
		}
	}
	return false
}

func (rep *Replica) hasRequestAck(REQ *pb.Request) bool {
	view := REQ.GetAck().View
	replica := REQ.GetAck().Replica
	viewchanger := REQ.GetAck().Viewchanger
	for _, req := range rep.requests["ack"] {
		v := req.GetAck().View
		r := req.GetAck().Replica
		vc := req.GetAck().Viewchanger
		if v == view && r == replica && vc == viewchanger {
			return true
		}
	}
	return false
}

func (rep *Replica) hasRequestNewView(REQ *pb.Request) bool {
	view := REQ.GetNewview().View
	for _, req := range rep.requests["new-view"] {
		v := req.GetNewview().View
		if v == view {
			return true
		}
	}
	return false
}

/*
func (rep *Replica) clearEntries(sequence uint32) {
	go rep.clearPrepared(sequence)
	go rep.clearPreprepared(sequence)
}

func (rep *Replica) clearPrepared(sequence uint32) {
	prepared := rep.prepared
	for idx, entry := range rep.prepared {
		s := entry.sequence
		if s <= sequence {
			prepared = append(prepared[:idx], prepared[idx+1:]...)
		}
	}
	rep.prepared = prepared
}

func (rep *Replica) clearPreprepared(sequence uint32) {
	prePrepared := rep.prePrepared
	for idx, entry := range rep.prePrepared {
		s := entry.sequence
		if s <= sequence {
			prePrepared = append(prePrepared[:idx], prePrepared[idx+1:]...)
		}
	}
	rep.prePrepared = prePrepared
}
*/

// Clear requests

func (rep *Replica) clearRequestsBySeq(sequence uint32) {
	rep.clearRequestClients()
	rep.clearRequestPrepreparesBySeq(sequence)
	rep.clearRequestPreparesBySeq(sequence)
	rep.clearRequestCommitsBySeq(sequence)
	rep.clearRequestCheckpointsBySeq(sequence)
}

func (rep *Replica) clearRequestClients() {
	clientReqs := rep.requests["client"]
	lastTimestamp := rep.theLastReply().Timestamp
	for idx, req := range rep.requests["client"] {
		timestamp := req.GetClient().Timestamp
		if lastTimestamp >= timestamp {
			clientReqs = append(clientReqs[:idx], clientReqs[idx+1:]...)
		}
	}
	rep.requests["client"] = clientReqs
}

func (rep *Replica) clearRequestPrepreparesBySeq(sequence uint32) {
	prePrepares := rep.requests["pre-prepare"]
	for idx, req := range rep.requests["pre-prepare"] {
		s := req.GetPreprepare().Sequence
		if s <= sequence {
			prePrepares = append(prePrepares[:idx], prePrepares[idx+1:]...)
		}
	}
	rep.requests["pre-prepare"] = prePrepares
}

func (rep *Replica) clearRequestPreparesBySeq(sequence uint32) {
	prepares := rep.requests["prepare"]
	for idx, req := range rep.requests["prepare"] {
		s := req.GetPrepare().Sequence
		if s <= sequence {
			prepares = append(prepares[:idx], prepares[idx+1:]...)
		}
	}
	rep.requests["prepare"] = prepares
}

func (rep *Replica) clearRequestCommitsBySeq(sequence uint32) {
	commits := rep.requests["commit"]
	for idx, req := range rep.requests["commit"] {
		s := req.GetCommit().Sequence
		if s <= sequence {
			commits = append(commits[:idx], commits[idx+1:]...)
		}
	}
	rep.requests["commit"] = commits
}

func (rep *Replica) clearRequestCheckpointsBySeq(sequence uint32) {
	checkpoints := rep.requests["checkpoint"]
	for idx, req := range rep.requests["checkpoint"] {
		s := req.GetCheckpoint().Sequence
		if s <= sequence {
			checkpoints = append(checkpoints[:idx], checkpoints[idx+1:]...)
		}
	}
	rep.requests["checkpoint"] = checkpoints
}

func (rep *Replica) clearRequestsByView(view uint32) {
	rep.clearRequestPrepreparesByView(view)
	rep.clearRequestPreparesByView(view)
	rep.clearRequestCommitsByView(view)
	//add others
}

func (rep *Replica) clearRequestPrepreparesByView(view uint32) {
	prePrepares := rep.requests["pre-prepare"]
	for idx, req := range rep.requests["pre-prepare"] {
		v := req.GetPreprepare().View
		if v < view {
			prePrepares = append(prePrepares[:idx], prePrepares[idx+1:]...)
		}
	}
	rep.requests["pre-prepare"] = prePrepares
}

func (rep *Replica) clearRequestPreparesByView(view uint32) {
	prepares := rep.requests["prepare"]
	for idx, req := range rep.requests["prepare"] {
		v := req.GetPrepare().View
		if v < view {
			prepares = append(prepares[:idx], prepares[idx+1:]...)
		}
	}
	rep.requests["prepare"] = prepares
}

func (rep *Replica) clearRequestCommitsByView(view uint32) {
	commits := rep.requests["commit"]
	for idx, req := range rep.requests["commit"] {
		v := req.GetCommit().View
		if v < view {
			commits = append(commits[:idx], commits[idx+1:]...)
		}
	}
	rep.requests["commit"] = commits
}

// Handle requests

func (rep *Replica) handleRequest(REQ *pb.Request) {

	switch REQ.Value.(type) {
	case *pb.Request_Client:

		rep.handleRequestClient(REQ)

	case *pb.Request_Preprepare:

		rep.handleRequestPreprepare(REQ)

	case *pb.Request_Prepare:

		rep.handleRequestPrepare(REQ)

	case *pb.Request_Commit:

		rep.handleRequestCommit(REQ)

	case *pb.Request_Checkpoint:

		rep.handleRequestCheckpoint(REQ)

	case *pb.Request_Viewchange:

		rep.handleRequestViewChange(REQ)

	case *pb.Request_Ack:

		rep.handleRequestAck(REQ)

	default:
		plog.Info("Replica %d received unrecognized request type\n", rep.ID)
	}
}

func (rep *Replica) handleRequestClient(REQ *pb.Request) {
	rep.sequence++
	client := REQ.GetClient().Client
	timestamp := REQ.GetClient().Timestamp
	lastReplyToClient := rep.lastReplyToClient(client)
	if lastReplyToClient != nil {
		if lastReplyToClient.Timestamp == timestamp {
			reply := pb.ToReply(rep.view, timestamp, client, rep.ID, lastReplyToClient.Result)
			rep.logReply(client, reply)
			go func() {
				rep.replyChan <- reply
			}()
			return
		}
	}

	rep.logRequest(REQ)

	if !rep.isPrimary(rep.ID) {
		return
	}

	req := pb.ToRequestPreprepare(rep.view, rep.sequence, REQ.Digest(), rep.ID)
	rep.logRequest(req)
	plog.Info("Client-request done")

	// prePrep := req.GetPreprepare()
	// rep.logPrePrep(prePrep)

	go func() {
		rep.requestChan <- req
	}()

	/*
		err := rep.multicast(prePrepare)
		if err != nil {
			return err
		}
	*/
}

func (rep *Replica) handleRequestPreprepare(REQ *pb.Request) {

	replica := REQ.GetPreprepare().Replica

	if !rep.isPrimary(replica) {
		return
	}

	view := REQ.GetPreprepare().View

	if rep.view != view {
		return
	}

	sequence := REQ.GetPreprepare().Sequence

	if !rep.sequenceInRange(sequence) {
		return
	}

	digest := REQ.GetPreprepare().Digest

	accept := true
	for _, req := range rep.requests["pre-prepare"] {
		v := req.GetPreprepare().View
		s := req.GetPreprepare().Sequence
		d := req.GetPreprepare().Digest

		if v == view && s == sequence && !pb.EQ(d, digest) {
			accept = false
			break
		}
	}

	if !accept {
		return
	}

	rep.logRequest(REQ)
	plog.Info("pre-prepare done")

	/*
		if rep.isPrimary() {
			return nil
		}
	*/

	req := pb.ToRequestPrepare(view, sequence, digest, rep.ID)

	if rep.hasRequest(req) {
		return
	}

	rep.logRequest(req)

	go func() {
		rep.requestChan <- req
	}()

	// prePrep := REQ.GetPreprepare()
	// rep.logPrePrep(prePrep)
}

func (rep *Replica) handleRequestPrepare(REQ *pb.Request) {

	replica := REQ.GetPrepare().Replica

	//if rep.isPrimary(replica) {
	//return
	//}

	view := REQ.GetPrepare().View

	if rep.view != view {
		return
	}

	sequence := REQ.GetPrepare().Sequence

	if !rep.sequenceInRange(sequence) {
		return
	}

	digest := REQ.GetPrepare().Digest

	rep.logRequest(REQ)

	//prePrepared := make(chan bool, 1)
	twoThirds := make(chan bool, 1)

	//go func() {
	//prePrepared <- rep.hasRequest(REQ)
	//}()

	go func() {
		count := 0
		for _, req := range rep.requests["prepare"] {
			v := req.GetPrepare().View
			s := req.GetPrepare().Sequence
			d := req.GetPrepare().Digest
			r := req.GetPrepare().Replica
			if v != view || s != sequence || !pb.EQ(d, digest) {
				continue
			}
			if r == replica {
				plog.Info("Replica %d sent multiple prepare requests\n", replica)
				//continue
			}
			count++
			if rep.overTwoThirds(count) {
				twoThirds <- true
				return
			}
			twoThirds <- false
		}
	}()

	if !<-twoThirds {
		return
	}

	req := pb.ToRequestCommit(view, sequence, rep.ID)
	if rep.hasRequest(req) {
		return
	}

	rep.logRequest(req)
	plog.Info("prepare done")
	go func() {
		rep.requestChan <- req
	}()
}

func (rep *Replica) handleRequestCommit(REQ *pb.Request) {

	view := REQ.GetCommit().View

	if rep.view != view {
		return
	}

	sequence := REQ.GetCommit().Sequence

	if !rep.sequenceInRange(sequence) {
		return
	}

	replica := REQ.GetCommit().Replica

	rep.logRequest(REQ)
	twoThirds := make(chan bool, 1)
	go func() {
		count := 0
		for _, req := range rep.requests["commit"] {
			v := req.GetCommit().View
			s := req.GetCommit().Sequence
			rr := req.GetCommit().Replica
			if v != view || s != sequence {
				continue
			}
			if rr == replica {
				plog.Info("Replica %d sent multiple commit requests\n", replica)
				//continue
			}
			count++
			if rep.overTwoThirds(count) {
				twoThirds <- true
				return
			}
		}
		twoThirds <- false
	}()
	if !<-twoThirds {
		return
	}
	//log.Println(count)
	//log.Println("overtwothirds done")
	var digest []byte
	digests := make(map[string]struct{})
	for _, req := range rep.requests["pre-prepare"] {
		v := req.GetPreprepare().View
		s := req.GetPreprepare().Sequence
		if v == view && s <= sequence {
			digests[string(req.GetPreprepare().Digest)] = struct{}{}
			//log.Println(string(req.Digest()))
			if s == sequence {
				//log.Println("break done ")
				digest = req.GetPreprepare().Digest
				break
			}
		}
	}
	for _, req := range rep.requests["client"] {
		d := req.Digest()
		//log.Println(string(d) )
		if _, exists := digests[string(d)]; !exists {
			//log.Println("not exisit digest")
			continue
		}
		if !pb.EQ(d, digest) {
			//log.Println("not eq error")
			// Unexecuted message op with
			// lower sequence number...
			continue
		}

		op := req.GetClient().Op
		timestamp := req.GetClient().Timestamp
		client := req.GetClient().Client
		result := &pb.Result{op.Value}

		rep.executed = append(rep.executed, sequence)
		reply := pb.ToReply(view, timestamp, client, rep.ID, result)

		rep.logReply(client, reply)
		plog.Info("commit done")
		rep.lastReply = reply

		go func() {
			rep.replyChan <- reply
		}()

		if !rep.isCheckpoint(sequence) {
			return
		}
		stateDigest := rep.stateDigest()
		req := pb.ToRequestCheckpoint(sequence, stateDigest, rep.ID)
		rep.logRequest(req)
		checkpoint := pb.ToCheckpoint(sequence, stateDigest)
		rep.addCheckpoint(checkpoint)
		go func() {
			rep.requestChan <- req
		}()
		return
	}

	return
}

func (rep *Replica) handleRequestCheckpoint(REQ *pb.Request) {

	sequence := REQ.GetCheckpoint().Sequence

	if !rep.sequenceInRange(sequence) {
		return
	}

	digest := REQ.GetCheckpoint().Digest
	replica := REQ.GetCheckpoint().Replica

	count := 0
	for _, req := range rep.requests["checkpoint"] {
		s := req.GetCheckpoint().Sequence
		d := req.GetCheckpoint().Digest
		r := req.GetCheckpoint().Replica
		if s != sequence || !pb.EQ(d, digest) {
			continue
		}
		if r == replica {
			plog.Info("Replica %d sent multiple checkpoint requests\n", replica)
			continue
		}
		count++
		if !rep.overTwoThirds(count) {
			continue
		}
		// rep.clearEntries(sequence)
		rep.clearRequestsBySeq(sequence)
		return
	}
	return
}

func (rep *Replica) handleRequestViewChange(REQ *pb.Request) {

	view := REQ.GetViewchange().View

	if view < rep.view {
		return
	}

	reqViewChange := REQ.GetViewchange()

	for _, prep := range reqViewChange.GetPreps() {
		v := prep.View
		s := prep.Sequence
		if v >= view || !rep.sequenceInRange(s) {
			return
		}
	}

	for _, prePrep := range reqViewChange.GetPrepreps() {
		v := prePrep.View
		s := prePrep.Sequence
		if v >= view || !rep.sequenceInRange(s) {
			return
		}
	}

	for _, checkpoint := range reqViewChange.GetCheckpoints() {
		s := checkpoint.Sequence
		if !rep.sequenceInRange(s) {
			return
		}
	}

	if rep.hasRequest(REQ) {
		return
	}

	rep.logRequest(REQ)

	viewchanger := reqViewChange.Replica

	req := pb.ToRequestAck(
		view,
		rep.ID,
		viewchanger,
		REQ.Digest())

	go func() {
		rep.requestChan <- req
	}()
}

func (rep *Replica) handleRequestAck(REQ *pb.Request) {

	view := REQ.GetAck().View
	primaryID := rep.newPrimary(view)

	if rep.ID != primaryID {
		return
	}

	if rep.hasRequest(REQ) {
		return
	}

	rep.logRequest(REQ)

	replica := REQ.GetAck().Replica
	viewchanger := REQ.GetAck().Viewchanger
	digest := REQ.GetAck().Digest

	reqViewChange := make(chan *pb.Request, 1)
	twoThirds := make(chan bool, 1)

	go func() {
		for _, req := range rep.requests["view-change"] {
			v := req.GetViewchange().View
			vc := req.GetViewchange().Replica
			if v == view && vc == viewchanger {
				reqViewChange <- req
			}
		}
		reqViewChange <- nil
	}()

	go func() {
		count := 0
		for _, req := range rep.requests["ack"] {
			v := req.GetAck().View
			r := req.GetAck().Replica
			vc := req.GetAck().Viewchanger
			d := req.GetAck().Digest
			if v != view || vc != viewchanger || !pb.EQ(d, digest) {
				continue
			}
			if r == replica {
				plog.Info("Replica %d sent multiple ack requests\n", replica)
				continue
			}
			count++
			if rep.twoThirds(count) {
				twoThirds <- true
				return
			}
		}
		twoThirds <- false
	}()

	req := <-reqViewChange

	if req == nil || !<-twoThirds {
		return
	}

	rep.logPendingVC(req)

	// When to send new view?
	rep.requestNewView(view)
}

func (rep *Replica) handleRequestNewView(REQ *pb.Request) {

	view := REQ.GetNewview().View

	if view == 0 || view < rep.view {
		return
	}

	replica := REQ.GetNewview().Replica
	primary := rep.newPrimary(view)

	if replica != primary {
		return
	}

	if rep.hasRequest(REQ) {
		return
	}

	rep.logRequest(REQ)

	rep.processNewView(REQ)
}

func (rep *Replica) correctViewChanges(viewChanges []*pb.ViewChange) (requests []*pb.Request) {

	// Returns requests if correct, else returns nil

	valid := false
	for _, vc := range viewChanges {
		for _, req := range rep.requests["view-change"] {
			d := req.Digest()
			if !pb.EQ(d, vc.Digest) {
				continue
			}
			requests = append(requests, req)
			v := req.GetViewchange().View
			// VIEW or rep.view??
			if v == rep.view {
				valid = true
				break
			}
		}
		if !valid {
			return nil
		}
	}

	if rep.isPrimary(rep.ID) {
		reps := make(map[uint32]int)
		valid = false
		for _, req := range rep.requests["ack"] {
			reqAck := req.GetAck()
			reps[reqAck.Replica]++
			if rep.twoThirds(reps[reqAck.Replica]) { //-2
				valid = true
				break
			}
		}
		if !valid {
			return nil
		}
	}

	return
}

func (rep *Replica) correctSummaries(requests []*pb.Request, summaries []*pb.Summary) (correct bool) {

	// Verify SUMMARIES

	var start uint32
	var digest []byte
	digests := make(map[uint32][]byte)

	for _, summary := range summaries {
		s := summary.Sequence
		d := summary.Digest
		if _d, ok := digests[s]; ok && !pb.EQ(_d, d) {
			return
		} else if !ok {
			digests[s] = d
		}
		if s < start || start == uint32(0) {
			start = s
			digest = d
		}
	}

	var A1 []*pb.Request
	var A2 []*pb.Request

	valid := false
	for _, req := range requests {
		reqViewChange := req.GetViewchange()
		s := reqViewChange.Sequence
		if s <= start {
			A1 = append(A1, req)
		}
		checkpoints := reqViewChange.GetCheckpoints()
		for _, checkpoint := range checkpoints {
			if checkpoint.Sequence == start && pb.EQ(checkpoint.Digest, digest) {
				A2 = append(A2, req)
				break
			}
		}
		if rep.twoThirds(len(A1)) && rep.oneThird(len(A2)) {
			valid = true
			break
		}
	}

	if !valid {
		return
	}

	end := start + CHECKPOINT_PERIOD*CONSTANT_FACTOR

	for seq := start; seq <= end; seq++ {

		valid = false

		for _, summary := range summaries {

			if summary.Sequence != seq {
				continue
			}

			if summary.Digest != nil {

				var view uint32

				for _, req := range requests {
					reqViewChange := req.GetViewchange()
					preps := reqViewChange.GetPreps()
					for _, prep := range preps {
						s := prep.Sequence
						d := prep.Digest
						if s != summary.Sequence || !pb.EQ(d, summary.Digest) {
							continue
						}
						v := prep.View
						if v > view {
							view = v
						}
					}
				}

				verifiedA1 := make(chan bool, 1)

				// Verify A1
				go func() {

					var A1 []*pb.Request

				FOR_LOOP:
					for _, req := range requests {
						reqViewChange := req.GetViewchange()
						s := reqViewChange.Sequence
						if s >= summary.Sequence {
							continue
						}
						preps := reqViewChange.GetPreps()
						for _, prep := range preps {
							s = prep.Sequence
							if s != summary.Sequence {
								continue
							}
							d := prep.Digest
							v := prep.View
							if v > view || (v == view && !pb.EQ(d, summary.Digest)) {
								continue FOR_LOOP
							}
						}
						A1 = append(A1, req)
						if rep.twoThirds(len(A1)) {
							verifiedA1 <- true
							return
						}
					}
					verifiedA1 <- false
				}()

				verifiedA2 := make(chan bool, 1)

				// Verify A2
				go func() {

					var A2 []*pb.Request

					for _, req := range requests {
						reqViewChange := req.GetViewchange()
						prePreps := reqViewChange.GetPrepreps()
						for _, prePrep := range prePreps {
							s := prePrep.Sequence
							d := prePrep.Digest
							v := prePrep.View
							if s == summary.Sequence && pb.EQ(d, summary.Digest) && v >= view {
								A2 = append(A2, req)
								break
							}
						}
						if rep.oneThird(len(A2)) {
							verifiedA2 <- true
							return
						}
					}
					verifiedA2 <- false
				}()

				if !<-verifiedA1 || !<-verifiedA2 {
					continue
				}

				valid = true
				break

			} else {

				var A1 []*pb.Request

			FOR_LOOP:

				for _, req := range requests {

					reqViewChange := req.GetViewchange()

					s := reqViewChange.Sequence

					if s >= summary.Sequence {
						continue
					}

					preps := reqViewChange.GetPreps()
					for _, prep := range preps {
						if prep.Sequence == summary.Sequence {
							continue FOR_LOOP
						}
					}

					A1 = append(A1, req)
					if rep.twoThirds(len(A1)) {
						valid = true
						break
					}
				}
				if valid {
					break
				}
			}
		}
		if !valid {
			return
		}
	}

	return true
}

func (rep *Replica) processNewView(REQ *pb.Request) (success bool) {

	if rep.activeView {
		return
	}

	reqNewView := REQ.GetNewview()

	viewChanges := reqNewView.GetViewchanges()
	requests := rep.correctViewChanges(viewChanges)

	if requests == nil {
		return
	}

	summaries := reqNewView.GetSummaries()
	correct := rep.correctSummaries(requests, summaries)

	if !correct {
		return
	}

	var h uint32
	for _, checkpoint := range rep.checkpoints {
		if checkpoint.Sequence < h || h == uint32(0) {
			h = checkpoint.Sequence
		}
	}

	var s uint32
	for _, summary := range summaries {
		if summary.Sequence < s || s == uint32(0) {
			s = summary.Sequence
		}
		if summary.Sequence > h {
			valid := false
			for _, req := range rep.requests["view-change"] { //in
				if pb.EQ(req.Digest(), summary.Digest) {
					valid = true
					break
				}
			}
			if !valid {
				return
			}
		}
	}

	if h < s {
		return
	}

	// Process new view
	rep.activeView = true

	for _, summary := range summaries {

		if rep.ID != reqNewView.Replica {
			req := pb.ToRequestPrepare(
				reqNewView.View,
				summary.Sequence,
				summary.Digest,
				rep.ID) // the backup sends/logs prepare

			go func() {
				if !rep.hasRequest(req) {
					rep.requestChan <- req
				}
			}()
			if summary.Sequence <= h {
				continue
			}

			if !rep.hasRequest(req) {
				rep.logRequest(req)
			}
		} else {
			if summary.Sequence <= h {
				break
			}
		}

		req := pb.ToRequestPreprepare(
			reqNewView.View,
			summary.Sequence,
			summary.Digest,
			reqNewView.Replica) // new primary pre-prepares

		if !rep.hasRequest(req) {
			rep.logRequest(req)
		}
	}

	var maxSequence uint32
	for _, req := range rep.requests["pre-prepare"] {
		reqPrePrepare := req.GetPreprepare()
		if reqPrePrepare.Sequence > maxSequence {
			maxSequence = reqPrePrepare.Sequence
		}
	}
	rep.sequence = maxSequence
	return true
}

func (rep *Replica) prePrepBySequence(sequence uint32) []*pb.Entry {
	var view uint32
	var requests []*pb.Request
	for _, req := range rep.requests["pre-prepare"] {
		v := req.GetPreprepare().View
		s := req.GetPreprepare().Sequence
		if v >= view && s == sequence {
			view = v
			requests = append(requests, req)
		}
	}
	if requests == nil {
		return nil
	}
	var prePreps []*pb.Entry
	for _, req := range requests {
		v := req.GetPreprepare().View
		if v == view {
			s := req.GetPreprepare().Sequence
			d := req.GetPreprepare().Digest
			prePrep := pb.ToEntry(s, d, v)
			prePreps = append(prePreps, prePrep)
		}
	}
FOR_LOOP:
	for _, prePrep := range prePreps {
		for _, req := range rep.allRequests() { //TODO: optimize
			if pb.EQ(req.Digest(), prePrep.Digest) {
				continue FOR_LOOP
			}
		}
		return nil
	}
	return prePreps
}

func (rep *Replica) prepBySequence(sequence uint32) ([]*pb.Entry, []*pb.Entry) {

	prePreps := rep.prePrepBySequence(sequence)

	if prePreps == nil {
		return nil, nil
	}

	var preps []*pb.Entry

FOR_LOOP:
	for _, prePrep := range prePreps {

		view := prePrep.View
		digest := prePrep.Digest

		replicas := make(map[uint32]int)
		for _, req := range rep.requests["prepare"] {
			reqPrepare := req.GetPrepare()
			v := reqPrepare.View
			s := reqPrepare.Sequence
			d := reqPrepare.Digest
			if v == view && s == sequence && pb.EQ(d, digest) {
				r := reqPrepare.Replica
				replicas[r]++
				if rep.twoThirds(replicas[r]) {
					prep := pb.ToEntry(s, d, v)
					preps = append(preps, prep)
					continue FOR_LOOP
				}
			}
		}
		return prePreps, nil
	}

	return prePreps, preps
}

/*
func (rep *Replica) prepBySequence(sequence uint32) *pb.Entry {
	var REQ *pb.Request
	var view uint32
	// Looking for a commit that
	// replica sent with sequence#
	for _, req := range rep.requests["commit"] {
		v := req.GetCommit().View
		s := req.GetCommit().Sequence
		r := req.GetCommit().Replica
		if v >= view && s == sequence && r == rep.ID {
			REQ = req
			view = v
		}
	}
	if REQ == nil {
		return nil
	}
	entry := pb.ToEntry(sequence, nil, view)
	return entry
}

func (rep *Replica) prePrepBySequence(sequence uint32) *pb.Entry {
	var REQ *pb.Request
	var view uint32
	// Looking for prepare/pre-prepare that
	// replica sent with matching sequence
	for _, req := range rep.requests["pre-prepare"] {
		v := req.GetPreprepare().View
		s := req.GetPreprepare().Sequence
		r := req.GetPreprepare().Replica
		if v >= view && s == sequence && r == rep.ID {
			REQ = req
			view = v
		}
	}
	for _, req := range rep.requests["prepare"] {
		v := req.GetPreprepare().View
		s := req.GetPreprepare().Sequence
		r := req.GetPreprepare().Replica
		if v >= view && s == sequence && r == rep.ID {
			REQ = req
			view = v
		}
	}
	if REQ == nil {
		return nil
	}
	var digest []byte
	switch REQ.Value.(type) {
	case *pb.Request_Preprepare:
		digest = REQ.GetPreprepare().Digest
	case *pb.Request_Prepare:
		digest = REQ.GetPrepare().Digest
	}
	entry := pb.ToEntry(sequence, digest, view)
	return entry
}
*/

func (rep *Replica) requestViewChange(view uint32) {

	if view != rep.view+1 {
		return
	}
	rep.view = view
	rep.activeView = false

	var prePreps []*pb.Entry
	var preps []*pb.Entry

	start := rep.lowWaterMark() + 1
	end := rep.highWaterMark()

	for s := start; s <= end; s++ {
		_prePreps, _preps := rep.prepBySequence(s)
		if _prePreps != nil {
			prePreps = append(prePreps, _prePreps...)
		}
		if _preps != nil {
			preps = append(preps, _preps...)
		}
	}

	sequence := rep.lowWaterMark()

	req := pb.ToRequestViewChange(
		view,
		sequence,
		rep.checkpoints, //
		preps,
		prePreps,
		rep.ID)

	rep.logRequest(req)

	go func() {
		rep.requestChan <- req
	}()

	rep.clearRequestsByView(view)
}

func (rep *Replica) createNewView(view uint32) (request *pb.Request) {

	// Returns RequestNewView if successful, else returns nil
	// create viewChanges
	viewChanges := make([]*pb.ViewChange, len(rep.pendingVC))

	for idx, _ := range viewChanges {
		req := rep.pendingVC[idx]
		viewchanger := req.GetViewchange().Replica
		vc := pb.ToViewChange(viewchanger, req.Digest())
		viewChanges[idx] = vc
	}

	var summaries []*pb.Summary
	var summary *pb.Summary

	start := rep.lowWaterMark() + 1
	end := rep.highWaterMark()

	// select starting checkpoint
FOR_LOOP_1:
	for seq := start; seq <= end; seq++ {

		overLWM := 0
		var digest []byte
		digests := make(map[string]int)

		for _, req := range rep.pendingVC {
			reqViewChange := req.GetViewchange()
			if reqViewChange.Sequence <= seq {
				overLWM++
			}
			for _, checkpoint := range reqViewChange.GetCheckpoints() {
				if checkpoint.Sequence == seq {
					d := checkpoint.Digest
					digests[string(d)]++
					if rep.oneThird(digests[string(d)]) {
						digest = d
						break
					}
				}
			}
			if rep.twoThirds(overLWM) && rep.oneThird(digests[string(digest)]) {
				summary = pb.ToSummary(seq, digest)
				continue FOR_LOOP_1
			}
		}
	}

	if summary == nil {
		return
	}

	summaries = append(summaries, summary)

	start = summary.Sequence
	end = start + CHECKPOINT_PERIOD*CONSTANT_FACTOR

	// select summaries
	// TODO: optimize
FOR_LOOP_2:
	for seq := start; seq <= end; seq++ {

		for _, REQ := range rep.pendingVC {

			sequence := REQ.GetViewchange().Sequence

			if sequence != seq {
				continue
			}

			var A1 []*pb.Request
			var A2 []*pb.Request

			view := REQ.GetViewchange().View
			digest := REQ.Digest()

		FOR_LOOP_3:
			for _, req := range rep.pendingVC {

				reqViewChange := req.GetViewchange()

				if reqViewChange.Sequence < sequence {
					preps := reqViewChange.GetPreps()
					for _, prep := range preps {
						if prep.Sequence != sequence {
							continue
						}
						if prep.View > view || (prep.View == view && !pb.EQ(prep.Digest, digest)) {
							continue FOR_LOOP_3
						}
					}
					A1 = append(A1, req)
				}
				prePreps := reqViewChange.GetPrepreps()
				for _, prePrep := range prePreps {
					if prePrep.Sequence != sequence {
						continue
					}
					if prePrep.View >= view && pb.EQ(prePrep.Digest, digest) {
						A2 = append(A2, req)
						continue FOR_LOOP_3
					}
				}
			}

			if rep.twoThirds(len(A1)) && rep.oneThird(len(A2)) {
				summary = pb.ToSummary(sequence, digest)
				summaries = append(summaries, summary)
				continue FOR_LOOP_2
			}
		}
	}

	request = pb.ToRequestNewView(view, viewChanges, summaries, rep.ID)
	return
}

func (rep *Replica) requestNewView(view uint32) {

	req := rep.createNewView(view)

	if req == nil || rep.hasRequest(req) {
		return
	}

	// Process new view

	success := rep.processNewView(req)

	if !success {
		return
	}

	rep.logRequest(req)

	go func() {
		rep.requestChan <- req
	}()

}
