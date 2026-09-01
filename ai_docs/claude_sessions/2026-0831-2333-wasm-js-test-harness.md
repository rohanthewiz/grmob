# Session: The WASM JavaScript Test Harness

**Session ID:** session_01LZZTYuh1UXxGjKfkomQRM6
**Date:** 2026-08-31, ~23:33
**Branch:** master
**Follows:** `2026-0831-2041-focus-traversal.md`

Last session's backlog opened with: "There is still no JS test harness. Three
sessions have now added renderer logic that only a browser can check, and this
one added a whole event path." This session built it.

Two design questions were put to the user up front; both took the recommended
option: a zero-dependency harness on `node:test` + `node:vm` with a
hand-written DOM, and a scope covering both unit tests and a Go→JS conformance
replay.

**The harness found two bugs on its first run**, one of them severe. That is
recorded in Part 4 rather than buried at the end, because it is the whole
argument for the session.

---

# Part 1 — Why zero dependencies

The repo has no `package.json`, no lockfile and no `node_modules`. The
precedent that settled it is `ios/verify`, whose header says "Needs only Go
and the Xcode Command Line Tools" — a hand-rolled harness, no package manager,
sub-second loop. `wasm/verify` is the same trade: **needs only Go and Node**.

The measurement that made it easy: the runtime touches **22 distinct DOM
members** in total (`createElement`, `querySelector`, `activeElement`,
`dataset`, `style`, `textContent`, `value`, `placeholder`, `src`, `disabled`,
`children`, `setAttribute`, `removeAttribute`, `appendChild`, `replaceWith`,
`remove`, `addEventListener`, `focus`, `blur`, `tagName`, `innerHTML`,
`getElementById`). That is small enough to *implement* rather than
approximate, which is what makes a shim defensible instead of a liability.

The two rejected options are worth recording. **jsdom** buys real, spec-tested
Element semantics but costs the repo its first `package.json`, lockfile and
network install — and still cannot answer the questions that are actually open
(`enterkeyhint` and `focus()` are just attributes to it too). **Playwright** is
the only option that answers those, and costs a ~150MB browser download per
environment plus a loop measured in minutes rather than the milliseconds
`ios/verify` set as this repo's bar.

The honest limitation is stated in `dom.mjs`'s header and in the docs: this
tests the runtime against a *model* of the DOM. Anything needing real
rendering — layout, whether `enterkeyhint` relabels a soft keyboard, whether
`focus()` opens one — is out of reach and stays out of reach, exactly as the
iOS view layer still needs a simulator.

---

# Part 2 — Loading a browser script that was never meant to be imported

`grmob-runtime.js` is a classic script, not a module. It declares
`const GrMob = (() => {...})()` at the top level and, at the bottom, calls
`waitForWasm()` and assigns `window.GrMobRequestPermission`. So it cannot be
`import`ed — it has to be **evaluated** with the globals a page would already
have provided, which is what `node:vm` is for.

**Nothing in the runtime was changed to make it testable.** The one addition
is a single appended statement publishing `GrMob` onto the context, because a
top-level `const` in a vm script stays in that script's lexical scope. (In a
browser it *is* reachable — a classic script's top-level const is a global
lexical binding, which is how `index.html`'s inline script calls
`GrMob.mount`.)

Three details the sandbox needed:

**`setTimeout` is a stub that schedules nothing.** `waitForWasm()` polls until
`window.GrMobWASM` appears, which never happens here — the harness drives
`mount()` and `patch()` directly rather than through the WASM push loop. A
real timer would spin forever and keep the test process alive.

**`requestAnimationFrame` is a drainable queue, not a timer.** The runtime
defers `focus()`/`blur()` by one frame, so a test has to be able to say "now
the frame happened" and observe the difference. With a real timer that would
be a race, and the deferral — the thing most worth testing — would be
untestable.

**Payloads are JSON round-tripped at the bridge boundary.** This started as a
vm-realm problem: an object built inside the context carries that context's
`Object.prototype`, so `assert.deepStrictEqual` against a host-realm literal
fails on prototype identity alone. The fix turned out to be the *more*
faithful model rather than a workaround — `index.html`'s real
`GoInvokeCallback` calls `JSON.stringify` before handing anything to Go, so
serializing at the same boundary is what the browser does anyway.

## The shim's one simplification, made into a guard

`textContent` is a stored string, not the live concatenation of descendant
text a browser computes. That is safe only because the runtime sets it on leaf
types alone (Text's `content`, Button's `label`) — so `appendChild` now
**throws** if it is called on an element that already has text. The moment the
runtime puts children under a node it has given text to, the simplification
has become a lie and says so. A shim that quietly diverges is worse than no
shim at all.

---

# Part 3 — The two layers

**`gen.go`** drives real example apps through `render.Manager` the way the
browser drives them — `RenderInitial` once, then one `Dispatch` per user event
— recording the initial tree, every patch batch, and the final tree. Two
scenarios, chosen because they exercise disjoint halves:

- **demo** (`examples/mobileapp`) — tab switches, keyed list rows, all four
  callback kinds. The structural half.
- **signup** (`examples/signup`) — focus commands, keyboard traversal, and the
  prop churn a validating form produces. The focus half.

**`replay_test.mjs`** mounts that initial tree through the real runtime,
applies the batches, walks the resulting DOM back into a flat description and
compares it with Go's final render. It also asserts each batch named at least
one live element — the patch handler returns early on a missing target, so a
batch that reached nothing would otherwise pass silently.

The prop table is restated from the contract rather than copied from the
implementation: two independent statements of one rule is the point of a
conformance test.

**`runtime_test.mjs`** covers the per-element logic no transcript reaches: the
Enter filter and the void envelope it sends, `enterkeyhint` in all four
states, the focus command's frame deferral, epoch-0 sentinel, blur guard,
empty action and stale-epoch guard, listener re-wiring, and the structural
patches.

One replay test is worth naming on its own: **a focus command actually moves
the browser's focus**. `examples/signup`'s server-error path issues a
`core.Focus`, and the assertion is that `document.activeElement` is the email
input once the deferred frame has run. That is the first time the WASM focus
path written two sessions ago has been executed by anything.

---

# Part 4 — What it found on its first run

**The event payload never worked for fields present at mount.** The two
listener sites passed different vocabularies into one lookup: `createElement`
passed the Go node type (`"Input"`), the update path passed the HTML tag
(`"input"`), and `extractEventPayload` matched lowercase only. So a text field
rendered at mount sent `{}` for every keystroke; Go, seeing no value, routed
the void envelope to the void callback map where a `txt_` ID does not exist,
and the handler silently never ran. **Typing did nothing at all.**

Fixed by settling on one vocabulary — the Go node type, which the element
already records precisely because `tagForType` collapses Input,
InputPassword, NumericInput and Checkbox all onto `<input>` and the tag cannot
recover it — and matching case-insensitively so the two sites cannot drift
apart again. Two regression tests pin it, including one asserting both
listener paths build the same envelope.

**The runtime never renders checkbox state.** `checked` has no branch in
either path, and `Checkbox` maps to `<input>` with no `type` attribute, so a
checkbox draws as a text box and its state never appears. Pre-existing and
unrelated to any recent work.

**Not fixed, deliberately.** That is a renderer feature, not a harness, and
folding it into this change would muddy both. The replay test names the gap in
a comment and leaves it *uncovered* rather than asserting the current
behavior, because a test that pinned it would make the gap harder to close.

---

# Testing the tests

**31 mutations of `grmob-runtime.js`, 31 caught — but not on the first pass.**
Four survived, and every one was a real hole:

**The transcripts had no shape changes.** Counting patch types showed only
`update-props` and `update-style` across both scenarios — so the replay's
whole structural claim ("catches add-child at the wrong index, replace losing
its path") was not being exercised at all. Fixed by extending the signup
scenario to submit successfully and come back, which replaces the form
wholesale and produces `replace` patches in both directions.

**Three cases needed tests that did not exist:** a Checkbox whose listener is
attached *by a patch* (the case that distinguishes node type from tag on the
update path — the same confusion as the bug above, in its other half); a
listener given a new callback ID *after* attachment (capturing the ID in the
closure works exactly once); and `mount` clearing a non-empty root.

Caught: every arm of the Enter filter and `preventDefault`; the void envelope
for keydown and for focus/blur; the payload vocabulary in both directions;
checkbox `.checked` vs `.value`; `enterkeyhint` over- and under-set, never
removed, and not recomputed on update; focus applied inline instead of next
frame; epoch 0 read as a command; blur clearing focus it does not hold; the
empty action falling through to focus; the stale-epoch guard; a bare
`focusAction` acted on; listeners re-attached per patch and firing stale IDs;
the node type not recorded; `add-child` at a fixed index; `replace` losing its
path; child paths losing their index; `update-props` dropping value or text;
`disabled` announced but not set; `update-style` losing the flex axis; and
`mount` not clearing its root.

The harness snapshots the runtime to the scratchpad and restores from there,
with a `changed()` guard reporting `NOT-APPLIED` rather than a false verdict.
`git checkout` was not used to undo a mutation.

## Files touched

`wasm/verify/dom.mjs` (**new**, 314), `wasm/verify/load.mjs` (**new**, 106),
`wasm/verify/gen.go` (**new**, 230), `wasm/verify/replay_test.mjs` (**new**,
158), `wasm/verify/runtime_test.mjs` (**new**, 517), `wasm/verify/run.sh`
(**new**), `wasm/grmob-runtime.js` (the payload vocabulary fix),
`docs/platforms/wasm.md` (a "Testing without a browser" section).

`.mjs` throughout is deliberate: it makes these ES modules on every Node from
12 onward, where a bare `.js` would depend on the module-detection behavior of
the version in use (it happens to work on the 22.12 here).

Gate: `gofmt` clean, `go vet` clean, full Go suite, `GOOS=js GOARCH=wasm`
build, `ios/verify` (data-layer replay + Swift typecheck), and
`wasm/verify/run.sh` (29 tests) — all green. (`examples/todoapp/store.go`
remains unformatted — pre-existing, untouched.)

## Not verified here

**Still no browser.** This closes the "nothing but reading" gap, not the
"needs a browser" one. `enterkeyhint`'s actual effect on a soft keyboard,
whether `focus()` opens one, and every layout question remain unanswerable
here by construction, and the docs say so next to the harness rather than
leaving it implied.

**The Android side is still unbuilt** and **iOS still type-checks without
running** — both unchanged from the last two sessions, and now the only
renderers with no executable coverage at all.

## Backlog

In Progress still holds only Packaging (`grmob build --target=…`).

Closed this session: the missing JS test harness, and — as a side effect — a
total failure of text input in the browser that no test could have caught
before it existed.

Still open from earlier sessions:

- Theme contrast, and `Variant`'s third consumer.
- **Hooks have no unmount signal.**
- **`docs/concepts/architecture.md` does not mention frames.**
- **`core.Image` bases its style on `Theme.Components.Camera`.**
- The WASM runtime's style mapping is still much thinner than `htmlout`'s.
- **A bottom-docked bar has no way to ask for the keyboard on its own.**
- **`htmlout` renders `Scroll` as a plain div** with no `overflow`.
- **A second imperative API would justify the bridge command channel.**
- **`core.SendSystemEvent` is a dead stub** — `core/toast.go` is its only
  caller, so `ShowToast` currently reaches nothing.
- **A `Cached` subtree silently swallows focus commands** and order
  membership.
- **An app-drawn keyboard toolbar has no worked example**, so
  `core.FocusPrevious` and the `OnFocus`-tracks-the-current-field pattern are
  described in prose only.
- **`imeAction` is a third prop that must not vanish**, guarded by a third
  sticky sentinel; a single helper could state the rule once.

Noticed this session, not acted on:

- **The WASM runtime does not render `Checkbox` state or `TextArea` rows.**
  `checked` and `rows` reach no DOM property, and a Checkbox has no
  `type="checkbox"` attribute, so it draws as a text box. `htmlout` handles
  all three. This is the largest known correctness gap in the WASM renderer
  and now has a harness ready to test the fix.
- **`gen.go`'s transcripts still emit no `add`, `remove` or `add-child`.**
  `replace` was added this session by extending the signup scenario; the other
  three are covered by unit tests but not by a realistic replay. A growing
  list — `examples/todoapp` — would supply them.
- **The demo scenario's tab switches produce only `update-props`**, which is
  worth understanding: three visibly different tabs diffing without a single
  structural patch is either a nice property of the reconciler or a sign the
  scenario is not switching what it thinks it is.
- **Nothing runs either verify harness automatically.** Both are shell scripts
  a human remembers to run; there is no CI, and `go test ./...` does not reach
  them.
