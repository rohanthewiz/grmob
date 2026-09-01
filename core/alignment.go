package core

// The enumerations of the three alignment types style.go declares, and the
// one subset that is not a whole type.
//
// Go cannot enumerate the constants of a named string type at run time, so
// every one of these lists is a second copy of a const block, which is the
// shape of duplication that quietly goes stale. Each is pinned back to its
// declaration by alignment_enum_test.go, which reads style.go's syntax tree —
// the same arrangement ContentModes() has, and for the same reason: adding a
// constant without adding it here should fail `go test ./...` rather than
// silently shrinking the set every renderer's coverage check rests on.
//
// They live here rather than beside the constants because style.go is the
// Style struct and its ~40 prop helpers; four enumerations and the paragraphs
// explaining what a renderer owes each of them would be the longest thing in
// it and the least related to the rest. The pin does not care where the list
// is, only where the constants are.
//
// # Why a renderer needs these at all
//
// Four renderers each map these values onto their own vocabulary, and two of
// the four map them onto nothing at all — htmlout and the WASM runtime pass
// JustifyContent and AlignItems through verbatim, because core's spellings
// *are* the CSS ones. That is why the DOM pair has no drift to check here and
// the natives do: SwiftUI and Compose dispatch on the string, arm by arm, and
// a value with no arm falls into a catch-all and renders as something else
// with no error anywhere.
//
//	list                | who is held to it
//	--------------------+-------------------------------------------------
//	Alignments()        | the census: every other list is a subset of this
//	TextAlignments()    | htmlout.TextAligns, the runtime's textAlignFor,
//	                    | Renderer.swift's grMobTextAlignment,
//	                    | Renderer.kt's Text textAlign
//	JustifyContents()   | GrMobFlex.swift's leading/gap,
//	                    | Renderer.kt's horizontal/verticalArrangement
//	AlignItemsValues()  | GrMobFlex.swift's crossOffset,
//	                    | Renderer.swift's crossAlignmentH,
//	                    | Renderer.kt's Row/Column/List cross alignment
//
// The native pins live in mobile/verify; the DOM ones in htmlout and
// wasm/verify. See mobile/verify/switchlabels_test.go for why those are a
// parse of the native source and not a compile.
//
// Every list returns a fresh slice per call rather than exposing a
// package-level var, which any importer could write to. A handful of elements
// is cheaper to build than to defend.

// Alignments returns every declared Alignment, in declaration order.
//
// This is the census list — the one that must equal the const block exactly —
// and it is deliberately the only one of the four that no renderer is held to.
// Alignment carries two roles that no single dispatch serves:
//
//	value    | text-align role      | cross-axis role
//	---------+----------------------+---------------------------------
//	start    | leading edge         | items packed to the start edge
//	center   | centered             | items centered on the cross axis
//	end      | trailing edge        | items packed to the end edge
//	justify  | justified text       | none
//	stretch  | none                 | items filled to the cross extent
//	baseline | none                 | items aligned on their baselines
//
// Style.Align feeds both: it is the text alignment of a Text node, and it is
// also the fallback the native containers read when AlignItems is unset (see
// crossAlignmentH in Renderer.swift). So a text dispatch that was required to
// answer for "stretch", or a cross-axis dispatch required to answer for
// "justify", would be made to write an arm that can never mean anything.
// TextAlignments is the subset that is a real text alignment;
// AlignItemsValues is the vocabulary the cross-axis dispatches actually
// dispatch on.
//
// The two roles are not split into two Go types because that would be a
// breaking change for any caller already passing AlignStretch to Align(), and
// because the split is real only at the point of *use* — Style has one Align
// field, and which role it plays depends on the node it lands on.
func Alignments() []Alignment {
	return []Alignment{
		AlignStart,
		AlignCenter,
		AlignEnd,
		AlignStretch,
		AlignBaseline,
		AlignJustify,
	}
}

// TextAlignments returns the Alignments that name a real text alignment, in
// declaration order: the coverage every renderer's text dispatch owes.
//
// AlignStretch and AlignBaseline are excluded because there is no such thing
// as stretched or baseline-aligned text — CSS text-align has no such keyword,
// SwiftUI's TextAlignment has three members, and Compose's TextAlign has no
// analogue either. They are cross-axis values that share the type; a text
// dispatch that receives one is right to fall through to its default.
//
// AlignJustify *is* included, and is the value that made this list worth
// writing. Before it existed, justified text rendered on exactly one of the
// four targets: Renderer.kt mapped it to TextAlign.Justify, Renderer.swift
// fell through to .leading, htmlout emitted no declaration at all, and the
// WASM runtime did not read Align in the first place. One value, four
// behaviors, and nothing anywhere that could notice.
//
// Being on this list does not mean a target can honor the value — SwiftUI
// genuinely cannot justify text — it means the target has to *say* what it
// does with it. An explicit arm that falls back to leading, with a comment
// naming the platform limit, is coverage; silence is not.
func TextAlignments() []Alignment {
	return []Alignment{
		AlignStart,
		AlignCenter,
		AlignEnd,
		AlignJustify,
	}
}

// JustifyContents returns every declared JustifyContent, in declaration order.
//
// Main-axis distribution. The DOM pair emits these verbatim (core's spellings
// are the CSS ones), so the drift risk is entirely on the natives, where the
// six values are spread across dispatches that each answer for part of the
// question: GrMobFlex.swift computes a leading offset in one switch and an
// inter-item gap in another, and a value absent from *both* silently renders
// as flex-start.
func JustifyContents() []JustifyContent {
	return []JustifyContent{
		JustifyStart,
		JustifyCenter,
		JustifyEnd,
		JustifyBetween,
		JustifyAround,
		JustifyEvenly,
	}
}

// AlignItemsValues returns every declared AlignItems, in declaration order.
//
// Named for its values rather than its type because the type name is taken —
// the same collision AlignItemsProp in style_props.go had to work around.
//
// Cross-axis placement. AlignItemsStretch is the member that behaves unlike
// the rest on every native: a stretched child is not *placed* differently, it
// is *measured* differently, so neither SwiftUI's alignment nor Compose's
// Alignment enum can express it and both runtimes handle it off the dispatch
// (a fill modifier on the child, a solver branch). A stretch arm in those
// switches is therefore expected to be a no-op — but an explicit no-op arm is
// how a reader learns the value was considered and handled elsewhere, which
// is the whole argument for holding a switch to a list.
func AlignItemsValues() []AlignItems {
	return []AlignItems{
		AlignItemsStart,
		AlignItemsCenter,
		AlignItemsEnd,
		AlignItemsStretch,
	}
}
