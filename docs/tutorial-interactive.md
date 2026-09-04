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

**Live: <https://rohanthewiz.github.io/grmob/>** — the tutorial is published
to GitHub Pages on every push to `master`, built by
[`.github/workflows/site.yml`](../.github/workflows/site.yml). The page
shows the app in a phone-sized frame on a wide screen and full-bleed on an
actual phone; nothing is installed and nothing leaves your browser.

Locally it is two commands from the repository root:

```bash
./build.sh          # wasm/main.wasm + a matching wasm/wasm_exec.js
go run ./serve      # http://localhost:8080, serving wasm/
```

`build.sh` is exactly what the site workflow runs, so a local build and the
deployed one are the same recipe. (The `wasm` package mounts whichever app
its dot-import names — see [WebAssembly](platforms/wasm.md) for the
host-page contract and how to switch apps.)

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

## Code blocks

Every snippet in the tutorial is syntax-highlighted in
[Darcula](https://plugins.jetbrains.com/plugin/12102-darcula-theme), so the
colours match what a reader already sees in GoLand or IntelliJ.

The lexer is `go/scanner` from the standard library
(`examples/tutorial/highlight.go`) rather than a set of regular expressions:
the snippets are real Go, so the only tokenizer guaranteed to agree with the
reader's editor is the compiler's, and string literals, rune literals and
block comments stop being special cases — a `//` inside a string is a string,
and the scanner knows that without being told. It is lexical only, so it
colours what a token *is* and never what it means; package qualifiers, struct
field names and user-defined type names stay in the default ink, because a
scanner cannot tell them from any other identifier.

A snippet that does not lex cleanly is rendered unhighlighted rather than
half-coloured — misleading colour is worse than none in a document teaching
the syntax — and a test holds every snippet the tutorial ships to the clean
path, so that fallback is proven unreached rather than merely believed to be.

The block itself is a [`core.TextGrid`](concepts/views.md), the node type
built for rows of styled runs. That is also what makes it fixed-pitch on every
target, keeps a line from wrapping (a wrapped code line restarts at column
zero, which reads as a new statement at the outermost indent), and preserves
indentation without the non-breaking-space substitution a `Column` of `Text`
needed.

## Deep links

On the web, every lesson has an address: the page's hash is the lesson ID,
so <https://rohanthewiz.github.io/grmob/#3.1> opens straight onto the
`UseInterval` lesson. A bare chapter number works too —
<https://rohanthewiz.github.io/grmob/#3> opens where chapter 3 starts. The
address bar follows you as you tap **Next**,
**Prev** and **‹ Contents** — the **Copy link** button in the header copies
the current one. The app itself knows nothing about URLs. It speaks two
generic channels (`examples/tutorial/deeplink.go`): a `"route"` host event
in, sent by the page at boot and on every `hashchange`, and a `"route"`
system event out, sent on every navigation, which the page turns into
`history.replaceState`. The natives drop the unknown system event and never
send the host event, so the same package ships there untouched.

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
