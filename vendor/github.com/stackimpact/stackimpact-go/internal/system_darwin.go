// +build darwin

package internal

import (
	"errors"
	"runtime"
	"syscall"
)

func readCPUTime() (int64, error) {
	rusage := new(syscall.Rusage)
	if err := syscall.Getrusage(0, rusage); err != nil {
		return 0, err
	}

	var cpuTimeNanos int64
	cpuTimeNanos =
		int64(rusage.Utime.Sec*1e9) +
			int64(rusage.Utime.Usec) +
			int64(rusage.Stime.Sec*1e9) +
			int64(rusage.Stime.Usec)

	return cpuTimeNanos, nil
}

func readMaxRSS() (int64, error) {
	rusage := new(syscall.Rusage)
	if err := syscall.Getrusage(0, rusage); err != nil {
		return 0, err
	}

	var maxRSS int64
	maxRSS = int64(rusage.Maxrss)

	if runtime.GOOS == "darwin" {
		maxRSS = maxRSS / 1000
	}

	return maxRSS, nil
}

func readCurrentRSS() (int64, error) {
	return 0, errors.New("readCurrentRSS is not supported on OS X")
}

func readVMSize() (int64, error) {
	return 0, errors.New("readVMSize is not supported on OS X")
}
