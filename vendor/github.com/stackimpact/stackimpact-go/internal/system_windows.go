// +build windows

package internal

import (
	"errors"
	"syscall"
)

var currentProcess syscall.Handle = 0

func readCPUTime() (int64, error) {
	if currentProcess == 0 {
		h, err := syscall.GetCurrentProcess()
		if err != nil {
			return 0, err
		}
		currentProcess = h
	}

	var rusage syscall.Rusage
	err := syscall.GetProcessTimes(
		currentProcess,
		&rusage.CreationTime,
		&rusage.ExitTime,
		&rusage.KernelTime,
		&rusage.UserTime)
	if err != nil {
		return 0, err
	}

	return rusage.KernelTime.Nanoseconds() + rusage.UserTime.Nanoseconds(), nil
}

func readMaxRSS() (int64, error) {
	return 0, errors.New("readMaxRSS is not supported on Windows")
}

func readCurrentRSS() (int64, error) {
	return 0, errors.New("readCurrentRSS is not supported on Windows")
}

func readVMSize() (int64, error) {
	return 0, errors.New("readVMSize is not supported on Windows")
}
