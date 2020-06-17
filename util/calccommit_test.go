// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetMostCommit(t *testing.T) {
	commits := [][]byte{[]byte("aa"), []byte("bb"), []byte("aa"), []byte("aa")}

	most, key := GetMostCommit(commits)

	assert.Equal(t, 3, most)
	assert.Equal(t, "aa", key)
}

func TestIsCommitDone(t *testing.T) {
	done := IsCommitDone(4, 2)
	assert.Equal(t, false, done)

	done = IsCommitDone(4, 3)
	assert.Equal(t, true, done)

}
