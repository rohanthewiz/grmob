package tutorial

import (
	"strings"
	"sync"

	"github.com/rohanthewiz/grmob/comps"
	"github.com/rohanthewiz/grmob/core"
)

// The two-pane ("split") layout: the lesson's guide — its prose, code and key
// points — on a wide reading pane on the left, and the lesson's live demos on
// a phone-sized screen on the right.
//
// The tutorial was written for a phone, and on a phone it is right: one
// column, prose then demo then prose. In a desktop browser that same column,
// squeezed into a phone bezel, makes the reader scroll a 400px strip to read
// a paragraph and scroll again to reach the demo it describes. The split puts
// the reading where there is room to read and keeps the demos on a phone,
// where their layouts are honest (a Row that fits 400px, not 1400px).
//
// # Who decides
//
// The host page. Only it knows how wide the window is and what the reader
// chose, so the mode arrives as a host event on the same generic channel the
// deep links use (deeplink.go), and the app holds it as session state:
//
//	page ──HostEvent("layout", {mode: "split" | "phone"})──▶ app
//
// No other host sends it, so the natives, the headless tests and the
// screenshot host all keep the phone layout by construction — the split is
// an opt-in the web page makes, not a platform check the app performs.
//
// # How a lesson is split without touching sixty lesson bodies
//
// Every lesson Body is one view tree that interleaves exposition and demos,
// and its hooks are positional. Rendering the body twice (once for each pane)
// would double every ticker, location watch and permission check in it;
// rendering its pieces in a different order per mode would shift hook slots
// under a reader who flips the toggle mid-lesson.
//
// So the body renders exactly as it always has, in its own frame, and the
// split happens afterwards, on the rendered *core.Node tree:
//
//	Navigator output (lesson frame)          split layout
//	Screen                                   Row #tutorial-split
//	  topbar, header                           Column #tutorial-guide
//	  Column (Body)                              Screen (copy)
//	    prose                                      topbar, header
//	    code                                       Column (copy)
//	    demoPanel  ◀── key "tryit/<hint>"            prose, code
//	    keyPoints                                    demoPointer  ◀── in its place
//	  nav                                            keyPoints
//	                                               nav
//	                                           Column #tutorial-phone
//	                                             Column #tutorial-phone-screen
//	                                               phone header
//	                                               Scroll: demoPanel …
//
// demoPanel marks its root with a key (see demoKeyPrefix), and liftDemos
// walks the lesson's tree, moving each marked subtree to the phone and
// leaving a pointer behind. Hook slots were all claimed during the render, so
// moving the nodes afterwards cannot disturb them; callback IDs travel with
// the nodes they are on; and a mode switch is only a different arrangement of
// the same nodes — the demo's state survives it.
//
// The nodes are never written: a rendered Node is frozen (see core.Node), and
// liftDemos copies only the spine from the root to each moved subtree.

// layoutEvent is the host event's name. Its payload is one string field,
// "mode": "split" turns the two panes on, anything else turns them off.
const layoutEvent = "layout"

// demoKeyPrefix marks a demo panel's root node, followed by the panel's hint.
//
// A key, rather than a new prop, because a key is the one thing a Node carries
// that every host already treats as identity and none renders: it adds no
// attribute to the DOM and no field a native shell has to learn. Every hint in
// the curriculum is a literal, so the key is stable across passes and the
// reconciler diffs a panel in place exactly as it did before it had one (see
// reconcile's keyed rule: keys only matter when they differ).
const demoKeyPrefix = "tryit/"

// lessonRootKey marks a lesson route's root node. Navigator prefixes a route's
// key with its frame ("nav:frame:7/tutorial-lesson"; see core.withFrameKey), so
// it is matched as a suffix. It is how the layout tells a lesson apart from a
// screen a demo pushed on top of one (chapter 6), which must not be split.
const lessonRootKey = "tutorial-lesson"

// The panes' document ids. core.AccessibilityID is the `id` attribute on the
// web and unread on the natives, which is exactly the reach needed: the
// panes' look — the bezel, the reading card, the page's own light and dark
// palette — is the host page's chrome and lives in its stylesheet
// (wasm/index.html), which finds the panes by these.
const (
	splitID       = "tutorial-split"
	guideID       = "tutorial-guide"
	phoneID       = "tutorial-phone"
	phoneScreenID = "tutorial-phone-screen"
	// phoneContentID wraps whatever the phone is showing; see phoneScreen.
	phoneContentID = "tutorial-phone-content"
	// guideNoteID is the bar under a covered lesson; see pushedScreenBar.
	guideNoteID = "tutorial-guide-note"
)

// bootLayout is the layout the page asked for before the app's first render.
//
// # Why there is a boot value at all
//
// The page learns the window's width and the reader's choice before it has
// mounted anything, but useLayoutMode subscribes during the first render, so
// the page used to mount first and send the mode after. Every wide-screen boot
// therefore built and mounted the whole phone layout, bezel and all, and then
// diffed it into the split when the event's patch arrived:
//
//	RenderInitial ─▶ mount (phone layout) ─▶ HostEvent("layout") ─▶ patch (split)
//	                 └── the tree this removes
//
// It was thought to be a visible frame. wasm/verify's browser check 17
// measured it with the old order restored: the patch is pushed from inside
// the HostEvent call, in the same task as the mount, so no frame of the phone
// layout was ever painted. What the order cost was the work: a tree built,
// mounted and patched away on every wide boot.
//
// So a package-level subscriber takes the event before any tree exists, and
// App seeds its split state from it. The page sends the mode first:
//
//	HostEvent("layout") ─▶ bootLayout ─▶ RenderInitial ─▶ mount (split)
//
// # Only while no tree is listening
//
// Once a tree has subscribed, the mode is that tree's state and this value is
// left alone. Otherwise a mode sent to one app would become the boot mode of
// the next one built in the same process, which is every test in this package
// after the first split test. A tree that closes stops counting, so a
// hot-reloaded module (a fresh process anyway) and a test's next app both boot
// from what was sent to them, or from the phone layout if nothing was.
var bootLayout struct {
	mu    sync.Mutex
	trees int // live useLayoutMode subscriptions
	split bool
}

func init() {
	core.OnHostEvent(layoutEvent, func(data map[string]any) {
		mode, _ := data["mode"].(string)
		bootLayout.mu.Lock()
		defer bootLayout.mu.Unlock()
		if bootLayout.trees == 0 {
			bootLayout.split = mode == "split"
		}
	})
}

// bootSplit is the split state an app starts in: what the page sent before
// the first render, or the phone layout.
func bootSplit() bool {
	bootLayout.mu.Lock()
	defer bootLayout.mu.Unlock()
	return bootLayout.split
}

// layoutRecord is the hook-slot memory of useLayoutMode, the same shape and
// for the same reason as useDeepLinks' routeRecord: App runs every pass, and
// the subscription must be taken once per context tree.
type layoutRecord struct {
	mu         sync.Mutex
	subscribed bool
}

// useLayoutMode subscribes the app to the page's layout event, once per
// context tree, and releases it when the tree closes. Called on the session
// scope, above the Navigator, because the mode outlives every frame.
func (t *tutorial) useLayoutMode(ctx *core.Context) {
	slot := core.NewState(ctx, &layoutRecord{})
	rec := slot.Get()

	rec.mu.Lock()
	already := rec.subscribed
	rec.subscribed = true
	rec.mu.Unlock()
	if already {
		return
	}

	bootLayout.mu.Lock()
	bootLayout.trees++
	bootLayout.mu.Unlock()
	cancel := core.OnHostEvent(layoutEvent, func(data map[string]any) {
		mode, _ := data["mode"].(string)
		split := mode == "split"
		// Only a change is a Set: the page restates the mode at every boot and
		// on every resize across its breakpoint, and a Set that changes
		// nothing would still request a pass.
		if t.split.Get() != split {
			t.split.Set(split)
		}
	})
	ctx.OnClose(func() {
		cancel()
		rec.mu.Lock()
		rec.subscribed = false
		rec.mu.Unlock()
		bootLayout.mu.Lock()
		bootLayout.trees--
		// The next tree boots from what is sent to it, not from this one's
		// mode (see bootLayout).
		if bootLayout.trees == 0 {
			bootLayout.split = false
		}
		bootLayout.mu.Unlock()
	})
}

// withLayout wraps the Navigator in the chosen layout. In the phone layout it
// is transparent — the Navigator's tree is returned as it was rendered, byte
// for byte — so the natives and every existing test see no change.
//
// In the split layout it arranges the tree by what is on top of the stack:
//
//	contents (the root frame)   guide: the contents     phone: a splash
//	a lesson                    guide: the lesson       phone: its demos
//	a screen a demo pushed      guide: a note           phone: that screen
//
// The last row is chapter 6's navigation demos, whose pushed screens are real
// frames on the tutorial's own stack. The pushed screen, which carries its
// own way back, runs on the phone, where it belongs. The lesson underneath is
// not rendered while it is up (Navigator renders only the top frame), so the
// guide shows the last rendering of it, made inert (see pushedScreenGuide).
func (t *tutorial) withLayout(nav core.View) core.View {
	return core.ComponentFunc(func(ctx *core.Context) *core.Node {
		n := nav.Render(ctx)
		// In either layout, so a reader who switches to the split while a
		// pushed screen is up still gets the lesson in the guide.
		if n != nil && strings.HasSuffix(n.Key, "/"+lessonRootKey) {
			t.lastLesson.Get().remember(t.current.Get(), n)
		}
		if !t.split.Get() || n == nil {
			return n
		}
		switch {
		case core.StackDepth(ctx) <= 1:
			return splitView(ctx, n, phoneScreen("splash", nil, phoneSplash(), nil))
		case strings.HasSuffix(n.Key, "/"+lessonRootKey):
			return t.splitLesson(ctx, n)
		default:
			// The pushed screen's root keeps its frame key, so each push is a
			// fresh screen on the phone exactly as it is in the phone layout.
			guide, tail := t.pushedScreenGuide(ctx)
			return splitView(ctx, guide, phoneScreen("pushed", nil, nodeView{n}, nil), tail...)
		}
	})
}

// splitLesson lifts a lesson's demos out of its tree and lays the two halves
// side by side.
func (t *tutorial) splitLesson(ctx *core.Context, n *core.Node) *core.Node {
	guide, demos, overlays := liftDemos(n, func(hint string) *core.Node {
		return demoPointer(hint).Render(ctx)
	})

	var content core.View
	if len(demos) == 0 {
		// Every lesson has at least one demo today; this is the honest screen
		// for one that does not, rather than an empty phone.
		content = core.Column(
			core.Padding(20),
			caption("This lesson has no live demo — everything it teaches is in the guide."),
		)
	} else {
		scroll := []core.PropsAndChildren{core.Padding(14), core.Gap(14)}
		for _, d := range demos {
			scroll = append(scroll, nodeView{d})
		}
		// Keyed by the lesson's frame, so Next — a new frame — replaces the
		// phone's content outright and its scroll starts at the top, as the
		// guide's does (core.withFrameKey is the same argument for the guide).
		content = core.Keyed(n.Key+"/demos", core.Scroll(scroll...))
	}
	// Modals declared outside every demo panel still belong to the demo: a
	// dialog is part of the app being demonstrated, and on the web the phone
	// screen is the containing block that keeps its backdrop inside the bezel
	// (see #tutorial-phone-screen in wasm/index.html). Left in the guide, it
	// would dim the reading pane instead.
	return splitView(ctx, guide, phoneScreen("lesson", phoneHeader(t.current.Get()), content, overlays))
}

// splitView is the two-pane frame: the guide, then the phone, in reading
// order. Both panes are labelled groups, so a screen reader announces which
// half it has entered — the visual bezel says the same to a sighted reader.
//
// The panes carry only identity and semantics here; their sizes and looks
// are the page's (wasm/index.html), which is why there are no style props.
//
// guideTail goes into the guide pane AFTER the guide, never before it: the
// reconciler matches children by position, and anything in front of the guide
// would move it to another slot and rebuild it, scroll position and all.
func splitView(ctx *core.Context, guide *core.Node, phone core.View, guideTail ...core.View) *core.Node {
	pane := []core.PropsAndChildren{
		core.AccessibilityID(guideID),
		core.AccessibilityRole(core.RoleGroup),
		core.AccessibilityLabel("Lesson guide"),
		nodeView{guide},
	}
	for _, v := range guideTail {
		pane = append(pane, v)
	}
	return core.Row(
		core.AccessibilityID(splitID),
		core.Column(pane...),
		core.Column(
			core.AccessibilityID(phoneID),
			core.AccessibilityRole(core.RoleGroup),
			core.AccessibilityLabel("Live demo"),
			phone,
		),
	).Render(ctx)
}

// phoneScreen is the phone's display: the element the page styles as the
// screen inside the bezel. It has three slots, because the page sizes them
// differently:
//
//	#tutorial-phone-screen        the glass: clips, and contains fixed overlays
//	  header                      optional; its natural height
//	  #tutorial-phone-content     takes the rest, and so does its one child
//	    content
//	  overlays…                   Modals; position: fixed, so out of the flow
//
// The content's wrapper is what lets one stylesheet rule size whatever the
// phone is showing — a Scroll of demos, the splash, a whole pushed screen —
// the way `#app > *` sizes the root in the phone layout.
//
// kind names what the screen is showing ("splash", "lesson", "pushed"), as
// its key: a different kind is a different screen to every host, while two
// lessons in a row share the glass and swap only the content, whose own key
// changes with the frame.
func phoneScreen(kind string, header, content core.View, overlays []*core.Node) core.View {
	items := []core.PropsAndChildren{core.AccessibilityID(phoneScreenID)}
	if header != nil {
		items = append(items, header)
	}
	items = append(items, core.Column(core.AccessibilityID(phoneContentID), content))
	for _, o := range overlays {
		items = append(items, nodeView{o})
	}
	return core.Keyed(kind, core.Column(items...))
}

// liftDemos splits a rendered lesson into its guide and its demos.
//
// Every subtree whose root carries demoKeyPrefix is moved to demos, in tree
// order, and replaced in the guide by pointer(hint). Every Modal outside those
// subtrees is moved to overlays and dropped from the guide (see splitLesson).
// A demo panel's own subtree is not descended into: whatever it holds,
// modals included, moves with it.
//
// Copy-on-write: a node none of whose descendants moved is returned as is,
// and a node that lost or swapped a child is a shallow copy with a fresh
// Children slice. The input tree is never written, which matters because a
// subtree may be a core.Cached node shared with other passes.
func liftDemos(n *core.Node, pointer func(hint string) *core.Node) (guide *core.Node, demos, overlays []*core.Node) {
	var lift func(n *core.Node) *core.Node
	lift = func(n *core.Node) *core.Node {
		if n == nil {
			return nil
		}
		if hint, ok := strings.CutPrefix(n.Key, demoKeyPrefix); ok {
			demos = append(demos, n)
			return pointer(hint)
		}
		if n.Type == "Modal" {
			overlays = append(overlays, n)
			return nil
		}
		changed := false
		kids := make([]*core.Node, 0, len(n.Children))
		for _, c := range n.Children {
			nc := lift(c)
			if nc != c {
				changed = true
			}
			if nc != nil {
				kids = append(kids, nc)
			}
		}
		if !changed {
			return n
		}
		cp := *n
		cp.Children = kids
		return &cp
	}
	guide = lift(n)
	return guide, demos, overlays
}

// demoPointer stands in the guide where a demo panel was: the panel's badge
// and hint, and an arrow to where the panel went. It keeps the panel's
// hairline border, so the guide still shows *where* in the lesson the demo
// sits — the prose around it often says "toggle the pieces below".
//
// Tapping it brings its panel into view on the phone (core.ScrollIntoView,
// the name demoPanel gives itself). A lesson with five demos stacks five
// panels on the phone, and the one the guide's prose is talking about is not
// always the one showing. It is a button to every host's accessibility: the
// role, and a name that says where it goes rather than its three texts in a
// row.
func demoPointer(hint string) core.View {
	return core.ComponentFunc(func(ctx *core.Context) *core.Node {
		th := ctx.Theme()
		return core.Row(
			core.OnClick(func() { core.ScrollIntoView(ctx, demoKeyPrefix+hint) }),
			core.AccessibilityRole(core.RoleButton),
			core.AccessibilityLabel("Show on the phone: "+hint),
			core.Gap(10),
			core.AlignItemsProp(core.AlignItemsCenter),
			core.BorderColor(th.Colors.BorderColor()),
			core.BorderWidth(1),
			core.BorderRadius(12),
			core.Padding(12),
			// FlexShrink(0) on both ends, for demoPanel's reason: the hint is
			// the one child that should give up width when it is long.
			comps.Badge{Text: "TRY IT", Style: []core.StyleProp{core.FlexShrink(0)}},
			core.Box(core.FlexGrow(1), core.FlexShrink(1), caption(hint)),
			core.Text("on the phone →",
				core.FontWeight(core.Bold),
				core.TextColor(th.Colors.Primary),
				core.FlexShrink(0),
			),
		).Render(ctx)
	})
}

// phoneHeader is the thin bar at the top of the phone while a lesson is open:
// which lesson the demos below belong to. The guide scrolls independently, so
// without it the phone would show demos with nothing saying whose.
func phoneHeader(id string) core.View {
	return core.ComponentFunc(func(ctx *core.Context) *core.Node {
		th := ctx.Theme()
		label := "Live demo"
		if e, ok := resolveRoute(id); ok {
			label = e.ID + "  " + e.Title
		}
		// A Separator under the bar rather than a bottom border: core.Style
		// has one border for all four sides.
		return core.Column(
			core.Row(
				core.Gap(8),
				core.AlignItemsProp(core.AlignItemsCenter),
				core.PaddingHorizontal(14),
				core.PaddingVertical(10),
				core.BackgroundColor(th.Colors.Surface),
				comps.Badge{Text: "LIVE", Variant: comps.VariantSuccess, Style: []core.StyleProp{core.FlexShrink(0)}},
				caption(label),
			),
			comps.Separator{},
		).Render(ctx)
	})
}

// phoneSplash is the phone's screen while the contents are open: nothing is
// running yet, and the screen says where the demos will appear.
func phoneSplash() core.View {
	return core.Column(
		core.Justify(core.JustifyCenter),
		core.AlignItemsProp(core.AlignItemsCenter),
		core.Gap(10),
		core.Padding(28),
		titleText("GrMob"),
		core.Text("Pick a lesson from the contents. Its live demo runs here, on the phone, "+
			"while you read the guide beside it.",
			core.Align(core.AlignCenter)),
	)
}

// coveredLesson is the last lesson tree the layout saw, held for the guide to
// show while a screen a demo pushed covers the lesson.
//
// A pointer in a state slot, written during render without a Set, for
// layoutRecord's reason: it is bookkeeping, and a Set would ask for a pass.
type coveredLesson struct {
	mu sync.Mutex
	// lesson is t.current when node was rendered: the memo stands for that
	// lesson only.
	lesson string
	node   *core.Node
}

func (m *coveredLesson) remember(lesson string, n *core.Node) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lesson, m.node = lesson, n
}

func (m *coveredLesson) recall(lesson string) *core.Node {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.lesson != lesson {
		return nil
	}
	return m.node
}

// pushedScreenGuide fills the guide while a demo's pushed screen is on the
// phone: the lesson underneath, as it was last drawn and made inert, with a
// bar under it saying why it does not respond.
//
// # Why the lesson, and why inert
//
// The guide used to show a note naming the lesson, so pushing a screen from a
// chapter 6 demo blanked the text the reader was following, the text that
// says what to try on the pushed screen. The lesson is not rendered while it
// is covered, and rendering it anyway would run its hooks a second time. Its
// last rendering is still a valid tree (a rendered Node is frozen), but its
// callback IDs are not: IDs are positional within a pass (see
// callbackRegistry), and by now they name the pushed screen's handlers. So the
// copy has every callback taken out (see inert) and its controls disabled.
// Reading is all a covered lesson is for, and the pushed screen's own back
// button brings the live one back.
//
// # Why the scroll position survives
//
// The copy keeps the lesson's root key and every node's type and position,
// and the bar goes after it in the pane (see splitView). So covering and
// uncovering the lesson changes its props and nothing else, the host patches
// the same elements in place, and the reader is where they were in the guide
// both times.
//
// Without a memo for the current lesson (the page switched to the split while
// the pushed screen was already up, before this lesson had been drawn),
// pushedScreenNote says where the reader is instead.
func (t *tutorial) pushedScreenGuide(ctx *core.Context) (*core.Node, []core.View) {
	lesson := t.lastLesson.Get().recall(t.current.Get())
	if lesson == nil {
		return t.pushedScreenNote(ctx).Render(ctx), nil
	}
	guide, _, _ := liftDemos(lesson, func(hint string) *core.Node {
		return demoPointer(hint).Render(ctx)
	})
	return inert(guide), []core.View{pushedScreenBar()}
}

// pushedScreenBar sits under the covered lesson in the guide. The page keeps
// it at its natural height (#tutorial-guide-note in wasm/index.html), where the
// guide's other child takes the rest.
func pushedScreenBar() core.View {
	return core.Column(
		core.AccessibilityID(guideNoteID),
		comps.Banner{
			Text: "A demo's screen is open on the phone. Its back button returns you " +
				"here; until then this guide is for reading.",
			Variant: comps.VariantDefault,
		},
	)
}

// inertCommands are node props that are not callbacks but would act again if
// a host met them on a node for the first time: a focus command fires on a
// field that mounts while it is the target (core/focus.go). The copy inert
// makes keeps its nodes in place, so this is belt and braces, for a host that
// rebuilds rather than patches.
var inertCommands = []string{"focusEpoch", "focusAction"}

// inert copies a rendered tree with every callback taken out: each prop named
// on… whose value is a callback ID is dropped, and a node that had one is
// marked core.Style.Disabled, so every host draws and announces it as the
// control it is, switched off. Keys, types and children are kept exactly,
// which is what lets a host patch the live tree into this one in place.
//
// The input is never written: every node is copied, props and style with it.
func inert(n *core.Node) *core.Node {
	if n == nil {
		return nil
	}
	cp := *n
	if len(n.Props) > 0 {
		props := make(map[string]any, len(n.Props))
		dropped := false
		for k, v := range n.Props {
			if id, ok := v.(string); ok && id != "" && strings.HasPrefix(k, "on") {
				dropped = true
				continue
			}
			// A core.Paragraph's link runs carry their callbacks inside the
			// runs prop, one level down; see inertRuns.
			if k == "runs" {
				if runs, ok := v.([]map[string]any); ok {
					if cp, had := inertRuns(runs); had {
						props[k] = cp
						continue
					}
				}
			}
			props[k] = v
		}
		for _, k := range inertCommands {
			if _, ok := props[k]; ok {
				delete(props, k)
			}
		}
		cp.Props = props
		if dropped {
			var st core.Style
			if n.Style != nil {
				st = *n.Style
			}
			st.Disabled = true
			cp.Style = &st
		}
	}
	if len(n.Children) > 0 {
		cp.Children = make([]*core.Node, len(n.Children))
		for i, c := range n.Children {
			cp.Children[i] = inert(c)
		}
	}
	return &cp
}

// inertRuns copies a Paragraph's runs without their callbacks, reporting
// whether any run had one. The link keeps its colour and marks, so the covered
// sentence reads as it did; it just stops being a link.
func inertRuns(runs []map[string]any) ([]map[string]any, bool) {
	found := false
	out := make([]map[string]any, len(runs))
	for i, r := range runs {
		if _, ok := r["cb"]; !ok {
			out[i] = r
			continue
		}
		found = true
		cp := make(map[string]any, len(r))
		for k, v := range r {
			if k != "cb" {
				cp[k] = v
			}
		}
		out[i] = cp
	}
	return out, found
}

// pushedScreenNote fills the guide while a demo's pushed screen is on the
// phone and there is no memo of the lesson under it (see pushedScreenGuide).
// It names the lesson the reader is in and says how to get back to it; the
// pushed screen's own back button is that way (navDemoScreen requires every
// one to have one).
func (t *tutorial) pushedScreenNote(ctx *core.Context) core.View {
	items := []core.PropsAndChildren{
		core.Gap(14),
		core.Padding(20),
	}
	if e, ok := resolveRoute(t.current.Get()); ok {
		items = append(items, lessonHeader(e))
	}
	items = append(items, prose("The demo pushed a screen of its own onto the app's navigation "+
		"stack, and it is running on the phone. Its back button pops it and brings this "+
		"lesson's guide back."))
	return core.Keyed("pushed-screen-note", core.Column(items...))
}

// nodeView puts an already-rendered node back into a view tree. The layout
// works on rendered nodes, and core's containers take views; rendering this
// returns the node untouched, so nothing in it is rendered — or claims a hook
// slot — a second time.
type nodeView struct{ n *core.Node }

func (v nodeView) Render(*core.Context) *core.Node { return v.n }
