package gocbcore

import (
	"encoding/binary"
	"github.com/golang/snappy"
	"sync/atomic"
	"time"
)

func isCompressibleOp(command commandCode) bool {
	switch command {
	case cmdSet:
		fallthrough
	case cmdAdd:
		fallthrough
	case cmdReplace:
		fallthrough
	case cmdAppend:
		fallthrough
	case cmdPrepend:
		return true
	}
	return false
}

type memdClient struct {
	parent       *Agent
	conn         memdConn
	opList       memdOpMap
	errorMap     *kvErrorMap
	features     []HelloFeature
	closeNotify  chan bool
	dcpAckSize   int
	dcpFlowRecv  int
	lastActivity int64
	connId       string
}

func newMemdClient(parent *Agent, conn memdConn) *memdClient {
	client := memdClient{
		parent:      parent,
		conn:        conn,
		closeNotify: make(chan bool),
		connId:      parent.clientId + "/" + formatCbUid(randomCbUid()),
	}
	client.run()
	return &client
}

func (client *memdClient) SupportsFeature(feature HelloFeature) bool {
	return checkSupportsFeature(client.features, feature)
}

func (client *memdClient) EnableDcpBufferAck(bufferAckSize int) {
	client.dcpAckSize = bufferAckSize
}

func (client *memdClient) maybeSendDcpBufferAck(packet *memdPacket) {
	packetLen := 24 + len(packet.Extras) + len(packet.Key) + len(packet.Value)

	client.dcpFlowRecv += packetLen
	if client.dcpFlowRecv < client.dcpAckSize {
		return
	}

	ackAmt := client.dcpFlowRecv

	extrasBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(extrasBuf, uint32(ackAmt))

	err := client.conn.WritePacket(&memdPacket{
		Magic:  reqMagic,
		Opcode: cmdDcpBufferAck,
		Extras: extrasBuf,
	})
	if err != nil {
		logWarnf("Failed to dispatch DCP buffer ack: %s", err)
	}

	client.dcpFlowRecv -= ackAmt
}

func (client *memdClient) SetErrorMap(errorMap *kvErrorMap) {
	client.errorMap = errorMap
}

func (client *memdClient) Address() string {
	return client.conn.RemoteAddr()
}

func (client *memdClient) CloseNotify() chan bool {
	return client.closeNotify
}

func (client *memdClient) SendRequest(req *memdQRequest) error {
	addSuccess := client.opList.Add(req)
	if !addSuccess {
		return ErrCancelled
	}

	packet := &req.memdPacket
	if client.SupportsFeature(FeatureSnappy) {
		isCompressed := (packet.Datatype & uint8(DatatypeFlagCompressed)) != 0
		if !isCompressed && isCompressibleOp(packet.Opcode) {
			newPacket := *packet
			newPacket.Value = snappy.Encode(nil, packet.Value)
			newPacket.Datatype = newPacket.Datatype | uint8(DatatypeFlagCompressed)
			packet = &newPacket
		}
	}

	logSchedf("Writing request. %s OP=0x%x. Opaque=%d", client.conn.LocalAddr(), req.Opcode, req.Opaque)

	client.parent.startNetTrace(req)

	err := client.conn.WritePacket(packet)
	if err != nil {
		logDebugf("memdClient write failure: %v", err)
		client.opList.Remove(req)
		return err
	}

	return nil
}

func (client *memdClient) resolveRequest(resp *memdQResponse) {
	opIndex := resp.Opaque

	// Find the request that goes with this response
	req := client.opList.FindAndMaybeRemove(opIndex, resp.Status != StatusSuccess)

	if req == nil {
		// There is no known request that goes with this response.  Ignore it.
		logDebugf("Received response with no corresponding request.")
		return
	}

	if !req.Persistent {
		client.parent.stopNetTrace(req, resp, client)
	}

	isCompressed := (resp.Datatype & uint8(DatatypeFlagCompressed)) != 0
	if isCompressed {
		newValue, err := snappy.Decode(nil, resp.Value)
		if err != nil {
			logDebugf("Failed to decompress value from the server for key `%s`.", req.Key)
			return
		}

		resp.Value = newValue
		resp.Datatype = resp.Datatype & ^uint8(DatatypeFlagCompressed)
	}

	// Give the agent an opportunity to intercept the response first
	var err error
	if resp.Magic == resMagic {
		if resp.Status != StatusSuccess {
			if ok, foundErr := findMemdError(resp.Status); ok {
				err = foundErr
			} else {
				err = newSimpleError(resp.Status)
			}
		}
	}

	if client.parent != nil {
		shortCircuited, routeErr := client.parent.handleOpRoutingResp(resp, req, err)
		if shortCircuited {
			logSchedf("Routing callback intercepted response")
			return
		}

		err = routeErr
	}

	// Call the requests callback handler...
	logSchedf("Dispatching response callback. OP=0x%x. Opaque=%d", resp.Opcode, resp.Opaque)
	req.tryCallback(resp, err)
}

func (client *memdClient) run() {
	dcpBufferQ := make(chan *memdQResponse)
	dcpKillSwitch := make(chan bool)
	dcpKillNotify := make(chan bool)
	go func() {
		for {
			select {
			case resp, more := <-dcpBufferQ:
				if !more {
					dcpKillNotify <- true
					return
				}

				logSchedf("Resolving response OP=0x%x. Opaque=%d", resp.Opcode, resp.Opaque)
				client.resolveRequest(resp)

				if client.dcpAckSize > 0 {
					client.maybeSendDcpBufferAck(&resp.memdPacket)
				}
			case <-dcpKillSwitch:
				close(dcpBufferQ)
			}
		}
	}()

	go func() {
		for {
			resp := &memdQResponse{
				sourceAddr:   client.conn.RemoteAddr(),
				sourceConnId: client.connId,
			}

			err := client.conn.ReadPacket(&resp.memdPacket)
			if err != nil {
				logErrorf("memdClient read failure: %v", err)
				break
			}

			atomic.StoreInt64(&client.lastActivity, time.Now().UnixNano())

			// We handle DCP no-op's directly here so we can reply immediately.
			if resp.memdPacket.Opcode == cmdDcpNoop {
				err := client.conn.WritePacket(&memdPacket{
					Magic:  resMagic,
					Opcode: cmdDcpNoop,
					Opaque: resp.Opaque,
				})
				if err != nil {
					logWarnf("Failed to dispatch DCP noop reply: %s", err)
				}
				continue
			}

			switch resp.memdPacket.Opcode {
			case cmdDcpDeletion:
				fallthrough
			case cmdDcpExpiration:
				fallthrough
			case cmdDcpMutation:
				fallthrough
			case cmdDcpSnapshotMarker:
				fallthrough
			case cmdDcpStreamEnd:
				dcpBufferQ <- resp
				continue
			default:
				logSchedf("Resolving response OP=0x%x. Opaque=%d", resp.Opcode, resp.Opaque)
				client.resolveRequest(resp)
			}
		}

		err := client.conn.Close()
		if err != nil {
			// Lets log an error, as this is non-fatal
			logErrorf("Failed to shut down client connection (%s)", err)
		}

		dcpKillSwitch <- true
		<-dcpKillNotify

		client.opList.Drain(func(req *memdQRequest) {
			req.tryCallback(nil, ErrNetwork)
		})

		close(client.closeNotify)
	}()
}

func (client *memdClient) Close() error {
	return client.conn.Close()
}
