package internal

import (
	"testing"
	"time"
)

type TestProfiler struct {
	profile    *BreakdownNode
	startCount int
	started    bool
	closed     bool
}

func (tp *TestProfiler) reset() {
	tp.profile = newBreakdownNode("root")
}

func (tp *TestProfiler) startProfiler() error {
	tp.started = true
	tp.startCount++
	return nil
}

func (tp *TestProfiler) stopProfiler() error {
	tp.started = false
	return nil
}

func (tp *TestProfiler) buildProfile(duration int64, workloads map[string]int64) ([]*ProfileData, error) {
	tp.closed = true

	data := []*ProfileData{
		&ProfileData{
			category:     "test-category",
			name:         "Test",
			unit:         "percent",
			unitInterval: 0,
			profile:      tp.profile,
		},
	}

	return data, nil
}

func TestMaxDuration(t *testing.T) {
	agent := NewAgent()
	agent.Debug = true

	prof := &TestProfiler{}

	conf := &ProfilerConfig{
		logPrefix:          "Test profiler",
		maxProfileDuration: 20,
		maxSpanDuration:    2,
		maxSpanCount:       30,
		spanInterval:       8,
		reportInterval:     120,
	}

	rep := newProfileReporter(agent, prof, conf)
	rep.start()

	rep.startProfiling(true, "")
	rep.stopProfiling()

	if prof.startCount != 1 {
		t.Errorf("Start count is not 1")
	}

	if prof.started {
		t.Error("Not stopped")
	}

	rep.profileDuration = 21 * 1e9

	rep.startProfiling(true, "")
	rep.stopProfiling()

	if prof.startCount > 1 {
		t.Error("Should not be started")
	}
}

func TestMaxSpanCount(t *testing.T) {
	agent := NewAgent()
	agent.Debug = true

	prof := &TestProfiler{}

	conf := &ProfilerConfig{
		logPrefix:          "Test profiler",
		maxProfileDuration: 20,
		maxSpanDuration:    2,
		maxSpanCount:       1,
		spanInterval:       8,
		reportInterval:     120,
	}

	rep := newProfileReporter(agent, prof, conf)
	rep.start()

	rep.startProfiling(true, "")
	rep.stopProfiling()

	if prof.startCount != 1 {
		t.Errorf("Start count is not 1")
	}

	rep.startProfiling(true, "")
	rep.stopProfiling()

	if prof.startCount > 1 {
		t.Error("Should not be started")
	}
}

func TestReportProfile(t *testing.T) {
	agent := NewAgent()
	agent.Debug = true

	prof := &TestProfiler{}

	conf := &ProfilerConfig{
		logPrefix:          "Test profiler",
		maxProfileDuration: 20,
		maxSpanDuration:    2,
		maxSpanCount:       30,
		spanInterval:       8,
		reportInterval:     120,
	}

	rep := newProfileReporter(agent, prof, conf)
	rep.start()

	rep.startProfiling(true, "")
	time.Sleep(10 * time.Millisecond)
	rep.stopProfiling()

	rep.report()

	if !prof.closed {
		t.Error("Should be closed")
	}
}
