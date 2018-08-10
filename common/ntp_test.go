package common

import (
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

func TestGetRealTime(t *testing.T) {
	hosts := NtpHosts
	nettime := GetRealTimeRetry(hosts, 10)
	local := time.Now()
	t.Log(nettime.Sub(local))
	assert.Equal(t, int(local.Sub(nettime)/time.Second), 0)
}
