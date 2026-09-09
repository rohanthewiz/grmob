package pinfixture

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"testing"
	"unicode"
)

// The Compose half of the pin, executed.
//
// Everything here runs MeasureCompose — the transcription of foundation-layout's
// zero-weight measure loop — and asks it the questions core.ShrinkNone's doc
// comment and Renderer.kt's pinMainAxis make claims about. Until this package
// existed those claims were prose in three files and a compile.
//
// The CSS half is not here. GrMobFlexSolver is Swift and ios/verify executes it;
// the cross-target comparison is checkPinnedRow in ios/verify/pin.swift, which
// gets these same cases through the transcript. What this file settles is what
// Go can settle alone: that the transcription says what the header's table says
// it says, and that the fixture cannot pass by measuring nothing.

// The table in this package's header, held to the code that produces it.
//
// Written out rather than recomputed, because the table is the claim. Every
// number in it was derived by hand from androidx's loop, and the whole value of
// the transcription is that the derivation is now executable — so an edit to
// MeasureCompose that changes an answer has to change these literals too, where
// a reader can see which answer moved.
func TestTheFixtureIsTheTableInTheHeader(t *testing.T) {
	want := []struct {
		what    string
		offered []int
		mains   []int
		gaps    []int
		rowMain int
		css     []int
		agrees  bool
		// gapsAgree is whether the Row inserted the spacing CSS inserts. It is
		// vacuously true wherever the Row's own gap is 0, which is every row
		// but the last two — and those two are the only reason the column
		// exists.
		gapsAgree bool
	}{
		// The control. Nothing is pinned, and the divergence from CSS
		// (24/80/16) is already here: Compose gives the first child everything
		// it asked for and the last one nothing.
		{"no pin: every child shrinks", []int{120, 60, 0}, []int{60, 60, 0},
			[]int{0, 0, 0}, 120, []int{24, 80, 16}, false, true},
		// The pin first. `fixedSpace` passes the Row's maximum on the first
		// child, so both siblings are offered 0 — which is the same three
		// numbers CSS produces, by a different route.
		{"the pinned child first", []int{120, 0, 0}, []int{200, 0, 0},
			[]int{0, 0, 0}, 200, []int{200, 0, 0}, true, true},
		// The pin in the middle: offered 60 and ignoring it.
		{"the pinned child between its siblings", []int{120, 60, 0}, []int{60, 200, 0},
			[]int{0, 0, 0}, 260, []int{0, 200, 0}, false, true},
		// The pin last: offered 20 and ignoring it. The most direct statement
		// of "remaining is ignored", since the number ignored is neither the
		// whole offer nor nothing.
		{"the pinned child last", []int{120, 60, 20}, []int{60, 40, 200},
			[]int{0, 0, 0}, 300, []int{0, 0, 200}, false, true},
		// The spacing row. Both columns of extents are the pin-middle row's,
		// unchanged, which is what makes this a statement about spacing: the
		// Row charges 8 after the lead child and nothing after the pin, and a
		// CSS flex line charges 8 in both places. The Row is therefore 8 wider
		// than its children and 8 narrower than a flex line laying the same
		// three out.
		{"the pinned child between its siblings, with spacing", []int{120, 52, 0},
			[]int{60, 200, 0}, []int{8, 0, 0}, 268, []int{0, 200, 0}, false, false},
		// The partial gap, which is the middle arm of the same min and the one
		// no other row reaches. Both columns of extents are the pin-last row's,
		// unchanged: the lead child is offered 120 and the tail 44, and both
		// fit. The Row charges 16 after the lead child, has 4 left when it
		// reaches the tail child, and charges 4 — neither the spacing nor zero,
		// which is the whole reason this row is here.
		{"the pinned child last, with partial spacing", []int{120, 44, 0},
			[]int{60, 40, 200}, []int{16, 4, 0}, 320, []int{0, 0, 200}, false, false},
	}

	cases := Cases()
	if len(cases) != len(want) {
		t.Fatalf("%d cases, and the header's table has %d rows", len(cases), len(want))
	}
	for i, c := range cases {
		w := want[i]
		if c.What != w.what {
			t.Errorf("case %d is %q and the table's row %d is %q — the two have been "+
				"reordered apart, so every comparison below is against the wrong row",
				i, c.What, i, w.what)
			continue
		}
		if !equal(c.Compose.Offered, w.offered) {
			t.Errorf("%s: the Row offered %v, and the table says %v", c.What,
				c.Compose.Offered, w.offered)
		}
		if !equal(c.Compose.Mains, w.mains) {
			t.Errorf("%s: the children came back at %v, and the table says %v", c.What,
				c.Compose.Mains, w.mains)
		}
		if !equal(c.Compose.Gaps, w.gaps) {
			t.Errorf("%s: the Row inserted %v of spacing and the table says %v. "+
				"spaceAfterLastNoWeight is min(spacing, what is left), which is the one "+
				"line of the measure policy nothing measured until there was a case "+
				"with a gap in it.", c.What, c.Compose.Gaps, w.gaps)
		}
		if !equal(c.CSS, w.css) {
			t.Errorf("%s: the fixture states the CSS extents as %v and the table says "+
				"%v. Nothing here computes them — a solver and a browser do — so this "+
				"literal is the claim both of them are held to.", c.What, c.CSS, w.css)
		}
		if c.Compose.RowMain != w.rowMain {
			t.Errorf("%s: the Row reports %d, and the table says %d", c.What,
				c.Compose.RowMain, w.rowMain)
		}
		if c.MainsAgreeWithCSS != w.agrees {
			t.Errorf("%s: MainsAgreeWithCSS is %v and the table says %v", c.What,
				c.MainsAgreeWithCSS, w.agrees)
		}
		if c.GapsAgreeWithCSS != w.gapsAgree {
			t.Errorf("%s: GapsAgreeWithCSS is %v and the table says %v", c.What,
				c.GapsAgreeWithCSS, w.gapsAgree)
		}
	}
}

// core.FlexShrink(0) means the same thing wherever the child sits.
//
// This is the sentence Renderer.kt's pinMainAxis makes ("order does not matter
// to the pinned child: `remaining` is ignored whether it is the first child or
// the last") and the one nothing executed. Both halves are asserted, because
// the first alone is satisfiable by a Row that never squeezed anybody:
//
//	the child keeps its base   in all three positions
//	the offer was smaller      in two of them — so the pin refused a real
//	                           squeeze rather than agreeing with an offer that
//	                           happened to be big enough
func TestThePinnedChildKeepsItsBaseWhereverItSits(t *testing.T) {
	refused := 0
	positions := 0
	// Which index the pin sits at, across the whole fixture. Counted as a SET
	// rather than as a total, because the fixture now holds more pinned rows
	// than there are positions — the spacing row is the pin-middle row again
	// with a gap — and "one in each of the three positions" is the claim, not
	// "three pinned children".
	at := map[int]bool{}
	for _, c := range Cases() {
		for i, child := range c.Children {
			if !child.Pinned {
				continue
			}
			positions++
			at[i] = true
			if got := c.Compose.Mains[i]; got != child.Base {
				t.Errorf("%s: the pinned child came back at %d and its base is %d. "+
					"core.FlexShrink(0) is a refusal to shrink, and on this target it is "+
					"the only shrink declaration that means anything at all — a pinned "+
					"child that shrinks is the declaration doing nothing, which is the "+
					"state Compose was in before pinMainAxis existed.",
					c.What, got, child.Base)
			}
			if c.Compose.Offered[i] < child.Base {
				refused++
			}
		}
	}
	if len(at) != 3 {
		t.Errorf("the fixture pins a child at %d of the three positions (%d pinned rows "+
			"in all), and the point of it is one in each: first, between its siblings "+
			"and last. `remaining` is ignored wherever the pin sits, and a position "+
			"nothing occupies is a position nothing says that about.", len(at), positions)
	}
	if refused < 2 {
		t.Errorf("the pin refused a smaller offer in only %d of %d positions. In the "+
			"rest the Row offered at least the child's base, so the child would have "+
			"kept its size with no pin at all and those rows say nothing about the "+
			"declaration.", refused, positions)
	}
}

// And the same child, one declaration apart.
//
// The control row exists to make the rows above a statement about
// core.FlexShrink(0) rather than about a Row with a big child in it. Same
// container, same three children, same order — only the flag differs.
func TestWithoutThePinTheSameChildIsSqueezed(t *testing.T) {
	control := Cases()[0]
	pinned := Cases()[2] // the same order, with the middle child pinned

	i := 1
	if control.Children[i].Pinned || !pinned.Children[i].Pinned {
		t.Fatalf("the control and the pinned-middle case are no longer one flag apart: "+
			"child %d is pinned=%v in %q and pinned=%v in %q",
			i, control.Children[i].Pinned, control.What, pinned.Children[i].Pinned, pinned.What)
	}
	if control.Children[i].Base != pinned.Children[i].Base {
		t.Fatalf("the two rows no longer hold the same child: base %d and %d",
			control.Children[i].Base, pinned.Children[i].Base)
	}
	if control.Compose.Mains[i] >= pinned.Compose.Mains[i] {
		t.Errorf("the child is %d wide unpinned and %d wide pinned. The pin is supposed "+
			"to be the difference between being measured against what is left and being "+
			"measured against its own content; if the two are the same, this fixture is "+
			"measuring a Row that was not squeezing anybody.",
			control.Compose.Mains[i], pinned.Compose.Mains[i])
	}
}

// A child after an overflow is offered nothing, and that is the divergence.
//
// CSS shares a deficit out among every item that can shrink, so a flex line's
// sizes do not depend on the order the children are written in. Compose's Row
// measures each unweighted child against `mainAxisMax - fixedSpace` — what the
// ones before it did not take — so its answer does. Two facts make that
// concrete here, and each fails on its own:
//
//	the arithmetic       once fixedSpace passes the Row's maximum, every later
//	                     child is offered exactly 0 and comes back at 0
//	the consequence      the same three children in three different orders get
//	                     three different sets of sizes
func TestComposeMeasuresEachChildAgainstWhatIsLeft(t *testing.T) {
	for _, c := range Cases() {
		// What the children before this one took: their extents AND the spacing
		// charged after each of them, which is what fixedSpace accumulates.
		taken := 0
		for i := range c.Children {
			if want := max(c.Offer-taken, 0); c.Compose.Offered[i] != want {
				t.Errorf("%s: child %d was offered %d and the space the ones before it "+
					"left is %d. That subtraction IS the Compose shrink rule — there is "+
					"no factor anywhere in it — so an offer that is anything else means "+
					"the transcription has stopped mirroring the loop it names.",
					c.What, i, c.Compose.Offered[i], want)
			}
			if c.Compose.Offered[i] == 0 && c.Compose.Mains[i] != 0 && !c.Children[i].Pinned {
				t.Errorf("%s: child %d was offered 0 and came back at %d. Only a pinned "+
					"child may ignore its offer; an unpinned one is a SizeElement with "+
					"enforceIncoming, which clamps into the incoming range.",
					c.What, i, c.Compose.Mains[i])
			}
			taken += c.Compose.Mains[i] + c.Compose.Gaps[i]
		}
	}

	// And the order-dependence itself, which is what the CSS side is compared
	// against. The three pinned rows hold the same three children; if their
	// answers ever coincide, either the transcription has stopped depending on
	// order or the fixture has stopped being an overflow.
	//
	// Grouped by the Row's own spacing, because the fixture holds one row that
	// is a REPEAT of an order rather than a new one: the spacing case is the
	// pin-middle case with a gap, and its extents are deliberately identical.
	// Comparing it with its twin would report the thing it was added to show.
	seen := map[string]string{}
	for _, c := range Cases()[1:] {
		key := itoa(c.Gap) + "|" + sizesByName(c)
		if prev, ok := seen[key]; ok {
			t.Errorf("%q and %q produce the same sizes (%s). The whole of the recorded "+
				"divergence is that Compose's answer depends on the order and CSS's does "+
				"not, and two orders agreeing takes half of that away.", prev, c.What, key)
		}
		seen[key] = c.What
	}
}

// The Row reports the overflow instead of clipping it.
//
// mainAxisLayoutSize is max(content, mainAxisMin) and is never coerced down to
// mainAxisMax, and Size.kt's own node reports layout(placeable.width, …)
// unclamped. So a fixed-width Row whose children overflow is measured wider
// than it was told to be, which is what makes the pinned child visible rather
// than cut off — the Compose analogue of `overflow: visible`.
//
// This is also where the mis-spelling of pinMainAxis that mobile/verify refuses
// by name would show up. Reporting `constraints.constrain(...)` instead of the
// measured size keeps fixedSpace inside the Row's maximum, so RowMain would
// never pass Offer and the child would be drawn spilling out of a box the Row
// believed it fitted inside.
func TestTheRowReportsTheOverflowRatherThanClippingIt(t *testing.T) {
	overflowed := 0
	for _, c := range Cases() {
		// The children and the spacing that survived the clamp — fixedSpace,
		// which is what mainAxisLayoutSize is the max of.
		content := 0
		for i, m := range c.Compose.Mains {
			content += m + c.Compose.Gaps[i]
		}
		if want := max(content, c.Offer); c.Compose.RowMain != want {
			t.Errorf("%s: the Row reports %d, and its children occupy %d of an offered "+
				"%d. mainAxisLayoutSize is max(fixedSpace, mainAxisMin) with no upper "+
				"bound; a Row clamped to its own maximum is one that clips a pinned "+
				"child instead of overflowing with it.",
				c.What, c.Compose.RowMain, content, c.Offer)
		}
		if c.Compose.RowMain > c.Offer {
			overflowed++
		}
	}
	if want := len(Cases()) - 1; overflowed != want {
		t.Errorf("%d of the %d rows overflow their container, and every pinned one is "+
			"supposed to: a pinned child whose base exceeds the Row's extent cannot fit "+
			"inside it by construction, and the control is the only row without one",
			overflowed, len(Cases()))
	}
}

// The spacing after a child collapses once the Row has overflowed.
//
// spaceAfterLastNoWeight is min(arrangementSpacing, what is left), so a Row that
// has already spent its main axis inserts no gap after the child that spent it.
// Nothing in the census said so and it follows from the same two lines the rest
// of this rests on — so it is transcribed, and exercised here rather than in the
// cases, which all carry gap 0 so that the cross-target comparison is about
// shrink and not about spacing.
func TestTheSpacingCollapsesOnceTheRowHasOverflowed(t *testing.T) {
	children := []Child{{Name: "lead", Base: 60}, {Name: "pinned", Base: 200, Pinned: true},
		{Name: "tail", Base: 40}}

	// With room to spare, the gap is charged in full: each child is offered
	// what the ones before it took, gaps included. (The Row itself reports
	// 1000 either way — a definite width is a minimum as well as a maximum,
	// which is why the offers are what this half reads.)
	roomy := MeasureCompose(1000, 8, children)
	for i, want := range []int{1000, 1000 - (60 + 8), 1000 - (60 + 8 + 200 + 8)} {
		if roomy.Offered[i] != want {
			t.Errorf("with room to spare, child %d was offered %d and the children plus "+
				"gaps before it come to %d", i, roomy.Offered[i], 1000-want)
		}
	}

	// Overflowing, the gap after the pinned child has nowhere to come from: the
	// Row is the lead child, one gap, and the pinned child, with nothing after.
	tight := MeasureCompose(120, 8, children)
	if want := 60 + 8 + 200; tight.RowMain != want {
		t.Errorf("overflowing, the Row is %d wide and the gaps that fit come to %d. "+
			"min(spacing, what is left) is the source line; a Row that kept spending "+
			"spacing it did not have would report a width no child accounts for.",
			tight.RowMain, want)
	}
	if tight.Offered[2] != 0 {
		t.Errorf("the child after the overflow was offered %d rather than 0",
			tight.Offered[2])
	}
}

// MainsAgreeWithCSS is a fact about the case, not a flag somebody set.
//
// The same argument bandfixture makes for SharesADeficit: the Swift side
// asserts the flag in both directions, so a flag that drifted from the case it
// describes would turn a real divergence into an expected agreement and nothing
// would report it. Derived here from the two properties the header's reasoning
// actually turns on.
func TestTheAgreementIsWhetherThePinComesFirst(t *testing.T) {
	for _, c := range Cases() {
		natural, shrinkable := 0, 0
		for _, ch := range c.Children {
			natural += ch.Base
			if !ch.Pinned {
				shrinkable += ch.Base
			}
		}
		// The gaps are used space in a flex line and belong in the deficit.
		// Nothing in this fixture turns on it — the deficit clears the
		// shrinkable total either way — and it is here because a rule that is
		// right about these numbers by omission is not the rule.
		natural += c.Gap * max(len(c.Children)-1, 0)

		// And the flag against the two columns it is about. This is the join
		// the fixture did not used to have: the CSS extents were three numbers
		// in a comment, the Compose ones came out of MeasureCompose, and the
		// flag was derived from the pin's position — three statements, no two
		// of which were ever compared. A wrong literal in the CSS column now
		// fails here, before a solver or a browser ever sees it.
		if got := equal(c.CSS, c.Compose.Mains); got != c.MainsAgreeWithCSS {
			t.Errorf("%s: the stated CSS extents are %v and Compose measures %v, which "+
				"%s — and MainsAgreeWithCSS says %v.\n\n"+
				"The flag is derived from where the pin sits and the columns are stated "+
				"and computed. All three are supposed to be about one Row; two of them "+
				"agreeing is not enough, because the third is what a solver and a "+
				"browser are held to.",
				c.What, c.CSS, c.Compose.Mains,
				map[bool]string{true: "are the same numbers", false: "are not"}[got],
				c.MainsAgreeWithCSS)
		}
		// Two conditions, and the flag is their conjunction. The first is the
		// position; the second is the premise that makes the position mean
		// what the header says it means — CSS clamps every shrinkable child to
		// zero exactly when the deficit is at least their combined base.
		clampsAll := natural-c.Offer >= shrinkable
		if c.Children[0].Pinned && !clampsAll {
			t.Errorf("%s: the pin comes first and the deficit (%d) no longer reaches the "+
				"shrinkable children's combined base (%d), so the row that is supposed "+
				"to be the AGREEING one has stopped being it. bandfixture states an "+
				"unbadged band for the same reason: an 'it diverges' with no case that "+
				"does not is a claim about whatever happened.",
				c.What, natural-c.Offer, shrinkable)
		}
		if want := c.Children[0].Pinned && clampsAll; c.MainsAgreeWithCSS != want {
			t.Errorf("%s: MainsAgreeWithCSS is %v; the first child is pinned=%v and the "+
				"deficit (%d) %s the shrinkable children's combined base (%d).\n\n"+
				"Compose only reaches CSS's answer when the pin overflows the Row before "+
				"any sibling has been offered anything, AND CSS has nothing left to give "+
				"those siblings either. The Swift side asserts this flag in both "+
				"directions, so one that drifted would turn a real divergence into an "+
				"expected agreement with nothing reporting it.",
				c.What, c.MainsAgreeWithCSS, c.Children[0].Pinned,
				natural-c.Offer, map[bool]string{true: "reaches", false: "falls short of"}[clampsAll],
				shrinkable)
		}
	}
}

// And the guard against a fixture that measures nothing.
//
// Validate is what a harness calls before trusting the cases. Its arms cannot
// be reached through Cases(), which is the point of testing it against rows
// built to have the fault: a guard nothing can fail is a guard nobody has
// checked.
// The resolution is the smallest gap between two of a case's numbers, and it is
// what two harnesses' two different tolerances are both held to.
//
// # Why this needs a test of its own
//
// The value is consumed on two targets and produced here, and its whole job is
// to be a FLOOR. A resolution that came out too large would raise the ceiling
// under both tolerances silently — every check would go on passing, and the one
// property either tolerance has ("it cannot confuse two of the fixture's own
// numbers") would have stopped being held. So the derivation is asked directly:
// against the numbers of the case the reader can see, and against a case built
// to have a resolution nobody would want.
func TestTheResolutionIsTheClosestTwoNumbersCome(t *testing.T) {
	// The partial-spacing row is the tightest, and the reason is the middle arm
	// of the collapse: it charges a gap of 4 where the Row declares 16 and a
	// fully collapsed gap would be 0. Those three numbers are what every
	// tolerance over this fixture has to be able to tell apart.
	var tightest Case
	for _, c := range Cases() {
		if tightest.What == "" || c.Resolution < tightest.Resolution {
			tightest = c
		}
	}
	if tightest.Resolution != 4 {
		t.Errorf("the tightest case is %q at %d and the fixture's closest pair of "+
			"numbers is the partial gap 4 against the 0 beside it. A floor larger than "+
			"the real one lets both harnesses widen past what they can actually "+
			"resolve.", tightest.What, tightest.Resolution)
	}

	// And every case is checked against its own numbers rather than against
	// this one reading of them.
	//
	// Over caseNumbers' READ entries, which is what the derivation ranges over
	// — spelled here as a second walk of the same table rather than as a second
	// list of fields, so this stays a check of the arithmetic and the coverage
	// question belongs to the test below.
	for _, c := range Cases() {
		var nums []int
		for _, r := range caseNumbers(c) {
			if r.Read != "" {
				nums = append(nums, r.Value)
			}
		}
		closest := 0
		for i := range nums {
			for j := range nums {
				d := nums[i] - nums[j]
				if d < 0 {
					d = -d
				}
				if d == 0 {
					continue
				}
				if closest == 0 || d < closest {
					closest = d
				}
			}
		}
		if c.Resolution != closest {
			t.Errorf("%q states a resolution of %d and the closest two of its numbers "+
				"come is %d", c.What, c.Resolution, closest)
		}
		if c.Resolution <= 0 {
			t.Errorf("%q resolves nothing (%d), so any tolerance at all is within a "+
				"factor of it and both harnesses' guards pass on arithmetic rather than "+
				"on a measurement", c.What, c.Resolution)
		}
	}
}

func TestValidateRefusesAFixtureThatCannotOverflow(t *testing.T) {
	if err := Validate(); err != nil {
		t.Fatalf("the fixture itself does not validate: %v", err)
	}

	for _, c := range []struct {
		name, mentions string
		bad            Case
	}{
		{
			name:     "a Row wide enough for its children",
			mentions: "nothing overflows",
			bad: Case{What: "roomy", Offer: 1000, Children: []Child{
				{Base: 60}, {Base: 200, Pinned: true}, {Base: 40}},
				CSS: []int{60, 200, 40}},
		},
		{
			name:     "a Row with nobody on either side of the pin",
			mentions: "fewer than three",
			bad: Case{What: "lonely", Offer: 120, Children: []Child{
				{Base: 60}, {Base: 200, Pinned: true}},
				CSS: []int{0, 200}},
		},
		{
			// The stated column, about a different Row from the one it is
			// printed beside. Both harnesses index it against the children, so
			// this arrives there as a crash or as a comparison that quietly
			// runs short — and the census prints it either way.
			name:     "a CSS column that is not about these children",
			mentions: "CSS extents",
			bad: Case{What: "mismatched", Offer: 120, Children: []Child{
				{Base: 60}, {Base: 200, Pinned: true}, {Base: 40}},
				CSS: []int{0, 200}},
		},
	} {
		err := validate([]Case{c.bad})
		if err == nil {
			t.Errorf("%s: validate accepted it, so a fixture edited into that shape "+
				"would go on passing every check over it", c.name)
			continue
		}
		if !strings.Contains(err.Error(), c.mentions) {
			t.Errorf("%s: validate refused it with %q, which does not say %q — the "+
				"message is the only thing telling the next reader which property went",
				c.name, err, c.mentions)
		}
	}
}

// equal is a slice comparison spelled out rather than reflect.DeepEqual, so a
// nil and an empty slice are the same thing here: MeasureCompose allocates for
// every case and a zero-length one is not a state the fixture has.
func equal(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// sizesByName keys a case by what each NAMED child got, so two orderings can be
// compared without the position doing the telling apart — which is exactly the
// property under test.
func sizesByName(c Case) string {
	var b strings.Builder
	for _, name := range []string{"lead", "pinned", "tail"} {
		for i, ch := range c.Children {
			if ch.Name == name {
				b.WriteString(name)
				b.WriteByte('=')
				b.WriteString(itoa(c.Compose.Mains[i]))
				b.WriteByte(' ')
			}
		}
	}
	return b.String()
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var d []byte
	for n > 0 {
		d = append([]byte{byte('0' + n%10)}, d...)
		n /= 10
	}
	return string(d)
}

// And the census's table is this fixture's numbers.
//
// docs/platforms/native.md prints the four rows above as a markdown table, and a
// doc table is exactly the kind of thing that stops being true quietly: the
// numbers are the whole content of the section, nothing compiles against them,
// and the fixture they came from is three directories away. This is the same
// arrangement wasm/verify makes for the fixed-size census — one fact, several
// places, pinned where the fact lives.
//
// The rows are found by their leading cell rather than by position, so a section
// that grows a row or reorders one is not a failure; a row whose numbers stopped
// matching is.
func TestTheCensusTableIsTheFixture(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "docs", "platforms", "native.md"))
	if err != nil {
		t.Fatalf("reading the census: %v", err)
	}
	doc := string(raw)

	// Which table row each case is printed as. Kept beside the cases rather
	// than derived from What, because the prose names the arrangements the way
	// a reader would (`pin first [P,A,B]`) and the fixture names them the way
	// the code does.
	rows := map[string]string{
		"no pin: every child shrinks":                         "| no pin |",
		"the pinned child first":                              "| pin first `[P,A,B]` |",
		"the pinned child between its siblings":               "| pin middle `[A,P,B]` |",
		"the pinned child last":                               "| pin last `[A,B,P]` |",
		"the pinned child between its siblings, with spacing": "| pin middle, 8px gap `[A,P,B]` |",
		"the pinned child last, with partial spacing":         "| pin last, 16px gap `[A,B,P]` |",
	}

	for _, c := range Cases() {
		lead, ok := rows[c.What]
		if !ok {
			t.Errorf("%q has no row in the census's table. Every case here is one row of "+
				"it; a case with none is a measurement nothing tells the reader about.",
				c.What)
			continue
		}
		at := strings.Index(doc, lead)
		if at < 0 {
			t.Errorf("docs/platforms/native.md has no table row beginning %q. If the "+
				"section moved, re-point this; if it was rewritten, the table is no "+
				"longer this fixture's numbers and nothing else says so.", lead)
			continue
		}
		row := doc[at : at+strings.IndexByte(doc[at:], '\n')]

		// Both columns, which is newer than the rest of this and is the point
		// of the CSS field existing.
		//
		// The Compose column has always been held here: this package computes
		// it, so the comparison is between a doc and the code that produced the
		// number. The CSS column is not computed here and is not going to be —
		// a flex line transcribed into Go would be a third spelling after the
		// solver and the browser. What made it holdable is that the fixture now
		// STATES it, so this compares the census's prose with the claim two
		// executable things are held to, rather than with nothing.
		if got, want := cellAt(row, 1), joinInts(c.CSS); got != want {
			t.Errorf("%s: the census's CSS column reads %q and the fixture states %q.\n\n"+
				"Nothing in Go computes these — ios/verify solves them through "+
				"GrMobFlexSolver and wasm/verify's check 12 measures them in a browser — "+
				"so the fixture's field is the claim and this is whether the census "+
				"prints it. The control row is the sharpest case: 24/80/16 is the one "+
				"row no other claim determines.", c.What, got, want)
		}
		if got, want := cellAt(row, 2), joinInts(c.Compose.Mains); got != want {
			t.Errorf("%s: the census's Compose column reads %q and the fixture measures "+
				"%q.\n\nThe table is the whole content of that section and nothing "+
				"compiles against it, so a number that drifts there is a paragraph "+
				"describing a layout no target produces.", c.What, got, want)
		}
	}

	// And the spacing table below it, which is one row per gapped case.
	//
	// Found by walking the fixture rather than by taking the last case, which
	// is what this did while there was one gapped row. There are two now — the
	// ends of min(spacing, what is left) and its middle — and "the last one"
	// would go on passing with the other printed nowhere.
	spacingRows := map[string]string{
		"the pinned child between its siblings, with spacing": "| pin middle, 8px gap |",
		"the pinned child last, with partial spacing":         "| pin last, 16px gap |",
	}
	gapped, middle := 0, false
	for _, spacing := range Cases() {
		if spacing.Gap == 0 {
			continue
		}
		gapped++
		// Whether this row charges a gap that is neither the Row's spacing nor
		// zero. The trailing entry is excluded for the reason the CSS column
		// excludes it: with n children there are n-1 places a gap can go.
		for i := 0; i+1 < len(spacing.Compose.Gaps); i++ {
			if g := spacing.Compose.Gaps[i]; g > 0 && g < spacing.Gap {
				middle = true
			}
		}

		lead, ok := spacingRows[spacing.What]
		if !ok {
			t.Errorf("%q carries a gap of %d and has no row in the census's spacing "+
				"table. The collapse is a sentence three documents repeat and that "+
				"table is the only place it is written as numbers.",
				spacing.What, spacing.Gap)
			continue
		}
		at := strings.Index(doc, lead)
		if at < 0 {
			t.Errorf("docs/platforms/native.md has no row beginning %q. If the section "+
				"moved, re-point this; if it was rewritten, the gap collapse is prose "+
				"again and nothing else says otherwise.", lead)
			continue
		}
		row := doc[at : at+strings.IndexByte(doc[at:], '\n')]

		// CSS charges the gap between every adjacent pair whatever happened to
		// the children, so its cell is the Row's own gap repeated. Written out
		// from the fixture rather than as a literal, so a gap changed there
		// moves the claim rather than making this test wrong about it.
		cssGaps := make([]int, len(spacing.Children)-1)
		for i := range cssGaps {
			cssGaps[i] = spacing.Gap
		}
		if got, want := cellAt(row, 1), joinInts(cssGaps); got != want {
			t.Errorf("%s: the census's spacing table gives CSS %q and a flex line "+
				"charges %q — its gap between every adjacent pair, whatever the "+
				"children did", spacing.What, got, want)
		}
		if got, want := cellAt(row, 2), joinInts(spacing.Compose.Gaps[:len(cssGaps)]); got != want {
			t.Errorf("%s: the census's spacing table gives Compose %q and the fixture "+
				"measures %q. spaceAfterLastNoWeight is min(spacing, what is left).",
				spacing.What, got, want)
		}
	}

	// And both kinds of gapped row, because a min has three answers and one row
	// reaches only two of them. With the spacing charged in full or not at all,
	// "charge the gap unless the Row has overflowed" produces every number in
	// the table — so a fixture without the middle arm cannot tell the rule it
	// transcribes from a simpler one that is wrong.
	if gapped < 2 {
		t.Errorf("%d of the fixture's rows carry a gap, and two are needed: one whose "+
			"charged gaps are the spacing or zero, and one where a charged gap is "+
			"neither.", gapped)
	}
	if !middle {
		t.Errorf("no gapped row charges a gap strictly between zero and the Row's own " +
			"spacing. That value is the whole content of the second spacing row (see " +
			"partialGap): it is what separates min(spacing, what is left) from a rule " +
			"that charges the gap until the Row overflows.")
	}
}

// cellAt pulls one cell out of a markdown table row and strips the bold markers
// the census uses to point at the pinned child.
//
// Deliberately positional — the census's tables have exactly three columns, the
// row's name then CSS then Compose — because a general markdown parser is not
// what this needs and would be far more code than the thing it checks. It used
// to take the LAST cell, which was the same thing while only one column was
// held; indexing says which column a failure is about, and a table that grew one
// lands here as a mismatch naming the row rather than as a silent shift.
func cellAt(row string, i int) string {
	cells := strings.Split(strings.Trim(row, "|"), "|")
	if i < 0 || i >= len(cells) {
		return ""
	}
	return strings.TrimSpace(strings.ReplaceAll(cells[i], "**", ""))
}

// joinInts spells a case's extents the way the census's cells do.
func joinInts(v []int) string {
	parts := make([]string, len(v))
	for i, n := range v {
		parts[i] = itoa(n)
	}
	return strings.Join(parts, ", ")
}

// Every number a Case carries says which check reads it.
//
// # The gap this closes
//
// resolution derives the floor two harnesses hold their tolerances to, and it
// used to derive it over every number in the case: offer, gap, both columns,
// every base, every measured extent. The argument for that was a sentence —
// "every number in the case is a number some assertion on some target compares
// a measurement against" — and it is a claim about two files in two other
// languages, made in a package that reads neither.
//
// True today, and true by nobody's decision tomorrow. A column added to
// Measured for a reader's benefit, a second offer, an intrinsic size stated for
// documentation: each would tighten a bound that ios/verify and browser.mjs
// both read, for a distinction no check is asked to make. And the failure would
// arrive as two harnesses being told their tolerances are too loose, in a file
// that says nothing about either.
//
// So caseNumbers names every number and the assertion that holds a measurement
// to it, and this holds that table to the struct. The walk is reflective on
// purpose: a field added to Case, Child or Measured appears here without
// anybody remembering to add it, which is the whole point — the failure is
// "somebody has to say whether a check reads this", which is a decision, and
// the alternative is a floor that moves on its own.
func TestEveryNumberInACaseSaysWhichCheckReadsIt(t *testing.T) {
	// Which numeric fields of a Case caseNumbers is supposed to cover, and the
	// one it deliberately does not.
	//
	// The value is whether the field is one of the numbers a harness compares
	// against. A field marked false is a number in the fixture that nothing
	// reads as a distinction, and it must not tighten the floor.
	covered := map[string]bool{
		"Offer":             true,
		"Gap":               true,
		"Children[].Base":   true,
		"CSS[]":             true,
		"Compose.Offered[]": true,
		"Compose.Mains[]":   true,
		"Compose.Gaps[]":    true,
		"Compose.RowMain":   true,
		// The floor itself, derived FROM the readings. A number that described
		// the others would be one of them, and this one is their answer.
		"Resolution": false,
	}

	for _, c := range Cases() {
		found := map[string][]int{}
		walkPinNumbers(t, "", reflect.ValueOf(c), found)

		for path := range found {
			if _, stated := covered[path]; !stated {
				t.Errorf("a Case carries the number %s and nothing says whether a check "+
					"reads it.\n\n"+
					"resolution derives the floor ios/verify's pinEpsilon and "+
					"browser.mjs's PIN_EPSILON are both held to, over the numbers some "+
					"assertion compares a measurement against. A new number is either "+
					"one of those — in which case caseNumbers has to name the assertion "+
					"— or it is not, in which case saying so here is what keeps it from "+
					"tightening a bound two other languages read for a distinction "+
					"nobody makes.", path)
			}
		}
		for path := range covered {
			if _, ok := found[path]; !ok {
				t.Errorf("this test says %s is one of a Case's numbers and the struct no "+
					"longer has it. A stale row here is a coverage claim about a field "+
					"that is gone", path)
			}
		}

		// And the two directions between the fields and the readings, which is
		// what makes "covered" a fact rather than a list.
		read := map[int]bool{}
		for _, r := range caseNumbers(c) {
			if r.Read != "" {
				read[r.Value] = true
			}
		}
		for path, values := range found {
			if !covered[path] {
				continue
			}
			for _, v := range values {
				if !read[v] {
					t.Errorf("%q carries %s = %d and caseNumbers produces no read entry "+
						"with that value. The table is what resolution ranges over, so a "+
						"number it does not carry is a distinction the floor does not "+
						"protect — and both harnesses would widen past it.",
						c.What, path, v)
				}
			}
		}
		inFields := map[int]bool{}
		for path, values := range found {
			if !covered[path] {
				continue
			}
			for _, v := range values {
				inFields[v] = true
			}
		}
		for _, r := range caseNumbers(c) {
			if r.Read != "" && !inFields[r.Value] {
				t.Errorf("%q: caseNumbers reads %s as %d and no covered field of the "+
					"Case holds that number. A reading with no field behind it is a "+
					"distinction invented in the derivation, and it tightens the floor "+
					"for nothing.", c.What, r.What, r.Value)
			}
		}
	}
}

// walkPinNumbers collects every int a Case reaches, by the path it sits at.
//
// Ints and slices of ints only: a bool is a flag rather than a distinction, and
// a string is a name. A slice contributes one path with every element's value
// under it, because a per-element path would make the coverage table above a
// function of how many children a case has.
func walkPinNumbers(t *testing.T, path string, v reflect.Value, out map[string][]int) {
	t.Helper()
	switch v.Kind() {
	case reflect.Int:
		out[path] = append(out[path], int(v.Int()))
	case reflect.Slice:
		for i := 0; i < v.Len(); i++ {
			walkPinNumbers(t, path+"[]", v.Index(i), out)
		}
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			name := v.Type().Field(i).Name
			at := name
			if path != "" {
				at = path + "." + name
			}
			walkPinNumbers(t, at, v.Field(i), out)
		}
	}
}

// Every reading names a harness, and every harness named still spells the
// assertion.
//
// # The half of caseNumbers that was prose
//
// resolution derives the floor two harnesses hold their tolerances to, over the
// numbers those harnesses are asked to tell apart, and caseNumbers is where
// "which numbers those are" is written down. The test above holds that table to
// the STRUCT in both directions: a field with no reading fails, a reading with
// no field fails. Nothing held it to the two files it is actually about.
//
// That is the direction the table goes wrong in. A row here is a claim that
// somebody, somewhere, in another language, compares a measurement against this
// number — and the day that assertion is deleted the row stays, still saying it
// is read, still tightening a bound the other harness also obeys. The failure
// would never arrive on its own: a floor that is too tight is a floor nothing
// violates.
//
// # What is asked
//
// Each reading names its consumers and the phrase each one's assertion is
// spelled with, and this asks that consumer's source for the phrase.
//
// A phrase rather than the field's own name, because a field name is not
// evidence. browser.mjs mounts internal/bandfixture as well, whose cases have
// an Offer of their own read as `c.offer` some two thousand lines before the pin
// check does the same — so deleting the pin check's comparison outright leaves
// the token behind and this test would pass. It was tried: it does. The phrase
// is the comparison, and it goes when the comparison goes.
//
// Whitespace in a phrase matches any run of whitespace, so a reformat or a
// line-wrap keeps the claim; a rewrite of the comparison breaks it, which is
// the point at which somebody has to look at whether the number is still read.
//
// # What it is worth
//
// It says the harness still spells that comparison, not that the comparison
// still holds a measurement to this number. That is the claim a search can
// support — the same one wasm/verify's citation walk makes about a `check N` —
// and it catches the way this table actually rots.
//
// The other direction is not checked and is not a gap of the same kind: a
// harness that STARTED reading a number nothing credits it with leaves the
// floor exactly where it was, because the floor is a union over the readings
// and any one consumer is enough to keep a number in it. What that would cost
// is a sentence naming the wrong file in a failure nobody has seen yet.
func TestEveryReadingNamesAHarnessThatSpellsTheNumber(t *testing.T) {
	root := filepath.Join("..", "..")

	// Each consumer's source, read once. A path that does not resolve is a
	// consumer nobody can go and look at, which is worse than an unchecked
	// claim: the table would name a file that is not there and this test would
	// have nothing to say about any row citing it.
	// Stripped of whitespace, and the phrases are stripped the same way before
	// they are looked for. That is what makes a phrase survive a reformat: an
	// indent, a line-wrap, a space after a comma are all gone from both sides,
	// so what is left to match is the comparison itself.
	//
	// And stripped of prose first. See pinCodeOnly: a citation is a claim that
	// the harness still ASSERTS something, and both of these files explain
	// every assertion in a comment above it, so a phrase looked for in the
	// whole file can be satisfied by the sentence describing an assertion that
	// is no longer there. Lexed, 16.8% of browser.mjs and 17.3% of pin.swift
	// are code, which is the measure of how much of the search space that
	// sentence was competing with.
	source := map[string]string{}
	// The same files unlexed, and the lines the lexer was unsure about. Both
	// are read only by the deletion report at the bottom, which is where the
	// difference between "gone from the file" and "gone from the code" decides
	// what the reader is sent to look at.
	wholeSource := map[string]string{}
	// The same files as they were read, line breaks and all. Only the
	// blind-spot arm of the deletion report reads this, and only to turn "one
	// of these lines" into "this line". See pinPhraseLines.
	rawSource := map[string]string{}
	openBy := map[string][]int{}
	for name, path := range pinConsumers {
		b, err := os.ReadFile(filepath.Join(root, path))
		if err != nil {
			t.Fatalf("pinConsumers says %q is the harness %q and it cannot be read: %v.\n\n"+
				"Every row of caseNumbers that names this consumer is a claim about that "+
				"file, and with the file gone the claims are about nothing — including "+
				"the ones that set the floor ios/verify and browser.mjs both hold their "+
				"tolerances to.", path, name, err)
		}
		code, openLines := pinCodeOnly(string(b), filepath.Ext(path))
		// The lexer is a lexer and not a parser, and the one construct it
		// declines to guess at — JavaScript's regex literal — can open a
		// string that was never opened. That damage is bounded to a line by
		// construction, so a run-away shows up as a file with almost nothing
		// left rather than as a file with one assertion missing; this is the
		// reading that would say so. Measured at 16.8% and 17.3%; a floor of a
		// twentieth is far below either and far above what a lexer that lost
		// its place would leave.
		const floor = 0.05
		whole, kept := pinDense(string(b)), pinDense(code)
		if whole > 0 && float64(kept)/float64(whole) < floor {
			t.Errorf("pinCodeOnly kept %d of %s's %d non-space bytes (%.1f%%), and the "+
				"two consumers measure 16.8%% and 17.3%%.\n\n"+
				"Every citation below is looked for in what is left, so a lexer that "+
				"lost its place reports a table full of deleted assertions. The one "+
				"construct it does not lex is JavaScript's regular-expression literal — "+
				"a quote inside one opens a string nothing closes — and that is bounded "+
				"to a line, so a reading this low is something else again.",
				kept, name, whole, 100*float64(kept)/float64(whole))
		}
		// And the small version of that damage, which no percentage can show.
		//
		// The floor is a bound on catastrophe: a lexer that lost its place
		// leaves 4% of a file, and these two measure 16.8% and 17.3%. What it
		// cannot see is one line — a regex whose quote opened a string that
		// swallowed the rest of the line an assertion happens to sit on. Two
		// hundred bytes moves no percentage, and what comes out is one
		// confident "DELETED assertion" about a harness that is fine.
		//
		// pinCodeOnly reports every line it ended inside a single-line string,
		// which is the complete set of places that can start. Both consumers
		// have none, so this is a claim with a measurement behind it rather
		// than a hedge — and the day one has some, the reader is told which
		// lines before any citation below is believed.
		openBy[name] = openLines
		if len(openLines) > 0 {
			t.Errorf("pinCodeOnly ended inside an unterminated string on %d of %s's "+
				"lines: %v.\n\n"+
				"A single-line string that runs into the newline is a syntax error in "+
				"both of these languages, so in a file that compiles it is the one "+
				"construct this lexer does not lex: a JavaScript regular-expression "+
				"literal carrying an odd number of quote characters, whose first quote "+
				"opens a string that nothing closes. Everything after it on that line "+
				"was blanked, and if an assertion was spelled there its citation below "+
				"reports as DELETED — one false failure in a table of forty, pointing "+
				"at a harness that still asserts exactly what this table says it does. "+
				"Either the regex moves onto a line of its own, or pinCodeOnly learns "+
				"to lex it and pinFreeForm's Lexis sentence for %s stops saying it "+
				"does not.", len(openLines), name, openLines, filepath.Ext(path))
		}
		source[name] = pinStripSpace(code)
		// The raw file, stripped the same way. Only the deletion report reads
		// it: a phrase that is gone from the CODE and still present in the file
		// is a different finding from one that is gone altogether, and it is
		// the finding the blind spot produces.
		wholeSource[name] = pinStripSpace(string(b))
		// And unstripped, for the same report to say WHERE. Stripping is what
		// makes a phrase survive a reformat and it is also what throws the line
		// numbers away, so the arm that has to compare a phrase's line against
		// the lexer's own list keeps the file as it was read. See
		// pinPhraseLines.
		rawSource[name] = string(b)
	}

	// Which consumers anything credits at all. A harness in the closed set that
	// no reading names is either one that has stopped reading this fixture — in
	// which case its row in pinConsumers is dead weight and every claim above
	// about it is vacuous — or a table that forgot to credit it.
	credited := map[string]bool{}

	for _, c := range Cases() {
		for _, r := range caseNumbers(c) {
			if r.Read == "" {
				// An unread number carries no consumers, and must not: a row
				// that named one would be claiming a reading it also says does
				// not happen.
				if len(r.By) > 0 {
					t.Errorf("%q: %s has no Read sentence and still names %d consumers. "+
						"An unread number is one no tolerance has to tell from its "+
						"neighbour, and a consumer beside it is two answers to the same "+
						"question", c.What, r.What, len(r.By))
				}
				continue
			}
			if len(r.By) == 0 {
				t.Errorf("%q: %s is read (%s) and names no consumer.\n\n"+
					"The sentence is what a failure carries and the consumers are what "+
					"hold it to the harnesses. Without them this row is back to being a "+
					"claim about two files in two other languages, made in a package that "+
					"reads neither.", c.What, r.What, r.Read)
				continue
			}
			for by, cited := range r.By {
				path, known := pinConsumers[by]
				if !known {
					t.Errorf("%q: %s says it is read by %q and pinConsumers has no such "+
						"harness. The set is closed so that a reading cannot name a "+
						"consumer nobody can go and look at", c.What, r.What, by)
					continue
				}
				credited[by] = true
				if cited.Phrase == "" {
					t.Errorf("%q: %s names %s as a consumer and gives no phrase to find "+
						"the assertion by. The name alone is the prose this pair replaced",
						c.What, r.What, by)
					continue
				}
				if cited.Field == "" {
					t.Errorf("%q: %s cites %s as %q and names no field.\n\n"+
						"The field is what tells a reworded assertion from a deleted one "+
						"when the phrase goes missing, and without it both edits arrive "+
						"here as the same failure — which is the state this half was "+
						"added to end.", c.What, r.What, by, cited.Phrase)
					continue
				}
				// The two halves have to be about the same number. A field that
				// is not IN the phrase would let a missing phrase be reported as
				// a reword on the strength of a token belonging to some other
				// reading — the wrong diagnosis, delivered confidently.
				if !strings.Contains(pinStripSpace(cited.Phrase), pinStripSpace(cited.Field)) {
					t.Errorf("%q: %s cites %s as %q and gives its field as %q, which does "+
						"not appear in it.\n\n"+
						"The phrase is the assertion that reads this number and the field "+
						"is the fixture's own spelling inside it. Two strings that are not "+
						"about the same number make the second question meaningless: a "+
						"phrase that had gone would be diagnosed by a token this row never "+
						"had a claim on.", c.What, r.What, by, cited.Phrase, cited.Field)
					continue
				}
				if pinSpells(source[by], cited.Phrase) {
					continue
				}
				// The phrase is gone. Which edit that was is the second
				// question, and the field answers it: an assertion deleted
				// outright takes the fixture's field with it, a reworded one
				// almost never moves it. See pinCite.
				if pinSpells(source[by], cited.Field) {
					t.Errorf("%q: caseNumbers says %s is read by %s (%s), and %s does not "+
						"contain %q — but it does still spell %q.\n\n"+
						"That is a REWORDED assertion rather than a deleted one: the "+
						"harness still names this number, so nothing here is stale and "+
						"resolution's floor is still derived over a distinction somebody "+
						"is still asked to make. Requote the phrase and change nothing "+
						"else. (The field is the weaker of the two claims and is only ever "+
						"asked after the phrase, because another fixture in the same file "+
						"can spell the same token — see pinCite.)",
						c.What, r.What, by, r.Read, path, cited.Phrase, cited.Field)
					continue
				}
				// The phrase is gone from the CODE. Whether it is gone from the
				// file is the next question, and it is not the same one: the
				// search runs over what pinCodeOnly left, so a phrase surviving
				// only in the comment above a deleted assertion is the rot this
				// half was added to catch — and a phrase surviving in code
				// pinCodeOnly BLANKED is the blind spot the same lexer comes
				// with. The two are told apart by looking, not guessed at.
				inFile := pinSpells(wholeSource[by], cited.Phrase) ||
					pinSpells(wholeSource[by], cited.Field)
				note := ""
				switch {
				case inFile && len(openBy[by]) > 0:
					// Where the surviving string actually is, rather than a
					// list for the reader to check it against.
					//
					// The lexer's blind spot is a line-shaped fault — a regex
					// literal with an odd quote blanks the rest of ITS OWN line
					// — and the lines it can have happened on are already
					// known. So "check those first" was asking the reader to do
					// a search and a comparison that the two values in hand
					// settle: the phrase's offset in the raw file is one match
					// away and the line it falls on is one count away. Asked of
					// the phrase, and of the field when the phrase is the one
					// that is gone from the file too.
					spellings := pinPhraseLines(rawSource[by], cited.Phrase)
					what := "the phrase"
					if spellings == nil {
						spellings, what =
							pinPhraseLines(rawSource[by], cited.Field), "the field"
					}
					// Every line every surviving copy sits on. The question is
					// whether ANY of them is on a line the lexer lost its place
					// on, and a phrase is spelled twice whenever the sentence
					// above an assertion quotes it — so answering over the
					// first copy would rule the blind spot out by looking at
					// the comment while the assertion sat on a blanked line.
					at := []int{}
					for _, spelling := range spellings {
						for _, line := range spelling {
							if !slices.Contains(at, line) {
								at = append(at, line)
							}
						}
					}
					// And how many copies there are, said when it is more than
					// one: "on two lines" and "twice" send a reader to
					// different places, and the lines alone cannot tell them
					// apart.
					lead := what
					if len(spellings) > 1 {
						lead = fmt.Sprintf("%s, spelled %d times in the file,",
							what, len(spellings))
					}
					on := []int{}
					for _, line := range at {
						if slices.Contains(openBy[by], line) {
							on = append(on, line)
						}
					}
					verdict := ""
					switch {
					case len(on) > 0:
						verdict = fmt.Sprintf("%s is on %v, which %s among them — so "+
							"this failure is pinCodeOnly's and not the harness's: the "+
							"assertion is there and the lexer blanked it.",
							lead, on, map[bool]string{true: "is", false: "are"}[len(on) == 1])
					case len(at) > 0:
						verdict = fmt.Sprintf("%s is on %v, which %s among them — so "+
							"the blind spot is ruled out and what survives at that "+
							"line is prose.", lead, at,
							map[bool]string{true: "is not", false: "are not"}[len(at) == 1])
					default:
						verdict = "neither string could be put on a line of the file, " +
							"which means the surviving copy is broken across lines in a " +
							"way pinPhraseLines cannot place — check those lines by hand."
					}
					note = fmt.Sprintf("\n\nBoth strings are still SOMEWHERE in %s, and "+
						"the lexer ended inside an unterminated string on %d of its "+
						"lines (%v). A regular-expression literal carrying an odd number "+
						"of quotes blanks the rest of its own line, and an assertion "+
						"spelled there arrives here as a deletion — %s", path,
						len(openBy[by]), openBy[by], verdict)
				case inFile:
					note = fmt.Sprintf("\n\nOne of the two strings is still somewhere in "+
						"%s and pinCodeOnly did not leave it in the code, so what is "+
						"left of it is prose: the sentence above an assertion outlives "+
						"the assertion, and a whole-file search would have gone on "+
						"passing. That is exactly the rot the lexer was added to catch, "+
						"and this is it caught. (The lexer ended inside no unterminated "+
						"string in this file, so its own blind spot is ruled out.)", path)
				}
				t.Errorf("%q: caseNumbers says %s is read by %s (%s), and %s contains "+
					"neither %q nor %q.\n\n"+
					"That is a DELETED assertion: the harness has stopped naming this "+
					"number at all. This row still says it is read, and resolution goes "+
					"on deriving a floor over it — a bound BOTH tolerances obey, for a "+
					"distinction one of them has stopped being asked to make. Either the "+
					"reading comes out of this table, or the assertion goes back into "+
					"that file.%s",
					c.What, r.What, by, r.Read, path, cited.Phrase, cited.Field, note)
			}
		}
	}

	for name, path := range pinConsumers {
		if !credited[name] {
			t.Errorf("pinConsumers names %q (%s) and no reading in caseNumbers says it "+
				"reads anything.\n\n"+
				"A consumer in the closed set that nothing cites is either a harness that "+
				"has stopped reading this fixture — in which case the row is dead weight "+
				"and every claim about it here is vacuous — or a table that forgot to "+
				"credit it.", name, path)
		}
	}
}

// The two things pinStripSpace is allowed to assume, held to.
//
// The normaliser is a lexer's job done by a rule of thumb: it keeps a space
// only between two word characters, which is where a space carries meaning in a
// free-form syntax and is not where it carries meaning in every syntax. The
// last session's note put the honest bound on that plainly — it canonicalises
// two specific files, and pinConsumers is the closed set that makes it true —
// and nothing tied the rule to the set. This is the tie, in two halves.
//
// # The language
//
// Every consumer's extension has to be one pinFreeForm names. That is where the
// assumption lives: a harness written in a language where an indent is a
// statement boundary would go through the same normaliser and the same
// pinSpells, and a phrase would match a source it is not really in.
//
// # The literals
//
// And within those languages there is one place a run of spaces still means
// something the normaliser flattens: inside a string, character or regular
// expression literal. Both sides are flattened the same way, so this never
// turns a present phrase into a missing one — it makes the citation WEAKER than
// it reads, because two sources that differ only inside a literal canonicalise
// to one string and either would satisfy the row.
//
// No citation quotes a literal today. Held here so that the first one that does
// is a decision rather than a quietly softer claim.
func TestEveryConsumerIsALanguageTheNormaliserFits(t *testing.T) {
	for name, path := range pinConsumers {
		ext := filepath.Ext(path)
		why, ok := pinFreeForm[ext]
		if !ok {
			t.Errorf("pinConsumers names %q (%s), whose extension %q is not one "+
				"pinFreeForm covers.\n\n"+
				"pinStripSpace deletes every space that is not between two word "+
				"characters, from the harness source and from the phrase looked for in "+
				"it. That is sound where whitespace between tokens does nothing else, "+
				"and it is not sound where an indent is a statement boundary or a "+
				"newline changes what the line above it meant. Under this normaliser a "+
				"phrase from such a file could match a source it is not in, which is a "+
				"row reporting a reading nobody makes. Either the language belongs in "+
				"pinFreeForm with a sentence saying why the rule holds for it, or this "+
				"harness needs a normaliser of its own.", name, path, ext)
			continue
		}
		for _, half := range []struct{ what, sentence, needs string }{
			{"Whitespace", why.Whitespace, "pinStripSpace deletes every space that is " +
				"not between two word characters, and rests on this half"},
			{"Lexis", why.Lexis, "pinCodeOnly blanks this language's comments and " +
				"literals so that a citation is looked for in the harness's code, and " +
				"rests on this half"},
		} {
			if half.sentence != "" {
				continue
			}
			t.Errorf("pinFreeForm covers %q and its %s is empty.\n\n"+
				"The sentence is the whole of what the row is for: the extension is a "+
				"proxy for a property of a grammar, and a row with nothing under it is "+
				"the proxy standing on its own. %s.",
				ext, half.what, half.needs)
		}
	}
}

// And the other half: no cited phrase carries a literal. See the test above for
// why this is the one place the normaliser's flattening softens a claim.
func TestNoCitationQuotesALiteral(t *testing.T) {
	// The three literal openers the two languages in pinFreeForm share. A
	// backtick is JavaScript's template literal, where a run of spaces is data
	// as surely as it is inside a quoted string.
	const openers = "\"'`"
	for _, c := range Cases() {
		for _, r := range caseNumbers(c) {
			for by, cited := range r.By {
				for _, part := range []struct{ what, text string }{
					{"phrase", cited.Phrase}, {"field", cited.Field},
				} {
					if !strings.ContainsAny(part.text, openers) {
						continue
					}
					t.Errorf("%q: caseNumbers cites %q as the %s %s spells %s, and it "+
						"contains a string, character or template literal.\n\n"+
						"pinStripSpace flattens a run of spaces to one and drops a space "+
						"that is not between two word characters — inside a literal that "+
						"is data, not layout. Both the source and this phrase go through "+
						"it, so the phrase will not go missing; what happens instead is "+
						"that two sources differing only inside the literal canonicalise "+
						"to the same string and either satisfies this row. The claim is "+
						"quietly weaker than it reads. Quote the comparison around the "+
						"literal instead, or give the normaliser a rule for literals and "+
						"say what it is.",
						c.What, part.text, part.what, by, r.What)
				}
			}
		}
	}
}

// pinCodeOnly blanks out every character of a source that is inside a comment
// or a string literal, leaving the code.
//
// # The half of the language the extension was standing in for
//
// pinFreeForm records, per extension, that whitespace between tokens is
// insignificant in that language — the property pinStripSpace needs. That is
// one property of a grammar used as a proxy for the grammar, and the note above
// TestEveryConsumerIsALanguageTheNormaliserFits already says which spans it is
// wrong about: the ones inside a literal, where a run of spaces is data.
//
// That was held at the citation — no row here quotes a literal — and not at the
// source, which leaves the more interesting direction open. A phrase with no
// quote in it can still MATCH inside a literal or, far more likely in these two
// files, inside a comment: both harnesses are written in prose as much as in
// code, and every one of these assertions is discussed somewhere above itself.
// A row whose phrase survives only in the sentence explaining the assertion is
// a row that would go on passing after the assertion was deleted — which is the
// exact rot this table exists to catch, arriving through the door the citation
// check was watching from the other side.
//
// So the source is lexed, and a citation has to be spelled in the code.
//
// # What this lexes, and what it deliberately does not
//
// The two languages in pinFreeForm share the C family's comment syntax and
// double-quoted strings with backslash escapes, which is the bulk of it. The
// differences that matter are named per language below. What is NOT handled is
// JavaScript's regular-expression literal: telling `/` the division operator
// from `/` the start of a regex needs the parser's context, and guessing wrong
// in the swallowing direction would blank out real code and turn a live
// citation into a false failure. A regex containing a quote character could
// therefore open a spurious string — which is why the two invariants below are
// asserted rather than assumed.
//
// Characters are replaced with spaces rather than removed, so pinStripSpace
// still sees a token boundary where a literal used to be: `f("x")` becomes
// `f(   )` and then `f()`, and `a"x"b` stays two identifiers rather than
// becoming one.
//
// # And the blind spot reports itself
//
// The floor below discriminates a lexer that lost its place — 4.0% of a file
// surviving — from two files that are mostly prose, at 16.8% and 17.3%. That
// is a real bound with a measurement on each side, and it is a bound on
// CATASTROPHE. What it cannot see is the small version: a regex whose quote
// opens a string that swallows the rest of ONE line, and that line being the
// one an assertion happens to sit on. Two hundred bytes out of a hundred
// thousand moves no percentage, and what it produces is a single false
// "DELETED assertion" in a table of forty-odd citations — a message that sends
// a reader to look at a harness which is perfectly fine.
//
// So the second return value is every line the lexer ended INSIDE a
// single-line string: the quote opened and the newline closed it, rather than
// its partner doing so. That is not a heuristic about regexes, it is the
// complete set of places the damage can start. In real source an unterminated
// single-line string is a syntax error, so every one of these is either the
// regex blind spot or a file that does not compile — and either way it is the
// one thing a reader needs to know before believing a deletion report about
// that file.
//
// A regex carrying an EVEN number of quotes pairs them among themselves and
// blanks its own body, which is inside the literal and not code an assertion
// can be spelled in. The odd case is the one that runs on, and the odd case is
// exactly what shows up here.
func pinCodeOnly(src, ext string) (code string, openQuoteLines []int) {
	// Swift nests block comments; JavaScript does not, and treating /* */ as
	// nesting there would swallow everything after a `/*` inside a comment.
	nested := ext == ".swift"
	// A backtick opens a template literal in JavaScript, where a run of spaces
	// is data; in Swift it quotes an identifier that would otherwise be a
	// keyword, which is code.
	tick := ext != ".swift"
	// Swift has no single-quoted string: `'` there is an ordinary punctuation
	// character (and appears in prose, in apostrophes, constantly).
	single := ext != ".swift"

	out := []byte(src)
	blank := func(i int) {
		if out[i] != '\n' {
			out[i] = ' '
		}
	}
	// Where a single-line string ran into a newline instead of into its closing
	// quote. Offsets while the scan runs, turned into line numbers in one pass
	// afterwards, so the scan stays linear and the newline bookkeeping does not
	// have to be threaded through every arm of it.
	var openAt []int
	for i := 0; i < len(src); {
		switch {
		case src[i] == '/' && i+1 < len(src) && src[i+1] == '/':
			for ; i < len(src) && src[i] != '\n'; i++ {
				blank(i)
			}
		case src[i] == '/' && i+1 < len(src) && src[i+1] == '*':
			depth := 1
			blank(i)
			blank(i + 1)
			i += 2
			for i < len(src) && depth > 0 {
				if nested && src[i] == '/' && i+1 < len(src) && src[i+1] == '*' {
					depth++
					blank(i)
					blank(i + 1)
					i += 2
					continue
				}
				if src[i] == '*' && i+1 < len(src) && src[i+1] == '/' {
					depth--
					blank(i)
					blank(i + 1)
					i += 2
					continue
				}
				blank(i)
				i++
			}
		case src[i] == '"' || (single && src[i] == '\'') || (tick && src[i] == '`'):
			q := src[i]
			// Swift's multi-line string, whose delimiter is three of them.
			// Handled first because `"""` also parses as an empty string
			// followed by a quote, which would leave the body as code.
			long := q == '"' && strings.HasPrefix(src[i:], `"""`)
			n := 1
			if long {
				n = 3
			}
			for k := 0; k < n; k++ {
				blank(i + k)
			}
			i += n
			for i < len(src) {
				if src[i] == '\\' && i+1 < len(src) {
					blank(i)
					blank(i + 1)
					i += 2
					continue
				}
				if long && strings.HasPrefix(src[i:], `"""`) {
					blank(i)
					blank(i + 1)
					blank(i + 2)
					i += 3
					break
				}
				if !long && src[i] == q {
					blank(i)
					i++
					break
				}
				// A single-quoted or double-quoted literal ends at the line in
				// both languages, and stopping here is what keeps an
				// apostrophe in a comment... which is already blanked... from
				// running to the end of the file. It also bounds the damage a
				// regex literal's quote can do to one line.
				if !long && q != '`' && src[i] == '\n' {
					// And it is recorded. A quote that opened and did not close
					// before the line ended is a syntax error in both these
					// languages, so in a file that compiles it is the regex
					// literal's own quote — the one construct this lexer
					// declines to guess at. See the note above: it is the
					// complete set of places the damage can start, and every
					// line here is a line whose code from the quote onward was
					// blanked.
					openAt = append(openAt, i)
					break
				}
				blank(i)
				i++
			}
		default:
			i++
		}
	}

	// The offsets, as line numbers. One walk over the source, because the
	// offsets came out of a left-to-right scan and are therefore sorted.
	at, line := 0, 1
	for i := 0; i < len(src) && at < len(openAt); i++ {
		for at < len(openAt) && openAt[at] == i {
			openQuoteLines = append(openQuoteLines, line)
			at++
		}
		if src[i] == '\n' {
			line++
		}
	}
	return string(out), openQuoteLines
}

// pinCodeOnly does what pinFreeForm's Lexis sentences say, per language.
//
// The lexer is the thing standing between a citation and a comment that quotes
// it, and the two consumers cannot demonstrate it: they are correct today, so
// running it over them shows only that nothing was broken. These are the spans
// it has to get right, one construct at a time, with the two languages'
// disagreements as separate rows — a nesting block comment, an apostrophe, a
// backtick — because those are the places a single C-family lexer would be
// wrong about one of them.
//
// `want` is what has to SURVIVE and `gone` what must not. Both directions,
// because a lexer that blanked everything would satisfy the second on its own.
func TestPinCodeOnlyBlanksWhatEachLanguageCallsProse(t *testing.T) {
	for _, c := range []struct {
		what, ext, src string
		want, gone     []string
		// openLines is what the lexer has to say about its own uncertainty:
		// the lines it ended inside a single-line string. See pinCodeOnly.
		openLines []int
	}{
		{"a line comment", ".mjs", "keep(1) // gone(2)\nkeep(3)",
			[]string{"keep(1)", "keep(3)"}, []string{"gone(2)"}, nil},
		{"a block comment", ".mjs", "keep(1) /* gone(2)\ngone(3) */ keep(4)",
			[]string{"keep(1)", "keep(4)"}, []string{"gone(2)", "gone(3)"}, nil},
		{"a double-quoted string", ".mjs", `keep(1, "gone(2)")`,
			[]string{"keep(1,"}, []string{"gone(2)"}, nil},
		{"a single-quoted string", ".mjs", "keep(1, 'gone(2)')",
			[]string{"keep(1,"}, []string{"gone(2)"}, nil},
		{"a template literal", ".mjs", "keep(1, `gone(2)\ngone(3)`)",
			[]string{"keep(1,"}, []string{"gone(2)", "gone(3)"}, nil},
		{"an escaped quote inside a string", ".mjs", `f("a\"gone(1)") keep(2)`,
			[]string{"keep(2)"}, []string{"gone(1)"}, nil},
		// JavaScript's block comment does not nest, so the FIRST */ ends it and
		// what follows is code again. A lexer that nested here would swallow
		// the rest of the file.
		{"a block comment that does not nest", ".mjs",
			"/* gone(1) /* gone(2) */ keep(3)",
			[]string{"keep(3)"}, []string{"gone(1)", "gone(2)"}, nil},
		// And Swift's does, so the same text is comment all the way to the
		// second closer.
		{"a block comment that nests", ".swift",
			"/* gone(1) /* gone(2) */ gone(3) */ keep(4)",
			[]string{"keep(4)"}, []string{"gone(1)", "gone(2)", "gone(3)"}, nil},
		// An apostrophe in Swift is prose, not a string opener. Treated as one,
		// everything to the end of the line would be blanked.
		{"an apostrophe outside a comment", ".swift", "keep(1) // it's here\nkeep(2)",
			[]string{"keep(1)", "keep(2)"}, []string{"here"}, nil},
		{"a multi-line string", ".swift",
			"keep(1)\n\"\"\"\ngone(2)\n\"\"\"\nkeep(3)",
			[]string{"keep(1)", "keep(3)"}, []string{"gone(2)"}, nil},
		// A backtick quotes an identifier in Swift. Read as a template opener
		// it would blank from there to the next one, or to the end of the file.
		{"a backtick-quoted identifier", ".swift", "keep(`class`) keep(2)",
			[]string{"keep(", "class", "keep(2)"}, nil, nil},
		// The construct this lexer declines to guess at, and the only damage it
		// can do. A regex with one quote in it opens a string that runs to the
		// newline, so the assertion after it on that line is blanked and its
		// citation reports as deleted. The line is named, which is the whole of
		// what this half adds: the floor cannot see two hundred bytes, and a
		// reader told "line 2" can look.
		{"a regex literal's odd quote, reported rather than lexed", ".mjs",
			"keep(1)\nif (/[\"]/.test(x)) gone(2)\nkeep(3)",
			[]string{"keep(1)", "keep(3)"}, []string{"gone(2)"}, []int{2}},
		// And the even case, which pairs among itself and damages nothing past
		// the literal: the first quote opens a string, the second closes it,
		// and what is blanked is the regex's own body. Reported as clean,
		// because reporting it would send a reader to a line where nothing
		// happened.
		//
		// Two quotes of the SAME kind. `/["']/` is not this case — in
		// JavaScript an apostrophe opens a string of its own, so a regex
		// carrying one of each has an odd number of each and runs on.
		{"a regex literal's paired quotes leave the line alone", ".mjs",
			"keep(1)\nif (/[\"gone(2)\"]/.test(x)) keep(3)\nkeep(4)",
			[]string{"keep(1)", "keep(3)", "keep(4)"}, []string{"gone(2)"}, nil},
		// A real unterminated string, which is a syntax error rather than the
		// blind spot — and is reported identically, because from here they are
		// the same event and the message says so.
		{"a string with no closing quote", ".mjs", "keep(1)\nf(\"gone(2)\nkeep(3)",
			[]string{"keep(1)", "keep(3)"}, []string{"gone(2)"}, []int{2}},
		// Swift's multi-line string spans lines on purpose and must not be
		// reported: it is closed by its own delimiter, not by a newline.
		{"a multi-line string is not an open quote", ".swift",
			"keep(1)\n\"\"\"\ngone(2)\n\"\"\"\nkeep(3)",
			[]string{"keep(1)", "keep(3)"}, []string{"gone(2)"}, nil},
	} {
		got, openLines := pinCodeOnly(c.src, c.ext)
		if !slices.Equal(openLines, c.openLines) {
			t.Errorf("%s (%s): pinCodeOnly reports unterminated strings on lines %v, "+
				"want %v.\n\n"+
				"That list is the whole of what this lexer knows about its own blind "+
				"spot. The floor bounds a lexer that lost its place; it cannot see one "+
				"line, and one line is what a regular-expression literal's stray quote "+
				"costs. A missing entry is a deletion report nothing warns about; a "+
				"spurious one sends a reader to a line where nothing "+
				"happened.\n\nsource: %q\ncode:   %q", c.what, c.ext, openLines,
				c.openLines, c.src, got)
		}
		for _, w := range c.want {
			if !strings.Contains(got, w) {
				t.Errorf("%s (%s): pinCodeOnly blanked %q, which is code.\n\n"+
					"pinFreeForm[%q].Lexis is what this lexer is held to, and code that "+
					"does not survive it is a citation that cannot be found — every row "+
					"of caseNumbers naming a harness in this language would report a "+
					"deleted assertion.\n\nsource: %q\ncode:   %q",
					c.what, c.ext, w, c.ext, c.src, got)
			}
		}
		for _, g := range c.gone {
			if strings.Contains(got, g) {
				t.Errorf("%s (%s): pinCodeOnly kept %q, which is prose.\n\n"+
					"That is the direction this lexer exists for: a citation looked for "+
					"in a file that still carries its comments can be satisfied by the "+
					"sentence ABOVE an assertion that has been deleted, which is the rot "+
					"caseNumbers' citations are there to catch.\n\nsource: %q\ncode:   %q",
					c.what, c.ext, g, c.src, got)
			}
		}
	}
}

// The blind-spot note's verdict rests on putting a phrase back on a line of the
// file it was read out of, and the whole difficulty is that the search runs
// over a string with the whitespace canonicalised out of it. This holds the
// index to the four shapes that can go wrong.
//
// The wrapped case is the interesting one: a phrase whose bytes came from two
// lines is reported on both, because "the assertion is on line 6167, which is
// one of them" is a sentence about a run of lines and not about a point, and a
// wrap that put half an assertion on a blanked line is exactly the case the
// note exists for.
//
// The twice-spelled case is the other one, and it is why the answer is grouped
// per copy rather than flattened here: a phrase quoted in the sentence above
// its own assertion survives in two places, and "on two lines" and "twice" are
// different things to tell a reader. Taking the first match, which is what
// this used to do, answers about whichever copy came earlier in the file — the
// comment, when the comment is above the code.
func TestAPhraseIsPutBackOnTheLinesItCameFrom(t *testing.T) {
	const src = "one\n  if (!pinSame(total, c.offer)) {\n three\n a = pinSame(x,\n   y)\n" +
		"// pinSame(total, c.offer) again, in prose\n"
	for _, c := range []struct {
		what   string
		phrase string
		want   [][]int
	}{
		// Two copies: the code on line 2 and the comment on line 6. Both are
		// reported, and it is the caller that decides which of them being on a
		// blanked line settles the verdict.
		{"a phrase spelled twice", "pinSame(total, c.offer)", [][]int{{2}, {6}}},
		{"the same phrase spelled without its spaces", "pinSame(total,c.offer)",
			[][]int{{2}, {6}}},
		{"a phrase broken across a line", "pinSame(x, y)", [][]int{{4, 5}}},
		{"a phrase that is not there at all", "pinSame(nothing)", nil},
	} {
		if got := pinPhraseLines(src, c.phrase); !slices.EqualFunc(got, c.want,
			slices.Equal[[]int]) {
			t.Errorf("%s: pinPhraseLines(%q) is %v, want %v.\n\n"+
				"The deletion report's blind-spot arm turns \"the lexer lost its place "+
				"on these lines\" into \"the surviving assertion is on this one, which "+
				"is among them\", and that verdict is this index. A phrase placed on "+
				"the wrong line sends the reader to look at code that is fine and "+
				"clears a lexer that is not, and a copy left out of the answer is a "+
				"blanked assertion the note rules out by reading its own comment.",
				c.what, c.phrase, got, c.want)
		}
	}
	// And the property the index rests on, which no row above can show: the
	// string the lines are indexed against is byte-for-byte the one pinSpells
	// searches. Two walks that disagreed by a byte would put every phrase after
	// the disagreement on the wrong line, and every row above would still pass
	// because they are all near the top.
	stripped, lines := pinStripSpaceLines(src)
	if stripped != pinStripSpace(src) {
		t.Errorf("pinStripSpaceLines returns %q and pinStripSpace returns %q.\n\n"+
			"The index is one entry per byte of the first and the search runs over "+
			"the second, so the two being the same string is what makes an offset in "+
			"one a line in the other.", stripped, pinStripSpace(src))
	}
	if len(lines) != len(stripped) {
		t.Errorf("pinStripSpaceLines returns %d bytes and %d line numbers.\n\n"+
			"They are one array indexed two ways: a phrase's match is a byte range in "+
			"the string and the lines it fell on are that same range in the index.",
			len(stripped), len(lines))
	}
}

// pinDense counts the bytes of a source that are not whitespace, which is the
// measure pinCodeOnly's floor is taken in: blanking a literal replaces it with
// spaces, so a count of every byte would not move at all.
func pinDense(s string) int {
	n := 0
	for i := 0; i < len(s); i++ {
		if !unicode.IsSpace(rune(s[i])) {
			n++
		}
	}
	return n
}

// pinStripSpace canonicalises whitespace out of a source or a phrase, keeping
// only the whitespace that separates one identifier from another.
//
// Both the harness source and the phrase looked for in it go through this, so
// the comparison is about what the assertion says rather than how it is laid
// out. A reformat that wraps `pinSame(measured, c.gap)` over four lines leaves
// the same string on both sides; a rewrite that changes what is compared does
// not, and that is the edit somebody has to look at.
//
// # Why a space between two word characters survives
//
// It used to remove every space, tab and newline, and that is one character too
// many: it glues neighbouring tokens together, so `where abs(css[i]` becomes
// `...countwhereabs(css[i]`, and pinSpells' leading word boundary — which is
// there to stop `pinSame(…)` matching inside `myPinSame(…)` — then fails
// against a phrase that IS present. Every phrase this table happened to carry
// began just after a bracket or a `!`, so the boundary held by luck and the
// first phrase quoted from a `where abs(…)` broke it.
//
// A space is deleted unless a word character sits on both sides of it, which is
// exactly the case where it is carrying meaning: `pinSame(total, c.offer)` and
// `pinSame(total,c.offer)` canonicalise to one string, and `where abs` and
// `whereabs` stay two.
func pinStripSpace(s string) string {
	stripped, _ := pinStripSpaceLines(s)
	return stripped
}

// pinStripSpaceLines is pinStripSpace with the raw line every surviving byte
// came from, one entry per byte of the returned string.
//
// The index is what lets a phrase found in the stripped text be put back on a
// line of the file it was read out of. See pinPhraseLines and the deletion
// report's blind-spot note: naming the lines a lexer lost its place on is a
// list to check, and saying whether the surviving string is ON one of them is
// the verdict that list was standing in for.
//
// Written as a walk rather than as Fields+Join because the join is where the
// positions were being thrown away. Line numbers are 1-based and count "\n",
// which is what a reader's editor counts; the field split is unicode.IsSpace's,
// which is strings.Fields', so the two passes below produce exactly the string
// the previous version did.
func pinStripSpaceLines(s string) (string, []int) {
	// Pass one: the fields, joined by a single space.
	var joined []byte
	var jline []int
	line, inField := 1, false
	for i, r := range s {
		if unicode.IsSpace(r) {
			if r == '\n' {
				line++
			}
			inField = false
			continue
		}
		if !inField && len(joined) > 0 {
			// The separator belongs to the field it precedes: a phrase that
			// wraps is reported at the line its first byte is on and at the
			// line its last byte is on, and a separator credited backwards
			// would put the wrap on the earlier line twice.
			joined = append(joined, ' ')
			jline = append(jline, line)
		}
		inField = true
		n := len(string(r))
		joined = append(joined, s[i:i+n]...)
		for k := 0; k < n; k++ {
			jline = append(jline, line)
		}
	}
	// Pass two: a space survives only between two word characters, which is
	// pinStripSpace's rule and the reason it has one. See the note above.
	var out []byte
	var oline []int
	for i := 0; i < len(joined); i++ {
		if joined[i] == ' ' {
			if i > 0 && i+1 < len(joined) && word(joined[i-1]) && word(joined[i+1]) {
				out = append(out, ' ')
				oline = append(oline, jline[i])
			}
			continue
		}
		out = append(out, joined[i])
		oline = append(oline, jline[i])
	}
	return string(out), oline
}

// pinPhraseLines is the lines of a raw file each spelling of the phrase
// occupies — one entry per surviving copy — or nil when the phrase is not in
// the file at all.
//
// Asked of the RAW file rather than of what pinCodeOnly left, because the one
// caller is the arm that has already established the phrase is gone from the
// code and still somewhere in the file. What it wants to know is where.
//
// # Every copy, not the first
//
// It used to take FindStringIndex and answer about one match. A phrase is
// spelled twice whenever the sentence above an assertion quotes it, which is
// the ordinary way a comment is written here — and the blind-spot verdict is
// "is a surviving copy on a line the lexer lost its place on". Answering that
// over whichever copy came first in the file is right when one of them is the
// survivor and wrong exactly when both survive and the LATER one is the
// blanked assertion: the note would rule its own blind spot out by looking at
// the comment.
//
// So all of them, grouped: a copy broken across a line is one entry naming two
// lines, and two copies on one line each are two entries. The caller flattens
// for the verdict and counts the entries for the sentence, because "the phrase
// is on two lines" and "there are two phrases" send a reader to different
// places.
func pinPhraseLines(raw, phrase string) [][]int {
	stripped, lines := pinStripSpaceLines(raw)
	var out [][]int
	// Non-overlapping, left to right, which is what FindAll gives: two matches
	// that overlapped would be one spelling counted twice.
	for _, loc := range pinSpellsPattern(phrase).FindAllStringIndex(stripped, -1) {
		var at []int
		for i := loc[0]; i < loc[1] && i < len(lines); i++ {
			if len(at) == 0 || at[len(at)-1] != lines[i] {
				at = append(at, lines[i])
			}
		}
		out = append(out, at)
	}
	return out
}

// pinSpells reports whether stripped source contains the phrase as a whole
// token rather than as the head of a longer one.
//
// The boundary is asked only at an end that is a word character, because most
// of these phrases end in a bracket: `pinSame(total,c.offer)` has nothing a
// word boundary could match after it, and `c.compose.offered` renamed to
// `c.compose.offeredX` would otherwise go on satisfying the row that says the
// original is read.
func pinSpells(stripped, phrase string) bool {
	return pinSpellsPattern(phrase).MatchString(stripped)
}

// pinSpellsPattern is the expression pinSpells matches with.
//
// Split out so pinPhraseLines can ask WHERE with the same expression that
// decides WHETHER. Two spellings of one pattern would be two answers about one
// phrase, and the second caller's job is to locate the very match the first
// one found.
func pinSpellsPattern(phrase string) *regexp.Regexp {
	want := regexp.QuoteMeta(pinStripSpace(phrase))
	if word(pinStripSpace(phrase)[0]) {
		want = `\b` + want
	}
	if last := pinStripSpace(phrase); word(last[len(last)-1]) {
		want += `\b`
	}
	return regexp.MustCompile(want)
}

// word reports whether a byte is one Go's regexp counts inside \b.
func word(b byte) bool {
	return b == '_' || ('0' <= b && b <= '9') ||
		('a' <= b && b <= 'z') || ('A' <= b && b <= 'Z')
}
