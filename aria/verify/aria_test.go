package verify

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/htmlout"
)

// roleSpec is one role's entry in the fixture. See doc.go for what is in scope.
type roleSpec struct {
	Attributes    []string `json:"attributes"`
	Orientation   string   `json:"orientation"`
	RequiredOwned []string `json:"requiredOwned"`
}

type ariaSpec struct {
	NameProhibited []string            `json:"nameProhibited"`
	Roles          map[string]roleSpec `json:"roles"`
}

func loadSpec(t *testing.T) ariaSpec {
	t.Helper()
	path := filepath.Join("testdata", "aria.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	var spec ariaSpec
	if err := json.Unmarshal(raw, &spec); err != nil {
		t.Fatalf("parsing %s: %v", path, err)
	}
	if len(spec.Roles) == 0 {
		t.Fatalf("%s parsed to no roles at all", path)
	}
	return spec
}

func (s ariaSpec) supports(role, attr string) bool {
	for _, a := range s.Roles[role].Attributes {
		if a == attr {
			return true
		}
	}
	return false
}

// The vocabulary is ARIA's, which is the premise the two DOM targets rest on:
// they write the value verbatim and need no mapping table. A core.Role that is
// not a real role would be written into the attribute all the same, and every
// browser would drop it silently.
func TestEveryCoreRoleIsAnARIARole(t *testing.T) {
	spec := loadSpec(t)
	for _, role := range core.Roles() {
		if _, ok := spec.Roles[string(role)]; !ok {
			t.Errorf("core.Role %q is not in the ARIA fixture — either it is not a real "+
				"role (both web targets would write it and every browser would drop it), "+
				"or the fixture needs the entry", role)
		}
	}
}

// The four state guards, each checked in both directions against the fixture.
//
// This is the check the prose could not be: a role list in a doc comment agrees
// with the switch beside it only if a person compared them, and the two web
// targets then each restate the same list a third and fourth time. Here the
// fixture says which roles ARIA scopes an attribute to, and the export is asked.
//
// Both directions fail differently. A role that should carry an attribute and
// does not is a state a reader never hears. A role that should not and does is
// invalid ARIA in the document — dropped by the reader, so it changes nothing a
// user hears and everything a developer believes.
func TestTheStateGuardsMatchARIAsScoping(t *testing.T) {
	spec := loadSpec(t)

	// One node per role, carrying every state at once. The exporter decides
	// which of them survive, which is exactly the question.
	for _, role := range core.Roles() {
		style := &core.Style{
			AccessibilityRole:         role,
			AccessibilityHeadingLevel: 3,
			AccessibilityNestingLevel: 3,
			AccessibilitySelected:     core.SelectedOn,
			AccessibilityExpanded:     core.ExpandedOpen,
			AccessibilityValue:        core.ValueOf(45, 0, 100).WithText("45 percent"),
		}
		out := htmlout.ExportHTML(&core.Node{Type: "Box", Style: style})

		for _, attr := range []string{
			"aria-level", "aria-selected", "aria-pressed", "aria-expanded",
			"aria-valuenow", "aria-valuemin", "aria-valuemax", "aria-valuetext",
		} {
			want := spec.supports(string(role), attr)
			got := strings.Contains(out, attr+"=")
			if got == want {
				continue
			}
			if want {
				t.Errorf("role %q: ARIA defines %s and the export writes none — the "+
					"state is stated in Go and never reaches a reader\n%s", role, attr, out)
			} else {
				t.Errorf("role %q: the export writes %s and ARIA does not define it "+
					"there — invalid ARIA, which a reader drops, so it changes nothing a "+
					"user hears and everything a developer believes\n%s", role, attr, out)
			}
		}
	}
}

// aria-level is the one attribute two Go fields become, and the fixture is what
// says which of core's roles each field is for. The heading tier and the
// nesting depth are validated differently and must not be interchangeable.
func TestTheTwoLevelFieldsReachTheRolesARIAScopesThemTo(t *testing.T) {
	spec := loadSpec(t)
	for _, role := range core.Roles() {
		if !spec.supports(string(role), "aria-level") {
			continue
		}
		heading := htmlout.ExportHTML(&core.Node{Type: "Box", Style: &core.Style{
			AccessibilityRole:         role,
			AccessibilityHeadingLevel: 3,
		}})
		nesting := htmlout.ExportHTML(&core.Node{Type: "Box", Style: &core.Style{
			AccessibilityRole:         role,
			AccessibilityNestingLevel: 3,
		}})
		// Exactly one of the two fields answers for each role, which is what
		// makes them unable to contend for the one attribute.
		if strings.Contains(heading, "aria-level") == strings.Contains(nesting, "aria-level") {
			t.Errorf("role %q reads both level fields or neither: one node has to answer "+
				"to exactly one of them, or the pair can produce two values for one "+
				"attribute\nheading:\n%s\nnesting:\n%s", role, heading, nesting)
		}
	}
}

// aria-orientation, in both directions: the table's defaults are ARIA's own,
// and it covers exactly the roles ARIA scopes the attribute to.
//
// The second half is what the entry that asked for this could not check. A
// composite whose default is missing takes the horizontal arrows by accident,
// and a role in the table that ARIA does not orient writes an attribute a
// reader ignores.
func TestTheOrientationTableIsARIAsOwn(t *testing.T) {
	spec := loadSpec(t)
	table := htmlout.AriaOrientationDefaults()

	for role, orientation := range table {
		entry, ok := spec.Roles[role]
		if !ok {
			t.Errorf("ariaOrientations names %q, which the fixture has no entry for", role)
			continue
		}
		if !spec.supports(role, "aria-orientation") {
			t.Errorf("ariaOrientations orients %q and ARIA does not define "+
				"aria-orientation there", role)
			continue
		}
		if entry.Orientation != orientation {
			t.Errorf("ariaOrientations[%q] = %q, ARIA's default is %q — a container "+
				"whose axis nothing set is announced the wrong way round",
				role, orientation, entry.Orientation)
		}
	}

	var missing []string
	for _, role := range core.Roles() {
		if spec.supports(string(role), "aria-orientation") {
			if _, ok := table[string(role)]; !ok {
				missing = append(missing, string(role))
			}
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		t.Errorf("core carries %v, which ARIA orients, and ariaOrientations has no row "+
			"for them — the axis is not announced, and on the runtime the arrow pair "+
			"falls back to horizontal whatever the layout is", missing)
	}
}

// The premise of the whole RoleGroup fallback, which both web exporters supply
// silently and which is argued at length in three places: a name on `generic`
// is prohibited and dropped, and `group` is the smallest role that makes it
// legal.
//
// If that premise were ever false, the fallback would be writing an attribute
// for nothing on every named container in every app.
func TestTheGroupFallbacksPremiseHolds(t *testing.T) {
	spec := loadSpec(t)
	prohibited := map[string]bool{}
	for _, r := range spec.NameProhibited {
		prohibited[r] = true
	}
	if !prohibited["generic"] {
		t.Error("the fixture says a name on `generic` is allowed, which is the opposite " +
			"of the reason ariaRole supplies `group` at all — a <div> is generic, and " +
			"the fallback exists because browsers prune the name")
	}
	if prohibited[string(core.RoleGroup)] {
		t.Errorf("the fixture says %q cannot be named, which would make the fallback "+
			"write an attribute that changes nothing", core.RoleGroup)
	}
	// And the other half of core.RoleGroup's argument: it promises nothing
	// about its children, which is what lets it be given to a container
	// nothing has looked inside.
	if owned := spec.Roles[string(core.RoleGroup)].RequiredOwned; len(owned) > 0 {
		t.Errorf("%q requires owned elements %v — the fallback supplies it to containers "+
			"nobody has inspected, so a role that claims children would make a claim the "+
			"framework cannot keep", core.RoleGroup, owned)
	}
}

// The runtime's keyboard table, against ARIA's own ownership. A composite
// container's members are the children its role requires; a table row that
// looked for the wrong member would find none and the widget would silently
// have no keyboard.
//
// Read out of the runtime source rather than from Go, because that is where the
// table lives — the same treatment wasm/verify gives it, one question further
// along: that file checks the pair against core's spellings, and this checks it
// against ARIA's structure.
func TestTheCompositeMembersAreARIAsRequiredChildren(t *testing.T) {
	spec := loadSpec(t)
	for _, pair := range []struct{ container, member core.Role }{
		{core.RoleListBox, core.RoleOption},
		{core.RoleTabList, core.RoleTab},
	} {
		owned := spec.Roles[string(pair.container)].RequiredOwned
		found := false
		for _, o := range owned {
			if o == string(pair.member) {
				found = true
			}
		}
		if !found {
			t.Errorf("the runtime looks for %q inside %q, and ARIA says a %s owns %v — "+
				"a container whose members are the wrong role has no members at all, "+
				"and the widget silently loses its keyboard",
				pair.member, pair.container, pair.container, owned)
		}
	}
}

// The near misses, which every guard in both exporters argues with in prose and
// which are assertions here. Each is a role that looks like one core carries and
// is scoped differently.
func TestTheNearMissesAreRealDistinctions(t *testing.T) {
	spec := loadSpec(t)
	for _, tc := range []struct {
		mine, theirs, attr, why string
	}{
		{"cell", "gridcell", "aria-selected",
			"a table cell is not selectable; a grid cell in an interactive grid is, " +
				"and core.Role has no grid"},
		{"listitem", "option", "aria-selected",
			"a list item is content and an option is a control in a listbox, which is " +
				"why comps.ListRow has to give up one role to take the other"},
		{"option", "listbox", "aria-expanded",
			"an option is a leaf choice; the thing that expands is the listbox around it"},
		{"progressbar", "slider", "aria-orientation",
			"a bar has no axis a user moves along, which is why core.Slider is a node " +
				"type with a native range rather than a role with an ARIA one"},
	} {
		if spec.supports(tc.mine, tc.attr) {
			t.Errorf("the fixture says %q supports %s, which would make the distinction "+
				"from %q that %s a distinction the exporters draw for no reason",
				tc.mine, tc.attr, tc.theirs, tc.why)
		}
		if !spec.supports(tc.theirs, tc.attr) {
			t.Errorf("the fixture says %q does not support %s — %s", tc.theirs, tc.attr, tc.why)
		}
	}
}
