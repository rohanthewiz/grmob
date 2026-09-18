package comps

import "github.com/rohanthewiz/grmob/core"

// ConcernLinkInert is raised, in debug builds only, when a Link has neither a
// URL nor an OnTap. It is drawn in the link colour and announced as a link,
// and a tap does nothing — a promise the screen makes and does not keep,
// which on screen looks exactly like a working link.
const ConcernLinkInert = "link-inert"

// Link is a line of text that goes somewhere: a terms page, a help article, a
// "Forgot password?" under a sign-in form.
//
//	comps.Link{Text: "Privacy policy", URL: "https://example.com/privacy"}
//	comps.Link{Text: "Forgot password?", OnTap: showReset}
//
//	┌ Box  role=link  name=Text  onClick ┐
//	│  Text  (Primary's on-light tone)   │
//	└────────────────────────────────────┘
//
// # A link and not a ghost Button
//
// The two look alike and the difference is the one core.RoleLink's doc draws:
// a button does something here, a link goes somewhere else. A reader
// deciding whether to follow a control needs to know which, so the node
// carries RoleLink, which the web maps to role="link" and both natives to
// their link trait. StaticMap's tappable form made the same call.
//
// # OnTap or URL
//
// OnTap wins when it is set: an in-app destination (a Navigator push, a
// sheet) is a link too, and the caller knows how to get there. Otherwise a
// tap calls core.OpenURL(URL), which hands the address to the platform — the
// browser, or the app registered for the scheme (mailto:, tel:).
//
// # On its own line, and inside a sentence
//
// Rendered, a Link is a line of its own, not underlined: the link colour and
// the role carry the distinction, which is enough for a line that is nothing
// but the link. Inside running text use Link.Span, a run of a core.Paragraph
// in the same colour and underlined, because there the colour is the only
// other thing that says which words are the link.
//
// # Theme roles read
//
//	Ink          Colors.Primary's on-light tone (Variant.OnLight)
//	Type         Typography.Body
type Link struct {
	// Text is the visible link text and its accessible name.
	Text string

	// URL is opened with core.OpenURL when OnTap is nil.
	URL string

	// OnTap handles the tap instead of opening URL.
	OnTap func()

	// AccessibilityHint describes where the link goes when Text alone does not
	// ("Opens in your browser").
	AccessibilityHint string

	// Style is applied to the link text after its defaults.
	Style []core.StyleProp
}

// Render draws the link. It takes no hook slot.
// Span is this link as a run of a core.Paragraph: the same colour, underlined,
// and the same tap (OnTap, else opening URL), inside a sentence rather than
// on a line of its own.
//
//	core.Paragraph([]core.Span{
//	    {Text: "By continuing you accept the "},
//	    comps.Link{Text: "terms", URL: termsURL}.Span(ctx),
//	    {Text: "."},
//	})
//
// Underlined where the standalone Link is not: on its own line a link is
// told apart by being a line of its own in the link colour, and inside a
// sentence the colour is the only thing left, which a reader who cannot see
// it would miss (WCAG 1.4.1).
func (l Link) Span(ctx *core.Context) core.Span {
	tap := l.OnTap
	if tap == nil && l.URL != "" {
		url := l.URL
		tap = func() { core.OpenURL(url) }
	}
	if tap == nil {
		// Still a link, for the reason Render gives: a run that looks like a
		// link and cannot be pressed is worse than one that does nothing.
		tap = func() {}
	}
	return core.Span{
		Text:      l.Text,
		Underline: true,
		Color:     VariantDefault.OnLight(ctx.Theme()),
		OnTap:     tap,
	}
}

func (l Link) Render(ctx *core.Context) *core.Node {
	t := ctx.Theme()

	tap := l.OnTap
	if tap == nil && l.URL != "" {
		url := l.URL
		tap = func() { core.OpenURL(url) }
	}
	if tap == nil {
		if core.IsDebugMode() {
			core.ReportConcern(ConcernLinkInert,
				"Link \""+l.Text+"\" has neither URL nor OnTap, so it is drawn and announced as a link that goes nowhere")
		}
		// Still registered: a role=link container with no handler would be
		// a tab stop that Enter does nothing on, and the concern has already
		// said why.
		tap = func() {}
	}

	text := make([]core.StyleProp, 0, len(l.Style)+3)
	text = append(text,
		core.UseStyle(t.Typography.Body),
		core.TextColor(VariantDefault.OnLight(t)),
	)
	text = append(text, l.Style...)

	box := []core.PropsAndChildren{
		// No padding of its own: the theme Box base would otherwise inset a
		// link past the text above and below it.
		core.Padding(0),
		// Hug the text. A Column stretches its children across, and a link
		// stretched to the screen's width would take taps on the empty space
		// beside it — a reader aiming past the end of "Terms" lands in a
		// browser.
		core.AlignSelf(core.AlignItemsStart),
		core.AccessibilityRole(core.RoleLink),
		core.AccessibilityLabel(l.Text),
	}
	if l.AccessibilityHint != "" {
		box = append(box, core.AccessibilityHint(l.AccessibilityHint))
	}
	box = append(box,
		core.OnClick(tap),
		// Hidden because the Box's label already says it; left readable, the
		// text would be announced twice on the targets that do not merge a
		// labelled container's children.
		core.Text(l.Text, append(text, core.AccessibilityHidden())...),
	)
	return core.Box(box...).Render(ctx)
}
