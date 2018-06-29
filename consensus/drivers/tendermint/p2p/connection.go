package p2p

import (
	"bufio"
	"fmt"
	"io"
	"math"
	"net"
	"runtime/debug"
	"sync/atomic"
	"time"

	"errors"

	"github.com/inconshreveable/log15"
	cmn "gitlab.33.cn/chain33/chain33/consensus/drivers/tendermint/common"
	"encoding/binary"
	"gitlab.33.cn/chain33/chain33/consensus/drivers/tendermint/types"
	"encoding/json"
)

const (
	numBatchMsgPackets = 10
	minReadBufferSize  = 1024
	minWriteBufferSize = 65536
	updateStats        = 2 * time.Second
	pingTimeout        = 40 * time.Second

	// some of these defaults are written in the user config
	// flushThrottle, sendRate, recvRate
	// TODO: remove values present in config
	defaultFlushThrottle = 100 * time.Millisecond

	defaultSendQueueCapacity   = 1
	defaultRecvBufferCapacity  = 4096
	defaultRecvMessageCapacity = 22020096      // 21MB
	defaultSendRate            = int64(512000) // 500KB/s
	defaultRecvRate            = int64(512000) // 500KB/s
	defaultSendTimeout         = 10 * time.Second
)

var (
	ErrAlreadyStarted = errors.New("already started")
	ErrAlreadyStopped = errors.New("already stopped")

	mconnectionlog = log15.New("module", "tendermint-mconnection")
)

type receiveCbFunc func(chID byte, msgBytes []byte)
type errorCbFunc func(interface{})

/*
Each peer has one `MConnection` (multiplex connection) instance.

__multiplex__ *noun* a system or signal involving simultaneous transmission of
several messages along a single channel of communication.

Each `MConnection` handles message transmission on multiple abstract communication
`Channel`s.  Each channel has a globally unique byte id.
The byte id and the relative priorities of each `Channel` are configured upon
initialization of the connection.

There are two methods for sending messages:
	func (m MConnection) Send(chID byte, msg interface{}) bool {}
	func (m MConnection) TrySend(chID byte, msg interface{}) bool {}

`Send(chID, msg)` is a blocking call that waits until `msg` is successfully queued
for the channel with the given id byte `chID`, or until the request times out.
The message `msg` is serialized using the `tendermint/wire` submodule's
`WriteBinary()` reflection routine.

`TrySend(chID, msg)` is a nonblocking call that returns false if the channel's
queue is full.

Inbound message bytes are handled with an onReceive callback function.
*/
type MConnection struct {
	cmn.BaseService

	conn        net.Conn
	bufReader   *bufio.Reader
	bufWriter   *bufio.Writer
	sendMonitor *cmn.Monitor
	recvMonitor *cmn.Monitor
	send        chan struct{}
	pong        chan struct{}
	channels    []*Channel
	channelsIdx map[byte]*Channel
	onReceive   receiveCbFunc
	onError     errorCbFunc
	errored     uint32
	config      *MConnConfig

	quit         chan struct{}
	flushTimer   *cmn.ThrottleTimer // flush writes as necessary but throttled.
	pingTimer    *cmn.RepeatTimer   // send pings periodically
	chStatsTimer *cmn.RepeatTimer   // update channel stats periodically

	LocalAddress  *NetAddress
	RemoteAddress *NetAddress
}

// MConnConfig is a MConnection configuration.
type MConnConfig struct {
	SendRate int64 `mapstructure:"send_rate"`
	RecvRate int64 `mapstructure:"recv_rate"`

	maxMsgPacketPayloadSize int

	flushThrottle time.Duration
}

func (cfg *MConnConfig) maxMsgPacketTotalSize() int {
	return cfg.maxMsgPacketPayloadSize + maxMsgPacketOverheadSize
}

// DefaultMConnConfig returns the default config.
func DefaultMConnConfig() *MConnConfig {
	return &MConnConfig{
		SendRate:                defaultSendRate,
		RecvRate:                defaultRecvRate,
		maxMsgPacketPayloadSize: defaultMaxMsgPacketPayloadSize,
		flushThrottle:           defaultFlushThrottle,
	}
}

// NewMConnection wraps net.Conn and creates multiplex connection
func NewMConnection(conn net.Conn, chDescs []*ChannelDescriptor, onReceive receiveCbFunc, onError errorCbFunc) *MConnection {
	return NewMConnectionWithConfig(
		conn,
		chDescs,
		onReceive,
		onError,
		DefaultMConnConfig())
}

// NewMConnectionWithConfig wraps net.Conn and creates multiplex connection with a config
func NewMConnectionWithConfig(conn net.Conn, chDescs []*ChannelDescriptor, onReceive receiveCbFunc, onError errorCbFunc, config *MConnConfig) *MConnection {
	mconn := &MConnection{
		conn:        conn,
		bufReader:   bufio.NewReaderSize(conn, minReadBufferSize),
		bufWriter:   bufio.NewWriterSize(conn, minWriteBufferSize),
		sendMonitor: cmn.New(0, 0),
		recvMonitor: cmn.New(0, 0),
		send:        make(chan struct{}, 1),
		pong:        make(chan struct{}),
		onReceive:   onReceive,
		onError:     onError,
		config:      config,

		LocalAddress:  NewNetAddress(conn.LocalAddr()),
		RemoteAddress: NewNetAddress(conn.RemoteAddr()),
	}

	// Create channels
	var channelsIdx = map[byte]*Channel{}
	var channels = []*Channel{}

	for _, desc := range chDescs {
		channel := newChannel(mconn, *desc)
		channelsIdx[channel.desc.ID] = channel
		channels = append(channels, channel)
	}
	mconn.channels = channels
	mconn.channelsIdx = channelsIdx

	mconn.BaseService = *cmn.NewBaseService("MConnection", mconn)

	return mconn
}

// OnStart implements BaseService
func (c *MConnection) OnStart() error {

	if err := c.BaseService.OnStart(); err != nil {
		return err
	}

	c.quit = make(chan struct{})
	c.flushTimer = cmn.NewThrottleTimer("flush", c.config.flushThrottle)
	c.pingTimer = cmn.NewRepeatTimer("ping", pingTimeout)
	c.chStatsTimer = cmn.NewRepeatTimer("chStats", updateStats)
	go c.sendRoutine()
	go c.recvRoutine()
	return nil
}

// OnStop implements BaseService
func (c *MConnection) OnStop() {
	c.BaseService.OnStop()
	c.flushTimer.Stop()
	c.pingTimer.Stop()
	c.chStatsTimer.Stop()
	if c.quit != nil {
		close(c.quit)
	}
	c.conn.Close() // nolint: errcheck
	// We can't close pong safely here because
	// recvRoutine may write to it after we've stopped.
	// Though it doesn't need to get closed at all,
	// we close it @ recvRoutine.
	// close(c.pong)
	return
}

func (c *MConnection) String() string {
	return fmt.Sprintf("MConn{%v}", c.conn.RemoteAddr())
}

func (c *MConnection) flush() {
	mconnectionlog.Debug("Flush", "conn", c)
	err := c.bufWriter.Flush()
	if err != nil {
		mconnectionlog.Error("MConnection flush failed", "err", err)
	}
}

// Catch panics, usually caused by remote disconnects.
func (c *MConnection) _recover() {
	if r := recover(); r != nil {
		stack := debug.Stack()
		err := cmn.StackError{r, stack}
		c.stopForError(err)
	}
}

func (c *MConnection) stopForError(r interface{}) {
	c.Stop()
	if atomic.CompareAndSwapUint32(&c.errored, 0, 1) {
		if c.onError != nil {
			c.onError(r)
		}
	}
}

func (c *MConnection) MsgToByte(msg types.ReactorMsg) ([]byte, error) {
	tmp, err := json.Marshal(msg)
	if err != nil {
		mconnectionlog.Error(cmn.Fmt("send bytes, marshal msg [%v] failed:%v", msg.TypeName(), err))
		return nil, err
	}
	context := json.RawMessage(tmp)
	enve := types.MsgEnvelope{
		Kind: msg.TypeName(),
		Data: &context,
	}
	msgBytes, err := json.Marshal(&enve)
	if err != nil {
		mconnectionlog.Error(cmn.Fmt("send bytes, marshal envelope failed:%v", err))
		return nil, err
	}
	return msgBytes, nil
}

// Queues a message to be sent to channel.
func (c *MConnection) Send(chID byte, msg interface{}) bool {

	if !c.IsRunning() {
		return false
	}

	//mconnectionlog.Debug("Send", "channel", chID, "conn", c, "msg", msg) //, "bytes", wire.BinaryBytes(msg))

	// Send message to channel.
	channel, ok := c.channelsIdx[chID]
	if !ok {
		mconnectionlog.Error(cmn.Fmt("Cannot send bytes, unknown channel %X", chID))
		return false
	}

	if v, ok := msg.(types.ReactorMsg); ok {  // checked type assertion
		msgBytes, err := c.MsgToByte(v)
		if err != nil {
			mconnectionlog.Error(cmn.Fmt("send bytes, marshal envelope failed:%v", err))
			return false
		}
		success := channel.sendBytes(msgBytes)
		if success {
			// Wake up sendRoutine if necessary
			select {
			case c.send <- struct{}{}:
			default:
			}
		} else {
			mconnectionlog.Error("Send failed", "channel", chID, "conn", c, "msg", msg)
		}
		return success

	}
	return false
}

// Queues a message to be sent to channel.
// Nonblocking, returns true if successful.
func (c *MConnection) TrySend(chID byte, msg interface{}) bool {

	if !c.IsRunning() {
		return false
	}

	//mconnectionlog.Debug("TrySend", "channel", chID, "conn", c, "msg", msg)

	// Send message to channel.
	channel, ok := c.channelsIdx[chID]
	if !ok {
		mconnectionlog.Error(cmn.Fmt("Cannot send bytes, unknown channel %X", chID))
		return false
	}
	if v, ok := msg.(types.ReactorMsg); ok { // checked type assertion
		msgBytes, err := c.MsgToByte(v)
		if err != nil {
			mconnectionlog.Error(cmn.Fmt("send bytes, marshal envelope failed:%v", err))
			return false
		}
		ok = channel.trySendBytes(msgBytes)
		if ok {
			// Wake up sendRoutine if necessary
			select {
			case c.send <- struct{}{}:
			default:
			}
		}

		return ok
	}
	return false
}

// CanSend returns true if you can send more data onto the chID, false
// otherwise. Use only as a heuristic.
func (c *MConnection) CanSend(chID byte) bool {

	if !c.IsRunning() {
		return false
	}

	channel, ok := c.channelsIdx[chID]
	if !ok {
		mconnectionlog.Error(cmn.Fmt("Unknown channel %X", chID))
		return false
	}
	return channel.canSend()
}

// sendRoutine polls for packets to send from channels.
func (c *MConnection) sendRoutine() {
	defer c._recover()

FOR_LOOP:
	for {
		var n int
		var err error
		select {
		case <-c.flushTimer.Ch:
			// NOTE: flushTimer.Set() must be called every time
			// something is written to .bufWriter.
			c.flush()
		case <-c.chStatsTimer.Chan():
			for _, channel := range c.channels {
				channel.updateStats()
			}
		case <-c.pingTimer.Chan():
			mconnectionlog.Debug("Send Ping")
			n, err = c.bufWriter.Write([]byte{packetTypePing})
			c.sendMonitor.Update(n)
			c.flush()
		case <-c.pong:
			mconnectionlog.Debug("Send Pong")
			n, err = c.bufWriter.Write([]byte{packetTypePong})
			c.sendMonitor.Update(n)
			c.flush()
		case <-c.quit:
			break FOR_LOOP
		case <-c.send:
			// Send some msgPackets
			eof := c.sendSomeMsgPackets()
			if !eof {
				// Keep sendRoutine awake.
				select {
				case c.send <- struct{}{}:
				default:
				}
			}
		}

		if !c.IsRunning() {
			break FOR_LOOP
		}

		if err != nil {
			mconnectionlog.Error("Connection failed @ sendRoutine", "conn", c, "err", err)
			c.stopForError(err)
			break FOR_LOOP
		}
	}

	// Cleanup
}

// Returns true if messages from channels were exhausted.
// Blocks in accordance to .sendMonitor throttling.
func (c *MConnection) sendSomeMsgPackets() bool {
	// Block until .sendMonitor says we can write.
	// Once we're ready we send more than we asked for,
	// but amortized it should even out.
	c.sendMonitor.Limit(c.config.maxMsgPacketTotalSize(), atomic.LoadInt64(&c.config.SendRate), true)

	// Now send some msgPackets.
	for i := 0; i < numBatchMsgPackets; i++ {
		if c.sendMsgPacket() {
			return true
		}
	}
	return false
}

// Returns true if messages from channels were exhausted.
func (c *MConnection) sendMsgPacket() bool {
	// Choose a channel to create a msgPacket from.
	// The chosen channel will be the one whose recentlySent/priority is the least.
	var leastRatio float32 = math.MaxFloat32
	var leastChannel *Channel
	for _, channel := range c.channels {
		// If nothing to send, skip this channel
		if !channel.isSendPending() {
			continue
		}
		// Get ratio, and keep track of lowest ratio.
		ratio := float32(channel.recentlySent) / float32(channel.desc.Priority)
		if ratio < leastRatio {
			leastRatio = ratio
			leastChannel = channel
		}
	}

	// Nothing to send?
	if leastChannel == nil {
		return true
	} else {
		// mconnectionlog.Info("Found a msgPacket to send")
	}

	// Make & send a msgPacket from this channel
	n, err := leastChannel.writeMsgPacketTo(c.bufWriter)
	if err != nil {
		mconnectionlog.Error("Failed to write msgPacket", "err", err)
		c.stopForError(err)
		return true
	}
	c.sendMonitor.Update(n)
	c.flushTimer.Set()
	return false
}

// recvRoutine reads msgPackets and reconstructs the message using the channels' "recving" buffer.
// After a whole message has been assembled, it's pushed to onReceive().
// Blocks depending on how the connection is throttled.
func (c *MConnection) recvRoutine() {
	defer c._recover()

FOR_LOOP:
	for {
		// Block until .recvMonitor says we can read.
		c.recvMonitor.Limit(c.config.maxMsgPacketTotalSize(), atomic.LoadInt64(&c.config.RecvRate), true)

		/*
			// Peek into bufReader for debugging
			if numBytes := c.bufReader.Buffered(); numBytes > 0 {
				log.Info("Peek connection buffer", "numBytes", numBytes, "bytes", log15.Lazy{func() []byte {
					bytes, err := c.bufReader.Peek(cmn.MinInt(numBytes, 100))
					if err == nil {
						return bytes
					} else {
						log.Warn("Error peeking connection buffer", "err", err)
						return nil
					}
				}})
			}
		*/

		// Read packet type
		var buf [1]byte
		n, err := io.ReadFull(c.bufReader, buf[:])
		pktType := buf[0]
		c.recvMonitor.Update(n)
		if err != nil {
			if c.IsRunning() {
				mconnectionlog.Error("Connection failed @ recvRoutine (reading byte)", "conn", c, "err", err)
				c.stopForError(err)
			}
			break FOR_LOOP
		}

		// Read more depending on packet type.
		switch pktType {
		case packetTypePing:
			// TODO: prevent abuse, as they cause flush()'s.
			mconnectionlog.Debug("Receive Ping")
			c.pong <- struct{}{}
		case packetTypePong:
			// do nothing
			mconnectionlog.Debug("Receive Pong")
		case packetTypeMsg:
			pkt, n, err := msgPacket{}, int(0), error(nil)
			//channelID+EOF+bytes_of_len+max_len_of_int(4)+len_of_byte=1+1+1+4+len(packet.Bytes)
			buf := make([]byte, 6)
			_, err = io.ReadFull(c.bufReader, buf)
			if err != nil  {
				mconnectionlog.Error("Connection failed @ recvRoutine", "conn", c, "err", err)
				c.stopForError(err)
				panic(fmt.Sprintf("MConnection recvRoutine packetTypeMsg failed 1:%v",err))
			}
			pkt.ChannelID = buf[0]
			pkt.EOF = buf[1]
			len := binary.BigEndian.Uint32(buf[2:])
			if len > 0 {
				buf2 := make([]byte, len)
				_, err = io.ReadFull(c.bufReader, buf2)
				if err != nil  {
					mconnectionlog.Error("Connection failed @ recvRoutine", "conn", c, "err", err)
					c.stopForError(err)
					panic(fmt.Sprintf("MConnection recvRoutine packetTypeMsg failed 2:%v",err))
				}
				pkt.Bytes = buf2
			}

			//wire.ReadBinaryPtr(&pkt, c.bufReader, c.config.maxMsgPacketTotalSize(), &n, &err)
			c.recvMonitor.Update(n)
			if err != nil {
				if c.IsRunning() {
					mconnectionlog.Error("Connection failed @ recvRoutine", "conn", c, "err", err)
					c.stopForError(err)
				}
				break FOR_LOOP
			}
			channel, ok := c.channelsIdx[pkt.ChannelID]
			if !ok || channel == nil {
				err := fmt.Errorf("Unknown channel %X", pkt.ChannelID)
				mconnectionlog.Error("Connection failed @ recvRoutine", "conn", c, "err", err)
				c.stopForError(err)
			}

			msgBytes, err := channel.recvMsgPacket(pkt)
			if err != nil {
				if c.IsRunning() {
					mconnectionlog.Error("Connection failed @ recvRoutine", "conn", c, "err", err)
					c.stopForError(err)
				}
				break FOR_LOOP
			}
			if msgBytes != nil {
				//mconnectionlog.Debug("Received bytes", "chID", pkt.ChannelID, "msgBytes", msgBytes)
				// NOTE: This means the reactor.Receive runs in the same thread as the p2p recv routine
				c.onReceive(pkt.ChannelID, msgBytes)
			}
		default:
			err := fmt.Errorf("Unknown message type %X", pktType)
			mconnectionlog.Error("Connection failed @ recvRoutine", "conn", c, "err", err)
			c.stopForError(err)
		}

		// TODO: shouldn't this go in the sendRoutine?
		// Better to send a ping packet when *we* haven't sent anything for a while.
		c.pingTimer.Reset()
	}

	// Cleanup
	close(c.pong)
	for range c.pong {
		// Drain
	}
}

type ConnectionStatus struct {
	SendMonitor cmn.Status
	RecvMonitor cmn.Status
	Channels    []ChannelStatus
}

type ChannelStatus struct {
	ID                byte
	SendQueueCapacity int
	SendQueueSize     int
	Priority          int
	RecentlySent      int64
}

func (c *MConnection) Status() ConnectionStatus {
	var status ConnectionStatus
	status.SendMonitor = c.sendMonitor.Status()
	status.RecvMonitor = c.recvMonitor.Status()
	status.Channels = make([]ChannelStatus, len(c.channels))
	for i, channel := range c.channels {
		status.Channels[i] = ChannelStatus{
			ID:                channel.desc.ID,
			SendQueueCapacity: cap(channel.sendQueue),
			SendQueueSize:     int(channel.sendQueueSize), // TODO use atomic
			Priority:          channel.desc.Priority,
			RecentlySent:      channel.recentlySent,
		}
	}
	return status
}

//-----------------------------------------------------------------------------

type ChannelDescriptor struct {
	ID                  byte
	Priority            int
	SendQueueCapacity   int
	RecvBufferCapacity  int
	RecvMessageCapacity int
}

func (chDesc ChannelDescriptor) FillDefaults() (filled ChannelDescriptor) {
	if chDesc.SendQueueCapacity == 0 {
		chDesc.SendQueueCapacity = defaultSendQueueCapacity
	}
	if chDesc.RecvBufferCapacity == 0 {
		chDesc.RecvBufferCapacity = defaultRecvBufferCapacity
	}
	if chDesc.RecvMessageCapacity == 0 {
		chDesc.RecvMessageCapacity = defaultRecvMessageCapacity
	}
	filled = chDesc
	return
}

// TODO: lowercase.
// NOTE: not goroutine-safe.
type Channel struct {
	conn          *MConnection
	desc          ChannelDescriptor
	sendQueue     chan []byte
	sendQueueSize int32 // atomic.
	recving       []byte
	sending       []byte
	recentlySent  int64 // exponential moving average

	maxMsgPacketPayloadSize int
}

func newChannel(conn *MConnection, desc ChannelDescriptor) *Channel {
	desc = desc.FillDefaults()
	if desc.Priority <= 0 {
		cmn.PanicSanity("Channel default priority must be a positive integer")
	}
	return &Channel{
		conn:                    conn,
		desc:                    desc,
		sendQueue:               make(chan []byte, desc.SendQueueCapacity),
		recving:                 make([]byte, 0, desc.RecvBufferCapacity),
		maxMsgPacketPayloadSize: conn.config.maxMsgPacketPayloadSize,
	}
}

// Queues message to send to this channel.
// Goroutine-safe
// Times out (and returns false) after defaultSendTimeout
func (ch *Channel) sendBytes(bytes []byte) bool {
	select {
	case ch.sendQueue <- bytes:
		atomic.AddInt32(&ch.sendQueueSize, 1)
		return true
	case <-time.After(defaultSendTimeout):
		return false
	}
}

// Queues message to send to this channel.
// Nonblocking, returns true if successful.
// Goroutine-safe
func (ch *Channel) trySendBytes(bytes []byte) bool {
	select {
	case ch.sendQueue <- bytes:
		atomic.AddInt32(&ch.sendQueueSize, 1)
		return true
	default:
		return false
	}
}

// Goroutine-safe
func (ch *Channel) loadSendQueueSize() (size int) {
	return int(atomic.LoadInt32(&ch.sendQueueSize))
}

// Goroutine-safe
// Use only as a heuristic.
func (ch *Channel) canSend() bool {
	return ch.loadSendQueueSize() < defaultSendQueueCapacity
}

// Returns true if any msgPackets are pending to be sent.
// Call before calling nextMsgPacket()
// Goroutine-safe
func (ch *Channel) isSendPending() bool {
	if len(ch.sending) == 0 {
		if len(ch.sendQueue) == 0 {
			return false
		}
		ch.sending = <-ch.sendQueue
	}
	return true
}

// Creates a new msgPacket to send.
// Not goroutine-safe
func (ch *Channel) nextMsgPacket() msgPacket {
	packet := msgPacket{}
	packet.ChannelID = ch.desc.ID
	maxSize := ch.maxMsgPacketPayloadSize
	packet.Bytes = ch.sending[:cmn.MinInt(maxSize, len(ch.sending))]
	if len(ch.sending) <= maxSize {
		packet.EOF = byte(0x01)
		ch.sending = nil
		atomic.AddInt32(&ch.sendQueueSize, -1) // decrement sendQueueSize
	} else {
		packet.EOF = byte(0x00)
		ch.sending = ch.sending[cmn.MinInt(maxSize, len(ch.sending)):]
	}
	return packet
}

// Writes next msgPacket to w.
// Not goroutine-safe
func (ch *Channel) writeMsgPacketTo(w io.Writer) (n int, err error) {
	packet := ch.nextMsgPacket()
	//mconnectionlog.Debug("Write Msg Packet", "conn", ch.conn, "packet", packet)
	writeMsgPacketTo(packet, w, &n, &err)
	if err == nil {
		ch.recentlySent += int64(n)
	}
	return
}

func writeMsgPacketTo(packet msgPacket, w io.Writer, n *int, err *error) {
	*n, *err = w.Write([]byte{packetTypeMsg})
	//channelID+EOF+lenNum(int)+len_of_byte=1+1+4+len(packet.Bytes)
	buf := make([]byte, 6+len(packet.Bytes))
	buf[0] = packet.ChannelID
	buf[1] = packet.EOF
	len := len(packet.Bytes)
	if len > 0 {
		frame := make([]byte, 4)
		binary.BigEndian.PutUint32(frame, uint32(len))
		copy(buf[2:], frame)
		copy(buf[6:], packet.Bytes)
	}
	w.Write(buf)
	//wire.WriteBinary(packet, w, n, err)
}

// Handles incoming msgPackets. Returns a msg bytes if msg is complete.
// Not goroutine-safe
func (ch *Channel) recvMsgPacket(packet msgPacket) ([]byte, error) {
	//ch.Logger.Debug("Read Msg Packet", "conn", ch.conn, "packet", packet)
	if ch.desc.RecvMessageCapacity < len(ch.recving)+len(packet.Bytes) {
		return nil, errors.New("Error: binary read overflow")
	}
	ch.recving = append(ch.recving, packet.Bytes...)
	if packet.EOF == byte(0x01) {
		msgBytes := ch.recving
		// clear the slice without re-allocating.
		// http://stackoverflow.com/questions/16971741/how-do-you-clear-a-slice-in-go
		//   suggests this could be a memory leak, but we might as well keep the memory for the channel until it closes,
		//	at which point the recving slice stops being used and should be garbage collected
		ch.recving = ch.recving[:0] // make([]byte, 0, ch.desc.RecvBufferCapacity)
		return msgBytes, nil
	}
	return nil, nil
}

// Call this periodically to update stats for throttling purposes.
// Not goroutine-safe
func (ch *Channel) updateStats() {
	// Exponential decay of stats.
	// TODO: optimize.
	ch.recentlySent = int64(float64(ch.recentlySent) * 0.8)
}

//-----------------------------------------------------------------------------

const (
	defaultMaxMsgPacketPayloadSize = 10*1024*1024//1024

	maxMsgPacketOverheadSize = 10 // It's actually lower but good enough
	packetTypePing           = byte(0x01)
	packetTypePong           = byte(0x02)
	packetTypeMsg            = byte(0x03)
)

// Messages in channels are chopped into smaller msgPackets for multiplexing.
type msgPacket struct {
	ChannelID byte
	EOF       byte // 1 means message ends here.
	Bytes     []byte
}

func (p msgPacket) String() string {
	return fmt.Sprintf("MsgPacket{%X:%X T:%X}", p.ChannelID, p.Bytes, p.EOF)
}
