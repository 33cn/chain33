package snow

import (
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
)

const (
	consensusTopic = "consensus"
)

func consensusMsg(msgID int64, data interface{}) *queue.Message {

	return queue.NewMessage(msgID, consensusTopic, types.EventForFinalizer, data)
}

func (s *snowman) sendQueryFailedMsg(msgID int64, reqID uint32, peerName string) {

	msg := &types.SnowFailedQuery{
		RequestID: reqID,
		PeerName:  peerName,
	}

	err := s.QueueClient.Send(consensusMsg(msgID, msg), false)

	if err != nil {

		log.Error("sendQueryFailedMsg", "reqID", reqID, "send queue err", err)
	}
}

func (s *snowman) getBlock(hash []byte) (*types.Block, error) {

	details, err := s.API.GetBlockByHashes(&types.ReqHashes{Hashes: [][]byte{hash}})
	if err != nil || len(details.GetItems()) < 1 {
		return nil, err
	}

	return details.GetItems()[0].GetBlock(), nil
}
