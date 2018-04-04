package internal

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestRecordError(t *testing.T) {
	agent := NewAgent()
	agent.Debug = true

	agent.errorReporter.reset()
	agent.errorReporter.started.Set()

	for i := 0; i < 100; i++ {
		go func() {
			agent.errorReporter.recordError("group1", errors.New("error1"), 0)

			go func() {
				agent.errorReporter.recordError("group1", errors.New("error2"), 0)
			}()
		}()
	}

	time.Sleep(50 * time.Millisecond)

	errorGraphs := agent.errorReporter.errorGraphs

	group1 := errorGraphs["group1"]
	group1.evaluateCounter()
	if group1.measurement != 200 {
		t.Errorf("Measurement is wrong: %v", group1.measurement)
	}

	if !strings.Contains(fmt.Sprintf("%v", group1.toMap()), "TestRecordError.func1.1") {
		t.Error("The test function is not found in the error profile")
	}
}
