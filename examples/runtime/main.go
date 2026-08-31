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

// renderTree runs one full render pass by hand. A host without a
// render.Manager (like this exporter) owns the whole pass boundary itself, and
// the four calls below are that boundary in order:
//
//	BeginRenderPass()      — callback ID counters restart, so the IDs a pass
//	                         hands the native side are stable pass to pass
//	                         ("txt_cb_0" is always this Input's onChange).
//	Reset()                — hook cursors restart; core.Render does this for us,
//	                         which is why the render entry point is core.Render
//	                         and not view.Render.
//	root.Render(ctx)       — components consume slots, cursor advances.
//	EndRenderPass()        — debug mode audits each context's cursor against its
//	                         slot count and against the previous pass. A no-op
//	                         (one atomic load) when debug mode is off, so it
//	                         costs nothing in production — call it always.
//	PurgeUnusedCallbacks() — drops handlers that were NOT re-registered in this
//	                         pass. Irrelevant on a single render, essential once
//	                         passes repeat: without it the registry grows with
//	                         every pass and keeps stale closures (and the state
//	                         they capture) alive forever.
func renderTree(ctx *core.Context) *core.Node {
	ctx.BeginRenderPass()
	node := core.Render(ctx, core.ComponentFunc(func(ctx *core.Context) *core.Node {
		return App(ctx).Render(ctx)
	}))
	ctx.EndRenderPass()
	ctx.PurgeUnusedCallbacks()
	return node
}

func main() {
	// Debug mode is process-wide and off by default. On, the pass boundary
	// above audits hook discipline (cursor drift) and container keys, recording
	// findings for DumpConcerns instead of panicking. Turning it on here makes
	// this example the end-to-end demonstration of the debug API.
	core.SetDebugMode(true)

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

	// Two passes in, the cursor audit has something to compare against. A
	// well-behaved app prints nothing here; DumpConcerns returns "" when clean,
	// so label the empty case explicitly rather than printing a blank line.
	fmt.Println("\nConcerns:")
	if dump := core.DumpConcerns(); dump != "" {
		fmt.Print(dump)
	} else {
		fmt.Println("(none)")
	}
}
