package internal

import (
	"fmt"
	"runtime"
	"strings"
	"testing"
)

var objs []string

func TestCreateAllocationCallGraph(t *testing.T) {
	agent := NewAgent()
	agent.Debug = true
	agent.ProfileAgent = true

	objs = make([]string, 0)
	for i := 0; i < 100000; i++ {
		objs = append(objs, string(i))
	}

	runtime.GC()
	runtime.GC()

	allocationProfiler := newAllocationProfiler(agent)

	p, _ := allocationProfiler.readHeapProfile()

	// size
	callGraph, err := allocationProfiler.createAllocationCallGraph(p)
	if err != nil {
		t.Error(err)
		return
	}
	callGraph.propagate()
	//fmt.Printf("ALLOCATED SIZE: %f\n", callGraph.measurement)
	//fmt.Printf("CALL GRAPH: %v\n", callGraph.printLevel(0))
	if callGraph.measurement < 100000 {
		t.Error("Allocated size is too low")
	}
	if callGraph.numSamples < 1 {
		t.Error("Number of samples should be > 0")
	}

	if !strings.Contains(fmt.Sprintf("%v", callGraph.toMap()), "TestCreateAllocationCallGraph") {
		t.Error("The test function is not found in the profile")
	}

	objs = nil
}
