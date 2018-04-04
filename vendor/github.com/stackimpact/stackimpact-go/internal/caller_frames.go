// +build go1.7

package internal

import (
	"fmt"
	"runtime"
)

func callerFrames(skip int, count int) []string {
	pc := make([]uintptr, count)
	n := runtime.Callers(skip+2, pc)
	if n == 0 {
		return []string{}
	}

	frames := runtime.CallersFrames(pc)

	fmtFrames := make([]string, 0)

	for i := 0; i < count; i++ {
		frame, more := frames.Next()

		if frame.Function != "" || frame.File != "" {
			if frame.Function != "runtime.goexit" {
				fmtFrames = append(fmtFrames, fmt.Sprintf("%v (%v:%v)", frame.Function, frame.File, frame.Line))
			}
		}

		if !more {
			break
		}
	}

	return fmtFrames
}
