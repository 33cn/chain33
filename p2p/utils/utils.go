// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package utils p2p utils
package utils

const (
	versionMask = 0xFFFF
)

// CalcChannelVersion  calc channel version
func CalcChannelVersion(channel, version int32) int32 {
	return channel<<16 + version
}

// DecodeChannelVersion decode channel version
func DecodeChannelVersion(channelVersion int32) (channel int32, version int32) {
	channel = channelVersion >> 16
	version = channelVersion & versionMask
	return
}
