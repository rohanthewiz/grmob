package core

import (
	"reflect"
	"testing"
)

// sampleValue fills a Style field with a distinctive non-zero value of the
// right type. It is driven by reflect.Kind rather than a per-field table so
// that a field added to Style is covered automatically — which is the whole
// point of the tests below.
func sampleValue(f reflect.Value, ft reflect.StructField) reflect.Value {
	switch f.Kind() {
	case reflect.Float64:
		return reflect.ValueOf(7.5).Convert(f.Type())
	case reflect.Int:
		return reflect.ValueOf(11).Convert(f.Type())
	case reflect.String:
		// Converted, not asserted: most of these are named string types
		// (Alignment, DisplayMode, JustifyContent, ...).
		return reflect.ValueOf("sentinel").Convert(f.Type())
	case reflect.Bool:
		return reflect.ValueOf(true)
	case reflect.Struct:
		// EdgeInsets is the only struct field; fill one edge so the value is
		// distinguishable from the zero EdgeInsets the merge tests against.
		v := reflect.New(f.Type()).Elem()
		v.Field(0).Set(reflect.ValueOf(3).Convert(v.Field(0).Type()))
		return v
	case reflect.Pointer:
		// *Style — a nested style carrying one recognizable field.
		return reflect.ValueOf(&Style{FontSize: 3})
	case reflect.Map:
		// map[string]Style — one recognizable entry.
		return reflect.ValueOf(map[string]Style{":hover": {FontSize: 3}})
	}
	panic("no sample value for Style field " + ft.Name + " of kind " + f.Kind().String())
}

// TestUseStyleMergesEveryField is the regression guard for the bug this file
// exists to prevent: UseStyle used to copy only fourteen of Style's fields, so
// Width, Height, Top, the entire flex group and the accessibility fields were
// silently dropped — the call compiled, applied, and did nothing.
//
// Rather than listing fields by hand (the list is exactly what rots), it walks
// Style reflectively: for each field it builds a Style with only that field
// set, applies it to an empty target, and asserts the field arrived. Add a
// field to Style without adding it to applyTo and this fails, naming it.
func TestUseStyleMergesEveryField(t *testing.T) {
	st := reflect.TypeOf(Style{})
	for i := 0; i < st.NumField(); i++ {
		ft := st.Field(i)
		t.Run(ft.Name, func(t *testing.T) {
			src := reflect.New(st).Elem()
			want := sampleValue(src.Field(i), ft)
			src.Field(i).Set(want)

			var target Style
			UseStyle(src.Interface().(Style)).Apply(&target)
			got := reflect.ValueOf(target).Field(i)

			switch ft.Type.Kind() {
			case reflect.Pointer:
				// Compared by value, not identity: the merge deliberately
				// allocates a fresh Style so the result is not aliased to the
				// source (see the aliasing note in applyTo).
				if got.IsNil() || !reflect.DeepEqual(got.Interface(), want.Interface()) {
					t.Fatalf("field %s: dropped by UseStyle (got %+v)", ft.Name, got)
				}
				if got.Pointer() == want.Pointer() {
					t.Fatalf("field %s: merged result aliases the source style", ft.Name)
				}
			case reflect.Map:
				if !reflect.DeepEqual(got.Interface(), want.Interface()) {
					t.Fatalf("field %s: dropped by UseStyle (got %+v)", ft.Name, got)
				}
				if got.Pointer() == want.Pointer() {
					t.Fatalf("field %s: merged result aliases the source map", ft.Name)
				}
			default:
				if !reflect.DeepEqual(got.Interface(), want.Interface()) {
					t.Fatalf("field %s: dropped by UseStyle (got %#v, want %#v)",
						ft.Name, got.Interface(), want.Interface())
				}
			}
		})
	}
}

// TestUseStyleZeroFieldsDoNotClobber is the other half of the merge contract:
// widening UseStyle must not turn it into a wholesale overwrite. An empty
// Style layered onto a fully populated target has to change nothing, or every
// role style would blank out the theme defaults underneath it.
func TestUseStyleZeroFieldsDoNotClobber(t *testing.T) {
	st := reflect.TypeOf(Style{})
	target := reflect.New(st).Elem()
	for i := 0; i < st.NumField(); i++ {
		target.Field(i).Set(sampleValue(target.Field(i), st.Field(i)))
	}
	populated := target.Interface().(Style)

	got := populated
	UseStyle(Style{}).Apply(&got)

	if !reflect.DeepEqual(got, populated) {
		t.Errorf("empty style clobbered a populated target:\n got %+v\nwant %+v", got, populated)
	}
}

// TestUseStyleDoesNotMutateSharedTheme covers the aliasing hazard that comes
// with merging the reference-typed fields. containerNode starts every node
// from `style := &base` where base is a *copy* of the theme's component Style
// — a shallow copy, so it shares the theme's HoverStyle pointer and
// PseudoStates map. If the merge wrote through either, one render would
// permanently edit the theme for every render after it.
func TestUseStyleDoesNotMutateSharedTheme(t *testing.T) {
	theme := Style{
		HoverStyle:   &Style{Background: "#theme"},
		PseudoStates: map[string]Style{":focus": {Background: "#theme"}},
	}

	// The shallow copy a container makes of its theme default.
	node := theme
	UseStyle(Style{
		HoverStyle:   &Style{Background: "#local"},
		PseudoStates: map[string]Style{":focus": {Background: "#local"}},
	}).Apply(&node)

	if theme.HoverStyle.Background != "#theme" {
		t.Errorf("merge wrote through the shared HoverStyle pointer: theme now %q",
			theme.HoverStyle.Background)
	}
	if theme.PseudoStates[":focus"].Background != "#theme" {
		t.Errorf("merge wrote into the shared PseudoStates map: theme now %q",
			theme.PseudoStates[":focus"].Background)
	}
	if node.HoverStyle.Background != "#local" {
		t.Errorf("node HoverStyle = %q, want the merged local value", node.HoverStyle.Background)
	}
	if node.PseudoStates[":focus"].Background != "#local" {
		t.Errorf("node :focus = %q, want the merged local value",
			node.PseudoStates[":focus"].Background)
	}
}

// TestUseStyleMergesNestedStyles pins the recursive semantics: a nested style
// is merged field by field like the top level, not swapped out wholesale, and
// pseudo-state entries the source does not mention survive.
func TestUseStyleMergesNestedStyles(t *testing.T) {
	target := Style{
		HoverStyle: &Style{Background: "#base", FontSize: 12},
		PseudoStates: map[string]Style{
			":focus": {Background: "#focus"},
			":hover": {Background: "#hover", FontSize: 12},
		},
	}
	UseStyle(Style{
		HoverStyle:   &Style{Background: "#over"},
		PseudoStates: map[string]Style{":hover": {Background: "#over"}},
	}).Apply(&target)

	if target.HoverStyle.Background != "#over" || target.HoverStyle.FontSize != 12 {
		t.Errorf("HoverStyle = %+v, want Background overridden and FontSize kept", *target.HoverStyle)
	}
	if got := target.PseudoStates[":hover"]; got.Background != "#over" || got.FontSize != 12 {
		t.Errorf(":hover = %+v, want Background overridden and FontSize kept", got)
	}
	if got := target.PseudoStates[":focus"].Background; got != "#focus" {
		t.Errorf(":focus = %q, want the untouched entry to survive", got)
	}
}

// TestUseStyleAppliesToWidgetSurfaces is the end-to-end shape of the bug as it
// was actually reported: a widget passing a Style value that carries sizing,
// flex and accessibility fields onto a real node. Before the fix the node came
// back with all four of these empty.
func TestUseStyleAppliesToWidgetSurfaces(t *testing.T) {
	ctx := NewContext()
	n := Row(UseStyle(Style{
		Width:              "44px",
		Height:             "1px",
		FlexGrow:           1,
		AccessibilityLabel: "separator",
	})).Render(ctx)

	if n.Style.Width != "44px" || n.Style.Height != "1px" {
		t.Errorf("sizing dropped: Width=%q Height=%q", n.Style.Width, n.Style.Height)
	}
	if n.Style.FlexGrow != 1 {
		t.Errorf("FlexGrow = %v, want 1", n.Style.FlexGrow)
	}
	if n.Style.AccessibilityLabel != "separator" {
		t.Errorf("AccessibilityLabel = %q, want %q", n.Style.AccessibilityLabel, "separator")
	}
}
