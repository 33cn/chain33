// +build !linux,!darwin,!windows

package internal

import (
	"errors"
)

func readCPUTime() (int64, error) {
	return 0, errors.New("readCPUTime is not supported.")
}

func readMaxRSS() (int64, error) {
	return 0, errors.New("readMaxRSS is not supported.")
}

func readCurrentRSS() (int64, error) {
	return 0, errors.New("readCurrentRSS is not supported.")
}

func readVMSize() (int64, error) {
	return 0, errors.New("readVMSize is not supported.")
}
