// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package common

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"net"
	"sort"
	"time"

	log "github.com/33cn/chain33/common/log/log15"
)

const ntpEpochOffset = 2208988800

// 如果获取的ntp时间和自己时间相差太大，超过阈值，保守起见，不采用
const safeDeltaScope = 300 * 1000 * int64(time.Millisecond)

//ErrNetWorkDealy error
var ErrNetWorkDealy = errors.New("ErrNetWorkDealy")

// NTP packet format (v3 with optional v4 fields removed)
//
// 0                   1                   2                   3
// 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |LI | VN  |Mode |    Stratum     |     Poll      |  Precision   |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |                         Root Delay                            |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |                         Root Dispersion                       |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |                          Reference ID                         |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |                                                               |
// +                     Reference Timestamp (64)                  +
// |                                                               |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |                                                               |
// +                      Origin Timestamp (64)                    +
// |                                                               |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |                                                               |
// +                      Receive Timestamp (64)                   +
// |                                                               |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |                                                               |
// +                      Transmit Timestamp (64)                  +
// |                                                               |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//
type packet struct {
	Settings       uint8  // leap yr indicator, ver number, and mode
	Stratum        uint8  // stratum of local clock
	Poll           int8   // poll exponent
	Precision      int8   // precision exponent
	RootDelay      uint32 // root delay
	RootDispersion uint32 // root dispersion
	ReferenceID    uint32 // reference id
	RefTimeSec     uint32 // reference timestamp sec
	RefTimeFrac    uint32 // reference timestamp fractional
	OrigTimeSec    uint32 // origin time secs
	OrigTimeFrac   uint32 // origin time fractional
	RxTimeSec      uint32 // receive time secs
	RxTimeFrac     uint32 // receive time frac
	TxTimeSec      uint32 // transmit time secs
	TxTimeFrac     uint32 // transmit time frac
}

/*
       t2      t3                 t6          t7
   ---------------------------------------------------------
            /\         \                 /\            \
            /           \                /              \
           /             \              /                \
          /               \/           /                 \/
   ---------------------------------------------------------
        t1                t4         t5                  t8
*/

//GetNtpTime 利用服务器返回的 t2, t3, 和本地的 t1, t4 校准时间
//delt = ((t2-t1)+(t3-t4))/2
//current = t4 + delt
func GetNtpTime(host string) (time.Time, error) {

	// Setup a UDP connection
	conn, err := net.Dial("udp", host)
	if err != nil {
		return time.Time{}, err
	}
	defer conn.Close()
	if err := conn.SetDeadline(time.Now().Add(3 * time.Second)); err != nil {
		return time.Time{}, err
	}
	// configure request settings by specifying the first byte as
	// 00 011 011 (or 0x1B)
	// |  |   +-- client mode (3)
	// |  + ----- version (3)
	// + -------- leap year indicator, 0 no warning
	req := &packet{Settings: 0x1B}
	t1 := time.Now()
	// send time request
	if err := binary.Write(conn, binary.BigEndian, req); err != nil {
		return time.Time{}, err
	}

	// block to receive server response
	rsp := &packet{}
	if err := binary.Read(conn, binary.BigEndian, rsp); err != nil {
		return time.Time{}, err
	}
	t2 := intToTime(rsp.RxTimeSec, rsp.RxTimeFrac)
	t3 := intToTime(rsp.TxTimeSec, rsp.TxTimeFrac)
	t4 := time.Now()
	// On POSIX-compliant OS, time is expressed
	// using the Unix time epoch (or secs since year 1970).
	// NTP seconds are counted since 1900 and therefore must
	// be corrected with an epoch offset to convert NTP seconds
	// to Unix time by removing 70 yrs of seconds (1970-1900)
	// or 2208988800 seconds.
	/*
		js, _ := json.MarshalIndent(rsp, "", "\t")
		fmt.Println(string(js))
	*/
	//t2 - t1 -> deltaNet + deltaTime = d1
	//t3 - t4 -> -deltaNet + deltaTime = d2
	//如果deltaNet相同
	//判断t2 - t1 和  t3 - t4 绝对值的 倍数，如果超过2倍，认为无效(请求和回复延时严重不对称)
	d1 := t2.Sub(t1)
	d2 := t3.Sub(t4)
	delt := (d1 + d2) / 2
	if delt > time.Duration(safeDeltaScope) || delt < time.Duration(-safeDeltaScope) {
		log.Error("GetNtpTime", "host", host, "delt", delt, "RxSec", rsp.RxTimeSec, "RxNs", rsp.RxTimeFrac, "TxSec", rsp.TxTimeSec, "TxNs", rsp.TxTimeFrac)
		log.Error("GetNtpTime", "delt", delt, "t1", t1, "t2", t2, "t3", t3, "now", t4, "d1", d1, "d2", d2)
		return time.Time{}, errors.New("WrongNtpDelteTime")
	}
	return t4.Add(delt), nil
}

//n timeserver
//n/2+1 is ok and the same
type durationSlice []time.Duration

func (s durationSlice) Len() int           { return len(s) }
func (s durationSlice) Less(i, j int) bool { return abs(s[i]) < abs(s[j]) }
func (s durationSlice) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

//GetRealTime 获取实际时间
func GetRealTime(hosts []string) time.Time {
	q := len(hosts)/2 + 1
	//q := 5
	ch := make(chan time.Duration, len(hosts))
	for i := 0; i < len(hosts); i++ {
		go func(host string) {
			ntptime, err := getTimeRetry(host, 1)
			if ntptime.IsZero() || err != nil {
				println("getTimeRetry", err.Error())
				ch <- time.Duration(math.MaxInt64)
			} else {
				dt := time.Until(ntptime)
				fmt.Println(host, dt)
				ch <- dt
			}
		}(hosts[i])
	}
	dtlist := make([]time.Duration, 0)
	for i := 0; i < len(hosts); i++ {
		t := <-ch
		if t == time.Duration(math.MaxInt64) {
			continue
		}
		dtlist = append(dtlist, t)
		if len(dtlist) >= q {
			calclist := make([]time.Duration, len(dtlist))
			copy(calclist, dtlist)
			sort.Sort(durationSlice(calclist))
			calclist = maxSubList(calclist, time.Millisecond*100)
			if len(calclist) < q {
				continue
			}
			drift := time.Duration(0)
			for i := 0; i < len(calclist); i++ {
				drift += calclist[i]
			}
			return time.Now().Add(drift / time.Duration(len(calclist)))
		}
	}
	return time.Time{}
}

func abs(t time.Duration) time.Duration {
	if t < 0 {
		return -t
	}
	return t
}

func maxSubList(list []time.Duration, dt time.Duration) (sub []time.Duration) {
	if len(list) == 0 {
		return list
	}
	var start int
	var next int
	for i := 0; i < len(list); i++ {
		var nextTime time.Duration
		next = i + 1
		if next == len(list) {
			nextTime = math.MaxInt64
		} else {
			nextTime = list[next]
		}
		if abs(nextTime-list[i]) > dt {
			if len(sub) < (next-start) && (next-start) > 1 {
				sub = list[start:next]
			}
			start = next
		}
	}
	return sub
}

//GetRealTimeRetry 重试获取实际时间
func GetRealTimeRetry(hosts []string, retry int) time.Time {
	for i := 0; i < retry; i++ {
		t := GetRealTime(hosts)
		if !t.IsZero() {
			return t
		}
	}
	return time.Time{}
}

func getTimeRetry(host string, retry int) (time.Time, error) {
	var lasterr error
	for i := 0; i < retry; i++ {
		t, err := GetNtpTime(host)
		if err != nil {
			lasterr = err
			if i < retry-1 {
				//have a rest
				time.Sleep(time.Millisecond * 10)
				continue
			}
			return time.Time{}, err
		}
		return t, nil
	}
	return time.Time{}, lasterr
}

func intToTime(sec, frac uint32) time.Time {
	secs := int64(sec) - int64(ntpEpochOffset)
	nanos := (int64(frac) * 1e9) >> 32
	return time.Unix(secs, nanos)
}
