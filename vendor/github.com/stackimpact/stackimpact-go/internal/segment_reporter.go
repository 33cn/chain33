package internal

import (
	"sync"
	"time"
)

type SegmentReporter struct {
	ReportInterval int64

	agent       *Agent
	started     *Flag
	reportTimer *Timer

	segmentNodes map[string]*BreakdownNode
	recordLock   *sync.RWMutex
}

func newSegmentReporter(agent *Agent) *SegmentReporter {
	sr := &SegmentReporter{
		ReportInterval: 60,

		agent:       agent,
		started:     &Flag{},
		reportTimer: nil,

		segmentNodes: nil,
		recordLock:   &sync.RWMutex{},
	}

	return sr
}

func (sr *SegmentReporter) reset() {
	sr.recordLock.Lock()
	defer sr.recordLock.Unlock()

	sr.segmentNodes = make(map[string]*BreakdownNode)
}

func (sr *SegmentReporter) start() {
	if !sr.agent.AutoProfiling {
		return
	}

	if !sr.started.SetIfUnset() {
		return
	}

	sr.reset()

	sr.reportTimer = sr.agent.createTimer(0, time.Duration(sr.ReportInterval)*time.Second, func() {
		sr.report()
	})
}

func (sr *SegmentReporter) stop() {
	if !sr.started.UnsetIfSet() {
		return
	}

	if sr.reportTimer != nil {
		sr.reportTimer.Stop()
	}
}

func (sr *SegmentReporter) recordSegment(name string, duration float64) {
	if !sr.started.IsSet() {
		return
	}

	if name == "" {
		sr.agent.log("Empty segment name")
		return
	}

	// Segment exists for the current interval.
	sr.recordLock.RLock()
	node, nExists := sr.segmentNodes[name]
	if nExists {
		node.updateP95(duration)
	}
	sr.recordLock.RUnlock()

	// Segment does not exist yet for the current interval.
	if !nExists {
		sr.recordLock.Lock()
		node, nExists := sr.segmentNodes[name]
		if !nExists {
			// If segment was not created by other recordSegment call between locks, create it.
			node = newBreakdownNode(name)
			sr.segmentNodes[name] = node
		}
		sr.recordLock.Unlock()

		sr.recordLock.RLock()
		node.updateP95(duration)
		sr.recordLock.RUnlock()
	}
}

func (sr *SegmentReporter) report() {
	if !sr.started.IsSet() {
		return
	}

	sr.recordLock.Lock()
	outgoing := sr.segmentNodes
	sr.segmentNodes = make(map[string]*BreakdownNode)
	sr.recordLock.Unlock()

	for _, segmentNode := range outgoing {
		segmentRoot := newBreakdownNode("root")
		segmentRoot.addChild(segmentNode)
		segmentRoot.evaluateP95()
		segmentRoot.propagate()

		metric := newMetric(sr.agent, TypeTrace, CategorySegmentTrace, segmentNode.name, UnitMillisecond)
		metric.createMeasurement(TriggerTimer, segmentRoot.measurement, 0, segmentRoot)
		sr.agent.messageQueue.addMessage("metric", metric.toMap())
	}
}
