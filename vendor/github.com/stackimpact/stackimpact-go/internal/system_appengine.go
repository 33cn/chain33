// +build appengine

package internal

import (
	"errors"
)

func readCPUTime() (int64, error) {
	return 0, errors.New("readCPUTime is not supported on App Engine")
}

func readMaxRSS() (int64, error) {
	return 0, errors.New("readMaxRSS is not supported on App Engine")
}

func readCurrentRSS() (int64, error) {
	return 0, errors.New("readCurrentRSS is not supported App Engine")
}

func readVMSize() (int64, error) {
	return 0, errors.New("readVMSize is not supported on App Engine")
}
