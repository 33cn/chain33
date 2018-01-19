// +build go1.9

package internal

import (
	"context"
	"fmt"
	"runtime/pprof"
	"strconv"
	"strings"
	"testing"
	"time"
)

func cpuWork1() {
	for i := 0; i < 5000000; i++ {
		str := "str" + strconv.Itoa(i)
		str = str + "a"
	}
}

func cpuWork2() {
	for i := 0; i < 5000000; i++ {
		str := "str" + strconv.Itoa(i)
		str = str + "a"
	}
}

func TestCPUProfileLabels(t *testing.T) {
	agent := NewAgent()
	agent.Debug = true
	agent.ProfileAgent = true

	done := make(chan bool)

	go func() {
		labelSet := pprof.Labels("workload", "label1")
		pprof.Do(context.Background(), labelSet, func(ctx context.Context) {
			cpuWork1()

			done <- true
		})

		cpuWork2()

		done <- true
	}()

	cpuProfiler := newCPUProfiler(agent)
	cpuProfiler.reset()
	cpuProfiler.startProfiler()
	time.Sleep(500 * time.Millisecond)
	cpuProfiler.stopProfiler()
	workload := map[string]int64{"label1": 2}
	data, _ := cpuProfiler.buildProfile(500*1e6, workload)

	if len(data) == 1 {
		fmt.Println("label profile is missing")
	}

	profile := data[1].profile

	if false {
		fmt.Printf("CALL GRAPH: %v\n", profile.printLevel(0))
	}

	if !strings.Contains(fmt.Sprintf("%v", profile.toMap()), "cpuWork1") {
		t.Error("The test function is not found in the profile")
	}

	if strings.Contains(fmt.Sprintf("%v", profile.toMap()), "cpuWork2") {
		t.Error("The test function is found in the profile")
	}

	<-done
}
