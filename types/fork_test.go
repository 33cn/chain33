// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestForks(t *testing.T) {
	cfg := NewChain33Config(GetDefaultCfgstring())
	cfg.forks.setLocalFork()
	assert.Equal(t, cfg.forks.IsFork(1, "ForkV1"), false)
	assert.Equal(t, cfg.forks.IsFork(1, "ForkV12"), false)
	assert.Equal(t, cfg.forks.IsFork(0, "ForkBlockHash"), false)
	assert.Equal(t, cfg.forks.IsFork(1, "ForkBlockHash"), true)
	assert.Equal(t, cfg.forks.IsFork(1, "ForkTransferExec"), true)
	assert.Equal(t, cfg.forks.IsFork(0, "ForkBlockHash"), false)
	assert.Equal(t, cfg.forks.IsFork(1, "ForkBlockHash"), true)
}

func TestForksSetAndGet(t *testing.T) {
	f := &Forks{}
	f.SetFork("ForkA", 100)
	assert.Equal(t, int64(100), f.GetFork("ForkA"))
	assert.True(t, f.HasFork("ForkA"))
	assert.False(t, f.HasFork("ForkB"))
	assert.Equal(t, int64(MaxHeight), f.GetFork("NonExist"))
}

func TestForksReplaceFork(t *testing.T) {
	f := &Forks{}
	f.SetFork("ForkA", 100)
	f.ReplaceFork("ForkA", 200)
	assert.Equal(t, int64(200), f.GetFork("ForkA"))

	assert.Panics(t, func() {
		f.ReplaceFork("NonExist", 100)
	})
}

func TestForksSetDappFork(t *testing.T) {
	f := &Forks{}
	f.SetDappFork("coins", "ForkTransfer", 100)
	assert.Equal(t, int64(100), f.GetDappFork("coins", "ForkTransfer"))
	assert.True(t, f.HasFork("coins.ForkTransfer"))
}

func TestForksReplaceDappFork(t *testing.T) {
	f := &Forks{}
	f.SetDappFork("coins", "ForkTransfer", 100)
	f.ReplaceDappFork("coins", "ForkTransfer", 200)
	assert.Equal(t, int64(200), f.GetDappFork("coins", "ForkTransfer"))

	assert.Panics(t, func() {
		f.ReplaceDappFork("coins", "NonExist", 100)
	})
}

func TestForksSetAllFork(t *testing.T) {
	f := &Forks{}
	f.SetFork("ForkA", 100)
	f.SetFork("ForkB", 200)
	f.SetAllFork(0)
	assert.Equal(t, int64(0), f.GetFork("ForkA"))
	assert.Equal(t, int64(0), f.GetFork("ForkB"))
}

func TestForksGetAll(t *testing.T) {
	f := &Forks{}
	f.SetFork("ForkA", 100)
	f.SetFork("ForkB", 200)
	all := f.GetAll()
	assert.Equal(t, 2, len(all))
	assert.Equal(t, int64(100), all["ForkA"])
	assert.Equal(t, int64(200), all["ForkB"])
}

func TestForksIsFork(t *testing.T) {
	f := &Forks{}
	f.SetFork("ForkA", 100)
	assert.True(t, f.IsFork(100, "ForkA"))
	assert.True(t, f.IsFork(200, "ForkA"))
	assert.True(t, f.IsFork(-1, "ForkA"))
	assert.False(t, f.IsFork(50, "ForkA"))
	assert.False(t, f.IsFork(100, "NonExist"))
}

func TestForksIsDappFork(t *testing.T) {
	f := &Forks{}
	f.SetDappFork("coins", "ForkTransfer", 100)
	assert.True(t, f.IsDappFork(100, "coins", "ForkTransfer"))
	assert.False(t, f.IsDappFork(50, "coins", "ForkTransfer"))
}

func TestForksCheckKeyPanic(t *testing.T) {
	f := &Forks{}
	assert.Panics(t, func() {
		f.SetFork("key.with.dot", 100)
	})
}

func TestChain33ConfigForkMethodsDefaultCfg(t *testing.T) {
	cfg := NewChain33Config(GetDefaultCfgstring())
	cfg.forks.setLocalFork()
	assert.True(t, cfg.HasFork("ForkBlockHash"))
	assert.True(t, cfg.HasFork("ForkTransferExec"))
	assert.True(t, cfg.IsFork(1, "ForkBlockHash"))
	assert.False(t, cfg.IsFork(0, "ForkBlockHash"))
}

func TestDappFork(t *testing.T) {
	f := &Forks{}
	f.SetDappFork("testdapp", "TestFork", 99)
	assert.Equal(t, int64(99), f.GetDappFork("testdapp", "TestFork"))
	f.ReplaceDappFork("testdapp", "TestFork", 150)
	assert.Equal(t, int64(150), f.GetDappFork("testdapp", "TestFork"))
}

func TestIsEnableFork(t *testing.T) {
	cfg := NewChain33Config(GetDefaultCfgstring())
	cfg.forks.setLocalFork()
	assert.False(t, cfg.IsEnableFork(1, "ForkBlockHash", false))
	assert.True(t, cfg.IsEnableFork(1, "ForkBlockHash", true))
	assert.False(t, cfg.IsEnableFork(0, "ForkBlockHash", true))
}

func TestRegisterSystemFork(t *testing.T) {
	f := &Forks{}
	f.RegisterSystemFork()
	assert.True(t, f.HasFork("ForkBlockHash"))
	assert.True(t, f.HasFork("ForkTransferExec"))
	assert.Equal(t, int64(209186), f.GetFork("ForkBlockHash"))
}

func TestCheckKey(t *testing.T) {
	assert.Panics(t, func() {
		checkKey("invalid.key")
	})
	assert.NotPanics(t, func() {
		checkKey("validKey")
	})
}
