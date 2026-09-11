// A fake Leaflet, for the map reconciliation the runtime does on top of it.
//
// # Why a fake and not the real thing
//
// The real Leaflet needs a layout engine. It measures its container to decide
// which tiles to fetch, positions everything with transforms, and reads
// getBoundingClientRect in several of its hot paths — none of which the harness
// DOM has (see dom.mjs: no layout, no boxes, no CSS). Loading it here would
// either crash or, worse, half-work in a way that made the assertions
// meaningless.
//
// What the runtime's map code actually contains, though, is not rendering. It is
// bookkeeping: when to call setView and when not to (the echo guard), which
// marker to move and which to remove, which id to report at fire time, whether
// a gesture the runtime itself caused gets reported back to Go. Every one of
// those is a decision about calls, and a fake that records calls is the exact
// instrument for it — the same move ios/verify makes with its fake
// LayoutSubview, for the same reason: the decisions are testable off-device and
// the rendering is not.
//
// What this cannot check is anything about the real library's behaviour — that
// `moveend` fires once per gesture, that `circleMarker` draws a circle, that
// unbindPopup is the opposite of bindPopup. Those are Leaflet's promises, and
// they belong in a browser.
//
// # The contract it models
//
// Only the surface grmob-runtime.js uses, which is deliberately small: one map,
// one tile layer, markers, two circle shapes, the locate pair, and `on`. Any
// call the runtime makes that this does not define is a TypeError rather than a
// silent no-op, which is how a widened runtime surface announces itself here.

// newLeaflet returns { L, maps } — the window.L a test installs, and the list of
// every map it has created, in order, for the assertions to read.
export function newLeaflet() {
    const maps = [];

    function handlers() {
        const byType = new Map();
        return {
            on(type, fn) {
                // Leaflet accepts space-separated types; the runtime registers
                // one at a time, and accepting both keeps this honest about
                // which shape is being relied on.
                for (const t of String(type).split(/\s+/)) {
                    if (!byType.has(t)) byType.set(t, []);
                    byType.get(t).push(fn);
                }
            },
            fire(type, event) {
                const list = byType.get(type) || [];
                for (const fn of [...list]) fn(event);
                return list.length;
            },
            count: (type) => (byType.get(type) || []).length,
        };
    }

    function newMarker(latlng, kind) {
        const h = handlers();
        return {
            kind,
            latlng: { lat: latlng[0], lng: latlng[1] },
            radius: 0,
            popup: null,
            // Every call the runtime makes, counted: a test asserting "this
            // marker did not move" is asserting that setLatLng was not called.
            calls: { setLatLng: 0, setRadius: 0, bindPopup: 0, unbindPopup: 0 },
            map: null,
            addTo(map) {
                this.map = map;
                map.layers.push(this);
                return this;
            },
            getLatLng() { return this.latlng; },
            // Leaflet's own L.latLng accepts an array or an object, and the
            // runtime uses both — an array for a marker it is moving to a
            // node's coordinates, the LatLng off a location event for the user
            // dot. Accepting both here is modelling the library rather than
            // being lenient: narrowing it would fail a caller the real one
            // serves.
            setLatLng(next) {
                this.calls.setLatLng++;
                this.latlng = Array.isArray(next)
                    ? { lat: next[0], lng: next[1] }
                    : { lat: next.lat, lng: next.lng };
                return this;
            },
            setRadius(r) { this.calls.setRadius++; this.radius = r; return this; },
            bindPopup(text) { this.calls.bindPopup++; this.popup = text; return this; },
            unbindPopup() { this.calls.unbindPopup++; this.popup = null; return this; },
            on: h.on,
            fire: h.fire,
            listenerCount: h.count,
        };
    }

    const L = {
        map(el, opts) {
            const h = handlers();
            const map = {
                el,
                opts,
                center: { lat: opts.center[0], lng: opts.center[1] },
                zoom: opts.zoom,
                layers: [],
                // setView calls, with their arguments: the echo guard's whole
                // observable effect is whether this list grows.
                views: [],
                locates: [],
                stops: 0,
                setView(center, zoom, options) {
                    this.views.push({ lat: center[0], lng: center[1], zoom, options });
                    this.center = { lat: center[0], lng: center[1] };
                    this.zoom = zoom;
                    return this;
                },
                getCenter() { return this.center; },
                getZoom() { return this.zoom; },
                removeLayer(layer) {
                    const i = this.layers.indexOf(layer);
                    if (i >= 0) this.layers.splice(i, 1);
                    layer.map = null;
                    return this;
                },
                locate(options) { this.locates.push(options); return this; },
                stopLocate() { this.stops++; return this; },
                on: h.on,
                fire: h.fire,
                listenerCount: h.count,
                // The markers this map holds, in creation order, ignoring the
                // tile layer and the user-location circles — which is what a
                // test asking "how many pins" means.
                markers() { return this.layers.filter((l) => l.kind === "marker"); },
                circles() { return this.layers.filter((l) => l.kind !== "marker" && l.kind !== "tiles"); },
            };
            maps.push(map);
            return map;
        },
        tileLayer(url, opts) {
            return {
                kind: "tiles", url, opts, map: null,
                addTo(map) { this.map = map; map.layers.push(this); return this; },
            };
        },
        marker(latlng) { return newMarker(latlng, "marker"); },
        circleMarker(latlng) { return newMarker([latlng.lat, latlng.lng], "circleMarker"); },
        circle(latlng, opts) {
            const m = newMarker([latlng.lat, latlng.lng], "circle");
            m.radius = opts && opts.radius ? opts.radius : 0;
            return m;
        },
    };

    return { L, maps };
}
