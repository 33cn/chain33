package peerinfo

import (
	"bufio"
	"io"
	"sync"

	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
	logging "github.com/ipfs/go-log"
	host "github.com/libp2p/go-libp2p-host"
	net "github.com/libp2p/go-libp2p-net"
)

var log = logging.Logger("peerinfo")

const ID = "/chain33/peerinfo/1.0.0"

//type Istream
type Service struct {
	outStream sync.Map
	inStream  sync.Map
}

func NewService(h host.Host, streams sync.Map) *Service {

	Server := &Service{}
	h.SetStreamHandler(ID, Server.Handler)
	Server.outStream = streams

	return Server
}

//p2pserver 端接收处理TX事件
func (t *Service) Handler(instream net.Stream) {

	t.inStream.Store(instream, true)
	var buf []byte
	for {
		_, err := io.ReadFull(instream, buf)
		if err != nil {
			if err == io.EOF {
				continue
			}
			instream.Close()
			t.inStream.Delete(instream)
			return

		}
		//解析处理
		var data types.P2PGetPeerInfo
		err = types.Decode(buf, &data)
		if err != nil {
			continue
		}
		//TODO

	}

}

func (t *Service) GetPeerInfo(msg *queue.Message, stream net.Stream) ([]*types.P2PPeerInfo, error) {

	data, ok := msg.GetData().(*types.P2PGetPeerInfo)
	if !ok {
		return
	}

	t.outStream.Range(func(k, v interface{}) bool {
		istream := v.(net.Stream)
		rw := bufio.NewReadWriter(bufio.NewReader(istream), bufio.NewWriter(istream))

		_, err := rw.Write(types.Encode(data))
		if err != nil {
			istream.Close()
			t.outStream.Delete(k)
		}

		var resp []byte
		_, err := rw.Read(resp)
		if err != nil {
			istream.Close()
			t.outStream.Delete(k)
		}

		if peerinfo, ok := resp.(types.P2PPeerInfo); ok {
			//TODO
		}

		return true

	})

}
