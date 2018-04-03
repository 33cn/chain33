package blockchain

import (
	"sync"

	dbm "gitlab.33.cn/chain33/chain33/common/db"
	"gitlab.33.cn/chain33/chain33/executor"
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
	exec, err := executor.LoadDriver(driver)
	if err != nil {
		chainlog.Error("load exec err", "name", driver)
		return nil, err
	}
	exec.SetQueryDB(q.db)
	exec.SetDB(executor.NewStateDB(q.client.Clone(), q.getStateHash()))
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
