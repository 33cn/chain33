// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package skiplist

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSkipListLevel(t *testing.T) {
	sl := NewSkipList(&SkipValue{})
	assert.Equal(t, 1, sl.Level())
}

func TestSkipListFindCount(t *testing.T) {
	sl := NewSkipList(&SkipValue{})
	assert.Equal(t, 0, sl.FindCount())
	sl.Insert(&SkipValue{})
	findCount := sl.FindCount()
	_ = sl.Find(&SkipValue{})
	assert.GreaterOrEqual(t, sl.FindCount(), findCount)
}

func TestIteratorPrev(t *testing.T) {
	sl := NewSkipList(&SkipValue{})
	sl.Insert(&SkipValue{})
	sl.Insert(&SkipValue{})
	it := sl.GetIterator()
	it.Last()
	prevIt := it.Prev()
	assert.NotNil(t, prevIt)
	// Calling Prev again on first element returns same iterator
	prevIt.Prev()
}

func TestIteratorNext(t *testing.T) {
	sl := NewSkipList(&SkipValue{})
	sl.Insert(&SkipValue{})
	sl.Insert(&SkipValue{})
	it := sl.GetIterator()
	it.First()
	nextIt := it.Next()
	assert.NotNil(t, nextIt)
}

func TestIteratorValue(t *testing.T) {
	sl := NewSkipList(&SkipValue{})
	v := &SkipValue{}
	sl.Insert(v)
	it := sl.GetIterator()
	it.First()
	assert.Equal(t, v, it.Value())
}

func TestSkipListNodePrev(t *testing.T) {
	var nilNode *skipListNode
	assert.Nil(t, nilNode.Prev())

	node := &skipListNode{}
	assert.Nil(t, node.Prev())
}

func TestSkipListNodeNext(t *testing.T) {
	var nilNode *skipListNode
	assert.Nil(t, nilNode.Next())

	node := &skipListNode{next: make([]*skipListNode, 1)}
	assert.Nil(t, node.Next())

	nextNode := &skipListNode{}
	node.next[0] = nextNode
	assert.Equal(t, nextNode, node.Next())
}

func BenchmarkFind(b *testing.B) {
	sl := NewSkipList(&SkipValue{})
	for i := 0; i < b.N; i++ {
		val := &SkipValue{}
		sl.Insert(val)
	}
	fmt.Println(sl.Len())
}
