package store

import (
	dbm "github.com/33cn/chain33/common/db"
	"github.com/ipfs/go-datastore"
	dsq "github.com/ipfs/go-datastore/query"
)

//Persistent wraps db.
type Persistent struct {
	db dbm.DB
}

//Get ...
func (p *Persistent) Get(key datastore.Key) (value []byte, err error) {
	v, e := p.db.Get(key.Bytes())
	if v == nil {
		return nil, datastore.ErrNotFound
	}
	return v, e
}

//Has ...
func (p *Persistent) Has(key datastore.Key) (exists bool, err error) {
	value, err := p.db.Get(key.Bytes())
	return value != nil, err
}

//GetSize ...
func (p *Persistent) GetSize(key datastore.Key) (size int, err error) {
	v, e := p.db.Get(key.Bytes())
	return len(v), e
}

//Query ...
func (p *Persistent) Query(q dsq.Query) (dsq.Results, error) {
	return dsq.ResultsWithEntries(q, nil), nil
}

//Put ...
func (p *Persistent) Put(key datastore.Key, value []byte) error {
	return p.db.Set(key.Bytes(), value)
}

//Delete ...
func (p *Persistent) Delete(key datastore.Key) error {
	return p.db.Delete(key.Bytes())
}

//Close ...
func (p *Persistent) Close() error {
	p.db.Close()
	return nil
}

//Batch ...
func (p *Persistent) Batch() (datastore.Batch, error) {
	return datastore.NewBasicBatch(p), nil
}
