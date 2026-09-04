package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"html"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
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
// Polling instead of fsnotify keeps the module at zero non-Go dependencies;
// a stat of a few hundred files every quarter second is not measurable. The
// file set is not "every .go under the repo" but the build graph of ./wasm
// as `go list -deps` reports it, so an edit to examples/social does not
// rebuild a tutorial that never imports it, and an import added to a file is
// picked up because the graph is re-read after every build.

//go:embed devclient.js
var devClient []byte

// devServer wraps the static file server with the three things hot reload
// needs from the HTTP side: an injected client script, an event stream, and
// no caching.
type devServer struct {
	dir  string       // the directory being served (wasm/)
	root string       // the module root, where build.sh lives
	next http.Handler // the plain file server

	mu       sync.Mutex
	subs     map[chan sseEvent]struct{}
	buildID  string // identity of the main.wasm currently on disk
	lastFail string // compiler output of the last failed build, "" if it passed
}

type sseEvent struct {
	name string
	data map[string]any
}

func newDevServer(dir string, next http.Handler) (*devServer, error) {
	out, err := exec.Command("go", "list", "-m", "-f", "{{.Dir}}").Output()
	if err != nil {
		return nil, fmt.Errorf("serve -dev: locating the module root: %w", err)
	}
	root := strings.TrimSpace(string(out))
	if _, err := os.Stat(filepath.Join(root, "build.sh")); err != nil {
		return nil, fmt.Errorf("serve -dev: %s has no build.sh to run", root)
	}
	d := &devServer{dir: dir, root: root, next: next, subs: map[chan sseEvent]struct{}{}}
	d.buildID = d.stampMainWasm()
	return d, nil
}

func (d *devServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Nothing may be cached in dev: the module changes under the same URL,
	// and a cached main.wasm would hand a hot reload the build it just
	// replaced. Set on everything rather than on main.wasm alone so an edit
	// to the runtime JS or the page is honoured by a plain refresh too.
	w.Header().Set("Cache-Control", "no-store")
	switch r.URL.Path {
	case "/__dev/events":
		d.serveEvents(w, r)
	case "/__dev/client.js":
		w.Header().Set("Content-Type", "application/javascript")
		w.Write(devClient)
	case "/", "/index.html":
		d.serveIndex(w, r)
	default:
		d.next.ServeHTTP(w, r)
	}
}

// serveIndex hands out the shipped page with the client script appended, so
// the page itself carries nothing dev-only: the same index.html deploys to
// GitHub Pages untouched. The build identity rides on the script tag so the
// client knows which main.wasm this document booted with, and can tell on
// its first "hello" whether a build slipped in between the page load and the
// stream connecting.
func (d *devServer) serveIndex(w http.ResponseWriter, r *http.Request) {
	page, err := os.ReadFile(filepath.Join(d.dir, "index.html"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
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
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(page)
}

// serveEvents is the server-sent event stream every open page subscribes to.
// SSE rather than a WebSocket because the traffic is one-way, EventSource
// reconnects by itself when the server restarts, and it needs no library on
// either side. The opening "hello" carries the current build identity and
// the standing compile error, if any, so a page that connects (or reconnects)
// late is brought up to date rather than told only about what happens next.
func (d *devServer) serveEvents(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Connection", "keep-alive")

	ch := make(chan sseEvent, 8)
	d.mu.Lock()
	d.subs[ch] = struct{}{}
	hello := sseEvent{"hello", map[string]any{"build": d.buildID, "error": d.lastFail}}
	d.mu.Unlock()
	defer func() {
		d.mu.Lock()
		delete(d.subs, ch)
		d.mu.Unlock()
	}()

	write := func(ev sseEvent) {
		data, _ := json.Marshal(ev.data)
		fmt.Fprintf(w, "event: %s\ndata: %s\n\n", ev.name, data)
		flusher.Flush()
	}
	write(hello)
	// The comment line is a keepalive: proxies and some browsers drop an
	// idle stream, and EventSource's reconnect would then re-"hello" for no
	// reason.
	keepalive := time.NewTicker(20 * time.Second)
	defer keepalive.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case ev := <-ch:
			write(ev)
		case <-keepalive.C:
			fmt.Fprint(w, ": ping\n\n")
			flusher.Flush()
		}
	}
}

// broadcast fans an event out to every subscribed page. A page that has
// fallen eight events behind is a page whose stream is dead; its event is
// dropped rather than letting it stall the watcher.
func (d *devServer) broadcast(ev sseEvent) {
	d.mu.Lock()
	defer d.mu.Unlock()
	for ch := range d.subs {
		select {
		case ch <- ev:
		default:
		}
	}
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
		d.mu.Lock()
		d.lastFail = msg
		d.mu.Unlock()
		d.broadcast(sseEvent{"buildfail", map[string]any{"output": msg}})
		return
	}
	id := d.stampMainWasm()
	d.mu.Lock()
	d.lastFail = ""
	d.buildID = id
	d.mu.Unlock()
	log.Printf("built in %s", time.Since(start).Round(time.Millisecond))
	d.broadcast(sseEvent{"reload", map[string]any{"kind": "wasm", "build": id}})
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
