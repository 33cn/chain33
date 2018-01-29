package common

import (
	"fmt"
	"net"
	"sync"
	"time"
)

var (
	bw_mutex sync.Mutex

	dl_last_sec     int64
	dl_bytes_so_far int

	DlBytesPrevSec    [0x100]uint64
	DlBytesPrevSecIdx int

	dl_bytes_priod uint64
	DlBytesTotal   uint64

	UploadLimit   uint
	DownloadLimit uint

	ul_last_sec     int64
	ul_bytes_so_far int

	UlBytesPrevSec    [0x100]uint64
	UlBytesPrevSecIdx int
	ul_bytes_priod    uint64
	UlBytesTotal      uint64
)

func TickRecv() {
	now := time.Now().Unix()
	if now != dl_last_sec {
		if now-dl_last_sec == 1 {
			DlBytesPrevSec[DlBytesPrevSecIdx] = dl_bytes_priod
		} else {
			DlBytesPrevSec[DlBytesPrevSecIdx] = 0
		}
		DlBytesPrevSecIdx = (DlBytesPrevSecIdx + 1) & 0xff
		dl_bytes_priod = 0
		dl_bytes_so_far = 0
		dl_last_sec = now
	}
}

func TickSent() {
	now := time.Now().Unix()
	if now != ul_last_sec {
		if now-ul_last_sec == 1 {
			UlBytesPrevSec[UlBytesPrevSecIdx] = ul_bytes_priod
		} else {
			UlBytesPrevSec[UlBytesPrevSecIdx] = 0
		}
		UlBytesPrevSecIdx = (UlBytesPrevSecIdx + 1) & 0xff
		ul_bytes_priod = 0
		ul_bytes_so_far = 0
		ul_last_sec = now
	}
}

func SockRead(con net.Conn, buf []byte) (n int, e error) {
	var toread int
	bw_mutex.Lock()
	TickRecv()
	if DownloadLimit == 0 {
		toread = len(buf)
	} else {
		toread = int(DownloadLimit) - dl_bytes_so_far
		if toread > len(buf) {
			toread = len(buf)
		} else if toread < 0 {
			toread = 0
		}
	}
	dl_bytes_so_far += toread
	bw_mutex.Unlock()

	if toread > 0 {
		// Wait 1 millisecond for a data, timeout if nothing there
		con.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
		n, e = con.Read(buf[:toread])
		bw_mutex.Lock()
		DlBytesTotal += uint64(n)
		dl_bytes_priod += uint64(n)
		bw_mutex.Unlock()
	} else {
		// supsend a task for awhile, to prevent stucking in a busy loop
		time.Sleep(100 * time.Millisecond)
	}
	return
}

// Send all the bytes, but respect the upload limit (force delays)
func SockWrite(con net.Conn, buf []byte) (n int, e error) {
	var tosend int
	bw_mutex.Lock()
	TickSent()
	if UploadLimit == 0 {
		tosend = len(buf)
	} else {
		tosend = int(UploadLimit) - ul_bytes_so_far
		if tosend > len(buf) {
			tosend = len(buf)
		} else if tosend < 0 {
			tosend = 0
		}
	}
	ul_bytes_so_far += tosend
	bw_mutex.Unlock()
	if tosend > 0 {
		// Set timeout to prevent thread from getting stuck if the other end does not read
		con.SetWriteDeadline(time.Now().Add(250 * time.Millisecond))
		n, e = con.Write(buf[:tosend])
		bw_mutex.Lock()
		UlBytesTotal += uint64(n)
		ul_bytes_priod += uint64(n)
		bw_mutex.Unlock()
		if e != nil {
			if nerr, ok := e.(net.Error); ok && nerr.Timeout() {
				e = nil
			}
		}
	} else {
		time.Sleep(250 * time.Millisecond)
	}
	return
}

func LockBw() {
	bw_mutex.Lock()
}

func UnlockBw() {
	bw_mutex.Unlock()
}

func GetAvgBW(arr []uint64, idx int, cnt int) uint64 {
	var sum uint64
	if cnt <= 0 {
		return 0
	}
	for i := 0; i < cnt; i++ {
		if idx == 0 {
			idx = len(arr) - 1
		} else {
			idx--
		}
		sum += arr[idx]
	}
	return sum / uint64(cnt)
}

func PrintStats() {
	bw_mutex.Lock()
	TickRecv()
	TickSent()
	fmt.Printf("Downloading at %d/%d KB/s, %s total",
		GetAvgBW(DlBytesPrevSec[:], DlBytesPrevSecIdx, 5)>>10, DownloadLimit>>10, BytesToString(DlBytesTotal))
	fmt.Printf("  |  Uploading at %d/%d KB/s, %s total\n",
		GetAvgBW(UlBytesPrevSec[:], UlBytesPrevSecIdx, 5)>>10, UploadLimit>>10, BytesToString(UlBytesTotal))
	bw_mutex.Unlock()
	return
}
