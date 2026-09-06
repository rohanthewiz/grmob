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
//
// The file carries one thing besides the transcript: a table of picker option
// lists with the menu core.SelectMenuSections says each decomposes into (see
// menuCase). It rides along here because the harness is a single executable,
// and it is generated rather than hand-written for the reason that whole
// mechanism exists — a fixture written out twice proves only that one person
// made the same mistake twice.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/mobile"

	// Imported for its init: registers the demo app with the bridge, the same
	// wiring the bound framework gets.
	_ "github.com/rohanthewiz/grmob/examples/mobileapp"
)

type transcript struct {
	Initial string   `json:"initial"`
	Steps   []string `json:"steps"`
	Final   string   `json:"final"`

	// The picker-menu cases, which have nothing to do with the replay above
	// and ride along in the same file because the harness is one executable.
	// See menuCases.
	MenuCases []menuCase `json:"menuCases"`
}

// menuCase is one option list and the menu Go says it decomposes into.
//
// A SwiftUI Menu's content is a closure of views and cannot be read back, so
// nothing short of a simulator can open a picker and check its Sections. What
// *can* be checked is the decision the view is a direct drawing of:
// GrMobSelectMenu.swift turns the flat option list into sections and rows, and
// Renderer.swift's Menu is one ForEach over the result. These cases put Go's
// answer and Swift's side by side.
//
// The expected value is computed by core.SelectMenuSections rather than
// written out here. That is the point of the file existing: htmlout calls that
// function, the Swift and Kotlin renderers each carry a transliteration of it,
// and a fixture written twice by hand would only prove that one person made
// the same mistake twice.
type menuCase struct {
	Name    string              `json:"name"`
	Options []map[string]string `json:"options"`
	Want    []wantSection       `json:"want"`
}

// wantSection and wantItem mirror core.SelectMenuSection / SelectMenuItem with
// JSON names the Swift side decodes without a CodingKeys block. First is
// carried explicitly, so the Swift section's own `first` computed property is
// compared rather than assumed.
type wantSection struct {
	Heading string     `json:"heading"`
	First   int        `json:"first"`
	Items   []wantItem `json:"items"`
}

type wantItem struct {
	Index    int    `json:"index"`
	Value    string `json:"value"`
	Label    string `json:"label"`
	Disabled bool   `json:"disabled"`
}

// menuCases is the table both sides are held to.
//
// Every case is one property of core.SelectOption.Group or .Disabled, and the
// two that matter most are the ones a transliteration gets wrong: a heading
// that reappears after a different one (runs, not a gather) and a run that
// ends the list (which nothing follows, so it needs an explicit flush — the
// bug htmlout shipped for a release).
func menuCases() []menuCase {
	lists := []struct {
		name    string
		options []map[string]string
	}{
		{"ungrouped", []map[string]string{
			{"value": "s", "label": "Small"},
			{"value": "m", "label": "Medium"},
		}},
		{"one run", []map[string]string{
			{"value": "pt", "label": "Portugal", "group": "Europe"},
			{"value": "fr", "label": "France", "group": "Europe"},
		}},
		{"a heading that comes back is a second run", []map[string]string{
			{"value": "pt", "label": "Portugal", "group": "Europe"},
			{"value": "us", "label": "United States", "group": "Americas"},
			{"value": "fr", "label": "France", "group": "Europe"},
		}},
		{"a run that ends the list", []map[string]string{
			{"value": "any", "label": "Pick one"},
			{"value": "pt", "label": "Portugal", "group": "Europe"},
			{"value": "fr", "label": "France", "group": "Europe"},
		}},
		{"a run that starts the list", []map[string]string{
			{"value": "pt", "label": "Portugal", "group": "Europe"},
			{"value": "any", "label": "Elsewhere"},
		}},
		{"disabled options", []map[string]string{
			{"value": "a", "label": "Aisle", "disabled": "true"},
			{"value": "b", "label": "Aisle"},
			{"value": "c", "label": "Window", "disabled": "false"},
		}},
		{"a disabled option inside a run", []map[string]string{
			{"value": "1", "label": "Row 1", "group": "Exit row", "disabled": "true"},
			{"value": "2", "label": "Row 2", "group": "Main cabin"},
		}},
		// The label default lives at core.Select's flattening seam; a
		// hand-assembled node can carry a map that never went through it.
		{"a label falling back to the value", []map[string]string{
			{"value": "42"},
		}},
		{"no options at all", []map[string]string{}},
	}

	cases := make([]menuCase, 0, len(lists))
	for _, l := range lists {
		// Both slices are allocated empty rather than left nil: a nil slice
		// marshals to JSON `null`, and Swift's Decodable refuses a null where
		// a non-optional array is declared — so the empty-list case would
		// bring the whole harness down on a usage message rather than
		// reporting a menu of no sections.
		c := menuCase{Name: l.name, Options: l.options, Want: []wantSection{}}
		for _, section := range core.SelectMenuSections(l.options) {
			w := wantSection{
				Heading: section.Heading,
				First:   section.First(),
				Items:   make([]wantItem, 0, len(section.Items)),
			}
			for _, item := range section.Items {
				w.Items = append(w.Items, wantItem{
					Index:    item.Index,
					Value:    item.Value,
					Label:    item.Label,
					Disabled: item.Disabled,
				})
			}
			c.Want = append(c.Want, w)
		}
		cases = append(cases, c)
	}
	return cases
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
	out, err := json.Marshal(transcript{
		Initial: initial, Steps: rec.steps, Final: final,
		MenuCases: menuCases(),
	})
	if err != nil {
		fatal("marshal transcript: %v", err)
	}
	os.Stdout.Write(out)
}
