# Session: Deduping the node-type → input-type table

**Session ID:** session_01LZZTYuh1UXxGjKfkomQRM6
**Date:** 2026-08-31, ~23:57
**Branch:** master
**Follows:** `2026-0831-2347-wasm-checkbox-textarea.md`

The previous session ended by noticing what it had just created:

> **`htmlout` and the WASM runtime now carry two copies of the node-type →
> input-type table** (`export.go` and `inputTypeFor`), plus a third
> restatement in the replay test. The third is deliberate — a conformance test
> must restate. The first two are a real duplication.

This session closed that.

---

# Part 1 — What "dedupe" can and cannot mean across two languages

The two copies are in Go and in JavaScript. One of them cannot be deleted:
the runtime sets the `type` attribute **in the browser**, at
`createElement` time, and has no way to call into Go to ask what it should
be. Any plan that ends with "one copy" has to answer where the JavaScript
gets its answer from, and there are only three answers:

| approach | cost |
|---|---|
| Carry the type in the node payload | New payload vocabulary on every node, and the previous sessions' worst bug was a payload-vocabulary drift |
| Generate the JS table from Go | A build step, in a harness whose stated virtue is "needs only Go and Node — no npm, no lockfile, no `node_modules`, no network" |
| One authority + a *checked* restatement | A test |

The third was chosen. The duplication that survives is no longer a
duplication anyone has to *remember* — which is what duplication actually
costs. A copy that cannot drift silently is a restatement, not a second
source.

## The Go authority

`htmlout/inputtype.go` holds the map, and exports two ways to reach it:

```go
func InputTypeFor(nodeType string) string  // query
func InputTypes() map[string]string        // enumerate, as a copy
```

Two exports for one table looks heavy, and the second exists for exactly one
caller: the conformance test has to compare *table against table*, and cannot
go through `InputTypeFor` one key at a time without first knowing the keys —
which is the thing it is trying to check. `InputTypes` returns a copy rather
than the map because a package-level map is writable by any importer, and a
four-entry table is cheaper to copy than to defend.

`export.go`'s four `<input>` cases now read `InputTypeFor(node.Type)` where
each previously spelled its own literal. The attribute *order* is untouched —
`withLead` still puts `type` first — so no exported HTML changed, which is
what kept `export_test.go` green without an edit.

The doc comment on `inputTypes` names all three copies and says which is the
authority, because the next person to add a node type needs to know where to
start and what will yell at them.

# Part 2 — Pinning the JavaScript copy

`wasm/verify/inputtype_test.go` reads the real `grmob-runtime.js` — the same
file `load.mjs` evaluates, at the same relative path — lifts the object
literal out of `inputTypeFor`, and compares it with `htmlout.InputTypes()`.

**It parses rather than executes.** Running the JS would need Node, and a Go
test that needs Node stops running for anyone who has only Go. The literal is
flat with no computation in it, so text is sufficient. The payoff is the
point of the whole exercise: this check runs under a plain `go test ./...`,
which is the one thing in this repo nobody has to remember to run. Both
verify harnesses are still shell scripts a human invokes — that backlog item
is unchanged, but this particular rule no longer depends on it.

Both directions are checked, and they are not symmetric:

- An entry the runtime has and Go does not → the runtime draws a control Go
  never asked for.
- An entry Go has and the runtime lacks → **the runtime draws a text box**,
  which is the exact shape of the Checkbox bug the previous session fixed.

The `|| ""` fallback is asserted separately, because it carries the other
half of the contract and lives outside the table: it is what makes a
`<textarea>` and a `<span>` get no `type` attribute at all. A table check
alone would pass a runtime that had grown a default of `"text"`.

## The guard that matters more than the comparison

Every parse step fails the test rather than returning a short map:

```
no inputTypeFor function found  →  fatal (named, so a rename says what to do)
no object literal               →  fatal
unterminated literal            →  fatal
braces not followed by [type]   →  fatal  (proves it grabbed the lookup)
table parsed as empty           →  fatal
```

This is the lesson from the previous session's false positive, where
`GRMOB_TRANSCRIPT` pointed at the wrong directory and two mutants read as
"caught" when the tests were merely failing to find a file. **A check that
reads nothing must not be able to read as a pass.** A rename of
`inputTypeFor` is meant to land as a loud failure that names the function,
not as an empty comparison that agrees with everything.

## Why the replay test's third copy stays

`replay_test.mjs`'s `INPUT_TYPE` was left independent on purpose, and the
comment now says why: pinning it to Go as well would close the loop back onto
the implementation and leave the rule with no witness written from the
contract. The chain that now exists is

```
replay_test.INPUT_TYPE  ←(replay)→  grmob-runtime.js  ←(go test)→  htmlout
        written from the contract        implementation        authority
```

Every copy is pinned to something. Only one is pinned to the implementation,
and that one is deliberately not the conformance test.

---

# Testing the tests

**7 mutations, 7 caught**, with an unmutated control run before *and* after
the table — the control is there because of the false positive above, so a
broken harness path cannot read as a good verdict:

| mutation | result |
|---|---|
| JS value drift (`Checkbox: "text"`) | caught |
| JS entry dropped (`NumericInput`) | caught |
| JS entry Go lacks (`DatePicker: "date"`) | caught |
| Go value drift (`InputPassword` → `text`) | caught |
| JS fallback removed (`}[type];`) | caught, by the second test |
| `inputTypeFor` renamed | caught, as a named fatal |
| table emptied | caught, as a named fatal |
| control: unmutated | passes |

Each mutation was restored from a scratchpad snapshot rather than
`git checkout`.

One further test, in `htmlout`: `InputTypes` hands out a copy. That contract
is stated in a doc comment and is otherwise unverified, and the conformance
test is the caller it exists for — it `delete`s entries from the returned map
as it matches them, which would rewrite the exporter's table if the copy were
not real.

## Files touched

`htmlout/inputtype.go` (new, +52: the table, `InputTypeFor`, `InputTypes`),
`htmlout/export.go` (4 literals → lookups), `htmlout/export_test.go` (+20:
the copy contract), `wasm/verify/inputtype_test.go` (new, +116: the parse and
both comparisons), `wasm/grmob-runtime.js` (comment: names the authority and
the test, and warns that the parse expects a flat literal in a function of
that name), `wasm/verify/replay_test.mjs` (comment: why this copy stays
independent), `docs/platforms/wasm.md` (+8: the same, in prose).

Gate: `gofmt` clean, `go vet` clean, full Go suite, `GOOS=js GOARCH=wasm`
build, `wasm/verify/run.sh` (36 tests, up from 34) and `ios/verify` (flex
solver + 9 patch batches + Swift typecheck) — all green.
(`examples/todoapp/store.go` remains unformatted — pre-existing, untouched.)

## Not verified here

No behavior changed. `htmlout`'s output is byte-identical (the attribute
order is unchanged) and the runtime's JavaScript is untouched apart from
comments — this session moved where a fact is *stated*, not what it says. The
existing suites passing unchanged is the evidence for that, and is the only
evidence available.

**Still no browser**, unchanged. **Android is still unbuilt** and **iOS still
type-checks without running**.

## Backlog

In Progress still holds only Packaging (`grmob build --target=…`).

Closed this session: the node-type → input-type duplication between
`htmlout` and the WASM runtime.

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
- **An app-drawn keyboard toolbar has no worked example.**
- **`imeAction` is a third prop that must not vanish**, guarded by a third
  sticky sentinel; a single helper could state the rule once.
- **`gen.go`'s transcripts still emit no `add`, `remove` or `add-child`.**
- **The demo scenario's tab switches produce only `update-props`.** Still not
  understood.
- **Nothing runs either verify harness automatically.** Unchanged as a
  general fact — but the input-type rule is now checked by `go test ./...`
  regardless, which is the first crack in it.
- **No example app uses `core.TextArea`** (nor `core.Image` / `CameraView`),
  so `rows` has unit coverage but no replay coverage.

Noticed this session, not acted on:

- **`tagForType` is duplicated the same way** — once in `htmlout/export.go`,
  once in `grmob-runtime.js` — and so is the `objectFit` / `objectFitFor`
  mapping. Both are the same shape as the table just fixed and would take the
  same treatment: a Go authority plus a parsed pin. The runtime's copy of
  `tagForType` is the larger of the two and covers node types `htmlout`'s
  `default` branch handles implicitly, so the two are not currently
  line-for-line comparable — a shared authority would have to settle that
  first.
- **The parse in `inputtype_test.go` is coupled to the shape of the JS**, not
  just its content: a flat object literal, in a function of a known name,
  followed by `[type]`. That is written down in both files, and every
  violation is a named fatal rather than a silent pass, but it is still a
  constraint on how that function may be rewritten.
