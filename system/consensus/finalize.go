// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package consensus

import (
	"github.com/33cn/chain33/queue"
)

// Finalizer block finalize
type Finalizer interface {
	Initialize(ctx *Context) error
	Start() error
	ProcessMsg(msg *queue.Message) (processed bool)
}

var finalizers = make(map[string]Finalizer)

// RegFinalizer register committer
func RegFinalizer(name string, f Finalizer) {

	if f == nil {
		panic("RegCommitter: committer is nil")
	}
	if _, dup := committers[name]; dup {
		panic("RegCommitter: duplicate committer " + name)
	}
	finalizers[name] = f
}

// LoadFinalizer load
func LoadFinalizer(name string) Finalizer {

	return finalizers[name]
}
