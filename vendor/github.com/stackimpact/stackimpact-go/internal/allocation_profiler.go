package internal

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"math"
	"runtime"
	"runtime/pprof"

	"github.com/stackimpact/stackimpact-go/internal/pprof/profile"
)

type recordSorter []runtime.MemProfileRecord

func (x recordSorter) Len() int {
	return len(x)
}

func (x recordSorter) Swap(i, j int) {
	x[i], x[j] = x[j], x[i]
}

func (x recordSorter) Less(i, j int) bool {
	return x[i].InUseBytes() > x[j].InUseBytes()
}

func readMemAlloc() float64 {
	memStats := &runtime.MemStats{}
	runtime.ReadMemStats(memStats)
	return float64(memStats.Alloc)
}

type AllocationProfiler struct {
	agent *Agent
}

func newAllocationProfiler(agent *Agent) *AllocationProfiler {
	ar := &AllocationProfiler{
		agent: agent,
	}

	return ar
}

func (ap *AllocationProfiler) reset() {
}

func (ap *AllocationProfiler) startProfiler() error {
	return nil
}

func (ap *AllocationProfiler) stopProfiler() error {
	return nil
}

func (ap *AllocationProfiler) buildProfile(duration int64, workloads map[string]int64) ([]*ProfileData, error) {
	p, err := ap.readHeapProfile()
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, errors.New("no profile returned")
	}

	if callGraph, aerr := ap.createAllocationCallGraph(p); err != nil {
		return nil, aerr
	} else {
		callGraph.propagate()
		// filter calls with lower than 10KB
		callGraph.filter(2, 10000, math.Inf(0))

		data := []*ProfileData{
			&ProfileData{
				category:     CategoryMemoryProfile,
				name:         NameHeapAllocation,
				unit:         UnitByte,
				unitInterval: 0,
				profile:      callGraph,
			},
		}

		return data, nil
	}
}

func (ap *AllocationProfiler) createAllocationCallGraph(p *profile.Profile) (*BreakdownNode, error) {
	// find "inuse_space" type index
	inuseSpaceTypeIndex := -1
	for i, s := range p.SampleType {
		if s.Type == "inuse_space" {
			inuseSpaceTypeIndex = i
			break
		}
	}

	// find "inuse_space" type index
	inuseObjectsTypeIndex := -1
	for i, s := range p.SampleType {
		if s.Type == "inuse_objects" {
			inuseObjectsTypeIndex = i
			break
		}
	}

	if inuseSpaceTypeIndex == -1 || inuseObjectsTypeIndex == -1 {
		return nil, errors.New("Unrecognized profile data")
	}

	// build call graph
	rootNode := newBreakdownNode("root")

	for _, s := range p.Sample {
		if !ap.agent.ProfileAgent && isAgentStack(s) {
			continue
		}

		value := s.Value[inuseSpaceTypeIndex]
		count := s.Value[inuseObjectsTypeIndex]
		if value == 0 {
			continue
		}

		currentNode := rootNode
		for i := len(s.Location) - 1; i >= 0; i-- {
			l := s.Location[i]
			funcName, fileName, fileLine := readFuncInfo(l)

			if (!ap.agent.ProfileAgent && isAgentFrame(fileName)) || funcName == "runtime.goexit" {
				continue
			}

			frameName := fmt.Sprintf("%v (%v:%v)", funcName, fileName, fileLine)
			currentNode = currentNode.findOrAddChild(frameName)
		}
		currentNode.increment(float64(value), int64(count))
	}

	return rootNode, nil
}

func (ap *AllocationProfiler) readHeapProfile() (*profile.Profile, error) {
	var buf bytes.Buffer
	w := bufio.NewWriter(&buf)

	err := pprof.WriteHeapProfile(w)
	if err != nil {
		return nil, err
	}

	w.Flush()
	r := bufio.NewReader(&buf)

	if p, perr := profile.Parse(r); perr == nil {
		if serr := symbolizeProfile(p); serr != nil {
			return nil, serr
		}

		if verr := p.CheckValid(); verr != nil {
			return nil, verr
		}

		return p, nil
	} else {
		return nil, perr
	}
}
