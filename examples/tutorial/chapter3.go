package tutorial

import (
	"fmt"
	"strings"
	"time"

	"github.com/rohanthewiz/grmob/components"
	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/hooks"
)

// chapter3 — Hooks & Effects: where impurity goes. Chapter 2 ended on the rule
// that render functions read state and return a tree, nothing else; this
// chapter is the other half of that bargain. Time, fetches, derived values,
// and multi-step updates all live in the hooks package, and every hook is
// slot-backed — the same positional identity as NewState, so the rules of
// hooks (unconditional, same order, every pass) apply to all of them.
//
// The demos lean on a Navigator fact worth knowing while reading them: each
// lesson renders in its own disposable stack frame, and leaving the lesson
// closes that frame's scope. Tickers stop, pending timeouts cancel, and
// reopening a lesson mounts everything fresh.
func chapter3() Chapter {
	return Chapter{
		Title:   "Hooks & Effects",
		Icon:    "⏱",
		Summary: "Intervals, timeouts, effects, memoized derivations, and reducers — impurity with a place to live.",
		Lessons: []Lesson{
			lessonClock(),
			lessonTimeout(),
			lessonEffects(),
			lessonMemo(),
			lessonReducer(),
		},
	}
}

// --- 3.1 -----------------------------------------------------------------

func lessonClock() Lesson {
	return Lesson{
		Title:   "The clock: UseInterval",
		Summary: "A ticker owned by the context: every tick runs the latest closure and requests its own render.",
		Body: func(ctx *core.Context) core.View {
			seconds := core.NewState(ctx, 0)
			paused := core.NewState(ctx, false)

			// The ticker starts on this lesson's first render and its duration
			// is fixed then; later renders only refresh the closure. That
			// refresh is why the pause checkbox works with no channel or flag
			// plumbing — each tick calls the newest closure, which reads the
			// current pause state like any render would.
			hooks.UseInterval(ctx, func() {
				if paused.Get() {
					return
				}
				seconds.Set(seconds.Get() + 1)
			}, time.Second)

			return core.Column(
				core.Gap(14),
				prose("A render function must not sleep, tick, or spawn anything — it is re-run "+
					"at arbitrary times and must only read state and return a tree. When a screen "+
					"needs time, time gets a hook:"),
				codeBlock(`seconds := core.NewState(ctx, 0)

hooks.UseInterval(ctx, func() {
    seconds.Set(seconds.Get() + 1) // Set → render, once a second
}, time.Second)`),
				prose("The ticker starts on the hook's first render. Each tick runs on a "+
					"background goroutine and its Set requests a render through the push channel — "+
					"no native event needs to be in flight for a timer to reach the screen. "+
					"Re-renders don't restart anything; they only hand the ticker the latest "+
					"closure, so ticks always see current state captures instead of the mount "+
					"render's stale ones."),
				demoPanel("Mounted with this lesson; each tick is a state write like any other.",
					core.ComponentFunc(func(ctx *core.Context) *core.Node {
						return core.Text(fmt.Sprintf("%d s", seconds.Get()),
							core.FontSize(40),
							core.FontWeight(core.Bold),
							core.TextColor(ctx.Theme().Colors.Primary),
						).Render(ctx)
					}),
					checkRow("Pause the count", paused),
					core.If(paused.Get(),
						caption("Paused — the ticker still ticks; the latest closure just declines to count."),
					),
				),
				keyPoints(
					"UseInterval is slot-backed like NewState: call it unconditionally, in the same order, every pass.",
					"The interval duration is fixed by the first render; re-renders only refresh the callback closure.",
					"Ticks request renders via the push channel, so timer-driven UI updates need no native event to ride on.",
					"The ticker dies with its context: leaving this lesson drops the frame's scope, which stops the clock.",
				),
			)
		},
	}
}

// --- 3.2 -----------------------------------------------------------------

// timeoutRevealDelay is long enough that a reader sees the before/after flip
// happen, and short enough that they haven't scrolled past the demo when it
// does. The test for this lesson polls with a deadline comfortably above it.
const timeoutRevealDelay = 2500 * time.Millisecond

func lessonTimeout() Lesson {
	return Lesson{
		Title:   "Once, later: UseTimeout",
		Summary: "One shot after a delay — re-renders refresh the closure but never re-arm the timer.",
		Body: func(ctx *core.Context) core.View {
			fired := core.NewState(ctx, false)
			pokes := core.NewState(ctx, 0)

			hooks.UseTimeout(ctx, func() { fired.Set(true) }, timeoutRevealDelay)

			return core.Column(
				core.Gap(14),
				prose("UseTimeout is UseInterval's one-shot sibling: it arms a timer on the "+
					"hook's first render and calls fn exactly once."),
				codeBlock(`fired := core.NewState(ctx, false)

hooks.UseTimeout(ctx, func() {
    fired.Set(true) // once, ~2.5 s after the first render
}, 2500*time.Millisecond)`),
				prose("The subtle part is what re-renders do: nothing. While the timer is "+
					"pending they refresh the closure (the fire runs the latest one); after it "+
					"has fired they are ignored — a fired slot stays fired, no matter how many "+
					"passes follow. The poke button below exists to prove that: every tap "+
					"re-renders this whole lesson, and the timer does not come back."),
				demoPanel("Armed on this lesson's first render — watch the space below.",
					core.IfElse(fired.Get(),
						core.ComponentFunc(func(ctx *core.Context) *core.Node {
							t := ctx.Theme()
							return core.Card(
								core.Gap(4),
								core.Text("⏰ Right on time",
									core.FontWeight(core.Bold),
									core.TextColor(t.Colors.Primary),
								),
								caption("Fired once, 2½ s after mount. It will not fire again on this visit."),
							).Render(ctx)
						}),
						caption("Nothing yet — the timer is pending…"),
					),
					components.Button{Label: "Poke a render", Emphasis: components.EmphasisOutlined,
						OnTap: func() { pokes.Set(pokes.Get() + 1) }},
					caption(fmt.Sprintf("renders poked: %d — the timeout stays fired-once regardless", pokes.Get())),
				),
				keyPoints(
					"One shot: fn runs once, delay after the hook's first render.",
					"Re-renders while pending refresh the closure; re-renders after the fire do nothing — no re-arm.",
					"A pending timer is cancelled when the context closes, so a dead screen can't receive a late fire.",
					"Reopen this lesson from Contents to watch it again — a fresh stack frame means a fresh slot, which re-arms.",
				),
			)
		},
	}
}

// --- 3.3 -----------------------------------------------------------------

var gopherNames = []string{"Ziggy", "Pip", "Nibbles"}

// gopherBio is the demo's stand-in for a network fetch's payload —
// deterministic so the test can await exact text.
func gopherBio(i int) string {
	bios := []string{
		"Ziggy digs tunnels at 60 fps and hoards spare slices.",
		"Pip is the smallest on the team and ships the largest diffs.",
		"Nibbles once ate an entire dependency tree.",
	}
	return bios[i]
}

// fetchedBio pairs the payload with the id it was fetched for, so the view can
// tell "loaded for the current selection" from "loaded for the previous one"
// without the handler having to clear anything — the loading state is derived
// (got.id != sel), not stored.
type fetchedBio struct {
	id  int
	bio string
}

// effectFetchDelay makes the async gap visible: long enough that the
// "fetching…" caption actually flashes, short enough not to feel broken.
const effectFetchDelay = 350 * time.Millisecond

func lessonEffects() Lesson {
	return Lesson{
		Title:   "Effects: UseEffect",
		Summary: "Side effects run off the render pass — on mount, and again only when their deps change.",
		Body: func(ctx *core.Context) core.View {
			sel := core.NewState(ctx, 0)
			got := core.NewState(ctx, fetchedBio{id: -1})
			runs := core.NewState(ctx, 0)
			mounted := core.NewState(ctx, false)

			// No deps: once for the lifetime of the slot, however many renders
			// follow — the "on mount" idiom.
			hooks.UseEffect(ctx, func() { mounted.Set(true) })

			// Capture the dep's value at render time rather than reading
			// sel.Get() inside the effect: the effect runs later, on its own
			// goroutine, by which point the selection may already have moved on
			// — the closure should fetch what this render asked for.
			id := sel.Get()
			hooks.UseEffect(ctx, func() {
				time.Sleep(effectFetchDelay) // a stand-in for network latency
				got.Set(fetchedBio{id: id, bio: gopherBio(id)})
				runs.Set(runs.Get() + 1)
			}, id)

			return core.Column(
				core.Gap(14),
				prose("Fetches, subscriptions, anything that talks to the world: UseEffect. It "+
					"runs the effect on mount and again whenever its deps change between renders "+
					"— compared with reflect.DeepEqual — and skips it otherwise:"),
				codeBlock(`id := sel.Get() // capture this render's dep

hooks.UseEffect(ctx, func() {
    profile := loadProfile(id) // own goroutine: renders never block
    got.Set(profile)           // reaches the screen via the push channel
}, id) // re-runs only when id changes`),
				prose("The effect runs on its own goroutine, so a slow fetch cannot stall a "+
					"render pass; whatever it Sets arrives like any other state write. Below, "+
					"picking a gopher re-renders immediately (the selection is plain state) while "+
					"the profile trails it by a simulated network delay — and re-picking the same "+
					"gopher re-renders without re-fetching, because the dep didn't change."),
				demoPanel("The selection is sync state; the profile arrives async, from the effect.",
					components.SegmentedControl{
						Style:     segWrap,
						Labels:    gopherNames,
						Selected:  sel.Get(),
						OnSelect:  func(i int) { sel.Set(i) },
						KeyPrefix: "gopher-",
					},
					core.IfElse(got.Get().id == sel.Get(),
						core.Card(
							core.Gap(4),
							core.Text(gopherNames[sel.Get()], core.FontWeight(core.Bold)),
							prose(got.Get().bio),
						),
						caption(fmt.Sprintf("fetching %s's profile…", gopherNames[sel.Get()])),
					),
					caption(fmt.Sprintf("fetch effect runs: %d", runs.Get())),
					core.If(mounted.Get(),
						caption("✓ the no-deps effect ran once, at mount"),
					),
				),
				keyPoints(
					"UseEffect(ctx, fn, deps...) runs on mount and when deps change (reflect.DeepEqual); no deps means exactly once.",
					"Effects run on their own goroutine — never block a render, never touch the tree; communicate through Set.",
					"Capture dep values at render time; by the time the effect runs, state may have moved on.",
					"Loading UI falls out of data: store what the fetch was for, and derive 'loading' by comparing it to the current selection.",
				),
			)
		},
	}
}

// --- 3.4 -----------------------------------------------------------------

// memoWords is the corpus the memo demo filters. Small enough to read, big
// enough that "recompute on every pass" is recognizably the thing UseMemo
// exists to avoid on real data.
var memoWords = []string{
	"channel", "closure", "context", "gopher", "goroutine", "interface",
	"method", "package", "pointer", "reconciler", "slice", "struct",
}

// computeMeter counts compute() calls by mutating through a stable pointer
// instead of State.Set — Set during a render pass would request a render from
// inside a render. This is the demo's one deliberate impurity: write-only
// instrumentation that cannot itself trigger passes, kept out of the compute
// path in real code (compute must stay pure).
type computeMeter struct{ n int }

func lessonMemo() Lesson {
	return Lesson{
		Title:   "Caching: UseMemo",
		Summary: "Skip recomputing an expensive derivation until the inputs it reads actually change.",
		Body: func(ctx *core.Context) core.View {
			query := core.NewState(ctx, "")
			forced := core.NewState(ctx, 0)
			meter := core.NewState(ctx, &computeMeter{})

			matches := hooks.UseMemo(ctx, func() []string {
				meter.Get().n++ // instrumentation only; see computeMeter
				q := strings.ToLower(query.Get())
				var out []string
				for _, w := range memoWords {
					if strings.Contains(w, q) {
						out = append(out, w)
					}
				}
				return out
			}, query.Get())

			return core.Column(
				core.Gap(14),
				prose("A render function re-runs in full on every pass — that is the model. "+
					"When one of its derivations is expensive relative to a pass (sorting or "+
					"filtering a big slice, parsing, building an index), UseMemo caches it "+
					"against its inputs:"),
				codeBlock(`matches := hooks.UseMemo(ctx, func() []string {
    return filter(words, query.Get()) // runs inline, on the render pass
}, query.Get()) // recomputes only when the query changes`),
				prose("Unlike UseEffect it runs inline — the result is needed to build this "+
					"pass's tree. The meter below counts compute() calls: type and it climbs, "+
					"because the query is a dep; poke re-renders and it holds, because nothing "+
					"the memo depends on moved. Every keystroke and every poke is a full render "+
					"either way — the memo saves the recompute, not the pass."),
				demoPanel("Filter the corpus; then force re-renders that change nothing the memo reads.",
					core.Input(query.Get(), "Filter the words…", func(v string) { query.Set(v) }),
					core.IfElse(len(matches) == 0,
						caption("No matches — the memo cached an empty result for this query."),
						core.Column(
							core.Gap(4),
							caption(fmt.Sprintf("%d of %d words match:", len(matches), len(memoWords))),
							core.Text(strings.Join(matches, " · "), core.FontWeight(core.Bold)),
						),
					),
					components.Button{Label: "Re-render (changes no dep)", Emphasis: components.EmphasisGhost,
						OnTap: func() { forced.Set(forced.Get() + 1) }},
					caption(fmt.Sprintf("compute() calls: %d · forced re-renders: %d", meter.Get().n, forced.Get())),
				),
				keyPoints(
					"UseMemo runs compute inline and returns the cached value until deps change (reflect.DeepEqual).",
					"compute must be pure, and the returned value is shared across cache hits — treat it as read-only.",
					"Reach for it only when the work outweighs a render pass; the pass itself still happens.",
					"There is no UseCallback: the reconciler diffs rendered trees, so a stable closure identity buys nothing.",
				),
			)
		},
	}
}

// --- 3.5 -----------------------------------------------------------------

// The reducer demo's vocabulary. A struct state with two fields that must
// move together is the smallest honest case for a reducer: score and move
// count updated in one atomic step, where two independent Sets could tear.
type scoreAction int

const (
	scorePlusOne scoreAction = iota
	scorePlusFive
	scoreReset
)

type scoreState struct {
	score int
	moves int
}

func lessonReducer() Lesson {
	return Lesson{
		Title:   "Actions: UseReducer",
		Summary: "State that evolves through named actions, each applied atomically under the hook's own lock.",
		Body: func(ctx *core.Context) core.View {
			score, dispatch := hooks.UseReducer(ctx, func(s scoreState, a scoreAction) scoreState {
				switch a {
				case scorePlusOne:
					return scoreState{s.score + 1, s.moves + 1}
				case scorePlusFive:
					return scoreState{s.score + 5, s.moves + 1}
				case scoreReset:
					return scoreState{}
				}
				return s
			}, scoreState{})

			return core.Column(
				core.Gap(14),
				prose("When updates come in named shapes — add, remove, reset — a reducer names "+
					"them. UseReducer returns the current state and a dispatch; dispatch runs the "+
					"reducer on the live state and requests a render, exactly as Set does:"),
				codeBlock(`score, dispatch := hooks.UseReducer(ctx,
    func(s scoreState, a scoreAction) scoreState {
        switch a {
        case scorePlusOne:  return scoreState{s.score + 1, s.moves + 1}
        case scorePlusFive: return scoreState{s.score + 5, s.moves + 1}
        case scoreReset:    return scoreState{}
        }
        return s
    }, scoreState{})

components.Button{Label: "+5", OnTap: func() { dispatch(scorePlusFive) }}`),
				prose("The draw over hand-rolled s.Set(reduce(s.Get(), a)) is atomicity: the "+
					"reducer runs under the hook's own lock, so two dispatches racing in from "+
					"different goroutines both land instead of one overwriting the other. Here "+
					"that shows as score and move count never drifting apart — every action "+
					"steps both in one indivisible update."),
				demoPanel("Two fields, one atomic step per action — they cannot drift apart.",
					core.ComponentFunc(func(ctx *core.Context) *core.Node {
						return core.Text(fmt.Sprintf("%d", score.score),
							core.FontSize(40),
							core.FontWeight(core.Bold),
							core.TextColor(ctx.Theme().Colors.Primary),
						).Render(ctx)
					}),
					caption(fmt.Sprintf("%d moves", score.moves)),
					core.Row(
						core.Gap(8),
						components.Button{Label: "+1",
							OnTap: func() { dispatch(scorePlusOne) }},
						components.Button{Label: "+5",
							OnTap: func() { dispatch(scorePlusFive) }},
						components.Button{Label: "Reset", Emphasis: components.EmphasisGhost,
							OnTap: func() { dispatch(scoreReset) }},
					),
				),
				keyPoints(
					"dispatch is atomic — the reducer runs under a lock — and safe from any goroutine, like Set.",
					"The reducer must be pure and return a new value; earlier renders still hold the old one.",
					"The reducer must not dispatch — it runs with the lock held, so re-entry deadlocks. Chain actions from handlers or effects.",
					"initial is evaluated every render but only the first pass's value is kept, same as NewState.",
				),
			)
		},
	}
}
