package verify

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// The Compose half of the fixed-size census, held to the foundation-layout
// this build actually resolves.
//
// # What was wrong with it
//
// docs/platforms/native.md records what each of the four targets does with a
// child bigger than a fixed-size container. Two of those rows are readings of
// somebody else's source:
//
//	SwiftUI   .frame(width:) proposes a size and does not enforce one, so the
//	          child spills across the cross axis. The main-axis squeeze is
//	          GrMobFlexSolver's, which is ours, and ios/verify measures it.
//	Compose   Modifier.width(n) sets the child's minimum AND maximum, so the
//	          child is squeezed on both axes. A Row has no proportional shrink
//	          at all: an unweighted child is measured against the main-axis
//	          space the ones before it did not take.
//
// The Compose reading was made against foundation-layout 1.10.0, because that
// is what happened to be in a gradle cache. This module's Compose BOM resolves
// foundation-layout to 1.6.8. A pin against the wrong version is the mistake
// gobindVersion exists to prevent one file over, and it is worse here than
// there: the census is prose about a third party's arithmetic, and the reader
// has no way to tell a paragraph that was checked from one that was true two
// releases ago.
//
// # What makes it checkable now
//
// Two things that were missing, and neither is a network call at test time:
//
//	the version    android/app/build.gradle declares a composeLayoutSources
//	               configuration for the sources jar, and the version it uses
//	               is derived here rather than trusted: the BOM's own pom is in
//	               the gradle cache, it lists foundation-layout, and that
//	               listing is what the gradle file's number is held to.
//
//	the source     `./gradlew :app:fetchComposeLayoutSources` puts the sources
//	               jar in the cache once, and the claims below are then read
//	               out of it. Machines that have never run it skip that half
//	               with the command in the failure text — which is the honest
//	               state for a check whose subject has to be downloaded, and is
//	               why the version half is separate: THAT one runs everywhere
//	               the BOM's pom is cached, which is every machine that has
//	               ever built this app.
const composeLayoutArtifact = "foundation-layout-android"

// The BOM coordinate and the sources version, as android/app/build.gradle
// spells them. Both are single-line declarations in a Groovy file this package
// does not otherwise parse.
var (
	composeBOMCoord    = regexp.MustCompile(`androidx\.compose:compose-bom:([0-9.]+)`)
	composeSourcesVers = regexp.MustCompile(`ext\.composeLayoutVersion = '([0-9.]+)'`)
)

// bomEntry matches one dependencyManagement entry of the BOM's pom. The pom is
// XML that lists a few hundred artifacts as three-line blocks, so a regexp over
// the artifact this file cares about is the whole parse — an XML decoder would
// be more machinery for the same one string.
var bomEntry = regexp.MustCompile(
	`<artifactId>foundation-layout</artifactId>\s*<version>([0-9.]+)</version>`)

var appGradle = nativeFile("android", "app", "build.gradle")

// The sources jar's version must be what the BOM resolves.
//
// This is the half that runs on any machine that has ever built the Android
// app, because the BOM's pom is cached by that build whether or not anyone has
// asked for sources. It is what stops the census going stale in the way it
// already had: a BOM bump moves foundation-layout, and nothing else in this
// repository would notice.
func TestTheComposeSourcesAreTheVersionTheBOMResolves(t *testing.T) {
	gradle := readNative(t, appGradle)

	bom := oneMatch(t, appGradle, gradle, composeBOMCoord, "the Compose BOM coordinate")
	sources := oneMatch(t, appGradle, gradle, composeSourcesVers,
		"ext.composeLayoutVersion, the version of the sources jar")
	if bom == "" || sources == "" {
		return
	}

	pom := gradleCachedFile(t, "androidx.compose", "compose-bom", bom,
		"compose-bom-"+bom+".pom")
	if pom == "" {
		t.Skipf("compose-bom %s is not in this machine's gradle cache, so the "+
			"version foundation-layout resolves to cannot be derived here. Any "+
			"`./gradlew` run against android/ populates it.", bom)
	}
	raw, err := os.ReadFile(pom)
	if err != nil {
		t.Fatalf("reading %s: %v", pom, err)
	}
	m := bomEntry.FindStringSubmatch(string(raw))
	if m == nil {
		t.Fatalf("%s lists no foundation-layout. This test derives the version out "+
			"of the BOM's dependencyManagement block; if the artifact was renamed, "+
			"the census's Compose row is about a module that no longer exists.", pom)
	}
	if m[1] != sources {
		t.Errorf("compose-bom %s resolves foundation-layout to %s, and "+
			"android/app/build.gradle asks for sources %s.\n\n"+
			"The census's Compose row is a reading of foundation-layout's source, and "+
			"the whole point of deriving the version is that the source read is the "+
			"source built against. Move ext.composeLayoutVersion to %s, run "+
			"`./gradlew :app:fetchComposeLayoutSources`, and re-read Size.kt and "+
			"RowColumnMeasurementHelper.kt against it — the check below is what says "+
			"whether the paragraph still holds.",
			bom, m[1], sources, m[1])
	}
}

// And the two claims the census's Compose row makes, read out of that source.
//
// Both are load-bearing sentences in docs/platforms/native.md, and both are
// about somebody else's code, which is exactly the kind of claim that rots
// silently: nothing in this repository compiles against them and no test could
// notice a Compose release changing either one.
//
//	Modifier.width sets a MAXIMUM   this is the whole of the Compose
//	                                divergence. The other three targets impose
//	                                a minimum and let the child overflow.
//
//	a Row has no proportional       this is why core.FlexShrink's fractional
//	shrink                          factors mean nothing here, and why
//	                                Modifier.pinMainAxis exists for the zero.
func TestTheComposeCensusClaimsAreWhatTheSourceSays(t *testing.T) {
	gradle := readNative(t, appGradle)
	version := oneMatch(t, appGradle, gradle, composeSourcesVers,
		"ext.composeLayoutVersion, the version of the sources jar")
	if version == "" {
		return
	}
	jar := gradleCachedFile(t, "androidx.compose.foundation", composeLayoutArtifact,
		version, composeLayoutArtifact+"-"+version+"-sources.jar")
	run, fail, why := composeSourcesVerdict(jar != "", os.Getenv(composeSourcesEnv))
	if !run {
		// Named on the way out either way. A machine that cannot read the
		// source is the normal case and not an error, but it is also the case
		// where this check is doing nothing, and a silent nothing is what
		// android/verify's step exists to end.
		if fail {
			t.Fatalf("foundation-layout %s: %s", version, why)
		}
		t.Skipf("foundation-layout %s: %s", version, why)
	}

	for _, claim := range []struct {
		file string
		// anchor cuts the declaration the claim is about; want is what the
		// census says is in it.
		anchor string
		want   []string
		why    string
	}{
		{
			file:   "Size.kt",
			anchor: "fun Modifier.width(width: Dp)",
			want:   []string{"minWidth = width", "maxWidth = width", "enforceIncoming = true"},
			why: "the census says Modifier.width sets the child's minimum and maximum " +
				"alike, and that a maximum is what the other three targets do not " +
				"impose. Without the maximum, Compose's row in the census is the same " +
				"as everyone else's and the whole paragraph is describing nothing",
		},
		{
			file:   "RowColumnMeasurementHelper.kt",
			anchor: "// First measure children with zero weight.",
			want:   []string{"mainAxisMax - fixedSpace"},
			why: "the census says a Row measures each unweighted child against the " +
				"main-axis space the ones before it did not take — no factor, no " +
				"proportion, just what is left. That is why core.FlexShrink's " +
				"fractional values have nothing to mean on this target and why " +
				"Modifier.pinMainAxis (Renderer.kt) is the only shrink declaration " +
				"Compose can honour",
		},
	} {
		src := jarEntry(t, jar, claim.file)
		if src == "" {
			continue
		}
		at := strings.Index(src, claim.anchor)
		if at < 0 {
			t.Errorf("foundation-layout %s: %s no longer contains %q, so the census's "+
				"claim about it cannot be checked. Re-read the file: %s",
				version, claim.file, claim.anchor, claim.why)
			continue
		}
		// A window rather than the whole file, so a phrase that happens to
		// appear elsewhere in a 40KB source file cannot stand in for the
		// declaration this claim is about. 1500 bytes is chosen against the
		// longer of the two subjects: the zero-weight measure branch reaches
		// its constraint arithmetic about 850 bytes past the comment that opens
		// it, and the SizeElement literal is inside 150. A window this size can
		// still only reach a few declarations either way, which is the property
		// that matters — it is a bound, not a measurement.
		window := src[at:min(at+1500, len(src))]
		for _, want := range claim.want {
			if !strings.Contains(window, want) {
				t.Errorf("foundation-layout %s: %s's %q no longer contains %q.\n\n%s",
					version, claim.file, claim.anchor, want, claim.why)
			}
		}
	}
}

// oneMatch pulls a single captured group out of a file, reporting rather than
// returning on a miss: a derivation that silently found nothing would make
// every comparison below it pass against an empty string.
func oneMatch(t *testing.T, file, src string, re *regexp.Regexp, what string) string {
	t.Helper()
	m := re.FindStringSubmatch(src)
	if m == nil {
		t.Errorf("%s no longer spells %s where this test looks for it (%s). If it "+
			"moved, re-point this; if it was removed, the census's Compose row is "+
			"back to being prose about an unknown version.", file, what, re)
		return ""
	}
	return m[1]
}

// gradleCachedFile finds one artifact in the local gradle cache, or returns ""
// when it is not there.
//
// The cache interposes a content hash between the version and the file
// (modules-2/files-2.1/<group>/<name>/<version>/<sha1>/<file>), which is why
// this globs rather than joins: the hash is not derivable from anything this
// repository knows, and it changes when the artifact is republished.
//
// Absence is a "" rather than a fatal because the two callers want different
// things from it — one skips, one would rather say which command fills the gap
// — and neither is an error in the code being checked.
func gradleCachedFile(t *testing.T, group, name, version, file string) string {
	t.Helper()
	home := os.Getenv("GRADLE_USER_HOME")
	if home == "" {
		userHome, err := os.UserHomeDir()
		if err != nil {
			t.Skipf("no home directory to find a gradle cache in: %v", err)
		}
		home = filepath.Join(userHome, ".gradle")
	}
	matches, err := filepath.Glob(filepath.Join(
		home, "caches", "modules-2", "files-2.1", group, name, version, "*", file))
	if err != nil || len(matches) == 0 {
		return ""
	}
	return matches[0]
}

// jarEntry reads one file out of a sources jar by base name.
//
// By base name and not by full path: the jar lays its sources out under source
// set roots (commonMain/, androidMain/, jvmMain/) and which root a file lives
// in is a detail of how the artifact was assembled rather than anything the
// census claims. The names asked for here are unique within the jar; a second
// match would mean the layout changed enough that the claim needs re-reading
// anyway, so it is reported rather than resolved.
func jarEntry(t *testing.T, jar, name string) string {
	t.Helper()
	r, err := zip.OpenReader(jar)
	if err != nil {
		t.Errorf("opening %s: %v", jar, err)
		return ""
	}
	defer r.Close()

	var found string
	for _, f := range r.File {
		if filepath.Base(f.Name) != name {
			continue
		}
		if found != "" {
			t.Errorf("%s holds more than one %s. The census's claims are about one "+
				"declaration each, and a jar with two files of this name no longer "+
				"says which one was read.", jar, name)
			return ""
		}
		rc, err := f.Open()
		if err != nil {
			t.Errorf("opening %s in %s: %v", f.Name, jar, err)
			return ""
		}
		raw, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			t.Errorf("reading %s in %s: %v", f.Name, jar, err)
			return ""
		}
		found = string(raw)
	}
	if found == "" {
		t.Errorf("%s holds no %s. Either the sources jar's layout changed or the "+
			"declaration the census reads moved to another file; either way the "+
			"claim about it is now unchecked.", jar, name)
	}
	return found
}

// The gradle cache is the one input to this pass that a machine may simply not
// have, and how that is reported is a decision of its own.
//
// # Why it is a decision and not an `if`
//
// The census's source half is the check most likely to catch an androidx
// change and the least likely to run: it needs a sources jar that no build
// fetches on its own, so a machine that has only ever built the app skips it —
// silently, because `go test` prints a skip only under -v. That is the honest
// state of a check whose subject has to be downloaded, and it is also exactly
// the shape of a check that quietly stops existing.
//
// So the skip is a value rather than a control-flow accident. Three things
// consume it:
//
//	go test ./...        skips, as before, and now with a reason that names
//	                     the fetch command rather than describing the cache.
//	android/verify       runs this check by name and prints the verdict as its
//	                     own step, so an Android pass that read nothing SAYS it
//	                     read nothing, beside the other SKIPs.
//	GRMOB_COMPOSE_SOURCES=required
//	                     turns the skip into a failure, for a machine that has
//	                     been set up to have the sources and would rather hear
//	                     about it than be quietly excused.
//
// This is the same shape as gate.sh's jvm_harness_verdict and browser.mjs's
// startupVerdict: a function of values, so its arms can be reached without
// owning a machine in each state.
const composeSourcesEnv = "GRMOB_COMPOSE_SOURCES"

// composeSourcesRequired is the one value the variable takes. Spelled out
// rather than "any non-empty value is truthy" so that a typo is a failure that
// names itself instead of a setting that silently did nothing.
const composeSourcesRequired = "required"

// composeSourcesVerdict decides whether the source half runs, and what to say
// when it does not.
//
// cached is whether the sources jar is in this machine's gradle cache; setting
// is GRMOB_COMPOSE_SOURCES as the environment spells it.
func composeSourcesVerdict(cached bool, setting string) (run, fail bool, why string) {
	const fetch = "    cd android && ./gradlew :app:fetchComposeLayoutSources\n\n"

	switch setting {
	case "", composeSourcesRequired:
	default:
		// Refused rather than treated as "not required", because the two
		// spellings a reader would reach for — "1" and "yes" — would otherwise
		// disable the very thing they were typed to enable.
		return false, true, fmt.Sprintf("%s is set to %q, which is not a value it takes. "+
			"The only one is %q; unset it to let a machine without the sources skip.",
			composeSourcesEnv, setting, composeSourcesRequired)
	}
	if cached {
		return true, false, "read out of the sources jar in this machine's gradle cache"
	}
	if setting == composeSourcesRequired {
		return false, true, "its sources are not in this machine's gradle cache, and " +
			composeSourcesEnv + "=" + composeSourcesRequired + " says this machine is " +
			"one that should have them.\n\n" + fetch +
			"fetches them once. Unset the variable to go back to skipping."
	}
	return false, false, "its sources are not in this machine's gradle cache, so the " +
		"census's Compose row cannot be read here.\n\n" + fetch +
		"fetches them once; it is a network call, which is why this skips rather than " +
		"fails. The version check above runs either way, and android/verify prints this " +
		"line as a step of its own so the gap is visible rather than merely honest."
}

// And the four states it can be in, none of which this machine can be put into
// by running the tests.
func TestTheComposeSourcesVerdictNamesEveryState(t *testing.T) {
	for _, c := range []struct {
		what      string
		cached    bool
		setting   string
		run, fail bool
		mentions  string
	}{
		{what: "cached, unset", cached: true, run: true,
			mentions: "gradle cache"},
		{what: "cached, required", cached: true, setting: composeSourcesRequired, run: true,
			mentions: "gradle cache"},
		{what: "absent, unset", cached: false,
			mentions: "fetchComposeLayoutSources"},
		{what: "absent, required", cached: false, setting: composeSourcesRequired, fail: true,
			mentions: "fetchComposeLayoutSources"},
		{what: "a spelling the variable does not take", cached: true, setting: "1", fail: true,
			mentions: composeSourcesRequired},
	} {
		run, fail, why := composeSourcesVerdict(c.cached, c.setting)
		if run != c.run || fail != c.fail {
			t.Errorf("%s: run=%v fail=%v, want run=%v fail=%v", c.what, run, fail, c.run, c.fail)
		}
		if !strings.Contains(why, c.mentions) {
			t.Errorf("%s: the verdict does not mention %q, so a reader is told the state "+
				"without being told what to do about it:\n%s", c.what, c.mentions, why)
		}
		// A verdict that both runs and fails would run the check and then
		// refuse its result; one that does neither would be the silent nothing
		// this exists to end.
		if run && fail {
			t.Errorf("%s: the verdict says to run the check AND to fail", c.what)
		}
	}
}
