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
//	               configuration for the sources jar, and names no version for
//	               it: the Compose BOM is on that configuration and resolves it,
//	               the same way it resolves every other Compose artifact the app
//	               builds against. So the source read cannot be a different
//	               release from the source built against — not because a test
//	               compares two numbers, but because there is one number.
//
//	the source     `./gradlew :app:fetchComposeLayoutSources` puts the sources
//	               jar in the cache once, and the claims below are then read
//	               out of it. Machines that have never run it skip that half
//	               with the command in the failure text — which is the honest
//	               state for a check whose subject has to be downloaded.
//
// # What this file derives, and why it derives the same thing gradle does
//
// The checks below need the version to find the jar, and they get it the way
// gradle does: out of the BOM's own pom, which any `./gradlew` run against
// android/ leaves in the cache. That is a second derivation of one fact rather
// than a second spelling of it — if it disagreed with gradle's, the jar it
// looked for would not be there and the check would say so, where two spellings
// of a version agree right up until somebody bumps one.
const composeLayoutArtifact = "foundation-layout-android"

// The BOM coordinate, as android/app/build.gradle spells it. A single-line
// declaration in a Groovy file this package does not otherwise parse.
var composeBOMCoord = regexp.MustCompile(`androidx\.compose:compose-bom:([0-9.]+)`)

// The sources coordinate, with whatever follows the artifact name captured.
//
// What is being looked at is the part after `foundation-layout-android`: a bare
// closing quote is a coordinate with no version, which is the shape that lets
// the BOM resolve it, and anything else is a version written back in.
var composeSourcesCoord = regexp.MustCompile(
	`androidx\.compose\.foundation:foundation-layout-android([^'"]*)['"]`)

// bomEntry matches one dependencyManagement entry of the BOM's pom. The pom is
// XML that lists a few hundred artifacts as three-line blocks, so a regexp over
// the artifact this file cares about is the whole parse — an XML decoder would
// be more machinery for the same one string.
var bomEntry = regexp.MustCompile(
	`<artifactId>foundation-layout</artifactId>\s*<version>([0-9.]+)</version>`)

var appGradle = nativeFile("android", "app", "build.gradle")

// The sources jar has no version of its own; the BOM gives it one.
//
// # What this replaced
//
// android/app/build.gradle used to carry `ext.composeLayoutVersion = '1.6.8'`
// and this test compared it with the BOM's pom. The number was right and the
// arrangement was not: a reader of the gradle file saw two versions and no
// mechanism, and the only thing making them agree was a Go test in another
// tree. The fetch now resolves through the BOM like every other Compose
// artifact in that file, so there is one version and it is the app's.
//
// # Why the shape still needs a check, when gradle is the one enforcing it
//
// Most of the shape does not. Take the platform off the configuration, or move
// `transitive = false` onto the configuration instead of onto the dependency,
// or go back to `:sources@jar` — and `./gradlew :app:fetchComposeLayoutSources`
// fails immediately and says why. Those are loud.
//
// Writing a version back into the coordinate is the one change that is not. It
// resolves perfectly, the jar is the right jar today, and the derivation is
// simply gone — which is the state this whole arrangement was moved out of. So
// that is what is asserted, and it is asserted as text because there is nothing
// else to ask: a resolved version looks the same either way.
//
// It runs on any machine with the repository, no gradle cache required, which
// is the other half of why it is separate from the reading below.
func TestTheComposeSourcesTakeTheirVersionFromTheBOM(t *testing.T) {
	// Comments blanked, literals kept — the same mask the checks on the two
	// renderers use, and needed for the same reason. The paragraph in the
	// gradle file explaining this arrangement quotes the declaration it
	// replaced, so a check reading the file raw would find `composeLayoutVersion`
	// in the very comment that says it is gone. Groovy spells `//` and `/* */`
	// the way Swift and Kotlin do.
	gradle := maskComments(readNative(t, appGradle))

	m := composeSourcesCoord.FindStringSubmatch(gradle)
	if m == nil {
		t.Fatalf("%s no longer names %s where this looks for it (%s). If the sources "+
			"configuration was removed, the census's Compose row is back to being "+
			"prose about a version nobody stated.",
			appGradle, composeLayoutArtifact, composeSourcesCoord)
	}
	if m[1] != "" {
		t.Errorf("%s asks for the sources jar as %q — the coordinate carries %q after "+
			"the artifact name, which is a version of its own.\n\n"+
			"The BOM on the composeLayoutSources configuration is what gives this "+
			"artifact its version, so that the source the census is read from is the "+
			"source the app is built against. A version written here resolves fine and "+
			"is held to nothing, which is exactly the arrangement this replaced.",
			appGradle, m[0], m[1])
	}
	// And the platform that supplies it. Without this line the coordinate above
	// has no version at all — gradle says so loudly — but the pair is what the
	// paragraph in the gradle file is about, and half a mechanism stated is
	// worse than none.
	if !strings.Contains(gradle, "composeLayoutSources composeBom") {
		t.Errorf("%s no longer puts the Compose BOM on the composeLayoutSources "+
			"configuration. That is what resolves the versionless coordinate above; "+
			"without it the fetch cannot run at all.", appGradle)
	}
	// The old spelling, refused by name. It is the thing a reader reaching for
	// "which version is this?" would add back.
	if strings.Contains(gradle, "composeLayoutVersion") {
		t.Errorf("%s declares a composeLayoutVersion again. The version is the BOM's; "+
			"a second spelling of it is a fact that can go stale, which is what this "+
			"file used to hold to the pom instead of removing.", appGradle)
	}
}

// composeLayoutVersion derives the release the sources jar will be, the way
// gradle derives it: out of the Compose BOM's own pom.
//
// Returns "" after reporting or skipping, so a caller that got nothing has
// already said why.
func composeLayoutVersion(t *testing.T) string {
	t.Helper()

	gradle := readNative(t, appGradle)
	bom := oneMatch(t, appGradle, gradle, composeBOMCoord, "the Compose BOM coordinate")
	if bom == "" {
		return ""
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
		t.Fatalf("%s lists no foundation-layout. This derives the version out of the "+
			"BOM's dependencyManagement block; if the artifact was renamed, the "+
			"census's Compose row is about a module that no longer exists.", pom)
	}
	return m[1]
}

// And every claim this repository makes about that source, read out of it.
//
// All of them are about somebody else's code, which is exactly the kind of
// claim that rots silently: nothing here compiles against them and no test
// could notice a Compose release changing one.
//
// # Two readers, and why the table serves both
//
// docs/platforms/native.md's fixed-size census is prose, and two of its
// sentences are readings of foundation-layout:
//
//	Modifier.width sets a MAXIMUM   this is the whole of the Compose
//	                                divergence. The other three targets impose
//	                                a minimum and let the child overflow.
//
//	a Row has no proportional       this is why core.FlexShrink's fractional
//	shrink                          factors mean nothing here, and why
//	                                Modifier.pinMainAxis exists for the zero.
//
// internal/pinfixture is the same source EXECUTED — a transcription of the
// zero-weight measure loop, so that what core.FlexShrink(0) does on Compose is
// a set of numbers rather than a paragraph. Its header states the chain each
// line hangs from and names its own weakest link: somebody read androidx's loop
// and wrote it out in Go.
//
// That link was weaker than it looked. The transcription quotes four decisions
// and only ONE of them — `mainAxisMax - fixedSpace` — was read out of the jar
// by anything. The other three were quoted in a Go comment and held to nothing:
//
//	the floor on the offer      `.coerceAtLeast(0)`, which is why a child after
//	                            an overflow is offered 0 rather than a negative
//	                            number
//	the gap collapses too       spaceAfterLastNoWeight is min(spacing, what is
//	                            left), so a Row that has overflowed inserts no
//	                            spacing after a child
//	the Row overflows itself    mainAxisLayoutSize is max(content, mainAxisMin)
//	                            with no upper bound anywhere, and SizeNode
//	                            reports `layout(placeable.width, …)` unclamped
//	                            — which is what makes a pin an overflow on this
//	                            target rather than a clip
//
// So the table lists them. Each row names what would be wrong if the reading
// changed, in the words of whichever document rests on it, and MeasureCompose's
// branches now each have a row here.
//
// # The negative claim
//
// One of the four is an ABSENCE — nothing coerces the Row's own size down to
// the space it was offered — and an absence cannot be read as a substring. It
// is a notWant over a window tight enough to be the expression itself, which is
// the only honest way to say "and there is nothing else here": a clamp added
// three lines further down would be a different claim, and the row says which
// window it is speaking about.
func TestTheComposeCensusClaimsAreWhatTheSourceSays(t *testing.T) {
	version := composeLayoutVersion(t)
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
		// anchor cuts the region the claim is about; want is what a document in
		// this repository says is in it and notWant is what it says is not.
		anchor  string
		want    []string
		notWant []string
		// window bounds the region, in bytes from the anchor. Zero takes the
		// default, which is chosen against the longest positive claim; a row
		// with a notWant sets its own, because "this is absent" is only a
		// statement about a region somebody has drawn.
		window int
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
			want: []string{
				// What is left, with no factor in it. The census's sentence and
				// MeasureCompose's `remaining`.
				"mainAxisMax - fixedSpace",
				// Floored, which is what makes a child after an overflow get an
				// offer of 0 instead of a negative one. MeasureCompose's max(…, 0).
				"(mainAxisMax - fixedSpace).coerceAtLeast(0).toInt()",
				// The offer's MINIMUM is cleared. Without this a child would be
				// forced to fill what it was offered, and SizeNode's
				// constrain(base) would come back as the offer rather than as
				// min(base, offer) — the whole of what an unpinned child does.
				"mainAxisMin = 0,",
				// The branch the transcription does not carry. Every case in
				// internal/pinfixture gives the Row a definite width, so this
				// arm is unreachable there and MeasureCompose says so rather
				// than modelling it; the quote in its header is elided at this
				// line and names the elision.
				"if (mainAxisMax == Constraints.Infinity)",
				// The spacing after a child is itself clamped to what is left,
				// so an overflowing Row inserts none. MeasureCompose's
				// spaceAfterLastNoWeight.
				"spaceAfterLastNoWeight = min(",
				"(mainAxisMax - fixedSpace - placeable.mainAxisSize())",
				// And what the next child's offer is subtracted from.
				"fixedSpace += placeable.mainAxisSize() + spaceAfterLastNoWeight",
			},
			why: "the census says a Row measures each unweighted child against the " +
				"main-axis space the ones before it did not take — no factor, no " +
				"proportion, just what is left. That is why core.FlexShrink's " +
				"fractional values have nothing to mean on this target and why " +
				"Modifier.pinMainAxis (Renderer.kt) is the only shrink declaration " +
				"Compose can honour. internal/pinfixture's MeasureCompose is this " +
				"loop transcribed into Go and executed, which is where the census's " +
				"Compose column comes from",
		},
		{
			file:   "RowColumnMeasurementHelper.kt",
			anchor: "// fixedSpace contains an extra spacing after the last non-weight child.",
			want:   []string{"fixedSpace -= spaceAfterLastNoWeight"},
			window: 200,
			why: "internal/pinfixture's MeasureCompose takes the trailing spacing back " +
				"off after the loop, on the strength of this line and of the fact that " +
				"none of its children is weighted. A Row that kept it would be one gap " +
				"wider than the fixture says, in every case that has a gap",
		},
		{
			file:   "RowColumnMeasurementHelper.kt",
			anchor: "val mainAxisLayoutSize = max(",
			want: []string{
				"(fixedSpace + weightedSpace).coerceAtLeast(0).toInt()",
				"constraints.mainAxisMin",
			},
			// The absence that makes a pin an overflow.
			//
			// The window is the expression plus the room a clamp appended to it
			// would take. The expression itself ends 142 bytes past the anchor,
			// and `).coerceAtMost(constraints.mainAxisMax)` — the shortest
			// plausible way to write the thing this refuses — is 38 more; 220
			// leaves margin for a longer spelling without reaching anything that
			// mentions a main-axis maximum for another reason. A window sized to
			// the expression alone was the first attempt and it was useless: the
			// clamp lands one byte past the end of what it was measuring, so the
			// break-test that appended one passed.
			notWant: []string{"mainAxisMax", "coerceAtMost"},
			window:  220,
			why: "internal/pinfixture says a fixed-width Row whose children overflow " +
				"reports the OVERFLOWING width rather than clipping to its own, and " +
				"that this is what makes core.FlexShrink(0) an overflow on Compose " +
				"rather than a clip. The claim is that the size is raised to the Row's " +
				"minimum and never lowered to its maximum. A clamp added here would " +
				"make the pinned child spill out of a box the Row believed it fitted " +
				"inside, which is the spelling mobile/verify refuses in Renderer.kt",
		},
		{
			file:   "Size.kt",
			anchor: "val wrappedConstraints = targetConstraints.let { targetConstraints ->",
			want: []string{
				// enforceIncoming, which every fixed-size Box in both harnesses
				// gets: the declared size clamped into the incoming range, so an
				// offer of `remaining` gives min(base, remaining).
				"constraints.constrain(targetConstraints)",
				// And the report, unconstrained on the way out. This is the
				// other half of the overflow: the node hands back what it
				// measured rather than what it was offered.
				"return layout(placeable.width, placeable.height)",
			},
			window: 1600,
			why: "both harnesses mount a fixed-size Box, so this node is what every " +
				"child in internal/pinfixture is. Its unpinned branch is " +
				"min(base, offered) because of the constrain above, and the pinned " +
				"branch of Renderer.kt's Modifier.pinMainAxis is a copy of the layout " +
				"call below it — measure unbounded, report what was measured. A " +
				"constrained report here would mean the Row never overflowed",
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
		// declaration this claim is about. 1500 bytes is the default and is a
		// bound rather than a measurement: the zero-weight measure branch runs
		// to its last claimed line about 1350 bytes past the comment that opens
		// it, and the SizeElement literal is inside 150. A window this size can
		// still only reach a few declarations either way.
		//
		// Rows that state an ABSENCE set their own, because a notWant is a
		// claim about a region and a loose region would make it a claim about
		// the file.
		size := claim.window
		if size == 0 {
			size = 1500
		}
		window := src[at:min(at+size, len(src))]
		for _, want := range claim.want {
			if !strings.Contains(window, want) {
				t.Errorf("foundation-layout %s: %s's %q no longer contains %q.\n\n%s",
					version, claim.file, claim.anchor, want, claim.why)
			}
		}
		for _, notWant := range claim.notWant {
			if strings.Contains(window, notWant) {
				t.Errorf("foundation-layout %s: %s's %q has grown a %q, and this claim "+
					"is that there is none within %d bytes of the anchor.\n\n%s",
					version, claim.file, claim.anchor, notWant, size, claim.why)
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
