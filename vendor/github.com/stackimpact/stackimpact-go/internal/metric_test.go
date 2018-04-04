package internal

import (
	"math/rand"
	"testing"
)

func TestCreateMeasurement(t *testing.T) {
	agent := NewAgent()
	agent.Debug = true

	m := newMetric(agent, TypeCounter, CategoryCPU, NameCPUUsage, UnitNone)

	m.createMeasurement(TriggerTimer, 100, 0, nil)

	if m.hasMeasurement() {
		t.Errorf("Should not have measurement")
	}

	m.createMeasurement(TriggerTimer, 110, 0, nil)

	if m.measurement.value != 10 {
		t.Errorf("Value should be 10, but is %v", m.measurement.value)
	}

	m.createMeasurement(TriggerTimer, 115, 0, nil)

	if m.measurement.value != 5 {
		t.Errorf("Value should be 5, but is %v", m.measurement.value)
	}

}

func TestBreakdownFilter(t *testing.T) {
	agent := NewAgent()
	agent.Debug = true

	root := newBreakdownNode("root")
	root.measurement = 10

	child1 := newBreakdownNode("child1")
	child1.measurement = 9
	root.addChild(child1)

	child2 := newBreakdownNode("child2")
	child2.measurement = 1
	root.addChild(child2)

	child2child1 := newBreakdownNode("child2child1")
	child2child1.measurement = 1
	child2.addChild(child2child1)

	root.filter(2, 3, 100)

	if root.findChild("child1") == nil {
		t.Errorf("child1 should not be filtered")
	}

	if root.findChild("child2") == nil {
		t.Errorf("child2 should not be filtered")
	}

	if child2.findChild("child2child1") != nil {
		t.Errorf("child2child1 should be filtered")
	}
}

func TestBreakdownDepth(t *testing.T) {
	root := newBreakdownNode("root")

	child1 := newBreakdownNode("child1")
	root.addChild(child1)

	child2 := newBreakdownNode("child2")
	root.addChild(child2)

	child2child1 := newBreakdownNode("child2child1")
	child2.addChild(child2child1)

	if root.depth() != 3 {
		t.Errorf("root depth should be 3, but is %v", root.depth())
	}

	if child1.depth() != 1 {
		t.Errorf("child1 depth should be 1, but is %v", child1.depth())
	}

	if child2.depth() != 2 {
		t.Errorf("child2 depth should be 2, but is %v", child2.depth())
	}
}

func TestBreakdownIncrement(t *testing.T) {
	root := newBreakdownNode("root")

	root.increment(12.3, 1)
	root.increment(0, 0)
	root.increment(5, 2)

	if root.measurement != 17.3 {
		t.Errorf("root measurement should be 17.3, but is %v", root.measurement)
	}

	if root.numSamples != 3 {
		t.Errorf("root numSamples should be 3, but is %v", root.numSamples)
	}
}

func TestBreakdownCounter(t *testing.T) {
	root := newBreakdownNode("root")

	child1 := newBreakdownNode("child1")
	root.addChild(child1)

	child2 := newBreakdownNode("child2")
	root.addChild(child2)

	child2child1 := newBreakdownNode("child2child1")
	child2.addChild(child2child1)

	child2child1.updateCounter(6, 1)
	child2child1.updateCounter(4, 1)
	child2child1.updateCounter(0, 0)
	child2child1.evaluateCounter()
	root.propagate()

	if root.measurement != 10 {
		t.Errorf("root measurement should be 10, but is %v", root.measurement)
	}

	if root.numSamples != 2 {
		t.Errorf("root numSamples should be 2, but is %v", root.numSamples)
	}
}

func TestBreakdownP95(t *testing.T) {
	root := newBreakdownNode("root")

	child1 := newBreakdownNode("child1")
	root.addChild(child1)

	child2 := newBreakdownNode("child2")
	root.addChild(child2)

	child2child1 := newBreakdownNode("child2child1")
	child2.addChild(child2child1)

	child2child1.updateP95(6.5)
	child2child1.updateP95(4.2)
	child2child1.updateP95(5.0)
	child2child1.evaluateP95()
	root.propagate()

	if root.measurement != 6.5 {
		t.Errorf("root measurement should be 6, but is %v", root.measurement)
	}

	if root.numSamples != 3 {
		t.Errorf("root numSamples should be 3, but is %v", root.numSamples)
	}
}

func TestBreakdownP95Big(t *testing.T) {
	root := newBreakdownNode("root")

	for i := 0; i < 10000; i++ {
		root.updateP95(200.0 + float64(rand.Intn(50)))
	}
	root.evaluateP95()

	if root.measurement < 200 || root.measurement > 250 {
		t.Errorf("root measurement should be in [200, 250], but is %v", root.measurement)
	}
}
