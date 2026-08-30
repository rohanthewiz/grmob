// Command gen emits a bridge transcript for the Swift data-layer conformance
// harness (see run.sh). It replays the exact call sequence the iOS shell
// makes — RenderInitial, SetListener, Trigger* per event — against the demo
// app, recording every patch batch in arrival order (synchronous trigger
// returns and asynchronous listener pushes land in one list, exactly as they
// reach the shell's main queue), then snapshots the final full tree. The
// Swift harness mounts the initial tree, applies the batches, and must arrive
// at that same final tree.
//
// This is the iOS analog of examples/mobileapp/app_test.go: it proves the
// Swift TreeStore/parser agree with the Go reconciler without needing Xcode
// or a simulator.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/rohanthewiz/grmob/mobile"

	// Imported for its init: registers the demo app with the bridge, the same
	// wiring the bound framework gets.
	_ "github.com/rohanthewiz/grmob/examples/mobileapp"
)

type transcript struct {
	Initial string   `json:"initial"`
	Steps   []string `json:"steps"`
	Final   string   `json:"final"`
}

// recorder collects patch batches in arrival order. Sync trigger returns are
// appended by the driver and async pushes by Go goroutines; one mutex makes
// the interleaving an explicit, recorded order — mirroring the shell's
// single main-queue funnel.
type recorder struct {
	mu    sync.Mutex
	steps []string
}

func (r *recorder) ApplyPatches(patches string) { r.append(patches) }

func (r *recorder) append(patches string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if patches != "" && patches != "[]" {
		r.steps = append(r.steps, patches)
	}
}

// node mirrors just enough of core.Node's JSON to hunt down callback IDs.
type node struct {
	Type     string
	Props    map[string]any
	Children []*node
}

func findProp(n *node, nodeType, prop string) (string, bool) {
	if n == nil {
		return "", false
	}
	if n.Type == nodeType {
		if v, ok := n.Props[prop].(string); ok {
			return v, true
		}
	}
	for _, c := range n.Children {
		if v, ok := findProp(c, nodeType, prop); ok {
			return v, true
		}
	}
	return "", false
}

func mustFind(tree string, nodeType, prop string) string {
	var n node
	if err := json.Unmarshal([]byte(tree), &n); err != nil {
		fatal("tree is not valid JSON: %v\n%s", err, tree)
	}
	v, ok := findProp(&n, nodeType, prop)
	if !ok {
		fatal("no %s with %s in tree:\n%s", nodeType, prop, tree)
	}
	return v
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

func main() {
	rec := &recorder{}

	// The shell's startup order: mount first, then attach the listener.
	initial := mobile.RenderInitial()
	mobile.SetListener(rec)

	// Void event: tap Increment.
	rec.append(mobile.TriggerCallback(mustFind(initial, "Button", "onClick")))

	// Int event: switch to the Form tab.
	rec.append(mobile.TriggerIntCallback(mustFind(initial, "TabView", "onTabChange"), 1))

	// Callback IDs shift across renders, so re-read the current tree before
	// each dispatch — the shell likewise only ever fires IDs from its live
	// tree. (RenderInitial re-renders the full current tree; it emits no
	// patches, so the recorded step list is unaffected.)
	current := mobile.RenderInitial()

	// Text events: type a name, then revise it.
	onChange := mustFind(current, "Input", "onChange")
	rec.append(mobile.TriggerTextCallback(onChange, "Ada"))
	rec.append(mobile.TriggerTextCallback(mustFind(mobile.RenderInitial(), "Input", "onChange"), "Grace"))

	// Bool event: toggle the subscription checkbox.
	rec.append(mobile.TriggerBoolCallback(mustFind(mobile.RenderInitial(), "Checkbox", "onToggle"), true))

	// Gap-5 surface: over to the Feed tab (index 2), then drive the List rows
	// through their container behavior props — a tap (onClick) selects a row,
	// a long-press (onLongPress) stars one. The first Row carrying each prop
	// is article 1's row; both events restyle rows and rewrite the status
	// line, exercising keyed-children diffs inside a List node.
	rec.append(mobile.TriggerIntCallback(mustFind(mobile.RenderInitial(), "TabView", "onTabChange"), 2))
	rec.append(mobile.TriggerCallback(mustFind(mobile.RenderInitial(), "Row", "onClick")))
	rec.append(mobile.TriggerCallback(mustFind(mobile.RenderInitial(), "Row", "onLongPress")))

	// Int event again: back to the Counter tab, which replaces the subtree.
	rec.append(mobile.TriggerIntCallback(mustFind(mobile.RenderInitial(), "TabView", "onTabChange"), 0))

	// Snapshot the end state the Swift tree must equal. The interval hook's
	// first tick is a full second after registration and this whole replay
	// runs in milliseconds, so the snapshot races nothing (any tick that did
	// sneak in was recorded as a step and is part of the final tree anyway).
	final := mobile.RenderInitial()

	rec.mu.Lock()
	defer rec.mu.Unlock()
	out, err := json.Marshal(transcript{Initial: initial, Steps: rec.steps, Final: final})
	if err != nil {
		fatal("marshal transcript: %v", err)
	}
	os.Stdout.Write(out)
}
