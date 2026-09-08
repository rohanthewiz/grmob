package verify

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"os/exec"
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

// How much room a claim's window must have past its own furthest positive
// claim, and why the answer is per row.
//
// # The direction the windows fail in
//
// The windows below are byte counts, and every one of them was derived by
// measuring foundation-layout's source as it is shipped today. That makes them
// measurements of a file this repository does not control, which is fine while
// they are generous and quietly wrong when they are not — and the way they go
// wrong has a direction. A window that has become too SHORT for a `want` drops
// the substring off its end and the row fails by name. A window that has become
// too short for a `notWant` finds nothing and passes, which is the same thing a
// correct file looks like.
//
// That is not hypothetical: the mainAxisLayoutSize row was first written with a
// window sized to the expression itself, and the break-test that appended a
// clamp to androidx's source passed — the clamp landed one byte past the end of
// what the window was measuring.
//
// So only a row that states an ABSENCE needs slack. A row with nothing but
// `want`s is already loud in the only direction it can fail.
//
// # Why the requirement is not one number
//
// It was: a single `windowSlack = 40`, and 40 was 38 rounded up — the length of
// `).coerceAtMost(constraints.mainAxisMax)`, which is the shortest plausible
// spelling of the one absence the table happened to state. That number was
// therefore about ONE row while being applied to five, and it had no way to
// stay right: a second `notWant`, about something else, refuses a different
// spelling of a different length, and nothing in the arrangement would have
// said so. The row would simply have been held to a floor derived from its
// neighbour's subject.
//
// A row states the spelling instead. `absence` is the shortest plausible way to
// write the thing that row refuses, its length is the slack that row needs, and
// the two are held together in both directions: a notWant with no absence is a
// claim with no floor, and an absence a notWant would not catch is a floor for
// a spelling this row cannot see anyway.
//
// One row states an absence today. Its window is 220 bytes, its positive claims
// end 132 in, and the spelling it refuses is 39 — so it has 88 where it needs
// 39, which is a floor that fires on a window that has stopped being a window
// rather than on ordinary drift.

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
	// valuesIn: comments blanked, literals kept, and both halves matter here.
	//
	// The comments, because the paragraph in the gradle file explaining this
	// arrangement quotes the declaration it replaced — a check reading the file
	// raw would find `composeLayoutVersion` in the very comment that says it is
	// gone. Groovy spells `//` and `/* */` the way Swift and Kotlin do.
	//
	// The literals, because every subject below IS one: a gradle coordinate is
	// a quoted string, so this is the "does it LIST this value" question and
	// not the "does it DO this" one.
	//
	// That reading used to hold by accident. maskNonCode did not know Groovy's
	// apostrophe strings, so the code-level reader left these coordinates
	// standing and answered the right question for the wrong reason — and
	// would have gone on doing so for a subject spelled with double quotes,
	// which gradle accepts equally. The scan knows both delimiters now, so this
	// call is a choice again: valuesIn keeps the literals BECAUSE they are the
	// subject, and codeIn would now blank them and find nothing.
	gradle := valuesIn(t, appGradle)

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
//
// # The skip, and what it now honours
//
// A machine with no gradle cache cannot derive anything, and that is not an
// error in the code being checked. It used to be an unconditional skip, which
// put it out of reach of GRMOB_COMPOSE_SOURCES=required — so a machine that had
// been set up to have the sources and had lost its cache was excused by the one
// switch that exists to refuse exactly that. It goes through the same verdict
// the sources half does now.
func composeLayoutVersion(t *testing.T) string {
	t.Helper()

	gradle := valuesIn(t, appGradle)
	bom := oneMatch(t, appGradle, gradle, composeBOMCoord, "the Compose BOM coordinate")
	if bom == "" {
		return ""
	}
	pom := gradleCachedFile(t, "androidx.compose", "compose-bom", bom,
		"compose-bom-"+bom+".pom")
	if pom == "" {
		why := fmt.Sprintf("compose-bom %s is not in this machine's gradle cache, so "+
			"the version foundation-layout resolves to cannot be derived here. Any "+
			"`./gradlew` run against android/ populates it.", bom)
		if os.Getenv(composeSourcesEnv) == composeSourcesRequired {
			t.Fatalf("%s\n\n%s=%s says this machine is one that should have them.",
				why, composeSourcesEnv, composeSourcesRequired)
		}
		t.Skip(why)
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
	// And gradle's own answer, where a fetch has left one. See
	// composeSourcesReceipt: this is the one thing the derivation above cannot
	// check about itself.
	checkResolutionAgrees(t, version, jar)
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
		// absence is the shortest plausible SPELLING of what this row refuses,
		// and its length is the slack this row's window has to have past its
		// own positive claims. Set exactly when notWant is: see the paragraph
		// above the table for why the requirement cannot be one number for
		// every row.
		absence string
		why     string
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
			// And what this row's window has to be able to hold: the shortest
			// way the absence above could be written back in. 38 bytes, against
			// a window whose positive claims end 142 in — so a clamp appended
			// to the expression is inside the region rather than one byte past
			// its end, which is exactly how the first version of this row
			// passed a break-test.
			absence: ").coerceAtMost(constraints.mainAxisMax)",
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
		// The row's absence and its marks, held to each other before either is
		// used. Both directions, and each one is a way the pair can rot:
		//
		//	a notWant with no absence   the row states a refusal and no floor,
		//	                            so its window is checked against nothing
		//	                            and the silent failure is back
		//	an absence with no notWant  a floor for a spelling this row has no
		//	                            mark for, which reads as a refusal and
		//	                            refuses nothing
		//	a mark the absence would    the two are meant to be one claim: the
		//	not catch                   absence is what the row refuses and the
		//	                            notWants are how it would be recognised.
		//	                            A mark that is not in the spelling means
		//	                            they have come apart.
		if (len(claim.notWant) > 0) != (claim.absence != "") {
			t.Errorf("foundation-layout %s: %s's %q states %d notWant marks and %s.\n\n"+
				"A row that refuses something states the shortest plausible SPELLING of "+
				"it, and that spelling's length is the slack its window must have past "+
				"its own positive claims. Without the pair, the requirement falls back "+
				"to whatever some other row's subject happened to need — which is what "+
				"a single windowSlack constant was.",
				version, claim.file, claim.anchor, len(claim.notWant),
				map[bool]string{
					true:  fmt.Sprintf("an absence of %q", claim.absence),
					false: "no absence",
				}[claim.absence != ""])
		}
		for _, notWant := range claim.notWant {
			if claim.absence != "" && !strings.Contains(claim.absence, notWant) {
				t.Errorf("foundation-layout %s: %s's %q refuses %q and states its "+
					"absence as %q, which does not contain it.\n\n"+
					"The two are one claim read two ways: the absence is what this row "+
					"says is not there, and the marks are how the check would recognise "+
					"it. A mark the spelling would not carry means the row is sized "+
					"against one thing and looking for another.",
					version, claim.file, claim.anchor, notWant, claim.absence)
			}
		}

		// How far into the window the furthest positive claim reaches, so the
		// window can be held to being a window rather than a coincidence.
		reach, allFound := 0, true
		for _, want := range claim.want {
			i := strings.Index(window, want)
			if i < 0 {
				allFound = false
				t.Errorf("foundation-layout %s: %s's %q no longer contains %q.\n\n%s",
					version, claim.file, claim.anchor, want, claim.why)
				continue
			}
			if end := i + len(want); end > reach {
				reach = end
			}
		}
		if allFound && claim.absence != "" && size-reach < len(claim.absence) {
			t.Errorf("foundation-layout %s: %s's %q window is %d bytes and its furthest "+
				"claim ends %d bytes in, leaving %d.\n\n"+
				"Every one of these numbers was arrived at by measuring androidx's file "+
				"as it is shipped, so a release that reformats this declaration — one "+
				"line wrapped differently, one comment grown — moves the claims down "+
				"inside a window that did not move with them. In the positive direction "+
				"that is loud: the substring falls off the end and the row fails. In the "+
				"negative direction it is silent, and it is the direction that matters, "+
				"because a notWant is a claim about a REGION: a window too short to hold "+
				"the thing it refuses finds no clamp and passes.\n\n"+
				"%d bytes is this row's floor: it is the length of %q, the shortest "+
				"plausible way to write the thing this row refuses, so a window with "+
				"less than that past its own positive claims could not see it appended "+
				"even in principle. The floor is the row's own rather than a constant "+
				"because it is a fact about what the row refuses. Widen this row's "+
				"window.",
				version, claim.file, claim.anchor, size, reach, size-reach,
				len(claim.absence), claim.absence)
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

// The receipt android/app/build.gradle's fetch leaves behind, and what it is
// for.
//
// # The one thing the derivation cannot check about itself
//
// composeLayoutVersion reads the BOM's pom the way gradle reads it. That is a
// second DERIVATION of one fact rather than a second spelling of it, and the
// argument for it is that a disagreement shows up as a jar that is not there.
// It does — and "not there" is spelled SKIP, which is the quiet answer for the
// one thing this whole arrangement exists to make loud. A platform enforced
// somewhere else, a different dependencyManagement entry winning, a classifier
// resolving to another module: each of them lands as a machine that "has not
// fetched the sources yet", on a machine that has.
//
// So `fetchComposeLayoutSources` writes down what it resolved, and this
// compares. It is not a third spelling of the version — nothing here reads a
// number a person typed; it is gradle's own resolution, in gradle's own words.
//
// # The BOM rides along, and that is what keeps the failure honest
//
// A receipt written before a BOM bump names the old version, and that is
// nobody's mistake: it means the fetch has not been re-run. Only a receipt for
// the SAME BOM can accuse the derivation, so the two situations get two
// different messages and only one of them is a failure.
//
// # And no receipt at all
//
// The third situation, and the one that used to be a silent `return`. A machine
// that has run `./gradlew` has everything the claims below need and nothing the
// derivation can be held to, so the loud answer arrived only for people who had
// already run the fetch — which is to say, only for people who were not in the
// state it protects against. receiptVerdict names it, and puts it under the
// same switch every other absence in this file is under.
const composeSourcesReceipt = "composeLayoutSources.txt"

var receiptBOM = regexp.MustCompile(`(?m)^bom=(.+)$`)
var receiptSources = regexp.MustCompile(`(?m)^sources=(.+)$`)

// The version out of a resolved sources jar's own file name. gradle writes the
// path it resolved; what is compared is the release, because that is what
// composeLayoutVersion derives and the only thing it can be wrong about.
var receiptVersion = regexp.MustCompile(
	`foundation-layout-android-([0-9.]+)-sources\.jar$`)

// receiptVerdict decides what a machine with no receipt is told.
//
// # The gap this closes
//
// The receipt is written by the fetch and by nothing else, so a machine that
// has run `./gradlew` — which populates the cache, which is all the claims
// below need — has a pom, derives a version, and is compared with nothing. That
// was the honest state and it was also a silent one: the whole point of the
// receipt is to make a wrong derivation LOUD, and the loud answer arrived only
// for people who had already run the one command that produces it.
//
// It is under the same switch every other absence in this file is. Unset, the
// state is named on the way past and named with the command that ends it; set,
// it is a failure, because GRMOB_COMPOSE_SOURCES=required is the sentence "this
// machine is one that settles these claims" and a machine that settles them
// without gradle's own answer settles half of them.
//
// The build directory is the reason this is not simply expected to exist:
// `./gradlew clean` removes the receipt and leaves the cached jar, so a machine
// that fetched last week and cleaned yesterday is in this state through no
// fault of anybody's. Re-running the fetch is the whole fix, and it is what the
// text says.
//
// A function of values, so both arms are reachable without owning a machine in
// each state — the same shape composeSourcesVerdict and importerVerdict have.
func receiptVerdict(present bool, setting string) (fail bool, why string) {
	if present {
		return false, ""
	}
	why = "no `:app:fetchComposeLayoutSources` has run here, so gradle has not " +
		"written down what it resolved and the version derived from the BOM's pom is " +
		"compared with nothing. That derivation's failure mode is silent — a version " +
		"nothing resolves to is a jar that is not in the cache, which reads as 'this " +
		"machine has not fetched the sources' on a machine that has — and the receipt " +
		"is the only thing that catches it:\n\n" +
		"    cd android && ./gradlew :app:fetchComposeLayoutSources\n"
	if setting == composeSourcesRequired {
		return true, why + "\n" + composeSourcesEnv + "=" + composeSourcesRequired +
			" says this machine is one that should settle these claims, and gradle's " +
			"own answer is half of settling them."
	}
	return false, why + "\nSet " + composeSourcesEnv + "=" + composeSourcesRequired +
		" on a machine that is supposed to, to hear about this rather than be excused."
}

// And both of its states, neither of which this machine can be put into by
// running the tests: the receipt is either there or it is not, and whichever it
// is, it is that for the whole run.
func TestTheReceiptVerdictNamesEveryState(t *testing.T) {
	for _, c := range []struct {
		what     string
		present  bool
		setting  string
		fail     bool
		mentions string
	}{
		{what: "a receipt, and the switch unset", present: true},
		{what: "a receipt, and the switch set", present: true,
			setting: composeSourcesRequired},
		{what: "no receipt, and the switch unset",
			mentions: "fetchComposeLayoutSources"},
		{what: "no receipt, and the switch set", setting: composeSourcesRequired,
			fail: true, mentions: "fetchComposeLayoutSources"},
	} {
		fail, why := receiptVerdict(c.present, c.setting)
		if fail != c.fail {
			t.Errorf("%s: fail=%v, want %v", c.what, fail, c.fail)
		}
		if c.mentions != "" && !strings.Contains(why, c.mentions) {
			t.Errorf("%s: the verdict does not mention %q, so a reader is told the "+
				"state without being told what to do about it:\n%s",
				c.what, c.mentions, why)
		}
		if c.present && why != "" {
			t.Errorf("%s: a machine with a receipt is told %q, and there is nothing to "+
				"tell it — the comparison below runs", c.what, why)
		}
		// The quiet arm still has to name the switch. An absence that does not
		// say it can be made loud is the state this whole arrangement exists to
		// move out of.
		if !c.present && !c.fail && !strings.Contains(why, composeSourcesEnv) {
			t.Errorf("%s: the note does not name %s, so a machine that is supposed to "+
				"settle this has no way to learn it can say so:\n%s",
				c.what, composeSourcesEnv, why)
		}
	}
}

// checkResolutionAgrees holds the derived version to the one gradle resolved,
// when a fetch on this machine has said what that was.
//
// version is what composeLayoutVersion derived and jar is the file the glob
// found for it, which is "" when there is none — the case this exists for.
//
// The comparison is between two VERSIONS rather than between two paths, and
// deliberately: a path is about a particular cache, and this check has to stay
// meaningful when GRADLE_USER_HOME points somewhere else — which is how the
// required arm is exercised (see TestTheRequiredArmFailsAPassRatherThanSkipping).
// The release is also the only thing the derivation produces, so it is the only
// thing it can get wrong.
func checkResolutionAgrees(t *testing.T, version, jar string) {
	t.Helper()

	path := nativeFile("android", "app", "build", composeSourcesReceipt)
	raw, err := os.ReadFile(path)
	if err != nil {
		// No fetch has been run here, so gradle has not resolved anything on
		// this machine and there is nothing to disagree with. Named on the way
		// past rather than returned from silently: see receiptVerdict.
		fail, why := receiptVerdict(false, os.Getenv(composeSourcesEnv))
		if fail {
			t.Errorf("%s: %s", path, why)
		} else {
			t.Logf("%s: %s", path, why)
		}
		return
	}
	got := string(raw)

	bom := receiptBOM.FindStringSubmatch(got)
	sources := receiptSources.FindAllStringSubmatch(got, -1)
	if bom == nil || len(sources) == 0 {
		t.Errorf("%s does not carry a bom= line and a sources= line where this looks "+
			"for them (%s, %s). It is written by :app:fetchComposeLayoutSources and read "+
			"here; if its shape changed, gradle's own answer is no longer being "+
			"compared with the version this file derives.", path, receiptBOM, receiptSources)
		return
	}

	// A receipt from a previous BOM is a stale fetch rather than a wrong
	// derivation, and saying which is the whole reason the coordinate is in it.
	wantBOM := oneMatch(t, appGradle, valuesIn(t, appGradle), composeBOMCoord,
		"the Compose BOM coordinate")
	if wantBOM == "" {
		return
	}
	if !strings.HasSuffix(strings.TrimSpace(bom[1]), ":"+wantBOM) {
		t.Logf("%s was written for %s and android/app/build.gradle now names "+
			"compose-bom %s, so gradle's answer is not about this BOM and is not "+
			"compared. Re-run:\n\n    cd android && ./gradlew "+
			":app:fetchComposeLayoutSources\n", path, strings.TrimSpace(bom[1]), wantBOM)
		return
	}

	if len(sources) != 1 {
		t.Errorf("%s names %d resolved sources jars, and the configuration is declared "+
			"non-transitive precisely so it resolves one. Every claim below is read out "+
			"of a single file.", path, len(sources))
		return
	}
	resolved := strings.TrimSpace(sources[0][1])
	m := receiptVersion.FindStringSubmatch(resolved)
	if m == nil {
		t.Errorf("%s names %q as the resolved sources jar, and this cannot read a "+
			"version out of that name (%s). The receipt is gradle's own answer to the "+
			"question composeLayoutVersion derives; a name this does not recognise means "+
			"the two are no longer being compared at all.", path, resolved, receiptVersion)
		return
	}
	if m[1] == version {
		return
	}
	t.Errorf("gradle resolved foundation-layout %s\n\n    %s\n\nand this file derived "+
		"%q from the BOM's pom, which %s.\n\n"+
		"The derivation is meant to be a second way of arriving at gradle's answer, "+
		"and its failure mode is silent: a version nothing resolves to is a jar that "+
		"is not in the cache, which reads as 'this machine has not fetched the "+
		"sources' on a machine that has. That is why the fetch writes down what it "+
		"resolved.\n\n"+
		"The BOM is the same one this receipt was written for, so this is not a stale "+
		"fetch. The derivation in composeLayoutVersion no longer matches how gradle "+
		"resolves this configuration, and every claim below is being read out of the "+
		"wrong release — or out of nothing at all.",
		m[1], resolved, version,
		map[bool]string{
			true:  fmt.Sprintf("is the file %s", jar),
			false: "is a jar this machine's cache does not hold",
		}[jar != ""])
}

// And the required arm, run as a pass rather than as a table of values.
//
// # What was missing
//
// composeSourcesVerdict is a function of values so that its arms can be reached
// without owning a machine in each state, and
// TestTheComposeSourcesVerdictNamesEveryState reaches all five. That settles
// what the verdict SAYS. What nothing settled is whether the census check acts
// on it: GRMOB_COMPOSE_SOURCES=required is a switch whose whole purpose is to
// turn a skip into a failure, and the arm that does the turning had never run.
// A `t.Skipf` written where a `t.Fatalf` belongs passes that unit test and
// leaves the switch inert — silently, since a machine with the switch set and
// the sources present takes the same arm as one with the switch unset.
//
// # Why a subprocess
//
// The state the arm is about is a machine WITHOUT the sources, and this machine
// may well have them. GRADLE_USER_HOME is the one input that decides, and it is
// read at the point of use — so a cache with the pom in it and no sources jar
// is a directory this test can build, and the check is then run against it for
// real. In-process would mean setting an environment variable a test in the
// same binary reads, and reading a t.Skip out of a helper rather than out of a
// pass.
//
// Three states, and the pair is what makes each one mean anything: the same
// doctored cache, with and without the variable, has to produce a failure and a
// skip. A check that failed either way would be one this arm had nothing to do
// with.
func TestTheRequiredArmFailsAPassRatherThanSkipping(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skipf("no go on PATH to run the census check as a pass with: %v", err)
	}

	// The BOM's pom, which composeLayoutVersion needs before the sources half
	// is reached at all. Taken from this machine's real cache, because a
	// hand-written one would be a fixture of somebody else's file format.
	gradle := valuesIn(t, appGradle)
	bom := oneMatch(t, appGradle, gradle, composeBOMCoord, "the Compose BOM coordinate")
	if bom == "" {
		return
	}
	pomName := "compose-bom-" + bom + ".pom"
	pom := gradleCachedFile(t, "androidx.compose", "compose-bom", bom, pomName)
	if pom == "" {
		t.Skipf("compose-bom %s is not in this machine's gradle cache, so a cache "+
			"holding the pom and no sources jar cannot be built from it. Any "+
			"`./gradlew` run against android/ populates it.", bom)
	}
	raw, err := os.ReadFile(pom)
	if err != nil {
		t.Fatalf("reading %s: %v", pom, err)
	}

	// A gradle cache with the BOM in it and nothing else. The hash directory is
	// part of the layout gradleCachedFile globs past; any name will do, and a
	// literal one says so.
	home := t.TempDir()
	dir := filepath.Join(home, "caches", "modules-2", "files-2.1",
		"androidx.compose", "compose-bom", bom, "0000000000000000000000000000000000000000")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("building a gradle cache to run against: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, pomName), raw, 0o644); err != nil {
		t.Fatalf("writing the BOM's pom into it: %v", err)
	}

	// An empty one as well, for the arm one step earlier: with no pom there is
	// no version to look a jar up by, and that skip is under the same switch.
	empty := t.TempDir()

	for _, c := range []struct {
		what     string
		home     string
		required bool
		// wantFail is whether the pass must fail; mentions is what its output
		// has to say, because a failure for another reason would satisfy the
		// first half on its own.
		wantFail bool
		mentions string
	}{
		{
			what: "the sources absent, and the switch unset", home: home,
			wantFail: false, mentions: "SKIP",
		},
		{
			what: "the sources absent, and the switch set", home: home, required: true,
			wantFail: true, mentions: "fetchComposeLayoutSources",
		},
		{
			what: "no gradle cache at all, and the switch set", home: empty, required: true,
			wantFail: true, mentions: "compose-bom",
		},
	} {
		cmd := exec.Command("go", "test", "./mobile/verify/",
			"-run", "^"+censusTest+"$", "-count=1", "-v")
		cmd.Dir = filepath.Join("..", "..")
		cmd.Env = append(os.Environ(), "GRADLE_USER_HOME="+c.home)
		if c.required {
			cmd.Env = append(cmd.Env, composeSourcesEnv+"="+composeSourcesRequired)
		} else {
			// Inherited from whatever ran this, and the pair is the whole
			// point: the same cache has to answer differently.
			cmd.Env = append(cmd.Env, composeSourcesEnv+"=")
		}
		out, err := cmd.CombinedOutput()

		if (err != nil) != c.wantFail {
			t.Errorf("%s: the census check %s, and it is supposed to %s.\n\n"+
				"%s=%s exists to turn a machine's missing sources from a skip into a "+
				"failure. The verdict function's arms are covered by a table one test "+
				"up; this is whether the check acts on them.\n\n%s",
				c.what,
				map[bool]string{true: "failed", false: "passed"}[err != nil],
				map[bool]string{true: "fail", false: "skip"}[c.wantFail],
				composeSourcesEnv, composeSourcesRequired, out)
			continue
		}
		if !strings.Contains(string(out), c.mentions) {
			t.Errorf("%s: the census check's output does not mention %q, so a reader is "+
				"told the state without being told what happened:\n\n%s",
				c.what, c.mentions, out)
		}
	}
}

// censusTest is the check the pass above runs, named once so the -run pattern
// and the check cannot drift apart.
const censusTest = "TestTheComposeCensusClaimsAreWhatTheSourceSays"

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
//	go test ./...        a fourth consumer, and the one that was missing: the
//	                     arm that does the TURNING had never run outside the
//	                     table below. A `t.Skipf` written where a `t.Fatalf`
//	                     belongs passes that table and leaves the switch inert,
//	                     silently — a machine with the switch set and the
//	                     sources present takes the same arm as one without it.
//	                     TestTheRequiredArmFailsAPassRatherThanSkipping builds a
//	                     gradle cache holding the BOM's pom and no sources jar
//	                     and runs the census check against it, with the switch
//	                     and without.
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
