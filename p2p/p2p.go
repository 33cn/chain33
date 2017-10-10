package p2p

import "code.aliyun.com/chain33/chain33/queue"

type P2p struct{}

func New() *P2p {
	return &P2p{}
}

func (network *P2p) SetQueue(q *queue.Queue) {

}
