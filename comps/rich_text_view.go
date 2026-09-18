package comps

import (
	"strconv"

	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/richtext"
)

// RichTextView draws a richtext.Doc to be read: the formatted document
// RichTextEditor edits, without the editor.
//
//	comps.RichTextView{Doc: note.Get()}
//	comps.RichTextView{Doc: richtext.FromMarkdown(md), OnLink: openInApp}
//
// # Why it waited for core.Paragraph
//
// A document's marks change inside a line: a bold word, an italic phrase, a link.
// Before core.Paragraph a view could only draw a line as one Text in one
// style, and a Row of Texts wraps between its children rather than inside
// them, so the sentence broke at every mark. Each block is one Paragraph now,
// its runs the block's runs, and the sentence wraps as one.
//
// # Blocks
//
//	p          a Paragraph in the Body role
//	h1 h2 h3   a Paragraph, bold, at 1.6, 1.35 and 1.15 of Body's size (the
//	           editor's own scale), announced as a heading of that level
//	bullet     a list item: the marker, then the Paragraph; consecutive items
//	numbered   are one list to a screen reader, numbered from 1 per run of them
//	quote      the Paragraph beside a rule, in the secondary ink
//	code       the Paragraph in a monospace box, every run code
//
// A block with no runs is an empty line, and draws as one, so a document's
// spacing is the author's.
//
// # Links
//
// A run's Link is a tappable run (core.Span.OnTap). OnLink receives the URL;
// nil opens it with core.OpenURL, which is what a reader expects of a link in
// a document. A link is drawn in Link's colour, underlined, so it is a link to
// a reader who cannot tell one colour from another.
//
// # Theme roles read
//
//	Typography.Body       every block's base
//	Colors.TextSecondary  a quote's ink
//	Colors.BorderColor()  a quote's rule
//	Colors.Surface        a code block's fill
type RichTextView struct {
	Doc richtext.Doc

	// OnLink receives a tapped link's URL. Nil opens it with core.OpenURL.
	OnLink func(url string)

	// Style is applied to the outer column after its defaults.
	Style []core.StyleProp
}

// richHeadingScale is the editor's heading scale, relative to Body, which
// both native editors and the web runtime use (GrMobRichText's paragraphSpans).
var richHeadingScale = map[richtext.BlockKind]struct {
	scale float64
	level int
}{
	richtext.Heading1: {1.6, 1},
	richtext.Heading2: {1.35, 2},
	richtext.Heading3: {1.15, 3},
}

func (v RichTextView) Render(ctx *core.Context) *core.Node {
	t := ctx.Theme()
	body := t.Typography.Body.FontSize
	if body == 0 {
		body = 17
	}

	items := make([]core.PropsAndChildren, 0, len(v.Style)+len(v.Doc.Blocks)+2)
	items = append(items, core.Padding(0), core.Gap(float64(t.Spacing.SM)))
	items = append(items, asProps(v.Style)...)

	// A run of consecutive list blocks is one list: gathered here, and flushed
	// as one RoleList column when a block of another kind (or the end) comes.
	var list []core.PropsAndChildren
	var listKind richtext.BlockKind
	flush := func() {
		if len(list) == 0 {
			return
		}
		items = append(items, core.Column(append([]core.PropsAndChildren{
			core.Padding(0),
			core.Gap(float64(t.Spacing.XS)),
			core.AccessibilityRole(core.RoleList),
		}, list...)...))
		list = nil
	}

	for _, b := range v.Doc.Blocks {
		if b.Kind != richtext.Bullet && b.Kind != richtext.Numbered || b.Kind != listKind {
			flush()
		}
		listKind = b.Kind
		runs := v.spans(ctx, b)
		switch b.Kind {
		case richtext.Heading1, richtext.Heading2, richtext.Heading3:
			h := richHeadingScale[b.Kind]
			items = append(items, core.Paragraph(runs,
				core.UseStyle(t.Typography.Body),
				core.FontSize(body*h.scale),
				core.FontWeight(core.Bold),
				core.AccessibilityRole(core.RoleHeading),
				core.AccessibilityHeadingLevel(h.level),
			))
		case richtext.Bullet, richtext.Numbered:
			marker := "•"
			if b.Kind == richtext.Numbered {
				marker = strconv.Itoa(len(list)+1) + "."
			}
			list = append(list, core.Row(
				core.Padding(0),
				core.Gap(float64(t.Spacing.SM)),
				core.AlignItemsProp(core.AlignItemsStart),
				core.AccessibilityRole(core.RoleListItem),
				core.Text(marker,
					core.UseStyle(t.Typography.Body),
					core.FontWeight(core.Bold),
					core.FlexShrink(0),
					core.AccessibilityHidden()),
				core.Paragraph(runs,
					core.UseStyle(t.Typography.Body),
					core.FlexGrow(1),
					core.FlexShrink(1)),
			))
		case richtext.Quote:
			items = append(items, core.Row(
				core.Padding(0),
				core.Gap(float64(t.Spacing.SM)),
				core.AlignItemsProp(core.AlignItemsStretch),
				core.Box(
					core.Width("3px"),
					core.FlexShrink(0),
					core.Padding(0),
					core.BackgroundColor(t.Colors.BorderColor()),
					core.AccessibilityHidden(),
				),
				core.Paragraph(runs,
					core.UseStyle(t.Typography.Body),
					core.TextColor(t.Colors.TextSecondary),
					core.FlexGrow(1),
					core.FlexShrink(1)),
			))
		case richtext.BlockCode:
			for i := range runs {
				runs[i].Code = true
			}
			items = append(items, core.Box(
				core.Padding(t.Spacing.SM),
				core.BorderRadius(8),
				core.BackgroundColor(t.Colors.Surface),
				core.Paragraph(runs, core.UseStyle(t.Typography.Body), core.FontSize(body*0.85)),
			))
		default:
			items = append(items, core.Paragraph(runs, core.UseStyle(t.Typography.Body)))
		}
	}
	flush()
	return core.Column(items...).Render(ctx)
}

// spans turns one block's runs into core.Spans. An empty block becomes one
// space, so the empty line keeps its height on every host (a Paragraph drops
// runs with no text, and an empty one would collapse to nothing).
func (v RichTextView) spans(ctx *core.Context, b richtext.Block) []core.Span {
	out := make([]core.Span, 0, len(b.Runs))
	for _, r := range b.Runs {
		s := core.Span{
			Text: r.Text, Bold: r.Bold, Italic: r.Italic,
			Underline: r.Underline, Strike: r.Strike, Code: r.Code,
		}
		if r.Link != "" {
			s = Link{Text: r.Text, URL: r.Link, OnTap: v.linkTap(r.Link)}.Span(ctx)
			s.Bold, s.Italic, s.Strike, s.Code = r.Bold, r.Italic, r.Strike, r.Code
		}
		out = append(out, s)
	}
	if len(out) == 0 {
		out = append(out, core.Span{Text: " "})
	}
	return out
}

// linkTap is the handler for a link to url: OnLink's when there is one, nil
// otherwise so Link.Span falls back to opening the URL.
func (v RichTextView) linkTap(url string) func() {
	if v.OnLink == nil {
		return nil
	}
	on := v.OnLink
	return func() { on(url) }
}
