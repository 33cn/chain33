// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSetTimeDeltaZero(t *testing.T) {
	SetTimeDelta(0)
	dt := deltaTime
	assert.Equal(t, int64(0), dt)
}

func TestSetTimeDeltaWithinRange(t *testing.T) {
	SetTimeDelta(int64(100 * time.Second))
	assert.Equal(t, int64(100*time.Second), deltaTime)
	SetTimeDelta(0)
}

func TestSetTimeDeltaTooLarge(t *testing.T) {
	SetTimeDelta(int64(400 * time.Second))
	assert.Equal(t, int64(0), deltaTime)

	SetTimeDelta(int64(-400 * time.Second))
	assert.Equal(t, int64(0), deltaTime)
}

func TestNow(t *testing.T) {
	SetTimeDelta(0)
	now := Now()
	assert.WithinDuration(t, time.Now(), now, time.Second)
}

func TestNowWithDelta(t *testing.T) {
	delta := int64(100 * time.Second)
	SetTimeDelta(delta)
	now := Now()
	expected := time.Now().Add(time.Duration(delta))
	assert.WithinDuration(t, expected, now, time.Second)
	SetTimeDelta(0)
}

func TestSince(t *testing.T) {
	SetTimeDelta(0)
	past := time.Now().Add(-10 * time.Second)
	d := Since(past)
	assert.True(t, d >= 9*time.Second && d <= 11*time.Second)
}
