package protocol

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"runtime"
	"time"

	"github.com/libp2p/go-libp2p/core"

	"github.com/33cn/chain33/common/log/log15"
	"github.com/33cn/chain33/queue"
	types2 "github.com/33cn/chain33/system/p2p/dht/types"
	"github.com/33cn/chain33/types"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-msgio"
)

var log = log15.New("module", "p2p.protocol")

func init() {
	rand.Seed(time.Now().UnixNano())
}

const messageHeaderLen = 17

var (
	messageHeader = headerSafe([]byte("/protobuf/msgio"))
)

// ReadStream reads message from stream.
func ReadStream(data types.Message, stream network.Stream) error {

	// 兼容历史版本header数据,需要优先读取
	var header [messageHeaderLen]byte
	_, err := io.ReadFull(stream, header[:])
	if err != nil || !bytes.Equal(header[:], messageHeader) {
		log.Error("ReadStream", "pid", stream.Conn().RemotePeer().Pretty(), "protocolID", stream.Protocol(), "read header err", err)
		return err
	}

	reader := msgio.NewReaderSize(stream, types.MaxBlockSize)
	msg, err := reader.ReadMsg()
	// 内部使用了内存池, 回收内存
	defer reader.ReleaseMsg(msg)
	if err != nil {
		log.Error("ReadStream", "pid", stream.Conn().RemotePeer().Pretty(), "protocolID", stream.Protocol(), "read msg err", err)
		return err
	}
	err = types.Decode(msg, data)
	if err != nil {
		log.Error("ReadStream", "pid", stream.Conn().RemotePeer().Pretty(), "protocolID", stream.Protocol(), "decode err", err)
		return err
	}
	return nil
}

// WriteStream writes message to stream.
func WriteStream(data types.Message, stream network.Stream) error {

	_, err := stream.Write(messageHeader)
	if err != nil {
		log.Error("WriteStream", "pid", stream.Conn().RemotePeer().Pretty(), "protocolID", stream.Protocol(), "write header err", err)
		return err
	}
	msg := types.Encode(data)
	writer := msgio.NewWriter(stream)
	err = writer.WriteMsg(msg)
	if err != nil {
		log.Error("WriteStream", "pid", stream.Conn().RemotePeer().Pretty(), "protocolID", stream.Protocol(), "write msg err", err)
		return err
	}
	return nil
}

// CloseStream closes the stream after writing, and waits for the EOF.
func CloseStream(stream network.Stream) {
	if stream == nil {
		return
	}
	err := stream.Close()
	if err != nil {
		log.Debug("CloseStream", "err", err, "protocol ID", stream.Protocol())
	}
}

// AuthenticateMessage authenticates p2p message.
func AuthenticateMessage(message types.Message, stream network.Stream) bool {
	var sign, bin []byte
	// store a temp ref to signature and remove it from message data
	// sign is a string to allow easy reset to zero-value (empty string)
	switch t := message.(type) {
	case *types.P2PRequest:
		sign = t.Headers.Sign
		t.Headers.Sign = nil
		// marshall data without the signature to protobuf3 binary format
		bin = types.Encode(t)
		// restore sig in message data (for possible future use)
		t.Headers.Sign = sign
	case *types.P2PResponse:
		sign = t.Headers.Sign
		t.Headers.Sign = nil
		// marshall data without the signature to protobuf3 binary format
		bin = types.Encode(t)
		// restore sig in message data (for possible future use)
		t.Headers.Sign = sign
	default:
		return false
	}

	// verify the data was authored by the signing peer identified by the public key
	// and signature included in the message
	return verifyData(bin, sign, stream.Conn().RemotePublicKey())
}

// Verify incoming p2p message data integrity
// data: data to verify
// signature: author signature provided in the message payload
// pubKey: public key of remote peer
func verifyData(data []byte, signature []byte, pubKey crypto.PubKey) bool {
	res, err := pubKey.Verify(data, signature)
	if err != nil {
		log.Error("Error authenticating data", "err", err)
		return false
	}

	return res
}

// ReadStreamAndAuthenticate verifies the message after reading it from the stream.
func ReadStreamAndAuthenticate(message types.Message, stream network.Stream) error {
	if err := ReadStream(message, stream); err != nil {
		return err
	}
	if !AuthenticateMessage(message, stream) {
		return types2.ErrWrongSignature
	}

	return nil
}

// signProtoMessage signs an outgoing p2p message payload.
func signProtoMessage(message types.Message, pk crypto.PrivKey) ([]byte, error) {
	if pk == nil {
		log.Error("signProtoMessage", "err msg", "prikey is nil")
		return nil, types.ErrInvalidParam
	}
	return pk.Sign(types.Encode(message))
}

// SignAndWriteStream signs the message before writing it to the stream.
func SignAndWriteStream(message types.Message, stream network.Stream, pk crypto.PrivKey) error {
	switch t := message.(type) {
	case *types.P2PRequest:
		t.Headers = &types.P2PMessageHeaders{
			Version:   types2.Version,
			Timestamp: time.Now().Unix(),
			Id:        rand.Int63(),
		}
		sign, err := signProtoMessage(t, pk)
		if err != nil {
			return err
		}
		t.Headers.Sign = sign
	case *types.P2PResponse:
		t.Headers = &types.P2PMessageHeaders{
			Version:   types2.Version,
			Timestamp: time.Now().Unix(),
			Id:        rand.Int63(),
		}
		sign, err := signProtoMessage(t, pk)
		if err != nil {
			return err
		}
		t.Headers.Sign = sign
	default:
		log.Error("SignAndWriteStream wrong message type")
		return types2.ErrInvalidMessageType
	}
	return WriteStream(message, stream)
}

// HandlerWithClose wraps handler with closing stream and recovering from panic.
func HandlerWithClose(f network.StreamHandler) network.StreamHandler {
	return func(stream network.Stream) {
		defer func() {
			if r := recover(); r != nil {
				log.Error("handle stream", "panic error", r)
				fmt.Println(string(panicTrace(4)))
				_ = stream.Reset()
			}
		}()
		f(stream)
		CloseStream(stream)
	}
}

// HandlerWithWrite wraps handler with writing, closing stream and recovering from panic.
func HandlerWithWrite(f func(resp *types.P2PResponse) error) network.StreamHandler {
	return func(stream network.Stream) {
		var res types.P2PResponse
		err := f(&res)
		if err != nil {
			res.Response = nil
			res.Error = err.Error()
		}
		res.Headers = &types.P2PMessageHeaders{
			Version:   types2.Version,
			Timestamp: time.Now().Unix(),
			Id:        rand.Int63(),
		}
		if err := WriteStream(&res, stream); err != nil {
			log.Error("HandlerWithWrite", "write stream error", err)
			return
		}
	}
}

// HandlerWithRead wraps handler with reading, closing stream and recovering from panic.
func HandlerWithRead(f func(request *types.P2PRequest)) network.StreamHandler {
	return func(stream network.Stream) {
		var req types.P2PRequest
		if err := ReadStream(&req, stream); err != nil {
			log.Error("HandlerWithRead", "read stream error", err)
			return
		}
		f(&req)
	}
}

// HandlerWithAuth wraps HandlerWithRead with authenticating.
func HandlerWithAuth(f func(request *types.P2PRequest)) network.StreamHandler {
	return func(stream network.Stream) {
		var req types.P2PRequest
		if err := ReadStream(&req, stream); err != nil {
			log.Error("HandlerWithAuthAndSign", "read stream error", err)
			return
		}
		if !AuthenticateMessage(&req, stream) {
			return
		}
		f(&req)
	}
}

// HandlerWithRW wraps handler with reading, writing, closing stream and recovering from panic.
func HandlerWithRW(f func(request *types.P2PRequest, response *types.P2PResponse) error) network.StreamHandler {
	return func(stream network.Stream) {
		var req types.P2PRequest
		if err := ReadStream(&req, stream); err != nil {
			log.Error("HandlerWithRW", "read stream error", err)
			return
		}
		var res types.P2PResponse
		err := f(&req, &res)
		if err != nil {
			res.Response = nil
			res.Error = err.Error()
		}
		res.Headers = &types.P2PMessageHeaders{
			Version:   types2.Version,
			Timestamp: time.Now().Unix(),
			Id:        rand.Int63(),
		}
		if err := WriteStream(&res, stream); err != nil {
			log.Error("HandlerWithAuthAndSign", "write stream error", err)
			return
		}
	}
}

// HandlerWithAuthAndSign wraps HandlerWithRW with signing and authenticating.
func HandlerWithAuthAndSign(h core.Host, f func(request *types.P2PRequest, response *types.P2PResponse) error) network.StreamHandler {
	return func(stream network.Stream) {
		var req types.P2PRequest
		if err := ReadStream(&req, stream); err != nil {
			log.Error("HandlerWithAuthAndSign", "read stream error", err)
			return
		}
		if !AuthenticateMessage(&req, stream) {
			return
		}
		var res types.P2PResponse
		err := f(&req, &res)
		if err != nil {
			res.Response = nil
			res.Error = err.Error()
		}
		res.Headers = &types.P2PMessageHeaders{
			Version:   types2.Version,
			Timestamp: time.Now().Unix(),
			Id:        rand.Int63(),
		}
		sign, err := signProtoMessage(&res, h.Peerstore().PrivKey(h.ID()))
		if err != nil {
			log.Error("HandlerWithAuthAndSign", "signProtoMessage error", err)
			return
		}
		res.Headers.Sign = sign
		if err := WriteStream(&res, stream); err != nil {
			log.Error("HandlerWithAuthAndSign", "write stream error", err)
			return
		}
	}
}

//TODO
// Any developer can define his own stream handler wrapper.

// EventHandlerWithRecover warps the event handler with recover for catching the panic while processing message.
func EventHandlerWithRecover(f func(m *queue.Message)) func(m *queue.Message) {
	return func(m *queue.Message) {
		defer func() {
			if r := recover(); r != nil {
				log.Error("handle event", "panic error", r)
				fmt.Println(string(panicTrace(4)))
			}
		}()
		f(m)
	}
}

//TODO
// Any developer can define his own event handler wrapper.

// panicTrace traces panic stack info.
func panicTrace(kb int) []byte {
	s := []byte("/src/runtime/panic.go")
	e := []byte("\ngoroutine ")
	line := []byte("\n")
	stack := make([]byte, kb<<10) //4KB
	length := runtime.Stack(stack, true)
	start := bytes.Index(stack, s)
	stack = stack[start:length]
	start = bytes.Index(stack, line) + 1
	stack = stack[start:]
	end := bytes.LastIndex(stack, line)
	if end != -1 {
		stack = stack[:end]
	}
	end = bytes.Index(stack, e)
	if end != -1 {
		stack = stack[:end]
	}
	stack = bytes.TrimRight(stack, "\n")
	return stack
}

// EOFTimeout is the maximum amount of time to wait to successfully observe an
// EOF on the stream. Defaults to 60 seconds.
var EOFTimeout = time.Second * 60

// ErrExpectedEOF is returned when we read data while expecting an EOF.
var ErrExpectedEOF = errors.New("read data when expecting EOF")

// AwaitEOF waits for an EOF on the given stream, returning an error if that
// fails. It waits at most EOFTimeout (defaults to 1 minute) after which it
// resets the stream.
func AwaitEOF(s network.Stream) error {
	// So we don't wait forever
	_ = s.SetDeadline(time.Now().Add(EOFTimeout))

	// We *have* to observe the EOF. Otherwise, we leak the stream.
	// Now, technically, we should do this *before*
	// returning from SendMessage as the message
	// hasn't really been sent yet until we see the
	// EOF but we don't actually *know* what
	// protocol the other side is speaking.
	n, err := s.Read([]byte{0})
	if n > 0 || err == nil {
		_ = s.Reset()
		return ErrExpectedEOF
	}
	if err != io.EOF {
		_ = s.Reset()
		return err
	}
	_ = s.Close()
	return nil
}

// migrated from github.com/multiformats/go-multicodec@v0.1.6: header.go
func headerSafe(path []byte) []byte {
	l := len(path) + 1 // + \n
	buf := make([]byte, l+1)
	buf[0] = byte(l)
	copy(buf[1:], path)
	buf[l] = '\n'
	return buf
}
