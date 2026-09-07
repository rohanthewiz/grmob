// Package pinfixture states one overflowing Row four ways, so that what
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
//	that version        TestTheComposeSourcesAreTheVersionTheBOMResolves derives
//	                    it from the BOM's own pom, so the source read is the
//	                    source the app builds against.
//	the renderer's half TestTheComposeChildrenLoopsPinOnTheirOwnAxis and
//	                    TestTheComposePinMeasuresUnboundedAndReportsWhatItMeasured
//	                    hold Renderer.kt to the two lines PinnedChild below is
//	                    a transcription of.
//
// So the weakest link is stated rather than hidden: somebody read androidx's
// loop and wrote it out here, and if they read it wrong no test in this
// repository would know. Every other link is checked.
//
// # The fixture, and what each case is for
//
// One Row, one set of children, four arrangements. The Row is 120 wide and the
// children want 60, 200 and 40, so the deficit (180) is larger than what the
// two shrinkable children have to give (100) — which is forced, not chosen: a
// pinned child whose base exceeds the container makes it arithmetically
// unavoidable, and a pinned child whose base does NOT exceed the container is
// not pinned against anything when it comes first.
//
//	                      CSS (GrMobFlexSolver, ios/verify)   Compose (below)
//	no pin                24, 80, 16                          60, 60,  0
//	pin first  [P,A,B]   200,  0,  0                         200,  0,  0
//	pin middle [A,P,B]     0,200,  0                          60,200,  0
//	pin last   [A,B,P]     0,  0,200                          60, 40,200
//
// Three claims come out of that table, and each needs a different row to be
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
// The browser. The DOM's answer for this Row is not measured anywhere;
// GrMobFlexSolver is this repository's CSS arithmetic and the census already
// records that it and a real Chrome diverge under overflow when a child has
// padding of its own. These children have none, so the two should agree — which
// is a sentence with nothing behind it, and is why the fixture carries no
// padding rather than carrying some and hand-waving the difference.
package pinfixture

import "fmt"

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
//	        mainAxisMax = (mainAxisMax - fixedSpace).coerceAtLeast(0).toInt(),
//	        crossAxisMin = 0
//	    ).toBoxConstraints(orientation)
//	)
//	spaceAfterLastNoWeight = min(arrangementSpacingPx.toInt(),
//	    (mainAxisMax - fixedSpace - placeable.mainAxisSize()).coerceAtLeast(0).toInt())
//	fixedSpace += placeable.mainAxisSize() + spaceAfterLastNoWeight
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
		fixedSpace += size + spaceAfterLastNoWeight
	}
	// No weighted children, so the trailing spacing counted by the loop is not
	// part of the Row.
	fixedSpace -= spaceAfterLastNoWeight

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
//	no padding, no gap     every number below is a child's extent against the
//	                       container's, with no subtraction for the reader to
//	                       do — and gap 0 keeps the solver's spacing out of a
//	                       comparison that is about shrink. MeasureCompose
//	                       implements the spacing line anyway and this package's
//	                       own test exercises it.
const (
	rowOffer   = 120
	rowGap     = 0
	leadBase   = 60
	pinnedBase = 200
	tailBase   = 40
)

// Cases returns the four arrangements, in the order the table in this package's
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
		build("no pin: every child shrinks", lead, unpinned, tail),
		build("the pinned child first", pin, lead, tail),
		build("the pinned child between its siblings", lead, pin, tail),
		build("the pinned child last", lead, tail, pin),
	}
}

// build measures one arrangement and states whether CSS would agree with it.
func build(what string, children ...Child) Case {
	c := Case{
		What:     what,
		Offer:    rowOffer,
		Gap:      rowGap,
		Children: children,
		Compose:  MeasureCompose(rowOffer, rowGap, children),
	}
	c.MainsAgreeWithCSS = agreesWithCSS(children)
	return c
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
func agreesWithCSS(children []Child) bool {
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
	}
	return nil
}
