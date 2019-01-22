// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package common

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var NtpHosts = []string{
	"time.windows.com:123",
	"ntp.ubuntu.com:123",
	"pool.ntp.org:123",
	"cn.pool.ntp.org:123",
	"time.apple.com:123",
}

// 在单元测试中对网络时间和本地时间进行单元测试的功能是没有意义的
// 只有在实际运行过程中，通过网络时间来纠正本地时间
// 本测试用例有概率导致误差大于1秒从而让测试用例运行失败
func TestGetRealTime(t *testing.T) {
	hosts := NtpHosts
	nettime := GetRealTimeRetry(hosts, 10)
	now := time.Now()
	//get nettime error, ignore
	if nettime.IsZero() {
		return
	}
	nettime2 := GetRealTimeRetry(hosts, 10)
	//get nettime error, ignore
	delt := time.Since(now)
	if nettime2.IsZero() {
		return
	}
	fmt.Println(nettime, nettime2)
	assert.Equal(t, nettime2.Sub(nettime)/time.Second, delt/time.Second)
}
