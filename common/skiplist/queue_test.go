package skiplist

import (
	"fmt"
	"testing"

	"github.com/33cn/chain33/types"
	"github.com/stretchr/testify/assert"
)

type scorer struct {
	score int64
	hash  string
}

func (item *scorer) GetScore() int64 {
	return item.score
}

func (item *scorer) Hash() []byte {
	return []byte(item.hash)
}

func (item *scorer) Compare(cmp Scorer) int {
	switch item.GetScore() - cmp.GetScore() {
	case Equal:
		return Equal
	case Big:
		return Big
	case Small:
		return Small
	default:
		return 0
	}
}

var (
	sc1 = &scorer{1, "111"}
	sc2 = &scorer{2, "222"}
	sc3 = &scorer{3, "333"}
	sc4 = &scorer{4, "444"}
)

func TestQueuePush(t *testing.T) {
	q := NewQueue(10)
	q.Push(sc1)
	assert.Equal(t, 1, q.Size())
	q.Push(sc2)
	assert.Equal(t, 2, q.Size())
	assert.Equal(t, sc2, q.First())
	assert.Equal(t, sc1, q.Last())
}

func TestQueueFind(t *testing.T) {
	q := NewQueue(10)
	q.Push(sc1)
	f1, _ := q.GetItem(string(sc1.Hash()))
	assert.Equal(t, sc1, f1)
	q.Push(sc2)
	f2, _ := q.GetItem(string(sc2.Hash()))
	assert.Equal(t, sc2, f2)
	q.Push(sc3)
	f3, _ := q.GetItem(string(sc3.Hash()))
	assert.Equal(t, sc3, f3)
	f4, err := q.GetItem(string(sc4.Hash()))
	assert.Equal(t, nil, f4)
	assert.Equal(t, types.ErrNotFound, err)
}

func TestQueueDelete(t *testing.T) {
	q := NewQueue(10)
	q.Push(sc1)
	q.Push(sc2)
	q.Push(sc3)
	q.Remove(string(sc3.Hash()))
	assert.Equal(t, 2, q.Size())
	f3, err := q.GetItem(string(sc3.Hash()))
	assert.Equal(t, nil, f3)
	assert.Equal(t, types.ErrNotFound, err)
}

func TestQueueWalk(t *testing.T) {
	q := NewQueue(10)
	q.Push(sc1)
	q.Push(sc2)
	var data [2]string
	i := 0
	q.Walk(0, func(value Scorer) bool {
		data[i] = string(value.Hash())
		i++
		return true
	})
	assert.Equal(t, data[0], "222")
	assert.Equal(t, data[1], "111")

	var data2 [2]string
	i = 0
	q.Walk(0, func(value Scorer) bool {
		data2[i] = string(value.Hash())
		i++
		return false
	})
	assert.Equal(t, data2[0], "222")
	assert.Equal(t, data2[1], "")

	i = 0
	q.Walk(0, func(value Scorer) bool {
		data2[i] = string(value.Hash())
		i++
		return !(i == 2)
	})
	assert.Equal(t, data2[0], "222")
	assert.Equal(t, data2[1], "111")
}

type scoreint struct {
	data int64
}

func (s *scoreint) GetScore() int64 {
	return s.data
}

func (s *scoreint) Hash() []byte {
	return []byte(fmt.Sprint(s.data))
}

func (s *scoreint) Compare(b Scorer) int {
	return Big
}

func TestQueue(t *testing.T) {
	queue := NewQueue(10)
	for i := 0; i < 11; i++ {
		err := queue.Push(&scoreint{data: int64(i)})
		assert.Nil(t, err)
	}
	assert.Equal(t, int64(10), queue.MaxSize())
	err := queue.Push(&scoreint{data: int64(0)})
	assert.Equal(t, err, types.ErrMemFull)
	err = queue.Push(&scoreint{data: int64(1)})
	assert.Equal(t, err, types.ErrTxExist)
	assert.Equal(t, int64(10), queue.MaxSize())

	item := &scoreint{data: int64(1)}
	item2, err := queue.GetItem(string(item.Hash()))
	assert.Nil(t, err)
	assert.Equal(t, item.Hash(), item2.Hash())
}
