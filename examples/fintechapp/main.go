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
	return core.SafeArea(
		core.Scroll(
			// Non-uniform spacing (24, 24, 28), so this column keeps explicit
			// Spacers. core.Gap sets one value for every gap in a container;
			// Spacer stays the right tool exactly when the gaps differ, and
			// that is the whole rule — Gap for uniform runs, Spacer for the
			// deliberate exception.
			core.Column(
				HeaderSection(ctx),
				core.Spacer(24),
				BalanceCard(ctx),
				core.Spacer(24),
				ActionsSection(ctx),
				core.Spacer(28),
				TransactionList(ctx),
			),
		),
	)
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
		MaterialButton("Transfer", t.Colors.Primary, "#FFF", func() {}),
		MaterialButton("Recharge", "#FFF", t.Colors.Secondary, func() {}),
	)
}

// MaterialButton is left hand-rolled rather than moved onto components.Chip.
// Chip is a *selection* affordance — it carries a Selected flag and renders a
// pressed-in variant, which is the filter-bar pattern it was extracted from.
// "Transfer" and "Recharge" are one-shot primary actions with no selected
// state, so a Chip here would be a Button wearing the wrong name.
func MaterialButton(label string, bg string, fg string, onClick func()) core.View {
	return core.Button(label,
		onClick,
		core.BackgroundColor(bg),
		core.TextColor(fg),
		core.Padding(12),
		core.BorderRadius(6),
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
			TransactionItem("Farmácia", "-750 MZN", t.Colors.Error, t.Colors.Background),
			TransactionItem("Transferência recebida", "+10,000 MZN", t.Colors.Secondary, t.Colors.TextPrimary),
			TransactionItem("Recarga de saldo", "+3,500 MZN", t.Colors.Secondary, t.Colors.TextPrimary),
		),
	)
}

// TransactionItem is built from two components: components.ListRow supplies
// the label-left / amount-right frame, and the amount itself is a
// components.Badge — a non-interactive status pill rather than a hand-styled
// Text. Badge owns the pill shape (an oversized radius that clamps to a
// stadium at any height) and the padding, so this example is left expressing
// only what is actually its subject: which theme color means what.
//
// The row previously pinned the amount with Justify(JustifyBetween). ListRow
// pins it with FlexGrow on its middle column instead, which is the same
// result here and the correct one in general: JustifyBetween spreads slack
// between *every* pair of children, so it silently stops meaning "pin the
// trailing element" the moment a row grows a third slot.
//
// ink is passed explicitly because Badge defaults its label color to the
// theme's Background, which is chosen to read on Primary. A semantic color
// picked per row — a dark red debit, a light teal credit — needs its own ink
// to stay legible, and Badge.TextColor is the slot for exactly that.
func TransactionItem(label, amount, color, ink string) core.View {
	return components.ListRow{
		Title:    label,
		Trailing: components.Badge{Text: amount, Color: color, TextColor: ink},
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
		},
		Typography: core.Typography{
			Title:    core.Style{FontSize: 24, FontWeight: core.Bold},
			Subtitle: core.Style{FontSize: 18},
			Body:     core.Style{FontSize: 14},
			Caption:  core.Style{FontSize: 12},
		},
		Spacing: core.SpacingScale{XS: 4, SM: 8, MD: 16, LG: 24, XL: 32},
	}
}
