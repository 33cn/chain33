package internal

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestCreateCPUProfile(t *testing.T) {
	agent := NewAgent()
	agent.Debug = true
	agent.ProfileAgent = true

	done := make(chan bool)

	go func() {
		// cpu
		//start := time.Now().UnixNano()
		for i := 0; i < 5000000; i++ {
			str := "str" + strconv.Itoa(i)
			str = str + "a"
		}
		//took := time.Now().UnixNano() - start
		//fmt.Printf("TOOK: %v\n", took)

		done <- true
	}()

	cpuProfiler := newCPUProfiler(agent)
	cpuProfiler.reset()
	cpuProfiler.startProfiler()
	time.Sleep(500 * time.Millisecond)
	cpuProfiler.stopProfiler()
	data, _ := cpuProfiler.buildProfile(500*1e6, nil)
	profile := data[0].profile

	if false {
		fmt.Printf("CPU USAGE: %v\n", profile.measurement)
		fmt.Printf("CALL GRAPH: %v\n", profile.printLevel(0))
	}
	if profile.measurement < 2 {
		t.Errorf("CPU usage is too low: %v", profile.measurement)
	}
	if profile.numSamples < 1 {
		t.Error("Number of samples should be > 0")
	}
	if !strings.Contains(fmt.Sprintf("%v", profile.toMap()), "TestCreateCPUProfile") {
		t.Error("The test function is not found in the profile")
	}

	<-done
}
