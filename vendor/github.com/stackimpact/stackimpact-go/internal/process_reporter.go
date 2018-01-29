package internal

import (
	"runtime"
	"time"
)

type ProcessReporter struct {
	ReportInterval int64

	agent       *Agent
	started     *Flag
	reportTimer *Timer

	metrics map[string]*Metric
}

func newProcessReporter(agent *Agent) *ProcessReporter {
	pr := &ProcessReporter{
		ReportInterval: 60,

		agent:       agent,
		started:     &Flag{},
		reportTimer: nil,

		metrics: nil,
	}

	return pr
}

func (pr *ProcessReporter) reset() {
	pr.metrics = make(map[string]*Metric)
}

func (pr *ProcessReporter) start() {
	if !pr.agent.AutoProfiling {
		return
	}

	if !pr.started.SetIfUnset() {
		return
	}

	pr.reset()

	pr.reportTimer = pr.agent.createTimer(0, time.Duration(pr.ReportInterval)*time.Second, func() {
		pr.report()
	})
}

func (pr *ProcessReporter) stop() {
	if !pr.started.UnsetIfSet() {
		return
	}

	if pr.reportTimer != nil {
		pr.reportTimer.Stop()
	}
}

func (pr *ProcessReporter) reportMetric(typ string, category string, name string, unit string, value float64) *Metric {
	key := typ + category + name
	var metric *Metric
	if existingMetric, exists := pr.metrics[key]; !exists {
		metric = newMetric(pr.agent, typ, category, name, unit)
		pr.metrics[key] = metric
	} else {
		metric = existingMetric
	}

	metric.createMeasurement(TriggerTimer, value, 0, nil)

	if metric.hasMeasurement() {
		pr.agent.messageQueue.addMessage("metric", metric.toMap())
	}

	return metric
}

func (pr *ProcessReporter) report() {
	if !pr.started.IsSet() {
		return
	}

	cpuTime, err := readCPUTime()
	if err == nil {
		cpuTimeMetric := pr.reportMetric(TypeCounter, CategoryCPU, NameCPUTime, UnitNanosecond, float64(cpuTime))
		if cpuTimeMetric.hasMeasurement() {
			cpuUsage := (float64(cpuTimeMetric.measurement.value) / float64(60*1e9)) * 100
			cpuUsage = cpuUsage / float64(runtime.NumCPU())
			pr.reportMetric(TypeState, CategoryCPU, NameCPUUsage, UnitPercent, float64(cpuUsage))
		}
	} else {
		pr.agent.error(err)
	}

	maxRSS, err := readMaxRSS()
	if err == nil {
		pr.reportMetric(TypeState, CategoryMemory, NameMaxRSS, UnitKilobyte, float64(maxRSS))
	} else {
		pr.agent.error(err)
	}

	currentRSS, err := readCurrentRSS()
	if err == nil {
		pr.reportMetric(TypeState, CategoryMemory, NameCurrentRSS, UnitKilobyte, float64(currentRSS))
	} else {
		pr.agent.error(err)
	}

	vmSize, err := readVMSize()
	if err == nil {
		pr.reportMetric(TypeState, CategoryMemory, NameVMSize, UnitKilobyte, float64(vmSize))
	} else {
		pr.agent.error(err)
	}

	memStats := &runtime.MemStats{}
	runtime.ReadMemStats(memStats)
	pr.reportMetric(TypeState, CategoryMemory, NameAllocated, UnitByte, float64(memStats.Alloc))
	pr.reportMetric(TypeCounter, CategoryMemory, NameLookups, UnitNone, float64(memStats.Lookups))
	pr.reportMetric(TypeCounter, CategoryMemory, NameMallocs, UnitNone, float64(memStats.Mallocs))
	pr.reportMetric(TypeCounter, CategoryMemory, NameFrees, UnitNone, float64(memStats.Frees))
	pr.reportMetric(TypeState, CategoryMemory, NameHeapIdle, UnitByte, float64(memStats.HeapIdle))
	pr.reportMetric(TypeState, CategoryMemory, NameHeapInuse, UnitByte, float64(memStats.HeapInuse))
	pr.reportMetric(TypeState, CategoryMemory, NameHeapObjects, UnitNone, float64(memStats.HeapObjects))
	pr.reportMetric(TypeCounter, CategoryGC, NameGCTotalPause, UnitNanosecond, float64(memStats.PauseTotalNs))
	pr.reportMetric(TypeCounter, CategoryGC, NameNumGC, UnitNone, float64(memStats.NumGC))
	pr.reportMetric(TypeState, CategoryGC, NameGCCPUFraction, UnitNone, float64(memStats.GCCPUFraction))

	numGoroutine := runtime.NumGoroutine()
	pr.reportMetric(TypeState, CategoryRuntime, NameNumGoroutines, UnitNone, float64(numGoroutine))

	numCgoCall := runtime.NumCgoCall()
	pr.reportMetric(TypeCounter, CategoryRuntime, NameNumCgoCalls, UnitNone, float64(numCgoCall))
}
