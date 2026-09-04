package tutorial

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/rohanthewiz/grmob/components"
	"github.com/rohanthewiz/grmob/core"
)

// chapter2 — State, Events & Lists: the interactive half of the framework.
// Chapter 1's demos borrowed core.NewState on credit; this chapter pays the
// debt. The through-line across all five lessons is one loop: state is the
// source of truth, the tree is a pure function of it, and events are the only
// place state changes — every demo is a different face of that loop.
func chapter2() Chapter {
	return Chapter{
		Title:   "State, Events & Lists",
		Icon:    "⚡",
		Summary: "NewState slots, event handlers, controlled inputs, and keyed lists — how screens change.",
		Lessons: []Lesson{
			lessonCounter(),
			lessonEvents(),
			lessonInputs(),
			lessonConditionals(),
			lessonLists(),
		},
	}
}

// --- 2.1 -----------------------------------------------------------------

func lessonCounter() Lesson {
	return Lesson{
		Title:   "State: the counter",
		Summary: "NewState claims a slot; Set writes it and the screen re-renders itself.",
		Body: func(ctx *core.Context) core.View {
			count := core.NewState(ctx, 0)

			return core.Column(
				core.Gap(14),
				prose("core.NewState is the primitive under every demo you have poked so far. "+
					"It hands back typed Get/Set accessors over one slot of the component's state:"),
				codeBlock(`count := core.NewState(ctx, 0) // slot 0, seeded once

core.Text(fmt.Sprintf("%d", count.Get()))
components.Button{Label: "+1", OnTap: func() {
    count.Set(count.Get() + 1) // write → dirty → re-render
}}`),
				prose("There is no update call to make afterwards: Set marks the tree dirty and "+
					"requests a render, the render function re-runs, and the reconciler ships only "+
					"the difference to the screen. Slots are claimed by call order — the same "+
					"NewState call site reads the same slot every pass — which is why hooks must be "+
					"called unconditionally, in the same order, every render."),
				demoPanel("The whole loop: read state, render, write state in a handler.",
					core.Row(
						core.Gap(12),
						core.AlignItemsProp(core.AlignItemsCenter),
						core.ComponentFunc(func(ctx *core.Context) *core.Node {
							return core.Text(fmt.Sprintf("%d", count.Get()),
								core.FontSize(40),
								core.FontWeight(core.Bold),
								core.TextColor(ctx.Theme().Colors.Primary),
							).Render(ctx)
						}),
						caption("← a pure function of the slot"),
					),
					core.Row(
						core.Gap(8),
						components.Button{Label: "−1", Emphasis: components.EmphasisOutlined,
							OnTap: func() { count.Set(count.Get() - 1) }},
						components.Button{Label: "+1",
							OnTap: func() { count.Set(count.Get() + 1) }},
						components.Button{Label: "Reset", Emphasis: components.EmphasisGhost,
							OnTap: func() { count.Set(0) }},
					),
				),
				keyPoints(
					"NewState allocates by call position: same call order every pass, or slots silently shift — debug mode catches the drift.",
					"Set writes the slot, marks the tree dirty, and requests a render; the UI is a pure function of state.",
					"Set is safe from any goroutine — timers, network callbacks — and a burst of writes coalesces into one render.",
					"The initial value is evaluated every pass but only the first render's is kept.",
				),
			)
		},
	}
}

// --- 2.2 -----------------------------------------------------------------

// eventLogCap bounds the demo's log so a reader mashing the card cannot grow
// the tree without limit — and trimming on write shows the immutable-update
// idiom on both ends of the slice.
const eventLogCap = 5

func lessonEvents() Lesson {
	return Lesson{
		Title:   "Events & handlers",
		Summary: "Behavior props attach handlers to any container; handlers are where state changes.",
		Body: func(ctx *core.Context) core.View {
			log := core.NewState(ctx, []string{})
			total := core.NewState(ctx, 0)

			// One append helper for both gestures. The slice is replaced, never
			// mutated: earlier renders still hold the old backing array, and the
			// reconciler diffs old tree against new — shared mutable state would
			// let a handler rewrite what the previous pass already captured.
			record := func(kind string) {
				n := total.Get() + 1
				total.Set(n)
				next := append([]string{fmt.Sprintf("%d · %s", n, kind)}, log.Get()...)
				if len(next) > eventLogCap {
					next = next[:eventLogCap]
				}
				log.Set(next)
			}

			return core.Column(
				core.Gap(14),
				prose("Widgets expose typed events — Button.OnTap, Checkbox's onToggle — but "+
					"events are not reserved for widgets. Behavior props attach a handler to any "+
					"container, turning a plain Card or Row into a touch target:"),
				codeBlock(`core.Card(
    core.OnClick(func() { record("tap") }),
    core.OnLongPress(func() { record("long press") }),
    core.Text("Tap or long-press me"),
)`),
				prose("A handler is ordinary Go running outside the render pass — that is where "+
					"state writes belong. The render function itself must stay pure: it reads state "+
					"and returns a tree, and calling Set during it would request renders from inside "+
					"a render."),
				demoPanel("Both gestures land on one card; each handler prepends to the log.",
					core.ComponentFunc(func(ctx *core.Context) *core.Node {
						t := ctx.Theme()
						return core.Card(
							core.OnClick(func() { record("tap") }),
							core.OnLongPress(func() { record("long press") }),
							core.Text("Tap or long-press me",
								core.FontWeight(core.Bold),
								core.TextColor(t.Colors.Primary),
								core.Align(core.AlignCenter),
							),
						).Render(ctx)
					}),
					core.IfElse(len(log.Get()) == 0,
						caption("No events yet — try the card above."),
						core.Column(
							core.Gap(4),
							caption(fmt.Sprintf("Last %d of %d events, newest first:", len(log.Get()), total.Get())),
							core.For(log.Get(), func(entry string, i int) core.View {
								return core.Text(entry, core.FontSize(13))
							}),
						),
					),
					components.Button{Label: "Clear log", Emphasis: components.EmphasisGhost,
						OnTap: func() { log.Set([]string{}); total.Set(0) }},
				),
				keyPoints(
					"Behavior props (OnClick, OnLongPress, OnTouch…) make any container a touch target; widgets wrap them as typed fields.",
					"Handlers run off the render pass — write state there, never while rendering.",
					"Update slices and maps immutably: build a fresh value and Set it; previous renders still hold the old one.",
					"One node may carry OnClick and OnLongPress — the renderers wire one gesture recognizer, so a long press never also fires the click.",
				),
			)
		},
	}
}

// --- 2.3 -----------------------------------------------------------------

func lessonInputs() Lesson {
	return Lesson{
		Title:   "Controlled inputs",
		Summary: "Inputs render your value and report intent; state stays the single source of truth.",
		Body: func(ctx *core.Context) core.View {
			name := core.NewState(ctx, "")
			upper := core.NewState(ctx, false)

			return core.Column(
				core.Gap(14),
				prose("Every GrMob input is controlled: it displays exactly the value you pass "+
					"and reports keystrokes through a callback. The field never keeps text of its "+
					"own — if the callback doesn't Set, the character doesn't appear."),
				codeBlock(`name := core.NewState(ctx, "")

core.Input(name.Get(), "Your name", func(v string) {
    name.Set(v) // the value round-trips through state
})`),
				prose("That round-trip is the point, not a tax. Because the app owns the value, "+
					"transforming it is one line in the callback, and clearing it is a Set from "+
					"anywhere — no reaching into a widget to ask what it holds."),
				demoPanel("The field, a transform on the way in, and a write from outside it.",
					core.Input(name.Get(), "Type your name…", func(v string) {
						if upper.Get() {
							v = strings.ToUpper(v)
						}
						name.Set(v)
					}),
					checkRow("UPPERCASE on the way in", upper),
					core.IfElse(name.Get() == "",
						caption("Nothing typed yet — the state slot is empty, so the field is too."),
						core.Column(
							core.Gap(4),
							core.Text(fmt.Sprintf("Hello, %s!", name.Get()), core.FontWeight(core.Bold)),
							caption(fmt.Sprintf("%d characters, straight from state", utf8.RuneCountInString(name.Get()))),
						),
					),
					components.Button{Label: "Clear", Emphasis: components.EmphasisOutlined,
						OnTap: func() { name.Set("") }},
				),
				keyPoints(
					"Inputs are controlled: value in, intent out — state is the single source of truth.",
					"Transform or validate in the onChange callback; what you Set is what the field shows.",
					"Writes from elsewhere (the Clear button) reach the field like any other render — it has no private copy to get stale.",
					"Checkbox, TextArea, NumericInput, and InputWithSubmit (keyboard submit) all follow the same contract.",
				),
			)
		},
	}
}

// --- 2.4 -----------------------------------------------------------------

// The status demo's segments map index → phase, the same index-is-the-value
// contract the chapter-1 alignment demos used.
var statusLabels = []string{"Loading", "Ready", "Error"}

func lessonConditionals() Lesson {
	return Lesson{
		Title:   "Conditional rendering",
		Summary: "If, IfElse, and Match choose views with plain Go — no template syntax to learn.",
		Body: func(ctx *core.Context) core.View {
			status := core.NewState(ctx, 0)
			showRaw := core.NewState(ctx, false)

			return core.Column(
				core.Gap(14),
				prose("Because views are values, showing and hiding is ordinary control flow. "+
					"Three helpers cover the shapes an if statement can't express inline:"),
				codeBlock(`core.If(loggedIn, adminPanel)      // view, or an empty Fragment

core.IfElse(busy, spinner, content)

core.Match(status,                 // a switch over the tree
    core.Case(loading, spinnerView),
    core.Case(failed,  errorView),
    core.Default[phase](listView),
)`),
				prose("Match picks the first Case equal to its input, falling through to "+
					"Default — the screen below re-renders through it every time the segmented "+
					"control writes the status state. Keep hooks out of the branches, though: "+
					"condition the views, never the NewState calls, or slots shift between passes."),
				demoPanel("One Match over a status value; the checkbox drives a separate If.",
					components.SegmentedControl{
						Style:     segWrap,
						Labels:    statusLabels,
						Selected:  status.Get(),
						OnSelect:  func(i int) { status.Set(i) },
						KeyPrefix: "status-",
					},
					core.Match(status.Get(),
						core.Case(0, core.Column(
							core.Gap(8),
							caption("Fetching gophers…"),
							components.ProgressBar{Value: 0.4, AccessibilityLabel: "Loading"},
						)),
						core.Case(1, core.Card(
							core.Gap(6),
							core.Row(
								core.Gap(8),
								core.AlignItemsProp(core.AlignItemsCenter),
								components.Badge{Text: "ready", Variant: components.VariantSuccess},
								core.Text("All systems go", core.FontWeight(core.Bold)),
							),
							caption("42 gophers loaded."),
						)),
						core.Default[int](core.ComponentFunc(func(ctx *core.Context) *core.Node {
							t := ctx.Theme()
							return core.Card(
								core.Gap(4),
								core.BorderColor(t.Colors.Error),
								core.BorderWidth(1),
								core.Text("Something went wrong",
									core.FontWeight(core.Bold),
									core.TextColor(t.Colors.Error),
								),
								caption("burrow not found — try again."),
							).Render(ctx)
						})),
					),
					checkRow("Show the raw status value (a core.If branch)", showRaw),
					core.If(showRaw.Get(),
						caption(fmt.Sprintf("status = %d (%s)", status.Get(), statusLabels[status.Get()])),
					),
				),
				keyPoints(
					"Conditionals return values: If yields the view or an empty Fragment, Match is a switch over one comparable input.",
					"A false If still leaves a Fragment child; a plain nil child is skipped entirely, and MaybeProp conditionally contributes one container item.",
					"Condition views, not hooks — every NewState must run on every pass regardless of which branch shows.",
					"Both arms of IfElse are built eagerly, like any Go arguments; branches are cheap values, not deferred work.",
				),
			)
		},
	}
}

// --- 2.5 -----------------------------------------------------------------

// demoTask is the keyed-list demo's row data. The id is the row's identity —
// stable for the item's whole life — while its slice index changes every time
// something is inserted above it; the demo exists to make that distinction
// visible.
type demoTask struct {
	id    int
	title string
}

func lessonLists() Lesson {
	return Lesson{
		Title:   "Lists & keys",
		Summary: "For maps data to rows; Keyed gives each row an identity that survives change.",
		Body: func(ctx *core.Context) core.View {
			tasks := core.NewState(ctx, []demoTask{
				{1, "Feed the gopher"},
				{2, "Write some Go"},
				{3, "Ship the app"},
			})
			nextID := core.NewState(ctx, 4)

			addToTop := func() {
				id := nextID.Get()
				nextID.Set(id + 1)
				tasks.Set(append(
					[]demoTask{{id, fmt.Sprintf("Task %d", id)}},
					tasks.Get()...,
				))
			}
			// removeTask builds the fresh slice a keyed update wants — filter,
			// don't splice in place (the immutability rule from 2.2 again).
			removeTask := func(id int) func() {
				return func() {
					cur := tasks.Get()
					next := make([]demoTask, 0, len(cur))
					for _, t := range cur {
						if t.id != id {
							next = append(next, t)
						}
					}
					tasks.Set(next)
				}
			}

			return core.Column(
				core.Gap(14),
				prose("core.For maps a slice to children — ordinary iteration, not a template "+
					"directive. Wrap each row in core.Keyed with a stable ID from the data:"),
				codeBlock(`core.For(tasks, func(t Task, i int) core.View {
    return core.Keyed(fmt.Sprintf("task-%d", t.ID),
        taskRow(t, remove))
})`),
				prose("Keys are what \"same row\" means to the reconciler. Adding at the top "+
					"shifts every index; without keys each row would diff against its old "+
					"neighbor's content, and native row state would stay at the old position. "+
					"With keys, patches and row state stay attached to the item. Keep rows pure "+
					"functions of their item — per-row NewState in a list that grows or reorders "+
					"would read a neighbor's slot after any structural change."),
				demoPanel("Insert at the top and remove anywhere; the ids don't renumber.",
					components.Button{Label: "＋ Add to top", Emphasis: components.EmphasisOutlined,
						OnTap: addToTop},
					core.IfElse(len(tasks.Get()) == 0,
						caption("All done — add a task to refill the list."),
						core.Column(
							core.Gap(6),
							core.For(tasks.Get(), func(t demoTask, i int) core.View {
								return core.Keyed(fmt.Sprintf("task-%d", t.id), taskRow(t, removeTask(t.id)))
							}),
						),
					),
					caption("The left number is the item's id — its key — not its position."),
				),
				keyPoints(
					"For is plain iteration producing children; order in the slice is order on screen.",
					"Key every row with a stable ID from the data — never the slice index, which is what keys exist to outlive.",
					"Handlers update the slice immutably: build a fresh slice and Set it.",
					"Column + For suits short lists; core.List virtualizes long ones (same keyed rows, lazy layout).",
				),
			)
		},
	}
}

// taskRow is one line of the keyed-list demo: id, title, and a remove button.
// A pure function of its arguments — the shape every row of a changing list
// must keep (state stays with the list owner above).
func taskRow(t demoTask, remove func()) core.View {
	return core.ComponentFunc(func(ctx *core.Context) *core.Node {
		th := ctx.Theme()
		return core.Row(
			core.Gap(10),
			core.AlignItemsProp(core.AlignItemsCenter),
			core.Text(fmt.Sprintf("%d", t.id),
				core.TextColor(th.Colors.Primary),
				core.FontWeight(core.Bold),
			),
			core.Text(t.title, core.FlexGrow(1)),
			components.Button{Label: "✕", Emphasis: components.EmphasisGhost, OnTap: remove},
		).Render(ctx)
	})
}
