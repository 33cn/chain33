package internal

import (
	"math/rand"
	"sync"
	"time"
)

type ProfilerConfig struct {
	logPrefix          string
	reportOnly         bool
	maxProfileDuration int64
	maxSpanDuration    int64
	maxSpanCount       int32
	spanInterval       int64
	reportInterval     int64
}

type Profiler interface {
	reset()
	startProfiler() error
	stopProfiler() error
	buildProfile(duration int64, workloads map[string]int64) ([]*ProfileData, error)
}

type ProfileData struct {
	category     string
	name         string
	unit         string
	unitInterval int64
	profile      *BreakdownNode
}

type ProfileReporter struct {
	agent                 *Agent
	started               *Flag
	profiler              Profiler
	config                *ProfilerConfig
	spanTimer             *Timer
	reportTimer           *Timer
	profileLock           *sync.Mutex
	profileStartTimestamp int64
	profileDuration       int64
	spanCount             int32
	spanActive            *Flag
	spanStart             int64
	spanTimeout           *Timer
	workloads             map[string]int64
}

func newProfileReporter(agent *Agent, profiler Profiler, config *ProfilerConfig) *ProfileReporter {
	pr := &ProfileReporter{
		agent:                 agent,
		started:               &Flag{},
		profiler:              profiler,
		config:                config,
		spanTimer:             nil,
		reportTimer:           nil,
		profileLock:           &sync.Mutex{},
		profileStartTimestamp: 0,
		profileDuration:       0,
		spanCount:             0,
		spanActive:            &Flag{},
		spanStart:             0,
		spanTimeout:           nil,
		workloads:             nil,
	}

	return pr
}

func (pr *ProfileReporter) start() {
	if !pr.started.SetIfUnset() {
		return
	}

	pr.profileLock.Lock()
	defer pr.profileLock.Unlock()

	pr.reset()

	if pr.agent.AutoProfiling {
		if !pr.config.reportOnly {
			pr.spanTimer = pr.agent.createTimer(0, time.Duration(pr.config.spanInterval)*time.Second, func() {
				time.Sleep(time.Duration(rand.Int63n(pr.config.spanInterval-pr.config.maxSpanDuration)) * time.Second)
				pr.startProfiling(false, "")
			})
		}

		pr.reportTimer = pr.agent.createTimer(0, time.Duration(pr.config.reportInterval)*time.Second, func() {
			pr.report()
		})
	}
}

func (pr *ProfileReporter) stop() {
	if !pr.started.UnsetIfSet() {
		return
	}

	if pr.spanTimer != nil {
		pr.spanTimer.Stop()
	}

	if pr.reportTimer != nil {
		pr.reportTimer.Stop()
	}
}

func (pr *ProfileReporter) reset() {
	pr.profiler.reset()
	pr.profileStartTimestamp = time.Now().Unix()
	pr.profileDuration = 0
	pr.spanCount = 0
	pr.workloads = make(map[string]int64)
}

func (pr *ProfileReporter) startProfiling(rateLimit bool, workload string) bool {
	if !pr.started.IsSet() {
		return false
	}

	pr.profileLock.Lock()
	defer pr.profileLock.Unlock()

	if pr.profileDuration > pr.config.maxProfileDuration*1e9 {
		pr.agent.log("%v: max profiling duration reached.", pr.config.logPrefix)
		return false
	}

	if rateLimit && pr.spanCount >= pr.config.maxSpanCount {
		pr.agent.log("%v: max profiling span count reached.", pr.config.logPrefix)
		return false
	}

	if !pr.agent.profilerActive.SetIfUnset() {
		pr.agent.log("%v: another profiler currently active.", pr.config.logPrefix)
		return false
	}

	pr.agent.log("%v: starting profiler.", pr.config.logPrefix)

	err := pr.profiler.startProfiler()
	if err != nil {
		pr.agent.profilerActive.Unset()
		pr.agent.error(err)
		return false
	}

	pr.spanTimeout = pr.agent.createTimer(time.Duration(pr.config.maxSpanDuration)*time.Second, 0, func() {
		pr.stopProfiling()
	})

	pr.spanCount++
	pr.spanActive.Set()
	pr.spanStart = time.Now().UnixNano()

	if workload != "" {
		if _, ok := pr.workloads[workload]; ok {
			pr.workloads[workload]++
		} else {
			pr.workloads[workload] = 1
		}
	}

	return true
}

func (pr *ProfileReporter) stopProfiling() {
	pr.profileLock.Lock()
	defer pr.profileLock.Unlock()

	if !pr.spanActive.UnsetIfSet() {
		return
	}
	pr.spanTimeout.Stop()

	defer pr.agent.profilerActive.Unset()

	err := pr.profiler.stopProfiler()
	if err != nil {
		pr.agent.error(err)
		return
	}
	pr.agent.log("%v: profiler stopped.", pr.config.logPrefix)

	pr.profileDuration += time.Now().UnixNano() - pr.spanStart
}

func (pr *ProfileReporter) report() {
	if !pr.started.IsSet() {
		return
	}

	pr.profileLock.Lock()
	defer pr.profileLock.Unlock()

	if !pr.agent.AutoProfiling {
		if pr.profileStartTimestamp > time.Now().Unix()-pr.config.reportInterval {
			return
		} else if pr.profileStartTimestamp < time.Now().Unix()-2*pr.config.reportInterval {
			pr.reset()
			return
		}
	}

	if !pr.config.reportOnly && pr.profileDuration == 0 {
		return
	}

	pr.agent.log("%v: reporting profile.", pr.config.logPrefix)

	profileData, err := pr.profiler.buildProfile(pr.profileDuration, pr.workloads)
	if err != nil {
		pr.agent.error(err)
		return
	}

	for _, d := range profileData {
		metric := newMetric(pr.agent, TypeProfile, d.category, d.name, d.unit)
		metric.createMeasurement(TriggerTimer, d.profile.measurement, d.unitInterval, d.profile)
		pr.agent.messageQueue.addMessage("metric", metric.toMap())
	}

	pr.reset()
}
