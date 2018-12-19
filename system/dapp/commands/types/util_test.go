// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCheckExpireOpt(t *testing.T) {
	expire := "0s"
	str, err := CheckExpireOpt(expire)
	assert.NoError(t, err)
	assert.Equal(t, "0s", str)

	expire = "14s"
	str, err = CheckExpireOpt(expire)
	assert.NoError(t, err)
	assert.Equal(t, "120s", str)

	expire = "14"
	str, err = CheckExpireOpt(expire)
	assert.NoError(t, err)
	assert.Equal(t, "14", str)

	expire = "H:123"
	str, err = CheckExpireOpt(expire)
	assert.NoError(t, err)
	assert.Equal(t, "H:123", str)

}
