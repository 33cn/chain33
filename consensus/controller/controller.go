package controller

import "code.aliyun.com/chain33/chain33/queue"
import "code.aliyun.com/chain33/chain33/consensus/solo"

func NewPlugin(consensusType string, q *queue.Queue) {
	if consensusType == "solo" {
		con := solo.NewSolo()
		con.SetQueue(q)
	} else if consensusType == "raft" {
		// TODO:
	} else if consensusType == "pbft" {
		// TODO:
	} else {
		panic("不支持的共识类型")
	}
}
