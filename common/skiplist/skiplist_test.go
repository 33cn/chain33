package skiplist

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	s1 = &SkipValue{1, "111"}
	s2 = &SkipValue{2, "222"}
	s3 = &SkipValue{3, "333"}
)

func TestInsert(t *testing.T) {
	l := NewSkipList(nil)
	l.Insert(s1)
	assert.Equal(t, 1, l.Len())
	l.Insert(s2)
	assert.Equal(t, 2, l.Len())
	iter := l.GetIterator()
	assert.Equal(t, int64(2), iter.First().Score)
	assert.Equal(t, "222", iter.First().Value.(string))
	assert.Equal(t, int64(1), iter.Last().Score)
	assert.Equal(t, "111", iter.Last().Value.(string))
}

func TestFind(t *testing.T) {
	l := NewSkipList(nil)
	l.Insert(s1)
	assert.Equal(t, s1, l.Find(s1))
	l.Insert(s2)
	assert.Equal(t, s2, l.Find(s2))
	l.Insert(s3)
	assert.Equal(t, s3, l.Find(s3))
}

func TestDelete(t *testing.T) {
	l := NewSkipList(nil)
	l.Insert(s1)
	l.Insert(s2)
	l.Delete(s1)
	assert.Equal(t, 1, l.Len())
	assert.Equal(t, (*SkipValue)(nil), l.Find(s1))
	assert.Equal(t, s2, l.Find(s2))
}

func TestWalk(t *testing.T) {
	l := NewSkipList(nil)
	l.Insert(s1)
	l.Insert(s2)
	var data [2]string
	i := 0
	l.Walk(func(value interface{}) bool {
		data[i] = value.(string)
		i++
		return true
	})
	assert.Equal(t, data[0], "222")
	assert.Equal(t, data[1], "111")

	var data2 [2]string
	i = 0
	l.Walk(func(value interface{}) bool {
		data2[i] = value.(string)
		i++
		return false
	})
	assert.Equal(t, data2[0], "222")
	assert.Equal(t, data2[1], "")

	l.Walk(nil)
	iter := l.GetIterator()
	assert.Equal(t, int64(2), iter.First().Score)
	assert.Equal(t, "222", iter.First().Value.(string))
	assert.Equal(t, int64(1), iter.Last().Score)
	assert.Equal(t, "111", iter.Last().Value.(string))
	l.Print()

}

func TestWalkS(t *testing.T) {
	l := NewSkipList(nil)
	l.Insert(s1)
	l.Insert(s2)
	var score [2]int64
	var data [2]string
	i := 0
	l.WalkS(func(value interface{}) bool {
		score[i] = value.(*SkipValue).Score
		data[i] = value.(*SkipValue).Value.(string)
		i++
		return true
	})
	assert.Equal(t, data[0], "222")
	assert.Equal(t, data[1], "111")
	assert.Equal(t, int64(2), score[0])
	assert.Equal(t, int64(1), score[1])

	var score2 [2]int64
	var data2 [2]string
	i = 0
	l.WalkS(func(value interface{}) bool {
		score2[i] = value.(*SkipValue).Score
		data2[i] = value.(*SkipValue).Value.(string)
		i++
		return false
	})
	assert.Equal(t, "222", data2[0])
	assert.Equal(t, "", data2[1])
	assert.Equal(t, int64(2), score2[0])
	assert.Equal(t, int64(0), score2[1])
}
