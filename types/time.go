package types

import (
	"sync/atomic"
	"time"
)

var deltaTime int64
var timeCalibration bool
var NtpHosts = []string{
	"time.windows.com:123",
	"ntp.ubuntu.com:123",
	"pool.ntp.org:123",
	"cn.pool.ntp.org:123",
	"time.asia.apple.com:123",
}

func SetFixTime(openTimeCalibration bool) {
	timeCalibration = openTimeCalibration
}

func IsFixTime() bool {
	return timeCalibration
}

//realtime - localtime
//超过60s 不做修正
//为了系统的安全，我们只做小范围时间错误的修复
func SetTimeDelta(dt int64) {
	if dt > 60*int64(time.Second) || dt < -60*int64(time.Second) {
		dt = 0
	}
	atomic.StoreInt64(&deltaTime, dt)
}

func Now() time.Time {
	dt := time.Duration(atomic.LoadInt64(&deltaTime))
	return time.Now().Add(dt)
}
