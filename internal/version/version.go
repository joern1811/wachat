package version

import (
	"flag"
	"runtime"
)

var (
	Version   = "develop"
	GitCommit = ""
	BuildDate = ""
)

type BuildInfo struct {
	Version   string `json:"version,omitempty"`
	GitCommit string `json:"gitCommit,omitempty"`
	BuildDate string `json:"buildDate,omitempty"`
	GoVersion string `json:"goVersion,omitempty"`
}

func Get() BuildInfo {
	v := BuildInfo{
		Version:   Version,
		GitCommit: GitCommit,
		BuildDate: BuildDate,
		GoVersion: runtime.Version(),
	}

	if flag.Lookup("test.v") != nil {
		v.GoVersion = ""
	}
	return v
}
