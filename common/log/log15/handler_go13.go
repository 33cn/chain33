// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build !go1.4

package log15

import (
	"sync/atomic"
	"unsafe"
)

// swapHandler wraps another handler that may be swapped out
// dynamically at runtime in a thread-safe fashion.
type swapHandler struct {
	handler unsafe.Pointer
}

func (h *swapHandler) Log(r *Record) error {
	return h.Get().Log(r)
}

func (h *swapHandler) MaxLevel() int {
	return h.Get().MaxLevel()
}

func (h *swapHandler) SetMaxLevel(maxLevel int) {
	h.Get().SetMaxLevel(maxLevel)
}

func (h *swapHandler) Get() Handler {
	return *(*Handler)(atomic.LoadPointer(&h.handler))
}

func (h *swapHandler) Swap(newHandler Handler) {
	atomic.StorePointer(&h.handler, unsafe.Pointer(&newHandler))
}
