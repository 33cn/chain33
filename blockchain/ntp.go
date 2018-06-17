package blockchain

import (
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"gitlab.33.cn/chain33/chain33/common"
)

const (
	ntpPool   = "time.windows.com:123" // ntpPool is the NTP server to query for the current time
	ntpChecks = 6                      //6 times all error
	//ntpFailureThreshold = 32               // Continuous timeouts after which to check NTP
	//ntpWarningCooldown  = 10 * time.Minute // Minimum amount of time to pass before repeating NTP warning
	driftThreshold = 10 * time.Second // Allowed clock drift before warning user

)

var (
	ntpLog = chainlog.New("submodule", "ntp")
	failed int32
)

// checkClockDrift queries an NTP server for clock drifts and warns the user if
// one large enough is detected.
func checkClockDrift() {
	realnow, err := common.GetNtpTime(ntpPool)
	if err != nil {
		ntpLog.Info("checkClockDrift", "sntpDrift err", err)
		return
	}
	drift := time.Since(realnow)
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
