package social

import (
	. "github.com/rohanthewiz/grmob/core"
)

func HomePage(ctx *Context) View {
	return Column(
		Text("🏠 Página Inicial", FontSize(24), FontWeight(Bold)),
		Spacer(12),
		Button("Abrir Detalhes", func() {
			Push(ctx, DetailsPage)
		}),
	)
}

func DetailsPage(ctx *Context) View {
	//counter := NewState(ctx, 0)

	return Column(
		Text("📄 Detalhes", FontSize(22), FontWeight(Bold)),
		Spacer(10),
		Spacer(8),
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
		Text("Nome: Ismael GraHms", FontSize(16)),
		Text("Profissão: Engenheiro de Software"),
	)
}
