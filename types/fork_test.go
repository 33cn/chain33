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
}

func TestParaFork(t *testing.T) {
	NewChain33Config(ReadFile("testdata/guodun.toml"))
	NewChain33Config(ReadFile("testdata/guodun2.toml"))
}
