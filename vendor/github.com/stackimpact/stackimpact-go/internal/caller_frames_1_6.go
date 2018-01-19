// +build !go1.7

package internal

import (
	"fmt"
	"runtime"
)

func callerFrames(skip int, count int) []string {
	stack := make([]uintptr, count)
	runtime.Callers(skip+2, stack)

	frames := make([]string, 0)
	for _, pc := range stack {
		if pc != 0 {
			if fn := runtime.FuncForPC(pc); fn != nil {
				funcName := fn.Name()

				if funcName == "runtime.goexit" {
					continue
				}

				fileName, lineNumber := fn.FileLine(pc)
				frames = append(frames, fmt.Sprintf("%v (%v:%v)", fn.Name(), fileName, lineNumber))
			}
		}
	}

	return frames
}
