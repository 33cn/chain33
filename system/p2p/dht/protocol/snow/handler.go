package snow

import (
	"encoding/hex"
	"time"

	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/system/p2p/dht/protocol"
	"github.com/33cn/chain33/types"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
)

/*

event handler 为内部系统模块通信消息处理

stream handler 为外部网络节点通信消息处理

*/

func (s *snowman) handleEventChits(msg *queue.Message) {

	req := msg.GetData().(*types.SnowChits)

	pid, _ := peer.Decode(req.PeerName)
	stream, err := s.Host.NewStream(s.Ctx, pid, snowChitsID)
	if err != nil {
		log.Error("handleEventChits", "reqID", req.RequestID, "peer", req.PeerName, "newStream err", err)
		s.sendQueryFailedMsg(types.EventSnowmanQueryFailed, req.RequestID, req.PeerName)
		return
	}

	_ = stream.SetDeadline(time.Now().Add(time.Second * 5))
	defer protocol.CloseStream(stream)

	err = protocol.WriteStream(req, stream)
	if err != nil {
		log.Error("handleEventChits", "reqID", req.RequestID, "peer", req.PeerName, "writeStream err", err)
		s.sendQueryFailedMsg(types.EventSnowmanQueryFailed, req.RequestID, req.PeerName)
	}

}

func (s *snowman) handleEventGetBlock(msg *queue.Message) {

	req := msg.GetData().(*types.SnowGetBlock)

	pid, _ := peer.Decode(req.PeerName)
	stream, err := s.Host.NewStream(s.Ctx, pid, snowGetBlock)
	if err != nil {
		log.Error("handleEventGetBlock", "reqID", req.RequestID, "peer", req.PeerName, "newStream err", err)
		s.sendQueryFailedMsg(types.EventSnowmanGetFailed, req.RequestID, req.PeerName)
		return
	}

	_ = stream.SetDeadline(time.Now().Add(time.Second * 5))
	defer protocol.CloseStream(stream)

	err = protocol.WriteStream(req, stream)
	if err != nil {
		log.Error("handleEventGetBlock", "reqID", req.RequestID, "peer", req.PeerName, "writeStream err", err)
		s.sendQueryFailedMsg(types.EventSnowmanGetFailed, req.RequestID, req.PeerName)
		return
	}

	rep := &types.SnowPutBlock{}
	err = protocol.ReadStream(rep, stream)

	if err != nil || len(rep.GetBlockData()) == 0 {
		log.Error("handleEventGetBlock", "reqID", req.RequestID, "peer", req.PeerName, "readStream err", err)
		s.sendQueryFailedMsg(types.EventSnowmanGetFailed, req.RequestID, req.PeerName)
		return
	}

	rep.PeerName = req.PeerName
	err = s.QueueClient.Send(consensusMsg(types.EventSnowmanPutBlock, rep), false)
	if err != nil {
		log.Error("handleEventGetBlock", "reqID", req.RequestID, "peer", req.PeerName, "send queue err", err)
	}

}

//func (s *snowman) handleEventPutBlock(msg *queue.Message) {
//
//	req := msg.GetData().(*types.SnowPutBlock)
//
//	pid, _ := peer.Decode(req.PeerName)
//	stream, err := s.Host.NewStream(s.Ctx, pid, snowPutBlock)
//	if err != nil {
//		log.Error("handleEventPutBlock", "reqID", req.RequestID, "peer", req.PeerName, "newStream err", err)
//		return
//	}
//
//	_ = stream.SetDeadline(time.Now().Add(time.Second * 5))
//	defer protocol.CloseStream(stream)
//
//	err = protocol.WriteStream(req, stream)
//	if err != nil {
//		log.Error("handleEventPutBlock", "reqID", req.RequestID, "peer", req.PeerName, "writeStream err", err)
//	}
//}

func (s *snowman) handleEventPullQuery(msg *queue.Message) {

	req := msg.GetData().(*types.SnowPullQuery)

	pid, _ := peer.Decode(req.PeerName)
	stream, err := s.Host.NewStream(s.Ctx, pid, snowPullQuery)
	if err != nil {
		log.Error("handleEventPullQuery", "reqID", req.RequestID, "peer", req.PeerName, "newStream err", err)
		s.sendQueryFailedMsg(types.EventSnowmanQueryFailed, req.RequestID, req.PeerName)
		return
	}

	_ = stream.SetDeadline(time.Now().Add(time.Second * 5))
	defer protocol.CloseStream(stream)

	err = protocol.WriteStream(req, stream)
	if err != nil {
		log.Error("handleEventPullQuery", "reqID", req.RequestID, "peer", req.PeerName, "writeStream err", err)
		s.sendQueryFailedMsg(types.EventSnowmanQueryFailed, req.RequestID, req.PeerName)
	}
}

func (s *snowman) handleEventPushQuery(msg *queue.Message) {

	req := msg.GetData().(*types.SnowPushQuery)

	pid, _ := peer.Decode(req.PeerName)
	stream, err := s.Host.NewStream(s.Ctx, pid, snowPushQuery)
	if err != nil {
		log.Error("handleEventPushQuery", "reqID", req.RequestID, "peer", req.PeerName, "newStream err", err)
		s.sendQueryFailedMsg(types.EventSnowmanQueryFailed, req.RequestID, req.PeerName)
		return
	}

	_ = stream.SetDeadline(time.Now().Add(time.Second * 5))
	defer protocol.CloseStream(stream)

	err = protocol.WriteStream(req, stream)
	if err != nil {
		log.Error("handleEventPushQuery", "reqID", req.RequestID, "peer", req.PeerName, "writeStream err", err)
		s.sendQueryFailedMsg(types.EventSnowmanQueryFailed, req.RequestID, req.PeerName)
	}
}

func (s *snowman) handleStreamChits(stream network.Stream) {

	req := &types.SnowChits{}
	err := protocol.ReadStream(req, stream)
	peerName := stream.Conn().RemotePeer().String()
	if err != nil {
		log.Error("handleStreamChits", "reqID", req.RequestID, "peer", peerName, "readStream err", err)
		return
	}
	req.PeerName = peerName
	err = s.QueueClient.Send(consensusMsg(types.EventSnowmanChits, req), false)

	if err != nil {
		log.Error("handleStreamChits", "reqID", req.RequestID, "peer", peerName, "send queue err", err)
	}
}

func (s *snowman) handleStreamGetBlock(stream network.Stream) {

	req := &types.SnowGetBlock{}
	err := protocol.ReadStream(req, stream)
	peerName := stream.Conn().RemotePeer().String()
	if err != nil {
		log.Error("handleStreamGetBlock", "peer", peerName, "readStream err", err)
		return
	}

	reply := &types.SnowPutBlock{RequestID: req.RequestID, PeerName: req.GetPeerName(), BlockHash: req.GetBlockHash()}

	blk, err := s.getBlock(req.GetBlockHash())

	if blk == nil {
		log.Error("handleStreamGetBlock", "reqID", req.RequestID, "hash", hex.EncodeToString(req.GetBlockHash()),
			"peer", peerName, "getBlock err", err)
	} else {
		reply.BlockData = types.Encode(blk)
	}

	err = protocol.WriteStream(reply, stream)
	if err != nil {
		log.Error("handleStreamGetBlock", "reqID", req.RequestID, "peer", peerName, "writeStream err", err)
	}
}

func (s *snowman) handleStreamPullQuery(stream network.Stream) {

	req := &types.SnowPullQuery{}
	err := protocol.ReadStream(req, stream)
	peerName := stream.Conn().RemotePeer().String()
	if err != nil {
		log.Error("handleStreamPullQuery", "reqID", req.RequestID, "peer", peerName, "readStream err", err)
		return
	}
	req.PeerName = peerName
	err = s.QueueClient.Send(consensusMsg(types.EventSnowmanPullQuery, req), false)

	if err != nil {
		log.Error("handleStreamPullQuery", "reqID", req.RequestID, "peer", req.PeerName, "send queue err", err)
	}
}

func (s *snowman) handleStreamPushQuery(stream network.Stream) {

	req := &types.SnowPushQuery{}
	err := protocol.ReadStream(req, stream)
	peerName := stream.Conn().RemotePeer().String()
	if err != nil {
		log.Error("handleStreamPushQuery", "reqID", req.RequestID, "peer", peerName, "readStream err", err)
		return
	}
	req.PeerName = peerName
	err = s.QueueClient.Send(consensusMsg(types.EventSnowmanPushQuery, req), false)

	if err != nil {
		log.Error("handleStreamPushQuery", "reqID", req.RequestID, "peer", req.PeerName, "send queue err", err)
	}
}
