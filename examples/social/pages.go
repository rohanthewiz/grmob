package social

import (
	"strconv"

	. "github.com/rohanthewiz/grmob/core"
)

func HomePage(ctx *Context) View {
	return Column(
		// Uniform spacing, so it is a Gap on the container rather than filler
		// views between the children.
		Gap(12),
		Text("🏠 Página Inicial", FontSize(24), FontWeight(Bold)),
		Button("Abrir Detalhes", func() {
			Push(ctx, DetailsPage)
		}),
	)
}

// DetailsPage is the pushed route, and it owns state — which is the whole
// lesson of this file.
//
// Navigator renders whichever route is on top into the *same* context the
// Navigator's initial route uses. Hook slots are positional, so a bare
// NewState(ctx, 0) here would claim slot 0 — the slot App's `currentTab`
// already owns. The two would alias: this counter would read a string, and
// popping back would find the tab state overwritten with an int.
//
// ctx.Scope("details") is the fix: a *named* child context, created on first
// use and stable forever after. A scope that renders on some passes and not
// others shifts nothing, because its slots are its own. That is exactly the
// route/screen case.
//
// The tripwire for getting this wrong is debug mode's cursor-drift check
// (core.SetDebugMode(true) + core.DumpConcerns()): a route allocating hooks on
// the shared context changes that context's hook count between passes, which
// is what the audit reports.
func DetailsPage(ctx *Context) View {
	sc := ctx.Scope("details")
	counter := NewState(sc, 0)

	return Column(
		Gap(10),
		Text("📄 Detalhes", FontSize(22), FontWeight(Bold)),
		Text("Contador: "+strconv.Itoa(counter.Get()), FontSize(16)),
		Button("➕ Incrementar", func() {
			counter.Set(counter.Get() + 1)
		}),
		Button("⬅️ Voltar", func() {
			Pop(ctx)
		}),
	)
}

func SearchPage(ctx *Context) View {
	return Column(
		Text("🔍 Pesquisa", FontSize(24), FontWeight(Bold)),
		Input("", "Digite algo...", func(val string) {}),
	)
}

func ProfilePage(ctx *Context) View {
	return Column(
		Text("👤 Perfil", FontSize(24), FontWeight(Bold)),
		Text("Nome: Fulano de Tal", FontSize(16)),
		Text("Profissão: Engenheiro de Software"),
	)
}
