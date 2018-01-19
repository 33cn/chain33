package internal

import (
	"fmt"
	"math"
	"testing"
)

func TestReport(t *testing.T) {
	agent := NewAgent()
	agent.Debug = true

	agent.processReporter.reset()
	agent.processReporter.started.Set()

	agent.processReporter.report()
	agent.processReporter.report()
	agent.processReporter.report()

	metrics := agent.processReporter.metrics

	isValid(t, metrics, TypeCounter, CategoryCPU, NameCPUTime, 0, math.Inf(0))
	isValid(t, metrics, TypeState, CategoryCPU, NameCPUUsage, 0, math.Inf(0))
	isValid(t, metrics, TypeState, CategoryMemory, NameMaxRSS, 0, math.Inf(0))
	isValid(t, metrics, TypeState, CategoryMemory, NameAllocated, 0, math.Inf(0))
	isValid(t, metrics, TypeCounter, CategoryMemory, NameLookups, 0, math.Inf(0))
	isValid(t, metrics, TypeCounter, CategoryMemory, NameMallocs, 0, math.Inf(0))
	isValid(t, metrics, TypeCounter, CategoryMemory, NameFrees, 0, math.Inf(0))
	isValid(t, metrics, TypeState, CategoryMemory, NameHeapIdle, 0, math.Inf(0))
	isValid(t, metrics, TypeState, CategoryMemory, NameHeapInuse, 0, math.Inf(0))
	isValid(t, metrics, TypeState, CategoryMemory, NameHeapObjects, 0, math.Inf(0))
	isValid(t, metrics, TypeCounter, CategoryGC, NameGCTotalPause, 0, math.Inf(0))
	isValid(t, metrics, TypeCounter, CategoryGC, NameNumGC, 0, math.Inf(0))
	isValid(t, metrics, TypeState, CategoryGC, NameGCCPUFraction, 0, 1)
	isValid(t, metrics, TypeState, CategoryRuntime, NameNumGoroutines, 0, math.Inf(0))
	isValid(t, metrics, TypeCounter, CategoryRuntime, NameNumCgoCalls, 0, math.Inf(0))
}

func isValid(t *testing.T, metrics map[string]*Metric, typ string, category string, name string, minValue float64, maxValue float64) {
	if metric, exists := metrics[typ+category+name]; exists {
		if metric.hasMeasurement() {
			valid := metric.measurement.value >= minValue && metric.measurement.value <= maxValue

			if !valid {
				t.Error(fmt.Sprintf("%v - %v: %v", category, name, metric.measurement.value))
			}

			return
		}
	}

	t.Errorf("%v - %v", category, name)
}
