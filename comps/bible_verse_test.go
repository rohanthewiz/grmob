package comps

import (
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

var john3 = []BibleVerseLine{
	{Number: 16, Text: "For God so loved the world."},
	{Number: 17, Text: "For God sent not his Son into the world to condemn the world."},
}

// paragraphText joins a Paragraph's runs back into the string a reader hears.
func paragraphText(n *core.Node) string {
	var b strings.Builder
	for _, r := range n.Props["runs"].([]map[string]any) {
		b.WriteString(r["t"].(string))
	}
	return b.String()
}

func findType(n *core.Node, typ string) *core.Node {
	return findFirst(n, func(x *core.Node) bool { return x.Type == typ })
}

// Several verses: one Paragraph, a number run before each verse, the
// reference and translation under it, and the link back.
func TestBibleVerseNumbersAMultiVersePassage(t *testing.T) {
	_, n := renderDebug(t, BibleVerse{
		Reference: "John 3:16-17", Translation: "KJV", Verses: john3,
		URL: "https://www.blueletterbible.org/kjv/jhn/3/16/",
	})
	if n.Type != "Card" {
		t.Errorf("root = %s, want Card", n.Type)
	}
	p := findType(n, "Paragraph")
	if p == nil {
		t.Fatal("no Paragraph")
	}
	want := "16 For God so loved the world. 17 For God sent not his Son into the world to condemn the world."
	if got := paragraphText(p); got != want {
		t.Errorf("passage = %q\nwant      %q", got, want)
	}
	if findText(n, "John 3:16-17 (KJV)") == nil {
		t.Error("the reference line should read \"John 3:16-17 (KJV)\"")
	}
	link := findFirst(n, func(x *core.Node) bool { return x.Style.AccessibilityRole == core.RoleLink })
	if link == nil || link.Style.AccessibilityLabel != "Read on Blue Letter Bible" {
		t.Fatal("the link back to Blue Letter Bible is missing")
	}
}

// One verse needs no number: the reference already names it. HideNumbers
// drops them from a longer passage too.
func TestBibleVerseNumbersOnlyWhenTheyHelp(t *testing.T) {
	_, n := renderDebug(t, BibleVerse{Reference: "John 3:16", Verses: john3[:1]})
	if got := paragraphText(findType(n, "Paragraph")); got != "For God so loved the world." {
		t.Errorf("single verse = %q", got)
	}
	_, n = renderDebug(t, BibleVerse{Reference: "John 3:16-17", Verses: john3, HideNumbers: true})
	if got := paragraphText(findType(n, "Paragraph")); strings.Contains(got, "16") {
		t.Errorf("HideNumbers left a number in: %q", got)
	}
}

// Verses win over Text; Text alone draws as a plain Text node.
func TestBibleVerseVersesWinOverText(t *testing.T) {
	_, n := renderDebug(t, BibleVerse{Text: "ignored", Verses: john3[:1]})
	if findText(n, "ignored") != nil {
		t.Error("Text was drawn although Verses was set")
	}
	_, n = renderDebug(t, BibleVerse{Text: "Jesus wept."})
	if findText(n, "Jesus wept.") == nil {
		t.Error("Text alone should be drawn")
	}
}

// Loading draws a skeleton named by the reference, and no passage.
func TestBibleVerseLoading(t *testing.T) {
	_, n := renderDebug(t, BibleVerse{Reference: "Psalm 23", Loading: true})
	sk := findFirst(n, func(x *core.Node) bool { return x.Style.AccessibilityLabel == "Loading Psalm 23" })
	if sk == nil {
		t.Error("the skeleton should be announced as \"Loading Psalm 23\"")
	}
	if findType(n, "Paragraph") != nil {
		t.Error("no passage while loading")
	}
}

// Error wins over Loading, is a polite status, and OnRetry adds a button
// that calls it.
func TestBibleVerseErrorAndRetry(t *testing.T) {
	retried := 0
	ctx, n := renderDebug(t, BibleVerse{
		Reference: "John 3:16", Loading: true,
		Error: "Couldn't load the verse.", OnRetry: func() { retried++ },
	})
	msg := findText(n, "Couldn't load the verse.")
	if msg == nil || msg.Style.AccessibilityRole != core.RoleStatus {
		t.Fatal("the error should be drawn as a status")
	}
	if findFirst(n, func(x *core.Node) bool { return strings.HasPrefix(x.Style.AccessibilityLabel, "Loading") }) != nil {
		t.Error("Error should win over Loading")
	}
	btns := buttonsOf(n)
	if len(btns) != 1 || btns[0].Props["label"] != "Retry" {
		t.Fatalf("buttons = %d, want one Retry", len(btns))
	}
	ctx.TriggerCallback(btns[0].Props["onClick"].(string))
	if retried != 1 {
		t.Errorf("OnRetry called %d times", retried)
	}

	_, n = renderDebug(t, BibleVerse{Error: "Offline"})
	if len(buttonsOf(n)) != 0 {
		t.Error("no Retry without OnRetry")
	}
}

// OnOpen wins over URL; with neither there is no link at all.
func TestBibleVerseLinkTargets(t *testing.T) {
	opened := 0
	ctx, n := renderDebug(t, BibleVerse{Verses: john3[:1], URL: "https://example.com", OnOpen: func() { opened++ }, LinkLabel: "Study"})
	link := findFirst(n, func(x *core.Node) bool { return x.Style.AccessibilityRole == core.RoleLink })
	if link == nil || link.Style.AccessibilityLabel != "Study" {
		t.Fatal("LinkLabel should name the link")
	}
	ctx.TriggerCallback(link.Props["onClick"].(string))
	if opened != 1 {
		t.Errorf("OnOpen called %d times", opened)
	}

	_, n = renderDebug(t, BibleVerse{Verses: john3[:1]})
	if findFirst(n, func(x *core.Node) bool { return x.Style.AccessibilityRole == core.RoleLink }) != nil {
		t.Error("a link with nowhere to go should be left out")
	}
}

// Nothing to quote and nothing pending is reported, not silently drawn.
func TestBibleVerseEmptyIsAConcern(t *testing.T) {
	core.SetDebugMode(true)
	core.ClearConcerns()
	t.Cleanup(func() { core.SetDebugMode(false); core.ClearConcerns() })
	ctx := core.NewContext()
	ctx.BeginRenderPass()
	BibleVerse{Reference: "John 3:16"}.Render(ctx)
	ctx.EndRenderPass()
	if !strings.Contains(core.DumpConcerns(), ConcernBibleVerseEmpty) {
		t.Error("an empty BibleVerse should raise ConcernBibleVerseEmpty")
	}
}

// The caller's Style lands on the card, after the widget's gap.
func TestBibleVerseCallerStyleWins(t *testing.T) {
	_, n := renderDebug(t, BibleVerse{Verses: john3[:1], Style: []core.StyleProp{core.Gap(31)}})
	if n.Style.Gap != 31 {
		t.Errorf("gap = %v, want the caller's 31", n.Style.Gap)
	}
}

// Every bundled theme renders without a concern.
func TestBibleVerseEveryTheme(t *testing.T) {
	for name, th := range core.BundledThemes() {
		core.SetDebugMode(true)
		core.ClearConcerns()
		ctx := core.NewContext().WithTheme(th)
		ctx.BeginRenderPass()
		n := BibleVerse{Reference: "John 3:16-17", Translation: "KJV", Verses: john3, URL: "https://x.test"}.Render(ctx)
		ctx.EndRenderPass()
		core.AuditTree(n)
		if dump := core.DumpConcerns(); dump != "" {
			t.Errorf("%s: %s", name, dump)
		}
		core.SetDebugMode(false)
		core.ClearConcerns()
	}
}
