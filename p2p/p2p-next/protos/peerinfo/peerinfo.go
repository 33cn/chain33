package peerinfo

import (
	"bufio"
	"errors"
	"io"
	"sync"
	"time"

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
	client    queue.Client
	host      host.Host
	node      *Node                       // local host
	requests  map[string]*p2p.PingRequest // used to access request data from response handlers
}

func NewService(h host.Host, streams sync.Map) *Service {

	Server := &Service{}
	h.SetStreamHandler(ID, Server.Handler)
	Server.outStream = streams
	Server.host = h

	return Server
}

//p2pserver 端接收处理TX事件
func (t *Service) Handler(stream net.Stream) {

	t.inStream.Store(stream, true)
	var buf []byte
	for {
		_, err := io.ReadFull(stream, buf)
		if err != nil {
			if err == io.EOF {
				continue
			}
			stream.Close()
			t.inStream.Delete(stream)
			return

		}
		//解析处理
		var data types.P2PGetPeerInfo
		err = types.Decode(buf, &data)
		if err != nil {
			continue
		}

		client := t.client
		var peerinfo types.P2PPeerInfo

		msg := client.NewMessage("mempool", types.EventGetMempoolSize, nil)
		err = client.SendTimeout(msg, true, time.Second*10)
		if err != nil {
			log.Error("GetPeerInfo mempool", "Error", err.Error())
		}
		resp, err := client.WaitTimeout(msg, time.Second*10)
		if err != nil {
			log.Error("GetPeerInfo EventGetLastHeader", "Error", err.Error())

		} else {
			meminfo := resp.GetData().(*types.MempoolSize)
			peerinfo.MempoolSize = int32(meminfo.GetSize())
		}

		msg = client.NewMessage("blockchain", types.EventGetLastHeader, nil)
		err = client.SendTimeout(msg, true, time.Minute)
		if err != nil {
			log.Error("GetPeerInfo EventGetLastHeader", "Error", err.Error())
			goto Jump

		}
		resp, err = client.WaitTimeout(msg, time.Second*10)
		if err != nil {
			log.Error("GetPeerInfo EventGetLastHeader", "Error", err.Error())

			goto Jump

		}
	Jump:
		header := resp.GetData().(*types.Header)
		peerinfo.Header = header
		peerinfo.Name = t.host.ID().Pretty()

		peerinfo.Addr = t.host.Addrs()[0].String()
		peerinfo.Port = t.host.Addrs()[0]

		sendbs := types.Encode(&peerinfo)
		_, err = stream.Write(sendbs)
		if err != nil {
			continue
		}

	}

}

func (t *Service) GetPeerInfo(msg *queue.Message, stream net.Stream) ([]*types.P2PPeerInfo, error) {

	data, ok := msg.GetData().(*types.P2PGetPeerInfo)
	if !ok {
		return nil, errors.New("wrong format")
	}
	var peerinfos []*types.P2PPeerInfo
	t.outStream.Range(func(k, v interface{}) bool {
		istream := v.(net.Stream)
		rw := bufio.NewReadWriter(bufio.NewReader(istream), bufio.NewWriter(istream))

		_, err := rw.Write(types.Encode(data))
		if err != nil {
			istream.Close()
			t.outStream.Delete(k)
		}

		var resp []byte
		rn, err := rw.Read(resp)
		if err != nil {
			istream.Close()
			t.outStream.Delete(k)
		}

		var pinfo types.P2PPeerInfo
		err = types.Decode(resp[:rn], &pinfo)
		if err == nil {
			peerinfos = append(peerinfos, &pinfo)
		}

		return true

	})

	return peerinfos, nil

}
