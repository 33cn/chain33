package controller

import (
	"code.aliyun.com/chain33/chain33/consensus/solo"
	"code.aliyun.com/chain33/chain33/queue"
)

func NewPlugin(consensusType string, q *queue.Queue) {

	if consensusType == "solo" {
		con := solo.NewSolo()
		con.SetQueue(q)
	} else if consensusType == "raft" {
		// TODO:
	} else if consensusType == "pbft" {
		// TODO:
	} else {
		panic("Unsupported consensus type")
	}
}
