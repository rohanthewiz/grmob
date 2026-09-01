package verify

import (
	"sort"
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

// The native alignment dispatches, held to core's three alignment
// enumerations.
//
// This is the ContentMode exercise repeated across a much wider surface, and
// with one difference that changes what the checks are for. ContentMode is
// restated four times and the two DOM copies are real tables, so they could be
// compared value-for-value. JustifyContent and AlignItems are not restated on
// the DOM side at all: core's spellings *are* the CSS ones, so htmlout and the
// WASM runtime emit them verbatim and cannot be wrong about a value they never
// interpret. All of the drift risk is native, and all of it is coverage.
//
//	value             | CSS               | SwiftUI                 | Compose
//	------------------+-------------------+-------------------------+------------------
//	justify-content   | verbatim          | solver arithmetic       | Arrangement
//	align-items       | verbatim          | solver offset / stack   | Alignment.*
//	text-align        | table (textalign) | TextAlignment           | TextAlign
//
// Eleven dispatches across three files, and every one of them ends in a
// catch-all that renders *something*. That is what makes the failure
// invisible: a value with no arm does not error, it packs to the start. A
// seventh JustifyContent added to core would today lay out correctly on both
// DOM targets — which need no arm — and silently as flex-start on iOS and
// Android.
//
// See switchlabels_test.go for the parse and why it is one.

// coverage is the comparison every check below makes: a dispatch's arms
// against a list core hands out, in both directions.
//
// Both directions matter and they fail for different reasons. A required value
// with no arm falls to the catch-all and renders as something else. An arm
// with no value behind it is dead code that reads as deliberate support — a
// typo that has quietly never matched, or a value removed from core and left
// behind — and the next person to read the renderer would reasonably conclude
// the framework supports it.
//
// allowed exists because several of these dispatches answer for two
// vocabularies at once. Style.Align doubles as the cross-axis fallback when
// AlignItems is unset, so a Column's alignment switch legitimately carries
// "start" and "end" arms that are core.Alignment values rather than
// core.AlignItems ones. Those are permitted, not required — a renderer may
// decline to honor the fallback, as GrMobRow does — and listing them here is
// what keeps the second direction sharp for everything else.
type coverage struct {
	// file and fn name the dispatch in failure messages.
	file, fn string
	// required is the list every value of which must have an arm.
	required []string
	// allowed may appear as arms without being required.
	allowed []string
	// consequence completes the sentence "…has no arm for X, so it ". It is
	// per-dispatch because the consequences genuinely differ: a missing
	// justify-content arm packs the children to the start, a missing text-align
	// arm renders leading, and a missing cross-axis arm places at the leading
	// edge. A generic message would be worth less than the sentence it saved.
	consequence string
}

func (c coverage) check(t *testing.T, arms []string) {
	t.Helper()

	missing := map[string]bool{}
	for _, value := range c.required {
		missing[value] = true
	}
	permitted := map[string]bool{}
	for _, value := range c.allowed {
		permitted[value] = true
	}

	seen := map[string]bool{}
	for _, label := range arms {
		switch {
		case seen[label]:
			// A second arm for a value the switch already matched is
			// unreachable in both languages.
			t.Errorf("%s: %s has a second arm for %q, which is unreachable", c.file, c.fn, label)
		case missing[label]:
			delete(missing, label)
		case permitted[label]:
			// A value from the other vocabulary this dispatch also serves.
		default:
			t.Errorf("%s: %s has an arm for %q, which is not a value core can produce here — "+
				"either a typo that has never matched, or a constant removed from core and left "+
				"behind", c.file, c.fn, label)
		}
		seen[label] = true
	}

	// Sorted so a run that reports several gaps reports them in the same order
	// twice; map order would otherwise shuffle them between identical runs.
	var gaps []string
	for value := range missing {
		gaps = append(gaps, value)
	}
	sort.Strings(gaps)
	for _, value := range gaps {
		t.Errorf("%s: %s has no arm for %q — it falls through to the catch-all, so it %s",
			c.file, c.fn, value, c.consequence)
	}
}

// justifyValues, alignItemsValues and textAlignValues drop the named Go types
// so the lists can be compared with labels read out of native source, which
// never had them.
func justifyValues() []string    { return asStrings(core.JustifyContents()) }
func alignItemsValues() []string { return asStrings(core.AlignItemsValues()) }
func textAlignValues() []string  { return asStrings(core.TextAlignments()) }

func asStrings[T ~string](values []T) []string {
	out := make([]string, len(values))
	for i, v := range values {
		out[i] = string(v)
	}
	return out
}

// crossAxisFallback is the set of core.Alignment spellings that reach a
// cross-axis dispatch through Style.Align when AlignItems is unset. Permitted
// wherever a container reads that fallback; absent from GrMobRow, which
// deliberately does not.
var crossAxisFallback = []string{
	string(core.AlignStart),
	string(core.AlignCenter),
	string(core.AlignEnd),
	string(core.AlignStretch),
	string(core.AlignBaseline),
}

// --- justify-content --------------------------------------------------------

// The iOS solver answers justify-content with two dispatches rather than one:
// the leftover space is spent either before the run of children or between
// them, and each function owns one half. Both list all six values and each
// returns 0 for the half the other handles.
//
// Checking them separately rather than unioning their arms is the point. A
// union would pass if `leading` covered space-between and `gap` covered
// flex-end — that is, if each half answered for values it has no business
// answering for — and the arrangement that makes this readable is precisely
// that each half states its own complete opinion.
func TestSwiftFlexSolverCoversEveryJustifyContent(t *testing.T) {
	for _, fn := range []string{"func leading(free:", "func gap(free:"} {
		syntax := swiftSwitch.with(swiftFlex, fn, "switch justify {")
		coverage{
			file:        "GrMobFlex.swift",
			fn:          fn,
			required:    justifyValues(),
			consequence: "contributes nothing and the children pack to the leading edge",
		}.check(t, syntax.labels(t))
	}
}

// justifyClaimsFreeSpace decides whether the container fills a definite offer
// or hugs its children, and it is the one copy of core's list in either
// renderer that is not a switch — so it is read as an array literal.
//
// The check is a classification rather than a coverage count: every
// JustifyContent must be either in the array or be flex-start, the single
// value that cannot spend leftover space (packing to the leading edge looks
// identical at any container size). Stated that way, a seventh value added to
// core has to be classified here rather than defaulting into "does not claim",
// which would make the container hug when it should fill — a layout that is
// wrong only when there is space to spare, which is to say on some screens.
func TestSwiftJustifyClaimsFreeSpaceClassifiesEveryJustifyContent(t *testing.T) {
	claims := map[string]bool{}
	for _, value := range stringArray(t, swiftFlex, "var justifyClaimsFreeSpace: Bool {") {
		if claims[value] {
			t.Errorf("GrMobFlex.swift: justifyClaimsFreeSpace lists %q twice", value)
		}
		claims[value] = true
	}

	for _, value := range justifyValues() {
		switch {
		case value == string(core.JustifyStart):
			if claims[value] {
				t.Errorf("GrMobFlex.swift: justifyClaimsFreeSpace lists %q, but packing to the "+
					"leading edge looks the same at any container size — listing it makes every "+
					"default-aligned container fill instead of hug", value)
			}
		case !claims[value]:
			t.Errorf("GrMobFlex.swift: justifyClaimsFreeSpace omits %q, so a container using it "+
				"hugs its children and has no leftover space to distribute — the alignment is "+
				"then computed correctly over a container that is the wrong size", value)
		}
		delete(claims, value)
	}
	for value := range claims {
		t.Errorf("GrMobFlex.swift: justifyClaimsFreeSpace lists %q, which is not a "+
			"core.JustifyContent", value)
	}
}

// Compose's two Arrangement dispatches. Unlike the iOS solver these each
// answer the whole question for one axis, so both must cover all six.
func TestKotlinArrangementsCoverEveryJustifyContent(t *testing.T) {
	for _, fn := range []string{"fun horizontalArrangement(", "fun verticalArrangement("} {
		syntax := kotlinWhen.with(kotlinRenderer, fn, "when (s?.justifyContent) {")
		coverage{
			file:        "Renderer.kt",
			fn:          fn,
			required:    justifyValues(),
			consequence: "packs the children to the start edge",
		}.check(t, syntax.labels(t))
	}
}

// --- align-items ------------------------------------------------------------

// The solver's cross-axis offset, used by every Row and Column on iOS.
func TestSwiftCrossOffsetCoversEveryAlignItems(t *testing.T) {
	syntax := swiftSwitch.with(swiftFlex, "static func crossOffset(align:", "switch align {")
	coverage{
		file:        "GrMobFlex.swift",
		fn:          "crossOffset",
		required:    alignItemsValues(),
		allowed:     crossAxisFallback,
		consequence: "places the child at the leading edge of the cross axis",
	}.check(t, syntax.labels(t))
}

// The LazyVStack's native alignment, the one cross-axis dispatch on iOS that
// is not the solver — a lazy stack cannot be replaced by a custom Layout
// without giving up laziness.
func TestSwiftCrossAlignmentCoversEveryAlignItems(t *testing.T) {
	syntax := swiftSwitch.with(swiftRenderer, "func crossAlignmentH(", "switch v {")
	coverage{
		file:        "Renderer.swift",
		fn:          "crossAlignmentH",
		required:    alignItemsValues(),
		allowed:     crossAxisFallback,
		consequence: "aligns the list's rows to the leading edge",
	}.check(t, syntax.labels(t))
}

// GrMobRow is the one cross-axis dispatch with no Style.Align fallback, so it
// gets no `allowed` set: a bare "start" or "end" arm here would be a value
// that cannot reach it and is therefore dead.
//
// That is not an oversight being frozen in. Align is a text-alignment concept
// and has never been read for a Row's vertical axis, and Renderer.swift draws
// the same line — GrMobFlexStack consults the fallback only when the axis is
// vertical. Encoding the asymmetry is what would make a future change to
// either native, without the other, fail here.
func TestKotlinRowAlignmentCoversEveryAlignItems(t *testing.T) {
	syntax := kotlinWhen.with(kotlinRenderer, "fun GrMobRow(", "when (s?.alignItems) {")
	coverage{
		file:        "Renderer.kt",
		fn:          "GrMobRow's verticalAlignment",
		required:    alignItemsValues(),
		consequence: "aligns the children to the top of the row",
	}.check(t, syntax.labels(t))
}

func TestKotlinColumnAlignmentCoversEveryAlignItems(t *testing.T) {
	syntax := kotlinWhen.with(kotlinRenderer, "fun GrMobColumn(",
		"when (s?.alignItems?.ifEmpty { s.align }) {")
	coverage{
		file:        "Renderer.kt",
		fn:          "GrMobColumn's horizontalAlignment",
		required:    alignItemsValues(),
		allowed:     crossAxisFallback,
		consequence: "aligns the children to the start edge",
	}.check(t, syntax.labels(t))
}

// The lazy sibling of the Column check. Anchored on GrMobList so the identical
// `when` header two hundred lines above it is not the one read — the anchor is
// the only thing telling these two apart.
func TestKotlinListAlignmentCoversEveryAlignItems(t *testing.T) {
	syntax := kotlinWhen.with(kotlinRenderer, "fun GrMobList(",
		"when (s?.alignItems?.ifEmpty { s.align }) {")
	coverage{
		file:        "Renderer.kt",
		fn:          "GrMobList's horizontalAlignment",
		required:    alignItemsValues(),
		allowed:     crossAxisFallback,
		consequence: "aligns the list's rows to the start edge",
	}.check(t, syntax.labels(t))
}

// --- the stretch fill bindings ----------------------------------------------

// Every check above reads a dispatch's arms, and stretch is the one value no
// dispatch can finish the story for: a stretched child is not *placed* but
// *measured*, so each List implements it off the dispatch as an equality test
// feeding a fill modifier — and an equality has no arms to hold to a list.
//
// That blind spot held a real divergence, on both natives at once. Each List's
// placement dispatch reads AlignItems with the Style.Align fallback, and its
// "stretch" arm says the fill modifier handles the rest; each fill binding
// tested AlignItems alone. Align: stretch with AlignItems unset therefore took
// the "stretch" arm's word for a fill that never happened — rows placed at the
// start edge, stretched nowhere — while a Column with the identical style
// stretched. The two Lists agreed with each other perfectly, which is exactly
// why nothing comparing renderers could notice; they agreed on the wrong
// answer.
//
// A substring is the strongest hold a test can take on an equality, so the pin
// is two-level, because either level alone is satisfiable by a rename: the
// List's fill binding must read the helper that owns the fallback
// (crossAxisValue on iOS; on Android isColumnStretch — the *column* spelling,
// a List's cross axis being horizontal like a Column's), and that helper must
// itself still read Align. The substrings are expression fragments rather than
// names so that a comment mentioning the helper cannot satisfy them.
func TestListStretchFillReadsTheAlignFallback(t *testing.T) {
	for _, pin := range []struct {
		file string
		// list anchors the List's declaration, which holds the fill binding;
		// helperCall is the fallback-aware read that binding must make.
		list, helperCall string
		// helper anchors the helper's declaration; fallbackRead is the Align
		// expression it must still contain.
		helper, fallbackRead string
	}{
		{swiftRenderer, "struct GrMobList", "crossAxisValue(s)",
			"func crossAxisValue(", `s?.align ?? ""`},
		{kotlinRenderer, "fun GrMobList(", "isColumnStretch(s)",
			"fun isColumnStretch(", "ifEmpty { s.align }"},

		// Column has exactly the same shape, and iOS had exactly the same
		// bug: FlexChildren — the binding that decides whether a child
		// *accepts* the stretched size GrMobFlexStack proposes — read
		// alignItems alone, so a Column written Align(AlignStretch) with no
		// AlignItems laid out stretched and rendered unstretched. Compose's
		// ColumnChildren had carried the fallback (isColumnStretch) since the
		// List fix; this row is the pair being held together from now on.
		{swiftRenderer, "struct FlexChildren", "crossAxisValue(node.style)",
			"func crossAxisValue(", `s?.align ?? ""`},
		{kotlinRenderer, "fun ColumnScope.ColumnChildren(", "isColumnStretch(node.style)",
			"fun isColumnStretch(", "ifEmpty { s.align }"},
	} {
		if !strings.Contains(declSource(t, pin.file, pin.list), pin.helperCall) {
			t.Errorf("%s: %s's fill binding does not read %s — the stretch equality has come apart "+
				"from the placement dispatch again, so Align: stretch with AlignItems unset places "+
				"the rows at the start edge and fills nothing", pin.file, pin.list, pin.helperCall)
		}
		if !strings.Contains(declSource(t, pin.file, pin.helper), pin.fallbackRead) {
			t.Errorf("%s: %s no longer reads %s — the Style.Align fallback is gone from the helper "+
				"the fill binding relies on, so Align: stretch with AlignItems unset stretches "+
				"nothing while the placement dispatch still claims it is handled",
				pin.file, pin.helper, pin.fallbackRead)
		}
	}
}

// --- text-align -------------------------------------------------------------

// The two native text dispatches, held to core.TextAlignments() — the subset
// of Alignment that names a real text alignment.
//
// This pair is why the whole exercise reached text alignment at all. Before
// core.TextAlignments() existed, core.AlignJustify rendered as justified text
// on Compose, as leading on SwiftUI, as no declaration at all from htmlout,
// and as nothing whatsoever in the WASM runtime, which did not read Style.Align
// in any form. One value, four behaviors, and no list for anything to notice
// against.
//
// SwiftUI still cannot justify text — TextAlignment has three members — so
// Renderer.swift's "justify" arm falls back to leading on purpose. Coverage
// here means the target has *said* what it does with a value, not that it can
// honor it; an explicit arm naming the platform limit is an answer, and silence
// is not.
func TestSwiftTextAlignmentCoversEveryTextAlignment(t *testing.T) {
	syntax := swiftSwitch.with(swiftRenderer, "func grMobTextAlignment(", "switch align {")
	coverage{
		file:        "Renderer.swift",
		fn:          "grMobTextAlignment",
		required:    textAlignValues(),
		consequence: "renders leading, while htmlout and the WASM runtime emit the CSS keyword",
	}.check(t, syntax.labels(t))
}

func TestKotlinTextAlignCoversEveryTextAlignment(t *testing.T) {
	syntax := kotlinWhen.with(kotlinRenderer, "fun textStyle(", "when (s.align) {")
	coverage{
		file:        "Renderer.kt",
		fn:          "textStyle's textAlign",
		required:    textAlignValues(),
		consequence: "renders Start, while htmlout and the WASM runtime emit the CSS keyword",
	}.check(t, syntax.labels(t))
}
