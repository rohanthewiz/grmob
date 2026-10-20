package comps

import (
	"fmt"
	"time"

	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/hooks"
)

// ConcernGalleryFrozen is raised, in debug builds only, when a Gallery is
// Controlled but has no OnChange. Every control then asks for a change that
// nobody makes, so the arrows, the dots, the flip and the zoom all look live
// and do nothing.
const ConcernGalleryFrozen = "gallery-frozen"

// Gallery is an immersive picture viewer: one large card at a time, each with
// a title, a subtitle and a description over its picture, a back face of
// details it flips over to, a zoom, a share and a "Learn more" action. The
// visual style, the motion and every label are fields, and each field's zero
// value is a reasonable default, so the shortest call is a working gallery:
//
//	comps.Gallery{Items: []comps.GalleryItem{
//	    {Src: "https://…/fjord.jpg", Alt: "A fjord at dawn",
//	        Title: "Geirangerfjord", Subtitle: "Norway",
//	        Description: "A fifteen-kilometre arm of the Storfjord…",
//	        Details:   []comps.KeyValue{{Key: "Length", Value: "15 km"}},
//	        ActionURL: "https://en.wikipedia.org/wiki/Geirangerfjord",
//	        ShareURL:  "https://example.com/p/fjord"},
//	}}
//
// Every setting at once, with the state owned by the caller:
//
//	view := core.NewState(ctx, comps.GalleryState{})
//	comps.Gallery{
//	    Items:      items,
//	    Controlled: true, State: view.Get(), OnChange: view.Set,
//	    Look:       comps.GalleryLookCinema,
//	    Motion:     comps.GalleryMotion{Transition: comps.GalleryTransitionDeck,
//	        Flip: comps.GalleryFlipRise, DurationMs: 450, Easing: core.EaseOut},
//	    Indicator:  comps.GalleryIndicatorThumbnails,
//	    Zoom:       comps.GalleryZoomFullscreen,
//	    Autoplay:   6 * time.Second,
//	    Wrap:       true,
//	    OnAction:   func(it comps.GalleryItem, i int) { core.Push(ctx, detail(it)) },
//	    OnShare:    func(it comps.GalleryItem, i int) { myShareSheet(it.ShareURL) },
//	}
//
// The tree:
//
//	┌ Column  role=group "Gallery", then Style ──────────────────────────┐
//	│ ┌ ZStack stage  Height (→ ZoomedHeight), clipped, then StageStyle ┐ │
//	│ │ ┌ Keyed card layer × n, last item first ───────────────────────┐ │ │
//	│ │ │ Box 100%×100%, inset; Translate/Opacity/Rotate per Motion    │ │ │
//	│ │ │ └ Theme(card theme) ─ ZStack frame, then CardStyle           │ │ │
//	│ │ │     ├ front ZStack: picture (Fill) + caption band at bottom  │ │ │
//	│ │ │     └ back Scroll: title, description, Details | Back        │ │ │
//	│ │ └──────────────────────────────────────────────────────────────┘ │ │
//	│ │ ┌ Row tap zones (hidden from readers; gone while flipped) ─────┐ │ │
//	│ │ │  ‹ previous  │   tap: flip · hold: zoom   │   next ›         │ │ │
//	│ │ └──────────────────────────────────────────────────────────────┘ │ │
//	│ └─────────────────────────────────────────────────────────────────┘ │
//	│ ┌ Row ──────────────────────────────────────────────────────────┐   │
//	│ │ [‹]          ● ━━ ● ●   (dots | 3 / 12 | thumbnails)      [›] │   │
//	│ └───────────────────────────────────────────────────────────────┘   │
//	│ ┌ Row (wraps) ──────────────────────────────────────────────────┐   │
//	│ │ [⟲ Details] [⤢ Zoom] [↗ Share] [❚❚]            [Learn more]  │   │
//	│ └───────────────────────────────────────────────────────────────┘   │
//	│  Lightbox (Zoom == GalleryZoomFullscreen only)                      │
//	└─────────────────────────────────────────────────────────────────────┘
//
// # Every card is in the tree, and moving between them is a style patch
//
// Each card is a layer of one ZStack, rendered on every pass and moved into
// place with core.Translate, core.Opacity and core.Rotate under a
// core.Transition. Changing the current card therefore changes only styles:
// each target sees an old style and a new one on a node that is displayed
// both before and after, and animates between them on its own frame clock
// (CSS transitions, Compose, SwiftUI). Nothing is stepped from Go and nothing
// crosses the bridge per frame, which is Drawer's "It slides" argument
// applied to n panels instead of one.
//
// The alternative was to render only the current card (and maybe its
// neighbours). A card that enters the tree has no "before" style to animate
// from, so it would appear rather than arrive, and a jump from card 1 to card
// 9 through the thumbnails would need a different path from a step. The
// cost of keeping every card is composition: a gallery of a few dozen
// pictures is cheap, a few hundred should be paged by the caller.
//
// The layers are emitted last item first, so that earlier cards draw over
// later ones. Slide and fade do not care about the order, but the deck does:
// the current card has to sit on top of the cards fanned out behind it, and
// core.ZIndex is honoured on the web only. A fixed order also means the
// layers never move between passes, which would cost the diff a reorder and
// the targets their transitions.
//
// # Motion is a field, and "none" is one of its values
//
// Motion.Transition picks how cards change:
//
//	slide (zero)  the next card slides in from the trailing edge as the
//	              current one leaves by the leading edge; both fade so a jump
//	              of several cards does not drag the ones between across
//	fade          a crossfade in place
//	deck          the following cards are fanned out behind the current one
//	              at small alternating angles; the current card flies off to
//	              the leading edge, and Previous brings it back on top
//	none          cards swap with no motion
//
// Motion.Flip picks how the back face appears: a crossfade (zero), a rise
// from the bottom edge, or none. Every duration is a field whose zero is a
// default, whose negative snaps, and which shares one Easing. Under the
// platform's reduce-motion setting every one of them snaps whatever it says,
// which is core.Transition's rule and is not overridable here.
//
// A flip is not a 3-D turn. core.Rotate turns a node about the axis out of
// the screen, and core has no rotation about the vertical axis, no
// perspective and no scale, so "card turns over" cannot be drawn the same
// way on every target. That is a gap in core's paint transforms rather than
// something this widget can paper over with platform code.
//
// # Touch, mouse and keyboard
//
// The stage carries three invisible tap zones over the current card's front:
// the leading quarter steps back, the trailing quarter steps forward, and the
// middle flips the card when it has a back, the way story viewers work. A
// long press in the middle zooms. The zones are a pointer convenience over
// the buttons below, which do the same things, so the zones are hidden from
// assistive technology and every action has a named control.
//
// A swipe is not among them. core has no drag or swipe gesture; the closest
// primitive is a horizontal core.Scroll, and a scroll reports no position, so
// nothing could say which card a swipe stopped on. A swipe is a core gesture
// primitive waiting to be added, not a widget problem.
//
// While the card is flipped or zoomed in place, Escape (any host with a
// keyboard) and Android's system back put it back, the way a Drawer claims
// both while open. Neither is claimed otherwise.
//
// # State: owned by default, controlled on request
//
// GalleryState is the current index and the flipped, zoomed and paused
// flags. By default the gallery keeps it itself, seeded from State on the
// first pass, so the shortest call above needs no state of its own. With
// Controlled set, State is read on every pass and every change is a request
// to OnChange, the shape of Wizard and SegmentedControl: that is what an app
// that deep-links to a card, or remembers the last one viewed, uses. OnChange
// is called in both modes, so an owned gallery can still be observed.
//
// The two modes share one implementation: the hook is taken either way, and
// only which copy is read differs. The slot count is therefore the same in
// both modes, and a gallery can switch modes without shifting its caller's
// hooks.
//
// Changes are clamped (an index outside Items lands on the nearest card),
// deduplicated (OnChange never hears a no-op), and moving to another card
// always shows its front.
//
// # Hooks
//
// It takes two hook slots, the state and the autoplay timer
// (hooks.UseTimeoutWhile), both before any branch. So a Gallery must be
// rendered unconditionally on every pass: never inside core.If, a
// conditional or a For whose length varies. Back and any other View in
// Items is rendered only on some passes (and inside a card) and must be
// hook-free.
//
// # Looks
//
// Look picks one of four styles. Two follow the app's theme, and two are
// fixed on purpose:
//
//	theme (zero)  a card in the theme's Card fill, radius and shadow, with
//	              the caption on an opaque strip in the same fill
//	minimal       no frame or shadow: the picture with rounded corners and
//	              the caption on the page's own background
//	cinema        black stage, full-bleed picture, caption on a dark
//	              translucent band; the card renders under core.DarkTheme
//	polaroid      a white print with a mat and a soft shadow, inset in the
//	              stage; the card renders under core.DefaultTheme
//
// Cinema is dark in a light app and polaroid is light in a dark app: that is
// what the look means, and it is Lightbox's "dark on every target, in every
// theme" decision for cinema. To keep everything inside the card legible,
// including a Details list and a caller's Back view, the card is rendered
// under a whole bundled theme (core.WithTheme) rather than given a few
// literal ink colours. CardTheme replaces that theme for any look. The
// controls below the stage always follow the app's theme.
//
// # Zoom
//
// Zoom picks what zooming the current card does. The zero value grows it in
// place: the stage animates from Height to ZoomedHeight, and the card drops
// its inset, corner radius and shadow, so the picture fills the stage edge to
// edge. GalleryZoomFullscreen opens the card's picture in a comps.Lightbox
// instead, letterboxed and uncropped, and GalleryZoomOff removes zooming.
// core has no scale transform or pinch gesture, so a zoom is a bigger box,
// not a magnification inside one box.
//
// # Learn more and Share
//
// The action button is drawn for an item with an ActionURL, or for every item
// when OnAction is set. OnAction receives the item; with it nil, the
// ActionURL is opened with core.OpenURL.
//
// The share button is drawn for an item with a ShareURL or ShareText, or for
// every item when OnShare is set. With OnShare nil, the share text (ShareText
// or the Title, then the ShareURL) is put on the clipboard and confirmed by a
// toast, CopyButton's behaviour. core has no system share sheet, so an app
// that wants one passes OnShare and calls its own.
//
// # Accessibility
//
// The root is a labelled group. Each card is a group named "3 of 12, Title"
// (Labels.Position formats the first part), and only the current card is in
// the accessibility tree: the others are AccessibilityHidden and Inert, as a
// shut Drawer is. The card title is a heading, at level 2 unless
// HeadingLevel says otherwise. The picture is named by its Alt and hidden
// without one, Avatar's rule.
//
// Every glyph button has a spoken name. Details states its expansion
// (core.AccessibilityExpanded on the control that toggles it), the in-place
// Zoom and Autoplay state that they are on with core.AccessibilitySelected,
// and the fullscreen Zoom states that it opens a dialog. Each dot or
// thumbnail is a button named for its card, and the current one carries
// core.CurrentTrue. Autoplay always has a pause control, as WCAG 2.2.2 asks of
// content that moves by itself, and pauses while a card is flipped or
// zoomed. The "3 / 12" counter is a status region only when nothing is
// moving by itself; with autoplay running, it would speak on every tick.
//
// # Theme roles read
//
//	Controls     as comps.Button; ghost for the tools, filled for the action
//	Dots         Colors.Primary (current), Colors.Border (the rest)
//	Counter      Typography.Caption, Colors.TextSecondary
//	Thumbnails   Colors.Primary ring on the current one
//	Card         per Look: Components.Card Background/BorderRadius/Shadow,
//	             Colors.Surface, Colors.Background (in the card's theme)
//	Caption      Typography.Subtitle bold, Caption and Body; TextPrimary,
//	             TextSecondary (in the card's theme)
//	Spacing      XS, SM and MD for gaps and insets
type Gallery struct {
	// Items are the cards, in order.
	Items []GalleryItem

	// State is the gallery's state. Owned (Controlled false), it is read on
	// the first pass only, to open on a given card or flipped; controlled,
	// it is what is drawn on every pass.
	State GalleryState

	// Controlled makes State the caller's: every change goes through
	// OnChange, and nothing changes until the caller passes a new State.
	Controlled bool

	// OnChange receives every change, in both modes. Controlled with a nil
	// OnChange reports ConcernGalleryFrozen.
	OnChange func(GalleryState)

	// Look selects the visual style; see "Looks". The zero value follows
	// the app's theme.
	Look GalleryLook

	// CardTheme replaces the theme the card is rendered under, for any
	// Look. Nil uses the Look's own: the app's theme for theme and minimal,
	// core.DarkTheme for cinema, core.DefaultTheme for polaroid.
	CardTheme *core.Theme

	// Motion configures every animation; see "Motion is a field".
	Motion GalleryMotion

	// Indicator selects what sits between the arrows. The zero value is
	// dots for up to ten cards and a "3 / 12" counter beyond that.
	Indicator GalleryIndicator

	// Zoom selects what zooming does; see "Zoom". The zero value grows the
	// card in place.
	Zoom GalleryZoom

	// Wrap lets Previous on the first card go to the last and Next on the
	// last go to the first. Without it the arrows stop at the ends (and are
	// disabled there). Autoplay always wraps.
	Wrap bool

	// Autoplay advances to the next card after each card has been shown
	// this long. Zero leaves the cards still. Moving by hand restarts the
	// delay. It pauses while a card is flipped or zoomed, and a pause button
	// is drawn whenever it is set.
	Autoplay time.Duration

	// Height is the stage's height in px; 0 means 360.
	Height float64

	// ZoomedHeight is the stage's height while zoomed in place, or the
	// Lightbox picture's height for GalleryZoomFullscreen. 0 means one and a
	// half times Height in place, and the Lightbox's own default fullscreen.
	ZoomedHeight float64

	// DescriptionLines caps the description on the front of the card; the
	// back always shows all of it. 0 means 2; negative means no cap.
	DescriptionLines int

	// HeadingLevel is the card title's heading level. 0 means 2, a section
	// of a screen, as a Card title is.
	HeadingLevel int

	// OnAction is called by the action button with the current item. Nil
	// opens the item's ActionURL. See "Learn more and Share".
	OnAction func(item GalleryItem, index int)

	// OnShare is called by the share button with the current item. Nil puts
	// the share text on the clipboard. See "Learn more and Share".
	OnShare func(item GalleryItem, index int)

	// HideControls removes the row of tool buttons and the action button
	// under the stage, for a purely visual gallery. The arrows, the
	// indicator and the tap zones stay.
	HideControls bool

	// Labels are the spoken and visible words; empty fields are English
	// defaults.
	Labels GalleryLabels

	// Style is applied to the root column after its defaults.
	Style []core.StyleProp

	// StageStyle is applied to the stage after its defaults, so it can
	// round, colour or border the stage.
	StageStyle []core.StyleProp

	// CardStyle is applied to every card's frame after the Look's props.
	CardStyle []core.StyleProp
}

// GalleryItem is one card of a Gallery.
type GalleryItem struct {
	// Key identifies the card across passes. Empty uses its index, which is
	// right for a fixed list; set it when items are inserted or removed, so
	// a card keeps its node and its animation.
	Key string

	// Src is the picture's URL, drawn to fill the card (cropped, never
	// distorted).
	Src string

	// Alt describes the picture. Empty hides the picture from assistive
	// technology, and the title and description are what is read.
	Alt string

	// Title, Subtitle and Description are the caption on the front of the
	// card. Any of them may be empty; with all three empty, no caption
	// strip is drawn and the picture fills the card.
	Title       string
	Subtitle    string
	Description string

	// Details are label/value facts on the back of the card, drawn as a
	// comps.KeyValueList. A card with Details or a Back can be flipped; one
	// with neither cannot, and draws no Details button.
	Details []KeyValue

	// Back replaces Details as the back of the card, under its title and
	// description. It is rendered only on some passes, so it must be
	// hook-free.
	Back core.View

	// ActionLabel replaces Labels.Action for this card only.
	ActionLabel string

	// ActionURL is opened by the action button when Gallery.OnAction is
	// nil. Empty, with OnAction nil, draws no action button for this card.
	ActionURL string

	// ShareURL and ShareText are what is shared. With OnShare nil, the
	// clipboard gets ShareText (or Title), then ShareURL.
	ShareURL  string
	ShareText string
}

// GalleryState is what a Gallery shows. It is comparable, so a caller can
// test two for equality, and OnChange is never called with an unchanged one.
type GalleryState struct {
	// Index is the 0-based current card.
	Index int
	// Flipped shows the current card's back.
	Flipped bool
	// Zoomed zooms the current card; see Gallery.Zoom.
	Zoomed bool
	// Paused stops Autoplay.
	Paused bool
}

// GalleryLook is a Gallery's visual style; see "Looks" on Gallery.
type GalleryLook string

const (
	// GalleryLookTheme is a themed card. It is the zero value.
	GalleryLookTheme GalleryLook = ""
	// GalleryLookMinimal is the picture alone, rounded, with no frame.
	GalleryLookMinimal GalleryLook = "minimal"
	// GalleryLookCinema is a full-bleed picture on black in every theme.
	GalleryLookCinema GalleryLook = "cinema"
	// GalleryLookPolaroid is a white print with a mat in every theme.
	GalleryLookPolaroid GalleryLook = "polaroid"
)

// GalleryTransition is how a Gallery moves between cards.
type GalleryTransition string

const (
	// GalleryTransitionSlide slides cards along the inline axis. It is the
	// zero value.
	GalleryTransitionSlide GalleryTransition = ""
	// GalleryTransitionFade crossfades cards in place.
	GalleryTransitionFade GalleryTransition = "fade"
	// GalleryTransitionDeck fans the following cards out behind the current
	// one and flies the current one off when it is left.
	GalleryTransitionDeck GalleryTransition = "deck"
	// GalleryTransitionNone swaps cards with no motion.
	GalleryTransitionNone GalleryTransition = "none"
)

// GalleryFlip is how a Gallery card shows its back.
type GalleryFlip string

const (
	// GalleryFlipFade crossfades the two faces. It is the zero value.
	GalleryFlipFade GalleryFlip = ""
	// GalleryFlipRise slides the back up over the front from the bottom
	// edge.
	GalleryFlipRise GalleryFlip = "rise"
	// GalleryFlipNone swaps the faces with no motion.
	GalleryFlipNone GalleryFlip = "none"
)

// GalleryZoom is what zooming a Gallery card does; see "Zoom" on Gallery.
type GalleryZoom string

const (
	// GalleryZoomExpand grows the card in place. It is the zero value.
	GalleryZoomExpand GalleryZoom = ""
	// GalleryZoomFullscreen opens the picture in a Lightbox.
	GalleryZoomFullscreen GalleryZoom = "fullscreen"
	// GalleryZoomOff removes zooming: no button and no long press.
	GalleryZoomOff GalleryZoom = "off"
)

// GalleryIndicator is what a Gallery draws between its arrows.
type GalleryIndicator string

const (
	// GalleryIndicatorAuto is dots for up to ten cards and a counter
	// beyond. It is the zero value.
	GalleryIndicatorAuto GalleryIndicator = ""
	// GalleryIndicatorDots is a tappable dot per card.
	GalleryIndicatorDots GalleryIndicator = "dots"
	// GalleryIndicatorCounter is "3 / 12".
	GalleryIndicatorCounter GalleryIndicator = "counter"
	// GalleryIndicatorThumbnails is a scrolling strip of tappable
	// thumbnails.
	GalleryIndicatorThumbnails GalleryIndicator = "thumbnails"
	// GalleryIndicatorNone draws nothing between the arrows.
	GalleryIndicatorNone GalleryIndicator = "none"
)

// GalleryMotion configures a Gallery's animations. Every duration is in
// milliseconds: zero is the default noted, and negative snaps.
type GalleryMotion struct {
	// Transition is how cards change; the zero value slides.
	Transition GalleryTransition
	// Flip is how a card shows its back; the zero value crossfades.
	Flip GalleryFlip
	// DurationMs times a change of card; 0 means 320.
	DurationMs int
	// FlipDurationMs times a flip; 0 means 260.
	FlipDurationMs int
	// ZoomDurationMs times an in-place zoom; 0 means 240.
	ZoomDurationMs int
	// Easing is the curve every animation shares; empty is EaseInOut.
	Easing core.Easing
}

// GalleryLabels are a Gallery's words. Every empty field takes the English
// default noted, so a localised app sets the ones it translates.
type GalleryLabels struct {
	// Gallery names the root group; "Gallery".
	Gallery string
	// Previous and Next name the arrows; "Previous", "Next".
	Previous, Next string
	// Details is the flip button's caption and name; "Details".
	Details string
	// Front names the button on a card's back that turns it back over;
	// "Show picture".
	Front string
	// Zoom is the zoom button's caption and name; "Zoom".
	Zoom string
	// Share is the share button's caption and name; "Share".
	Share string
	// Shared is the toast after a clipboard share; "Copied to clipboard".
	Shared string
	// Autoplay names the pause/play toggle; "Autoplay".
	Autoplay string
	// Action is the action button's caption; "Learn more".
	Action string
	// Empty is drawn on the stage when there are no Items; "Nothing to
	// show".
	Empty string
	// Position formats a card's place: "3 of 12" by default. index is
	// 0-based.
	Position func(index, count int) string
}

// The defaults the zero values stand for, named so the doc comments and
// the code cannot drift apart.
const (
	galleryDefaultHeight        = 360.0
	galleryDefaultZoomFactor    = 1.5
	galleryDefaultLines         = 2
	galleryDefaultCardMs        = 320
	galleryDefaultFlipMs        = 260
	galleryDefaultZoomMs        = 240
	galleryAutoDotsMax          = 10
	galleryThumbSize            = 56.0
	galleryDotSize              = 8.0
	galleryDotCurrentWidth      = 20.0
	galleryDeckFanned           = 2     // cards visible behind the current one
	galleryDeckTiltDeg          = 3.0   // per card behind, alternating sides
	galleryDeckDropPx           = 6     // per card behind, downwards
	galleryDeckFlyTiltDeg       = -10.0 // a card that has been left behind
	galleryDeckFlyOffsetPercent = -120
)

// galleryCinemaBand is the cinema caption band: black at 70%. White ink on
// it over a white picture is about 8:1, so the caption is legible over any
// photograph, and it is light enough to leave the picture visible under it.
// The stage and frame reuse lightboxPanel, because a cinema gallery is a
// photo viewer and should be the same black as the Lightbox it can open.
const galleryCinemaBand = "#000000B3"

// Render builds the tree drawn in the type doc.
func (g Gallery) Render(ctx *core.Context) *core.Node {
	// Both hook slots first, before any branch; see "Hooks".
	owned := core.NewState(ctx, g.State)
	n := len(g.Items)

	// latest reads the state a handler should act on. Owned, that is the
	// state slot at the moment of the tap or tick, not the copy this pass
	// rendered: an autoplay tick that lands before the re-render of the
	// previous one would otherwise advance from a stale index and stand still.
	// Controlled, the caller's State is the only truth there is.
	latest := func() GalleryState {
		s := g.State
		if !g.Controlled {
			s = owned.Get()
		}
		s.Index = galleryClamp(s.Index, n)
		return s
	}
	cur := latest()

	// update is every control's single write path: clamp, show the front of
	// a newly current card, drop a no-op, then write (owned) and report.
	update := func(mut func(*GalleryState)) {
		prev := latest()
		next := prev
		mut(&next)
		next.Index = galleryClamp(next.Index, n)
		if next.Index != prev.Index {
			next.Flipped = false
		}
		if next == prev {
			return
		}
		if !g.Controlled {
			owned.Set(next)
		}
		if g.OnChange != nil {
			g.OnChange(next)
		}
	}

	// Autoplay is a one-shot timeout re-armed per card, not an interval:
	// the deps (the index and the interval) restart the delay whenever the
	// card changes, so a card reached by hand gets its full time on screen
	// before autoplay moves on, and a changed Autoplay takes effect on the
	// next pass. hooks.UseIntervalWhile fixes its duration on the first pass
	// and keeps its own beat, which is wrong on both counts here. While
	// inactive the hook arms nothing, so a still gallery holds no timer.
	playing := g.Autoplay > 0 && n > 1 && !cur.Paused && !cur.Flipped && !cur.Zoomed
	hooks.UseTimeoutWhile(ctx, playing, func() {
		update(func(s *GalleryState) { s.Index = galleryStep(s.Index, 1, n, true) })
	}, g.Autoplay, cur.Index, g.Autoplay)

	if core.IsDebugMode() && g.Controlled && g.OnChange == nil {
		core.ReportConcern(ConcernGalleryFrozen,
			"Gallery is Controlled with no OnChange, so none of its controls can change the card, flip it or zoom it")
	}

	t := ctx.Theme()
	r := galleryRender{g: g, t: t, n: n, cur: cur, update: update, playing: playing}
	r.look = g.lookSpec(t)
	r.motion = g.Motion.resolve()

	root := make([]core.PropsAndChildren, 0, len(g.Style)+10)
	root = append(root,
		// A theme Column carries the screen inset; the gallery is a block
		// inside a screen, so it clears it, and the stage runs edge to edge
		// of whatever the caller put it in.
		core.Padding(0),
		core.Gap(float64(t.Spacing.SM)),
		core.Width("100%"),
		core.AccessibilityRole(core.RoleGroup),
		core.AccessibilityLabel(orDefault(g.Labels.Gallery, "Gallery")),
	)
	// Escape and system back put a flipped or zoomed-in-place card back,
	// and are claimed only while there is something to put back, so a still
	// gallery never steals the press from a Navigator or a Drawer.
	if undo := r.undoLast(); undo != nil {
		root = append(root, core.OnEscape(undo), core.OnBack(undo))
	}
	root = append(root, asProps(g.Style)...)
	root = append(root, r.stage())
	if n > 0 {
		root = append(root, r.navRow())
		if !g.HideControls {
			root = append(root, r.toolRow())
		}
		if g.Zoom == GalleryZoomFullscreen {
			root = append(root, r.lightbox())
		}
	}
	return core.Column(root...).Render(ctx)
}

// galleryRender carries one pass's resolved settings through the helpers,
// so each builder takes no long parameter list and none of them re-derives
// the state, the look or the motion.
type galleryRender struct {
	g       Gallery
	t       *core.Theme
	n       int
	cur     GalleryState
	update  func(func(*GalleryState))
	playing bool
	look    galleryLookSpec
	motion  galleryMotion
}

// undoLast returns what Escape and back do now, or nil when they should not
// be claimed. A flip is undone before a zoom, because the back face is the
// thing drawn on top.
func (r galleryRender) undoLast() func() {
	switch {
	case r.n == 0:
		return nil
	case r.cur.Flipped:
		return func() { r.update(func(s *GalleryState) { s.Flipped = false }) }
	case r.cur.Zoomed && r.g.Zoom == GalleryZoomExpand:
		return func() { r.update(func(s *GalleryState) { s.Zoomed = false }) }
	}
	return nil
}

// expanded reports whether the card is zoomed in place this pass.
func (r galleryRender) expanded() bool {
	return r.cur.Zoomed && r.g.Zoom == GalleryZoomExpand
}

// stageHeight is the stage's height this pass: Height, or ZoomedHeight while
// zoomed in place.
func (r galleryRender) stageHeight() float64 {
	h := r.g.Height
	if h <= 0 {
		h = galleryDefaultHeight
	}
	if !r.expanded() {
		return h
	}
	if r.g.ZoomedHeight > 0 {
		return r.g.ZoomedHeight
	}
	return h * galleryDefaultZoomFactor
}

// inset is the gap between the stage edge and a card. The deck needs one
// even for looks that have none, or the fanned cards behind would have no
// room to show their corners; a zoomed card has none, which is what zooming
// in place means.
func (r galleryRender) inset() int {
	if r.expanded() {
		return 0
	}
	in := r.look.inset
	if r.g.Motion.Transition == GalleryTransitionDeck && in < r.t.Spacing.MD {
		in = r.t.Spacing.MD
	}
	return in
}

// stage builds the clipped ZStack of card layers plus the tap-zone layer.
func (r galleryRender) stage() core.View {
	items := make([]core.PropsAndChildren, 0, len(r.g.StageStyle)+r.n+8)
	items = append(items,
		core.Width("100%"),
		core.Height(px(r.stageHeight())),
		// Clips the cards that are translated off the stage, and the deck's
		// fanned corners, to the stage's own box.
		core.Overflow("hidden"),
		r.motion.zoom,
	)
	if r.look.stage != "" {
		items = append(items, core.BackgroundColor(r.look.stage))
	}
	items = append(items, asProps(r.g.StageStyle)...)

	if r.n == 0 {
		// Nothing to show: say so in the stage's place, so a gallery still
		// loading occupies the space it will fill.
		items = append(items, core.Text(orDefault(r.g.Labels.Empty, "Nothing to show"),
			core.UseStyle(r.t.Typography.Body),
			core.TextColor(r.t.Colors.TextSecondary)))
		return core.ZStack(items...)
	}

	// Last item first; see "Every card is in the tree".
	for i := r.n - 1; i >= 0; i-- {
		items = append(items, r.cardLayer(i))
	}
	items = append(items, r.tapZones())
	return core.ZStack(items...)
}

// cardLayer builds card i's stage layer: a full-stage Box positioned by the
// transition, holding the card frame under the card's theme.
func (r galleryRender) cardLayer(i int) core.View {
	it := r.g.Items[i]
	current := i == r.cur.Index
	inset := r.inset()

	items := []core.PropsAndChildren{
		core.Width("100%"),
		core.Height("100%"),
		core.Padding(inset),
		r.motion.card,
		core.AccessibilityRole(core.RoleGroup),
		core.AccessibilityLabel(joinNonEmpty(", ", r.position(i), it.Title)),
	}
	items = append(items, r.placement(i-r.cur.Index)...)
	items = append(items, core.ZIndex(r.zOrder(i-r.cur.Index)))
	if !current {
		// Out of reach, as a shut Drawer's panel is: no reader finds a card
		// that is off the stage or behind the current one, and on the web Tab
		// and the pointer pass it by.
		items = append(items, core.AccessibilityHidden(), core.Inert(true))
	}

	frame := r.frame(i, r.stageHeight()-2*float64(inset))
	// Under the card's own theme when the look has one, so a Details list or
	// a caller's Back view inside it reads that theme's inks. See "Looks".
	if r.look.theme != r.t {
		frame = core.WithTheme(r.look.theme, frame)
	}
	items = append(items, frame)
	return core.Keyed("card:"+r.key(i), core.Box(items...))
}

// placement is where a card sits relative to the current one, d cards away
// (negative is behind, positive is ahead), for the configured transition.
//
//	        d = -1           d = 0            d = +1
//	slide   ◀ -100%, clear   0, opaque        +100% ▶, clear
//	fade    0, clear         0, opaque        0, clear
//	deck    -120% ↺10°,      0, opaque        0 ↻3° ↓6px opaque; +2 ↺6°
//	        clear                             ↓12px; beyond +2 clear
//	none    clear            opaque           clear
//
// Every arm states every property it uses, opacity included, so a card moving
// from one pose to another always has both ends of each property to
// interpolate, and one arriving at 0 states Opacity(1) explicitly, because an
// unset opacity in the patch would not undo a previous Opacity(0).
func (r galleryRender) placement(d int) []core.PropsAndChildren {
	opacity := func(on bool) core.StyleProp {
		if on {
			return core.Opacity(1)
		}
		return core.Opacity(0)
	}
	switch r.g.Motion.Transition {
	case GalleryTransitionSlide:
		// Clamped to one stage width: a card several away waits just off the
		// edge it will come from, and a jump of several cards moves only the
		// two that are seen. The fade on everything but the current card is
		// what keeps the cards in between, which cross from one side to the
		// other on a jump, from being dragged across the stage visibly.
		off := galleryClampInt(d, -1, 1) * 100
		return []core.PropsAndChildren{
			core.Translate(fmt.Sprintf("%d%%", off), ""),
			opacity(d == 0),
		}
	case GalleryTransitionDeck:
		switch {
		case d < 0:
			return []core.PropsAndChildren{
				core.Translate(fmt.Sprintf("%d%%", galleryDeckFlyOffsetPercent), ""),
				core.Rotate(galleryDeckFlyTiltDeg),
				opacity(false),
			}
		case d == 0:
			// Rotate(0) and an empty Translate are written out, not left
			// unset, so a card returning from the fan or the fly-off has a
			// zero end to animate to.
			return []core.PropsAndChildren{core.Translate("", ""), core.Rotate(0), opacity(true)}
		default:
			fan := galleryClampInt(d, 1, galleryDeckFanned)
			tilt := galleryDeckTiltDeg * float64(fan)
			if fan%2 == 0 {
				tilt = -tilt
			}
			return []core.PropsAndChildren{
				core.Translate("", fmt.Sprintf("%dpx", galleryDeckDropPx*fan)),
				core.Rotate(tilt),
				opacity(d <= galleryDeckFanned),
			}
		}
	default:
		// Fade and none differ only in duration, which motion.card carries.
		return []core.PropsAndChildren{opacity(d == 0)}
	}
}

// zOrder is the web's paint order for a card d away from the current one,
// restating the child order the natives already paint by.
//
// On the web, a translated or rotated element is its own stacking context
// and paints above a sibling that is neither, whatever their order in the
// document. The current card is the one card at rest, with no transform, so
// under the deck the cards fanned out behind it were painting over it.
// core.ZIndex is honoured by both web targets and ignored by the natives,
// which is exactly the split needed: the natives paint by child order (last
// item first, see "Every card is in the tree"), and this gives the web the
// same order explicitly. Cards already left behind (d < 0) are above the
// current one, as they fly off over it and come back on top of it; the
// cards ahead descend from the current one. Every value is at least 1,
// because zero is core's "unset".
func (r galleryRender) zOrder(d int) int {
	switch {
	case d < 0:
		return r.n + 2
	case d == 0:
		return r.n + 1
	}
	return r.n + 1 - d
}

// tapZOrder is the web's paint order for the tap-zone layer: one above the
// highest card (zOrder's n+2), so the zones take the pointer.
//
// The layer is the stage's last child, which is all the natives need, but on
// the web child order loses to z-index: every card layer carries one, so a
// layer with none painted under them all, the card's picture took every
// click, and tapping the photo flipped nothing and the edges stepped
// nowhere. Hit testing follows paint order, so the zones have to be stated
// above the cards rather than merely after them.
func (r galleryRender) tapZOrder() int { return r.n + 3 }

// frame builds a card: a clipped ZStack of the front and back faces, in the
// look's fill, radius and shadow, with CardStyle last.
func (r galleryRender) frame(i int, height float64) core.View {
	zoomed := r.expanded()

	items := make([]core.PropsAndChildren, 0, len(r.g.CardStyle)+10)
	items = append(items,
		core.Width("100%"),
		core.Height(px(height)),
		// The faces are clipped to the frame's rounded corners, and the
		// rising back face waits below the frame's bottom edge.
		core.Overflow("hidden"),
		r.motion.zoom,
	)
	if r.look.frame != "" {
		items = append(items, core.BackgroundColor(r.look.frame))
	}
	// Zooming in place drops the frame's corners and lift, so the picture
	// meets the stage edge to edge. Stated as zero rather than left out, so
	// the change animates back and forth like the height does.
	if zoomed {
		items = append(items, core.BorderRadius(0), core.Shadow(0))
	} else {
		items = append(items, core.BorderRadius(r.look.radius), core.Shadow(r.look.shadow))
	}
	items = append(items, asProps(r.g.CardStyle)...)
	items = append(items, r.front(i, height), r.back(i, height))
	return core.ZStack(items...)
}

// front builds the picture with its caption strip along the bottom.
//
// The caption overlays the bottom of the picture in every look rather than
// sitting under it in a column. The picture is drawn Fill, so covering its
// bottom only crops a little more of it, and the overlay lets the picture
// take a known pixel height. A column would need the caption's height to
// size the picture, and that height is not known in Go; giving the picture
// a flexible height instead relies on how each target sizes an image inside
// a flex column, which they do not agree on. The looks that read as
// "caption below" paint the strip opaque in the frame's fill.
func (r galleryRender) front(i int, height float64) core.View {
	it := r.g.Items[i]
	flipped := r.cur.Flipped && i == r.cur.Index
	mat := r.look.mat
	if r.expanded() {
		mat = 0
	}

	img := []core.StyleProp{
		core.Width("100%"),
		core.Height(px(height - 2*float64(mat))),
		r.motion.zoom,
	}
	if r.look.imageRadius > 0 && !r.expanded() {
		img = append(img, core.BorderRadius(r.look.imageRadius))
	}
	if it.Alt != "" {
		img = append(img, core.AccessibilityLabel(it.Alt))
	} else {
		img = append(img, core.AccessibilityHidden())
	}

	items := []core.PropsAndChildren{
		core.Width("100%"),
		core.Height(px(height)),
		r.motion.flip,
	}
	items = append(items, r.facePose(false, flipped)...)
	items = append(items,
		core.Box(core.Width("100%"), core.Padding(mat), r.motion.zoom,
			core.ImageWithMode(it.Src, core.ContentModeFill, img...)),
	)
	if caption := r.caption(it, i == r.cur.Index); caption != nil {
		items = append(items, caption)
	}
	return core.ZStack(items...)
}

// caption builds the title, subtitle and clamped description strip, or nil
// when the item has none of the three.
//
// Only the current card's caption is drawn opaque. The deck fans the cards
// ahead out below the current one, and their caption strips peeked out
// under its bottom edge as stray half-lines of text; the slide and fade
// hide those cards anyway. Faded rather than removed, so it fades in with
// the card that arrives, on the card's own timing.
func (r galleryRender) caption(it GalleryItem, current bool) core.View {
	if it.Title == "" && it.Subtitle == "" && it.Description == "" {
		return nil
	}
	ct := r.look.theme
	items := []core.PropsAndChildren{
		core.StackAlign(core.StackAlignBottom),
		core.Width("100%"),
		core.Padding(0),
		core.PaddingHorizontal(ct.Spacing.MD),
		core.PaddingVertical(ct.Spacing.SM),
		core.Gap(2),
		r.motion.card,
	}
	if current {
		items = append(items, core.Opacity(1))
	} else {
		items = append(items, core.Opacity(0))
	}
	if r.look.band != "" {
		items = append(items, core.BackgroundColor(r.look.band))
	}
	if it.Title != "" {
		title := []core.StyleProp{
			core.UseStyle(ct.Typography.Subtitle),
			core.FontWeight(core.Bold),
			// Primary ink stated: the Subtitle scale carries a secondary tone
			// in some themes, and the card's name should not read as a caption.
			core.TextColor(ct.Colors.TextPrimary),
			core.MaxLines(1),
		}
		title = append(title, headingProps(r.g.HeadingLevel, headingLevelSection)...)
		items = append(items, core.Text(it.Title, title...))
	}
	if it.Subtitle != "" {
		items = append(items, core.Text(it.Subtitle,
			core.UseStyle(ct.Typography.Caption),
			core.TextColor(ct.Colors.TextSecondary),
			core.MaxLines(1)))
	}
	if it.Description != "" {
		desc := []core.StyleProp{
			core.UseStyle(ct.Typography.Body),
			core.TextColor(ct.Colors.TextPrimary),
		}
		if lines := r.descriptionLines(); lines > 0 {
			desc = append(desc, core.MaxLines(lines))
		}
		items = append(items, core.Text(it.Description, desc...))
	}
	return core.Column(items...)
}

// back builds the back face: a scrolling column with a "show picture"
// button, the title, the whole description, and Back or the Details list.
func (r galleryRender) back(i int, height float64) core.View {
	it := r.g.Items[i]
	ct := r.look.theme
	flipped := r.cur.Flipped && i == r.cur.Index

	fill := r.look.frame
	if fill == "" {
		// The minimal look has no frame, and the back has to cover the
		// picture, so it takes the card theme's panel fill.
		fill = ct.Colors.Surface
	}

	items := []core.PropsAndChildren{
		core.Width("100%"),
		core.Height(px(height)),
		core.BackgroundColor(fill),
		r.motion.flip,
	}
	items = append(items, r.facePose(true, flipped)...)

	body := []core.PropsAndChildren{
		core.Padding(ct.Spacing.MD),
		core.Gap(float64(ct.Spacing.SM)),
	}
	head := []core.PropsAndChildren{
		core.Padding(0),
		core.Gap(float64(ct.Spacing.SM)),
		core.AlignItemsProp(core.AlignItemsCenter),
	}
	// The title here is drawn bold but is not a second heading: the front's
	// title already is the card's heading, and readers meet only one face.
	head = append(head, core.Text(it.Title,
		core.UseStyle(ct.Typography.Subtitle),
		core.FontWeight(core.Bold),
		core.TextColor(ct.Colors.TextPrimary),
		core.FlexGrow(1),
		core.FlexBasis("0"),
	))
	// The turn-back button, on the back itself, so a touch user can flip
	// back with the controls hidden. Only the current card's is live.
	turn := func() {}
	if flipped {
		turn = func() { r.update(func(s *GalleryState) { s.Flipped = false }) }
	}
	head = append(head, Button{
		Label:              "⟲",
		OnTap:              turn,
		Emphasis:           EmphasisGhost,
		AccessibilityLabel: orDefault(r.g.Labels.Front, "Show picture"),
	})
	body = append(body, core.Row(head...))
	if it.Description != "" {
		body = append(body, core.Text(it.Description,
			core.UseStyle(ct.Typography.Body),
			core.TextColor(ct.Colors.TextPrimary)))
	}
	switch {
	case it.Back != nil:
		// The slot wins over the simple field, as everywhere in comps.
		body = append(body, it.Back)
	case len(it.Details) > 0:
		body = append(body, KeyValueList{
			Rows:     it.Details,
			Dividers: true,
			Label:    joinNonEmpty(" ", it.Title, orDefault(r.g.Labels.Details, "Details")),
		})
	}
	items = append(items, core.Column(body...))
	return core.Scroll(items...)
}

// facePose is the style that shows or hides one face of a card. back says
// which face; flipped is whether the card shows its back.
//
//	         front shown        back shown
//	fade     front 1, back 0    front 0, back 1
//	rise     back below +100%   back at 0, over the front
//	none     as fade, with no duration (motion.flip carries the timing)
//
// The hidden face is also out of the accessibility tree and inert, so a
// reader meets one face at a time and, on the web, Tab and the pointer pass
// through a face that is not showing.
func (r galleryRender) facePose(back, flipped bool) []core.PropsAndChildren {
	shown := back == flipped
	var out []core.PropsAndChildren
	if r.g.Motion.Flip == GalleryFlipRise {
		if back {
			y := "100%"
			if shown {
				// "0" rather than "": an empty axis would leave the
				// previous translation where it was.
				y = "0"
			}
			out = append(out, core.Translate("", y))
		}
	} else if shown {
		out = append(out, core.Opacity(1))
	} else {
		out = append(out, core.Opacity(0))
	}
	if !shown {
		out = append(out, core.AccessibilityHidden(), core.Inert(true))
	}
	return out
}

// tapZones builds the invisible pointer layer over the stage: previous,
// flip (with a long press to zoom), next. See "Touch, mouse and keyboard".
//
// The layer is the same three boxes on every pass, so the tree keeps its
// shape; a zone with nothing to do gets no handler at all, because a
// registered no-op would make an inert zone look tappable to a host (the
// Drawer scrim's rule). While the card is flipped the layer is not
// displayed, so the back face, which may hold a caller's buttons, takes the
// taps.
func (r galleryRender) tapZones() core.View {
	it := r.g.Items[r.cur.Index]
	zone := func(grow float64) []core.PropsAndChildren {
		return []core.PropsAndChildren{
			core.FlexGrow(grow),
			// Shares that ignore padding, on the web; see the comps flex note.
			core.FlexBasis("0"),
			core.Height("100%"),
		}
	}
	prev, mid, next := zone(1), zone(2), zone(1)
	if r.hasPrev() {
		prev = append(prev, core.OnClick(r.step(-1)))
	}
	if r.hasNext() {
		next = append(next, core.OnClick(r.step(1)))
	}
	if galleryFlippable(it) {
		mid = append(mid, core.OnClick(r.toggleFlip()))
	}
	if r.g.Zoom != GalleryZoomOff {
		mid = append(mid, core.OnLongPress(r.toggleZoom()))
	}

	items := []core.PropsAndChildren{
		core.Padding(0),
		core.Gap(0),
		core.Width("100%"),
		core.Height("100%"),
		// Above every card on the web; see tapZOrder.
		core.ZIndex(r.tapZOrder()),
		// A pointer convenience over named controls; see "Touch, mouse and
		// keyboard".
		core.AccessibilityHidden(),
	}
	if r.cur.Flipped {
		items = append(items, core.Display(core.DisplayNone))
	}
	items = append(items, core.Box(prev...), core.Box(mid...), core.Box(next...))
	return core.Row(items...)
}

// navRow builds [‹] indicator [›]. A single card has nowhere to go, so the
// row is drawn without arrows or indicator, as nothing.
func (r galleryRender) navRow() core.View {
	if r.n < 2 {
		return nil
	}
	items := []core.PropsAndChildren{
		core.Padding(0),
		core.Gap(float64(r.t.Spacing.SM)),
		core.AlignItemsProp(core.AlignItemsCenter),
		core.Width("100%"),
	}
	items = append(items, r.arrow(-1))
	// The indicator box grows even when empty, so the arrows stay at the
	// ends. MinWidth 0 lets it shrink below its content on the web, where a
	// flex item's minimum is otherwise its content width: without it a long
	// thumbnail strip widened the row and pushed Next off the screen instead
	// of scrolling. Thumbnails start at the leading edge rather than the
	// centre, because a centred strip that overflows loses its first
	// thumbnails off the leading edge, out of the scroll's reach.
	justify := core.JustifyCenter
	if r.g.Indicator == GalleryIndicatorThumbnails {
		justify = core.JustifyStart
	}
	items = append(items, core.Row(
		core.Padding(0),
		core.FlexGrow(1),
		core.FlexBasis("0"),
		core.MinWidth("0"),
		core.Justify(justify),
		core.AlignItemsProp(core.AlignItemsCenter),
		r.indicator(),
	))
	items = append(items, r.arrow(1))
	return core.Row(items...)
}

// arrow builds the previous (delta -1) or next (delta +1) button, disabled
// at an end when the gallery does not wrap.
func (r galleryRender) arrow(delta int) core.View {
	glyph, name, able := "‹", orDefault(r.g.Labels.Previous, "Previous"), r.hasPrev()
	if delta > 0 {
		glyph, name, able = "›", orDefault(r.g.Labels.Next, "Next"), r.hasNext()
	}
	return Button{
		Label:              glyph,
		OnTap:              r.step(delta),
		Emphasis:           EmphasisGhost,
		Disabled:           !able,
		AccessibilityLabel: name,
	}
}

// indicator builds the configured indicator, or nil for none.
func (r galleryRender) indicator() core.View {
	kind := r.g.Indicator
	if kind == GalleryIndicatorAuto {
		kind = GalleryIndicatorDots
		if r.n > galleryAutoDotsMax {
			kind = GalleryIndicatorCounter
		}
	}
	switch kind {
	case GalleryIndicatorDots:
		return r.dots()
	case GalleryIndicatorCounter:
		return r.counter()
	case GalleryIndicatorThumbnails:
		return r.thumbnails()
	}
	return nil
}

// dots builds one tappable dot per card; the current one is a wider pill in
// the primary colour, and the width change animates with the cards.
func (r galleryRender) dots() core.View {
	items := []core.PropsAndChildren{core.Padding(0), core.Gap(0), core.AlignItemsProp(core.AlignItemsCenter)}
	for i := range r.g.Items {
		current := i == r.cur.Index
		w, fill := galleryDotSize, r.t.Colors.BorderColor()
		if current {
			w, fill = galleryDotCurrentWidth, r.t.Colors.Primary
		}
		dot := core.Box(
			core.Width(px(w)),
			core.Height(px(galleryDotSize)),
			core.BorderRadius(galleryDotSize/2),
			core.BackgroundColor(fill),
			r.motion.card,
		)
		items = append(items, core.Keyed("dot:"+r.key(i), r.jumpButton(i, current,
			// The padding is the touch target around an 8px dot.
			[]core.PropsAndChildren{core.Padding(r.t.Spacing.XS)}, dot)))
	}
	return core.Row(items...)
}

// counter builds "3 / 12", a status region unless autoplay is moving it.
func (r galleryRender) counter() core.View {
	props := []core.StyleProp{
		core.UseStyle(r.t.Typography.Caption),
		core.TextColor(r.t.Colors.TextSecondary),
		core.AccessibilityLabel(r.position(r.cur.Index)),
	}
	if r.g.Autoplay <= 0 {
		props = append(props, core.AccessibilityRole(core.RoleStatus))
	}
	return core.Text(fmt.Sprintf("%d / %d", r.cur.Index+1, r.n), props...)
}

// thumbnails builds a sideways-scrolling strip of tappable thumbnails, the
// current one ringed. The strip is a horizontal core.Scroll, so it pans by
// touch, trackpad and wheel natively on every target.
func (r galleryRender) thumbnails() core.View {
	strip := []core.PropsAndChildren{core.Padding(0), core.Gap(float64(r.t.Spacing.XS))}
	for i, it := range r.g.Items {
		current := i == r.cur.Index
		ring := "transparent"
		if current {
			ring = r.t.Colors.Primary
		}
		img := core.ImageWithMode(it.Src, core.ContentModeFill,
			core.Width(px(galleryThumbSize)),
			core.Height(px(galleryThumbSize)),
			core.BorderRadius(4),
			// The button around it carries the name.
			core.AccessibilityHidden())
		strip = append(strip, core.Keyed("thumb:"+r.key(i), r.jumpButton(i, current,
			[]core.PropsAndChildren{
				core.Padding(2),
				core.BorderWidth(2),
				core.BorderColor(ring),
				core.BorderRadius(6),
				r.motion.card,
			}, img)))
	}
	return core.Scroll(core.Horizontal(), core.MaxWidth("100%"), core.Row(strip...))
}

// jumpButton wraps a dot or a thumbnail in a tappable Box named for card i,
// with the current card marked by core.CurrentTrue.
func (r galleryRender) jumpButton(i int, current bool, style []core.PropsAndChildren, child core.View) core.View {
	items := []core.PropsAndChildren{
		core.OnClick(func() { r.update(func(s *GalleryState) { s.Index = i }) }),
		core.AccessibilityRole(core.RoleButton),
		core.AccessibilityLabel(joinNonEmpty(", ", r.position(i), r.g.Items[i].Title)),
	}
	if current {
		items = append(items, core.AccessibilityCurrent(core.CurrentTrue))
	}
	items = append(items, style...)
	items = append(items, child)
	return core.Box(items...)
}

// toolRow builds [Details] [Zoom] [Share] [Autoplay] … [Learn more]. Each
// tool is drawn only when it does something for the current card, and the
// row wraps on a narrow screen rather than squeezing its labels.
func (r galleryRender) toolRow() core.View {
	it := r.g.Items[r.cur.Index]
	i := r.cur.Index
	l := r.g.Labels

	items := []core.PropsAndChildren{
		core.Padding(0),
		core.Gap(float64(r.t.Spacing.XS)),
		core.AlignItemsProp(core.AlignItemsCenter),
		core.FlexWrap(true),
		core.Width("100%"),
	}
	if galleryFlippable(it) {
		name := orDefault(l.Details, "Details")
		items = append(items, Button{
			Label:              "⟲ " + name,
			OnTap:              r.toggleFlip(),
			Emphasis:           EmphasisGhost,
			AccessibilityLabel: name,
			// One stable name, with the state stated on the control that
			// toggles it.
			Style: []core.StyleProp{core.AccessibilityExpanded(core.ExpandedWhen(r.cur.Flipped))},
		})
	}
	if r.g.Zoom != GalleryZoomOff {
		name := orDefault(l.Zoom, "Zoom")
		state := core.AccessibilitySelected(core.SelectedWhen(r.cur.Zoomed))
		if r.g.Zoom == GalleryZoomFullscreen {
			// Not a toggle here: it presents a Lightbox, which is a dialog.
			state = core.AccessibilityHasPopup(core.PopupDialog)
		}
		items = append(items, Button{
			Label:              "⤢ " + name,
			OnTap:              r.toggleZoom(),
			Emphasis:           EmphasisGhost,
			AccessibilityLabel: name,
			Style:              []core.StyleProp{state},
		})
	}
	if r.g.OnShare != nil || it.ShareURL != "" || it.ShareText != "" {
		name := orDefault(l.Share, "Share")
		items = append(items, Button{
			Label:              "↗ " + name,
			OnTap:              r.share(it, i),
			Emphasis:           EmphasisGhost,
			AccessibilityLabel: name,
		})
	}
	if r.g.Autoplay > 0 && r.n > 1 {
		glyph := "❚❚"
		if r.cur.Paused {
			glyph = "▶"
		}
		items = append(items, Button{
			Label:              glyph,
			OnTap:              func() { r.update(func(s *GalleryState) { s.Paused = !s.Paused }) },
			Emphasis:           EmphasisGhost,
			AccessibilityLabel: orDefault(l.Autoplay, "Autoplay"),
			// Selected while autoplay is on, not while it is moving this
			// instant: a flip pauses the motion without switching it off.
			Style: []core.StyleProp{core.AccessibilitySelected(core.SelectedWhen(!r.cur.Paused))},
		})
	}
	if r.g.OnAction != nil || it.ActionURL != "" {
		// A sibling that grows pushes the action to the trailing edge; see
		// the comps note on JustifyBetween.
		items = append(items, core.Box(core.FlexGrow(1)))
		items = append(items, Button{
			Label: orDefault(it.ActionLabel, orDefault(l.Action, "Learn more")),
			OnTap: r.action(it, i),
		})
	}
	return core.Row(items...)
}

// lightbox builds the fullscreen zoom: the current picture in a Lightbox,
// open while Zoomed. It is in the tree on every pass, as a Modal is.
func (r galleryRender) lightbox() core.View {
	it := r.g.Items[r.cur.Index]
	return Lightbox{
		Src:       it.Src,
		Alt:       it.Alt,
		Caption:   it.Title,
		Open:      r.cur.Zoomed,
		OnDismiss: func() { r.update(func(s *GalleryState) { s.Zoomed = false }) },
		Height:    r.g.ZoomedHeight,
	}
}

// step returns the handler that moves delta cards, wrapping only when the
// gallery does.
func (r galleryRender) step(delta int) func() {
	n, wrap := r.n, r.g.Wrap
	return func() {
		r.update(func(s *GalleryState) { s.Index = galleryStep(s.Index, delta, n, wrap) })
	}
}

func (r galleryRender) toggleFlip() func() {
	return func() { r.update(func(s *GalleryState) { s.Flipped = !s.Flipped }) }
}

func (r galleryRender) toggleZoom() func() {
	return func() { r.update(func(s *GalleryState) { s.Zoomed = !s.Zoomed }) }
}

// share returns the share button's handler: the caller's OnShare, or the
// clipboard with a confirming toast and haptic, CopyButton's behaviour.
// The item is captured by value, so a tap acts on the card this pass drew.
func (r galleryRender) share(it GalleryItem, i int) func() {
	if r.g.OnShare != nil {
		onShare := r.g.OnShare
		return func() { onShare(it, i) }
	}
	text := galleryShareText(it)
	message := orDefault(r.g.Labels.Shared, "Copied to clipboard")
	return func() {
		// Guarded: an empty write would clear the clipboard; see CopyButton.
		if text == "" {
			return
		}
		core.WriteClipboard(text)
		core.Haptic(core.HapticLight)
		core.ShowToast(message)
	}
}

// action returns the action button's handler: the caller's OnAction, or the
// item's ActionURL.
func (r galleryRender) action(it GalleryItem, i int) func() {
	if r.g.OnAction != nil {
		onAction := r.g.OnAction
		return func() { onAction(it, i) }
	}
	url := it.ActionURL
	return func() {
		if url != "" {
			core.OpenURL(url)
		}
	}
}

func (r galleryRender) hasPrev() bool {
	return r.n > 1 && (r.g.Wrap || r.cur.Index > 0)
}

func (r galleryRender) hasNext() bool {
	return r.n > 1 && (r.g.Wrap || r.cur.Index < r.n-1)
}

func (r galleryRender) key(i int) string {
	return galleryItemKey(r.g.Items[i], i)
}

func (r galleryRender) position(i int) string {
	if r.g.Labels.Position != nil {
		return r.g.Labels.Position(i, r.n)
	}
	return fmt.Sprintf("%d of %d", i+1, r.n)
}

func (r galleryRender) descriptionLines() int {
	switch {
	case r.g.DescriptionLines == 0:
		return galleryDefaultLines
	case r.g.DescriptionLines < 0:
		return 0
	}
	return r.g.DescriptionLines
}

// galleryLookSpec is a Look resolved against a theme: everything the card
// builders read, so each Look is one table row rather than a branch in
// every builder.
type galleryLookSpec struct {
	theme       *core.Theme // the theme the card is rendered under
	stage       string      // the stage's fill; "" for none
	frame       string      // the card's fill; "" for none
	band        string      // the caption strip's fill; "" for none
	radius      float64     // the card's corner radius
	imageRadius float64     // the picture's own radius, inside a mat
	shadow      float64     // the card's elevation
	inset       int         // stage edge to card
	mat         int         // card edge to picture
}

// lookSpec resolves g.Look; see "Looks" on Gallery.
func (g Gallery) lookSpec(t *core.Theme) galleryLookSpec {
	pick := func(own *core.Theme) *core.Theme {
		if g.CardTheme != nil {
			return g.CardTheme
		}
		return own
	}
	switch g.Look {
	case GalleryLookCinema:
		return galleryLookSpec{
			theme: pick(core.DarkTheme),
			stage: lightboxPanel,
			frame: lightboxPanel,
			band:  galleryCinemaBand,
		}
	case GalleryLookPolaroid:
		ct := pick(core.DefaultTheme)
		paper := orDefault(ct.Components.Card.Background, ct.Colors.Surface)
		return galleryLookSpec{
			theme:       ct,
			frame:       paper,
			band:        paper,
			radius:      2,
			imageRadius: 1,
			shadow:      3,
			inset:       ct.Spacing.MD,
			mat:         ct.Spacing.SM,
		}
	case GalleryLookMinimal:
		ct := pick(t)
		return galleryLookSpec{
			theme:  ct,
			band:   ct.Colors.Background,
			radius: ct.Components.Card.BorderRadius,
		}
	default:
		ct := pick(t)
		card := orDefault(ct.Components.Card.Background, ct.Colors.Surface)
		return galleryLookSpec{
			theme:  ct,
			frame:  card,
			band:   card,
			radius: ct.Components.Card.BorderRadius,
			shadow: ct.Components.Card.Shadow,
			inset:  ct.Spacing.SM,
		}
	}
}

// galleryMotion is GalleryMotion resolved into the three Transition props
// the builders attach: one for changing card, one for flipping, one for
// zooming. A zero duration produces a Transition that declares nothing,
// which is how "none" and a negative duration snap.
type galleryMotion struct {
	card, flip, zoom core.StyleProp
}

func (m GalleryMotion) resolve() galleryMotion {
	ms := func(v, def int) int {
		switch {
		case v == 0:
			return def
		case v < 0:
			return 0
		}
		return v
	}
	easing := m.Easing
	if easing == "" {
		easing = core.EaseInOut
	}
	card := ms(m.DurationMs, galleryDefaultCardMs)
	if m.Transition == GalleryTransitionNone {
		card = 0
	}
	flip := ms(m.FlipDurationMs, galleryDefaultFlipMs)
	if m.Flip == GalleryFlipNone {
		flip = 0
	}
	return galleryMotion{
		card: core.Transition(card, easing),
		flip: core.Transition(flip, easing),
		zoom: core.Transition(ms(m.ZoomDurationMs, galleryDefaultZoomMs), easing),
	}
}

// galleryFlippable reports whether a card has a back worth turning to.
func galleryFlippable(it GalleryItem) bool {
	return it.Back != nil || len(it.Details) > 0
}

// galleryShareText is what the default share puts on the clipboard:
// ShareText, or the Title, followed by the ShareURL.
func galleryShareText(it GalleryItem) string {
	text := it.ShareText
	if text == "" && it.ShareURL != "" {
		text = it.Title
	}
	return joinNonEmpty(" ", text, it.ShareURL)
}

// galleryItemKey is item i's identity: its Key, or its index.
func galleryItemKey(it GalleryItem, i int) string {
	if it.Key != "" {
		return it.Key
	}
	return fmt.Sprintf("#%d", i)
}

// galleryClamp keeps an index on a card: inside [0, n), or 0 with no cards.
func galleryClamp(i, n int) int {
	if n == 0 {
		return 0
	}
	return galleryClampInt(i, 0, n-1)
}

// galleryStep moves i by delta in n cards, around the ends when wrap is set
// and stopping at them otherwise.
func galleryStep(i, delta, n int, wrap bool) int {
	if n == 0 {
		return 0
	}
	j := i + delta
	if wrap {
		return ((j % n) + n) % n
	}
	return galleryClampInt(j, 0, n-1)
}

func galleryClampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// joinNonEmpty joins the non-empty parts with sep, so a missing title or
// URL leaves no dangling separator.
func joinNonEmpty(sep string, parts ...string) string {
	out := ""
	for _, p := range parts {
		if p == "" {
			continue
		}
		if out != "" {
			out += sep
		}
		out += p
	}
	return out
}
