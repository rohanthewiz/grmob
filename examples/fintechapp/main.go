package main

import (
	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/htmlout"
	"os"
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

	node := App(ctx).Render(ctx)
	html := htmlout.ExportHTML(node)
	_ = os.WriteFile("materialwallet.html", []byte(html), 0644)
}

func App(ctx *core.Context) core.View {
	return core.SafeArea(
		core.Scroll(
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
	return core.Column(
		core.Image("https://dummyimage.com/60x60/6200EE/ffffff&text=G"),
		core.Spacer(12),
		core.Text("GrMob Wallet", core.FontSize(t.Typography.Title.FontSize), core.FontWeight(t.Typography.Title.FontWeight), core.TextColor(t.Colors.TextPrimary)),
		core.Spacer(4),
		core.Text("Welcome back, Ismael", core.FontSize(15), core.TextColor(t.Colors.TextSecondary)),
	)
}

func BalanceCard(ctx *core.Context) core.View {
	t := ctx.Theme()
	return core.Card(
		core.Column(
			core.Text("Available Balance", core.FontSize(12), core.TextColor(t.Colors.TextSecondary)),
			core.Spacer(8),
			core.Text("MZN 42,750.00", core.FontSize(24), core.FontWeight(core.Bold), core.TextColor(t.Colors.Primary)),
		),
	)
}

func ActionsSection(ctx *core.Context) core.View {
	t := ctx.Theme()
	return core.Row(
		MaterialButton("Transfer", t.Colors.Primary, "#FFF", func() {}),
		core.Spacer(12),
		MaterialButton("Recharge", "#FFF", t.Colors.Secondary, func() {}),
	)
}

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
		core.Spacer(16),
		TransactionItem("Farmácia", "-750 MZN", t.Colors.Error),
		TransactionItem("Transferência recebida", "+10,000 MZN", t.Colors.Secondary),
		TransactionItem("Recarga de saldo", "+3,500 MZN", t.Colors.Secondary),
	)
}

func TransactionItem(label, amount, color string) core.View {
	return core.Column(
		core.Text(label),
		core.Spacer(4),
		core.Text(amount, core.TextColor(color)),
		core.Spacer(12),
	)
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
