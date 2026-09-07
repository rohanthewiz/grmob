package components

import (
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

// knownThemes is every theme this package can measure a rule against: the two
// bundled palettes and the fixture that carries what they have lost.
//
// midTonePrimaryTheme is in here on equal footing deliberately. It is not a
// bundled theme and the census's whole subject is that this matters — but a
// list that excluded it would report "no witness" for two rules that do in
// fact have one, and the distinction that matters is recorded below, in the
// expected table, where a reader can see which rows rest on the fixture alone.
func knownThemes() map[string]*core.Theme {
	themes := map[string]*core.Theme{"midTonePrimary": midTonePrimaryTheme()}
	// The bundled ones come from core's own census rather than being listed
	// again here: a fourth palette that this file never asked the questions of
	// would not fail anything, it would simply be absent — which is the exact
	// shape of miss this census exists to report one level up.
	for name, theme := range core.BundledThemes() {
		themes[name] = theme
	}
	return themes
}

// wantWitnesses is the census: for each rule, the themes that can currently
// show it, sorted.
//
// Two rows rest on the fixture alone — "the declaration outranks the
// measurement" and "OnLight moves the Primary role" — and those two are the
// Next-list item this file was written for. The other three are witnessed by
// a bundled theme and need nothing.
var wantWitnesses = map[string][]string{
	"the declaration outranks the measurement": {"AmberTheme", "midTonePrimary"},
	"the declaration flips the ink pole":       {"midTonePrimary"},
	"OnLight moves the Primary role":           {"AmberTheme", "midTonePrimary"},
	"OnLight moves the Error role":             {"DefaultTheme", "midTonePrimary"},
	"OnLight moves the Success role":           {"DefaultTheme"},
	"OnLight moves the Warning role":           {"AmberTheme", "DefaultTheme", "MaterialTheme"},
}

func TestEveryPaletteRuleStillHasAWitness(t *testing.T) {
	themes := knownThemes()

	if len(paletteRules) != len(wantWitnesses) {
		t.Fatalf("%d rules and %d recorded rows — every rule needs a row, so that adding "+
			"one forces somebody to work out which themes can show it",
			len(paletteRules), len(wantWitnesses))
	}

	for _, rule := range paletteRules {
		var got []string
		for name, theme := range themes {
			if rule.witnesses(theme) {
				got = append(got, name)
			}
		}
		sort.Strings(got)

		want, recorded := wantWitnesses[rule.name]
		if !recorded {
			t.Errorf("rule %q has no row in wantWitnesses — record the themes that can "+
				"show it, or none if it has become unobservable", rule.name)
			continue
		}

		if len(got) == 0 {
			t.Errorf("rule %q now has no witness in any known theme. Nothing can show it "+
				"working, which means %s. Either a theme moved and should move back, or "+
				"the fixture that carried it has been edited — see midTonePrimaryTheme",
				rule.name, rule.deletable)
			continue
		}
		if strings.Join(got, ",") != strings.Join(want, ",") {
			t.Errorf("rule %q is witnessed by %v, recorded as %v — a palette or fixture "+
				"edit has changed what this rule's evidence rests on. If a witness was "+
				"lost, check whether the remaining ones are fixtures; if one was gained, "+
				"the fixture carrying this rule may no longer be load-bearing. Then "+
				"update wantWitnesses", rule.name, got, want)
		}
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
