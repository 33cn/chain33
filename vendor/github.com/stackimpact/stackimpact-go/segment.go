package stackimpact

import (
	"time"
)

type Segment struct {
	agent     *Agent
	Name      string
	startTime time.Time
	Duration  float64
}

func newSegment(agent *Agent, name string) *Segment {
	s := &Segment{
		agent:    agent,
		Name:     name,
		Duration: 0,
	}

	return s
}

func (s *Segment) start() {
	s.startTime = time.Now()
}

// Stops the measurement of a code segment execution time.
func (s *Segment) Stop() {
	s.Duration = float64(time.Since(s.startTime).Nanoseconds()) / 1e6

	s.agent.internalAgent.RecordSegment(s.Name, s.Duration)
}
