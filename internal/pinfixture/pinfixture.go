// Package pinfixture states one overflowing Row six ways, so that what
// core.FlexShrink(0) does on Compose stops being a paragraph and becomes a set
// of numbers something can disagree with.
//
// # What was missing
//
// Renderer.kt honours core.FlexShrink(0) with Modifier.pinMainAxis: the child
// is measured with an unbounded main axis and reports its own size, so it keeps
// its extent and the Row overflows. Everything holding that up was source-level
// — mobile/verify reads the call sites, the axis argument and the two lines
// inside the modifier, and `compileDebugKotlin` proves it parses. Nothing
// anywhere produced the arithmetic it causes.
//
// That arithmetic is androidx's, not ours, which is why it had stayed prose.
// foundation-layout's measure policy cannot be executed here: it measures
// Measurables into Placeables through compose-ui, whose Placeable has internal
// abstract members, so a plain JVM cannot even implement the interfaces — let
// alone without the Android runtime. android/verify's whole premise is Kotlin
// that imports nothing, and this imports the world.
//
// # What this is instead, and what holds it up
//
// A TRANSCRIPTION of the two decisions, in Go, executed. It is not androidx's
// code and does not pretend to be; what makes it worth more than the paragraph
// it replaces is the chain each line hangs from:
//
//	the transcription   every branch below names the source line it mirrors,
//	                    from foundation-layout's RowColumnMeasurementHelper.kt
//	                    and Size.kt.
//	those lines         mobile/verify's TestTheComposeCensusClaimsAreWhatTheSourceSays
//	                    reads them out of the sources jar. A release that
//	                    changes them fails there, by name, and sends the reader
//	                    here.
//	that version        is the BOM's, and there is no second spelling of it:
//	                    android/app/build.gradle asks for the sources jar with
//	                    no version at all and lets the platform resolve it, so
//	                    the source read is the source the app builds against.
//	the renderer's half TestTheComposeChildrenLoopsPinOnTheirOwnAxis and
//	                    TestTheComposePinMeasuresUnboundedAndReportsWhatItMeasured
//	                    hold Renderer.kt to the two lines PinnedChild below is
//	                    a transcription of.
//
// # The link that used to be weaker than the list above made it look
//
// "Every branch names the source line it mirrors" was true, and for a while
// only ONE of those lines was read out of the jar by anything —
// `mainAxisMax - fixedSpace`. The floor on it, the spacing clamp, the missing
// upper bound on the Row's own size and SizeNode's unclamped report were quoted
// in this comment and held to nothing, which is a chain with one link drawn in.
// They are rows in that claims table now, so a release that changes any of them
// fails by name and sends the reader to the branch it belongs to.
//
// What is still nobody's but the reader's: the transcription is Go somebody
// wrote from those lines. The lines are pinned; that they add up to this
// arithmetic is a reading. That is the last link, it is stated rather than
// hidden, and nothing short of running androidx's measure policy would close
// it — see above for why that cannot happen here.
//
// # The fixture, and what each case is for
//
// One Row, one set of children, six arrangements. The Row is 120 wide and the
// children want 60, 200 and 40, so the deficit (180) is larger than what the
// two shrinkable children have to give (100) — which is forced, not chosen: a
// pinned child whose base exceeds the container makes it arithmetically
// unavoidable, and a pinned child whose base does NOT exceed the container is
// not pinned against anything when it comes first.
//
//	                          CSS (Case.CSS)   Compose (MeasureCompose)
//	no pin                    24, 80, 16       60, 60,  0
//	pin first      [P,A,B]   200,  0,  0      200,  0,  0
//	pin middle     [A,P,B]     0,200,  0       60,200,  0
//	pin last       [A,B,P]     0,  0,200       60, 40,200
//	pin middle+8   [A,P,B]     0,200,  0       60,200,  0   gaps  8, 8 /  8,0
//	pin last+16    [A,B,P]     0,  0,200       60, 40,200   gaps 16,16 / 16,4
//
// BOTH columns are fields now, and that is newer than the rest of this. The CSS
// one used to be three numbers in this comment, beside two mechanisms that
// produce them and neither of which was ever compared with it — so the control
// row's 24/80/16 was the one row nothing derived from a stated claim, and a
// reader who trusted those numbers was trusting a paragraph. Nothing here
// computes them: a flex line transcribed into Go would be a third spelling
// after GrMobFlexSolver and a browser, and every comparison would become a
// question about whether two transcriptions agree. What changed is that the
// claim is now a value, so a solver and a browser can be held to it.
//
// Five claims come out of that table, and each needs a different row to be
// visible:
//
//	the declaration means the same thing   the pinned child is 200 in every
//	                                       pinned row, on both targets, wherever
//	                                       it sits. That is what core.ShrinkNone
//	                                       says, and it is now measured on the
//	                                       target that had no way to express it.
//
//	the pin is load-bearing                the control row. Same Row, same
//	                                       children, one factor apart: without
//	                                       it that child is 60 on Compose and 80
//	                                       on CSS.
//
//	the SIBLINGS diverge                   CSS's answer does not depend on the
//	                                       order and Compose's does. Compose
//	                                       gives each child what the ones before
//	                                       it left; CSS shares the deficit over
//	                                       every child that can shrink, so the
//	                                       same three children get the same three
//	                                       sizes in any order.
//
//	the SPACING diverges too               the two gapped rows are the
//	                                       pin-middle and pin-last rows again,
//	                                       and both columns of extents are those
//	                                       rows' unchanged — a gap is used space
//	                                       in a flex line, and the shrinkable
//	                                       children were clamped to zero
//	                                       already. What differs is underneath:
//	                                       spaceAfterLastNoWeight is
//	                                       min(spacing, what is left), where a
//	                                       flex line charges its gap between
//	                                       every adjacent pair regardless. Every
//	                                       case here carried gap 0 until those
//	                                       rows, so the line MeasureCompose
//	                                       implements had never been put in
//	                                       front of a browser or a solver.
//
//	the collapse is a MIN                  and a min has three answers, not two.
//	                                       The 8px row shows the ends: the gap
//	                                       in full after the lead child, nothing
//	                                       after the pin. The 16px row is the
//	                                       one that shows the middle — 4 charged
//	                                       where 16 was asked for, because that
//	                                       is what was left. Without it every
//	                                       gap in this fixture is either the
//	                                       spacing or zero, which is also what
//	                                       "charge the gap unless the Row has
//	                                       overflowed" would produce, and the
//	                                       two rules would be indistinguishable
//	                                       on every row here.
//
// The pin-first row is the one where they agree, and it is here for the reason
// bandfixture states an unbadged band: an "it diverges" with no case that does
// not is a claim about whatever happened. The agreement is also a coincidence
// of two different rules rather than a shared one — CSS clamps the shrinkable
// children to zero because the deficit exceeds their bases, Compose offers them
// nothing because the pinned child already took more than the Row had — and
// saying so is the point of writing both answers down.
//
// # What this does not carry
//
// A weighted child. The renderer never pins one (Modifier.weight fixes that
// child's main axis as both minimum and maximum, and CSS never applies grow and
// shrink at once either), so the weighted branch of androidx's loop is outside
// what the pin can reach and transcribing it would be modelling for its own
// sake. Child has no weight field at all, so the refusal is in the type rather
// than in a check somebody has to remember: adding one is a change to the
// fixture's shape, which is a change a reader sees.
//
// Padding. The fixture carries none, which is what keeps its CSS column
// comparable rather than hand-waved: GrMobFlexSolver shrinks each child in
// proportion to a base that INCLUDES the child's own padding, and CSS
// distributes shrink over the inner flex base size, which excludes it — a
// divergence ios/verify/band.swift records and wasm/verify's check 9 has since
// watched a real Chrome confirm. With no padding anywhere the two rules
// coincide.
//
// That used to be the end of the paragraph, and it was reasoning rather than a
// measurement: "the known difference does not apply here", said by whoever
// wrote the fixture and asked of nobody. wasm/verify's check 12 mounts these
// Rows in a browser now and holds them to what this file states — a pinned
// child keeps its base, the extents match the Compose column exactly where
// MainsAgreeWithCSS says they do and differ where it says they differ, a
// child's width does not depend on where it sits, and the spacing between two
// adjacent children is the Row's gap wherever the fixture says a flex line
// keeps it.
//
// And the CSS column itself, which is what closed the last of it. Those claims
// pin three of the four ungapped rows to their exact numbers; the control row's
// proportional split was determined by none of them and was three numbers in
// this comment. It is Case.CSS now — still stated rather than computed, for the
// reason that field gives — so GrMobFlexSolver and a browser are each held to
// the whole column, the control row included, and the census's table is a claim
// two executable things fail on rather than a paragraph beside them.
package pinfixture

import (
	"fmt"
	"sort"
)

// Child is one unweighted child of the Row.
//
// Base is the main-axis size it measures itself at when it is offered room for
// it — a fixed-size Box, which is what both harnesses mount, so the number is
// the declared one rather than a text measurement.
type Child struct {
	Name string `json:"name"`
	Base int    `json:"base"`
	// Pinned is core.FlexShrink(0): the renderer gives this child
	// Modifier.pinMainAxis, and GrMobFlexSolver gets a shrink factor of 0 for
	// it. One field, two spellings, because it is one declaration.
	Pinned bool `json:"pinned"`
}

// Measured is what the Compose transcription produces for one Row.
type Measured struct {
	// Offered is the main-axis MAXIMUM the Row's measure policy handed each
	// child. Carried even though it is not compared with anything on the CSS
	// side — CSS has no such number — because it is what makes the Compose
	// column readable: a child offered 0 got 0 for a reason, and a pinned child
	// ignoring a 20 is the whole of what the pin does.
	Offered []int `json:"offered"`
	// Mains is the main-axis extent each child came back at.
	Mains []int `json:"mains"`
	// Gaps is the spacing the Row actually inserted after each child —
	// spaceAfterLastNoWeight, per child, with the trailing one taken back off
	// the way the measure policy takes it off fixedSpace.
	//
	// It is a separate answer from Mains and not a restatement of the Row's
	// own Gap, which is the whole point: min(spacing, what is left) means an
	// overflowing Row inserts NO spacing after the child that spent the axis,
	// and a CSS flex line inserts it regardless. Three documents repeat that
	// sentence and until this field nothing measured it.
	Gaps []int `json:"gaps"`
	// RowMain is the measure policy's mainAxisLayoutSize — the Row's own extent
	// — which is NOT clamped to the maximum it was given. See MeasureCompose.
	RowMain int `json:"rowMain"`
}

// MeasureCompose is the transcription: what a Compose Row does with these
// children, given a definite main-axis extent.
//
// # The source it mirrors
//
// foundation-layout's RowColumnMeasurementHelper.kt, `measureWithoutPlacing`,
// the branch commented "First measure children with zero weight". Every child
// here is unweighted, so the weighted half of that function is not reached and
// is not transcribed.
//
//	// First measure children with zero weight.
//	val mainAxisMax = constraints.mainAxisMax
//	val placeable = placeables[i] ?: child.measure(
//	    constraints.copy(
//	        mainAxisMin = 0,
//	        mainAxisMax = if (mainAxisMax == Constraints.Infinity) {
//	            Constraints.Infinity
//	        } else {
//	            (mainAxisMax - fixedSpace).coerceAtLeast(0).toInt()
//	        },
//	        crossAxisMin = 0
//	    ).toBoxConstraints(orientation)
//	)
//	spaceAfterLastNoWeight = min(arrangementSpacingPx.toInt(),
//	    (mainAxisMax - fixedSpace - placeable.mainAxisSize()).coerceAtLeast(0).toInt())
//	fixedSpace += placeable.mainAxisSize() + spaceAfterLastNoWeight
//
// The Infinity arm is a Row that was itself offered no definite width. Every
// case here gives one, so MeasureCompose does not carry that branch — an
// unreachable arm transcribed is a second thing to keep true. It is quoted
// rather than elided because a quote with a branch quietly taken out of it is
// how a reader ends up believing the else arm is the whole rule.
//
// and, after the loop, for a Row with no weighted children at all:
//
//	// fixedSpace contains an extra spacing after the last non-weight child.
//	fixedSpace -= spaceAfterLastNoWeight
//	...
//	val mainAxisLayoutSize = max(
//	    (fixedSpace + weightedSpace).coerceAtLeast(0).toInt(),
//	    constraints.mainAxisMin
//	)
//
// # The three consequences that are easy to get wrong by reading it
//
//	there is no factor      `mainAxisMax - fixedSpace` is what is left, not a
//	                        share of a deficit. A fractional flex-shrink has
//	                        nothing here to scale, which is why the renderer
//	                        reads only shrinkPinned.
//
//	the gap collapses too   spaceAfterLastNoWeight is min(spacing, what is
//	                        left), so once the Row has overflowed the spacing
//	                        after a child is 0 as well. Nothing in the census
//	                        said so, and it follows from the same two lines.
//
//	the Row overflows its   mainAxisLayoutSize is max(content, mainAxisMin) and
//	                        is never coerced DOWN to mainAxisMax, and Size.kt's
//	                        own node reports `layout(placeable.width, ...)`
//	                        unclamped — so a fixed-width Row whose children
//	                        overflow reports the overflowing width rather than
//	                        clipping to its own. That is what makes the pin an
//	                        overflow on this target rather than a clip, which is
//	                        what `overflow: visible` does on the other three.
//
// # What each child does with what it is offered
//
// Both harnesses mount a fixed-size Box, so the child is Size.kt's SizeElement
// with `enforceIncoming = true`, whose node measures with
// `constraints.constrain(targetConstraints)` — the declared size clamped into
// the incoming range. With an offer of `remaining` that is min(base,
// remaining); with the pin's unbounded offer it is the base.
//
// The pinned branch is Renderer.kt's Modifier.pinMainAxis, whose two lines are
// the whole of it: measure against Constraints.Infinity, and report
// `layout(placeable.width, placeable.height)` — the measured size, NOT
// constrained back on the way out. Reporting a constrained size compiles and
// looks better behaved, and would make `fixedSpace` never exceed the Row's
// maximum, so nothing would overflow and the child would be drawn spilling out
// of a box the Row believed it fitted inside. mobile/verify refuses that
// spelling in the renderer by name; here it would show up as a RowMain that
// never passes Offer.
func MeasureCompose(offer, gap int, children []Child) Measured {
	m := Measured{
		Offered: make([]int, len(children)),
		Mains:   make([]int, len(children)),
		Gaps:    make([]int, len(children)),
	}

	// The Row's own constraint. Both the minimum and the maximum, because a
	// definite width is what Modifier.width sets and what every case here
	// gives it; the two are separate names below only where androidx uses
	// them differently.
	mainAxisMax, mainAxisMin := offer, offer

	fixedSpace := 0
	spaceAfterLastNoWeight := 0
	for i, c := range children {
		// What the Row offers this child: the space the ones before it did not
		// take, floored at zero. This is the number the whole divergence is
		// about — it depends on the children's ORDER, and CSS's answer does
		// not.
		remaining := max(mainAxisMax-fixedSpace, 0)
		m.Offered[i] = remaining

		var size int
		if c.Pinned {
			// pinMainAxis: the offer is replaced by Constraints.Infinity, so
			// `remaining` is ignored entirely — which is why a pin is honoured
			// wherever the child sits — and the size reported back is the one
			// measured.
			size = c.Base
		} else {
			// SizeNode with enforceIncoming: constraints.constrain(base).
			size = min(c.Base, remaining)
		}
		m.Mains[i] = size

		// The spacing after this child is itself clamped to what is left, so a
		// Row that has already overflowed inserts none.
		spaceAfterLastNoWeight = min(gap, max(mainAxisMax-fixedSpace-size, 0))
		m.Gaps[i] = spaceAfterLastNoWeight
		fixedSpace += size + spaceAfterLastNoWeight
	}
	// No weighted children, so the trailing spacing counted by the loop is not
	// part of the Row. Taken off the record as well as off the total: a gap
	// after the last child is not a gap anybody can measure, and leaving it in
	// Gaps would make the sum below disagree with the Row's own extent.
	fixedSpace -= spaceAfterLastNoWeight
	if len(children) > 0 {
		m.Gaps[len(children)-1] = 0
	}

	// weightedSpace is 0 here. Note the absence of any upper bound: this is
	// where the overflow becomes the Row's own size.
	m.RowMain = max(max(fixedSpace, 0), mainAxisMin)
	return m
}

// Case is one Row, with the Compose answer already computed so a harness in
// another language can compare against it without transcribing the loop again.
type Case struct {
	What     string  `json:"what"`
	Offer    int     `json:"offer"`
	Gap      int     `json:"gap"`
	Children []Child `json:"children"`

	// Compose is MeasureCompose's answer for this case.
	Compose Measured `json:"compose"`

	// CSS is the main-axis extent a CSS flex line gives each child.
	//
	// # Why it is stated here and computed nowhere
	//
	// This package deliberately does not solve a flex line. Two things already
	// do — GrMobFlexSolver, which ios/verify runs, and a real browser, which
	// wasm/verify's check 12 mounts — and a third spelling in Go would make
	// every comparison a question about whether two transcriptions agree.
	//
	// So this is the CLAIM, and both of those hold it. That is the same
	// arrangement the Compose column has with the census's prose: the table is
	// what the repository says, and the executable things are what say whether
	// it is true.
	//
	// It used to live only in this package's header, as three numbers in a
	// comment beside two mechanisms that produce them. The control row's
	// 24/80/16 was the sharpest case — the one row nothing derived from a
	// stated claim, so a reader who trusted those three numbers was trusting a
	// comment. It is a field now, and a wrong number here fails on a solver and
	// on a browser.
	CSS []int `json:"css"`

	// MainsAgreeWithCSS is whether the Compose extents are the same numbers a
	// CSS flex line produces for these children. Stated by the fixture and
	// asserted in BOTH directions by whoever solves the CSS half, for the
	// reason bandfixture states SharesADeficit: a check that only ever
	// confirmed a divergence would pass just as well if the divergence quietly
	// stopped being there.
	//
	// It is a fact about the case rather than a flag somebody set, and the
	// fixture's own test derives it — see TestTheAgreementIsWhetherThePinComesFirst.
	MainsAgreeWithCSS bool `json:"mainsAgreeWithCSS"`

	// GapsAgreeWithCSS is whether the Row inserted the spacing a CSS flex line
	// inserts: Gap between every adjacent pair, whatever happened to the
	// children.
	//
	// It is false exactly where the Row overflowed before it reached the last
	// gap, which is a divergence of the same kind as MainsAgreeWithCSS and
	// entirely separate from it — the gapped case below has the SAME extents on
	// both targets as its ungapped twin and different spacing. Asserted in both
	// directions by whoever measures the CSS half, for the reason the other
	// flag is: a check that only ever confirmed a collapse would pass just as
	// well against a target that had stopped collapsing.
	GapsAgreeWithCSS bool `json:"gapsAgreeWithCSS"`

	// Resolution is the smallest distance apart any two of this case's numbers
	// are, and therefore the finest distinction a harness checking it has to be
	// able to make. See resolution: it is what the two harnesses' two different
	// tolerances are both held to, and until it existed neither of them was
	// held to anything.
	//
	// Derived rather than stated, unlike the CSS column beside it. That column
	// is a claim about a flex line and has to be written down by somebody who
	// knows one; this is a property of the numbers already on the page, and a
	// stated copy of it would be a second thing to keep true.
	Resolution int `json:"resolution"`
}

// The Row every case is a rearrangement of.
//
// The numbers are chosen so the fixture cannot degenerate, and each is doing a
// job the others cannot:
//
//	pinnedBase > offer     otherwise a pinned child that comes FIRST is offered
//	                       more than it wants, keeps its base without the pin
//	                       having done anything, and the "order does not matter"
//	                       row proves nothing.
//	two other children     one sibling before and one after the pin, so "what
//	                       the ones before it left" is a different number from
//	                       "everything" and from "nothing".
//	unequal siblings       60 and 40 rather than 50 and 50: CSS shares the
//	                       deficit in proportion to the bases, so equal ones
//	                       would make a proportional answer indistinguishable
//	                       from an even split.
//	no padding             every number below is a child's extent against the
//	                       container's, with no subtraction for the reader to
//	                       do.
//	gap 0, except twice    four of the six rows carry none, which keeps the
//	                       solver's spacing out of a comparison that is about
//	                       shrink. The other two carry 8 and 16 and are about
//	                       spacing alone — each is an ungapped row repeated,
//	                       with a gap chosen so that neither column of extents
//	                       moves, so the only thing either can be measuring is
//	                       the gap. Two of them because the collapse is a min
//	                       and one row reaches only its ends: see partialGap.
const (
	rowOffer   = 120
	rowGap     = 0
	leadBase   = 60
	pinnedBase = 200
	tailBase   = 40
)

// spacedGap is the one case that has spacing, and the size is chosen rather
// than picked.
//
// Small enough that adding it changes no child's extent — the deficit already
// exceeds what the shrinkable children have to give, so CSS clamps them to zero
// with or without it, and Compose's offers are past zero by the time the second
// gap would be charged. That is what makes the spacing row a statement about
// SPACING: its two columns of extents are identical to the ungapped row it
// sits beside, and the only thing that differs is the gaps.
//
// Large enough to be a whole point of layout, so a browser measuring 8 against
// 0 is not measuring a rounding.
const spacedGap = 8

// partialGap is the second spacing row's, and it exists because the collapse is
// a min over TWO quantities and spacedGap only ever varies one of them.
//
// With 8 the Row charges the gap in full after the lead child (8 is less than
// the 60 left) and nothing at all after the pin (nothing is left). Both are
// arms of min(spacing, what is left), and both are ENDS of it: the answer is
// either the spacing or zero. The arm nobody had seen is the middle one, where
// what is left is a real number smaller than the gap and the Row charges a gap
// that is neither.
//
// Reaching it takes a gap larger than what remains before the pin, and the
// arithmetic bounds it on both sides. With the children written [lead, tail,
// pin] and a Row of 120, the space left after the tail child is `20 - g`:
//
//	g <= 10   20-g is at least g, so the spacing is charged in full and the
//	          middle arm is not reached
//	g >= 20   20-g is at or below zero, which is the collapse spacedGap
//	          already shows
//	g == 16   4 is charged where 16 was asked for — the arm itself
//
// 16 is also the largest of those that stays a round number of layout units,
// and it is small enough to leave both columns of extents identical to the
// ungapped row it sits beside (that needs g <= 20, since the tail child is 40
// wide and is offered 60-g). So the second spacing row is the same kind of
// statement the first one is: everything else about it is its twin's, and the
// only thing it can be measuring is the gap.
const partialGap = 16

// Cases returns the six arrangements, in the order the table in this package's
// header lists them.
func Cases() []Case {
	lead := Child{Name: "lead", Base: leadBase}
	pin := Child{Name: "pinned", Base: pinnedBase, Pinned: true}
	tail := Child{Name: "tail", Base: tailBase}

	// The control: the same three children with nothing pinned. Built from the
	// pinned one by clearing the flag rather than by writing a fourth literal,
	// so the pair really is "one declaration apart" and cannot drift into being
	// two different rows.
	unpinned := pin
	unpinned.Pinned = false

	return []Case{
		build("no pin: every child shrinks", rowGap, []int{24, 80, 16}, lead, unpinned, tail),
		build("the pinned child first", rowGap, []int{200, 0, 0}, pin, lead, tail),
		build("the pinned child between its siblings", rowGap, []int{0, 200, 0}, lead, pin, tail),
		build("the pinned child last", rowGap, []int{0, 0, 200}, lead, tail, pin),

		// And the one with spacing. Same three children in the same order as
		// the pin-middle row above, so the pair is one declaration apart the
		// way the control row is: both columns of extents are identical and
		// the gaps are not.
		//
		// The CSS column is that row's, unchanged, and saying why is the whole
		// content of this case. A flex line's gaps are used space like any
		// other: the deficit grows by 16 and both shrinkable children were
		// already clamped to zero, so nothing moves. Compose's does not move
		// either, for its own reason — the first child was offered the whole
		// Row and the pin ignores what it is offered. What differs is
		// underneath: `spaceAfterLastNoWeight` is min(spacing, what is left),
		// so the Row inserts 8 after the lead child and nothing after the pin,
		// where CSS inserts 8 in both places.
		build("the pinned child between its siblings, with spacing", spacedGap,
			[]int{0, 200, 0}, lead, pin, tail),

		// And the spacing row that reaches the middle arm of the same min. Same
		// three children as the pin-LAST row above, in the same order, so it is
		// that row plus a gap the way the row above is the pin-middle row plus
		// one — and its extents are that row's for the same two reasons: a flex
		// line's shrinkable children were already clamped to zero, and the
		// Compose offers are 120 and 44 against bases of 60 and 40, both of
		// which fit.
		//
		// What differs is the second gap. 16 is asked for and 4 is left, so the
		// Row charges 4 — a number that is neither the spacing nor zero, and
		// the only value in this fixture that could tell min(spacing, what is
		// left) from `spacing unless the Row has overflowed`. Those two rules
		// agree on every other row here.
		build("the pinned child last, with partial spacing", partialGap,
			[]int{0, 0, 200}, lead, tail, pin),
	}
}

// build measures one arrangement, states the CSS extents a flex line gives it,
// and derives the two flags.
func build(what string, gap int, css []int, children ...Child) Case {
	c := Case{
		What:     what,
		Offer:    rowOffer,
		Gap:      gap,
		Children: children,
		Compose:  MeasureCompose(rowOffer, gap, children),
		CSS:      css,
	}
	c.MainsAgreeWithCSS = agreesWithCSS(gap, children)
	c.GapsAgreeWithCSS = gapsAgreeWithCSS(gap, c.Compose.Gaps)
	c.Resolution = resolution(c)
	return c
}

// resolution is the smallest distance apart any two of this case's numbers are.
//
// # What it is for
//
// Two harnesses compare measurements against these numbers and each carries a
// tolerance of its own: ios/verify's pinEpsilon is 0.0001, because what it
// compares is GrMobFlexSolver's CGFloat arithmetic against integers, and
// browser.mjs's PIN_EPSILON is 0.05, because what IT compares is a real
// browser's LayoutUnits — sixty-fourths of a pixel — against the same integers.
// Two numbers, four hundred apart, for one fixture, and nothing anywhere said
// what either had to be true of.
//
// They are not supposed to be equal: they bound different errors, and a
// tolerance is only ever a claim about the machinery on ONE side of a
// comparison. What they share is the other side, which is this table, and the
// property that makes either of them safe is the same: a tolerance must be far
// below the smallest distinction the fixture asks a harness to make. Above it,
// a check accepts one of the fixture's numbers where another was meant — the
// partial-spacing case charges a gap of 4 against a Row that declares 16 and
// against the 0 a collapsed one would give, and a tolerance of 4 makes those
// three the same answer.
//
// So the fixture states the floor, both harnesses read it off the case they are
// checking, and each holds its own number to it. That is one claim in one place
// with two consumers, rather than two numbers nobody could compare.
//
// # Why every number and not the compared pairs
//
// Which pairs a harness actually distinguishes differs between the two — the
// browser measures gaps the Swift solver has no equivalent for — and a floor
// derived per-consumer would be two floors again. Every number in the case is a
// number some assertion on some target compares a measurement against, so the
// smallest gap between any two of them is the bound that serves both.
func resolution(c Case) int {
	seen := map[int]bool{c.Offer: true, c.Gap: true}
	for _, ch := range c.Children {
		seen[ch.Base] = true
	}
	for _, n := range c.CSS {
		seen[n] = true
	}
	for _, n := range c.Compose.Mains {
		seen[n] = true
	}
	for _, n := range c.Compose.Gaps {
		seen[n] = true
	}
	for _, n := range c.Compose.Offered {
		seen[n] = true
	}
	seen[c.Compose.RowMain] = true

	nums := make([]int, 0, len(seen))
	for n := range seen {
		nums = append(nums, n)
	}
	sort.Ints(nums)
	// A case with one distinct number cannot be told apart from anything, which
	// validate refuses long before this by insisting the children overflow the
	// offer. Reported as zero so that a harness holding a positive tolerance to
	// it fails rather than dividing by nothing.
	smallest := 0
	for i := 1; i < len(nums); i++ {
		if d := nums[i] - nums[i-1]; smallest == 0 || d < smallest {
			smallest = d
		}
	}
	return smallest
}

// gapsAgreeWithCSS is whether the Row inserted the spacing CSS inserts.
//
// A CSS flex line puts `gap` between every adjacent pair whatever happened to
// the children's sizes; a Compose Row clamps each one to what was left. So this
// is the whole rule, and it is derived from the measured gaps rather than from
// the position of the pin — the collapse depends on where the axis ran out,
// which is arithmetic and not a property of the arrangement.
func gapsAgreeWithCSS(gap int, gaps []int) bool {
	// The last entry is the trailing spacing the measure policy takes back off,
	// and CSS has no such gap either: with n children there are n-1 places a
	// gap can go.
	for i := 0; i+1 < len(gaps); i++ {
		if gaps[i] != gap {
			return false
		}
	}
	return true
}

// agreesWithCSS is the rule the table in this package's header shows.
//
// The two targets land on the same three numbers exactly when the pinned child
// is the FIRST one, and the reasoning is worth writing out because the answer
// is a coincidence rather than a shared rule:
//
//	CSS       the deficit (the three bases, less the offer) is larger than what
//	          the shrinkable children have to give, so every child that can
//	          shrink is clamped to 0 and the pinned one keeps its base. That is
//	          true in EVERY order, since a CSS flex line's sizes do not depend
//	          on position.
//	Compose   with the pin first, `fixedSpace` passes the Row's maximum
//	          immediately and every later child is offered 0. With it anywhere
//	          else, the children before it were offered room and took it.
//
// So this is not "the pin came first, therefore they agree" as a general law —
// it is that law for THIS fixture, whose deficit is deliberately larger than
// its shrinkable total. Both halves are tested rather than the position alone,
// so a change to the numbers that broke the premise makes the rule stop
// matching instead of quietly meaning something else.
//
// A row with NO pin never agrees, and that is the pre-existing divergence
// rather than a special case: CSS shares the deficit over every child in
// proportion to its base and Compose hands out what is left in order, so an
// overflowing row with more than one shrinkable child cannot come out the same.
// The control row is what that looks like — 24/80/16 against 60/60/0.
func agreesWithCSS(gap int, children []Child) bool {
	if len(children) == 0 || !children[0].Pinned {
		return false
	}
	shrinkable, natural := 0, 0
	for _, c := range children {
		natural += c.Base
		if !c.Pinned {
			shrinkable += c.Base
		}
	}
	// The gaps are used space in a flex line, so they are part of the deficit.
	// Nothing in this fixture turns on it — the deficit clears the shrinkable
	// total either way — and it is written out because leaving it out would
	// make the rule right about these numbers and wrong about the next ones.
	natural += gap * max(len(children)-1, 0)
	// The premise: CSS clamps every shrinkable child to zero. Without it the
	// rule above is about a fixture this is not.
	return natural-rowOffer >= shrinkable
}

// Validate reports what would make the fixture vacuous, so a harness can refuse
// to pass on numbers that measure nothing.
//
// Every case here is about an OVERFLOW: a Row that comfortably fits its
// children squeezes nobody, and a pinned child is then indistinguishable from
// an unpinned one — both harnesses would keep agreeing, about nothing. This is
// the same guard wasm/verify's fixed-size check makes when it asks whether the
// child is still bigger than the container.
func Validate() error { return validate(Cases()) }

// validate is the decision, as a function of the cases, so its arms can be
// reached by handing it a row that has the fault — which Cases() by
// construction never does, and a guard nothing can fail is a guard nobody has
// checked.
func validate(cases []Case) error {
	for _, c := range cases {
		natural := 0
		for _, ch := range c.Children {
			natural += ch.Base
		}
		if natural <= c.Offer {
			return fmt.Errorf("%q: the children want %d and the Row offers %d, so "+
				"nothing overflows and every check over this case passes by never "+
				"reaching the squeeze", c.What, natural, c.Offer)
		}
		if len(c.Children) < 3 {
			return fmt.Errorf("%q has %d children: with fewer than three there is no "+
				"child both before and after the pin, and 'what the ones before it "+
				"left' stops being a different number from 'everything'",
				c.What, len(c.Children))
		}
		// And the stated CSS column is a claim about THESE children. A column
		// of the wrong length is one about a different Row, and it would
		// otherwise reach both harnesses and fail there as an index error or
		// as a silence, depending on which read it first. Last of the three
		// because the two above are about the Row and this is about the claim
		// printed beside it.
		if len(c.CSS) != len(c.Children) {
			return fmt.Errorf("%q states %d CSS extents for %d children, so the column "+
				"the census prints and the row it prints it for are about different Rows",
				c.What, len(c.CSS), len(c.Children))
		}
	}
	return nil
}
