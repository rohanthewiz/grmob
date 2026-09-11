package components

import (
	"math"
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

func renderMapPanel(t *testing.T, p MapPanel) *core.Node {
	t.Helper()
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	return p.Render(ctx)
}

func mapNode(t *testing.T, n *core.Node) *core.Node {
	t.Helper()
	m := findFirst(n, func(n *core.Node) bool { return n.Type == "MapView" })
	if m == nil {
		t.Fatal("no MapView in the rendered panel")
	}
	return m
}

var twoPins = []MapPin{
	{ID: "hall", Lat: 38.7223, Lng: -9.1393, Title: "The hall"},
	{ID: "annex", Lat: 38.7251, Lng: -9.1402, Title: "The annex"},
}

// Every pin is a keyed child, and the map opens at a region fitted to them
// rather than at a default somebody has to supply.
func TestMapPanelDrawsEveryPinAndFitsThem(t *testing.T) {
	n := renderMapPanel(t, MapPanel{Pins: twoPins})
	m := mapNode(t, n)

	markers := 0
	walk(m, func(n *core.Node) {
		if n.Type == "Marker" {
			markers++
		}
	})
	if markers != len(twoPins) {
		t.Errorf("%d markers for %d pins", markers, len(twoPins))
	}

	want, _ := FitRegion(twoPins)
	if lat, _ := m.Props["lat"].(float64); lat != want.Lat {
		t.Errorf("lat = %v, want the fitted %v", m.Props["lat"], want.Lat)
	}
	if zoom, _ := m.Props["zoom"].(float64); zoom != want.Zoom {
		t.Errorf("zoom = %v, want the fitted %v", m.Props["zoom"], want.Zoom)
	}
}

// A set with nothing in it renders the empty state and no map. The alternative
// is a map of 0,0, which is a real place in the Gulf of Guinea.
func TestMapPanelWithNoPinsDrawsNoMap(t *testing.T) {
	n := renderMapPanel(t, MapPanel{Empty: EmptyState{Title: "No events with a location"}})
	if findFirst(n, func(n *core.Node) bool { return n.Type == "MapView" }) != nil {
		t.Error("an empty panel drew a map")
	}
	if findText(n, "No events with a location") == nil {
		t.Error("the caller's empty state title is missing")
	}
}

// And a caller who said nothing still gets a sentence rather than a blank.
func TestMapPanelSuppliesAnEmptyStateTitle(t *testing.T) {
	n := renderMapPanel(t, MapPanel{})
	found := findFirst(n, func(n *core.Node) bool {
		text, _ := n.Props["content"].(string)
		return n.Type == "Text" && strings.TrimSpace(text) != ""
	})
	if found == nil {
		t.Error("an empty panel with no configured empty state said nothing at all")
	}
}

// The fit, which is the whole of what this widget adds to core.MapView. The
// assertions are on the ground the view covers rather than on the zoom number:
// what a caller needs is that every pin is on screen.
func TestFitRegionContainsEveryPin(t *testing.T) {
	cases := []struct {
		name    string
		pins    []MapPin
		clamped bool // the widest view this returns does not contain them
		why     string
	}{
		{"one pin", []MapPin{{Lat: 38.7223, Lng: -9.1393}},
			false, "a zero spread has no logarithm; the floor is what answers it"},
		{"two in a city", twoPins, false, "a few hundred metres apart"},
		{"two in a region", []MapPin{{Lat: 38.7223, Lng: -9.1393}, {Lat: 41.1579, Lng: -8.6291}},
			false, "Lisbon and Porto"},
		{"a high-latitude pair", []MapPin{{Lat: 64.1466, Lng: -21.9426}, {Lat: 64.13, Lng: -21.82}},
			false, "a longitude degree in Reykjavik is under half the width of one at the equator"},
		{"across the equator", []MapPin{{Lat: 1, Lng: 10}, {Lat: -1, Lng: 10}},
			false, "a span straddling zero is still a span"},
		{"across a hemisphere", []MapPin{{Lat: -33.8688, Lng: 151.2093}, {Lat: 51.5074, Lng: -0.1278}},
			false, "Sydney and London, 151 degrees apart: the widest set this " +
				"still contains, and the reason MinFitZoom is 1 rather than 2"},
		{"across the antimeridian", []MapPin{{Lat: 35.6812, Lng: 139.7671}, {Lat: 21.3069, Lng: -157.8583}},
			true, "Tokyo and Honolulu are 62 degrees apart going east and are " +
				"measured here as 298 going the other way — the antimeridian " +
				"limitation FitRegion's doc states, pinned so that fixing it is " +
				"a change to this line and not a surprise"},
	}

	for _, c := range cases {
		region, ok := FitRegion(c.pins)
		if !ok {
			t.Errorf("%s: no region for %d pins", c.name, len(c.pins))
			continue
		}
		if region.Zoom < MinFitZoom || region.Zoom > MaxFitZoom {
			t.Errorf("%s: zoom %g is outside the tile pyramid — %s", c.name, region.Zoom, c.why)
		}
		if c.clamped {
			// The documented failure, asserted rather than skipped: a set this
			// wide gets the widest view available and not a containing one, and
			// the thing that must not change silently is which of the two it is.
			if region.Zoom != MinFitZoom {
				t.Errorf("%s: zoom %g, want the clamp at %g — %s",
					c.name, region.Zoom, MinFitZoom, c.why)
			}
			continue
		}
		// Half the view's height in degrees of latitude, from the same slippy
		// relationship the zoom came out of.
		halfSpan := 360 / math.Pow(2, region.Zoom) / 2
		lngHalfSpan := halfSpan / math.Cos(region.Lat*math.Pi/180)
		for _, p := range c.pins {
			if math.Abs(p.Lat-region.Lat) > halfSpan {
				t.Errorf("%s: a pin at lat %g is off the opening view (centre %g ± %g) — %s",
					c.name, p.Lat, region.Lat, halfSpan, c.why)
			}
			if math.Abs(p.Lng-region.Lng) > lngHalfSpan {
				t.Errorf("%s: a pin at lng %g is off the opening view (centre %g ± %g) — %s",
					c.name, p.Lng, region.Lng, lngHalfSpan, c.why)
			}
		}
	}
}

// Containing every pin is easy by zooming out to the whole world, which is not
// a map of anything. The other half of the claim.
func TestFitRegionDoesNotShowAContinentForOneStreet(t *testing.T) {
	near, _ := FitRegion(twoPins)
	if near.Zoom < 13 {
		t.Errorf("two pins a few hundred metres apart opened at zoom %g, which is a country", near.Zoom)
	}
	single, _ := FitRegion([]MapPin{{Lat: 38.7223, Lng: -9.1393}})
	if single.Zoom < 15 {
		t.Errorf("one pin opened at zoom %g, which is a city", single.Zoom)
	}
}

// An empty set has no answer, and specifically not 0,0. The boolean is the
// whole point: a caller that ignored it would render the Gulf of Guinea.
func TestFitRegionRefusesAnEmptySet(t *testing.T) {
	region, ok := FitRegion(nil)
	if ok {
		t.Error("FitRegion answered for no pins")
	}
	if region != (core.Region{}) {
		t.Errorf("region = %+v, want the zero value", region)
	}
}

// A stated Zoom keeps the centring and drops the fit, which is the one knob
// this widget offers over its own arithmetic.
func TestAStatedZoomOverridesTheFit(t *testing.T) {
	m := mapNode(t, renderMapPanel(t, MapPanel{Pins: twoPins, Zoom: 11}))
	if zoom, _ := m.Props["zoom"].(float64); zoom != 11 {
		t.Errorf("zoom = %v, want the caller's 11", m.Props["zoom"])
	}
	want, _ := FitRegion(twoPins)
	if lat, _ := m.Props["lat"].(float64); lat != want.Lat {
		t.Errorf("a stated zoom moved the centre: lat = %v, want %v", m.Props["lat"], want.Lat)
	}
}

// The count caption, which exists because "places" and "things" are different
// numbers and only one of them is true of a map.
func TestPlaceCountSaysPlaces(t *testing.T) {
	for _, c := range []struct {
		n    int
		want string
	}{{0, "0 places"}, {1, "1 place"}, {2, "2 places"}} {
		if got := PlaceCount(c.n); got != c.want {
			t.Errorf("PlaceCount(%d) = %q, want %q", c.n, got, c.want)
		}
	}
}

// walk visits every node in a tree. findFirst stops at the first match; the
// marker count needs all of them.
func walk(n *core.Node, fn func(*core.Node)) {
	if n == nil {
		return
	}
	fn(n)
	for _, c := range n.Children {
		walk(c, fn)
	}
}
