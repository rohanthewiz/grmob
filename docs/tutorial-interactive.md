# The Interactive Tutorial

GrMob ships a second tutorial that is not a document at all: an app.
[`examples/tutorial`](../examples/tutorial) is a GrMob app that teaches GrMob
— 40 lessons across 8 chapters, and every lesson is a live screen with three
parts: an explanation, the code under discussion, and a bordered **TRY IT**
panel wired to real state and real callbacks. You learn `NewState` by tapping
a counter it drives, keyed reconciliation by shuffling a list that breaks
without keys, and error boundaries by planting a panic and watching the
fallback swap in — then heal.

Where [Building a Todo App](tutorial-todo.md) applies the framework end to
end on one app, the interactive tutorial goes concept by concept, one
tappable claim at a time. They pair well: walk the interactive tutorial to
build the mental model, then the todo guide to see the model shipped to
simulators.

## Running it

The tutorial is the app the WASM host currently mounts, so the browser route
is two commands from the repository root:

```bash
GOOS=js GOARCH=wasm go build -o wasm/main.wasm ./wasm
(cd wasm && python3 -m http.server 8080)
```

Open <http://localhost:8080>. (The `wasm` package mounts whichever app its
dot-import names — see [WebAssembly](platforms/wasm.md) for the host-page
contract and how to switch apps.)

It also ships natively: the package registers itself with the mobile bridge
(`init` + `AppName`, the same contract as `examples/todoapp`), so the
[native build steps](platforms/native.md) apply unchanged.

## The curriculum

| Chapter | What you tap |
|---|---|
| 1 · Views & Layout | Composition, text and typography, rows/columns, alignment and flex, surfaces |
| 2 · State, Events & Lists | `NewState` mechanics, controlled inputs, conditionals, `For` + `Keyed` and why keys matter |
| 3 · Hooks & Effects | `UseInterval`, `UseTimeout`, `UseEffect`, `UseMemo`, `UseReducer` — and the rules of hooks |
| 4 · The Widget Library | Button variants × emphasis, badges/chips/segmented controls, `ListRow`, `Accordion`, `Tabs` |
| 5 · Forms & Validation | Rules, reveal policies, focus and blur, `FormField` |
| 6 · Navigation & Overlays | Push/Pop/Replace/Reset, modals, toasts |
| 7 · Theming & Styling | The style pipeline, `UseStyle` merging, theme anatomy, a live Default ↔ Material switcher, transitions |
| 8 · Robustness | Error boundaries, handler panic guards, [debug mode](concepts/debug-mode.md)'s live concern inspector, [`Cached`](concepts/caching.md) |

Progress is tracked in-app: the contents screen shows how many lessons you
have opened, and **Next** walks the whole curriculum in order.

## How it stays honest

Being an ordinary app in `examples/` is the design. The same package runs in
the browser, ships natively through the mobile bridge, and — the part that
matters for trust — is driven headless by its test suite through
`render.Manager` with [debug mode](concepts/debug-mode.md) on: every demo is
tapped by tests, every render pass is audited for concerns, and a lesson
whose demo breaks fails CI. The chapter on robustness even tests its own
provocations both ways: that the planted panic is caught, and that the
expected concern was filed.

That also makes the tutorial's source a worked example in itself. Nothing in
it is framework-privileged — every panel is built from exactly the API it
teaches, so once you have walked the lessons, reading
[`examples/tutorial`](../examples/tutorial) is a natural next step.
