package blockchain

import (
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/types"
)

const (
	ntpChecks      = 6
	driftThreshold = 10 * time.Second // Allowed clock drift before warning user
)

var (
	ntpLog = chainlog.New("submodule", "ntp")
	failed int32
)

// checkClockDrift queries an NTP server for clock drifts and warns the user if
// one large enough is detected.
func checkClockDrift() {
	realnow := common.GetRealTimeRetry(types.NtpHosts, 10)
	if realnow.IsZero() {
		ntpLog.Info("checkClockDrift", "sntpDrift err", "get ntptime error")
		return
	}
	now := types.Now()
	drift := now.Sub(realnow)
	if drift < -driftThreshold || drift > driftThreshold {
		warning := fmt.Sprintf("System clock seems off by %v, which can prevent network connectivity", drift)
		howtofix := fmt.Sprintf("Please enable network time synchronisation in system settings")
		separator := strings.Repeat("-", len(warning))
		ntpLog.Warn(fmt.Sprint(separator))
		ntpLog.Warn(fmt.Sprint(warning))
		ntpLog.Warn(fmt.Sprint(howtofix))
		ntpLog.Warn(fmt.Sprint(separator))
		atomic.AddInt32(&failed, 1)
		if atomic.LoadInt32(&failed) == ntpChecks {
			ntpLog.Error("System clock seems ERROR")
			UpdateNtpClockSyncStatus(false)
		}
	} else {
		atomic.StoreInt32(&failed, 0)
		UpdateNtpClockSyncStatus(true)
		ntpLog.Info(fmt.Sprintf("Sanity NTP check reported %v drift, all ok", drift))
	}
}
