package store

import (
	dbm "github.com/33cn/chain33/common/db"
	"github.com/ipfs/go-datastore"
	dsq "github.com/ipfs/go-datastore/query"
)

type Persistent struct {
	db dbm.DB
}

func (p *Persistent) Get(key datastore.Key) (value []byte, err error) {
	v, e := p.db.Get(key.Bytes())
	if v == nil {
		return nil, datastore.ErrNotFound
	}
	return v, e
}

func (p *Persistent) Has(key datastore.Key) (exists bool, err error) {
	value, err := p.db.Get(key.Bytes())
	return value != nil, err
}

func (p *Persistent) GetSize(key datastore.Key) (size int, err error) {
	return -1, datastore.ErrNotFound
}

func (p *Persistent) Query(q dsq.Query) (dsq.Results, error) {
	return dsq.ResultsWithEntries(q, nil), nil
}

func (p *Persistent) Put(key datastore.Key, value []byte) error {
	return p.db.Set(key.Bytes(), value)
}

func (p *Persistent) Delete(key datastore.Key) error {
	return p.db.Delete(key.Bytes())
}

func (p *Persistent) Close() error {
	p.db.Close()
	return nil
}

func (p *Persistent) Batch() (datastore.Batch, error) {
	return datastore.NewBasicBatch(p), nil
}
