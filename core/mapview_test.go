package core

import (
	"math"
	"testing"
)

// The node a MapView builds: the region as three props, and the markers as
// keyed children rather than as a prop of their own.
func TestMapViewCarriesItsRegionAndKeysItsMarkers(t *testing.T) {
	ctx := NewContext()
	n := MapView(Region{Lat: 38.7223, Lng: -9.1393, Zoom: 14},
		Marker("hall", 38.7223, -9.1393, "The hall"),
		Marker("annex", 38.7251, -9.1402, "The annex"),
	).Render(ctx)

	if n.Type != "MapView" {
		t.Fatalf("type = %q", n.Type)
	}
	if n.Props["lat"] != 38.7223 || n.Props["lng"] != -9.1393 || n.Props["zoom"] != 14.0 {
		t.Errorf("props = %v", n.Props)
	}
	if len(n.Children) != 2 {
		t.Fatalf("%d children, want the two markers", len(n.Children))
	}
	// Keys, which are what make a moved marker one patch rather than a rebuilt
	// annotation layer. See core.Marker.
	if n.Children[0].Key != "marker:hall" || n.Children[1].Key != "marker:annex" {
		t.Errorf("marker keys = %q, %q", n.Children[0].Key, n.Children[1].Key)
	}
	if n.Children[0].Type != "Marker" {
		t.Errorf("child type = %q, want Marker", n.Children[0].Type)
	}
	if n.Children[0].Props["id"] != "hall" || n.Children[0].Props["title"] != "The hall" {
		t.Errorf("marker props = %v", n.Children[0].Props)
	}
}

// A zero Zoom is the default rather than the whole world, and the coordinates
// go through location.go's two rules — clamp the latitude, wrap the longitude.
func TestARegionIsNormalisedBeforeItReachesAHost(t *testing.T) {
	ctx := NewContext()

	n := MapView(Region{Lat: 1, Lng: 2}).Render(ctx)
	if n.Props["zoom"] != DefaultMapZoom {
		t.Errorf("zoom = %v, want the default %v — a map with no zoom is not a map of the world",
			n.Props["zoom"], DefaultMapZoom)
	}

	n = MapView(Region{Lat: 91, Lng: 190, Zoom: 30}).Render(ctx)
	if n.Props["lat"] != 90.0 {
		t.Errorf("lat = %v, want clamped to 90", n.Props["lat"])
	}
	if n.Props["lng"] != -170.0 {
		t.Errorf("lng = %v, want wrapped to -170", n.Props["lng"])
	}
	if n.Props["zoom"] != 22.0 {
		t.Errorf("zoom = %v, want clamped to 22 — deeper is a grey screen", n.Props["zoom"])
	}
}

// A marker's own coordinates go through the same rules. A marker is the one
// place a caller's loop over server data lands directly in a host's annotation
// API, so an out-of-range coordinate has to be caught here rather than there.
func TestAMarkersCoordinatesAreNormalisedToo(t *testing.T) {
	n := Marker("x", 100, 200, "").Render(NewContext())
	if n.Props["lat"] != 90.0 || n.Props["lng"] != -160.0 {
		t.Errorf("marker at %v, %v; want 90, -160", n.Props["lat"], n.Props["lng"])
	}
}

// An unnamed marker is allowed and keys nothing: there is nothing to tell it
// apart from, and nothing will ever report a tap on it by name.
func TestAnUnnamedMarkerCarriesNoKey(t *testing.T) {
	n := Marker("", 1, 2, "You are here").Render(NewContext())
	if n.Key != "" {
		t.Errorf("key = %q, want none", n.Key)
	}
}

// The three callbacks, each registered only when it has a handler: a prop map
// carrying a live callback ID for a handler nobody wrote is an ID the render
// pass's sweep has to keep alive for nothing.
func TestMapCallbacksAreRegisteredOnlyWhenGiven(t *testing.T) {
	ctx := NewContext()
	bare := MapView(Region{Lat: 1, Lng: 2},
		OnRegionChange(nil), OnMarkerTap(nil), OnMapTap(nil),
	).Render(ctx)
	for _, key := range []string{"onRegionChange", "onMarkerTap", "onMapTap"} {
		if _, has := bare.Props[key]; has {
			t.Errorf("nil handler registered %s: %v", key, bare.Props)
		}
	}
	if _, has := bare.Props["showUser"]; has {
		t.Errorf("showUser is set on a map that did not ask: %v", bare.Props)
	}

	var region Region
	var marker string
	var tapLat, tapLng float64
	n := MapView(Region{Lat: 1, Lng: 2},
		ShowUserLocation(),
		OnRegionChange(func(r Region) { region = r }),
		OnMarkerTap(func(id string) { marker = id }),
		OnMapTap(func(lat, lng float64) { tapLat, tapLng = lat, lng }),
	).Render(ctx)

	if n.Props["showUser"] != true {
		t.Errorf("showUser = %v", n.Props["showUser"])
	}
	ctx.TriggerTextCallback(n.Props["onRegionChange"].(string), "38.7223,-9.1393,16.5")
	if region != (Region{Lat: 38.7223, Lng: -9.1393, Zoom: 16.5}) {
		t.Errorf("region = %+v", region)
	}
	ctx.TriggerTextCallback(n.Props["onMarkerTap"].(string), "hall")
	if marker != "hall" {
		t.Errorf("marker = %q", marker)
	}
	ctx.TriggerTextCallback(n.Props["onMapTap"].(string), "10.5,-20.25")
	if tapLat != 10.5 || tapLng != -20.25 {
		t.Errorf("tap = %v, %v", tapLat, tapLng)
	}
}

// A payload that does not parse is dropped, never delivered as zeros. 0,0 is a
// real place, and an app that recentred there because one host formatted a
// float with a comma would be looking at the Gulf of Guinea with no error
// anywhere.
func TestAnUnparseablePayloadIsDroppedRatherThanDeliveredAsZeros(t *testing.T) {
	ctx := NewContext()
	calls := 0
	n := MapView(Region{Lat: 1, Lng: 2},
		OnRegionChange(func(Region) { calls++ }),
		OnMapTap(func(float64, float64) { calls++ }),
	).Render(ctx)

	region := n.Props["onRegionChange"].(string)
	tap := n.Props["onMapTap"].(string)
	for _, bad := range []string{"", "nope", "1,2", "1,2,3,4", "1;2;3", "a,b,c"} {
		ctx.TriggerTextCallback(region, bad)
	}
	for _, bad := range []string{"", "1", "1,2,3", "x,y"} {
		ctx.TriggerTextCallback(tap, bad)
	}
	if calls != 0 {
		t.Errorf("%d handler calls from unparseable payloads", calls)
	}

	// And the good case still arrives, so the guard above is not simply off.
	ctx.TriggerTextCallback(region, "1,2,3")
	if calls != 1 {
		t.Errorf("a well-formed region did not reach the handler (%d calls)", calls)
	}
}

// The wire format, both ways, in one test: what Go writes is what Go reads, and
// the shortest round-trip form is what the three host parsers are written
// against.
func TestTheWireFormatRoundTrips(t *testing.T) {
	for _, r := range []Region{
		{Lat: 38.7223, Lng: -9.1393, Zoom: 14},
		{Lat: 0, Lng: 0, Zoom: 1},
		{Lat: -33.8688, Lng: 151.2093, Zoom: 16.25},
	} {
		want := r.normalised()
		got, ok := ParseRegion(FormatRegion(want))
		if !ok {
			t.Fatalf("FormatRegion(%+v) did not parse back", want)
		}
		if math.Abs(got.Lat-want.Lat) > 1e-9 || math.Abs(got.Lng-want.Lng) > 1e-9 ||
			math.Abs(got.Zoom-want.Zoom) > 1e-9 {
			t.Errorf("round trip: %+v -> %q -> %+v", want, FormatRegion(want), got)
		}
	}

	// No exponent form, which is the reason the formatter is 'f' and not 'g': a
	// hand-written host parser splitting on commas does not expect "1e-05".
	if got := FormatLatLng(0.00001, 0.0000001); got != "0.00001,0.0000001" {
		t.Errorf("FormatLatLng small values = %q, want plain decimals", got)
	}
}

// A region read off a host is normalised on the way in as well as on the way
// out, so an app that echoes OnRegionChange straight back into MapView gets the
// same numbers it would have written itself. Without this, a user dragging past
// the antimeridian hands back a longitude of 181 that the app stores and
// re-renders, and the map jumps.
func TestAHostsRegionIsNormalisedOnTheWayIn(t *testing.T) {
	got, ok := ParseRegion("91,181,0")
	if !ok {
		t.Fatal("a well-formed region did not parse")
	}
	if got.Lat != 90 || got.Lng != -179 || got.Zoom != DefaultMapZoom {
		t.Errorf("parsed %+v, want the clamped, wrapped, defaulted region", got)
	}
}

// Extra children beyond the markers are kept, in order: a map is a container,
// and a legend or a recentre button laid over it is the CameraView overlay
// shape.
func TestAMapKeepsItsNonMarkerChildren(t *testing.T) {
	n := MapView(Region{Lat: 1, Lng: 2},
		Marker("a", 1, 2, ""),
		Text("Legend"),
	).Render(NewContext())
	if len(n.Children) != 2 {
		t.Fatalf("%d children", len(n.Children))
	}
	if n.Children[1].Type != "Text" {
		t.Errorf("second child = %q, want the overlay Text", n.Children[1].Type)
	}
}

// A caller's style props reach the node, which is what lets a map be sized —
// and a map with no size is a zero-height box on every target.
func TestAMapTakesItsSizeFromStyleProps(t *testing.T) {
	n := MapView(Region{Lat: 1, Lng: 2}, Width("100%"), Height("280px")).Render(NewContext())
	if n.Style == nil || n.Style.Width != "100%" || n.Style.Height != "280px" {
		t.Errorf("style = %+v", n.Style)
	}
}
