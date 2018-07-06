package gocbcore

import (
	"sync"
	"sync/atomic"
	"unsafe"
)

type memdOpMapItem struct {
	value *memdQRequest
	next  *memdOpMapItem
}

// This is used to store operations while they are pending
//  a response from the server to allow mapping of a response
//  opaque back to the originating request.  This queue takes
//  advantage of the monotonic nature of the opaque values
//  and synchronous responses from the server to nearly always
//  return the request without needing to iterate at all.
type memdOpMap struct {
	lock    sync.Mutex
	opIndex uint32

	first *memdOpMapItem
	last  *memdOpMapItem
}

// Add a new request to the bottom of the op queue.
func (q *memdOpMap) Add(req *memdQRequest) bool {
	q.lock.Lock()

	if !atomic.CompareAndSwapPointer(&req.waitingIn, nil, unsafe.Pointer(q)) {
		logDebugf("Attempted to put dispatched op in new opmap")
		q.lock.Unlock()
		return false
	}

	if req.isCancelled() {
		atomic.CompareAndSwapPointer(&req.waitingIn, unsafe.Pointer(q), nil)
		q.lock.Unlock()
		return false
	}

	q.opIndex++
	req.Opaque = q.opIndex

	item := &memdOpMapItem{
		value: req,
		next:  nil,
	}

	if q.last == nil {
		q.first = item
		q.last = item
	} else {
		q.last.next = item
		q.last = item
	}

	q.lock.Unlock()
	return true
}

// Removes a request from the op queue.  Expects to be passed
//  the request to remove, along with the request that
//  immediately precedes it in the queue.
func (q *memdOpMap) remove(prev *memdOpMapItem, req *memdOpMapItem) {
	// TODO(brett19): Maybe should ensure this was meant to be in this opmap.
	atomic.CompareAndSwapPointer(&req.value.waitingIn, unsafe.Pointer(q), nil)

	if prev == nil {
		q.first = req.next
		if q.first == nil {
			q.last = nil
		}
		return
	}
	prev.next = req.next
	if prev.next == nil {
		q.last = prev
	}
}

// Removes a specific request from the op queue.
func (q *memdOpMap) Remove(req *memdQRequest) bool {
	q.lock.Lock()

	cur := q.first
	var prev *memdOpMapItem
	for cur != nil {
		if cur.value == req {
			q.remove(prev, cur)
			q.lock.Unlock()
			return true
		}
		prev = cur
		cur = cur.next
	}

	q.lock.Unlock()
	return false
}

// Locates a request (searching FIFO-style) in the op queue using
// the opaque value that was assigned to it when it was dispatched.
// It then removes the request from the queue if it is not persistent
// or if alwaysRemove is set to true.
func (q *memdOpMap) FindAndMaybeRemove(opaque uint32, force bool) *memdQRequest {
	q.lock.Lock()

	cur := q.first
	var prev *memdOpMapItem
	for cur != nil {
		if cur.value.Opaque == opaque {
			if !cur.value.Persistent || force {
				q.remove(prev, cur)
			}

			q.lock.Unlock()
			return cur.value
		}
		prev = cur
		cur = cur.next
	}

	q.lock.Unlock()
	return nil
}

// Clears the queue of all requests and calls the passed function
// once for each request found in the queue.
func (q *memdOpMap) Drain(cb func(*memdQRequest)) {
	for cur := q.first; cur != nil; cur = cur.next {
		// TODO(brett19): Maybe should ensure this was meant to be in this opmap.
		atomic.CompareAndSwapPointer(&cur.value.waitingIn, unsafe.Pointer(q), nil)
		cb(cur.value)
	}
	q.first = nil
	q.last = nil
}
