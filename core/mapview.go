package core

import (
	"strconv"
	"strings"
)

// MapView is a live map: the platform's own map widget, panned and zoomed by
// the user, with markers as child nodes.
//
//	core.MapView(core.Region{Lat: 38.7223, Lng: -9.1393, Zoom: 14},
//	    core.Width("100%"), core.Height("280px"),
//	    core.ShowUserLocation(),
//	    core.OnMarkerTap(func(id string) { open(id) }),
//	    core.Marker("hall", 38.7223, -9.1393, "The hall"),
//	    core.Marker("annex", 38.7251, -9.1402, "The annex"),
//	)
//
// Each host draws its platform's map:
//
//	iOS       MapKit, which is free and needs no key
//	Android   osmdroid over OpenStreetMap tiles — no key, and no dependency on
//	          Play Services being present on the device
//	Browser   Leaflet over OpenStreetMap tiles, loaded by the host page
//	htmlout   a placeholder box, as CameraView is: a static snapshot has no
//	          engine to run and no tiles to fetch
//
// # When to use comps.StaticMap instead
//
// Almost always, if the question is "where is this". A static map is an image
// and a hand-off to the platform's maps app: no engine, no tile budget, no
// key, nothing to keep in step, and the directions the user actually wanted
// come from the app that has their home address in it.
//
// This node is for the cases that need the map to be *part of* the screen: a
// set of markers to compare, a region the user explores, a position they pick
// by tapping. Those are interactions, and an image cannot have them.
//
// # Markers are children, not a prop
//
// The same decision core.TextGrid makes about its rows, for the same reason.
// A marker set sent as one prop means every marker is re-read whenever any of
// them moves: the reconciler sees one changed value and the host rebuilds the
// annotation layer, which on every platform is a visible flicker and on two of
// them loses the selected callout.
//
// As children they are ordinary nodes. The reconciler pairs them by key,
// emits an update-props patch for the one marker that moved, an add for the
// one that appeared and a remove for the one that left — and each host's
// annotation bookkeeping is the patch handling it already has.
//
// Marker keys itself from its id, so a caller writing core.Marker in a loop
// gets stable identity without having to remember core.Keyed. See Marker.
//
// # The map is controlled, with the echo guard a drag needs
//
// Region is Go's statement of where the map should be, and it is applied to
// the host widget *only when it changes*. It is not re-asserted on every
// patch.
//
// That sounds like a detail and it is the whole usability of the node. A map
// is the one widget whose value the user changes continuously by touching it:
// if every render re-centred the host map on Go's Region, then an app that
// does not echo OnRegionChange back into its own state would snap the map back
// under the user's finger on the next unrelated re-render — and an app that
// does echo it would fight its own round trip, because the echo arrives a
// frame late and moves the map again.
//
// So each host remembers the Region it last applied and compares: Go moving
// the map is an instruction, and Go merely re-rendering is not. It is the same
// compromise the text fields and the Slider make — the value shown is Go's
// except where the finger is the authority — and it has to be implemented the
// same way in all three live hosts, which mobile/verify and wasm/verify check.
//
// An app that wants the map pinned to its own state does nothing special: it
// echoes OnRegionChange into state, and every Region it renders is one it
// chose. An app that wants "show me this place, then let the user wander"
// renders a constant Region, which is applied once.
//
// # Two memories, and the one thing that cannot be said
//
// Each host keeps the Region *Go* last asked for and, separately, where the
// *map* last came to rest. They are the same value until somebody touches the
// map, and each direction reads the one that answers its own question — Go
// changing its mind is an instruction; the map already being there is not.
//
// Folding those into one slot is the bug this contract exists to prevent, and
// it shipped in all three hosts: a pan wrote the user's Region into the slot
// the apply path reads, so Go's *unchanged* Region read as a change and the
// next patch to reach the map — a pin dropped, a marker moved, any unrelated
// re-render — snapped the map back. It survived a unit test because a fake map
// can be panned without firing the event a real one always fires, and it was
// found by opening a browser.
//
// The consequence a caller can see is this: re-rendering the *same* Region is
// never a re-centre. An app that pans away and then wants the opening view
// back cannot get it by handing the same numbers over again, because from
// here that is indistinguishable from the unrelated re-render above. The
// remedy is the one this node already recommends — echo OnRegionChange into
// state, so the Region an app renders tracks where the map is and a "back to
// the start" button is a genuine change. There is deliberately no imperative
// recentre command; adding one is a host feature in three languages, and the
// echo costs one line.
//
// # Tiles are somebody else's bandwidth
//
// Two of the three hosts draw OpenStreetMap tiles from the project's own
// servers, which have a usage policy: identify your app, do not bulk download,
// and expect to be blocked if you send a million tile requests a day. An app
// shipping this to a real user base should point its host at a tile provider it
// pays for. That decision lives in each host rather than in this node — it is
// a URL template in osmdroid's configuration and in Leaflet's layer — because
// it is a deployment fact rather than a property of the view.
func MapView(region Region, props ...PropsAndChildren) View {
	return ComponentFunc(func(ctx *Context) *Node {
		region = region.normalised()
		p := map[string]any{
			"lat":  region.Lat,
			"lng":  region.Lng,
			"zoom": region.Zoom,
		}
		// containerNode, not leafNode: markers are children, and a map with an
		// overlay (a legend, a recentre button) is a container whose extra
		// children every host draws over the map the way CameraView draws its
		// overlay.
		return containerNode(ctx, "MapView", Style{}, append([]PropsAndChildren{mapProps(p)}, props...))
	})
}

// mapProps seeds containerNode's node with the map's own props before the
// caller's items run.
//
// containerNode builds its node from a Style and children and has no prop-map
// argument — the leaf builders take one, and a map is the first container with
// intrinsic props. Rather than widen containerNode for one caller, the seed
// arrives as the first BehaviorProp, which is exactly what a BehaviorProp is
// for: it runs before every item a caller wrote, so a caller's own prop can
// still override one of these.
func mapProps(props map[string]any) BehaviorProp {
	return behaviorFunc(func(_ *Context, n *Node) {
		if n.Props == nil {
			n.Props = map[string]any{}
		}
		for k, v := range props {
			n.Props[k] = v
		}
	})
}

// Region is a place and a scale: where a map is looking and how closely.
//
// One struct rather than three floats at every call site, and the same struct
// in both directions — MapView takes one and OnRegionChange hands one back, so
// an app that echoes the user's pan into its own state is storing the type it
// renders from. Two types here (a "MapRegion" in and a "RegionChange" out)
// would differ in nothing and convert at every seam.
type Region struct {
	// Lat and Lng are the centre of the view, in degrees, WGS-84.
	Lat, Lng float64

	// Zoom is the slippy-tile zoom level every one of these engines speaks: 0
	// is the whole world, 19 is a building. Fractional values are allowed and
	// meaningful — a pinch lands between levels, and the hosts report what the
	// user actually reached rather than rounding it.
	//
	// A zero Zoom means DefaultMapZoom rather than "the whole world", which is
	// the one legitimate value this type spends on a default. It is the same
	// trade comps.StaticMap.Zoom makes and for the same reason: a map with
	// no zoom stated is a map somebody forgot to scale, and the world is never
	// what they meant.
	Zoom float64
}

// DefaultMapZoom is the scale a Region with no Zoom is drawn at: a
// neighbourhood, which is close enough to read street names and wide enough to
// hold more than one marker.
//
// 14 rather than comps.DefaultMapZoom's 15, and the difference is the
// difference between the two widgets. A static map answers "where is this
// one place"; a live map is usually showing a set, and one level out is about
// four times the area.
const DefaultMapZoom = 14.0

// normalised applies the zoom default and the coordinate rules — latitude
// clamped, longitude wrapped — so that every host is handed a region that
// exists. The rules themselves are location.go's; see ReceiveLocation for why
// the two coordinates are treated differently.
func (r Region) normalised() Region {
	if r.Zoom <= 0 {
		r.Zoom = DefaultMapZoom
	}
	// 22 is past the deepest tiles any of these engines has, and asking for it
	// yields a blank grey screen rather than a close map. Clamped high enough
	// that an over-zoom request still shows the deepest real tiles.
	if r.Zoom > 22 {
		r.Zoom = 22
	}
	r.Lat = clampLatitude(r.Lat)
	r.Lng = WrapLongitude(r.Lng)
	return r
}

// ShowUserLocation asks the host's map to draw the user's position — the blue
// dot, with the platform's own styling and its own accuracy halo.
//
// It is the host's location, not core.Location's. Every map SDK has this
// built in, reads the OS permission itself, and tells Go nothing about where
// anybody is. A screen that needs the coordinates wants hooks.UseLocation,
// which is a separate feature that happens to need the same permission — see
// core.Location's "What this is not".
//
// The dot needs that permission. On a platform where it has not been granted,
// every one of these hosts draws the map and no dot, with nothing reported: a
// map is still a map. An app that wants to explain the absence has to check
// the permission itself, which is what permission.Location is for.
func ShowUserLocation() BehaviorProp {
	return behaviorFunc(func(_ *Context, n *Node) {
		if n.Props == nil {
			n.Props = map[string]any{}
		}
		n.Props["showUser"] = true
	})
}

// OnRegionChange reports where the user moved the map to, after they stop
// moving it.
//
// "After" is the contract and the hosts enforce it, because a pan is a stream:
// a finger dragging across a map generates a region per frame, and each one
// that crossed the bridge would be a full Go render pass. Every host therefore
// throttles — it reports on the gesture's end, and at a bounded rate during a
// sustained one — which is the same arrangement the sensors have, for the same
// reason, and is checked in the same places.
//
// The region arrives through the text callback channel as "lat,lng,zoom",
// which is how Slider's float crosses and why no new bridge channel was added
// for this node. A payload that does not parse is dropped rather than
// delivered as zeros: 0,0 is a real place, and an app that centred on it
// because of a formatting bug in one host would be looking at the Gulf of
// Guinea with no error anywhere.
func OnRegionChange(fn func(Region)) BehaviorProp {
	return behaviorFunc(func(ctx *Context, n *Node) {
		if fn == nil {
			return
		}
		if n.Props == nil {
			n.Props = map[string]any{}
		}
		n.Props["onRegionChange"] = ctx.registerTextCallback(func(s string) {
			if r, ok := ParseRegion(s); ok {
				fn(r)
			}
		})
	})
}

// OnMarkerTap reports the id of the marker the user tapped — the id given to
// core.Marker, unchanged.
//
// An id rather than the marker's coordinates, because the id is what the app
// has an index of. Two markers can share a position (a building with two
// tenants, a rounded coordinate) and no app wants to identify a row by
// comparing floats.
func OnMarkerTap(fn func(id string)) BehaviorProp {
	return behaviorFunc(func(ctx *Context, n *Node) {
		if fn == nil {
			return
		}
		if n.Props == nil {
			n.Props = map[string]any{}
		}
		n.Props["onMarkerTap"] = ctx.registerTextCallback(fn)
	})
}

// OnMapTap reports where the user tapped on the map, in degrees — the prop a
// "choose a place" screen is built on.
//
// It does not fire for a tap that hit a marker: that is OnMarkerTap's event,
// and a host that sent both would make every marker tap also drop a pin. Each
// host suppresses it the way its own map does — an annotation's hit test runs
// first on all three.
//
// Like OnRegionChange it crosses as text, "lat,lng", and an unparseable
// payload is dropped rather than delivered as 0,0.
func OnMapTap(fn func(lat, lng float64)) BehaviorProp {
	return behaviorFunc(func(ctx *Context, n *Node) {
		if fn == nil {
			return
		}
		if n.Props == nil {
			n.Props = map[string]any{}
		}
		n.Props["onMapTap"] = ctx.registerTextCallback(func(s string) {
			if lat, lng, ok := ParseLatLng(s); ok {
				fn(lat, lng)
			}
		})
	})
}

// Marker is one pin on a MapView: a child node, keyed by its id.
//
//	core.Marker("hall", 38.7223, -9.1393, "The hall")
//
// # It keys itself
//
// The key is "marker:" + id, written here rather than left to the caller,
// which is a departure from how every other keyed child in this framework
// works — core.List's rows are the caller's to key, and forgetting is a
// documented mistake with a debug-mode concern behind it.
//
// The difference is that a marker already carries its identity. The id is
// required, it is what OnMarkerTap reports, and there is no sensible second
// answer to "which marker is this" — so a caller writing core.Keyed around one
// would be restating the id, and a caller forgetting to would get the failure
// keys exist to prevent (a marker layer rebuilt on every change) in the one
// place it is most expensive.
//
// An empty id is allowed and keys nothing, which is the honest answer for the
// single unnamed marker a "you are here" view draws: there is nothing to tell
// it apart from, and nothing will ever report a tap on it by name.
//
// # A marker is data, not a box
//
// It carries no style and draws no element of its own. Every host reads its
// props and creates a native annotation; the node exists so the reconciler can
// address it. Both DOM renderers give it a hidden element, because a patch
// path is positional and a node with no element would put every later patch in
// the wrong place — the same reason htmlout and the WASM runtime disagree about
// Fragment (see htmlout's transparentTypes).
//
// Title is the callout the platform shows when a marker is tapped, and may be
// empty. It is not an accessibility label: a map's annotations are announced by
// each platform's own map accessibility, which this framework does not reach
// into.
func Marker(id string, lat, lng float64, title string, props ...PropsAndChildren) View {
	return ComponentFunc(func(ctx *Context) *Node {
		n := leafNode(ctx, "Marker", Style{}, map[string]any{
			"id":    id,
			"lat":   clampLatitude(lat),
			"lng":   WrapLongitude(lng),
			"title": title,
		}, props)
		if id != "" {
			n.Key = "marker:" + id
		}
		return n
	})
}

// ParseRegion reads a host's "lat,lng,zoom" payload. The bool is false for
// anything that does not parse, and a caller must not substitute zeros: see
// OnRegionChange.
//
// Exported because all three hosts format this string and a test in each
// harness has to read one back. Keeping the parse in one place is also what
// makes the wire format a single fact rather than three.
func ParseRegion(s string) (Region, bool) {
	parts := strings.Split(s, ",")
	if len(parts) != 3 {
		return Region{}, false
	}
	lat, err1 := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	lng, err2 := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
	zoom, err3 := strconv.ParseFloat(strings.TrimSpace(parts[2]), 64)
	if err1 != nil || err2 != nil || err3 != nil {
		return Region{}, false
	}
	// Normalised on the way in as well as on the way out. A host reporting a
	// longitude of 181 after the user dragged past the antimeridian is
	// reporting a real view of a real place, and an app echoing that region
	// back into MapView must get the same number it would have got by writing
	// it itself.
	return Region{Lat: lat, Lng: lng, Zoom: zoom}.normalised(), true
}

// ParseLatLng reads a host's "lat,lng" payload, for OnMapTap. Same contract as
// ParseRegion: false rather than zeros.
func ParseLatLng(s string) (lat, lng float64, ok bool) {
	parts := strings.Split(s, ",")
	if len(parts) != 2 {
		return 0, 0, false
	}
	lat, err1 := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	lng, err2 := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
	if err1 != nil || err2 != nil {
		return 0, 0, false
	}
	return clampLatitude(lat), WrapLongitude(lng), true
}

// FormatRegion writes the wire form of a region, "lat,lng,zoom".
//
// Nothing in Go sends one today — the hosts are the writers, in Swift, Kotlin
// and JavaScript — and this exists so that the format has a Go statement for
// the harnesses to check those three against, and so a Go-side host (a test, an
// embedder driving the tree directly) has the same spelling available rather
// than inventing one.
//
// 'f' with -1 precision: the shortest form that round-trips, so 38.7223 stays
// "38.7223" and a whole degree stays "38". Deliberately not 'g', which switches
// to exponent form for small numbers — "1e-05" is a valid float in Go and is
// not what a hand-written host parser expects to find in a comma-separated
// coordinate.
func FormatRegion(r Region) string {
	return formatCoord(r.Lat) + "," + formatCoord(r.Lng) + "," + formatCoord(r.Zoom)
}

// FormatLatLng writes the wire form of a point, "lat,lng". See FormatRegion.
func FormatLatLng(lat, lng float64) string {
	return formatCoord(lat) + "," + formatCoord(lng)
}

func formatCoord(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}
