package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"html"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/rohanthewiz/rweb"
)

// Hot reload for the WASM target.
//
// The loop is the ordinary edit-build-refresh cycle with the two manual steps
// taken out, and one property added that a page refresh does not have — the
// app stays where it was:
//
//	editor saves a .go file
//	   │
//	   ▼  (poll, 250 ms)
//	watch ──▶ ./build.sh ──▶ wasm/main.wasm      (the site's own recipe)
//	   │            │
//	   │            └─ compile error ──▶ SSE "buildfail" ──▶ overlay on the page
//	   ▼
//	SSE "reload" ──▶ page: GrMobWASM.Shutdown()   stop the old module
//	                       GrMobHost.boot()       fetch + instantiate the new one
//	                       route + scroll replay  the same lesson, same place
//
// What survives a swap and what does not is the interesting part, and it is
// decided by where state lives. Go-side state — every NewState slot, the
// navigation stack, form values — is heap memory of the module being
// discarded, and there is no way to carry a Go heap across two WebAssembly
// instances. What does survive is state that has a representation *outside*
// the module: the lesson, because the tutorial reports it to the page as a
// route and accepts it back (examples/tutorial/deeplink.go), and the scroll
// offsets, because the client reads them off the DOM before the swap and
// writes them back after. An app that wants more of itself to survive a
// reload has the same tool: report it as a system event, accept it as a host
// event. Positional replay of hook slots is discussed in docs/platforms/wasm.md
// and deliberately not attempted here.
//
// Polling instead of fsnotify keeps the watcher free of dependencies;
// a stat of a few hundred files every quarter second is not measurable. The
// file set is not "every .go under the repo" but the build graph of ./wasm
// as `go list -deps` reports it, so an edit to examples/social does not
// rebuild a tutorial that never imports it, and an import added to a file is
// picked up because the graph is re-read after every build.

//go:embed devclient.js
var devClient []byte

// devServer adds to the static file server the three things hot reload
// needs from the HTTP side: an injected client script, an event stream, and
// no caching.
//
// Every open page holds one SSE connection, and each connection is one
// buffered channel registered with an rweb.SSEHub. The hub is the fan-out
// (BroadcastRaw puts an event on every channel without blocking), the
// keepalive (a comment line every 20 s) and the reaper (a channel that has
// refused MaxDropped consecutive events is closed and forgotten); RWeb's SSE
// sender drains each channel onto its socket.
//
//	watch/build ──broadcast──▶ SSEHub ──▶ chan(page 1) ──▶ rweb sendSSE ──▶ EventSource
//	                                 ├──▶ chan(page 2) ──▶ ...
//	/__dev/events ──subscribe──▶ (SSEHub.Handler registers, then hello is broadcast)
type devServer struct {
	dir  string // the directory being served (wasm/)
	root string // the module root, where build.sh lives
	hub  *rweb.SSEHub

	// mu makes "read the build state, then tell the pages" one step. It is
	// held across every broadcast and across a subscribe, which is what keeps
	// a page from getting a hello that is older than a reload it already
	// has; see subscribe.
	mu       sync.Mutex
	buildID  string // identity of the main.wasm currently on disk
	lastFail string // compiler output of the last failed build, "" if it passed
}

type sseEvent struct {
	name string
	data map[string]any
}

// sseChannelSize is how many events a page may fall behind before the hub
// starts dropping for it. The traffic is a handful of events per save, so a
// page eight behind is a page whose stream is dead.
const sseChannelSize = 8

func newDevServer(dir string) (*devServer, error) {
	out, err := exec.Command("go", "list", "-m", "-f", "{{.Dir}}").Output()
	if err != nil {
		return nil, fmt.Errorf("serve -dev: locating the module root: %w", err)
	}
	root := strings.TrimSpace(string(out))
	if _, err := os.Stat(filepath.Join(root, "build.sh")); err != nil {
		return nil, fmt.Errorf("serve -dev: %s has no build.sh to run", root)
	}
	d := &devServer{dir: dir, root: root, hub: newDevHub()}
	d.buildID = d.stampMainWasm()
	return d, nil
}

// newDevHub builds the hub the pages subscribe to. Every channel comes from
// the hub's own Handler, which registers an on-close cleanup with RWeb, so a
// page that goes away is unregistered the moment its stream ends. MaxDropped
// is therefore only the backstop it was meant to be: a page whose socket is
// alive but not draining (a suspended tab, a wedged proxy) is dropped from
// after the third event it refuses, rather than holding events for pages
// that are still reading.
func newDevHub() *rweb.SSEHub {
	return rweb.NewSSEHub(rweb.SSEHubOptions{
		ChannelSize: sseChannelSize,
		MaxDropped:  3,
		// The comment line is a keepalive: proxies and some browsers drop an
		// idle stream, and EventSource's reconnect would then re-"hello" for
		// no reason.
		HeartbeatInterval: 20 * time.Second,
	})
}

// routes registers the dev server's handlers on s, in front of the plain
// file server. The router prefers a fixed route to the wildcard, so the four
// fixed paths are claimed here and everything else falls through to files.
func (d *devServer) routes(s *rweb.Server, files *staticFiles) {
	// Nothing may be cached in dev: the module changes under the same URL,
	// and a cached main.wasm would hand a hot reload the build it just
	// replaced. Set on everything rather than on main.wasm alone so an edit
	// to the runtime JS or the page is honoured by a plain refresh too. (The
	// event stream overrides it with RWeb's SSE headers, which is harmless:
	// a stream is not cached either way.)
	s.Use(func(ctx rweb.Context) error {
		ctx.Response().SetHeader("Cache-Control", "no-store")
		return ctx.Next()
	})
	s.Get("/__dev/events", d.serveEvents)
	getAndHead(s, "/__dev/client.js", func(ctx rweb.Context) error {
		ctx.Response().SetHeader("Content-Type", "application/javascript")
		return ctx.Bytes(devClient)
	})
	getAndHead(s, "/", d.serveIndex)
	getAndHead(s, "/index.html", d.serveIndex)
	getAndHead(s, "/*path", files.serve)
}

// serveIndex hands out the shipped page with the client script appended, so
// the page itself carries nothing dev-only: the same index.html deploys to
// GitHub Pages untouched. The build identity rides on the script tag so the
// client knows which main.wasm this document booted with, and can tell on
// its first "hello" whether a build slipped in between the page load and the
// stream connecting.
func (d *devServer) serveIndex(ctx rweb.Context) error {
	page, err := os.ReadFile(filepath.Join(d.dir, "index.html"))
	if err != nil {
		return ctx.SetStatus(404).WriteString(err.Error())
	}
	d.mu.Lock()
	id := d.buildID
	d.mu.Unlock()
	tag := fmt.Sprintf(`<script src="/__dev/client.js" data-build="%s"></script>`, html.EscapeString(id))
	if i := bytes.LastIndex(page, []byte("</body>")); i >= 0 {
		page = append(page[:i:i], append([]byte(tag+"\n"), page[i:]...)...)
	} else {
		page = append(page, []byte("\n"+tag)...)
	}
	ctx.Response().SetHeader("Content-Type", "text/html; charset=utf-8")
	return ctx.Bytes(page)
}

// serveEvents is the server-sent event stream every open page subscribes to.
// SSE rather than a WebSocket because the traffic is one-way, EventSource
// reconnects by itself when the server restarts, and the browser side needs
// no library.
//
// The hub's own Handler is what makes the channel, registers it and hands it
// to RWeb, and it is used rather than a hand-rolled SetupSSE for one reason:
// only a channel that came from Handler gets RWeb's on-close callback, which
// unregisters it the instant the page's stream ends. A channel registered by
// hand outlives its page until the keepalive has filled it and three
// broadcasts have been refused — minutes of events fanned out to a socket
// nobody is reading.
func (d *devServer) serveEvents(ctx rweb.Context) error {
	var err error
	d.subscribe(func() { err = d.hub.Handler(ctx.Server())(ctx) })
	return err
}

// subscribe runs join — which must register the new page's channel with the
// hub — and then greets it. The "hello" carries the current build identity
// and the standing compile error, if any, so a page that connects (or
// reconnects) late is brought up to date rather than told only about what
// happens next.
//
// Registering first and greeting second is the opposite of queueing the
// hello into the channel before sharing it, and it is what lets the channel
// come from the hub's Handler (see serveEvents). The consistency the old
// order bought is instead bought by mu, which is held across both steps and
// across every broadcast: a build cannot land between the registration and
// the hello, so the page sees either hello(old) then reload(new), or
// hello(new) alone. The order hello(old)-after-reload(new), which the client
// would read as a newer build to swap *back* to, cannot occur.
//
// The price is that hello is a broadcast — the hub addresses channels, not
// pages, and this one is not ours to name — so every already-open page is
// greeted again whenever a new one connects. That is why the client only
// reacts to a hello that tells it something (see devclient.js): a repeat of
// what a page already has is dropped there rather than filtered here.
func (d *devServer) subscribe(join func()) {
	d.mu.Lock()
	defer d.mu.Unlock()
	join()
	d.hub.BroadcastRaw(rawEvent(sseEvent{"hello", map[string]any{"build": d.buildID, "error": d.lastFail}}))
}

// broadcast fans an event out to every subscribed page. The hub never
// blocks on a page: a full channel drops the event, and a page that keeps
// refusing them is evicted (see newDevHub).
func (d *devServer) broadcast(ev sseEvent) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.hub.BroadcastRaw(rawEvent(ev))
}

// rawEvent renders an event as RWeb writes it on the wire: the name as the
// SSE "event:" field, the data as a pre-encoded JSON string, which RWeb
// prints verbatim with %s. BroadcastRaw rather than Broadcast because the
// client listens by event name (addEventListener("reload", …)); Broadcast
// would wrap everything as one "message" event.
func rawEvent(ev sseEvent) rweb.SSEvent {
	data, _ := json.Marshal(ev.data)
	return rweb.SSEvent{Type: ev.name, Data: string(data)}
}

// --- The watcher -------------------------------------------------------------

// watch runs forever: one build at startup — so `go run ./serve -dev` is
// the whole dev setup, with no separate ./build.sh first — then a rebuild
// for every change to the Go build graph and a plain page reload for every
// change to the host files.
func (d *devServer) watch() {
	d.build()
	goFiles, hostFiles := d.stamps()
	for {
		time.Sleep(250 * time.Millisecond)
		g, h := d.stamps()
		if changed := diffStamps(goFiles, g); len(changed) > 0 {
			// Editors save in bursts (a write, then a gofmt rewrite; a
			// multi-file refactor). Wait for the tree to hold still for a
			// beat so one burst is one build, then take the stamps from
			// *before* the build as the baseline: an edit that lands while
			// the compiler runs shows up as a difference on the next tick
			// and gets its own build, instead of being silently folded into
			// a build that never saw it.
			g = d.settle()
			goFiles = g
			log.Printf("changed: %s", strings.Join(changed, ", "))
			d.build()
			// The build graph may have grown (a new import); re-stamp so the
			// new files are watched from now on, without treating their
			// appearance as a second change.
			goFiles, _ = d.stamps()
		}
		if changed := diffStamps(hostFiles, h); len(changed) > 0 {
			hostFiles = h
			log.Printf("changed: %s (page reload)", strings.Join(changed, ", "))
			d.broadcast(sseEvent{"reload", map[string]any{"kind": "page"}})
		}
	}
}

// settle waits until two consecutive polls of the Go file set agree, and
// returns that set.
func (d *devServer) settle() map[string]string {
	prev, _ := d.stamps()
	for {
		time.Sleep(150 * time.Millisecond)
		cur, _ := d.stamps()
		if len(diffStamps(prev, cur)) == 0 {
			return cur
		}
		prev = cur
	}
}

// build runs the site's own build script and reports the outcome to the
// pages. Using build.sh rather than a private `go build` line keeps one
// recipe: what hot reload shows is byte-for-byte what deploys, including the
// wasm_exec.js refresh.
func (d *devServer) build() {
	d.broadcast(sseEvent{"building", map[string]any{}})
	start := time.Now()
	cmd := exec.Command("sh", "build.sh")
	cmd.Dir = d.root
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		log.Printf("build failed:\n%s", msg)
		d.setStateAndBroadcast(d.buildID, msg, sseEvent{"buildfail", map[string]any{"output": msg}})
		return
	}
	id := d.stampMainWasm()
	log.Printf("built in %s", time.Since(start).Round(time.Millisecond))
	d.setStateAndBroadcast(id, "", sseEvent{"reload", map[string]any{"kind": "wasm", "build": id}})
}

// setStateAndBroadcast records a build's outcome and tells the pages in one
// hold of mu, so no subscribe can slip in between and hand a new page the
// old state after this event (see subscribe).
func (d *devServer) setStateAndBroadcast(buildID, lastFail string, ev sseEvent) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.buildID = buildID
	d.lastFail = lastFail
	d.hub.BroadcastRaw(rawEvent(ev))
}

// stampMainWasm identifies the module on disk by size and modification time —
// enough to tell two builds apart, which is all the client compares.
func (d *devServer) stampMainWasm() string {
	return stamp(filepath.Join(d.dir, "main.wasm"))
}

func stamp(path string) string {
	fi, err := os.Stat(path)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%d-%d", fi.Size(), fi.ModTime().UnixNano())
}

// stamps takes a snapshot of every watched file: the Go build graph of
// ./wasm in the first map, the host page's own files in the second. The two
// are separate because they call for different responses — a rebuild versus
// a page reload — and because build.sh itself rewrites one file in the
// served directory (wasm_exec.js), which must not read as an edit or every
// build would trigger a reload of the page it just hot-swapped.
func (d *devServer) stamps() (goFiles, hostFiles map[string]string) {
	goFiles = map[string]string{}
	for _, f := range d.goSources() {
		goFiles[f] = stamp(f)
	}
	hostFiles = map[string]string{}
	entries, _ := os.ReadDir(d.dir)
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || name == "wasm_exec.js" || name == "main.wasm" {
			continue
		}
		if name == "index.html" || strings.HasSuffix(name, ".js") {
			hostFiles[name] = stamp(filepath.Join(d.dir, name))
		}
	}
	return goFiles, hostFiles
}

// goSources lists the files whose change means "rebuild": the non-test .go
// files of every in-module package in ./wasm's js/wasm build graph, plus the
// module files and the build script. `go list -e` tolerates a package that
// currently fails to parse, so a half-typed edit keeps the set intact rather
// than emptying it.
func (d *devServer) goSources() []string {
	cmd := exec.Command("go", "list", "-e", "-deps", "-f", "{{if not .Standard}}{{.Dir}}{{end}}", "./wasm")
	cmd.Dir = d.root
	cmd.Env = append(os.Environ(), "GOOS=js", "GOARCH=wasm")
	out, err := cmd.Output()
	if err != nil {
		log.Printf("go list: %v", err)
	}
	files := []string{
		filepath.Join(d.root, "go.mod"),
		filepath.Join(d.root, "go.sum"),
		filepath.Join(d.root, "build.sh"),
	}
	for _, dir := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if dir == "" || !strings.HasPrefix(dir, d.root) {
			continue // the module cache is immutable; nothing to watch there
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			name := e.Name()
			if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
				continue
			}
			files = append(files, filepath.Join(dir, name))
		}
	}
	sort.Strings(files)
	return files
}

// diffStamps names the files whose stamp differs between two snapshots —
// edited, added or removed — relative to the module root for the log.
func diffStamps(before, after map[string]string) []string {
	var changed []string
	seen := map[string]bool{}
	for path, s := range after {
		seen[path] = true
		if before[path] != s {
			changed = append(changed, path)
		}
	}
	for path := range before {
		if !seen[path] {
			changed = append(changed, path)
		}
	}
	sort.Strings(changed)
	for i, p := range changed {
		if rel, err := filepath.Rel(mustGetwd(), p); err == nil && !strings.HasPrefix(rel, "..") {
			changed[i] = rel
		}
	}
	return changed
}

func mustGetwd() string {
	wd, _ := os.Getwd()
	return wd
}
