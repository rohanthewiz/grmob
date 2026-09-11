package verify

import (
	"strings"
	"testing"
)

// core.MapView's host contract, on both native shells.
//
// # Why this file exists at all
//
// The map is the one node whose correctness is mostly *not* in Go. Each host
// holds an imperative map object and reconciles it against the node, and the
// three rules that make that work — apply the region only when it changes,
// report a gesture only when it ends somewhere else, move a marker rather than
// rebuilding the layer — are decisions each host makes for itself. The web half
// has a real test (wasm/verify/mapview_test.mjs, against a fake Leaflet); the
// iOS half is type-checked by ios/verify and the Kotlin half is compiled by
// nothing in this repository, because that needs gradle and the Android SDK.
//
// So what is held here is the shape of the contract, read out of the source. It
// is a weaker instrument than a test that drives the code, and it is the
// strongest one available for the claim "all three hosts implement the same
// rule" — which is a claim about three files in three languages and is exactly
// the kind of agreement this package exists for.
//
// See switchlabels_test.go for why reading source is the technique and what it
// costs.

var (
	swiftMap  = nativeFile("ios", "GrMob", "Runtime", "GrMobMapView.swift")
	kotlinMap = nativeFile("android", "app", "src", "main", "java", "com",
		"grmob", "runtime", "GrMobMapView.kt")
)

// The echo guard, which is the rule that makes a controlled map usable at all.
//
// Every host must compare Go's region against the one it last applied, and must
// record what it applied on both paths — the apply and the report. A host
// missing either half is not subtly wrong: it either snaps the map out from
// under the user's finger on the next unrelated render, or it reports its own
// recentring back to Go as a gesture the user never made, which an app echoing
// the region turns into an infinite loop.
func TestBothNativeMapsCompareBeforeTheyMove(t *testing.T) {
	for _, pin := range []struct{ file, apply, record, report string }{
		{
			file: swiftMap,
			// The apply path: compared, then remembered, then set.
			apply:  "if let applied, applied.isSame(as: want) { return }",
			record: "applied = want",
			// The report path: the same comparison, the other way round.
			report: "if let applied, applied.isSame(as: next) { return }",
		},
		{
			file:   kotlinMap,
			apply:  "if (holder.hasApplied(lat, lng, zoom)) return",
			record: "holder.remember(lat, lng, zoom)",
			report: "if (holder.hasApplied(lat, lng, zoom)) return",
		},
	} {
		src := valuesIn(t, pin.file)
		if !strings.Contains(src, pin.apply) {
			t.Errorf("%s: Go's region is applied without being compared (%s) — every "+
				"re-render would put the map back where Go last said, under the user's "+
				"finger", pin.file, pin.apply)
		}
		if !strings.Contains(src, pin.record) {
			t.Errorf("%s: the applied region is never recorded (%s), so the comparison "+
				"above has nothing to compare against", pin.file, pin.record)
		}
		if !strings.Contains(src, pin.report) {
			t.Errorf("%s: a region is reported without being compared (%s) — this host's "+
				"own recentring reaches Go as a user gesture", pin.file, pin.report)
		}
	}
}

// Both hosts must throttle their region stream. A pan generates a region per
// frame; each one that crossed the bridge would be a full Go render pass, and
// the throttle has to be on the host side to save anything — the same
// arrangement the two sensors have, for the same reason.
//
// The interval is the same number in all three hosts (120ms), and it is pinned
// as a number rather than as a name: a host that kept the mechanism and changed
// the figure to two seconds would still pass a name check, and a map that
// reports a pan two seconds after it ends looks broken.
func TestBothNativeMapsThrottleTheirRegionStream(t *testing.T) {
	for _, pin := range []struct{ file, mechanism, interval string }{
		{
			file:      swiftMap,
			mechanism: "reportWork?.cancel()",
			interval:  "GrMobMapRegionQuiet: TimeInterval = 0.12",
		},
		{
			file: kotlinMap,
			// osmdroid has the throttle built in, which is the right answer
			// here: a library's own debounce is one fewer thing to get wrong.
			mechanism: "DelayedMapListener(",
			interval:  "GRMOB_MAP_REGION_QUIET_MS = 120L",
		},
	} {
		src := valuesIn(t, pin.file)
		if !strings.Contains(src, pin.mechanism) {
			t.Errorf("%s: no region throttle (%s) — a drag is one Go render pass per frame",
				pin.file, pin.mechanism)
		}
		if !strings.Contains(src, pin.interval) {
			t.Errorf("%s: the quiet interval is not the one the other hosts use (%s)",
				pin.file, pin.interval)
		}
	}
}

// Markers are reconciled by id, not rebuilt. Both platforms redraw every
// annotation they are handed afresh and drop any open callout, so a list where
// one pin moved would flicker all of them — which is the entire reason
// core.Marker is a keyed child node rather than an entry in a prop array.
func TestBothNativeMapsMoveMarkersRatherThanRebuildingThem(t *testing.T) {
	for _, pin := range []struct{ file, byID, guard string }{
		{
			file: swiftMap,
			byID: "private var markers: [String: GrMobMarkerAnnotation] = [:]",
			// The guarded write: assigning a coordinate is a KVO change MapKit
			// animates, so a pin that has not moved must not be told where it is.
			guard: "if existing.coordinate.latitude != coordinate.latitude",
		},
		{
			file:  kotlinMap,
			byID:  "val markers = HashMap<String, Marker>()",
			guard: "if (marker.position.latitude != point.latitude ||",
		},
	} {
		src := valuesIn(t, pin.file)
		if !strings.Contains(src, pin.byID) {
			t.Errorf("%s: markers are not held by id (%s) — the annotation layer is being "+
				"rebuilt rather than patched", pin.file, pin.byID)
		}
		if !strings.Contains(src, pin.guard) {
			t.Errorf("%s: a marker's position is written unguarded (%s), so a sibling's "+
				"move re-places every pin", pin.file, pin.guard)
		}
	}
}

// A tap that hit a marker is OnMarkerTap's event and must not also be OnMapTap's.
// core.OnMapTap promises that suppression, and each host gets it a different way
// — which is the kind of divergence worth pinning, because the two mechanisms
// look nothing alike and neither one is obviously the other's equivalent.
func TestBothNativeMapsSuppressAMapTapThatHitAMarker(t *testing.T) {
	for _, pin := range []struct{ file, mechanism, why string }{
		{
			file:      swiftMap,
			mechanism: "if current is MKAnnotationView { return false }",
			why: "the tap recognizer refuses a touch that landed on an annotation, " +
				"walked up the view tree because a callout and an image are subviews",
		},
		{
			file:      kotlinMap,
			mechanism: "setOnMarkerClickListener",
			why: "the Marker overlay consumes the tap before the MapEventsOverlay is " +
				"consulted, which is osmdroid's own overlay ordering",
		},
	} {
		if src := valuesIn(t, pin.file); !strings.Contains(src, pin.mechanism) {
			t.Errorf("%s: nothing stops a marker tap from also being a map tap (%s) — %s",
				pin.file, pin.mechanism, pin.why)
		}
	}
}

// The user-location dot is the platform's own and is not followed. core.MapView
// says so in as many words: the dot is information, not a command to go there,
// and an app that wants to follow the user renders a Region from
// hooks.UseLocation — which is a decision it makes rather than one a renderer
// makes for it.
func TestNeitherNativeMapFollowsTheUser(t *testing.T) {
	for _, pin := range []struct{ file, enable, forbidden string }{
		{swiftMap, "map.showsUserLocation = node.boolProp(\"showUser\")", "userTrackingMode"},
		{kotlinMap, "overlay.enableMyLocation()", "enableFollowLocation()"},
	} {
		src := valuesIn(t, pin.file)
		if !strings.Contains(src, pin.enable) {
			t.Errorf("%s: core.ShowUserLocation never reaches the map (%s)", pin.file, pin.enable)
		}
		if strings.Contains(src, pin.forbidden) {
			t.Errorf("%s: the map follows the user (%s) — that is the app's decision, not "+
				"this renderer's", pin.file, pin.forbidden)
		}
	}
}

// Neither host renders the Marker children as views. They are data the node
// above them reads — the same arrangement core.TextGrid has with its rows — and
// a host that rendered them too would draw an empty box per pin on top of its
// own annotations.
func TestNeitherNativeRendererDrawsAMarkerAsAView(t *testing.T) {
	for _, pin := range []struct{ file, arm string }{
		{swiftRenderer, `case "Marker": EmptyView()`},
		{kotlinRenderer, `"Marker" -> Unit`},
	} {
		if src := valuesIn(t, pin.file); !strings.Contains(src, pin.arm) {
			t.Errorf("%s: a Marker reached on its own is not drawn as nothing (%s) — it "+
				"would fall through to the container default and put an empty box in a "+
				"layout", pin.file, pin.arm)
		}
	}
}
