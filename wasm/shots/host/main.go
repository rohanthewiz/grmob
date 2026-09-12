//go:build js && wasm

// Command host is the browser host the screenshots in docs/images are taken
// through. It is wasm/main.go's wiring with one difference: which app it
// mounts is a query parameter rather than a dot-import.
//
// # Why that difference is the whole design
//
// wasm/main.go names its app in an import:
//
//	. "github.com/rohanthewiz/grmob/examples/tutorial"
//
// which is the right shape for a deployed site that shows one app, and the
// wrong one for a camera. The first version of this harness lived outside
// the repository and rewrote that line with sed before each build — which
// works, and costs a build per shot, an edit to a tracked file per shot,
// and a rule about what happens if the process dies between the edit and
// the restore.
//
// A registry costs a map. Every app that has a screenshot is imported here
// and selected at run time, so one build serves every shot and no tracked
// file is ever rewritten. The cost is a bigger module — five apps rather
// than one — which is a cost paid by a local harness and by nobody else.
//
// # Why it is not wasm/main.go with a parameter
//
// Because wasm/main.go is what ./build.sh ships. Linking five example apps
// into the module a visitor downloads, so that a harness nobody runs in
// production can choose between them, is a real cost (the tutorial alone
// is ~7MB) charged to the wrong person. The site mounts one app; this
// mounts any of them; they are different programs and this is the smaller
// one.
//
//	./build.sh            wasm/main.go       → one app, shipped
//	wasm/shots/shoot.sh   wasm/shots/host    → any app, local only
package main

import (
	"encoding/json"
	"log"
	"net/url"
	"sort"
	"strings"
	"sync"
	"syscall/js"

	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/examples/counter"
	"github.com/rohanthewiz/grmob/examples/mobileapp"
	"github.com/rohanthewiz/grmob/examples/signup"
	"github.com/rohanthewiz/grmob/examples/todoapp"
	"github.com/rohanthewiz/grmob/examples/tutorial"
	"github.com/rohanthewiz/grmob/render"
)

// apps is every app a screenshot is taken of.
//
// The keys are the directory names under examples/, so that a claim in
// internal/shotclaims naming "examples/todoapp" and the `?app=` this host
// is asked for are the same word with a prefix removed. Anything else
// would be a third name for the same thing.
var apps = map[string]func(*core.Context) core.View{
	"counter":   counter.App,
	"mobileapp": mobileapp.App,
	"signup":    signup.App,
	"todoapp":   todoapp.App,
	"tutorial":  tutorial.App,
}

var (
	ctx     = core.NewContext().WithTheme(core.DefaultTheme)
	manager *render.Manager

	// done parks main until the page calls Shutdown. See shutdown below.
	done     = make(chan struct{})
	doneOnce sync.Once
)

// mounted is the app named by the page's query string.
//
// Read from location rather than handed in by the page, because the page
// is one static file serving every shot: giving it the app would mean
// templating it, and reading the URL is what the page would have had to do
// anyway to pass it on.
func mounted() (string, func(*core.Context) core.View, error) {
	search := js.Global().Get("location").Get("search").String()
	q, err := url.ParseQuery(strings.TrimPrefix(search, "?"))
	if err != nil {
		return "", nil, err
	}
	name := q.Get("app")
	if name == "" {
		name = "tutorial"
	}
	app, ok := apps[name]
	if !ok {
		return name, nil, errUnknownApp(name)
	}
	return name, app, nil
}

// errUnknownApp names what IS available, because the whole failure is a
// typo and the answer to it is the list.
func errUnknownApp(name string) error {
	known := make([]string, 0, len(apps))
	for k := range apps {
		known = append(known, k)
	}
	sort.Strings(known)
	return &appError{name: name, known: known}
}

type appError struct {
	name  string
	known []string
}

func (e *appError) Error() string {
	return "no app named " + e.name + " is mounted by this host; it knows " +
		strings.Join(e.known, ", ") +
		". Add it to `apps` in wasm/shots/host/main.go, with its import."
}

func renderInitial(this js.Value, args []js.Value) any {
	if manager != nil {
		manager.Close()
	}
	name, app, err := mounted()
	if err != nil {
		// Returned as a string the page shows rather than panicking: a
		// harness that dies here leaves the driver waiting for a ready
		// flag that will never be set, and "the page never became ready"
		// is the least useful sentence this failure could produce.
		log.Printf("grmob/shots: %v", err)
		return js.ValueOf(`{"Type":"Text","Props":{"content":` +
			mustJSON(err.Error()) + `}}`)
	}
	log.Printf("grmob/shots: mounting %s", name)
	manager = render.New(ctx, app)
	if js.Global().Get("GrMobApplyPatches").Type() == js.TypeFunction {
		manager.SetListener(jsPatchListener{})
	}
	return js.ValueOf(manager.RenderInitial())
}

// mustJSON quotes a string for embedding in the fallback tree above. The
// error text is this program's own, so the only way to fail is a bug here.
func mustJSON(s string) string {
	b, err := json.Marshal(s)
	if err != nil {
		return `"the host could not report its own error"`
	}
	return string(b)
}

type jsPatchListener struct{}

func (jsPatchListener) ApplyPatches(patches string) {
	js.Global().Call("GrMobApplyPatches", patches)
}

func renderAgain(this js.Value, args []js.Value) any {
	return js.ValueOf(manager.RenderAgain())
}

func isDirty(this js.Value, args []js.Value) any {
	return js.ValueOf(ctx.IsDirty())
}

// receiveEvent routes a DOM event back into Go. Guarded for the reason
// wasm/main.go's is: a panic escaping a handler would unwind into the
// js.Func callback and abort the Go runtime, which in this program means
// the camera photographs a dead page instead of failing.
func receiveEvent(this js.Value, args []js.Value) any {
	id := args[0].String()
	var payload map[string]any
	if err := json.Unmarshal([]byte(args[1].String()), &payload); err != nil {
		log.Printf("grmob/shots: dropping malformed payload for %s: %v", id, err)
		return nil
	}
	if rerr := core.Guard(func() {
		ctx.ReceiveEventPayload(map[string]any{
			"callback": id, "value": payload["value"],
		})
	}); rerr != nil {
		log.Printf("grmob/shots: recovered panic in handler %s: %v\n%s",
			id, rerr.Value, rerr.Stack)
	}
	return nil
}

// hostEvent is the channel an app listens on for things no callback
// answers — the tutorial's deep-link route, the audio player's status
// ticks. Present so that an app which needs one is photographable at all;
// no shot uses it today.
func hostEvent(this js.Value, args []js.Value) any {
	if len(args) < 2 {
		return nil
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(args[1].String()), &payload); err != nil {
		log.Printf("grmob/shots: dropping malformed host event: %v", err)
		return nil
	}
	if rerr := core.Guard(func() {
		core.ReceiveHostEvent(args[0].String(), payload)
	}); rerr != nil {
		log.Printf("grmob/shots: recovered panic in host event: %v", rerr.Value)
	}
	return nil
}

// shutdown is the hot-reload hook, the same two steps in the same order as
// wasm/main.go's (which carries the long version): close the manager so no
// hook-owned timer keeps pushing patches into a tree the next module replaces,
// then release main so the instance's memory is dropped and go.run's promise
// settles.
//
// The screenshot driver never calls it — it closes the browser — so this is
// here for the other way the host can be loaded: under `serve -dev`, whose
// client calls Shutdown before booting a new build and falls back to a full
// page reload when there is none. A reload loses the state being set up for a
// shot. It also means the host installs exactly webhost.Bindings, so the
// three browser hosts in this repository speak one page contract with no
// exceptions to remember (webhost_test.go).
func shutdown(this js.Value, args []js.Value) any {
	if manager != nil {
		manager.Close()
		manager = nil
	}
	doneOnce.Do(func() { close(done) })
	return nil
}

func main() {
	js.Global().Set("GrMobWASM", map[string]any{
		"RenderInitial": js.FuncOf(renderInitial),
		"RenderAgain":   js.FuncOf(renderAgain),
		"ReceiveEvent":  js.FuncOf(receiveEvent),
		"IsDirty":       js.FuncOf(isDirty),
		"HostEvent":     js.FuncOf(hostEvent),
		"Shutdown":      js.FuncOf(shutdown),
	})
	if js.Global().Get("GrMobSystemEvent").Type() == js.TypeFunction {
		core.SetSystemEventHandler(func(name string, data map[string]any) {
			payload, err := json.Marshal(data)
			if err != nil {
				return
			}
			js.Global().Call("GrMobSystemEvent", name, string(payload))
		})
	}
	println("GrMob shots host ready.")
	// Parked until Shutdown. The screenshot driver closes the browser rather
	// than calling it, so under shoot.sh this waits forever, as it always did.
	<-done
	println("GrMob shots host stopped.")
}
