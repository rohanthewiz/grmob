package core

import (
	"reflect"
	"strings"
	"testing"
)

// TestEveryWireFieldOmitsZero walks every struct reachable from core.Node and
// asserts each exported field carries `,omitzero`.
//
// # Why this is a test and not a convention
//
// The rule — a field at its zero value is not written to the wire — was worth
// 370KB and 1,320ms of an Android cold launch when it was applied to
// core.Style, and it was applied there because someone measured a screen and
// found that 92.4% of its bytes were style fields nobody had set. Nothing
// stopped those fields being written for as long as the struct existed, and
// nothing would have stopped the next fifty-eight either.
//
// The failure mode is not a wrong pixel. It is a screen that quietly costs
// more to send than it should, on a device that is not attached, discovered
// by whoever next thinks to marshal a tree and look at the size. core.Style
// carried it for the life of the struct; core.EdgeInsets carried it for two
// sessions longer, through the very session that fixed the parent — because
// the parent's tags made the nested struct's absence invisible.
//
//	Style            omitzero on every field       ← fixed by measurement
//	  Padding        EdgeInsets, itself omitzero   ← so the whole struct
//	    Horizontal   int, no tag                   ← was dropped when empty
//	    ...                                        ← and fully written when not
//
// That is precisely the shape a reader misses: the outer tag is right there,
// and it is doing something, and it hides that the inner fields are not.
//
// # What it does not check
//
// Node.Props is map[string]any, so what a widget puts in it is not a type
// this can reach. A kilobyte in Props is still possible and still nobody's
// error to catch here; examples/tutorial's TestHomeTreeSize is the thing that
// notices, by printing a number.
//
// # The one exception
//
// Node.Type, which is deliberately untagged — see the note on that field. A
// node without a type is not a node, and a renderer reading an absent Type
// would fall through its dispatch to whatever its default arm is rather than
// say what is wrong. The exception is named here so that adding a second one
// is a decision someone writes down rather than a tag they forget.
func TestEveryWireFieldOmitsZero(t *testing.T) {
	// exempt is keyed "Type.Field" and must stay this short. Every entry is a
	// field that is written even when it says nothing.
	exempt := map[string]string{
		"Node.Type": "a node without a type is not a node; see core.Node",
	}

	seen := map[reflect.Type]bool{}
	var walk func(rt reflect.Type, path string)
	walk = func(rt reflect.Type, path string) {
		// Unwrap the containers a wire field can arrive in. A *Style, a
		// []*Node and a map[string]Style all put the same struct on the wire.
		for rt.Kind() == reflect.Pointer || rt.Kind() == reflect.Slice ||
			rt.Kind() == reflect.Map || rt.Kind() == reflect.Array {
			rt = rt.Elem()
		}
		if rt.Kind() != reflect.Struct || seen[rt] {
			return
		}
		seen[rt] = true

		for i := 0; i < rt.NumField(); i++ {
			f := rt.Field(i)
			if !f.IsExported() {
				continue // unexported fields never reach encoding/json
			}
			key := rt.Name() + "." + f.Name
			tag := f.Tag.Get("json")
			switch {
			case tag == "-":
				// Explicitly off the wire; nothing to omit.
			case hasOption(tag, "omitzero"):
				if why, ok := exempt[key]; ok {
					t.Errorf("%s is tagged omitzero but listed as exempt (%q) — "+
						"remove it from the exempt map", key, why)
				}
			default:
				if _, ok := exempt[key]; ok {
					continue
				}
				t.Errorf("%s (%s) crosses the bridge without `json:\",omitzero\"`, so "+
					"it is written out on every node that leaves it alone. Add the "+
					"tag, or add the field to this test's exempt map with the reason "+
					"a renderer needs to see its zero. Reached at %s.",
					key, f.Type, path)
			}
			walk(f.Type, path+"."+f.Name)
		}
	}
	walk(reflect.TypeOf(Node{}), "Node")

	// A guard on the guard: if the walk stopped reaching the tree — a field
	// renamed, a type replaced by an interface — every assertion above passes
	// vacuously and says nothing. These three are the structs the tree is
	// made of today.
	for _, want := range []any{Node{}, Style{}, EdgeInsets{}, ValueRange{}} {
		rt := reflect.TypeOf(want)
		if !seen[rt] {
			t.Errorf("the walk never reached %s — it is reachable from core.Node "+
				"today, so either the tree changed shape or this walk stopped "+
				"following it, and every check above passed without looking at it",
				rt.Name())
		}
	}
}

// hasOption reports whether a struct tag's comma-separated option list carries
// one particular option. `json:",omitzero"` is name-then-options, so the first
// segment is the (here always empty) wire name and is skipped.
func hasOption(tag, want string) bool {
	parts := strings.Split(tag, ",")
	for _, p := range parts[1:] {
		if p == want {
			return true
		}
	}
	return false
}
