// Package bandfixture states the geometry of components.GroupHeader's band in
// the two arrangements its insets have had, so a renderer other than the web
// can be asked whether they are the same band.
//
// # The claim this exists to settle
//
// The band's padding used to be on the Row and is now on the control inside it
// (see components.bandInsets). The argument for the move is that a press should
// land on the whole band rather than on a strip in the middle of it, and the
// argument that the move is *free* is that padding on a stretched child fills
// exactly the space the same padding on its parent held:
//
//	 Row ───────────────────────────────────
//	│        ┌───────────────┐      ┌───┐    │  before
//	│  16px  │ January 2026  │  8px │ 3 │ 16 │  the insets are the Row's and
//	│        └───────────────┘      └───┘    │  the button is the box
//	 ────────────────────────────────────────
//
//	 Row ───────────────────────────────────
//	│┌────────────────────────────┐ ┌───┐    │  after
//	││  16px January 2026     8px │ │ 3 │ 16 │  the insets are the control's
//	│└────────────────────────────┘ └───┘    │  and the target is the band
//	 ────────────────────────────────────────
//
// That is exactly true in CSS flex, where it was verified by the pixels the
// change did not move. On the other three targets it was an *assumption*: each
// renderer resolves padding and flex-grow with its own code, and nothing
// anywhere had asked whether the two arrangements come out the same there.
//
// ios/verify has the one thing that can answer it — GrMobFlexSolver, the CSS
// flex arithmetic the SwiftUI Layout runs on, split out precisely so it can be
// checked without a simulator. So this is the band's numbers, carried in the
// same transcript internal/menufixture's picker cases ride in, and ios/verify's
// band.swift solves both arrangements and compares.
//
// It rides in wasm/verify's transcript too, and for the half ios/verify cannot
// settle: browser.mjs lays both arrangements out in a real Chrome at the same
// offers and measures the rects. That is the target the original "verified by
// the pixels the change did not move" claim was about, made into a check — and
// it is what turned the solver's overflow difference from a suspected artefact
// into a recorded divergence between two renderers. See SharesADeficit.
//
// # Why the numbers are read off a rendered band
//
// A transcribed 16 is a copy, and the whole point is to check the arrangement
// the widget actually builds. Cases renders a real components.GroupHeader and
// reads the insets, the gap and the flex-grow off the resulting nodes, so a
// band whose recipe changed changes the fixture, and the Swift check is about
// the new band rather than about a picture of the old one.
//
// The *before* arrangement is derived from the *after* one by a stated rule
// (see Arrangement and Rewind), not transcribed either: it is "the same pixels,
// one node out", which is the claim, so building it any other way would be
// assuming the answer in the fixture.
//
// # What the sizes are, and what they are not
//
// Label and Badge are synthetic. A label's rendered width is a text
// measurement, which is SwiftUI's to make and nothing in Go can know — and it
// is not the subject: what is under test is whether one arrangement of insets,
// gaps and grow weights resolves to the same geometry as another, which is
// arithmetic over whatever the content happens to measure. The sizes are chosen
// to make the cases distinguishable (a label narrower than the offer so there
// is slack to grow into, a badge with a width of its own so the trailing edge
// has something to be hard against), and one case gives the badge a
// deliberately oversized height, because that is the one place the two
// arrangements do *not* agree.
//
// # The half made-up sizes cannot reach, and where it is checked instead
//
// Synthetic sizes are right for the arithmetic and they take one claim out of
// reach. Every SameHeight case rests on the padded control being the band's
// tallest child, and the reason has two halves: the control's vertical insets
// are larger than the badge's, and both wrap the same caption tier. The first
// is arithmetic and bandfixture_test.go checks it. The second is a statement
// about glyphs — the label is a *bold* caption and the badge's text a plain one
// — and "no shorter" is a measurement nothing in Go can take.
//
// It is not a formality: the insets differ by four points, so a bold caption
// shorter than a plain one by more than that would make the badge the tallest
// child of a real band, and every SameHeight case here would be describing a
// layout the framework does not build.
//
// wasm/verify's gen.go renders real components.GroupHeaders through every
// bundled theme and browser.mjs measures them with real glyphs in them (see
// bandRender there, and check 10). That is where this half is checked, and it is
// the only place it can be.
//
// It lives under internal/ for the reason menufixture does: it is not part of
// the framework's API, it is a fact this repository's own harnesses share.
package bandfixture

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/rohanthewiz/grmob/components"
	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/htmlout"
)

// Insets are the four resolved sides, in points.
//
// Resolved, not raw: core.EdgeInsets carries the four sides *and* a
// Horizontal/Vertical shorthand pair, and every renderer resolves a side as
// "the explicit field if non-zero, otherwise the shorthand for that axis". The
// band's own recipe uses both spellings — PaddingHorizontal and PaddingVertical
// for the pair, PaddingRight for the badge's own inset — so a fixture reading
// the raw fields would report a band with no top padding.
type Insets struct {
	Top    float64 `json:"top"`
	Right  float64 `json:"right"`
	Bottom float64 `json:"bottom"`
	Left   float64 `json:"left"`
}

// Size is a synthetic content box. See the package comment.
type Size struct {
	W float64 `json:"w"`
	H float64 `json:"h"`
}

// Arrangement is one way of splitting the band's chrome between the Row and the
// control inside it.
type Arrangement struct {
	// What names the arrangement in a failure.
	What string `json:"what"`
	// Row is the band Row's own padding, and Gap its inter-item gap.
	Row Insets  `json:"row"`
	Gap float64 `json:"gap"`
	// Control is the padding on the Row's growing child, and Grow its weight.
	Control Insets  `json:"control"`
	Grow    float64 `json:"grow"`

	// Align is the Row's cross-axis alignment, carried because the whole
	// cross-axis half of the comparison depends on it: with children centred, a
	// Row's height is its tallest child plus its own vertical padding, and
	// moving that padding onto one child therefore stops it being added to the
	// other. Under `stretch` it would not be — every child would be the Row's
	// height and the padding would be inside them — so a band that quietly
	// stopped centring would make band.swift's height comparison model a layout
	// nothing builds. It refuses a case that is not centred rather than
	// answering for one.
	Align string `json:"align"`
}

// Rewind returns the arrangement a's insets came from: the same chrome, one
// node out.
//
// The rule is the move, read backwards, and it is assignments rather than a
// table of numbers because the numbers are the widget's:
//
//	the control's leading and vertical insets   -> the Row's
//	the control's trailing inset                -> the Row's gap, when something
//	                                               follows the control
//	                                            -> the Row's trailing inset,
//	                                               when nothing does
//	the Row's own trailing inset                -> unchanged
//
// trailing says whether anything follows the control, and it is what makes the
// asymmetric side answerable. The control's trailing padding is two different
// things depending on that: with a badge it *is* the gap before the badge — it
// was a Gap on the Row before the move — and with the count hidden there is
// nothing to be a gap before, so it is the band's own trailing breathing room
// and belongs on the Row beside the leading one. components.bandInsets' own
// `trailing` parameter is the same fact from the other side, and this is why it
// takes 0 to mean "whatever the leading inset is".
//
// The Row's *own* trailing inset is a third thing again and never moves: it is
// the badge's breathing room past its right edge, which was always outside the
// control.
func (a Arrangement) Rewind(trailing bool) Arrangement {
	out := Arrangement{
		What: "on the Row",
		Row: Insets{
			Top:    a.Control.Top,
			Right:  a.Row.Right,
			Bottom: a.Control.Bottom,
			Left:   a.Control.Left,
		},
		Control: Insets{},
		Grow:    a.Grow,
		// The move was about padding. Alignment did not change, and a Rewind
		// that let it would be comparing two different layouts.
		Align: a.Align,
	}
	if trailing {
		out.Gap = a.Control.Right
	} else {
		out.Row.Right += a.Control.Right
	}
	return out
}

// Case is one band, both ways, with the sizes and offers to solve it at.
type Case struct {
	What string `json:"what"`
	// Now is the arrangement components.GroupHeader builds today; Before is
	// Now.Rewind().
	Now    Arrangement `json:"now"`
	Before Arrangement `json:"before"`

	// Label is the growing child's content box and Badge the trailing item's.
	// A zero Badge width means the band has no badge.
	Label Size `json:"label"`
	Badge Size `json:"badge"`

	// BadgeInsets is the badge's own padding, carried so the cross-axis note
	// below can be checked rather than asserted: the two arrangements agree on
	// the band's height exactly while the padded control is the tallest child,
	// and the control's vertical insets being larger than the badge's is half
	// of why it is.
	BadgeInsets Insets `json:"badgeInsets"`

	// Offers are the container widths to solve at. A negative offer means "no
	// definite offer" — SwiftUI probing for an ideal size — which is the case
	// the two arrangements have to agree on as well, and the only one where the
	// answer is a sum rather than a distribution.
	Offers []float64 `json:"offers"`

	// SameHeight is whether the two arrangements produce the same band height.
	// True for every real band and false for the oversized-badge case; see
	// band.swift, which asserts both directions.
	SameHeight bool `json:"sameHeight"`

	// SharesADeficit is whether an offer narrower than the band's content is
	// divided between more than one child.
	//
	// It decides what happens in the one case where the two arrangements are
	// not the same band. Under overflow the solver shrinks each child in
	// proportion to its base size, and a base includes the child's own padding
	// — so the control's 32 points of insets are inside the proportion in one
	// arrangement and outside it in the other, and the label ends up with
	// different room. That only shows when there is a second child to divide
	// the deficit with: a lone growing control is clamped to the container
	// either way, and the two arrangements land on the same number for the same
	// reason a single child always does.
	//
	// So a banded case diverges and an unbadged one does not, and band.swift
	// asserts both — the second is what keeps the first from being "whatever
	// happened".
	//
	// It is a claim about the SwiftUI solver alone. wasm/verify/browser.mjs
	// mounts these same arrangements in a real Chrome, and CSS shrinks in
	// proportion to the *inner* flex base size — which excludes the child's own
	// padding — so there the badged case agrees under overflow like every
	// other. Both directions are asserted on both targets, so the divergence is
	// a recorded difference between two renderers rather than a defect in
	// either.
	//
	// # The third renderer, and why it is derived rather than measured
	//
	// Compose is the target this field has no row for. Its Row has no
	// proportional shrink at all: an unweighted child is measured with what is
	// left of the main axis, a weighted one gets (available - fixed) / total
	// weight, and this band's badge is unweighted. So the whole deficit lands
	// on the growing control in both arrangements — the badge keeps its width
	// where both other renderers shrink it — and the two arrangements come out
	// equal, because the control's padding is inside its weighted extent
	// either way. Three renderers, three different reasons, and only one of
	// them (the SwiftUI solver) makes the two arrangements differ.
	//
	// That paragraph is a derivation and not a measurement, and the difference
	// matters enough to say twice. The other two answers are executable because
	// the arithmetic is ours or the browser is a browser; Compose's Row is
	// androidx's code, needs the Android runtime to measure anything, and
	// cannot even be pinned by reading — this repository's Compose BOM resolves
	// foundation-layout to a version whose sources a gradle cache does not
	// hold. mobile/verify's TestTheComposeRowDelegatesItsDistributionToCompose
	// holds the premise instead: the census stops at three rows because Android
	// delegates, and a renderer that stopped delegating puts the answer back
	// within reach and makes it ours to be wrong about.
	SharesADeficit bool `json:"sharesADeficit"`
}

// Cases returns the band cases, read off real components.GroupHeader renders.
func Cases() []Case {
	withBadge := arrangementOf("GroupHeader, with a count badge", true)
	noBadge := arrangementOf("GroupHeader, count hidden", false)

	// The offers: wider than the content, narrower than it (so the shrink arm
	// runs), exactly the natural width, and no definite offer at all.
	offers := []float64{360, 240, 120, -1}

	badge := Size{W: 24, H: 16}
	return []Case{
		{
			What: withBadge.arr.What, Now: withBadge.arr, Before: withBadge.arr.Rewind(true),
			Label: Size{W: 100, H: 16}, Badge: badge, BadgeInsets: withBadge.badge,
			Offers: offers, SameHeight: true, SharesADeficit: true,
		},
		{
			// No badge: one child, so the growing control is the whole line and
			// the trailing inset is the control's own rather than the gap.
			What: noBadge.arr.What, Now: noBadge.arr, Before: noBadge.arr.Rewind(false),
			Label: Size{W: 100, H: 16}, Badge: Size{}, BadgeInsets: noBadge.badge,
			Offers: offers, SameHeight: true, SharesADeficit: false,
		},
		{
			// The one place the arrangements differ, and it is not a bug in
			// either: with align-items centre, a Row's height is the tallest
			// child plus the Row's own vertical padding. Move that padding onto
			// one child and it stops being added to the *other* one — so a
			// badge taller than the padded control makes the band shorter by
			// exactly the vertical insets.
			//
			// It cannot happen to the real band, which is why it is a case with
			// a made-up badge rather than a defect: the control's vertical
			// insets are larger than the badge's (checked in Go, see
			// bandfixture_test.go) and both wrap the same caption type, so the
			// control is the taller child in every theme.
			What: "a badge taller than the control (not a shape the band has)",
			Now:  withBadge.arr, Before: withBadge.arr.Rewind(true),
			Label: Size{W: 100, H: 16}, Badge: Size{W: 24, H: 48},
			BadgeInsets: withBadge.badge,
			Offers:      offers, SameHeight: false, SharesADeficit: true,
		},
	}
}

// read is one render's worth of geometry.
type read struct {
	arr   Arrangement
	badge Insets
}

// arrangementOf renders a plain (non-collapsible) band and reads its geometry.
//
// Plain rather than a disclosure, deliberately. The disclosure branch puts the
// insets on a *button* one level inside the Row's growing heading wrapper, which
// is a different arrangement asking a different question — whether a
// non-growing child fills a growing parent — and the flex solver, which is a
// main-axis distributor, is not what answers it.
func arrangementOf(what string, badge bool) read {
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	row := components.GroupHeader{
		Group:     components.Group{Key: "2026-01", Label: "January 2026", Count: 3},
		HideCount: !badge,
	}.Render(ctx)

	if row.Style == nil {
		panic("bandfixture: the band Row rendered with no Style")
	}
	wantChildren := 1
	if badge {
		wantChildren = 2
	}
	if len(row.Children) != wantChildren {
		panic(fmt.Sprintf("bandfixture: the band rendered %d children, want %d — the "+
			"shape this fixture reads has changed", len(row.Children), wantChildren))
	}
	control := row.Children[0]
	if control.Style == nil || control.Style.FlexGrow == 0 {
		panic("bandfixture: the band's first child does not grow — the whole claim " +
			"this fixture carries is about padding on a *stretched* child")
	}
	out := read{arr: Arrangement{
		What:    what,
		Row:     resolve(row.Style.Padding),
		Gap:     row.Style.Gap,
		Control: resolve(control.Style.Padding),
		Grow:    control.Style.FlexGrow,
		Align:   string(row.Style.AlignItems),
	}}
	if badge {
		if row.Children[1].Style == nil {
			panic("bandfixture: the band's badge rendered with no Style")
		}
		out.badge = resolve(row.Children[1].Style.Padding)
	}
	return out
}

// resolve turns a core.EdgeInsets into its four resolved sides, through
// htmlout.EdgeCSS.
//
// Through the renderer's own function rather than by reimplementing the
// "explicit side, else the axis shorthand" rule: that rule exists four times in
// this repository (htmlout.EdgeCSS, edgeToCSS in the WASM runtime, parseEdges
// in both native GrMobStyle files) and a fifth copy here would be one more
// thing to keep in step. Parsing the CSS back out is the cost, and it is small:
// the output is four "<n>px" tokens in CSS order, which is the one shape that
// function has ever had.
func resolve(e core.EdgeInsets) Insets {
	parts := strings.Fields(htmlout.EdgeCSS(e))
	if len(parts) != 4 {
		panic("bandfixture: htmlout.EdgeCSS produced " + htmlout.EdgeCSS(e) +
			", which is not four sides")
	}
	n := func(i int) float64 {
		v, err := strconv.Atoi(strings.TrimSuffix(parts[i], "px"))
		if err != nil {
			panic("bandfixture: htmlout.EdgeCSS produced " + parts[i])
		}
		return float64(v)
	}
	return Insets{Top: n(0), Right: n(1), Bottom: n(2), Left: n(3)}
}
