package protocol

import (
	"bufio"
	"bytes"
	"fmt"
	"math/rand"
	"runtime"
	"time"

	"github.com/33cn/chain33/queue"
	types2 "github.com/33cn/chain33/system/p2p/dht/types"
	"github.com/33cn/chain33/types"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/helpers"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	protobufCodec "github.com/multiformats/go-multicodec/protobuf"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// ReadStream reads message from stream.
func ReadStream(data types.Message, stream network.Stream) error {
	decoder := protobufCodec.Multicodec(nil).Decoder(bufio.NewReader(stream))
	err := decoder.Decode(data)
	if err != nil {
		log.Error("ReadStream", "pid", stream.Conn().RemotePeer().Pretty(), "protocolID", stream.Protocol(), "decode err", err)
		return err
	}
	return nil
}

// WriteStream writes message to stream.
func WriteStream(data types.Message, stream network.Stream) error {
	switch data.(type) {
	case *types.P2PRequest, *types.P2PResponse:
	default:
		return types2.ErrInvalidMessageType
	}
	writer := bufio.NewWriter(stream)
	enc := protobufCodec.Multicodec(nil).Encoder(writer)
	err := enc.Encode(data)
	if err != nil {
		log.Error("WriteStream", "pid", stream.Conn().RemotePeer().Pretty(), "protocolID", stream.Protocol(), "encode err", err)
		return err
	}
	err = writer.Flush()
	if err != nil {
		log.Error("WriteStream", "pid", stream.Conn().RemotePeer().Pretty(), "protocolID", stream.Protocol(), "flush err", err)
	}
	return nil
}

// CloseStream closes the stream after writing, and wait for the EOF.
func CloseStream(stream network.Stream) {
	if stream == nil {
		return
	}
	err := helpers.FullClose(stream)
	if err != nil {
		//just log it because it dose not matter
		log.Debug("CloseStream", "err", err)
	}
}

// AuthenticateMessage auth p2p request
func AuthenticateRequest(req *types.P2PRequest, stream network.Stream) bool {
	// store a temp ref to signature and remove it from message data
	// sign is a string to allow easy reset to zero-value (empty string)
	sign := req.Headers.Sign
	req.Headers.Sign = nil

	// marshall data without the signature to protobuf3 binary format
	bin := types.Encode(req)

	// restore sig in message data (for possible future use)
	req.Headers.Sign = sign

	// verify the data was authored by the signing peer identified by the public key
	// and signature included in the message
	return verifyData(bin, sign, stream.Conn().RemotePeer(), stream.Conn().RemotePublicKey())
}

// AuthenticateMessage auth p2p request
func AuthenticateResponse(res *types.P2PResponse, stream network.Stream) bool {
	// store a temp ref to signature and remove it from message data
	// sign is a string to allow easy reset to zero-value (empty string)
	sign := res.Headers.Sign
	res.Headers.Sign = nil

	// marshall data without the signature to protobuf3 binary format
	bin := types.Encode(res)

	// restore sig in message data (for possible future use)
	res.Headers.Sign = sign

	// verify the data was authored by the signing peer identified by the public key
	// and signature included in the message
	return verifyData(bin, sign, stream.Conn().RemotePeer(), stream.Conn().RemotePublicKey())
}

// Verify incoming p2p message data integrity
// data: data to verify
// signature: author signature provided in the message payload
// id: node id of remote peer
// pubKey: public key of remote peer
func verifyData(data []byte, signature []byte, id peer.ID, pubKey crypto.PubKey) bool {
	res, err := pubKey.Verify(data, signature)
	if err != nil {
		log.Error("Error authenticating data", "err", err)
		return false
	}

	return res
}

// ReadResponseAndAuthenticate verifies the message after reading it from the stream.
func ReadResponseAndAuthenticate(res *types.P2PResponse, stream network.Stream) error {
	if err := ReadStream(res, stream); err != nil {
		return err
	}
	if !AuthenticateResponse(res, stream) {
		return types2.ErrWrongSignature
	}
	return nil
}

// SignProtoMessage sign an outgoing p2p message payload
func SignProtoMessage(message types.Message, stream network.Stream) ([]byte, error) {
	privKey := stream.Conn().LocalPrivateKey()
	return privKey.Sign(types.Encode(message))
}

// SignAndWriteStream signs the message before writing it to the stream.
func SignAndWriteStream(message types.Message, stream network.Stream) error {
	switch t := message.(type) {
	case *types.P2PRequest:
		t.Headers = &types.P2PMessageHeaders{
			Version:   types2.Version,
			Timestamp: time.Now().Unix(),
			Id:        rand.Int63(),
		}
		sign, err := SignProtoMessage(t, stream)
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
		sign, err := SignProtoMessage(t, stream)
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

//HandlerWithClose wraps handler with closing stream and recovering from panic.
func HandlerWithClose(f network.StreamHandler) network.StreamHandler {
	return func(stream network.Stream) {
		defer func() {
			if r := recover(); r != nil {
				log.Error("handle stream", "panic error", r)
				fmt.Println(string(panicTrace(4)))
				_ = stream.Reset()
			}
			CloseStream(stream)
		}()
		f(stream)
	}
}

// HandlerWithRead wraps handler with reading, closing stream and recovering from panic.
func HandlerWithRead(f func(stream network.Stream, request *types.P2PRequest)) network.StreamHandler {
	readFunc := func(stream network.Stream) {
		var req types.P2PRequest
		if err := ReadStream(&req, stream); err != nil {
			log.Error("HandlerWithSignCheck", "read stream error", err)
			return
		}
		f(stream, &req)
	}
	return HandlerWithClose(readFunc)
}

// HandlerWithAuth wraps handler with reading, closing stream and recovering from panic.
func HandlerWithAuth(f func(stream network.Stream, request *types.P2PRequest)) network.StreamHandler {
	readFunc := func(stream network.Stream) {
		var req types.P2PRequest
		if err := ReadStream(&req, stream); err != nil {
			log.Error("HandlerWithSignCheck", "read stream error", err)
			return
		}
		if !AuthenticateRequest(&req, stream) {
			return
		}
		f(stream, &req)
	}
	return HandlerWithClose(readFunc)
}

// HandlerWithRW wraps handler with reading, writing, closing stream and recovering from panic.
func HandlerWithRW(f func(request *types.P2PRequest, response *types.P2PResponse) error) network.StreamHandler {
	rwFunc := func(stream network.Stream) {
		var req types.P2PRequest
		if err := ReadStream(&req, stream); err != nil {
			log.Error("HandlerWithSignCheck", "read stream error", err)
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
			log.Error("HandlerWithSignCheck", "write stream error", err)
			return
		}
	}
	return HandlerWithClose(rwFunc)
}

// HandlerWithSignCheck wraps handler with signature and authenticating besides reading, writing, closing stream and recovering from panic.
func HandlerWithSignCheck(f func(request *types.P2PRequest, response *types.P2PResponse) error) network.StreamHandler {
	rwFunc := func(stream network.Stream) {
		var req types.P2PRequest
		if err := ReadStream(&req, stream); err != nil {
			log.Error("HandlerWithSignCheck", "read stream error", err)
			return
		}
		if !AuthenticateRequest(&req, stream) {
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
		sign, err := SignProtoMessage(&res, stream)
		if err != nil {
			log.Error("HandlerWithSignCheck", "SignProtoMessage error", err)
			return
		}
		res.Headers.Sign = sign
		if err := WriteStream(&res, stream); err != nil {
			log.Error("HandlerWithSignCheck", "write stream error", err)
			return
		}
	}
	return HandlerWithClose(rwFunc)
}

//TODO
// Any developer can define his own stream handler wrapper.

// EventHandlerWithRecover warps the event handler with recover for catching the panic while processing.
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
