package components

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

// Which themes can still show each palette rule doing something.
//
// # The problem this is for
//
// A rule with no witness is a rule an implementation could delete: the pixels
// would be identical and every other test in the repository would pass. Two
// rules were in that position under the bundled palettes, and their only
// evidence was midTonePrimaryTheme, a fixture:
//
//	inkOn's first step        reads the theme's declared pair before measuring.
//	OnLight's Primary arm     returns the role's ink-weight tone.
//
// AmberTheme was written to answer that, and it answers most of it. Its brand
// is a fill that cannot be ink (amber 700 is 2.04:1 on white), so its Primary
// role and its on-light tone are genuinely different colours; and its button
// declares MD brown 900 over that amber where measurement would pick the page's
// near-black, so deleting declaredInk moves pixels there. Two rows that rested
// on a fixture now rest on a shipped palette.
//
// # What no bundled theme can carry, and why that is arithmetic
//
// The *strong* form of the first rule — the declaration choosing the opposite
// ink pole from the measurement, which is the white-on-system-blue pairing
// inkOn was actually written for — is still the fixture's alone, and now
// provably so. A declaration that loses the measurement is by definition the
// lower-contrast of the theme's two inks, and the two ratios multiply to at
// most 21, so the loser can never exceed sqrt(21) = 4.58:1 — while
// TestVariantInkIsLegibleOnEveryThemeAndVariant asks 4.5:1 of it. The whole
// band a bundled witness could occupy is [4.50, 4.58].
// TestThePoleFlipBandIsTooNarrowToShip is that argument as a test, and it also
// reports which bundled themes could host such a fill at all (the answer turns
// out to depend on their inks being close to pure black and white).
//
// So the census records two rules where there used to be one, and the split is
// the honest shape: one is witnessed by a real theme, the other is witnessed by
// a fixture for a reason nobody can retint away.
//
// # What this census does
//
// It stops a witness disappearing without a sound. That failure is not
// hypothetical: the evidence for inkOn's first step *used* to be DefaultTheme,
// and it was lost to a palette improvement — Colors.Primary moved from iOS
// systemBlue to Apple's accessible blue, which was the right move and quietly
// cost one assertion its teeth (TestVariantDefaultKeepsTheThemePairing says so
// at the point where it now reaches for the fixture instead). The same move is
// available again on any of these rows, and the row is what would report it.
//
// So the table below is recorded per (rule, theme) rather than summarised.
// A retint that removes the last witness fails loudly; one that *adds* a
// witness also fails, because a rule that has become observable under a
// bundled theme is a rule whose fixture may no longer be load-bearing, and
// that is worth a human looking at rather than a silent pass.
//
// # It is read in both directions
//
// A (rule, theme) matrix has two axes and the edits that reach it come down
// each. A retint moves a *row*, which is what wantWitnesses records and
// TestEveryPaletteRuleStillHasAWitness checks. A new palette adds a *column*,
// and its two extremes — a theme that witnesses nothing and one that witnesses
// everything — produce row diffs that look alike and mean opposite things, so
// they are checked separately by TestEveryKnownThemeLandsSomewhereStated
// against two exception tables that each cost a sentence.

// paletteRule is one behaviour and the question "can this theme show it".
type paletteRule struct {
	name string
	// deletable says what an implementation could remove and still pass, for
	// any theme that does not witness the rule. It goes in the failure
	// message, because the whole point of a witness is what it rules out.
	deletable string
	witnesses func(*core.Theme) bool
}

var paletteRules = []paletteRule{
	{
		name: "the declaration outranks the measurement",
		deletable: "the declaredInk call at the top of inkOn — an inkOn that only " +
			"measured would agree with this theme everywhere",
		witnesses: func(t *core.Theme) bool {
			fill := t.Components.Button.Background
			declared := declaredInk(t, fill)
			if declared == "" {
				return false
			}
			// The measurement, run exactly as inkOn's fallback runs it.
			measured := contrastInk(fill, t.Colors.Background, t.Colors.TextPrimary)
			return !strings.EqualFold(declared, measured)
		},
	},
	{
		// The strong form, and the one that stayed with the fixture.
		//
		// The rule above is witnessed by any theme whose declared ink is not
		// the measurement's answer — AmberTheme's brand brown over its amber
		// fill, where the measurement would have picked the page's near-black.
		// That is a real witness and it is not the dramatic case: both answers
		// there are comfortably legible and one is merely the brand's.
		//
		// The dramatic case is the declaration choosing the *other pole* — a
		// light ink where measurement says dark, or the reverse — which is the
		// pairing inkOn was written for (white on iOS system blue, against a
		// measurement that picked black). It is separated out because it is the
		// half no bundled theme can carry, and TestThePoleFlipBandIsTooNarrow
		// ToShip says why that is arithmetic rather than an accident of these
		// three palettes.
		name: "the declaration flips the ink pole",
		deletable: "declaredInk's authority over a *legibility* disagreement — for " +
			"this theme the declared ink and the measured one are not opposite poles, " +
			"so nothing here shows the rule overriding the more contrasting choice",
		witnesses: func(t *core.Theme) bool {
			fill := t.Components.Button.Background
			declared := declaredInk(t, fill)
			if declared == "" {
				return false
			}
			measured := contrastInk(fill, t.Colors.Background, t.Colors.TextPrimary)
			if strings.EqualFold(declared, measured) {
				return false
			}
			// Both must be the theme's own two ink roles. A declaration that is
			// neither — a brand ink — differs from the measurement without
			// contradicting it.
			isPole := func(c string) bool {
				return strings.EqualFold(c, t.Colors.Background) ||
					strings.EqualFold(c, t.Colors.TextPrimary)
			}
			return isPole(declared) && isPole(measured)
		},
	},
	// One row per toned role. Written out rather than looped, because the
	// four are independent: Primary is the one with no bundled witness, and
	// a row that quietly merged the four would report "three of four" and
	// hide exactly the fact this census exists for.
	onLightRule("Primary", func(c core.ColorPalette) string { return c.Primary }),
	onLightRule("Error", func(c core.ColorPalette) string { return c.Error }),
	onLightRule("Success", func(c core.ColorPalette) string { return c.SuccessColor() }),
	onLightRule("Warning", func(c core.ColorPalette) string { return c.WarningColor() }),
}

// onLightRule builds the "the reverse lookup moves this role's colour" rule.
//
// Asked through ColorPalette.OnLight rather than through the role's own
// resolver, because OnLight is the arm that can be deleted: a widget holding
// a hex and no name for it is the position the reverse lookup exists for (see
// chipAccent, which reads the theme's Button base and so has a colour rather
// than a role).
func onLightRule(role string, get func(core.ColorPalette) string) paletteRule {
	return paletteRule{
		name: "OnLight moves the " + role + " role",
		deletable: "the " + role + " arm of ColorPalette.OnLight — for this theme the " +
			"lookup is an identity and returning the colour unchanged would agree",
		witnesses: func(t *core.Theme) bool {
			c := get(t.Colors)
			return c != "" && !strings.EqualFold(t.Colors.OnLight(c), c)
		},
	}
}

// knownTheme is a palette this package can put a rule to, and whether it is
// one a real app can install.
//
// The flag is the census's load-bearing distinction — evidence from a shipped
// theme and evidence from a test fixture are worth different things — and it
// is derived rather than remembered: core.BundledThemes() is the authority, so
// a fixture promoted to a shipped palette, or a palette withdrawn, moves this
// bit with nobody editing a list.
type knownTheme struct {
	theme   *core.Theme
	bundled bool
}

// knownThemes is every theme this package can measure a rule against: the
// bundled palettes and the fixture that carries what they have lost.
//
// midTonePrimaryTheme is in here on equal footing deliberately. It is not a
// bundled theme and the census's whole subject is that this matters — but a
// list that excluded it would report "no witness" for a rule that does in
// fact have one, and the distinction that matters is recorded below, in the
// rests column, where it is derived from this flag rather than from a reader
// knowing which of the names is the fixture.
func knownThemes() map[string]knownTheme {
	// The bundled ones come from core's own census rather than being listed
	// again here: a fourth palette that this file never asked the questions of
	// would not fail anything, it would simply be absent — which is the exact
	// shape of miss this census exists to report one level up.
	themes := map[string]knownTheme{"midTonePrimary": {theme: midTonePrimaryTheme()}}
	for name, theme := range core.BundledThemes() {
		themes[name] = knownTheme{theme: theme, bundled: true}
	}
	return themes
}

// restsOn is what a row's witness list amounts to, in the only three states a
// list of themes can be in.
//
// Stated in the table and derived by the test, which is the whole point: the
// column is a claim about the row that the row itself can falsify. Before it
// existed this file's own prose said two rules rested on the fixture alone
// while the table beside it listed AmberTheme for both — true when it was
// written, wrong the moment a palette was added, and nothing could tell.
type restsOn string

const (
	// A shipped palette shows the rule. Nothing is owed: an app can be pointed
	// at a theme where the behaviour is visible.
	restsOnBundled restsOn = "a bundled theme"
	// Only the fixture shows it. Legal and sometimes unavoidable — see the
	// pole flip — and it needs an argument, because a fixture is a thing
	// somebody edits while tidying a test helper.
	restsOnFixture restsOn = "the fixture alone"
	// Nothing shows it. The rule is one an implementation could delete with
	// every test in the repository still passing, so this needs an argument
	// saying why it is being kept anyway rather than restored to a witness.
	restsOnNothing restsOn = "nothing"
)

// restsOnWitnesses derives the column above from a witness list.
//
// Bundled beats fixture: a row with both is a row whose evidence survives the
// fixture being deleted, which is the question the column is asked.
func restsOnWitnesses(names []string, themes map[string]knownTheme) restsOn {
	if len(names) == 0 {
		return restsOnNothing
	}
	for _, n := range names {
		if themes[n].bundled {
			return restsOnBundled
		}
	}
	return restsOnFixture
}

// witnessRow is one rule's evidence.
type witnessRow struct {
	rule   string
	themes []string // sorted; the themes that can currently show the rule
	rests  restsOn  // what that list amounts to — derived and compared
	// why is required when rests is not restsOnBundled. Same rule
	// internal/palette's backdrop exclusions follow: an entry that only
	// records a state is a state nobody argued for, and evidence going quiet
	// is this file's entire subject.
	why string
}

// wantWitnesses is the census: for each rule, the themes that can currently
// show it, and what that list is worth.
//
// # Adding a theme
//
// A new palette is asked every rule the moment core.BundledThemes() returns
// it, and its answers arrive here as a diff. Both extremes are legal and they
// mean opposite things, which is why neither is silent:
//
//	in no row     the palette witnesses nothing. Legal — most themes carry a
//	              few rules — but a palette that shows none of them is one the
//	              census cannot use, so it is named in quietThemes with a
//	              reason.
//	in every row  the palette witnesses everything, which means every
//	              fixture-only row has just gained a bundled witness and those
//	              fixtures may have stopped being load-bearing. Named in
//	              universalThemes with a reason, and the fixtures re-read.
//
// Anything between the two is the ordinary case: add the name to the rows it
// witnesses, and let the rests column be re-derived. A row moving from
// "the fixture alone" to "a bundled theme" is the good news this file exists
// to notice — and it is also the moment to check whether the fixture is still
// carrying anything.
var wantWitnesses = []witnessRow{
	{
		rule:   "the declaration outranks the measurement",
		themes: []string{"AmberTheme", "midTonePrimary"},
		rests:  restsOnBundled,
	},
	{
		rule:   "the declaration flips the ink pole",
		themes: []string{"midTonePrimary"},
		rests:  restsOnFixture,
		why: "no bundled palette can host one: a declaration that loses the " +
			"measurement is the lower-contrast ink, and the AA floor leaves it a band " +
			"1.8% wide — see TestThePoleFlipBandIsTooNarrowToShip",
	},
	{
		rule:   "OnLight moves the Primary role",
		themes: []string{"AmberTheme", "midTonePrimary"},
		rests:  restsOnBundled,
	},
	{
		rule:   "OnLight moves the Error role",
		themes: []string{"DefaultTheme", "midTonePrimary"},
		rests:  restsOnBundled,
	},
	{
		rule:   "OnLight moves the Success role",
		themes: []string{"DefaultTheme"},
		rests:  restsOnBundled,
	},
	{
		rule:   "OnLight moves the Warning role",
		themes: []string{"AmberTheme", "DefaultTheme", "MaterialTheme"},
		rests:  restsOnBundled,
	},
}

// The two degenerate columns of the matrix, as data with arguments rather
// than as states the census refuses to describe.
//
// Both are empty today and that is the honest starting point: every theme the
// package knows about witnesses some rules and not others. What an entry costs
// is a sentence saying why the extreme is acceptable, which is the same price
// internal/palette charges a backdrop exclusion — and the reason the price is
// worth paying is that both extremes look, in a diff, exactly like a theme
// somebody forgot to finish adding.
//
// Empty also means the arms that read them cannot run against these palettes,
// which is why the classification is a function (landingComplaints) rather than
// a run of t.Errorf calls: TestTheLandingArmsEachReportTheirCase drives every
// arm over constructed theme sets, so the message the first person to add a
// quiet palette gets is one somebody has already read.
var (
	// theme -> why it is allowed to witness no rule at all.
	quietThemes = map[string]string{}
	// theme -> why it is allowed to witness every rule.
	universalThemes = map[string]string{}
)

func TestEveryPaletteRuleStillHasAWitness(t *testing.T) {
	themes := knownThemes()

	if len(paletteRules) != len(wantWitnesses) {
		t.Fatalf("%d rules and %d recorded rows — every rule needs a row, so that adding "+
			"one forces somebody to work out which themes can show it",
			len(paletteRules), len(wantWitnesses))
	}

	rows := map[string]witnessRow{}
	for _, r := range wantWitnesses {
		if _, dup := rows[r.rule]; dup {
			t.Fatalf("two rows named %q — the count above would still agree while one "+
				"rule went unrecorded", r.rule)
		}
		rows[r.rule] = r
	}

	for _, rule := range paletteRules {
		var got []string
		for name, kt := range themes {
			if rule.witnesses(kt.theme) {
				got = append(got, name)
			}
		}
		sort.Strings(got)

		row, recorded := rows[rule.name]
		if !recorded {
			t.Errorf("rule %q has no row in wantWitnesses — record the themes that can "+
				"show it, with the rests column that list implies", rule.name)
			continue
		}

		if strings.Join(got, ",") != strings.Join(row.themes, ",") {
			gained, lost := diffThemes(got, row.themes)
			t.Errorf("rule %q is witnessed by %v, recorded as %v (gained %v, lost %v) — "+
				"if a theme was just added to core.BundledThemes(), put its name in the "+
				"rows it witnesses and see wantWitnesses' \"Adding a theme\" note; if a "+
				"witness was lost, %s", rule.name, got, row.themes, gained, lost,
				rule.deletable)
		}

		// The column, derived. A stale rests is the failure this census had
		// before the column existed: the list said one thing and the prose
		// above it said another.
		if want := restsOnWitnesses(got, themes); want != row.rests {
			t.Errorf("rule %q now rests on %s, recorded as %s — %s", rule.name,
				want, row.rests, restsAdvice(want, row.rests, rule))
		}
		if row.rests != restsOnBundled && row.why == "" {
			t.Errorf("rule %q rests on %s and gives no reason. A row that does not rest "+
				"on a shipped palette is a row somebody has to defend — say why the "+
				"evidence is where it is, as the pole-flip row does", rule.name, row.rests)
		}
	}
}

// restsAdvice says what a moved rests column means, which is a different
// thing in each direction and is the reason the column is worth having.
func restsAdvice(got, recorded restsOn, rule paletteRule) string {
	switch {
	case got == restsOnNothing:
		return "nothing can show it working, which means " + rule.deletable +
			". Either a theme moved and should move back, or the fixture that carried " +
			"it has been edited — see midTonePrimaryTheme"
	case got == restsOnBundled && recorded == restsOnFixture:
		return "a shipped palette now shows it, which is good news and is also the " +
			"moment to check whether the fixture is still carrying anything — see " +
			"TestTheFixtureStillCarriesWhatNoBundledThemeCan"
	case got == restsOnFixture && recorded == restsOnBundled:
		return "the shipped palette that used to show it no longer does, so the rule's " +
			"only evidence is now a test fixture. That is the drift this file was " +
			"written for: " + rule.deletable
	default:
		return "update the column, and check the reason beside it still describes the row"
	}
}

// diffThemes reports what a row gained and lost, so the failure message can
// tell "a theme was added to core" apart from "a palette was retinted" —
// which are the two edits that reach this test and want opposite responses.
func diffThemes(got, want []string) (gained, lost []string) {
	in := func(names []string, n string) bool {
		for _, x := range names {
			if x == n {
				return true
			}
		}
		return false
	}
	for _, g := range got {
		if !in(want, g) {
			gained = append(gained, g)
		}
	}
	for _, w := range want {
		if !in(got, w) {
			lost = append(lost, w)
		}
	}
	return gained, lost
}

// Where a newly added theme lands, read down the matrix's columns rather than
// across its rows.
//
// The row test above catches a palette that changed what it witnesses. It
// cannot catch the shape of a theme, because a theme that witnesses nothing
// and one that witnesses everything both produce ordinary-looking row diffs —
// a handful of names appearing, or none appearing at all — and the two mean
// opposite things about the census as a whole.
func TestEveryKnownThemeLandsSomewhereStated(t *testing.T) {
	themes := knownThemes()

	// A census over no rules classifies every theme as quiet, which is a
	// sentence about the rules rather than about the palettes. Checked before
	// the counts, because with paletteRules emptied every complaint below
	// would be true and none of them would be the finding.
	if len(paletteRules) == 0 {
		t.Fatal("there are no palette rules — every theme witnesses all zero of them " +
			"and none of them, and the two exception tables would both be right")
	}

	carried := make(map[string][]string, len(themes))
	for name, kt := range themes {
		for _, rule := range paletteRules {
			if rule.witnesses(kt.theme) {
				carried[name] = append(carried[name], rule.name)
			}
		}
		// Present with a nil slice for a theme that carries nothing: the map's
		// keys are the theme list landingComplaints holds the tables to, and a
		// quiet theme that was simply absent from it would be indistinguishable
		// from one this package does not measure.
		if _, ok := carried[name]; !ok {
			carried[name] = nil
		}
	}

	for _, c := range landingComplaints(carried, len(paletteRules), quietThemes, universalThemes) {
		t.Error(c)
	}
}

// landingComplaints is the whole of what the census says about a *theme*, as
// data rather than as t.Errorf calls.
//
// # Why it is a function
//
// Both exception tables are empty and have been since the day they were
// written, which means five of the six arms below have never run — a table with
// no entries cannot produce an entry naming a theme that does not exist, and a
// package where every palette witnesses some rules and not others never reaches
// either degenerate case. That is the ordinary state and it is also the state
// in which an arm can be wrong for as long as it likes: the first person to add
// a quiet palette would be the first person to find out whether the check that
// was supposed to greet them works.
//
// Taking the counts and the two tables as parameters is what makes those arms
// reachable without inventing an entry. The census keeps its real, empty tables;
// TestTheLandingArmsEachReportTheirCase drives this function with the theme sets
// that would produce each one. The same split internal/valuefixture's own tests
// make for internal/valuefixture: exercise the classifier over constructed
// inputs, and let the real data stay whatever it happens to be.
//
// carried is theme name → the rules it witnesses, and its keys are the theme
// list — so a table entry naming something not in it is an entry arguing for
// nothing. rules is how many rules there are in total, which is what "every
// rule" means; it is passed rather than derived from carried because a theme
// witnessing all of them and a census with only those rules are different
// facts.
func landingComplaints(carried map[string][]string, rules int, quiet, universal map[string]string) []string {
	var out []string

	// Both exception tables are held to the theme list in the other direction,
	// for the reason internal/palette holds its backdrop exclusions to
	// core.ComponentDefaults: an entry naming a theme that no longer exists is
	// an argument for nothing, and it would sit here looking like coverage.
	//
	// Sorted, so a failure reports the same order every run: a map's range is
	// randomized and a two-entry complaint list that reshuffled between runs
	// would read as a flapping test.
	for _, table := range []struct {
		name    string
		entries map[string]string
	}{
		{"quietThemes", quiet},
		{"universalThemes", universal},
	} {
		for _, name := range sortedKeys(table.entries) {
			if _, known := carried[name]; !known {
				out = append(out, fmt.Sprintf("%s names %q, which is not a theme this "+
					"package measures — the entry argues for nothing", table.name, name))
			}
			if table.entries[name] == "" {
				out = append(out, fmt.Sprintf("%s[%q] gives no reason; the reason is "+
					"the entry", table.name, name))
			}
		}
	}

	for _, name := range sortedKeys(carried) {
		got := carried[name]
		switch {
		case len(got) == 0:
			if _, allowed := quiet[name]; !allowed {
				out = append(out, fmt.Sprintf("%s witnesses no palette rule at all. That "+
					"is legal and it is not nothing: a palette the census cannot use adds "+
					"no evidence to any row, so it cannot be the answer when a witness is "+
					"lost. If it is a theme being added, record it in quietThemes with the "+
					"reason its palette shows none of these rules", name))
			}
		case len(got) == rules:
			if _, allowed := universal[name]; !allowed {
				out = append(out, fmt.Sprintf("%s witnesses every palette rule, including "+
					"the ones that rested on the fixture. Check whether midTonePrimaryTheme "+
					"is still carrying anything before recording it in universalThemes with "+
					"a reason — see TestTheFixtureStillCarriesWhatNoBundledThemeCan", name))
			}
		default:
			if _, q := quiet[name]; q {
				out = append(out, fmt.Sprintf("%s is recorded in quietThemes and witnesses "+
					"%d rules %v", name, len(got), got))
			}
			if _, a := universal[name]; a {
				out = append(out, fmt.Sprintf("%s is recorded in universalThemes and "+
					"witnesses %d of %d rules", name, len(got), rules))
			}
		}
	}
	return out
}

// sortedKeys is the deterministic order a complaint list is built in.
func sortedKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// Each arm of the landing check, driven over a theme set that reaches it.
//
// # What this is for
//
// quietThemes and universalThemes are both empty, and the honest reading of
// that is not "the arms are fine" — it is that the arms have never run. Five
// of the six have no way to run against the real palettes at all: every theme
// core bundles witnesses some rules and not others, so the two degenerate
// columns are unreachable, and an empty map cannot hold a stale name or a
// missing reason.
//
// The first person to meet those arms would be somebody adding a palette on a
// day when they were doing something else, and the message they get is the
// whole value of the check — it is what says which of quietThemes and
// universalThemes to write in, and why an entry costs a sentence. So the arms
// are exercised here, over constructed inputs, and the real tables stay empty.
//
// The pairs are deliberate: every case that must complain is written beside the
// one that must not, because "the check fires" and "the exception silences it"
// are two claims and a table that only made the first would pass with the
// exception lookups deleted.
func TestTheLandingArmsEachReportTheirCase(t *testing.T) {
	const rules = 3
	all := []string{"a", "b", "c"}
	some := []string{"a"}
	reason := map[string]string{"Palette": "because"}

	cases := []struct {
		name             string
		carried          map[string][]string
		quiet, universal map[string]string
		want             string // a substring of the one complaint expected, or "" for silence
	}{
		{
			name:    "a quiet palette nobody recorded",
			carried: map[string][]string{"Palette": nil},
			want:    "witnesses no palette rule at all",
		},
		{
			name:    "a quiet palette with a reason",
			carried: map[string][]string{"Palette": nil},
			quiet:   reason,
		},
		{
			name:    "a universal palette nobody recorded",
			carried: map[string][]string{"Palette": all},
			want:    "witnesses every palette rule",
		},
		{
			name:      "a universal palette with a reason",
			carried:   map[string][]string{"Palette": all},
			universal: reason,
		},
		{
			name:    "an ordinary palette is neither, and is recorded as quiet",
			carried: map[string][]string{"Palette": some},
			quiet:   reason,
			want:    "is recorded in quietThemes and witnesses 1 rules",
		},
		{
			name:      "an ordinary palette recorded as universal",
			carried:   map[string][]string{"Palette": some},
			universal: reason,
			want:      "is recorded in universalThemes and witnesses 1 of 3 rules",
		},
		{
			name:    "an ordinary palette in neither table",
			carried: map[string][]string{"Palette": some},
		},
		{
			// The direction that has nothing to do with counts: an entry for a
			// theme that is not measured. It is what a retired palette leaves
			// behind, and it reads in a diff exactly like coverage.
			name:    "an entry naming a theme that is gone",
			carried: map[string][]string{"Palette": some},
			quiet:   map[string]string{"Retired": "because"},
			want:    `quietThemes names "Retired"`,
		},
		{
			name:    "an entry with no reason",
			carried: map[string][]string{"Palette": nil},
			quiet:   map[string]string{"Palette": ""},
			want:    "gives no reason",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := landingComplaints(c.carried, rules, c.quiet, c.universal)
			if c.want == "" {
				if len(got) != 0 {
					t.Errorf("complained about a case that is in order: %v", got)
				}
				return
			}
			if len(got) != 1 {
				t.Fatalf("want one complaint containing %q, got %d: %v", c.want, len(got), got)
			}
			if !strings.Contains(got[0], c.want) {
				t.Errorf("complaint is %q, want one containing %q", got[0], c.want)
			}
		})
	}
}

// A theme carrying every rule is universal and a theme carrying none is quiet,
// and the two are the same theme when there are no rules.
//
// That is why TestEveryKnownThemeLandsSomewhereStated refuses an empty
// paletteRules before it counts anything: the switch below takes the quiet arm
// first, so a census whose rules had all been deleted would report every
// palette as quiet — a true statement about a table that no longer says
// anything, offered as a finding about the palettes.
func TestAnEmptyCensusClassifiesEveryPaletteAsQuiet(t *testing.T) {
	got := landingComplaints(map[string][]string{"Palette": nil}, 0, nil, nil)
	if len(got) != 1 || !strings.Contains(got[0], "witnesses no palette rule at all") {
		t.Fatalf("want the quiet complaint, got %v", got)
	}
	// And recording it in universalThemes — which is equally true of a theme
	// that carries all zero rules — does not silence it. The order of the arms
	// is the thing being pinned: it is what makes the guard above necessary
	// rather than decorative.
	got = landingComplaints(map[string][]string{"Palette": nil}, 0,
		nil, map[string]string{"Palette": "carries all zero of them"})
	if len(got) != 1 || !strings.Contains(got[0], "witnesses no palette rule at all") {
		t.Fatalf("the quiet arm did not win over the universal one: %v", got)
	}
}

// The fixture is the thread, so it is checked as one.
//
// midTonePrimaryTheme now carries exactly one rule alone — "the declaration
// flips the ink pole" — and that is the rule to hold it to. A fixture that
// stopped witnessing it would leave the strong form of inkOn's first step with
// no evidence in the repository at all, and it would do so by an edit that
// looks like tidying a test helper rather than like changing behaviour.
func TestTheFixtureStillCarriesWhatNoBundledThemeCan(t *testing.T) {
	const soleRule = "the declaration flips the ink pole"

	rule := paletteRuleNamed(t, soleRule)
	if !rule.witnesses(midTonePrimaryTheme()) {
		t.Errorf("midTonePrimaryTheme no longer witnesses %q: %s. It is the only theme in "+
			"the repository that can — see TestThePoleFlipBandIsTooNarrowToShip for why "+
			"a bundled one cannot — so this rule now has no evidence at all", soleRule,
			rule.deletable)
	}

	// And no bundled theme does, which is the other half of why the fixture
	// exists. A bundled witness here would be good news and is still a failure:
	// it means a palette has been tuned into a band 1.8% wide at the AA floor,
	// which is a decision somebody should have to defend rather than one that
	// arrives with a retint.
	for name, theme := range core.BundledThemes() {
		if rule.witnesses(theme) {
			t.Errorf("%s now witnesses %q — its button declares one of its own two ink "+
				"roles and the other measures higher. That puts its label between 4.50:1 "+
				"and 4.58:1 (see TestThePoleFlipBandIsTooNarrowToShip), and it makes this "+
				"file's opening, inkOn's \"what that leaves here\" section and "+
				"midTonePrimaryTheme's doc wrong. Update them and wantWitnesses",
				name, rule.name)
		}
	}
}

// paletteRuleNamed looks a rule up by name, failing rather than returning nil:
// a renamed rule would otherwise turn a guard into a no-op.
func paletteRuleNamed(t *testing.T, name string) *paletteRule {
	t.Helper()
	for i := range paletteRules {
		if paletteRules[i].name == name {
			return &paletteRules[i]
		}
	}
	t.Fatalf("no rule named %q — a test names it and it has been renamed or removed", name)
	return nil
}

// Why the pole flip cannot be shipped in a bundled palette, as arithmetic.
//
// # The squeeze
//
// contrastInk returns whichever of the theme's two ink roles has the higher
// ratio against the fill. So a *declaration* that disagrees with it is, by
// construction, the lower-contrast of the two. Write the two ratios against a
// fill of luminance L, with ink roles at luminance Lb (light) and Lt (dark):
//
//	light ratio  =  (Lb + 0.05) / (L  + 0.05)
//	dark  ratio  =  (L  + 0.05) / (Lt + 0.05)
//	product      =  (Lb + 0.05) / (Lt + 0.05)      — independent of the fill
//
// The product is a constant of the *palette*, and for pure white over pure
// black it is 21. Two numbers whose product is fixed and whose smaller one we
// are constraining: the loser is at most the geometric mean, sqrt(21) = 4.5826.
// Meanwhile TestVariantInkIsLegibleOnEveryThemeAndVariant asks 4.5:1 of every
// variant's ink, VariantDefault included. So a bundled theme witnessing the
// pole flip would have to place its button label inside [4.50, 4.58] — 1.8% of
// a scale that runs to 21, at its very bottom.
//
// # And it needs near-pure inks to have a band at all
//
// Feasibility is 4.5 * (Lt + 0.05) < sqrt((Lb + 0.05)(Lt + 0.05)), i.e.
// (Lt + 0.05) < (Lb + 0.05) / 20.25. Against white that is Lt < 0.0019 —
// darker than #101010. A palette using a *soft* near-black, which is the modern
// default (MD3's on-surface, Material's #212121), has no feasible fill at all:
// its ratios cannot spread far enough apart for the loser to reach 4.5.
//
// Both halves are asserted below by scanning rather than by restating the
// algebra, so this fails if contrastInk's choice rule or the AA floor ever
// changes rather than if somebody's derivation was wrong.
func TestThePoleFlipBandIsTooNarrowToShip(t *testing.T) {
	const wcagAA = 4.5
	geometricMean := math.Sqrt(21) // pure white over pure black

	// Scan the whole luminance range at a step far finer than a hex step.
	// A counterexample is what would matter, so the assertion is over every
	// point rather than at the endpoints.
	widest := 0.0
	found := false
	for i := 0; i <= 200000; i++ {
		L := float64(i) / 200000
		light := contrastRatio(L, 1.0) // white
		dark := contrastRatio(L, 0.0)  // black
		lo := math.Min(light, dark)
		if lo < wcagAA {
			continue
		}
		found = true
		if lo > widest {
			widest = lo
		}
		if lo > geometricMean+1e-9 {
			t.Fatalf("a fill at luminance %.6f gives the losing ink %.4f:1, above the "+
				"geometric mean %.4f:1 — the product of the two ratios is not the "+
				"constant this argument rests on", L, lo, geometricMean)
		}
	}
	if !found {
		t.Fatal("no fill lets the losing ink clear AA at all, even between pure black and " +
			"white — the scan or contrastRatio has changed under this argument")
	}
	if widest < geometricMean-0.01 {
		t.Errorf("the widest reachable losing ratio is %.4f:1, not the geometric mean "+
			"%.4f:1 — the band is narrower than the argument says and the numbers in "+
			"this file's prose are wrong", widest, geometricMean)
	}

	// The band's width, stated as the thing a palette author would have to hit.
	if band := (widest - wcagAA) / wcagAA; band > 0.02 {
		t.Errorf("the witness band is %.1f%% of the floor, not the ~1.8%% this file and "+
			"AmberTheme's doc both quote", band*100)
	}

	// And per bundled theme: can a fill exist at all under its own two inks?
	// This is the concrete reason none of them is a witness, and it is a fact
	// about their *inks* rather than about their brand colours.
	for name, theme := range core.BundledThemes() {
		lb, ok := relativeLuminance(theme.Colors.Background)
		if !ok {
			t.Errorf("%s: Background %q does not parse", name, theme.Colors.Background)
			continue
		}
		lt, ok := relativeLuminance(theme.Colors.TextPrimary)
		if !ok {
			t.Errorf("%s: TextPrimary %q does not parse", name, theme.Colors.TextPrimary)
			continue
		}
		feasible := (lt + 0.05) < (lb+0.05)/(wcagAA*wcagAA)
		reachable := false
		for i := 0; i <= 200000 && !reachable; i++ {
			L := float64(i) / 200000
			lo := math.Min(contrastRatio(L, lb), contrastRatio(L, lt))
			reachable = lo >= wcagAA
		}
		if feasible != reachable {
			t.Errorf("%s: the closed-form feasibility test says %v and the scan says %v — "+
				"the algebra in this test's doc does not describe what contrastRatio does",
				name, feasible, reachable)
		}
		t.Logf("%s: a pole-flipping fill is %s under its inks %q/%q",
			name, map[bool]string{true: "reachable", false: "unreachable"}[reachable],
			theme.Colors.Background, theme.Colors.TextPrimary)
	}
}
