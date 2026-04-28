// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mempool

import (
	"testing"

	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
	"github.com/stretchr/testify/assert"
)

func TestMempoolReg(t *testing.T) {
	assert.Panics(t, func() {
		Reg("test_nil_driver", nil)
	})

	testCreate := func(cfg *types.Mempool, sub []byte) queue.Module {
		return nil
	}
	Reg("test_driver_1", testCreate)

	assert.Panics(t, func() {
		Reg("test_driver_1", testCreate)
	})
}

func TestMempoolLoad(t *testing.T) {
	testCreate := func(cfg *types.Mempool, sub []byte) queue.Module {
		return nil
	}
	Reg("test_load_driver", testCreate)

	create, err := Load("test_load_driver")
	assert.Nil(t, err)
	assert.NotNil(t, create)

	_, err = Load("nonexistent_driver")
	assert.Equal(t, types.ErrNotFound, err)
}
