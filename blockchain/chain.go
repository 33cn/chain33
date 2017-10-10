package blockchain

import "code.aliyun.com/chain33/chain33/queue"

type BlockChain struct{}

func New() *BlockChain {
	return &BlockChain{}
}

func (chain *BlockChain) SetQueue(q *queue.Queue) {

}
