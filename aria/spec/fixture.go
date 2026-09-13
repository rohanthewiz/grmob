package spec

import (
	"fmt"
	"strings"

	"github.com/rohanthewiz/grmob/core"
)

// InScopeAttributes are the aria-* attributes this framework can write, in the
// order the fixture lists them.
//
// The fixture is deliberately partial and this list is the whole of why: a
// fixture is worth exactly what is checked against it, and an entry no guard
// argues with is prose again with braces around it. Every attribute here is
// written by htmlout, by the WASM runtime, or by both, and has a test in
// aria/verify holding that writer to this fixture's answer.
//
// The order is the reading order rather than alphabetical: the level, then the
// three selection spellings, then the disclosure, then the axis, then the value
// family. Emitted in this order into every role's "attributes" list, so a
// regenerated fixture diffs against the previous one line for line.
//
// Adding a row here widens every generated entry and nothing else; it does not
// add a guard. Adding a *guard* is what makes the row worth having.
var InScopeAttributes = []string{
	"aria-level",
	"aria-selected",
	"aria-pressed",
	"aria-checked",
	"aria-expanded",
	"aria-orientation",
	"aria-valuenow",
	"aria-valuemin",
	"aria-valuemax",
	"aria-valuetext",
}

// NearMisses are roles core deliberately does not carry, kept in the fixture so
// the guards have something to argue with.
//
// Every one of these is a role some comment in this repository names as the
// thing a value is *not*: `cell` is not `gridcell`, `progressbar` is not
// `meter` or `slider`, `option` is not `treeitem`, `columnheader` is not
// `rowheader`, `listbox` is not `combobox`, `table` is not `grid`. Having them
// here turns each of those remarks into an assertion — see
// TestTheNearMissesAreRealDistinctions.
//
// `generic`, `dialog` and `switch` are here for a second reason. `generic` is
// the premise of the RoleGroup fallback both web exporters supply: a name on it
// is prohibited, which is why `group` exists in the vocabulary at all.
// `dialog` and `switch` are the two roles a node *type* writes rather than a
// core.Role — core.Modal's chassis and core.Switch, the whole of
// htmlout.ownRoles — so they are roles this framework emits and does not name.
// Being here is what lets TestEveryCoreRoleIsAnARIARole's sibling checks argue
// about them at all: a role no fixture carries is a string two renderers happen
// to agree on.
//
// `menu`, `menubar`, `tree`, `treegrid` and `grid` are the composite patterns
// core has no vocabulary for, and `menuitem`, `menuitemcheckbox`,
// `menuitemradio` and `treeitem` are the member roles those patterns move
// between. (`radiogroup` and `radio` sat here until core.RoleRadioGroup and
// core.RoleRadio made them vocabulary; they are now in the fixture as core's
// own roles.) Both halves are here because
// aria/verify/refusals_test.go holds each refusal to what the pattern actually
// requires rather than to a sentence somebody wrote once — and the member roles
// are the half that decides which blocker a refusal is still standing on.
var NearMisses = []string{
	"generic",
	"dialog",
	"switch",
	"gridcell",
	"meter",
	"scrollbar",
	"slider",
	"spinbutton",
	"menu",
	"menubar",
	"separator",
	"tree",
	"treegrid",
	"treeitem",
	"rowheader",
	"combobox",
	"grid",
	"menuitem",
	"menuitemcheckbox",
	"menuitemradio",
}

// ScopedRoles is every role the fixture carries, in the order it carries them:
// core's own vocabulary first, in core.Roles() order, then the near misses.
//
// core.Roles() rather than a second list, so a role added to the vocabulary is
// in the fixture the next time it is generated and nobody has to remember.
func ScopedRoles() []string {
	out := make([]string, 0, len(core.Roles())+len(NearMisses))
	for _, r := range core.Roles() {
		out = append(out, string(r))
	}
	return append(out, NearMisses...)
}

// Entry is one role as the fixture states it: the in-scope attributes only.
type Entry struct {
	Attributes    []string
	Orientation   string
	RequiredOwned []string
}

// Fixture is the whole file: the roles in scope, plus every role ARIA forbids a
// name on.
type Fixture struct {
	NameProhibited []string
	Roles          map[string]Entry
	// Order is Roles' keys in emission order. A map has none and the file
	// wants one; see Render.
	Order []string
}

// Scope reduces a parsed specification to the fixture this framework checks
// against: the roles in ScopedRoles, each holding only the InScopeAttributes it
// supports.
//
// A role in scope that the specification has no section for is an error rather
// than an omission. It means core carries a role ARIA does not define — which
// the two web exporters would write into the attribute verbatim and every
// browser would drop — and finding that at generation time is better than
// finding it in TestEveryCoreRoleIsAnARIARole, because here it names the file.
func Scope(doc Doc) (Fixture, error) {
	order := ScopedRoles()
	fx := Fixture{Roles: make(map[string]Entry, len(order)), Order: order}

	for _, name := range order {
		r, ok := doc[name]
		if !ok {
			return Fixture{}, fmt.Errorf("role %q is in the fixture's scope and "+
				"the specification defines no such role", name)
		}
		// Non-nil so the empty case marshals as [] rather than null. The
		// fixture's readers range over it either way; the difference is only
		// in the file, and "attributes": null reads as a gap in the data
		// where "attributes": [] reads as the answer it is.
		attrs := []string{}
		for _, a := range InScopeAttributes {
			if contains(r.Attributes, a) {
				attrs = append(attrs, a)
			}
		}
		fx.Roles[name] = Entry{
			Attributes:    attrs,
			Orientation:   r.Orientation,
			RequiredOwned: r.RequiredOwned,
		}
	}

	// Over the whole specification rather than over the scoped roles: this
	// list is a complete fact and a cheap one, and the guard that reads it
	// asks about `generic` — which is in scope — and about `group`, which
	// must be absent from it. A list narrowed to the scope could not tell
	// "not prohibited" from "not looked at".
	for name, r := range doc {
		if r.NameProhibited {
			fx.NameProhibited = append(fx.NameProhibited, name)
		}
	}
	sortStrings(fx.NameProhibited)
	return fx, nil
}

func contains(hay []string, needle string) bool {
	for _, s := range hay {
		if s == needle {
			return true
		}
	}
	return false
}

func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}

// about is the header the generated file carries. It is the first thing anyone
// opening aria.json reads, and its whole job is to stop them editing it.
var about = []string{
	"GENERATED by aria/gen from the W3C ARIA specification. Do not edit by hand.",
	"Regenerate: sh aria/fetch.sh && go run ./aria/gen",
	"ARIA 1.2 role definitions, scoped to the attributes this framework can write.",
	"See aria/spec/spec.go for what is read out of the specification, and",
	"aria/verify/doc.go for what this file is checked against.",
}

// Render writes the fixture as the JSON aria/verify reads.
//
// Hand-emitted rather than encoding/json, for one reason: order. The roles are
// listed in core.Roles() order followed by the near misses, which is the order
// somebody reading the file wants and the order a diff between two generations
// stays legible in; a map marshals sorted by key and would reshuffle the whole
// file the first time a role was renamed. Each role's arrays are kept on one
// line for the same reason — an entry is a row, and a row that spans nine lines
// makes a one-attribute change look like a rewrite.
func (f Fixture) Render() []byte {
	var b strings.Builder
	b.WriteString("{\n")

	b.WriteString("  \"_about\": [\n")
	for i, line := range about {
		b.WriteString("    " + quote(line))
		if i < len(about)-1 {
			b.WriteString(",")
		}
		b.WriteString("\n")
	}
	b.WriteString("  ],\n")

	b.WriteString("  \"nameProhibited\": " + jsonArray(f.NameProhibited) + ",\n")

	b.WriteString("  \"roles\": {\n")
	for i, name := range f.Order {
		e := f.Roles[name]
		b.WriteString("    " + quote(name) + ": {\n")
		b.WriteString("      \"attributes\": " + jsonArray(e.Attributes))
		if e.Orientation != "" {
			b.WriteString(",\n      \"orientation\": " + quote(e.Orientation))
		}
		if len(e.RequiredOwned) > 0 {
			b.WriteString(",\n      \"requiredOwned\": " + jsonArray(e.RequiredOwned))
		}
		b.WriteString("\n    }")
		if i < len(f.Order)-1 {
			b.WriteString(",")
		}
		b.WriteString("\n")
	}
	b.WriteString("  }\n")
	b.WriteString("}\n")
	return []byte(b.String())
}

func jsonArray(items []string) string {
	if len(items) == 0 {
		return "[]"
	}
	parts := make([]string, len(items))
	for i, s := range items {
		parts[i] = quote(s)
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

// quote is enough for this file's alphabet: role names and attribute names are
// ASCII letters and hyphens, and the _about lines are plain prose. Anything
// needing an escape would be a value that does not belong in this fixture.
func quote(s string) string {
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(s) + `"`
}

// The two paths this package's facts live at, relative to the module root.
//
// Stated here rather than in aria/gen because aria/verify's conformance test
// needs both of them too, and a second spelling in a second package is how the
// generator and the check end up reading different files.
const (
	// LocalPath is where aria/fetch.sh puts the specification. Not committed.
	//
	// The revision is in the filename and comes from Version, so bumping the
	// edition moves the download, the parser's guard and this path together
	// rather than leaving a 1.2 filename holding a 1.3 document.
	LocalPath = "aria/spec/testdata/wai-aria-" + Version + ".html"
	// FixturePath is the generated fixture. Committed, and the only one of the
	// two anything on a verification path reads.
	FixturePath = "aria/verify/testdata/aria.json"
)
