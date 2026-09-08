package pinfixture

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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
	for _, c := range Cases() {
		nums := append([]int{c.Offer, c.Gap, c.Compose.RowMain}, c.CSS...)
		nums = append(nums, c.Compose.Mains...)
		nums = append(nums, c.Compose.Gaps...)
		nums = append(nums, c.Compose.Offered...)
		for _, ch := range c.Children {
			nums = append(nums, ch.Base)
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
