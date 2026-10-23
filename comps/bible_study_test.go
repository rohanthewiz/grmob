package comps

import (
	"fmt"
	"strings"
	"testing"

	"github.com/rohanthewiz/grmob/core"
)

// john3Study is a passage keyed the way blb.Passage.StudyURL keys it, without
// importing blb: John 3 is BLB's chapter 1000.
func john3Study() BibleStudy {
	return BibleStudy{
		Passage: BibleVerse{
			Reference: "John 3:16-17", Translation: "KJV", Verses: john3,
			URL: "https://www.blueletterbible.org/kjv/jhn/3/16/",
		},
		StudyURL: func(verse int, tool string) string {
			return fmt.Sprintf("https://www.blueletterbible.org/kjv/jhn/3/%d/t_%s_%d", verse, tool, 1000000+verse)
		},
	}
}

// verseRow finds the tappable row for verse n by its spoken name.
func verseRow(n *core.Node, num int) *core.Node {
	prefix := fmt.Sprintf("Verse %d, ", num)
	return findFirst(n, func(x *core.Node) bool {
		return x.Style != nil && x.Style.AccessibilityRole == core.RoleButton && strings.HasPrefix(x.Style.AccessibilityLabel, prefix)
	})
}

func sheetOf(n *core.Node) *core.Node { return findType(n, "Modal") }

// captureOpenURL records every core.OpenURL for the rest of the test.
func captureOpenURL(t *testing.T) *[]string {
	var opened []string
	core.SetSystemEventHandler(func(name string, data map[string]any) {
		if name == "open_url" {
			opened = append(opened, data["url"].(string))
		}
	})
	t.Cleanup(func() { core.SetSystemEventHandler(nil) })
	return &opened
}

// Loaded: a button row per verse with its number and text, the prompt
// hidden from readers, BibleVerse's reference and link, and a closed sheet.
func TestBibleStudyDrawsATappableRowPerVerse(t *testing.T) {
	_, n := renderDebug(t, john3Study())
	if n.Type != "Box" || findType(n, "Card") == nil {
		t.Fatalf("root = %s, want a Box around BibleVerse's Card", n.Type)
	}
	if findType(n, "Paragraph") != nil {
		t.Error("the verses should be rows, not BibleVerse's Paragraph")
	}
	for _, v := range john3 {
		row := verseRow(n, v.Number)
		if row == nil {
			t.Fatalf("no button row for verse %d", v.Number)
		}
		if want := fmt.Sprintf("Verse %d, %s", v.Number, v.Text); row.Style.AccessibilityLabel != want {
			t.Errorf("verse %d named %q, want %q", v.Number, row.Style.AccessibilityLabel, want)
		}
		if row.Style.AccessibilityHint != "Opens study tools on Blue Letter Bible" {
			t.Errorf("verse %d hint = %q", v.Number, row.Style.AccessibilityHint)
		}
		if _, ok := row.Props["onClick"].(string); !ok {
			t.Errorf("verse %d row has no click handler", v.Number)
		}
		if findText(row, fmt.Sprint(v.Number)) == nil || findText(row, v.Text) == nil {
			t.Errorf("verse %d row should draw its number and its text", v.Number)
		}
	}
	prompt := findText(n, "Tap a verse to dig deeper on Blue Letter Bible")
	if prompt == nil || !prompt.Style.AccessibilityHidden {
		t.Error("the prompt should be drawn and hidden from readers (the rows' hints say it)")
	}
	if findText(n, "John 3:16-17 (KJV)") == nil {
		t.Error("the reference line is BibleVerse's and should still be drawn")
	}
	if findFirst(n, func(x *core.Node) bool { return x.Style != nil && x.Style.AccessibilityRole == core.RoleLink }) == nil {
		t.Error("the link back to Blue Letter Bible should still be drawn")
	}
	if s := sheetOf(n); s == nil || s.Props["visible"] == true {
		t.Error("the sheet should be in the tree and closed")
	}
}

// A tap opens the sheet named for that verse, with the default tools; a
// tool opens StudyURL's address for that verse and closes the sheet.
func TestBibleStudyTapOpensTheVersesTools(t *testing.T) {
	opened := captureOpenURL(t)
	w := john3Study()
	ctx, n := renderDebug(t, w)

	ctx.TriggerCallback(verseRow(n, 17).Props["onClick"].(string))
	n = renderPass(ctx, w)
	sheet := sheetOf(n)
	if sheet == nil || sheet.Props["visible"] != true {
		t.Fatal("tapping verse 17 should open the sheet")
	}
	if findText(sheet, "John 3:17 (KJV)") == nil {
		t.Error("the sheet should be titled for the tapped verse, \"John 3:17 (KJV)\"")
	}
	for _, tool := range DefaultBibleStudyTools() {
		if findButton(sheet, tool.Label) == nil {
			t.Errorf("the sheet has no %q", tool.Label)
		}
	}

	ctx.TriggerCallback(findButton(sheet, "Cross-references").Props["onClick"].(string))
	want := "https://www.blueletterbible.org/kjv/jhn/3/17/t_corr_1000017"
	if len(*opened) != 1 || (*opened)[0] != want {
		t.Errorf("opened %v, want [%s]", *opened, want)
	}
	if n = renderPass(ctx, w); sheetOf(n).Props["visible"] == true {
		t.Error("picking a tool should close the sheet")
	}
}

// OnStudy wins over StudyURL: it gets the verse and the key, and nothing is
// opened. Cancel closes without calling either.
func TestBibleStudyOnStudyAndCancel(t *testing.T) {
	opened := captureOpenURL(t)
	var got []string
	w := john3Study()
	w.Tools = []BibleStudyTool{{Tool: "notes", Label: "My notes"}}
	w.OnStudy = func(v BibleVerseLine, tool string) { got = append(got, fmt.Sprintf("%d:%s", v.Number, tool)) }
	ctx, n := renderDebug(t, w)

	ctx.TriggerCallback(verseRow(n, 16).Props["onClick"].(string))
	n = renderPass(ctx, w)
	if findButton(sheetOf(n), "Interlinear & Strong's") != nil {
		t.Error("Tools replaces the defaults rather than adding to them")
	}
	ctx.TriggerCallback(findButton(sheetOf(n), "My notes").Props["onClick"].(string))
	if len(got) != 1 || got[0] != "16:notes" || len(*opened) != 0 {
		t.Errorf("OnStudy got %v and %v were opened; want [16:notes] and none", got, *opened)
	}

	n = renderPass(ctx, w)
	ctx.TriggerCallback(verseRow(n, 17).Props["onClick"].(string))
	n = renderPass(ctx, w)
	ctx.TriggerCallback(findButton(sheetOf(n), "Cancel").Props["onClick"].(string))
	if n = renderPass(ctx, w); sheetOf(n).Props["visible"] == true || len(got) != 1 {
		t.Error("Cancel should close the sheet and study nothing")
	}
}

// A StudyURL with nothing to build (a hand-made passage) falls back to the
// passage page rather than a sheet entry that does nothing.
func TestBibleStudyFallsBackToThePassageURL(t *testing.T) {
	opened := captureOpenURL(t)
	w := john3Study()
	w.StudyURL = func(int, string) string { return "" }
	ctx, n := renderDebug(t, w)
	ctx.TriggerCallback(verseRow(n, 16).Props["onClick"].(string))
	n = renderPass(ctx, w)
	ctx.TriggerCallback(findButton(sheetOf(n), "Commentaries").Props["onClick"].(string))
	if len(*opened) != 1 || (*opened)[0] != w.Passage.URL {
		t.Errorf("opened %v, want the passage URL", *opened)
	}
}

// Loading and failed are BibleVerse's states: no rows, no sheet. The hook
// slot is the same in every state, so one context renders all three.
func TestBibleStudyLoadingAndFailedAreBibleVerses(t *testing.T) {
	w := john3Study()
	w.Passage.Loading = true
	ctx, n := renderDebug(t, w)
	if findType(n, "Skeleton") == nil && findFirst(n, func(x *core.Node) bool {
		return x.Style != nil && strings.HasPrefix(x.Style.AccessibilityLabel, "Loading ")
	}) == nil {
		t.Error("Loading should draw BibleVerse's skeleton")
	}
	if verseRow(n, 16) != nil {
		t.Error("no verse rows while loading")
	}

	w.Passage.Loading, w.Passage.Error = false, "Could not reach Blue Letter Bible"
	n = renderPass(ctx, w)
	if findText(n, w.Passage.Error) == nil || verseRow(n, 16) != nil {
		t.Error("Error should draw BibleVerse's message and no rows")
	}

	w.Passage.Error = ""
	n = renderPass(ctx, w)
	if verseRow(n, 16) == nil {
		t.Error("once loaded the rows appear on the same context")
	}
}

// A sheet left open while the passage changes to one without that verse
// closes rather than titling itself for a verse no longer on screen.
func TestBibleStudySheetClosesWhenItsVerseLeaves(t *testing.T) {
	w := john3Study()
	ctx, n := renderDebug(t, w)
	ctx.TriggerCallback(verseRow(n, 17).Props["onClick"].(string))
	w.Passage.Verses = john3[:1]
	w.Passage.Reference = "John 3:16"
	if n = renderPass(ctx, w); sheetOf(n).Props["visible"] == true {
		t.Error("verse 17 left the passage; its sheet should be closed")
	}
}

// One verse has no number, but is still a row to tap.
func TestBibleStudySingleVerseAndHiddenNumbers(t *testing.T) {
	w := john3Study()
	w.Passage.Verses = john3[:1]
	_, n := renderDebug(t, w)
	row := verseRow(n, 16)
	if row == nil || findText(row, "16") != nil {
		t.Error("a single verse is a tappable row with no number drawn")
	}

	w = john3Study()
	w.Passage.HideNumbers = true
	w.HidePrompt = true
	_, n = renderDebug(t, w)
	if findText(verseRow(n, 17), "17") != nil {
		t.Error("HideNumbers drops the numbers")
	}
	if findText(n, "Tap a verse to dig deeper on Blue Letter Bible") != nil {
		t.Error("HidePrompt drops the prompt")
	}
}

// With nowhere for a tool to go, or no verses to study, it is a plain
// BibleVerse, and debug mode says why.
func TestBibleStudyConcerns(t *testing.T) {
	for _, c := range []struct {
		name    string
		w       BibleStudy
		concern string
	}{
		{"inert", BibleStudy{Passage: john3Study().Passage}, ConcernBibleStudyInert},
		{"text only", BibleStudy{
			Passage:  BibleVerse{Reference: "John 3:16", Text: "For God so loved the world."},
			StudyURL: john3Study().StudyURL,
		}, ConcernBibleStudyNoVerses},
	} {
		t.Run(c.name, func(t *testing.T) {
			core.SetDebugMode(true)
			core.ClearConcerns()
			t.Cleanup(func() { core.SetDebugMode(false); core.ClearConcerns() })
			ctx := core.NewContext()
			ctx.BeginRenderPass()
			n := c.w.Render(ctx)
			ctx.EndRenderPass()
			if !strings.Contains(core.DumpConcerns(), c.concern) {
				t.Errorf("want %s, got:\n%s", c.concern, core.DumpConcerns())
			}
			if findType(n, "Paragraph") == nil && findText(n, "For God so loved the world.") == nil {
				t.Error("it should draw the passage as BibleVerse does")
			}
			if findFirst(n, func(x *core.Node) bool { return x.Style != nil && x.Style.AccessibilityRole == core.RoleButton }) != nil {
				t.Error("no tappable rows")
			}
		})
	}
}

// The verse is named in the sheet from the passage's reference, however the
// reference names its chapter.
func TestBibleStudyVerseRef(t *testing.T) {
	for _, c := range []struct{ ref, tr, want string }{
		{"John 3:16-18", "KJV", "John 3:17 (KJV)"},
		{"Romans 8:28,31", "", "Romans 8:17"},
		{"Psalms 23", "NKJV", "Psalms 23:17 (NKJV)"},
		{"", "", "Verse 17"},
	} {
		w := BibleStudy{Passage: BibleVerse{Reference: c.ref, Translation: c.tr}}
		if got := w.verseRef(BibleVerseLine{Number: 17}); got != c.want {
			t.Errorf("%q: %q, want %q", c.ref, got, c.want)
		}
	}
}

// Every bundled theme renders it without a concern.
func TestBibleStudyEveryTheme(t *testing.T) {
	for i, th := range core.BundledThemes() {
		core.SetDebugMode(true)
		core.ClearConcerns()
		ctx := core.NewContext().WithTheme(th)
		ctx.BeginRenderPass()
		n := john3Study().Render(ctx)
		ctx.EndRenderPass()
		core.AuditTree(n)
		if d := core.DumpConcerns(); d != "" {
			t.Errorf("theme %s: %s", i, d)
		}
	}
	core.SetDebugMode(false)
	core.ClearConcerns()
}
