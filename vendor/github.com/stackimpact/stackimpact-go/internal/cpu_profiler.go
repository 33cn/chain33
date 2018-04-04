package internal

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"math"
	"runtime"
	"runtime/pprof"
	"sort"
	"strings"
	"time"

	"github.com/stackimpact/stackimpact-go/internal/pprof/profile"
)

type CPUProfiler struct {
	agent         *Agent
	profile       *BreakdownNode
	labelProfiles map[string]*BreakdownNode
	profWriter    *bufio.Writer
	profBuffer    *bytes.Buffer
	startNano     int64
}

func newCPUProfiler(agent *Agent) *CPUProfiler {
	cp := &CPUProfiler{
		agent:         agent,
		profile:       nil,
		labelProfiles: nil,
		profWriter:    nil,
		profBuffer:    nil,
		startNano:     0,
	}

	return cp
}

func (cp *CPUProfiler) reset() {
	cp.profile = newBreakdownNode("root")
	cp.labelProfiles = make(map[string]*BreakdownNode)
}

func (cp *CPUProfiler) startProfiler() error {
	err := cp.startCPUProfiler()
	if err != nil {
		return err
	}

	return nil
}

func (cp *CPUProfiler) stopProfiler() error {
	p, err := cp.stopCPUProfiler()
	if err != nil {
		return err
	}
	if p == nil {
		return errors.New("no profile returned")
	}

	if uerr := cp.updateCPUProfile(p); uerr != nil {
		return uerr
	}

	return nil
}

type ProfileDataSorter []*ProfileData

func (pds ProfileDataSorter) Len() int {
	return len(pds)
}
func (pds ProfileDataSorter) Swap(i, j int) {
	pds[i].profile.measurement, pds[j].profile.measurement = pds[j].profile.measurement, pds[i].profile.measurement
}
func (pds ProfileDataSorter) Less(i, j int) bool {
	return pds[i].profile.measurement < pds[j].profile.measurement
}

func (cp *CPUProfiler) buildProfile(duration int64, workloads map[string]int64) ([]*ProfileData, error) {
	cp.profile.convertToPercentage(float64(duration * int64(runtime.NumCPU())))
	cp.profile.propagate()
	// filter calls with lower than 1% CPU stake
	cp.profile.filter(2, 1, 100)

	data := make([]*ProfileData, 0)

	for label, labelProfile := range cp.labelProfiles {
		numSpans, ok := workloads[label]
		if !ok {
			continue
		}

		labelProfile.normalize(float64(numSpans * 1e6))
		labelProfile.propagate()
		labelProfile.filter(2, 1, math.Inf(0))

		data = append(data, &ProfileData{
			category:     CategoryCPUTrace,
			name:         fmt.Sprintf("%v: %v", NameCPUTime, label),
			unit:         UnitMillisecond,
			unitInterval: 0,
			profile:      labelProfile,
		})
	}

	sort.Sort(sort.Reverse(ProfileDataSorter(data)))
	if len(data) > 5 {
		data = data[:5]
	}

	// prepend full profile
	data = append([]*ProfileData{
		&ProfileData{
			category:     CategoryCPUProfile,
			name:         NameCPUUsage,
			unit:         UnitPercent,
			unitInterval: 0,
			profile:      cp.profile,
		},
	}, data...)

	return data, nil
}

func (cp *CPUProfiler) updateCPUProfile(p *profile.Profile) error {
	samplesIndex := -1
	cpuIndex := -1
	for i, s := range p.SampleType {
		if s.Type == "samples" {
			samplesIndex = i
		} else if s.Type == "cpu" {
			cpuIndex = i
		}
	}

	if samplesIndex == -1 || cpuIndex == -1 {
		return errors.New("Unrecognized profile data")
	}

	// build call graph
	for _, s := range p.Sample {
		if !cp.agent.ProfileAgent && isAgentStack(s) {
			continue
		}

		stackSamples := s.Value[samplesIndex]
		stackDuration := float64(s.Value[cpuIndex])

		currentNode := cp.profile
		for i := len(s.Location) - 1; i >= 0; i-- {
			l := s.Location[i]
			funcName, fileName, fileLine := readFuncInfo(l)

			if (!cp.agent.ProfileAgent && isAgentFrame(fileName)) || funcName == "runtime.goexit" {
				continue
			}

			frameName := fmt.Sprintf("%v (%v:%v)", funcName, fileName, fileLine)
			currentNode = currentNode.findOrAddChild(frameName)
		}

		currentNode.increment(stackDuration, stackSamples)

		labelProfile := cp.findLabelProfile(s)
		if labelProfile != nil {
			currentNode := labelProfile
			for i := len(s.Location) - 1; i >= 0; i-- {
				l := s.Location[i]
				funcName, fileName, fileLine := readFuncInfo(l)

				if (!cp.agent.ProfileAgent && isAgentFrame(fileName)) || funcName == "runtime.goexit" {
					continue
				}

				frameName := fmt.Sprintf("%v (%v:%v)", funcName, fileName, fileLine)
				currentNode = currentNode.findOrAddChild(frameName)
			}

			currentNode.increment(stackDuration, stackSamples)
		}
	}

	return nil
}

func (cp *CPUProfiler) findLabelProfile(sample *profile.Sample) *BreakdownNode {
	if sample.Label != nil {
		if labels, ok := sample.Label["workload"]; ok {
			if len(labels) > 0 {
				if lp, ok := cp.labelProfiles[labels[0]]; ok {
					return lp
				} else {
					lp = newBreakdownNode("root")
					cp.labelProfiles[labels[0]] = lp
					return lp
				}
			}
		}
	}

	return nil
}

func (cp *CPUProfiler) startCPUProfiler() error {
	cp.profBuffer = &bytes.Buffer{}
	cp.profWriter = bufio.NewWriter(cp.profBuffer)
	cp.startNano = time.Now().UnixNano()

	err := pprof.StartCPUProfile(cp.profWriter)
	if err != nil {
		return err
	}

	return nil
}

func (cp *CPUProfiler) stopCPUProfiler() (*profile.Profile, error) {
	pprof.StopCPUProfile()

	cp.profWriter.Flush()
	r := bufio.NewReader(cp.profBuffer)

	if p, perr := profile.Parse(r); perr == nil {
		cp.profWriter = nil
		cp.profBuffer = nil

		if p.TimeNanos == 0 {
			p.TimeNanos = cp.startNano
		}
		if p.DurationNanos == 0 {
			p.DurationNanos = time.Now().UnixNano() - cp.startNano
		}

		if serr := symbolizeProfile(p); serr != nil {
			return nil, serr
		}

		if verr := p.CheckValid(); verr != nil {
			return nil, verr
		}

		return p, nil
	} else {
		cp.profWriter = nil
		cp.profBuffer = nil

		return nil, perr
	}
}

func isAgentStack(sample *profile.Sample) bool {
	return stackContains(sample, "", agentPathInternal)
}

func isAgentFrame(fileNameTest string) bool {
	return strings.Contains(fileNameTest, agentPath) &&
		!strings.Contains(fileNameTest, agentPathExamples)
}

func stackContains(sample *profile.Sample, funcNameTest string, fileNameTest string) bool {
	for i := len(sample.Location) - 1; i >= 0; i-- {
		l := sample.Location[i]
		funcName, fileName, _ := readFuncInfo(l)

		if (funcNameTest == "" || strings.Contains(funcName, funcNameTest)) &&
			(fileNameTest == "" || strings.Contains(fileName, fileNameTest)) {
			return true
		}
	}

	return false
}

func readFuncInfo(l *profile.Location) (funcName string, fileName string, fileLine int64) {
	for li := range l.Line {
		if fn := l.Line[li].Function; fn != nil {
			return fn.Name, fn.Filename, l.Line[li].Line
		}
	}

	return "", "", 0
}
