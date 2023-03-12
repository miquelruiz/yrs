package vcs

import (
	"runtime/debug"
)

var (
	Version string
	Time    string
)

func init() {
	var ok bool
	infoMap := make(map[string]string, 0)
	if bi, ok := debug.ReadBuildInfo(); ok {
		for _, s := range bi.Settings {
			infoMap[s.Key] = s.Value
		}
	}

	// If it's been initialized via ldflags, don't overwrite it
	if Version == "" {
		if Version, ok = infoMap["vcs.revision"]; !ok {
			Version = "unknown"
		}
	}

	if Time, ok = infoMap["vcs.time"]; !ok {
		Time = "unknown"
	}
}
