package internal

import (
	"testing"
	"time"
)

func TestStart(t *testing.T) {
	agent := NewAgent()
	agent.DashboardAddress = "http://localhost:5000"
	agent.AgentKey = "key"
	agent.AppName = "GoTestApp"
	agent.Debug = true
	agent.Start()

	if agent.AgentKey == "" {
		t.Error("AgentKey not set")
	}

	if agent.AppName == "" {
		t.Error("AppName not set")
	}

	if agent.HostName == "" {
		t.Error("HostName not set")
	}
}

func TestCalculateProgramSHA1(t *testing.T) {
	agent := NewAgent()
	agent.Debug = true
	hash := agent.calculateProgramSHA1()

	if hash == "" {
		t.Error("failed calculating program SHA1")
	}
}

func TestStartStopProfiling(t *testing.T) {
	agent := NewAgent()
	agent.Debug = true
	agent.AutoProfiling = false

	agent.cpuReporter.start()
	agent.StartProfiling("")

	time.Sleep(50 * time.Millisecond)

	agent.StopProfiling()

	if agent.cpuReporter.profileDuration == 0 {
		t.Error("profileDuration should be > 0")
	}
}

func TestTimerPeriod(t *testing.T) {
	agent := NewAgent()
	agent.Debug = true

	fired := 0
	timer := agent.createTimer(0, 10*time.Millisecond, func() {
		fired++
	})

	time.Sleep(20 * time.Millisecond)

	timer.Stop()

	time.Sleep(30 * time.Millisecond)

	if fired > 2 {
		t.Errorf("interval fired too many times: %v", fired)
	}
}

func TestTimerDelay(t *testing.T) {
	agent := NewAgent()
	agent.Debug = true

	fired := 0
	timer := agent.createTimer(10*time.Millisecond, 0, func() {
		fired++
	})

	time.Sleep(20 * time.Millisecond)

	timer.Stop()

	if fired != 1 {
		t.Errorf("delay should fire once: %v", fired)
	}
}

func TestTimerDelayStop(t *testing.T) {
	agent := NewAgent()
	agent.Debug = true

	fired := 0
	timer := agent.createTimer(10*time.Millisecond, 0, func() {
		fired++
	})

	timer.Stop()

	time.Sleep(20 * time.Millisecond)

	if fired == 1 {
		t.Errorf("delay should not fire")
	}
}
