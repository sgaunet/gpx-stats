package cli

import (
	"fmt"
	"io"
	"runtime"
	"runtime/debug"
	"strings"
)

// Version, Commit and Date are stamped at link time by GoReleaser:
//
//	-X github.com/sgaunet/gpx-stats/internal/cli.Version={{ .Version }}
//
// They are empty in every other build — go build, go install, go test — which is
// why versionFrom falls back to the build information the toolchain embeds on
// its own. They must stay exported package-level string vars that something
// actually reads: rename one, retype one, or stop reading one and -X becomes a
// silent no-op, with no build error and no lint error to catch it.
var (
	Version string
	Commit  string
	Date    string
)

const (
	// binaryName labels the version row, so the block identifies itself when it
	// is pasted into a bug report beside another tool's --version output.
	binaryName = "gpx-stats"

	// unknownField is what a row shows when neither the linker nor the toolchain
	// supplied the value — a source build of a tree with no VCS, or a module
	// fetched from the proxy, which carries a version but no commit. Never the
	// empty string: that would pad the label column and leave trailing space.
	unknownField = "unknown"
)

// versionInfo is the four resolved rows of the --version block. Resolution and
// rendering are separate so the fallback branches can be exercised without
// building the test binary six different ways.
type versionInfo struct {
	version   string
	commit    string
	date      string
	goVersion string
}

// versionFrom resolves the link-time stamps, falling back per field to the build
// information the Go toolchain embeds on its own.
//
// The fallback is what makes a `go install ...@v0.4.0` binary report 0.4.0
// rather than "unknown": the toolchain records the main module's version, and
// records vcs.revision and vcs.time for a build made inside a repository.
// GoReleaser's -trimpath does not strip them.
//
// It is per field rather than all-or-nothing because the two sources have
// different gaps: an install from the module proxy yields a version but no vcs
// settings, and a build of an untagged tree yields vcs settings and only a
// pseudo-version.
func versionFrom(version, commit, date string, bi *debug.BuildInfo) versionInfo {
	v := versionInfo{
		version: version,
		commit:  commit,
		date:    date,
		// Never read from bi: runtime.Version is what ReadBuildInfo itself
		// assigns to bi.GoVersion, and GOOS/GOARCH are compile-time constants,
		// so this row stays correct even with no build information at all.
		goVersion: fmt.Sprintf("%s %s/%s", runtime.Version(), runtime.GOOS, runtime.GOARCH),
	}

	if bi != nil {
		if v.version == "" {
			// The leading "v" is trimmed from the derived value and never from
			// the stamped one: GoReleaser's {{ .Version }} is the tag without
			// it, Main.Version is the tag with it, and trimming here is what
			// makes one tag report one string however it was built.
			v.version = strings.TrimPrefix(bi.Main.Version, "v")
		}
		for _, s := range bi.Settings {
			switch s.Key {
			case "vcs.revision":
				if v.commit == "" {
					v.commit = s.Value
				}
			case "vcs.time":
				if v.date == "" {
					v.date = s.Value
				}
			}
		}
	}

	if v.version == "" {
		v.version = unknownField
	}
	if v.commit == "" {
		v.commit = unknownField
	}
	if v.date == "" {
		v.date = unknownField
	}
	return v
}

// resolveVersion reads this binary's own build information and resolves it
// against the link-time stamps.
func resolveVersion() versionInfo {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return versionFrom(Version, Commit, Date, nil)
	}
	return versionFrom(Version, Commit, Date, bi)
}

// WriteVersion renders the build identity of this binary to w: the release
// version, the commit it was built from, when it was built, and the toolchain
// that built it.
//
// It goes through the same table as the statistics report, so the four values
// share one column whose width comes from the labels themselves. Nothing in the
// block is styled, which is why it is byte-identical with and without
// --no-color; the option is accepted so the call site reads like every other
// writer in the package.
func WriteVersion(w io.Writer, opts ...Option) {
	writeVersionInfo(w, styleFor(w, opts), resolveVersion())
}

// writeVersionInfo renders already-resolved values. The labels are lowercase on
// purpose — they name fields of a build identity, not metrics — and, like every
// other label in the report, carry no trailing colon.
//
// Every row is text, never num: num right-aligns against the widest value, and
// the 40-character commit would push the version out to the right margin.
func writeVersionInfo(w io.Writer, st style, v versionInfo) {
	t := &table{st: st}
	t.text(binaryName, v.version)
	t.text("commit", v.commit)
	t.text("built", v.date)
	t.text("go", v.goVersion)
	t.flush(w)
}
