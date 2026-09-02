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

	"github.com/rohanthewiz/grmob/components"
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

	// No Gap and no Fill: the header sits directly on the tab view, and the
	// TabView owns the height it needs. The zero value is the whole scaffold
	// here, which is exactly the case that used to be four lines of nesting.
	return components.Screen{
		Children: []core.View{
			appHeader,
			tabs(tab),
		},
	}
}

func tabs(tab core.State[int]) core.View {
	return core.TabView(
		core.SelectedIndex(tab.Get()),
		core.OnTabChange(func(i int) { tab.Set(i) }),
		core.Tabs(
			core.Tab("Counter", ""),
			core.Tab("Form", ""),
			core.Tab("Feed", ""),
			core.Tab("Audio", ""),
		),
		core.Content(
			counterTab(),
			formTab(),
			feedTab(),
			audioTab(),
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
			// Non-uniform spacing, so this one stays a Spacer: the count and
			// its button are one block with nothing between them, and the 16
			// separates that block from the passive timer line below. core.Gap
			// sets a single value for *every* gap in a container, so it cannot
			// express "nothing here, 16 there" — that is precisely where
			// Spacer keeps its job (contrast formTab, a uniform run on Gap).
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
			// Decorative divider. components.Separator supplies the hairline
			// tint, so the color is no longer written out per package.
			components.Separator{},
			core.List(
				core.FlexGrow(1),
				core.For(articles, func(n int, _ int) core.View {
					title := fmt.Sprintf("Article %d", n)
					if starred.Get() == n {
						title += " ★"
					}
					// A components.ListRow, which collapses what this used to
					// hand-roll in two ways. The row was built by appending to
					// a []core.PropsAndChildren so the selected background
					// could be added conditionally — core.If emits a real
					// child node, so it was not usable for a *style* prop.
					// (core.MaybeProp is the general answer to that now;
					// ListRow.SelectedStyle is the same conditional declared,
					// which is better still because the widget owns it.)
					// And the ", selected" suffix on the accessibility label
					// was appended by hand in the same branch; ListRow owns
					// that convention, so the label here is just the name.
					return core.Keyed(fmt.Sprintf("article-%d", n), components.ListRow{
						Title:       title,
						Selected:    selected.Get() == n,
						OnTap:       func() { selected.Set(n) },
						OnLongPress: func() { starred.Set(n) },
						Style: []core.StyleProp{
							core.Padding(12),
							// Selection restyles the row; Transition makes the
							// highlight fade in natively (Compose/SwiftUI drive
							// the frames — no patches during the animation).
							core.Transition(250, core.EaseInOut),
						},
						// Overrides ListRow's theme default (a Surface tint):
						// this demo wants the same pale blue the todoapp
						// filter chips use.
						SelectedStyle: []core.StyleProp{
							core.BackgroundColor("#E8F0FE"),
						},
						AccessibilityLabel: fmt.Sprintf("Article %d", n),
						AccessibilityHint:  "Selects the article; long-press to star it",
					})
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

		// Uniform 8 between every child, so the spacing is one prop on the
		// container rather than filler views between the children: fewer nodes
		// for the reconciler to walk each pass, and no way to add a field and
		// forget its separator.
		return core.Column(
			core.Gap(8),
			core.Input(name.Get(), "Your name", func(v string) { name.Set(v) }),
			core.Text(greeting),
			// Checkbox plus its label is the ListRow shape at its smallest:
			// a leading control and a title, no trailing slot. Worth the
			// widget even here, because it is ListRow that vertically centres
			// the box against the text — the bare Row this replaced left both
			// on their top edges.
			components.ListRow{
				Leading: core.Checkbox(subscribed.Get(), func(v bool) { subscribed.Set(v) }),
				Title:   subLabel,
			},
		).Render(ctx)
	})
}

// demoTrack is a freely licensed sample stream (SoundHelix's test songs are
// published for exactly this purpose). Any HTTP(S) URL that serves ranges
// works — seeking needs Range requests.
var demoTrack = core.AudioTrack{
	URL:    "https://www.soundhelix.com/examples/mp3/SoundHelix-Song-1.mp3",
	Title:  "SoundHelix Song 1",
	Artist: "SoundHelix",
	Album:  "GrMob Demo",
}

// audioTab exercises core's audio service end to end on each native shell:
// load, play/pause, seek by slider, skip, speed, stop — with the status
// ticks arriving over the host-event channel (mobile.ReportHostEvent) and
// re-rendering through hooks.UseAudio. Background the app while it plays to
// see the lock-screen controls.
func audioTab() core.View {
	return core.ComponentFunc(func(ctx *core.Context) *core.Node {
		status := hooks.UseAudio(ctx)
		// The slider's own value while the thumb is down, or -1. Kept in
		// state so the time label follows the drag; the seek itself happens
		// once, on release.
		scrub := core.NewState(ctx, -1.0)

		mine := status.Track.URL == demoTrack.URL
		position := status.Position
		if scrub.Get() >= 0 {
			position = scrub.Get()
		}

		line := "Nothing loaded"
		if mine {
			line = string(status.State)
			if status.State == core.AudioError {
				line += ": " + status.Error
			}
		}

		playLabel := "Play"
		if mine && status.State == core.AudioPlaying {
			playLabel = "Pause"
		}
		onPlay := func() {
			if mine {
				core.AudioToggle()
			} else {
				core.AudioLoad(demoTrack)
			}
		}

		muted := core.UseStyle(core.Style{FontSize: 13, TextColor: "#3C3C4399"})
		return core.Column(
			core.Gap(8),
			core.Text(demoTrack.Title, core.UseStyle(core.Style{FontSize: 17, FontWeight: core.Bold})),
			core.Text(line, muted),
			core.Slider(position, 0, status.Duration,
				func(v float64) { scrub.Set(v) },
				core.OnSliderChangeEnd(func(v float64) {
					scrub.Set(-1)
					core.AudioSeek(v)
				}),
				core.Width("100%"),
				core.Disabled(!mine || status.Duration <= 0),
			),
			core.Row(
				// Full width, or the Row hugs its two labels and
				// space-between has nothing to distribute.
				core.Width("100%"),
				core.Justify(core.JustifyBetween),
				core.Text(clock(position), muted),
				core.Text(clock(status.Duration), muted),
			),
			core.Row(
				core.Gap(8),
				core.Button("−15s", func() { core.AudioSkip(-15) }, core.Disabled(!mine)),
				core.Button(playLabel, onPlay),
				core.Button("+15s", func() { core.AudioSkip(15) }, core.Disabled(!mine)),
			),
			core.Row(
				core.Gap(8),
				core.Button(fmt.Sprintf("Speed %gx", status.Rate), func() {
					core.AudioSetRate(nextRate(status.Rate))
				}, core.Disabled(!mine)),
				core.Button("Stop", core.AudioStop, core.Disabled(!mine)),
			),
		).Render(ctx)
	})
}

// nextRate cycles 1 → 1.25 → 1.5 → 2 → 0.75 → 1.
func nextRate(r float64) float64 {
	switch {
	case r < 1:
		return 1
	case r < 1.25:
		return 1.25
	case r < 1.5:
		return 1.5
	case r < 2:
		return 2
	default:
		return 0.75
	}
}

// clock formats seconds as m:ss (or h:mm:ss past an hour).
func clock(seconds float64) string {
	if seconds < 0 {
		seconds = 0
	}
	s := int(seconds + 0.5)
	h, m, sec := s/3600, (s%3600)/60, s%60
	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, sec)
	}
	return fmt.Sprintf("%d:%02d", m, sec)
}
