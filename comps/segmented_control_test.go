package comps

import (
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/htmlout"
)

var segLabels = []string{"All", "Active", "Done"}

func renderSeg(t *testing.T, theme *core.Theme, s SegmentedControl) (*core.Node, *core.Context) {
	t.Helper()
	ctx := core.NewContext().WithTheme(theme)
	ctx.BeginRenderPass()
	return s.Render(ctx), ctx
}

// Segments must be *direct* children of the gapped row. A grouping wrapper
// around them — which is what core.For would introduce — becomes the row's
// single flex item and the gap then has nothing to space, the exact defect
// the htmlout grouping-node fix addressed one layer down.
func TestSegmentedControlSegmentsAreDirectChildrenOfTheRow(t *testing.T) {
	root, _ := renderSeg(t, core.DefaultTheme, SegmentedControl{Labels: segLabels})

	if root.Type != "Row" {
		t.Fatalf("root = %s, want Row: %s", root.Type, describe(root))
	}
	if len(root.Children) != 3 {
		t.Fatalf("want 3 direct children, got %d: %s", len(root.Children), describe(root))
	}
	for i, c := range root.Children {
		if c.Type != "Button" {
			t.Fatalf("child %d is %s, want Button (a Chip renders as one): %s", i, c.Type, describe(root))
		}
		if c.Props["label"] != segLabels[i] {
			t.Errorf("child %d label = %v, want %q", i, c.Props["label"], segLabels[i])
		}
	}
}

// Exactly one segment carries the selected treatment, and it is the one the
// index names. Selection is read off the rendered style rather than off the
// struct, so a control that computed the flag but dropped it would fail.
func TestSegmentedControlSelectsExactlyTheNamedIndex(t *testing.T) {
	sel := []core.StyleProp{core.BackgroundColor("#SELECTED")}

	for want := range segLabels {
		root, _ := renderSeg(t, core.DefaultTheme, SegmentedControl{
			Labels:   segLabels,
			Selected: want,
			Segment:  Chip{SelectedStyle: sel},
		})
		for i, c := range root.Children {
			isSel := c.Style.Background == "#SELECTED"
			if isSel != (i == want) {
				t.Errorf("Selected=%d: segment %d selected=%v, want %v", want, i, isSel, i == want)
			}
		}
	}
}

// An out-of-range Selected is a legal "nothing chosen" state, not something
// to clamp: a scope picker that starts with no scope says so with -1 rather
// than by growing a fourth segment.
func TestSegmentedControlOutOfRangeSelectsNothing(t *testing.T) {
	sel := []core.StyleProp{core.BackgroundColor("#SELECTED")}

	for _, idx := range []int{-1, 3, 99} {
		root, _ := renderSeg(t, core.DefaultTheme, SegmentedControl{
			Labels:   segLabels,
			Selected: idx,
			Segment:  Chip{SelectedStyle: sel},
		})
		for i, c := range root.Children {
			if c.Style.Background == "#SELECTED" {
				t.Errorf("Selected=%d still selected segment %d", idx, i)
			}
		}
	}
}

// Each segment must report its *own* index. This is the classic closure-over-
// the-loop-variable bug: a control that captured the variable rather than the
// value would report the last index from every segment.
func TestSegmentedControlReportsTheTappedIndex(t *testing.T) {
	got := []int{}
	root, ctx := renderSeg(t, core.DefaultTheme, SegmentedControl{
		Labels:   segLabels,
		OnSelect: func(i int) { got = append(got, i) },
	})

	for _, c := range root.Children {
		ctx.TriggerCallback(c.Props["onClick"].(string))
	}
	if len(got) != 3 || got[0] != 0 || got[1] != 1 || got[2] != 2 {
		t.Fatalf("tapping each segment reported %v, want [0 1 2]", got)
	}
}

// Keys are invisible in exported HTML but drive reconciler matching and native
// view recycling, so they are pinned here rather than left to a markup diff.
func TestSegmentedControlKeysEachSegment(t *testing.T) {
	root, _ := renderSeg(t, core.DefaultTheme, SegmentedControl{
		Labels:    segLabels,
		KeyPrefix: "filter-",
	})
	for i, c := range root.Children {
		if want := "filter-" + segLabels[i]; c.Key != want {
			t.Errorf("segment %d key = %q, want %q", i, c.Key, want)
		}
	}

	// Without a prefix the caption is the key, rather than no key at all —
	// an unkeyed sibling list is matched by position.
	root, _ = renderSeg(t, core.DefaultTheme, SegmentedControl{Labels: segLabels})
	for i, c := range root.Children {
		if c.Key != segLabels[i] {
			t.Errorf("segment %d key = %q, want %q", i, c.Key, segLabels[i])
		}
	}
}

// The template's shared fields reach every segment. Without this the widget
// would silently drop the styling that distinguishes one app's bar from
// another's.
func TestSegmentedControlAppliesTheSegmentTemplate(t *testing.T) {
	root, _ := renderSeg(t, core.DefaultTheme, SegmentedControl{
		Labels:   segLabels,
		Selected: 1,
		Segment: Chip{
			Style:             []core.StyleProp{core.FontSize(13)},
			SelectedStyle:     []core.StyleProp{core.BackgroundColor("#E8F0FE")},
			AccessibilityHint: "Filters the task list",
		},
	})

	for i, c := range root.Children {
		if c.Style.FontSize != 13 {
			t.Errorf("segment %d lost the template Style: FontSize = %v", i, c.Style.FontSize)
		}
		if c.Style.AccessibilityHint != "Filters the task list" {
			t.Errorf("segment %d lost the template hint: %q", i, c.Style.AccessibilityHint)
		}
	}
	if root.Children[1].Style.Background != "#E8F0FE" {
		t.Errorf("SelectedStyle did not reach the selected segment: %q", root.Children[1].Style.Background)
	}
}

// The three fields the control computes are the control's, whatever the
// template says — otherwise a stray Label on the template would caption every
// segment identically and a stray OnTap would swallow every selection.
func TestSegmentedControlTemplateCannotOverrideComputedFields(t *testing.T) {
	stolen := 0
	got := []int{}
	root, ctx := renderSeg(t, core.DefaultTheme, SegmentedControl{
		Labels:   segLabels,
		Selected: 0,
		OnSelect: func(i int) { got = append(got, i) },
		Segment: Chip{
			Label:    "TEMPLATE",
			Selected: true,
			OnTap:    func() { stolen++ },
		},
	})

	for i, c := range root.Children {
		if c.Props["label"] != segLabels[i] {
			t.Errorf("segment %d captioned %v, want %q", i, c.Props["label"], segLabels[i])
		}
		ctx.TriggerCallback(c.Props["onClick"].(string))
	}
	if stolen != 0 {
		t.Errorf("the template's OnTap ran %d times; it must be overwritten", stolen)
	}
	if len(got) != 3 {
		t.Errorf("OnSelect ran %d times, want 3", len(got))
	}
}

// The accessibility name is derived per segment, and the selection is
// announced beside it as the control state it is — so a derivation that
// renames a segment cannot take the state with it.
//
// The two used to be one string (Chip appended ", selected" to whichever name
// it ended up with), which made every segment's name change on every tap.
func TestSegmentedControlDerivesPerSegmentAccessibilityLabels(t *testing.T) {
	root, _ := renderSeg(t, core.DefaultTheme, SegmentedControl{
		Labels:   segLabels,
		Selected: 1,
		SegmentLabel: func(label string, _ int) string {
			return "Show " + strings.ToLower(label) + " tasks"
		},
	})

	want := []string{"Show all tasks", "Show active tasks", "Show done tasks"}
	wantState := []core.SelectedState{core.SelectedOff, core.SelectedOn, core.SelectedOff}
	for i, c := range root.Children {
		if c.Style.AccessibilityLabel != want[i] {
			t.Errorf("segment %d announced %q, want %q", i, c.Style.AccessibilityLabel, want[i])
		}
		if c.Style.AccessibilitySelected != wantState[i] {
			t.Errorf("segment %d state = %q, want %q — every segment answers, not just the "+
				"live one", i, c.Style.AccessibilitySelected, wantState[i])
		}
	}
}

// A nil SegmentLabel must leave the template's own label alone rather than
// writing an empty one over it, which would drop the caller's name.
//
// This mattered more before the state moved out of the name: Chip emits the
// label prop only when the name is non-empty, so an overwritten name once
// silenced the selection along with it. The state no longer rides on the
// name, which is checked below — the two failures are now independent, and
// that is the improvement.
func TestSegmentedControlNilSegmentLabelFallsBackToChip(t *testing.T) {
	root, _ := renderSeg(t, core.DefaultTheme, SegmentedControl{
		Labels:   segLabels,
		Selected: 1,
		Segment:  Chip{AccessibilityLabel: "Filter"},
	})

	for i, c := range root.Children {
		if got := c.Style.AccessibilityLabel; got != "Filter" {
			t.Errorf("segment %d announced %q, want the template's %q", i, got, "Filter")
		}
	}
	if got := root.Children[1].Style.AccessibilitySelected; got != core.SelectedOn {
		t.Errorf("selected segment state = %q, want %q", got, core.SelectedOn)
	}
	if got := root.Children[0].Style.AccessibilitySelected; got != core.SelectedOff {
		t.Errorf("unselected segment state = %q, want %q", got, core.SelectedOff)
	}
}

// Same rule as InputRow: the gap is the control's own internal layout, so zero
// means the theme's step. Both bundled themes set SM to 8 — the literal the
// migrated call site used to write — so a hardcoded 8 passes under either;
// this uses a theme where reading the theme and hardcoding disagree.
func TestSegmentedControlGapDefaultsToTheThemeSMStep(t *testing.T) {
	theme := &core.Theme{
		Colors:     core.DefaultTheme.Colors,
		Typography: core.DefaultTheme.Typography,
		Spacing:    core.SpacingScale{XS: 2, SM: 5, MD: 11, LG: 19, XL: 27},
	}

	root, _ := renderSeg(t, theme, SegmentedControl{Labels: segLabels})
	if g := root.Style.Gap; g != 5 {
		t.Fatalf("default Gap = %v, want the theme's SM step 5", g)
	}

	root, _ = renderSeg(t, core.DefaultTheme, SegmentedControl{Labels: segLabels})
	if g := root.Style.Gap; g != 8 {
		t.Fatalf("DefaultTheme: Gap = %v, want 8", g)
	}

	root, _ = renderSeg(t, core.DefaultTheme, SegmentedControl{Labels: segLabels, Gap: 20})
	if g := root.Style.Gap; g != 20 {
		t.Fatalf("explicit Gap = %v, want 20", g)
	}
}

// Style lands after the gap, which is the only way to ask for no spacing at
// all given that the Gap field's zero means the theme's step.
func TestSegmentedControlStyleOverridesTheWidgetsOwnGap(t *testing.T) {
	root, _ := renderSeg(t, core.DefaultTheme, SegmentedControl{
		Labels: segLabels,
		Gap:    20,
		Style:  []core.StyleProp{core.Gap(0), core.Padding(4)},
	})

	if g := root.Style.Gap; g != 0 {
		t.Errorf("Style Gap did not win: got %v, want 0", g)
	}
	if root.Style.Padding.Top != 4 {
		t.Errorf("Style Padding not applied: %+v", root.Style.Padding)
	}
}

// core.Button registers whatever handler it is given, so a nil OnSelect would
// put a nil in the callback registry and crash on the first native tap. The
// control renders inert instead.
func TestSegmentedControlWithoutOnSelectIsInert(t *testing.T) {
	root, ctx := renderSeg(t, core.DefaultTheme, SegmentedControl{Labels: segLabels})
	for _, c := range root.Children {
		ctx.TriggerCallback(c.Props["onClick"].(string))
	}
}

// A data-driven control legitimately has nothing to show on its first pass.
func TestSegmentedControlWithNoLabelsRendersAnEmptyRow(t *testing.T) {
	root, _ := renderSeg(t, core.DefaultTheme, SegmentedControl{})
	if root.Type != "Row" || len(root.Children) != 0 {
		t.Fatalf("want an empty Row, got %s", describe(root))
	}
	if root.Style.Gap != 8 {
		t.Errorf("props still apply with no segments: Gap = %v, want 8", root.Style.Gap)
	}
}

// A caller can turn the row into a tab strip with two props and no new field,
// and the state each segment already sets follows the role it is given.
//
// This is the payoff of putting the state on Style and the attribute choice in
// the exporters: neither SegmentedControl nor Chip knows which arrangement it
// is in, and the same Selected bool comes out as aria-pressed in a filter bar
// and aria-selected in a tab strip. Pinned end to end — through the widgets in
// Go and out through htmlout — because the two halves are in different
// packages and a change to either would leave the other looking correct.
func TestSegmentedControlBecomesATabStripByRoleAlone(t *testing.T) {
	root, _ := renderSeg(t, core.DefaultTheme, SegmentedControl{
		Labels:   segLabels,
		Selected: 1,
		Style:    []core.StyleProp{core.AccessibilityRole(core.RoleTabList)},
		Segment:  Chip{Style: []core.StyleProp{core.AccessibilityRole(core.RoleTab)}},
	})

	if got := root.Style.AccessibilityRole; got != core.RoleTabList {
		t.Errorf("row role = %q, want %q", got, core.RoleTabList)
	}
	for i, c := range root.Children {
		if got := c.Style.AccessibilityRole; got != core.RoleTab {
			t.Errorf("segment %d role = %q, want %q", i, got, core.RoleTab)
		}
	}

	// The exporter's half: the same state field becomes the other attribute.
	// A tablist whose tabs said aria-pressed would announce three toggle
	// buttons inside a tab strip, which is a shape no reader has a story for.
	out := htmlout.ExportHTML(root)
	if strings.Count(out, `aria-selected="false"`) != 2 ||
		!strings.Contains(out, `aria-selected="true"`) {
		t.Errorf("want one selected tab and two unselected:\n%s", out)
	}
	if strings.Contains(out, "aria-pressed") {
		t.Errorf("a tab is selected, not pressed:\n%s", out)
	}
}
