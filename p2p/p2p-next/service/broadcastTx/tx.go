package tx

import (
	core "github.com/libp2p/go-libp2p-core"
	"io"
	"sync"

	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
	logging "github.com/ipfs/go-log"
	host "github.com/libp2p/go-libp2p-core/host"
	net "github.com/libp2p/go-libp2p-core/network"
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
func (t *TxService) txHandler(inStream net.Stream) {

	t.inStream.Store(inStream, true)
	var buf []byte

	for {
		_, err := io.ReadFull(inStream, buf)
		if err != nil {
			if err == io.EOF {
				continue
			}
			inStream.Close()
			t.inStream.Delete(inStream)
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
		istream := v.(core.Stream)
		_, err := istream.Write(types.Encode(tx))
		if err != nil {
			istream.Close()
			t.outStream.Delete(k)
		}
		return true

	})

	//把同样的消息发给instream的那些节点
	t.inStream.Range(func(k, v interface{}) bool {
		istream := k.(core.Stream)
		_, err := istream.Write(types.Encode(tx))
		if err != nil {
			istream.Close()
			t.outStream.Delete(k)
		}
		return true

	})
}
