//go:build js && wasm

package main

import (
	"encoding/json"
	"log"

	"github.com/rohanthewiz/grmob/core"
	// The mounted app. Point this dot-import at any package in examples/
	// that exports an App root view — examples/social was the previous
	// occupant — and rebuild to switch what the browser shows.
	. "github.com/rohanthewiz/grmob/examples/tutorial"
	"github.com/rohanthewiz/grmob/render"
	"syscall/js"
)

var (
	ctx = core.NewContext().WithTheme(core.DefaultTheme)
)

var manager *render.Manager

func renderInitial(this js.Value, args []js.Value) any {
	// Re-initialization (the page calling RenderInitial again): close the
	// previous manager first, which stops its pump and the hook resources
	// (interval tickers, timeouts) registered on the shared ctx. This
	// replaces the old global hooks.ClearIntervals() sweep — cleanup is
	// per-context-tree now, reached through the manager that owns it.
	if manager != nil {
		manager.Close()
	}
	// App is examples/social's root view (reached through the dot-import
	// above). The app tree belongs to the example package; this file is host
	// wiring only — JS bindings, the manager, and the event bridge.
	manager = render.New(ctx, App)
	// Push channel: if the host page defines GrMobApplyPatches, async state
	// changes (timers, goroutines) are pushed to it as patch JSON instead of
	// relying on the IsDirty polling loop. Pages without it keep polling —
	// the manager never consumes a diff unless a listener is attached.
	if js.Global().Get("GrMobApplyPatches").Type() == js.TypeFunction {
		manager.SetListener(jsPatchListener{})
	}
	out := manager.RenderInitial()
	return js.ValueOf(out)
}

// jsPatchListener forwards pushed patches to the page's GrMobApplyPatches
// handler. ApplyPatches runs on the pump goroutine, which on js/wasm is
// scheduled cooperatively on the single JS thread, so calling into JS here is
// safe without extra marshalling.
type jsPatchListener struct{}

func (jsPatchListener) ApplyPatches(patches string) {
	js.Global().Call("GrMobApplyPatches", patches)
}
func RequestPermission(p Permission, onResult func(granted bool)) {
	js.Global().Call("GrMobRequestPermission", string(p), js.FuncOf(func(this js.Value, args []js.Value) any {
		granted := args[0].Bool()
		onResult(granted)
		return nil
	}))
}

func isDirty(this js.Value, args []js.Value) any {
	return js.ValueOf(ctx.IsDirty())
}

func renderAgain(this js.Value, args []js.Value) any {
	out := manager.RenderAgain()
	return js.ValueOf(out)
}

type Permission string

const (
	PermissionCamera      Permission = "camera"
	PermissionMicrophone  Permission = "microphone"
	PermissionGeolocation Permission = "geolocation"
)

func receiveEvent(this js.Value, args []js.Value) any {
	id := args[0].String()
	payloadStr := args[1].String()

	var payload map[string]any
	err := json.Unmarshal([]byte(payloadStr), &payload)
	if err != nil {
		println("Erro ao fazer parse do payload JSON:", err.Error())
		return nil
	}

	// Dispatch through the app's own context: the callback registry is
	// per-context-tree state now, not a package global.
	//
	// Guarded because this host dispatches directly rather than through
	// render.Manager's Dispatch* (which carry the same guard): a panic
	// escaping a handler here would unwind into the js.Func callback and
	// abort the Go runtime, taking the page's app with it. The same partial
	// recovery caveat applies as on the native path — the handler's work is
	// abandoned wherever it stopped, and the next render shows whatever state
	// actually exists.
	if rerr := core.Guard(func() {
		ctx.ReceiveEventPayload(map[string]any{
			"callback": id,
			"value":    payload["value"],
		})
	}); rerr != nil {
		log.Printf("grmob: recovered panic in handler %s: %v\n%s", id, rerr.Value, rerr.Stack)
	}
	return nil
}

func registerCallbacks() {
	js.Global().Set("GrMobWASM", map[string]any{
		"RenderInitial": js.FuncOf(renderInitial),
		"RenderAgain":   js.FuncOf(renderAgain),
		"ReceiveEvent":  js.FuncOf(receiveEvent),
		"IsDirty":       js.FuncOf(isDirty),
	})
}

func main() {
	c := make(chan struct{})
	registerCallbacks()
	println("GrMob WASM ready.")
	<-c
}

func TabsComponent(ctx *core.Context, activeTab core.State[string]) core.View {
	tabButton := func(label, key string) core.View {
		active := activeTab.Get() == key
		return core.Button(label, func() {
			activeTab.Set(key)
		},
			core.Padding(10),
			core.Margin(4),
			core.BorderRadius(6),
			core.FontWeight(core.Bold),
			core.BackgroundColor(ifThen(active, "#007AFF", "#E0E0E0")),
			core.TextColor(ifThen(active, "#FFFFFF", "#000000")),
		)
	}

	return core.Column(
		core.Text("🗂️ Selecione uma aba:", core.FontSize(20), core.Margin(8)),

		core.Row(
			tabButton("Informações", "info"),
			tabButton("Configurações", "settings"),
			tabButton("Ajuda", "help"),
		),

		core.Spacer(16),

		core.Match(activeTab.Get(),
			core.Case("info", core.Text("📘 Esta é a aba de informações.")),
			core.Case("settings", core.Text("⚙️ Configurações do sistema.")),
			core.Case("help", core.Text("🆘 Ajuda e suporte técnico.")),
			core.Default[string](core.Text("❓ Aba desconhecida")),
		),
	)
}

func ifThen(cond bool, a, b string) string {
	if cond {
		return a
	}
	return b
}

func HomeScreen(ctx *core.Context) core.View {
	return core.Column(
		core.Text("🏠 Tela Inicial", core.FontSize(22), core.FontWeight(core.Bold)),
		core.Spacer(12),
		core.Button("Ir para Detalhes", func() {
			core.Push(ctx, DetailsScreen)
		}),
	)
}

func DetailsScreen(ctx *core.Context) core.View {

	return core.Column(
		core.Text("📄 Tela de Detalhes", core.FontSize(20), core.FontWeight(core.Bold)),
		core.Spacer(8),
		core.Button("Incrementar", func() {

		}),
		core.Spacer(12),
		core.Button("⬅️ Voltar", func() {
			core.Pop(ctx)
		}),
	)
}
