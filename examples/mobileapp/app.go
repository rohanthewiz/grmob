// Package mobileapp is the demo app bound into the Android AAR alongside the
// mobile bridge (see android/build.sh). Its only integration point is the
// init below: gomobile runs package inits when the native library loads, so
// by the time the Kotlin shell calls Mobile.renderInitial() the app is
// already registered. Replace this package in build.sh to ship your own app.
//
// The view deliberately exercises every event kind the bridge carries — void
// (Button), text (Input), bool (Checkbox), int (TabView) — plus the async
// push path (UseInterval ticking with no native event in flight) and the
// gap-5 renderer surface (List virtualization, container gestures via
// OnClick/OnLongPress, accessibility labels), so it doubles as a smoke test
// for the native runtimes.
package mobileapp

import (
	"fmt"
	"time"

	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/hooks"
	"github.com/rohanthewiz/grmob/mobile"
)

func init() {
	mobile.Register(core.NewContext(), App)
}

// AppName exists to be bindable. gobind only imports a bound package when it
// references at least one bindable exported symbol; App above is not bindable
// (function-typed parameters are unsupported), and with zero bindable symbols
// the package — including the init that registers the app — is never linked
// into the AAR, leaving the bridge with no app (nil manager) at runtime.
func AppName() string { return "GrMob Demo" }

// appHeader is a static banner routed through core.Cached: it renders once,
// and because Cached replays the identical *Node every pass, the reconciler's
// pointer-equality fast path skips the whole subtree on every re-render. It
// must be a package-level var — App runs on every pass, so a Cached built
// inside its body would be a fresh wrapper each time and cache nothing. Kept
// deliberately inert (no hooks, no callbacks, no theme reads): those are the
// things a cached view must not contain — see core.Cached.
var appHeader = core.Cached(core.Text("GrMob Demo", core.UseStyle(core.Style{
	FontSize:   17,
	FontWeight: core.Bold,
	Padding:    core.EdgeInsets{Top: 8, Bottom: 8, Left: 16, Right: 16},
})))

func App(ctx *core.Context) core.View {
	tab := core.NewState(ctx, 0)

	return core.SafeArea(
		core.Column(
			appHeader,
			tabs(tab),
		),
	)
}

func tabs(tab core.State[int]) core.View {
	return core.TabView(
		core.SelectedIndex(tab.Get()),
		core.OnTabChange(func(i int) { tab.Set(i) }),
		core.Tabs(
			core.Tab("Counter", ""),
			core.Tab("Form", ""),
			core.Tab("Feed", ""),
		),
		core.Content(
			counterTab(),
			formTab(),
			feedTab(),
		),
	)
}

func counterTab() core.View {
	return core.ComponentFunc(func(ctx *core.Context) *core.Node {
		count := core.NewState(ctx, 0)
		seconds := core.NewState(ctx, 0)

		// Drives the Go→native push channel: each tick re-renders with no
		// native event pending a response.
		hooks.UseInterval(ctx, func() { seconds.Set(seconds.Get() + 1) }, time.Second)

		return core.Column(
			core.Text(fmt.Sprintf("Count: %d", count.Get()), core.UseStyle(core.Style{
				FontSize:   28,
				FontWeight: core.Bold,
			})),
			core.Button("Increment", func() { count.Set(count.Get() + 1) }),
			core.Spacer(16),
			core.Text(fmt.Sprintf("App running for %ds", seconds.Get()), core.UseStyle(core.Style{
				FontSize:  13,
				TextColor: "#3C3C4399",
			})),
		).Render(ctx)
	})
}

// feedTab exercises the gap-5 surface: a virtualized List of keyed rows,
// row-level gestures (tap selects, long-press stars), and accessibility
// semantics a screen reader can traverse.
func feedTab() core.View {
	return core.ComponentFunc(func(ctx *core.Context) *core.Node {
		selected := core.NewState(ctx, 0)
		starred := core.NewState(ctx, 0)

		status := "Nothing selected"
		if selected.Get() > 0 {
			status = fmt.Sprintf("Selected: Article %d", selected.Get())
		}
		if starred.Get() > 0 {
			status += fmt.Sprintf(" · Starred: Article %d", starred.Get())
		}

		articles := make([]int, 30)
		for i := range articles {
			articles[i] = i + 1
		}

		return core.Column(
			core.Text(status, core.UseStyle(core.Style{
				FontSize:  13,
				TextColor: "#3C3C4399",
			})),
			// Decorative divider: hidden from the accessibility tree.
			core.Box(
				core.Height("1px"),
				core.BackgroundColor("#E5E5EA"),
				core.AccessibilityHidden(),
			),
			core.List(
				core.FlexGrow(1),
				core.For(articles, func(n int, _ int) core.View {
					title := fmt.Sprintf("Article %d", n)
					if starred.Get() == n {
						title += " ★"
					}
					label := fmt.Sprintf("Article %d", n)
					parts := []core.PropsAndChildren{
						core.Padding(12),
						// Selection restyles the row; Transition makes the
						// highlight fade in natively (Compose/SwiftUI drive
						// the frames — no patches during the animation).
						core.Transition(250, core.EaseInOut),
						core.OnClick(func() { selected.Set(n) }),
						core.OnLongPress(func() { starred.Set(n) }),
					}
					if selected.Get() == n {
						parts = append(parts, core.BackgroundColor("#E8F0FE"))
						label += ", selected"
					}
					parts = append(parts,
						core.AccessibilityLabel(label),
						core.AccessibilityHint("Selects the article; long-press to star it"),
						core.Text(title),
					)
					return core.Keyed(fmt.Sprintf("article-%d", n), core.Row(parts...))
				}),
			),
		).Render(ctx)
	})
}

func formTab() core.View {
	return core.ComponentFunc(func(ctx *core.Context) *core.Node {
		name := core.NewState(ctx, "")
		subscribed := core.NewState(ctx, false)

		greeting := "Hello, stranger."
		if name.Get() != "" {
			greeting = "Hello, " + name.Get() + "!"
		}
		subLabel := "Not subscribed"
		if subscribed.Get() {
			subLabel = "Subscribed"
		}

		return core.Column(
			core.Input(name.Get(), "Your name", func(v string) { name.Set(v) }),
			core.Spacer(8),
			core.Text(greeting),
			core.Spacer(8),
			core.Row(
				core.Checkbox(subscribed.Get(), func(v bool) { subscribed.Set(v) }),
				core.Text(subLabel),
			),
		).Render(ctx)
	})
}
