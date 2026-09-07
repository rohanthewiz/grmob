package main

import (
	"regexp"
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/htmlout"
)

// The two web targets deliberately do different things with a listbox and a
// tablist, and this file holds the line between them.
//
//	wasm/grmob-runtime.js   the whole ARIA keyboard pattern: a roving
//	                        tabindex, arrow movement, Home/End, activation
//	htmlout                 nothing at all
//
// Everywhere else in this repository a difference between the two DOM targets
// is a bug being closed — the Modal chassis, the tab bar, the stack axes are
// all "the two web targets must agree" pins. This one is the exception, and it
// needs saying out loud or the next reader closes it:
//
// A roving tabindex without the key handler that moves it is strictly worse
// than no pattern. It takes every member but one out of the page's tab order
// and supplies nothing that reaches the rest, so a static export would go from
// "three tab stops" to "one tab stop and two unreachable rows". htmlout is not
// a runtime (docs/platforms/exporters.md, and its own TabView bar is inert
// chrome), so the attribute has nothing to move it and must not be written.
//
// tabindex is behaviour here, not semantics. Every *semantic* half of the two
// patterns — the roles, aria-selected, the aria-controls wiring — is already
// written by both targets and pinned as such elsewhere.

// The composite table's two rows, as core spells them. A rename in core has to
// reach the runtime, and the failure if it does not is the quiet kind: the
// role goes on being written, the runtime goes on not recognising it, and the
// widget silently loses its keyboard again.
var compositeRoles = []struct{ container, member core.Role }{
	{core.RoleListBox, core.RoleOption},
	{core.RoleTabList, core.RoleTab},
}

// COMPOSITE_MEMBERS is a const object literal rather than a lookup function,
// so it is read directly instead of through parseRuntimeTable — which requires
// the `[param] || fallback` tail that proves it found the right braces. The
// equivalent proof here is the assignment being matched whole, name included.
func TestRuntimeCompositeRolesMatchCore(t *testing.T) {
	src := runtimeSource(t)

	decl := regexp.MustCompile(`const COMPOSITE_MEMBERS = \{([^}]*)\};`)
	m := decl.FindStringSubmatch(src)
	if m == nil {
		t.Fatal("grmob-runtime.js: no COMPOSITE_MEMBERS object literal — if it was " +
			"renamed or given a computed shape, update this test rather than deleting it")
	}

	table := map[string]string{}
	for _, pair := range jsPair.FindAllStringSubmatch(m[1], -1) {
		table[pair[1]] = pair[2]
	}
	if len(table) != len(compositeRoles) {
		t.Fatalf("COMPOSITE_MEMBERS has %d rows, want %d: %v",
			len(table), len(compositeRoles), table)
	}
	for _, want := range compositeRoles {
		got, ok := table[string(want.container)]
		if !ok {
			t.Errorf("COMPOSITE_MEMBERS has no row for core.Role %q — a container "+
				"carrying it gets no keyboard, and nothing says so", want.container)
			continue
		}
		if got != string(want.member) {
			t.Errorf("COMPOSITE_MEMBERS[%q] = %q, want %q — the runtime is looking "+
				"for members that no exporter writes", want.container, got, want.member)
		}
	}

	// Every composite must also be an *oriented* role, because that is where
	// the arrow pair now comes from: compositeIsVertical reads aria-orientation
	// off the container and falls back to ARIA_ORIENTATIONS, so a container
	// role missing from that table would take the horizontal arrows by accident
	// rather than by decision. The table's own contents are held against Go by
	// TestRuntimeOrientationTableMatchesGo in orientation_test.go; what is
	// checked here is only that the two tables cover the same containers.
	for _, want := range compositeRoles {
		if _, ok := htmlout.AriaOrientationDefaults()[string(want.container)]; !ok {
			t.Errorf("core.Role %q has a keyboard pattern but no aria-orientation "+
				"default — a container of that role whose axis nothing set would "+
				"take the horizontal arrows by accident, and would announce nothing",
				want.container)
		}
	}
}

// The composites whose members ARIA does not name, and what the runtime takes
// as a member instead.
//
// A second table rather than a third column on compositeRoles, because the two
// are different kinds of answer: a row above is a fact about ARIA's vocabulary
// — a listbox's members are options, and the fixture in aria/verify says so —
// where this is a decision about a walk, made because ARIA defines no
// `toolbaritem` and a toolbar therefore has to be told what its controls are.
var focusableComposites = []core.Role{core.RoleToolbar}

// The two tables above are one list read through two questions, and core now
// states that list: core.KeyboardComposites() is the roles for which asking
// for a composite keyboard means anything.
//
// It exists because a *third* reader arrived. core.AuditTree reports a
// selection-follows-focus flag on a role with no arrows, and it cannot get the
// answer from the runtime — the runtime is JavaScript, and the one place that
// writes the attribute deliberately does not consult these tables (knowing
// them there would put them in two places). So the list moved to core, and
// what this check does is keep the runtime's split-by-membership version and
// core's split-by-nothing version from drifting: a fourth pattern implemented
// in the runtime and not declared in core would be reported as inert by the
// audit while working perfectly in a browser, and a fourth declared in core
// and not implemented would be the reverse.
//
// The union, not either half: which of the two tables a role lands in is the
// runtime's business (does ARIA name its members) and core has no opinion
// about it, so imposing one here would be inventing a fact to check.
func TestTheRuntimeCompositesAreTheOnesCoreDeclares(t *testing.T) {
	runtime := map[core.Role]bool{}
	for _, pair := range compositeRoles {
		runtime[pair.container] = true
	}
	for _, r := range focusableComposites {
		runtime[r] = true
	}

	declared := map[core.Role]bool{}
	for _, r := range core.KeyboardComposites() {
		declared[r] = true
		if !runtime[r] {
			t.Errorf("core.KeyboardComposites() names %q and the runtime has no "+
				"keyboard for it — core.AuditTree would call a working widget's "+
				"selection-follows-focus flag inert, and a container of that role "+
				"would have no arrow keys", r)
		}
	}
	for r := range runtime {
		if !declared[r] {
			t.Errorf("the runtime gives %q a keyboard and core does not declare it — "+
				"core.AuditTree reports a selection-follows-focus flag on it as a "+
				"claim about nothing, and it is not", r)
		}
	}
}

// The roles a Box may carry that make it one of a toolbar's controls.
//
// Asked of core rather than written out here, which is the change that makes
// this a pin rather than a third copy. It used to be a pair of constants in
// this file, held against a pair of bare strings in the runtime: two spellings
// of one fact, neither of which was the fact, and both of which a *new* role
// would leave untouched. That is the failure that mattered — a future
// RoleCheckbox is a tappable container by exactly the argument core.Role makes
// for RoleButton, and it would have shipped as a role a toolbar steps over
// with nothing anywhere disagreeing.
//
// core.TappableContainerRoles() is the fact, and core's own
// role_control_test.go is what keeps it complete: every role in the vocabulary
// is either in that list or has a stated reason for not being. What this file
// still owns is the other half — that the runtime's copy says the same thing.
func controlRoles() []core.Role { return core.TappableContainerRoles() }

// The second member rule's tables, held to core the way COMPOSITE_MEMBERS is.
//
// Both are Sets rather than object literals, so they are matched whole — name
// included — rather than through parseRuntimeTable.
func TestTheFocusableCompositeTablesMatchCore(t *testing.T) {
	src := runtimeSource(t)

	for _, want := range []struct {
		expr string
		why  string
	}{
		{jsSet("COMPOSITE_FOCUSABLE", focusableComposites),
			"a toolbar's members are not named by its role, so the runtime takes " +
				"every focusable control inside it. A container role missing here " +
				"is announced with an axis and given no keyboard, which is what a " +
				"toolbar was for two releases"},
		{jsSet("CONTROL_ROLES", controlRoles()),
			"the two roles core.Role documents as a tappable container. A Box " +
				"carrying one, with an OnTap, is a control the browser gives no tab " +
				"stop of its own — so a toolbar of icon boxes depends entirely on " +
				"this set"},
	} {
		if !strings.Contains(src, want.expr) {
			t.Errorf("grmob-runtime.js: %s not found — %s", want.expr, want.why)
		}
	}

	// Every focusable composite must be an oriented role too, and for the same
	// reason the two structural pairs must be: compositeIsVertical reads
	// aria-orientation off the container, so a container role missing from that
	// table takes the horizontal arrows by accident rather than by decision.
	for _, role := range focusableComposites {
		if _, ok := htmlout.AriaOrientationDefaults()[string(role)]; !ok {
			t.Errorf("core.Role %q has a keyboard pattern but no aria-orientation "+
				"default", role)
		}
	}

	// The two member rules must not both claim a role. compositeMembersOf asks
	// COMPOSITE_MEMBERS first, so a role in both tables would silently take the
	// role-named walk and its focusable half would be dead.
	for _, role := range focusableComposites {
		for _, pair := range compositeRoles {
			if pair.container == role {
				t.Errorf("core.Role %q is in both member tables — compositeMembersOf "+
					"asks the role-named one first, so the focusable rule would "+
					"never run for it", role)
			}
		}
	}
}

// jsSet renders the runtime's spelling of a role set, so the pin is the literal
// source line rather than a parse of it.
func jsSet(name string, roles []core.Role) string {
	parts := make([]string, len(roles))
	for i, r := range roles {
		parts[i] = `"` + string(r) + `"`
	}
	return "const " + name + " = new Set([" + strings.Join(parts, ", ") + "]);"
}

// htmlout writes the semantics and stops there. See the file comment for why
// this is the one two-web-target difference that is deliberate.
//
// The toolbar is checked beside the two structural pairs, and the argument
// carries over unchanged: a static export has no key handler, so a roving
// tabindex on a chip strip would take every chip but one out of the tab order
// and reach none of them.
func TestTheStaticExportWritesNoRovingTabindex(t *testing.T) {
	for _, role := range focusableComposites {
		out := htmlout.ExportHTML(&core.Node{
			Type:  "Row",
			Style: &core.Style{AccessibilityRole: role},
			Children: []*core.Node{
				{Type: "Button", Props: map[string]any{"label": "All", "onClick": "cb_0"}},
				{Type: "Button", Props: map[string]any{"label": "Sermons", "onClick": "cb_1"}},
			},
		})
		if strings.Contains(out, "tabindex") {
			t.Errorf("htmlout wrote a tabindex for a %s:\n%s", role, out)
		}
		if !strings.Contains(out, `role="`+string(role)+`"`) {
			t.Errorf("htmlout dropped role=%q:\n%s", role, out)
		}
	}

	for _, pair := range compositeRoles {
		container := &core.Node{
			Type:  "Column",
			Style: &core.Style{AccessibilityRole: pair.container},
			Children: []*core.Node{
				{Type: "Box", Style: &core.Style{
					AccessibilityRole:     pair.member,
					AccessibilitySelected: core.SelectedOn,
				}},
				{Type: "Box", Style: &core.Style{
					AccessibilityRole:     pair.member,
					AccessibilitySelected: core.SelectedOff,
				}},
			},
		}
		out := htmlout.ExportHTML(container)

		if strings.Contains(out, "tabindex") {
			t.Errorf("htmlout wrote a tabindex for a %s:\n%s\n\n"+
				"A roving tabindex with nothing to move it takes every member but "+
				"one out of the tab order and reaches none of them. The static "+
				"export has no key handler, so it must write no tab stops.",
				pair.container, out)
		}
		// The semantic half is still there, and is the reason the runtime can
		// do the rest without a single new prop.
		for _, want := range []string{
			`role="` + string(pair.container) + `"`,
			`role="` + string(pair.member) + `"`,
			`aria-selected="true"`,
			`aria-selected="false"`,
		} {
			if !strings.Contains(out, want) {
				t.Errorf("htmlout dropped %s from a %s:\n%s", want, pair.container, out)
			}
		}
	}
}

// Typeahead: the half of ARIA's listbox keyboard that makes a long one usable,
// and the one piece of state this whole section owns.
//
// keynav_test.mjs holds the behaviour. What is pinned from Go is the two things
// that are decisions rather than mechanics, because both are silent when
// undone: which composites have a search at all, and that the buffer is not
// held by a timer.
func TestTheTypeaheadIsScopedAndUntimered(t *testing.T) {
	src := runtimeSource(t)
	for _, want := range []struct{ expr, why string }{
		{`const COMPOSITE_TYPEAHEAD = new Set(["listbox"]);`,
			"a listbox and not a tablist, which is ARIA's own division: type-to-jump is " +
				"part of the listbox pattern because a listbox can be a hundred options " +
				"long, and is not part of the tab pattern because a strip's members are " +
				"all on screen"},
		{`const typeahead = { container: null, text: "", at: 0 };`,
			"one buffer, not one per widget. Only one thing has focus at a time, so a " +
				"second widget's buffer could never be the live one — and the container " +
				"is recorded beside the text so a keystroke elsewhere starts over"},
		{`now - typeahead.at > TYPEAHEAD_RESET_MS`,
			"expiry checked on the next keystroke rather than driven by a timer. A " +
				"setTimeout would need cancelling on unmount, and a widget removed by a " +
				"patch has no unmount hook to cancel it from"},
		{`const repeated = /^(.)\1*$/.test(typeahead.text);`,
			`ARIA's two search modes, which are one rule: "sss" is the third press of s ` +
				`and cycles, where "seq" is a query that refines`},
	} {
		if !strings.Contains(src, want.expr) {
			t.Errorf("grmob-runtime.js: %q not found — %s", want.expr, want.why)
		}
	}

	// The name walk descends to the leaves rather than reading textContent at
	// each level, and this is pinned from Go because keynav_test.mjs *cannot*
	// tell the two apart: dom.mjs stores textContent on leaves only, so a
	// container answers "" and a per-level read adds nothing there. In a
	// browser it concatenates every descendant's text, and the same code would
	// count a member's words once per ancestor — so "Sermons" inside two
	// wrappers matches "sermons sermons sermons" and nothing else.
	//
	// This is item-22 shaped: the shim models enough for everything the roving
	// tabindex does and is not a browser, and where the difference matters the
	// check has to be a pin rather than a behaviour.
	if !strings.Contains(src, `        if (el.children.length === 0) {
            if (el.textContent) out.push(el.textContent);
            return out;
        }`) {
		t.Error("memberText no longer descends to the leaves. Reading textContent at " +
			"each level is the obvious rewrite and is wrong in a browser, where it " +
			"concatenates every descendant — the harness DOM cannot tell, which is " +
			"why this is asserted here rather than in keynav_test.mjs")
	}

	// A timer would be the natural implementation and is the thing this must
	// not become, so the absence is asserted rather than left to the comment.
	// The search is bounded to the typeahead's own source, since the runtime
	// uses setTimeout legitimately elsewhere (the long-press timer).
	body := src[strings.Index(src, "function compositeTypeahead("):]
	body = body[:strings.Index(body, "\n    }")]
	if strings.Contains(body, "setTimeout") {
		t.Error("compositeTypeahead has grown a timer. Expiry is checked on the next " +
			"keystroke because that is the only moment it can matter, and because a " +
			"timer outlives the widget that started it")
	}
}

// The rule the audit's sentence and the runtime's two walks both answer to.
//
// # What was not held together
//
// core.KeyboardComposites() has been pinned to the runtime's two tables for a
// while: the roles that get a keyboard are the same on both sides. What was
// never pinned is what a walk *does when it meets one composite inside
// another*, and core.AuditTree reports a finding whose text depends on it —
// ConcernNestedComposite tells an author either that the outer widget's arrows
// step over the inner one or that they can land inside it, and those are
// different bugs to go looking for.
//
// The Go side had been asserting the first for every pair. It is true for four
// of the nine and false for five, because compositeMembers deliberately
// descends through a composite of the other kind — "an option below a tablist
// is still the listbox's option", in its own words. So a Go author with a
// tablist inside a listbox was being told the strip was stepped over, while
// the runtime was pooling any option buried in it into the outer rotation.
//
// # What is checked
//
// Both halves of core's rule against both walks, at the source level, because
// the rule is two lines of JavaScript and each line is one of core's cases:
//
//	compositeMembers   stops only where the nested container's member role is
//	                   the walk's own — core.CompositeWalkStopsAt's second case
//	focusableMembers   stops at any composite — its first case, which is what
//	                   a container that names no member role must do
//
// A source pin rather than a behavioural one here because this is the Go half:
// keynav_test.mjs runs a descending pair in a real DOM, and what this adds is
// that the two lines it runs are the two lines core claims to be describing.
// Either alone would let the pair drift — a behavioural test on one shape says
// nothing about the other eight, and core's rule with no reader is prose.
func TestTheRuntimeWalksStopWhereCoreSaysTheyDo(t *testing.T) {
	src := runtimeSource(t)

	// core's member-role table is the runtime's, read through the independent
	// restatement above rather than through the runtime text — the point of
	// compositeRoles is to be a second statement of COMPOSITE_MEMBERS, and
	// core is now a third that has to agree with it.
	for _, pair := range compositeRoles {
		if got := core.CompositeMemberRole(pair.container); got != pair.member {
			t.Errorf("core.CompositeMemberRole(%q) = %q, want %q — the audit would "+
				"name the wrong member role in a nested-composite finding, and "+
				"CompositeWalkStopsAt would put the pair in the wrong case",
				pair.container, got, pair.member)
		}
	}
	for _, r := range focusableComposites {
		if got := core.CompositeMemberRole(r); got != "" {
			t.Errorf("core.CompositeMemberRole(%q) = %q — this is a composite whose "+
				"members ARIA does not name, and a member role here would make "+
				"CompositeWalkStopsAt stop descending for the wrong reason", r, got)
		}
	}

	// And the two lines the rule is a description of.
	for _, pin := range []struct{ walk, stop, why string }{
		{"function compositeMembers(container, memberRole",
			"if (compositeMemberRole(child) === memberRole) continue;",
			"a role-named composite stops at a nested container whose members are " +
				"its own and descends through every other — core.CompositeWalkStopsAt " +
				"compares the two member roles for exactly this line"},
		{"function focusableMembers(container",
			"if (isComposite(child)) continue;",
			"a composite that names no member role stops at any nested composite — " +
				"core.CompositeWalkStopsAt's first case, and the one that has no " +
				"member role to compare"},
	} {
		at := strings.Index(src, pin.walk)
		if at < 0 {
			t.Errorf("grmob-runtime.js: no %s — if the walk was renamed, update this "+
				"test rather than deleting it", pin.walk)
			continue
		}
		body := src[at:]
		if end := strings.Index(body, "\n    }\n"); end >= 0 {
			body = body[:end]
		}
		if !strings.Contains(body, pin.stop) {
			t.Errorf("grmob-runtime.js: %s no longer stops with %q — %s",
				pin.walk, pin.stop, pin.why)
		}
	}
}

// Every ordered pair of composites lands in one of core's two outcomes, and
// both outcomes are reachable.
//
// A rule that answered the same way for all nine pairs would pass every check
// above — the source pins hold the runtime to two lines, not to what those
// lines produce — and would put the audit back where it started, telling every
// author the same sentence. This is the check that the rule discriminates.
func TestBothNestedCompositeOutcomesAreReachable(t *testing.T) {
	var stops, descends int
	for _, outer := range core.KeyboardComposites() {
		for _, inner := range core.KeyboardComposites() {
			if core.CompositeWalkStopsAt(outer, inner) {
				stops++
			} else {
				descends++
			}
			// Whatever the pair, a composite always stops at its own kind:
			// pooling two listboxes' options is the case the runtime's walk
			// was written to prevent.
			if outer == inner && !core.CompositeWalkStopsAt(outer, inner) {
				t.Errorf("a %q inside a %q descends — the two would pool their "+
					"members and one widget's arrows would walk out into the "+
					"other's rows", inner, outer)
			}
		}
	}
	if stops == 0 || descends == 0 {
		t.Errorf("CompositeWalkStopsAt answered %d stop / %d descend over the %d "+
			"ordered pairs — one outcome is unreachable, so the audit reports one "+
			"sentence for every pair again", stops, descends,
			len(core.KeyboardComposites())*len(core.KeyboardComposites()))
	}
}
