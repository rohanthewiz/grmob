package comps

import (
	"fmt"
	"math"
	"sort"

	"github.com/rohanthewiz/grmob/core"
)

// MapPanel is "where are these": a live map over a set of points, opened at a
// view that contains all of them.
//
//	comps.MapPanel{
//	    Pins: []comps.MapPin{
//	        {ID: "hall", Lat: 38.7223, Lng: -9.1393, Title: "The hall"},
//	        {ID: "annex", Lat: 38.7251, Lng: -9.1402, Title: "The annex"},
//	    },
//	    OnPinTap: func(id string) { open(id) },
//	    Height:   "320px",
//	}
//
// # What this adds to core.MapView, which is the whole of its case
//
// One thing, and it is the thing every consumer of a set of points needs and
// would otherwise write itself: the *opening region*. core.MapView takes a
// Region and applies it only when it changes, which is correct and leaves the
// caller holding a question — what region shows all of my points? — whose
// answer is a bounding box, a projection correction and a logarithm. See
// FitRegion, which is that arithmetic and is exported for a caller who wants
// it without this widget.
//
// Everything else here is arrangement a caller could write in ten lines and
// which is worth having in one place anyway: the pins as keyed children, the
// empty state for a set with nothing in it, and an optional caption.
//
// # What it deliberately does not add
//
// No region state, no recentre control, no echo of OnRegionChange. The region
// is computed once from the pins and handed over, and where the reader takes
// the map from there is the reader's. An app that wants to follow the map
// reaches for core.MapView directly and holds the Region itself — which is the
// arrangement core.MapView's "The map is controlled, with the echo guard a
// drag needs" describes, and it is a different widget's job.
//
// The pins changing DOES move the map: a new set is a new fitted region, which
// core.MapView applies because it changed. That is the right behaviour for the
// case this widget is for — the set is the subject — and it is the reason a
// caller whose pins update every few seconds wants core.MapView instead.
//
// # Tiles are somebody else's bandwidth
//
// Inherited from core.MapView, and unchanged by this widget: two of the three
// hosts draw OpenStreetMap tiles from the project's own servers, under a usage
// policy. See that node's doc. Unlike StaticMap this needs no key from anyone,
// because each host's own engine draws the map — which is the odd asymmetry
// that the richer widget is the free one.
type MapPanel struct {
	// Pins are the points to show. An empty set renders Empty rather than a
	// map, because the alternative is a map of 0,0 — see FitRegion, which
	// refuses to answer for a set with nothing in it.
	Pins []MapPin

	// OnPinTap is called with the id of the pin the user tapped. Pins with an
	// empty id are drawn and never reported, which is core.Marker's own rule.
	OnPinTap func(id string)

	// OnMapTap is called with the coordinates of a tap that did not hit a pin.
	// The suppression is each host's; see core.OnMapTap.
	OnMapTap func(lat, lng float64)

	// Height sizes the map. "" means DefaultMapPanelHeight. Width is always
	// 100%: a map narrower than its column is a map with a gutter beside it,
	// and a caller who wants one wraps this in a box.
	Height string

	// Caption is a line under the map — a count, a legend, a note about what
	// is not on it. Empty draws nothing, including no padding.
	Caption string

	// Empty is what to render for a set with no pins in it. The zero value is
	// a generic EmptyState; a caller who knows what the reader was looking for
	// should say so, because "nothing to show" is the least useful sentence a
	// screen can end on.
	Empty EmptyState

	// Zoom overrides the fitted zoom level. 0 means "fit the pins", which is
	// what this widget is for; a value is for a caller who wants a fixed scale
	// and only the centring.
	Zoom float64

	// Style is applied last, to the outer column, so a caller can give the
	// panel a margin or a border without reaching inside it.
	Style []core.StyleProp
}

// MapPin is one point in a MapPanel: the data a core.Marker needs, as a value
// a caller can hold in a slice and sort.
//
// A struct rather than four arguments for the reason StaticMapArea is one: a
// caller builds these in a loop from their own data, and a field added here is
// a field existing code ignores.
type MapPin struct {
	// ID is what OnPinTap reports, and what the reconciler keys the marker by.
	// Empty is allowed — the single unnamed "you are here" pin — and is never
	// reported. See core.Marker.
	ID string
	// Lat and Lng are the point, in degrees.
	Lat, Lng float64
	// Title is the callout the platform shows when the pin is tapped. It is
	// not an accessibility label; a map's annotations are announced by each
	// platform's own map accessibility.
	Title string
}

const (
	// DefaultMapPanelHeight is a map tall enough to have a shape and short
	// enough to leave a caption and a row or two of context on a phone screen.
	DefaultMapPanelHeight = "280px"

	// MinFitSpread is the tightest view FitRegion will open at, in degrees of
	// latitude — about 550 metres, which is a few blocks.
	//
	// It exists because a single pin has a spread of zero and the logarithm of
	// a division by zero is not a zoom level. A floor rather than a branch:
	// one rule, no arm that runs only for one point, and the floor is the view
	// a single pin wants anyway.
	MinFitSpread = 0.005

	// FitPadding widens the fitted span so the outermost pins are not against
	// the edge of the frame. A fifth, which is about one pin's height at the
	// sizes a panel is drawn at.
	FitPadding = 1.2

	// MinFitZoom and MaxFitZoom bound the answer. 19 is as deep as the
	// standard tile pyramid goes; 1 is a view half the globe wide, which is
	// the widest this returns.
	//
	// Not 0, even though 0 is the world and would fit anything: core.Region
	// reads a zero Zoom as "unstated" and substitutes core.DefaultMapZoom, so
	// returning 0 here would hand back a neighbourhood view of a set spanning
	// continents — the exact opposite of what was asked for. See
	// core.Region.Zoom, which explains why zero cannot mean the world.
	MinFitZoom = 1.0
	MaxFitZoom = 19.0
)

// FitRegion is the view that contains every pin: a centre and the zoom level
// at which the whole set is on screen.
//
// Exported separately from the widget because it is the reusable half. A
// caller driving core.MapView directly — holding the region in state, echoing
// OnRegionChange — still needs this answer for its opening view, and the
// alternative is every such caller deriving the same logarithm.
//
//	zoom ≈ log2(360 / spread in degrees)
//
// # The projection correction
//
// The longitude spread is scaled by the cosine of the centre latitude before
// the two spans are compared. A degree of longitude narrows towards the poles
// and a degree of latitude does not, so a fit computed from raw degrees is too
// tight in Reykjavík and about right in Quito. Whichever span needs the wider
// view decides.
//
// Very near a pole the cosine approaches zero and the correction stops being
// meaningful, so it is floored — a map that far north is showing one place
// anyway.
//
// # Two sets it does not fit, and says so here rather than pretending
//
// A set spanning more than half the globe is clamped to MinFitZoom rather than
// contained: that is a view 180 degrees wide, and the alternative would be a
// zoom core.Region cannot express (see MinFitZoom). Pins outside it are off
// screen, and a map of the whole planet would not have shown them usefully
// anyway.
//
// A set straddling the antimeridian used to be measured the long way round:
// Tokyo and Honolulu are 62 degrees apart going east and were read as 298
// going the other way, so the fit opened on the Atlantic with both pins off
// the edges. That is fixed, and the fix is longitudeSpan — the smallest arc of
// longitude containing every point, found as the complement of the widest gap
// between adjacent points.
//
// The thing that makes it a fix rather than a trade is that the answer does
// not change for any set that does not straddle. For those, the widest gap IS
// the one that wraps from the easternmost point back round to the westernmost,
// so its complement runs from min to max and the centre is the same place the
// average used to give, arrived at by the general rule.
//
// The same *place*, not the same float. The old path added two longitudes and
// halved; this one adds half a width to an endpoint, which is one subtraction
// fewer and cancels less. A real set of pins in Austin moved from
// -97.75800000000001 to -97.758 — fourteen significant figures of agreement,
// a few nanometres of ground, and a re-recorded snapshot. Worth saying plainly
// because "nothing moves" is what this paragraph wanted to claim and is not
// quite what is true.
//
// The centre it returns is the middle of that arc, which is the contract a
// *fit* wants. A circular mean — the direction of the summed unit vectors —
// is the other candidate and is the wrong one here: it is pulled by clusters,
// so nineteen pins in Tokyo and one in Honolulu would centre on Tokyo and
// leave the twentieth off screen, which is precisely what a fit must not do.
//
// # An empty set has no answer
//
// FitRegion of nothing returns the zero Region and false. There is no sensible
// centre for no points, and the tempting answer — 0,0 — is a real place in the
// Gulf of Guinea that a map will happily draw. Same rule as
// comps.StaticMap's "Zero is a place": a caller with nothing to show must
// not render the map at all.
func FitRegion(pins []MapPin) (core.Region, bool) {
	if len(pins) == 0 {
		return core.Region{}, false
	}

	minLat, maxLat := pins[0].Lat, pins[0].Lat
	lngs := make([]float64, 0, len(pins))
	for _, p := range pins {
		minLat = math.Min(minLat, p.Lat)
		maxLat = math.Max(maxLat, p.Lat)
		lngs = append(lngs, p.Lng)
	}
	lat := (minLat + maxLat) / 2
	// Latitude is a plain interval and longitude is not: it lives on a circle,
	// where min and max are not the ends of anything. See longitudeSpan.
	lng, lngSpread := longitudeSpan(lngs)

	cos := math.Cos(lat * math.Pi / 180)
	if cos < 0.01 {
		cos = 0.01
	}
	spread := math.Max(maxLat-minLat, lngSpread*cos)
	if spread < MinFitSpread {
		spread = MinFitSpread
	}
	spread *= FitPadding

	zoom := math.Log2(360 / spread)
	if zoom > MaxFitZoom {
		zoom = MaxFitZoom
	}
	if zoom < MinFitZoom {
		zoom = MinFitZoom
	}
	return core.Region{Lat: lat, Lng: lng, Zoom: zoom}, true
}

// longitudeSpan returns the centre and the width, in degrees, of the smallest
// arc of longitude that contains every value in lngs.
//
// # Why longitude cannot be an interval
//
// Latitude has ends: -90 and +90 are places you can stand and there is nothing
// past them, so min and max are the extremes of a set and (min+max)/2 is its
// middle. Longitude has no ends. It is a circle, 180 and -180 are the same
// meridian, and a set has no single "widest pair" — Tokyo and Honolulu are 62
// degrees apart one way and 298 the other, and taking the max minus the min
// silently picks the second.
//
// # The algorithm, which needs no special case
//
// Sort the values and walk round the circle measuring the gap to the next one,
// counting the wrap from the last back to the first. The WIDEST of those gaps
// is the part of the circle with nothing in it, so its complement is the
// smallest arc that holds everything:
//
//	0°        90°       180°/-180°   -90°        0°
//	|    T    |         |        H   |          |
//	          └── widest gap: 298° ──┘
//	arc = 360 - 298 = 62°, running H..T through 180°
//
// A set that does not straddle falls out of the same rule rather than being
// detected: its widest gap is the wrap, so the arc runs from the westernmost
// point to the easternmost and the centre is the ordinary average. That is why
// this could replace the average outright instead of being a branch beside it.
//
// # Two details a reader should not have to rediscover
//
// The slice is sorted in place, so callers pass one they own — FitRegion
// builds a fresh one for exactly this reason.
//
// Ties go to the first gap found, which matters only for a set whose points
// divide the circle into equal empty halves (0° and 180°, say). Both arcs are
// then 180 degrees wide and equally correct; there is no third answer to
// prefer, and a map that wide is showing a hemisphere either way.
func longitudeSpan(lngs []float64) (centre, width float64) {
	sort.Float64s(lngs)
	n := len(lngs)

	widest, after := 0.0, 0
	for i := 0; i < n; i++ {
		gap := lngs[(i+1)%n] - lngs[i]
		if i == n-1 {
			// The wrap: from the easternmost value round through the
			// antimeridian back to the westernmost. For n == 1 this is the
			// whole circle, 360, which correctly leaves a zero-width arc at
			// the single point.
			gap += 360
		}
		if gap > widest {
			widest, after = gap, i
		}
	}

	width = 360 - widest
	// The arc begins at the value on the far side of the empty stretch and
	// runs `width` degrees east. Wrapped because that run can cross the
	// antimeridian, which is the whole point of the exercise.
	return core.WrapLongitude(lngs[(after+1)%n] + width/2), width
}

func (m MapPanel) Render(ctx *core.Context) *core.Node {
	t := ctx.Theme()

	region, ok := FitRegion(m.Pins)
	if !ok {
		empty := m.Empty
		if empty.Title == "" {
			empty.Title = "Nothing to show on a map"
		}
		return core.Column(append(styleItems(m.Style), empty)...).Render(ctx)
	}
	if m.Zoom > 0 {
		region.Zoom = m.Zoom
	}

	height := m.Height
	if height == "" {
		height = DefaultMapPanelHeight
	}

	mapItems := []core.PropsAndChildren{
		core.Width("100%"),
		core.Height(height),
	}
	if m.OnPinTap != nil {
		mapItems = append(mapItems, core.OnMarkerTap(m.OnPinTap))
	}
	if m.OnMapTap != nil {
		mapItems = append(mapItems, core.OnMapTap(m.OnMapTap))
	}
	for _, p := range m.Pins {
		mapItems = append(mapItems, core.Marker(p.ID, p.Lat, p.Lng, p.Title))
	}

	items := append(styleItems(m.Style), core.MapView(region, mapItems...))
	if m.Caption != "" {
		items = append(items, core.Box(
			core.PaddingHorizontal(16),
			core.PaddingVertical(10),
			core.BackgroundColor(t.Colors.Surface),
			core.Text(m.Caption, core.TextColor(t.Colors.TextSecondary)),
		))
	}
	return core.Column(items...).Render(ctx)
}

// styleItems opens an item list with a caller's style props, which is the
// order every widget here uses: the widget's own children follow, and a
// caller's Style lands before them so it is an override rather than a
// suggestion.
func styleItems(style []core.StyleProp) []core.PropsAndChildren {
	items := make([]core.PropsAndChildren, 0, len(style)+4)
	// No gap and no padding of its own. core.Column takes both from the theme,
	// which is right for a column of text and wrong for a frame around a map:
	// a panel that inset itself would put a gutter round every map its caller
	// placed edge to edge, and the caller cannot subtract one.
	items = append(items, core.Gap(0), core.Padding(0))
	for _, sp := range style {
		items = append(items, sp)
	}
	return items
}

// PlaceCount is the caption a MapPanel usually wants: how many places are on
// the map, which is not the same number as how many things the caller has.
//
// A recurring event is many entries and one pin; an item with no coordinates
// is not on the map at all. Saying "3 places" rather than "3 events" is the
// difference between a caption and a wrong count, and it is a mistake worth
// one exported function to not make twice.
func PlaceCount(n int) string {
	if n == 1 {
		return "1 place"
	}
	return fmt.Sprintf("%d places", n)
}
