package comps

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/rohanthewiz/grmob/core"
)

// galleryItems is a three-card fixture: a flippable, shareable, actionable
// first card, a bare second card, and a third with a Back slot.
func galleryItems() []GalleryItem {
	return []GalleryItem{
		{Key: "fjord", Src: "https://example.com/fjord.jpg", Alt: "A fjord at dawn",
			Title: "Fjord", Subtitle: "Norway", Description: "Steep and deep.",
			Details:   []KeyValue{{Key: "Length", Value: "15 km"}},
			ActionURL: "https://example.com/learn/fjord",
			ShareURL:  "https://example.com/p/fjord"},
		{Key: "dune", Src: "https://example.com/dune.jpg", Title: "Dune"},
		{Key: "reef", Src: "https://example.com/reef.jpg", Title: "Reef",
			Back: core.Text("Custom back")},
	}
}

// galleryPass renders a later pass on ctx and audits it as renderDebug
// does, so concerns raised after a tap are seen too.
func galleryPass(t *testing.T, ctx *core.Context, v core.View) *core.Node {
	t.Helper()
	core.ClearConcerns()
	n := renderPass(ctx, v)
	ctx.EndRenderPass()
	core.AuditTree(n)
	if dump := core.DumpConcerns(); dump != "" {
		t.Errorf("concerns raised:\n%s", dump)
	}
	return n
}

// galleryNamed finds the node a reader hears as name.
func galleryNamed(n *core.Node, name string) *core.Node {
	return findFirst(n, func(x *core.Node) bool { return x.Style.AccessibilityLabel == name })
}

// galleryTap fires the onClick of the control named name, failing when there
// is none. A card and its dot share a name, so it looks for the named node
// that takes a tap.
func galleryTap(t *testing.T, ctx *core.Context, n *core.Node, name string) {
	t.Helper()
	b := findFirst(n, func(x *core.Node) bool {
		_, tappable := x.Props["onClick"].(string)
		return tappable && x.Style.AccessibilityLabel == name
	})
	if b == nil {
		t.Fatalf("no tappable control named %q", name)
	}
	ctx.TriggerCallback(b.Props["onClick"].(string))
}

// galleryCards returns the card layers by their card label prefix, in item
// order (the stage emits them last first).
func galleryCard(n *core.Node, label string) *core.Node {
	return findFirst(n, func(x *core.Node) bool {
		return x.Style.AccessibilityRole == core.RoleGroup && x.Style.AccessibilityLabel == label
	})
}

// The zero value is a usable, silent widget: an empty stage saying so.
func TestGalleryZeroValueRendersAnEmptyStage(t *testing.T) {
	_, n := renderDebug(t, Gallery{})
	if findText(n, "Nothing to show") == nil {
		t.Error("an empty gallery says it has nothing to show")
	}
	if n.Style.AccessibilityRole != core.RoleGroup || n.Style.AccessibilityLabel != "Gallery" {
		t.Errorf("root = %q %q, want a group named Gallery", n.Style.AccessibilityRole, n.Style.AccessibilityLabel)
	}
}

// Only the current card is reachable; the others are hidden and inert. The
// current card's title is a level-2 heading and its picture is named by Alt.
func TestGalleryOnlyTheCurrentCardIsReachable(t *testing.T) {
	_, n := renderDebug(t, Gallery{Items: galleryItems()})

	first := galleryCard(n, "1 of 3, Fjord")
	second := galleryCard(n, "2 of 3, Dune")
	if first == nil || second == nil {
		t.Fatal("each card is a group named by its position and title")
	}
	if first.Style.AccessibilityHidden || first.Style.Inert {
		t.Error("the current card is reachable")
	}
	if !second.Style.AccessibilityHidden || !second.Style.Inert {
		t.Error("a card off the stage is hidden and inert")
	}
	title := findText(first, "Fjord")
	if title == nil || title.Style.AccessibilityRole != core.RoleHeading || title.Style.AccessibilityHeadingLevel != 2 {
		t.Errorf("the title is a level-2 heading: %+v", title)
	}
	if galleryNamed(first, "A fjord at dawn") == nil {
		t.Error("the picture is named by its Alt")
	}
}

// Next moves to the next card, reports the change, and the new card is the
// reachable one. Previous is disabled on the first card without Wrap.
func TestGalleryNextAdvancesAndReports(t *testing.T) {
	var got []GalleryState
	g := Gallery{Items: galleryItems(), OnChange: func(s GalleryState) { got = append(got, s) }}
	ctx, n := renderDebug(t, g)

	if prev := galleryNamed(n, "Previous"); prev == nil || !prev.Style.Disabled {
		t.Error("Previous is disabled on the first card of a gallery that does not wrap")
	}
	galleryTap(t, ctx, n, "Next")
	if len(got) != 1 || got[0].Index != 1 {
		t.Fatalf("OnChange got %+v, want one change to index 1", got)
	}
	n = galleryPass(t, ctx, g)
	if c := galleryCard(n, "2 of 3, Dune"); c == nil || c.Style.AccessibilityHidden {
		t.Error("the second card is current after Next")
	}
}

// With Wrap, Previous on the first card goes to the last; without it, Next on
// the last card is a no-op that OnChange never hears.
func TestGalleryWrapsOnlyWhenAsked(t *testing.T) {
	var got []GalleryState
	g := Gallery{Items: galleryItems(), Wrap: true, OnChange: func(s GalleryState) { got = append(got, s) }}
	ctx, n := renderDebug(t, g)
	galleryTap(t, ctx, n, "Previous")
	if len(got) != 1 || got[0].Index != 2 {
		t.Fatalf("wrapping Previous got %+v, want index 2", got)
	}

	got = nil
	g = Gallery{Items: galleryItems(), State: GalleryState{Index: 2},
		OnChange: func(s GalleryState) { got = append(got, s) }}
	ctx, n = renderDebug(t, g)
	if next := galleryNamed(n, "Next"); next == nil || !next.Style.Disabled {
		t.Error("Next is disabled on the last card")
	}
	galleryTap(t, ctx, n, "Next")
	if len(got) != 0 {
		t.Errorf("a step past the end is a no-op, got %+v", got)
	}
}

// Controlled, a tap is a request: OnChange hears it, and the drawn card
// follows State, not the tap.
func TestGalleryControlledFollowsState(t *testing.T) {
	var got []GalleryState
	g := Gallery{Items: galleryItems(), Controlled: true, State: GalleryState{Index: 1},
		OnChange: func(s GalleryState) { got = append(got, s) }}
	ctx, n := renderDebug(t, g)
	if c := galleryCard(n, "2 of 3, Dune"); c == nil || c.Style.AccessibilityHidden {
		t.Fatal("a controlled gallery opens on State.Index")
	}
	galleryTap(t, ctx, n, "Next")
	if len(got) != 1 || got[0].Index != 2 {
		t.Fatalf("OnChange got %+v, want index 2", got)
	}
	n = galleryPass(t, ctx, g)
	if c := galleryCard(n, "2 of 3, Dune"); c == nil || c.Style.AccessibilityHidden {
		t.Error("until the caller passes a new State, the card does not move")
	}
}

// Controlled with no OnChange is reported, since every control is dead.
func TestGalleryControlledWithoutOnChangeIsReported(t *testing.T) {
	core.SetDebugMode(true)
	core.ClearConcerns()
	t.Cleanup(func() { core.SetDebugMode(false); core.ClearConcerns() })
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	Gallery{Items: galleryItems(), Controlled: true}.Render(ctx)
	ctx.EndRenderPass()
	if !strings.Contains(core.DumpConcerns(), ConcernGalleryFrozen) {
		t.Errorf("concerns = %q, want %s", core.DumpConcerns(), ConcernGalleryFrozen)
	}
}

// Details flips the card: its state is stated on the control, the back face
// becomes the reachable one, Escape is claimed only while flipped, and a
// move to another card shows that card's front.
func TestGalleryFlipShowsTheBack(t *testing.T) {
	g := Gallery{Items: galleryItems()}
	ctx, n := renderDebug(t, g)

	details := galleryNamed(n, "Details")
	if details == nil || details.Style.AccessibilityExpanded != core.ExpandedWhen(false) {
		t.Fatalf("Details states it is collapsed: %+v", details)
	}
	if _, claimed := n.Props["onEscape"]; claimed {
		t.Error("a gallery showing a front claims no Escape")
	}
	galleryTap(t, ctx, n, "Details")
	n = galleryPass(t, ctx, g)

	if d := galleryNamed(n, "Details"); d.Style.AccessibilityExpanded != core.ExpandedWhen(true) {
		t.Error("Details states it is expanded once flipped")
	}
	if _, claimed := n.Props["onEscape"]; !claimed {
		t.Error("a flipped card claims Escape, to turn it back")
	}
	back := findText(galleryCard(n, "1 of 3, Fjord"), "Length")
	if back == nil {
		t.Fatal("the back shows the Details list")
	}

	galleryTap(t, ctx, n, "Next")
	n = galleryPass(t, ctx, g)
	if d := galleryNamed(n, "Details"); d != nil {
		t.Error("the second card has no back, so no Details button")
	}
	if _, claimed := n.Props["onEscape"]; claimed {
		t.Error("moving on shows the next card's front, so Escape is released")
	}
}

// The Back slot replaces Details on the back face.
func TestGalleryBackSlotWins(t *testing.T) {
	_, n := renderDebug(t, Gallery{Items: galleryItems(), State: GalleryState{Index: 2, Flipped: true}})
	if findText(n, "Custom back") == nil {
		t.Error("the Back view is drawn on the back face")
	}
}

// The default share copies the share text and confirms it; OnShare and
// OnAction replace the defaults and receive the current item.
func TestGalleryShareAndAction(t *testing.T) {
	seen := recordSystemEvents(t)
	ctx, n := renderDebug(t, Gallery{Items: galleryItems()})
	galleryTap(t, ctx, n, "Share")
	want := []string{"clipboard:write:Fjord https://example.com/p/fjord", "haptic:light", "toast:Copied to clipboard"}
	if fmt.Sprint(*seen) != fmt.Sprint(want) {
		t.Errorf("default share = %v, want %v", *seen, want)
	}

	var shared, acted []string
	g := Gallery{Items: galleryItems(), State: GalleryState{Index: 1},
		OnShare:  func(it GalleryItem, i int) { shared = append(shared, fmt.Sprint(it.Key, i)) },
		OnAction: func(it GalleryItem, i int) { acted = append(acted, fmt.Sprint(it.Key, i)) },
	}
	ctx, n = renderDebug(t, g)
	galleryTap(t, ctx, n, "Share")
	learn := findFirst(n, func(x *core.Node) bool { return x.Type == "Button" && x.Props["label"] == "Learn more" })
	if learn == nil {
		t.Fatal("OnAction draws the action button for every card")
	}
	ctx.TriggerCallback(learn.Props["onClick"].(string))
	if fmt.Sprint(shared) != "[dune1]" || fmt.Sprint(acted) != "[dune1]" {
		t.Errorf("shared %v, acted %v; want the current card each", shared, acted)
	}
}

// A card with nothing to share and no action draws neither button.
func TestGalleryOmitsToolsWithNothingToDo(t *testing.T) {
	_, n := renderDebug(t, Gallery{Items: galleryItems(), State: GalleryState{Index: 1}})
	if galleryNamed(n, "Share") != nil {
		t.Error("no ShareURL, ShareText or OnShare: no share button")
	}
	if findFirst(n, func(x *core.Node) bool { return x.Props["label"] == "Learn more" }) != nil {
		t.Error("no ActionURL or OnAction: no action button")
	}
}

// Zooming in place grows the stage; fullscreen zoom opens a Lightbox and is
// announced as opening a dialog; zoom off draws no Zoom button.
func TestGalleryZoomModes(t *testing.T) {
	stage := func(n *core.Node) *core.Node {
		return findFirst(n, func(x *core.Node) bool { return x.Type == "ZStack" && x.Style.Overflow == "hidden" })
	}
	g := Gallery{Items: galleryItems()}
	ctx, n := renderDebug(t, g)
	if h := stage(n).Style.Height; h != "360px" {
		t.Errorf("stage height = %q, want the 360px default", h)
	}
	galleryTap(t, ctx, n, "Zoom")
	n = galleryPass(t, ctx, g)
	if h := stage(n).Style.Height; h != "540px" {
		t.Errorf("zoomed stage height = %q, want one and a half times", h)
	}
	if z := galleryNamed(n, "Zoom"); z.Style.AccessibilitySelected != core.SelectedWhen(true) {
		t.Error("the in-place zoom states that it is on")
	}

	_, n = renderDebug(t, Gallery{Items: galleryItems(), Zoom: GalleryZoomFullscreen, State: GalleryState{Zoomed: true}})
	if z := galleryNamed(n, "Zoom"); z.Style.AccessibilityHasPopup != core.PopupDialog {
		t.Error("the fullscreen zoom says it opens a dialog")
	}
	if findFirst(n, func(x *core.Node) bool { return x.Type == "Modal" }) == nil {
		t.Error("the fullscreen zoom is a Lightbox")
	}

	_, n = renderDebug(t, Gallery{Items: galleryItems(), Zoom: GalleryZoomOff})
	if galleryNamed(n, "Zoom") != nil {
		t.Error("zoom off draws no Zoom button")
	}
}

// The tap zones step, flip and zoom, and are hidden from readers.
func TestGalleryTapZones(t *testing.T) {
	var got []GalleryState
	g := Gallery{Items: galleryItems(), OnChange: func(s GalleryState) { got = append(got, s) }}
	ctx, n := renderDebug(t, g)
	zones := findFirst(n, func(x *core.Node) bool {
		return x.Type == "Row" && x.Style.AccessibilityHidden && len(x.Children) == 3
	})
	if zones == nil {
		t.Fatal("the stage carries a hidden row of three tap zones")
	}
	if _, ok := zones.Children[0].Props["onClick"]; ok {
		t.Error("the previous zone on the first card has no handler")
	}
	ctx.TriggerCallback(zones.Children[1].Props["onClick"].(string))
	ctx.TriggerCallback(zones.Children[1].Props["onLongPress"].(string))
	ctx.TriggerCallback(zones.Children[2].Props["onClick"].(string))
	want := "[{0 true false false} {0 true true false} {1 false true false}]"
	if fmt.Sprint(got) != want {
		t.Errorf("tap, hold, tap-next = %v, want %v", got, want)
	}
}

// A swipe on the stage steps: leading for the next card, trailing for the
// previous one, and it still steps while the card is flipped, showing the
// next card's front. A single card has nowhere to go, so its stage claims no
// swipe and a Scroll around it keeps every drag.
func TestGallerySwipeSteps(t *testing.T) {
	var got []int
	g := Gallery{Items: galleryItems(), OnChange: func(s GalleryState) { got = append(got, s.Index) }}
	ctx, n := renderDebug(t, g)

	swipe := func(n *core.Node, d core.SwipeDirection) {
		t.Helper()
		stage := findFirst(n, func(x *core.Node) bool { _, ok := x.Props["onSwipe"]; return ok })
		if stage == nil || stage.Type != "ZStack" {
			t.Fatalf("the stage carries onSwipe, got %+v", stage)
		}
		ctx.TriggerTextCallback(stage.Props["onSwipe"].(string), string(d))
	}

	// Trailing on the first card of a gallery that does not wrap is a no-op
	// OnChange never hears.
	swipe(n, core.SwipeTrailing)
	swipe(n, core.SwipeLeading)
	n = galleryPass(t, ctx, g)
	swipe(n, core.SwipeLeading)
	n = galleryPass(t, ctx, g)
	// The third card has a back; flip it, then swipe back from the back face.
	galleryTap(t, ctx, n, "Details")
	n = galleryPass(t, ctx, g)
	swipe(n, core.SwipeTrailing)
	if fmt.Sprint(got) != "[1 2 2 1]" {
		t.Errorf("trailing, leading, leading, flip, trailing gave indexes %v, want [1 2 2 1]", got)
	}
	n = galleryPass(t, ctx, g)
	if c := galleryCard(n, "2 of 3, Dune"); c == nil || c.Style.AccessibilityHidden {
		t.Error("the second card is current, front up, after the swipe back")
	}

	_, one := renderDebug(t, Gallery{Items: galleryItems()[:1]})
	if s := findFirst(one, func(x *core.Node) bool { _, ok := x.Props["onSwipe"]; return ok }); s != nil {
		t.Error("a one-card gallery claims no swipe")
	}
}

// The tap zones paint above every card on the web, whatever the transition
// and whichever card is current. The card layers carry core.ZIndex, and a
// zone row without one painted under them all, so in the browser the
// picture took every click and the zones never fired. Checked from both ends
// of the deck, since zOrder lifts the cards already left behind highest.
func TestGalleryTapZonesPaintAboveTheCards(t *testing.T) {
	for _, tr := range []GalleryTransition{GalleryTransitionSlide, GalleryTransitionFade, GalleryTransitionDeck, GalleryTransitionNone} {
		for _, at := range []int{0, len(galleryItems()) - 1} {
			t.Run(fmt.Sprintf("%s/at-%d", tr, at), func(t *testing.T) {
				g := Gallery{Items: galleryItems(), Motion: GalleryMotion{Transition: tr},
					State: GalleryState{Index: at}}
				_, n := renderDebug(t, g)
				zones := findFirst(n, func(x *core.Node) bool {
					return x.Type == "Row" && x.Style.AccessibilityHidden && len(x.Children) == 3
				})
				if zones == nil {
					t.Fatal("no tap-zone row on the stage")
				}
				for i, it := range galleryItems() {
					card := galleryCard(n, fmt.Sprintf("%d of 3, %s", i+1, it.Title))
					if card == nil {
						t.Fatalf("no card layer for %s", it.Title)
					}
					if zones.Style.ZIndex <= card.Style.ZIndex {
						t.Errorf("tap zones z-index %d does not clear %s's %d",
							zones.Style.ZIndex, it.Title, card.Style.ZIndex)
					}
				}
			})
		}
	}
}

// Each transition poses the current card opaque and at rest, and the slide
// waits the next card one stage width along.
func TestGalleryTransitionsPoseTheCards(t *testing.T) {
	for _, tr := range []GalleryTransition{GalleryTransitionSlide, GalleryTransitionFade, GalleryTransitionDeck, GalleryTransitionNone} {
		t.Run(string(tr)+"/", func(t *testing.T) {
			_, n := renderDebug(t, Gallery{Items: galleryItems(), Motion: GalleryMotion{Transition: tr}})
			cur := galleryCard(n, "1 of 3, Fjord")
			if cur.Style.Opacity != 1 {
				t.Errorf("current card opacity = %v, want 1", cur.Style.Opacity)
			}
			if tr == GalleryTransitionNone && cur.Style.Transition != "" {
				t.Errorf("none declares no transition, got %q", cur.Style.Transition)
			}
			if tr == GalleryTransitionSlide {
				if next := galleryCard(n, "2 of 3, Dune"); next.Style.TranslateX != "100%" {
					t.Errorf("next card waits at %q, want 100%%", next.Style.TranslateX)
				}
			}
		})
	}
	_, n := renderDebug(t, Gallery{Items: galleryItems(), Motion: GalleryMotion{DurationMs: 500, Easing: core.EaseOut}})
	if tr := galleryCard(n, "1 of 3, Fjord").Style.Transition; tr != "500ms ease-out" {
		t.Errorf("configured transition = %q", tr)
	}
}

// Every look renders silently under every bundled theme, and the fixed looks
// render the card under their own theme.
func TestGalleryLooksUnderEveryTheme(t *testing.T) {
	looks := []GalleryLook{GalleryLookTheme, GalleryLookMinimal, GalleryLookCinema, GalleryLookPolaroid}
	for name, th := range core.BundledThemes() {
		for _, look := range looks {
			t.Run(name+"/"+string(look), func(t *testing.T) {
				core.SetDebugMode(true)
				core.ClearConcerns()
				t.Cleanup(func() { core.SetDebugMode(false); core.ClearConcerns() })
				ctx := core.NewContext().WithTheme(th)
				ctx.BeginRenderPass()
				n := Gallery{Items: galleryItems(), Look: look, State: GalleryState{Flipped: true}}.Render(ctx)
				ctx.EndRenderPass()
				core.AuditTree(n)
				if dump := core.DumpConcerns(); dump != "" {
					t.Errorf("concerns:\n%s", dump)
				}
				themed := findFirst(n, func(x *core.Node) bool { return x.Type == "Theme" }) != nil
				fixed := look == GalleryLookCinema || look == GalleryLookPolaroid
				if fixed && !themed && th != core.DarkTheme && th != core.DefaultTheme {
					t.Error("a fixed look renders the card under its own theme")
				}
				if !fixed && themed {
					t.Error("a themed look renders the card under the app's theme")
				}
			})
		}
	}
}

// The indicator defaults to dots for a few cards and a counter for many.
func TestGalleryIndicatorAuto(t *testing.T) {
	_, n := renderDebug(t, Gallery{Items: galleryItems()})
	dot := galleryNamed(n, "2 of 3, Dune")
	if dot == nil {
		t.Fatal("dots: each is a button named for its card")
	}
	cur := findFirst(n, func(x *core.Node) bool {
		return x.Style.AccessibilityRole == core.RoleButton && x.Style.AccessibilityCurrent == core.CurrentTrue
	})
	if cur == nil || cur.Style.AccessibilityLabel != "1 of 3, Fjord" {
		t.Error("the current dot is marked current")
	}

	many := make([]GalleryItem, 12)
	for i := range many {
		many[i] = GalleryItem{Key: fmt.Sprint(i), Src: "x"}
	}
	_, n = renderDebug(t, Gallery{Items: many})
	if findText(n, "1 / 12") == nil {
		t.Error("twelve cards show a counter")
	}
}

// Thumbnails are a horizontal scroll of named buttons that jump.
func TestGalleryThumbnailsJump(t *testing.T) {
	var got []GalleryState
	g := Gallery{Items: galleryItems(), Indicator: GalleryIndicatorThumbnails,
		OnChange: func(s GalleryState) { got = append(got, s) }}
	ctx, n := renderDebug(t, g)
	galleryTap(t, ctx, n, "3 of 3, Reef")
	if len(got) != 1 || got[0].Index != 2 {
		t.Errorf("thumbnail tap got %+v, want index 2", got)
	}
}

// Autoplay draws a pause toggle that states whether it is on.
func TestGalleryAutoplayHasAPause(t *testing.T) {
	g := Gallery{Items: galleryItems(), Autoplay: 5 * time.Second}
	ctx, n := renderDebug(t, g)
	toggle := galleryNamed(n, "Autoplay")
	if toggle == nil || toggle.Style.AccessibilitySelected != core.SelectedWhen(true) {
		t.Fatalf("autoplay draws a toggle stated on: %+v", toggle)
	}
	if c := findText(n, "1 / 3"); c != nil && c.Style.AccessibilityRole == core.RoleStatus {
		t.Error("a moving counter is not a status region")
	}
	galleryTap(t, ctx, n, "Autoplay")
	n = galleryPass(t, ctx, g)
	if galleryNamed(n, "Autoplay").Style.AccessibilitySelected != core.SelectedWhen(false) {
		t.Error("paused, the toggle states it is off")
	}
}

// A caller's StageStyle and Style outrank the widget's own props.
func TestGalleryCallerStyleWins(t *testing.T) {
	_, n := renderDebug(t, Gallery{Items: galleryItems(),
		Style:      []core.StyleProp{core.Gap(31)},
		StageStyle: []core.StyleProp{core.Height("200px")},
	})
	if n.Style.Gap != 31 {
		t.Errorf("root gap = %v, want the caller's 31", n.Style.Gap)
	}
	if findFirst(n, func(x *core.Node) bool { return x.Type == "ZStack" && x.Style.Height == "200px" }) == nil {
		t.Error("StageStyle's height replaces the default")
	}
}

// Labels localise every word, including the position.
func TestGalleryLabelsLocalise(t *testing.T) {
	_, n := renderDebug(t, Gallery{Items: galleryItems(), Labels: GalleryLabels{
		Gallery: "Galerie", Next: "Suivant", Details: "Détails",
		Position: func(i, c int) string { return fmt.Sprintf("%d sur %d", i+1, c) },
	}})
	for _, name := range []string{"Galerie", "Suivant", "Détails", "1 sur 3, Fjord"} {
		if galleryNamed(n, name) == nil {
			t.Errorf("no control named %q", name)
		}
	}
}

// Autoplay advances by itself after the delay, and wraps at the end.
func TestGalleryAutoplayAdvances(t *testing.T) {
	changes := make(chan GalleryState, 4)
	g := Gallery{Items: galleryItems(), State: GalleryState{Index: 2}, Autoplay: 10 * time.Millisecond,
		OnChange: func(s GalleryState) { changes <- s }}
	renderDebug(t, g)
	select {
	case s := <-changes:
		if s.Index != 0 {
			t.Errorf("autoplay from the last card went to %d, want a wrap to 0", s.Index)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("autoplay never advanced")
	}
}

// A paused, flipped or zoomed gallery does not autoplay.
func TestGalleryAutoplayHoldsWhilePausedOrFlipped(t *testing.T) {
	for _, st := range []GalleryState{{Paused: true}, {Flipped: true}, {Zoomed: true}} {
		changes := make(chan GalleryState, 4)
		renderDebug(t, Gallery{Items: galleryItems(), State: st, Autoplay: 5 * time.Millisecond,
			OnChange: func(s GalleryState) { changes <- s }})
		select {
		case s := <-changes:
			t.Errorf("state %+v autoplayed to %+v", st, s)
		case <-time.After(60 * time.Millisecond):
		}
	}
}
