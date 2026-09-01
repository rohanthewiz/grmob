package tutorial

import (
	"fmt"

	"github.com/rohanthewiz/grmob/components"
	"github.com/rohanthewiz/grmob/core"
	"github.com/rohanthewiz/grmob/forms"
)

// chapter6 — Navigation & Overlays: the route stack and the two kinds of
// chrome that draw over it. The through-line is ownership of state: a
// Navigator frame owns its hooks and takes them to the grave (Pop, Replace,
// Reset), a Modal owns nothing — its content renders where it is declared and
// merely hides — and a Toast is not in the tree at all. Every navigation demo
// here drives the tutorial's own Navigator: the pushed screens are real
// frames on the real stack, which is why the depth captions count the
// contents screen and this lesson underneath them.
func chapter6() Chapter {
	return Chapter{
		Title:   "Navigation & Overlays",
		Icon:    "🧭",
		Summary: "Push, Pop, Replace and the two unwinds — plus Modal and Toast, the overlays that never touch the stack.",
		Lessons: []Lesson{
			lessonStack(),
			lessonReplace(),
			lessonUnwind(),
			lessonModal(),
			lessonToast(),
		},
	}
}

// navDemoScreen scaffolds a pushed demo screen: badge, live stack telemetry,
// title, then the lesson's own controls. Pushed routes render outside the
// lesson scaffold — they are full frames, exactly like the lesson itself — so
// they carry their own minimal chrome, and every one of them must offer a way
// back (a Pop of some kind), because the lesson's ‹ Contents button is on the
// frame underneath, not this one.
//
// body is a function rather than a slice so it runs inside the route, per
// pass, with the frame's own context: its views may read live stack state and
// its closures may claim hooks in the frame — which is half of what this
// chapter teaches.
func navDemoScreen(title string, body func(ctx *core.Context) []core.View) func(*core.Context) core.View {
	return func(ctx *core.Context) core.View {
		children := []core.View{
			core.Row(
				core.AlignItemsProp(core.AlignItemsCenter),
				components.Badge{Text: "NAV DEMO"},
				core.Box(core.FlexGrow(1)), // slack, so the telemetry pins right
				stackCaption(ctx),
			),
			titleText(title),
		}
		children = append(children, body(ctx)...)
		return components.Screen{Scroll: true, Gap: 16, Children: children}
	}
}

// stackCaption is the live depth/CanPop readout every demo screen (and demo
// panel) carries. It re-reads the stack per pass, so a Push or Pop is visible
// in the numbers the moment it lands.
func stackCaption(ctx *core.Context) core.View {
	return caption(fmt.Sprintf("StackDepth %d · CanPop %v",
		core.StackDepth(ctx), core.CanPop(ctx)))
}

// --- 6.1 -----------------------------------------------------------------

// detailScreen is 6.1's pushed route. Package-level rather than a closure in
// the lesson because it needs nothing from the lesson — which is itself the
// point: a route is just a view function, and this one's counter lives in the
// frame the Push mints, not in the screen that pushed it.
func detailScreen() func(*core.Context) core.View {
	return navDemoScreen("The detail screen", func(ctx *core.Context) []core.View {
		// Slot 0 of THIS frame. The lesson underneath also has a counter in
		// its slot 0, and the two can never alias: each frame renders into
		// its own scope of the Navigator's host context.
		taps := core.NewState(ctx, 0)

		return []core.View{
			prose("This screen is a new frame on the same stack — the contents screen and " +
				"the lesson are still under it, state and all. The counter below lives in " +
				"this frame: pop back and it is discarded with the frame; push again and a " +
				"fresh frame starts it at zero."),
			demoPanel("This frame's own state — take it to a value you'll recognize, then pop and re-push.",
				caption(fmt.Sprintf("This frame's taps: %d", taps.Get())),
				core.Row(
					core.Gap(8),
					components.Button{Label: "+1", OnTap: func() { taps.Set(taps.Get() + 1) }},
					components.Button{
						Label:    "‹ Pop back to the lesson",
						Emphasis: components.EmphasisOutlined,
						OnTap:    func() { core.Pop(ctx) },
					},
				),
			),
			caption("The lesson's counter is untouched by any of this: its frame never left " +
				"the stack, so its hooks were never disturbed."),
		}
	})
}

func lessonStack() Lesson {
	return Lesson{
		Title:   "Screens are a stack",
		Summary: "Navigator renders the top frame; Push adds one, Pop discards one — and each frame owns its hooks.",
		Body: func(ctx *core.Context) core.View {
			// The proof that a covered frame keeps its state: bump it, push,
			// pop, and find it unchanged. Slot 0 of the lesson's frame.
			taps := core.NewState(ctx, 0)

			return core.Column(
				core.Gap(14),
				prose("core.Navigator renders a stack of routes and shows the top one. A route "+
					"is just a view function — func(*core.Context) core.View — so there is no "+
					"route table, no registration step, and no string identifiers to keep in "+
					"sync. You are inside a worked example right now: the contents screen is the "+
					"tutorial's root frame, opening this lesson was a core.Push, and the "+
					"‹ Contents button up top is a core.Pop."),
				codeBlock(`func App(ctx *core.Context) core.View {
    return core.Navigator(HomeScreen)   // the initial route seeds the stack
}

func HomeScreen(ctx *core.Context) core.View {
    return core.Button("Details ›", func() {
        core.Push(ctx, DetailsScreen)   // adds a frame; Home keeps its state
    })
}

func DetailsScreen(ctx *core.Context) core.View {
    return core.Button("‹ Back", func() {
        core.Pop(ctx)                   // discards this frame — state and all
    })
}`),
				prose("Every frame renders into its own hook namespace, and two consequences "+
					"follow. A route may use hooks freely: core.NewState in a pushed screen "+
					"claims slot 0 of that frame, not slot 0 of whatever is underneath it. And "+
					"state lives exactly as long as its frame: a screen still on the stack keeps "+
					"everything when it is covered and uncovered, while a popped frame takes its "+
					"state — and any background resource its hooks started, like a "+
					"UseInterval ticker — with it. Pop at the root is a no-op: the stack is "+
					"never left empty."),
				demoPanel("Bump the counter, push the detail screen, poke around, pop back — the count is waiting for you.",
					stackCaption(ctx),
					caption(fmt.Sprintf("This lesson frame's taps: %d", taps.Get())),
					core.Row(
						core.Gap(8),
						components.Button{Label: "+1", OnTap: func() { taps.Set(taps.Get() + 1) }},
						components.Button{
							Label: "Push the detail screen ›",
							OnTap: func() { core.Push(ctx, detailScreen()) },
						},
					),
				),
				keyPoints(
					"A route is a plain view function — no route table, no registration, no string IDs.",
					"The stack lives on the context tree: one per app, and Push/Pop work from any handler, effect, or goroutine.",
					"Each frame owns a hook namespace: routes cannot alias each other's slots, with no effort on your part.",
					"A frame still on the stack keeps its state while covered; a popped frame's state and background resources are discarded.",
					"Pop at the root is a no-op — a Navigator with an empty stack would have nothing to render.",
				),
			)
		},
	}
}

// --- 6.2 -----------------------------------------------------------------

// checkoutStep1 is the screen Replace removes with no way back. Its one field
// uses UseForm rather than a bare string state to make the discard vivid: a
// whole form record — value, touched, the works — dies with the frame.
func checkoutStep1() func(*core.Context) core.View {
	return navDemoScreen("Checkout — step 1", func(ctx *core.Context) []core.View {
		form := forms.UseForm(ctx, forms.Spec{
			Fields: []forms.Field{{Name: "note"}},
		})

		return []core.View{
			prose("Type something below, then place the order. The confirmation screen " +
				"arrives by core.Replace: same stack depth, but this frame — note and all — " +
				"is discarded. Pop from the confirmation and you land on the lesson, not " +
				"here; abandon checkout and re-enter, and the note is gone, because a new " +
				"Push is a new frame."),
			components.FormField{
				Label: "Gift note",
				Input: form.Input("note", "A line for the card"),
			},
			core.Row(
				core.Gap(8),
				components.Button{
					Label: "Place the order",
					OnTap: func() { core.Replace(ctx, orderConfirmed()) },
				},
				components.Button{
					Label:    "‹ Abandon checkout",
					Emphasis: components.EmphasisOutlined,
					OnTap:    func() { core.Pop(ctx) },
				},
			),
		}
	})
}

func orderConfirmed() func(*core.Context) core.View {
	return navDemoScreen("Order confirmed", func(ctx *core.Context) []core.View {
		return []core.View{
			prose("Look at the depth up top: still what it was on step 1, because Replace " +
				"swapped the top frame instead of stacking a new one. The step-1 frame is " +
				"gone — Pop from here goes straight to the lesson, and the half-filled " +
				"checkout can never be revisited by walking back."),
			components.Button{
				Label:    "‹ Back to the lesson",
				Emphasis: components.EmphasisOutlined,
				OnTap:    func() { core.Pop(ctx) },
			},
		}
	})
}

func lessonReplace() Lesson {
	return Lesson{
		Title:   "Replace: steps with no way back",
		Summary: "Swap the top frame instead of stacking one — for the screens Back should never return to.",
		Body: func(ctx *core.Context) core.View {
			return core.Column(
				core.Gap(14),
				prose("Some screens should not be in the user's history: the login form after "+
					"a successful login, the payment sheet after the charge, step 1 of a wizard "+
					"once step 2 is committed. core.Replace swaps the top frame for a new one — "+
					"depth unchanged, outgoing state discarded — so the Back gesture skips the "+
					"stale step instead of resurrecting it."),
				codeBlock(`// After a successful login, the form must not be one Back away:
core.Replace(ctx, LoggedInScreen)

// This tutorial's Prev/Next footer (lesson_screen.go) is a Replace —
// the stack stays [contents, lesson] however far you read, so the
// system back gesture always means "back to contents":
OnTap: func() {
    t.markVisited(target.ID)
    core.Replace(ctx, t.lessonRoute(target.Index))
}`),
				prose("The tutorial itself leans on both effects. Prev/Next replace the lesson "+
					"frame rather than pushing, which keeps the back stack two deep instead of "+
					"thirty — and because Replace discards the outgoing frame, every lesson's "+
					"demo state starts fresh with zero cleanup code. The counter you may have "+
					"bumped in 6.1 is already gone, and this lesson never had to know it "+
					"existed."),
				demoPanel("Run the checkout: type a note, place the order, note the depth, then pop from the confirmation.",
					stackCaption(ctx),
					components.Button{
						Label: "Start checkout ›",
						OnTap: func() { core.Push(ctx, checkoutStep1()) },
					},
					caption("Then abandon and re-enter: the note is gone. A new Push is a new frame."),
				),
				keyPoints(
					"Replace = Pop + Push in one motion: depth unchanged, outgoing frame discarded, Back skips the replaced step.",
					"Reach for it whenever returning to a screen would be wrong — login forms, committed wizard steps, payment sheets.",
					"The tutorial's Prev/Next uses Replace: a shallow, predictable back stack, and per-lesson demo state reset for free.",
					"The discard is real: a form's whole record dies with its frame, which is the cleanup-free reset every wizard wants.",
				),
			)
		},
	}
}

// --- 6.3 -----------------------------------------------------------------

// drillRoute builds the level-n screen of 6.3's drill-down. The level rides
// in on the closure — with no route table there is no parameter syntax to
// learn, an argument is just a captured value — and each "Deeper" mints the
// next level's route on the spot.
func drillRoute(level int) func(*core.Context) core.View {
	return navDemoScreen(fmt.Sprintf("Drill-down — level %d", level), func(ctx *core.Context) []core.View {
		return []core.View{
			prose(fmt.Sprintf("This screen came from drillRoute(%d): routes are closures, so "+
				"\"parameters\" are captured values, and a screen can push a deeper copy of "+
				"itself forever. Settings trees, folder browsers and thread-within-thread "+
				"chats are all this shape.", level)),
			core.Row(
				core.Gap(8),
				components.Button{
					Label: "Deeper ›",
					OnTap: func() { core.Push(ctx, drillRoute(level+1)) },
				},
				components.Button{
					Label:    "‹ Pop one level",
					Emphasis: components.EmphasisOutlined,
					OnTap:    func() { core.Pop(ctx) },
				},
			),
			components.Button{
				Label:   "Done — back to contents",
				Variant: components.VariantSuccess,
				// One call unwinds everything: this level, every level under
				// it, and the lesson frame too — the root is the contents
				// screen. Progress survives because it lives above the
				// Navigator; the contents screen itself survives because
				// PopToRoot keeps the root frame rather than replacing it.
				OnTap: func() { core.PopToRoot(ctx) },
			},
			caption("Done unwinds past the lesson as well — the tutorial's root frame is the " +
				"contents screen. Your progress is intact when you land."),
		}
	})
}

func lessonUnwind() Lesson {
	return Lesson{
		Title:   "Unwinding: PopToRoot vs Reset",
		Summary: "Identical on screen, different in what survives: escape a drill-down, or end a session.",
		Body: func(ctx *core.Context) core.View {
			return core.Column(
				core.Gap(14),
				prose("Two calls throw away everything above the bottom of the stack, and they "+
					"look identical on screen — which is exactly why they are worth a lesson. "+
					"core.PopToRoot unwinds to the EXISTING root frame: scroll position, "+
					"selected tab, form contents all intact. core.Reset replaces the whole "+
					"stack with a NEW root frame, even when the route is the same function the "+
					"old root ran. Use PopToRoot for \"Done\" out of a deep drill-down; use "+
					"Reset to end a session — logout, onboarding complete, account switch — "+
					"where showing the previous user's half-filled form would be a bug."),
				codeBlock(`core.PopToRoot(ctx)          // the EXISTING root frame — state intact
core.Reset(ctx, HomeScreen)  // a NEW root frame — nothing survives

// State that must outlive every frame belongs ABOVE the Navigator.
// This tutorial keeps your progress exactly that way (app.go):
sctx := ctx.Scope("tutorial-session")
t := &tutorial{visited: core.NewState(sctx, map[string]bool{})}
return core.Navigator(t.Home)`),
				prose("The second half of the snippet is the design that makes Reset safe to "+
					"reach for: anything stored in a frame dies with the frame, so session "+
					"state — who is logged in, what has been read — is hoisted to a scope of "+
					"the context the Navigator renders into. Routes are closures and capture "+
					"it. The tutorial's progress lives there, which is why the drill-down demo "+
					"below can unwind straight past this lesson and land on a contents screen "+
					"that still knows what you've opened."),
				demoPanel("Go three levels deep, pop one to feel the difference, then hit Done and watch the whole stack unwind.",
					stackCaption(ctx),
					components.Button{
						Label: "Enter the drill-down ›",
						OnTap: func() { core.Push(ctx, drillRoute(1)) },
					},
					caption("Done uses PopToRoot, so it lands on the contents screen — re-open "+
						"this lesson from there. A live Reset demo would land in the same "+
						"place; only the root frame's own state could tell them apart."),
				),
				keyPoints(
					"PopToRoot keeps the existing root frame — reach for it to escape a drill-down without losing the root's state.",
					"Reset mints a fresh root even for the same route function — reach for it at logout, where surviving state would be a leak.",
					"Route parameters are closure captures: drillRoute(level+1) is the whole deep-linking mechanism.",
					"Session state belongs above the Navigator, in a scope of its host context — frames die, that scope doesn't.",
					"CanPop answers the hardware back button too: pop if true, otherwise let the platform close the app.",
				),
			)
		},
	}
}

// --- 6.4 -----------------------------------------------------------------

func lessonModal() Lesson {
	return Lesson{
		Title:   "Modal: the overlay that hides",
		Summary: "No frame, no stack — a controlled overlay whose content stays mounted while it's closed.",
		Body: func(ctx *core.Context) core.View {
			// All three live in the LESSON's frame — a modal has no frame of
			// its own, which is the lesson. open drives Visible; the other two
			// belong to the dialog's content and survive a close because
			// nothing here ever unmounts.
			open := core.NewState(ctx, false)
			giftReceipt := core.NewState(ctx, false)
			decision := core.NewState(ctx, "")

			confirm := func(d string) func() {
				return func() {
					decision.Set(d)
					open.Set(false)
				}
			}

			return core.Column(
				core.Gap(14),
				prose("A modal interrupts; navigation relocates. core.Modal draws its content "+
					"in a centered overlay above the current screen without touching the "+
					"stack: StackDepth is the same with the dialog up, and the screen "+
					"underneath keeps rendering. It is a controlled component in exactly the "+
					"sense an Input is — Visible renders your state, OnDismiss reports the "+
					"user's backdrop tap as intent — so the modal never decides to close "+
					"itself, and a dialog that must not be dismissed mid-operation simply "+
					"ignores the intent by leaving its state true."),
				codeBlock(`open := core.NewState(ctx, false)

core.Modal(
    core.Visible(open.Get()),                   // render state…
    core.OnDismiss(func() { open.Set(false) }), // …report intent
    core.Backdrop("#00000088"),                 // the scrim (this is the default)
    core.ModalContent(
        core.Card(
            core.Text("Discard this draft?"),
            components.Button{Label: "Discard", OnTap: discard},
        ),
    ),
)`),
				prose("The content inside ModalContent renders every pass, visible or not — a "+
					"modal hides, it does not unmount. Toggling Visible is a cheap prop patch "+
					"rather than a subtree add/remove, and any state the dialog's controls use "+
					"survives a close: unlike 6.1's popped detail screen, closing this dialog "+
					"and reopening it finds the checkbox exactly as you left it. When a dialog "+
					"should forget on dismissal, reset its state in OnDismiss — the one place "+
					"the intent is recorded."),
				demoPanel("Open the dialog, tick the box, then dismiss by tapping the dark backdrop — and reopen.",
					stackCaption(ctx),
					components.Button{
						Label: "Open the modal",
						OnTap: func() { open.Set(true) },
					},
					core.IfElse(decision.Get() == "",
						caption("No decision yet — Cancel and a backdrop tap both leave it that way."),
						caption("✓ "+decision.Get()),
					),
					core.Modal(
						core.Visible(open.Get()),
						core.OnDismiss(func() { open.Set(false) }),
						core.ModalContent(
							core.Card(
								core.Gap(10),
								core.Text("Confirm your order", core.FontWeight(core.Bold)),
								checkRow("Add a gift receipt", giftReceipt),
								core.Row(
									core.Gap(8),
									components.Button{
										Label:    "Cancel",
										Emphasis: components.EmphasisGhost,
										OnTap:    func() { open.Set(false) },
									},
									components.Button{
										Label: "Confirm",
										OnTap: confirm("Order confirmed" + map[bool]string{
											true: ", gift receipt included", false: "",
										}[giftReceipt.Get()]),
									},
								),
							),
						),
					),
				),
				keyPoints(
					"A modal is an overlay, not a frame: the stack is untouched, and the screen underneath keeps its state and keeps rendering.",
					"Controlled like an Input: Visible renders state, OnDismiss reports the backdrop tap — closing is always the app's decision.",
					"Content stays mounted while hidden — state survives a close, the exact opposite of a popped frame's fate.",
					"To forget on dismissal, reset state in OnDismiss; to refuse dismissal mid-operation, ignore the intent.",
					"Interruptions belong in modals; journeys belong in frames — if it has a Back, it wanted to be a Push.",
				),
			)
		},
	}
}

// --- 6.5 -----------------------------------------------------------------

func lessonToast() Lesson {
	return Lesson{
		Title:   "Toast: fire and forget",
		Summary: "Not a node, not a prop — a system event the host draws, gone by itself moments later.",
		Body: func(ctx *core.Context) core.View {
			// The tree-side evidence: the toasts themselves never appear in
			// the view tree, so the demo counts them here to have something
			// the reconciler (and the tests) can see.
			sent := core.NewState(ctx, 0)

			toast := func(fire func()) func() {
				return func() {
					fire()
					sent.Set(sent.Get() + 1)
				}
			}

			return core.Column(
				core.Gap(14),
				prose("core.ShowToast is the one piece of UI in this tutorial that is not a "+
					"view. It takes no context and returns nothing to render: the call emits a "+
					"system event that leaves the tree entirely, and the HOST draws the notice "+
					"— the wasm runtime keeps a toast layer above the app (and above any "+
					"modal), a native shell maps it to the platform's own toast, and a "+
					"headless test registers a recorder. Nothing about it is reconciled, "+
					"which is why it must be called from an event handler or an effect, never "+
					"during render: a render pass can run any number of times, and each call "+
					"would fire another toast."),
				codeBlock(`core.ShowToast("Saved")                           // 2000 ms default
core.ShowToast("Uploading…", core.Duration(5000)) // linger longer
core.ShowToast("Deployed", core.UseToastStyle(core.Style{
    Background: "#1F8A70", TextColor: "#FFFFFF",
}))

// The host owns the pixels. It registers the sink once at startup —
// the wasm host forwards to the page's toast layer; with no handler
// registered (a headless run), the event is dropped silently.
core.SetSystemEventHandler(func(name string, data map[string]any) { ... })`),
				prose("Use a toast for exactly one thing: confirming that a fire-and-forget "+
					"action landed, when nothing needs to be read carefully and nothing asks "+
					"for a decision. The moment a notice needs a button, a choice, or more "+
					"than a glance, it has outgrown the toast — that is 6.4's modal. And "+
					"because a toast vanishes on its own schedule, never gate anything on the "+
					"user having seen it."),
				demoPanel("Fire a few — watch the bottom of the screen, and note the tree only ever sees the counter.",
					core.Row(
						core.Gap(8),
						components.Button{
							Label: "Show a toast",
							OnTap: toast(func() { core.ShowToast("Nicely done — that landed") }),
						},
						components.Button{
							Label:    "Linger for five seconds",
							Emphasis: components.EmphasisOutlined,
							OnTap: toast(func() {
								core.ShowToast("This one waits around", core.Duration(5000))
							}),
						},
					),
					components.Button{
						Label:    "Show it styled",
						Emphasis: components.EmphasisOutlined,
						OnTap: toast(func() {
							core.ShowToast("Success, in green", core.UseToastStyle(core.Style{
								Background: "#1F8A70",
								TextColor:  "#FFFFFF",
							}))
						}),
					},
					caption(fmt.Sprintf("Toasts sent this visit: %d — and none of them are in the view tree.", sent.Get())),
				),
				keyPoints(
					"ShowToast emits a system event; the host draws it — above the app, above any modal — and removes it after Duration (default 2000 ms).",
					"Call it from handlers and effects only: render runs any number of times, and each call is another toast.",
					"It is not in the tree: nothing to reconcile, nothing to dismiss, nothing to assert on but the event itself.",
					"Confirmation only — the moment a notice needs a button or a decision, it has outgrown the toast.",
					"Hosts opt in with core.SetSystemEventHandler; with none registered, toasts drop silently — correct for headless runs.",
				),
			)
		},
	}
}
