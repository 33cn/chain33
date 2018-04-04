package internal

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"math"
	"runtime"
	"runtime/pprof"

	profile "github.com/stackimpact/stackimpact-go/internal/pprof/profile"
)

type BlockValues struct {
	delay       float64
	contentions int64
}

type BlockProfiler struct {
	agent          *Agent
	blockProfile   *BreakdownNode
	blockTrace     *BreakdownNode
	prevValues     map[string]*BlockValues
	partialProfile *pprof.Profile
}

func newBlockProfiler(agent *Agent) *BlockProfiler {
	br := &BlockProfiler{
		agent:          agent,
		blockProfile:   nil,
		blockTrace:     nil,
		prevValues:     make(map[string]*BlockValues),
		partialProfile: nil,
	}

	return br
}

func (bp *BlockProfiler) reset() {
	bp.blockProfile = newBreakdownNode("root")
	bp.blockTrace = newBreakdownNode("root")
}

func (bp *BlockProfiler) startProfiler() error {
	err := bp.startBlockProfiler()
	if err != nil {
		return err
	}

	return nil
}

func (bp *BlockProfiler) stopProfiler() error {
	p, err := bp.stopBlockProfiler()
	if err != nil {
		return err
	}
	if p == nil {
		return errors.New("no profile returned")
	}

	if uerr := bp.updateBlockProfile(p); uerr != nil {
		return uerr
	}

	return nil
}

func (bp *BlockProfiler) buildProfile(duration int64, workloads map[string]int64) ([]*ProfileData, error) {
	bp.blockProfile.normalize(float64(duration) / 1e9)
	bp.blockProfile.propagate()
	bp.blockProfile.filter(2, 1, math.Inf(0))

	bp.blockTrace.evaluateP95()
	bp.blockTrace.propagate()
	bp.blockTrace.round()
	bp.blockTrace.filter(2, 1, math.Inf(0))

	data := []*ProfileData{
		&ProfileData{
			category:     CategoryBlockProfile,
			name:         NameBlockingCallTimes,
			unit:         UnitMillisecond,
			unitInterval: 1,
			profile:      bp.blockProfile,
		},
		&ProfileData{
			category:     CategoryBlockTrace,
			name:         NameBlockingCallTimes,
			unit:         UnitMillisecond,
			unitInterval: 0,
			profile:      bp.blockTrace,
		},
	}

	return data, nil
}

func (bp *BlockProfiler) updateBlockProfile(p *profile.Profile) error {
	contentionIndex := -1
	delayIndex := -1
	for i, s := range p.SampleType {
		if s.Type == "contentions" {
			contentionIndex = i
		} else if s.Type == "delay" {
			delayIndex = i
		}
	}

	if contentionIndex == -1 || delayIndex == -1 {
		return errors.New("Unrecognized profile data")
	}

	for _, s := range p.Sample {
		if !bp.agent.ProfileAgent && isAgentStack(s) {
			continue
		}

		delay := float64(s.Value[delayIndex])
		contentions := s.Value[contentionIndex]

		valueKey := generateValueKey(s)
		delay, contentions = bp.getValueChange(valueKey, delay, contentions)

		if contentions == 0 || delay == 0 {
			continue
		}

		// to milliseconds
		delay = delay / 1e6

		currentNode := bp.blockProfile
		for i := len(s.Location) - 1; i >= 0; i-- {
			l := s.Location[i]
			funcName, fileName, fileLine := readFuncInfo(l)

			if (!bp.agent.ProfileAgent && isAgentFrame(fileName)) || funcName == "runtime.goexit" {
				continue
			}

			frameName := fmt.Sprintf("%v (%v:%v)", funcName, fileName, fileLine)
			currentNode = currentNode.findOrAddChild(frameName)
		}
		currentNode.increment(delay, contentions)

		currentNode = bp.blockTrace
		for i := len(s.Location) - 1; i >= 0; i-- {
			l := s.Location[i]
			funcName, fileName, fileLine := readFuncInfo(l)

			if (!bp.agent.ProfileAgent && isAgentFrame(fileName)) || funcName == "runtime.goexit" {
				continue
			}

			frameName := fmt.Sprintf("%v (%v:%v)", funcName, fileName, fileLine)
			currentNode = currentNode.findOrAddChild(frameName)
		}
		currentNode.updateP95(delay / float64(contentions))
	}

	return nil
}

func generateValueKey(s *profile.Sample) string {
	key := ""
	for _, l := range s.Location {
		key += fmt.Sprintf("%v:", l.Address)
	}

	return key
}

func (bp *BlockProfiler) getValueChange(key string, delay float64, contentions int64) (float64, int64) {
	if pv, exists := bp.prevValues[key]; exists {
		delayChange := delay - pv.delay
		contentionsChange := contentions - pv.contentions

		pv.delay = delay
		pv.contentions = contentions

		return delayChange, contentionsChange
	} else {
		bp.prevValues[key] = &BlockValues{
			delay:       delay,
			contentions: contentions,
		}

		return delay, contentions
	}
}

func (bp *BlockProfiler) startBlockProfiler() error {
	bp.partialProfile = pprof.Lookup("block")
	if bp.partialProfile == nil {
		return errors.New("No block profile found")
	}

	runtime.SetBlockProfileRate(1e6)

	return nil
}

func (bp *BlockProfiler) stopBlockProfiler() (*profile.Profile, error) {
	runtime.SetBlockProfileRate(0)

	var buf bytes.Buffer
	w := bufio.NewWriter(&buf)

	err := bp.partialProfile.WriteTo(w, 0)
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
