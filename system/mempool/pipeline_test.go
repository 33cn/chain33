// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mempool

import (
	"runtime"
	"testing"
	"time"

	"github.com/33cn/chain33/queue"
	"github.com/stretchr/testify/assert"
)

func TestStep(t *testing.T) {
	done := make(chan struct{})
	in := make(chan *queue.Message)
	msg := &queue.Message{ID: 0}
	cb := func(in *queue.Message) *queue.Message {
		in.ID++
		time.Sleep(time.Microsecond)
		return in
	}
	out := step(done, in, cb)
	in <- msg
	msg2 := <-out
	assert.Equal(t, msg2.ID, int64(1))
	close(done)
}

func TestMutiStep(t *testing.T) {
	done := make(chan struct{})
	in := make(chan *queue.Message)
	msg := &queue.Message{ID: 0}
	step1 := func(in *queue.Message) *queue.Message {
		in.ID++
		time.Sleep(time.Microsecond)
		return in
	}
	out1 := step(done, in, step1)
	step2 := func(in *queue.Message) *queue.Message {
		in.ID++
		time.Sleep(time.Microsecond)
		return in
	}
	out21 := step(done, out1, step2)
	out22 := step(done, out1, step2)

	out3 := mergeList(done, out21, out22)
	in <- msg
	msg2 := <-out3
	assert.Equal(t, msg2.ID, int64(2))
	close(done)
}

func BenchmarkStep(b *testing.B) {
	done := make(chan struct{})
	in := make(chan *queue.Message)
	msg := &queue.Message{ID: 0}
	cb := func(in *queue.Message) *queue.Message {
		in.ID++
		time.Sleep(100 * time.Microsecond)
		return in
	}
	out := step(done, in, cb)
	go func() {
		for i := 0; i < b.N; i++ {
			in <- msg
		}
	}()
	for i := 0; i < b.N; i++ {
		msg2 := <-out
		assert.Equal(b, msg2.ID, int64(1))
	}
	close(done)
}

func BenchmarkStepMerge(b *testing.B) {
	done := make(chan struct{})
	in := make(chan *queue.Message)
	msg := &queue.Message{ID: 0}
	cb := func(in *queue.Message) *queue.Message {
		in.ID++
		time.Sleep(100 * time.Microsecond)
		return in
	}
	chs := make([]<-chan *queue.Message, runtime.NumCPU())
	for i := 0; i < runtime.NumCPU(); i++ {
		chs[i] = step(done, in, cb)
	}
	out := merge(done, chs)
	go func() {
		for i := 0; i < b.N; i++ {
			in <- msg
		}
	}()
	for i := 0; i < b.N; i++ {
		msg2 := <-out
		assert.Equal(b, msg2.ID, int64(1))
	}
	close(done)
}

func mergeList(done <-chan struct{}, cs ...<-chan *queue.Message) <-chan *queue.Message {
	return merge(done, cs)
}
