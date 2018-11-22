// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import (
	"sync/atomic"
	"time"
)

var deltaTime int64

//NtpHosts ntp hosts
var NtpHosts = []string{
	"time.windows.com:123",
	"ntp.ubuntu.com:123",
	"pool.ntp.org:123",
	"cn.pool.ntp.org:123",
	"time.apple.com:123",
}

//SetTimeDelta realtime - localtime
//超过60s 不做修正
//为了系统的安全，我们只做小范围时间错误的修复
func SetTimeDelta(dt int64) {
	if dt > 300*int64(time.Second) || dt < -300*int64(time.Second) {
		dt = 0
	}
	atomic.StoreInt64(&deltaTime, dt)
}

//Now 获取当前时间戳
func Now() time.Time {
	dt := time.Duration(atomic.LoadInt64(&deltaTime))
	return time.Now().Add(dt)
}

//Since Since时间
func Since(t time.Time) time.Duration {
	return Now().Sub(t)
}
