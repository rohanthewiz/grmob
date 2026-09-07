package core

import (
	"reflect"
	"sort"
	"testing"
)

// Every core.ComponentDefaults field must reach a widget.
//
// # The field that did not, and what it cost
//
// ComponentDefaults' own doc says "every widget merges the caller's props over
// the field named for it", and for eight of the nine fields that was true.
// Components.Text was the ninth. core.Text builds its Style from its own props
// and never touches the theme (core/text.go is fifteen lines and takes no
// Context beyond the render pass), so two bundled themes stated a Components.Text
// — a font size, an ink, a white Background, twelve points of padding and a
// border radius, copied from the field base beside it — and described a widget
// that does not exist.
//
// Nothing could notice. The struct compiled, the values marshalled, the census
// in internal/palette dutifully classified the field, and the only person who
// would ever find out was a theme author who filled it in and watched nothing
// happen. That is the worst shape a framework default can have: it is not wrong,
// it is inert, and inert is invisible.
//
// The theme's authority over a run of words is Typography — widgets say
// core.UseStyle(t.Typography.Caption) and mean it — so the field was not a
// missing feature with a hole where its implementation should be. It was a
// second authority for something that already had one.
//
// # Why a witness and not a list
//
// The obvious guard is a list of "fields that are read" kept beside the struct,
// and it fails the same way the field did: a list is a claim nobody executes.
// This renders a real widget through a theme whose base for that one field
// carries a marker, and looks for the marker on the node that comes back. A
// field whose widget stopped merging its base fails here, and so does a field
// added with no widget behind it — which is the case that would have caught
// Components.Text on the day it landed.
//
// The two directions are both stated, which is the coupling: the struct decides
// what must have a witness, and the witness table decides what is checked. A
// field with no entry is a field nothing reads (or a witness somebody forgot to
// write, and the failure says to write one); an entry naming no field is a
// widget whose base has been deleted out from under it.

// componentDefaultWitness is a widget that merges one ComponentDefaults field,
// and the node type it renders as.
//
// A constructor rather than a rendered node, because the base is read out of the
// context's theme at render time — which is the whole thing under test.
type componentDefaultWitness struct {
	// nodeType is what the widget renders as, checked alongside the marker so a
	// witness that started rendering something else is a failure rather than a
	// silent change of subject.
	nodeType string
	build    func() View
}

// One entry per ComponentDefaults field. Several fields have more than one
// reader — Components.Column is merged by core.Column and by core.List,
// Components.Input by five input flavours — and the witness is one of them
// rather than all: the claim here is that the field reaches the framework at
// all, not that every constructor spends it. The second is Style.Merge's
// business and core/style_merge_test.go's.
var componentDefaultWitnesses = map[string]componentDefaultWitness{
	"Button":   {"Button", func() View { return Button("Go", nil) }},
	"Card":     {"Card", func() View { return Card() }},
	"Input":    {"Input", func() View { return Input("", "", nil) }},
	"Column":   {"Column", func() View { return Column() }},
	"Row":      {"Row", func() View { return Row() }},
	"CheckBox": {"Checkbox", func() View { return Checkbox(false, nil) }},
	"TextArea": {"TextArea", func() View { return TextArea("", nil, 3) }},
	// Camera's base is spent by imageNode, which is core.Image's builder as
	// well as the camera preview's — the field is named for the viewfinder and
	// is the fill both share. core.Image is the reachable half of that pair
	// from a test with no device.
	"Camera": {"Image", func() View { return Image("x.png") }},
}

// componentDefaultMarker is a declaration no widget in core sets for itself, so
// finding it on a rendered node can only mean it arrived from the theme.
//
// Animation is that declaration: it is a CSS-shaped string with no default, no
// resolver and no widget that spends it. A marker on Background or BorderRadius
// would be indistinguishable from a widget that happens to state the same
// value, which is a witness that passes without witnessing anything.
func componentDefaultMarker(field string) string { return "witness-" + field + " 1s" }

func TestEveryComponentDefaultReachesAWidget(t *testing.T) {
	fields := map[string]bool{}
	dt := reflect.TypeOf(ComponentDefaults{})
	for i := 0; i < dt.NumField(); i++ {
		fields[dt.Field(i).Name] = true
	}

	// The coupling, both ways, before any rendering: a mismatch here means the
	// loop below would either skip a field or render a widget for one that is
	// gone, and both of those read as a pass.
	for name := range fields {
		if _, ok := componentDefaultWitnesses[name]; !ok {
			t.Errorf("ComponentDefaults.%s has no witness. Either a widget merges it "+
				"and this table is missing the entry, or nothing reads it — in which "+
				"case it is a default a theme author can fill in and watch do nothing, "+
				"which is what Components.Text was. The theme's authority over a run "+
				"of words is Typography; over a widget it is this struct, and a field "+
				"here that no widget merges belongs in neither.", name)
		}
	}
	for name := range componentDefaultWitnesses {
		if !fields[name] {
			t.Errorf("there is a witness for ComponentDefaults.%s and no such field — "+
				"the base has been deleted out from under a widget that was merging it",
				name)
		}
	}

	names := make([]string, 0, len(fields))
	for name := range fields {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		w, ok := componentDefaultWitnesses[name]
		if !ok {
			continue // already reported above
		}
		want := componentDefaultMarker(name)

		// A theme whose ONLY component base is this one, so a marker that
		// arrives could not have come from a neighbour. Built off the zero
		// Theme rather than off DefaultTheme for the same reason: a base
		// somebody else's widget merges is a base this loop would credit to
		// the wrong field.
		theme := &Theme{}
		base := reflect.ValueOf(&theme.Components).Elem().FieldByName(name)
		style := Style{Animation: want}
		base.Set(reflect.ValueOf(style))

		ctx := NewContext().WithTheme(theme)
		ctx.BeginRenderPass()
		node := w.build().Render(ctx)

		if node == nil {
			t.Errorf("ComponentDefaults.%s: its witness rendered nothing", name)
			continue
		}
		if node.Type != w.nodeType {
			t.Errorf("ComponentDefaults.%s: its witness rendered a %q and the table "+
				"says %q — the witness has changed subject, so whatever it is proving "+
				"is not about this field", name, node.Type, w.nodeType)
			continue
		}
		if node.Style == nil {
			t.Errorf("ComponentDefaults.%s: its witness rendered a node with no Style "+
				"at all, so the theme's base for it reaches nothing", name)
			continue
		}
		if node.Style.Animation != want {
			t.Errorf("ComponentDefaults.%s: the theme's base for it carries %q and the "+
				"rendered node carries %q. The widget named for this field does not "+
				"merge it, so a theme filling it in changes nothing on screen.",
				name, want, node.Style.Animation)
		}
	}
}
