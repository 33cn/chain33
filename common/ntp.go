package common

import (
	"encoding/binary"
	"errors"
	"math"
	"net"
	"time"
)

const ntpEpochOffset = 2208988800

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
//利用服务器返回的 t2, t3, 和本地的 t1, t4 校准时间
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
		fmt.Println("========")
		fmt.Println("t1", t1)
		fmt.Println("t2", t2, t2.Sub(t1))
		fmt.Println("t3", t3, t3.Sub(t2))
		fmt.Println("t4", t4, t4.Sub(t3))
		fmt.Println("========")
	*/
	//t2 - t1 -> deltaNet + deltaTime
	//t3 - t4 -> -deltaNet + deltaTime
	//如果deltaNet相同
	//判断t2 - t1 和  t3 - t4 绝对值的 倍数，如果超过2倍，认为无效(请求和回复延时严重不对称)
	d1 := t2.Sub(t1)
	d2 := t3.Sub(t4)
	rate := math.Abs(float64(d1)) / math.Abs(float64(d2))
	if rate >= 2 || rate <= 0.5 {
		return time.Time{}, ErrNetWorkDealy
	}
	delt := d1 + d2
	return t4.Add(delt / 2), nil
}

func intToTime(sec, frac uint32) time.Time {
	secs := int64(sec) - int64(ntpEpochOffset)
	nanos := (int64(frac) * 1e9) >> 32
	return time.Unix(secs, nanos)
}
