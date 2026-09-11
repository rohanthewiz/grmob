package components

import (
	"fmt"
	"math"
	"net/url"
	"strconv"

	"github.com/rohanthewiz/grmob/core"
)

// StaticMap is "where is this": a map image of one point, which hands off to
// the platform's own maps app when it is tapped.
//
//	components.StaticMap{
//	    Lat: 38.7223, Lng: -9.1393,
//	    Label:  "Lisbon Baptist Church",
//	    Marker: true,
//	}
//
//	┌─────────────────────────┐
//	│                         │   a core.Image of a rendered map,
//	│           📍            │   fetched from a tile provider
//	│                         │
//	└─────────────────────────┘
//	        tap ──▶ core.OpenURL ──▶ Google Maps / Apple Maps / a browser
//
// # Why an image and a hand-off rather than a map view
//
// Because this answers the question an app actually asks. "Where is the
// church", "where is this event" wants a picture that says *there*, and then
// the user wants directions — which is a thing the platform's maps app does
// better than any embedded view could, with the user's own saved home address,
// their transport preferences and their downloaded tiles.
//
// It is also the version that exists today on all four targets with no
// renderer work at all: a core.Image the reconciler already knows how to patch
// and a core.OpenURL the three hosts already hand to the system. A live
// panning, zooming map is a node type (MapKit, osmdroid, Leaflet — three host
// implementations and a marker protocol), which is a much larger feature and
// is the thing to build when an app needs to *interact* with a map rather than
// point at one.
//
// # The image is a network fetch, and the provider is required
//
// Nothing here draws a map. The widget builds a URL and hands it to
// core.Image, so what arrives is whatever the provider serves — which makes
// the provider a decision an app has to own:
//
//	GoogleStaticMap(key)  a paid API with an availability promise, and the
//	                      only bundled provider whose host resolves.
//	a func of your own    StaticMapProvider is one function; a provider this
//	                      package has never heard of is five lines.
//	OSMStaticMap          deprecated, and dead. See that function.
//
// There is no default, and there used to be. OSMStaticMap was it, on the
// argument that a widget nobody can render without first buying something is
// a widget nobody evaluates — which was the right argument for as long as
// there was a keyless service to point at. There is not one now: every
// static-map service the OpenStreetMap wiki still lists takes a key, and the
// two keyless entries on that page are a web form and an HTML-embed
// generator, neither of which is a URL an image node can fetch.
//
// So a StaticMap with no Provider renders its frame and no image, exactly as
// GoogleStaticMap("") does, and records ConcernNoMapProvider in debug mode.
// A build that has not chosen should look unfinished and say so, rather than
// draw a map of a host it cannot reach.
//
// # Why this cannot just draw the tiles itself
//
// The obvious escape is to stop asking for a rendered image and compose one
// out of the raster tiles this repository already uses — tile.openstreetmap.org
// is keyless, alive, and is what core.MapView draws through on all three
// hosts. It is not available here. A tile grid centred on an arbitrary point
// needs its tiles placed at pixel offsets inside a clipped box, which is
// absolute positioning, and absolute positioning is the one Style.Position
// value with no Compose analog (see GrMobStyle.kt, which says so). A widget
// that laid out correctly on web and iOS and drifted on Android is worse than
// one that asks for a key.
//
// # Device pixel ratio
//
// Width and Height are *logical* pixels: they set the size of the box on the
// screen. Scale is how many device pixels the provider should put inside each
// of them. Two numbers because they answer two questions — how big is this,
// and how sharp is it — and until Scale existed one number answered both: a
// 320px request stretched across a 2x phone's 640 real pixels, which arrives
// visibly soft and could not be said otherwise from the call site.
//
//	Width: 320, Scale: 2    a 320-logical-px box holding a 640px image
//
// Nothing reads the ratio off the device, because nothing in this framework
// knows it: there is no host event that reports screen metrics, on any of the
// three renderers. A caller who has the number states it; a caller who says
// nothing is choosing 1x, which is what every caller got before the field
// existed.
//
// Whether the number can be spent is the provider's business. Google's API
// has a scale parameter that doubles the raster without touching the
// cartography; a provider with no such parameter ignores the field, which is
// why this is a StaticMapArea value rather than a multiply applied to Width
// before the provider sees it. The multiply would be wrong twice over: asking
// a map service for twice the pixels at the same zoom returns twice as much
// *map*, and asking at zoom+1 returns the same ground drawn for a deeper
// zoom, whose labels then land at half the physical size they were drawn for.
//
// # The hand-off is one URL for three platforms
//
// Each platform has a scheme of its own — `geo:` on Android, `maps://` on iOS
// — and nothing in this framework knows which platform it is running on
// (deliberately: see core.OpenURL, which promises only the part that is
// portable). So the default Handoff is an https URL that all three resolve,
// and that on both phones reaches the installed maps app rather than a
// browser tab.
//
// An app that does know its platform can say so in one line —
// `Handoff: func(lat, lng float64, label string) string { return "geo:..." }`
// — and an app that would rather not leave for Google has
// OpenStreetMapHandoff, which opens a browser everywhere.
//
// # Accessibility
//
// A map image read by a screen reader is a rectangle with nothing in it: the
// meaning is in the arrangement, which is exactly the case core.RoleImg
// exists for (see its doc, and components.Compass, which made the same
// argument about a compass rose). So the widget announces once and hides the
// image inside it.
//
// When it is tappable the role is core.RoleLink rather than RoleButton, and
// the distinction is the one core.RoleLink's doc draws: a button does
// something here and a link goes somewhere else. Tapping this leaves the app
// entirely, which is as far as "somewhere else" goes, and a reader deciding
// whether to follow it deserves to know that before they do.
//
// # Zero is a place
//
// Lat 0, Lng 0 is the Gulf of Guinea, and this widget draws it. There is no
// "unset" coordinate to detect — a float64 pair has no third state — so a
// caller whose location has not loaded yet must not render the widget at all,
// exactly as they would not render an EmptyState's action with no handler.
// components.Skeleton is the placeholder for that gap.
type StaticMap struct {
	// Lat and Lng are the point to show, in degrees.
	//
	// Latitude is clamped to the Web Mercator limit (±85.0511°) rather than to
	// ±90: every tile provider here projects with Mercator, where the poles
	// are at infinity, and a request past the limit comes back as an error
	// image rather than as a view of the Arctic. Longitude wraps instead of
	// clamping, because longitude is a circle — 190° is 170°W, and clamping it
	// to 180 would move the point rather than name it.
	Lat, Lng float64

	// Zoom is the slippy-tile zoom level both providers share: 0 is the whole
	// world, 19 is a building. 0 means DefaultMapZoom (street level), which is
	// the scale "where is this" is asking at.
	//
	// Zero cannot mean "the whole world" here, and that is the one place this
	// widget spends a legitimate value on a default. A world map centred on a
	// church is a picture of the Atlantic; a caller who genuinely wants zoom 0
	// has a provider of their own to ask.
	Zoom int

	// Width and Height are the image's size in px. 0 means
	// DefaultMapWidth/DefaultMapHeight, a card-width landscape panel.
	//
	// Both are clamped to MaxMapDimension, which is the smaller of what the
	// two bundled providers will serve. A request past it is refused by the
	// server, and a refusal arrives as a broken image with no error anywhere —
	// so the clamp is what turns "nothing rendered and nobody knows why" into
	// "a slightly smaller map than you asked for".
	Width, Height int

	// Scale is the device pixel ratio to ask the provider for: 1, 2 or 3.
	// 0 means DefaultMapScale, which is 1x.
	//
	// It changes the image, never the box: a Scale of 2 leaves the widget
	// exactly Width by Height logical pixels on screen and asks for four
	// times as many actual pixels to fill them. Clamped to MaxMapScale and
	// then spent, or ignored, by the provider — see "Device pixel ratio" on
	// the type for both halves of that.
	Scale int

	// Marker draws a pin at the point. Off by default: a map whose centre *is*
	// the subject does not always need one, and a single pin in the middle of
	// a small image can hide the very corner a reader is looking at.
	Marker bool

	// Label is what the place is called — "Lisbon Baptist Church", "The
	// rehearsal hall". It is the spoken name of the image, and it is handed to
	// the Handoff func, which may or may not have anywhere to put it: neither
	// built-in hand-off does (see GoogleMapsHandoff), and a platform-specific
	// one a caller writes can.
	//
	// Empty falls back to the coordinates, which is honest and reads badly —
	// "Map at 38.7223, -9.1393". Worth setting for that reason alone.
	Label string

	// Provider builds the image URL. Required: nil renders no image at all
	// and reports ConcernNoMapProvider in debug mode. See "The image is a
	// network fetch, and the provider is required" on the type for why this
	// stopped having a default.
	Provider StaticMapProvider

	// Handoff builds the URL a tap opens. nil means GoogleMapsHandoff.
	//
	// Setting it to a func that returns "" is how a caller says *no hand-off*:
	// core.OpenURL drops an empty URL, and this widget reads the same answer
	// one step earlier and renders a picture with no link role and no
	// callback, rather than a control that does nothing when tapped.
	Handoff MapHandoff

	// OnTap replaces the hand-off entirely — a caller who wants to push their
	// own map screen, open a sheet of directions, or log the tap first.
	//
	// It takes precedence over Handoff, and it does not compose with it: a
	// widget that both called back and opened an external app would leave the
	// caller no way to express either one alone. An OnTap that wants the
	// hand-off too can ask for it: core.OpenURL(components.GoogleMapsHandoff(
	// lat, lng, label)).
	OnTap func()

	// Style is applied last, to the outer box, so a caller can give the map a
	// margin, a radius or a border without reaching inside it.
	Style []core.StyleProp

	// AccessibilityLabel overrides the spoken name. AccessibilityHint
	// overrides the "opens in your maps app" description a tappable map
	// carries; it is ignored when there is nothing to tap.
	AccessibilityLabel string
	AccessibilityHint  string
}

// The defaults and the one clamp, named so a caller can reason about them
// without reading the render function.
const (
	// DefaultMapZoom is street level: a block or two across the image, which
	// is the scale at which "where is this" is legible. 15 on the slippy
	// scale, the same number both providers mean by it.
	//
	// One level deeper than core.MapView's default, and the gap is the
	// difference between the two widgets: a static map answers "where is this
	// one place", a live map is usually showing a set, and one level out is
	// about four times the area.
	DefaultMapZoom = 15

	// DefaultMapWidth and DefaultMapHeight are a 16:9 panel the width of a
	// phone card. Landscape because a map of one point has nothing to say
	// vertically that it does not say horizontally, and a tall map wastes the
	// screen a caption wants.
	DefaultMapWidth  = 320
	DefaultMapHeight = 180

	// MaxMapDimension is the smaller of the two bundled providers' limits.
	// Google's free static API refuses above 640 per side; the OSM service's
	// ceiling is the same order. See StaticMap.Width for why this clamps
	// rather than passing the request through to be refused.
	MaxMapDimension = 640

	// DefaultMapScale is one image pixel per logical pixel, which is what a
	// caller who says nothing about the device gets — and what every caller
	// got before Scale existed. See "Device pixel ratio" on StaticMap for why
	// it is not read off the screen.
	DefaultMapScale = 1

	// MaxMapScale is 3, the deepest ratio shipping phones use. It clamps the
	// *request*, and is not a promise about the answer: Google's API tops out
	// at MaxGoogleMapScale and a provider with no scale of its own serves 1x
	// whatever it is handed.
	MaxMapScale = 3

	// MaxGoogleMapScale is what the Maps Static API accepts — 1 or 2, and
	// nothing else. A 3x device therefore gets the 2x image, which is the
	// sharpest thing that API will serve and still four times the pixels of
	// the 1x it used to get.
	//
	// Google applies its size limit *before* scaling, which is why Width and
	// Height are not also divided down to make room: a scale=2 request for a
	// 640px box is accepted and served as 1280px.
	MaxGoogleMapScale = 2

	// MercatorLatLimit is where Web Mercator stops. Every tile provider here
	// projects with it, so this is the edge of the addressable world rather
	// than a choice — see StaticMap.Lat.
	MercatorLatLimit = 85.0511
)

// StaticMapArea is the view a provider is asked to render: the point, the
// scale, the pixel size, and whether to pin it.
//
// A struct rather than five arguments because a provider is a function a caller
// writes, and a signature that grows is a signature that breaks every one of
// them. A field added here is a field an existing provider ignores.
//
// The values arrive already defaulted and already clamped — a provider never
// sees a zero Zoom, a 4000px Width or a latitude off the end of Mercator — so
// every provider is spared the same four lines and none of them can disagree
// about what a zero means.
type StaticMapArea struct {
	Lat, Lng      float64
	Zoom          int
	Width, Height int

	// Scale is the device pixel ratio asked for, already defaulted and
	// clamped. Width and Height stay logical: a provider that honours Scale
	// returns Width*Scale actual pixels for the same map, and a provider that
	// cannot honour it returns Width by Height and is not wrong to. See
	// "Device pixel ratio" on StaticMap.
	Scale int

	Marker bool
}

// ConcernNoMapProvider: a StaticMap rendered with no Provider, which draws an
// empty frame. It is a development-time finding rather than a panic because
// the failure is survivable — a screen missing its map is still a screen — and
// because the fix is configuration, which is exactly the class of mistake that
// is invisible in a running app and obvious in a concern list.
const ConcernNoMapProvider = "no-map-provider"

// StaticMapProvider turns a view into an image URL. See StaticMap.Provider.
type StaticMapProvider func(StaticMapArea) string

// noStaticMapProvider is what an unconfigured widget renders through: a
// provider that declines. Named rather than written inline at the call site so
// that "no provider" and "a provider with no key" travel the same path to the
// same empty src, instead of being two shapes of nothing that could drift.
func noStaticMapProvider(StaticMapArea) string { return "" }

// MapHandoff turns a point and its name into a URL for core.OpenURL. See
// StaticMap.Handoff.
type MapHandoff func(lat, lng float64, label string) string

// OSMStaticMap renders through staticmap.openstreetmap.de, the OpenStreetMap
// community's static-image service.
//
// Deprecated: that service no longer exists. The OpenStreetMap wiki's
// StaticMapLite page says "This service has been discontinued", and the host
// stopped resolving — NXDOMAIN, checked 2026-09-11. Every URL this function
// builds is now a fetch that fails and a frame that stays empty.
//
// It is kept rather than deleted for two reasons. A caller that names it still
// compiles, which is the difference between a deprecation and a breakage; and
// a provider that returns a URL to a dead host is a far easier thing to
// diagnose than a symbol that has gone away, because the URL is printable and
// the failure is one nslookup from being understood.
//
// It was this package's default, on the strength of being keyless. See "The
// image is a network fetch, and the provider is required" on StaticMap for
// what replaced it, which is nothing, and why.
//
// Scale is ignored. The service had no scale parameter while it was up.
//
// The marker style name ("ol-marker") is the service's own vocabulary rather
// than anything this package defines, which is the general shape of a provider
// function — it translates a StaticMapArea into one service's dialect and
// nothing more.
func OSMStaticMap(a StaticMapArea) string {
	q := url.Values{}
	q.Set("center", coordPair(a.Lat, a.Lng))
	q.Set("zoom", strconv.Itoa(a.Zoom))
	q.Set("size", fmt.Sprintf("%dx%d", a.Width, a.Height))
	q.Set("maptype", "mapnik")
	if a.Marker {
		q.Set("markers", coordPair(a.Lat, a.Lng)+",ol-marker")
	}
	return "https://staticmap.openstreetmap.de/staticmap.php?" + q.Encode()
}

// GoogleStaticMap renders through the Google Static Maps API with the given
// key, which the caller has obtained and is billed for.
//
// A constructor rather than a bare provider because the key is the caller's:
// it is configuration, it differs per build, and a provider that read it out
// of a package variable would be a second place for a deployment to be wrong.
//
// An empty key yields a provider that returns "", which renders the widget as
// a box with no image in it rather than as a map of Google's "this request is
// not authorized" error tile. A misconfigured build should look unfinished,
// not broken.
func GoogleStaticMap(key string) StaticMapProvider {
	return func(a StaticMapArea) string {
		if key == "" {
			return ""
		}
		q := url.Values{}
		q.Set("center", coordPair(a.Lat, a.Lng))
		q.Set("zoom", strconv.Itoa(a.Zoom))
		q.Set("size", fmt.Sprintf("%dx%d", a.Width, a.Height))
		// size stays logical and scale multiplies it, which is the API's own
		// arrangement — see MaxGoogleMapScale. Written only when it is asked
		// for, because scale=1 is the documented default and a parameter that
		// says the default says nothing except in a diff.
		if a.Scale > 1 {
			q.Set("scale", strconv.Itoa(MaxGoogleMapScale))
		}
		if a.Marker {
			q.Set("markers", "color:red|"+coordPair(a.Lat, a.Lng))
		}
		q.Set("key", key)
		return "https://maps.googleapis.com/maps/api/staticmap?" + q.Encode()
	}
}

// GoogleMapsHandoff is the default tap target: Google's documented
// cross-platform maps URL.
//
// It is the one URL all three platforms resolve, and on both phones it reaches
// the installed maps app rather than a browser tab — which is the whole point
// of handing off at all.
//
// The label is deliberately not in it, and the reason is worth stating because
// the URL has a slot that looks like it wants one. `query` is a *search*: given
// a name it runs a geocode, which for "St Mary's" lands on whichever St Mary's
// the search liked, possibly in another country. Given a coordinate pair it
// lands on the coordinate. The app knows exactly where this place is, so it
// says so, and the pin is unnamed rather than wrong.
//
// A caller who wants the name *and* the point has to pick a platform to say it
// to — `geo:lat,lng?q=lat,lng(Label)` on Android,
// `https://maps.apple.com/?ll=lat,lng&q=Label` on iOS — which is what the
// label parameter on this signature is for.
func GoogleMapsHandoff(lat, lng float64, label string) string {
	q := url.Values{}
	q.Set("api", "1")
	q.Set("query", coordPair(lat, lng))
	return "https://www.google.com/maps/search/?" + q.Encode()
}

// OpenStreetMapHandoff opens the point on openstreetmap.org, for an app that
// would rather not send its users to Google.
//
// It opens a browser on every platform, including the two with a maps app
// installed — OSM has no app with a URL scheme to claim the link. That is the
// trade, and it is the reason this is not the default: directions are what a
// person taps a map for, and a browser is a worse place to get them.
func OpenStreetMapHandoff(lat, lng float64, label string) string {
	q := url.Values{}
	q.Set("mlat", coord(lat))
	q.Set("mlon", coord(lng))
	// The fragment is what sets the view; the query is what drops the marker.
	// Both are needed: openstreetmap.org centres on the fragment and shows a
	// pin for the query, and a URL with only one of them either lands on the
	// right place with no pin or pins a place it is not looking at.
	return fmt.Sprintf("https://www.openstreetmap.org/?%s#map=%d/%s/%s",
		q.Encode(), DefaultMapZoom, coord(lat), coord(lng))
}

// coord formats one degree value in the shortest form that round-trips, the
// same 'g'/-1 rule htmlout's formatNumber follows: 38.7223 stays "38.7223" and
// a whole degree stays "38" rather than becoming "38.000000".
//
// Shortest-round-trip rather than a fixed six places because these strings go
// into URLs that end up in bug reports, and a coordinate a person can read
// against the one in their source is worth more than aligned columns. Nothing
// is lost: a float64's shortest form is exact.
func coord(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}

// coordPair is the "lat,lng" both providers and both hand-offs spell the same
// way. One function because four call sites writing it themselves is four
// chances to swap the order, which is a bug that renders a perfectly good map
// of the wrong hemisphere.
func coordPair(lat, lng float64) string {
	return coord(lat) + "," + coord(lng)
}

// Area resolves the caller's fields into the view a provider is handed: the
// defaults applied, the size clamped, the latitude clamped and the longitude
// wrapped. Everything downstream reads this rather than the struct, so there
// is one statement of what a zero means.
//
// Exported because the answer is worth asking for from outside. A caller who
// wants to know what URL this widget will request — to log it, to pre-warm a
// cache, to show it in a tutorial — can ask the provider about this rather
// than re-deriving the defaults, which is the one way to get a second answer
// that disagrees.
func (m StaticMap) Area() StaticMapArea {
	zoom := m.Zoom
	if zoom <= 0 {
		zoom = DefaultMapZoom
	}
	// 19 is as deep as the standard tile pyramid goes. A request past it is
	// served as a blank or an error tile by both providers, so the clamp turns
	// a typo into a very close map instead of an empty box.
	if zoom > 19 {
		zoom = 19
	}

	w, h := m.Width, m.Height
	if w <= 0 {
		w = DefaultMapWidth
	}
	if h <= 0 {
		h = DefaultMapHeight
	}
	if w > MaxMapDimension {
		w = MaxMapDimension
	}
	if h > MaxMapDimension {
		h = MaxMapDimension
	}

	// The size clamp above is on the LOGICAL size, and the scale is resolved
	// after it rather than folded into it. Google applies its own 640 limit
	// before scaling, so a 640px box at 2x is a legal request for a 1280px
	// image — folding the two together here would refuse that request on
	// behalf of a service that accepts it. See MaxGoogleMapScale.
	scale := m.Scale
	if scale <= 0 {
		scale = DefaultMapScale
	}
	if scale > MaxMapScale {
		scale = MaxMapScale
	}

	return StaticMapArea{
		Lat:    clampLat(m.Lat),
		Lng:    wrapLng(m.Lng),
		Zoom:   zoom,
		Width:  w,
		Height: h,
		Scale:  scale,
		Marker: m.Marker,
	}
}

// clampLat holds a latitude inside Web Mercator's addressable range. See
// StaticMap.Lat.
func clampLat(lat float64) float64 {
	if lat > MercatorLatLimit {
		return MercatorLatLimit
	}
	if lat < -MercatorLatLimit {
		return -MercatorLatLimit
	}
	return lat
}

// wrapLng brings a longitude into (-180, 180] by going round rather than by
// stopping at the edge, because longitude is a circle: 190° east and 170° west
// are the same meridian, and a clamp would quietly move the point to the
// antimeridian instead.
//
// math.Mod keeps the sign of its first argument, so a negative input stays
// west, and the two adjustments below are what carry a value past ±180 round
// to the other side.
func wrapLng(lng float64) float64 {
	lng = math.Mod(lng, 360)
	if lng > 180 {
		lng -= 360
	}
	if lng <= -180 {
		lng += 360
	}
	return lng
}

// handoffURL is the URL a tap opens, or "" when there is none. See
// StaticMap.Handoff on why an empty answer is a supported one.
func (m StaticMap) handoffURL(a StaticMapArea) string {
	handoff := m.Handoff
	if handoff == nil {
		handoff = GoogleMapsHandoff
	}
	return handoff(a.Lat, a.Lng, m.Label)
}

// label is what the widget announces. The place's name when it has one,
// because that is what a reader wants to hear; the coordinates when it does
// not, because a map with no name at all is still a map *of* somewhere and
// "Map" alone says less than the numbers do.
func (m StaticMap) label(a StaticMapArea) string {
	if m.AccessibilityLabel != "" {
		return m.AccessibilityLabel
	}
	if m.Label != "" {
		return "Map of " + m.Label
	}
	return fmt.Sprintf("Map at %s, %s", coord(a.Lat), coord(a.Lng))
}

func (m StaticMap) Render(ctx *core.Context) *core.Node {
	t := ctx.Theme()
	a := m.Area()

	provider := m.Provider
	if provider == nil {
		// No default any more — see "The image is a network fetch, and the
		// provider is required" on the type. Reported rather than substituted,
		// so a build that has not chosen one gets an empty frame in production
		// and a named concern in development: the same two outcomes
		// GoogleStaticMap("") already produces, reached one step earlier.
		//
		// Gated on IsDebugMode per ReportConcern's contract: the detail costs
		// a Sprintf, and a release build should not pay for one on every
		// frame this widget renders.
		if core.IsDebugMode() {
			core.ReportConcern(ConcernNoMapProvider, fmt.Sprintf(
				"a StaticMap at %s has no Provider, so it renders an empty "+
					"%dx%d frame. Set Provider to GoogleStaticMap(key) or a "+
					"func of your own; OSMStaticMap is deprecated and its host "+
					"no longer resolves.",
				coordPair(a.Lat, a.Lng), a.Width, a.Height))
		}
		provider = noStaticMapProvider
	}

	// Whether this map is a control at all, decided once. OnTap wins outright
	// (see the field), and a Handoff that returns "" is a caller declining the
	// link rather than a bug to route around.
	tap := m.OnTap
	if tap == nil {
		if u := m.handoffURL(a); u != "" {
			tap = func() { core.OpenURL(u) }
		}
	}

	size := []core.StyleProp{
		core.Width(fmt.Sprintf("%dpx", a.Width)),
		core.Height(fmt.Sprintf("%dpx", a.Height)),
	}

	// The image. ContentModeFill rather than Fit: the URL already asks for
	// exactly these pixels, so the two agree in the ordinary case — and when
	// they do not (a provider that rounds, a device that rasterises at a
	// different ratio) a map that fills its frame and loses a sliver of edge
	// reads as a map, where one letterboxed inside a coloured box reads as a
	// broken image.
	//
	// Hidden from the accessibility tree: the container speaks for it. An
	// unhidden <img> with no alt inside a role="link" is announced by its URL
	// on the web, which for a provider URL is a minute of read-aloud query
	// string.
	//
	// The fill is stated because core.Image's theme base is Components.Camera,
	// whose background is black in every bundled theme — right for a
	// viewfinder showing nothing yet, wrong for a map, where it is a black
	// rectangle for the length of a network fetch. Surface is what the frame
	// behind it is, so the widget is one colour until the tiles arrive.
	img := append([]core.StyleProp{
		core.AccessibilityHidden(),
		core.BackgroundColor(t.Colors.Surface),
	}, size...)

	items := make([]core.PropsAndChildren, 0, 10+len(m.Style))
	items = append(items,
		core.Padding(0),
		// A backdrop behind the image, so the frame is the right shape and
		// colour while the fetch is in flight and stays legible if it fails.
		// Surface rather than a literal: a hole in a dark theme should be
		// dark.
		core.BackgroundColor(t.Colors.Surface),
	)
	for _, sp := range size {
		items = append(items, sp)
	}

	// What this is, then what it is called, then what tapping it does — the
	// order a reader meets them in. See ListRow's render for the same
	// arrangement and why the order is for this function's readers rather than
	// for the result.
	if tap != nil {
		// A link, not a button: tapping leaves the app. See "Accessibility".
		items = append(items, core.AccessibilityRole(core.RoleLink))
	} else {
		items = append(items, core.AccessibilityRole(core.RoleImg))
	}
	items = append(items, core.AccessibilityLabel(m.label(a)))
	if tap != nil {
		hint := m.AccessibilityHint
		if hint == "" {
			hint = "Opens this place in your maps app"
		}
		items = append(items, core.AccessibilityHint(hint))
	}

	for _, sp := range m.Style {
		items = append(items, sp)
	}

	if tap != nil {
		items = append(items, core.OnClick(tap))
	}

	// The image goes in last, after the style props, for the reason every
	// widget here orders them that way: a caller's Style must be able to
	// override the widget's own, and a child in the middle of the list would
	// put the override on the far side of it.
	items = append(items, core.ImageWithMode(provider(a), core.ContentModeFill, img...))

	return core.Box(items...).Render(ctx)
}
