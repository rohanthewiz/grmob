package comps

import (
	"strconv"
	"strings"

	"github.com/rohanthewiz/grmob/core"
)

// ConcernBibleStudyInert is raised, in debug builds only, when a BibleStudy
// has neither StudyURL nor OnStudy. Its verses would then lead nowhere, so
// it draws a plain BibleVerse instead of rows that look tappable and are not.
const ConcernBibleStudyInert = "bible-study-inert"

// ConcernBibleStudyNoVerses is raised, in debug builds only, when a
// BibleStudy's passage has Text but no Verses. A single run of text has no
// verse to study, so it draws as a plain BibleVerse.
const ConcernBibleStudyNoVerses = "bible-study-no-verses"

// BibleStudyTool is one entry in BibleStudy's sheet: a study tool on Blue
// Letter Bible, opened for the tapped verse.
type BibleStudyTool struct {
	// Tool is the key BibleStudy passes to StudyURL and OnStudy. For Blue
	// Letter Bible it is one of blb's Tool constants (blb.ToolCrossRefs,
	// …), which are BLB's names for the tabs of a verse's study page.
	Tool string

	// Label is the action's caption in the sheet.
	Label string
}

// DefaultBibleStudyTools is the sheet BibleStudy shows when its Tools is
// empty: the five study tabs Blue Letter Bible keeps for every verse, in the
// order its own verse menu lists them, without Misc (images and maps that
// most verses have none of). The keys match blb's Tool constants; a test in
// blb holds the two lists together, since comps cannot import blb (see
// BibleVerse, "Where the text comes from"). Each call returns a fresh slice,
// so a caller can edit one into its own list.
func DefaultBibleStudyTools() []BibleStudyTool {
	return []BibleStudyTool{
		{Tool: "conc", Label: "Interlinear & Strong's"},
		{Tool: "corr", Label: "Cross-references"},
		{Tool: "comms", Label: "Commentaries"},
		{Tool: "refs", Label: "Dictionaries"},
		{Tool: "bibles", Label: "Compare translations"},
	}
}

// BibleStudy is a BibleVerse whose verses can be tapped to dig deeper: each
// verse is a row of its own, and a tap opens a sheet of Blue Letter Bible's
// study tools for that verse (the interlinear with Strong's numbers,
// cross-references, commentaries, dictionaries, other translations). The
// passage a study app opens on, a devotional's verse with "go deeper" under
// it.
//
//	p, err := blb.Fetch(ctx, "John 3:16-18", "KJV") // off the render path
//	comps.BibleStudy{
//	    Passage: comps.BibleVerse{
//	        Reference:   p.Reference,
//	        Translation: p.Translation,
//	        Verses:      lines(p.Verses), // []blb.Verse → []comps.BibleVerseLine
//	        URL:         p.URL,
//	    },
//	    StudyURL: p.StudyURL, // a verse's study tab on BLB
//	}
//
//	┌ Box ─────────────────────────────────────────────┐
//	│ ┌ Card (BibleVerse's) ─────────────────────────┐ │
//	│ │ ┌ Row, button "Verse 16, For God so…" ─────┐ │ │
//	│ │ │ 16  For God so loved the world, that he  │ │ │  one row per verse,
//	│ │ │     gave his only begotten Son…          │ │ │  the whole row the
//	│ │ └──────────────────────────────────────────┘ │ │  tap target
//	│ │ ┌ Row, button "Verse 17, …" ───────────────┐ │ │
//	│ │ │ 17  For God sent not his Son into…       │ │ │
//	│ │ └──────────────────────────────────────────┘ │ │
//	│ │ Tap a verse to dig deeper on Blue Letter…    │ │  Prompt (hidden from
//	│ │                                              │ │  readers: the rows'
//	│ │ John 3:16-17 (KJV)                           │ │  hint says it)
//	│ │ Read on Blue Letter Bible                    │ │
//	│ └──────────────────────────────────────────────┘ │
//	│ ActionSheet  "John 3:17 (KJV)"                   │  sibling, not child:
//	│   Interlinear & Strong's / Cross-references /    │  shown while a verse
//	│   Commentaries / Dictionaries /                  │  is picked
//	│   Compare translations / Cancel                  │
//	└──────────────────────────────────────────────────┘
//
// # Built on BibleVerse
//
// Passage is the card, configured exactly as a BibleVerse is: Reference,
// Translation, Verses, URL, OnOpen, LinkLabel, Loading, Error, OnRetry,
// RetryLabel and Style all mean what they mean there, and Loading and Error
// draw BibleVerse's skeleton and message. Only the quoted text changes, from
// one Paragraph to a row per verse, because a run inside a Paragraph can be
// a link but a verse-long link would colour the whole passage as one. Rows
// read one verse to a line, the layout BLB's own chapter view uses for the
// same reason: each verse is something to tap.
//
// HideNumbers drops the numbers as it does on BibleVerse, and a single verse
// has none; the rows stay tappable either way. Text has no verses to study:
// a passage with Text and no Verses draws as the plain BibleVerse, and
// raises ConcernBibleStudyNoVerses.
//
// # Where a tool goes
//
// OnStudy, when set, is called with the verse and the tool's key: an app
// that opens the study page in its own web view, or that keeps a history.
// Otherwise StudyURL builds the address and core.OpenURL opens it; pass the
// fetched passage's blb.Passage.StudyURL method. When StudyURL returns ""
// (a hand-built passage blb could not key), the passage's URL opens instead,
// so a tap never does nothing while a link exists. With neither callback the
// rows would be inert, so BibleStudy draws a plain BibleVerse and raises
// ConcernBibleStudyInert.
//
// The tool keys are Tools' (DefaultBibleStudyTools when empty). They are
// opaque to the widget, which only hands them back, so a caller can point
// StudyURL at another site's study pages with keys of its own.
//
// # One hook
//
// Which verse's sheet is open is BibleStudy's own state: no app reads or
// drives it, as with an Accordion's open section. So BibleStudy takes one
// hook slot and must be rendered unconditionally, every pass; switch it
// between loading, failed and loaded with Passage's fields, never by
// rendering it inside core.If.
//
// # Accessibility
//
// Each verse row is a button named "Verse 16, " and the verse's text, with
// VerseHint saying what the tap does. A row's children are presentational,
// so the name carries the whole verse; a reader moving through the passage
// hears it a verse at a time, which is how a passage is read aloud. The
// sheet is named for the verse ("John 3:17 (KJV)"), so a reader landing in
// it knows which verse its tools are for.
//
// # Theme roles read
//
//	Card         as BibleVerse
//	Rows         Spacing.XS between them and between number and text
//	Verse text   Typography.Body, Colors.TextPrimary
//	Numbers      Typography.Body, Colors.TextSecondary, bold, right-aligned
//	             in a gutter sized from Typography.Body.FontSize
//	Prompt       Typography.Caption, Colors.TextSecondary
//	Sheet        as ActionSheet
type BibleStudy struct {
	// Passage is the card: everything BibleVerse draws, configured the same
	// way. See "Built on BibleVerse".
	Passage BibleVerse

	// StudyURL is the address of verse's study page opened on tool, where
	// verse is a BibleVerseLine.Number and tool a BibleStudyTool.Tool. The
	// fetched passage's blb.Passage.StudyURL. Used when OnStudy is nil.
	StudyURL func(verse int, tool string) string

	// OnStudy handles a tool instead of opening StudyURL's address. The
	// sheet closes after it, as ActionSheet closes after any action.
	OnStudy func(verse BibleVerseLine, tool string)

	// Tools are the sheet's actions, top to bottom. Empty uses
	// DefaultBibleStudyTools.
	Tools []BibleStudyTool

	// Prompt replaces "Tap a verse to dig deeper on Blue Letter Bible", the
	// caption under the verses that says they can be tapped.
	Prompt string

	// HidePrompt leaves the caption out, for an app whose readers already
	// know the verses are tappable.
	HidePrompt bool

	// VerseHint replaces "Opens study tools on Blue Letter Bible", what a
	// screen reader says a verse row does.
	VerseHint string

	// CancelLabel replaces "Cancel", the sheet's way out.
	CancelLabel string
}

// Render draws the card and the sheet beside it. It takes one hook slot,
// on every pass, before anything that could branch.
func (s BibleStudy) Render(ctx *core.Context) *core.Node {
	// The open sheet's verse number; 0 is closed. A verse number is never 0,
	// so the zero value doubles as "none" without a second slot.
	open := core.NewState(ctx, 0)
	t := ctx.Theme()

	v := s.Passage
	studiable := s.StudyURL != nil || s.OnStudy != nil
	// The verses are on screen only in the card's loaded state; while it
	// loads or shows an error there is nothing to tap and no sheet to show.
	showing := v.Error == "" && !v.Loading && len(v.Verses) > 0

	if core.IsDebugMode() && v.Error == "" && !v.Loading {
		switch {
		case !studiable && (len(v.Verses) > 0 || v.Text != ""):
			core.ReportConcern(ConcernBibleStudyInert,
				"BibleStudy \""+v.Reference+"\" has neither StudyURL nor OnStudy, so its verses lead nowhere; it draws as a plain BibleVerse")
		case len(v.Verses) == 0 && v.Text != "":
			core.ReportConcern(ConcernBibleStudyNoVerses,
				"BibleStudy \""+v.Reference+"\" has Text but no Verses, so there is no verse to study; it draws as a plain BibleVerse")
		}
	}

	var body core.View
	if studiable && showing {
		body = s.verses(t, open)
	}
	card := core.ComponentFunc(func(ctx *core.Context) *core.Node { return v.card(ctx, body) })

	// The open verse is looked up again on every pass rather than stored, so
	// a sheet left open while the caller swaps in another passage closes on
	// its own when that number is not in the new one.
	line, found := s.line(open.Get())
	closeSheet := func() { open.Set(0) }
	sheet := ActionSheet{
		Visible:   studiable && showing && found,
		Cancel:    orDefault(s.CancelLabel, "Cancel"),
		OnDismiss: closeSheet,
	}
	if found {
		sheet.Title = s.verseRef(line)
		sheet.Actions = s.actions(line)
	}

	// Box, not Column, as DatePicker does: the card is the widget's whole
	// footprint, and the sheet is a Modal beside it rather than inside it,
	// so the card's own Gap and clip never reach the overlay. The ActionSheet
	// is in the tree on every pass, open or not, so opening it patches one
	// node instead of inserting one.
	return core.Box(card, sheet).Render(ctx)
}

// verses is the body that replaces BibleVerse's Paragraph: one tappable row
// per verse, then the prompt.
func (s BibleStudy) verses(t *core.Theme, open core.State[int]) core.View {
	v := s.Passage
	numbered := len(v.Verses) > 1 && !v.HideNumbers
	hint := orDefault(s.VerseHint, "Opens study tools on Blue Letter Bible")
	gutter := ""
	if numbered {
		gutter = verseGutter(t, v.Verses)
	}

	items := make([]core.PropsAndChildren, 0, len(v.Verses)+3)
	// Padding(0): the theme Column's screen inset would double the card's.
	items = append(items, core.Padding(0), core.Gap(float64(t.Spacing.XS)))
	for _, line := range v.Verses {
		text := core.Text(line.Text,
			core.UseStyle(t.Typography.Body),
			core.TextColor(t.Colors.TextPrimary),
			// Basis 0 so the text takes the row's remaining width and wraps
			// inside it, rather than sizing to its own unwrapped length and
			// pushing the number off the start.
			core.FlexGrow(1), core.FlexBasis("0"),
		)
		row := []core.PropsAndChildren{
			core.Padding(0),
			core.Gap(float64(t.Spacing.XS)),
			// The number sits on the first line of a wrapped verse, as it
			// does in print, not centred against the whole block.
			core.AlignItemsProp(core.AlignItemsStart),
		}
		if line.Number > 0 {
			n := line.Number
			// Role, name and handler on the same node: the row is the
			// button, so the whole verse is the tap target.
			row = append(row,
				core.OnClick(func() { open.Set(n) }),
				core.AccessibilityRole(core.RoleButton),
				core.AccessibilityLabel("Verse "+strconv.Itoa(n)+", "+line.Text),
				core.AccessibilityHint(hint),
			)
		}
		// A line numbered 0 is not a verse BLB can key, so it is drawn as
		// text and nothing more: a tap would set the "closed" value.
		if numbered {
			// Every row gets the gutter, so every verse's text starts at one
			// x: "9" and "10", or "1" and "2" in a proportional face, would
			// otherwise each push their text to a different place. The
			// number is right-aligned in it, as BulletList sets "9." over
			// "10.", so the digits' ends line up against the text.
			gut := []core.StyleProp{
				core.Width(gutter),
				core.FlexShrink(0),
				core.Align(core.AlignEnd),
			}
			if line.Number > 0 {
				row = append(row, core.Text(strconv.Itoa(line.Number), append(gut,
					core.UseStyle(t.Typography.Body),
					core.TextColor(t.Colors.TextSecondary),
					core.FontWeight(core.Bold),
				)...))
			} else {
				// An unnumbered line keeps the gutter empty rather than
				// starting its text under the numbers.
				row = append(row, core.Box(core.Width(gutter), core.FlexShrink(0), core.AccessibilityHidden()))
			}
		}
		row = append(row, text)
		// Keyed by number so each row's callback ID is scoped to its verse,
		// and a passage that grows or shrinks keeps every other row's.
		items = append(items, core.Keyed("verse:"+strconv.Itoa(line.Number), core.Row(row...)))
	}
	if !s.HidePrompt {
		items = append(items, core.Text(
			orDefault(s.Prompt, "Tap a verse to dig deeper on Blue Letter Bible"),
			core.UseStyle(t.Typography.Caption),
			core.TextColor(t.Colors.TextSecondary),
			// Each row's hint already says this, verse by verse; read once
			// more after the last verse it would only interrupt the passage.
			core.AccessibilityHidden(),
		))
	}
	return core.Column(items...)
}

// verseGutter is the width of the number column: the widest verse number's
// digit count at the body size, 0.62 em a digit. No host reports a rendered
// width, so this is BulletList's estimate for its ordered markers: digits are
// tabular in every bundled face, and 0.62 em covers the widest of them in
// bold with room to spare.
func verseGutter(t *core.Theme, verses []BibleVerseLine) string {
	size := t.Typography.Body.FontSize
	if size == 0 {
		size = 16
	}
	widest := 1
	for _, l := range verses {
		if d := len(strconv.Itoa(l.Number)); d > widest {
			widest = d
		}
	}
	return strconv.FormatFloat(float64(widest)*0.62*size, 'f', 1, 64) + "px"
}

// line finds the verse numbered n in the passage.
func (s BibleStudy) line(n int) (BibleVerseLine, bool) {
	if n <= 0 {
		return BibleVerseLine{}, false
	}
	for _, l := range s.Passage.Verses {
		if l.Number == n {
			return l, true
		}
	}
	return BibleVerseLine{}, false
}

// actions builds the sheet's entries for one verse.
func (s BibleStudy) actions(line BibleVerseLine) []SheetAction {
	tools := s.Tools
	if len(tools) == 0 {
		tools = DefaultBibleStudyTools()
	}
	out := make([]SheetAction, 0, len(tools))
	for _, tool := range tools {
		out = append(out, SheetAction{Label: tool.Label, OnTap: s.study(line, tool.Tool)})
	}
	return out
}

// study returns the handler for one tool on one verse. ActionSheet closes
// the sheet after it, so it only has to go somewhere.
func (s BibleStudy) study(line BibleVerseLine, tool string) func() {
	return func() {
		if s.OnStudy != nil {
			s.OnStudy(line, tool)
			return
		}
		u := ""
		if s.StudyURL != nil {
			u = s.StudyURL(line.Number, tool)
		}
		if u == "" {
			// The passage page still has the verse on it, and is better than
			// a sheet entry that does nothing.
			u = s.Passage.URL
		}
		core.OpenURL(u) // drops an empty url itself
	}
}

// verseRef names one verse of the passage: "John 3:17 (KJV)" from a
// Reference of "John 3:16-18". The chapter is everything up to the last
// colon; a whole-chapter reference ("Psalm 23") has none, and the verse is
// added to it. Without a Reference it is "Verse 17".
func (s BibleStudy) verseRef(line BibleVerseLine) string {
	n := strconv.Itoa(line.Number)
	ref := strings.TrimSpace(s.Passage.Reference)
	switch {
	case ref == "":
		ref = "Verse " + n
	case strings.Contains(ref, ":"):
		ref = ref[:strings.LastIndex(ref, ":")+1] + n
	default:
		ref += ":" + n
	}
	if s.Passage.Translation != "" {
		ref += " (" + s.Passage.Translation + ")"
	}
	return ref
}
