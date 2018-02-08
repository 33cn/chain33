package blockchain

import (
	"sync"

	dbm "code.aliyun.com/chain33/chain33/common/db"
	"code.aliyun.com/chain33/chain33/execs/execdrivers"
	"code.aliyun.com/chain33/chain33/queue"
	"code.aliyun.com/chain33/chain33/types"
)

type Query struct {
	db        dbm.DB
	stateHash []byte
	q         *queue.Queue
	mu        sync.Mutex
}

func NewQuery(db dbm.DB, q *queue.Queue, stateHash []byte) *Query {
	return &Query{db: db, q: q, stateHash: stateHash}
}

func (q *Query) Query(driver string, funcname string, param []byte) (types.Message, error) {
	exec, err := execdrivers.LoadExecute(driver)
	if err != nil {
		storelog.Error("load exec err", "name", driver)
		return nil, err
	}
	exec.SetQueryDB(q.db)
	exec.SetDB(execdrivers.NewStateDB(q.q, q.getStateHash()))
	return exec.Query(funcname, param)
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
