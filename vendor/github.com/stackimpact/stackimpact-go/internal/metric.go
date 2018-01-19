package internal

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

const TypeState string = "state"
const TypeCounter string = "counter"
const TypeProfile string = "profile"
const TypeTrace string = "trace"

const CategoryCPU string = "cpu"
const CategoryMemory string = "memory"
const CategoryGC string = "gc"
const CategoryRuntime string = "runtime"
const CategoryCPUProfile string = "cpu-profile"
const CategoryMemoryProfile string = "memory-profile"
const CategoryBlockProfile string = "block-profile"
const CategoryLockProfile string = "lock-profile"
const CategoryCPUTrace string = "cpu-trace"
const CategoryBlockTrace string = "block-trace"
const CategorySegmentTrace string = "segment-trace"
const CategoryErrorProfile string = "error-profile"

const NameCPUTime string = "CPU time"
const NameCPUUsage string = "CPU usage"
const NameMaxRSS string = "Max RSS"
const NameCurrentRSS string = "Current RSS"
const NameVMSize string = "VM Size"
const NameNumGoroutines string = "Number of goroutines"
const NameNumCgoCalls string = "Number of cgo calls"
const NameAllocated string = "Allocated memory"
const NameLookups string = "Lookups"
const NameMallocs string = "Mallocs"
const NameFrees string = "Frees"
const NameHeapSys string = "Heap obtained"
const NameHeapIdle string = "Heap idle"
const NameHeapInuse string = "Heap non-idle"
const NameHeapReleased string = "Heap released"
const NameHeapObjects string = "Heap objects"
const NameGCTotalPause string = "GC total pause"
const NameNumGC string = "Number of GCs"
const NameGCCPUFraction string = "GC CPU fraction"
const NameHeapAllocation string = "Heap allocation"
const NameBlockingCallTimes string = "Blocking call times"
const NameHTTPTransactionBreakdown string = "HTTP transaction breakdown"

const UnitNone string = ""
const UnitMillisecond string = "millisecond"
const UnitMicrosecond string = "microsecond"
const UnitNanosecond string = "nanosecond"
const UnitByte string = "byte"
const UnitKilobyte string = "kilobyte"
const UnitPercent string = "percent"

const TriggerTimer string = "timer"
const TriggerAnomaly string = "anomaly"

const ReservoirSize int = 1000

type filterFuncType func(name string) bool

type Reservoir []uint64

func (r Reservoir) Len() int {
	return len(r)
}
func (r Reservoir) Swap(i, j int) {
	r[i], r[j] = r[j], r[i]
}
func (r Reservoir) Less(i, j int) bool {
	return r[i] < r[j]
}

type BreakdownNode struct {
	name        string
	measurement float64
	numSamples  int64
	counter     int64
	reservoir   Reservoir
	children    map[string]*BreakdownNode
	updateLock  *sync.RWMutex
}

func newBreakdownNode(name string) *BreakdownNode {
	bn := &BreakdownNode{
		name:        name,
		measurement: 0,
		numSamples:  0,
		counter:     0,
		reservoir:   nil,
		children:    make(map[string]*BreakdownNode),
		updateLock:  &sync.RWMutex{},
	}

	return bn
}

func (bn *BreakdownNode) findChild(name string) *BreakdownNode {
	bn.updateLock.RLock()
	defer bn.updateLock.RUnlock()

	if child, exists := bn.children[name]; exists {
		return child
	}

	return nil
}

func (bn *BreakdownNode) maxChild() *BreakdownNode {
	bn.updateLock.RLock()
	defer bn.updateLock.RUnlock()

	var maxChild *BreakdownNode = nil
	for _, child := range bn.children {
		if maxChild == nil || child.measurement > maxChild.measurement {
			maxChild = child
		}
	}
	return maxChild
}

func (bn *BreakdownNode) minChild() *BreakdownNode {
	bn.updateLock.RLock()
	defer bn.updateLock.RUnlock()

	var minChild *BreakdownNode = nil
	for _, child := range bn.children {
		if minChild == nil || child.measurement < minChild.measurement {
			minChild = child
		}
	}
	return minChild
}

func (bn *BreakdownNode) addChild(child *BreakdownNode) {
	bn.updateLock.Lock()
	defer bn.updateLock.Unlock()

	bn.children[child.name] = child
}

func (bn *BreakdownNode) removeChild(child *BreakdownNode) {
	bn.updateLock.Lock()
	defer bn.updateLock.Unlock()

	delete(bn.children, child.name)
}

func (bn *BreakdownNode) findOrAddChild(name string) *BreakdownNode {
	child := bn.findChild(name)
	if child == nil {
		child = newBreakdownNode(name)
		bn.addChild(child)
	}

	return child
}

func (bn *BreakdownNode) filter(fromLevel int, min float64, max float64) {
	bn.filterLevel(1, fromLevel, min, max)
}

func (bn *BreakdownNode) filterLevel(currentLevel int, fromLevel int, min float64, max float64) {
	for key, child := range bn.children {
		if currentLevel >= fromLevel && (child.measurement < min || child.measurement > max) {
			delete(bn.children, key)
		} else {
			child.filterLevel(currentLevel+1, fromLevel, min, max)
		}
	}
}

func (bn *BreakdownNode) filterByName(filterFunc filterFuncType) {
	for key, child := range bn.children {
		if filterFunc(child.name) {
			child.filterByName(filterFunc)
		} else {
			delete(bn.children, key)
		}
	}
}

func (bn *BreakdownNode) depth() int {
	max := 0
	for _, child := range bn.children {
		cd := child.depth()
		if cd > max {
			max = cd
		}
	}

	return max + 1
}

func (bn *BreakdownNode) propagate() {
	for _, child := range bn.children {
		child.propagate()
		bn.measurement += child.measurement
		bn.numSamples += child.numSamples
	}
}

func (bn *BreakdownNode) increment(value float64, numSamples int64) {
	bn.measurement += value
	bn.numSamples += numSamples
}

func (bn *BreakdownNode) updateCounter(value int64, numSamples int64) {
	atomic.AddInt64(&bn.counter, value)
	atomic.AddInt64(&bn.numSamples, numSamples)
}

func (bn *BreakdownNode) evaluateCounter() {
	bn.measurement = float64(bn.counter)

	for _, child := range bn.children {
		child.evaluateCounter()
	}
}

func (bn *BreakdownNode) updateP95(value float64) {
	rLen := 0
	rExists := true

	bn.updateLock.RLock()
	if bn.reservoir == nil {
		rExists = false
	} else {
		rLen = len(bn.reservoir)
	}
	bn.updateLock.RUnlock()

	if !rExists {
		bn.updateLock.Lock()
		bn.reservoir = make(Reservoir, 0, ReservoirSize)
		bn.updateLock.Unlock()
	}

	if rLen < ReservoirSize {
		bn.updateLock.Lock()
		bn.reservoir = append(bn.reservoir, math.Float64bits(value))
		bn.updateLock.Unlock()
	} else {
		atomic.StoreUint64(&bn.reservoir[rand.Intn(ReservoirSize)], math.Float64bits(value))
	}

	atomic.AddInt64(&bn.numSamples, 1)
}

func (bn *BreakdownNode) evaluateP95() {
	if bn.reservoir != nil && len(bn.reservoir) > 0 {
		sort.Sort(bn.reservoir)
		index := int(math.Floor(float64(len(bn.reservoir)) / 100.0 * 95.0))
		bn.measurement = math.Float64frombits(bn.reservoir[index])

		bn.reservoir = bn.reservoir[:0]
	}

	for _, child := range bn.children {
		child.evaluateP95()
	}
}

func (bn *BreakdownNode) convertToPercentage(total float64) {
	bn.measurement = (bn.measurement / total) * 100.0
	for _, child := range bn.children {
		child.convertToPercentage(total)
	}
}

func (bn *BreakdownNode) normalize(factor float64) {
	bn.measurement = bn.measurement / factor
	bn.numSamples = int64(math.Ceil(float64(bn.numSamples) / factor))
	for _, child := range bn.children {
		child.normalize(factor)
	}
}

func (bn *BreakdownNode) round() {
	_, d := math.Modf(bn.measurement)
	if d > 0.5 {
		bn.measurement = math.Ceil(bn.measurement)
	} else {
		bn.measurement = math.Floor(bn.measurement)
	}
	for _, child := range bn.children {
		child.round()
	}
}

func round(val float64, roundOn float64, places int) float64 {
	var round float64
	pow := math.Pow(10, float64(places))
	digit := pow * val
	_, div := math.Modf(digit)
	if div >= roundOn {
		round = math.Ceil(digit)
	} else {
		round = math.Floor(digit)
	}

	return round / pow
}

func (bn *BreakdownNode) clone() *BreakdownNode {
	cln := newBreakdownNode(bn.name)
	cln.measurement = bn.measurement
	cln.numSamples = bn.numSamples

	for _, child := range bn.children {
		cln.addChild(child.clone())
	}

	return cln
}

func (bn *BreakdownNode) toMap() map[string]interface{} {
	childrenMap := make([]interface{}, 0)
	for _, child := range bn.children {
		childrenMap = append(childrenMap, child.toMap())
	}

	nodeMap := map[string]interface{}{
		"name":        bn.name,
		"measurement": bn.measurement,
		"num_samples": bn.numSamples,
		"children":    childrenMap,
	}

	return nodeMap
}

func (bn *BreakdownNode) printLevel(level int) string {
	str := ""

	for i := 0; i < level; i++ {
		str += "  "
	}

	str += fmt.Sprintf("%v - %v (%v)\n", bn.name, bn.measurement, bn.numSamples)
	for _, child := range bn.children {
		str += child.printLevel(level + 1)
	}

	return str
}

type Measurement struct {
	id        string
	trigger   string
	value     float64
	duration  int64
	breakdown *BreakdownNode
	timestamp int64
}

type Metric struct {
	agent        *Agent
	id           string
	typ          string
	category     string
	name         string
	unit         string
	measurement  *Measurement
	hasLastValue bool
	lastValue    float64
}

func newMetric(agent *Agent, typ string, category string, name string, unit string) *Metric {
	metricID := sha1String(agent.AppName + agent.AppEnvironment + agent.HostName + typ + category + name + unit)

	m := &Metric{
		agent:        agent,
		id:           metricID,
		typ:          typ,
		category:     category,
		name:         name,
		unit:         unit,
		measurement:  nil,
		hasLastValue: false,
		lastValue:    0,
	}

	return m
}

func (m *Metric) hasMeasurement() bool {
	return m.measurement != nil
}

func (m *Metric) createMeasurement(trigger string, value float64, duration int64, breakdown *BreakdownNode) {
	ready := true

	if m.typ == TypeCounter {
		if !m.hasLastValue {
			ready = false
			m.hasLastValue = true
			m.lastValue = value
		} else {
			tmpValue := value
			value = value - m.lastValue
			m.lastValue = tmpValue
		}
	}

	if ready {
		m.measurement = &Measurement{
			id:        m.agent.uuid(),
			trigger:   trigger,
			value:     value,
			duration:  duration,
			breakdown: breakdown,
			timestamp: time.Now().Unix(),
		}
	}
}

func (m *Metric) toMap() map[string]interface{} {
	var measurementMap map[string]interface{} = nil
	if m.measurement != nil {
		var breakdownMap map[string]interface{} = nil
		if m.measurement.breakdown != nil {
			breakdownMap = m.measurement.breakdown.toMap()
		}

		measurementMap = map[string]interface{}{
			"id":        m.measurement.id,
			"trigger":   m.measurement.trigger,
			"value":     m.measurement.value,
			"duration":  m.measurement.duration,
			"breakdown": breakdownMap,
			"timestamp": m.measurement.timestamp,
		}
	}

	metricMap := map[string]interface{}{
		"id":          m.id,
		"type":        m.typ,
		"category":    m.category,
		"name":        m.name,
		"unit":        m.unit,
		"measurement": measurementMap,
	}

	return metricMap
}
