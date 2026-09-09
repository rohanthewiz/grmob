package pinfixture

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
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
	// And what pinCodeOnly left, byte for byte.
	//
	// The lexer replaces prose with spaces IN PLACE, so its output is the
	// file's own bytes at the file's own offsets — which makes "what did this
	// pass do to the bytes this copy occupies" a slice. The deletion report is
	// the only reader: when a phrase survives twice in the file, the count
	// alone sends a reader to open both, and the pass that blanked one of them
	// already knows which.
	//
	// Kept whole rather than split into lines, because a line is the wrong
	// unit for the question. The lexer blanks SPANS: a phrase inside a string
	// literal on a line that also carries code leaves that line looking live
	// while the copy itself is gone, and a note reading lines calls that copy
	// an assertion and sends a reader to open it. See pinCopyNote.
	codeSource := map[string]string{}
	// And what pinCodeOnly was lexing when it blanked each of those bytes. Same
	// offsets, same walk, one byte per source byte — read only by the deletion
	// report, which is the one place that has to tell a phrase inside a string
	// literal from one inside a comment trailing an assertion. See
	// pinBlankKind.
	blankedBy := map[string][]byte{}
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
		code, openLines, blanked := pinCodeOnly(string(b), filepath.Ext(path))
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
		codeSource[name] = code
		blankedBy[name] = blanked
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
					spellings := pinPhraseCopies(rawSource[by], cited.Phrase)
					what := "the phrase"
					if spellings == nil {
						spellings, what =
							pinPhraseCopies(rawSource[by], cited.Field), "the field"
					}
					// Every line every surviving copy sits on. The question is
					// whether ANY of them is on a line the lexer lost its place
					// on, and a phrase is spelled twice whenever the sentence
					// above an assertion quotes it — so answering over the
					// first copy would rule the blind spot out by looking at
					// the comment while the assertion sat on a blanked line.
					at := []int{}
					for _, spelling := range spellings {
						for _, line := range spelling.At {
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
					// And which of those copies is the prose. See
					// pinCopyNote: the count says there are two and the
					// argument turns on which.
					which := pinCopyNote(codeSource[by], blankedBy[by], spellings)
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
							"assertion is there and the lexer blanked it.%s",
							lead, on,
							map[bool]string{true: "is", false: "are"}[len(on) == 1],
							which)
					case len(at) > 0:
						verdict = fmt.Sprintf("%s is on %v, which %s among them — so "+
							"the blind spot is ruled out and what survives at that "+
							"line is prose.%s", lead, at,
							map[bool]string{true: "is not", false: "are not"}[len(at) == 1],
							which)
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

// What pinCodeOnly was lexing when it blanked a byte.
//
// # A count-shaped answer, one level down
//
// pinCopyNote tells a reader which surviving copy of a phrase is the assertion
// and which is the sentence above it. Its middle arm — a copy blanked on a line
// that kept code — used to name all three things that produce that shape and
// let the reader sort it out: "a literal, a comment trailing an assertion, or
// the runaway that blanks the rest of its own line". Three constructs, three
// different things to find at that line, and one sentence covering them is the
// count-shaped answer that whole note was rebuilt to stop giving.
//
// The pass that blanked the bytes knows which it was. pinCodeOnly is a lexer
// with a case per construct and it returned none of that, so the note was
// reconstructing from a line what the lexer had decided from a quote.
//
// Zero is code the lexer kept, which is also what an untouched newline reads
// as: neither is a construct, and the only reader here asks about spans it has
// already established were blanked.
const (
	pinKeptCode      byte = 0
	pinLineComment   byte = 'l'
	pinBlockComment  byte = 'b'
	pinStringLiteral byte = 's'
	// A single-line string the lexer never saw closed — see pinCodeOnly's own
	// note. Recorded apart from an ordinary literal because it is the one that
	// makes the deletion report lie: the code BEFORE the quote still stands on
	// that line, so the line looks live while the assertion after it is gone.
	pinRunawayString byte = 'r'
)

// pinConstructName is one of those, spelled the way a sentence needs it.
func pinConstructName(kind byte) string {
	switch kind {
	// Reachable from a copy that STRADDLES: a phrase whose first half a
	// construct swallowed and whose second half the lexer kept is part code,
	// and the sentence that describes it has to be able to say so. Not
	// reachable from a copy that is wholly one thing — that copy is live and is
	// the line to open. See pinBlankedAs.
	case pinKeptCode:
		return "code the lexer kept"
	case pinLineComment:
		return "a comment running to the end of the line"
	case pinBlockComment:
		return "a block comment"
	case pinStringLiteral:
		return "a string literal"
	case pinRunawayString:
		return "a string the lexer never saw closed, which blanks the rest of its " +
			"own line"
	}
	// Not reachable from a blanked span: a phrase's span always contains a byte
	// that was not a space in the raw file, and a blanked copy is one where
	// those bytes became spaces. Named rather than omitted, because a sentence
	// that quietly drops its noun is worse than one that says the noun is
	// missing.
	return "a construct pinCodeOnly did not record"
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
//
// # And what it was doing when it blanked each byte
//
// The third return value is one byte per source byte saying which construct
// took it — see pinBlankKind. The switch below already decides that, a case at
// a time, and returning none of it left pinCopyNote reconstructing from a LINE
// what this had decided from a quote: its middle arm named all three
// constructs at once and let the reader work out which. A literal, a comment
// trailing an assertion and this blind spot's runaway produce the same shape
// from outside and send a reader to three different places.
//
// Recorded at the source's own offsets, like the blanking itself, so a copy's
// span is a slice of it and no second walk can disagree with the one that did
// the work. The runaway is written twice: once as the literal it looked like,
// and again from its opening quote when the newline closes it, because that is
// the moment the lexer learns which of the two it was.
func pinCodeOnly(src, ext string) (code string, openQuoteLines []int, blankedBy []byte) {
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
	// And which construct was being lexed when each byte went. See
	// pinBlankKind: the switch below already knows — it has a case per
	// construct — and until now it returned none of it, so the one reader that
	// has to tell a literal from a trailing comment was left listing all three.
	//
	// One byte per source byte, at the source's own offsets, for the same
	// reason the blanking is done in place: a copy's span is then a slice of
	// this, and no second walk can disagree with the one that did the work.
	blankedBy = make([]byte, len(src))
	blank := func(i int, kind byte) {
		if out[i] != '\n' {
			out[i] = ' '
			blankedBy[i] = kind
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
				blank(i, pinLineComment)
			}
		case src[i] == '/' && i+1 < len(src) && src[i+1] == '*':
			depth := 1
			blank(i, pinBlockComment)
			blank(i+1, pinBlockComment)
			i += 2
			for i < len(src) && depth > 0 {
				if nested && src[i] == '/' && i+1 < len(src) && src[i+1] == '*' {
					depth++
					blank(i, pinBlockComment)
					blank(i+1, pinBlockComment)
					i += 2
					continue
				}
				if src[i] == '*' && i+1 < len(src) && src[i+1] == '/' {
					depth--
					blank(i, pinBlockComment)
					blank(i+1, pinBlockComment)
					i += 2
					continue
				}
				blank(i, pinBlockComment)
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
			// Where this literal opened, so that a run to the newline can go
			// back and say what these bytes actually were. The lexer does not
			// know it was looking at a runaway until it hits one, and a reader
			// sent to a line where the code before a quote is all that stands
			// is being told about a different construct from a reader sent to
			// a literal the author closed.
			from := i
			for k := 0; k < n; k++ {
				blank(i+k, pinStringLiteral)
			}
			i += n
			for i < len(src) {
				if src[i] == '\\' && i+1 < len(src) {
					blank(i, pinStringLiteral)
					blank(i+1, pinStringLiteral)
					i += 2
					continue
				}
				if long && strings.HasPrefix(src[i:], `"""`) {
					blank(i, pinStringLiteral)
					blank(i+1, pinStringLiteral)
					blank(i+2, pinStringLiteral)
					i += 3
					break
				}
				if !long && src[i] == q {
					blank(i, pinStringLiteral)
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
					// Retold, now that the lexer knows what it was in. Every
					// byte from the opening quote to here was recorded as an
					// ordinary literal; what it actually is is the one
					// construct this lexer declines to guess at, running past
					// the code it swallowed.
					for k := from; k < i; k++ {
						if blankedBy[k] == pinStringLiteral {
							blankedBy[k] = pinRunawayString
						}
					}
					break
				}
				blank(i, pinStringLiteral)
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
	return string(out), openQuoteLines, blankedBy
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
		got, openLines, _ := pinCodeOnly(c.src, c.ext)
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
		copies := pinPhraseCopies(src, c.phrase)
		at := [][]int{}
		for _, copy := range copies {
			at = append(at, copy.At)
		}
		if len(at) == 0 {
			at = nil
		}
		if !slices.EqualFunc(at, c.want, slices.Equal[[]int]) {
			t.Errorf("%s: pinPhraseCopies(%q) is on %v, want %v.\n\n"+
				"The deletion report's blind-spot arm turns \"the lexer lost its place "+
				"on these lines\" into \"the surviving assertion is on this one, which "+
				"is among them\", and that verdict is this index. A phrase placed on "+
				"the wrong line sends the reader to look at code that is fine and "+
				"clears a lexer that is not, and a copy left out of the answer is a "+
				"blanked assertion the note rules out by reading its own comment.",
				c.what, c.phrase, at, c.want)
		}
		// And the other half of the entry, which is what pinCopyNote reads:
		// the raw bytes this copy occupies. Held to the phrase itself under
		// the same stripping the search uses — a span off by a byte at either
		// end slices a literal's quote or drops a bracket, and every arm of
		// that note is a question about what the lexer did to exactly these
		// bytes.
		for _, copy := range copies {
			if copy.Start < 0 || copy.End > len(src) || copy.Start >= copy.End {
				t.Errorf("%s: pinPhraseCopies(%q) places a copy at [%d %d) in a "+
					"%d-byte file.\n\n"+
					"The span is an index into what pinCodeOnly left, which is this "+
					"file with prose spaced out. A range outside it is a copy "+
					"pinCopyNote drops, and it drops it because the two readings have "+
					"come apart rather than because a copy is missing.",
					c.what, c.phrase, copy.Start, copy.End, len(src))
				continue
			}
			if got := pinStripSpace(src[copy.Start:copy.End]); got !=
				pinStripSpace(c.phrase) {
				t.Errorf("%s: the copy on %v spans %q, want the phrase %q.\n\n"+
					"pinCopyNote asks what pinCodeOnly did to THESE bytes and answers "+
					"prose, literal or live code from it. A span that reached past the "+
					"copy would take in the code around a literal and report the copy "+
					"as standing; one that fell short would report an assertion as "+
					"blanked.", c.what, copy.At, got, pinStripSpace(c.phrase))
			}
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

// Which transitions between two constructs a straddle can actually be built
// out of, walked rather than argued about.
//
// # A general reader with evidence from one shape
//
// pinBlankedAs reads every construct a copy's span meets, in order, and the
// sentence it feeds names them all. That machinery is general. Its evidence was
// not: both fixtures that exercised it opened with `//`, and the note above it
// said why — a line comment and a runaway string end AT the newline, the
// newline is whitespace, and every other transition needs "a delimiter in the
// raw file and a delimiter breaks the match".
//
// The second half of that is wrong, and it is wrong in the direction that
// matters. A delimiter breaks the match when the phrase is spelled WITHOUT it.
// A citation that quotes a call carries its own quotes:
//
// \tlog(pinSame("x"))    code the lexer kept, then a string literal, and not a
// \t                     newline anywhere in it
//
// So the rule is about the bytes at the boundary and not about which construct
// is on either side of it: a straddle is buildable when those bytes are
// whitespace (which the strip removes) or are part of the citation. Under that
// rule every ordered pair below is reachable, including the block comment
// closing mid-line into a literal that the old note called hypothetical.
//
// # What each row asserts
//
// Both directions, because either alone is half a reading. The phrase has to
// produce exactly one copy and that copy's own bytes have to have been taken by
// the run of constructs the row names — the sentence is not asked here, that is
// the test above; this is about what the lexer's record says. And for every row
// whose boundary is raw bytes, the SAME source with those bytes taken out of
// the phrase has to produce no copy at all, which is what shows the delimiter
// is being carried rather than skipped over.
//
// A row that stopped straddling would be pinCodeOnly having changed what it
// records; a `bare` that started matching would be pinStripSpace having grown
// looser, and either one moves what pinCopyNote can see without touching a
// line of it.
// pinRecordedAs reports whether some occurrence of one byte in a source was
// recorded as a given construct.
//
// Read off pinCodeOnly's own record at the source's own offsets, which is the
// only place the answer exists: whether a `'` opened a literal or stood there
// as punctuation is a decision the lexer made and nothing about the byte says
// which way it went.
func pinRecordedAs(src string, by []byte, c, kind byte) bool {
	for i := 0; i < len(src) && i < len(by); i++ {
		if src[i] == c && by[i] == kind {
			return true
		}
	}
	return false
}

// pinLexDiffers reports whether one source lexes into different constructs
// under two extensions.
//
// This is the only evidence an extension flag did anything. `nested`, `single`
// and `tick` are three booleans off the extension, and a row that claims to
// exercise one of them while both paths record the same constructs is a row
// about the path it names in name only — the same argument the Swift nested
// comment row is built on, applied to the other two flags.
func pinLexDiffers(src, a, b string) bool {
	_, openA, byA := pinCodeOnly(src, a)
	_, openB, byB := pinCodeOnly(src, b)
	return !slices.Equal(byA, byB) || !slices.Equal(openA, openB)
}

// pinLiteralPastNewline reports whether a literal was still open on the byte
// after a newline.
//
// Which is the whole difference between the two things a quote can start. A
// single-line literal that meets a newline is a runaway and the bytes after it
// are code again; a run of three quotes opens one that carries on, and the
// byte after the newline is still inside it.
func pinLiteralPastNewline(src string, by []byte) bool {
	for i := 0; i+1 < len(src) && i+1 < len(by); i++ {
		if src[i] == '\n' && by[i+1] == pinStringLiteral {
			return true
		}
	}
	return false
}

// pinEscapeHeldLiteral reports whether a backslash inside a literal carried the
// quote after it.
//
// The escape arm's whole job. Without it the quote after the backslash closes
// the literal, the bytes after that are code, and the literal's real closing
// quote opens a runaway — so "no line ran away" is not decoration here, it is
// the half of the reading that says the arm ran.
func pinEscapeHeldLiteral(src string, open []int, by []byte) bool {
	if len(open) > 0 {
		return false
	}
	for i := 0; i+1 < len(src) && i+1 < len(by); i++ {
		if src[i] == '\\' && by[i] == pinStringLiteral &&
			src[i+1] == '"' && by[i+1] == pinStringLiteral {
			return true
		}
	}
	return false
}

// Every place pinCodeOnly decides which construct it is in, and how a row is
// shown to have reached it.
//
// # The dimension the pair census does not count
//
// The census below enumerates ordered pairs of the five constructs and holds
// itself to having a row for each. That is a complete walk of ONE dimension,
// and the completeness arm says so in pairs — so a lexer path could lose a
// construct entirely and the grid would go on reporting itself complete,
// because a pair is named by the two kinds at the boundary and not by the
// bytes that opened them.
//
// The other dimension is this one: pinCodeOnly reaches each of those five
// kinds by more than one route, and the routes are what the extension actually
// changes. Three booleans come off the extension — `nested`, `single`, `tick`
// — and a run of three quotes and a backslash change the answer without any
// extension being consulted at all. Nine of the eleven branches below produce
// or decline to produce the same kind, `a string literal`, so none of them
// adds a PAIR and the census above cannot see any of them go.
//
// When this was written the grid ran 22 rows through `.go`, one `.mjs` and two
// `.swift`, and reached five of these eleven. The five it missed were every
// route that is not a double quote or a comment: the single quote, the
// backtick, both of Swift's refusals to treat them as quotes at all, the
// three-quote run and the escape.
//
// # And what a row has to do to claim one
//
// Read off pinCodeOnly's record rather than off a label. A `via` column would
// be a row asserting its own coverage, which is the shape of evidence this
// file keeps replacing: the probes below ask the lexer what it recorded for
// the row's own bytes, and the four that are about an extension flag ask the
// stronger question — whether the OTHER path records something different.
// Under `.swift` a `'` is punctuation and under `.go` it opens a literal that
// runs to the newline, so a row that claims either has to be a row the two
// paths disagree about.
//
// One correction the probes made to this file's prose on the way in:
// pinCodeOnly's comment calls the three-quote run "Swift's multi-line string",
// and the branch has no extension guard on it. A `"""` in a `.go` harness is
// lexed as a literal that carries past newlines, on every path. The row that
// reaches it is a `.swift` one because that is where the construct is real,
// and the name here says what the code does rather than what the comment
// meant.
// Every place pinCodeOnly decides which construct it is in, and how a row is
// shown to have reached it.
//
// # The dimension the pair census does not count
//
// The census below enumerates ordered pairs of the five constructs and holds
// itself to having a row for each. That is a complete walk of ONE dimension,
// and the completeness arm says so in pairs — so a lexer path could lose a
// construct entirely and the grid would go on reporting itself complete,
// because a pair is named by the two kinds at the boundary and not by the
// bytes that opened them.
//
// The other dimension is this one: pinCodeOnly reaches each of those five
// kinds by more than one route, and the routes are what the extension actually
// changes. Three booleans come off the extension — `nested`, `single`, `tick`
// — and a run of three quotes and a backslash change the answer without any
// extension being consulted at all. Nine of the eleven branches below produce
// or decline to produce the same kind, `a string literal`, so none of them
// adds a PAIR and the census above cannot see any of them go.
//
// When this was written the grid ran 22 rows through `.go`, one `.mjs` and two
// `.swift`, and reached five of these eleven. The five it missed were every
// route that is not a double quote or a comment: the single quote, the
// backtick, both of Swift's refusals to treat them as quotes at all, the
// three-quote run and the escape.
//
// # And what a row has to do to claim one
//
// Read off pinCodeOnly's record rather than off a label. A `via` column would
// be a row asserting its own coverage, which is the shape of evidence this
// file keeps replacing: the probes below ask the lexer what it recorded for
// the row's own bytes, and the four that are about an extension flag ask the
// stronger question — whether the OTHER path records something different.
// Under `.swift` a `'` is punctuation and under `.go` it opens a literal that
// runs to the newline, so a row that claims either has to be a row the two
// paths disagree about.
//
// One correction the probes made to this file's prose on the way in:
// pinCodeOnly's comment calls the three-quote run "Swift's multi-line string",
// and the branch has no extension guard on it. A `"""` in a `.go` harness is
// lexed as a literal that carries past newlines, on every path. The row that
// reaches it is a `.swift` one because that is where the construct is real,
// and the name here says what the code does rather than what the comment
// meant.
//
// # And the list itself is held to the lexer
//
// This table was written to fix a census complete in the dimension it counts,
// and for one session it was itself a hand-written list of eleven checked for
// COVERAGE and not for COMPLETENESS. A twelfth branch in pinCodeOnly — a
// fourth extension, a new delimiter, a raw-string prefix — left this census
// reporting 11 of 11 and the pair grid reporting 20 of 20, with the new route
// reached by nothing. The same complaint one level up, for the fourth time in
// this file.
//
// So the set of decisions is no longer written down here. It is read off
// pinCodeOnly's syntax tree by pinLexDecisionSites, and each row names the
// conditions it is evidence about in `sites`. A decision the tree has and no
// row claims fails TestEveryDecisionPinCodeOnlyMakesIsClaimedByTheBranchCensus
// unless pinLexNotAConstruct excuses it in writing, and a `sites` entry the
// tree does not have fails too — so a reworded condition is a re-read rather
// than a silent pass.
//
// What is still a judgement is the SPLIT: which conditions decide a construct
// and which only say how far one runs. That judgement is now written out
// per condition in pinLexNotAConstruct rather than exercised by omission.
var pinLexBranches = []struct {
	name string
	// The decisions in pinCodeOnly this row is evidence about, spelled as the
	// conditions themselves — the text go/printer produces for them, so a
	// reformat of the lexer is a re-read here and a reword is a failure.
	//
	// More than one row may claim a site, and several do: a flag off the
	// extension is ONE decision with two outcomes, and the row for each
	// outcome is evidence about the same condition. Both rows are needed —
	// the flag read one way is a different route into the source than the
	// flag read the other — and neither is a different decision.
	sites   []string
	reached func(src, ext string, open []int, by []byte) bool
}{
	{name: "// — a comment to the end of the line",
		sites: []string{"src[i] == '/' && i+1 < len(src) && src[i+1] == '/'"},
		reached: func(src, ext string, open []int, by []byte) bool {
			return slices.Contains(by, pinLineComment)
		}},
	// The two block-comment rows claim the same three conditions, because the
	// open and the close are shared and the nesting test is the one flag that
	// tells the two languages apart. Which of them is in force is the `nested`
	// assignment, and that is why both rows name it.
	{name: "/* — a block comment the lexer does not nest",
		sites: []string{
			"src[i] == '/' && i+1 < len(src) && src[i+1] == '*'",
			"src[i] == '*' && i+1 < len(src) && src[i+1] == '/'",
			"nested && src[i] == '/' && i+1 < len(src) && src[i+1] == '*'",
			"nested := ext == \".swift\"",
		},
		reached: func(src, ext string, open []int, by []byte) bool {
			return ext != ".swift" && slices.Contains(by, pinBlockComment)
		}},
	{name: "/* — a block comment the lexer nests, which only Swift does",
		sites: []string{
			"src[i] == '/' && i+1 < len(src) && src[i+1] == '*'",
			"src[i] == '*' && i+1 < len(src) && src[i+1] == '/'",
			"nested && src[i] == '/' && i+1 < len(src) && src[i+1] == '*'",
			"nested := ext == \".swift\"",
		},
		reached: func(src, ext string, open []int, by []byte) bool {
			return ext == ".swift" && slices.Contains(by, pinBlockComment) &&
				pinLexDiffers(src, ext, ".go")
		}},
	// The closing quote of a single-line literal is claimed by all three rows
	// that open one, for the same reason: it is the arm that ends what they
	// started, and a literal none of them can close is a runaway rather than
	// a literal.
	{name: `" — a double-quoted literal`,
		sites: []string{"src[i] == '\"'", "!long && src[i] == q"},
		reached: func(src, ext string, open []int, by []byte) bool {
			return pinRecordedAs(src, by, '"', pinStringLiteral) ||
				pinRecordedAs(src, by, '"', pinRunawayString)
		}},
	{name: "' — a single-quoted literal, which Swift does not have",
		sites: []string{
			"single && src[i] == '\\''",
			"single := ext != \".swift\"",
			"!long && src[i] == q",
		},
		reached: func(src, ext string, open []int, by []byte) bool {
			return ext != ".swift" &&
				(pinRecordedAs(src, by, '\'', pinStringLiteral) ||
					pinRecordedAs(src, by, '\'', pinRunawayString)) &&
				pinLexDiffers(src, ext, ".swift")
		}},
	{name: "` — a template literal, which Swift does not have",
		sites: []string{
			"tick && src[i] == '`'",
			"tick := ext != \".swift\"",
			"!long && src[i] == q",
		},
		reached: func(src, ext string, open []int, by []byte) bool {
			return ext != ".swift" && pinRecordedAs(src, by, '`', pinStringLiteral) &&
				pinLexDiffers(src, ext, ".swift")
		}},
	{name: "' — an apostrophe that is code, because Swift has no single-quoted string",
		sites: []string{"single && src[i] == '\\''", "single := ext != \".swift\""},
		reached: func(src, ext string, open []int, by []byte) bool {
			return ext == ".swift" && pinRecordedAs(src, by, '\'', pinKeptCode) &&
				pinLexDiffers(src, ext, ".go")
		}},
	{name: "` — a backtick that is code, because Swift quotes identifiers with it",
		sites: []string{"tick && src[i] == '`'", "tick := ext != \".swift\""},
		reached: func(src, ext string, open []int, by []byte) bool {
			return ext == ".swift" && pinRecordedAs(src, by, '`', pinKeptCode) &&
				pinLexDiffers(src, ext, ".mjs")
		}},
	// The three-quote run is three decisions and one construct: whether the
	// delimiter is long, how many bytes of it to blank, and what closes it.
	{name: `""" — a literal a run of three quotes opens, which carries past newlines`,
		sites: []string{
			"long := q == '\"' && strings.HasPrefix(src[i:], `\"\"\"`)",
			"long",
			"long && strings.HasPrefix(src[i:], `\"\"\"`)",
		},
		reached: func(src, ext string, open []int, by []byte) bool {
			return strings.Contains(src, `"""`) && len(open) == 0 &&
				pinLiteralPastNewline(src, by)
		}},
	{name: `\ — an escape carrying the quote after it, which keeps the literal open`,
		sites: []string{"src[i] == '\\\\' && i+1 < len(src)"},
		reached: func(src, ext string, open []int, by []byte) bool {
			return pinEscapeHeldLiteral(src, open, by)
		}},
	{name: "the newline that turns an unclosed literal into a runaway",
		sites: []string{"!long && q != '`' && src[i] == '\\n'"},
		reached: func(src, ext string, open []int, by []byte) bool {
			return len(open) > 0
		}},
}

// The conditions in pinCodeOnly that are not a decision about which construct
// it is in, each with the reason it is not.
//
// pinLexDecisionSites takes every condition the function turns on, because a
// rule that took only some of them would be the judgement it is here to
// remove. Most of what it finds is a construct decision and is claimed by a
// row above; the rest is here, and it is all of one shape — an EXTENT (how far
// a construct already decided on runs) or the second pass's bookkeeping.
//
// Written out per condition rather than filtered by a rule about node kinds,
// because "loop conditions do not decide constructs" is exactly the sort of
// argument this file keeps converting into a statement. A new loop that DOES
// decide one arrives here as an unexcused site and has to be answered.
var pinLexNotAConstruct = []struct{ site, why string }{
	{"i < len(src)",
		"the scan's own bound, in the main loop and again in the literal body: " +
			"where the source ends, not what is in it"},
	{"i < len(src) && src[i] != '\\n'",
		"how far a line comment runs, once the `//` has decided it is one"},
	{"i < len(src) && depth > 0",
		"how far a block comment runs, once the `/*` has decided it is one — " +
			"`depth` is the nesting count and the arm that moves it is claimed"},
	{"k < n",
		"blanking the opening delimiter's own bytes, one or three of them; `n` " +
			"was chosen by the `long` decision, which a row claims"},
	{"k < i",
		"the walk back over a runaway's bytes to retell them, after the newline " +
			"has decided what they were"},
	{"i < len(src) && at < len(openAt)",
		"the second pass, which turns recorded offsets into line numbers and " +
			"lexes nothing"},
	{"at < len(openAt) && openAt[at] == i",
		"the same pass, emitting the offsets that land on this byte"},
	{"src[i] == '\\n'",
		"counting lines in that pass. The newline that DECIDES something is the " +
			"one inside the literal arm, and it is claimed by the runaway row"},
	{"out[i] != '\\n'",
		"blank() leaving newlines alone, so a blanked span keeps its lines — a " +
			"rule about what a construct's bytes become, not about which it is"},
	{"blankedBy[k] == pinStringLiteral",
		"the retell rewriting only the bytes the literal arm recorded, so a " +
			"comment inside a runaway's span keeps its own kind"},
}

// pinLexDecisionSites is every condition pinCodeOnly turns on, read off the
// function's syntax tree rather than listed beside it.
//
// # What counts as one
//
// Every `if` condition, every `case` expression, every `for` condition and
// every boolean assignment in the body — and each of those split on its
// top-level `||`, because a disjunction is the lexer offering separate routes
// to one arm and the three quote characters are exactly that: `"` always,
// `'` when `single`, a backtick when `tick`. A rule that took the case
// expression whole would call those one branch and let two of them go without
// a row.
//
// A site is identified by the text go/printer produces for it, which has three
// consequences worth stating. Two conditions that read identically are one
// site — `i < len(src)` appears twice and is the same claim both times.
// Reformatting the lexer does not move a site, because the printer normalises
// spacing. And REWORDING one does: a condition spelled differently is a
// different site, which fails as an unclaimed decision and an orphaned claim
// at once. That is the intended cost — the reword is the moment to re-read
// whether the row above still describes what the lexer does.
//
// A boolean assignment is identified by the whole statement, not by its
// right-hand side, because `single` and `tick` are both `ext != ".swift"` and
// are two different decisions.
//
// # Why the source and not a coverage tool
//
// Go can report which lines a test executed, and that would answer a narrower
// question: whether some test somewhere ran the branch. What this census is
// for is whether a row in THIS table demonstrates it, on a source whose
// extension turns the flag on, with the other path recording something
// different. Coverage cannot tell that from a line reached incidentally by a
// harness scan.
func pinLexDecisionSites(t *testing.T) []string {
	t.Helper()

	// The package's own directory, whatever file the lexer happens to live in:
	// a function moved between files is not a change to the census.
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("reading this package's directory: %v", err)
	}
	fset := token.NewFileSet()
	var fn *ast.FuncDecl
	var in string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") {
			continue
		}
		file, err := parser.ParseFile(fset, e.Name(), nil, 0)
		if err != nil {
			t.Fatalf("parsing %s: %v", e.Name(), err)
		}
		for _, d := range file.Decls {
			fd, ok := d.(*ast.FuncDecl)
			if ok && fd.Recv == nil && fd.Name.Name == "pinCodeOnly" && fd.Body != nil {
				fn, in = fd, e.Name()
			}
		}
	}
	if fn == nil {
		t.Fatalf("no func pinCodeOnly in this package.\n\n" +
			"This census reads the lexer's decisions off its syntax tree. A lexer " +
			"that has been renamed or moved out of the package is a census with " +
			"nothing to be about, and reporting zero decisions would be a green " +
			"run saying every one of them is covered.")
	}
	t.Logf("pinCodeOnly's decisions read off %s", in)

	text := func(n ast.Node) string {
		var b strings.Builder
		if err := printer.Fprint(&b, fset, n); err != nil {
			t.Fatalf("printing a node of pinCodeOnly: %v", err)
		}
		return b.String()
	}
	// Split on top-level `||`, through parentheses. See the note above: a
	// disjunct is a route, and the routes are what this census counts.
	var disjuncts func(ast.Expr) []ast.Expr
	disjuncts = func(e ast.Expr) []ast.Expr {
		switch x := e.(type) {
		case *ast.ParenExpr:
			return disjuncts(x.X)
		case *ast.BinaryExpr:
			if x.Op == token.LOR {
				return append(disjuncts(x.X), disjuncts(x.Y)...)
			}
		}
		return []ast.Expr{e}
	}

	seen, sites := map[string]bool{}, []string{}
	add := func(n ast.Node) {
		if s := text(n); !seen[s] {
			seen[s] = true
			sites = append(sites, s)
		}
	}
	addCond := func(e ast.Expr) {
		for _, d := range disjuncts(e) {
			add(d)
		}
	}
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		switch s := n.(type) {
		case *ast.IfStmt:
			addCond(s.Cond)
		case *ast.CaseClause:
			for _, e := range s.List {
				addCond(e)
			}
		case *ast.ForStmt:
			if s.Cond != nil {
				addCond(s.Cond)
			}
		case *ast.AssignStmt:
			// A boolean assignment is a decision taken once and read later —
			// the three extension flags and `long`. Identified by the whole
			// statement: `single` and `tick` have identical right-hand sides.
			for _, r := range s.Rhs {
				if pinBoolExpr(r) {
					add(s)
				}
			}
		}
		return true
	})
	slices.Sort(sites)
	return sites
}

// pinBoolExpr reports whether an expression is a comparison or a logical
// operation, which is how a boolean assignment is told from any other.
//
// Syntactic rather than typed, because running the type checker over the
// package to classify four assignments would be a second toolchain in a test
// about a lexer. The cost of the approximation is in the safe direction: a
// boolean assignment spelled some other way (a call returning bool, a copy of
// another flag) is a site this census does not see, and the arms that DECIDE a
// construct — `if`, `case` — are seen whatever they are spelled with.
func pinBoolExpr(e ast.Expr) bool {
	switch x := e.(type) {
	case *ast.BinaryExpr:
		switch x.Op {
		case token.EQL, token.NEQ, token.LSS, token.GTR, token.LEQ, token.GEQ,
			token.LAND, token.LOR:
			return true
		}
	case *ast.UnaryExpr:
		return x.Op == token.NOT
	case *ast.ParenExpr:
		return pinBoolExpr(x.X)
	}
	return false
}

// The branch census, held to the lexer in both directions.
//
// pinLexBranches fixed a pair grid that was complete in the dimension it
// counted and blind to the routes into it. This is the same question asked of
// the fix: the list of eleven was hand-written, checked for coverage, and a
// twelfth branch in pinCodeOnly would have left it reporting eleven of eleven
// with the new route reached by nothing.
//
// Both directions, because either alone is a half-reading:
//
//	a decision with no row and no excuse   a branch nothing demonstrates
//	a row naming no decision               a claim about a lexer that changed
//
// The second is the one a reword produces, and it is a failure rather than a
// silent pass on purpose: a condition spelled differently is a condition
// somebody edited, and the row above it is prose written about the old one.
func TestEveryDecisionPinCodeOnlyMakesIsClaimedByTheBranchCensus(t *testing.T) {
	sites := pinLexDecisionSites(t)
	derived := map[string]bool{}
	for _, s := range sites {
		derived[s] = true
	}

	// Row → decisions, and the inverse, so the message can say which row is
	// the evidence for a decision rather than that there is some.
	claimedBy := map[string][]string{}
	for _, branch := range pinLexBranches {
		if len(branch.sites) == 0 {
			t.Errorf("the census row %q names no decision in pinCodeOnly.\n\n"+
				"A row without one is a probe that demonstrates something about the "+
				"lexer and does not say what, which leaves the completeness arm "+
				"below unable to count it — and completeness is the whole reason "+
				"this column exists.", branch.name)
			continue
		}
		for _, site := range branch.sites {
			if !derived[site] {
				t.Errorf("the census row %q is evidence about `%s`, and pinCodeOnly "+
					"has no such condition.\n\n"+
					"Sites are the text go/printer gives a condition, so this is "+
					"either a decision that has been deleted — in which case the row "+
					"is about a lexer that no longer exists — or one that has been "+
					"reworded, in which case it is worth reading whether the row's "+
					"own sentence still describes what the code does. The decisions "+
					"pinCodeOnly makes today are:\n\t%s",
					branch.name, site, strings.Join(sites, "\n\t"))
				continue
			}
			claimedBy[site] = append(claimedBy[site], branch.name)
		}
	}

	excused := map[string]string{}
	for _, e := range pinLexNotAConstruct {
		if !derived[e.site] {
			t.Errorf("`%s` is excused from this census as %s, and pinCodeOnly has no "+
				"such condition.\n\n"+
				"An excuse for a decision that is not there excuses nothing, and it "+
				"is the shape that goes stale silently: the condition it was written "+
				"about was reworded or removed, and what is left is a sentence that "+
				"will one day be read as covering something else.", e.site, e.why)
			continue
		}
		if rows := claimedBy[e.site]; len(rows) > 0 {
			t.Errorf("`%s` is both excused as %s and claimed as evidence by %s.\n\n"+
				"One of the two is wrong about what the condition does. A decision "+
				"that selects a construct has a row; one that says how far an "+
				"already-selected construct runs has an excuse; nothing is both.",
				e.site, e.why, strings.Join(rows, " and "))
		}
		excused[e.site] = e.why
	}

	unclaimed := []string{}
	for _, site := range sites {
		if len(claimedBy[site]) == 0 && excused[site] == "" {
			unclaimed = append(unclaimed, site)
		}
	}
	if len(unclaimed) > 0 {
		t.Errorf("%d of the %d decisions pinCodeOnly makes are claimed by no census "+
			"row and excused by nothing:\n\t%s\n\n"+
			"This is the arm the branch census was missing when it was itself a "+
			"hand-written list of eleven: a route added to the lexer left the "+
			"census reporting every branch reached and the pair grid reporting "+
			"every pair built, with nothing anywhere reaching the new one. Either "+
			"build a row that demonstrates it — a source carrying the bytes, an "+
			"extension turning the flag on, and for a flag the two paths recording "+
			"something different — or say in pinLexNotAConstruct why it decides no "+
			"construct.",
			len(unclaimed), len(sites), strings.Join(unclaimed, "\n\t"))
	}

	t.Logf("pinCodeOnly turns on %d distinct conditions: %d of them decide which "+
		"construct the lexer is in and are claimed by the %d rows of this census, "+
		"and %d say how far an already-decided construct runs or belong to the "+
		"line-number pass, each with its reason. The list of branches is no longer "+
		"written beside the lexer — it is read off it, so a twelfth route cannot "+
		"arrive with this census still saying every route is covered.",
		len(sites), len(sites)-len(excused), len(pinLexBranches), len(excused))
}

func TestWhichConstructTransitionsAStraddleCanBeBuiltFrom(t *testing.T) {
	grid := []struct {
		// The pair, as the bytes pinCodeOnly records — not as prose. The
		// sentence a failure prints goes through pinConstructName, so the two
		// cannot drift; and the completeness check below counts ordered pairs,
		// which it can only do over the kinds themselves.
		from, to byte
		src, ext string
		// The citation, and the same citation with the boundary bytes taken
		// out. Empty when the boundary is a newline and there is nothing to
		// take out.
		phrase, bare string
		// What the boundary is made of, for the line at the end.
		delim string
		// The run of constructs the copy's span meets, as pinBlankedAs returns
		// it. Written out so a row that quietly started meeting a different
		// construct is a failure and not a passing test about something else.
		kinds []byte
		// The lines pinCodeOnly ended inside a single-line string on. The
		// runaway rows have one by construction — that is what makes them
		// runaways — and every other row has none.
		open []int
	}{
		// The four a citation can walk OUT of live code into. Each boundary is
		// the construct's own opening delimiter, and the phrase carries it.
		{from: pinKeptCode, to: pinLineComment, delim: "//",
			src: "x = pinSame( // total,\ny)\n", ext: ".go",
			phrase: "pinSame( // total, y)", bare: "pinSame( total, y)",
			kinds: []byte{pinKeptCode, pinLineComment}},
		{from: pinKeptCode, to: pinBlockComment, delim: "/*",
			src: "x = pinSame(/*total*/)\n", ext: ".go",
			phrase: "pinSame(/*total*/)", bare: "pinSame(total)",
			kinds: []byte{pinKeptCode, pinBlockComment}},
		{from: pinKeptCode, to: pinStringLiteral, delim: `"`,
			src: "log(pinSame(\"x\"))\n", ext: ".go",
			phrase: "pinSame(\"x\")", bare: "pinSame(x)",
			kinds: []byte{pinKeptCode, pinStringLiteral}},
		{from: pinKeptCode, to: pinRunawayString, delim: `"`,
			src: "x = pinSame(\"total,\ny)\n", ext: ".go",
			phrase: "pinSame(\"total, y)", bare: "pinSame(total, y)",
			kinds: []byte{pinKeptCode, pinRunawayString}, open: []int{1}},

		// Out of a line comment, whose boundary is the newline and therefore
		// costs the phrase nothing — the two rows the fixtures above were
		// built on, and a third showing the same head reaching a comment.
		{from: pinLineComment, to: pinKeptCode,
			src: "x = 1 // pinSame(total,\nc.offer) + 2\n", ext: ".go",
			phrase: "pinSame(total, c.offer)",
			kinds:  []byte{pinLineComment, pinKeptCode}},
		{from: pinLineComment, to: pinStringLiteral, delim: `"`,
			src: "x = 1 // pinSame(\n\"x\") + y\n", ext: ".mjs",
			phrase: "pinSame(\"x\")", bare: "pinSame(x)",
			kinds: []byte{pinLineComment, pinStringLiteral, pinKeptCode}},
		{from: pinLineComment, to: pinBlockComment, delim: "/*",
			src: "x = 1 // pinSame(\n/*total*/)\n", ext: ".go",
			phrase: "pinSame( /*total*/)", bare: "pinSame(total)",
			kinds: []byte{pinLineComment, pinBlockComment, pinKeptCode}},

		// Out of a block comment, whose boundary is `*/` — the transition the
		// note above pinBlankedAs called unreachable. It is reachable, and the
		// citation that reaches it is one that quotes the comment's own close.
		{from: pinBlockComment, to: pinKeptCode, delim: "*/",
			src: "/* pinSame( */ x)\n", ext: ".go",
			phrase: "pinSame( */ x)", bare: "pinSame( x)",
			kinds: []byte{pinBlockComment, pinKeptCode}},
		{from: pinBlockComment, to: pinStringLiteral, delim: `*/ "`,
			src: "/* pinSame( */ \"x\")\n", ext: ".go",
			phrase: "pinSame( */ \"x\")", bare: "pinSame( x)",
			kinds: []byte{pinBlockComment, pinStringLiteral, pinKeptCode}},

		// Out of a closed literal, whose boundary is its own closing quote.
		{from: pinStringLiteral, to: pinKeptCode, delim: `"`,
			src: "x = \"pinSame(\" + y\n", ext: ".go",
			phrase: "pinSame(\" + y", bare: "pinSame( + y",
			kinds: []byte{pinStringLiteral, pinKeptCode}},
		{from: pinStringLiteral, to: pinLineComment, delim: `" //`,
			src: "a = \"pinSame(\" // total)\n", ext: ".go",
			phrase: "pinSame(\" // total)", bare: "pinSame( total)",
			kinds: []byte{pinStringLiteral, pinLineComment}},

		// And out of the runaway, which the newline closes. This is the blind
		// spot the kind exists for: the code before the quote stands, so the
		// line reads live while the assertion after it is gone.
		{from: pinRunawayString, to: pinKeptCode,
			src: "x = 1 + \"pinSame(total,\nc.offer) + 2\n", ext: ".go",
			phrase: "pinSame(total, c.offer)",
			kinds:  []byte{pinRunawayString, pinKeptCode}, open: []int{1}},
		{from: pinRunawayString, to: pinStringLiteral, delim: `"`,
			src: "x = 1 + \"pinSame(\n\"total\")\n", ext: ".go",
			phrase: "pinSame( \"total\")", bare: "pinSame( total)",
			kinds: []byte{pinRunawayString, pinStringLiteral, pinKeptCode},
			open:  []int{1}},

		// And the seven the first version of this census left out, which is
		// what made it a sample calling itself a grid. Every one of them is
		// reachable, and each is the same rule again: the boundary is the
		// newline, or the citation spells it.
		{from: pinLineComment, to: pinRunawayString, delim: `"`,
			src: "x = 1 // pinSame(\n\"total,\ny)\n", ext: ".go",
			phrase: "pinSame( \"total, y)", bare: "pinSame( total, y)",
			kinds: []byte{pinLineComment, pinRunawayString, pinKeptCode},
			open:  []int{2}},
		{from: pinBlockComment, to: pinLineComment, delim: "*/ //",
			src: "/* pinSame( */ // total)\n", ext: ".go",
			phrase: "pinSame( */ // total)", bare: "pinSame( total)",
			kinds: []byte{pinBlockComment, pinLineComment}},
		{from: pinBlockComment, to: pinRunawayString, delim: `*/ "`,
			src: "/* pinSame( */ \"total,\ny)\n", ext: ".go",
			phrase: "pinSame( */ \"total, y)", bare: "pinSame( total, y)",
			kinds: []byte{pinBlockComment, pinRunawayString, pinKeptCode},
			open:  []int{1}},
		{from: pinStringLiteral, to: pinBlockComment, delim: `"/*`,
			src: "a = \"pinSame(\"/*total*/)\n", ext: ".go",
			phrase: "pinSame(\"/*total*/)", bare: "pinSame(total)",
			kinds: []byte{pinStringLiteral, pinBlockComment, pinKeptCode}},
		{from: pinStringLiteral, to: pinRunawayString, delim: `" + "`,
			src: "a = \"pinSame(\" + \"total,\ny)\n", ext: ".go",
			phrase: "pinSame(\" + \"total, y)", bare: "pinSame( + total, y)",
			kinds: []byte{pinStringLiteral, pinKeptCode, pinRunawayString},
			open:  []int{1}},
		{from: pinRunawayString, to: pinLineComment, delim: "//",
			src: "x = \"pinSame(\n// total)\n", ext: ".go",
			phrase: "pinSame( // total)", bare: "pinSame( total)",
			kinds: []byte{pinRunawayString, pinLineComment}, open: []int{1}},
		{from: pinRunawayString, to: pinBlockComment, delim: "/*",
			src: "x = \"pinSame(\n/*total*/)\n", ext: ".go",
			phrase: "pinSame( /*total*/)", bare: "pinSame( total)",
			kinds: []byte{pinRunawayString, pinBlockComment, pinKeptCode},
			open:  []int{1}},

		// And the third lexer, which no row above reaches.
		//
		// pinCodeOnly branches on the extension three ways — Swift nests block
		// comments, has no single-quoted string, and treats a backtick as code
		// — and every row above runs one of the two non-nesting paths. A
		// nested comment closing into code is a boundary only Swift can spell,
		// and this row is written so the two lexers disagree about it: with
		// nesting the first `*/` closes the inner comment and the literal
		// after it is still inside the outer one, so the copy meets a comment
		// and then code; without nesting the first `*/` ends the comment
		// outright and the same bytes are a literal the copy passes through.
		// A row where both paths answer alike would have been a Swift row in
		// name only.
		{from: pinBlockComment, to: pinKeptCode, delim: `*/ "y" */`,
			src: "/* /* pinSame( */ \"y\" */ z)\n", ext: ".swift",
			phrase: "pinSame( */ \"y\" */ z)", bare: "pinSame( z)",
			kinds: []byte{pinBlockComment, pinKeptCode}},
		{from: pinLineComment, to: pinKeptCode,
			src: "x = 1 // pinSame(total,\nc.offer) + 2\n", ext: ".swift",
			phrase: "pinSame(total, c.offer)",
			kinds:  []byte{pinLineComment, pinKeptCode}},

		// And the six routes into a literal that no pair can be missing,
		// because every one of them ends at the same kind. See pinLexBranches:
		// the census above walks ordered PAIRS and is complete in that
		// dimension, and nine of the lexer's eleven ways of deciding a
		// construct produce or decline to produce `a string literal` — so all
		// six of these rows are the pair `code the lexer kept` then `a string
		// literal`, which had a row already, and the grid could have lost any
		// of them without a word.
		//
		// The first two are the flags read one way and the next two are the
		// same flags read the other, which is where Swift differs from
		// JavaScript in the direction nothing else here reaches: the byte that
		// opens a literal on one path is ordinary code on the other, and a row
		// is only about a flag if the two paths disagree about its source.
		{from: pinKeptCode, to: pinStringLiteral, delim: `'`,
			src: "x = pinSame('a')\n", ext: ".go",
			phrase: "pinSame('a')", bare: "pinSame(a)",
			kinds: []byte{pinKeptCode, pinStringLiteral}},
		{from: pinKeptCode, to: pinStringLiteral, delim: "`",
			src: "x = pinSame(`a`)\n", ext: ".mjs",
			phrase: "pinSame(`a`)", bare: "pinSame(a)",
			kinds: []byte{pinKeptCode, pinStringLiteral}},
		// An apostrophe in Swift code, which `.go` would read as a quote
		// opening a literal that runs to the newline — so the citation after
		// it is a live literal on one path and inside a runaway on the other.
		{from: pinKeptCode, to: pinStringLiteral, delim: `"`,
			src: "let a = it's\npinSame(\"x\") + y\n", ext: ".swift",
			phrase: "pinSame(\"x\")", bare: "pinSame(x)",
			kinds: []byte{pinKeptCode, pinStringLiteral}},
		// And a backtick quoting a keyword as an identifier, which `.mjs`
		// would read as a template literal swallowing the rest of the file.
		{from: pinKeptCode, to: pinStringLiteral, delim: `"`,
			src: "let `class` = pinSame(\"x\")\n", ext: ".swift",
			phrase: "pinSame(\"x\")", bare: "pinSame(x)",
			kinds: []byte{pinKeptCode, pinStringLiteral}},
		// The run of three quotes, whose literal carries past the newline
		// instead of ending there — the one construct in the lexer that is
		// spelled in quotes and does NOT produce a runaway. `open` being empty
		// is half of what this row asserts.
		{from: pinKeptCode, to: pinStringLiteral, delim: `"""`,
			src: "x = pinSame(\"\"\"a\nb\"\"\")\n", ext: ".swift",
			phrase: "pinSame(\"\"\"a b\"\"\")", bare: "pinSame(a b)",
			kinds: []byte{pinKeptCode, pinStringLiteral}},
		// And the escape, which is the other way a quote fails to close a
		// literal. Without that arm the quote after the backslash ends the
		// literal, `b` is code, and the real closing quote opens a runaway —
		// so this row's empty `open` is the assertion that the arm ran.
		{from: pinKeptCode, to: pinStringLiteral, delim: `"\"`,
			src: "x = pinSame(\"a\\\"b\") + c\n", ext: ".go",
			phrase: "pinSame(\"a\\\"b\")", bare: "pinSame(ab)",
			kinds: []byte{pinKeptCode, pinStringLiteral}},
	}
	// How each row's boundary is crossed, for the line at the end: the two the
	// strip removes for free, and the rest the citation has to spell.
	across, delims := 0, []string{}
	seen := map[string]bool{}
	for _, c := range grid {
		if c.delim == "" {
			across++
			continue
		}
		if !seen[c.delim] {
			seen[c.delim] = true
			delims = append(delims, fmt.Sprintf("%q", c.delim))
		}
	}
	// And which of pinCodeOnly's own branches each row's source reaches. See
	// pinLexBranches: the pair census below is a complete walk of one
	// dimension and this is the other one, collected here off the same lex the
	// row's assertions are read from rather than from a second one.
	routes := map[string][]string{}
	for _, c := range grid {
		// Spelled through the same function the sentence under test uses, so a
		// row naming a pair and a message naming a pair cannot disagree.
		what := pinConstructName(c.from) + " then " + pinConstructName(c.to)
		code, open, by := pinCodeOnly(c.src, c.ext)
		for _, branch := range pinLexBranches {
			if branch.reached(c.src, c.ext, open, by) {
				routes[branch.name] = append(routes[branch.name], c.ext)
			}
		}
		if !slices.Equal(open, c.open) {
			t.Errorf("%s: pinCodeOnly ended inside an unterminated string on %v of "+
				"this row's lines and the row says %v.\n\n"+
				"The runaway is one of the five constructs and it is the one this "+
				"return value is the whole record of, so a row that gained or lost "+
				"one is a row about a different pair than the one it names.",
				what, open, c.open)
			continue
		}
		copies := pinPhraseCopies(c.src, c.phrase)
		if len(copies) != 1 {
			t.Errorf("%s: this row spells %q once in %q and pinPhraseCopies finds "+
				"%d.\n\n"+
				"A transition is buildable when the bytes between the two halves "+
				"survive pinStripSpace or are carried by the phrase itself. This row "+
				"is the claim that this pair is buildable, and without exactly one "+
				"match there is no copy to ask what it was taken by.",
				what, c.phrase, c.src, len(copies))
			continue
		}
		if kinds := pinBlankedAs(code, by, copies[0]); !slices.Equal(kinds, c.kinds) {
			t.Errorf("%s: the copy's own bytes were taken by %q and the row says "+
				"%q.\n\n"+
				"pinBlankedAs reads the record the lexer wrote at the source's own "+
				"offsets, so this is what pinCopyNote's sentence will name and in "+
				"what order. A run that changed is pinCodeOnly recording a different "+
				"construct for these bytes — which moves every sentence built on it "+
				"without touching one of them.",
				what, string(kinds), string(c.kinds))
		}
		// And the other direction: the delimiter is carried, not skipped.
		if c.bare == "" {
			if c.delim != "" {
				t.Errorf("%s: the row names a boundary of %q and gives no phrase "+
					"without it.\n\n"+
					"The negative is half the reading. A raw-byte boundary is "+
					"reachable only because the citation spells it, and the way that "+
					"is shown is the same source declining to match the same phrase "+
					"with those bytes taken out.", what, c.delim)
			}
			continue
		}
		if n := len(pinPhraseCopies(c.src, c.bare)); n != 0 {
			t.Errorf("%s: %q — the phrase with its %q taken out — matches %d "+
				"time(s) in %q.\n\n"+
				"pinStripSpace removes whitespace between tokens and nothing else, "+
				"so a boundary made of raw bytes has to be spelled by the citation "+
				"to be crossed. A bare phrase that matches means the strip has grown "+
				"looser, and a looser strip matches phrases the harness does not "+
				"actually contain — which is this whole table reporting citations to "+
				"assertions that are not there.",
				what, c.bare, c.delim, n, c.src)
		}
	}
	// And the grid being a grid.
	//
	// The first version of this census asked about thirteen ordered pairs and
	// called them "the grid", which is a sample wearing a walk's name: seven
	// pairs were missing, every one of them reachable, and nothing said which
	// or why. A reader had the same problem the fixtures had one level down —
	// general machinery, and evidence from whatever somebody thought of.
	//
	// So the pairs are enumerated rather than listed. Five constructs, twenty
	// ordered pairs with the two ends different, and a row for each. A pair
	// with no row is named here rather than quietly absent; a self-pair is not
	// a transition and is not asked for, because a copy wholly inside one
	// construct is the case pinBlankedAs returns one kind for and the wrapped
	// fixture above is about.
	//
	// Rows beyond twenty are fine and are the point of the extension column: a
	// pair can be reachable by more than one boundary, and Swift's nested
	// block comment is the same pair through a different lexer path.
	kinds := []byte{pinKeptCode, pinLineComment, pinBlockComment,
		pinStringLiteral, pinRunawayString}
	asked := map[[2]byte]bool{}
	for _, c := range grid {
		asked[[2]byte{c.from, c.to}] = true
	}
	pairs, missing := 0, []string{}
	for _, from := range kinds {
		for _, to := range kinds {
			if from == to {
				continue
			}
			pairs++
			if !asked[[2]byte{from, to}] {
				missing = append(missing,
					pinConstructName(from)+" then "+pinConstructName(to))
			}
		}
	}
	if len(missing) > 0 {
		t.Errorf("%d of the %d ordered construct pairs have no row in this census: "+
			"%s.\n\n"+
			"pinBlankedAs walks a span and names every construct it meets, in the "+
			"order it meets them, and that walk is written for constructs in general "+
			"— so a pair with no row is a transition the machinery handles and "+
			"nothing has asked it about. The seven this census was missing when it "+
			"was written were all reachable, and each of them was reachable the same "+
			"way: the boundary is a newline the strip removes, or the citation is "+
			"spelled with the delimiter in it. Build the row rather than deleting "+
			"this arm — a pair that genuinely cannot be built is a fact about "+
			"pinCodeOnly worth writing down, and it belongs here with its reason.",
			len(missing), pairs, strings.Join(missing, ", "))
	}

	// And the coverage the line below claims, asserted rather than counted on.
	//
	// The sentence says every construct appears on both sides of a boundary,
	// and a row deleted or reworded would leave it saying so over a grid that
	// no longer does — which is the count-shaped evidence this census was
	// written to replace, arriving in the census's own recital. Read off the
	// kind runs the rows assert rather than off their prose names, so the two
	// cannot drift apart.
	heads, tails := map[byte]bool{}, map[byte]bool{}
	for _, c := range grid {
		if len(c.kinds) == 0 {
			continue
		}
		heads[c.kinds[0]] = true
		for _, kind := range c.kinds[1:] {
			tails[kind] = true
		}
	}
	for _, kind := range kinds {
		if !heads[kind] || !tails[kind] {
			t.Errorf("%s leads a row in this grid: %v; and is crossed INTO by one: "+
				"%v.\n\n"+
				"pinBlankedAs walks a span and names whatever it meets, in order, and "+
				"a construct that only ever appears at one end of a boundary is a "+
				"half of that walk nothing here has asked about. The line below says "+
				"every construct appears on both sides — that claim is this grid's "+
				"and has to be held by it.",
				pinConstructName(kind), heads[kind], tails[kind])
		}
	}
	// # And the same completeness question in the other dimension
	//
	// The pairs above are a complete walk of the five kinds and say nothing
	// about the routes into them — see pinLexBranches. Nine of the eleven
	// branches end at `a string literal`, so a lexer path could stop producing
	// one and every arm above would still pass: the pair is named by the kinds
	// at the boundary, not by the byte that opened them.
	//
	// Held the same way the pairs are, and for the same reason: a branch with
	// no row is a decision pinCodeOnly makes that nothing in this file has put
	// a citation across, and that is a fact worth a sentence rather than a gap
	// somebody notices later.
	unreached := []string{}
	for _, branch := range pinLexBranches {
		if len(routes[branch.name]) == 0 {
			unreached = append(unreached, branch.name)
		}
	}
	if len(unreached) > 0 {
		t.Errorf("%d of the %d branches pinCodeOnly decides a construct at have no "+
			"row in this census: %s.\n\n"+
			"The pair census above is complete in pairs and cannot see this: most of "+
			"these branches end at the same kind, so a route into a literal can go "+
			"without changing which pairs exist. Each of them is reached by a row "+
			"whose source carries the bytes and whose extension turns the flag on — "+
			"and, for the four that are about an extension flag, by the two paths "+
			"recording something different about the same source, which is the only "+
			"evidence the flag did anything. Build the row rather than deleting this "+
			"arm.",
			len(unreached), len(pinLexBranches), strings.Join(unreached, "; "))
	}

	// And what the walk came to, which is the reading that says the general
	// machinery has general evidence.
	//
	// Counted off the same rows the assertions are, and split by what the
	// boundary is made of, because that is the distinction the whole census is
	// about: the newline pairs cost the citation nothing and were the only ones
	// anybody had built, and the rest are reachable the moment a citation
	// quotes the delimiter — which is the ordinary shape of a citation that
	// quotes a call.
	// Which extensions reached each branch, so a green run's line says where the
	// evidence for the other dimension actually is rather than that there is
	// some.
	routeSaid := make([]string, 0, len(pinLexBranches))
	for _, branch := range pinLexBranches {
		seenExt, exts := map[string]bool{}, []string{}
		for _, ext := range routes[branch.name] {
			if !seenExt[ext] {
				seenExt[ext] = true
				exts = append(exts, ext)
			}
		}
		routeSaid = append(routeSaid, fmt.Sprintf("%s (%d row(s), %s)",
			branch.name, len(routes[branch.name]), strings.Join(exts, " ")))
	}
	t.Logf("every one of the %d ordered construct pairs this census asks about is "+
		"reachable — %d across a newline, which the strip removes and which "+
		"therefore costs the citation nothing, and %d across raw bytes the "+
		"citation itself carries (%s), each of those refusing to match the moment "+
		"the same phrase is spelled without them. The four constructs pinCodeOnly "+
		"records and the code it keeps all appear on both sides of a boundary, so "+
		"pinBlankedAs's run of kinds is exercised over the transitions rather than "+
		"over the one a fixture happened to be written with, and all %d ordered "+
		"pairs of them have a row. And the other dimension, which pairs cannot "+
		"count: %d of the %d branches pinCodeOnly decides a construct at are "+
		"reached — %s",
		len(grid), across, len(grid)-across, strings.Join(delims, ", "), pairs,
		len(pinLexBranches)-len(unreached), len(pinLexBranches),
		strings.Join(routeSaid, ", "))
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
	stripped, lines, _ := pinStripSpaceMap(s)
	return stripped, lines
}

// pinStripSpaceMap is that index and the one beside it: the raw BYTE every
// surviving byte came from.
//
// The line was enough while the only question was "which line is this copy
// on". It is not enough for the question underneath it — what did pinCodeOnly
// do to this copy — because that pass blanks spans and not lines: a phrase
// inside a string literal on a line that also carries code sits on a line that
// still looks live. The offset is what turns the copy into a slice of the
// lexed source, and the lexer's in-place blanking is what makes that slice
// mean anything: same bytes, same offsets, prose replaced by spaces.
//
// One array per byte of the stripped string, indexed the same way, so a match
// there is a line range and a byte range in one lookup. See pinPhraseCopies.
func pinStripSpaceMap(s string) (string, []int, []int) {
	// Pass one: the fields, joined by a single space.
	var joined []byte
	var jline, joff []int
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
			// The separator is credited to the first byte of the field it
			// precedes, for the same reason its line is: a span that began at
			// the whitespace run would reach back over bytes no copy occupies,
			// and the run can be a newline and an indent.
			joff = append(joff, i)
		}
		inField = true
		n := len(string(r))
		joined = append(joined, s[i:i+n]...)
		for k := 0; k < n; k++ {
			jline = append(jline, line)
			joff = append(joff, i+k)
		}
	}
	// Pass two: a space survives only between two word characters, which is
	// pinStripSpace's rule and the reason it has one. See the note above.
	var out []byte
	var oline, ooff []int
	for i := 0; i < len(joined); i++ {
		if joined[i] == ' ' {
			if i > 0 && i+1 < len(joined) && word(joined[i-1]) && word(joined[i+1]) {
				out = append(out, ' ')
				oline = append(oline, jline[i])
				ooff = append(ooff, joff[i])
			}
			continue
		}
		out = append(out, joined[i])
		oline = append(oline, jline[i])
		ooff = append(ooff, joff[i])
	}
	return string(out), oline, ooff
}

// pinCopyNote says which of a phrase's surviving copies pinCodeOnly took for
// prose, which of them it blanked inside a line it left standing, and which it
// kept — or "" when it cannot place any of them.
//
// # The count was the whole of what the reader got
//
// The deletion report can find a phrase spelled twice in the file and gone from
// the code, and it said so: "the phrase, spelled 2 times in the file, is on
// [40 42]". Two lines and a number, and the note's own argument turns on
// exactly the distinction it left out — the sentence above an assertion
// outliving the assertion is the rot this half exists to catch, and a copy
// sitting in live code is the lexer's blind spot or a string literal. Those
// send a reader to two different places, and both arrived as the same sentence.
//
// # And a line was the wrong thing to ask
//
// The pass that blanked one of them knows, and it knows it in SPANS.
// pinCodeOnly replaces prose with spaces in place, so its output has the file's
// bytes at the file's offsets — and the first version of this note read that
// output a line at a time: a line that came back whitespace was prose, a line
// with anything on it was live.
//
// That is right about the line and wrong about the copy, and it is wrong in the
// direction that costs a reader an afternoon. A phrase inside a string literal
// on a line that also carries code — `log("pinSame(total, c.offer)")` — leaves
// that line live while the copy itself is blanked, and the note would call it
// the surviving assertion and name it as the line to open. What survives there
// is the call around the literal, not the string the report is about. The same
// shape covers a comment trailing real code, and the blind spot this arm exists
// for: a runaway string blanks the rest of ITS OWN line, leaving the code
// before it standing.
//
// So the copy is placed by its own bytes, and its line is asked as a second
// question rather than as a proxy for the first:
//
//	blanked, on a line with nothing left    a comment of its own — prose
//	blanked, on a line that kept code       a literal, a trailing comment, or
//	                                        the runaway — and what stands on
//	                                        that line is not this string
//	not blanked                             code the lexer kept, which is the
//	                                        line to open
//
// # And the middle row was three answers in one sentence
//
// It said "a literal, a comment trailing an assertion, or the runaway that
// blanks the rest of its own line" and left the reader to work out which. Those
// send a reader to three different places, and the lexer had already decided:
// it has a case per construct and now records which one blanked each byte. See
// pinBlankKind. The row is read off the copy's own span, the same way the row
// itself is, so the construct named and the copy placed cannot be about
// different bytes.
//
//	code     what pinCodeOnly left, whole and unsplit
//	by       pinCodeOnly's third answer, one byte per source byte
//	copies   the surviving spellings, from pinPhraseCopies over the raw file
//
// A copy whose span is not inside that string is dropped rather than guessed
// at: the two are the same bytes by construction — one is the other with prose
// spaced out — and a disagreement is a bug in one of them and not a fact about
// a copy.
//
// Returns a sentence beginning with a space, to be appended to a verdict.
func pinCopyNote(code string, by []byte, copies []pinCopy) string {
	prose, live := []int{}, []int{}
	// The shadowed ones carry their constructs with them, because the sentence
	// that names them has to name them too.
	shadowed := []pinShadow{}
	// And the ones that are in a construct AND in live code, which is a copy
	// half deleted. Kept apart from both buckets: it is not the line to open —
	// what stands there is one end of the string — and it is not a copy the
	// lexer took whole either. See pinBlankedAs.
	straddling := []pinShadow{}
	for _, c := range copies {
		if len(c.At) == 0 || c.Start < 0 || c.End > len(code) || c.Start >= c.End {
			continue
		}
		// The line the copy starts on, which is the line the report already
		// names it at. A copy broken across lines is one entry either way —
		// see pinPhraseCopies — and its span carries the wrap.
		at := c.At[0]
		// What the lexer was in for each of this copy's own bytes. Placed by
		// the record rather than by the span test this used to make, which
		// could only ask whether ANYTHING in the span survived and therefore
		// read a half-deleted copy as a live one.
		kinds := pinBlankedAs(code, by, c)
		kept := false
		for _, kind := range kinds {
			if kind == pinKeptCode {
				kept = true
			}
		}
		switch {
		case len(kinds) == 1 && kept:
			live = append(live, at)
		case kept:
			straddling = append(straddling, pinShadow{lines: c.At, kinds: kinds})
		case len(kinds) > 0 &&
			strings.TrimSpace(pinLinesAround(code, c.Start, c.End)) != "":
			shadowed = append(shadowed, pinShadow{lines: c.At, kinds: kinds})
		// No record to read: a caller with no lexer behind it. The line is all
		// there is then, and it is exactly what this reading replaced — kept
		// so that arm answers as it did rather than dropping the copy.
		case len(kinds) == 0 && strings.TrimSpace(code[c.Start:c.End]) != "":
			live = append(live, at)
		case strings.TrimSpace(pinLinesAround(code, c.Start, c.End)) == "":
			prose = append(prose, at)
		default:
			shadowed = append(shadowed, pinShadow{lines: c.At, kinds: kinds})
		}
	}
	clauses := []string{}
	if len(prose) > 0 {
		clauses = append(clauses,
			fmt.Sprintf("blanked the copy on %v whole", prose))
	}
	if len(shadowed) > 0 {
		clauses = append(clauses, pinShadowClause(shadowed))
	}
	if len(straddling) > 0 {
		clauses = append(clauses, pinStraddleClause(straddling))
	}
	if len(live) > 0 {
		clauses = append(clauses,
			fmt.Sprintf("left the copy on %v standing in code", live))
	}
	if len(clauses) == 0 {
		return ""
	}
	note := " pinCodeOnly " + strings.Join(clauses, ", ") + ", so "
	switch {
	case len(live) > 0:
		return note + fmt.Sprintf("%v is the line to open.", live)
	case len(straddling) > 0:
		return note + fmt.Sprintf("no copy of this string survives whole: part of "+
			"the copy on %v is code the lexer kept and the rest of it went to %s, "+
			"so that line reads live for a fragment of this string and the "+
			"assertion is gone either way.",
			pinShadowLines(straddling), pinStraddleConstructs(straddling))
	case len(shadowed) > 0:
		return note + fmt.Sprintf("no copy of this string is code the lexer kept: "+
			"what stands on %v is the code AROUND %s, and not the string itself.",
			pinShadowLines(shadowed), pinShadowConstructs(shadowed))
	default:
		return note + "every surviving copy is prose and nothing here is an " +
			"assertion the lexer misread."
	}
}

// pinShadow is one copy the lexer blanked on a line it left code on, and what
// it was lexing when it did.
//
// Every construct the copy's own span met, in the order it met them, because a
// copy can be in more than one — see pinBlankedAs. Nil when there was no record
// to read, which pinConstructNames says in words rather than leaving the noun
// out of the sentence.
type pinShadow struct {
	// Every line the copy occupies, not the first of them. A copy inside one
	// construct is on one line and the two readings agree; a straddle is the
	// case where they cannot, because the halves are on different lines and
	// naming the first would send a reader to the half that is standing.
	lines []int
	kinds []byte
}

// pinBlankedAs is every construct a copy's own bytes were taken by, in the
// order the span meets them.
//
// Read off the span rather than off the line, for the reason the placement
// itself is: a line can carry a literal and a trailing comment at once, and a
// copy is inside one of them.
//
// # And a copy can be inside two
//
// This returned the FIRST recorded byte and called that the construct, on the
// argument that a phrase straddling two would have had to be blanked by both
// and the clause would say so. It cannot: one byte per copy is one construct
// per copy, and the multi-construct clause was about two COPIES in two
// constructs. A straddle arrived as whichever construct the phrase started in.
//
// Straddles are not hypothetical, and the rule for when one is buildable is
// not the one this note first gave. What has to be true is that the bytes
// BETWEEN the two halves survive the strip or are carried by the citation
// itself — see TestWhichConstructTransitionsAStraddleCanBeBuiltFrom, which
// walks the grid rather than arguing about it.
//
// Two transitions need no byte at all, because a line comment and a runaway
// string both end AT the newline and the newline is whitespace:
//
//	x = 1 // pinSame(total,      the head is a line comment's
//	c.offer)                     and the tail is code the lexer kept
//
// Every other transition puts raw bytes at the boundary — `*/`, a quote, `//`,
// `/*` — and those are reachable too, whenever the phrase is spelled with them
// in it, which is the ordinary shape of a citation that quotes a call:
//
//	log(pinSame("x"))            code the lexer kept, then a literal, and no
//	                             newline anywhere in it
//
// The census finds every one of the fourteen ordered pairs it asks about
// reachable, and finds each raw-byte boundary unreachable the moment the phrase
// is spelled without the delimiter. So this reads the whole span rather than
// the transition shapes somebody could think of.
//
// That copy is half gone. Answering "a comment running to the end of the line"
// sends a reader to a line where the phrase is not, and answering "live code"
// — which is what the span test above did, since something in the span
// survived — sends them to a line that holds half of it and calls it the
// assertion.
//
// # What is skipped, and why the answer is not always "code"
//
// pinCodeOnly blanks in place and leaves newlines alone, so a raw newline and
// a raw space between tokens both read as kept code. A copy broken across
// lines carries at least one of them, and counting those would make every
// multi-line copy a straddle. A byte is therefore only read as kept code when
// something is actually standing there: no recorded construct, and not
// whitespace in the lexed output.
//
// Nil when there is no record to read (a `by` of nil), which is a caller with
// no lexer behind it rather than a copy with no construct — pinCopyNote places
// those by their line, which is what this whole reading replaced and is still
// the only thing available without a record.
func pinBlankedAs(code string, by []byte, c pinCopy) []byte {
	if by == nil {
		return nil
	}
	var kinds []byte
	seen := map[byte]bool{}
	for i := c.Start; i < c.End && i < len(code) && i < len(by); i++ {
		kind := by[i]
		if kind == pinKeptCode && pinSpaceByte(code[i]) {
			continue
		}
		if seen[kind] {
			continue
		}
		seen[kind] = true
		kinds = append(kinds, kind)
	}
	return kinds
}

// pinSpaceByte reports whether a byte of the lexed source is whitespace.
//
// The bytes pinCodeOnly writes when it blanks a construct are spaces too, and
// the caller tells the two apart by the record and not by this: a space with a
// construct recorded against it is prose, and a space with nothing recorded is
// the source's own layout. Written over bytes rather than runes because both
// readers index the same offsets the lexer wrote at, and every whitespace
// character either language separates tokens with is ASCII.
func pinSpaceByte(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r' || b == '\f' || b == '\v'
}

// pinShadowClause is the middle clause of the note: which copies were blanked,
// and by what.
//
// Grouped by construct so two copies a literal swallowed are one clause and a
// literal and a block comment are two. Ordered by first appearance, which is
// the order the copies came in, so the sentence reads in the file's own order
// rather than in a map's.
//
// Grouped by the whole run of constructs and not by one of them, because a copy
// carries every construct its span met — see pinBlankedAs. Two copies a literal
// swallowed still group; a copy a literal swallowed and a copy split between a
// literal and a comment do not, and reading as one clause is exactly the
// summary this was rebuilt out of.
func pinShadowClause(shadowed []pinShadow) string {
	parts := pinShadowParts(shadowed, "inside", "across")
	if len(parts) == 1 {
		return "blanked " + parts[0] + " on a line it left code on"
	}
	return "blanked " + strings.Join(parts, ", and ") +
		", each on a line it left code on"
}

// pinStraddleClause is the same for the copies that are in a construct and in
// live code at once.
//
// Said as a split rather than as a blanking, because neither word is true of
// the whole copy: half of it is gone and half of it is standing on a line that
// reads live. "Blanked" would be the note claiming the copy is prose and
// "left standing" would be it naming a line to open.
func pinStraddleClause(straddling []pinShadow) string {
	parts := pinShadowParts(straddling, "across", "across")
	return "split " + strings.Join(parts, ", and ")
}

// pinShadowParts is one phrase per group of copies that met the same run of
// constructs, in the order the copies came in.
func pinShadowParts(shadowed []pinShadow, one, many string) []string {
	byKinds := map[string][]int{}
	shape := map[string][]byte{}
	order := []string{}
	for _, s := range shadowed {
		key := string(s.kinds)
		if _, seen := byKinds[key]; !seen {
			order = append(order, key)
			shape[key] = s.kinds
		}
		byKinds[key] = append(byKinds[key], s.lines...)
	}
	parts := make([]string, 0, len(order))
	for _, key := range order {
		// "inside" is true of a copy one construct swallowed and is a claim
		// about a copy that met several: it was not in them, it crossed them.
		how := one
		if len(shape[key]) > 1 {
			how = many
		}
		parts = append(parts, fmt.Sprintf("the copy on %v %s %s",
			byKinds[key], how, pinConstructNames(shape[key])))
	}
	return parts
}

// pinConstructNames is a copy's own run of constructs, spelled in span order.
//
// Joined with "then" rather than "and": these are not two things the copy is
// inside, they are what the lexer was in as it walked from the copy's first
// byte to its last, and the order is where the reader has to look first.
func pinConstructNames(kinds []byte) string {
	if len(kinds) == 0 {
		// Not reachable from a lexed span — see pinBlankedAs — and named
		// rather than omitted, because a sentence that quietly drops its noun
		// is worse than one that says the noun is missing.
		return "a construct pinCodeOnly did not record"
	}
	names := make([]string, 0, len(kinds))
	for _, kind := range kinds {
		names = append(names, pinConstructName(kind))
	}
	if len(names) == 1 {
		return names[0]
	}
	return strings.Join(names[:len(names)-1], ", ") + " then " + names[len(names)-1]
}

// pinShadowLines is every line a shadowed copy starts on, in order.
func pinShadowLines(shadowed []pinShadow) []int {
	out := make([]int, 0, len(shadowed))
	for _, s := range shadowed {
		out = append(out, s.lines...)
	}
	return out
}

// pinShadowConstructs names what stands around those copies, once each.
//
// The old sentence listed all three constructs every time and let the reader
// sort it out. This lists the ones the lexer actually met — usually one, and
// when it is more than one that is a finding rather than a hedge.
//
// Over every construct of every copy, because one copy can carry several.
func pinShadowConstructs(shadowed []pinShadow) string {
	return pinNameSet(shadowed, false)
}

// pinStraddleConstructs is the same over the straddling copies, with the code
// half left out.
//
// The sentence it lands in has already said that half is standing; naming it
// again in the list of what TOOK the string would have the note reporting live
// code as a construct that swallowed something.
func pinStraddleConstructs(straddling []pinShadow) string {
	return pinNameSet(straddling, true)
}

// pinNameSet is every construct these copies met, once each, in the order they
// were met — optionally without the one that is not a construct at all.
func pinNameSet(shadowed []pinShadow, skipKept bool) string {
	seen := map[byte]bool{}
	names := []string{}
	for _, s := range shadowed {
		for _, kind := range s.kinds {
			if seen[kind] || (skipKept && kind == pinKeptCode) {
				continue
			}
			seen[kind] = true
			names = append(names, pinConstructName(kind))
		}
	}
	if len(names) == 0 {
		return "a construct pinCodeOnly did not record"
	}
	if len(names) == 1 {
		return names[0]
	}
	return strings.Join(names[:len(names)-1], ", ") + " and " + names[len(names)-1]
}

// pinLinesAround is every line a span touches, whole.
//
// The span says what happened to the copy; this says what happened to the rest
// of the line it sits on, which is the difference between a comment of its own
// and a string literal with code around it. Both readings come off the same
// lexed source, so neither can be about a file the other is not.
func pinLinesAround(s string, start, end int) string {
	from := strings.LastIndexByte(s[:start], '\n') + 1
	to := len(s)
	if i := strings.IndexByte(s[end:], '\n'); i >= 0 {
		to = end + i
	}
	return s[from:to]
}

// pinCopy is one surviving spelling of a phrase in a raw file: the lines it
// occupies, and the bytes.
//
// The two answer different questions and the second was missing. `At` is where
// to send a reader. `Start`/`End` is what pinCodeOnly did to this copy — the
// lexer blanks spans in place, so the same range in its output is either
// whitespace, which is a copy it took for prose, or not, which is a copy
// standing in code. A line cannot be asked that: a phrase inside a string
// literal on a line that also carries code is on a line the lexer left live
// and is itself gone. See pinCopyNote.
type pinCopy struct {
	At         []int
	Start, End int
}

// pinPhraseCopies is each spelling of the phrase in a raw file — one entry per
// surviving copy — or nil when the phrase is not in the file at all.
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
func pinPhraseCopies(raw, phrase string) []pinCopy {
	stripped, lines, offs := pinStripSpaceMap(raw)
	var out []pinCopy
	// Non-overlapping, left to right, which is what FindAll gives: two matches
	// that overlapped would be one spelling counted twice.
	for _, loc := range pinSpellsPattern(phrase).FindAllStringIndex(stripped, -1) {
		c := pinCopy{Start: -1, End: -1}
		for i := loc[0]; i < loc[1] && i < len(lines); i++ {
			if len(c.At) == 0 || c.At[len(c.At)-1] != lines[i] {
				c.At = append(c.At, lines[i])
			}
			// Both walks are left to right over one index, so the first byte
			// of the match is the copy's first raw byte and the last is its
			// last. Written as a widening rather than as two lookups because
			// the loop is already here and a truncated match — the guard above
			// — would leave the second wrong.
			if c.Start < 0 || offs[i] < c.Start {
				c.Start = offs[i]
			}
			if offs[i]+1 > c.End {
				c.End = offs[i] + 1
			}
		}
		out = append(out, c)
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

// Every arm of pinCopyNote, reached without a file that has the fault.
//
// The three arms fire on a run where a cited phrase has been deleted from a
// harness and survives somewhere in the file, on a line the lexer lost its
// place on — a state neither consumer has ever been in, and one that cannot be
// contrived by editing this test. As a function of two values every arm is a
// call, which is the move bandtarget.mjs and startupVerdict were split out for.
//
// `code` is what pinCodeOnly leaves: prose replaced by spaces, in place, so the
// Every arm of the note, over copies a lexer actually produced.
//
// The fixture is three spellings of one phrase and one pass of pinCodeOnly
// over them, because the distinction this note was rebuilt for is a lexer's
// and not a fixture author's: the copy on line 3 sits inside a string literal
// on a line that also carries code, so the LINE comes back live and the COPY
// comes back blanked. Reading lines, as this did, called that copy the
// surviving assertion and sent a reader to open a line where what stands is
// the call around the literal.
//
// The two arms nothing in a real file reaches — a span outside the lexed
// source, and no placeable copy at all — are handed in as values, which is the
// move pinCopyNote was extracted for.
func TestPinCopyNoteNamesTheCopyThatSurvivesInCode(t *testing.T) {
	const src = "// pinSame(total, c.offer) in prose\n" +
		"assert(pinSame(total, c.offer))\n" +
		"log(\"pinSame(total, c.offer)\")\n" +
		"x = 1; /* pinSame(total, c.offer) */\n"
	code, open, by := pinCodeOnly(src, ".go")
	if len(open) > 0 {
		t.Fatalf("pinCodeOnly ended inside an unterminated string on %v of this "+
			"fixture's lines, so what it left is not what the arms below are about.",
			open)
	}
	all := pinPhraseCopies(src, "pinSame(total, c.offer)")
	if len(all) != 4 {
		t.Fatalf("the fixture spells the phrase 4 times and pinPhraseCopies finds "+
			"%d, so the rows below are about some other file.", len(all))
	}
	// By the line each copy starts on, so a row names the case rather than an
	// index into a slice.
	on := func(lines ...int) []pinCopy {
		out := []pinCopy{}
		for _, copy := range all {
			for _, line := range lines {
				if copy.At[0] == line {
					out = append(out, copy)
				}
			}
		}
		return out
	}
	for _, c := range []struct {
		what   string
		copies []pinCopy
		says   []string
		quiet  []string
	}{
		{
			what:   "prose and live code names both and says which to open",
			copies: on(1, 2),
			says: []string{"blanked the copy on [1] whole",
				"left the copy on [2] standing in code", "[2] is the line to open"},
		},
		{
			what:   "a copy inside a literal is not the line to open",
			copies: on(1, 3),
			says: []string{"blanked the copy on [1] whole",
				"blanked the copy on [3] inside a string literal on a line it " +
					"left code on",
				"no copy of this string is code the lexer kept",
				"the code AROUND a string literal"},
			quiet: []string{"line to open", "every surviving copy is prose",
				// The three-constructs-in-one-sentence answer this arm was
				// rebuilt out of. The lexer knew which it was; naming all of
				// them is the count-shaped answer one level down.
				"block comment", "never saw closed"},
		},
		{
			what:   "a literal and live code still send the reader to the code",
			copies: on(2, 3),
			says: []string{
				"blanked the copy on [3] inside a string literal on a line it " +
					"left code on",
				"[2] is the line to open"},
			quiet: []string{"whole"},
		},
		{
			// Two constructs at once, which the old sentence could not tell
			// apart from one: a reader looking at line 3 is looking for a
			// quote and a reader looking at line 4 is looking for a comment.
			what:   "two constructs are two answers, not a longer list",
			copies: on(3, 4),
			says: []string{
				"blanked the copy on [3] inside a string literal, and the copy on " +
					"[4] inside a block comment, each on a line it left code on",
				"the code AROUND a string literal and a block comment"},
			quiet: []string{"line to open", "whole"},
		},
		{
			what:   "every copy prose is nothing to open",
			copies: on(1),
			says: []string{"blanked the copy on [1] whole",
				"every surviving copy is prose"},
			quiet: []string{"line to open", "left code on"},
		},
		{
			// The report reaches this arm with no copies when
			// pinPhraseCopies could place none — the default verdict
			// already says so, and a second sentence guessing at lines
			// would be the note contradicting it.
			what:   "no placeable copy says nothing at all",
			copies: nil,
			quiet:  []string{"pinCodeOnly"},
		},
		{
			// A span the lexed source does not have is the two readings
			// having come apart, which is not a fact about a copy. Dropped
			// rather than reported, and the copies that CAN be placed still
			// get their sentence.
			what: "a span outside the file is dropped and the rest still answer",
			copies: append(on(2),
				pinCopy{At: []int{99}, Start: len(code) + 10, End: len(code) + 30}),
			says:  []string{"left the copy on [2] standing in code"},
			quiet: []string{"99"},
		},
	} {
		got := pinCopyNote(code, by, c.copies)
		for _, want := range c.says {
			if !strings.Contains(got, want) {
				t.Errorf("%s: pinCopyNote does not say %q.\n\nIt said: %q\n\n"+
					"The count of surviving copies is what this replaces, and it "+
					"replaces it by naming which line a reader has to open. A note "+
					"missing that is the count again with more words around it.",
					c.what, want, got)
			}
		}
		for _, never := range c.quiet {
			if strings.Contains(got, never) {
				t.Errorf("%s: pinCopyNote says %q and must not.\n\nIt said: %q\n\n"+
					"Each arm sends the reader somewhere different, and a sentence "+
					"carrying another arm's phrase sends them to two places at once.",
					c.what, never, got)
			}
		}
	}

	// And the third construct, which the fixture above cannot produce.
	//
	// A JavaScript regular-expression literal carrying an odd number of quotes
	// is the one thing pinCodeOnly declines to lex — see its own note — and the
	// damage it does is shaped exactly like a string literal's from the outside:
	// the copy is blanked, the line keeps code. What differs is what a reader
	// finds there. In a literal the code around it is a real call; here it is
	// the half of an assertion that happened to sit before the quote, and the
	// rest of the line is gone. Those are two different afternoons.
	const runaway = "x = /\"/; assert(pinSame(total, c.offer))\n"
	blanked, ran, ranBy := pinCodeOnly(runaway, ".mjs")
	if len(ran) != 1 {
		t.Fatalf("the runaway fixture is one line whose regex opens a string nothing "+
			"closes, and pinCodeOnly reports unterminated strings on %v. Without one "+
			"there is no runaway here and the row below is about an ordinary "+
			"literal.", ran)
	}
	loose := pinPhraseCopies(runaway, "pinSame(total, c.offer)")
	if len(loose) != 1 {
		t.Fatalf("the runaway fixture spells the phrase once and pinPhraseCopies "+
			"finds %d.", len(loose))
	}
	got := pinCopyNote(blanked, ranBy, loose)
	for _, want := range []string{
		"blanked the copy on [1] inside a string the lexer never saw closed",
		"the code AROUND a string the lexer never saw closed",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("the runaway arm does not say %q.\n\nIt said: %q\n\n"+
				"pinCodeOnly knows this was the construct it declines to guess at — "+
				"it records the whole literal again once the newline closes it — and "+
				"a note that calls it a string literal sends the reader looking for a "+
				"quote somebody wrote on purpose.", want, got)
		}
	}
	if strings.Contains(got, "line to open") {
		t.Errorf("the runaway arm offers a line to open: %q. What stands on that "+
			"line is the code before the quote, which is not this string.", got)
	}
}

// A copy that is in more than one construct, which every reading above places
// by the first thing it met.
//
// # The shape, and why it is not contrived
//
// A phrase is matched over the space-stripped source and a newline is
// whitespace — see pinStripSpaceMap. Two transitions therefore need no byte
// between them at all: a line comment and a single-line string both end AT the
// newline. So a phrase can begin inside one and finish in whatever the next
// line starts with, and what it finishes in is a different construct from what
// it started in.
//
// The rest need a delimiter in the raw file, and a delimiter the citation
// carries does not break the match — `log(pinSame("x"))` is a copy half in
// code and half in a literal with no newline in it at all. Which transitions
// are buildable and which are not is measured rather than reasoned about in
// TestWhichConstructTransitionsAStraddleCanBeBuiltFrom; the rows below are the
// SENTENCE, over heads that are not all the same construct.
//
//	x = 1 // pinSame(total,      the head is a comment's
//	c.offer) + 2                 and the tail is code the lexer kept
//
// That copy is half deleted. The span test this note was built on asks whether
// anything in the span survived, and something did — so it called this the
// surviving assertion and named line 1 as the line to open, where the phrase
// is not. Reading the record and stopping at the first kind is the same answer
// from the other side: "a comment running to the end of the line", about a
// copy half of which is standing in code.
//
// # And three constructs is the same fault with more of it
//
// The second fixture crosses a comment, a string literal the author closed, and
// live code — one copy, three places to look, and the clause has to name them
// in the order the reader meets them.
//
// # And the head is not always a line comment
//
// The first two rows both open with `//`, which is the transition the census
// finds cheapest to write and is not the only one the machinery handles. The
// third opens with a string the lexer never saw closed — the blind spot the
// runaway kind exists for, whose own line still reads live — and the fourth
// opens inside a block comment and ends inside a literal, with no newline
// anywhere in it. Three of the four constructs lead a fixture, and the fourth
// (kept code) leads one in the census.
func TestPinCopyNotePlacesACopyThatStraddlesTwoConstructs(t *testing.T) {
	for _, c := range []struct {
		what, src, ext, phrase string
		// The lines pinCodeOnly ended inside a single-line string on, which is
		// nil for every fixture whose quotes close. Declared per row rather
		// than refused for all of them: the runaway IS one of the constructs a
		// copy can straddle out of, and a fixture that produces it has to be
		// able to say so without the guard below reading it as the lexer
		// having lost its place.
		open  []int
		says  []string
		quiet []string
	}{
		{
			what:   "a comment and the code on the next line",
			src:    "x = 1 // pinSame(total,\nc.offer) + 2\n",
			ext:    ".go",
			phrase: "pinSame(total, c.offer)",
			says: []string{
				"split the copy on [1 2] across a comment running to the end of " +
					"the line then code the lexer kept",
				"no copy of this string survives whole",
				"part of the copy on [1 2] is code the lexer kept",
				"the rest of it went to a comment running to the end of the line",
			},
			// Not the line to open: what stands on line 2 is the tail of a
			// phrase whose head a comment took, and sending a reader there is
			// the failure this whole note exists to stop, one level in.
			quiet: []string{"line to open", "whole,", "inside a comment"},
		},
		{
			what:   "a comment, a literal and the code after it",
			src:    "x = 1 // pinSame(\n\"x\") + y\n",
			ext:    ".mjs",
			phrase: "pinSame(\"x\")",
			says: []string{
				"across a comment running to the end of the line, a string " +
					"literal then code the lexer kept",
				"the rest of it went to a comment running to the end of the line " +
					"and a string literal",
			},
			quiet: []string{"line to open"},
		},
		{
			// A head that is not a comment. The runaway is the construct the
			// deletion report lies about — the code BEFORE the quote still
			// stands, so the line reads live while the assertion after it is
			// gone — and it is also one of the two transitions that need no
			// delimiter, because the newline closes it. Both halves of that
			// are in this one fixture.
			what:   "a string the lexer never saw closed, and the code after it",
			src:    "x = 1 + \"pinSame(total,\nc.offer) + 2\n",
			ext:    ".go",
			phrase: "pinSame(total, c.offer)",
			open:   []int{1},
			says: []string{
				"split the copy on [1 2] across a string the lexer never saw " +
					"closed, which blanks the rest of its own line then code the " +
					"lexer kept",
				"no copy of this string survives whole",
			},
			quiet: []string{"line to open", "whole,", "inside a"},
		},
		{
			// And a straddle with no newline in it at all, which the note above
			// this test used to say was unreachable. The boundary is `*/` and
			// the phrase carries it, which is the whole of what the census
			// finds separates a buildable transition from an unbuildable one.
			what:   "a block comment, the code after it and a literal",
			src:    "/* pinSame( */ x + \"y\")\n",
			ext:    ".go",
			phrase: "pinSame( */ x + \"y\")",
			says: []string{
				"across a block comment, code the lexer kept then a string literal",
				"the rest of it went to a block comment and a string literal",
			},
			quiet: []string{"line to open"},
		},
	} {
		code, open, by := pinCodeOnly(c.src, c.ext)
		if !slices.Equal(open, c.open) {
			t.Fatalf("%s: pinCodeOnly ended inside an unterminated string on %v of "+
				"this fixture's lines and the row says %v, so what it left is not "+
				"what the rows below are about.", c.what, open, c.open)
		}
		copies := pinPhraseCopies(c.src, c.phrase)
		if len(copies) != 1 {
			t.Fatalf("%s: the fixture spells %q once and pinPhraseCopies finds %d. "+
				"The straddle is a property of one copy crossing a newline, and "+
				"without exactly one match this is about some other string.",
				c.what, c.phrase, len(copies))
		}
		// The record itself, before the sentence: a copy whose bytes are all
		// one kind cannot straddle anything, and a test asserting the sentence
		// alone would pass on a fixture that never produced the shape.
		kinds := pinBlankedAs(code, by, copies[0])
		if len(kinds) < 2 {
			t.Fatalf("%s: the copy's own bytes were taken by %d construct(s) (%q), "+
				"so this fixture is not the straddle it is here to be. Both halves "+
				"have to be in the span — see pinBlankedAs.",
				c.what, len(kinds), string(kinds))
		}
		got := pinCopyNote(code, by, copies)
		for _, want := range c.says {
			if !strings.Contains(got, want) {
				t.Errorf("%s: pinCopyNote does not say %q.\n\nIt said: %q\n\n"+
					"A copy that crosses constructs is in each of them for part of "+
					"its length, and the reader has to be sent to all of them in the "+
					"order the phrase meets them. Naming the first is the "+
					"count-shaped answer this note was rebuilt out of, arriving one "+
					"copy further in.", c.what, want, got)
			}
		}
		for _, never := range c.quiet {
			if strings.Contains(got, never) {
				t.Errorf("%s: pinCopyNote says %q and must not.\n\nIt said: %q\n\n"+
					"Half of this copy is standing and half of it is gone, so it is "+
					"neither a line to open nor a copy the lexer took whole. Either "+
					"sentence sends a reader somewhere the string is not.",
					c.what, never, got)
			}
		}
	}
	// And the other half of the reading: a copy broken across lines INSIDE one
	// construct is not a straddle.
	//
	// pinCodeOnly blanks in place and leaves newlines alone, so the newline a
	// two-line copy carries comes back as a byte nothing recorded a construct
	// against — which is indistinguishable, byte for byte, from code the lexer
	// kept. Counting it would make every multi-line copy a straddle between its
	// own construct and a newline, and the note would report a block comment
	// that swallowed a phrase whole as a phrase half standing in code. See
	// pinBlankedAs, which skips a byte only when nothing was recorded against
	// it AND nothing is standing there.
	const wrapped = "/* pinSame(total,\nc.offer) */\n"
	code, _, by := pinCodeOnly(wrapped, ".go")
	copies := pinPhraseCopies(wrapped, "pinSame(total, c.offer)")
	if len(copies) != 1 {
		t.Fatalf("the wrapped fixture spells the phrase once across two lines and "+
			"pinPhraseCopies finds %d.", len(copies))
	}
	if kinds := pinBlankedAs(code, by, copies[0]); len(kinds) != 1 {
		t.Errorf("a copy a block comment swallowed across two lines was taken by %q "+
			"— %d constructs.\n\n"+
			"One comment took every character of it. The extra kind is the newline "+
			"between the two halves, which pinCodeOnly leaves alone and which "+
			"therefore reads exactly like a byte of live code. A copy that wraps is "+
			"the ordinary case and reporting it as half deleted would bury the "+
			"straddle above in false ones.", string(kinds), len(kinds))
	}
	// And placed as prose, at the line the report names it at: nothing stands
	// on either of its lines, so there is no assertion here for the lexer to
	// have misread and no second line to send anybody to. The straddles above
	// name both of theirs for the opposite reason — there the two halves are in
	// different places and which one a reader opens decides what they find.
	if note := pinCopyNote(code, by, copies); !strings.Contains(note,
		"blanked the copy on [1] whole") ||
		!strings.Contains(note, "every surviving copy is prose") {
		t.Errorf("the wrapped copy is not reported as prose: %q\n\n"+
			"Nothing stands on either of its lines, so there is no assertion here "+
			"for the lexer to have misread.", note)
	}
}
