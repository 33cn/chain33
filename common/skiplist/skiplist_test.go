package skiplist

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	s1  = &SkipValue{1, "111"}
	s2  = &SkipValue{2, "222"}
	s3  = &SkipValue{3, "333"}
	s4  = &SkipValue{4, "444"}
	s5  = &SkipValue{5, "555"}
	s6  = &SkipValue{6, "666"}
	s7  = &SkipValue{7, "777"}
	s8  = &SkipValue{8, "888"}
	s9  = &SkipValue{9, "999"}
	s10 = &SkipValue{10, "101010"}
	s11 = &SkipValue{11, "111111"}
	s12 = &SkipValue{12, "121212"}
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
	l.Insert(s3)
	l.Delete(s3)
	assert.Equal(t, 2, l.Len())
	assert.Equal(t, (*SkipValue)(nil), l.Find(s3))
	assert.Equal(t, s2, l.Find(s2))
}

func TestWalk(t *testing.T) {
	l := NewSkipList(nil)
	l.Insert(s1)
	l.Insert(s2)
	l.Insert(s3)
	l.Insert(s4)
	l.Insert(s5)
	l.Insert(s6)
	l.Insert(s7)
	l.Insert(s8)
	l.Insert(s9)
	l.Insert(s10)
	l.Insert(s11)
	l.Insert(s12)
	var data [100]string
	i := 0
	l.Walk(func(value interface{}) bool {
		data[i] = value.(string)
		i++
		return true
	})
	assert.Equal(t, data[0], "121212")
	assert.Equal(t, data[1], "111111")

	var data2 [100]string
	i = 0
	l.Walk(func(value interface{}) bool {
		data2[i] = value.(string)
		i++
		return false
	})
	assert.Equal(t, data2[0], "121212")
	assert.Equal(t, data2[1], "")

	l.Walk(nil)
	iter := l.GetIterator()
	assert.Equal(t, int64(12), iter.First().Score)
	assert.Equal(t, "121212", iter.First().Value.(string))
	assert.Equal(t, int64(1), iter.Last().Score)
	assert.Equal(t, "111", iter.Last().Value.(string))
	l.Print()
}

func TestWalkS(t *testing.T) {
	l := NewSkipList(nil)
	l.Insert(s1)
	l.Insert(s2)
	l.Insert(s3)
	l.Insert(s4)
	l.Insert(s5)
	l.Insert(s6)
	l.Insert(s7)
	l.Insert(s8)
	l.Insert(s9)
	l.Insert(s10)
	l.Insert(s11)
	l.Insert(s12)
	var score [100]int64
	var data [100]string
	i := 0
	l.WalkS(func(value interface{}) bool {
		score[i] = value.(*SkipValue).Score
		data[i] = value.(*SkipValue).Value.(string)
		i++
		return true
	})
	assert.Equal(t, data[0], "121212")
	assert.Equal(t, data[1], "111111")
	assert.Equal(t, int64(12), score[0])
	assert.Equal(t, int64(11), score[1])

	var score2 [2]int64
	var data2 [2]string
	i = 0
	l.WalkS(func(value interface{}) bool {
		score2[i] = value.(*SkipValue).Score
		data2[i] = value.(*SkipValue).Value.(string)
		i++
		return false
	})
	assert.Equal(t, "121212", data2[0])
	assert.Equal(t, "", data2[1])
	assert.Equal(t, int64(12), score2[0])
	assert.Equal(t, int64(0), score2[1])
	l.WalkS(func(value interface{}) bool {
		e := value.(*SkipValue)
		if e != nil {
			fmt.Print(e.Score)
			fmt.Print("    ")
			fmt.Print(e.Value)
			fmt.Println("")
			return true
		}
		return false
	})
}
