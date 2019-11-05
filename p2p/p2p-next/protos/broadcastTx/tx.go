package tx

import (
	"io"
	"sync"

	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
	logging "github.com/ipfs/go-log"
	host "github.com/libp2p/go-libp2p-host"
	net "github.com/libp2p/go-libp2p-net"
)

var log = logging.Logger("broadcastTx")

const ID = "/chain33/broadcastTx/1.0.0"

//type Istream
type TxService struct {
	outStream sync.Map
	inStream  sync.Map
}

func NewService(h host.Host, streams sync.Map) *TxService {

	txServer := &TxService{}
	h.SetStreamHandler(ID, txServer.txHandler)
	txServer.outStream = streams

	return txServer
}

//p2pserver 端接收处理TX事件
func (t *TxService) txHandler(instream net.Stream) {

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
		var tx types.Transaction
		err = types.Decode(buf, &tx)
		if err != nil {
			continue
		}
		//TODO

	}

}

//暂时不考虑短哈希广播
func (t *TxService) BroadCastTx(msg *queue.Message) {
	tx, ok := msg.GetData().(*types.Transaction)
	if !ok {
		return
	}

	t.outStream.Range(func(k, v interface{}) bool {
		istream := v.(net.Stream)
		_, err := istream.Write(types.Encode(tx))
		if err != nil {
			istream.Close()
			t.outStream.Delete(k)
		}
		return true

	})

	//把同样的消息发给instream的那些节点
	t.inStream.Range(func(k, v interface{}) bool {
		istream := k.(net.Stream)
		_, err := istream.Write(types.Encode(tx))
		if err != nil {
			istream.Close()
			t.outStream.Delete(k)
		}
		return true

	})
}
