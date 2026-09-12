package comps

import (
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

// appendRows' parameters used to be twelve positional arguments shared by two
// widgets, which had two consequences worth pinning now that they are a
// struct.
//
// The first is transposition: hideTrailingCount and sticky sat next to each
// other, both bool, and swapping them compiled. That one was survivable
// because each flag happens to have a test of its own in each widget.
//
// The second is the one nothing guarded, and it is what these tests are for:
// a knob wired at one call site and forgotten at the other. Positional
// arguments make that shape of edit easy — DataTable's call passed its
// twelve on three wrapped lines, and a reader checking them against
// GroupedList's was counting commas. DataTable.HeadingLevel was in fact
// never asserted anywhere before this file.

// wantRowsSpecFields is the census. It exists so that adding a knob to
// rowsSpec fails here rather than silently reaching one widget, and the
// failure message says what else to go and do.
//
// # The third column, and why it had to be added
//
// Most of these are forwarded by both widgets and are checked together by
// TestBothWidgetsForwardEveryRowsSpecKnob. Two are not, and the count is now
// the point: `Wrap` has always been DataTable's alone, and `Collapse` is
// GroupedList's. Each single-owner field needs its own test, because the
// shared one cannot assert an effect the other widget has no way to produce.
//
// Recording the owner per field rather than leaving the exceptions implicit
// is the smallest honest fix. The alternative was a census whose failure
// message says "forward it from both" when the right answer is sometimes
// "forward it from one and say why here", which is advice that gets followed.
//
// # When a single-owner field is admissible
//
// The open question the count below was watching — does a widget-specific
// knob belong in a shared spec at all, or should the decoration move into a
// wrapper around appendRows' output — has an answer, and it is a property of
// the output rather than a matter of taste.
//
// appendRows' output is opaque. Every child it appends is a
// core.ComponentFunc: core.Keyed wraps its argument in a closure that sets
// the key at render time, so a band, a separator and a row come back as the
// same dynamic type, with the key sealed inside until somebody renders it.
// A wrapper around that slice cannot pick a row out of it, cannot read a key
// without rendering the child (which would then be rendered again by the
// container), and has no route back to the item a row was built from.
// TestAppendRowsOutputIsOpaqueToItsCaller is that fact as an assertion.
//
// So the admission test is: can this knob be done to appendRows' result?
//
//	no  -> it belongs in the spec, single-owner or not
//	yes -> it belongs in a wrapper the widget applies itself, because the
//	       wrapper needs nothing appendRows has
//
// All three current fields fail the wrapper test, each for its own reason,
// and the three reasons are one per direction the loop has an edge on — its
// item, its output, and its input:
//
//	Wrap              needs the item. A row's T is captured inside the closure
//	                  and is not recoverable from it, and DataTable's tap
//	                  handler and selection tint are both functions of the row.
//	Collapse          needs rows *not to be produced*. No pass over produced
//	                  children can undo their production — hiding them is a
//	                  different thing, with a different reconciler cost and a
//	                  different export.
//	AutoLoadWithheld  needs to reach a band's *input*. It is stamped onto the
//	                  Group before the Header override or the default band is
//	                  built, and both consume that Group and return an opaque
//	                  child; a wrapper over the output is handed the view, and
//	                  the argument it was built from is gone.
//
// That third reason is the one worth stating separately, because it is the
// only one that is not about rows at all. Wrap and Collapse are decisions
// about children; this is a fact travelling to a view the *caller* wrote, and
// the loop is simply the last place that fact still exists.
//
// A fourth field is fine if it fails the test too, and is a mistake if it
// passes. What the count below is for is making somebody run the test.
// The fourth column, feeds, is what makes the third one checkable. An owner
// is a claim about the two widgets, and the claim is only true while exactly
// one of them has the fields that supply the knob — so the census names those
// fields and TestEachSingleWidgetKnobIsReallySingleWidget derives the owner
// from the widgets rather than trusting the string.
//
// Empty for a shared field, deliberately: a "both" row is checked by its entry
// in sharedKnobEffects, which asserts an effect rather than a field, and
// several of them have no matching widget field to name (GroupedList spells
// Rows as Items, DataTable synthesizes Row from Columns).
//
// # What each of the two claims rests on
//
// The two columns are checked by opposite mechanisms, and it is worth being
// explicit about which, because a reader who assumes one covers the other is
// making the mistake this census was built twice to stop:
//
//	owner "X"     falsified against the widgets. feeds names the fields that
//	              supply the knob and TestEachSingleWidgetKnobIsReallySingle
//	              Widget computes who has them, so the string is derived
//	              rather than trusted.
//	owner "both"  falsified against the *rendered tree*. There is nothing to
//	              derive — a shared field is one both widgets forward, which is
//	              a fact about two call sites and not about either struct — so
//	              the claim is only as good as an assertion that fails when one
//	              of the two stops forwarding it. TestEverySharedKnobHasItsOwn
//	              EffectAssertion requires one per row.
var wantRowsSpecFields = []struct {
	name  string
	kind  reflect.Kind
	owner string   // "both", or the single widget that forwards it
	feeds []string // the owner's own fields that supply it; empty when shared
}{
	{"Rows", reflect.Slice, "both", nil},
	{"Key", reflect.Func, "both", nil},
	{"Row", reflect.Func, "both", nil},
	{"GroupBy", reflect.Func, "both", nil},
	{"Header", reflect.Func, "both", nil},
	{"HideTrailingCount", reflect.Bool, "both", nil},
	{"StickyHeaders", reflect.Bool, "both", nil},
	{"HeadingLevel", reflect.Int, "both", nil},
	{"Dividers", reflect.Bool, "both", nil},
	// GroupedList's alone, and the one field here that is an *answer* rather
	// than a knob: the list has already decided whether it is attaching its
	// edge sensor, and this carries that decision down to the run it is about
	// so a Header override can read it off the Group. See
	// GroupedList.AutoLoadWithheld and Group.AutoLoadWithheld.
	//
	// A table has no OnEndReached, so there is no sensor for a shut run to
	// withhold and nothing for DataTable to forward. The two fields named
	// below are what compose the answer and are GroupedList's alone; Items and
	// GroupBy go into it too and are not named, because DataTable has rows and
	// a GroupBy of its own and naming them would make the owner underivable.
	//
	// Fails the admission test on the third count: the Group it stamps is
	// consumed while the children are built, so no pass over the returned
	// children can put it there.
	{"AutoLoadWithheld", reflect.Bool, "GroupedList", []string{"OnEndReached", "Collapse"}},
	// GroupedList's alone. A DataTable band sits inside the body's rowgroup,
	// where ARIA has no reading for it even as a plain heading (see
	// DataTable.Render's note); making it a button would be a second claim on
	// a structure that is already the wrong shape. The fix is per-band
	// rowgroups, which is a change to this function that both widgets share
	// and so not one to make on the way past.
	//
	// Fails the admission test on the second count: it decides that rows are
	// not produced, which no pass over produced children can undo.
	{"Collapse", reflect.Struct, "GroupedList", []string{"Collapse"}},
	// DataTable's alone: the tap target and the selection tint, which a
	// GroupedList row (the caller's own view, emitted as it came back) has no
	// use for. It is the one spec field with no field of its own name behind
	// it, so the two it is assembled from are what identify the owner.
	//
	// Fails the admission test on the first count: both are functions of the
	// row, and the row is captured inside a closure the output does not expose.
	{"Wrap", reflect.Func, "DataTable", []string{"OnRowTap", "Selected"}},
}

func TestRowsSpecCensus(t *testing.T) {
	rt := reflect.TypeOf(rowsSpec[sermon]{})
	if rt.NumField() != len(wantRowsSpecFields) {
		t.Fatalf("rowsSpec has %d fields, the census lists %d — a new knob must be added "+
			"here with its owner, and either forwarded by BOTH GroupedList.Render and "+
			"DataTable.Render with an entry of its own in sharedKnobEffects, or "+
			"forwarded by one with a test of its own and the reason recorded above",
			rt.NumField(), len(wantRowsSpecFields))
	}
	for i, w := range wantRowsSpecFields {
		f := rt.Field(i)
		if f.Name != w.name || f.Type.Kind() != w.kind {
			t.Errorf("field %d = %s %s, want %s (%s)", i, f.Name, f.Type.Kind(), w.name, w.kind)
		}
	}
}

// A widget-specific knob in a shared spec is a trade, and the census records
// the two it has made. A third is where somebody has to stop and apply the
// admission test, because each individual addition looks like the two before
// it and nothing else in the package would say otherwise.
//
// A count rather than a ban, and the threshold is a prompt rather than a
// limit: three is the state of the world and is fine, and a fourth that fails
// the wrapper test is fine too — it is added here with its reason and the
// number below goes up by one. What must not happen quietly is one that
// *passes* the wrapper test, because that one is a decoration sitting in a
// shared parameter list for no reason but proximity.
//
// The number has moved once, for AutoLoadWithheld, and the move is what the
// prompt is for rather than a failure of it: the field was written, the census
// failed, and the admission test turned up a reason the two existing entries
// did not cover.
func TestRowsSpecHasNotAccumulatedMoreSingleWidgetKnobs(t *testing.T) {
	var single []string
	for _, w := range wantRowsSpecFields {
		if w.owner != "both" {
			single = append(single, w.name+" ("+w.owner+")")
		}
	}
	if len(single) > 3 {
		t.Errorf("rowsSpec now carries %d single-widget knobs (%s) — apply the admission "+
			"test on wantRowsSpecFields: can the new one be done to appendRows' "+
			"returned slice instead? If yes it is a wrapper the widget applies to "+
			"that slice, not a spec field. If no, record which of the three things it "+
			"needs (the item, rows not being produced, or a band's input) and raise "+
			"this number",
			len(single), strings.Join(single, ", "))
	}
}

// The output a wrapper would have to work on, asserted rather than argued.
//
// This is the evidence under the admission rule on wantRowsSpecFields. Three
// separate things stand between a caller and a row, and each is one of the
// reasons a per-row knob cannot live outside the loop:
//
//   - what comes back is not a list of children. appendRows appends into the
//     container's own argument list and returns it, so the slice is the
//     caller's props followed by the rows — which is the shape both widgets
//     really have, and is why this test hands it a prefix rather than nil;
//   - a band, a separator and a row are the same dynamic type, so nothing
//     type-switches a row out of the part that is children;
//   - that type is a closure with no fields, so neither the key nor the item
//     is reachable from the value — the key is assigned during Render, which
//     means even prefix-sniffing costs a render of a child the container is
//     about to render itself.
//
// The first leg is what makes the other two careful rather than obvious. A
// core prop is a closure too — styleFunc is a func(*Style), behaviorFunc a
// func(*Context, *Node) — so "it is a func" identifies nothing, and a wrapper
// that reached for reflect.Kind first would take the container's Padding for
// a row. The one sound separator is the core.View assertion, so that is
// asserted in both directions before the children are looked at.
//
// Together those are why Wrap is a spec field rather than something DataTable
// does to the slice it gets back.
func TestAppendRowsOutputIsOpaqueToItsCaller(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()

	// The prefix GroupedList.Render has already appended by the time it calls
	// appendRows: the two theme defaults it sheds, the edge sensor, and a
	// caller's own Style prop. This is what a wrapper would be handed.
	prefix := []core.PropsAndChildren{
		core.Padding(0),
		core.Gap(0),
		core.OnEndReached(func() {}),
		core.BackgroundColor("#FFFFFF"),
	}

	// A grouped, divided run, so the slice holds all three kinds of child:
	// three bands, five rows and two separators.
	out := appendRows(ctx, append([]core.PropsAndChildren(nil), prefix...), rowsSpec[sermon]{
		Rows: sermons, Key: sermonKey, Row: sermonRow, GroupBy: byMonth,
		Dividers: true,
	})
	if len(out) != len(prefix)+10 {
		t.Fatalf("appendRows returned %d items, want %d (%d props, then 3 bands, 5 rows "+
			"and 2 separators) — the fixture this test reasons about has changed",
			len(out), len(prefix)+10, len(prefix))
	}

	// The props are still in front of the children, and they are closures as
	// well — which is the whole of why the filter below has to be an interface
	// assertion and not a Kind test.
	var propFuncs int
	for i, p := range out[:len(prefix)] {
		if _, isView := p.(core.View); isView {
			t.Fatalf("prop %d in the returned slice satisfies core.View — the assertion "+
				"below is the only way a wrapper can find the children among the props, "+
				"and it has stopped separating them", i)
		}
		if reflect.TypeOf(p).Kind() == reflect.Func {
			propFuncs++
		}
	}
	if propFuncs == 0 {
		t.Errorf("no prop in the prefix is a closure — this test's first leg is that " +
			"reflect.Kind cannot tell a core prop from a keyed child, and the fixture " +
			"no longer demonstrates it")
	}

	// So the filter is the interface, and it finds exactly the ten appended
	// children and nothing the caller put there itself.
	var children []core.PropsAndChildren
	for _, item := range out {
		if _, ok := item.(core.View); ok {
			children = append(children, item)
		}
	}
	if len(children) != 10 {
		t.Fatalf("core.View picks %d items out of the returned slice, want the 10 "+
			"children appendRows appended", len(children))
	}

	// One type for all ten. A wrapper cannot ask "is this a row".
	kinds := map[reflect.Type]int{}
	for _, c := range children {
		kinds[reflect.TypeOf(c)]++
	}
	if len(kinds) != 1 {
		t.Fatalf("appendRows' children have %d distinct types %v — if a row is now "+
			"distinguishable from a band by type, the admission rule on "+
			"wantRowsSpecFields has lost its second leg and a per-row wrapper around "+
			"this slice may have become possible", len(kinds), kinds)
	}

	// And that one type is a bare function value: no fields, so no key and no
	// item are reachable without calling it. reflect is the strongest
	// available form of "a caller of appendRows has nothing to go on".
	var only reflect.Type
	for k := range kinds {
		only = k
	}
	if only.Kind() != reflect.Func {
		t.Errorf("appendRows' children are %s, not a closure — core.Keyed's shape is "+
			"what seals the key and the item away from a wrapper, and the admission "+
			"rule on wantRowsSpecFields rests on it", only.Kind())
	}
}

// "Single-owner" is a claim about the two widgets, not a note in a comment,
// so it is checked against them.
//
// The census's third column says Wrap is DataTable's and Collapse is
// GroupedList's. That is only true while the other widget has no field that
// could reach the knob — and the way it would stop being true is the ordinary
// one: somebody adds Collapse to DataTable because a caller asked, forwards
// it, and leaves the census saying GroupedList. The shared-forwarding test
// cannot catch that, because it only asserts what *is* forwarded.
func TestEachSingleWidgetKnobIsReallySingleWidget(t *testing.T) {
	widgets := map[string]reflect.Type{
		"GroupedList": reflect.TypeOf(GroupedList[sermon]{}),
		"DataTable":   reflect.TypeOf(DataTable[sermon]{}),
	}
	hasAll := func(rt reflect.Type, names []string) bool {
		for _, n := range names {
			if _, ok := rt.FieldByName(n); !ok {
				return false
			}
		}
		return true
	}

	for _, w := range wantRowsSpecFields {
		// A spec field that no widget spells the same way cannot be checked
		// against either of them by name, so it must say what supplies it.
		// Wrap is the only such field and this is what keeps it from being
		// downgraded: recording it as shared means deleting its feeds, and a
		// shared row with no feeds is a row nothing verifies.
		//
		// The break-test that found this flipped Wrap to "both" with a nil
		// feeds and passed the whole suite — the count came down by one, the
		// derived-owner check below skipped it as shared, and
		// TestBothWidgetsForwardEveryRowsSpecKnob asserts a tap target on the
		// table only.
		namedSomewhere := false
		for _, rt := range widgets {
			if _, ok := rt.FieldByName(w.name); ok {
				namedSomewhere = true
			}
		}
		if !namedSomewhere && len(w.feeds) == 0 {
			t.Errorf("%s is a spec field neither widget has by name, and it names no "+
				"feeding field either — there is nothing left to check its owner "+
				"column against", w.name)
			continue
		}

		if w.owner == "both" {
			if len(w.feeds) != 0 {
				t.Errorf("%s is recorded as shared but names feeding fields %v — feeds is "+
					"how a single owner is checked, and a shared field's effect is "+
					"asserted by TestBothWidgetsForwardEveryRowsSpecKnob instead",
					w.name, w.feeds)
			}
			continue
		}
		if len(w.feeds) == 0 {
			t.Errorf("%s is recorded as %s's alone but names no feeding field — without "+
				"one the owner column is a string nothing checks", w.name, w.owner)
			continue
		}

		// Derive the owner from the widgets, then compare. This is the whole
		// point of the column: "Wrap is DataTable's" stops being true the
		// moment GroupedList grows an OnRowTap, and the ordinary way that
		// happens is a caller asking for it and nobody revisiting this file.
		var owners []string
		for name, rt := range widgets {
			if hasAll(rt, w.feeds) {
				owners = append(owners, name)
			}
		}
		sort.Strings(owners)

		switch {
		case len(owners) == 0:
			t.Errorf("%s is recorded as %s's, but no widget has all of %v — the fields "+
				"that supply this knob have been renamed or removed",
				w.name, w.owner, w.feeds)
		case len(owners) > 1:
			t.Errorf("%s is recorded as %s's, but %v all have %v — it has a second "+
				"owner now, so the census should say \"both\", the effect should be "+
				"asserted in TestBothWidgetsForwardEveryRowsSpecKnob, and the "+
				"single-widget count should come down",
				w.name, w.owner, owners, w.feeds)
		case owners[0] != w.owner:
			t.Errorf("%s is recorded as %s's, but the widget carrying %v is %s",
				w.name, w.owner, w.feeds, owners[0])
		}
	}
}

// The shared knobs, one effect assertion each, run against both widgets.
//
// # Why a table and not a function
//
// This used to be one `check` closure asserting nine knobs in one pass, and it
// worked in the direction it was written for: a call site that forwards eight
// of nine fails. What it could not do is answer for a *tenth*. The census
// requires a new spec field to have a row in wantRowsSpecFields; nothing
// required it to have an assertion here, so a knob added, marked "both",
// forwarded by one widget and never asserted passed everything — and the
// owner column, which is the other half of the census, is only checkable
// against the widgets for a *single*-owner field (see feeds, and
// TestEachSingleWidgetKnobIsReallySingleWidget). "Shared" was the claim with
// nothing behind it.
//
// So the effects are a table keyed by field name, and
// TestEverySharedKnobHasItsOwnEffectAssertion makes the census and the table
// answer for each other in both directions. A tenth shared knob now has two
// places to be added and fails in the one that names the field.
//
// The split also bought two assertions the merged pass did not have. `Row` and
// `Header` had none at all — the row count stood in for `Rows`, `Key` and `Row`
// together, and nothing anywhere rendered a Header override through this path
// — while `Rows` and `Key` are now separable: the row *count* is Rows', and the
// row *keys* are Key's.
//
// # Why some effects render twice
//
// A Header override owns its own band, so it turns off the default band's three
// knobs (HideTrailingCount, StickyHeaders, HeadingLevel) by design — appendRows
// hands the Group to the override and builds no GroupHeader. One render cannot
// therefore witness both, and an effect that needs the other shape asks the
// harness for it rather than the table carrying two parallel node lists.

// rowsSpecHarness renders one widget's fixture in whichever band shape an
// effect needs, and returns the child list appendRows produced.
//
// Both shapes carry every shared knob turned on, so an effect that reads the
// wrong shape still sees a fully-forwarded spec and fails for its own reason
// rather than for the shape's.
type rowsSpecHarness struct {
	widget string
	// defaultBands: the built-in GroupHeader, so the three band knobs are live.
	defaultBands func(t *testing.T) *core.Node
	// overriddenBands: a caller-supplied Header, which owns the band entirely.
	overriddenBands func(t *testing.T) *core.Node
}

// The band label an override writes. Distinct from the default band's
// "Month 2026-01" so that "the override was used" and "the override was
// ignored" are different strings rather than the same one.
func overrideBandLabel(g Group) string { return "override " + g.Key }

// rowsSpecChildren splits a rendered body into the three kinds of child
// appendRows emits. A row is what is left over rather than what matches
// "sermon:", deliberately: Key's own effect is that the keys are the caller's,
// and a classifier that assumed them would make Key's assertion vacuous.
func rowsSpecChildren(body *core.Node) (bands, seps, rows []*core.Node) {
	for _, c := range body.Children {
		switch {
		case strings.HasPrefix(c.Key, "group:"):
			bands = append(bands, c)
		case strings.HasPrefix(c.Key, "sep:"):
			seps = append(seps, c)
		default:
			rows = append(rows, c)
		}
	}
	return bands, seps, rows
}

// sharedKnobEffects is one assertion per shared rowsSpec field, keyed by the
// field's name so the census can require one and the failure can name it.
//
// Each entry asserts the effect the *field* has and nothing else, which is what
// makes a dropped forward attributable. The messages say which field is
// missing rather than what the tree looks like, because that is the only thing
// a reader can act on from here.
var sharedKnobEffects = map[string]func(t *testing.T, h rowsSpecHarness){
	// The item slice reaches the emitter: five items in, five row children out.
	// Counted as "not a band and not a separator" so this stays independent of
	// how the rows are keyed, which is the next field's business.
	"Rows": func(t *testing.T, h rowsSpecHarness) {
		_, _, rows := rowsSpecChildren(h.defaultBands(t))
		if len(rows) != len(sermons) {
			t.Errorf("%s: %d row children, want %d — Rows is not reaching appendRows",
				h.widget, len(rows), len(sermons))
		}
	},

	// The caller's key function is used rather than the positional fallback.
	// This is the assertion the row count used to stand in for and could not
	// make: appendRows keys "row:0" when Key is nil, so a dropped Key emits
	// exactly as many children and reconciles by position.
	"Key": func(t *testing.T, h rowsSpecHarness) {
		_, _, rows := rowsSpecChildren(h.defaultBands(t))
		var got []string
		for _, r := range rows {
			got = append(got, r.Key)
		}
		sort.Strings(got)
		want := []string{"sermon:1", "sermon:2", "sermon:3", "sermon:4", "sermon:5"}
		if strings.Join(got, ",") != strings.Join(want, ",") {
			t.Errorf("%s: row keys %v, want %v — Key not forwarded, so rows fall back to "+
				"positional keys and a reorder remounts every one of them",
				h.widget, got, want)
		}
	},

	// The row drawing is the spec's. GroupedList passes the caller's function
	// and DataTable a closure over its resolved columns, and both put the
	// item's own text in the row — so the assertion is that each row carries
	// its item's title, which no other field can produce.
	"Row": func(t *testing.T, h rowsSpecHarness) {
		_, _, rows := rowsSpecChildren(h.defaultBands(t))
		byKey := map[string]*core.Node{}
		for _, r := range rows {
			byKey[r.Key] = r
		}
		for _, s := range sermons {
			row, ok := byKey[sermonKey(s)]
			if !ok {
				continue // Key's assertion owns that failure
			}
			if findText(row, s.Title) == nil {
				t.Errorf("%s: row %q does not draw %q — Row not forwarded, so the rows are "+
					"not the ones this widget draws", h.widget, row.Key, s.Title)
			}
		}
	},

	// Three runs of equal month, in first-appearance order. Both the count and
	// the keys, since a GroupBy that reached appendRows and returned a constant
	// would give one band and a dropped one gives none.
	"GroupBy": func(t *testing.T, h rowsSpecHarness) {
		bands, _, _ := rowsSpecChildren(h.defaultBands(t))
		want := []string{"group:2026-01", "group:2025-12", "group:2025-11"}
		var got []string
		for _, b := range bands {
			got = append(got, b.Key)
		}
		if strings.Join(got, ",") != strings.Join(want, ",") {
			t.Errorf("%s: bands %v, want %v — GroupBy not forwarded, so the run is flat",
				h.widget, got, want)
		}
	},

	// The override is what draws the band. Asserted on the shape built for it,
	// and from both sides: the override's text is present and the default
	// band's label is not, because a widget that built both would show the
	// first alone if only presence were checked.
	"Header": func(t *testing.T, h rowsSpecHarness) {
		bands, _, _ := rowsSpecChildren(h.overriddenBands(t))
		if len(bands) == 0 {
			t.Fatalf("%s: no bands to check a Header override against", h.widget)
		}
		for _, b := range bands {
			key := strings.TrimPrefix(b.Key, "group:")
			if findText(b, overrideBandLabel(Group{Key: key})) == nil {
				t.Errorf("%s: band %q was not drawn by the override — Header not forwarded",
					h.widget, b.Key)
			}
			if findText(b, "Month "+key) != nil {
				t.Errorf("%s: band %q still carries the default label — Header reached the "+
					"spec but appendRows built a GroupHeader as well", h.widget, b.Key)
			}
		}
	},

	// The trailing run is still open, so publishing its count would state a
	// total that the next page changes. Both halves: the closed bands keep
	// their badge, so a widget that hid every count fails too.
	"HideTrailingCount": func(t *testing.T, h rowsSpecHarness) {
		bands, _, _ := rowsSpecChildren(h.defaultBands(t))
		if len(bands) != 3 {
			t.Fatalf("%s: %d bands, want 3 (GroupBy's assertion owns this)", h.widget, len(bands))
		}
		if findText(bands[0], "2") == nil {
			t.Errorf("%s: a closed band lost its count badge — the flag is hiding every "+
				"count, not the trailing one", h.widget)
		}
		if findText(bands[2], "2") != nil {
			t.Errorf("%s: the open trailing band published a count — HideTrailingCount not "+
				"forwarded", h.widget)
		}
	},

	// The pin goes on the band node itself — the child the list sees — which is
	// why this reads the node's own Style rather than looking for a wrapper.
	"StickyHeaders": func(t *testing.T, h rowsSpecHarness) {
		bands, _, _ := rowsSpecChildren(h.defaultBands(t))
		for _, b := range bands {
			if b.Style == nil || b.Style.Position != core.PositionSticky {
				t.Errorf("%s: band %q is not pinned — StickyHeaders not forwarded",
					h.widget, b.Key)
			}
		}
	},

	// Four, not the default two, so a call site forwarding a zero is
	// distinguishable from one forwarding the field.
	"HeadingLevel": func(t *testing.T, h rowsSpecHarness) {
		bands, _, _ := rowsSpecChildren(h.defaultBands(t))
		if len(bands) == 0 {
			t.Fatalf("%s: no bands to read a heading level from", h.widget)
		}
		label := findText(bands[0], "Month 2026-01")
		if label == nil {
			t.Fatalf("%s: first band has no label", h.widget)
		}
		if label.Style.AccessibilityHeadingLevel != 4 {
			t.Errorf("%s: band heading level = %d, want 4 — HeadingLevel not forwarded",
				h.widget, label.Style.AccessibilityHeadingLevel)
		}
	},

	// Between rows of a run and never after the last one, where a band or the
	// footer follows. Runs of 2, 1, 2 give 1+0+1 — and the position matters as
	// much as the count, so the child list is walked rather than tallied: a
	// separator emitted after every row would also number two if one run were
	// dropped.
	"Dividers": func(t *testing.T, h rowsSpecHarness) {
		body := h.defaultBands(t)
		_, seps, _ := rowsSpecChildren(body)
		if len(seps) != 2 {
			t.Errorf("%s: %d separators, want 2 — Dividers not forwarded",
				h.widget, len(seps))
		}
		for i, c := range body.Children {
			if !strings.HasPrefix(c.Key, "sep:") {
				continue
			}
			if i == len(body.Children)-1 {
				t.Errorf("%s: separator %q is the last child — a run's final row must not "+
					"be followed by one", h.widget, c.Key)
				continue
			}
			if next := body.Children[i+1]; strings.HasPrefix(next.Key, "group:") {
				t.Errorf("%s: separator %q sits between a run's last row and the next band",
					h.widget, c.Key)
			}
		}
	},
}

// The census and the effects table answer for each other.
//
// This is what makes "both" a checkable claim rather than a word in a struct
// literal. A shared field with no effect here is a forward nothing asserts —
// the exact gap that let a knob be wired at one call site and forgotten at the
// other, which is what this whole file exists for. An effect naming a field
// that is not shared is the same mistake from the other end: either the census
// is stale or the assertion is aimed at a single-owner knob, whose own test is
// the one that has to change.
func TestEverySharedKnobHasItsOwnEffectAssertion(t *testing.T) {
	shared := map[string]bool{}
	for _, w := range wantRowsSpecFields {
		if w.owner != "both" {
			// A single-owner knob must NOT have an entry, because the shared
			// runner below drives every entry through both widgets and one of
			// them has no way to produce the effect.
			if _, ok := sharedKnobEffects[w.name]; ok {
				t.Errorf("%s is recorded as %s's alone but has an entry in "+
					"sharedKnobEffects, which runs every effect against both widgets — a "+
					"single-owner knob is asserted by a test of its own",
					w.name, w.owner)
			}
			continue
		}
		shared[w.name] = true
		if _, ok := sharedKnobEffects[w.name]; !ok {
			t.Errorf("%s is recorded as shared and has no effect assertion in "+
				"sharedKnobEffects — \"both\" is then a claim with nothing behind it, "+
				"since the owner column is only checkable against the widgets for a "+
				"single-owner field. Add one that fails when this field alone stops "+
				"being forwarded", w.name)
		}
	}
	for name := range sharedKnobEffects {
		if !shared[name] {
			t.Errorf("sharedKnobEffects has an entry for %q, which is not a field recorded "+
				"as shared in wantRowsSpecFields", name)
		}
	}
}

// Every shared knob, through both widgets, so a call site that forwards eight
// of nine fails on the one it dropped and says which.
//
// The fixture is deliberately the same rows for both: three runs of sermons
// (2/1/2), so a divider count, a band count and a trailing-count check all have
// known answers.
func TestBothWidgetsForwardEveryRowsSpecKnob(t *testing.T) {
	for _, h := range []rowsSpecHarness{
		{
			widget: "GroupedList",
			defaultBands: func(t *testing.T) *core.Node {
				ctx := core.NewContext()
				ctx.BeginRenderPass()
				return GroupedList[sermon]{
					Items: sermons, Key: sermonKey, Row: sermonRow, GroupBy: byMonth,
					HideTrailingCount: true,
					StickyHeaders:     true,
					HeadingLevel:      4,
					Dividers:          true,
				}.Render(ctx)
			},
			overriddenBands: func(t *testing.T) *core.Node {
				ctx := core.NewContext()
				ctx.BeginRenderPass()
				return GroupedList[sermon]{
					Items: sermons, Key: sermonKey, Row: sermonRow, GroupBy: byMonth,
					Header:   func(g Group) core.View { return core.Text(overrideBandLabel(g)) },
					Dividers: true,
				}.Render(ctx)
			},
		},
		{
			widget: "DataTable",
			defaultBands: func(t *testing.T) *core.Node {
				ctx := core.NewContext()
				ctx.BeginRenderPass()
				n := DataTable[sermon]{
					Columns: sermonColumns, Rows: sermons, Key: sermonKey, GroupBy: byMonth,
					HideTrailingCount: true,
					StickyHeaders:     true,
					HeadingLevel:      4,
					Dividers:          true,
				}.Render(ctx)
				_, body := tableParts(t, n)
				return body
			},
			overriddenBands: func(t *testing.T) *core.Node {
				ctx := core.NewContext()
				ctx.BeginRenderPass()
				n := DataTable[sermon]{
					Columns: sermonColumns, Rows: sermons, Key: sermonKey, GroupBy: byMonth,
					Header:   func(g Group) core.View { return core.Text(overrideBandLabel(g)) },
					Dividers: true,
				}.Render(ctx)
				_, body := tableParts(t, n)
				return body
			},
		},
	} {
		// Sorted, so a run that fails on three fields reports them in the same
		// order every time and a diff of two runs is readable.
		names := make([]string, 0, len(sharedKnobEffects))
		for name := range sharedKnobEffects {
			names = append(names, name)
		}
		sort.Strings(names)

		for _, name := range names {
			t.Run(h.widget+"/"+name, func(t *testing.T) {
				sharedKnobEffects[name](t, h)
			})
		}
	}
}

// Wrap is DataTable's alone, so its effect is asserted here rather than in the
// shared table: a GroupedList row is the caller's own view, emitted as it came
// back, and there is nothing for the table's runner to compare against.
//
// It is how d.wrapRow gets around each cell row, and a dropped Wrap loses every
// row's tap target — silently, since a row with no handler renders identically
// until somebody presses it.
func TestDataTableForwardsWrap(t *testing.T) {
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	var tapped sermon
	n := DataTable[sermon]{
		Columns: sermonColumns, Rows: sermons, Key: sermonKey, GroupBy: byMonth,
		OnRowTap: func(s sermon) { tapped = s },
	}.Render(ctx)
	_, body := tableParts(t, n)

	row := findFirst(body, func(c *core.Node) bool { return c.Key == "sermon:1" })
	if row == nil {
		t.Fatal("no keyed row to tap")
	}
	id, ok := row.Props["onClick"].(string)
	if !ok {
		t.Fatal("row carries no tap handler (Wrap not forwarded)")
	}
	ctx.TriggerCallback(id)
	if tapped.ID != 1 {
		t.Errorf("tap reached %+v, want sermon 1", tapped)
	}
}

// The two bools are adjacent in the struct as they were in the parameter
// list, so this pins that they remain distinguishable: each flag on alone
// must produce exactly its own effect and not the other's. A literal that
// crossed them — HideTrailingCount: g.StickyHeaders — reads plausibly and
// would otherwise render a list that pins nothing and hides a count nobody
// asked it to hide.
func TestTheTwoBandFlagsAreNotInterchangeable(t *testing.T) {
	render := func(hide, sticky bool) *core.Node {
		ctx := core.NewContext()
		ctx.BeginRenderPass()
		return GroupedList[sermon]{
			Items: sermons, Key: sermonKey, Row: sermonRow, GroupBy: byMonth,
			HideTrailingCount: hide,
			StickyHeaders:     sticky,
		}.Render(ctx)
	}
	bandsOf := func(n *core.Node) []*core.Node {
		var out []*core.Node
		for _, c := range n.Children {
			if strings.HasPrefix(c.Key, "group:") {
				out = append(out, c)
			}
		}
		return out
	}

	hideOnly := bandsOf(render(true, false))
	if findText(hideOnly[2], "2") != nil {
		t.Error("HideTrailingCount alone did not hide the trailing count")
	}
	for _, b := range hideOnly {
		if b.Style != nil && b.Style.Position == core.PositionSticky {
			t.Error("HideTrailingCount alone pinned a band — the flags are crossed")
		}
	}

	stickyOnly := bandsOf(render(false, true))
	if findText(stickyOnly[2], "2") == nil {
		t.Error("StickyHeaders alone hid the trailing count — the flags are crossed")
	}
	for _, b := range stickyOnly {
		if b.Style == nil || b.Style.Position != core.PositionSticky {
			t.Error("StickyHeaders alone did not pin the bands")
		}
	}
}
