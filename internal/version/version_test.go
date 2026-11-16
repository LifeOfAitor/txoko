package version

import (
	"runtime"
	"runtime/debug"
	"strings"
	"testing"
)

func fakeReader(bi *debug.BuildInfo, ok bool) buildInfoReader {
	return func() (*debug.BuildInfo, bool) { return bi, ok }
}

func TestResolve(t *testing.T) {
	vcs := &debug.BuildInfo{
		Main: debug.Module{Version: "v0.3.0"},
		Settings: []debug.BuildSetting{
			{Key: "vcs.revision", Value: "abcdef1234567890"},
			{Key: "vcs.time", Value: "2026-11-16T10:00:00Z"},
		},
	}

	tests := []struct {
		name                string
		ver, commit, date   string
		reader              buildInfoReader
		wantVer, wantCommit string
		wantDate            string
	}{
		{
			name: "ldflags win over build info",
			ver:  "v1.0.0", commit: "1111111222", date: "2026-01-01",
			reader:  fakeReader(vcs, true),
			wantVer: "v1.0.0", wantCommit: "1111111", wantDate: "2026-01-01",
		},
		{
			name:    "build info fills gaps for dev builds",
			ver:     "dev",
			reader:  fakeReader(vcs, true),
			wantVer: "v0.3.0", wantCommit: "abcdef1", wantDate: "2026-11-16T10:00:00Z",
		},
		{
			name:    "devel module version is ignored",
			ver:     "dev",
			reader:  fakeReader(&debug.BuildInfo{Main: debug.Module{Version: "(devel)"}}, true),
			wantVer: "dev", wantCommit: unknown, wantDate: unknown,
		},
		{
			name:    "no build info at all",
			ver:     "dev",
			reader:  fakeReader(nil, false),
			wantVer: "dev", wantCommit: unknown, wantDate: unknown,
		},
		{
			name: "short commit is kept as is",
			ver:  "dev", commit: "abc",
			reader:  fakeReader(nil, false),
			wantVer: "dev", wantCommit: "abc", wantDate: unknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolve(tt.ver, tt.commit, tt.date, tt.reader)
			if got.Version != tt.wantVer {
				t.Errorf("Version = %q, want %q", got.Version, tt.wantVer)
			}
			if got.Commit != tt.wantCommit {
				t.Errorf("Commit = %q, want %q", got.Commit, tt.wantCommit)
			}
			if got.Date != tt.wantDate {
				t.Errorf("Date = %q, want %q", got.Date, tt.wantDate)
			}
			if got.GoVersion != runtime.Version() {
				t.Errorf("GoVersion = %q, want %q", got.GoVersion, runtime.Version())
			}
			if want := runtime.GOOS + "/" + runtime.GOARCH; got.Platform != want {
				t.Errorf("Platform = %q, want %q", got.Platform, want)
			}
		})
	}
}

func TestInfoString(t *testing.T) {
	info := Info{
		Version:   "v0.1.0",
		Commit:    "1a2b3c4",
		Date:      "2026-11-16T15:00:00Z",
		GoVersion: "go1.27.4",
		Platform:  "darwin/arm64",
	}
	want := strings.Join([]string{
		"txoko v0.1.0",
		"  commit  1a2b3c4",
		"  built   2026-11-16T15:00:00Z",
		"  go      go1.27.4 darwin/arm64",
		"",
	}, "\n")

	if got := info.String(); got != want {
		t.Errorf("String() mismatch\n got:\n%s\nwant:\n%s", got, want)
	}
}

func TestGet(t *testing.T) {
	// Smoke test: Get must always return a fully populated Info.
	got := Get()
	for field, v := range map[string]string{
		"Version": got.Version, "Commit": got.Commit, "Date": got.Date,
		"GoVersion": got.GoVersion, "Platform": got.Platform,
	} {
		if v == "" {
			t.Errorf("%s is empty", field)
		}
	}
}
