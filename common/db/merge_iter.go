package db

//合并两个迭代器成一个迭代器

import (
	"errors"

	"github.com/syndtr/goleveldb/leveldb/comparer"
)

type dir int

const (
	dirReleased dir = iota - 1
	dirSOI
	dirEOI
	dirForward
	dirSeek
)

// 合并错误列表
var (
	ErrIterReleased = errors.New("ErrIterReleased")
)

type mergedIterator struct {
	itBase
	cmp     comparer.Comparer
	iters   []Iterator
	strict  bool
	reverse bool
	keys    [][]byte
	prevKey []byte
	index   int
	dir     dir
	err     error
}

func assertKey(key []byte) []byte {
	if key == nil {
		panic("leveldb/iterator: nil key")
	}
	return key
}

func (i *mergedIterator) iterErr(iter Iterator) bool {
	if err := iter.Error(); err != nil {
		if i.strict {
			i.err = err
			return true
		}
	}
	return false
}

func (i *mergedIterator) Valid() bool {
	return i.err == nil && i.dir > dirEOI
}

func (i *mergedIterator) Rewind() bool {
	if i.err != nil {
		return false
	} else if i.dir == dirReleased {
		i.err = ErrIterReleased
		return false
	}

	for x, iter := range i.iters {
		switch {
		case iter.Rewind():
			i.keys[x] = assertKey(iter.Key())
		case i.iterErr(iter):
			return false
		default:
			i.keys[x] = nil
		}
	}
	i.dir = dirSOI
	return i.selectKey()
}

func (i *mergedIterator) Seek(key []byte) bool {
	if i.err != nil {
		return false
	} else if i.dir == dirReleased {
		i.err = ErrIterReleased
		return false
	}
	for x, iter := range i.iters {
		switch {
		case iter.Seek(key):
			i.keys[x] = assertKey(iter.Key())
		case i.iterErr(iter):
			return false
		default:
			i.keys[x] = nil
		}
	}
	i.dir = dirSOI
	if i.selectKey() {
		i.dir = dirSeek
		return true
	}
	i.dir = dirSOI
	return false
}

func (i *mergedIterator) compare(key1 []byte, key2 []byte) int {

	if key1 == nil && key2 != nil {
		return 1
	}
	if key1 != nil && key2 == nil {
		return -1
	}
	result := i.cmp.Compare(key1, key2)
	if i.reverse {
		return -result
	}
	return result
}

func (i *mergedIterator) selectKey() bool {
	var key []byte
	for x, tkey := range i.keys {
		if tkey != nil && (key == nil || i.compare(tkey, key) < 0) {
			key = tkey
			i.index = x
		}
	}
	if key == nil {
		i.dir = dirEOI
		return false
	}
	if i.dir == dirSOI {
		i.prevKey = cloneByte(key)
	}
	i.dir = dirForward
	return true
}

// Next next key
func (i *mergedIterator) Next() bool {
	for {

		// 首次next, 需要对内部各迭代器都重置
		if i.dir == dirSOI {
			return i.Rewind()
		}

		if !i.next() {
			return false
		}

		currKey := i.Key()
		if i.compare(currKey, i.prevKey) != 0 {
			i.updatePrevKey(currKey)
			return true
		}
	}
}

func (i *mergedIterator) updatePrevKey(currKey []byte) {

	if cap(i.prevKey) < len(currKey) {
		i.prevKey = make([]byte, len(currKey))
	}
	i.prevKey = i.prevKey[:len(currKey)]
	copy(i.prevKey, currKey)
}

func (i *mergedIterator) next() bool {
	if i.dir == dirEOI || i.err != nil {
		return false
	} else if i.dir == dirReleased {
		i.err = ErrIterReleased
		return false
	}

	x := i.index
	iter := i.iters[x]
	switch {
	case iter.Next():
		i.keys[x] = assertKey(iter.Key())
	case i.iterErr(iter):
		return false
	default:
		i.keys[x] = nil
	}
	return i.selectKey()
}

func (i *mergedIterator) Key() []byte {
	if i.err != nil || i.dir <= dirEOI {
		return nil
	}
	return i.keys[i.index]
}

func (i *mergedIterator) Value() []byte {
	if i.err != nil || i.dir <= dirEOI {
		return nil
	}
	return i.iters[i.index].Value()
}

func (i *mergedIterator) ValueCopy() []byte {
	if i.err != nil || i.dir <= dirEOI {
		return nil
	}
	v := i.iters[i.index].Value()
	return cloneByte(v)
}

func (i *mergedIterator) Close() {
	if i.dir != dirReleased {
		i.dir = dirReleased
		for _, iter := range i.iters {
			iter.Close()
		}
		i.iters = nil
		i.keys = nil
	}
}

func (i *mergedIterator) Error() error {
	return i.err
}

// NewMergedIterator returns an iterator that merges its input. Walking the
// resultant iterator will return all key/value pairs of all input iterators
// in strictly increasing key order, as defined by cmp.
// The input's key ranges may overlap, but there are assumed to be no duplicate
// keys: if iters[i] contains a key k then iters[j] will not contain that key k.
// None of the iters may be nil.
//
// If strict is true the any 'corruption errors' (i.e errors.IsCorrupted(err) == true)
// won't be ignored and will halt 'merged iterator', otherwise the iterator will
// continue to the next 'input iterator'.
func NewMergedIterator(iters []Iterator) Iterator {
	reverse := true
	if len(iters) >= 2 {
		reverse = iters[0].IsReverse()
		for i := 1; i < len(iters); i++ {
			if reverse != iters[i].IsReverse() {
				panic("merge iter not support diff reverse flag")
			}
		}
	}
	return &mergedIterator{
		iters:   iters,
		reverse: reverse,
		cmp:     comparer.DefaultComparer,
		strict:  true,
		keys:    make([][]byte, len(iters)),
		prevKey: make([]byte, 128),
	}
}

type mergedIteratorDB struct {
	iters []IteratorDB
}

// NewMergedIteratorDB 合并两个迭代数据库
func NewMergedIteratorDB(iters []IteratorDB) IteratorDB {
	return &mergedIteratorDB{iters: iters}
}

func (merge *mergedIteratorDB) Iterator(start []byte, end []byte, reverse bool) Iterator {
	iters := make([]Iterator, len(merge.iters))
	for i := 0; i < len(merge.iters); i++ {
		iters[i] = merge.iters[i].Iterator(start, end, reverse)
	}
	return NewMergedIterator(iters)
}
