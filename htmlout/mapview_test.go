package htmlout

import (
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

func exportedMap(t *testing.T, props ...core.PropsAndChildren) string {
	t.Helper()
	ctx := core.NewContext()
	return ExportHTML(core.MapView(core.Region{Lat: 38.7223, Lng: -9.1393, Zoom: 14},
		props...).Render(ctx))
}

// A map exports as a placeholder box that has not lost the region it was
// looking at. An export has no engine and no tiles; what it can do is keep the
// information, which is what makes the same div upgradeable by a page that does
// load Leaflet.
func TestAMapExportsItsRegionAsData(t *testing.T) {
	out := exportedMap(t, core.Width("100%"), core.Height("280px"))
	for _, want := range []string{
		`data-lat="38.7223"`, `data-lng="-9.1393"`, `data-zoom="14"`,
		"width:100%", "height:280px",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("map export lacks %s:\n%s", want, out)
		}
	}
	// The request for the blue dot is recorded only when it was made, so an
	// ordinary map exports what it always did.
	if strings.Contains(out, "data-show-user") {
		t.Errorf("an unasked-for user location reached the export:\n%s", out)
	}
	if out := exportedMap(t, core.ShowUserLocation()); !strings.Contains(out, `data-show-user="true"`) {
		t.Errorf("ShowUserLocation did not reach the export:\n%s", out)
	}
}

// Markers export as data too: id, position, and the callout when there is one.
// They are elements rather than nothing because a patch path is positional —
// see the Marker row in the tag table.
func TestMarkersExportAsPositionedData(t *testing.T) {
	out := exportedMap(t,
		core.Marker("hall", 38.7223, -9.1393, "The hall"),
		core.Marker("annex", 38.7251, -9.1402, ""),
	)
	for _, want := range []string{
		`data-marker-id="hall"`, `data-title="The hall"`,
		`data-marker-id="annex"`, `data-lat="38.7251"`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("marker export lacks %s:\n%s", want, out)
		}
	}
	// One title, for the one marker that has one: an attribute with an empty
	// value is one a reader has to test for rather than look up.
	if got := strings.Count(out, "data-title"); got != 1 {
		t.Errorf("%d title attributes, want 1:\n%s", got, out)
	}
	// And an unnamed marker carries no id attribute, on the same rule.
	out = exportedMap(t, core.Marker("", 1, 2, ""))
	if strings.Contains(out, "data-marker-id") {
		t.Errorf("an unnamed marker exported an empty id:\n%s", out)
	}
}

// The three callback IDs ride out as data attributes, like every other callback
// this exporter records. That is the half that makes a static export into a live
// map for a loader that wires them — the region is in the document and the
// handlers are addressable.
func TestAMapExportsItsCallbackIDs(t *testing.T) {
	out := exportedMap(t,
		core.OnRegionChange(func(core.Region) {}),
		core.OnMarkerTap(func(string) {}),
		core.OnMapTap(func(float64, float64) {}),
	)
	for _, want := range []string{"data-onregionchange=", "data-onmarkertap=", "data-onmaptap="} {
		if !strings.Contains(out, want) {
			t.Errorf("map export lacks %s:\n%s", want, out)
		}
	}
}

// A marker is childless and closes immediately; a map is a container and keeps
// whatever was laid over it. The pair is what says the two node types are
// different kinds of thing rather than one with a flag.
func TestAMapIsAContainerAndAMarkerIsNot(t *testing.T) {
	out := exportedMap(t,
		core.Marker("a", 1, 2, ""),
		core.Text("Legend"),
	)
	if !strings.Contains(out, "Legend") {
		t.Errorf("the overlay child did not reach the export:\n%s", out)
	}
	// Both node types are in the shared tag table rather than falling through
	// to the default, which is what the table's census property is for.
	for _, nodeType := range []string{"MapView", "Marker"} {
		if TagFor(nodeType) != "div" {
			t.Errorf("TagFor(%s) = %q, want div", nodeType, TagFor(nodeType))
		}
		if _, ok := Tags()[nodeType]; !ok {
			t.Errorf("%s is not in the tag table; it would fall back to the default "+
				"and the WASM runtime's copy would not be checked for it", nodeType)
		}
	}
}

// Neither node type resets the user-agent border, and neither is a form
// control. Both are divs the style owns outright, so there is nothing for the
// browser to draw and nothing to take away.
func TestTheMapNodesAreNeitherFormControlsNorBorderResets(t *testing.T) {
	for _, nodeType := range []string{"MapView", "Marker"} {
		if ResetsUABorder(nodeType) {
			t.Errorf("%s is in borderResetTypes; a <div> has no user-agent frame", nodeType)
		}
		if isFormControl(nodeType) {
			t.Errorf("%s reads as a form control", nodeType)
		}
	}
}
