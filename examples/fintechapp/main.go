package main

import (
	"os"

	"github.com/rohanthewiz/grmob/components"
	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/htmlout"
)

func main() {
	ctx := core.NewContext().With(
		core.WithThemeOpt(MaterialTheme()),
		core.WithConfigOpt(&core.AppConfig{
			Name:        "GrMob Material Wallet",
			Description: "A Material Design wallet interface built with GrMob",
			Version:     "1.0",
		}),
	)

	// core.Render is the documented render entry point: it resets the hook
	// cursor before rendering, so it stays correct when called on every pass,
	// not just the first. This exporter renders once, but view.Render(ctx) is
	// what people copy into hosts that re-render — where a missing reset means
	// every hook reads the previous pass's slot.
	node := core.Render(ctx, App(ctx))
	html := htmlout.ExportHTML(node)
	_ = os.WriteFile("materialwallet.html", []byte(html), 0644)
}

func App(ctx *core.Context) core.View {
	// The one screen here that scrolls as a whole: its content is a single
	// run of sections with nothing pinned, so Screen.Scroll owns the whole
	// viewport rather than a region inside it.
	//
	// Screen.Gap is left unset. The spacing is non-uniform (24, 24, 28), so
	// this column keeps explicit Spacers — Gap sets one value for every gap in
	// a container, and Spacer stays the right tool exactly when the gaps
	// differ. That is the whole rule: Gap for uniform runs, Spacer for the
	// deliberate exception.
	return components.Screen{
		Scroll: true,
		Children: []core.View{
			HeaderSection(ctx),
			core.Spacer(24),
			BalanceCard(ctx),
			core.Spacer(24),
			ActionsSection(ctx),
			core.Spacer(28),
			TransactionList(ctx),
		},
	}
}

func HeaderSection(ctx *core.Context) core.View {
	t := ctx.Theme()
	// Non-uniform spacing (12 under the avatar, 4 between the two text lines):
	// Spacer, per the rule stated in App.
	return core.Column(
		core.Image("https://dummyimage.com/60x60/6200EE/ffffff&text=G"),
		core.Spacer(12),
		core.Text("GrMob Wallet", core.FontSize(t.Typography.Title.FontSize), core.FontWeight(t.Typography.Title.FontWeight), core.TextColor(t.Colors.TextPrimary)),
		core.Spacer(4),
		core.Text("Welcome back, Ismael", core.FontSize(15), core.TextColor(t.Colors.TextSecondary)),
	)
}

// BalanceCard stays on core.Card rather than components.Card. The component
// earns its keep when a card has distinct header/body/footer regions; this one
// is a caption above a figure, and components.Card would render the caption in
// the theme's *Subtitle* role — visibly wrong for what is a caption. Reaching
// for a component that does not fit teaches the wrong lesson.
func BalanceCard(ctx *core.Context) core.View {
	t := ctx.Theme()
	return core.Card(
		core.Column(
			core.Gap(8),
			core.Text("Available Balance", core.FontSize(12), core.TextColor(t.Colors.TextSecondary)),
			core.Text("MZN 42,750.00", core.FontSize(24), core.FontWeight(core.Bold), core.TextColor(t.Colors.Primary)),
		),
	)
}

func ActionsSection(ctx *core.Context) core.View {
	t := ctx.Theme()
	return core.Row(
		core.Gap(12),
		// The local MaterialButton helper this replaces took a background and
		// a matching foreground per call — the two obligations components.Button
		// removes. "Transfer" is the primary action and takes the theme's
		// Button base untouched.
		components.Button{Label: "Transfer", OnTap: func() {}},
		// "Recharge" is the secondary action, and its treatment is the one
		// case the semantic variants deliberately do not cover: Secondary is a
		// brand slot, not a status role, so there is no VariantSecondary to
		// ask for. Style is the documented escape hatch — an outlined button
		// re-tinted to the brand's second color.
		components.Button{
			Label:    "Recharge",
			OnTap:    func() {},
			Emphasis: components.EmphasisOutlined,
			Style: []core.StyleProp{
				core.TextColor(t.Colors.Secondary),
				core.BorderColor(t.Colors.Secondary),
			},
		},
	)
}

func TransactionList(ctx *core.Context) core.View {
	t := ctx.Theme()
	return core.Column(
		core.Text("Recent Transactions", core.TextColor(t.Colors.TextPrimary), core.FontSize(16), core.FontWeight(core.Bold)),
		// One deliberate 16 under the heading, then a uniform 12 between rows:
		// the two runs have different values, so they are expressed by
		// different means — a Spacer for the one-off, Gap on the row list.
		core.Spacer(16),
		core.Column(
			// Row spacing belongs to the list, not to the row. Before, each
			// TransactionItem carried a trailing Spacer(12) of its own, which
			// also put a phantom gap after the last row and made the item
			// unusable anywhere the spacing differed. Gap is what replaces
			// that pattern.
			core.Gap(12),
			TransactionItem("Farmácia", "-750 MZN", components.VariantError),
			// Credits take Success, not Secondary. They read the same under
			// the old palette only by accident: Secondary is a *brand* slot a
			// theme may set to any hue, so a rebrand to magenta would have
			// turned "money in" magenta. Success carries the meaning.
			TransactionItem("Transferência recebida", "+10,000 MZN", components.VariantSuccess),
			TransactionItem("Recarga de saldo", "+3,500 MZN", components.VariantSuccess),
		),
	)
}

// TransactionItem is built from two components: components.ListRow supplies
// the label-left / amount-right frame, and the amount itself is a
// components.Badge — a non-interactive status pill rather than a hand-styled
// Text. Badge owns the pill shape (an oversized radius that clamps to a
// stadium at any height) and the padding, so this example is left expressing
// only what is actually its subject: which status each row carries.
//
// The row previously pinned the amount with Justify(JustifyBetween). ListRow
// pins it with FlexGrow on its middle column instead, which is the same
// result here and the correct one in general: JustifyBetween spreads slack
// between *every* pair of children, so it silently stops meaning "pin the
// trailing element" the moment a row grows a third slot.
//
// The row names a *variant* rather than a pair of colors. It used to pass a
// background and a matching ink pulled off the theme by hand, which put two
// obligations on every call site: know which palette role means "money in",
// and know which ink stays legible on it. Both now live in the widget — the
// fill comes from the palette's status role and the ink is picked by contrast
// against it — so this function is left expressing only its actual subject,
// which is that a debit is an error and a credit is a success.
func TransactionItem(label, amount string, variant components.Variant) core.View {
	return components.ListRow{
		Title:    label,
		Trailing: components.Badge{Text: amount, Variant: variant},
	}
}

func MaterialTheme() *core.Theme {
	return &core.Theme{
		Colors: core.ColorPalette{
			Primary:       "#6200EE",
			Secondary:     "#03DAC6",
			Error:         "#B00020",
			TextPrimary:   "#000000",
			TextSecondary: "#666666",
			Background:    "#FFFFFF",
			Surface:       "#F5F5F5",
			// The status triad. Error was already here; Success and Warning
			// complete it, so a credit row can name what it *means* instead of
			// borrowing the brand's Secondary. Border gives the hairline its
			// own role rather than leaning on Surface, which is a fill.
			Border:  "#E0E0E0",
			Success: "#2E7D32",
			Warning: "#EF6C00",
		},
		Typography: core.Typography{
			Title:    core.Style{FontSize: 24, FontWeight: core.Bold},
			Subtitle: core.Style{FontSize: 18},
			Body:     core.Style{FontSize: 14},
			Caption:  core.Style{FontSize: 12},
		},
		Spacing: core.SpacingScale{XS: 4, SM: 8, MD: 16, LG: 24, XL: 32},
		// This theme had no Components block at all, which stayed invisible
		// only because every widget in the app was hand-styled at the call
		// site. Moving the action row onto components.Button made it visible
		// immediately: the widget's zero value deliberately applies *nothing*
		// so a theme's own Button base carries the look, and here there was no
		// base — so the button rendered with no style whatsoever.
		//
		// These are the values the local MaterialButton helper used to hardcode
		// per call, so the rendered result is unchanged; they now live in one
		// place and retint with the theme.
		//
		// The rest of ComponentDefaults (Card, Input, Row, Column, ...) is
		// still empty. Nothing in this app reads them yet, but the same
		// surprise waits there for the next widget that does.
		Components: core.ComponentDefaults{
			Button: core.Style{
				Background:   "#6200EE",
				TextColor:    "#FFF",
				Padding:      core.EdgeInsets{Top: 12, Bottom: 12, Left: 12, Right: 12},
				BorderRadius: 6,
			},
		},
	}
}
