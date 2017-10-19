package blockchain


import (
	//. "common"
	dbm "common/db"
	"container/list"
	"errors"
	"fmt"
	"queue"
	"time"
	"types"
)
type BlockChain struct{}

func New() *BlockChain {
	return &BlockChain{}
}

func (chain *BlockChain) SetQueue(q *queue.Queue) {

}
