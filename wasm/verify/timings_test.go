package main

import (
	"fmt"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

// The machine every wall-clock number in this package's prose was taken on.
//
// # Why a number here needs a machine attached and the others do not
//
// Everything else recorded in this package is re-derived on every run.
// affordedMeasuredOn keeps eighty names and four counts and the census over
// them is re-walked; foldMeasuredOn keeps the fold's counts and names the
// Unicode build they are a reading of, so a count that moved is a count about
// the DATA. Either way a number that moved is a finding.
//
// The timings cannot work that way. `685ms`, `112ms`, `2530.0us`, `5.5s`,
// `128ms`: every one of them is a number somebody took once and typed in, and
// a number in a note is a number that has already moved.
//
// It cannot become an arm either, and not for want of trying. A wall clock is
// a reading of the machine, the load on it, the Go version's scheduler and
// whatever else was compiling at the time; an assertion over one would fail on
// a busy laptop and on every CI runner, and a test that fails for reasons
// nobody can act on is a test people learn to ignore. That is a worse outcome
// than an unattributed number.
//
// What a timing CAN carry is where it came from. `go test ./wasm/verify`
// spread over 2.49–2.59s across seven runs on one idle machine when that
// sentence was first written, roughly 4% wide by itself — so a reader holding
// a 2.7s run against a 2.54s written down somewhere has nothing to reason
// with: the difference is inside one machine's own spread, or it is a
// regression, or it is a different computer, and the number alone
// distinguishes none of them. Those two figures are kept as the illustration
// they were rather than re-taken, because what they demonstrate is the WIDTH
// of one machine's spread and that has not changed; `wholeFile` below is what
// this package actually costs now, and it is wider than that.
//
// How much wider is deliberately not written here, and the reason is worth the
// three lines it takes.
//
// This said "3% wide" for several sessions. That was true of the band as it
// stood when somebody computed it — 2.88–2.97s — and not of the band under it,
// which has been widened four times since. A derivable figure copied into
// prose, which is the defect class this repository spends most of its censuses
// on, arriving in the record that exists to stop it.
//
// The sentence that REPLACED it carried the corrected percentage, and that
// percentage was stale one iteration later: the floor moved again in the same
// session, found by the arm that places this figure. So the number is gone
// rather than corrected. The verdict line prints the width of whichever band
// it compared against, which is the one place it cannot drift. See
// bandPlacement.
//
// The spread is why the recorded
// figures are RANGES and the machine is why there is a record at all; the arm
// below says which computer this run is standing on when it is not this one.
//
// # Why this is its own file
//
// It began inside themenearmiss_test.go, named for that file's `afforded*`
// family, describing "every wall-clock number in this file". That was already
// untrue when it was written: inkglyph_test.go states the astral fold walk's
// cost twice, and it is as much a reading of this machine as any of
// themenearmiss's. A record cited by two files that lives inside one of them
// reads as that file's property, and the next person adding a timing to a
// third does not find it.
//
// # Why the fields are the ones a program can read
//
// A model name is what a person wants and what nothing can check, so the
// record carries both: `machine` for the reader, and four fields the runtime
// answers for itself. A mismatch in those is a machine difference stated as a
// fact rather than guessed at from a number that looks wrong.
//
// # What a re-taking also moves, which attribution alone does not cover
//
// Every wall clock in this package's prose says "where verifyTimingsTakenOn
// was taken". That is the discipline this record exists for and it turns out
// not to be sufficient, because the reference MOVES: when `wholeFile` was
// re-taken and went from `2.78–2.93s` to `2.88–2.97s`, every sentence citing
// this record silently began claiming to be from a taking it was not from. One of
// the figures had moved with it and two had not, and nothing said which.
//
// A wall clock cannot be an arm — that is the argument above and it still
// holds — so what there is instead is a list, and the list is short because
// most of these are now fields:
//
//	walkEnumerate, walkRead, walkParse   fields. Two failure messages in
//	                                     repowalks_test.go read them, so a
//	                                     re-taking is one edit here
//	repowalks_test.go's header           nothing to move any more. It said
//	                                     "1.18–1.40s of a 2.88–2.97s
//	                                     package" — four walkParse, two
//	                                     walkRead and one walkEnumerate
//	                                     summed against `wholeFile` — and it
//	                                     was the last entry on this list that
//	                                     a re-taking moved by hand. It now
//	                                     states the PROPORTION and names the
//	                                     fields, because prose cannot read a
//	                                     field but a reader can, and a
//	                                     proportion does not drift
//	the 0.01s below                      the walk census's own cost, and the
//	                                     "ten of forty-one" beside it
//	the besides row in repowalks         0.010s, and it carries its own method
//
// The general lesson is the one this repository keeps arriving at from
// different directions: a figure quoted in two places is a copy, and
// attributing the copy to the original does not make it one thing. Moving it
// into the original does.
//
// # Re-taking it, and CHECKING it without re-taking it
//
// Set GRMOB_BAND_VERDICT=required and the run compares its own reading
// against the band and says where it fell. That is how to find out whether a
// figure here has gone stale, and it is cheaper than re-taking one: it is
// the command below with four more words in front of it.
//
//	GRMOB_BAND_VERDICT=required go test -count=1 -v ./wasm/verify
//
// On a run of THIS package alone — `go test ./...` runs package binaries in
// parallel and reports 0.3s more, which is a band's whole width. See
// wholefileband_test.go for why that cannot be detected from inside and is
// asked for instead.
//
// And not directly after something heavy. The first reading taken this way,
// immediately after `go test -race ./...` and the three verify scripts, came
// in at 3.176s and reported OVER; six runs later, with nothing else running,
// it read 2.684–2.741s. Which is the verdict's own advice working — one
// reading over a ceiling is not a regression, take it several times — and is
// the reason that sentence is in the message.
//
// One command per field, written out rather than abbreviated. The three walk
// depths below were fragments for several sessions — no `go test`, no package
// path, and two of the three test names cut off at an ellipsis — which is the
// fault internal/themehistory's wholeRun field names in its own history: "a
// recipe rather than a command, and it made this the only figure in either
// record a person could not re-derive by running something".
//
//	wholeFile
//	  go test -count=1 ./wasm/verify
//	  GRMOB_PACKAGE_READING=<that figure> go test -count=1 -v \
//	      -run TestTheFigureGoTestPrinted ./wasm/verify
//
//	wholeFileInProcess
//	  GRMOB_BAND_VERDICT=required go test -count=1 -v ./wasm/verify
//
// Two lines for the first field, because the figure being placed is the first
// command's OWN wall clock and only the `go` command can see it — so the second
// hands it back to the code that knows the band. See
// TestTheFigureGoTestPrintedForThisPackageIsPlacedInItsBand for why it is not
// a nested run.
//
// That last one is the verdict command above, read for its reading rather
// than for its verdict — the line says what this run took before it says
// where that fell. This field had NO entry here for six iterations after it
// was added, which is the failure a table of takings has: a field arrives
// with a band and the method it was taken by stays in whoever's head took
// it. Nine of the ten band fields across the two records had one; this was
// the tenth.
//
// It is held by an arm now rather than by whoever notices:
// checkEveryBandFieldHasATakingCommand reads this table out of this comment
// and every band out of the literal below, and says which fields each one has
// that the other does not. It was measured on the way in — ten band fields,
// ten entries, zero findings — which is the residue a rule is normally
// declined for, and is not the measurement that matters here: the defect it
// covers was real, it was this field, and it stood for six iterations because
// a table kept by hand goes stale when somebody adds a field rather than when
// anything changes.
//
//	foldWalk
//	  go test -count=1 -run TestHowWideTheNarrowerFold ./wasm/verify
//
// And the three walk depths, which are the wall clock of ONE TEST and not of
// the package — read off `-v`'s own `--- PASS:` line, seven runs apiece:
//
//	walkEnumerate
//	  go test -count=1 -v -run TestTheCitationSkipsGitAlreadyMakes ./wasm/verify
//
//	walkRead
//	  go test -count=1 -v -run TestEveryGitListingInAScriptAsksForNulSeparatedPaths ./wasm/verify
//
//	walkParse
//	  go test -count=1 -v -run TestTheDottedVersionParsersAreTheOnesTheReasonCovers ./wasm/verify
//
// The command matters more here than anywhere else in this table, because
// these three were once written with no method beside them and a re-taking
// that reconstructed the work by hand — git ls-files, a stat, a read, a parse
// — came back at a FIFTH of them. What the recorded figures include and that
// reconstruction did not is the rest of what each test does: citingFiles
// scans every file's text for citations on the way past, and the test then
// asks its own question of what came back. The depth is the shape of the
// walk, not the whole of the cost, and the figure is the whole of the cost.
//
// There is no third line, and there was: this table used to offer `go test
// -bench . -run '^$' ./wasm/verify` for "the per-walk ones" — the 96.7us,
// 7.4us and 2530.0us in themenearmiss_test.go's account of what affordedEachSet
// replaced. That command does nothing. The benchmarks it names were written to
// take those three numbers and removed once they had been taken, so the
// re-taking instruction outlived the thing it instructed. Those three are in
// the same position as the 128ms below and for the same reason: they are a
// record of a change that was already made, not a property anything asserts.
//
// Anything re-taken here is re-taken WITH this record: a run on another machine
// that updates a timing and leaves the machine alone has put the same
// unattributed number back, one commit later.
//
// # And a band is taken from SITTINGS, not from runs
//
// Three bands in this repository were re-taken in one evening and every one
// of them was contradicted by the next sitting, because every one was set
// from runs taken back to back. Consecutive runs of a package are
// correlated: a dozen of them sample about half the spread of the figure and
// look tight doing it.
//
// So the count beside a range is the weaker half of its method and the
// number of separated sittings is the stronger one. Ten runs at three
// different times beats fifty in a row. The argument and the measurement
// are at wholeFileInProcess, which is the field that cost the most to learn
// it on.
//
// # And it applies to the AGGREGATE figures only, which was tested
//
// The rule above was written as though it governed every band here. It does
// not. The three walk depths were taken across three sittings with the whole
// verification suite run in between, twenty-one runs of walkParse, and the
// spread did not widen — it is two values, 0.19s and 0.20s, and 0.01s is the
// resolution of the line they are read off. The other two depths behave the
// same way.
//
// What separates them is what is being timed, not whether it has a band. A
// single CPU-bound test over a warm tree repeats to the limit of the
// reporting for twenty-odd runs at a time. A whole package of most of a
// thousand tests, with node and git subprocesses in it, spreads 8% and does
// so continuously.
//
// But "reproducible" is the wrong word for the first of those and this
// record said it for one iteration before thirty-two runs took it back. The
// middle of walkParse is two values and its tail reaches 0.24s, which is
// where its ceiling already was. See that field.
//
// # Three rules about ends, which cost five re-takings to arrive at
//
// Every band in this repository was re-taken at least once in one evening,
// several of them twice, and every re-taking that went wrong went wrong the
// same way.
//
//	widen, almost never narrow    a ceiling nothing has reached costs a
//	                              reader nothing. A ceiling somebody
//	                              tightened costs a false verdict the first
//	                              time the tail shows up, and walkParse's
//	                              did, five runs after it was tightened
//
//	an unreached end is           walkEnumerate's floor is 0.08s and seven
//	evidence of NOTHING           runs read 0.09–0.10s. That is not a floor
//	                              to raise. It is an end nobody sampled this
//	                              time, and walkParse's ceiling looked
//	                              exactly the same way for twenty-six runs
//	                              before a run landed on it
//
//	an end is a READING, not      which is the same rule stated so it can be
//	a choice                      checked. If an end is not a number
//	                              somebody took, it is a guess, and a guess
//	                              at an end is precisely where a false
//	                              verdict comes from. Outward rounding to
//	                              the record's two decimals is the one
//	                              allowed departure, and it rounds OUT
//
// The corollary worth saying out loud, because it is counter-intuitive and
// it was got wrong here: a band that looks too wide is cheap and a band that
// looks right is expensive. Re-taking a band is for when a reading falls
// OUTSIDE it. A reading comfortably inside is not an invitation.
var verifyTimingsTakenOn = struct {
	// For a reader. Nothing checks this.
	machine string
	// For the arm. runtime answers all four.
	goos, goarch, goVersion string
	cores                   int
	// The whole package, `go test -count=1 ./wasm/verify`. Not the sum of the
	// numbers in the prose — those are pieces of it, taken at different times
	// — and the one number a reader is most likely to be holding this record
	// up against, because it is the one they get by running the tests.
	//
	// It moved 2.49–2.59s → 2.70–2.80s when two repository-wide censuses were
	// added: TestTheDottedVersionParsersAreTheOnesTheReasonCovers and
	// TestTheShortLeversAreTheOnesThisRepositoryHasDecidedOn, each of which
	// parses every Go file in the tree and reports 0.18s of doing it.
	//
	// Two arms at 0.18s against a total that moved 0.21s do not add up, and
	// the gap is not explained here for the same reason the 60% one in the
	// other record is not: these are two readings taken in two sessions, and a
	// difference of that size is what a wall clock is worth. Each of these
	// walks is 0.18s whether it runs alone or beside the others, which was
	// measured rather than assumed.
	//
	// # How many of them there are, which this comment used to get wrong
	//
	// It said three, naming the copies census — then called
	// `TestEveryTimingsRecordIsTheSameShape`, then
	// `TestTheShapesThisRepositoryKeepsTwoCopiesOfAreInStep`, now
	// TestTheQuestionsOnTheSharedRepositoryParseAreTheOnesDecidedOn — as
	// the last. The two dead names are in backquotes and the live one is
	// not, which is the whole convention in one sentence: see
	// quotedprose_test.go.
	// There were four: TestEveryGitListingAsksForNulSeparatedPaths has parsed
	// the whole tree since before any of the other three existed and was
	// simply not counted. And a walk in this package is not always a test —
	// checkCitationsResolve is a helper, and it walks once per caller.
	//
	// So the count stopped being kept here. TestTheRepositoryWideWalksIn-
	// ThisPackageAreTheOnesDecidedOn reads it out of the source: SEVEN walks a
	// run, four of them parsing every Go file, each with a row saying what it
	// asks the repository and how deep it goes. About a second of the figure
	// below, and the number that decides when one shared parse becomes worth
	// its lifetime rules is that arm's repositoryParseBudget rather than a
	// sentence here.
	//
	// That arm costs 0.01s: it reads this ONE DIRECTORY and parses only the
	// files whose bytes name an enumeration — ten of forty-one where this
	// record was taken, and a ratio that moves with every file added here,
	// which is why the arm prints both numbers on every run rather than
	// leaving this sentence to be the record of them. A census of repository
	// walks that was itself a repository walk would have been the eighth
	// walk, and the figure below did not move for it — 2.70–2.80s became
	// 2.73–2.84s, which is one machine's spread.
	//
	// # And the readings the copies census grew, which is the same story again
	//
	// TestTheQuestionsOnTheSharedRepositoryParseAreTheOnesDecidedOn took on two
	// more questions — the declarations kept in two copies held identical
	// across both packages, which at the time were the import-resolving
	// helpers alone, and every core-count read held to being named in its
	// package's cores note — and both are read off declarations the walk had
	// already
	// built. Measured by taking them out and putting them back: 0.18s to
	// 0.19s, which is the fourth parse walk costing a hundredth more than the
	// three beside it.
	//
	// The figure below moved further than that, 2.73–2.84s to 2.78–2.92s over
	// fourteen runs, and the 0.01s does not account for it. What the rest is
	// cannot be said from here — it is the width of one machine's own spread,
	// which is the reason this record holds ranges and the reason none of
	// these numbers is an assertion.
	//
	// # A hundredth further, and what was under it
	//
	// The top end went to 2.93s on one reading of seven the next time this
	// was taken, and the row is widened rather than left to be contradicted by
	// a run. Two things changed in between and they pull opposite ways: the
	// cores-note check grew a second direction, which reads the two
	// record-carrying directories at 0.011s (see repositoryWalkRow.besides),
	// and the walk-counting pass stopped recomputing what each function binds
	// once per walk name, which took six traversals out of seven.
	//
	// Neither is a hundredth of a second on its own and the pair of them
	// certainly is not. The same afternoon put internal/themehistory a tenth
	// above ITS recorded range with no change to that package at all, which
	// was measured properly there — alternating the old code and the new —
	// and came out the same for both. So this is the machine, said once in
	// each record rather than argued about twice.
	//
	// # And how wide this range actually is, which is wider than it says
	//
	// It is twenty-one readings across two sessions, which is more than most
	// figures here and is still not enough to have found its ends. The other
	// record learned that the expensive way in the same afternoon: its plain
	// row was set from three readings, contradicted by the next run, set
	// again, and contradicted again — a floor being CHASED rather than
	// measured — and settled only when it was taken from sixteen at once. The
	// argument is written out at internal/themehistory/timings_test.go's
	// wholePackage and it is not about that package.
	//
	// It applies to every figure in this record and to every inline number in
	// this package's prose, all of which were set from three or seven
	// readings. None of them is wrong. All of them are NARROWER THAN THE
	// TRUTH by roughly what that row gained when it was taken properly —
	// somewhere around a twentieth at each end — and a reader landing just
	// outside one of them has been told, here, that the range is not tight
	// enough to judge them by.
	//
	// Not fixed, because fixing it is a few hundred runs to learn what one
	// row has already said, and because the direction it is wrong in is the
	// harmless one: a range too narrow reports a difference that is not there,
	// which sends somebody to look and find nothing. A range too wide would
	// hide one.
	//
	// # The fourth widening, and the arm that found it rather than an eye
	//
	// 2.78 → 2.76, on two readings of 2.762s and 2.765s in one sitting of ten.
	// Nineteen readings across two sittings this session ran 2.762–2.894s;
	// the lowest two are under the floor and one more, 2.775s, is AT it once
	// rounded to the hundredth the band is written to.
	//
	// What is new is not the widening — it is the fourth — but WHO found it.
	// The three before were found by a person running the command a dozen
	// times and comparing two ranges by eye, which is how this floor came to
	// be wrong three times. This one was reported by
	// TestTheFigureGoTestPrintedForThisPackageIsPlacedInItsBand on the first
	// sitting after that arm existed: "UNDER the band, by 15.2ms".
	//
	// Nothing was made faster. Two small tests were ADDED to this package in
	// the same session, which costs time rather than saving it, so this is
	// the floor having been set from too few sittings again — which is what
	// the paragraph below and the sittings rule at the top of this record
	// both predict, and is the fourth piece of evidence for them.
	//
	// The ceiling is untouched at 2.97s. Nothing this session came near it
	// (the highest was 2.894s) and an end nothing has reached is evidence of
	// nothing — see bandPlacement, which now says so on every reading.
	//
	// # The floor, which the taking that set it was already under
	//
	// The session after that re-taking read this package twelve times and got
	// 2.840–2.955s — nine of the twelve under the `2.88s` floor it had just
	// written. Nothing had been made faster; one file and one code path had
	// been ADDED. So the floor was wrong when it was typed, and it can be
	// seen to have been wrong in the session that typed it: that session's
	// own closing figures were "2.866–2.946s, in band", and 2.866 is not in
	// a band that starts at 2.88.
	//
	// The machine was ruled out the way this record always rules it out, by
	// reading the untouched sibling at the same moment: internal/themehistory
	// came back 3.018–3.131s against a recorded 2.92–3.22s, which is mid-band
	// and slightly SLOW rather than fast. An afternoon that made this package
	// quick would have made that one quick too.
	//
	// # That reading was wrong, and the next iteration found out how
	//
	// The sibling consulted was that package's whole-package figure, whose
	// band was 300ms wide WHEN IT WAS CONSULTED. It has been widened since
	// and is wider now; the figure is kept in the past tense because it is
	// the instrument that was used, and re-taking it here would describe an
	// instrument nobody reached for. Its TIGHT figures were checked an hour
	// later and
	// every one of them was under its floor — `wholeRun` by 10%,
	// `perObjectRun` by 6%, and all three of the terms on that arm's table.
	// Six figures across two packages, all low, in one session.
	//
	// So the machine was not ruled out. It was consulted through the one
	// instrument in either record too blunt to answer, and the answer came
	// back "no difference" because a 140ms difference does not show in a
	// 300ms band. See themehistoryTimingsTakenOn.wholeRun, which carries the
	// six readings and the argument.
	//
	// The floor here is still widened rather than replaced, which is what
	// this paragraph concluded and is the right treatment for a reading
	// whose cause is not known — it is only the REASON that was wrong. And
	// the lesson generalises past this record: a sibling is a control only
	// if its band is tighter than the difference being ruled out.
	//
	// So the range is widened to hold both takings rather than replaced with
	// the newer one — the same treatment, and for the same reason, as the
	// -race row in internal/themehistory's record. Two overlapping readings
	// of one program are one thing measured twice, and a record that reports
	// only the last afternoon is a record that cannot be checked against the
	// afternoon before it.
	//
	// What this cost is worth stating plainly, because it is the argument for
	// the whole file: the error was a hundredth of a second, it was in the
	// record's own headline figure, it was written and read and quoted in a
	// wrap-up on the same day, and nothing caught it — because a band is
	// checked by a person comparing two numbers, and that is the one kind of
	// check this repository has no arm for. See the header for why it cannot
	// have one.
	//
	// # The re-taking before it, and how that was told apart from the machine
	//
	// It said `2.78–2.93s` and then said `2.88–2.97s`. What moved is this
	// package:
	// the shared repository parse went from three questions to six in one
	// session, and the three that arrived — the file counts in prose, the two
	// comment rules, and the tests named in prose — cost between nothing and
	// ten milliseconds each, measured one at a time by taking each out and
	// putting it back. They do not add up to the whole of it, and the rest is
	// the same walk reading more: a per-question exemption replaced a
	// walk-level skip, so a file that used to be invisible is now parsed and
	// read by every rule.
	//
	// The reason this is recorded as the CODE and not as the afternoon is
	// that the other record was read at the same time, on the same machine,
	// against a package this session did not touch: internal/themehistory
	// came back at 3.01–3.16s against a recorded 2.92–3.22s, which is the
	// middle of its own range. A slow machine moves both. This moved one.
	//
	// Twenty-one runs again, which is the same sample the figure it replaces
	// was taken from, so the two are comparable — and still, by the argument
	// above, narrower than the truth.
	wholeFile string
	// The same tests, timed from INSIDE the binary — the clock TestMain puts
	// around m.Run(). See wholefileband_test.go.
	//
	// # Why this is a second figure and not a check on the first
	//
	// It is a smaller quantity and reliably so: 2.55–2.78s against a
	// wholeFile of 2.80–2.97s, a gap of about 190ms that is the build
	// check, the process starting, and package initialisation — everything
	// `go test` counts and a clock inside the process cannot see. The build
	// itself is NOT in it, which was measured: see below. Comparing
	// this reading against wholeFile's band would be comparing two different
	// things and reporting the difference as drift, which is the mistake
	// this whole record exists to stop somebody making.
	//
	// So it sits beside wholeFile exactly as wholeRun sits beside
	// wholePackage in internal/themehistory: the package total a person
	// reads off `go test`, and the in-process figure something can actually
	// measure. That record grew the pair for this reason and this one
	// arrives at it two sessions later from the other direction.
	//
	// # What having it buys
	//
	// wholeFile drifted twice and both times a person found it by running
	// the command a dozen times and comparing by eye, because nothing in
	// this package could see the number. This one is measured on every run
	// and reported against its band, so a real regression shows up here
	// first. What is left unwatched is only the 170ms of overhead, which is
	// the part no change to this repository's code can move.
	//
	// # And a prediction about its width, which the takings since have
	// # refuted — and which was not evidence about anything
	//
	// This said "the TIGHTER of the two — 130ms wide against 130ms, over a
	// smaller figure", the reasoning being that the overhead this excludes is
	// the noisiest part of what `go test` reports, so the in-process figure
	// should spread less.
	//
	// Both bands have been widened several times since and the order has
	// reversed: this one is now the WIDER of the two. That is not a refutation
	// of where the noise lives, because the comparison was never about that —
	// this band is taken over five sittings and `wholeFile`'s over four, and
	// the sittings rule at the top of this record says a band grows with
	// sittings until it has found its ends. Two bands with different numbers
	// of sittings behind them cannot be compared for width at all.
	//
	// So the claim is struck rather than re-taken, and the two numbers it
	// carried are gone: the verdict line prints the width of the band it
	// compared against, which is the one place it cannot go stale.
	//
	// # Rounded outward, which is the part that had to be learned twice
	//
	// Sixteen runs read 2.666–2.767s and the band says 2.65–2.78s. The
	// first attempt wrote the observed ends exactly, and the next run came
	// in a fraction under the floor and reported itself outside a band it
	// had helped set.
	//
	// A range taken from N runs is a sample, and its ends are the two most
	// extreme readings in it rather than the limits of anything. Writing
	// them down as limits guarantees the next sample argues with them. The
	// two decimal places every figure in both records carries are the
	// rounding: outward at both ends, to the precision the record writes.
	//
	// This is the same lesson wholeRun arrived at by having its floor set
	// three times in one evening, and it is worth stating here because it
	// is cheaper than that: it costs a hundredth of a second of width and
	// it removes a whole class of verdict that is about the sample rather
	// than about the code.
	//
	// # And the reason outward rounding was not enough either
	//
	// The band was set at 2.65–2.78s from sixteen runs, and read 2.608s two
	// hours later in the same session with nothing changed in between. It
	// was re-set at 2.60s and reported UNDER within four runs. It is the
	// third band re-taken in one evening, at which point the re-taking is
	// the finding — but the first explanation for it was wrong and is worth
	// keeping here beside the right one.
	//
	// **What it looked like**: a fall. 2.767s early, 2.593s hours later,
	// one direction, no code change — so the account written first was that
	// a reading of this package falls over a session as caches warm.
	//
	// **What it is**: a spread. A later sitting of twelve consecutive runs
	// went the OTHER way, 2.555s up to 2.733s with no trend. Across the
	// whole session the figure runs 2.555–2.767s, a spread of 210ms, and
	// any one sitting of eight or twelve samples about half of that and
	// looks tight.
	//
	// # Consecutive runs are correlated, which is the whole lesson
	//
	// That is why three bands were re-taken in one evening and why each
	// re-taking was contradicted by the next sitting. Every one of them was
	// set from runs taken back to back. Sixteen consecutive runs are not
	// sixteen samples of this figure; they are something closer to one
	// sample of a sitting, and the next sitting lands somewhere else in a
	// distribution twice as wide as they suggested.
	//
	// So the count of runs is the wrong statistic and always was. **A band
	// needs sittings, spread apart, not runs.** Ten runs at three different
	// times of day is worth more than fifty back to back, and the method
	// line is the only place a reader can tell one from the other — which
	// is why this field carries one and why every field that has a band
	// should.
	//
	// # What was ruled out by experiment
	//
	// The build cache: a run under a fresh empty GOCACHE, which rebuilds
	// the standard library and everything else, produced an in-process
	// figure of 2.643s — indistinguishable from the warm runs either side
	// of it. Obvious in hindsight, since this clock starts after the binary
	// is built, and it was measured rather than assumed because the first
	// written explanation named a warming cache.
	//
	// Compilation is ruled out of `wholeFile` too, and that corrects
	// something this record said one session ago. That cold run took 6.57s
	// of wall clock and `go test` still REPORTED 2.849s: the reported
	// figure excludes compiling. The ~190ms between the two fields is the
	// process starting, package initialisation and the build CHECK — not
	// the build.
	//
	// What is left is the machine: the file cache, and whatever else about
	// an eight-core laptop differs between one ten-minute stretch and
	// another. Unmeasurable here without a reboot or a cache purge, and
	// bounded rather than explained — 210ms, which is what the band holds.
	//
	// # So the floor is not the lowest reading
	//
	// Outward rounding to two decimals is the right treatment for sampling
	// noise within a sitting. It is not enough for a spread BETWEEN
	// sittings, because the next sitting is not drawn from the one that set
	// the band.
	//
	// The floor sits at 2.55s under a lowest reading of 2.555s and the
	// ceiling at 2.78s over a highest of 2.767s. The band is 230ms for a
	// figure whose observed spread is 210ms, which is the honest width and
	// not a padded one: it is wide because the figure is, and the way it
	// gets narrower is somebody explaining the machine rather than somebody
	// taking more readings.
	wholeFileInProcess string
	// TestHowWideTheNarrowerFoldIsAndWhatHoldsTheGap end to end, which is what
	// inkglyph_test.go's `128ms` sits inside.
	//
	// The 128ms itself is NOT re-taken here and the difference matters: it is
	// the astral walk alone, measured once from inside the test, and isolating
	// it again would mean either instrumenting the test to print a number for
	// a comment or running a second copy of the walk outside it. Both are
	// worse than an honest note. What is re-derivable without either is the
	// enclosing test's wall clock, which brackets it — so that is what is
	// recorded, and the 128ms is a sub-figure of it rather than a number this
	// record vouches for.
	//
	// # And whether it earns its place at all, which is a question this record
	// # cannot answer and inkglyph_test.go can
	//
	// A bracketed number from an unknown day is still a number from an unknown
	// day, and no amount of attribution makes it re-derivable. The question is
	// therefore not "how do we re-take it" but "what is it holding up", and
	// the answer turned out to be nothing that needs it: the astral walk's
	// AFFORDABILITY is demonstrated by the test running it on every green run,
	// and the walk still going all the way up is asserted by foldMeasuredOn's
	// six astral counts, which a narrowing would fail. Both are re-derived; the
	// wall clock is not, and does not have to be.
	//
	// So the figure is kept where it was taken, marked as the reading a past
	// decision was made on, and the sentence it used to carry now stands on
	// the arms instead. That is the resolution rather than a re-measurement:
	// the number was never the load-bearing part, and pricing it a second time
	// would have been work to keep something honest that could instead stop
	// being asked to hold anything.
	//
	// This field, foldWalk, is the one that IS re-takeable, and the command
	// for it is in the table above.
	foldWalk string

	// What one repository-wide walk costs at each of the three depths
	// repositoryWalks records, as the whole wall clock of a test that does
	// only that.
	//
	// # Why these live here and not in the sentences that quote them
	//
	// They were literals inside two failure messages in repowalks_test.go,
	// attributed — "0.18s where verifyTimingsTakenOn was taken" — which is
	// the discipline this record exists for and is not enough. Attribution by
	// reference to a record that gets re-taken is a pointer that moves: when
	// `wholeFile` was re-taken and went from `2.78–2.93s` to `2.88–2.97s`,
	// every figure citing this record silently began claiming to be from a
	// taking it was not from. One of these three had moved with it and two had not,
	// and nothing anywhere said which.
	//
	// So they are fields. A re-taking edits the record, the messages read the
	// record, and there is one place the figure lives — which is the whole of
	// what this repository's copies census is about, applied to a number
	// instead of a struct.
	//
	// # What the three depths are
	//
	//	walkEnumerate  one `git ls-files`, a stat per file, and the citation
	//	               scan citingFiles does on the way past
	//	walkRead       that, plus every tracked file read into memory
	//	walkParse      that, plus go/parser over every Go file in the tree
	//
	// Cumulative, so the interesting number is the step: reading the tree
	// costs about as much again as enumerating it, and parsing it costs about
	// as much again as both. A walk that grows a parse has roughly doubled,
	// which is what makes repositoryParseBudget the number that decides and
	// not repositoryWalkBudget.
	//
	// # How reproducible these three are, which took four sittings to get
	// # right and was got wrong twice on the way
	//
	// Measured because the sittings rule at the top of this record predicted
	// these would spread across sittings the way the package figure does.
	// Twenty-one runs of walkParse in three sittings, with the whole
	// verification suite run in between to move the machine:
	//
	//	sitting 1   0.20 0.19 0.19 0.20 0.20 0.20 0.19
	//	sitting 2   0.20 0.20 0.19 0.20 0.20 0.20 0.19
	//	sitting 3   0.20 0.20 0.20 0.20 0.19 0.19 0.20
	//
	// Two values, in all three sittings, where 0.01s is the resolution of
	// the `--- PASS:` line these are read off. On that evidence this comment
	// said the three depths were REPRODUCIBLE and the band was re-taken at
	// 0.19–0.20s.
	//
	//	sitting 4   0.19 0.19 0.20 0.20 0.22
	//
	// Five runs later. One reading in twenty-six, outside a band set on the
	// other twenty-five, in the sitting immediately after the one that set
	// it.
	//
	// # What that actually settles
	//
	// Not "these are reproducible" and not "bands need sittings". Both were
	// written tonight off a sample that had not finished. What twenty-six
	// runs say is narrower and more useful: **the middle of this figure is
	// tight and its tail is not**, which is what a wall clock on a shared
	// machine is, and three sittings of seven did not find the tail.
	//
	// The recorded band was 0.20–0.24s and it would have HELD that 0.22s.
	// What was wrong with it was only its floor. So the fix is the floor,
	// one resolution step down, and the ceiling stays where somebody put it.
	//
	//	sitting 5   0.20 0.19 0.19 0.24 0.19 0.19
	//
	// Which is the ceiling, exactly, on the sitting after that decision.
	// Thirty-two runs: twenty-nine at 0.19 or 0.20, one at 0.22, one at
	// 0.24. Whoever first wrote 0.20–0.24s had seen the tail and put the
	// ceiling on it, and three sittings of this session's re-measuring had
	// not reached it. The band is the original with its floor let out.
	//
	// # The rule that was broken in the re-taking, by the session that wrote
	// # the rule
	//
	// "Widen to hold what has been SEEN, then leave it alone." Re-taking
	// 0.20–0.24s as 0.19–0.20s is not widening: it is re-centring on the
	// latest sitting, which is the thing the rule exists to forbid, and it
	// was falsified within five runs. The rule was written two iterations
	// before it was broken, by the same hand, on the same record.
	//
	// Which is the argument for a band being widened and almost never
	// narrowed. A ceiling nothing has reached in twenty-six runs costs a
	// reader nothing; a ceiling somebody tightened costs a false verdict the
	// first time the tail shows up.
	//
	// # Why these three alone carry no method in their VALUE
	//
	// Because they are the only figures in either record interpolated into a
	// sentence. repowalks_test.go's failure messages read them into a list —
	// "%s, %s and %s respectively where verifyTimingsTakenOn was taken" —
	// and a value carrying a comma turns that list into nonsense, which is
	// what a first attempt at writing the method here produced. So the
	// method lives in this comment and the values stay bare.
	walkEnumerate, walkRead, walkParse string
}{
	machine:   "Apple M3 (Mac15,13), macOS 26.2",
	goos:      "darwin",
	goarch:    "arm64",
	goVersion: "go1.26.1",
	cores:     8,
	wholeFileInProcess: "2.55–2.78s over about sixty runs in five sittings " +
		"across one session, the clock TestMain puts around m.Run(). The " +
		"sittings are the statistic and the runs are not — see the comment",
	wholeFile: "2.76–2.97s over sixty-five runs in six sittings across three " +
		"sessions, the floor " +
		"widened four times in three sessions — three times on readings " +
		"under it, and once by the session that worked out why that keeps " +
		"happening: consecutive runs are correlated and a band needs " +
		"sittings rather than runs. See wholeFileInProcess",
	walkEnumerate: "0.08–0.10s",
	walkRead:      "0.15–0.17s",
	walkParse:     "0.19–0.24s",
	foldWalk:      "0.40–0.52s over seven runs, node v22.12.0",
}

// Which of this package's figures move with the core count, and which do not.
//
// # Why every record carries one of these
//
// `cores` is the one machine field that changes a recorded number by a term a
// reader can name, and the two records in this repository were saying
// different amounts about it: internal/themehistory's carries a measured table
// — 0.90s at one worker down to 0.22s at eight — and this one said nothing,
// which a reader holding a 3.1s run against the 2.78s below reads as "nobody
// measured that" rather than as "that is not where the difference is".
//
// Silence is the wrong answer either way round, so both records now state it
// and wasm/verify/copies_test.go holds every record to having one — and holds
// this one to naming every term in this package that actually scales.
//
// # Why the note names declarations and not only a family
//
// It is held to being COMPLETE, not merely to being there — copies_test.go's
// checkCoresNoteNamesEveryScaledTerm finds every declaration in this package
// that reads runtime.NumCPU or runtime.GOMAXPROCS and requires this sentence
// to name it. A note that says which terms scale is a claim about an inventory,
// and the inventory is the half a parse can settle: the numbers below are a
// wall clock and cannot be asserted, but a pool arriving in a third place is
// somebody adding code, and this fails on it.
//
// So `affordedKLeafBandWalk` and `affordedTwoStepBands` are written out. They
// are the whole of the `afforded*` family's concurrency — the reporting arm's
// own NumCPU is the comparison against the record rather than a term, and is
// skipped there by name.
//
// # What this package's answer actually is, which is not "nothing"
//
// The expensive half of wholeFile does not move. The four repository-wide
// walks hand every Go file in the tree to go/parser one after another — 450
// tracked Go files where this record was taken, 0.18s each, single-threaded,
// the same number on any machine — and foldWalk is a node process this package waits on
// rather than shares a core with.
//
// What does move is the afforded* band family in themenearmiss_test.go —
// affordedKLeafBandWalk and affordedTwoStepBands, which are the two that take
// `workers := runtime.GOMAXPROCS(0)` and divide their combinations across it.
// Measured by fixing GOMAXPROCS and re-running, three runs apiece on the eight
// cores named above:
//
//	GOMAXPROCS    the whole package
//	1             3.35–3.49s
//	2             2.92–3.00s
//	4             2.87–2.91s
//	8             2.88–2.92s
//
// Three runs a row, and this taking REPLACES the ranges rather than widening
// them, which is a departure from how the three takings before it were
// recorded and is the reason it is written down.
//
// Those three each held every taking's range together, because each was of
// the same program: the row above had not moved, so a reading from a
// different afternoon was another reading of one thing. This one is not. The
// shared repository parse went from three questions to six and the row above
// left its range, so unioning would produce a span covering two programs — a
// one-core figure of 3.08–3.49s, most of which no version of this package has
// ever taken. A range that wide hides a difference, which is the failure the
// other direction at least does not have.
//
// The whole table and not the row that moved, which is the part the previous
// taking established and this one keeps. Widening `wholeFile` and leaving
// these four would make a record where the figure at the top is from one
// afternoon and the table explaining it is from another, with nothing on
// either saying which — a reader comparing a one-core run against the number
// above would be comparing two days. That is the fault this record exists to
// end, and there is no version of it that is acceptable inside the record
// itself.
//
// Every row moved, by about the same amount, which is the useful half of
// re-taking a whole table: the added work is single-threaded, so it lands on
// every core count equally and none of the movement is about concurrency.
//
// The SHAPE is what the sentence below is about and it did not change, though
// the size of it did. One core is now about a sixth dearer than four rather
// than a tenth, because the denominator went up and the serial half of the
// package went up with it; two is a couple of hundredths above four, and four
// and eight are the same number. So: a core count that differs from eight is
// worth a sixth of the figure between one and two, a fiftieth between two and
// four, and nothing above that. internal/themehistory's keeps improving all
// the way to eight, because its term is a git process per commit rather than
// one Go
// program's own goroutines. A reader on a four-core machine should expect
// this package's number and not that one.
// # How this has to be written, which is a constraint from outside
//
// Literals joined by `+`, and nothing else. wasm/verify/copies_test.go reads
// this note without running the package — stringLiteralValue evaluates a
// string constant of exactly that shape — and holds what it names to being
// terms this package still has, in both directions. A note assembled by a
// function, or built out of other constants, comes back empty and is reported
// rather than silently exempt, which is a finding about this declaration
// arriving in another package's test.
//
// The terms themselves go in BACKQUOTES. That is what says a word is meant as
// the name of a declaration rather than as English, and it is the only thing
// either direction reads — a term named in prose is neither claimed nor
// checked.
const coresAttribution = "The four repository-wide walks in this package are " +
	"single-threaded — go/parser over 450 tracked Go files " +
	"where this record was taken, 0.18s each — and " +
	"`foldWalk` is a node process. Those do not move with the core count. " +
	"Two declarations do, and they are the whole of it: " +
	"`affordedKLeafBandWalk` and `affordedTwoStepBands` in " +
	"themenearmiss_test.go each take `workers := runtime.GOMAXPROCS(0)` and " +
	"divide the afforded* band family's combinations across it. The package is 3.35–3.49s at one core, " +
	"2.92–3.00s at two, and flat from four to eight. So a differing core " +
	"count is worth about a sixth of the figure above between one core and " +
	"two and a fiftieth between two and four — which is the opposite shape " +
	"from " +
	"internal/themehistory's, where the term is a git process per commit " +
	"and the " +
	"improvement runs all the way to eight. `recordMachineDiffers` also " +
	"reads the count and does not scale with it: it compares this machine's " +
	"cores against the record's, so that a verdict says nothing at all on a " +
	"computer the band is not about. It is named here because this note is " +
	"held to naming every reader of the count in the package, which is " +
	"stricter than naming every term that scales with it — and the name it " +
	"names moved, which is the census doing its job: the comparison was " +
	"inline in `wholeFileRunIsTheRecordedOne` and in the reporting arm, and " +
	"became one function when a third caller wanted it."

// The range a record field opens with.
//
// A second copy. The first is recordedBand in
// internal/themehistory/band_test.go and this is the same function, for the
// same reason the import-resolving helpers are two copies: these are two
// separate `package main` programs and neither can import the other's tests.
//
// The two ARE held identical, by checkTwoCopyDecls — the same census that
// holds the seven import-resolving helpers, and the reason that census is no
// longer named for them. This function is in twoCopyFunctionShapes and the
// pattern above is in twoCopyValueShapes; change one copy and the shared
// repository parse fails, naming both files.
//
// The alternative considered and declined was for the records to carry their
// bands as DURATIONS and render the prose, which would leave nothing to
// parse. It does not remove the copy — two `package main` programs cannot
// import each other's tests, so a renderer is duplicated exactly as a parser
// is — and it costs more than it saves. See ai_docs/plans/non_goals.md.
var recordedBandForm = regexp.MustCompile(
	`^(\d+(?:\.\d+)?)–(\d+(?:\.\d+)?)(µs|ms|s)\b`)

// recordedBand reads that prefix back as two durations. See the other copy.
func recordedBand(field string) (lo, hi, step time.Duration, ok bool) {
	m := recordedBandForm.FindStringSubmatch(field)
	if m == nil {
		return 0, 0, 0, false
	}
	unit := map[string]time.Duration{
		"µs": time.Microsecond,
		"ms": time.Millisecond,
		"s":  time.Second,
	}[m[3]]
	parse := func(s string) time.Duration {
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return 0
		}
		return time.Duration(f * float64(unit))
	}
	lo, hi = parse(m[1]), parse(m[2])
	// And the PRECISION the ends are written at, which is the third thing a
	// band says and the one nothing was reading.
	//
	// An end is a reading rounded outward to the record's own two decimals —
	// that is the one departure from "an end is a reading" the record allows
	// — so "does this reading reach the floor" has an exact answer at that
	// precision and no answer at all without it. A reading of 2.5512s is the
	// floor of a band written 2.55–2.78s; it is not the floor of one written
	// 2.551–2.780s.
	//
	// Taken from the FINER of the two ends. A band spelled "0.19–0.245s" is a
	// hundredth at one end and a thousandth at the other, and the step has to
	// be fine enough not to call a reading an end it is not.
	digits := 0
	for _, end := range []string{m[1], m[2]} {
		if dot := strings.IndexByte(end, '.'); dot >= 0 {
			if d := len(end) - dot - 1; d > digits {
				digits = d
			}
		}
	}
	step = unit
	for i := 0; i < digits && step > 1; i++ {
		step /= 10
	}
	// A range written backwards is a typing error in the record rather than a
	// reading of anything, and it would otherwise make every run "outside the
	// band" with no clue as to why.
	if lo <= 0 || hi <= 0 || hi < lo {
		return 0, 0, 0, false
	}
	return lo, hi, step, true
}

// againstBandGiven is that sentence, as a function of nothing but its
// arguments.
//
// # Why the gate and the machine check are the caller's
//
// This package's one caller is the arm that places the figure `go test` itself
// printed — a reading taken OUTSIDE this process, whose gate is the presence
// of that figure rather than the verdict lever. The other package has three
// arms that clock their own run and gate on the lever. Pushing the gate and
// the machine out to the callers is what lets one function serve both.
//
// Pushing both out leaves a function of four arguments that returns a
// sentence, which is the form that can be ASSERTED — see
// TestWhereAReadingFellInItsBandIsReadOffTheBandsOwnPrecision, which is the
// only kind of test anything in this file can carry. The gate and the machine
// read the environment and the runtime, and a test over either is a test of
// the computer it runs on.
//
// # A second copy
//
// The first is in internal/themehistory/band_test.go and this is the same
// function: two `package main` programs, neither able to import the other's
// tests. Held identical by checkTwoCopyDecls — it is in twoCopyFunctionShapes
// beside recordedBand and bandPlacement, because two packages spelling "UNDER
// the band, by 14ms" differently is two records a reader cannot hold against
// each other, which is the whole point of their being a pair.
func againstBandGiven(fieldName, field string, differs []string,
	got time.Duration) string {

	lo, hi, step, ok := recordedBand(field)
	if !ok {
		return fmt.Sprintf("\n\nNo band was read out of %s. Its value has to "+
			"OPEN with the range, the way every field in both records is "+
			"written — see recordedBandForm. Until it does, the reading "+
			"above is not being compared with anything.", fieldName)
	}
	if len(differs) > 0 {
		return fmt.Sprintf("\n\nNot compared against %s (%v–%v): this run is "+
			"not on the machine that record was taken on — %s. A reading "+
			"from a different computer is not evidence about the band.",
			fieldName, lo, hi, strings.Join(differs, ", "))
	}
	// Rounded against the BAND and not against a fixed unit. batchRetire's
	// range is 180µs–290µs, and a difference rounded to the millisecond
	// printed there as "by 0s" — a verdict that says the reading is outside
	// a range and then says by nothing, which is worse than not printing it.
	// A hundredth of the band's own width is fine enough to be true at every
	// scale either record holds and coarse enough not to print seven digits.
	// Rounded against the BAND and not against a fixed unit: batchRetire's
	// range is 180µs–290µs, and a difference rounded to the millisecond
	// printed there as "by 0s".
	//
	// That was the first half of the fix and it was not enough. A reading
	// one step under the floor still rounds to zero, and "outside the band
	// by nothing" is the same useless sentence arrived at from the other
	// side. So a difference that rounds away is reported at microsecond
	// resolution instead — whatever it is, it is not nothing, because the
	// arm that prints it only runs when the reading is outside.
	round := func(d time.Duration) time.Duration {
		step := (hi - lo) / 100
		if step <= 0 {
			step = time.Microsecond
		}
		if r := d.Round(step); r != 0 {
			return r
		}
		return d.Round(time.Microsecond)
	}
	// Compared at the precision the band is WRITTEN to, and not at the clock's.
	//
	// # The false UNDER this fixes, which this machinery produced twice in one
	// # hour
	//
	// An end is a reading rounded outward to the record's two decimals — that
	// is the one departure from "an end is a reading" the record allows, and it
	// means a floor of `2.76s` stands for readings down to 2.755s. A raw
	// comparison calls 2.755s UNDER by 5ms, which sends a reader to re-take a
	// band that is right. Both records did exactly that, within an hour of the
	// arm that places these figures existing: wasm/verify read 2.755s against a
	// 2.76s floor and internal/themehistory 2.816s against 2.82s, and both
	// reported UNDER.
	//
	// bandPlacement already judged "does this reading reach an end" this way.
	// The in-band decision did not, so the two halves of one sentence
	// disagreed: a reading could be reported outside a band and, by the
	// placement rule, be AT its floor.
	//
	// The distance printed is still the true one, lo-got rather than a rounded
	// difference. What the rounding decides is WHICH branch, not what to say
	// once the branch is chosen.
	switch rounded := got.Round(step); {
	case rounded < lo:
		return fmt.Sprintf("\n\nUNDER the band %s records (%v–%v), by %v. On "+
			"the machine that record names, so it is not another computer. "+
			"Either this got faster and the floor is stale, or the floor was "+
			"set from too few runs — which has happened to both records in "+
			"this repository and is why this line is printed at all. Re-take "+
			"it: widen the range to hold both takings rather than replacing "+
			"it, unless something is known to have changed the code.",
			fieldName, lo, hi, round(lo-got))
	case rounded > hi:
		return fmt.Sprintf("\n\nOVER the band %s records (%v–%v), by %v. On "+
			"the machine that record names, so it is not another computer — "+
			"but it may well be a busy one, and one reading over a ceiling "+
			"is not a regression. Take it several times. If it holds, "+
			"something here costs more than it did and the record is the "+
			"place that says so.",
			fieldName, lo, hi, round(got-hi))
	default:
		// The placement is the half of this line that is about the BAND
		// rather than about the reading: a reading inside a band is a pass,
		// and which part of the band it is in is the only thing it tells
		// anybody about the two ends. See bandPlacement.
		return fmt.Sprintf("\n\nIn the band %s records (%v–%v), on the "+
			"machine it names — %s.", fieldName, lo, hi,
			bandPlacement(lo, hi, got, step))
	}
}

// recordMachineDiffers is every way this computer is not the one this
// package's record was taken on, in the words the reporting arm prints.
//
// # Why it is a function, which is that it had three callers and two bodies
//
// The comparison was written out twice: once in the reporting arm and once in
// the whole-file verdict's own guard, with the comments in the two explaining
// the same two subtleties in different words. Both were right. A third caller
// — the arm that places a figure `go test` printed, which needs the same four
// answers about a reading taken outside the process — is the point at which
// keeping them in step becomes somebody remembering to, and this repository
// writes functions instead of remembering.
//
// internal/themehistory reached this a run earlier and for the same reason;
// its copy is the same four comparisons over its own record.
//
// # Why the two packages' copies are not held identical
//
// They cannot be. Each reads its own record, and the two records are separate
// anonymous struct literals in separate `package main` programs — which is the
// argument for their being two copies in the first place. A shared function
// would need a shared type, and there is none to have. So the two are a pair
// by intent and not by census, and the thing that IS held identical is the
// sentence built out of the result: see againstBandGiven.
func recordMachineDiffers() []string {
	rec := verifyTimingsTakenOn
	var differs []string
	if got := runtime.GOOS; got != rec.goos {
		differs = append(differs, fmt.Sprintf("GOOS %s against %s", got, rec.goos))
	}
	if got := runtime.GOARCH; got != rec.goarch {
		differs = append(differs, fmt.Sprintf("GOARCH %s against %s", got,
			rec.goarch))
	}
	// The toolchain, because the numbers are of code this compiles and the
	// scheduler that runs it. Compared as a prefix: a patch release is a
	// different toolchain and worth naming, and `devel` builds carry a suffix
	// no equality test would ever match.
	if got := runtime.Version(); !strings.HasPrefix(got, rec.goVersion) {
		differs = append(differs, fmt.Sprintf("%s against %s", got, rec.goVersion))
	}
	// NumCPU and not GOMAXPROCS: the band walks split their family across
	// GOMAXPROCS, which a caller can set, and what the record is about is the
	// machine underneath it.
	if got := runtime.NumCPU(); got != rec.cores {
		differs = append(differs, fmt.Sprintf("%d cores against %d", got,
			rec.cores))
	}
	return differs
}

// bandPlacement is where in a band a reading fell, and which of the band's two
// ends — if either — the reading is evidence FOR.
//
// # What this is for, which is an end nobody took
//
// A band's ends are readings, not choices. That rule is written out at
// wasm/verify/timings_test.go and it cost five re-takings to arrive at, and it
// has a consequence nothing was acting on: an end that no reading has reached
// is an end standing on whatever the session that wrote it had in front of it,
// and there is no way to tell one of those from an end twenty runs have landed
// on. Both are two decimals in a struct literal.
//
// What can be said cheaply, on every run that asks for a verdict, is whether
// THIS reading reaches an end. That is the evidence accumulating in the only
// place it honestly can — beside the reading, in the line a person reads when
// they take figures at the end of a session — rather than in a mark somebody
// has to remember to keep.
//
// # Why it does not suggest moving anything
//
// Because the rule says not to, and the rule is the expensive half of this
// record's history. A reading in the middle of a band says nothing about
// either end; a band nothing has reached the ends of is not a band to narrow,
// and walkParse was narrowed on exactly that evidence and falsified five runs
// later. So the sentence names what the reading is evidence for and stops,
// and the one instruction it carries is the one that was got wrong.
//
// # The width, which is here because prose kept copying it
//
// Three sentences in these two records stated a band's width as a number —
// `3% wide`, `300ms wide`, `130ms wide against 130ms` — and all three were
// stale, two of them describing bands that had since been widened and one
// making a comparison that had since reversed. Every one of them was
// derivable from two numbers in the same file. So the width is printed here,
// beside the reading, and the prose says what it is FOR instead of what it
// was.
func bandPlacement(lo, hi, got, step time.Duration) string {
	width := hi - lo
	reaches := func(end time.Duration) bool {
		if step <= 0 {
			return got == end
		}
		return got.Round(step) == end
	}
	if width <= 0 {
		return fmt.Sprintf("a band with one value in it, %v wide", width)
	}
	switch {
	case reaches(lo):
		return fmt.Sprintf("at the floor of a band %v wide, at the precision "+
			"the band is written to (%v) — so this reading is one the floor "+
			"stands on", width, step)
	case reaches(hi):
		return fmt.Sprintf("at the ceiling of a band %v wide, at the "+
			"precision the band is written to (%v) — so this reading is one "+
			"the ceiling stands on", width, step)
	}
	// A ratio of two durations is a count rather than a duration, which is
	// why it is converted before it is printed: %d over a time.Duration
	// prints its nanoseconds.
	return fmt.Sprintf("%d%% up a band %v wide, so it reaches neither end. An "+
		"end no reading has reached is evidence of nothing, and not a reason "+
		"to move one", int(100*(got-lo)/width), width)
}

// The verdict is asked for, and the reason is `go test ./...`.
//
// `go test` runs package binaries in PARALLEL. The bands here were taken
// with one package running alone, which is the command their method lines
// name, and a package sharing an eight-core laptop with fifteen other test
// binaries is not that command. Measured: this package reads 2.5-2.7s alone
// and 2.98-3.10s inside `go test ./...`, and internal/themehistory's
// wholeRun reads 1.4-1.6s alone and 1.70s there. Both report OVER.
//
// That path runs in every session's verification. A verdict that is wrong
// every time somebody runs the whole tree is not a weaker verdict — it is
// the noise this file's own header says argues for deleting the record, and
// it would teach a reader to skip the line in the one case it is right.
//
// There is no way for a test binary to know what else `go test` is running.
// So it is asked for instead, spelled out the way GRMOB_PER_OBJECT_FETCH is
// and for the same reason: a typo should be a setting that names itself
// rather than one that silently did nothing. The reporting arm, which
// prints on every `-v` run, says how to ask.
const (
	bandVerdictEnv   = "GRMOB_BAND_VERDICT"
	bandVerdictAsked = "required"
)

// bandVerdictWanted is whether this run asked for a band verdict. See above.
func bandVerdictWanted() bool {
	return os.Getenv(bandVerdictEnv) == bandVerdictAsked
}

// This run says whether it is standing on the machine the timings came from.
//
// # Why this reports and does not assert
//
// A timing cannot be an arm — see verifyTimingsTakenOn — and neither can the
// machine, for a reason that is not the same: asserting the machine would fail
// on every computer that is not this one, which is every computer except this
// one. What is worth having is the SENTENCE, in front of the person comparing
// a number in a comment with a number on their screen, saying that the two
// were taken on different things before they go looking for a regression.
//
// So it is a Logf, and the file is honest about what that buys: it is visible
// under -v and on any run where something else in this package has failed,
// which are the two occasions anybody reads this output. A green quiet run
// does not need it.
//
// The one thing here that IS an assertion is that the record is filled in at
// all. A zeroed field is a record somebody added a timing to without saying
// where it came from, which is the state this whole thing exists to end, and
// it is a fact about the source rather than about the machine.
func TestTheTimingsInThisPackageSayWhichMachineTheyCameFrom(t *testing.T) {
	rec := verifyTimingsTakenOn
	if rec.machine == "" || rec.goos == "" || rec.goarch == "" ||
		rec.goVersion == "" || rec.cores == 0 || rec.wholeFile == "" ||
		rec.wholeFileInProcess == "" ||
		rec.foldWalk == "" {
		t.Fatalf("verifyTimingsTakenOn has an empty field (%+v).\n\n"+
			"Every wall-clock number in this package's prose is attributed to "+
			"this record, and a record with a hole in it attributes them to "+
			"nothing. That is the state this record exists to end: a reader "+
			"a few percent off one of these numbers cannot tell a regression "+
			"from a different computer.", rec)
	}

	// The four comparisons are recordMachineDiffers, which is where they went
	// when a third caller needed them. This arm prints them, the whole-file
	// verdict reads whether the list is empty, and the arm that places a
	// figure taken outside the process does the same — three readings of one
	// answer, which is one more than it takes for a second copy to drift.
	differs := recordMachineDiffers()

	if len(differs) == 0 {
		t.Logf("this run is on the machine the timings in this package were "+
			"taken on: %s, %s, %s/%s, %d cores. The whole package was %s "+
			"there, and the fold walk %s.\n\n"+
			"To have this run's own reading compared against the band it "+
			"records, set %s=%s — on a run of THIS package alone, because "+
			"`go test ./...` runs package binaries in parallel and the "+
			"bands are of a package running on its own. See "+
			"wholefileband_test.go.",
			rec.machine, rec.goVersion, rec.goos, rec.goarch, rec.cores,
			rec.wholeFile, rec.foldWalk, bandVerdictEnv, bandVerdictAsked)
		return
	}
	// The core count, said out loud whenever it differs. A reader told "8
	// cores against 4" and nothing else has to go and find out which term
	// that moves, and the honest answer here is "a tenth of it, and only
	// below two" — which is worth as much as a table would be, and is the
	// half that used to be missing. See coresAttribution.
	cores := ""
	if runtime.NumCPU() != rec.cores {
		cores = "\n\n" + coresAttribution
	}
	t.Logf("the wall-clock numbers in this package's comments were taken on %s "+
		"(%s, %s/%s, %d cores), and this run is not on it: %s.\n\n"+
		"The whole package was %s there and the fold walk %s, `go test "+
		"-count=1 ./wasm/verify`. A reading of these tests that is some "+
		"percent off one of their numbers is a difference between two "+
		"computers before it is anything else — which is what this record is "+
		"for, and why none of these numbers is an assertion.%s",
		rec.machine, rec.goVersion, rec.goos, rec.goarch, rec.cores,
		strings.Join(differs, ", "), rec.wholeFile, rec.foldWalk, cores)
}

// The figure `go test` printed, for the arm that places it in its band.
//
// Spelled out and the same name in both packages, the way GRMOB_BAND_VERDICT
// is: a person who has the recipe for one record has it for the other.
const packageReadingEnv = "GRMOB_PACKAGE_READING"

// The figure `go test` prints, handed back to the code that knows the band.
//
// # Why this one had no verdict, and why that cost something
//
// verifyTimingsTakenOn.wholeFile is what `go test -count=1 ./wasm/verify` reports, and that number is
// produced by the `go` command rather than by the test binary. No test can see
// it. It is also the number a person is most likely to be holding this record
// up against, because it is the one they get by running the tests — and it has
// drifted twice, both times found by somebody running the command a dozen
// times and comparing two ranges by eye.
//
// That comparison was made by eye once more in the run that added this, and it
// nearly went wrong in the direction the rule exists for: the package read
// 2.778s against a floor of 2.78s, which LOOKS like a reading under the floor
// and is the floor — an end is a reading rounded outward to the record's two
// decimals, so at the precision the band is written to those are the same
// number. A person without that rule in front of them re-takes a floor that
// was reached, which is the false verdict every paragraph in this record
// warns about.
//
// # Why the reading comes from the environment and not from a nested run
//
// The obvious alternative is an opt-in arm that runs `go test` itself and
// reads the `ok` line. Measured before it was written: three nested runs of
// wasm/verify read 2.751–2.902s where nine plain runs read 2.778–2.891s, and
// one of the three was under the recorded floor. A figure taken by a `go test`
// running underneath another process is a figure from a different invocation,
// and "the method is part of the reading" is the thing this record repeats
// most often. A nested run would have added a new measurement in order to
// check an old one.
//
// So the reading is the one `go test` already printed, for the run the person
// actually made, and the only new thing is the arithmetic — which is
// againstBandGiven, the same sentence the other record's arms print.
//
// # What it cannot check
//
// That the figure came from a run of this package ALONE. `go test ./...` runs
// package binaries in parallel and reports a figure a band's width above this
// one; nothing here can tell one from the other, which is the same limit the
// verdict lever has and is stated for the same reason. The command below is
// the one to take it from.
func TestTheFigureGoTestPrintedForThisPackageIsPlacedInItsBand(t *testing.T) {
	raw := os.Getenv(packageReadingEnv)
	if raw == "" {
		t.Skipf("%s is not set. This places the figure `go test` prints for "+
			"this package — the one reading in verifyTimingsTakenOn that no test can see — "+
			"in the band the record carries for it:\n\n"+
			"    go test -count=1 ./wasm/verify\n"+
			"    %s=<that figure> go test -count=1 -v -run "+
			"TestTheFigureGoTestPrinted ./wasm/verify\n\n"+
			"Two commands because the first one's own wall clock is what is "+
			"being placed, and only the `go` command can see it.",
			packageReadingEnv, packageReadingEnv)
	}
	// A duration, spelled the way `go test` spells it: `2.891s`. Parsed rather
	// than scanned for digits, so that a figure copied with its unit attached
	// is the form that works and a bare number is refused rather than read as
	// nanoseconds.
	got, err := time.ParseDuration(raw)
	if err != nil || got <= 0 {
		t.Fatalf("%s=%q is not a duration this can read (%v).\n\n"+
			"It takes the figure `go test` prints, with its unit: `2.891s`. "+
			"Refused rather than guessed at, for the reason the verdict "+
			"lever is spelled out rather than defaulted — a value nothing "+
			"understood should say so instead of placing a reading nobody "+
			"took.", packageReadingEnv, raw, err)
	}
	// TrimLeft because againstBandGiven's sentence is built to land at the end
	// of a line that already reports a reading, and here it IS the line.
	t.Log(strings.TrimLeft(againstBandGiven("verifyTimingsTakenOn.wholeFile",
		verifyTimingsTakenOn.wholeFile, recordMachineDiffers(), got), "\n"))
}
