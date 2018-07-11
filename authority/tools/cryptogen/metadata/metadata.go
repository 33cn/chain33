package metadata

import (
	"fmt"
	"runtime"
)

var Version string

const ProgramName = "cryptogen"

func GetVersionInfo() string {
	if Version == "" {
		Version = "development build"
	}

	return fmt.Sprintf("%s:\n Version: %s\n Go version: %s\n OS/Arch: %s",
		ProgramName, Version, runtime.Version(),
		fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH))
}
