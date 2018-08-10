package logc

import (
	"fmt"
	"testing"
)

func TestLogMonitor(t *testing.T) {
	m := NewLogMonitor("ExecBlock", false)
	ch, err := m.GetCurrentKeyInfo()
	if err != nil {
		fmt.Println("Get current info failed.")
		panic(err)
	}

	for line := range <-ch {
		fmt.Println(line)
	}
}
