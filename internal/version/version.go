// Package version reports build information about the running txoko binary.
package version

import (
	"fmt"
	"runtime"
	"runtime/debug"
	"strings"
)

// Build metadata. Release builds override these with
// -ldflags "-X github.com/LifeOfAitor/txoko/internal/version.Version=v1.2.3".
var (
	Version = "dev"
	Commit  = ""
	Date    = ""
)

const unknown = "unknown"

// Info describes the running build.
type Info struct {
	Version   string
	Commit    string
	Date      string
	GoVersion string
	Platform  string
}

// Get returns build information for the running binary. Values injected at
// link time take priority; anything missing is filled in from the metadata
// the Go toolchain embeds (for example when installed with `go install`).
func Get() Info {
	return resolve(Version, Commit, Date, debug.ReadBuildInfo)
}

// buildInfoReader matches debug.ReadBuildInfo so tests can provide fakes.
type buildInfoReader func() (*debug.BuildInfo, bool)

func resolve(ver, commit, date string, read buildInfoReader) Info {
	info := Info{
		Version:   ver,
		Commit:    commit,
		Date:      date,
		GoVersion: runtime.Version(),
		Platform:  runtime.GOOS + "/" + runtime.GOARCH,
	}

	if bi, ok := read(); ok && bi != nil {
		if info.Version == "dev" && bi.Main.Version != "" && bi.Main.Version != "(devel)" {
			info.Version = bi.Main.Version
		}
		for _, s := range bi.Settings {
			switch s.Key {
			case "vcs.revision":
				if info.Commit == "" {
					info.Commit = s.Value
				}
			case "vcs.time":
				if info.Date == "" {
					info.Date = s.Value
				}
			}
		}
	}

	info.Commit = shortCommit(info.Commit)
	if info.Date == "" {
		info.Date = unknown
	}
	return info
}

func shortCommit(c string) string {
	switch {
	case c == "":
		return unknown
	case len(c) > 7:
		return c[:7]
	default:
		return c
	}
}

// String renders the build information as a small aligned block:
//
//	txoko v0.1.0
//	  commit  1a2b3c4
//	  built   2026-11-16T15:00:00Z
//	  go      go1.27.4 darwin/arm64
func (i Info) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "txoko %s\n", i.Version)
	fmt.Fprintf(&b, "  commit  %s\n", i.Commit)
	fmt.Fprintf(&b, "  built   %s\n", i.Date)
	fmt.Fprintf(&b, "  go      %s %s\n", i.GoVersion, i.Platform)
	return b.String()
}
