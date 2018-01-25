package blockchain

import (
	dbm "code.aliyun.com/chain33/chain33/common/db"
	"code.aliyun.com/chain33/chain33/execs/execdrivers"
	"code.aliyun.com/chain33/chain33/types"
)

type Query struct {
	db dbm.DB
}

func NewQuery(db dbm.DB) *Query {
	return &Query{db}
}

func (q *Query) Query(driver string, funcname string, param types.Message) (types.Message, error) {
	exec, err := execdrivers.LoadExecute(driver)
	if err != nil {
		return nil, err
	}
	exec.SetQueryDB(q.db)
	return exec.Query(funcname, param)
}
