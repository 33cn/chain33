// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package queue

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSetTopic(t *testing.T) {
	client := &client{}
	hi := "hello"
	client.setTopic(hi)
	ret := client.getTopic()
	assert.Equal(t, hi, ret)
}
