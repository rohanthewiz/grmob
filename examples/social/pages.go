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

// DetailsPage is the pushed route, and it owns state — which used to be the
// whole lesson of this file.
//
// It once needed ctx.Scope("details") to survive: Navigator rendered every
// route into the *same* context, hook slots are positional, so a bare
// NewState(ctx, 0) here claimed slot 0 — the slot App's `currentTab` already
// owned. The two aliased: this counter read a string, and popping back found
// the tab state overwritten with an int.
//
// Navigator now gives each stack frame its own scope, so a route's hooks are
// its own by construction and the plain NewState below is correct. Two
// consequences are worth seeing in this example:
//
//   - Popping back to the shell still finds `currentTab` intact, because the
//     root frame never left the stack (TestNavigationKeepsRouteAndTabStateSeparate).
//   - Pushing Details again after a Pop starts the counter back at 0, because
//     the popped frame's state was discarded (TestPushedRouteStateIsDiscardedOnPop).
//     State that should outlive a screen belongs above the Navigator — see the
//     note in App.
//
// ctx.Scope is still the right tool for a *long-lived* named sub-tree within
// one screen; it is simply no longer load-bearing for route isolation.
func DetailsPage(ctx *Context) View {
	counter := NewState(ctx, 0)

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
