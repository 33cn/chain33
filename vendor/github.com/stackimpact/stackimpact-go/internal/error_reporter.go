package internal

import (
	"sync"
	"time"
)

type ErrorReporter struct {
	ReportInterval int64

	agent       *Agent
	started     *Flag
	reportTimer *Timer

	recordLock  *sync.RWMutex
	errorGraphs map[string]*BreakdownNode
}

func newErrorReporter(agent *Agent) *ErrorReporter {
	er := &ErrorReporter{
		ReportInterval: 60,

		agent:       agent,
		started:     &Flag{},
		reportTimer: nil,

		recordLock: &sync.RWMutex{},
	}

	return er
}

func (er *ErrorReporter) reset() {
	er.recordLock.Lock()
	defer er.recordLock.Unlock()

	er.errorGraphs = make(map[string]*BreakdownNode)
}

func (er *ErrorReporter) start() {
	if !er.agent.AutoProfiling {
		return
	}

	if !er.started.SetIfUnset() {
		return
	}

	er.reset()

	er.reportTimer = er.agent.createTimer(0, time.Duration(er.ReportInterval)*time.Second, func() {
		er.report()
	})
}

func (er *ErrorReporter) stop() {
	if !er.started.UnsetIfSet() {
		return
	}

	if er.reportTimer != nil {
		er.reportTimer.Stop()
	}
}

func (er *ErrorReporter) incrementError(group string, errorGraph *BreakdownNode, err error, frames []string) {
	currentNode := errorGraph
	currentNode.updateCounter(1, 0)
	for i := len(frames) - 1; i >= 0; i-- {
		f := frames[i]
		currentNode = currentNode.findOrAddChild(f)
		currentNode.updateCounter(1, 0)
	}

	message := err.Error()
	if message == "" {
		message = "Undefined"
	}
	messageNode := currentNode.findChild(message)
	if messageNode == nil {
		if len(currentNode.children) < 5 {
			messageNode = currentNode.findOrAddChild(message)
		} else {
			messageNode = currentNode.findOrAddChild("Other")
		}
	}
	messageNode.updateCounter(1, 0)
}

func (er *ErrorReporter) recordError(group string, err error, skip int) {
	if !er.started.IsSet() {
		return
	}

	frames := callerFrames(skip+1, 25)

	if err == nil {
		er.agent.log("Missing error object")
		return
	}

	// Error graph exists for the current interval.
	er.recordLock.RLock()
	errorGraph, exists := er.errorGraphs[group]
	if exists {
		er.incrementError(group, errorGraph, err, frames)
	}
	er.recordLock.RUnlock()

	// Error graph does not exist yet for the current interval.
	if !exists {
		er.recordLock.Lock()
		errorGraph, exists := er.errorGraphs[group]
		if !exists {
			// If segment was not created by other recordError call between locks, create it.
			errorGraph = newBreakdownNode(group)
			er.errorGraphs[group] = errorGraph
		}
		er.recordLock.Unlock()

		er.recordLock.RLock()
		er.incrementError(group, errorGraph, err, frames)
		er.recordLock.RUnlock()
	}
}

func (er *ErrorReporter) report() {
	if !er.started.IsSet() {
		return
	}

	er.recordLock.Lock()
	outgoing := er.errorGraphs
	er.errorGraphs = make(map[string]*BreakdownNode)
	er.recordLock.Unlock()

	for _, errorGraph := range outgoing {
		errorGraph.evaluateCounter()

		metric := newMetric(er.agent, TypeState, CategoryErrorProfile, errorGraph.name, UnitNone)
		metric.createMeasurement(TriggerTimer, errorGraph.measurement, 60, errorGraph)
		er.agent.messageQueue.addMessage("metric", metric.toMap())
	}
}
