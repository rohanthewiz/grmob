package render

import (
	"fmt"
	"math"
	"reflect"
	"sort"

	"github.com/rohanthewiz/grmob/core"
)

// firstNonFinite finds the first NaN or ±Inf in a tree, which is what makes a
// pass unencodable in practice: encoding/json refuses both, and a float is the
// one ordinary value an app can hand a node that JSON has no spelling for. A
// done/total with total == 0 in a core.Slider value or a FlexGrow is enough.
//
// It answers with the node's path (the same "root/0/2" a patch's TargetID
// uses, so it can be found in a transcript) and the node's type, and with the
// field inside the node: "Style.FlexGrow = NaN", "Props.value = +Inf". Walked
// depth first in child order, so "first" means the first a reader of the tree
// would meet, and map keys in sorted order, so the same tree always names the
// same field.
//
// Only called when a marshal has already failed, so its cost is never paid on
// a pass that encodes.
func firstNonFinite(n *core.Node, path string) (where string, ok bool) {
	if n == nil {
		return "", false
	}
	if n.Style != nil {
		if field, ok := nonFiniteIn(reflect.ValueOf(*n.Style), "Style", 0); ok {
			return fmt.Sprintf("%s (%s) %s", path, n.Type, field), true
		}
	}
	if field, ok := nonFiniteIn(reflect.ValueOf(n.Props), "Props", 0); ok {
		return fmt.Sprintf("%s (%s) %s", path, n.Type, field), true
	}
	for i, c := range n.Children {
		if where, ok := firstNonFinite(c, fmt.Sprintf("%s/%d", path, i)); ok {
			return where, true
		}
	}
	return "", false
}

// maxNonFiniteDepth bounds the walk inside one value. Props are app data and
// can in principle point back at themselves; encoding/json would have failed on
// such a cycle for its own reason, and this walk must still come back.
const maxNonFiniteDepth = 32

// nonFiniteIn reports the field path of the first non-finite float inside v.
func nonFiniteIn(v reflect.Value, at string, depth int) (string, bool) {
	if !v.IsValid() || depth > maxNonFiniteDepth {
		return "", false
	}
	switch v.Kind() {
	case reflect.Float32, reflect.Float64:
		if f := v.Float(); math.IsNaN(f) || math.IsInf(f, 0) {
			return fmt.Sprintf("%s = %v", at, f), true
		}
	case reflect.Pointer, reflect.Interface:
		if v.IsNil() {
			return "", false
		}
		return nonFiniteIn(v.Elem(), at, depth+1)
	case reflect.Struct:
		t := v.Type()
		for i := 0; i < v.NumField(); i++ {
			if !t.Field(i).IsExported() {
				continue // encoding/json never reads it, so it cannot be the cause
			}
			if field, ok := nonFiniteIn(v.Field(i), at+"."+t.Field(i).Name, depth+1); ok {
				return field, true
			}
		}
	case reflect.Map:
		keys := v.MapKeys()
		sort.Slice(keys, func(i, j int) bool { return fmt.Sprint(keys[i]) < fmt.Sprint(keys[j]) })
		for _, k := range keys {
			// A string key reads as a field ("Props.value"); any other key
			// as an index.
			name := fmt.Sprintf("%s[%v]", at, k)
			if k.Kind() == reflect.String {
				name = at + "." + k.String()
			}
			if field, ok := nonFiniteIn(v.MapIndex(k), name, depth+1); ok {
				return field, true
			}
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < v.Len(); i++ {
			if field, ok := nonFiniteIn(v.Index(i), fmt.Sprintf("%s[%d]", at, i), depth+1); ok {
				return field, true
			}
		}
	}
	return "", false
}
