package components

import "github.com/rohanthewiz/grmob/core"

// Banner is the inline strip that tells the user something about the screen
// they are on: a failed refresh over content that is still good, a
// "Reconnecting…", an offline notice, a "Your changes were saved".
//
//	components.Banner{Text: "Could not refresh. Showing saved copy.",
//	    Variant: components.VariantWarning, ActionLabel: "Retry", OnAction: reload}
//
//	┌ Row ─────────────────────────────────────────────────────────┐
//	│ ⚠  ┌ Box FlexGrow(1) ───────────────┐  [Retry]  [✕]          │
//	│    │ Text                           │                        │
//	│    └────────────────────────────────┘                        │
//	└──────────────────────────────────────────────────────────────┘
//
// # It is not a toast
//
// core.ShowToast reaches the platform's own transient overlay and disappears
// on a timer. A Banner is part of the tree: it stays until the state that
// produced it changes, which is what a condition the user may need to act on
// requires. Use the toast for "Copied", the banner for "You are offline".
//
// # The variant is a tint, not a fill
//
// Badge and a filled Button spend the whole variant color as a background.
// A strip that runs the width of the screen cannot: a saturated Error red
// across a screen reads as a failure of the app rather than of one fetch, and
// the palette carries no muted container tone to fill with instead (one value
// per role — see Button's note on what a second "on-light" tone would fix).
//
// So the variant is spent on the edges: a hairline border and the leading
// glyph take the role color, the fill stays the theme's Surface, and the text
// keeps the primary ink so it is legible whatever the role. That also means a
// Banner's contrast does not depend on which variant it is, which the
// alternatives could not promise.
//
// # Color is not the message, again
//
// Nothing here announces "error" to a screen reader: the glyph is marked
// decorative (a reader saying "circled times" is worse than silence) and a
// border has no voice. Text must therefore carry the meaning on its own —
// "Could not refresh", not "Something went wrong" next to a red edge. Same
// rule Badge documents, and the same WCAG 1.4.1 behind it.
//
// # It announces itself when it appears
//
// A banner is a live region: it turns up because something changed, usually
// while the reader is somewhere else on the screen, and a message nobody is
// looking at is a message nobody gets. So the strip carries
// core.RoleAlert when its variant is Error and core.RoleStatus otherwise —
// the same split the variant already draws visually, in the one vocabulary
// that has a word for "interrupt" and a word for "mention at the next pause".
//
// Error interrupts because a failed action is the case where continuing is
// the wrong thing to do; everything else waits, because "Saved" arriving
// mid-sentence is how a live region becomes a thing users switch off.
//
// The role is on the strip rather than on the message so that an appearing
// banner is announced whole — the text, and the label of any action beside
// it, which is the part the reader needs in order to know what to do about
// it.
//
// It is a default, not a fixture. Style is applied after the widget's own
// props, so a caller can name a different role, or core.RoleNone for a strip
// that is really static content and should not interrupt anything.
type Banner struct {
	// Text is the message. Content is the escape hatch for the growing middle
	// — an arbitrary view in its place, taking precedence when set.
	Text    string
	Content core.View

	// Variant selects the semantic role: Success, Warning, Error, or the zero
	// value for the theme's Primary — the neutral "here is some information"
	// strip.
	Variant Variant

	// Glyph is the leading mark. Empty takes the variant's default (ⓘ ✓ ⚠ ⊗);
	// NoGlyph drops it entirely, for a strip that should read as quietly as
	// possible.
	Glyph   string
	NoGlyph bool

	// ActionLabel and OnAction render a single trailing action — "Retry",
	// "Reconnect", "Undo". Action is the slot form and takes precedence.
	//
	// The built button is a *default* ghost, not one tinted with the banner's
	// variant. The strip already says what kind of thing it is twice, in the
	// border and the glyph; a third telling would put the least legible
	// combination this package has (see Button's contrast table — outlined
	// and ghost own neither their fill nor their backdrop) on the one control
	// the user is meant to hit.
	ActionLabel string
	OnAction    func()
	Action      core.View

	// OnDismiss adds a trailing ✕ that closes the banner. It is the caller's
	// job to stop rendering the widget; nothing here holds state.
	OnDismiss func()

	// Style is applied after the widget's own treatment, so any of it can be
	// overridden. A caller who wants an edge-to-edge strip with no frame —
	// the shape a banner pinned directly under an AppBar usually wants —
	// spells that out:
	//
	//	Style: []core.StyleProp{core.BorderWidth(0), core.BorderRadius(0)}
	Style []core.StyleProp
}

func (b Banner) Render(ctx *core.Context) *core.Node {
	t := ctx.Theme()
	accent := b.Variant.Color(t)

	items := make([]core.PropsAndChildren, 0, len(b.Style)+8)
	items = append(items,
		core.BackgroundColor(t.Colors.Surface),
		core.BorderColor(accent),
		core.BorderWidth(1),
		// The theme's own surface radius rather than a literal: Card is where
		// a theme states how round a panel is (12 under DefaultTheme, 8 under
		// MaterialTheme), and a banner is a panel. A theme that leaves it zero
		// gets square corners, which is that theme's answer, not a bug.
		core.BorderRadius(t.Components.Card.BorderRadius),
		core.AlignItemsProp(core.AlignItemsCenter),
		core.Gap(float64(t.Spacing.SM)),
		core.AccessibilityRole(b.liveRegion()),
	)
	for _, sp := range b.Style {
		items = append(items, sp)
	}

	if glyph := b.glyph(); glyph != "" {
		items = append(items, core.Text(glyph,
			core.TextColor(accent),
			core.FontSize(t.Typography.Body.FontSize),
			// Decoration: the meaning is in Text. See the type comment.
			core.AccessibilityHidden(),
		))
	}

	middle := b.Content
	if middle == nil {
		middle = core.Text(b.Text,
			core.UseStyle(t.Typography.Body),
			core.TextColor(t.Colors.TextPrimary),
		)
	}
	// FlexGrow on the message is what pins the controls to the trailing edge
	// in every configuration — the same technique, and the same reason, as
	// ListRow's middle column.
	items = append(items, core.Box(core.FlexGrow(1), middle))

	switch {
	case b.Action != nil:
		items = append(items, b.Action)
	case b.ActionLabel != "" && b.OnAction != nil:
		items = append(items, Button{
			Label:    b.ActionLabel,
			Emphasis: EmphasisGhost,
			OnTap:    b.OnAction,
			Style:    []core.StyleProp{core.FontSize(t.Typography.Caption.FontSize)},
		})
	}

	if b.OnDismiss != nil {
		items = append(items, Button{
			Label:              "✕",
			Emphasis:           EmphasisGhost,
			OnTap:              b.OnDismiss,
			AccessibilityLabel: "Dismiss",
			Style: []core.StyleProp{
				core.TextColor(t.Colors.TextSecondary),
				core.PaddingHorizontal(6),
				core.PaddingVertical(0),
			},
		})
	}

	return core.Row(items...).Render(ctx)
}

// glyph resolves the leading mark: the caller's, the variant's default, or
// none.
//
// The defaults are shape-distinct as well as color-distinct — a check, a
// triangle, a circle — so the four roles stay apart for a reader who cannot
// tell the tints apart. They are still decoration (nothing announces them);
// what they buy is a second visual channel, not an accessible one.
func (b Banner) glyph() string {
	if b.NoGlyph {
		return ""
	}
	if b.Glyph != "" {
		return b.Glyph
	}
	switch b.Variant {
	case VariantSuccess:
		return "✓"
	case VariantWarning:
		return "⚠"
	case VariantError:
		return "⊗"
	default:
		return "ⓘ"
	}
}

// liveRegion is how loudly this banner interrupts, from the variant it is
// already tinted by. See the type comment for why Error alone cuts in.
func (b Banner) liveRegion() core.Role {
	if b.Variant == VariantError {
		return core.RoleAlert
	}
	return core.RoleStatus
}
