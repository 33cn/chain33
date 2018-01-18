package sys

import (
	"crypto/rand"
	"encoding/hex"
	"io/ioutil"
	"os"
	"runtime"
	"runtime/debug"
)

func BitcoinHome() (res string) {
	res = os.Getenv("APPDATA")
	if res != "" {
		res += "\\Bitcoin\\"
		return
	}
	res = os.Getenv("HOME")
	if res != "" {
		res += "/.bitcoin/"
	}
	return
}

func is_hex_string(s []byte) string {
	var res string
	for i := range s {
		c := byte(s[i])
		if c <= '9' && c >= '0' || c <= 'f' && c >= 'a' || c <= 'F' && c >= 'A' {
			res += string(c)
		} else if c != ' ' && c != '\n' && c != '\r' && c != '\t' {
			return ""
		}
	}
	return res
}

// reads tx from the file or (if there is no such a file) decodes the text
func GetRawData(fn string) (dat []byte) {
	d, er := ioutil.ReadFile(fn)
	if er == nil {
		hexdump := is_hex_string(d)
		if len(hexdump) >= 2 || (len(hexdump)&1) == 1 {
			dat, _ = hex.DecodeString(hexdump)
		} else {
			dat = d
		}
	} else {
		dat, _ = hex.DecodeString(fn)
	}
	return
}

func ClearBuffer(buf []byte) {
	rand.Read(buf[:])
}

var secrespass func([]byte) int

func getline(buf []byte) (n int) {
	n, er := os.Stdin.Read(buf[:])
	if er != nil {
		ClearBuffer(buf)
		return -1
	}
	for n > 0 && buf[n-1] < ' ' {
		n--
		buf[n] = 0
	}
	return n
}

// Reads a password from console
// Returns -1 on error
func ReadPassword(buf []byte) (n int) {
	if secrespass != nil {
		return secrespass(buf)
	}
	return getline(buf)
}

// Returns Alloc and Sys (how much memory is used)
func MemUsed() (uint64, uint64) {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	return ms.Alloc, ms.Sys
}

// Run GC and free as much mem as possible
func FreeMem() {
	runtime.GC()
	debug.FreeOSMemory()
}
