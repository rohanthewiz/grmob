package main

import (
	"runtime/debug"
	"strings"
)

// versionString is `grmob version`'s answer: the module version this binary
// was built from, and where that is not a release, what it is instead.
//
// The grmob module is the main module of this binary (it holds package
// main) however the command is run: `go tool grmob` in an app,
// `go run ...@vX`, or a build inside a checkout. Three cases follow:
//
//	a release or pseudo-version      printed as it is
//	a replace directive (an app      the go.mod placeholder version means
//	  made with new -replace)        nothing, so the directory is printed
//	a checkout build ("(devel)")     the VCS revision stamped into it
//
// A bug report that says "(devel)" alone does not say which commit, which is
// why the last two cases say more.
func versionString() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "grmob (no build information)"
	}
	m := &info.Main
	if m.Path != grmobModule {
		// Defensive: a binary that embeds this package under another main
		// module. Look for grmob among the dependencies instead.
		for _, dep := range info.Deps {
			if dep.Path == grmobModule {
				m = dep
				break
			}
		}
	}
	v := m.Version
	if m.Replace != nil {
		v = "(replaced by " + m.Replace.Path
		if m.Replace.Version != "" && m.Replace.Version != "(devel)" {
			v += " " + m.Replace.Version
		}
		v += ")"
	}
	if v == "" || v == "(devel)" {
		v = "(devel)"
		var rev, dirty string
		for _, s := range info.Settings {
			switch s.Key {
			case "vcs.revision":
				rev = s.Value
			case "vcs.modified":
				if s.Value == "true" {
					dirty = ", modified"
				}
			}
		}
		if rev != "" {
			if len(rev) > 12 {
				rev = rev[:12]
			}
			v = "(devel " + rev + dirty + ")"
		}
	}
	return strings.TrimSpace("grmob " + v + " " + info.GoVersion)
}
