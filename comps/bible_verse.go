package comps

import (
	"strconv"

	"github.com/rohanthewiz/grmob/core"
)

// ConcernBibleVerseEmpty is raised, in debug builds only, when a BibleVerse
// has no text, no verses, no error and is not Loading. It draws a card with a
// reference and nothing quoted under it, which reads as a failed load that
// nobody reported.
const ConcernBibleVerseEmpty = "bible-verse-empty"

// BibleVerseLine is one numbered verse of a BibleVerse. It has the shape of
// blb.Verse, so a fetched passage converts with a loop (see BibleVerse).
type BibleVerseLine struct {
	Number int
	Text   string
}

// BibleVerse is a passage of scripture on a card: the text, the reference
// under it, and a link to the passage on Blue Letter Bible. The verse of the
// day on a home screen, the passage a devotional opens with, a reference in a
// study app tapped to see what it says.
//
//	p, err := blb.Fetch(ctx, "John 3:16", "KJV") // off the render path
//	comps.BibleVerse{
//	    Reference:   p.Reference,
//	    Translation: p.Translation,
//	    Verses:      lines(p.Verses), // []blb.Verse → []comps.BibleVerseLine
//	    URL:         p.URL,
//	}
//
//	┌ Card ──────────────────────────────────────────┐
//	│  16 For God so loved the world, that he gave   │  Paragraph: a number run
//	│  his only begotten Son… 17 For God sent not    │  per verse (two or more),
//	│  his Son into the world…                       │  then the verse's text
//	│                                                │
//	│  John 3:16-17 (KJV)                            │  Text, Caption, bold
//	│  Read on Blue Letter Bible                     │  Link → URL / OnOpen
//	└────────────────────────────────────────────────┘
//
//	Loading: the Paragraph is a 3-line Skeleton.
//	Error:   the Paragraph is the message (RoleStatus) and, with OnRetry,
//	         an outlined Retry button.
//
// # Where the text comes from
//
// The caller. The widget fetches nothing: comps is pure view code and is
// compiled into every browser build, where a network client would cost size
// for apps that never quote a verse. The blb package does the fetching, from
// Blue Letter Bible's ScriptTagger feed, and returns the reference, the
// numbered verses and the passage's address; the call above is the whole
// integration. Fetch in a goroutine or an effect, set Loading while it runs,
// and Error if it fails.
//
// Text is for a passage that arrives as one string (a stored quote, another
// source). When Verses has any lines it wins, as a slot wins over a simple
// field elsewhere in comps.
//
// # Verse numbers
//
// Drawn before each verse when there are two or more, in the secondary text
// colour and bold, the way a printed Bible sets them; a single verse has no
// number, since the reference under it already says which one it is.
// HideNumbers drops them for a passage read as prose. A reader hears the
// number before each verse, which is how a passage is read aloud too.
//
// # The link back
//
// Blue Letter Bible serves its text to embedders on the understanding that it
// links back, so the link is drawn whenever there is somewhere for it to go:
// OnOpen if set (an in-app reader), else URL, opened with core.OpenURL. With
// neither the link is left out rather than drawn inert. blb.SearchURL builds
// a working address from a reference alone, for the case where the fetch
// failed and Passage.URL never arrived.
//
// # No hooks
//
// Everything is the caller's, so BibleVerse may be rendered conditionally.
//
// # Theme roles read
//
//	Card         the theme's Card base, Spacing.SM between the parts
//	Verse text   Typography.Body, Colors.TextPrimary
//	Numbers      Colors.TextSecondary, bold
//	Reference    Typography.Caption, Colors.TextSecondary, bold
//	Error        Typography.Body, Colors.Error
//	Link         as Link: Colors.Primary's ink tone
type BibleVerse struct {
	// Reference names the passage ("John 3:16"). Drawn under the text, and
	// the name the skeleton is announced by while Loading.
	Reference string

	// Translation is drawn after the reference, in parentheses ("KJV").
	// Empty leaves the parentheses out.
	Translation string

	// Verses are the passage's numbered verses. They win over Text.
	Verses []BibleVerseLine

	// Text is the passage as one run, for a source without verse numbers.
	Text string

	// HideNumbers leaves out the verse numbers of a multi-verse passage.
	HideNumbers bool

	// URL is the passage's page, opened with core.OpenURL from the link.
	// blb.Passage.URL, or blb.SearchURL(ref, translation).
	URL string

	// OnOpen handles the link instead of opening URL: an in-app reader.
	OnOpen func()

	// LinkLabel replaces "Read on Blue Letter Bible".
	LinkLabel string

	// Loading draws a skeleton in place of the text.
	Loading bool

	// Error is drawn in place of the text when it is not empty: the fetch
	// failed. It wins over Loading, so a caller that forgets to clear
	// Loading on failure still shows why.
	Error string

	// OnRetry adds a Retry button under Error.
	OnRetry func()

	// RetryLabel replaces "Retry".
	RetryLabel string

	// Style is applied to the card after the widget's own props.
	Style []core.StyleProp
}

// Render draws the card. It takes no hook slot.
func (v BibleVerse) Render(ctx *core.Context) *core.Node {
	t := ctx.Theme()

	items := make([]core.PropsAndChildren, 0, len(v.Style)+5)
	items = append(items, core.Gap(float64(t.Spacing.SM)))
	items = append(items, asProps(v.Style)...)

	switch {
	case v.Error != "":
		// Status, not alert: a verse that did not load is worth hearing at
		// the next pause, not worth cutting off what the reader was doing.
		items = append(items, core.Text(v.Error,
			core.UseStyle(t.Typography.Body),
			core.TextColor(t.Colors.Error),
			core.AccessibilityRole(core.RoleStatus),
		))
		if v.OnRetry != nil {
			items = append(items, Button{
				Label:    orDefault(v.RetryLabel, "Retry"),
				OnTap:    v.OnRetry,
				Emphasis: EmphasisOutlined,
				// A button stretched across the card would read as the card's
				// main action; Retry is a small recovery beside a message.
				Style: []core.StyleProp{core.AlignSelf(core.AlignItemsStart)},
			})
		}
	case v.Loading:
		label := "Loading verse"
		if v.Reference != "" {
			label = "Loading " + v.Reference
		}
		items = append(items, Skeleton{Lines: 3, AccessibilityLabel: label})
	case len(v.Verses) > 0 || v.Text != "":
		items = append(items, v.passage(t))
	default:
		if core.IsDebugMode() {
			core.ReportConcern(ConcernBibleVerseEmpty,
				"BibleVerse \""+v.Reference+"\" has no Verses, Text or Error and is not Loading, so it draws a reference with nothing quoted under it")
		}
	}

	if ref := v.referenceLine(); ref != "" {
		items = append(items, core.Text(ref,
			core.UseStyle(t.Typography.Caption),
			core.TextColor(t.Colors.TextSecondary),
			core.FontWeight(core.Bold),
		))
	}

	if v.OnOpen != nil || v.URL != "" {
		items = append(items, Link{
			Text:              orDefault(v.LinkLabel, "Read on Blue Letter Bible"),
			URL:               v.URL,
			OnTap:             v.OnOpen,
			AccessibilityHint: "Opens the passage on Blue Letter Bible",
		})
	}

	return core.Card(items...).Render(ctx)
}

// passage is the quoted text: one Paragraph, so the verses wrap as a single
// run the way they do on a page, with each verse's number as a run of its own
// in front of it.
func (v BibleVerse) passage(t *core.Theme) core.View {
	if len(v.Verses) == 0 {
		return core.Text(v.Text,
			core.UseStyle(t.Typography.Body),
			core.TextColor(t.Colors.TextPrimary),
		)
	}
	numbered := len(v.Verses) > 1 && !v.HideNumbers
	runs := make([]core.Span, 0, 2*len(v.Verses))
	for i, line := range v.Verses {
		text := line.Text
		if i < len(v.Verses)-1 {
			// The space between verses belongs to the verse before, so a
			// number never starts a wrapped line with a space in front of it.
			text += " "
		}
		if numbered {
			// A no-break space ties the number to its first word; a line
			// that ends on a bare "17" reads as a stray figure.
			runs = append(runs, core.Span{
				Text:  strconv.Itoa(line.Number) + " ",
				Bold:  true,
				Color: t.Colors.TextSecondary,
			})
		}
		runs = append(runs, core.Span{Text: text})
	}
	return core.Paragraph(runs,
		core.UseStyle(t.Typography.Body),
		core.TextColor(t.Colors.TextPrimary),
	)
}

// referenceLine is "John 3:16 (KJV)", or whichever half is set.
func (v BibleVerse) referenceLine() string {
	switch {
	case v.Reference != "" && v.Translation != "":
		return v.Reference + " (" + v.Translation + ")"
	case v.Translation != "":
		return "(" + v.Translation + ")"
	}
	return v.Reference
}
