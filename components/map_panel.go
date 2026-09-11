package components

import (
	"fmt"
	"math"

	"github.com/rohanthewiz/grmob/core"
)

// MapPanel is "where are these": a live map over a set of points, opened at a
// view that contains all of them.
//
//	components.MapPanel{
//	    Pins: []components.MapPin{
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
// A set straddling the antimeridian is measured the long way round. Tokyo and
// Honolulu are 40 degrees apart going east and are read here as 200 degrees
// apart, so the fit opens on the Atlantic with both pins at the edges. Fixing
// it is a circular mean rather than an average, which is a different function
// with a different contract for the centre — and no caller has yet had a set
// that crosses it. A caller who does should compute the region themselves and
// pass it to core.MapView, which is the arrangement this widget is a shortcut
// for.
//
// # An empty set has no answer
//
// FitRegion of nothing returns the zero Region and false. There is no sensible
// centre for no points, and the tempting answer — 0,0 — is a real place in the
// Gulf of Guinea that a map will happily draw. Same rule as
// components.StaticMap's "Zero is a place": a caller with nothing to show must
// not render the map at all.
func FitRegion(pins []MapPin) (core.Region, bool) {
	if len(pins) == 0 {
		return core.Region{}, false
	}

	minLat, maxLat := pins[0].Lat, pins[0].Lat
	minLng, maxLng := pins[0].Lng, pins[0].Lng
	for _, p := range pins[1:] {
		minLat = math.Min(minLat, p.Lat)
		maxLat = math.Max(maxLat, p.Lat)
		minLng = math.Min(minLng, p.Lng)
		maxLng = math.Max(maxLng, p.Lng)
	}
	lat := (minLat + maxLat) / 2
	lng := (minLng + maxLng) / 2

	cos := math.Cos(lat * math.Pi / 180)
	if cos < 0.01 {
		cos = 0.01
	}
	spread := math.Max(maxLat-minLat, (maxLng-minLng)*cos)
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
