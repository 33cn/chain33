package blockchain

import (
	"sync"

	"github.com/33cn/chain33/client"
	dbm "github.com/33cn/chain33/common/db"
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
)

type Query struct {
	db        dbm.DB
	stateHash []byte
	client    queue.Client
	mu        sync.Mutex
	api       client.QueueProtocolAPI
}

func NewQuery(db dbm.DB, qclient queue.Client, stateHash []byte) *Query {
	query := &Query{db: db, client: qclient, stateHash: stateHash}
	query.api, _ = client.New(qclient, nil)
	return query
}

func (q *Query) Query(driver string, funcname string, param types.Message) (types.Message, error) {
	query := &types.ChainExecutor{
		Driver:    driver,
		FuncName:  funcname,
		Param:     types.Encode(param),
		StateHash: q.getStateHash(),
	}
	return q.api.QueryChain(query)
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
