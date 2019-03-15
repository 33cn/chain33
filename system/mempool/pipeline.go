// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mempool

import (
	"sync"

	"github.com/33cn/chain33/queue"
)

//pipeline 适用于 一个问题，分成很多步完成，每步的输出作为下一步的输入

func step(done <-chan struct{}, in <-chan *queue.Message, cb func(*queue.Message) *queue.Message) <-chan *queue.Message {
	out := make(chan *queue.Message)
	go func() {
		defer close(out)
		for n := range in {
			select {
			case out <- cb(n):
			case <-done:
				return
			}
		}
	}()
	return out
}

func merge(done <-chan struct{}, cs []<-chan *queue.Message) <-chan *queue.Message {
	var wg sync.WaitGroup
	out := make(chan *queue.Message)
	output := func(c <-chan *queue.Message) {
		defer wg.Done()
		for n := range c {
			select {
			case out <- n:
			case <-done:
				return
			}
		}
	}
	wg.Add(len(cs))
	for _, c := range cs {
		go output(c)
	}
	go func() {
		wg.Wait()
		close(out)
	}()
	return out
}
