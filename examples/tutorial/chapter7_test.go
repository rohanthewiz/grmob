package tutorial

import (
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

// Chapter 7 demo-liveness tests. Unlike the earlier chapters, most of what
// these lessons teach lands in Node.Style rather than in text or structure —
// prop order, merge semantics, a theme swap, a transition declaration — so
// the assertions here read the decoded Style off actual nodes (see nodeStyle
// in app_test.go) and watch the struct change as the demos are driven.

// styledText finds the Text node with exactly this content and fails if it
// carries no style — the chapter's demos hang their whole argument on the
// styles, so a missing struct is a broken demo, not a soft miss.
func styledText(t *testing.T, root *node, content string) *node {
	t.Helper()
	n := findNode(root, func(n *node) bool {
		return n.Type == "Text" && n.Props["content"] == content
	})
	if n == nil {
		t.Fatalf("no Text node with content %q", content)
	}
	if n.Style == nil {
		t.Fatalf("Text %q carries no style", content)
	}
	return n
}

// demoBoxNode finds a lesson's demo container: the innermost Column that both
// wears a background and contains the marker text. Ancestors (the demo panel,
// the lesson column) carry no background, so the background test alone
// excludes them — but the marker keeps the finder honest if that changes.
func demoBoxNode(t *testing.T, root *node, marker string) *node {
	t.Helper()
	n := findNode(root, func(n *node) bool {
		return n.Type == "Column" && n.Style != nil && n.Style.Background != "" && hasText(n, marker)
	})
	if n == nil {
		t.Fatalf("no styled Column containing %q", marker)
	}
	return n
}

// --- 7.1 The style pipeline --------------------------------------------------

func TestStylePipelineLastPropWins(t *testing.T) {
	mgr := newApp(t)
	openLesson(t, mgr, "The style pipeline")

	const marker = "Painted by the last prop standing"

	// Base coat only: the box wears the blue it started with.
	if got := demoBoxNode(t, tree(t, mgr), marker).Style.Background; got != boxBlue {
		t.Fatalf("with no coats on, the box should be %s, got %s", boxBlue, got)
	}

	// Teal appended after the base overrides it.
	toggleCheckbox(t, mgr, 0, true)
	if got := demoBoxNode(t, tree(t, mgr), marker).Style.Background; got != boxTeal {
		t.Fatalf("the teal coat should win over the base, got %s", got)
	}

	// Both on: plum is appended after teal, so plum wins — the whole rule.
	toggleCheckbox(t, mgr, 1, true)
	cur := tree(t, mgr)
	if got := demoBoxNode(t, cur, marker).Style.Background; got != boxPlum {
		t.Fatalf("with both coats on the later prop must win, got %s", got)
	}
	if !hasTextContaining(cur, "appended last, so it wins") {
		t.Fatal("the demo should narrate the winner")
	}

	// Removing the later coat re-resolves the list: teal shows again.
	toggleCheckbox(t, mgr, 1, false)
	if got := demoBoxNode(t, tree(t, mgr), marker).Style.Background; got != boxTeal {
		t.Fatalf("dropping plum should fall back to teal, got %s", got)
	}
	assertNoConcerns(t)
}

// --- 7.2 UseStyle merging ----------------------------------------------------

func TestUseStyleMergesAndCannotClear(t *testing.T) {
	mgr := newApp(t)
	openLesson(t, mgr, "UseStyle: layers that merge")

	const marker = "One node, one Style — watch the corners and the type."

	// The named role alone: calloutStyle's fields, verbatim.
	s := styledText(t, tree(t, mgr), marker).Style
	if s.Background != boxPlum || s.FontSize != 15 || s.BorderRadius != 14 {
		t.Fatalf("the callout role should apply verbatim, got %+v", s)
	}

	// Layering bigType changes ONLY the type fields — fill, ink and corners
	// pass through because the layer never set them. This is the merge rule.
	toggleCheckbox(t, mgr, 0, true)
	s = styledText(t, tree(t, mgr), marker).Style
	if s.FontSize != 24 || s.FontWeight != int(core.Bold) {
		t.Fatalf("bigType should set the type fields, got %+v", s)
	}
	if s.Background != boxPlum || s.TextColor != "#FFFFFF" || s.BorderRadius != 14 {
		t.Fatalf("fields bigType never set must survive the layer, got %+v", s)
	}

	// The documented no-op: a zero field in a UseStyle merge holds no opinion.
	toggleCheckbox(t, mgr, 1, true)
	if s = styledText(t, tree(t, mgr), marker).Style; s.BorderRadius != 14 {
		t.Fatalf("UseStyle(Style{BorderRadius: 0}) must change nothing, got %v", s.BorderRadius)
	}

	// The direct prop assigns unconditionally — and it runs last.
	toggleCheckbox(t, mgr, 2, true)
	if s = styledText(t, tree(t, mgr), marker).Style; s.BorderRadius != 0 {
		t.Fatalf("core.BorderRadius(0) must force the field to zero, got %v", s.BorderRadius)
	}
	assertNoConcerns(t)
}

// --- 7.3 Theme anatomy -------------------------------------------------------

func TestThemeInspectorReadsBothBundledThemes(t *testing.T) {
	mgr := newApp(t)
	openLesson(t, mgr, "Inside a Theme")

	// The inspector opens on DefaultTheme's data: iOS blue, and a scale
	// topping out at 28. The hex captions are literal Text nodes.
	cur := tree(t, mgr)
	if !hasText(cur, core.DefaultTheme.Colors.Primary) {
		t.Fatal("the Default palette should list its Primary hex")
	}
	if !hasTextContaining(cur, "Title 28") {
		t.Fatal("the Default type scale line should read Title 28")
	}

	// Flip to Material: same rows, re-read from the other value — including
	// the teal Secondary the prose points at, and the smaller scale.
	tap(t, mgr, "Material")
	cur = tree(t, mgr)
	if !hasText(cur, core.MaterialTheme.Colors.Primary) {
		t.Fatal("the Material palette should list its Primary hex")
	}
	if !hasText(cur, core.MaterialTheme.Colors.Secondary) {
		t.Fatal("the Material palette should list its teal Secondary")
	}
	if !hasTextContaining(cur, "Title 22") {
		t.Fatal("the Material type scale line should read Title 22")
	}
	if !hasTextContaining(cur, "radius 4") {
		t.Fatal("the Material button base line should read radius 4")
	}

	// And back — the inspector is plain state over plain data.
	tap(t, mgr, "Default")
	if !hasText(tree(t, mgr), core.DefaultTheme.Colors.Primary) {
		t.Fatal("switching back should re-read DefaultTheme")
	}
	assertNoConcerns(t)
}

// --- 7.4 The live theme switcher ---------------------------------------------

func TestThemeSwitcherReskinsOnlyTheWrappedSubtree(t *testing.T) {
	mgr := newApp(t)
	openLesson(t, mgr, "Two themes, one tree")

	followBtn := func(root *node, label string) *node {
		n := findNode(root, func(n *node) bool {
			return n.Type == "Button" && n.Props["label"] == label
		})
		if n == nil || n.Style == nil {
			t.Fatalf("no styled Button labeled %q", label)
		}
		return n
	}

	// Under Default, the preview's button wears DefaultTheme's Button base.
	cur := tree(t, mgr)
	def := core.DefaultTheme.Components.Button
	if got := followBtn(cur, "Follow").Style; got.Background != def.Background {
		t.Fatalf("under Default the preview button should be %s, got %s",
			def.Background, got.Background)
	}

	// Swap the subtree to Material: the same button re-resolves to the
	// Material base — fill and corner radius both.
	tap(t, mgr, "Material")
	cur = tree(t, mgr)
	mat := core.MaterialTheme.Components.Button
	got := followBtn(cur, "Follow").Style
	if got.Background != mat.Background || got.BorderRadius != mat.BorderRadius {
		t.Fatalf("under Material the preview button should be %s/r%.0f, got %s/r%.0f",
			mat.Background, mat.BorderRadius, got.Background, got.BorderRadius)
	}

	// The scoping claim: the lesson's own footer sits OUTSIDE the wrapper and
	// must still wear the app's DefaultTheme.
	if got := followBtn(cur, "Next ›").Style; got.Background != def.Background {
		t.Fatalf("outside the wrapper the app theme must hold, got %s", got.Background)
	}

	// Handlers work across the themed boundary: the preview's state lives in
	// the lesson's frame and arrives by closure.
	tap(t, mgr, "Follow")
	cur = tree(t, mgr)
	if findNode(cur, func(n *node) bool { return n.Props["label"] == "Following ✓" }) == nil {
		t.Fatal("tapping Follow should flip the label through lesson state")
	}
	// The interaction must not have disturbed the theme choice.
	if got := followBtn(cur, "Following ✓").Style; got.Background != mat.Background {
		t.Fatalf("the theme choice should survive the tap, got %s", got.Background)
	}

	tap(t, mgr, "Default")
	if got := followBtn(tree(t, mgr), "Following ✓").Style; got.Background != def.Background {
		t.Fatalf("switching back should restore the Default base, got %s", got.Background)
	}
	assertNoConcerns(t)
}

// --- 7.5 Transitions ---------------------------------------------------------

func TestTransitionDeclarationRidesTheStyle(t *testing.T) {
	mgr := newApp(t)
	openLesson(t, mgr, "Transitions: declare the motion")

	const marker = "Same patch, different journey"

	// The demo opens at 250 ms: the canonical "<ms>ms <easing>" serialization
	// should already be on the box.
	box := demoBoxNode(t, tree(t, mgr), marker)
	if box.Style.Transition != "250ms ease-in-out" {
		t.Fatalf("the box should open declaring 250ms ease-in-out, got %q", box.Style.Transition)
	}
	if box.Style.Background != boxBlue {
		t.Fatalf("the box should open calm (blue), got %s", box.Style.Background)
	}

	// The flip is an ordinary state change — the new style just lands.
	tap(t, mgr, "Flip the look")
	box = demoBoxNode(t, tree(t, mgr), marker)
	if box.Style.Background != boxPlum || box.Style.BorderRadius != 24 {
		t.Fatalf("the flip should swap to the alert look, got %+v", box.Style)
	}
	// The declaration is unchanged by the flip: it rides every pass.
	if box.Style.Transition != "250ms ease-in-out" {
		t.Fatalf("the declaration should persist across the flip, got %q", box.Style.Transition)
	}

	// Snap is Transition(0): the declaration clears entirely rather than
	// serializing a zero duration.
	tap(t, mgr, "Snap")
	box = demoBoxNode(t, tree(t, mgr), marker)
	if box.Style.Transition != "" {
		t.Fatalf("Transition(0) must clear the declaration, got %q", box.Style.Transition)
	}

	tap(t, mgr, "800 ms")
	box = demoBoxNode(t, tree(t, mgr), marker)
	if box.Style.Transition != "800ms ease-in-out" {
		t.Fatalf("the pace picker should re-declare at 800 ms, got %q", box.Style.Transition)
	}
	assertNoConcerns(t)
}
