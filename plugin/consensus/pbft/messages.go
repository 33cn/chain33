package pbft

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"io"
	"net"

	"github.com/golang/protobuf/proto"
	"gitlab.33.cn/chain33/chain33/types"
)

// Digest

func EQ(d1 []byte, d2 []byte) bool {
	if len(d1) != len(d2) {
		return false
	}
	for idx, b := range d1 {
		if b != d2[idx] {
			return false
		}
	}
	return true
}

// Checkpoint

func ToCheckpoint(sequence uint32, digest []byte) *types.Checkpoint {
	return &types.Checkpoint{sequence, digest}
}

// Entry

func ToEntry(sequence uint32, digest []byte, view uint32) *types.Entry {
	return &types.Entry{sequence, digest, view}
}

// ViewChange

func ToViewChange(viewchanger uint32, digest []byte) *types.ViewChange {
	return &types.ViewChange{viewchanger, digest}
}

// Summary

func ToSummary(sequence uint32, digest []byte) *types.Summary {
	return &types.Summary{sequence, digest}
}

// Request

func ToRequestClient(op *types.Operation, timestamp, client string) *types.Request {
	return &types.Request{
		Value: &types.Request_Client{
			&types.RequestClient{op, timestamp, client}},
	}
}

func ToRequestPreprepare(view, sequence uint32, digest []byte, replica uint32) *types.Request {
	return &types.Request{
		Value: &types.Request_Preprepare{
			&types.RequestPrePrepare{view, sequence, digest, replica}},
	}
}

func ToRequestPrepare(view, sequence uint32, digest []byte, replica uint32) *types.Request {
	return &types.Request{
		Value: &types.Request_Prepare{
			&types.RequestPrepare{view, sequence, digest, replica}},
	}
}

func ToRequestCommit(view, sequence, replica uint32) *types.Request {
	return &types.Request{
		Value: &types.Request_Commit{
			&types.RequestCommit{view, sequence, replica}},
	}
}

func ToRequestCheckpoint(sequence uint32, digest []byte, replica uint32) *types.Request {
	return &types.Request{
		Value: &types.Request_Checkpoint{
			&types.RequestCheckpoint{sequence, digest, replica}},
	}
}

func ToRequestViewChange(view, sequence uint32, checkpoints []*types.Checkpoint, preps, prePreps []*types.Entry, replica uint32) *types.Request {
	return &types.Request{
		Value: &types.Request_Viewchange{
			&types.RequestViewChange{view, sequence, checkpoints, preps, prePreps, replica}},
	}
}

func ToRequestAck(view, replica, viewchanger uint32, digest []byte) *types.Request {
	return &types.Request{
		Value: &types.Request_Ack{
			&types.RequestAck{view, replica, viewchanger, digest}},
	}
}

func ToRequestNewView(view uint32, viewChanges []*types.ViewChange, summaries []*types.Summary, replica uint32) *types.Request {
	return &types.Request{
		Value: &types.Request_Newview{
			&types.RequestNewView{view, viewChanges, summaries, replica}},
	}
}

// Request Methods

func ReqDigest(req *types.Request) []byte {
	if req == nil {
		return nil
	}
	bytes := md5.Sum([]byte(req.String()))
	return bytes[:]
}

/*func (req *Request) LowWaterMark() uint32 {
	// only for requestViewChange
	reqViewChange := req.GetViewchange()
	checkpoints := reqViewChange.GetCheckpoints()
	lastStable := checkpoints[len(checkpoints)-1]
	lwm := lastStable.Sequence
	return lwm
}*/

// Reply

func ToReply(view uint32, timestamp, client string, replica uint32, result *types.Result) *types.ClientReply {
	return &types.ClientReply{view, timestamp, client, replica, result}
}

// Reply Methods

func RepDigest(reply fmt.Stringer) []byte {
	if reply == nil {
		return nil
	}
	bytes := md5.Sum([]byte(reply.String()))
	return bytes[:]
}

// Write proto message

func WriteMessage(addr string, msg proto.Message) error {
	conn, err := net.Dial("tcp", addr)
	defer conn.Close()
	if err != nil {
		return err
	}
	bz, err := proto.Marshal(msg)
	if err != nil {
		return err
	}
	n, err := conn.Write(bz)
	plog.Debug("size of byte is", "", n)
	return err
}

// Read proto message

func ReadMessage(conn io.Reader, msg proto.Message) error {
	var buf bytes.Buffer
	n, err := io.Copy(&buf, conn)
	plog.Debug("size of byte is", "", n)
	if err != nil {
		return err
	}
	err = proto.Unmarshal(buf.Bytes(), msg)
	return err
}
