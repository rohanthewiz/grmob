package themeleaves

import (
	"go/token"
	"os"
	"reflect"
	"slices"
	"sort"
	"testing"
)

// The shapes below are what this package is a walker OF, as opposed to what it
// currently answers.
//
// Its only arm until now lived in wasm/verify and asked one question — does
// this package agree with reflect about core.Theme — which it does, at eighty
// names, on a struct that happens to contain no pointer to a struct, no
// embedded field, no generic and no type from another package. Every rule in
// walk is exercised by that answer only as far as core.Theme exercises it, and
// the rules that are not are three lines of source text each.
//
// So they are declared here as REAL Go types, and this file's own source is
// handed to Of. The two mechanisms then have something to disagree about that
// nobody had to write down twice: the source the parser reads and the type the
// reflect walk reads are the same declaration, and a shape that only one of
// them handles shows up as a difference rather than as a fixture nobody
// updated.

// shapeLeafKinds is a plain struct with two leaves, used as the child in most
// of what follows.
type shapeLeafKinds struct {
	Size   int
	Weight string
}

// shapeSiblings is the ninety-one-name bug's own shape: three fields of ONE
// struct type, which are three subtrees and not a recursion.
//
// A `seen` set that grew as the field loop ran gives Size, Weight, Caption and
// Subtitle here — the first sibling descends and the other two are recorded as
// leaves under their own names. See this package's doc comment, which draws
// it against Typography.
type shapeSiblings struct {
	Body     shapeLeafKinds
	Caption  shapeLeafKinds
	Subtitle shapeLeafKinds
}

type shapeCorner struct{ Radius int }

// shapePointer holds one of the same struct by value and one by pointer.
//
// reflect recurses on Kind() == Struct and a *T is not one, so Aside is a leaf
// under its FIELD name where Held disappears into its children. unwrap is what
// keeps this walk on the same side of that line.
type shapePointer struct {
	Held  shapeLeafKinds
	Aside *shapeLeafKinds
}

// shapeEmbedded is the two embedded spellings that resolve here: a value embed
// of a local struct, which descends and loses its own name, and a pointer
// embed, which is a leaf named by its type.
type shapeEmbedded struct {
	shapeLeafKinds
	*shapeCorner
	Extra int
}

// shapeShared is one leaf name under two parents, which the population counts
// once — the same reason affordedLeafNames deduplicates (Colors.Surface and
// Colors.Overlay.Surface).
type shapeShared struct {
	Left  shapeCorner
	Right shapeCorner
}

// shapeAnonymous is a struct type written in the field rather than named,
// which has no entry in `structs` to be looked up and is descended into
// directly.
type shapeAnonymous struct {
	Inline struct {
		Depth int
	}
	Flat int
}

type shapeNoFields struct{}

// shapeEmpty holds a struct with nothing in it. reflect contributes no path
// for such a field — not even the field's own name — because the recursion
// returns an empty list rather than a leaf, and this walk has to do the same.
type shapeEmpty struct {
	Nothing   shapeNoFields
	Something int
}

// shapeMultiName is two names on one field, in both the descending and the
// leaf case. Go gives each name its own field and so does reflect, and the
// walk here loops over the names for exactly that reason.
type shapeMultiName struct {
	First, Second shapeCorner
	Third, Fourth int
}

type shapeDeepC struct{ C int }
type shapeDeepB struct{ B shapeDeepC }

// shapeDeep is three levels, so that "take the last dotted segment" is asked
// something a single level cannot ask it.
type shapeDeep struct{ A shapeDeepB }

// The three shapes below are where the two mechanisms are NOT the same walk,
// and the difference is one-sided in a direction this package documents: a
// type it cannot resolve to a struct declaration it parsed stops the walk, and
// the FIELD's name is kept where reflect would have gone in and kept the
// children.

type shapeBox[T any] struct{ Value T }

// shapeGeneric embeds an instantiation. reflect sees a struct and recurses;
// this walk sees an *ast.IndexExpr, which is not a name in `structs`.
type shapeGeneric struct {
	shapeBox[int]
	Own int
}

type shapeAlias = shapeCorner

// shapeAliased holds a field whose type is an alias. go/ast records the alias
// as a TypeSpec whose Type is an identifier rather than a struct, so it never
// enters `structs` and the field is a leaf.
type shapeAliased struct {
	Corner shapeAlias
	Own    int
}

// shapeForeign holds a struct from another package — the case that is not
// hypothetical, since it is what "Theme is assembled from more than core/"
// would mean in wasm/verify's failure message.
type shapeForeign struct {
	Where token.Position
	Own   int
}

// reflectLeafNames is affordedLeafNames' rule, over reflect.
//
// A second implementation on purpose. The whole worth of this package is that
// two mechanisms produce one population, and a test that shared code with
// either of them would be measuring one mechanism twice. It is eight lines,
// which is what makes that affordable.
//
// # Which makes three copies of the rule, and this is the third
//
// "Recurse on a struct, take the last dotted segment, keep each name once"
// now exists here:
//
//	wasm/verify/gen.go              themeLeafPaths     the walk, as paths
//	wasm/verify/themenearmiss_      affordedLeafNames  the dedup on top of it;
//	  test.go                                          the population that
//	                                                   file measures
//	internal/themeleaves/           reflectLeafNames   both of those collapsed
//	  themeleaves_test.go                              — this one
//
// Two of those are one mechanism in two pieces and could be one. This one
// could not: it is what the PARSE in themeleaves.go is held against, and a
// comparison whose two sides share code is a comparison of a thing with
// itself. The cost of that argument is a copy nobody else reads, which is
// exactly the copy that can drift without a failure — so it is named here, at
// all three sites, rather than left to be discovered as a coincidence.
//
// If wasm/verify ever exports the pair, this file importing them would be the
// wrong economy for the same reason.
func reflectLeafNames(v reflect.Value) []string {
	seen := map[string]bool{}
	var walk func(v reflect.Value, name string)
	walk = func(v reflect.Value, name string) {
		if v.Kind() == reflect.Struct {
			for i := 0; i < v.NumField(); i++ {
				walk(v.Field(i), v.Type().Field(i).Name)
			}
			return
		}
		seen[name] = true
	}
	walk(v, "")
	out := make([]string, 0, len(seen))
	for name := range seen {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// shapeSources is this file's own text, under a name Of will read.
//
// Of skips _test.go on purpose — a test file can declare a struct of the same
// name in the same package and reflect over the built package would never see
// it — and this file IS one, so it goes in under another key. The key is only
// what go/parser reports positions against; nothing opens it.
func shapeSources(t *testing.T) map[string]string {
	t.Helper()
	src, err := os.ReadFile("themeleaves_test.go")
	if err != nil {
		t.Fatalf("this test reads its own source so that the declarations "+
			"reflect walks and the declarations go/parser walks are one text: %v",
			err)
	}
	return map[string]string{"shapes.go": string(src)}
}

// Every shape both mechanisms claim to handle, handled the same way.
func TestTheWalkAgreesWithReflectOnShapesCoreThemeDoesNotHave(t *testing.T) {
	sources := shapeSources(t)
	for _, c := range []struct {
		root string
		// A zero value of that same type, which is what reflect is given.
		value any
		// What the shape is here to ask, for the failure message.
		asks string
	}{
		{"shapeSiblings", shapeSiblings{},
			"three fields of one struct type are three subtrees, not a recursion"},
		{"shapePointer", shapePointer{},
			"a pointer to a struct is a leaf and the same struct by value is not"},
		{"shapeEmbedded", shapeEmbedded{},
			"an embedded value descends and loses its name; an embedded pointer " +
				"is a leaf named by its type"},
		{"shapeShared", shapeShared{},
			"one leaf name under two parents is counted once"},
		{"shapeAnonymous", shapeAnonymous{},
			"a struct type written in the field is descended into like a named one"},
		{"shapeEmpty", shapeEmpty{},
			"a field whose struct has no fields contributes nothing, not even " +
				"its own name"},
		{"shapeMultiName", shapeMultiName{},
			"two names on one field are two fields, descending or not"},
		{"shapeDeep", shapeDeep{},
			"three levels deep, the leaf is named by the last segment"},
		{"shapeLeafKinds", shapeLeafKinds{},
			"the flat case, which every other one is measured against"},
	} {
		exp := Of(sources, c.root)
		if !exp.Found {
			t.Errorf("Of did not find a struct named %s in this file's own "+
				"source, which declares one.", c.root)
			continue
		}
		want := reflectLeafNames(reflect.ValueOf(c.value))
		if slices.Equal(exp.Names, want) {
			continue
		}
		t.Errorf("%s expands to %v parsed and %v over reflect.\n\n"+
			"What this shape asks: %s.\n\n"+
			"Both readings are of the same declaration — reflect over the type "+
			"and go/parser over the text of this very file — so there is no "+
			"fixture between them to have gone stale. A name only the PARSE has "+
			"is a type it could not resolve to a struct declared here; a name "+
			"only REFLECT has is the children behind that stop, or a field "+
			"spelling the parse produced no name for at all.\n\n"+
			"only the parse has:  %v\nonly reflect has:    %v",
			c.root, exp.Names, want, c.asks,
			notIn(exp.Names, want), notIn(want, exp.Names))
	}
}

// And the three shapes where they do not agree, which is not a bug but the
// documented edge of what a parse can resolve.
//
// Asserted rather than left to a doc comment for the reason the group headings
// in wasm/verify's foldUnwordedCategories are: a stated limit that nothing
// re-derives is prose, and prose stays where it was written while the thing it
// describes moves. If a later go/types-backed resolution made any of these
// descend, this test is where that would be noticed — as a failure saying the
// walker got BETTER, which is a finding the table in internal/themehistory
// needs, because every historical row was taken by the walker that did not.
func TestTheWalkStopsWhereItSaysItStops(t *testing.T) {
	sources := shapeSources(t)
	for _, c := range []struct {
		root string
		// The field name the parse keeps because it stopped there.
		stopsAt string
		// A name reflect reaches by going in, which the parse therefore misses.
		behind string
		why    string
	}{
		{"shapeGeneric", "shapeBox", "Value",
			"an embedded generic instantiation is an *ast.IndexExpr, and " +
				"`shapeBox[int]` is not a key in the map of parsed structs"},
		{"shapeAliased", "Corner", "Radius",
			"an alias declaration's TypeSpec holds an identifier rather than a " +
				"struct type, so the alias never enters that map either"},
		{"shapeForeign", "Where", "Filename",
			"a struct from another package was never parsed — this walk reads " +
				"one directory, which is the premise the whole reading rests on"},
	} {
		exp := Of(sources, c.root)
		parsed, reflected := exp.Names, reflectLeafNames(reflect.ValueOf(
			shapeValueOf(t, c.root)))
		if !slices.Contains(parsed, c.stopsAt) {
			t.Errorf("%s expands to %v parsed, and %q is not in it.\n\n"+
				"This walk is supposed to STOP at that field and keep its own "+
				"name: %s. A parse that no longer keeps it either resolved the "+
				"type — in which case this list should hold %q instead, and the "+
				"asymmetry internal/themehistory's table was taken under has "+
				"changed — or it dropped the field silently, which is the one "+
				"direction that makes the population smaller than reflect's "+
				"without saying so.",
				c.root, parsed, c.stopsAt, c.why, c.behind)
		}
		if slices.Contains(parsed, c.behind) {
			t.Errorf("%s expands to %v parsed, which holds %q — a name only "+
				"reflect was supposed to reach, by descending where this walk "+
				"stops (%s).",
				c.root, parsed, c.behind, c.why)
		}
		if !slices.Contains(reflected, c.behind) {
			t.Errorf("reflect expands %s to %v, which does not hold %q.\n\n"+
				"That name is this case's whole subject: it is what lies behind "+
				"the stop, and without it the two mechanisms are not being shown "+
				"to differ, they are being shown to agree by accident. The shape "+
				"in this file has moved.", c.root, reflected, c.behind)
		}
		if slices.Contains(reflected, c.stopsAt) {
			t.Errorf("reflect expands %s to %v, which holds %q — the name this "+
				"walk keeps BECAUSE it stopped. reflect reaching it too means the "+
				"field stopped being a struct, and the case no longer separates "+
				"the two mechanisms.", c.root, reflected, c.stopsAt)
		}
	}
}

// shapeValueOf is a zero value of one of the three divergent shapes, by name.
//
// Written as a switch rather than carried in the table above so the table's
// rows say what they are about and nothing else; the alternative is an `any`
// column whose only job is to be type-asserted back.
func shapeValueOf(t *testing.T, root string) any {
	t.Helper()
	switch root {
	case "shapeGeneric":
		return shapeGeneric{}
	case "shapeAliased":
		return shapeAliased{}
	case "shapeForeign":
		return shapeForeign{}
	}
	t.Fatalf("no value for %s", root)
	return nil
}

// A struct that holds itself terminates.
//
// reflect never meets one — a value of such a type cannot be built, so
// affordedLeafNames could not produce this case if it wanted to — and that is
// exactly why the guard needs an arm of its own: the mechanism that would
// notice a missing guard is the one that cannot reach it. go/parser is happy
// to hand back either of these, and a walk without `seen` recurses until the
// stack goes.
//
// Both directions are here because they fail differently. The direct one is
// caught by the root's own entry in `seen`; the mutual one is only caught
// because `seen` is carried DOWN the descent rather than kept per struct.
//
// What the stop LEAVES is the field's own name, which is worth stating because
// it is not obviously the only choice — the field could as easily contribute
// nothing. It is the same rule every other stop in this walk follows: a field
// whose type does not resolve to a struct this walk may enter is a leaf named
// by the field. A self-reference is that, with "may" doing the work.
func TestTheWalkTerminatesOnAStructThatHoldsItself(t *testing.T) {
	for _, c := range []struct {
		what, src string
		root      string
		want      []string
	}{
		{"directly", `package p
type Knot struct {
	Self Knot
	Leaf int
}`, "Knot", []string{"Leaf", "Self"}},
		{"through another struct", `package p
type A struct {
	B    B
	AOwn int
}
type B struct {
	A    A
	BOwn int
}`, "A", []string{"A", "AOwn", "BOwn"}},
		{"through a sibling that also holds it", `package p
type Root struct {
	One  Mid
	Two  Mid
	Leaf int
}
type Mid struct {
	Back Root
	Deep int
}`, "Root", []string{"Back", "Deep", "Leaf"}},
	} {
		// If the guard is gone this does not fail, it does not return: the test
		// binary dies on a stack overflow and takes the package's other results
		// with it. There is no way to assert termination from inside the walk,
		// so what the arm asserts is the ANSWER a terminating walk gives.
		exp := Of(map[string]string{"p.go": c.src}, c.root)
		if !slices.Equal(exp.Names, c.want) {
			t.Errorf("a struct that holds itself %s expands to %v, and %v is what "+
				"a walk that stops at the second sighting produces.\n\n"+
				"The guard is the path from the root to the field being entered "+
				"(see walk's `next`), so a type already on that path is not "+
				"descended into again and the FIELD's name is kept — the same "+
				"thing this walk does at every other stop. A different answer "+
				"here is a guard that has become either too wide (siblings of one "+
				"type seeing each other's descent, which is the ninety-one-name "+
				"bug, and which the third row below would show as a missing "+
				"second `Back`) or too narrow, which does not fail: it hangs.",
				c.what, exp.Names, c.want)
		}
	}
}

// Of reads the files it says it reads, and says which they were.
//
// Files is not decoration. internal/themehistory filters a revision's paths
// with its own copy of the rule below — it drops test files before fetching
// them, because a `git cat-file` per test file per revision is time spent on
// text nobody parses — and two copies of one rule are two things that can move
// apart silently. The comparison lives in that package's own test; what this
// one holds is the half it compares against.
func TestOfReadsOnlyTheFilesItSaysItReads(t *testing.T) {
	const decl = `package p
type Theme struct {
	Colors Colors
	Own    int
}
type Colors struct{ Background int }`
	sources := map[string]string{
		"theme.go": decl,
		// A test file declaring the same struct differently. reflect over the
		// built package would never see this one, so neither may the parse.
		"theme_test.go": `package p
type Theme struct{ NotThis int }`,
		// Not Go at all.
		"theme.mjs":  `export const Theme = {};`,
		"README.md":  "# not a package",
		"theme.go.x": "trailing suffix, not a .go file",
	}
	exp := Of(sources, "Theme")
	if want := []string{"theme.go"}; !slices.Equal(exp.Files, want) {
		t.Errorf("Of read %v and %v is what its own rule selects.\n\n"+
			"A .go suffix and no _test.go, which is what internal/themehistory "+
			"filters a revision with too. Files is how those two filters are "+
			"held against each other at HEAD — see that package's "+
			"TestTheRevisionsFileSetIsTheOneTheWorkingTreeWalkReads — so a rule "+
			"that moved here without moving there is a table taken over a "+
			"population nothing else measures.", exp.Files, want)
	}
	if want := []string{"Background", "Own"}; !slices.Equal(exp.Names, want) {
		t.Errorf("Theme expands to %v and %v is what theme.go alone declares.\n\n"+
			"NotThis in the list is theme_test.go having been parsed: a test file "+
			"can declare a struct of the same name in the same package and "+
			"nothing built from this package would ever hold it.", exp.Names, want)
	}

	// A file go/parser returns nothing for is reported and not fatal, because
	// the two callers want opposite things from it — see Expansion. It is in
	// Files as well, since it is one of the files this reading was taken over.
	broken := map[string]string{
		"theme.go":  decl,
		"broken.go": "package p\nthis is not Go at all {{{",
	}
	exp = Of(broken, "Theme")
	if want := []string{"broken.go"}; !slices.Equal(exp.Unparsed, want) {
		t.Errorf("Of reports %v unparsed and %v is the file that does not "+
			"parse.\n\n"+
			"A revision caught mid-refactor is ordinary in a history and a "+
			"failure in a working tree, and the two callers decide that for "+
			"themselves — which they can only do if the fact comes back at all.",
			exp.Unparsed, want)
	}
	if !slices.Contains(exp.Files, "broken.go") {
		t.Errorf("Of read %v, which does not hold broken.go.\n\n"+
			"Unparsed is a subset of Files by construction: a file that failed "+
			"to parse was still one of the files this expansion was taken over, "+
			"and a Files that quietly drops it is a file-set comparison that "+
			"cannot see a revision losing one.", exp.Files)
	}
	if !exp.Found || !slices.Contains(exp.Names, "Background") {
		t.Errorf("with one file unparsed, Theme came back Found=%v with %v.\n\n"+
			"go/parser returns the declarations it did manage and this walk uses "+
			"them; the point of reporting Unparsed separately is that the names "+
			"are still worth having.", exp.Found, exp.Names)
	}

	// Found says whether the struct was declared, which is never the same
	// finding as its having no fields. A history walks revisions that predate
	// the type, and a row that reads "every leaf removed" is what an empty
	// expansion looks like from the diff.
	for _, c := range []struct {
		what  string
		src   string
		found bool
	}{
		{"a package that does not declare it", `package p
type Other struct{ A int }`, false},
		{"a declaration with no fields", `package p
type Theme struct{}`, true},
	} {
		exp := Of(map[string]string{"p.go": c.src}, "Theme")
		if exp.Found != c.found || len(exp.Names) != 0 {
			t.Errorf("%s gives Found=%v with %v, and Found=%v with no names is "+
				"the reading.\n\n"+
				"The two cases print the same empty population and are not the "+
				"same fact: one is a revision that predates the struct and the "+
				"other is a struct somebody emptied. internal/themehistory says "+
				"the first out loud on stderr so the row it produces is not read "+
				"as a fact about the type.",
				c.what, exp.Found, exp.Names, c.found)
		}
	}
}

// A name declared twice is resolved by SORT ORDER, and the resolution is
// reported rather than made silently.
//
// # Why this can happen at all
//
// Of keys every struct it parses by its bare name in one flat map, so a second
// declaration of a name overwrites the first and the winner is whichever path
// sorted last. Inside one compiling package that never arises — which was the
// reason it went unrecorded — and two callers hand Of file sets that are not
// one package:
//
//	a union probe   internal/themehistory parses core/ together with a
//	                subdirectory of it, on purpose, to find out whether the
//	                files both walks decline would move the population
//	a revision       the history walks every commit that touched core/, and a
//	                commit caught mid-refactor need not build
//
// # And the direction of it is the part worth pinning
//
// The two cases below are the same collision with the subdirectory renamed,
// and they resolve opposite ways:
//
//	core/sub/x.go   sorts BEFORE core/theme.go, so theme.go overwrites it and
//	                the collision moves nothing — invisible
//	core/zsub/x.go  sorts AFTER, so it takes the name and the expansion is
//	                over a struct from another package
//
// A recipe written to demonstrate the hazard used sub/ and reported that
// nothing happened. The hazard was real both times; only one spelling of it
// shows. Shadowed is what makes the other one visible.
func TestANameDeclaredTwiceIsResolvedBySortOrderAndSaidSoOutLoud(t *testing.T) {
	const top = `package p
type Theme struct{ Colors Colors }
type Colors struct{ Background int }`
	const elsewhere = `package q
type Colors struct{ Ghost int }`

	for _, c := range []struct {
		what  string
		other string
		want  []string // the leaf names, which say which declaration answered
		files []string // Shadowed[0].Files, in parse order
	}{
		{
			what:  "a subdirectory that sorts before the top level",
			other: "core/sub/x.go",
			want:  []string{"Background"},
			files: []string{"core/sub/x.go", "core/theme.go"},
		},
		{
			what:  "a subdirectory that sorts after it",
			other: "core/zsub/x.go",
			want:  []string{"Ghost"},
			files: []string{"core/theme.go", "core/zsub/x.go"},
		},
	} {
		exp := Of(map[string]string{
			"core/theme.go": top,
			c.other:         elsewhere,
		}, "Theme")

		if !slices.Equal(exp.Names, c.want) {
			t.Errorf("with %s declaring Colors too, Theme expands to %v and %v "+
				"is what the LAST path to declare it gives.\n\n"+
				"Of parses its paths in sorted order and a later declaration "+
				"overwrites an earlier one, so which of two Colors answers for "+
				"the field is a comparison between %q and \"core/theme.go\" and "+
				"nothing else. A different answer here is that rule having "+
				"changed, and every reading Of has ever taken over more than one "+
				"package was over the other declaration.",
				c.what, exp.Names, c.want, c.other)
		}

		want := []Shadow{{Name: "Colors", Files: c.files}}
		if !shadowsEqual(exp.Shadowed, want) {
			t.Errorf("with %s declaring Colors too, Of reports %v shadowed and "+
				"%v is the collision it resolved.\n\n"+
				"The expansion above is over one of the two declarations and "+
				"came back whole and plausible either way. This is the only "+
				"place that says a choice was made: internal/themehistory's "+
				"nested arm prints two possible causes for a moved population — "+
				"a leaf the subpackage really contributes, or a name it shadowed "+
				"— and without this field it cannot tell them apart.",
				c.what, exp.Shadowed, want)
		}
		// And that Files' ORDER is the finding rather than incidental. The
		// last entry is meant to be the declaration that survived, so the two
		// halves of each row are held against each other here instead of both
		// being read from the table: if the winner is the other package's
		// file, the leaf is the other package's leaf.
		if len(exp.Shadowed) == 1 {
			last := exp.Shadowed[0].Files[len(exp.Shadowed[0].Files)-1]
			wonByOther := slices.Equal(exp.Names, []string{"Ghost"})
			if (last == c.other) != wonByOther {
				t.Errorf("%s: Shadowed names %q as the last declaration and "+
					"Theme expands to %v.\n\n"+
					"Those two disagree about which Colors answered. Files is "+
					"documented as being in parse order with the winner last, "+
					"and a reader meeting this record in a failure message will "+
					"use it to decide which declaration to go and look at.",
					c.what, last, exp.Names)
			}
		}
	}

	// One compiling package, which is every revision in this repository's
	// history and the working tree. Nothing to report, and the field costs
	// those readings nothing.
	exp := Of(map[string]string{
		"core/theme.go":  top,
		"core/colors.go": "package p\ntype Other struct{ A int }",
	}, "Theme")
	if len(exp.Shadowed) != 0 {
		t.Errorf("a file set declaring each name once reports %v shadowed.\n\n"+
			"Shadowed is meant to be empty for anything that would compile, "+
			"which is what makes a non-empty one a finding rather than noise. "+
			"internal/themehistory prints it per revision on stderr the way it "+
			"prints Unparsed, and a field that fired on ordinary input would "+
			"put a line under all eighty-eight of them.", exp.Shadowed)
	}

	// A name declared twice in ONE file. go/parser reads it and the compiler
	// does not accept it, which is the mid-refactor revision this record is
	// partly for. The path is listed once per declaration rather than
	// deduplicated: the count is how many declarations there were.
	exp = Of(map[string]string{"p.go": `package p
type Theme struct{ Colors Colors }
type Colors struct{ Background int }
type Colors struct{ Ghost int }`}, "Theme")
	want := []Shadow{{Name: "Colors", Files: []string{"p.go", "p.go"}}}
	if !shadowsEqual(exp.Shadowed, want) {
		t.Errorf("one file declaring Colors twice reports %v and %v is the "+
			"reading.\n\n"+
			"Two declarations, so two entries under the one path. A shape that "+
			"does not build is exactly what a revision caught mid-refactor can "+
			"hold, and the history parses those rather than skipping them.",
			exp.Shadowed, want)
	}
}

// shadowsEqual compares two Shadowed lists, which slices.Equal cannot do: a
// Shadow holds a slice and is therefore not comparable.
func shadowsEqual(these, those []Shadow) bool {
	if len(these) != len(those) {
		return false
	}
	for i := range these {
		if these[i].Name != those[i].Name ||
			!slices.Equal(these[i].Files, those[i].Files) {
			return false
		}
	}
	return true
}

// notIn is the members of one list the other does not hold.
func notIn(these, those []string) []string {
	have := make(map[string]bool, len(those))
	for _, name := range those {
		have[name] = true
	}
	out := []string{}
	for _, name := range these {
		if !have[name] {
			out = append(out, name)
		}
	}
	return out
}
