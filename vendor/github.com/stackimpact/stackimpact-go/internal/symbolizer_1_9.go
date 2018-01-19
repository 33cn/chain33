// +build !go1.10

package internal

import (
	"github.com/stackimpact/stackimpact-go/internal/pprof/profile"
	"runtime"
)

func symbolizeProfile(p *profile.Profile) error {
	functions := make(map[string]*profile.Function)

	for _, l := range p.Location {
		if l.Address != 0 && len(l.Line) == 0 {
			if f := runtime.FuncForPC(uintptr(l.Address)); f != nil {
				name := f.Name()
				fileName, lineNumber := f.FileLine(uintptr(l.Address))

				pf := functions[name]
				if pf == nil {
					pf = &profile.Function{
						ID:         uint64(len(p.Function) + 1),
						Name:       name,
						SystemName: name,
						Filename:   fileName,
					}

					functions[name] = pf
					p.Function = append(p.Function, pf)
				}

				line := profile.Line{
					Function: pf,
					Line:     int64(lineNumber),
				}

				l.Line = []profile.Line{line}
				if l.Mapping != nil {
					l.Mapping.HasFunctions = true
					l.Mapping.HasFilenames = true
					l.Mapping.HasLineNumbers = true
				}
			}
		}
	}

	return nil
}
