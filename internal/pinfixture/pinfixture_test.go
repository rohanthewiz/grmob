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
		rowMain int
		agrees  bool
	}{
		// The control. Nothing is pinned, and the divergence from CSS
		// (24/80/16) is already here: Compose gives the first child everything
		// it asked for and the last one nothing.
		{"no pin: every child shrinks", []int{120, 60, 0}, []int{60, 60, 0}, 120, false},
		// The pin first. `fixedSpace` passes the Row's maximum on the first
		// child, so both siblings are offered 0 — which is the same three
		// numbers CSS produces, by a different route.
		{"the pinned child first", []int{120, 0, 0}, []int{200, 0, 0}, 200, true},
		// The pin in the middle: offered 60 and ignoring it.
		{"the pinned child between its siblings", []int{120, 60, 0}, []int{60, 200, 0}, 260, false},
		// The pin last: offered 20 and ignoring it. The most direct statement
		// of "remaining is ignored", since the number ignored is neither the
		// whole offer nor nothing.
		{"the pinned child last", []int{120, 60, 20}, []int{60, 40, 200}, 300, false},
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
		if c.Compose.RowMain != w.rowMain {
			t.Errorf("%s: the Row reports %d, and the table says %d", c.What,
				c.Compose.RowMain, w.rowMain)
		}
		if c.MainsAgreeWithCSS != w.agrees {
			t.Errorf("%s: MainsAgreeWithCSS is %v and the table says %v", c.What,
				c.MainsAgreeWithCSS, w.agrees)
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
	for _, c := range Cases() {
		for i, child := range c.Children {
			if !child.Pinned {
				continue
			}
			positions++
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
	if positions != 3 {
		t.Errorf("%d pinned children across the fixture, and the point of it is one in "+
			"each of the three positions", positions)
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
			taken += c.Compose.Mains[i]
		}
	}

	// And the order-dependence itself, which is what the CSS side is compared
	// against. The three pinned rows hold the same three children; if their
	// answers ever coincide, either the transcription has stopped depending on
	// order or the fixture has stopped being an overflow.
	seen := map[string]string{}
	for _, c := range Cases()[1:] {
		key := sizesByName(c)
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
		content := 0
		for _, m := range c.Compose.Mains {
			content += m
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
	if overflowed != 3 {
		t.Errorf("%d of the four rows overflow their container, and the three pinned "+
			"ones are supposed to: a pinned child whose base exceeds the Row's extent "+
			"cannot fit inside it by construction", overflowed)
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
				{Base: 60}, {Base: 200, Pinned: true}, {Base: 40}}},
		},
		{
			name:     "a Row with nobody on either side of the pin",
			mentions: "fewer than three",
			bad: Case{What: "lonely", Offer: 120, Children: []Child{
				{Base: 60}, {Base: 200, Pinned: true}}},
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
		"no pin: every child shrinks":           "| no pin |",
		"the pinned child first":                "| pin first `[P,A,B]` |",
		"the pinned child between its siblings": "| pin middle `[A,P,B]` |",
		"the pinned child last":                 "| pin last `[A,B,P]` |",
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

		// Only the Compose column is held here. The CSS column is
		// GrMobFlexSolver's answer and this package does not compute it — that
		// is ios/verify/pin.swift's half, and transcribing it into Go to check
		// the doc would be writing the number a third time.
		compose := composeCell(row)
		if got := joinInts(c.Compose.Mains); compose != got {
			t.Errorf("%s: the census's Compose column reads %q and the fixture measures "+
				"%q.\n\nThe table is the whole content of that section and nothing "+
				"compiles against it, so a number that drifts there is a paragraph "+
				"describing a layout no target produces.", c.What, compose, got)
		}
	}
}

// composeCell pulls the last cell out of a markdown table row and strips the
// bold markers the census uses to point at the pinned child.
//
// Deliberately positional — the census's table has exactly three columns and the
// Compose answer is the last — because a general markdown parser is not what this
// needs and would be far more code than the thing it checks. A table that grew a
// column would land here as a mismatch that names the row, which is the failure
// a reader can act on.
func composeCell(row string) string {
	cells := strings.Split(strings.Trim(row, "|"), "|")
	if len(cells) == 0 {
		return ""
	}
	return strings.TrimSpace(strings.ReplaceAll(cells[len(cells)-1], "**", ""))
}

// joinInts spells a case's extents the way the census's cells do.
func joinInts(v []int) string {
	parts := make([]string, len(v))
	for i, n := range v {
		parts[i] = itoa(n)
	}
	return strings.Join(parts, ", ")
}
