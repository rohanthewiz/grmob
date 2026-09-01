package tutorial

import (
	"fmt"
	"time"

	"github.com/rohanthewiz/grmob/components"
	"github.com/rohanthewiz/grmob/core"
)

// chapter8 — Robustness: the production posture. The framework assumes app
// code will eventually fail and is built so one failure costs a panel or a
// frame, never the process: ErrorBoundary contains render panics (8.1), the
// driver's guards contain handler panics (8.2), debug mode turns the silent
// misuses into named, countable concerns (8.3), and Cached lets a
// proven-static subtree opt out of the per-frame budget entirely (8.4). The
// finale (8.5) closes the curriculum: the whole model in one page, and where
// to go next.
//
// A note on how these lessons coexist with the tutorial's own tests, which
// run every pass under debug mode and assert zero concerns: every demo that
// deliberately provokes a panic or a concern is OFF by default, so walking
// the curriculum records nothing; the chapter's tests flip the provokers on,
// assert the expected concern was filed (a positive assertion — the check
// checking the checker), and clear the collector before finishing.
func chapter8() Chapter {
	return Chapter{
		Title:   "Robustness",
		Icon:    "🛡️",
		Summary: "Error boundaries, the driver's panic guards, debug-mode concerns, and Cached — how an app stays alive, observable, and fast.",
		Lessons: []Lesson{
			lessonErrorBoundaries(),
			lessonHandlerGuard(),
			lessonDebugMode(),
			lessonCached(),
			lessonFinale(),
		},
	}
}

// --- 8.1 -----------------------------------------------------------------

func lessonErrorBoundaries() Lesson {
	return Lesson{
		Title:   "Error boundaries",
		Summary: "A panic escaping Render would kill a native app — ErrorBoundary trades the crash for a fallback, and heals on its own.",
		Body: func(ctx *core.Context) core.View {
			// The saboteur. It lives OUTSIDE the boundary — both so it stays
			// tappable while the panel inside is down, and because that is
			// the honest shape: the boundary protects a subtree, not the
			// screen around it.
			explode := core.NewState(ctx, false)

			// The protected panel, with the classic planted bug: it was
			// "written" when the inbox could never be empty, so it reads
			// messages[0] unconditionally. Emptying the slice makes that
			// read panic — the exact stale-index failure boundaries exist
			// for. Hook-free on purpose: the boundary renders its child into
			// a private child context (its own hook namespace), so the demo
			// keeps its one state in the lesson's frame and reaches it by
			// closure, the same ownership shape as 7.4's themed subtree.
			inbox := core.ComponentFunc(func(c *core.Context) *core.Node {
				messages := []string{
					"Standup moved to 10:30",
					"Chapter 8 has shipped 🎉",
				}
				if explode.Get() {
					messages = messages[:0]
				}
				return core.Column(
					core.Gap(6),
					core.Text("Inbox", core.FontWeight(core.Bold)),
					core.Text("Latest: "+messages[0]),
					caption(fmt.Sprintf("%d message(s) total", len(messages))),
				).Render(c)
			})

			// A custom fallback: it receives the full *RenderError — err's
			// message carries the actual panic value, and the demo shows it
			// so the healing below is visibly tied to the real failure. This
			// is also the notification hook: a production fallback would log
			// or report here (rate-limited — it runs on every failing pass).
			fallback := func(err error) core.View {
				return core.ComponentFunc(func(c *core.Context) *core.Node {
					t := c.Theme()
					return core.Column(
						core.Gap(4),
						core.Text("Inbox is unavailable",
							core.FontWeight(core.Bold),
							core.TextColor(t.Colors.Error),
						),
						caption(err.Error()),
					).Render(c)
				})
			}

			return core.Column(
				core.Gap(14),
				prose("On a phone, a render pass runs on the native UI thread with a bridge "+
					"call in flight — a panic that escapes Render unwinds straight through the "+
					"JNI or cgo boundary and kills the process. One stale index in one panel "+
					"takes down the whole app. core.ErrorBoundary is the containment wall: it "+
					"renders its child, and if the child's render panics, it renders "+
					"fallback(err) in its place. The err is a *core.RenderError carrying the "+
					"actual panic value — stack included, and errors.Is/As reach through it — "+
					"so a fallback can tell a bad index from a real error that travelled by "+
					"panic."),
				codeBlock(`core.ErrorBoundary(
    ProfilePanel(user),
    func(err error) core.View {
        log.Printf("profile panel down: %v", err) // the notification hook
        return core.Text("Profile unavailable")
    },
)

core.SafeRender(panel) // same boundary, built-in fallback card

// err is the real panic value in an envelope — this works even
// though the error travelled by panic():
//   errors.Is(err, sql.ErrNoRows)`),
				prose("Two behaviors are deliberate. The boundary does not latch: the tree is "+
					"rebuilt from scratch every pass, so a child that panicked on stale state "+
					"renders normally the moment the state settles — no reset step exists, "+
					"because none is needed. And the boundary itself logs nothing: the "+
					"fallback is the notification hook, called on every failing pass — which "+
					"also means a deterministic panic calls it at frame rate, so rate-limit "+
					"what you log. The fallback runs under the same guard (one that panics "+
					"degrades to a plain built-in message), and SafeRender's default card "+
					"shows the panic detail only in debug mode: a panic message identifies "+
					"the bug for you, and leaks internals in a user's screenshot."),
				demoPanel("Plant the classic bug: empty the data, keep the read of element [0].",
					core.ErrorBoundary(inbox, fallback),
					checkRow("Empty the inbox (the panel still reads messages[0])", explode),
					caption("The checkbox sits outside the boundary, so it keeps working "+
						"while the panel is down. Untick it and the panel is back on the "+
						"very next pass — the boundary retries every pass and heals on its "+
						"own."),
				),
				keyPoints(
					"A panic escaping Render on a native host unwinds through the bridge and kills the process — the boundary is what stands between one bad index and a dead app.",
					"ErrorBoundary(child, fallback) renders fallback(err) when the child's render panics; SafeRender is the built-in-fallback one-liner.",
					"It does not latch: the tree rebuilds every pass, so the boundary retries and heals the moment the child renders clean.",
					"The fallback is the notification hook — it gets the full *RenderError, stack included, on every failing pass; rate-limit what you log.",
					"In debug mode every catch is also filed as a render-panic concern — 8.3 shows where those go.",
				),
			)
		},
	}
}

// --- 8.2 -----------------------------------------------------------------

func lessonHandlerGuard() Lesson {
	return Lesson{
		Title:   "When a handler panics",
		Summary: "Handlers run between passes, where no boundary can see them — the driver guards every dispatch, and honesty about half-applied state is the lesson.",
		Body: func(ctx *core.Context) core.View {
			// Two counters that one handler is supposed to move in lockstep,
			// plus the tripwire between the writes. The skew after a panic is
			// the whole exhibit: work before the panic stuck, work after it
			// never ran.
			a := core.NewState(ctx, 0)
			b := core.NewState(ctx, 0)
			explode := core.NewState(ctx, false)

			advance := func() {
				a.Set(a.Get() + 1)
				if explode.Get() {
					panic("wired to fail between the two writes")
				}
				b.Set(b.Get() + 1)
			}

			status := "A and B are in step — the handler completed."
			if skew := a.Get() - b.Get(); skew > 0 {
				status = fmt.Sprintf("Skewed by %d: the panic landed between the writes, "+
					"and the next pass rendered the state that actually exists.", skew)
			}

			return core.Column(
				core.Gap(14),
				prose("Boundaries cover renders, but an event handler runs between passes — "+
					"when an OnTap panics there is no tree in flight, so no boundary in it "+
					"can help, and on a native host the same bridge unwinding applies: a nil "+
					"dereference in a tap handler is as fatal as a panicking Render. So the "+
					"render driver guards every dispatched handler itself. The panic is "+
					"caught and logged, in debug mode it is filed as a handler-panic "+
					"concern, and the app lives. There is nothing to opt into — every host "+
					"that dispatches through the manager gets it."),
				codeBlock(`core.Button("Transfer", func() {
    from.Set(from.Get() - amount) // ran
    mustNotFail()                 // panicked here
    to.Set(to.Get() + amount)     // never ran — state is now half-applied
})

// The driver's guard catches it: the app lives, and the next pass
// renders the state that actually exists. What "half-applied" means
// is the app's to repair — nothing generic can know.`),
				prose("What the guard cannot do is undo. The handler stopped wherever the "+
					"panic happened, so writes before it stuck and writes after it never "+
					"ran. Only the app can know what that half means, so the framework's "+
					"answer is honest: the next pass renders whatever state exists, which "+
					"is strictly better than the same half-written state plus a dead "+
					"process. The demo advances two counters in one handler with a planted "+
					"panic between the writes — watch them skew, then repair them, because "+
					"repair is policy and policy is yours."),
				demoPanel("Advance both counters — then arm the panic between their writes.",
					core.Row(
						core.Gap(10),
						demoBox(fmt.Sprintf("A: %d", a.Get()), boxBlue, 0),
						demoBox(fmt.Sprintf("B: %d", b.Get()), boxTeal, 0),
					),
					caption(status),
					core.Row(
						core.Gap(8),
						components.Button{Label: "Advance both counters", OnTap: advance},
						components.Button{
							Label:    "Repair (set B = A)",
							Emphasis: components.EmphasisOutlined,
							OnTap:    func() { b.Set(a.Get()) },
						},
					),
					checkRow("Panic between the two writes", explode),
				),
				keyPoints(
					"Handlers run between passes: no boundary in the tree can cover them, so the driver guards every dispatch itself.",
					"Recovery is deliberately partial — work before the panic sticks, work after it is lost.",
					"The next pass renders whatever state exists; deciding what half-applied means, and repairing it, is app policy.",
					"Debug mode files each recovered handler panic as a handler-panic concern.",
					"The guard converts a crash into a bug — it keeps users running, not code correct. Fix the panic.",
				),
			)
		},
	}
}

// --- 8.3 -----------------------------------------------------------------

func lessonDebugMode() Lesson {
	return Lesson{
		Title:   "Debug mode",
		Summary: "One switch turns the silent failure modes — conditional hooks, duplicate keys, misused caches, swallowed panics — into named, counted concerns.",
		Body: func(ctx *core.Context) core.View {
			provoke := core.NewState(ctx, false)

			// The wrinkle this demo has to handle, and worth learning from:
			// SetDebugMode and the concerns collector are process-wide
			// values, not state. Writing them marks no tree dirty, so a
			// handler that only touches them would take effect invisibly —
			// the pass that would SHOW the effect never gets scheduled.
			// Each such control therefore also bumps this throwaway state:
			// state is the only render trigger.
			poke := core.NewState(ctx, 0)
			repaint := func() { poke.Set(poke.Get() + 1) }

			// Live inspector over the process-wide collector. Reading the
			// collector during render is fine (it is a snapshot); filing or
			// clearing belongs in handlers, like any other mutation. It
			// renders BELOW the provoked list in the tree, so a concern
			// filed earlier in this same pass is already visible to it.
			inspector := core.ComponentFunc(func(c *core.Context) *core.Node {
				t := c.Theme()
				items := []core.PropsAndChildren{
					core.Gap(6),
					core.Text("Concerns", core.FontWeight(core.Bold)),
				}
				list := core.Concerns()
				if len(list) == 0 {
					items = append(items, caption("None — the collector is empty."))
				}
				for _, cn := range list {
					items = append(items,
						core.Text(fmt.Sprintf("[%s] ×%d", cn.Kind, cn.Count),
							core.FontSize(13),
							core.FontWeight(core.Bold),
							core.TextColor(t.Colors.Error),
						),
						caption(cn.Detail),
					)
				}
				return core.Column(items...).Render(c)
			})

			return core.Column(
				core.Gap(14),
				prose("Positional hooks and keyed reconciliation fail silently: a hook "+
					"called conditionally shifts every later component's state, colliding "+
					"sibling keys quietly degrade to positional matching, and a boundary "+
					"doing its job can hide a panel that has been dead for weeks. Debug "+
					"mode — core.SetDebugMode(true), one process-wide switch — makes the "+
					"framework check for all of it during every pass: cursor drift, "+
					"duplicate keys, arguments a container had to drop, hooks or callbacks "+
					"inside a Cached view, and both panic kinds from 8.1 and 8.2. Findings "+
					"land in a collector as concerns, deduplicated by kind and detail with "+
					"a count — a bug that fires every frame is one line with a rising "+
					"number, not a flood."),
				codeBlock(`core.SetDebugMode(true) // at startup, in a development build

// after driving the app:
fmt.Print(core.DumpConcerns()) // human-readable, one line per finding
core.Concerns()                // sorted snapshot — what tests assert on
core.ClearConcerns()

// The discipline this tutorial's own tests follow:
func TestMain(m *testing.M) { core.SetDebugMode(true); m.Run() }
// ...and every test ends by asserting core.Concerns() is empty.`),
				prose("This lesson practices what it preaches: the tutorial's test suite "+
					"drives every lesson you have tapped under debug mode and fails if a "+
					"single concern is recorded. When the checks are off, every detection "+
					"site guards behind one atomic load, so a release build pays nothing "+
					"— the demo below proves that too: with the switch off, the same bad "+
					"list records nothing at all."),
				demoPanel("Flip the switch, render a bad list, and read the findings live.",
					core.Row(
						core.Gap(6),
						core.AlignItemsProp(core.AlignItemsCenter),
						core.Checkbox(core.IsDebugMode(), func(v bool) {
							core.SetDebugMode(v)
							repaint()
						}),
						caption("Debug mode (core.SetDebugMode — process-wide)"),
					),
					checkRow("Render two rows that share the key \"dup\"", provoke),
					core.If(provoke.Get(), core.Column(
						core.Gap(4),
						core.Keyed("dup", core.Text("Row A — Keyed(\"dup\", …)")),
						core.Keyed("dup", core.Text("Row B — Keyed(\"dup\", …)")),
					)),
					inspector,
					components.Button{
						Label:    "Clear concerns",
						Emphasis: components.EmphasisOutlined,
						OnTap: func() {
							core.ClearConcerns()
							repaint()
						},
					},
					caption("While the bad list renders under debug mode, every pass "+
						"re-files the finding and the count climbs. The collector is "+
						"process-wide — if you tripped 8.1's or 8.2's panics, they are "+
						"still listed here."),
				),
				keyPoints(
					"SetDebugMode is one process-wide switch; off, every check is a single atomic load — a release build pays nothing.",
					"It names the silent failures: cursor drift from conditional hooks, duplicate sibling keys, dropped container arguments, Cached misuse, and recovered panics.",
					"Concerns dedupe by kind and detail with a count — a persistent bug is one line with a rising number.",
					"Concerns() is the sorted snapshot tests assert on; DumpConcerns() is for humans; ClearConcerns() resets.",
					"The switch and the collector are globals, not state — pair writes to them with a state bump, because state is the only render trigger.",
				),
			)
		},
	}
}

// --- 8.4 -----------------------------------------------------------------

// stampCard is 8.4's probe: a colored card stamping the wall-clock time of
// the render pass that built it. The stamp is exactly the kind of per-pass
// value a real cached view must NOT depend on — it is used here so that the
// staleness IS the display: a frozen stamp is the cache working.
func stampCard(label, color string) core.View {
	return core.ComponentFunc(func(ctx *core.Context) *core.Node {
		return core.Column(
			core.BackgroundColor(color),
			core.BorderRadius(8),
			core.Padding(12),
			core.Text(label+" — rendered at "+time.Now().Format("15:04:05.000"),
				core.TextColor("#FFFFFF"),
				core.FontWeight(core.Bold),
			),
		).Render(ctx)
	})
}

// cachedStamp is the demo's cached subtree, constructed ONCE at package
// level — the construct-once rule: the lesson Body runs on every pass, so a
// Cached built inside it would be a fresh wrapper each time and cache
// nothing. Beyond the deliberate clock read above, the probe honors the
// contract: no hooks, no callbacks, no theme reads (its colors are the
// tutorial's fixed demo palette), so even under debug mode's bypass it files
// no cached-hooks/cached-callbacks concern.
var cachedStamp = core.Cached(stampCard("Cached", boxPlum))

func lessonCached() Lesson {
	return Lesson{
		Title:   "Cached: freeze the static",
		Summary: "Cached renders a subtree once and replays the same *Node — pointer equality short-circuits the diff, so static chrome costs zero per frame.",
		Body: func(ctx *core.Context) core.View {
			passes := core.NewState(ctx, 0)

			// The mode note is honest about where you are running: the
			// tutorial's tests (and 8.3's switch, if you flipped it) change
			// what this demo shows, and saying so beats a demo that appears
			// broken in one of the two modes.
			modeNote := "Debug mode is OFF, so the cache is real: the cached card's " +
				"stamp froze at its first render — force passes, leave and come back, " +
				"it will not change. The cache is process-wide and outlives this " +
				"lesson's frame."
			if core.IsDebugMode() {
				modeNote = "Debug mode is ON (8.3's switch), so Cached is bypassed: " +
					"the cached card re-renders every pass here so the audits can see " +
					"inside it. Flip debug off in 8.3, come back, and watch the stamp " +
					"freeze."
			}

			return core.Column(
				core.Gap(14),
				prose("Most subtrees change; some provably never do — a logo row, a "+
					"footer, fixed chrome. core.Cached(view) renders its view once and "+
					"replays the same *Node pointer on every later pass, and the "+
					"reconciler's first move in Diff is a pointer comparison: the identical "+
					"pointer is proof of \"unchanged\", so a cached subtree costs zero "+
					"re-render AND zero diff work, every frame. It drops out of the "+
					"per-frame budget entirely."),
				codeBlock(`// Package level — constructed ONCE. Built inside a render body it
// would be a fresh wrapper every pass, and would cache nothing.
var header = core.Cached(core.Text("My App"))

// The reconciler's fast path is pointer equality:
//     if old == new { return nil }      // first line of reconcile.Diff
// Same *Node every pass ⇒ no walk, no patches, no cost.`),
				prose("The price is a contract: a cached view must be a pure function of "+
					"its construction arguments, because it renders on pass one and never "+
					"again. No hooks — its slots would vanish from pass two onward, "+
					"shifting every later component's state. No callbacks or interactive "+
					"widgets — its handlers would be purged and every later callback ID "+
					"would shift. No theme or other per-pass reads — the first render is "+
					"baked in forever. Debug mode changes the behavior on purpose: the "+
					"cache is bypassed so the audits can see the real subtree, and the "+
					"contract violations are measured directly — filed as cached-hooks and "+
					"cached-callbacks concerns instead of exhibited as corruption."),
				demoPanel("Force passes and compare the stamps.",
					stampCard("Live", boxTeal),
					cachedStamp,
					core.Row(
						core.Gap(8),
						core.AlignItemsProp(core.AlignItemsCenter),
						components.Button{
							Label: "Force another pass",
							OnTap: func() { passes.Set(passes.Get() + 1) },
						},
						caption(fmt.Sprintf("passes forced: %d", passes.Get())),
					),
					caption(modeNote),
				),
				keyPoints(
					"Cached renders once and replays the same *Node; pointer equality is the diff's \"unchanged\" evidence, so the subtree costs zero per frame.",
					"Construct the wrapper once — a package-level var. Inside a render body it is rebuilt each pass and caches nothing.",
					"The cached view must be pure: no hooks, no callbacks, no theme or per-pass values — the first render is baked in forever.",
					"Debug mode bypasses the cache and files contract violations as concerns instead of letting them corrupt silently.",
					"Reserve it for proven-static, hot chrome: the contract is a real burden, and the payoff is frame time.",
				),
			)
		},
	}
}

// --- 8.5 -----------------------------------------------------------------

func lessonFinale() Lesson {
	return Lesson{
		Title:   "The whole model",
		Summary: "Eight chapters in one sentence, the same counter you started with — and where to take it next.",
		Body: func(ctx *core.Context) core.View {
			return core.Column(
				core.Gap(14),
				prose("The whole machine now fits in one sentence: a GrMob app is a pure "+
					"Go function from state to a view tree, re-run when state changes, "+
					"diffed against the previous tree, and shipped to the platform as a "+
					"minimal set of patches. Everything since Chapter 1 has been that "+
					"sentence wearing different clothes — views and layout compose the "+
					"function, state and events drive it, hooks give it memory and timing, "+
					"the widget library is its vocabulary, forms make intent safe, "+
					"navigation stacks whole screens of it, themes restyle it as data, and "+
					"this chapter keeps it alive when the code is wrong."),
				codeBlock(`func App(ctx *core.Context) core.View {      // a view fn   (Ch.1)
    count := core.NewState(ctx, 0)           // state       (Ch.2)
    return core.Column(                      // layout      (Ch.1)
        core.Text(fmt.Sprintf("Taps: %d", count.Get())),
        core.Button("Tap me", func() {       // an event    (Ch.2)
            count.Set(count.Get() + 1)       // dirty → render → diff → patch
        }),
    )
}
// You can now read every line of this — and of the framework behind it.`),
				prose("Where next: the Building a Todo App guide in docs/ applies this "+
					"model end to end and ships it to the iOS simulator and Android "+
					"emulator; examples/todoapp and examples/social are complete apps to "+
					"read; the components package is the widget library's source, written "+
					"against public core API only. And the app you have been tapping for "+
					"eight chapters is examples/tutorial — every panel of it built from "+
					"exactly what it was teaching, tested headless under debug mode. "+
					"Nothing in it is framework-privileged, which makes reading its source "+
					"the fastest next step of all."),
				demoPanel("One last tap.",
					core.ComponentFunc(func(c *core.Context) *core.Node {
						t := c.Theme()
						return core.Column(
							core.Gap(6),
							core.Text("Curriculum complete",
								core.UseStyle(t.Typography.Subtitle)),
							prose(fmt.Sprintf("%d lessons across %d chapters — every one "+
								"a live GrMob screen, rendered, diffed and patched by the "+
								"framework it was teaching.",
								len(flatLessons), len(Chapters))),
						).Render(c)
					}),
					components.Button{
						Label:   "Take a bow 🎉",
						Variant: components.VariantSuccess,
						OnTap: func() {
							core.ShowToast("Tutorial complete — now go build something")
						},
					},
				),
				keyPoints(
					"A GrMob app is one pure function from state to view; every frame is a fresh render, diffed against the last, shipped as minimal patches.",
					"State lives in context slots addressed by call order — hooks are positional, so call them unconditionally, every render.",
					"Widgets are controlled: they render your value and report intent; you own every byte of state.",
					"Themes, navigation, overlays and transitions are all data on the tree or the context — restyling and rerouting are ordinary renders.",
					"Boundaries, guards, debug mode and Cached keep the app alive, the bugs visible, and the frame budget honest.",
				),
			)
		},
	}
}
