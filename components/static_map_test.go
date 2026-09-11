package components

import (
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

// renderStaticMap renders m against a fresh context and returns the root node.
func renderStaticMap(t *testing.T, m StaticMap) *core.Node {
	t.Helper()
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	return m.Render(ctx)
}

// mapImage finds the one Image node in a rendered map.
func mapImage(t *testing.T, n *core.Node) *core.Node {
	t.Helper()
	img := findFirst(n, func(n *core.Node) bool { return n.Type == "Image" })
	if img == nil {
		t.Fatal("no Image in the rendered map")
	}
	return img
}

// The defaults, which are most of what this widget is: a caller who states a
// coordinate and a provider gets a street-level card-width map.
//
// The provider is stated because there is no longer a default one to inherit
// — see StaticMap's "The image is a network fetch, and the provider is
// required" — so this test drives OSMStaticMap by name to pin the URL shape a
// provider is asked for, which is a different claim from whether that host is
// still up.
func TestStaticMapDefaultsToAStreetLevelImage(t *testing.T) {
	n := renderStaticMap(t, StaticMap{Lat: 38.7223, Lng: -9.1393, Provider: OSMStaticMap})
	src, _ := mapImage(t, n).Props["src"].(string)

	for _, want := range []string{
		"staticmap.openstreetmap.de",
		"center=38.7223%2C-9.1393",
		"zoom=15",
		"size=320x180",
	} {
		if !strings.Contains(src, want) {
			t.Errorf("src lacks %q:\n%s", want, src)
		}
	}
	// No marker unless asked: see the field.
	if strings.Contains(src, "markers") {
		t.Errorf("an unasked-for marker reached the URL:\n%s", src)
	}
}

// A widget with no Provider draws nothing and says so. Both halves matter:
// the empty src is what a user sees, and the concern is the only thing that
// distinguishes "nobody configured this" from "the fetch failed" — which are
// the same grey rectangle on screen.
func TestAMapWithNoProviderRendersNothingAndReportsIt(t *testing.T) {
	core.SetDebugMode(true)
	core.ClearConcerns()
	defer func() { core.SetDebugMode(false); core.ClearConcerns() }()

	n := renderStaticMap(t, StaticMap{Lat: 38.7223, Lng: -9.1393})
	if src, _ := mapImage(t, n).Props["src"].(string); src != "" {
		t.Errorf("src = %q, want empty: there is no provider to have built one", src)
	}
	if !hasConcern(ConcernNoMapProvider) {
		t.Errorf("no %s concern; a build that has not chosen a provider is told nothing",
			ConcernNoMapProvider)
	}
}

// And the concern is a development-time cost only. A release build renders the
// same empty frame without paying for the Sprintf that describes it.
func TestTheNoProviderConcernCostsNothingOutsideDebug(t *testing.T) {
	core.ClearConcerns()
	defer core.ClearConcerns()

	renderStaticMap(t, StaticMap{Lat: 1, Lng: 2})
	if hasConcern(ConcernNoMapProvider) {
		t.Error("a concern was recorded with debug mode off")
	}
}

// A provider is handed values that are already resolved, so every provider is
// spared the defaulting and none of them can disagree about what a zero means.
// This is the contract StaticMapArea's doc states, pinned by observing it.
func TestAProviderSeesResolvedValues(t *testing.T) {
	var seen StaticMapArea
	renderStaticMap(t, StaticMap{
		Lat: 10, Lng: 20,
		Marker:   true,
		Provider: func(a StaticMapArea) string { seen = a; return "x" },
	})
	want := StaticMapArea{Lat: 10, Lng: 20, Zoom: DefaultMapZoom,
		Width: DefaultMapWidth, Height: DefaultMapHeight,
		Scale: DefaultMapScale, Marker: true}
	if seen != want {
		t.Errorf("provider saw %+v, want %+v", seen, want)
	}
}

// The size clamp. A request past what the providers serve comes back as a
// broken image with no error anywhere, which is the failure the clamp turns
// into a slightly smaller map.
func TestOversizedMapsAreClampedBeforeTheyAreRequested(t *testing.T) {
	var seen StaticMapArea
	renderStaticMap(t, StaticMap{
		Width: 4000, Height: 4000,
		Provider: func(a StaticMapArea) string { seen = a; return "x" },
	})
	if seen.Width != MaxMapDimension || seen.Height != MaxMapDimension {
		t.Errorf("size = %dx%d, want both clamped to %d", seen.Width, seen.Height, MaxMapDimension)
	}
}

// Latitude clamps and longitude wraps, which are two different geographic
// facts rather than one rule applied twice. A clamped longitude would move the
// point to the antimeridian; a wrapped latitude would move it to the other
// hemisphere.
func TestLatitudeClampsAndLongitudeWraps(t *testing.T) {
	cases := []struct {
		lat, lng float64
		wantLat  float64
		wantLng  float64
		why      string
	}{
		{89, 0, MercatorLatLimit, 0, "past Mercator's limit, held at the edge"},
		{-89, 0, -MercatorLatLimit, 0, "the same, southward"},
		{0, 190, 0, -170, "190°E is 170°W — the same meridian, the other spelling"},
		{0, -190, 0, 170, "and back the other way"},
		{0, 180, 0, 180, "the antimeridian itself is a longitude, not an overflow"},
		{45.5, -73.6, 45.5, -73.6, "an ordinary coordinate passes through untouched"},
	}
	for _, c := range cases {
		var seen StaticMapArea
		renderStaticMap(t, StaticMap{Lat: c.lat, Lng: c.lng,
			Provider: func(a StaticMapArea) string { seen = a; return "x" }})
		if seen.Lat != c.wantLat || seen.Lng != c.wantLng {
			t.Errorf("(%g, %g) -> (%g, %g), want (%g, %g): %s",
				c.lat, c.lng, seen.Lat, seen.Lng, c.wantLat, c.wantLng, c.why)
		}
	}
}

// A tappable map is a *link*, not a button, because the tap leaves the app —
// the distinction core.RoleLink's doc draws and the one a reader needs before
// they follow it.
func TestATappableMapIsALinkThatOpensTheHandoff(t *testing.T) {
	n := renderStaticMap(t, StaticMap{Lat: 1, Lng: 2, Label: "The hall"})
	if n.Style == nil || n.Style.AccessibilityRole != core.RoleLink {
		t.Fatalf("role = %q, want link", roleOf(n))
	}
	if n.Props["onClick"] == nil {
		t.Error("no onClick — a map with a hand-off must be tappable")
	}
	if got := n.Style.AccessibilityLabel; got != "Map of The hall" {
		t.Errorf("label = %q, want the place's name", got)
	}
	if n.Style.AccessibilityHint == "" {
		t.Error("a tappable map should say what tapping does")
	}
}

// And a map with nothing to tap is a picture: core.RoleImg, no callback, no
// hint about an app it will not open. The empty answer from a Handoff is a
// caller declining the link, which is a supported thing to say.
func TestAMapWithNoHandoffIsAPicture(t *testing.T) {
	n := renderStaticMap(t, StaticMap{
		Lat: 1, Lng: 2,
		Handoff: func(lat, lng float64, label string) string { return "" },
	})
	if n.Style == nil || n.Style.AccessibilityRole != core.RoleImg {
		t.Fatalf("role = %q, want img", roleOf(n))
	}
	if n.Props["onClick"] != nil {
		t.Error("a map with no hand-off registered a callback")
	}
	if n.Style.AccessibilityHint != "" {
		t.Errorf("hint = %q on a map that opens nothing", n.Style.AccessibilityHint)
	}
}

// OnTap replaces the hand-off rather than composing with it. Both firing would
// leave a caller no way to ask for either one alone.
func TestOnTapReplacesTheHandoff(t *testing.T) {
	var tapped int
	handoffAsked := false

	// Rendered against a context this test keeps, so the registered callback
	// can be fired: that the map is tappable says nothing about *whose*
	// handler it carries, which is the actual claim.
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	n := StaticMap{
		Lat: 1, Lng: 2,
		OnTap:   func() { tapped++ },
		Handoff: func(lat, lng float64, label string) string { handoffAsked = true; return "https://example.test" },
	}.Render(ctx)

	if handoffAsked {
		t.Error("the hand-off was built for a map that has its own OnTap")
	}
	id, ok := n.Props["onClick"].(string)
	if !ok {
		t.Fatalf("onClick = %v, want a callback ID", n.Props["onClick"])
	}
	if tapped != 0 {
		t.Fatal("the handler ran during the render")
	}
	ctx.TriggerCallback(id)
	if tapped != 1 {
		t.Errorf("the tap ran the caller's handler %d times, want 1", tapped)
	}
}

// The map with no name at all still says where it is. "Map" alone says less
// than the numbers do.
func TestAnUnnamedMapAnnouncesItsCoordinates(t *testing.T) {
	n := renderStaticMap(t, StaticMap{Lat: 38.7223, Lng: -9.1393})
	if got := n.Style.AccessibilityLabel; got != "Map at 38.7223, -9.1393" {
		t.Errorf("label = %q, want the coordinates", got)
	}
}

// The image inside is hidden, so the widget announces once. An unhidden <img>
// with no alt inside a role="link" is read out as its own URL on the web, and
// a provider URL is a minute of query string.
func TestTheImageInsideIsHidden(t *testing.T) {
	img := mapImage(t, renderStaticMap(t, StaticMap{Lat: 1, Lng: 2}))
	if img.Style == nil || !img.Style.AccessibilityHidden {
		t.Error("the map image is not hidden; the container speaks for it")
	}
}

// A caller's Style lands after the widget's own, which is what makes it an
// override rather than a suggestion. Pinned on the size, because that is the
// prop a caller is most likely to want back.
func TestCallerStyleOverridesTheWidgetsOwn(t *testing.T) {
	n := renderStaticMap(t, StaticMap{Lat: 1, Lng: 2,
		Style: []core.StyleProp{core.Width("100%")}})
	if n.Style.Width != "100%" {
		t.Errorf("width = %q, want the caller's 100%%", n.Style.Width)
	}
}

// An unconfigured Google provider renders no image rather than a picture of
// Google's "not authorized" tile: a misconfigured build should look
// unfinished, not broken.
func TestGoogleStaticMapWithNoKeyRendersNoImage(t *testing.T) {
	if got := GoogleStaticMap("")(StaticMapArea{Zoom: 15, Width: 10, Height: 10}); got != "" {
		t.Errorf("keyless provider returned %q", got)
	}
	src := GoogleStaticMap("abc123")(StaticMapArea{Lat: 1.5, Lng: -2.25, Zoom: 12, Width: 300, Height: 200, Marker: true})
	for _, want := range []string{
		"maps.googleapis.com", "center=1.5%2C-2.25", "zoom=12",
		"size=300x200", "markers=color%3Ared%7C1.5%2C-2.25", "key=abc123",
	} {
		if !strings.Contains(src, want) {
			t.Errorf("google src lacks %q:\n%s", want, src)
		}
	}
}

// The two hand-offs, and the claim each one makes. The coordinates are in both
// because they are the fact; the label is in neither, for the reason
// GoogleMapsHandoff states.
func TestTheHandoffsCarryTheCoordinates(t *testing.T) {
	g := GoogleMapsHandoff(38.7223, -9.1393, "Somewhere")
	if !strings.Contains(g, "query=38.7223%2C-9.1393") || !strings.Contains(g, "api=1") {
		t.Errorf("google hand-off = %q", g)
	}
	if strings.Contains(g, "Somewhere") {
		t.Errorf("the label reached a URL that geocodes it: %q", g)
	}

	o := OpenStreetMapHandoff(38.7223, -9.1393, "Somewhere")
	for _, want := range []string{"mlat=38.7223", "mlon=-9.1393", "#map=15/38.7223/-9.1393"} {
		if !strings.Contains(o, want) {
			t.Errorf("osm hand-off lacks %q: %q", want, o)
		}
	}
}

// roleOf reads a node's role for a failure message, nil-Style included.
func roleOf(n *core.Node) core.Role {
	if n == nil || n.Style == nil {
		return core.RoleNone
	}
	return n.Style.AccessibilityRole
}

// Scale is the device pixel ratio, and the whole of its claim is that it
// changes the image without changing the box. A widget that grew with its
// scale would be a widget no layout could hold.
func TestScaleChangesTheRequestAndNotTheFrame(t *testing.T) {
	n := renderStaticMap(t, StaticMap{
		Lat: 1, Lng: 2, Scale: 2,
		Provider: GoogleStaticMap("k"),
	})
	if n.Style.Width != "320px" || n.Style.Height != "180px" {
		t.Errorf("frame = %s x %s, want the logical 320px x 180px",
			n.Style.Width, n.Style.Height)
	}
	src, _ := mapImage(t, n).Props["src"].(string)
	if !strings.Contains(src, "size=320x180") {
		t.Errorf("size should stay logical — Google scales it itself:\n%s", src)
	}
	if !strings.Contains(src, "scale=2") {
		t.Errorf("src lacks scale=2:\n%s", src)
	}
}

// The scale a provider is handed, across the range a caller can write. The
// clamp is MaxMapScale and the default is 1x — the ratio every caller got
// before the field existed, which is what makes adding it a no-op for them.
func TestScaleIsDefaultedAndClampedBeforeAProviderSeesIt(t *testing.T) {
	cases := []struct {
		in, want int
		why      string
	}{
		{0, DefaultMapScale, "unset is 1x, the ratio callers had before this field"},
		{1, 1, "1x asked for is 1x"},
		{2, 2, "the ordinary retina phone"},
		{3, 3, "the deepest ratio shipping phones use"},
		{4, MaxMapScale, "past the clamp, held at it"},
		{-1, DefaultMapScale, "a negative is not a ratio; it means the same as unset"},
	}
	for _, c := range cases {
		var seen StaticMapArea
		renderStaticMap(t, StaticMap{Lat: 1, Lng: 2, Scale: c.in,
			Provider: func(a StaticMapArea) string { seen = a; return "x" }})
		if seen.Scale != c.want {
			t.Errorf("Scale %d -> %d, want %d: %s", c.in, seen.Scale, c.want, c.why)
		}
	}
}

// Google accepts 1 or 2 and nothing else, so a 3x device gets the 2x image.
// The clamp lives in the provider rather than in Area() because it is one
// service's limit: a provider that can serve 3x must still be handed the 3.
func TestGoogleSpendsAtMostTwoOfWhateverScaleItIsHanded(t *testing.T) {
	for _, scale := range []int{2, 3} {
		src := GoogleStaticMap("k")(StaticMapArea{Zoom: 15, Width: 320, Height: 180, Scale: scale})
		if !strings.Contains(src, "scale=2") {
			t.Errorf("scale %d produced %q, want scale=2", scale, src)
		}
	}
	// And 1x writes no parameter at all: scale=1 is the API's own default, so
	// a parameter saying it adds a byte and no information.
	src := GoogleStaticMap("k")(StaticMapArea{Zoom: 15, Width: 320, Height: 180, Scale: 1})
	if strings.Contains(src, "scale") {
		t.Errorf("a 1x request wrote a scale parameter: %s", src)
	}
}

// OSMStaticMap ignores Scale, which is the case the field's doc names: a
// provider with no scale of its own serves 1x whatever it is asked. Pinned
// because a silently-honoured scale would be a URL parameter the dead service
// never had.
func TestOSMStaticMapIgnoresScale(t *testing.T) {
	one := OSMStaticMap(StaticMapArea{Lat: 1, Lng: 2, Zoom: 15, Width: 320, Height: 180, Scale: 1})
	two := OSMStaticMap(StaticMapArea{Lat: 1, Lng: 2, Zoom: 15, Width: 320, Height: 180, Scale: 2})
	if one != two {
		t.Errorf("scale changed a URL that has no scale parameter:\n%s\n%s", one, two)
	}
}
