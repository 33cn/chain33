package internal

import (
	"testing"
	"time"
)

func TestRecordSegment(t *testing.T) {
	agent := NewAgent()
	agent.Debug = true

	agent.segmentReporter.reset()
	agent.segmentReporter.started.Set()

	for i := 0; i < 100; i++ {
		go func() {
			defer agent.segmentReporter.recordSegment("seg1", 10)

			time.Sleep(10 * time.Millisecond)
		}()
	}

	time.Sleep(150 * time.Millisecond)

	segmentNodes := agent.segmentReporter.segmentNodes
	agent.segmentReporter.report()

	seg1Counter := segmentNodes["seg1"]
	if seg1Counter.name != "seg1" || seg1Counter.measurement < 10 {
		t.Errorf("Measurement of seg1 is too low: %v", seg1Counter.measurement)
	}
}
