package mempool

import "code.aliyun.com/chain33/chain33/queue"

type Mempool struct{}

func New() *Mempool {
	return &Mempool{}
}

func (mem *Mempool) SetQueue(q *queue.Queue) {

}
