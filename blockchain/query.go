package blockchain

import (
	"sync"

	dbm "gitlab.33.cn/chain33/chain33/common/db"
	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/types"
)

type Query struct {
	db        dbm.DB
	stateHash []byte
	client    queue.Client
	mu        sync.Mutex
}

func NewQuery(db dbm.DB, client queue.Client, stateHash []byte) *Query {
	return &Query{db: db, client: client, stateHash: stateHash}
}

func (q *Query) Query(driver string, funcname string, param []byte) (types.Message, error) {
	msg := q.client.NewMessage("execs", types.EventBlockChainQuery, &types.BlockChainQuery{driver, funcname, q.getStateHash(), param})
	q.client.Send(msg, true)
	msg, err := q.client.Wait(msg)
	if err != nil {
		return nil, err
	}

	return msg.GetData().(types.Message), nil
}

func (q *Query) updateStateHash(stateHash []byte) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.stateHash = stateHash
}

func (q *Query) getStateHash() (stateHash []byte) {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.stateHash
}
