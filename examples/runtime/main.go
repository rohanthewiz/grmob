package main

import (
	"fmt"

	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/htmlout"
)

func App(ctx *core.Context) core.View {
	name := core.NewState(ctx, "")

	return core.Column(
		core.Text("Bem-vindo ao GrMob"),
		core.Input(name.Get(), "Digite o seu nome", func(val string) {
			name.Set(val)
		}),
		core.Text("Olá, "+name.Get()),
	)
}

// renderTree runs one full render pass by hand — a host without a
// render.Manager (like this HTML exporter) drives the pass boundary itself:
// BeginRenderPass so callback IDs restart at zero, then render from the root.
func renderTree(ctx *core.Context) *core.Node {
	ctx.BeginRenderPass()
	return core.Render(ctx, core.ComponentFunc(func(ctx *core.Context) *core.Node {
		return App(ctx).Render(ctx)
	}))
}

func main() {
	ctx := core.NewContext()

	// First render
	tree := renderTree(ctx)
	fmt.Println("Primeiro Render:")
	fmt.Println(htmlout.ExportHTML(tree))

	// Simula evento de input vindo do nativo, endereçado pelo callback ID que
	// o primeiro render registou ("txt_cb_0": primeiro text-callback do passo).
	ctx.ReceiveEventPayload(map[string]any{
		"callback": "txt_cb_0",
		"value":    "Ismael",
	})

	// Re-render após evento
	tree = renderTree(ctx)
	fmt.Println("\nApós evento de input:")
	fmt.Println(htmlout.ExportHTML(tree))
}
